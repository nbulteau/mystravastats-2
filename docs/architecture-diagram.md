# Architecture Diagram

This page gives a high-level view of how MyStravaStats is structured.

## Main Components

- `front-vue`: user interface
- `back-kotlin`: main modern backend
- `back-go`: legacy or packaging-oriented backend
- `strava-cache`: local persisted activity cache
- Strava API: remote activity source

## System Diagram

```mermaid
flowchart LR
    U["User"] --> F["Vue Frontend<br/>front-vue"]
    F -->|"/api"| K["Kotlin Backend<br/>back-kotlin"]
    F -->|"/api"| G["Go Backend<br/>back-go"]

    K --> C["Local Cache<br/>strava-cache"]
    G --> C

    K --> S["Strava API"]
    G --> S

    K --> D["GPX / FIT files"]
```

## Kotlin Backend Layers

```mermaid
flowchart TD
    A["Controllers"] --> B["Services"]
    B --> C["Activity Providers"]
    C --> D["Local Repositories"]
    C --> E["Strava API Adapter"]
    D --> F["strava-cache"]
```

## Request Flow

Typical flow for a frontend request:

1. The user changes year, activity type, or view in the frontend.
2. The frontend store builds a request under `/api/...`.
3. The backend resolves the current data source.
4. Activities are read from cache or fetched from Strava when needed.
5. Services compute statistics, charts, dashboard data, badges, or detailed activity data.
6. The frontend renders charts, maps, tables, or detailed views.

## Data Sources

The Kotlin backend supports:
- Strava API
- local Strava cache
- GPX files
- FIT files

The Go backend supports:
- Strava API
- local Strava cache

## Current Practical Status

Today, the repository contains two backend implementations:
- Kotlin is the richest implementation and the best target for future work
- Go still matters because some packaging scripts still use it

That is why both appear in the repository and in the build flows.
