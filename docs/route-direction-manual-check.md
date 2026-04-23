# Route direction check (global heading matrix)

This guide validates `ROUTE-P0-03` on real target-generation responses.

Validated rules:

- generation succeeds with and without `startDirection`,
- when direction is explicitly honored (`Direction: ...`), top-route alignment stays high,
- when exact direction cannot be retained, fallback is explicit (`DIRECTION_RELAXED` or `DIRECTION_BEST_EFFORT`),
- each reference scenario keeps at least one explicit direction match.

## Prerequisites

- OSRM is running and reachable.
- Go backend is running from current source (`back-go`), default URL `http://127.0.0.1:8080`.

## Quick run

```bash
./scripts/manual-route-direction-check.sh
```

The script:

- checks `/api/health/details`,
- runs two target scenarios (`dense-urban`, `rural-peri`),
- runs a direction matrix per scenario (`NONE`, `N`, `E`, `S`, `W`),
- requires at least one returned route per run,
- checks top-route direction behavior:
  - if `Direction: ...` is present, it must match requested direction and respect alignment floor,
  - otherwise, requires explicit fallback diagnostics (`DIRECTION_RELAXED`/`DIRECTION_BEST_EFFORT`) or `Direction relaxed: ...` reason,
- requires at least one explicit direction match per scenario,
- prints compact JSON summaries.

## Useful overrides

```bash
BACKEND_URL=http://127.0.0.1:8080 \
ACTIVITY_TYPE=Ride \
VALIDATION_YEAR=1990 \
ROUTE_TYPE=RIDE \
VARIANT_COUNT=6 \
MIN_ALIGNMENT_PERCENT=85 \
MIN_EXPLICIT_DIRECTION_MATCHES=1 \
./scripts/manual-route-direction-check.sh
```

You can also override scenario targets:

```bash
DENSE_DISTANCE_KM=40 \
DENSE_ELEVATION_M=800 \
RURAL_DISTANCE_KM=30 \
RURAL_ELEVATION_M=350 \
./scripts/manual-route-direction-check.sh
```

## Expected outcome

- Both scenarios return routes for every direction mode.
- Directional requests are either:
  - explicitly matched on top route with alignment `>= MIN_ALIGNMENT_PERCENT`,
  - or explicitly relaxed via diagnostics/reasons.
- Each scenario has at least `MIN_EXPLICIT_DIRECTION_MATCHES` explicit direction matches.
