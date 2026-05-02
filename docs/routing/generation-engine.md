# Route Generation Engine (Unified Spec)

This document is the single source of truth for route generation behavior in both backends (Go and Kotlin).

The public generator is now the Strava Art studio: one Draw art workflow that snaps a drawn/imported shape to routable roads.

## Scope

Covered endpoints:

- `POST /api/routes/generate/shape`
- `GET /api/routes/{routeId}/gpx`

Current runtime behavior:

- Draw art is the only public route generation mode
- distance/elevation/direction targets are not part of the Strava Art request contract
- shape generation keeps parity between Go and Kotlin
- Strava Art optimizes drawing resemblance first; retracing is allowed when it materially improves the match to the user model
- parity objective between Go and Kotlin remains mandatory
- history-profile indexing (step 1) is available behind feature flags and propagated to the routing engine request

History-profile feature flags:

- `OSM_ROUTING_HISTORY_BIAS_ENABLED` (default `false`)
- `OSM_ROUTING_HISTORY_HALF_LIFE_DAYS` (default `75`)

## Inputs And Normalization

Strava Art input is normalized as follows:

- `shapeInputType`: `draw|polyline|gpx|svg`
- `shapeData`: required non-empty payload
- `startPoint`: optional preferred anchor point
- `routeType`: `RIDE|MTB|GRAVEL|RUN|TRAIL|HIKE` (fallback `RIDE`)
- `variantCount`: clamped to safe max
- shape inference accepts JSON coordinates and encoded polyline strings

## Route Type Matrix

`routeType` drives the route intent:

1. OSRM profile selection
2. surface fitness scoring
3. cache fallback scoring profile
4. returned activity tag

| Route type | OSRM profile | OSRM scoring weights (Distance / Elevation / Direction / Diversity) | Min segment diversity | In-cache scoring (Distance / Elevation / Duration) | Returned tag |
|---|---|---:|---:|---:|---|
| `RIDE` | `cycling` | `0.58 / 0.30 / 0.08 / 0.04` | `0.45` | `0.52 / 0.30 / 0.18` | `Ride` |
| `MTB` | `cycling` | `0.48 / 0.38 / 0.09 / 0.05` | `0.40` | `0.44 / 0.39 / 0.17` | `MountainBikeRide` |
| `GRAVEL` | `cycling` | `0.54 / 0.33 / 0.08 / 0.05` | `0.42` | `0.48 / 0.34 / 0.18` | `GravelRide` |
| `RUN` | `walking` | `0.55 / 0.20 / 0.15 / 0.10` | `0.35` | `0.45 / 0.22 / 0.33` | `Run` |
| `TRAIL` | `walking` | `0.42 / 0.33 / 0.15 / 0.10` | `0.30` | `0.36 / 0.40 / 0.24` | `TrailRun` |
| `HIKE` | `walking` | `0.34 / 0.41 / 0.15 / 0.10` | `0.28` | `0.30 / 0.45 / 0.25` | `Hike` |

The numeric columns still describe the internal route explorer and routing engine scoring profiles. Public Strava Art generation does not ask the user for distance, elevation, or direction targets.

## Strava Art Generator Algorithm

### 1. Validate request

- validate shape payload and optional start point
- normalize route type and variant count
- reject empty/unsupported shape inputs

### 2. Parse and infer shape

- parse drawn JSON coordinates, wrapped coordinate objects, encoded polylines, or GPX points
- infer shape family (`LOOP`, `OUT_AND_BACK`, `POINT_TO_POINT`) when coordinates are available
- keep the raw shape payload so both backends can project/snap the same input

### 3. Snap artwork to roads

The shape endpoint asks the route explorer for shape-aware candidates:

- `shapeMatches`: OSRM shape-mode candidates generated from the drawing; cache-derived shape matches remain internal explorer data
- `roadGraphLoops`: target/road-graph candidates used by the explorer, not public Strava Art replacements unless they are explicitly shape-mode generated
- `closestLoops`: historical candidates kept for recommendations and diagnostics, not returned as Strava Art proposals
- `shapeRemixes`: historical remix candidates for shape composition

