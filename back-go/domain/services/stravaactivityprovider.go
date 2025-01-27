package services

import (
	"log"
	"mystravastats/adapters/localrepository"
	"mystravastats/adapters/stravaapi"
	"mystravastats/domain/services/strava"
	"strconv"
	"strings"
	"time"
)

type StravaActivityProvider struct {
	clientId             string
	StravaApi            *stravaapi.StravaApi
	localStorageProvider *localrepository.StravaRepository
	activities           []strava.Activity
	stravaAthlete        strava.Athlete
}

func NewStravaActivityProvider(stravaCache string) *StravaActivityProvider {
	provider := &StravaActivityProvider{
		localStorageProvider: localrepository.NewStravaRepository(stravaCache),
	}

	id, secret, useCache := provider.localStorageProvider.ReadStravaAuthentication(stravaCache)
	if id == "" {
		log.Fatal("Strava authentication not found")
	}

	provider.clientId = id
	if useCache {
		provider.stravaAthlete = provider.localStorageProvider.LoadAthleteFromCache(id)
		provider.activities = provider.loadFromLocalCache(id)
	} else {
		if secret != "" {
			provider.StravaApi = stravaapi.NewStravaApi(id, secret)

			provider.localStorageProvider.InitLocalStorageForClientId(id)
			provider.stravaAthlete = provider.retrieveLoggedInAthlete(id)
			provider.activities = provider.loadCurrentYearFromStrava(id)
		} else {
			log.Fatal("Strava authentication not found")
		}
	}

	log.Printf("ActivityService initialized with clientId=%s and %d activities", provider.clientId, len(provider.activities))
	return provider
}

func (provider *StravaActivityProvider) GetDetailedActivity(activityId int64) *strava.DetailedActivity {
	log.Printf("Get detailed activity for activity id %d", activityId)

	activity := provider.findActivityById(activityId)
	if activity == nil {
		return nil
	}

	startDate := activity.StartDate
	year, _ := strconv.Atoi(startDate[:4])

	stravaDetailedActivity := provider.localStorageProvider.LoadDetailedActivityFromCache(provider.clientId, year, activityId)
	if provider.StravaApi != nil && stravaDetailedActivity == nil {
		detailedActivity, err := provider.StravaApi.GetDetailedActivity(activityId)
		if err == nil {
			provider.localStorageProvider.SaveDetailedActivityToCache(provider.clientId, year, *detailedActivity)
			stravaDetailedActivity = detailedActivity
		}
	}

	if stravaDetailedActivity == nil {
		stravaDetailedActivity = activity.ToStravaDetailedActivity()
	}

	stream := provider.localStorageProvider.LoadActivitiesStreamsFromCache(provider.clientId, year, *activity)
	if provider.StravaApi != nil && stream == nil {
		stream, err := provider.StravaApi.GetActivityStream(*activity)
		if err == nil {
			provider.localStorageProvider.SaveActivitiesStreamsToCache(provider.clientId, year, *activity, *stream)
		}
	}
	stravaDetailedActivity.Stream = stream

	return stravaDetailedActivity
}

func (provider *StravaActivityProvider) loadFromLocalCache(clientId string) []strava.Activity {
	log.Println("Load activities from local cache ...")

	var loadedActivities []strava.Activity
	startYear := time.Now().Year()
	for year := startYear; year >= 2010; year-- {
		log.Printf("Load %d activities ...", year)
		loadedActivities = append(loadedActivities, provider.localStorageProvider.LoadActivitiesFromCache(clientId, year)...)
	}

	log.Printf("%d activities loaded.", len(loadedActivities))
	return loadedActivities
}

func (provider *StravaActivityProvider) loadCurrentYearFromStrava(clientId string) []strava.Activity {
	log.Println("Load activities from Strava ...")

	var loadedActivities []strava.Activity
	currentYear := time.Now().Year()
	loadedActivities = append(loadedActivities, provider.retrieveActivities(clientId, currentYear)...)

	for year := currentYear - 1; year >= 2010; year-- {
		if provider.localStorageProvider.IsLocalCacheExistForYear(clientId, year) {
			activities := provider.localStorageProvider.LoadActivitiesFromCache(clientId, year)
			provider.loadActivitiesStreams(clientId, year, activities)
			loadedActivities = append(loadedActivities, activities...)
		} else {
			loadedActivities = append(loadedActivities, provider.retrieveActivities(clientId, year)...)
		}
	}

	log.Printf("%d activities loaded.", len(loadedActivities))
	return loadedActivities
}

func (provider *StravaActivityProvider) loadActivitiesStreams(clientId string, year int, activities []strava.Activity) {
	streamIdsSet := provider.localStorageProvider.BuildStreamIdsSet(clientId, year)

	for _, activity := range activities {
		var stream *strava.Stream
		if streamIdsSet[activity.Id] {
			stream = provider.localStorageProvider.LoadActivitiesStreamsFromCache(clientId, year, activity)
		} else {
			if provider.StravaApi != nil {
				stream, _ = provider.StravaApi.GetActivityStream(activity)
				if stream != nil {
					provider.localStorageProvider.SaveActivitiesStreamsToCache(clientId, year, activity, *stream)
				}
			}
		}
		activity.Stream = stream
	}
}

func (provider *StravaActivityProvider) retrieveLoggedInAthlete(clientId string) strava.Athlete {
	log.Printf("Load athlete with id %s description from Strava", clientId)

	if provider.StravaApi != nil {
		athlete, err := provider.StravaApi.RetrieveLoggedInAthlete()
		if err == nil {
			provider.localStorageProvider.SaveAthleteToCache(clientId, *athlete)

			return *athlete
		} else {
			log.Printf("Failed to load athlete from Strava: %v", err)

			return strava.Athlete{Id: 0}
		}
	}

	return provider.localStorageProvider.LoadAthleteFromCache(clientId)
}

