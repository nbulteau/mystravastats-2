#!/usr/bin/env python3
"""Refresh famous-climb difficulty points and categories from Climbfinder.

The catalog profiles primarily come from cols-cyclisme.com, which does not publish
the Cotacol difficulty used by the application.  This script matches every catalog
alternative with the public Climbfinder catalog, copies its published ``cotacol``
and ``category`` values, and writes an auditable report.  It never derives the
category from distance and average gradient.
"""

from __future__ import annotations

import argparse
import concurrent.futures
import datetime as dt
import difflib
import hashlib
import html
import json
import math
import os
from pathlib import Path
import re
import sys
import tempfile
import threading
import time
import unicodedata
import urllib.error
import urllib.parse
import urllib.request


API_BASE = "https://uphill.climbfinder.com/v2"
CLIMBFINDER_HEADERS = {
    "Origin": "https://climbfinder.com",
    "Referer": "https://climbfinder.com/",
    "User-Agent": "Mozilla/5.0 (compatible; MyStravaStats catalog audit)",
    "Accept": "application/json",
}
CATALOG_NAMES = ("france", "suisse", "italie", "espagne")
VALID_CATEGORIES = {"HC", "1", "2", "3", "4"}
PRINT_LOCK = threading.Lock()
GENERIC_CLIMB_TOKENS = {
    "alto", "alpe", "col", "coll", "collada", "collado", "cote", "de", "del", "della",
    "des", "di", "du", "el", "la", "le", "les", "pass", "passo", "port", "puerto",
    "station", "the", "to", "zur",
}


def parse_args() -> argparse.Namespace:
    repo_root = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--source-root",
        type=Path,
        default=repo_root / "back-go" / "famous-climb",
        help="Directory containing the authoritative national catalogs.",
    )
    parser.add_argument(
        "--mirror-root",
        action="append",
        type=Path,
        default=None,
        help="Catalog mirror directory. Repeat as needed.",
    )
    parser.add_argument(
        "--report",
        type=Path,
        default=repo_root / "docs" / "data-sources" / "climb-classification-audit.json",
        help="JSON audit report path.",
    )
    parser.add_argument(
        "--cache-dir",
        type=Path,
        default=Path(tempfile.gettempdir()) / "mystravastats-climbfinder-cache-v2",
        help="HTTP response cache directory.",
    )
    parser.add_argument("--workers", type=int, default=6)
    parser.add_argument(
        "--apply",
        action="store_true",
        help="Write accepted classifications to the source and mirror catalogs.",
    )
    parser.add_argument(
        "--accept-medium-confidence",
        action="store_true",
        help="Also apply matches that require manual review. High-confidence matches are always applied.",
    )
    args = parser.parse_args()
    if args.mirror_root is None:
        args.mirror_root = [
            repo_root / "back-kotlin" / "famous-climb",
            repo_root / "strava-cache" / "famous-climb",
        ]
    return args


def log(message: str) -> None:
    with PRINT_LOCK:
        print(message, flush=True)


def normalize_text(value: str) -> str:
    decoded = html.unescape(value or "")
    decomposed = unicodedata.normalize("NFKD", decoded)
    ascii_text = "".join(char for char in decomposed if not unicodedata.combining(char))
    return " ".join(re.findall(r"[a-z0-9]+", ascii_text.lower()))


def name_similarity(expected: str, candidate: str, alternative: str) -> float:
    expected_normalized = normalize_text(expected)
    candidate_normalized = normalize_text(candidate)
    sequence = difflib.SequenceMatcher(None, expected_normalized, candidate_normalized).ratio()
    expected_tokens = set(expected_normalized.split())
    candidate_tokens = set(candidate_normalized.split())
    union = expected_tokens | candidate_tokens
    jaccard = len(expected_tokens & candidate_tokens) / len(union) if union else 0.0
    alternative_tokens = {
        token
        for token in normalize_text(alternative).split()
        if token not in {"via", "par", "route", "road", "d", "de", "la", "le", "les"}
    }
    alternative_coverage = (
        len(alternative_tokens & candidate_tokens) / len(alternative_tokens)
        if alternative_tokens
        else 0.0
    )
    return max(sequence, 0.55 * jaccard + 0.45 * alternative_coverage)


