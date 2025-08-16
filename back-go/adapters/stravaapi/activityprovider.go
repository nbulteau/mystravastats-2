package stravaapi

import (
	"log"
	"mystravastats/adapters/localrepository"
	"mystravastats/domain/business"
	"mystravastats/domain/helpers"
	"mystravastats/domain/strava"
	"strconv"
	"strings"
	"sync"
	"time"
)

type StravaActivityProvider struct {
	clientId             string
	StravaApi            *StravaApi
	localStorageProvider *localrepository.StravaRepository
	activities           []*strava.Activity
	stravaAthlete        strava.Athlete
}

func NewStravaActivityProvider(stravaCache string) *StravaActivityProvider {
	log.Printf("Initialize StravaActivityProvider using %s ...", stravaCache)

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
			provider.localStorageProvider.InitLocalStorageForClientId(id)
			provider.StravaApi = NewStravaApi(id, secret)
			provider.stravaAthlete = provider.retrieveLoggedInAthlete(id)
			provider.activities = provider.loadCurrentYearFromStrava(id)
		} else {
			log.Fatal("Strava authentication not found")
		}
	}

	// Open the browser
	helpers.OpenBrowser("http://localhost:8080")

	log.Printf("✅ MyStravastats ready with clientId=%s and %d activities", provider.clientId, len(provider.activities))

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

func (provider *StravaActivityProvider) loadFromLocalCache(clientId string) []*strava.Activity {
	startTime := time.Now()
	log.Println("⌛ Load activities from local cache ...")

	var loadedActivities []*strava.Activity
	activityCh := make(chan []strava.Activity, 20) // Buffered channel to collect results
	var wg sync.WaitGroup

	startYear := time.Now().Year()
	for year := startYear; year >= 2010; year-- {
		wg.Add(1)
		go func(year int) {
			defer wg.Done()
			activities := provider.localStorageProvider.LoadActivitiesFromCache(clientId, year)
			activityCh <- activities
		}(year)
	}

	// Close the channel once all goroutines are done
	go func() {
		wg.Wait()
		close(activityCh)
	}()

	// Collect results from the channel
	for activities := range activityCh {
		for _, activity := range activities {
			loadedActivities = append(loadedActivities, &activity)
		}
	}

	duration := time.Since(startTime)
	log.Printf("Loaded activities from local cache in %s", duration)
	return loadedActivities
}

func (provider *StravaActivityProvider) loadCurrentYearFromStrava(clientId string) []*strava.Activity {
	startTime := time.Now()
	log.Println("⌛ Load activities from Strava ...")

	var loadedActivities []*strava.Activity
	activityCh := make(chan []strava.Activity, 20) // Buffered channel to collect results
	var wg sync.WaitGroup

	currentYear := time.Now().Year()
	wg.Add(1)
	go func() {
		defer wg.Done()
		activities := provider.retrieveActivities(clientId, currentYear)
		activityCh <- activities
	}()

	for year := currentYear - 1; year >= 2010; year-- {
		wg.Add(1)
		go func(year int) {
			defer wg.Done()
			var activities []strava.Activity
			if provider.localStorageProvider.IsLocalCacheExistForYear(clientId, year) {
				activities = provider.localStorageProvider.LoadActivitiesFromCache(clientId, year)
			} else {
				activities = provider.retrieveActivities(clientId, year)
			}
			activityCh <- activities
		}(year)
	}

	// Close the channel once all goroutines are done
	go func() {
		wg.Wait()
		close(activityCh)
	}()

	// Collect results from the channel
	for activities := range activityCh {
		for _, activity := range activities {
			loadedActivities = append(loadedActivities, &activity)
		}
	}

	duration := time.Since(startTime)
	log.Printf("⏰ %d activities loaded in %s", len(loadedActivities), duration)
	return loadedActivities
}

func (provider *StravaActivityProvider) loadActivitiesStreams(clientId string, year int, activities []strava.Activity) []strava.Activity {
	streamIdsSet := provider.localStorageProvider.BuildStreamIdsSet(clientId, year)

	for i, activity := range activities {
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
		activities[i].Stream = stream
	}

	return activities
}

func (provider *StravaActivityProvider) retrieveLoggedInAthlete(clientId string) strava.Athlete {
	log.Printf("⌛ Load loggedInAthlete with id %s description from Strava", clientId)
	var loggedInAthlete *strava.Athlete
	if provider.StravaApi != nil {
		athlete, err := provider.StravaApi.RetrieveLoggedInAthlete()
		if err == nil {
			provider.localStorageProvider.SaveAthleteToCache(clientId, *athlete)
			loggedInAthlete = athlete
		}
	}

	if loggedInAthlete == nil {
		return provider.localStorageProvider.LoadAthleteFromCache(clientId)
	}

	return *loggedInAthlete
}

