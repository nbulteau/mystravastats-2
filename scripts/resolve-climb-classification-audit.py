#!/usr/bin/env python3
"""Apply reviewed exact matches and close the remaining classification audit."""

from __future__ import annotations

import argparse
from collections import Counter
import datetime as dt
import json
from pathlib import Path
from urllib.parse import urlparse


CATALOG_NAMES = ("france", "suisse", "italie", "espagne", "andorre")


def load_json(path: Path) -> object:
    return json.loads(path.read_text(encoding="utf-8"))


def write_json(path: Path, value: object) -> None:
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def main() -> None:
    repository = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--audit",
        type=Path,
        default=repository / "docs/data-sources/climb-classification-audit.json",
    )
    parser.add_argument(
        "--resolutions",
        type=Path,
        default=repository / "docs/data-sources/climb-classification-resolutions.json",
    )
    parser.add_argument("--check", action="store_true")
    arguments = parser.parse_args()

    audit = load_json(arguments.audit)
    resolutions = load_json(arguments.resolutions)
    if audit.get("generatedAt") != resolutions.get("auditGeneratedAt"):
        raise ValueError("classification resolutions target a different audit generation")

    catalogs: dict[str, list[dict]] = {}
    alternatives_by_key: dict[tuple[str, str, str], dict] = {}
    alternatives_by_id: dict[str, dict] = {}
    for catalog_name in CATALOG_NAMES:
        catalog_path = repository / "back-go/famous-climb" / f"{catalog_name}.json"
        catalog = load_json(catalog_path)
        catalogs[catalog_name] = catalog
        for climb in catalog:
            for alternative in climb["alternatives"]:
                alternatives_by_key[(catalog_name, climb["name"], alternative["name"])] = alternative
                alternatives_by_id[alternative["id"]] = alternative

    accepted = {
        row["variantId"]: int(row["candidateId"])
        for row in resolutions.get("acceptedExactMatches", [])
    }
    accepted_seen: set[str] = set()
    changed = 0
    unchanged = 0
    kept_current = 0
    unresolved = 0
    catalog_mismatches = 0

    for entry in audit["entries"]:
        alternative = alternatives_by_key.get(
            (entry["catalog"], entry["climb"], entry["alternative"])
        )
        if alternative is None:
            raise ValueError(f"audit entry no longer exists: {entry}")
        confidence = entry.get("confidence")
        if confidence == "high":
            continue

        variant_id = alternative["id"]
        entry["variantId"] = variant_id
        accepted_candidate = accepted.get(variant_id)
        if accepted_candidate is not None:
            if entry.get("candidateId") != accepted_candidate:
                raise ValueError(f"accepted candidate changed for {variant_id}")
            new_difficulty = int(entry["sourceDifficulty"])
            new_category = str(entry["category"])
            catalog_mismatches += int(
                int(alternative.get("difficulty") or 0) != new_difficulty
                or str(alternative.get("category") or "") != new_category
            )
            alternative["difficulty"] = new_difficulty
            alternative["category"] = new_category
            was_changed = (
                int(entry.get("previousDifficulty") or 0) != new_difficulty
                or str(entry.get("previousCategory") or "") != new_category
            )
            expected_status = "updated" if was_changed else "verified"
            if arguments.check:
                catalog_mismatches += int(
                    entry.get("status") != expected_status
                    or entry.get("resolution") != "accepted-exact-match"
                )
            entry["status"] = expected_status
            entry["resolution"] = "accepted-exact-match"
            accepted_seen.add(variant_id)
            changed += int(was_changed)
            unchanged += int(not was_changed)
            continue

        if arguments.check:
            catalog_mismatches += int(
                entry.get("status") != "reviewed-kept-current"
                or int(entry.get("retainedDifficulty") or 0)
                != int(alternative.get("difficulty") or 0)
                or str(entry.get("retainedCategory") or "")
                != str(alternative.get("category") or "")
            )
        entry["status"] = "reviewed-kept-current"
        entry["resolution"] = resolutions["defaultReason"]
        entry["retainedDifficulty"] = int(alternative.get("difficulty") or 0)
        entry["retainedCategory"] = str(alternative.get("category") or "")
        entry["retainedSourceUrl"] = str(alternative.get("sourceUrl") or "")
        kept_current += 1

    missing_acceptances = set(accepted) - accepted_seen
    if missing_acceptances:
        raise ValueError(f"accepted variants missing from audit: {sorted(missing_acceptances)}")

    unresolved = sum(
        entry.get("status") in {"review-required", "unmatched"}
        for entry in audit["entries"]
    )
    previous_summary = audit["summary"]
    high_confidence = int(previous_summary["highConfidence"])
    high_changed = sum(
        entry.get("confidence") == "high"
        and (
            int(entry.get("previousDifficulty") or 0) != int(entry["sourceDifficulty"])
            or str(entry.get("previousCategory") or "") != str(entry["category"])
        )
        for entry in audit["entries"]
    )
    audit["resolvedAt"] = dt.datetime.now(dt.timezone.utc).isoformat()
    audit["resolutionMethod"] = (
        "Manual identity review accepts only exact documented routes. Ambiguous or absent "
        "Climbfinder matches retain the catalog classification and its traceable source URL."
    )
    retained_source_domains = Counter(
        urlparse(entry.get("retainedSourceUrl", "")).hostname or "missing"
        for entry in audit["entries"]
        if entry.get("status") == "reviewed-kept-current"
    )
    audit["reviewedRetainedSourceDomains"] = dict(sorted(retained_source_domains.items()))
    audit["summary"] = {
        **previous_summary,
        "applied": high_confidence + len(accepted_seen),
        "unchanged": high_confidence - high_changed + unchanged,
        "changed": high_changed + changed,
        "manuallyAcceptedExact": len(accepted_seen),
        "reviewedKeptCurrent": kept_current,
        "unresolved": unresolved,
    }

    if arguments.check:
        if unresolved or catalog_mismatches:
            raise SystemExit("classification audit or catalogs are not in their resolved state")
        return

    for catalog_name, catalog in catalogs.items():
        for root in (repository / "back-go/famous-climb", repository / "back-kotlin/famous-climb"):
            write_json(root / f"{catalog_name}.json", catalog)
    write_json(arguments.audit, audit)


if __name__ == "__main__":
    main()