def summit_name_match(expected: str, candidate: str) -> tuple[float, float]:
    candidate_base = re.split(
        r"\b(?:from|via|desde|da)\b", html.unescape(candidate), maxsplit=1
    )[0]
    expected_normalized = normalize_text(expected)
    candidate_normalized = normalize_text(candidate_base)
    sequence = difflib.SequenceMatcher(None, expected_normalized, candidate_normalized).ratio()
    candidate_tokens = {
        token for token in candidate_normalized.split() if token not in GENERIC_CLIMB_TOKENS
    }
    expected_parts = [part for part in re.split(r"[/()]", expected_normalized) if part.strip()]
    part_coverages = []
    all_expected_tokens: set[str] = set()
    for part in expected_parts:
        expected_tokens = {
            token for token in part.split() if token not in GENERIC_CLIMB_TOKENS
        }
        all_expected_tokens.update(expected_tokens)
        fuzzy_matches = sum(
            1
            for expected_token in expected_tokens
            if any(
                difflib.SequenceMatcher(None, expected_token, candidate_token).ratio() >= 0.72
                for candidate_token in candidate_tokens
            )
        )
        if expected_tokens:
            part_coverages.append(fuzzy_matches / len(expected_tokens))
    union = all_expected_tokens | candidate_tokens
    jaccard = (
        len(all_expected_tokens & candidate_tokens) / len(union) if union else 0.0
    )
    coverage = max(part_coverages, default=0.0)
    return max(sequence, jaccard), coverage


def haversine_km(first: dict, second: dict) -> float:
    first_lat = math.radians(float(first["latitude"]))
    second_lat = math.radians(float(second["latitude"]))
    delta_lat = second_lat - first_lat
    delta_lng = math.radians(float(second["longitude"]) - float(first["longitude"]))
    value = (
        math.sin(delta_lat / 2) ** 2
        + math.cos(first_lat) * math.cos(second_lat) * math.sin(delta_lng / 2) ** 2
    )
    return 6371.0088 * 2 * math.atan2(math.sqrt(value), math.sqrt(max(0.0, 1 - value)))


def relative_difference(first: float, second: float) -> float:
    denominator = max(abs(first), abs(second), 1.0)
    return abs(first - second) / denominator


class ApiClient:
    def __init__(self, cache_dir: Path):
        self.cache_dir = cache_dir
        self.cache_dir.mkdir(parents=True, exist_ok=True)
        self.request_count = 0
        self.cache_hits = 0
        self._count_lock = threading.Lock()
        self._rate_lock = threading.Lock()
        self._next_request_at = 0.0

    def _wait_for_rate_slot(self) -> None:
        # Climbfinder currently allows 100 requests per minute. Reserving one
        # request every 0.8 seconds leaves headroom for retries and browser use.
        with self._rate_lock:
            now = time.time()
            reserved_at = max(now, self._next_request_at)
            self._next_request_at = reserved_at + 0.8
        delay = reserved_at - now
        if delay > 0:
            time.sleep(delay)

    def _pause_until_reset(self, reset_epoch: str | None) -> None:
        try:
            delay = max(float(reset_epoch or 0) - time.time() + 2, 5)
        except ValueError:
            delay = 65
        delay = min(delay, 90)
        with self._rate_lock:
            self._next_request_at = max(self._next_request_at, time.time() + delay)
        log(f"Climbfinder rate limit reached; resuming in {math.ceil(delay)} s")

    def get(self, url: str) -> dict:
        cache_key = hashlib.sha256(url.encode("utf-8")).hexdigest()
        cache_path = self.cache_dir / f"{cache_key}.json"
        if cache_path.exists():
            with self._count_lock:
                self.cache_hits += 1
            return json.loads(cache_path.read_text(encoding="utf-8"))

        request = urllib.request.Request(url, headers=CLIMBFINDER_HEADERS)
        for attempt in range(1, 7):
            try:
                self._wait_for_rate_slot()
                with urllib.request.urlopen(request, timeout=30) as response:
                    payload = response.read().decode("utf-8")
                    remaining = response.headers.get("x-ratelimit-remaining")
                    reset_epoch = response.headers.get("x-ratelimit-reset")
                    if remaining is not None and int(remaining) <= 2:
                        self._pause_until_reset(reset_epoch)
                parsed = json.loads(payload)
                temporary_path = cache_path.with_suffix(f".{os.getpid()}.{threading.get_ident()}.tmp")
                temporary_path.write_text(payload, encoding="utf-8")
                temporary_path.replace(cache_path)
                with self._count_lock:
                    self.request_count += 1
                return parsed
            except urllib.error.HTTPError as error:
                if error.code == 429:
                    self._pause_until_reset(error.headers.get("x-ratelimit-reset"))
                elif attempt == 6:
                    raise RuntimeError(f"Unable to download {url}: {error}") from error
                else:
                    time.sleep(min(2**attempt, 20))
            except (urllib.error.URLError, TimeoutError, json.JSONDecodeError) as error:
                if attempt == 6:
                    raise RuntimeError(f"Unable to download {url}: {error}") from error
                time.sleep(min(2**attempt, 20))
        raise AssertionError("unreachable")

    def search(self, term: str) -> list[dict]:
        query = urllib.parse.urlencode({"term": term, "language": "en", "limit": 50})
        payload = self.get(f"{API_BASE}/search?{query}")
        data = payload.get("data") or {}
        if isinstance(data, list):
            return data
        return data.get("climbs") or []

    def detail(self, candidate_id: int) -> dict:
        payload = self.get(f"{API_BASE}/climbs/{candidate_id}?language=en")
        return payload.get("data") or {}


