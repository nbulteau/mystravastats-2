# Module Boundaries and Decomposition Plan

## Enforced boundaries

`node scripts/check-architecture.mjs` enforces these rules:

- Kotlin domain code cannot import `api` or `adapters`; outer configuration injects domain ports.
- Go `domain` and `application` packages cannot import an `infrastructure` package.
- Vue views, components and stores cannot call `fetch` or embed public API paths directly.
- Existing modules above 1,000 lines cannot grow beyond their recorded ceiling, and no new production module may cross 1,000 lines.

The module-size baseline is a ratchet, not a target. When a module is reduced, lower its ceiling. Remove its baseline entry after it falls below 1,000 lines.

## Decomposition sequence

1. **Route engines:** shape parsing, shape scoring, constraints, surface scoring,
   history bias, profile detection and shared transport models now live in
   focused Go/Kotlin modules. Continue by extracting OSRM request orchestration
   and candidate exploration from the remaining adapters. Preserve shared
   fixtures and parity at every extraction.
2. **Diagnostics UI:** formatting and normalization now live behind a typed
   service and presentation styles are isolated. Continue by splitting source
   configuration, runtime health and data-quality correction workflows into
   feature components and composables.
3. **Activity and route views:** activity power analysis, GPS Art presentation,
   browser geolocation, the activity hero and view styles have been extracted.
   Continue with map lifecycle, chart lifecycle, route editing and the remaining
   comparison panels.
4. **Backend conversion layers:** group DTO converters by capability and keep controllers limited to validation, use-case invocation and response mapping.
5. **Remaining providers:** isolate parsing, storage and enrichment so FIT, GPX, Strava and composite providers depend on small ports.

Each extraction must reduce or preserve its ratchet ceiling, add focused tests and avoid mixing behavior changes with file movement.
