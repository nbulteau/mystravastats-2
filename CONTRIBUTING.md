# Contributing

## Principles

- Keep Go and Kotlin behavior aligned for shared endpoints and route generation.
- Preserve API contracts; update OpenAPI and generated models together.
- Keep historical routes as a positive routing signal, preserve strict
  anti-retrace outside the explicit 2 km hub and retain request diagnostics.
- Put domain behavior behind small ports and keep transport/UI concerns in
  adapters, services, components and composables.
- Do not commit personal activity files, Strava credentials, tokens or backups.

## Development workflow

1. Read [developer setup](./docs/getting-started/developer-setup.md) and the
   repository [agent guidance](./AGENTS.md).
2. Add focused tests for the touched behavior. Route-engine changes require
   equivalent Go and Kotlin coverage.
3. Run the checks appropriate to the scope:

   ```shell
   cd back-go && go test ./... && go vet ./...
   cd back-kotlin && ./gradlew check
   cd front-vue && npm run lint && npm run type-check && npm test
   node scripts/check-architecture.mjs
   node scripts/generate-api-contracts.mjs --check
   ```

4. For cross-backend source behavior, run both source-mode smoke suites:

   ```shell
   node scripts/smoke-source-modes.mjs --backend go
   node scripts/smoke-source-modes.mjs --backend kotlin
   ```

5. Update `docs/TODO.md`, architecture documentation and the changelog when the
   implementation materially advances a tracked item or contract.

Keep changes focused and do not revert unrelated work already present in the
working tree.
