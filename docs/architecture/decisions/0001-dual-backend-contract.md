# ADR 0001 — Dual-backend contract authority

- Status: accepted
- Date: 2026-08-27

## Context

The repository ships a Go backend and retains a Kotlin/Spring backend. Both expose the same public API and implement route generation, but neither codebase can safely be treated as the implicit specification of the other.

## Decision

The canonical OpenAPI document is the authority for the public HTTP surface. Go and Kotlin are peer implementations of that contract. The Go backend remains the packaged local-binary runtime; Kotlin remains a supported implementation and a parity target.

Any public endpoint change must update the OpenAPI contract first, regenerate the three contract artifacts, and keep both backend route inventories aligned. Route-generation behavior additionally requires equivalent tests in both backends, including history, direction, anti-retrace and the explicit 2 km start/finish zone.

Implementation details may differ when they are not observable through the contract or diagnostics. Intentional capability differences must be recorded in the backend capability matrix.

## Consequences

- Contract drift fails CI before backend tests run.
- A backend cannot silently add or remove a public route.
- Removing either backend is a separate architectural decision, not an incidental refactor.
- Cross-backend behavior fixtures remain necessary for algorithms whose output is user-visible.
