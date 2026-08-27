# Module Boundaries and Decomposition Plan

## Enforced boundaries

`node scripts/check-architecture.mjs` enforces these rules:

- Kotlin domain code cannot import `api` or `adapters`; outer configuration injects domain ports.
- Go `domain` and `application` packages cannot import an `infrastructure` package.
- Vue views, components and stores cannot call `fetch` or embed public API paths directly.
- Existing modules above 1,000 lines cannot grow beyond their recorded ceiling, and no new production module may cross 1,000 lines.

The module-size baseline is a ratchet, not a target. When a module is reduced, lower its ceiling. Remove its baseline entry after it falls below 1,000 lines.

## Decomposition sequence

1. **Route engines:** extract graph access, candidate exploration, scoring, geometry validation and diagnostics from the Go and Kotlin routing adapters. Preserve shared fixtures and parity at every extraction.
2. **Diagnostics UI:** split source configuration, runtime health, data-quality corrections and maintenance actions into feature components and composables.
3. **Activity and route views:** extract map lifecycle, charts, comparison logic, GPS Art drawing and route editing behind typed feature services.
4. **Backend conversion layers:** group DTO converters by capability and keep controllers limited to validation, use-case invocation and response mapping.
5. **Remaining providers:** isolate parsing, storage and enrichment so FIT, GPX, Strava and composite providers depend on small ports.

Each extraction must reduce or preserve its ratchet ceiling, add focused tests and avoid mixing behavior changes with file movement.