Public `POST /api/routes/generate/shape` responses are stricter than the internal explorer result: they only return OSRM candidates generated from the drawing (`Shape mode:*`). If the strict strategies fail, shape-mode best-effort strategies may still return a low-confidence OSRM route with a weak `Art fit` and explicit fallback reasons; historical activities are not substituted as Strava Art proposals.

Routing strategy parity (Go/Kotlin):

- `nearest-road trace`: snap sampled drawing points to their nearest OSRM-routable anchors, route each anchor-to-anchor segment with OSRM, then stitch the routed geometries; this is the preferred GPS drawing strategy when it stays close enough to the road graph
- `segment stitched alternatives`: route each drawing segment with OSRM alternatives and stitch the best segment matches together
- `dense sketch anchors`: denser projected shape waypoints for simple forms that need more contour fidelity
- `map sketch waypoints`: projected shape waypoints (high geometry fidelity)
- `simplified sketch anchors`: reduced projected shape waypoints for better routability
- `road-first`: compact road anchors from the projected shape (better routability in sparse/complex areas)
- best-effort shape fallbacks: simplified/envelope variants returned only when normal shape strategies cannot provide enough candidates

Retrace policy is mode-specific:

- for classic sport loops and the internal explorer, hard anti-backtracking rules remain owned by the routing engine
- for public Strava Art, `Art fit` wins over novelty: opposite traversal and axis reuse may be accepted when they preserve the drawing
- Strava Art should still expose retrace/backtracking diagnostics as rideability signals, not as automatic rejection reasons
- keep the 2 km start/finish tolerance behavior explicit for classic route-generation checks

### 4. Score each candidate

Main Strava Art response scores:

- `global`: backend match score
- `shape`: anchored shape match score when available
- `roadFitness`: surface mix fitness (`paved/gravel/trail/unknown`)
- `distance`, `elevation`, `duration`, `direction`: mirror `global` because those are no longer user constraints for Strava Art

Shape-mode scoring is stricter than generic route scoring:

- contour similarity after normalization checks the overall form,
- anchored proximity checks the route against the projected sketch in real map space,
- ordered path similarity penalizes routes that touch similar areas in the wrong sequence,
- centroid drift penalizes candidates shifted away from the drawing,
- low-similarity shape-mode candidates must not receive a flattering `Art fit`; they can still be returned as weak proposals with explicit diagnostics when they are the best road-snapped drawing match.

Surface scoring signals (Go + Kotlin parity):

- primary source: `steps[].classes` + travel `mode`
- tag-aware enrichment when available: `surface` / `tracktype` (direct fields or class tokens such as `surface=asphalt`, `surface:fine_gravel`, `tracktype=grade3`)
- `tracktype` mapping:
  - `grade1` -> paved
  - `grade2/grade3` -> gravel
  - `grade4/grade5` -> trail
- route-type intent:
  - `Ride`: prioritize paved
  - `Gravel`: require at least 25% path ratio (`gravel + trail`), then prefer higher path share
  - `MTB`: strongly reward path-heavy candidates
  - fallback route type policy remains `MTB -> GRAVEL -> RIDE` and `GRAVEL -> RIDE` when constraints cannot be satisfied

Scores:

- `matchScore` for user-facing ranking
- `effectiveMatchScore` for stronger anti-overlap penalties during selection

### 5. Deduplicate by geometry

- build normalized geometry signatures from sampled points
- collapse reverse-equivalent geometry
- keep one candidate per signature

### 6. Diagnostics

- return generated routes with scores and reasons
- return non-blocking fallback/relaxation diagnostics when relevant, including on success
- return `NO_CANDIDATE` and `FAILURE_SUMMARY` when no route can be produced

Examples of success diagnostics:

- `DIRECTION_RELAXED`
- `DIRECTION_BEST_EFFORT`
- `BACKTRACKING_RELAXED`
- `ROUTE_TYPE_FALLBACK`
- `START_POINT_SNAPPED`
- `ENGINE_FALLBACK_LEGACY`
- `SELECTION_RELAXED`
- `EMERGENCY_FALLBACK`

## Shape Payload Notes

