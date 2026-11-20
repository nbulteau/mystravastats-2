# Copilot Instructions

Also follow `AGENTS.md` (repo root) and `CONTRIBUTING.md`.

## Architecture

Monorepo with **two interchangeable backends** and one frontend:

- `back-go` – Go (Gorilla Mux, Clean Architecture). `api/` holds handlers/router/DTOs; `internal/<feature>/{domain,application,infrastructure}` holds each feature (e.g. `routes`, `activities`, `dashboard`).
- `back-kotlin` – Spring Boot (Kotlin) with `domain`, `api` and `adapters` layers under `src/main/kotlin/me/nicolas/stravastats`.
- `front-vue` – Vue 3 + TypeScript + Vite + Pinia. Talks to either backend through `/api/...` (Nginx proxies in Docker). Generated API models live in `src/generated`.
- `docs/api/openapi.json` (+ `*.openapi.yaml`) is the API contract; frontend models and response validators are generated/checked from it.
- `test-fixtures/` holds fixtures shared by both backends for parity tests (api, routes, source-modes, data-quality).
- Data sources: Strava cache (`strava-cache/`, `STRAVA_CACHE_PATH`), FIT (`FIT_FILES_PATH`), GPX (`GPX_FILES_PATH`), or a composite when two or more are configured. See `docs/architecture/runtime-config.md` and `cache-layout.md`.
- The production UI bundle is built only from `front-vue`; `back-go/public` and Kotlin static resources are generated, not versioned (`scripts/sync-frontend-assets.sh go`, `./gradlew bootJarWithFrontend`).

### Enforced boundaries (`node scripts/check-architecture.mjs`)

- Kotlin `domain` must not import `api` or `adapters`.
- Go `domain`/`application` packages must not import any `infrastructure` package.
- Vue views, components and stores must not call `fetch` or hardcode API paths; go through `src/services`.
- Modules over 1,000 lines can't grow beyond their ceiling in `docs/architecture/module-size-baseline.json`; no new module may exceed 1,000 lines. Lower the ceiling when you shrink a file.

## Build, test, lint

Toolchains: Go 1.27.1, Java 25 / Gradle 9.8.0, Node >=26.10.0 (`./scripts/check-toolchains.sh` checks drift).

```sh
# Go
cd back-go && go vet ./... && go test ./...
cd back-go && go test ./api -run TestName               # single test
./scripts/check-go-coverage.sh                           # CI coverage gate (50%)

# Kotlin
cd back-kotlin && ./gradlew check
cd back-kotlin && ./gradlew test --tests "*ClassName*"   # single test class

# Frontend
cd front-vue && npm run lint:check && npm run type-check
cd front-vue && npx vitest run path/to/file.test.ts      # single test (npm run test:unit for all)
cd front-vue && npm run test:coverage && npm run build:check
cd front-vue && npm run test:e2e                         # Playwright

# Repo-level (run `npm ci --prefix scripts --ignore-scripts` first)
node scripts/generate-api-contracts.mjs --check          # OpenAPI/generated models in sync
node scripts/check-architecture.mjs
node scripts/smoke-source-modes.mjs --backend go|kotlin
python3 scripts/audit-climb-catalog.py --check
```

Run: `go run .` (back-go), `./gradlew bootRun` (back-kotlin), `npm run dev` (front-vue), or `docker compose -f docker-compose-{go,kotlin}.yml up --build` (add `-f docker-compose-routing-osrm.yml` for OSRM routing).

`npm run lint` auto-fixes (`--fix`); use `lint:check` to avoid modifying files.

## Conventions

- **Go and Kotlin behavior must stay aligned** for shared endpoints and route generation. Document intentional differences in `docs/architecture/backend-capability-matrix.md`.
- Never silently change an API contract: update OpenAPI and regenerate models together, and keep DTOs aligned across backends.
- Keep controllers/handlers limited to validation, use-case invocation and response mapping; domain logic sits behind small ports.
- Never commit personal activity files, Strava credentials/tokens or backups (`strava-cache`, `.strava`, FIT/GPX folders, `.env`).

### Route generation

- Historical routes are a positive signal (known usable corridors), not a novelty penalty.
- Anti-retrace rules outside the 2 km start/finish hub stay strict; the hub behavior must be explicit and tested.
- Preserve `X-Request-Id` propagation and route-generation diagnostics.
- Route-engine changes need equivalent tests in both backends (anti-retrace, direction, shape, surface, history); parity fixtures are in `test-fixtures/routes`. Manual checks: `docs/routing/manual-checks.md` and `scripts/manual-route-*.sh`.

### Housekeeping

- Prefer minimal, focused diffs; don't touch IDE/local-only files or revert unrelated working-tree changes.
- Update `docs/TODO.md`, architecture docs and `CHANGELOG.md` when work advances a tracked item or changes a contract.