func (provider *StravaActivityProvider) retrieveActivities(clientId string, year int) []strava.Activity {
	log.Printf("⌛ Load activities from Strava for year %d", year)

	if provider.StravaApi != nil {
		retrievedActivities, err := provider.StravaApi.GetActivities(year)
		if err == nil {
			filteredActivities := filterByActivityTypes(retrievedActivities)
			provider.localStorageProvider.SaveActivitiesToCache(clientId, year, filteredActivities)

			return provider.loadActivitiesStreams(clientId, year, filteredActivities)
		} else {
			log.Printf("Failed to load activities from Strava: %v", err)
		}
	}
	return nil
}

func (provider *StravaActivityProvider) findActivityById(activityId int64) *strava.Activity {
	for _, activity := range provider.activities {
		if activity.Id == activityId {
			return activity
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
			return activity
		}
	}
	return nil
}

func (provider *StravaActivityProvider) GetActivitiesByActivityTypeGroupByActiveDays(activityTypes ...business.ActivityType) map[string]int {
	log.Printf("Get activities by stravaActivity type (%s) group by active days\n", activityTypes)

	filteredActivities := FilterActivitiesByType(provider.activities, activityTypes...)

	result := make(map[string]int)
	for _, activity := range filteredActivities {
		date := strings.Split(activity.StartDateLocal, "T")[0]
		result[date] += int(activity.Distance / 1000)
	}
	return result
}

func (provider *StravaActivityProvider) GetActivitiesByActivityTypeByYearGroupByActiveDays(year *int, activityTypes ...business.ActivityType) map[string]int {
	log.Printf("Get activities by stravaActivity type (%s) group by active days for year %d\n", activityTypes, year)

	filteredActivities := FilterActivitiesByYear(provider.activities, year)
	filteredActivities = FilterActivitiesByType(filteredActivities, activityTypes...)

	result := make(map[string]int)
	for _, activity := range filteredActivities {
		date := strings.Split(activity.StartDateLocal, "T")[0]
		result[date] += int(activity.Distance / 1000)
	}
	return result
}

func (provider *StravaActivityProvider) GetActivitiesByActivityTypeAndYear(year *int, activityTypes ...business.ActivityType) []*strava.Activity {
	filteredActivities := FilterActivitiesByYear(provider.activities, year)
	filteredActivities = FilterActivitiesByType(filteredActivities, activityTypes...)

	return filteredActivities
}

func (provider *StravaActivityProvider) GetActivitiesByActivityTypeGroupByYear(activityTypes ...business.ActivityType) map[string][]*strava.Activity {
	log.Printf("Get activities by stravaActivity type (%s) group by year\n", activityTypes)

	filteredActivities := FilterActivitiesByType(provider.activities, activityTypes...)
	return provider.groupActivitiesByYear(filteredActivities)
}

func (provider *StravaActivityProvider) groupActivitiesByYear(activities []*strava.Activity) map[string][]*strava.Activity {
	activitiesByYear := make(map[string][]*strava.Activity)
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
				activitiesByYear[yearStr] = []*strava.Activity{}
			}
		}
	}
	return activitiesByYear
}

func (provider *StravaActivityProvider) GetAthlete() strava.Athlete {
	return provider.stravaAthlete
}

func filterByActivityTypes(activities []strava.Activity) []strava.Activity {
	var filtered []strava.Activity
	for _, activity := range activities {
		for _, activityType := range business.ActivityTypes {
			if activity.Type == activityType.String() {
				filtered = append(filtered, activity)
				break
			}
		}
	}
	return filtered
}

func FilterActivitiesByType(activities []*strava.Activity, activityTypes ...business.ActivityType) []*strava.Activity {
	if len(activityTypes) == 0 {
		return []*strava.Activity{}
	}

	var filtered []*strava.Activity

	for _, activity := range activities {
		for _, activityType := range activityTypes {
			if activityType == business.Commute {
				if activity.Type == business.Ride.String() && activity.Commute {
					filtered = append(filtered, activity)
					break
				}
				continue
			}

			if activity.Type == activityType.String() && !activity.Commute {
				filtered = append(filtered, activity)
				break
			}
		}
	}

	return filtered
}

func FilterActivitiesByYear(activities []*strava.Activity, year *int) []*strava.Activity {
	if year == nil {
		return activities
	}

	var filtered []*strava.Activity
	for _, activity := range activities {
		activityYear, _ := strconv.Atoi(activity.StartDateLocal[:4])
		if activityYear == *year {
			filtered = append(filtered, activity)
		}
	}
	return filtered
}

func minKey(m map[string][]*strava.Activity) string {
	minKey := ""
	for k := range m {
		if minKey == "" || k < minKey {
			minKey = k
		}
	}
	return minKey
}

func maxKey(m map[string][]*strava.Activity) string {
	maxKey := ""
	for k := range m {
		if maxKey == "" || k > maxKey {
			maxKey = k
		}
	}
	return maxKey
}
