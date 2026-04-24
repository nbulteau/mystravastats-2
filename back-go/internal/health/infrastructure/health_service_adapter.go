package infrastructure

import (
	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/platform/runtimeconfig"
	routeApp "mystravastats/internal/routes/application"
)

// HealthServiceAdapter bridges the current internal/services layer
// to the hexagonal outbound ports used by health use cases.
type HealthServiceAdapter struct {
	routingEngine routeApp.RoutingEnginePort
}

func NewHealthServiceAdapter(routingEngine routeApp.RoutingEnginePort) *HealthServiceAdapter {
	return &HealthServiceAdapter{
		routingEngine: routingEngine,
	}
}

func (adapter *HealthServiceAdapter) FindCacheHealthDetails() map[string]any {
	diagnostics := activityprovider.Get().CacheDiagnostics()
	if diagnostics == nil {
		diagnostics = map[string]any{}
	}
	if adapter.routingEngine != nil {
		diagnostics["routing"] = adapter.routingEngine.HealthDetails()
	}
	diagnostics["runtimeConfig"] = runtimeconfig.Details()
	return diagnostics
}
