# Route fallback manual check (API + UI)

This guide validates that fallback diagnostics are now visible both in API responses and in the UI even when route generation succeeds.

## Prerequisites

- OSRM is running and reachable.
- Backend is running (Go or Kotlin).
- Front dev server is running for UI checks.

Typical local URLs:
- Backend: `http://127.0.0.1:8090`
- Front: `http://127.0.0.1:5173`

## Quick run (API + UI protocol)

```bash
./scripts/manual-route-fallback-check.sh
```

The script:
- checks backend health,
- optionally resolves a start point from `ACTIVITY_ID`,
- calls `POST /api/routes/generate/target`,
- validates that at least one fallback diagnostic code is present,
- prints a short UI checklist.

## Useful overrides

```bash
BACKEND_URL=http://127.0.0.1:8096 \
FRONT_URL=http://127.0.0.1:5173 \
ACTIVITY_ID=1112134846 \
ACTIVITY_TYPE=Run \
ROUTE_TYPE=RUN \
START_DIRECTION=N \
DISTANCE_TARGET_KM=30 \
VARIANT_COUNT=1 \
./scripts/manual-route-fallback-check.sh
```

If you do not want activity lookup, force coordinates:

```bash
ACTIVITY_ID= \
START_LAT=48.157563 \
START_LNG=-1.587309 \
./scripts/manual-route-fallback-check.sh
```

## Expected outcome

- API returns at least one route.
- API diagnostics include at least one non-blocking fallback code such as:
  - `DIRECTION_RELAXED`
  - `DIRECTION_BEST_EFFORT`
  - `BACKTRACKING_RELAXED`
  - `ENGINE_FALLBACK_LEGACY`
- In the UI, a warning toast appears when such fallback diagnostics are returned.
- In the UI, diagnostics are visible under the generation panel even when routes are present.
