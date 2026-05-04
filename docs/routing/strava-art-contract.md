# Strava Art route contract

This document is the shared contract for the public Strava Art route flow in Go, Kotlin, and the Vue frontend.

Until `TECH-P1-01` centralizes the full OpenAPI source of truth, the contract is maintained as:

- OpenAPI fragment: [`../api/strava-art-routes.openapi.yaml`](../api/strava-art-routes.openapi.yaml)
- Go DTOs: `back-go/api/dto/route_generation.go`
- Kotlin DTOs: `back-kotlin/src/main/kotlin/me/nicolas/stravastats/api/dto/RouteRecommendationDto.kt`
- Front model: `front-vue/src/models/route-recommendation.model.ts`
- Shared fixtures: `test-fixtures/routes/strava-art-smoke.json` and `test-fixtures/routes/target-diagnostics-parity.json`

## Endpoints

`POST /api/routes/generate/shape`

- Input: drawn or imported artwork.
- Output: generated OSRM route proposals and diagnostics.
- `X-Request-Id` is accepted and returned. Failure diagnostics include the request id.
- Empty `routes` with diagnostics is a valid `200` response when no shape-mode road candidate exists.

`GET /api/routes/{routeId}/gpx`

- Exports a generated route from the short-lived generated-route cache.
- Returns `application/gpx+xml`.
- A missing cache entry returns `404`; callers should export soon after generation.

## Request

Required fields:

- `shapeInputType`: `draw`, `polyline`, `gpx`, or `svg`.
- `shapeData`: non-empty raw payload.

Optional fields:

- `startPoint`: preferred anchor `{ "lat": number, "lng": number }`. This is a placement hint, not a hard contour start for closed sketches.
- `routeType`: `RIDE`, `MTB`, `GRAVEL`, `RUN`, `TRAIL`, or `HIKE`; invalid or missing values fall back to `RIDE`.
- `variantCount`: `1..24`; missing values use backend defaults.

Strava Art does not accept public distance, elevation, duration, or direction targets. Those scores stay in the response for contract compatibility, but `score.shape` is the user-facing Art fit.

## Response

`routes` contains only generated shape-mode candidates:

- `variantType` is `SHAPE_MATCH` or `ROAD_GRAPH`.
- `reasons` must contain a `Shape mode:*` reason.
- historical activity candidates and remixes are not returned as public Strava Art proposals.
- ordering is `score.shape` first, then `score.global`, `score.roadFitness`, shorter distance, then stable `routeId`.

`GeneratedRoute` fields are stable across Go and Kotlin:

- identity: `routeId`, `title`, `variantType`, `routeType`, `activityId`
- metrics: `distanceKm`, `elevationGainM`, `durationSec`, `estimatedDurationSec`
- scores: `score.global`, `score.distance`, `score.elevation`, `score.duration`, `score.direction`, `score.shape`, `score.roadFitness`
- geometry: `previewLatLng`, optional `start`, optional `end`
- diagnostics context: `reasons`, `isRoadGraphGenerated`

## Art Fit First

Strava Art is optimized for drawing resemblance before sport-loop novelty.

- Retracing or opposite traversal is allowed when it preserves the user model.
- Retrace remains visible through route reasons and diagnostics as rideability context.
- Classic route explorer and sport-loop generation keep strict anti-retrace checks outside the start/finish zone.
- The backend diagnostic for this public policy is `ART_FIT_RETRACE_ALLOWED`.

## Diagnostics

Failure diagnostics:

- `NO_CANDIDATE`: no generated shape-mode route matched.
- `NON_SHAPE_CANDIDATES_IGNORED`: historical/internal candidates existed but were intentionally not returned.
- `FAILURE_SUMMARY`: compact failure summary with route type, inferred shape, input type, and request id.

Successful non-blocking diagnostics:

- `ART_FIT_RETRACE_ALLOWED`: drawing resemblance won; overlap/retrace is rideability context.
- `BACKTRACKING_RELAXED`: anti-backtracking constraints were softened to keep a route.
- `DIRECTION_RELAXED`, `DIRECTION_BEST_EFFORT`: heading constraints were softened internally.
- `ROUTE_TYPE_FALLBACK`: requested route type was adjusted.
- `START_POINT_SNAPPED`: start point moved to a routable point.
- `ENGINE_FALLBACK_LEGACY`, `ENGINE_CACHE_FALLBACK`: fallback engine/source was used.
- `SELECTION_RELAXED`, `EMERGENCY_FALLBACK`: selection rules were softened.

Diagnostics are parity-tested by both backends through `test-fixtures/routes/target-diagnostics-parity.json`.

## Smoke Checks

Backend/unit parity:

```bash
cd back-go && GOCACHE=/tmp/mystravastats-go-build go test ./api ./internal/routes/...
cd back-kotlin && ./gradlew test --tests me.nicolas.stravastats.api.controllers.RoutesControllerTest --tests me.nicolas.stravastats.api.controllers.RouteDiagnosticsParityFixtureTest --tests 'me.nicolas.stravastats.domain.services.routing.*'
```

OSRM-backed local smoke:

```bash
BACKEND_URL=http://127.0.0.1:8080 ./scripts/smoke-strava-art.sh
```

Docker stack smoke with Strava Art enabled:

```bash
RUN_STRAVA_ART_SMOKE=1 ./scripts/smoke-docker-compose.sh go
RUN_STRAVA_ART_SMOKE=1 ./scripts/smoke-docker-compose.sh kotlin
```

The smoke fixture covers Heart, Circle, Star, and Square without personal data.