def load_catalogs(source_root: Path) -> tuple[dict[str, list], list[dict]]:
    catalogs: dict[str, list] = {}
    climbs: list[dict] = []
    for catalog_name in CATALOG_NAMES:
        path = source_root / f"{catalog_name}.json"
        catalog = json.loads(path.read_text(encoding="utf-8-sig"))
        catalogs[catalog_name] = catalog
        for climb_index, climb in enumerate(catalog):
            climb_ref = {
                "catalog": catalog_name,
                "climbIndex": climb_index,
                "climb": climb,
                "country": climb.get("country", ""),
                "summit": climb["geoCoordinate"],
                "key": f"{catalog_name}:{climb_index}",
            }
            climbs.append(climb_ref)
    return catalogs, climbs


def parallel_map(function, values: list, workers: int, label: str) -> dict:
    results = {}
    completed = 0
    total = len(values)
    with concurrent.futures.ThreadPoolExecutor(max_workers=max(1, workers)) as executor:
        futures = {executor.submit(function, value): value for value in values}
        for future in concurrent.futures.as_completed(futures):
            value = futures[future]
            results[value] = future.result()
            completed += 1
            if completed % 50 == 0 or completed == total:
                log(f"{label}: {completed}/{total}")
    return results


def candidate_start(candidate: dict) -> dict | None:
    if candidate.get("lat") is None or candidate.get("lng") is None:
        return None
    return {"latitude": candidate["lat"], "longitude": candidate["lng"]}


def prefilter_candidates(climb_ref: dict, candidates: list[dict]) -> list[dict]:
    climb = climb_ref["climb"]
    alternatives = climb["alternatives"]
    country_candidates = [
        candidate
        for candidate in candidates
        if candidate_start(candidate) is not None
    ]
    selected: dict[int, dict] = {}
    for alternative in alternatives:
        expected_name = f"{climb['name']} from {alternative['name']}"
        ranked = []
        for candidate in country_candidates:
            start_distance = haversine_km(
                alternative["geoCoordinate"], candidate_start(candidate)
            )
            similarity = name_similarity(
                expected_name, candidate.get("name", ""), alternative["name"]
            )
            source_country = candidate.get("countryIso")
            if source_country != climb_ref["country"] and not (
                start_distance <= 7 and similarity >= 0.65
            ):
                continue
            if start_distance > 25 and not (start_distance <= 50 and similarity >= 0.76):
                continue
            preliminary_score = start_distance + (1 - similarity) * 10
            ranked.append((preliminary_score, start_distance, candidate))
        ranked.sort(key=lambda item: item[0])
        if not ranked:
            continue
        best_score, _, best = ranked[0]
        selected[int(best["id"])] = best
        # Keep a near-tied candidate so one-to-one assignment can distinguish
        # alternatives sharing the same town but taking different roads. Also
        # keep the popular route at an identical start: it often has the exact
        # Tour finish while another result stops earlier in the resort.
        for score, start_distance, candidate in ranked[1:]:
            is_near_tie = score - best_score <= 1.5
            is_nearby_popular = bool(candidate.get("variantPopular")) and start_distance <= 2
            if is_near_tie or is_nearby_popular:
                selected[int(candidate["id"])] = candidate
            if len(selected) >= len(alternatives) + 2:
                break
    return list(selected.values())


