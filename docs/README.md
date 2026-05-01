# MyStravaStats Documentation

MyStravaStats is a personal analytics application for Strava and local activity files. It lets you explore activities, dashboards, charts, heatmaps, maps, badges, best efforts, personal records, gear data, and Strava Art route drawing.

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

## Routing

- [OSRM Setup](./routing/osrm-setup.md)
- [Route Generation Engine](./routing/generation-engine.md) - Strava Art / Draw art route generation.
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