func (provider *StravaActivityProvider) retrieveActivities(clientId string, year int) []strava.Activity {
	log.Printf("Load activities from Strava for year %d", year)

	if provider.StravaApi != nil {
		retrievedActivities, err := provider.StravaApi.GetActivities(year)
		if err == nil {
			filteredActivities := filterByActivityTypes(retrievedActivities)
			provider.localStorageProvider.SaveActivitiesToCache(clientId, year, filteredActivities)
			provider.loadActivitiesStreams(clientId, year, filteredActivities)
			log.Printf("%d activities loaded", len(filteredActivities))
			return filteredActivities
		} else {
			log.Printf("Failed to load activities from Strava: %v", err)
		}
	}
	return nil
}

func (provider *StravaActivityProvider) findActivityById(activityId int64) *strava.Activity {
	for _, activity := range provider.activities {
		if activity.Id == activityId {
			return &activity
		}
	}
	return nil
}

func (provider *StravaActivityProvider) Athlete() strava.Athlete {
	return provider.stravaAthlete
}

func (provider *StravaActivityProvider) GetActivity(activityId int64) *strava.Activity {
	log.Printf("Get stravaActivity for stravaActivity id %d\n", activityId)

	for _, activity := range provider.activities {
		if activity.Id == activityId {
			return &activity
		}
	}
	return nil
}

func (provider *StravaActivityProvider) GetActivitiesByActivityTypeGroupByActiveDays(activityType ActivityType) map[string]int {
	log.Printf("Get activities by stravaActivity type (%s) group by active days\n", activityType)

	filteredActivities := filterActivitiesByType(provider.activities, activityType)

	result := make(map[string]int)
	for _, activity := range filteredActivities {
		date := strings.Split(activity.StartDateLocal, "T")[0]
		result[date] += int(activity.Distance / 1000)
	}
	return result
}

func (provider *StravaActivityProvider) GetActivitiesByActivityTypeByYearGroupByActiveDays(activityType ActivityType, year *int) map[string]int {
	log.Printf("Get activities by stravaActivity type (%s) group by active days for year %d\n", activityType, year)

	filteredActivities := filterActivitiesByYear(provider.activities, year)
	filteredActivities = filterActivitiesByType(filteredActivities, activityType)

	result := make(map[string]int)
	for _, activity := range filteredActivities {
		date := strings.Split(activity.StartDateLocal, "T")[0]
		result[date] += int(activity.Distance / 1000)
	}
	return result
}

func (provider *StravaActivityProvider) GetActivitiesByActivityTypeAndYear(activityType ActivityType, year *int) []strava.Activity {
	key := activityType.String()
	if year != nil {
		key = key + "-" + strconv.Itoa(*year)
	}

	filteredActivities := filterActivitiesByYear(provider.activities, year)
	filteredActivities = filterActivitiesByType(filteredActivities, activityType)

	return filteredActivities
}

func (provider *StravaActivityProvider) GetActivitiesByActivityTypeGroupByYear(activityType ActivityType) map[string][]strava.Activity {
	log.Printf("Get activities by stravaActivity type (%s) group by year\n", activityType)

	filteredActivities := filterActivitiesByType(provider.activities, activityType)
	return provider.groupActivitiesByYear(filteredActivities)
}

func (provider *StravaActivityProvider) groupActivitiesByYear(activities []strava.Activity) map[string][]strava.Activity {
	activitiesByYear := make(map[string][]strava.Activity)
	for _, activity := range activities {
		year := activity.StartDateLocal[:4]
		activitiesByYear[year] = append(activitiesByYear[year], activity)
	}

	if len(activitiesByYear) > 0 {
		minYear, _ := strconv.Atoi(minKey(activitiesByYear))
		maxYear, _ := strconv.Atoi(maxKey(activitiesByYear))
		for year := minYear; year <= maxYear; year++ {
			yearStr := strconv.Itoa(year)
			if _, exists := activitiesByYear[yearStr]; !exists {
				activitiesByYear[yearStr] = []strava.Activity{}
			}
		}
	}
	return activitiesByYear
}

func filterByActivityTypes(activities []strava.Activity) []strava.Activity {
	var filtered []strava.Activity
	for _, activity := range activities {
		for _, activityType := range ActivityTypes {
			if activity.Type == activityType.String() {
				filtered = append(filtered, activity)
				break
			}
		}
	}
	return filtered
}

func filterActivitiesByType(activities []strava.Activity, activityType ActivityType) []strava.Activity {
	var filtered []strava.Activity
	for _, activity := range activities {
		if activityType == Commute && activity.Type == Ride.String() && activity.Commute {
			filtered = append(filtered, activity)
		} else if activityType == RideWithCommute && activity.Type == Ride.String() {
			filtered = append(filtered, activity)
		} else if activity.Type == activityType.String() && !activity.Commute {
			filtered = append(filtered, activity)
		}
	}
	return filtered
}

func filterActivitiesByYear(activities []strava.Activity, year *int) []strava.Activity {
	if year == nil {
		return activities
	}

	var filtered []strava.Activity
	for _, activity := range activities {
		activityYear, _ := strconv.Atoi(activity.StartDateLocal[:4])
		if activityYear == *year {
			filtered = append(filtered, activity)
		}
	}
	return filtered
}

func minKey(m map[string][]strava.Activity) string {
	minKey := ""
	for k := range m {
		if minKey == "" || k < minKey {
			minKey = k
		}
	}
	return minKey
}

func maxKey(m map[string][]strava.Activity) string {
	maxKey := ""
	for k := range m {
		if maxKey == "" || k > maxKey {
			maxKey = k
		}
	}
	return maxKey
}
