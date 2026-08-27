# Go backend

The Go backend is the packaged local-binary runtime for My Activity Stats. It
serves the public API and, in standalone builds, the compiled Vue application.
Go and Kotlin are peer implementations of the canonical HTTP contract; shared
route-generation behavior and diagnostics must remain aligned.

## Requirements

- Go version declared by [`go.mod`](./go.mod)
- Node.js version declared by [`../front-vue/package.json`](../front-vue/package.json)
  when rebuilding embedded frontend assets
- optional local OSRM instance for GPS Art generation

Run `./scripts/check-toolchains.sh` from the repository root to check the local
toolchains.

## Run locally

```sh
go run .
```

The server binds to `localhost:8080` by default. Useful entry points are:

- application: <http://localhost:8080/>
- health and diagnostics: <http://localhost:8080/api/health/details>
- Swagger UI: <http://localhost:8080/swagger/index.html>

Use `SERVER_HOST`, `PORT`, or the `-host` and `-port` flags to change the
listener. Keep the default loopback binding unless the deployment is protected.

## Activity sources

The backend supports Strava and its local cache (`STRAVA_CACHE_PATH`), FIT
directories (`FIT_FILES_PATH`), GPX directories (`GPX_FILES_PATH`), and
automatic composite mode when two or more sources are explicitly configured.
With no explicit source, the default is the local `strava-cache` directory.

Source configuration, preview, synchronization, and Strava OAuth enrollment
are also available from the Diagnostics screen. See
[`../docs/data-sources/fit-gpx.md`](../docs/data-sources/fit-gpx.md) and
[`../docs/data-sources/strava-oauth.md`](../docs/data-sources/strava-oauth.md).

## Architecture

Feature modules under `internal/` use application ports, domain types, and
infrastructure adapters. HTTP handlers and dependency wiring live in `api/`.
The canonical API inventory is [`../docs/api/openapi.json`](../docs/api/openapi.json).

Architecture decisions and enforced boundaries are documented in:

- [`../docs/architecture/overview.md`](../docs/architecture/overview.md)
- [`../docs/architecture/module-boundaries.md`](../docs/architecture/module-boundaries.md)
- [`../docs/architecture/backend-capability-matrix.md`](../docs/architecture/backend-capability-matrix.md)

## Validate and build

```sh
go test ./...
go vet ./...
../scripts/check-go-coverage.sh
```

Before building a standalone binary directly, rebuild and embed the frontend:

```sh
../scripts/sync-frontend-assets.sh go
go build .
```

For Docker development, run from the repository root:

```sh
docker compose -f docker-compose-go.yml up --build
```

See [`../docs/getting-started/developer-setup.md`](../docs/getting-started/developer-setup.md)
for the complete validation and packaging workflow.
