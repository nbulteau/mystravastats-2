// Package api contains the HTTP handlers, routes, and API layer of the application.
// The handlers are split across the following files:
//   - handlers_health.go     – health endpoint
//   - handlers_athlete.go    – athlete endpoints
//   - handlers_activities.go – activities and maps endpoints
//   - handlers_statistics.go – statistics and segments endpoints
//   - handlers_routes.go     – route recommendations and generation endpoints
//   - handlers_charts.go     – chart endpoints
//   - handlers_dashboard.go  – dashboard endpoints
//   - handlers_badges.go     – badges endpoint
//   - param_parsers.go       – shared query/path parameter parsing helpers
//   - response_writers.go    – shared HTTP response writing helpers
//   - route_cache.go         – in-memory TTL cache for generated routes
//   - route_generation.go    – route scoring, GPX building, shape inference
package api
