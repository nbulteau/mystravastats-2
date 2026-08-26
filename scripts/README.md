# Scripts

## sync-frontend-assets

Copy a fresh Vite production bundle into the generated backend asset location:

```shell
scripts/sync-frontend-assets.sh go
scripts/sync-frontend-assets.sh kotlin
```

Use `--skip-build` only when `front-vue/dist` was produced earlier in the same
build run:

```shell
scripts/sync-frontend-assets.sh go --skip-build
```

The PowerShell variant exposes the same targets:

```powershell
.\scripts\sync-frontend-assets.ps1 -Target go
.\scripts\sync-frontend-assets.ps1 -Target kotlin
```

See [Frontend Assets Strategy](../docs/architecture/frontend-assets.md) for the
release rules and backend-specific destinations.

## refresh-climb-classifications.py

Refresh the difficulty points and categories of the national famous-climb
catalogs from the public Climbfinder API:

```shell
py -3 scripts/refresh-climb-classifications.py
py -3 scripts/refresh-climb-classifications.py --apply
```

The first command is a dry run. The second writes high-confidence, one-to-one
matches to `back-go/famous-climb`, `back-kotlin/famous-climb` and
`strava-cache/famous-climb`. Matching uses the country, start and summit
coordinates, route length, ascent, full variant name and existing source URL.
`SHC` is normalized to `HC` because the application exposes categories
`HC` through `4`.

The full provenance and all rejected or ambiguous matches are written to
`docs/data-sources/climb-classification-audit.json`. The France and Spain
profile scrapers only build geometry and base metrics; always run this
classification refresh after regenerating either catalog so their provisional
average-gradient estimate is not committed as a published difficulty.

## audit-climb-catalog.py

Validate every Go/Kotlin catalog mirror, stable identity, profile, coordinate and
HTTPS source, then regenerate the deterministic coverage and manual-review report:

```shell
python3 scripts/audit-climb-catalog.py
python3 scripts/audit-climb-catalog.py --check
```

Country-level origin and verification dates live in
`docs/data-sources/climb-catalog-sources.json`.

## resolve-climb-classification-audit.py

Apply the exact matches approved in
`docs/data-sources/climb-classification-resolutions.json`, retain existing values
for every reviewed ambiguous or unmatched variant, and close the generated audit:

```shell
python3 scripts/resolve-climb-classification-audit.py
python3 scripts/resolve-climb-classification-audit.py --check
```

The resolutions are bound to the audit generation timestamp, so a refreshed
Climbfinder audit must be reviewed before it can be resolved.

## setup-strava-oauth.mjs

Guide the local Strava enrollment after the Strava developer application has
been created manually on `https://www.strava.com/settings/api`.

Usage from the repository root:

```shell
node scripts/setup-strava-oauth.mjs
node scripts/setup-strava-oauth.mjs --cache /absolute/path/to/strava-cache
```

The script writes `.strava`, opens the Strava OAuth authorization page, receives
the local callback, validates the configured Strava API `athlete` endpoint and
stores `.strava-token.json` for later refresh-token reuse.

Set `STRAVA_API_BASE_URL=https://www.api-v3.strava.com` to validate against the
new Strava V3 API host while keeping OAuth on `https://www.strava.com`.

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

Capture documentation screenshots for My Activity Stats.

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
