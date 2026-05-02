# Backend Capability Matrix

The repository contains two backend implementations. Keep shared API behavior aligned, and document intentional differences here.

| Capability | Go backend | Kotlin backend | Notes |
| --- | --- | --- | --- |
| Strava API | yes | yes | Live synchronization and OAuth-backed refresh. |
| Local Strava cache | yes | yes | Shared cache layout. |
| FIT files | yes | yes | Selected with `FIT_FILES_PATH`. |
| GPX files | yes | yes | Selected with `GPX_FILES_PATH`; `FIT_FILES_PATH` has priority when both are set. |
| Dashboard/statistics APIs | yes | yes | Keep DTO contracts aligned when both expose the endpoint. |
| Activity details and streams | yes | yes | Used by detailed activity, charts, efforts, and corrections. |
| Local non-destructive corrections | yes | yes | Corrected view is the default; raw view remains available. |
| Gear maintenance | yes | yes | Local service log and gear mileage behavior should remain aligned. |
| OSRM-backed route generation | yes | yes | Route generation parity is mandatory. |
| OSRM start control from Diagnostics | yes | yes | Runs a fixed local `docker compose ... up -d osrm` command. |
| GPX route export | yes | yes | Keep route contracts and diagnostics aligned. |
| Docker frontend proxy | yes | yes | Frontend container proxies `/api/...` to backend service. |

When this table changes, update [Runtime Configuration](./runtime-config.md) and any impacted setup docs.
