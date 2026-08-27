# Changelog

All notable changes to My Activity Stats are documented in this file.

## Unreleased

### Added

- GitHub Actions CI for Go, Kotlin, Vue, browser journeys, catalog parity, API contracts, and Docker images.
- Coverage regression gates for all three application stacks.
- Canonical OpenAPI inventory covering all 52 public backend operations.
- Generated TypeScript, Go, and Kotlin contract models sourced from OpenAPI.
- Playwright journeys for dashboards and activity details, source onboarding and synchronization, safe data corrections, and GPS Art generation.
- Architecture decision records, executable dependency rules, and a module-size ratchet.
- Local deployment security guidance, contributor workflow and a prioritized product roadmap.

### Changed

- Synchronized the Go, Kotlin, and Vue README files with the canonical setup,
  architecture, source-mode, contract, and validation documentation.
- Split the largest Go/Kotlin routing adapters and Vue screens into focused
  shape, scoring, constraint, formatting, power-analysis, presentation,
  geolocation, component, and stylesheet modules.
- Added focused regression tests for the extracted Vue services, geolocation
  composable and activity hero; Kotlin routing tests now exercise extracted
  internal functions directly without reflection compatibility wrappers.
- Corrected the activity power-curve series so durations remain sorted and no
  longer trigger Highcharts warning #15.
- Route coordinates and route-generation diagnostics now derive from shared generated contract schemas.
- Route-generation, athlete-performance and data-quality schemas now generate shared models across all three stacks.
- OSRM HTTP transport now lives in dedicated Go and Kotlin clients.
- Docker services bind to loopback by default, and both backends reject browser mutations from origins outside the configured allow-list.
- Main navigation labels are consistently English, and gravel/virtual ride tracks use higher-contrast map colors.
- Developer checks and documentation now describe the same commands run by CI.
- Frontend API calls now resolve typed operation identifiers through a single URL and HTTP boundary.
- Kotlin source synchronization now depends on an injected FIT decoder port instead of a concrete adapter.
- The documentation screenshot gallery now reflects the current UI and covers the annual recap, equipment, GPS Art, settings, and diagnostics screens.

## 1.3.0 - 2026-04-17

- Previous tagged release. See Git history for the complete set of changes.
