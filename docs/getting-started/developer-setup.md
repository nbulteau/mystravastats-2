# Developer Setup

This guide collects the commands used most often while developing My Activity Stats.

## Toolchain Versions

Use the same toolchain versions in local development, CI, Docker, and release scripts.

| Area | Version source | Supported version |
| --- | --- | --- |
| Go backend | `back-go/go.mod` | Go `1.27.1` |
| Kotlin backend | `back-kotlin/build.gradle.kts` | Java `25` |
| Kotlin build | `back-kotlin/gradle/wrapper/gradle-wrapper.properties` | Gradle `9.8.0` |
| Frontend | `front-vue/package.json` | Node.js `>=26.10.0` |

Dependency refresh (2026-10-06): Kotlin `2.4.20`, Gradle `9.8.0`, Vue `3.5.43`,
Vite `8.3.3`, Vitest `5.0.3`, and the latest compatible direct and transitive
packages. Spring Boot remains on `4.1.1` (latest stable reported by the dependency
check); Java stays on the existing `25` LTS line. Spring-managed dependencies
continue to follow the Boot BOM.

TypeScript stays on `~6.0.3`: `vue-tsc` still uses the JavaScript compiler API
(`typescript/lib/tsc`), so adopting the native TypeScript 7 compiler requires a
separate Vue tooling migration.

`npm audit --omit=dev` reports no vulnerabilities at this refresh. The full audit
still reports four high-severity entries for the same development-only dependency
chain (`@vue/eslint-config-typescript` → `fast-glob` → `micromatch` → `braces`,
[GHSA-vfj7-8cjw-p6xm](https://github.com/advisories/GHSA-vfj7-8cjw-p6xm)).
No patched `braces` release is available; npm's forced fix would downgrade the
Vue ESLint configuration. Recheck when an upstream fix is published.

The CI and local scripts can check drift with:

```sh
./scripts/check-toolchains.sh
```

## Docker Stacks

Kotlin backend:

```sh
docker compose -f docker-compose-kotlin.yml up --build
```

Go backend:

```sh
docker compose -f docker-compose-go.yml up --build
```

Both stacks expose the UI on [http://localhost/](http://localhost/) and the backend on [http://localhost:8080/](http://localhost:8080/). Nginx proxies `/api/...` to the backend service.

After OSRM data is prepared, add the routing compose file:

```sh
docker compose -f docker-compose-go.yml -f docker-compose-routing-osrm.yml up --build
docker compose -f docker-compose-kotlin.yml -f docker-compose-routing-osrm.yml up --build
```

Smoke checks:

```sh
./scripts/smoke-docker-compose.sh go
./scripts/smoke-docker-compose.sh kotlin
```

## Local Backend Commands

Kotlin:

```sh
cd back-kotlin
./gradlew build
./gradlew bootRun
```

For a standalone Kotlin jar that also serves the Vue UI, use:

```sh
cd back-kotlin
./gradlew bootJarWithFrontend
```

Go:

```sh
cd back-go
go test ./...
go run .
```

Before building a standalone Go binary directly with `go build`, inject a fresh
frontend bundle:

```sh
scripts/sync-frontend-assets.sh go
```

## Frontend Development

```sh
cd front-vue
npm install
npm run dev
```

Useful check:

```sh
npm run lint:check
npm run type-check
npm run test:coverage
npm run test:e2e
npm run build:check
```

## Screenshots

Documentation screenshots are captured by:

```sh
node scripts/capture-doc-screenshots.mjs
```

The default output directory is `docs/assets/screenshots`.

## Validation Shortcuts

- Contracts: `node scripts/generate-api-contracts.mjs --check`
- Architecture boundaries: `node scripts/check-architecture.mjs`
- Frontend: `cd front-vue && npm run lint:check && npm run type-check && npm run test:coverage && npm run build:check`
- Browser journeys: `cd front-vue && npm run test:e2e`
- Go backend: `./scripts/check-go-coverage.sh`
- Kotlin backend: `cd back-kotlin && ./gradlew check`
- Route generation: run targeted Go/Kotlin tests plus the relevant [manual route checks](../routing/manual-checks.md)

The same checks run in `.github/workflows/ci.yml`. Initial regression thresholds are
50% statement coverage for Go, 60% line coverage for Kotlin, and 18% lines/functions,
17% statements and 14% branches for the frontend. Raise them gradually as large views
and adapters are split.