def normalize_category(category: object) -> str | None:
    normalized = str(category or "").strip().upper()
    if normalized == "SHC":
        return "HC"
    if normalized in VALID_CATEGORIES:
        return normalized
    if normalized in {"5", "NC"}:
        return "4"
    return None


def build_pair(climb_ref: dict, alternative_index: int, detail: dict) -> dict | None:
    climb = climb_ref["climb"]
    alternative = climb["alternatives"][alternative_index]
    marker = (detail.get("marker") or {}).get("coordinates") or []
    if len(marker) < 2 or detail.get("length") is None or detail.get("ascent") is None:
        return None
    source_category = str(detail.get("category") or "").upper()
    category = normalize_category(source_category)
    difficulty = detail.get("cotacol")
    if category is None or not isinstance(difficulty, (int, float)) or difficulty <= 0:
        return None

    detail_start = None
    search_start = detail.get("_searchStart")
    if search_start:
        detail_start = search_start
    if detail_start is None:
        return None

    summit = {"latitude": marker[1], "longitude": marker[0]}
    full_expected_name = f"{climb['name']} from {alternative['name']}"
    candidate_name = detail.get("nameComplete") or detail.get("name") or ""
    start_distance = haversine_km(alternative["geoCoordinate"], detail_start)
    summit_distance = haversine_km(climb_ref["summit"], summit)
    source_length_km = float(detail["length"]) / 1000.0
    length_difference = relative_difference(float(alternative["length"]), source_length_km)
    ascent_difference = relative_difference(float(alternative["totalAscent"]), float(detail["ascent"]))
    similarity = name_similarity(full_expected_name, candidate_name, alternative["name"])
    summit_similarity, summit_token_coverage = summit_name_match(climb["name"], candidate_name)
    catalog_source_url = str(alternative.get("sourceUrl") or "").rstrip("/")
    candidate_source_url = str(detail.get("link") or "").rstrip("/")
    source_url_matches = bool(catalog_source_url) and catalog_source_url == candidate_source_url

    eligible = (
        summit_distance <= 12
        and length_difference <= 0.55
        and ascent_difference <= 0.65
        and (start_distance <= 25 or (start_distance <= 50 and similarity >= 0.76))
        and summit_token_coverage >= 0.50
    )
    if not eligible:
        return None

    score = (
        min(start_distance, 50) * 0.8
        + summit_distance * 2.0
        + length_difference * 25
        + ascent_difference * 15
        + (1 - similarity) * 10
        + (1 - summit_similarity) * 10
        - (8 if source_url_matches else 0)
    )
    confidence = "high" if (
        score <= 20
        and start_distance <= 7
        and summit_distance <= 4
        and length_difference <= 0.30
        and ascent_difference <= 0.40
        and summit_similarity >= 0.50
        and summit_token_coverage >= 0.66
    ) else "medium"
    return {
        "alternativeIndex": alternative_index,
        "candidateId": int(detail["id"]),
        "candidateName": html.unescape(candidate_name),
        "sourceUrl": detail.get("link"),
        "catalogSourceUrlMatches": source_url_matches,
        "sourceDifficulty": int(round(float(difficulty))),
        "sourceCategory": source_category,
        "category": category,
        "confidence": confidence,
        "score": round(score, 3),
        "startDistanceKm": round(start_distance, 3),
        "summitDistanceKm": round(summit_distance, 3),
        "lengthDifferencePercent": round(length_difference * 100, 1),
        "ascentDifferencePercent": round(ascent_difference * 100, 1),
        "nameSimilarity": round(similarity, 3),
        "summitNameSimilarity": round(summit_similarity, 3),
        "summitTokenCoverage": round(summit_token_coverage, 3),
        "sourceLengthKm": round(source_length_km, 3),
        "sourceAscent": int(round(float(detail["ascent"]))),
    }


def assign_candidates(climb_ref: dict, details: list[dict]) -> tuple[dict[int, dict], dict[int, list[dict]]]:
    pairs: list[dict] = []
    pairs_by_alternative: dict[int, list[dict]] = {}
    for alternative_index, _ in enumerate(climb_ref["climb"]["alternatives"]):
        for detail in details:
            pair = build_pair(climb_ref, alternative_index, detail)
            if pair is not None:
                pairs.append(pair)
                pairs_by_alternative.setdefault(alternative_index, []).append(pair)
    for candidates in pairs_by_alternative.values():
        candidates.sort(key=lambda pair: pair["score"])

    assignments: dict[int, dict] = {}
    assigned_candidates: set[int] = set()
    for pair in sorted(pairs, key=lambda value: value["score"]):
        alternative_index = pair["alternativeIndex"]
        candidate_id = pair["candidateId"]
        if alternative_index in assignments or candidate_id in assigned_candidates:
            continue
        assignments[alternative_index] = pair
        assigned_candidates.add(candidate_id)
    return assignments, pairs_by_alternative


