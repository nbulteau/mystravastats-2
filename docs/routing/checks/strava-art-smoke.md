# Strava Art smoke check

This check validates the public MVP flow against a running backend:

- call `POST /api/routes/generate/shape`,
- require at least one generated route,
- export the selected route through `GET /api/routes/{routeId}/gpx`,
- verify that the exported payload is valid GPX-shaped XML with track points.
- run the same flow for the built-in Heart, Circle, Star, and Square sketches.

## Prerequisites

- OSRM is running and reachable.
- Backend is running from current source, default URL `http://localhost:8080`.

## Quick run

```bash
./scripts/manual-strava-art-smoke-check.sh
```

## Useful overrides

```bash
BACKEND_URL=http://localhost:8090 \
ACTIVITY_TYPE=Ride \
ROUTE_TYPE=RIDE \
VARIANT_COUNT=3 \
START_LAT=45.18 \
START_LNG=5.72 \
SHAPE_DATA='[[45.18,5.72],[45.19,5.745],[45.205,5.725],[45.18,5.72]]' \
./scripts/manual-strava-art-smoke-check.sh
```

To run a different smoke matrix, point `FIXTURE_PATH` to a JSON file with a
`cases` array using the same fields as `test-fixtures/routes/strava-art-smoke.json`.

## Expected outcome

- The shape generation endpoint returns at least one route.
- The first route has a `routeId`.
- The generated route GPX endpoint returns a GPX payload containing track points.
- A low `artFit` score is acceptable for hard road layouts, but a missing route
  is a regression for these simple shapes.
