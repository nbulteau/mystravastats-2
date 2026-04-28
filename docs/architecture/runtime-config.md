# Runtime Configuration

Runtime configuration is centralized in the backend diagnostics payload. `GET /api/health/details` exposes non-sensitive effective values under `runtimeConfig`, and the Diagnostics page renders the same information.

| Variable | Go backend | Kotlin backend | Default | Notes |
| --- | --- | --- | --- | --- |
| `STRAVA_CACHE_PATH` | yes | yes | `strava-cache` | Strava cache directory. |
| `FIT_FILES_PATH` | yes | yes | unset | Selects the FIT provider when set. |
| `GPX_FILES_PATH` | reported only | yes | unset | Selects the GPX provider in Kotlin. Go reports the value but does not support GPX files yet. |
| `CORS_ALLOWED_ORIGINS` | yes | yes | `http://localhost,http://localhost:5173` | Comma-separated list of allowed browser origins. |
| `OPEN_BROWSER` | yes | yes | `true` | Set to `false` in Docker or headless runs. |
| `SERVER_HOST` / `HOST` | yes | no | `localhost` | Go listen host. `SERVER_HOST` wins over `HOST`. |
| `PORT` | yes | reported fallback | `8080` | Go listen port. Kotlin reports it only as a fallback when `SERVER_PORT` is absent. |
| `SERVER_ADDRESS` | no | yes | `0.0.0.0` | Kotlin listen address. |
| `SERVER_PORT` | no | yes | `8080` | Kotlin listen port. |
| `OSM_ROUTING_ENABLED` | yes | yes | `true` | Enables OSRM-backed routing checks and route generation. |
| `OSM_ROUTING_BASE_URL` | yes | yes | `http://localhost:5000` | Docker compose overrides this to the OSRM service URL. |
| `OSM_ROUTING_TIMEOUT_MS` | yes | yes | `3000` | Go rejects values below `200`; Kotlin clamps values below `300` to `300`. |
| `OSM_ROUTING_PROFILE` | yes | yes | unset | Optional routing profile override. |
| `OSM_ROUTING_EXTRACT_PROFILE` | yes | yes | unset | Optional extract profile name. |
| `OSM_ROUTING_EXTRACT_PROFILE_FILE` | yes | yes | `./osm/region.osrm.profile` | OSRM extract profile path used for diagnostics. |
| `OSM_ROUTING_V3_ENABLED` | yes | yes | `true` | Enables the v3 route-generation pipeline. |
| `OSM_ROUTING_DEBUG` | yes | yes | `false` | Adds verbose routing diagnostics. |
| `OSM_ROUTING_HISTORY_BIAS_ENABLED` | yes | yes | `false` | Treats historical routes as a positive signal when enabled. |
| `OSM_ROUTING_HISTORY_HALF_LIFE_DAYS` | yes | yes | `75` | Decay window for historical route weighting. |
| `OSRM_CONTROL_ENABLED` | yes | yes | `true` | Allows the Diagnostics tab to run the fixed local OSRM start command. |
| `OSRM_CONTROL_TIMEOUT_MS` | yes | yes | `30000` | Timeout for the OSRM start command. |
| `OSRM_CONTROL_PROJECT_DIR` | yes | yes | unset | Optional project-root override used by the OSRM start command. |
| `OSRM_CONTROL_COMPOSE_FILE` | yes | yes | `docker-compose-routing-osrm.yml` | Compose file used by the OSRM start command. |
| `OSRM_CONTROL_DOCKER_BIN` | yes | yes | unset | Optional Docker CLI path override. |
| `API_BACKEND_URL` | frontend Docker | frontend Docker | `http://back:8080` | Backend upstream used by the Docker frontend Nginx proxy. |
| `https_proxy` / `HTTPS_PROXY` | no | yes | unset | Proxy support for Strava API access in the Kotlin backend. |

Related docs:

- [Backend Capability Matrix](./backend-capability-matrix.md)
- [OSRM Setup](../routing/osrm-setup.md)
