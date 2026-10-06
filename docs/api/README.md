# API Contract

`openapi.json` is the canonical inventory of the public HTTP API shared by the Go
and Kotlin backends. It covers all 52 operations, their query parameters, JSON
request bodies, success formats and HTTP errors. Domain response schemas remain
partial where `additionalProperties` is allowed; they are not yet exhaustive DTO
validation.

Activity lists/details (including streams, provenance, efforts and comparisons),
FTP estimates/settings, statistics, personal records, heart-rate analysis and
segment summaries now have concrete response schemas. Units are recorded in
property descriptions and `x-unit`. Most of these response objects reject
undocumented fields; source/settings request compatibility is retained where noted.

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

`test-fixtures/api/business-responses.json` adds 18 success scenarios through the
real controllers and DTO converters in both backends. It covers irregular sample
times, actual zero power, absent sensors, empty streams/collections, unavailable
FTP, dated FTP normalization, body mass, persistence of settings and formatted
statistics with activity references. Domain providers are stubbed; settings
normalization and unavailable-FTP handling use the actual services/use cases.

The tests compare values (numeric tolerance `1e-6`) and allow omitted versus null
optional properties. They also export the unmodified HTTP JSON for a second check
against OpenAPI. That check enforces required nullable fields and rejects unknown
fields and incorrect types; it runs in each backend CI job.

Run the focused checks from the repository root:

```sh
(cd back-go && go test ./api -run TestSharedBusinessResponses -count=1)
(cd back-kotlin && ./gradlew test --tests '*BusinessResponseFixtureTest')
node scripts/validate-api-responses.mjs test-results/api-go-responses.json test-results/api-kotlin-responses.json
node --test scripts/validate-api-responses.test.mjs
```

The generated response files are ignored build artifacts. The OpenAPI generator
also validates the shared expected responses on every `--check` run. These tests
check transport values and serialization, not parity of every statistics algorithm.

## Measurement semantics and compatibility

- `stream.time` is elapsed time in seconds, not a sample index. Never assume a
  one-second interval. Sensor arrays can have different lengths.
- Missing/null/empty sensor arrays have no usable samples. In a populated sensor
  array, zero is a reading. Summary metrics still use legacy zero sentinels for
  unavailable data, so their presence alone does not establish coverage.
- Both backends preserve null power samples and their positions through ingestion,
  JSON storage and API responses. Go represents missing readings internally as NaN;
  `PowerSamples` serializes them as null. Measured zeros remain valid readings.
  Best-power windows containing a missing sample are excluded; other effort averages
  are unavailable for incomplete windows. Charts retain gaps and power zones omit
  missing readings. FIT-derived summary estimates require a complete power stream
  unless a session summary is available. These summaries retain legacy zero sentinels.
- Effort caches fingerprint stream contents, so changed readings invalidate results.
  Previously stored zeros cannot be identified retrospectively as missing samples;
  those activities require reimport from the original source to recover their gaps.
  Time weighting for irregular recordings remains separate work.
- `weightKg` is current manual body mass, not a history of weight measurements.
  FTP history uses inclusive local dates; no entry means no manual FTP for that date.
- Statistic `value` fields are display strings, potentially containing units or
  unavailable markers. They are not numeric inputs for training-load calculations.
- Both backends now emit `bestPowerFor20minutes`/`bestPowerFor60minutes` and the
  historical Go aliases `bestPowerFor20Minutes`/`bestPowerFor60Minutes`. The aliases
  are deprecated in OpenAPI but retained; no public field was removed.
- Go now handles missing altitude on detail responses like Kotlin: unavailable
  derived efforts are omitted instead of triggering a panic. No routing algorithm
  or power-window calculation changed.

Remaining work includes the other domain responses, broader success payload
fixtures and content-negotiation/size-limit parity. Pagination, resource naming
changes and asynchronous job resources require separate contract migrations.
