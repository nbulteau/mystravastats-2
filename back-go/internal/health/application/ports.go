package application

// HealthReader is an outbound port used by health use cases.
// Infrastructure adapters implement this interface.
type HealthReader interface {
	FindCacheHealthDetails() map[string]any
}
