package activityprovider

import (
	"sync"

	"mystravastats/internal/helpers"
	"mystravastats/internal/platform/runtimeconfig"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	compositeprovider "mystravastats/internal/shared/infrastructure/composite"
	fitprovider "mystravastats/internal/shared/infrastructure/fit"
	gpxprovider "mystravastats/internal/shared/infrastructure/gpx"
	"mystravastats/internal/shared/infrastructure/stravaapi"
)

var (
	provider     ActivityProvider
	providerOnce sync.Once
	serverPort   string
)

type ActivityProvider interface {
	GetDetailedActivity(activityId int64) *strava.DetailedActivity
	GetCachedDetailedActivity(activityId int64) *strava.DetailedActivity
	GetActivitiesByYearAndActivityTypes(year *int, activityTypes ...business.ActivityType) []*strava.Activity
	GetActivitiesByActivityTypeGroupByYear(activityTypes ...business.ActivityType) map[string][]*strava.Activity
	GetActivitiesByActivityTypeGroupByActiveDays(activityTypes ...business.ActivityType) map[string]int
	GetAthlete() strava.Athlete
	GetHeartRateZoneSettings() business.HeartRateZoneSettings
	SaveHeartRateZoneSettings(settings business.HeartRateZoneSettings) business.HeartRateZoneSettings
	GetPerformanceSettings() business.AthletePerformanceSettings
	SavePerformanceSettings(settings business.AthletePerformanceSettings) business.AthletePerformanceSettings
	CacheDiagnostics() map[string]any
	ClientID() string
	CacheRootPath() string
}

type ReloadableActivityProvider interface {
	Reload()
}

// Init eagerly initializes the activity provider singleton.
func Init(port string) {
	serverPort = port
	_ = Get()
}

// Get returns the singleton activity provider (FIT, GPX or Strava).
func Get() ActivityProvider {
	providerOnce.Do(func() {
		stravaCachePath, stravaConfigured := runtimeconfig.OptionalValue("STRAVA_CACHE_PATH")
		fitFilesPath, fitConfigured := runtimeconfig.OptionalValue("FIT_FILES_PATH")
		gpxFilesPath, gpxConfigured := runtimeconfig.OptionalValue("GPX_FILES_PATH")

		configuredSources := 0
		if stravaConfigured {
			configuredSources++
		}
		if fitConfigured {
			configuredSources++
		}
		if gpxConfigured {
			configuredSources++
		}

		if configuredSources > 1 {
			sources := make([]compositeprovider.Source, 0, configuredSources)
			if stravaConfigured {
				sources = append(sources, compositeprovider.Source{
					Name:     "strava",
					Provider: stravaapi.NewStravaActivityProvider(stravaCachePath, serverPort),
				})
			}
			if fitConfigured {
				sources = append(sources, compositeprovider.Source{
					Name:     "fit",
					Provider: fitprovider.NewFITActivityProvider(fitFilesPath),
				})
			}
			if gpxConfigured {
				sources = append(sources, compositeprovider.Source{
					Name:     "gpx",
					Provider: gpxprovider.NewGPXActivityProvider(gpxFilesPath),
				})
			}
			provider = compositeprovider.NewCompositeActivityProvider(sources)
			return
		}

		if fitConfigured {
			provider = fitprovider.NewFITActivityProvider(fitFilesPath)
			return
		}
		if gpxConfigured {
			provider = gpxprovider.NewGPXActivityProvider(gpxFilesPath)
			return
		}
		provider = stravaapi.NewStravaActivityProvider(helpers.StravaCachePath, serverPort)
	})
	return provider
}

func Reload() {
	currentProvider := Get()
	if reloadable, ok := currentProvider.(ReloadableActivityProvider); ok {
		reloadable.Reload()
	}
}
