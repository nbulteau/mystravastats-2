package infrastructure

import "mystravastats/internal/platform/activityprovider"

// HealthServiceAdapter bridges the current internal/services layer
// to the hexagonal outbound ports used by health use cases.
type HealthServiceAdapter struct{}

func NewHealthServiceAdapter() *HealthServiceAdapter {
	return &HealthServiceAdapter{}
}

func (adapter *HealthServiceAdapter) FindCacheHealthDetails() map[string]any {
	return activityprovider.Get().CacheDiagnostics()
}
