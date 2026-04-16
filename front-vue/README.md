# MyStravaStats

MyStravaStats is a web application that allows users to visualize and analyze their Strava activities. The application provides various charts and statistics to help users track their performance over time.

## Installation

To install npm (Node Package Manager), you first need to install Node.js, as npm comes bundled with it. Here are the steps to install npm:

### Download and Install Node.js

- Go to the [Node.js official website](https://nodejs.org/).
- Download the installer for your operating system (Windows, macOS, or Linux).
- Run the installer and follow the on-screen instructions to complete the installation.

### Verify the Installation

- Open a terminal or command prompt.
- Run the following commands to check if Node.js and npm are installed correctly:

```sh
node -v
npm -v
```

To get started with MyStravaStats, follow these steps:

### Clone the repository

```sh
git clone https://github.com/yourusername/mystravastats.git
cd mystravastats
```

### Install npm dependencies

```sh
npm install
```

### Run the development server

```sh
npm run serve
```

### Open the application

Open your browser and navigate to <http://localhost:5173/>

## Architecture

### Project Structure

MyStravaStats follows a modern Vue 3 + TypeScript architecture with a clear separation of concerns:

```
front-vue/
├── src/
│   ├── App.vue                    # Root component with main navigation
│   ├── main.ts                    # Application entry point (Pinia, Router setup)
│   │
│   ├── router/
│   │   └── index.ts               # Vue Router configuration (lazy-loaded routes)
│   │
│   ├── stores/                    # Pinia state management
│   │   ├── context.ts             # Global context (current view, year, activity type)
│   │   ├── api.ts                 # API utility functions
│   │   ├── athlete.ts             # Athlete profile & settings
│   │   ├── activities.ts          # Activities cache
│   │   ├── statistics.ts          # Statistics data
│   │   ├── charts.ts              # Chart data
│   │   ├── dashboard.ts           # Dashboard data (heatmap, Eddington)
│   │   ├── map.ts                 # Map tracks & viewport
│   │   ├── badges.ts              # Badge achievements
│   │   ├── segments.ts            # Segment analysis
│   │   ├── routes.ts              # Route generation
│   │   └── ui.ts                  # UI state (toasts)
│   │
│   ├── views/                     # Page-level components (lazy-loaded)
│   │   ├── StatisticsView.vue
│   │   ├── ActivitiesView.vue
│   │   ├── ChartsView.vue
│   │   ├── DashboardView.vue
│   │   ├── DetailedActivityView.vue
│   │   ├── MapView.vue
│   │   ├── HeatmapView.vue
│   │   ├── BadgesView.vue
│   │   ├── SegmentsView.vue
│   │   └── RoutesView.vue
│   │
│   ├── components/                # Reusable UI components
│   │   ├── HeaderBar.vue
│   │   ├── StatisticsGrid.vue
│   │   ├── ActivitiesGrid.vue
│   │   ├── AllTracksMap.vue
│   │   └── charts/                # Chart components
│   │       ├── ByMonthsChart.vue
│   │       ├── EddingtonNumberChart.vue
│   │       └── ...
│   │
│   ├── models/                    # TypeScript interfaces
│   │   ├── activity.model.ts      # Activity, DetailedActivity, ActivityEffort
│   │   ├── statistics.model.ts
│   │   ├── badge.model.ts
│   │   ├── error.model.ts
│   │   └── ...
│   │
│   ├── services/                  # Business logic
│   │   └── error.service.ts       # Error handling & toast notifications
│   │
│   ├── utils/                     # Helper functions
│   │   ├── formatters.ts          # Date, time, speed formatting
│   │   ├── charts.ts              # Chart utilities
│   │   ├── mapTrackColors.ts
│   │   └── heart-rate-zones.ts
│   │
│   └── assets/                    # Static files
│       ├── main.css
│       ├── base.css
│       ├── buttons/               # SVG icons
│       ├── badges/
│       └── images/
│
└── Configuration files
    ├── vite.config.ts             # Vite bundler config
    ├── tsconfig.json              # TypeScript config
    ├── package.json               # Dependencies
    └── eslint.config.cjs          # Linting rules
```

### Data Flow Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        User Interface                            │
│                      (Vue Components)                            │
└──────────┬──────────────────────┬──────────────────────────────┘
           │                      │
           │ (computed/watch)     │ (@click, @change)
           ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Pinia State Stores                           │
│  ┌───────────────┐  ┌───────────────┐  ┌──────────────────┐    │
│  │ contextStore  │  │ activitiesStore   │ statisticsStore  │    │
│  │ (global ctx) │  │ (cache)        │  │ (cache)       │    │
│  └───────────────┘  └───────────────┘  └──────────────────┘    │
│                                                                  │
│  ┌─────────────┐  ┌───────────┐  ┌───────────────┐            │
│  │ athleteStore│  │ chartsStore   │ dashboardStore │            │
│  └─────────────┘  └───────────┘  └───────────────┘            │
└──────────┬──────────────────────────────────────────────────────┘
           │
           │ actions (async)
           ▼
┌─────────────────────────────────────────────────────────────────┐
│                    API Layer (Pinia actions)                    │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ api.ts (requestJson)  →  Error Handling (catchError)     │  │
│  └──────────────────────────────────────────────────────────┘  │
└──────────┬──────────────────────────────────────────────────────┘
           │
           │ fetch() HTTP
           ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Backend API Server                           │
│              (/api/activities, /api/statistics, ...)            │
└─────────────────────────────────────────────────────────────────┘
```

### Key Design Patterns

#### 1. **State Management (Pinia)**
- Each feature area has its own store
- **Cache Strategy**: Data cached by filter key `${activityType}__${year}`
- **Lazy-loaded routes**: Code-splitting for better performance

#### 2. **Error Handling**
- Centralized `ErrorService.catchError()` 
- IAP (Identity-Aware Proxy) support (401/403 redirect)
- Toast notifications for all errors
- `Promise<never>` return type ensures execution stops after error

#### 3. **Reactive Dependencies**
- `contextStore.currentFiltersKey` getter — eliminates key duplication
- Computed properties auto-update when filters change
- Stores refresh data on context changes

#### 4. **Type Safety**
- All models as TypeScript **interfaces** (zero runtime overhead)
- Strict `verbatimModuleSyntax` requires `import type` for interfaces
- Type narrowing prevents runtime errors

### Component Lifecycle

Views follow this pattern:
```typescript
// In onMounted() — NOT in setup()
onMounted(() => {
  contextStore.updateCurrentView("view-name");
  // Fetch data only when component is mounted
});
```

Benefits:
- Avoids side effects during component setup
- Lazy fetching improves initial load
- Cleaner separation of setup vs. side effects

## Dependency Management

### View Current Dependencies

```sh
# List all dependencies with their versions
npm list

# Check for outdated packages
npm outdated
```

### Upgrade Dependencies

#### Semantic Versioning (Safest)

```sh
# Upgrade to latest patch version (e.g., 3.5.32 → 3.5.33)
npm update

# Upgrade to latest minor version (e.g., 3.5.32 → 3.6.0)
npm install vue@^3
```

### Testing After Upgrade

```sh
# 1. Type-check the project
npm run type-check

# 2. Lint code for style issues
npm run lint

# 3. Run dev server to verify
npm run dev

# 4. Build for production
npm run build

# 5. Preview production build locally
npm run preview
```

### Key Dependencies to Monitor

| Package | Purpose | Update Frequency |
|---------|---------|------------------|
| `vue` | Core framework | Monthly |
| `vue-router` | Routing | Monthly |
| `pinia` | State management | Monthly |
| `vite` | Bundler | Every 2-3 weeks |
| `typescript` | Type checker | Monthly |
| `highcharts` | Charts library | Quarterly |
| `leaflet` | Map library | Quarterly |
| `bootstrap` | CSS framework | Quarterly |

### Current Versions (as of 2026-04-16)

```json
{
  "vue": "^3.5.32",
  "vue-router": "^5.0.4",
  "pinia": "^3.0.4",
  "vite": "^8.0.8",
  "typescript": "~6.0.2",
  "highcharts": "^12.6.0",
  "leaflet": "^1.9.4",
  "bootstrap": "^5.3.8",
  "mitt": "^3.0.1"
}
```

### Best Practices

✅ **DO:**
- Upgrade patch versions regularly (`npm update`)
- Test after each minor/major version upgrade
- Read changelog before upgrading major versions
- Keep Node.js up to date (`node >= 25.9.0`)
- Use `npm ci` for production builds (instead of `npm install`)

❌ **DON'T:**
- Upgrade all dependencies at once
- Skip type-checking after upgrades
- Ignore breaking changes in changelogs
- Use outdated security-critical packages

### Node.js Version

This project requires **Node.js 25.9.0 or higher**. Check your current version:

```sh
node -v
npm -v
```

To upgrade Node.js, visit [nodejs.org](https://nodejs.org/)