def collect_searches(client: ApiClient, climbs: list[dict], workers: int) -> dict[str, list[dict]]:
    values = [climb_ref["key"] for climb_ref in climbs]
    refs_by_key = {climb_ref["key"]: climb_ref for climb_ref in climbs}
    return parallel_map(
        lambda key: client.search(refs_by_key[key]["climb"]["name"]),
        values,
        workers,
        "Summit searches",
    )


def fetch_details(client: ApiClient, candidates: dict[int, dict], workers: int) -> dict[int, dict]:
    ids = sorted(candidates)

    def fetch(candidate_id: int) -> dict:
        detail = client.detail(candidate_id)
        start = candidate_start(candidates[candidate_id])
        if start is not None:
            detail["_searchStart"] = start
        return detail

    return parallel_map(fetch, ids, workers, "Climb details")


def collect_candidate_pool(climbs: list[dict], searches: dict[str, list[dict]]) -> dict[int, dict]:
    candidates: dict[int, dict] = {}
    for climb_ref in climbs:
        for candidate in prefilter_candidates(climb_ref, searches[climb_ref["key"]]):
            candidates[int(candidate["id"])] = candidate
    return candidates


def add_fallback_candidates(
    client: ApiClient,
    climbs: list[dict],
    details: dict[int, dict],
    workers: int,
) -> tuple[dict[int, dict], dict[str, list[dict]]]:
    queries: dict[str, tuple[dict, int]] = {}
    current_assignments: dict[str, dict[int, dict]] = {}
    for climb_ref in climbs:
        climb_details = [
            detail
            for detail in details.values()
            if detail.get("countryIso") == climb_ref["country"]
            and normalize_text(climb_ref["climb"]["name"]) in normalize_text(detail.get("nameComplete", ""))
        ]
        assignments, _ = assign_candidates(climb_ref, climb_details)
        current_assignments[climb_ref["key"]] = assignments
        for alternative_index, alternative in enumerate(climb_ref["climb"]["alternatives"]):
            if alternative_index not in assignments:
                query_key = f"{climb_ref['key']}:{alternative_index}"
                queries[query_key] = (
                    climb_ref,
                    alternative_index,
                )
    if not queries:
        return {}, {}

    search_results = parallel_map(
        lambda key: client.search(
            f"{queries[key][0]['climb']['name']} {queries[key][0]['climb']['alternatives'][queries[key][1]]['name']}"
        ),
        list(queries),
        workers,
        "Variant fallback searches",
    )
    candidates: dict[int, dict] = {}
    for key, results in search_results.items():
        climb_ref, _ = queries[key]
        for candidate in prefilter_candidates(climb_ref, results):
            candidates[int(candidate["id"])] = candidate
    return candidates, search_results


def candidate_details_for_climb(climb_ref: dict, details: dict[int, dict]) -> list[dict]:
    result = []
    normalized_name = normalize_text(climb_ref["climb"]["name"])
    summit = climb_ref["summit"]
    for detail in details.values():
        marker = (detail.get("marker") or {}).get("coordinates") or []
        if len(marker) < 2:
            continue
        candidate_summit = {"latitude": marker[1], "longitude": marker[0]}
        summit_distance = haversine_km(summit, candidate_summit)
        candidate_name = normalize_text(detail.get("nameComplete") or detail.get("name") or "")
        name_score = difflib.SequenceMatcher(None, normalized_name, candidate_name).ratio()
        source_country = detail.get("countryIso")
        country_matches = source_country == climb_ref["country"]
        start = detail.get("_searchStart")
        nearest_start = (
            min(
                haversine_km(start, alternative["geoCoordinate"])
                for alternative in climb_ref["climb"]["alternatives"]
            )
            if start
            else math.inf
        )
        if summit_distance <= 12 and name_score >= 0.25 and (
            country_matches or nearest_start <= 7
        ):
            result.append(detail)
    return result


