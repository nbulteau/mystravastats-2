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
   history bias, profile detection, shared transport models and OSRM HTTP clients
   now live in focused Go/Kotlin modules. Continue by extracting candidate
   exploration from the remaining adapters. Preserve shared fixtures and parity
   at every extraction.
2. **Diagnostics UI:** formatting, normalization and data-quality presentation
   now live behind typed services and presentation styles are isolated. Continue by splitting source
   configuration, runtime health and data-quality correction workflows into
   feature components and composables.
3. **Activity and route views:** activity power analysis and presentation, GPS
   Art presentation and PNG export, browser geolocation, the activity hero and
   view styles have been extracted.
   Continue with map lifecycle, chart lifecycle, route editing and the remaining
   comparison panels.
4. **Backend conversion layers:** group DTO converters by capability and keep controllers limited to validation, use-case invocation and response mapping.
5. **Remaining providers:** isolate parsing, storage and enrichment so FIT, GPX, Strava and composite providers depend on small ports.

Each extraction must reduce or preserve its ratchet ceiling, add focused tests and avoid mixing behavior changes with file movement.

## Refactoring stabilization

The first stabilization pass now protects the extracted Vue power-analysis,
diagnostics-formatting, route-presentation and geolocation modules with direct
unit tests. The activity hero is covered through server-side component
rendering and the browser journey verifies the integrated activity screen.

Kotlin direction, anti-retrace, shape and surface tests call the extracted
`internal` routing functions and models directly. The temporary private
wrappers kept on `OsmRoutingEngineAdapter` for reflection-based tests have been
removed.

The second pass extracted OSRM transport from both route adapters, reduced the
three largest Vue views again and added focused tests for each new boundary.
OpenAPI now generates the shared route-generation, athlete-performance and
data-quality models used by the frontend.
