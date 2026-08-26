#!/usr/bin/env python3
"""Validate mirrored famous-climb catalogs and publish deterministic coverage."""

from __future__ import annotations

import argparse
from collections import Counter
from datetime import date
import json
from pathlib import Path
from urllib.parse import urlparse


CATALOGS = ("france", "suisse", "italie", "espagne", "andorre")
EXPECTED = {
    # Two legacy Les Deux-Alpes records intentionally share one semantic summit identity.
    "FR": {"summits": 245, "variants": 508},
    "CH": {"summits": 23, "variants": 47},
    "IT": {"summits": 31, "variants": 78},
    "ES": {"summits": 91, "variants": 127},
    "AD": {"summits": 19, "variants": 24},
}


def validate_variant(climb: dict, variant: dict, variant_ids: set[str]) -> None:
    variant_id = variant.get("id", "")
    if not variant_id.startswith(f"{climb['id']}--") or variant_id in variant_ids:
        raise ValueError(f"invalid or duplicate variant id: {variant_id!r}")
    variant_ids.add(variant_id)
    length = float(variant.get("length", 0))
    ascent = int(variant.get("totalAscent", 0))
    average = float(variant.get("averageGradient", 0))
    maximum = float(variant.get("maximumGradient", 0))
    minimum_altitude = int(variant.get("minimumAltitude", max(0, climb["topOfTheAscent"] - ascent)))
    if length <= 0 or ascent <= 0 or average <= 0:
        raise ValueError(f"flat or incomplete profile: {variant_id}")
    estimated_ascent = length * average * 10
    if not ascent * 0.75 <= estimated_ascent <= ascent * 1.25:
        raise ValueError(f"distance/ascent/gradient mismatch: {variant_id}")
    altitude_ascent = climb["topOfTheAscent"] - minimum_altitude
    if minimum_altitude > 0 and abs(altitude_ascent - ascent) > max(250, ascent * 0.35):
        raise ValueError(f"altitude/ascent mismatch: {variant_id}")
    if maximum < 0 or maximum > 30 or (maximum > 0 and maximum + 0.1 < average):
        raise ValueError(f"aberrant maximum gradient: {variant_id}")
    source = urlparse(variant.get("sourceUrl", ""))
    if source.scheme != "https" or not source.netloc:
        raise ValueError(f"missing traceable source: {variant_id}")


def load_sources(path: Path) -> dict:
    metadata = json.loads(path.read_text(encoding="utf-8"))
    catalogs = metadata.get("catalogs", {})
    if set(catalogs) != set(CATALOGS):
        raise ValueError("catalog source metadata differs from the catalog list")
    for catalog_name, row in catalogs.items():
        if row.get("country") not in EXPECTED:
            raise ValueError(f"invalid source country for {catalog_name}")
        date.fromisoformat(row.get("verifiedAt", ""))
        source = urlparse(row.get("sourceUrl", ""))
        if source.scheme != "https" or not source.netloc:
            raise ValueError(f"invalid catalog source for {catalog_name}")
    return metadata


def audit(source: Path, mirror: Path, sources_path: Path) -> dict:
    sources = load_sources(sources_path)
    countries: dict[str, dict] = {}
    summit_coordinates: dict[str, tuple[float, float]] = {}
    variant_ids: set[str] = set()
    for catalog_name in CATALOGS:
        source_path = source / f"{catalog_name}.json"
        mirror_path = mirror / f"{catalog_name}.json"
        if source_path.read_bytes() != mirror_path.read_bytes():
            raise ValueError(f"catalog mirror differs: {catalog_name}")
        climbs = json.loads(source_path.read_text(encoding="utf-8"))
        for climb in climbs:
            country = climb["country"]
            summit_id = climb.get("id", "")
            coordinate = climb["geoCoordinate"]
            point = (coordinate["latitude"], coordinate["longitude"])
            previous = summit_coordinates.get(summit_id)
            if not summit_id or (previous and max(abs(previous[0] - point[0]), abs(previous[1] - point[1])) > 0.002):
                raise ValueError(f"invalid semantic summit identity: {summit_id!r}")
            summit_coordinates[summit_id] = point
            country_row = countries.setdefault(country, {"massifs": Counter(), "summitIds": set(), "variants": 0})
            country_row["massifs"][climb["massif"]] += len(climb["alternatives"])
            country_row["summitIds"].add(summit_id)
            for variant in climb["alternatives"]:
                validate_variant(climb, variant, variant_ids)
                country_row["variants"] += 1

    output_countries = []
    for country in sorted(countries):
        row = countries[country]
        expected = EXPECTED[country]
        found_summits = len(row["summitIds"])
        found_variants = row["variants"]
        if found_summits != expected["summits"] or found_variants != expected["variants"]:
            raise ValueError(f"coverage regression for {country}: {found_summits}/{found_variants}")
        catalog_name = next(
            name for name, metadata in sources["catalogs"].items()
            if metadata["country"] == country
        )
        source_metadata = sources["catalogs"][catalog_name]
        output_countries.append({
            "country": country,
            "expectedSummits": expected["summits"],
            "foundSummits": found_summits,
            "expectedVariants": expected["variants"],
            "foundVariants": found_variants,
            "verifiedAt": source_metadata["verifiedAt"],
            "sourceUrl": source_metadata["sourceUrl"],
            "massifs": [
                {"name": name, "foundVariants": count}
                for name, count in sorted(row["massifs"].items())
            ],
        })
    return {
        "schemaVersion": 2,
        "summitIdentityCount": len(summit_coordinates),
        "variantCount": len(variant_ids),
        "countries": output_countries,
        "manualReviewVariants": sources.get("manualReviewVariants", []),
    }


def main() -> None:
    repository = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="Fail if the committed report is stale.")
    args = parser.parse_args()
    report_path = repository / "docs/data-sources/climb-catalog-coverage.json"
    report = json.dumps(
        audit(
            repository / "back-go/famous-climb",
            repository / "back-kotlin/famous-climb",
            repository / "docs/data-sources/climb-catalog-sources.json",
        ),
        ensure_ascii=False,
        indent=2,
    ) + "\n"
    if args.check:
        if not report_path.is_file() or report_path.read_text(encoding="utf-8") != report:
            raise SystemExit("climb catalog coverage report is stale; run scripts/audit-climb-catalog.py")
        return
    report_path.write_text(report, encoding="utf-8")


if __name__ == "__main__":
    main()
