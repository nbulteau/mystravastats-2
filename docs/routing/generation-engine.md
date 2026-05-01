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

Public `POST /api/routes/generate/shape` responses are stricter than the internal explorer result: they only return OSRM candidates generated from the drawing (`Shape mode:*`). If OSRM cannot produce such a candidate, the response is empty with diagnostics instead of substituting an old activity.

Routing strategy parity (Go/Kotlin):

- `shape-first`: projected shape waypoints (high geometry fidelity)
- `road-first`: compact road anchors from the projected shape (better routability in sparse/complex areas)

Hard anti-backtracking rules remain owned by the routing engine:

- reject opposite traversal on the same axis
- reject candidates when axis reuse exceeds hard caps
- keep the 2 km start/finish tolerance behavior explicit

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
- low-similarity shape-mode candidates are rejected before selection instead of being shown with a flattering `Art fit`.

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
- strategy scoring includes an adaptive low-similarity drift penalty (`road-first` stricter than `shape-first`) so highly off-shape routes are naturally deprioritized
- both strategies are scored, deduplicated by geometry, then merged into the generated route payload

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

for candidate in publicCandidates:
  computeMetrics(candidate)
  computeScores(candidate)

publicCandidates = dedupeByGeometry(publicCandidates)
candidates = sortBySelectionPriority(candidates)

selected = candidates.take(limit)
return selected + diagnostics
```

## Acceptance Targets

- Strava Art generation returns a practicable road-snapped route for drawn/GPX/polyline inputs in dense-area local tests
- anti-backtracking remains strongly constrained outside start/finish tolerance zone
- route-type behavior remains meaningfully distinct (`Ride` vs `Gravel` vs `MTB`)
- parity checks remain mandatory across Go/Kotlin

## Related Docs

- [OSRM Setup](./osrm-setup.md)
- [Manual Route Checks](./manual-checks.md)
- [Main project doc](../README.md)
