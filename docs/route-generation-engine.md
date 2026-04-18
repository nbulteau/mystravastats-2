# Route Generation Engine (Type Matrix)

This document explains how `routeType` is used by the route generation engine in both backends (Go and Kotlin).

It focuses on the **Target loop generator** behavior (`/api/routes/generate/target`) and the scoring/fallback logic.

## Accepted route types

Valid values:

- `RIDE`
- `MTB`
- `GRAVEL`
- `RUN`
- `TRAIL`
- `HIKE`

Invalid or missing values are normalized to `RIDE`.

## Where `routeType` is used

`routeType` is used in 5 places:

1. **OSRM profile selection**
2. **OSRM candidate scoring weights**
3. **Anti-overlap diversity thresholds**
4. **In-cache fallback scoring weights**
5. **Returned sport tag in API response**

## Algorithm description (Target loop generator)

This is the high-level algorithm used by `/api/routes/generate/target`.

### 1. Normalize and validate input

- Normalize `routeType` (`RIDE|MTB|GRAVEL|RUN|TRAIL|HIKE`, fallback `RIDE`).
- Normalize `startDirection` (`N|S|E|W` or empty).
- Validate start point and target distance.
- Normalize variant count / limit.

### 2. Select routing profile

- `RUN/TRAIL/HIKE` -> OSRM `walking` profile.
- `RIDE/MTB/GRAVEL` -> OSRM `cycling` profile.

### 3. Generate candidate loops from synthetic waypoint patterns

For each call iteration:

- Compute a loop radius from target distance.
- Apply rotation and radius multipliers to diversify geometry.
- Build synthetic waypoints around the start point (multiple shapes).
- Call OSRM route API with:
  - `alternatives=true`
  - `overview=full`
  - `geometries=geojson`
  - `continue_straight=true` (to reduce local U-turns at waypoints)
- Convert each OSRM route into an internal candidate.

### 4. Compute quality metrics per candidate

For each candidate route:

- **distanceDeltaRatio**: relative error to distance target.
- **directionPenalty**: heading + half-plane constraint from start direction.
- **backtrackingRatio**: opposite traversal of same edge.
- **corridorOverlap**: reuse of same road corridor (same axis, near-parallel/opposite).
- **segmentDiversity**: share of unique edges.
- **elevationEstimate**: estimated D+ from target constraints.
- **matchScore**: weighted score from distance/elevation/direction/diversity.
- **effectiveMatchScore**: internal score with stronger penalties for overlap/backtracking.

### 5. Deduplicate candidates

- Build a geometry signature from sampled polyline points.
- Keep only one candidate per signature.

### 6. Progressive relaxation selection

Candidates are sorted primarily by:

1. lowest corridor overlap
2. lowest backtracking ratio
3. highest effective score

Then selected through relaxation levels:

- `strict`
- `balanced`
- `relaxed`
- `fallback`

Each level applies limits on:

- max direction penalty
- max backtracking ratio
- max corridor overlap
- min segment diversity
- max distance delta ratio

If strict cannot fill requested routes, next levels progressively loosen constraints.

### 7. Return generated routes

- Return selected routes with reasons and score.
- Tag each route with the corresponding sport for the selected `routeType`.
- If no acceptable OSRM route is found, return empty generated set.

### 8. In-cache fallback (historical recommender)

In parallel to road-graph generation, the system keeps cache-based route recommendation logic.
Its scoring profile is also adjusted by `routeType` (distance/elevation/duration emphasis).

## OSRM accessible vs not accessible

### Health states

Routing health is exposed via `/api/health` (field `routing`), with:

- `status = up` and `reachable = true`: OSRM is available.
- `status = down`: OSRM endpoint is configured but unreachable/error.
- `status = misconfigured`: routing is enabled but base URL is missing/invalid.
- `status = disabled`: routing engine is disabled by configuration.

### Runtime behavior

- If OSRM is **up**:
  - road-graph candidates are generated and filtered by quality constraints.
- If OSRM is **down/misconfigured/disabled**:
  - road-graph generation returns no generated loop,
  - the app still returns cache-based recommendations (historical fallback),
  - startup and core statistics remain functional.

### Partial failures during generation

If some OSRM calls fail during one generation:

- failures are counted (`OSRM_CALL_FAILED`) and logged,
- other calls continue,
- if at least one valid candidate survives filtering, a route is returned,
- otherwise the generated route set is empty and fallback recommendations remain available.

## Pseudo-code (simplified)

```text
normalize(input)
profile = profileFromRouteType(routeType)
candidates = []

for call in routingBudget:
  waypoints = syntheticLoopWaypoints(start, targetDistance, direction, call)
  osrmRoutes = osrmRoute(profile, waypoints)
  for route in osrmRoutes:
    candidate = computeMetricsAndScores(route, routeType, constraints)
    if valid(candidate) and not duplicate(candidate):
      candidates.add(candidate)

sort candidates by (corridorOverlap, backtrackingRatio, -effectiveScore, ...)

selected = []
for level in [strict, balanced, relaxed, fallback]:
  selected += candidates that pass level thresholds
  stop when selected.size == limit

return selected
```

## Mini matrix by type

| Route type | OSRM profile | OSRM scoring weights (Distance / Elevation / Direction / Diversity) | Min segment diversity threshold | In-cache fallback scoring weights (Distance / Elevation / Duration) | Returned activity tag |
|---|---|---:|---:|---:|---|
| `RIDE` | `cycling` | `0.58 / 0.30 / 0.08 / 0.04` | `0.45` | `0.52 / 0.30 / 0.18` | `Ride` |
| `MTB` | `cycling` | `0.48 / 0.38 / 0.09 / 0.05` | `0.40` | `0.44 / 0.39 / 0.17` | `MountainBikeRide` |
| `GRAVEL` | `cycling` | `0.54 / 0.33 / 0.08 / 0.05` | `0.42` | `0.48 / 0.34 / 0.18` | `GravelRide` |
| `RUN` | `walking` | `0.55 / 0.20 / 0.15 / 0.10` | `0.35` | `0.45 / 0.22 / 0.33` | `Run` |
| `TRAIL` | `walking` | `0.42 / 0.33 / 0.15 / 0.10` | `0.30` | `0.36 / 0.40 / 0.24` | `TrailRun` |
| `HIKE` | `walking` | `0.34 / 0.41 / 0.15 / 0.10` | `0.28` | `0.30 / 0.45 / 0.25` | `Hike` |

## Notes on dynamic weighting

Two dynamic adjustments are applied:

- If no elevation target is provided, elevation weight is redistributed mostly to distance and partly to diversity.
- If no departure direction is provided, direction weight is redistributed mostly to distance and partly to diversity.

So, the matrix above reflects the **base profile** before dynamic redistribution.

## Practical interpretation

- `RIDE`/`GRAVEL`: prioritize distance fit and smooth road-like loops.
- `MTB`: give more importance to elevation and terrain variation.
- `RUN`: prioritize distance + duration regularity, with stronger directional sensitivity.
- `TRAIL`/`HIKE`: prioritize elevation and direction coherence over pure distance matching.

## Related docs

- [OSM Routing Setup](./osm-routing-setup.md)
- [Main project doc](./README.md)
