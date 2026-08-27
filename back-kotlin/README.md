# Kotlin backend

The Kotlin/Spring Boot backend is a supported implementation of the My Activity
Stats API and a parity target for the packaged Go runtime. Public endpoints,
route-generation behavior, and diagnostics must remain aligned with Go.

## Requirements

- Java 25
- the checked-in Gradle wrapper
- Node.js from [`../front-vue/package.json`](../front-vue/package.json) only when
  building a standalone jar with frontend assets
- optional local OSRM instance for GPS Art generation

Run `./scripts/check-toolchains.sh` from the repository root to check the local
toolchains.

## Run locally

```sh
./gradlew bootRun
```

The backend binds to `127.0.0.1:8080` by default. Useful endpoints are:

- health: <http://localhost:8080/api/actuator/health>
- runtime diagnostics: <http://localhost:8080/api/health/details>
- Swagger UI: <http://localhost:8080/api/swagger-ui/index.html>

Set `SERVER_ADDRESS` or `SERVER_PORT` when a different listener is required.
Keep the default loopback binding unless the deployment is protected.

## Activity sources

The backend supports Strava, local FIT and GPX directories, and automatic
composite mode. It shares `STRAVA_CACHE_PATH`, `FIT_FILES_PATH`, and
`GPX_FILES_PATH` with Go. Source preview, persistence, synchronization, and
Strava OAuth enrollment are available from Diagnostics. SRTM elevation
enrichment remains available for local activities.

See [`../docs/data-sources/fit-gpx.md`](../docs/data-sources/fit-gpx.md),
[`../docs/data-sources/strava-oauth.md`](../docs/data-sources/strava-oauth.md), and
[`../docs/architecture/runtime-config.md`](../docs/architecture/runtime-config.md).

## Architecture and contract

Spring controllers and configuration live under `api/`; external repositories
and Strava/SRTM integrations under `adapters/`; business types, ports, and
services under `domain/`. The canonical shared API inventory is
[`../docs/api/openapi.json`](../docs/api/openapi.json).

- [`../docs/architecture/decisions/0001-dual-backend-contract.md`](../docs/architecture/decisions/0001-dual-backend-contract.md)
- [`../docs/architecture/backend-capability-matrix.md`](../docs/architecture/backend-capability-matrix.md)
- [`../docs/architecture/module-boundaries.md`](../docs/architecture/module-boundaries.md)

## Validate and package

```sh
./gradlew check
./gradlew build
```

To create a standalone Spring Boot jar that serves the Vue application:

```sh
./gradlew bootJarWithFrontend
```

For Docker development, run from the repository root:

```sh
docker compose -f docker-compose-kotlin.yml up --build
```

See [`../docs/getting-started/developer-setup.md`](../docs/getting-started/developer-setup.md)
for the complete workflow.
