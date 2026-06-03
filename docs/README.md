# My Activity Stats Documentation

My Activity Stats is a personal analytics application for Strava and local activity files. It lets you explore activities, dashboards, charts, heatmaps, maps, badges, best efforts, personal records, gear data, and GPS Art route drawing.

This directory is organized by intent: start here, then jump to the page matching the work you are doing.

## Start Here

- [Quick Start](./getting-started/quick-start.md) - build and run the application.
- [Developer Setup](./getting-started/developer-setup.md) - local development, Docker stacks, toolchains, and checks.
- [Strava OAuth Setup](./data-sources/strava-oauth.md) - configure `.strava` and first synchronization.
- [Screenshots](./reference/screenshots.md) - current UI reference images.

## Architecture

- [Architecture Overview](./architecture/overview.md) - high-level components and request flow.
- [Backend Capability Matrix](./architecture/backend-capability-matrix.md) - what Go and Kotlin currently support.
- [Runtime Configuration](./architecture/runtime-config.md) - environment variables exposed by diagnostics.
- [Cache Layout](./architecture/cache-layout.md) - on-disk Strava cache structure.

## Data Sources

- [Strava OAuth Setup](./data-sources/strava-oauth.md)
- [OAuth and Cache Troubleshooting](./data-sources/strava-oauth-troubleshooting.md)
- [FIT and GPX Sources](./data-sources/fit-gpx.md)

### Current Source Model

Both Go and Kotlin now support the same source modes:

- `STRAVA`: Strava cache and optional Strava API refresh.
- `FIT`: local FIT files grouped by year.
- `GPX`: local GPX files grouped by year.
- `composite`: automatic mixed mode when two or more sources are explicitly configured.

Composite mode currently supports `Strava + FIT + GPX`. RideWithGPS and TCX are planned, but not implemented yet. When composite mode is active, `/api/health/details` reports `provider=composite`, lists `activeProviders`, and exposes merge diagnostics. The Status page at `http://localhost:8080/diagnostics` renders these details in the `Data Source` section.

The `Data Source` section can check a Strava cache, FIT directory or GPX directory, then persist the selected source with `Use this source`. This writes the matching key to the backend working-directory `.env` file and preserves unrelated settings. The running provider is not hot-reloaded: restart the backend with the usual command, then use `Verify active mode`.

For FIT, the Go local backend also exposes a `Synchronize` action on the Status
page. It can copy new Garmin USB FIT files from `GARMIN_FIT_SOURCE_PATH` or an
auto-detected `/Volumes/.../GARMIN/ACTIVITY` directory into
`FIT_FILES_PATH/<year>/`, then reload the active FIT/composite provider.

Source selection is automatic:

- with no explicit local source, the backend uses the default Strava provider and `strava-cache`;
- with exactly one configured source (`STRAVA_CACHE_PATH`, `FIT_FILES_PATH`, or `GPX_FILES_PATH`), that provider stays exclusive;
- with two or more configured sources, the backend switches to the composite provider.

Composite matching keeps the existing source caches unchanged. If a local FIT/GPX activity matches a Strava activity, the Strava activity ID and metadata remain canonical. Local streams can enrich the composite view, and local activities without a Strava match remain visible in union mode.

Recommended priority for analytics:

- use Strava for metadata, names, sport type, gear, and social/API-linked fields;
- prefer FIT for serious sport data, especially heart rate, cadence, power, timing, and device-recorded streams;
- use GPX as a simple, portable fallback for GPS traces and manual imports;
- choose fields source-by-source instead of treating one format as globally better.

In practice, the best default setup is `STRAVA_CACHE_PATH + FIT_FILES_PATH`, with `GPX_FILES_PATH` accepted as an additional import or fallback source. See [Runtime Configuration](./architecture/runtime-config.md) and [FIT and GPX Sources](./data-sources/fit-gpx.md) for details.

## Routing

- [OSRM Setup](./routing/osrm-setup.md)
- [Route Generation Engine](./routing/generation-engine.md) - GPS Art / Draw art route generation.
- [GPS Art Route Contract](./routing/strava-art-contract.md) - public GPS Art API contract and diagnostics.
- [Manual Route Checks](./routing/manual-checks.md)

## Reference

- [Statistics Reference](./reference/statistics.md)
- [Screenshots](./reference/screenshots.md)

## Planning

- [TODO](./TODO.md) (TODO is in French) remains at the docs root because project instructions refer to this exact path.

## Repository Layout

- [front-vue](../front-vue) - Vue 3 frontend.
- [back-kotlin](../back-kotlin) - Spring Boot + Kotlin backend.
- [back-go](../back-go) - Go backend and local binary packaging path.
- [scripts](../scripts) - local smoke checks, screenshots, and maintenance scripts.

The frontend talks to one backend through `/api/...`. Keep Go and Kotlin behavior aligned when a feature is shared by both backends.
