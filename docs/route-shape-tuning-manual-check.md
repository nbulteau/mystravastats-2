# Route shape tuning check (dense + rural)

This guide validates `ROUTE-P1-04` shape tuning on real routing responses:

- dense urban scenario should keep a shape-faithful strategy (`projected waypoints`)
- rural/peri-rural scenario should keep a road-routability strategy (`road-first anchors`)

## Prerequisites

- OSRM is running and reachable.
- Go backend is running from current source (`back-go`), default URL `http://127.0.0.1:8080`.

## Quick run

```bash
./scripts/manual-route-shape-tuning-check.sh
```

The script:

- checks `/api/health/details`,
- runs two `POST /api/routes/generate/shape` scenarios,
- enforces that each scenario returns at least one route,
- enforces expected strategy mode per scenario,
- prints compact JSON summaries (distance/elevation/shape score/modes).

## Useful overrides

```bash
BACKEND_URL=http://127.0.0.1:8080 \
ACTIVITY_TYPE=Ride \
VALIDATION_YEAR=1990 \
ROUTE_TYPE=RIDE \
VARIANT_COUNT=12 \
./scripts/manual-route-shape-tuning-check.sh
```

You can also override expected modes if the tuning target changes:

```bash
DENSE_EXPECTED_MODE="projected waypoints" \
RURAL_EXPECTED_MODE="road-first anchors" \
./scripts/manual-route-shape-tuning-check.sh
```

## Expected outcome

- Dense scenario: at least one generated route, expected mode present.
- Rural/peri-rural scenario: at least one generated route, expected mode present.
- No `NO_CANDIDATE` for the selected reference shapes.
