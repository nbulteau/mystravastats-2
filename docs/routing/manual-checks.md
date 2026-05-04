# Manual Route Checks

These checks validate Strava Art route-generation behavior on real OSRM-backed responses. The Strava Art smoke matrix is now scriptable and can be wired into local/CI smoke runs when OSRM data is available.

## Prerequisites

- OSRM is running and reachable.
- A backend is running from the current source.
- For UI fallback checks, the frontend dev server is also running.

## Check Matrix

| Check | Script | Doc | Main acceptance |
| --- | --- | --- | --- |
| Strava Art smoke | `./scripts/smoke-strava-art.sh` | [strava-art-smoke](./checks/strava-art-smoke.md) | shape generation returns a route and GPX export works |
| Anti-retrace | Legacy target script retired | [anti-retrace](./checks/anti-retrace.md) | strict for classic explorer routes; diagnostic-only for Strava Art when retrace improves drawing fit |
| Direction | Legacy target script retired | [direction](./checks/direction.md) | internal relaxation diagnostics remain mapped for parity |
| Surface | Legacy target script retired | [surface](./checks/surface.md) | surface reasons are exposed and route-type fallback behavior is calibrated |
| Fallback diagnostics | API protocol in doc | [fallback](./checks/fallback.md) | API and UI expose fallback diagnostics even when routes are returned |
| Shape tuning | `./scripts/manual-route-shape-tuning-check.sh` | [shape tuning](./checks/shape-tuning.md) | dense and rural scenarios keep expected strategy modes |

## Typical Flow

1. Start OSRM from the Diagnostics tab or with `docker compose -f docker-compose-routing-osrm.yml up -d osrm`.
2. Start the backend you want to validate.
3. Run the specific script from the repository root.
4. Keep the compact JSON summaries in the issue or PR when behavior changes.

Automatable Strava Art smoke:

```bash
BACKEND_URL=http://127.0.0.1:8080 ./scripts/smoke-strava-art.sh
RUN_STRAVA_ART_SMOKE=1 ./scripts/smoke-docker-compose.sh go
RUN_STRAVA_ART_SMOKE=1 ./scripts/smoke-docker-compose.sh kotlin
```

Related docs:

- [OSRM Setup](./osrm-setup.md)
- [Route Generation Engine](./generation-engine.md)
- [Strava Art Route Contract](./strava-art-contract.md)
- [TODO](../TODO.md)
