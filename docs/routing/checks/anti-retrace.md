# Route anti-retrace check (outside start zone 2 km)

This guide validates `ROUTE-P0-02` constraints on real target-generation responses.

Validated hard rules outside start/finish zone:

- max axis reuse remains `<= 1x`
- opposite-axis overlap remains `0%`

## Prerequisites

- OSRM is running and reachable.
- Backend is running from current source, default URL `http://127.0.0.1:8080`.

## Quick run

```bash
./scripts/manual-route-anti-retrace-check.sh
```

The script:

- checks `/api/health/details`,
- runs two target scenarios (`dense-urban`, `rural-peri`),
- requires at least one generated route per scenario,
- checks every returned route for:
  - `Max axis reuse outside start zone: Xx (limit 1x)` with `X <= 1`
  - `Opposite-axis overlap outside start zone: Y% (limit 0%)` with `Y == 0`
- prints compact scenario summaries.

## Useful overrides

```bash
BACKEND_URL=http://127.0.0.1:8080 \
ACTIVITY_TYPE=Ride \
VALIDATION_YEAR=1990 \
ROUTE_TYPE=RIDE \
VARIANT_COUNT=6 \
./scripts/manual-route-anti-retrace-check.sh
```

You can also override scenario targets:

```bash
DENSE_DISTANCE_KM=40 \
DENSE_ELEVATION_M=800 \
RURAL_DISTANCE_KM=30 \
RURAL_ELEVATION_M=350 \
./scripts/manual-route-anti-retrace-check.sh
```

## Expected outcome

- Both scenarios return routes.
- For every returned route:
  - outside-start axis reuse does not exceed `1x`
  - opposite overlap outside start zone stays `0%`.
