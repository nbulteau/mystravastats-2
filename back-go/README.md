## Back-Go

## Hexagonal Refactor (In Progress)

Goal: migrate to a maintainable hexagonal architecture incrementally (no big-bang rewrite).

Current migrated slices:
- `GET /api/health/details`
- `GET /api/activities/{activityId}`
- `GET /api/activities`
- `GET /api/activities/csv`
- `GET /api/maps/gpx`
- `GET /api/athletes/me`
- `GET /api/badges`
- `GET /api/statistics`
- `GET /api/statistics/personal-records-timeline`
- `GET /api/statistics/segment-climb-progression`
- `GET /api/segments`
- `GET /api/segments/{segmentId}/efforts`
- `GET /api/segments/{segmentId}/summary`
- `GET /api/athletes/me/heart-rate-zones`
- `PUT /api/athletes/me/heart-rate-zones`
- `GET /api/statistics/heart-rate-zones`
- `GET /api/charts/distance-by-period`
- `GET /api/charts/elevation-by-period`
- `GET /api/charts/average-speed-by-period`
- `GET /api/charts/average-cadence-by-period`
- `GET /api/dashboard`
- `GET /api/dashboard/cumulative-data-per-year`
- `GET /api/dashboard/eddington-number`
- `GET /api/dashboard/activity-heatmap`
- HTTP handler (`api`) -> use case (`internal/<bounded-context>/application`) -> outbound port -> infrastructure adapter -> legacy services

New packages introduced:
- `internal/activities/application`
- `internal/activities/domain`
- `internal/activities/infrastructure`
- `internal/athlete/application`
- `internal/athlete/infrastructure`
- `internal/badges/application`
- `internal/badges/infrastructure`
- `internal/charts/application`
- `internal/charts/infrastructure`
- `internal/dashboard/application`
- `internal/dashboard/domain`
- `internal/dashboard/infrastructure`
- `internal/heartrate/application`
- `internal/heartrate/infrastructure`
- `internal/health/application`
- `internal/health/infrastructure`
- `internal/platform/activityprovider`
- `internal/segments/application`
- `internal/segments/domain`
- `internal/segments/infrastructure`
- `internal/statistics/application`
- `internal/statistics/infrastructure`

Migration principles:
- keep existing API contract unchanged
- migrate endpoint-by-endpoint
- keep legacy `internal/services` as a temporary adapter target
- add tests at the use-case layer first

Latest cleanup pass:
- `main.go` no longer initializes provider through `internal/services`
- `api/handlers.go` no longer imports `internal/services`
- provider singleton extracted to `internal/platform/activityprovider`

Next recommended slices:
1. keep shrinking `internal/services` (split by bounded context)
2. add contract tests at HTTP boundary per migrated endpoint
3. standardize package names and boundaries (`*_service_adapter` + bounded contexts)

Target shape:
```text
api/                       # primary adapters (HTTP)
internal/<bounded-context>/
  domain/                  # entities/value objects/domain errors
  application/             # use cases + ports
  infrastructure/          # secondary adapters (strava, cache, file system)
```

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
