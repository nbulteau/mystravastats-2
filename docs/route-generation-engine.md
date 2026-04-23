# Route Generation Engine (Unified Spec)

This document is the single source of truth for route generation behavior in both backends (Go and Kotlin).

It merges the former engine matrix and the former v3 constraint-first spec.

## Scope

Covered endpoints:

- `POST /api/routes/generate/target`
- `POST /api/routes/generate/shape`

Current runtime behavior:

- v3 disjoint-anchor generation is enabled by default for target mode
- automatic fallback to legacy synthetic-waypoint generation when needed
- parity objective between Go and Kotlin remains mandatory
- history-profile indexing (step 1) is available behind feature flags and propagated to the routing engine request

History-profile feature flags:

- `OSM_ROUTING_HISTORY_BIAS_ENABLED` (default `false`)
- `OSM_ROUTING_HISTORY_HALF_LIFE_DAYS` (default `75`)

## Inputs And Normalization

Target input is normalized as follows:

- `routeType`: `RIDE|MTB|GRAVEL|RUN|TRAIL|HIKE` (fallback `RIDE`)
- `startDirection`: `N|S|E|W|UNDEFINED` (`UNDEFINED` means no global direction constraint)
- `generationMode`: `AUTOMATIC|CUSTOM`
- `distanceTargetKm`: required and `> 0`
- `elevationTargetM`: optional and `>= 0`
- `variantCount`: clamped to safe max

Shape input is normalized as follows:

- `shapeInputType`: `draw|polyline|gpx|svg`
- `shapeData`: required non-empty payload
- shape inference accepts JSON coordinates and encoded polyline strings

## Route Type Matrix

`routeType` drives five decisions:

1. OSRM profile selection
2. candidate scoring weights
3. anti-overlap/diversity thresholds
4. cache fallback scoring profile
5. returned activity tag

| Route type | OSRM profile | OSRM scoring weights (Distance / Elevation / Direction / Diversity) | Min segment diversity | In-cache scoring (Distance / Elevation / Duration) | Returned tag |
|---|---|---:|---:|---:|---|
| `RIDE` | `cycling` | `0.58 / 0.30 / 0.08 / 0.04` | `0.45` | `0.52 / 0.30 / 0.18` | `Ride` |
| `MTB` | `cycling` | `0.48 / 0.38 / 0.09 / 0.05` | `0.40` | `0.44 / 0.39 / 0.17` | `MountainBikeRide` |
| `GRAVEL` | `cycling` | `0.54 / 0.33 / 0.08 / 0.05` | `0.42` | `0.48 / 0.34 / 0.18` | `GravelRide` |
| `RUN` | `walking` | `0.55 / 0.20 / 0.15 / 0.10` | `0.35` | `0.45 / 0.22 / 0.33` | `Run` |
| `TRAIL` | `walking` | `0.42 / 0.33 / 0.15 / 0.10` | `0.30` | `0.36 / 0.40 / 0.24` | `TrailRun` |
| `HIKE` | `walking` | `0.34 / 0.41 / 0.15 / 0.10` | `0.28` | `0.30 / 0.45 / 0.25` | `Hike` |

Dynamic redistribution:

- without elevation target, elevation weight is redistributed mostly to distance then diversity
- without direction target, direction weight is redistributed mostly to distance then diversity

## Target Generator Algorithm

### 1. Validate request

- validate start point and numeric targets
- normalize route type, direction, generation mode, limits

### 2. Select routing profile

- `RUN/TRAIL/HIKE` -> walking profile
- `RIDE/MTB/GRAVEL` -> cycling profile

### 3. Generate loop candidates (v3 first)

#### 3.1 v3 disjoint-anchor generation

For each sampled anchor around the start point:

- compute outbound path `start -> anchor`
- compute inbound path `anchor -> start` with axis reuse penalties/constraints
- merge both into a loop candidate

Hard anti-backtracking rules during construction:

- reject opposite traversal on the same axis
- reject candidate when axis reuse exceeds hard cap
- keep closure-to-start behavior only for final return logic

State tracked per candidate includes:

- used axis counters
- opposite-axis usage flags
- distance/elevation aggregates
- direction dominance signals

#### 3.2 Legacy fallback generation

If v3 yields no accepted candidate:

- generate synthetic waypoints around start
- call OSRM with alternatives/full geometry
- convert routes to internal candidates

### 4. Score each candidate

Main metrics:

- distance delta ratio
- elevation delta ratio
- direction penalty (global orientation)
- backtracking ratio
- corridor overlap
- segment diversity
- surface mix fitness (`paved/gravel/trail/unknown`)

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

### 6. Progressive relaxation

Candidates are sorted by:

1. corridor overlap ascending
2. backtracking ratio ascending
3. effective score descending

Selection levels:

- `strict`
- `balanced`
- `relaxed`
- `fallback`

Relaxed dimensions:

- direction tolerance
- distance tolerance
- elevation tolerance

Never relaxed:

- opposite-axis traversal ban
- hard axis reuse cap

### 7. Return routes and diagnostics

- return generated routes with scores and reasons
- surface fallback/relaxation diagnostics when relevant, including on success

Examples of success diagnostics:

- `DIRECTION_RELAXED`
- `DIRECTION_BEST_EFFORT`
- `BACKTRACKING_RELAXED`
- `ROUTE_TYPE_FALLBACK`
- `START_POINT_SNAPPED`
- `ENGINE_FALLBACK_LEGACY`
- `SELECTION_RELAXED`
- `EMERGENCY_FALLBACK`

## Shape Generator Notes

For `POST /api/routes/generate/shape`:

- accepts `draw|polyline|gpx|svg`
- keeps raw shape payload for shape projection/routing
- infers shape family (`LOOP`, `OUT_AND_BACK`, `POINT_TO_POINT`) when coordinates are available
- coordinate parsing supports:
  - JSON array of `[lat, lng]`
  - wrapped JSON fields (`points`, `coordinates`, `latLng`)
  - encoded polyline string

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

- when routing is up: road-graph generation executes
- when routing is degraded: no road-graph result, historical fallback stays available
- partial per-call failures remain non-fatal when at least one valid candidate survives

## Simplified Pseudocode

```text
normalize(request)
profile = profileFromRouteType(routeType)

candidates = v3DisjointAnchorGenerate(start, target, profile, constraints)
if candidates is empty:
  candidates = legacySyntheticGenerate(start, target, profile)

for candidate in candidates:
  computeMetrics(candidate)
  computeScores(candidate)

candidates = dedupeByGeometry(candidates)
candidates = sortBySelectionPriority(candidates)

selected = selectWithRelaxationLevels(candidates, limit)
return selected + diagnostics
```

## Acceptance Targets

- target generation returns a practicable loop in most dense-area local tests
- anti-backtracking remains strongly constrained outside start/finish tolerance zone
- route-type behavior remains meaningfully distinct (`Ride` vs `Gravel` vs `MTB`)
- parity checks remain mandatory across Go/Kotlin

## Related Docs

- [OSM Routing Setup](./osm-routing-setup.md)
- [Main project doc](./README.md)
