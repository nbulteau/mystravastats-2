# API Contract

`openapi.json` is the canonical inventory of the public HTTP API shared by the Go
and Kotlin backends. It covers all 52 operations, their query parameters, JSON
request bodies, success formats and HTTP errors. Domain response schemas remain
partial where `additionalProperties` is allowed; they are not yet exhaustive DTO
validation.

Generate the cross-platform contract models and operation registries with:

```sh
npm ci --prefix scripts --ignore-scripts
node scripts/generate-api-contracts.mjs
```

Validate OpenAPI and check that the generated files and both routers are aligned
without writing files:

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

## HTTP behavior

- GET endpoints accept HEAD. A HEAD response has no body on the wire.
- An unknown API path returns JSON `404`, never the Vue entry point. An unsupported
  method returns JSON `405` with `Allow`, including HEAD for GET endpoints.
- Every API response carries `X-Request-Id`. A supplied ID is echoed after trimming
  if it matches `[A-Za-z0-9._:-]{1,128}`; otherwise a new ID is generated. Route
  diagnostics use the same identifier.
- JSON errors retain `message`, `description`, and the legacy numeric `code`, with
  an additive `requestId`. HTTP status is the error classification; wording may
  differ between runtimes. Origin rejections retain their existing `code: 403`.
- Missing activities return `404`, invalid IDs return `400`, and storage failures
  return `500`. Unexpected Kotlin exceptions are no longer classified as `400`.
- Failed synchronization returns `500` in both backends and retains its existing
  `SourceSyncResult` payload. It is deliberately not replaced by an error envelope.
- Maintenance creation returns `201`; deletion returns `204` with no body. Backups
  use JSON, OAuth callback pages use HTML, and route exports use GPX.
- Route calculation remains `200` with proposals and diagnostics, including when
  no usable proposal is found. No route-generation algorithm was changed.

The correction POST and DELETE use one OpenAPI template, `/corrections/{id}`:
POST takes an issue ID, DELETE an applied correction ID. Runtime parameter names
may differ; actual URLs and operation IDs are unchanged.

## Shared regression cases

`test-fixtures/api/http-contract.json` is consumed by Go's `TestSharedHTTPContract`
and Kotlin's `HttpContractFixtureTest`. It covers empty collections, validation,
malformed/missing JSON, unknown paths, unsupported methods and `Allow`, activity
storage failures, synchronization statuses, and request ID propagation. Services
are stubbed: these tests never synchronize the user's files or contact Strava.

Kotlin's MockMvc verifies HEAD routing, status and headers but retains the generated
representation: Servlet 6.1 delegates body suppression to the servlet container.
The fixture's empty-body assertion is therefore only made by the Go harness.

Remaining work includes fully typed domain responses, broader success payload
fixtures and content-negotiation/size-limit parity. Pagination, resource naming
changes and asynchronous job resources require separate contract migrations.
