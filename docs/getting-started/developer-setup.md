# Developer Setup

This guide collects the commands used most often while developing MyStravaStats.

## Toolchain Versions

Use the same toolchain versions in local development, CI, Docker, and release scripts.

| Area | Version source | Supported version |
| --- | --- | --- |
| Go backend | `back-go/go.mod` | Go `1.26.2` |
| Kotlin backend | `back-kotlin/build.gradle.kts` | Java `25` |
| Kotlin build | `back-kotlin/gradle/wrapper/gradle-wrapper.properties` | Gradle `9.4.1` |
| Frontend | `front-vue/package.json` | Node.js `>=25.9.0` |

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

Go:

```sh
cd back-go
go test ./...
go run .
```

## Frontend Development

```sh
cd front-vue
npm install
npm run dev
```

Useful check:

```sh
npm run type-check
```

## Screenshots

Documentation screenshots are captured by:

```sh
node scripts/capture-doc-screenshots.mjs
```

The default output directory is `docs/assets/screenshots`.

## Validation Shortcuts

- Frontend: `cd front-vue && npm run type-check && npm run test:unit`
- Go backend: `cd back-go && go test ./...`
- Kotlin backend: `cd back-kotlin && ./gradlew test`
- Route generation: run targeted Go/Kotlin tests plus the relevant [manual route checks](../routing/manual-checks.md)
