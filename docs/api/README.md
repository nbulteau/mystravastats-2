# API Contract

`openapi.json` is the canonical inventory of the public HTTP API shared by the Go
and Kotlin backends. It currently covers all 52 registered Go routes.

Generate the cross-platform contract models and operation registries with:

```sh
node scripts/generate-api-contracts.mjs
```

Check that the generated files and the Go router are aligned without writing files:

```sh
node scripts/generate-api-contracts.mjs --check
```

The generated outputs are:

- `front-vue/src/generated/api-contract.ts`;
- `back-go/api/dto/generated_contract.go`;
- `back-kotlin/src/main/kotlin/me/nicolas/stravastats/api/dto/GeneratedApiContract.kt`.

`RouteCoordinate` and `RouteGenerationDiagnostic` already use these generated
schemas in all three implementations. Migrate the remaining DTO families
incrementally whenever their API contract changes.

The older `badges.openapi.yaml`, `strava-art-routes.openapi.yaml`, and generated Go
Swagger files remain useful detailed references while their schemas are progressively
merged into the canonical contract. They must not introduce endpoints absent from
`openapi.json`.
