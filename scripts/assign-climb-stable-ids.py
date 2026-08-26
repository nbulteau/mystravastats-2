#!/usr/bin/env python3
"""Assign durable summit and variant identifiers to the famous-climb catalogs."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
import re
import unicodedata


CATALOG_NAMES = ("france", "suisse", "italie", "espagne", "andorre")


def slug(value: str) -> str:
    decomposed = unicodedata.normalize("NFKD", value)
    ascii_value = "".join(character for character in decomposed if not unicodedata.combining(character))
    return re.sub(r"(^-|-$)", "", re.sub(r"[^a-z0-9]+", "-", ascii_value.lower())) or "unknown"


def assign_ids(catalog: list[dict]) -> list[dict]:
    for climb in catalog:
        summit_id = climb.get("id") or f"climb-{slug(climb['country'])}-{slug(climb['name'])}"
        climb["id"] = summit_id
        for alternative in climb.get("alternatives", []):
            alternative["id"] = alternative.get("id") or f"{summit_id}--{slug(alternative['name'])}"
    return catalog


def validate(catalogs: list[list[dict]]) -> None:
    summit_coordinates: dict[str, tuple[float, float]] = {}
    variant_ids: set[str] = set()
    for catalog in catalogs:
        for climb in catalog:
            summit_id = climb["id"]
            coordinate = climb["geoCoordinate"]
            current = (coordinate["latitude"], coordinate["longitude"])
            previous = summit_coordinates.get(summit_id)
            if previous and max(abs(previous[0] - current[0]), abs(previous[1] - current[1])) > 0.002:
                raise ValueError(f"summit id {summit_id!r} refers to distant coordinates")
            summit_coordinates[summit_id] = current
            for alternative in climb.get("alternatives", []):
                variant_id = alternative["id"]
                if variant_id in variant_ids:
                    raise ValueError(f"duplicate variant id {variant_id!r}")
                if not variant_id.startswith(f"{summit_id}--"):
                    raise ValueError(f"variant id {variant_id!r} is not attached to {summit_id!r}")
                variant_ids.add(variant_id)


def main() -> None:
    repository = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, default=repository / "back-go" / "famous-climb")
    parser.add_argument(
        "--mirror",
        action="append",
        type=Path,
        default=None,
        help="Mirror directory to update. Repeat for multiple mirrors.",
    )
    arguments = parser.parse_args()
    mirrors = arguments.mirror or [repository / "back-kotlin" / "famous-climb"]

    loaded: dict[str, list[dict]] = {}
    for name in CATALOG_NAMES:
        source_path = arguments.source / f"{name}.json"
        loaded[name] = assign_ids(json.loads(source_path.read_text(encoding="utf-8")))
    validate(list(loaded.values()))

    for name, catalog in loaded.items():
        content = json.dumps(catalog, ensure_ascii=False, indent=2) + "\n"
        (arguments.source / f"{name}.json").write_text(content, encoding="utf-8")
        for mirror in mirrors:
            mirror.mkdir(parents=True, exist_ok=True)
            (mirror / f"{name}.json").write_text(content, encoding="utf-8")


if __name__ == "__main__":
    main()
