# Frontend Assets Strategy

`front-vue` is the only source of truth for the production UI bundle. Backend
projects must not keep committed Vite output in their source resources.

## Runtime Modes

- Local frontend development uses Vite from `front-vue`.
- Docker stacks run the `front-vue` Nginx container and proxy `/api/...` to the
  selected backend.
- Standalone Go binaries embed a fresh copy of `front-vue/dist` in
  `back-go/public`.
- Standalone Kotlin jars embed frontend assets only when built with
  `bootJarWithFrontend` or `-PincludeFrontendAssets=true`.

## Generated Locations

| Target | Generated directory | Versioned |
| --- | --- | --- |
| Frontend build | `front-vue/dist` | no |
| Go embed input | `back-go/public` | no |
| Kotlin static resources | `back-kotlin/build/generated/frontend-static` | no |

## Commands

Build and inject assets for Go:

```sh
scripts/sync-frontend-assets.sh go
```

Build a Kotlin standalone jar with fresh frontend assets:

```sh
cd back-kotlin
./gradlew bootJarWithFrontend
```

Use `--skip-build` only when the current command already built
`front-vue/dist`, for example after downloading a CI artifact:

```sh
scripts/sync-frontend-assets.sh go --skip-build
```

## Release Guardrail

Release scripts must either rebuild `front-vue` immediately before copying it
or consume a freshly produced `front-vue/dist` artifact from the same run.
Destinations are always deleted before copy so removed hashed files cannot stay
embedded in a backend package.

Documentation screenshot scripts should target either Vite, the Docker frontend
container, or a backend package produced through the same sync path.
