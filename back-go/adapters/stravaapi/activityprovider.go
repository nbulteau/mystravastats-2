package stravaapi

import (
	"fmt"
	"log"
	"mystravastats/adapters/localrepository"
	"mystravastats/domain/business"
	"mystravastats/domain/statistics"
	"mystravastats/domain/strava"
	"mystravastats/internal/helpers"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type StravaActivityProvider struct {
	clientId              string
	clientSecret          string
	useCacheAuth          bool
	StravaApi             *StravaApi
	localStorageProvider  *localrepository.StravaRepository
	activities            []*strava.Activity
	activityByID          map[int64]*strava.Activity
	filteredActivities    map[string][]*strava.Activity
	heartRateZoneSettings business.HeartRateZoneSettings
	cacheMutex            sync.RWMutex
	dataMutex             sync.RWMutex
	apiMutex              sync.Mutex
	backgroundRefresh     atomic.Bool
	warmupInProgress      atomic.Bool
	stravaAthlete         strava.Athlete
	serverPort            string
	cacheRoot             string
	manifestMutex         sync.Mutex
	cacheManifest         cacheManifest
	rateLimitUntilUnix    atomic.Int64
}

const detailedBackfillRequestDelay = 1500 * time.Millisecond
const stravaRateLimitCooldown = 15 * time.Minute

func NewStravaActivityProvider(stravaCache string, serverPort string) *StravaActivityProvider {
	log.Printf("Initialize StravaActivityProvider using %s ...", stravaCache)

	provider := &StravaActivityProvider{
		localStorageProvider: localrepository.NewStravaRepository(stravaCache),
		serverPort:           serverPort,
		cacheRoot:            stravaCache,
	}

	id, secret, useCache := provider.localStorageProvider.ReadStravaAuthentication(stravaCache)
	if id == "" {
		log.Fatal("Strava authentication not found")
	}

	provider.clientId = id
	provider.clientSecret = secret
	provider.useCacheAuth = useCache
	provider.localStorageProvider.InitLocalStorageForClientId(id)
	provider.heartRateZoneSettings = provider.localStorageProvider.LoadHeartRateZoneSettings(id)
	provider.cacheManifest = defaultCacheManifest(id)
	provider.loadPersistentCacheArtifacts()

	// Fast startup path: load athlete and activities from local cache first.
	provider.stravaAthlete = provider.localStorageProvider.LoadAthleteFromCache(id)
	provider.activities = provider.loadFromLocalCache(id)

	// First-run fallback when cache is empty: bootstrap synchronously from Strava.
	if len(provider.activities) == 0 && !provider.useCacheAuth && provider.clientSecret != "" {
		provider.ensureStravaAPI()
		provider.stravaAthlete = provider.retrieveLoggedInAthlete(id)
		provider.activities = provider.loadCurrentYearFromStrava(id)
	}

	provider.replaceActivities(provider.activities)

	// Background refresh: fetch new activities and missing streams without blocking startup.
	if !provider.useCacheAuth && provider.clientSecret != "" && len(provider.activities) > 0 {
		provider.launchBackgroundDataRefresh()
	}
	if len(provider.activities) > 0 {
		provider.launchBackgroundWarmup("startup")
	}

	url := fmt.Sprintf("http://localhost:%s", provider.serverPort)
	helpers.OpenBrowser(url)
	fmt.Println("To view your Strava activities, open the following URL in your browser:", url)

	log.Printf("✅ MyStravastats ready with clientId=%s and %d activities (cache-first startup)", provider.clientId, len(provider.activities))

	return provider
}

func (provider *StravaActivityProvider) shouldBootstrapFromStravaAPI(clientId string) bool {
	currentYear := time.Now().Year()

	if !provider.localStorageProvider.IsLocalCacheExistForYear(clientId, currentYear) {
		return true
	}

	if provider.shouldReloadFromStravaAPI(clientId, currentYear) {
		return true
	}

	athlete := provider.localStorageProvider.LoadAthleteFromCache(clientId)
	return athlete.Id == 0
}

func (provider *StravaActivityProvider) GetDetailedActivity(activityId int64) *strava.DetailedActivity {
	log.Printf("Get detailed activity for activity id %d", activityId)

	activity := provider.findActivityById(activityId)
	if activity == nil {
		return nil
	}

	year := resolveActivityYear(activity)
	api := provider.StravaApi
	if provider.isStravaRateLimitedNow() {
		api = nil
	}
	if api == nil && !provider.useCacheAuth && provider.clientSecret != "" && !provider.isStravaRateLimitedNow() {
		api = provider.ensureStravaAPI()
	}

	stravaDetailedActivity := provider.loadDetailedActivityFromCacheAnyYear(activityId, year)
	if api != nil && stravaDetailedActivity == nil {
		detailedActivity, err := api.GetDetailedActivity(activityId)
		if err == nil && detailedActivity != nil {
			provider.localStorageProvider.SaveDetailedActivityToCache(provider.clientId, year, *detailedActivity)
			stravaDetailedActivity = detailedActivity
		} else if err != nil {
			provider.markStravaRateLimited(err, fmt.Sprintf("detailed activity %d", activityId))
			log.Printf("Unable to load detailed activity %d from Strava API: %v", activityId, err)
			if provider.isStravaRateLimitedNow() {
				api = nil
			}
		}
	}

	if stravaDetailedActivity == nil {
		stravaDetailedActivity = activity.ToStravaDetailedActivity()
	}

	stream := provider.localStorageProvider.LoadActivitiesStreamsFromCache(provider.clientId, year, *activity)
	if api != nil && stream == nil {
		stream, err := api.GetActivityStream(*activity)
		if err == nil && stream != nil {
			provider.localStorageProvider.SaveActivitiesStreamsToCache(provider.clientId, year, *activity, *stream)
		} else if err != nil {
			provider.markStravaRateLimited(err, fmt.Sprintf("stream activity %d", activityId))
		}
	}
	stravaDetailedActivity.Stream = stream

	return stravaDetailedActivity
}

func (provider *StravaActivityProvider) GetCachedDetailedActivity(activityId int64) *strava.DetailedActivity {
	activity := provider.findActivityById(activityId)
	if activity == nil {
		return nil
	}

	year := resolveActivityYear(activity)
	if detailed := provider.loadDetailedActivityFromCacheAnyYear(activityId, year); detailed != nil {
		return detailed
	}

	return nil
}

func (provider *StravaActivityProvider) loadFromLocalCache(clientId string) []*strava.Activity {
	startTime := time.Now()

	loadedActivities := make([]*strava.Activity, 0)
	activityCh := make(chan []strava.Activity, 20) // Buffered channel to collect results
	var wg sync.WaitGroup

	startYear := time.Now().Year()
	for year := startYear; year >= 2010; year-- {
		wg.Add(1)
		go func(year int) {
			defer wg.Done()
			activities := provider.localStorageProvider.LoadActivitiesFromCache(clientId, year)
			activities = provider.loadActivitiesStreams(clientId, year, activities)
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
		loadedActivities = appendActivityPointers(loadedActivities, activities)
	}

	duration := time.Since(startTime)
	log.Printf("Loaded activities from local cache in %s", duration)

	return loadedActivities
}

func (provider *StravaActivityProvider) loadCurrentYearFromStrava(clientId string) []*strava.Activity {
	startTime := time.Now()

	loadedActivities := make([]*strava.Activity, 0)
	activityCh := make(chan []strava.Activity, 20) // Buffered channel to collect results
	var wg sync.WaitGroup

	currentYear := time.Now().Year()
	wg.Add(1)
	go func() {
		defer wg.Done()
		activities, err := provider.retrieveActivities(clientId, currentYear, true)
		if err != nil {
			log.Printf("Failed to load current-year activities from Strava: %v", err)
			activities = nil
		}
		if len(activities) > 0 {
			activities = provider.loadActivitiesStreams(clientId, currentYear, activities)
		} else {
			activities = []strava.Activity{}
		}

		activityCh <- activities
	}()

	for year := currentYear - 1; year >= 2010; year-- {
		wg.Add(1)
		go func(year int) {
			defer wg.Done()
			var activities []strava.Activity
			if provider.localStorageProvider.IsLocalCacheExistForYear(clientId, year) && !provider.shouldReloadFromStravaAPI(clientId, year) {
				activities = provider.localStorageProvider.LoadActivitiesFromCache(clientId, year)
				if len(activities) > 0 {
					activities = provider.loadActivitiesStreams(clientId, year, activities)
				} else {
					activities = []strava.Activity{}
				}
			} else {
				retrieved, err := provider.retrieveActivities(clientId, year, true)
				if err != nil {
					log.Printf("Failed to load activities for year %d from Strava: %v", year, err)
					retrieved = nil
				}
				activities = retrieved
				if len(activities) > 0 {
					activities = provider.loadActivitiesStreams(clientId, year, activities)
				} else {
					activities = []strava.Activity{}
				}
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
		loadedActivities = appendActivityPointers(loadedActivities, activities)
	}

	duration := time.Since(startTime)
	log.Printf("⏰ %d activities loaded in %s", len(loadedActivities), duration)

	return loadedActivities
}

// Determine if the local cache should be reloaded from Strava API
func (provider *StravaActivityProvider) shouldReloadFromStravaAPI(clientId string, year int) bool {
	const cutoffMillis int64 = 1755408900000 // 17 August 2025 in milliseconds (UTC)
	lastModifiedMillis := provider.localStorageProvider.GetLocalCacheLastModified(clientId, year)
	return lastModifiedMillis < cutoffMillis
}

func (provider *StravaActivityProvider) loadActivitiesStreams(clientId string, year int, activities []strava.Activity) []strava.Activity {
	streamIdsSet := provider.localStorageProvider.BuildStreamIdsSet(clientId, year)
	if len(activities) == 0 {
		return activities
	}

	workerCount := min(len(activities), max(2, runtime.NumCPU()))
	indexCh := make(chan int, len(activities))
	var wg sync.WaitGroup
	var stopRequested atomic.Bool

	for worker := 0; worker < workerCount; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range indexCh {
				if stopRequested.Load() || provider.isStravaRateLimitedNow() {
					continue
				}

				activity := activities[i]
				var stream *strava.Stream
				if streamIdsSet[activity.Id] {
					stream = provider.localStorageProvider.LoadActivitiesStreamsFromCache(clientId, year, activity)
				} else if provider.StravaApi != nil && !provider.isStravaRateLimitedNow() {
					loadedStream, err := provider.StravaApi.GetActivityStream(activity)
					if err != nil {
						provider.markStravaRateLimited(err, fmt.Sprintf("stream activity %d", activity.Id))
						if IsRateLimitError(err) {
							stopRequested.Store(true)
						}
					}
					stream = loadedStream
					if loadedStream != nil {
						provider.localStorageProvider.SaveActivitiesStreamsToCache(clientId, year, activity, *stream)
					}
				}
				activities[i].Stream = stream
			}
		}()
	}

	for i := range activities {
		indexCh <- i
	}
	close(indexCh)
	wg.Wait()

	return activities
}

func (provider *StravaActivityProvider) retrieveLoggedInAthlete(clientId string) strava.Athlete {
	log.Printf("⌛ Load loggedInAthlete with id %s description from Strava", clientId)
	var loggedInAthlete *strava.Athlete
	if provider.StravaApi != nil && !provider.isStravaRateLimitedNow() {
		athlete, err := provider.StravaApi.RetrieveLoggedInAthlete()
		if err == nil {
			provider.localStorageProvider.SaveAthleteToCache(clientId, *athlete)
			loggedInAthlete = athlete
		} else {
			provider.markStravaRateLimited(err, "athlete")
		}
	}

	if loggedInAthlete == nil {
		return provider.localStorageProvider.LoadAthleteFromCache(clientId)
	}

	return *loggedInAthlete
}

func (provider *StravaActivityProvider) retrieveActivities(clientId string, year int, failFastOnRateLimit bool) ([]strava.Activity, error) {
	log.Printf("⌛ Load activities from Strava for year %d", year)
	if provider.isStravaRateLimitedNow() {
		return nil, ErrStravaRateLimitReached
	}

	api := provider.StravaApi
	if api == nil && !provider.useCacheAuth && provider.clientSecret != "" {
		api = provider.ensureStravaAPI()
	}

	if api != nil {
		var (
			retrievedActivities []strava.Activity
			err                 error
		)
		if failFastOnRateLimit {
			retrievedActivities, err = api.GetActivitiesFailFastOnRateLimit(year)
		} else {
			retrievedActivities, err = api.GetActivities(year)
		}

		if err == nil {
			filteredActivities := filterByActivityTypes(retrievedActivities)
			provider.localStorageProvider.SaveActivitiesToCache(clientId, year, filteredActivities)

			return filteredActivities, nil
		}
		provider.markStravaRateLimited(err, fmt.Sprintf("activities year %d", year))

		return nil, err
	}
	if provider.isStravaRateLimitedNow() {
		return nil, ErrStravaRateLimitReached
	}
	return nil, fmt.Errorf("strava api unavailable for year %d retrieval", year)
}

func (provider *StravaActivityProvider) findActivityById(activityId int64) *strava.Activity {
	provider.dataMutex.RLock()
	defer provider.dataMutex.RUnlock()
	if provider.activityByID == nil {
		return nil
	}
	if activity, ok := provider.activityByID[activityId]; ok {
		return activity
	}
	return nil
}

func (provider *StravaActivityProvider) Athlete() strava.Athlete {
	return provider.stravaAthlete
}

func (provider *StravaActivityProvider) ClientID() string {
	return provider.clientId
}

func (provider *StravaActivityProvider) CacheRootPath() string {
	return provider.cacheRoot
}

func (provider *StravaActivityProvider) GetActivity(activityId int64) *strava.Activity {
	log.Printf("Get stravaActivity for stravaActivity id %d\n", activityId)

	return provider.findActivityById(activityId)
}

func (provider *StravaActivityProvider) GetActivitiesByActivityTypeGroupByActiveDays(activityTypes ...business.ActivityType) map[string]int {
	log.Printf("Get activities by stravaActivity type (%s) group by active days\n", activityTypes)

	filteredActivities := FilterActivitiesByType(provider.getActivitiesSnapshot(), activityTypes...)

	result := make(map[string]int)
	for _, activity := range filteredActivities {
		date := strings.Split(activity.StartDateLocal, "T")[0]
		result[date] += int(activity.Distance / 1000)
	}
	return result
}

func (provider *StravaActivityProvider) GetActivitiesByActivityTypeByYearGroupByActiveDays(year *int, activityTypes ...business.ActivityType) map[string]int {
	log.Printf("Get activities by stravaActivity type (%s) group by active days for year %d\n", activityTypes, year)

	filteredActivities := FilterActivitiesByYear(provider.getActivitiesSnapshot(), year)
	filteredActivities = FilterActivitiesByType(filteredActivities, activityTypes...)

	result := make(map[string]int)
	for _, activity := range filteredActivities {
		date := strings.Split(activity.StartDateLocal, "T")[0]
		result[date] += int(activity.Distance / 1000)
	}
	return result
}

func (provider *StravaActivityProvider) GetActivitiesByYearAndActivityTypes(year *int, activityTypes ...business.ActivityType) []*strava.Activity {
	cacheKey := buildFilterCacheKey(year, activityTypes...)
	provider.cacheMutex.RLock()
	if cachedActivities, ok := provider.filteredActivities[cacheKey]; ok {
		provider.cacheMutex.RUnlock()
		return cloneActivityPointers(cachedActivities)
	}
	provider.cacheMutex.RUnlock()

	filteredActivities := FilterActivitiesByYear(provider.getActivitiesSnapshot(), year)
	filteredActivities = FilterActivitiesByType(filteredActivities, activityTypes...)

	provider.cacheMutex.Lock()
	provider.filteredActivities[cacheKey] = filteredActivities
	provider.cacheMutex.Unlock()

	return cloneActivityPointers(filteredActivities)
}

func (provider *StravaActivityProvider) GetActivitiesByActivityTypeGroupByYear(activityTypes ...business.ActivityType) map[string][]*strava.Activity {
	log.Printf("Get activities by stravaActivity type (%s) group by year\n", activityTypes)

	filteredActivities := FilterActivitiesByType(provider.getActivitiesSnapshot(), activityTypes...)
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

func (provider *StravaActivityProvider) GetHeartRateZoneSettings() business.HeartRateZoneSettings {
	provider.dataMutex.RLock()
	defer provider.dataMutex.RUnlock()

	return provider.heartRateZoneSettings
}

func (provider *StravaActivityProvider) SaveHeartRateZoneSettings(settings business.HeartRateZoneSettings) business.HeartRateZoneSettings {
	provider.dataMutex.Lock()
	provider.heartRateZoneSettings = settings
	provider.dataMutex.Unlock()

	provider.localStorageProvider.SaveHeartRateZoneSettings(provider.clientId, settings)
	return settings
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
		sportType := activity.SportType
		if sportType == "" {
			sportType = activity.Type
		}

		for _, activityType := range activityTypes {
			if activityType == business.Commute {
				if sportType == business.Ride.String() && activity.Commute {
					filtered = append(filtered, activity)
					break
				}
				continue
			}

			if sportType == activityType.String() && !activity.Commute {
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

func (provider *StravaActivityProvider) replaceActivities(activities []*strava.Activity) {
	provider.dataMutex.Lock()
	provider.activities = activities
	provider.activityByID = make(map[int64]*strava.Activity, len(activities))
	for _, activity := range activities {
		provider.activityByID[activity.Id] = activity
	}
	provider.dataMutex.Unlock()

	provider.cacheMutex.Lock()
	provider.filteredActivities = make(map[string][]*strava.Activity)
	provider.cacheMutex.Unlock()
}

// indexActivities keeps backward compatibility for existing tests/helpers.
func (provider *StravaActivityProvider) indexActivities() {
	provider.replaceActivities(provider.activities)
}

func (provider *StravaActivityProvider) getActivitiesSnapshot() []*strava.Activity {
	provider.dataMutex.RLock()
	defer provider.dataMutex.RUnlock()

	snapshot := make([]*strava.Activity, len(provider.activities))
	copy(snapshot, provider.activities)
	return snapshot
}

func (provider *StravaActivityProvider) ensureStravaAPI() *StravaApi {
	provider.apiMutex.Lock()
	defer provider.apiMutex.Unlock()
	if provider.isStravaRateLimitedNow() {
		return nil
	}
	if provider.StravaApi != nil {
		return provider.StravaApi
	}
	if provider.clientSecret == "" {
		return nil
	}

	stravaApi := NewStravaApi(provider.clientId, provider.clientSecret)
	if stravaApi == nil {
		// Token fetch failed, switch to cache-only mode
		log.Printf("Failed to initialize Strava API (token fetch error)")
		log.Printf("Switching to cache-only mode: setting useCache=true in .strava file")

		// Update the .strava file to enable cache mode
		provider.localStorageProvider.UpdateStravaAuthentication(provider.cacheRoot, provider.clientId, provider.clientSecret, true)
		provider.useCacheAuth = true
		log.Printf("Successfully switched to cache-only mode")
		return nil
	}

	provider.StravaApi = stravaApi
	return provider.StravaApi
}

func (provider *StravaActivityProvider) isStravaRateLimitedNow() bool {
	untilUnix := provider.rateLimitUntilUnix.Load()
	if untilUnix <= 0 {
		return false
	}
	return time.Now().UTC().Before(time.Unix(untilUnix, 0))
}

func (provider *StravaActivityProvider) markStravaRateLimited(err error, source string) {
	if !IsRateLimitError(err) {
		return
	}

	previousUntilUnix := provider.rateLimitUntilUnix.Load()
	until := time.Now().UTC().Add(stravaRateLimitCooldown)
	provider.rateLimitUntilUnix.Store(until.Unix())
	if previousUntilUnix > time.Now().UTC().Unix() {
		return
	}
	log.Printf(
		"Strava rate limit detected (%s). Switching to immediate cache-only mode until %s",
		source,
		until.Format(time.RFC3339),
	)
}

func (provider *StravaActivityProvider) launchBackgroundDataRefresh() {
	if !provider.backgroundRefresh.CompareAndSwap(false, true) {
		return
	}

	go func() {
		defer provider.backgroundRefresh.Store(false)

		log.Printf("Background data refresh started")
		if provider.ensureStravaAPI() == nil {
			log.Printf("Background data refresh skipped: Strava API unavailable")
			return
		}

		provider.stravaAthlete = provider.retrieveLoggedInAthlete(provider.clientId)
		currentYear := time.Now().Year()
		stoppedByRateLimit := provider.refreshAllYearsActivitiesFromCurrentYear(currentYear)
		if stoppedByRateLimit {
			log.Printf("Background refresh stopped early due to Strava rate limit")
		} else {
			provider.backfillMissingStreams()
			detailedStoppedByRateLimit := provider.backfillMissingDetailedActivities(currentYear)
			if detailedStoppedByRateLimit {
				log.Printf("Background detailed backfill stopped early due to Strava rate limit")
			}
		}
		provider.runWarmupPipeline("post-refresh")

		log.Printf("Background data refresh completed")
	}()
}

func resolveActivityYear(activity *strava.Activity) int {
	if activity == nil {
		return time.Now().Year()
	}
	if len(activity.StartDateLocal) >= 4 {
		if parsedYear, err := strconv.Atoi(activity.StartDateLocal[:4]); err == nil {
			return parsedYear
		}
	}
	if len(activity.StartDate) >= 4 {
		if parsedYear, err := strconv.Atoi(activity.StartDate[:4]); err == nil {
			return parsedYear
		}
	}
	return time.Now().Year()
}

func (provider *StravaActivityProvider) loadDetailedActivityFromCacheAnyYear(activityId int64, preferredYear int) *strava.DetailedActivity {
	triedYears := map[int]struct{}{}
	yearsToTry := make([]int, 0, (time.Now().Year()-2010)+2)

	if preferredYear >= 2010 {
		yearsToTry = append(yearsToTry, preferredYear)
		triedYears[preferredYear] = struct{}{}
	}
	for year := time.Now().Year(); year >= 2010; year-- {
		if _, alreadyAdded := triedYears[year]; alreadyAdded {
			continue
		}
		yearsToTry = append(yearsToTry, year)
	}

	for _, year := range yearsToTry {
		if detailed := provider.localStorageProvider.LoadDetailedActivityFromCache(provider.clientId, year, activityId); detailed != nil {
			return detailed
		}
	}
	return nil
}

func (provider *StravaActivityProvider) refreshAllYearsActivitiesFromCurrentYear(startYear int) bool {
	for year := startYear; year >= 2010; year-- {
		refreshed, err := provider.retrieveActivities(provider.clientId, year, true)
		if err != nil {
			if IsRateLimitError(err) {
				log.Printf("Background refresh stopped at year %d due to Strava rate limit", year)
				return true
			}
			log.Printf("Background refresh: failed for year %d: %v", year, err)
			continue
		}

		if len(refreshed) > 0 {
			refreshed = provider.loadActivitiesStreams(provider.clientId, year, refreshed)
			if provider.isStravaRateLimitedNow() {
				log.Printf("Background refresh stream load stopped at year %d due to Strava rate limit", year)
				return true
			}
		}

		refreshedPointers := appendActivityPointers(make([]*strava.Activity, 0, len(refreshed)), refreshed)
		existing := provider.getActivitiesSnapshot()
		merged := make([]*strava.Activity, 0, len(existing)+len(refreshedPointers))
		for _, activity := range existing {
			if activity == nil {
				continue
			}
			if len(activity.StartDateLocal) >= 4 {
				if y, parseErr := strconv.Atoi(activity.StartDateLocal[:4]); parseErr == nil && y == year {
					continue
				}
			}
			merged = append(merged, activity)
		}
		merged = append(merged, refreshedPointers...)
		provider.replaceActivities(merged)
		invalidatedActivityIDs := map[int64]struct{}{}
		for _, activity := range existing {
			if activity == nil {
				continue
			}
			activityYear := resolveActivityYear(activity)
			if activityYear == year {
				invalidatedActivityIDs[activity.Id] = struct{}{}
			}
		}
		for _, activity := range refreshedPointers {
			if activity == nil {
				continue
			}
			invalidatedActivityIDs[activity.Id] = struct{}{}
		}
		removedEntries := statistics.InvalidateBestEffortCacheByActivityIDs(invalidatedActivityIDs)
		if removedEntries > 0 {
			log.Printf("Invalidated %d best-effort cache entries after refreshing year %d", removedEntries, year)
		}

		log.Printf("Background refresh merged year %d activities (%d total)", year, len(merged))
	}

	return false
}

func (provider *StravaActivityProvider) backfillMissingStreams() {
	activities := provider.getActivitiesSnapshot()
	if len(activities) == 0 {
		return
	}

	activitiesByYear := make(map[int][]*strava.Activity)
	for _, activity := range activities {
		if activity == nil || activity.Stream != nil {
			continue
		}
		year := time.Now().Year()
		if len(activity.StartDateLocal) >= 4 {
			if parsedYear, err := strconv.Atoi(activity.StartDateLocal[:4]); err == nil {
				year = parsedYear
			}
		}
		activitiesByYear[year] = append(activitiesByYear[year], activity)
	}

	years := make([]int, 0, len(activitiesByYear))
	for year := range activitiesByYear {
		years = append(years, year)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))

	for _, year := range years {
		if provider.isStravaRateLimitedNow() {
			log.Printf("Stream backfill stopped early due to Strava rate limit")
			return
		}
		yearActivities := activitiesByYear[year]
		provider.loadMissingStreamsForPointers(year, yearActivities)
		if provider.isStravaRateLimitedNow() {
			log.Printf("Stream backfill stopped at year %d due to Strava rate limit", year)
			return
		}
	}
}

func (provider *StravaActivityProvider) backfillMissingDetailedActivities(startYear int) bool {
	activities := provider.getActivitiesSnapshot()
	if len(activities) == 0 {
		return false
	}
	if provider.isStravaRateLimitedNow() {
		log.Printf("Detailed backfill skipped: Strava rate limit is active")
		return true
	}

	api := provider.ensureStravaAPI()
	if api == nil {
		log.Printf("Detailed backfill skipped: Strava API unavailable")
		return false
	}

	activitiesByYear := make(map[int][]*strava.Activity)
	missingDetails := 0

	for _, activity := range activities {
		if activity == nil {
			continue
		}
		year := resolveActivityYear(activity)
		if year < 2010 || year > startYear {
			continue
		}
		if provider.localStorageProvider.LoadDetailedActivityFromCache(provider.clientId, year, activity.Id) != nil {
			continue
		}
		activitiesByYear[year] = append(activitiesByYear[year], activity)
		missingDetails++
	}

	if missingDetails == 0 {
		log.Printf("All cached activities already have detailed payloads; skipping detailed backfill")
		return false
	}

	log.Printf("Detailed backfill started for %d missing activities", missingDetails)
	lastRequestAt := time.Time{}
	totalLoaded := 0

	for year := startYear; year >= 2010; year-- {
		yearActivities := activitiesByYear[year]
		if len(yearActivities) == 0 {
			continue
		}

		sort.Slice(yearActivities, func(i, j int) bool {
			return yearActivities[i].StartDateLocal > yearActivities[j].StartDateLocal
		})

		loadedForYear := 0
		for _, activity := range yearActivities {
			if provider.isStravaRateLimitedNow() {
				log.Printf("Detailed backfill stopped before activity %d because Strava rate limit is active", activity.Id)
				return true
			}

			if !lastRequestAt.IsZero() {
				if wait := detailedBackfillRequestDelay - time.Since(lastRequestAt); wait > 0 {
					time.Sleep(wait)
				}
			}
			lastRequestAt = time.Now()

			detailedActivity, err := api.GetDetailedActivity(activity.Id)
			if err != nil {
				provider.markStravaRateLimited(err, fmt.Sprintf("detailed backfill activity %d", activity.Id))
				if IsRateLimitError(err) {
					log.Printf(
						"Detailed backfill stopped at year %d for activity %d due to Strava rate limit",
						year,
						activity.Id,
					)
					return true
				}
				log.Printf("Unable to backfill detailed activity %d: %v", activity.Id, err)
				continue
			}
			if detailedActivity == nil {
				continue
			}

			provider.localStorageProvider.SaveDetailedActivityToCache(provider.clientId, year, *detailedActivity)
			loadedForYear++
			totalLoaded++
		}

		if loadedForYear > 0 {
			log.Printf("Detailed backfill cached %d activities for year %d", loadedForYear, year)
		}
	}

	log.Printf("Detailed backfill completed (%d activities cached)", totalLoaded)
	return false
}

func (provider *StravaActivityProvider) loadMissingStreamsForPointers(year int, activities []*strava.Activity) {
	if len(activities) == 0 {
		return
	}

	streamIDs := provider.localStorageProvider.BuildStreamIdsSet(provider.clientId, year)
	workerCount := min(len(activities), max(2, runtime.NumCPU()))
	indexCh := make(chan int, len(activities))
	var wg sync.WaitGroup
	var stopRequested atomic.Bool

	for worker := 0; worker < workerCount; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range indexCh {
				if stopRequested.Load() || provider.isStravaRateLimitedNow() {
					continue
				}

				activity := activities[idx]
				if activity == nil || activity.Stream != nil {
					continue
				}

				var stream *strava.Stream
				if streamIDs[activity.Id] {
					stream = provider.localStorageProvider.LoadActivitiesStreamsFromCache(provider.clientId, year, *activity)
				} else if provider.StravaApi != nil && !provider.isStravaRateLimitedNow() {
					loaded, err := provider.StravaApi.GetActivityStream(*activity)
					if err == nil && loaded != nil {
						provider.localStorageProvider.SaveActivitiesStreamsToCache(provider.clientId, year, *activity, *loaded)
						stream = loaded
					} else if err != nil {
						provider.markStravaRateLimited(err, fmt.Sprintf("stream backfill activity %d", activity.Id))
						if IsRateLimitError(err) {
							stopRequested.Store(true)
						}
					}
				}
				activity.Stream = stream
			}
		}()
	}

	for idx := range activities {
		indexCh <- idx
	}
	close(indexCh)
	wg.Wait()
}

func appendActivityPointers(destination []*strava.Activity, activities []strava.Activity) []*strava.Activity {
	for i := range activities {
		destination = append(destination, &activities[i])
	}
	return destination
}

func cloneActivityPointers(activities []*strava.Activity) []*strava.Activity {
	if len(activities) == 0 {
		return []*strava.Activity{}
	}

	cloned := make([]*strava.Activity, len(activities))
	copy(cloned, activities)
	return cloned
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func buildFilterCacheKey(year *int, activityTypes ...business.ActivityType) string {
	yearKey := "all"
	if year != nil {
		yearKey = strconv.Itoa(*year)
	}

	return fmt.Sprintf("%s:%v", yearKey, activityTypes)
}