def make_report_and_apply(
    catalogs: dict[str, list],
    climbs: list[dict],
    details: dict[int, dict],
    accept_medium: bool,
) -> dict:
    entries = []
    summary = {
        "totalAlternatives": 0,
        "matched": 0,
        "applied": 0,
        "unchanged": 0,
        "changed": 0,
        "highConfidence": 0,
        "mediumConfidence": 0,
        "unmatched": 0,
    }
    for climb_ref in climbs:
        climb = climb_ref["climb"]
        assignments, pairs_by_alternative = assign_candidates(
            climb_ref, candidate_details_for_climb(climb_ref, details)
        )
        for alternative_index, alternative in enumerate(climb["alternatives"]):
            summary["totalAlternatives"] += 1
            old_difficulty = int(alternative.get("difficulty") or 0)
            old_category = str(alternative.get("category") or "")
            pair = assignments.get(alternative_index)
            entry = {
                "catalog": climb_ref["catalog"],
                "climb": climb["name"],
                "alternative": alternative["name"],
                "previousDifficulty": old_difficulty,
                "previousCategory": old_category,
            }
            if pair is None:
                summary["unmatched"] += 1
                alternatives = pairs_by_alternative.get(alternative_index, [])[:3]
                entry["status"] = "unmatched"
                entry["candidates"] = alternatives
                entries.append(entry)
                continue

            confidence = pair["confidence"]
            summary["matched"] += 1
            summary[f"{confidence}Confidence"] += 1
            should_apply = confidence == "high" or accept_medium
            changed = old_difficulty != pair["sourceDifficulty"] or old_category != pair["category"]
            if should_apply:
                alternative["difficulty"] = pair["sourceDifficulty"]
                alternative["category"] = pair["category"]
                summary["applied"] += 1
                summary["changed" if changed else "unchanged"] += 1
                status = "updated" if changed else "verified"
            else:
                status = "review-required"
            entry.update(pair)
            entry["status"] = status
            entries.append(entry)
    return {
        "generatedAt": dt.datetime.now(dt.timezone.utc).isoformat(),
        "source": "Climbfinder public API",
        "method": (
            "One-to-one variant matching by country, start coordinate, summit coordinate, "
            "distance, ascent and normalized full variant name. SHC is normalized to HC; "
            "Climbfinder categories 5/NC are normalized to application category 4."
        ),
        "summary": summary,
        "entries": entries,
    }


def write_json(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        json.dumps(value, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )


def main() -> int:
    args = parse_args()
    if args.workers < 1 or args.workers > 12:
        raise ValueError("--workers must be between 1 and 12")
    catalogs, climbs = load_catalogs(args.source_root)
    alternative_count = sum(len(ref["climb"]["alternatives"]) for ref in climbs)
    log(f"Loaded {len(climbs)} summits and {alternative_count} alternatives")

    client = ApiClient(args.cache_dir)
    searches = collect_searches(client, climbs, args.workers)
    candidate_pool = collect_candidate_pool(climbs, searches)
    log(f"Initial candidate pool: {len(candidate_pool)} climbs")
    details = fetch_details(client, candidate_pool, args.workers)

    fallback_candidates, _ = add_fallback_candidates(client, climbs, details, args.workers)
    missing_candidate_ids = {
        candidate_id: candidate
        for candidate_id, candidate in fallback_candidates.items()
        if candidate_id not in details
    }
    if missing_candidate_ids:
        log(f"Fallback candidate pool: {len(missing_candidate_ids)} additional climbs")
        details.update(fetch_details(client, missing_candidate_ids, args.workers))

    report = make_report_and_apply(
        catalogs,
        climbs,
        details,
        accept_medium=args.accept_medium_confidence,
    )
    write_json(args.report, report)

    if args.apply:
        roots = [args.source_root, *args.mirror_root]
        for root in roots:
            for catalog_name, catalog in catalogs.items():
                write_json(root / f"{catalog_name}.json", catalog)
        log(f"Updated {len(roots)} synchronized catalog roots")
    else:
        log("Dry run: catalogs were not changed (use --apply to write accepted matches)")

    summary = report["summary"]
    log(
        "Classification audit: "
        f"{summary['matched']}/{summary['totalAlternatives']} matched, "
        f"{summary['highConfidence']} high confidence, "
        f"{summary['mediumConfidence']} medium confidence, "
        f"{summary['unmatched']} unmatched, {summary['changed']} changed"
    )
    log(f"Audit report: {args.report}")
    log(f"HTTP: {client.request_count} downloads, {client.cache_hits} cache hits")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except KeyboardInterrupt:
        log("Interrupted")
        raise SystemExit(130)