- accepts `draw|polyline|gpx|svg`
- keeps raw shape payload for shape projection/routing
- infers shape family (`LOOP`, `OUT_AND_BACK`, `POINT_TO_POINT`) when coordinates are available
- coordinate parsing supports:
  - JSON array of `[lat, lng]`
  - wrapped JSON fields (`points`, `coordinates`, `latLng`)
  - encoded polyline string
  - GPX points (`trkpt`, `rtept`, `wpt`) when payload is GPX XML
- strategy scoring includes an adaptive low-similarity drift penalty (`road-first` stricter than shape-first strategies) so highly off-shape routes are naturally deprioritized
- closed sketches are generated from several contour anchors and scored with a contour-start invariant ordered score: the requested start point is only a placement hint, not a reason to prefer a visibly worse drawing
- shape candidates are scored, deduplicated by geometry, then selected by `Art fit` first before the strictest compatible relaxation profile is attached
- the `nearest-road trace` strategy is explicit about its trade-off: it preserves the drawing order with nearby routable anchors, but the exported geometry always comes from OSRM-routed segments instead of straight lines between anchors
- if no strict/balanced/relaxed shape candidate survives, an `Art fit` oriented soft fallback is tried before the absolute emergency fallback

## History Profile (Step 1)

When enabled, both backends build a local history profile from cached activities before calling the routing engine.

Profile content:

- route type normalized to `RIDE|MTB|GRAVEL|RUN|TRAIL|HIKE`
- weighted axis scores (undirected segment key)
- weighted zone scores (coarse geographic buckets)
- activity count, segment count, latest activity timestamp

Weighting model:

- contribution is proportional to segment length
- recency decay uses an exponential half-life (`OSM_ROUTING_HISTORY_HALF_LIFE_DAYS`)

Current usage status:

- profile is computed and propagated to the engine request in Go and Kotlin
- anchor/edge scoring bias integration is intentionally left to step 2

## Routing Health And Degraded Modes

Routing health is exposed by `/api/health/details`:

- `up`: OSRM available
- `down`: configured but unreachable
- `misconfigured`: enabled but invalid/missing base URL
- `disabled`: routing intentionally disabled

Behavior:

- when routing is up: OSRM shape-mode generation executes from the drawing
- when routing is degraded: no public Strava Art route is returned; historical candidates stay available only to the explorer/recommendation layer
- partial per-call failures remain non-fatal when at least one valid candidate survives

## Simplified Pseudocode

```text
normalize(request)
profile = profileFromRouteType(routeType)
shape = parseShape(shapeInputType, shapeData)
shapeFamily = inferShapeFamily(shape)

explorerCandidates = shapeMatches(shape, profile)
explorerCandidates += roadGraphShapeRoutes(shape, profile)
explorerCandidates += closestHistoricalRoutes(shapeFamily, profile)
explorerCandidates += shapeRemixes(shape, profile)

publicCandidates = onlyOSRMShapeMode(explorerCandidates)
publicCandidates += nearestRoadTrace(shape, profile)
publicCandidates += stitchedSegmentAlternatives(shape, profile)

for candidate in publicCandidates:
  computeMetrics(candidate)
  computeScores(candidate)

publicCandidates = dedupeByGeometry(publicCandidates)
candidates = sortByArtFitThenRouteQuality(publicCandidates)
selected = selectStrictestCompatibleProfiles(candidates)

if selected is not full:
  selected += artFitFirstSoftFallback(candidates)

return selected + diagnostics
```

## Acceptance Targets

- Strava Art generation returns a practicable road-snapped route for drawn/GPX/polyline inputs in dense-area local tests
- Strava Art may return retracing routes when they better preserve the drawing, with clear rideability diagnostics
- classic route-generation and explorer anti-backtracking checks remain strongly constrained outside the start/finish tolerance zone
- route-type behavior remains meaningfully distinct (`Ride` vs `Gravel` vs `MTB`)
- parity checks remain mandatory across Go/Kotlin

## Related Docs

- [OSRM Setup](./osrm-setup.md)
- [Manual Route Checks](./manual-checks.md)
- [Main project doc](../README.md)
