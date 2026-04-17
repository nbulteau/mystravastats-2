# Scripts

## capture-doc-screenshots.mjs

Capture documentation screenshots for MyStravaStats.

Usage:
node capture-doc-screenshots.mjs [options]

Options:
--base-url <url>            Front URL (default: http://localhost:8080)
--out-dir <path>            Output directory (default: ./docs)
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

```shell
node capture-doc-screenshots.mjs --year 2025
```