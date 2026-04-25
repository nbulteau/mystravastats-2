# AGENTS.md

Project-wide instructions for coding agents (Codex, Claude, etc.).

## Scope

These instructions apply to the whole repository.

## Repository Layout

- `back-go`: Go backend (Clean Architecture, Gorilla Mux).
- `back-kotlin`: Kotlin backend (Spring Boot).
- `front-vue`: Vue 3 + TypeScript frontend.
- `docs`: project documentation and TODO tracking.
- `test-fixtures`: shared fixtures for cross-backend parity tests.

## Core Principles

- Keep **Go and Kotlin behavior aligned** for route generation.
- Prefer deterministic behavior and explicit diagnostics over hidden heuristics.
- Never silently change API contracts.
- Keep route-generation changes covered by tests in both backends.

## Route Generation Guardrails

- Historical routes are a **positive signal** (known usable corridors), not a novelty penalty.
- Anti-retrace rules outside the start/finish hub must remain strict.
- Keep the 2 km start/finish zone behavior explicit and tested.
- Preserve `X-Request-Id` propagation and route generation diagnostics.

## Testing Expectations

Run only what is needed for the touched scope, then widen if risky.

- Go backend:
  - `cd back-go && go test ./...`
- Kotlin backend:
  - `cd back-kotlin && ./gradlew test`
- Frontend:
  - `cd front-vue && npm run type-check`

For route engine changes, prioritize targeted anti-retrace/direction/history tests in both backends.

## Editing and Git Hygiene

- Do not modify IDE/local-only files unless explicitly requested.
- Do not revert unrelated local changes.
- Keep commits focused and small.
- Update `docs/TODO.md` when completing or significantly advancing a tracked item.

## Documentation Sync

When route generation behavior changes, update:

- `docs/TODO.md` progression/status.
- Diagnostics wording if user-facing behavior changed.
- Tests/fixtures if parity or contract expectations changed.

