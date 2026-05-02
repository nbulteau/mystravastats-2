# Scripts

## smoke-source-modes.mjs

Launch a backend on temporary ports and validate the critical source-mode
journey for `STRAVA`, `FIT` and `GPX` with local fixtures.

Usage from the repository root:

```shell
node scripts/smoke-source-modes.mjs --backend go
node scripts/smoke-source-modes.mjs --backend kotlin
```

Options:

```text
--backend <go|kotlin>    Backend to launch (default: go)
--backend-url <url>      Validate one already-running backend instead of launching
--modes <list>           Comma list: STRAVA,FIT,GPX (default: all)
--port-start <port>      First temporary port when launching (default: 19080)
--timeout-ms <ms>        Backend startup timeout (default: 90000)
--keep-temp              Keep copied fixtures and built binary for inspection
--help                   Show help
```

The script checks `/api/health/details`, `/api/source-modes/preview`,
dashboard, activity list, activity detail, maps GPX and data-quality report. It
also rejects non-serializable JSON values such as `NaN` or `Infinity`.

The FIT fixture is generated from source:

```shell
cd back-go
go run ../scripts/generate-source-mode-fit-fixture.go --out ../test-fixtures/source-modes/fit/2026/smoke-ride.fit
```

## capture-doc-screenshots.mjs

Capture documentation screenshots for MyStravaStats.

Usage from the repository root:

```shell
node scripts/capture-doc-screenshots.mjs [options]
```

Options:
```text
--base-url <url>            Front URL (default: http://localhost:8080)
--out-dir <path>            Output directory (default: ./docs/assets/screenshots)
--year <value>              Year filter (example: 2025 or "All years")
--activities <list>         Activity selection (same group only).
                            Examples:
                            Ride
                            Run,TrailRun
                            Commute_GravelRide_MountainBikeRide_Ride_VirtualRide
--detailed-activity-id <id> Activity id for detailed screenshot (default: 15340076302)
--wait-ms <n>               Wait before each screenshot (default: 1800)
--viewport <WxH>            Viewport size (default: 1720x1080)
--full-page                 Capture full page screenshots
--screens <list>            Comma list: dashboard,charts,heatmap,statistics,badges,activities,map,segments,detailed
--help                      Show this help
```

```shell
node scripts/capture-doc-screenshots.mjs --year 2025
```
