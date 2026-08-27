# ADR 0002 — Frontend API boundary

- Status: accepted
- Date: 2026-08-27

## Context

Views and Pinia stores historically assembled URLs and occasionally called `fetch` directly. That duplicated endpoint knowledge and allowed frontend code to drift from the backend contract.

## Decision

The generated operation catalog owns endpoint paths. Application code selects an `ApiOperationId` and builds URLs through `services/api-url.ts`. All network access passes through `services/http-client.ts`; stores own state and orchestration, not transport infrastructure.

The architecture check rejects raw `/api/...` literals and direct `fetch` calls in production frontend modules. Tests and generated files are excluded because they need fixtures and literal expectations.

## Consequences

- Renaming an operation or removing it becomes a TypeScript error.
- Path parameters are encoded consistently.
- HTTP error handling remains centralized.
- OpenAPI response schemas can progressively replace handwritten frontend models without changing the transport boundary.
