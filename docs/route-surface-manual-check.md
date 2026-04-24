# Route surface scoring check (OSM `surface` / `tracktype`)

This guide validates `ROUTE-P1-01` on real target-generation responses.

Validated rules:

- surface reasons are always exposed (`Surface mix`, `Path ratio`, `Surface fitness`, `Surface source`),
- dense paved profile keeps `RIDE` as-is while `GRAVEL/MTB` trigger route-type fallback,
- mixed urban/paths profile keeps route-type behavior distinct:
  - `RIDE` remains ride-oriented with high surface fitness,
  - `GRAVEL` triggers fallback when path ratio is still below gravel minimum,
  - `MTB` stays in MTB mode (no route-type fallback) with a clearly lower fitness on mostly paved corridors.

## Prerequisites

- OSRM is running and reachable.
- Go backend is running from current source (`back-go`), default URL `http://127.0.0.1:8080`.

## Quick run

```bash
./scripts/manual-route-surface-check.sh
```

The script:

- checks `/api/health/details`,
- runs route generation for `RIDE`, `GRAVEL`, `MTB` on two scenarios:
  - `dense-urban` (paved profile),
  - `mixed-urban-paths` (small offroad share),
- enforces surface reasons presence on all top routes,
- enforces fallback/non-fallback expectations per route type,
- checks calibrated thresholds on path ratio and surface fitness,
- prints compact JSON summaries.

## Useful overrides

```bash
BACKEND_URL=http://127.0.0.1:8080 \
ACTIVITY_TYPE=Ride \
VALIDATION_YEAR=1990 \
VARIANT_COUNT=6 \
./scripts/manual-route-surface-check.sh
```

You can also override both scenario coordinates/targets:

```bash
DENSE_START_LAT=48.1175 \
DENSE_START_LNG=-1.6780 \
DENSE_DISTANCE_KM=40 \
DENSE_ELEVATION_M=800 \
MIXED_START_LAT=48.1141 \
MIXED_START_LNG=-1.6144 \
MIXED_DISTANCE_KM=25 \
MIXED_ELEVATION_M=250 \
./scripts/manual-route-surface-check.sh
```

## Expected outcome

- `dense-urban`:
  - `RIDE` without `ROUTE_TYPE_FALLBACK`,
  - `GRAVEL` and `MTB` with `ROUTE_TYPE_FALLBACK`,
  - path ratio remains `0%` on top route.
- `mixed-urban-paths`:
  - `RIDE` and `MTB` without `ROUTE_TYPE_FALLBACK`,
  - `GRAVEL` with `ROUTE_TYPE_FALLBACK`,
  - MTB surface fitness significantly lower than RIDE/GRAVEL fallback path.
