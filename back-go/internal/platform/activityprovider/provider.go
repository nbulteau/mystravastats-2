package activityprovider

import (
	"mystravastats/internal/helpers"
	"mystravastats/internal/shared/infrastructure/stravaapi"
	"sync"
)

var (
	provider     *stravaapi.StravaActivityProvider
	providerOnce sync.Once
	serverPort   string
)

// Init eagerly initializes the activity provider singleton.
func Init(port string) {
	serverPort = port
	_ = Get()
}

// Get returns the singleton Strava activity provider.
func Get() *stravaapi.StravaActivityProvider {
	providerOnce.Do(func() {
		provider = stravaapi.NewStravaActivityProvider(helpers.StravaCachePath, serverPort)
	})
	return provider
}
