# Strava Art fallback manual check (API + UI)

This guide validates that fallback diagnostics are visible both in API responses and in the UI for Strava Art generation.

## Prerequisites

- OSRM is running and reachable.
- Backend is running (Go or Kotlin).
- Front dev server is running for UI checks.

Typical local URLs:
- Backend: `http://127.0.0.1:8090`
- Front: `http://127.0.0.1:5173`

## API protocol

```bash
curl -sS -X POST "$BACKEND_URL/api/routes/generate/shape?activityType=$ACTIVITY_TYPE" \
  -H 'Content-Type: application/json' \
  -H 'X-Request-Id: route-fallback-manual' \
  -d '{
    "shapeInputType": "draw",
    "shapeData": "[[48.157563,-1.587309],[48.165000,-1.570000],[48.157563,-1.587309]]",
    "startPoint": {"lat": 48.157563, "lng": -1.587309},
    "routeType": "RUN",
    "variantCount": 2
  }' | jq '{routesCount: (.routes | length), diagnostics}'
```

## Useful overrides

```bash
BACKEND_URL=http://127.0.0.1:8096 \
FRONT_URL=http://127.0.0.1:5173 \
ACTIVITY_TYPE=Run \
ROUTE_TYPE=RUN \
VARIANT_COUNT=1 \
START_LAT=48.157563 \
START_LNG=-1.587309
```

## Expected outcome

- API returns at least one route.
- API diagnostics include at least one non-blocking fallback code such as:
  - `BACKTRACKING_RELAXED`
  - `ENGINE_FALLBACK_LEGACY`
  - `ROUTE_TYPE_FALLBACK`
  - `START_POINT_SNAPPED`
  - `SELECTION_RELAXED`
  - `EMERGENCY_FALLBACK`
- Strava Art routes may also include `ART_FIT_RETRACE_ALLOWED` when drawing resemblance wins and retrace is only rideability context.
- In the UI, a warning toast appears when such fallback diagnostics are returned.
- In the UI, diagnostics are visible under the generation panel even when routes are present.
