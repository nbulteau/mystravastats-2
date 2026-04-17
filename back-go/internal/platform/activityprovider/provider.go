package activityprovider

import (
	"os"
	"strings"

	"mystravastats/internal/helpers"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	fitprovider "mystravastats/internal/shared/infrastructure/fit"
	"mystravastats/internal/shared/infrastructure/stravaapi"
	"sync"
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
	CacheDiagnostics() map[string]any
	ClientID() string
	CacheRootPath() string
}

// Init eagerly initializes the activity provider singleton.
func Init(port string) {
	serverPort = port
	_ = Get()
}

// Get returns the singleton activity provider (FIT or Strava).
func Get() ActivityProvider {
	providerOnce.Do(func() {
		fitFilesPath := strings.TrimSpace(os.Getenv("FIT_FILES_PATH"))
		if fitFilesPath != "" {
			provider = fitprovider.NewFITActivityProvider(fitFilesPath)
			return
		}
		provider = stravaapi.NewStravaActivityProvider(helpers.StravaCachePath, serverPort)
	})
	return provider
}
