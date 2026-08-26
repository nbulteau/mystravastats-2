# Famous-climb identity

The national catalogs are the source of truth for climb identity. Every summit
has an explicit `id`, and every alternative/ascent side has an explicit `id`
prefixed by its summit identity:

```text
climb-fr-col-du-galibier
climb-fr-col-du-galibier--valloire
```

These identifiers are API data, not presentation slugs. Renaming a displayed
badge, correcting a coordinate, profile, metric, source, or category must not
change an existing identifier. New catalog entries receive identifiers through
`scripts/assign-climb-stable-ids.py`; the script preserves identifiers already
present and refuses duplicated variant identities or distant summits sharing an
identity.

Go and Kotlin expose the values as `climbDetails.summitId` and
`climbDetails.variantId`. Their shared response contract is
[`../api/badges.openapi.yaml`](../api/badges.openapi.yaml). The frontend uses
those values for detail URLs and only derives a legacy fallback when talking to
an older backend.

## Catalog conventions

- `name` identifies the summit; `Alternative.name` identifies the departure or
  discriminating route, never a marketing label.
- `geoCoordinate` on the summit and alternative are respectively the published
  finish and route start. Correcting them must preserve existing IDs.
- `routeCheckpoints` are ordered, route-specific passages. Add them only when
  start/finish and distance cannot distinguish nearby alternatives.
- `summitToleranceMeters` defaults to 500 m and may only be reduced to prevent a
  known false positive near a junction or shared road.
- every alternative must retain an HTTPS `sourceUrl`; metric corrections without
  a traceable source are rejected by the catalog audit.

Run `scripts/audit-climb-catalog.py --check` to verify mirror equality, semantic
identity uniqueness, altitude/ascent/gradient plausibility, source traceability,
and the committed coverage baseline. Running it without `--check` refreshes
[`../data-sources/climb-catalog-coverage.json`](../data-sources/climb-catalog-coverage.json).
