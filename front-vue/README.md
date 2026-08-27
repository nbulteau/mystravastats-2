# Vue frontend

The Vue 3 and TypeScript frontend provides the dashboards, activity
exploration, maps, statistics, badges, equipment analysis, diagnostics, and GPS
Art studio. It connects to either supported backend through the same canonical
HTTP contract.

## Local development

Use the Node.js version declared by [`package.json`](./package.json), then run:

```sh
npm install
npm run dev
```

Vite serves <http://localhost:5173/> and proxies `/api` to
`http://localhost:8080` by default. Set `VITE_API_PROXY_TARGET` to use another
backend. Use `npm ci` for reproducible CI and packaging installs.

## Architecture

- `views/`: lazy-loaded route-level screens
- `components/`: reusable presentation and feature components
- `stores/`: Pinia state and feature orchestration
- `services/api-url.ts`: generated operation identifiers to URLs
- `services/http-client.ts`: single production HTTP transport boundary
- `generated/api-contract.ts`: generated canonical operation catalog
- `models/`: frontend view models not yet migrated to generated schemas
- `utils/`: stateless formatting and analysis helpers

Production views, components, and stores must not call `fetch` directly or
embed raw `/api/...` paths. See
[`../docs/architecture/decisions/0002-frontend-api-boundary.md`](../docs/architecture/decisions/0002-frontend-api-boundary.md).

The router exposes dashboard, annual and commute recaps, activities, detailed
activity, statistics, charts, maps, heatmap, segments, badges and climb details,
equipment, GPS Art, settings, and diagnostics. Current screenshots are in
[`../docs/reference/screenshots.md`](../docs/reference/screenshots.md).

## Validation

```sh
npm run lint:check
npm run type-check
npm run test:coverage
npm run test:e2e
npm run build:check
```

Architecture and generated-contract checks run from the repository root:

```sh
node scripts/check-architecture.mjs
node scripts/generate-api-contracts.mjs --check
```

See [`../docs/getting-started/developer-setup.md`](../docs/getting-started/developer-setup.md)
for the complete multi-stack workflow.
