## Back-Go

MyStravaStats Backend API - A Golang backend service for Strava statistics and activity analysis built with Clean Architecture principles.

## Architecture Overview

Back-Go follows a **Clean Architecture** pattern with clear separation of concerns. The project is organized into layered modules, each with its own domain, application (use cases), and infrastructure layers.

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                           HTTP Server                            │
│              (Port 8080 + CORS Support + Static Files)          │
└────────────┬────────────────────────────────────────────────────┘
             │
┌────────────▼────────────────────────────────────────────────────┐
│                    API Layer (api/)                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   Router     │  │  Handlers    │  │  Container   │          │
│  │  (routes)    │  │  (endpoints) │  │  (wiring)    │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└────────────┬────────────────────────────────────────────────────┘
             │
┌────────────▼────────────────────────────────────────────────────┐
│              Internal Modules (Clean Architecture)               │
│                                                                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Module (e.g., Activities, Athlete, Statistics, etc.)  │   │
│  │                                                          │   │
│  │  ┌──────────────────────────────────────────────────┐  │   │
│  │  │     Application Layer                            │  │   │
│  │  │  ├─ Use Cases (business logic)                   │  │   │
│  │  │  └─ Ports (interfaces)                           │  │   │
│  │  └──────────────────────────────────────────────────┘  │   │
│  │                      △                                  │   │
│  │                      │                                  │   │
│  │  ┌──────────────────────────────────────────────────┐  │   │
│  │  │     Domain Layer                                 │  │   │
│  │  │  ├─ Business Rules                               │  │   │
│  │  │  └─ Domain Models                                │  │   │
│  │  └──────────────────────────────────────────────────┘  │   │
│  │                      △                                  │   │
│  │                      │                                  │   │
│  │  ┌──────────────────────────────────────────────────┐  │   │
│  │  │     Infrastructure Layer                         │  │   │
│  │  │  ├─ Service Adapters (Strava API)                │  │   │
│  │  │  └─ External Dependencies                        │  │   │
│  │  └──────────────────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                   │
│  Available Modules:                                              │
│  • Activities - Activity retrieval and export (CSV/GPX)          │
│  • Athlete - Athlete profile information                         │
│  • Statistics - Performance metrics & personal records           │
│  • Badges - Achievement system                                  │
│  • Charts - Data visualization & trends                          │
│  • Dashboard - Cumulative data & heat maps                       │
│  • Routes - Route exploration & generation                       │
│  • Segments - Segment analysis & progression                     │
│  • HeartRate - HR zone analysis & configuration                  │
│  • Health - System health monitoring                             │
└─────────────────────────────────────────────────────────────────┘
             │
┌────────────▼────────────────────────────────────────────────────┐
│              External Services & Storage                         │
│  ┌──────────────────────┐  ┌──────────────────────────────────┐ │
│  │   Strava API         │  │  Cache Storage (Memory/Disk)     │ │
│  │  (Activity Data)     │  │  (Configuration)                 │ │
│  └──────────────────────┘  └──────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

### Key Components

- **API Layer** (`api/`): HTTP handlers, routing, and manual dependency wiring (singleton `container` that instantiates and connects use cases to their adapters)
- **Internal Modules** (`internal/`): Feature-specific modules following clean architecture
  - Each module has: `application/` (use cases), `domain/` (business logic), `infrastructure/` (external adapters)
- **Domain** (`domain/`): Cross-cutting domain logic (statistics, badges)
- **Adapters** (`adapters/`): External service integrations

### Data Flow

1. **Request** → HTTP Handler (Gorilla Mux)
2. **Handler** → Container (resolve Use Case via singleton wiring)
3. **Use Case** → Domain Logic / Application Service
4. **Service** → Infrastructure Adapter (Strava API)
5. **Response** → DTO → JSON

### Technology Stack

- **Framework**: Gorilla Mux (HTTP routing)
- **Documentation**: Swagger (Swag)
- **CORS**: rs/cors
- **Language**: Go 1.25.2

---

## Running With FIT Files (No Strava API)

The Go backend can now run directly from local FIT files, similarly to Kotlin.

Expected directory layout:

```text
fit-nicolas/
  2026/
    activity-1.fit
    activity-2.fit
  2025/
    activity-3.fit
```

Set:

```shell
export FIT_FILES_PATH=/absolute/path/to/fit-nicolas
```

Then start the backend as usual.  
When `FIT_FILES_PATH` is set, the Go backend uses the FIT provider instead of Strava API/bootstrap.

---

## Quick API Links
### athlete
http://localhost:8080/api/athletes/me

### activities
http://localhost:8080/api/activities

http://localhost:8080/api/activities?year=2025&activityType=VirtualRide

### statistics

http://localhost:8080/api/statistics

http://localhost:8080/api/statistics?year=2025&activityType=VirtualRide

http://localhost:8080/api/statistics/personal-records-timeline?year=2025&activityType=Ride

### charts

http://localhost:8080/api/charts/distance-by-period?activityType=Ride&year=2025&period=MONTHS

http://localhost:8080/api/charts/elevation-by-period?activityType=Ride&year=2025&period=MONTHS

http://localhost:8080/api/charts/average-speed-by-period?activityType=Ride&year=2025&period=MONTHS

### dashboard

http://localhost:8080/api/dashboard/cumulative-data-per-year?activityType=Ride&year=2025

## Swagger

```shell
swag init
```

### swagger-ui
http://localhost:8080/swagger/index.html

### Update dependencies

```shell
go get -u ./...
go mod tidy
```

### Run tests

```shell
go test -v ./...
```
