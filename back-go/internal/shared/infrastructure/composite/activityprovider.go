package composite

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"mystravastats/internal/helpers"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
)

const (
	sourceStrava = "strava"
	sourceFIT    = "fit"
	sourceGPX    = "gpx"
)

const (
	sameActivityStartTolerance          = 10 * time.Minute
	sameActivityTimezoneOffsetTolerance = 2 * time.Minute
	sameActivityTimezoneName            = "Europe/Paris"
)

var (
	sameActivityFallbackTimezoneOffsets = []time.Duration{time.Hour, 2 * time.Hour}
	sameActivityTimezoneLocation        = loadSameActivityTimezoneLocation()
)

var allActivityTypes = []business.ActivityType{
	business.Run,
	business.TrailRun,
	business.Ride,
	business.GravelRide,
	business.MountainBikeRide,
	business.VirtualRide,
	business.InlineSkate,
	business.Hike,
	business.Walk,
	business.Commute,
	business.AlpineSki,
}

type SourceProvider interface {
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

type Source struct {
	Name     string
	Provider SourceProvider
}

type ActivitySourceRef struct {
	Provider       string  `json:"provider"`
	ActivityID     int64   `json:"activityId"`
	StartDateLocal string  `json:"startDateLocal"`
	Distance       float64 `json:"distance"`
	MovingTime     int     `json:"movingTime"`
	HasStream      bool    `json:"hasStream"`
}

type MergeConflict struct {
	Field   string `json:"field"`
	Primary string `json:"primary"`
	Other   string `json:"other"`
	Source  string `json:"source"`
}

type CompositeActivityProvider struct {
	sources                   []Source
	sourcePriority            map[string]int
	athlete                   strava.Athlete
	activities                []*strava.Activity
	activityByID              map[int64]*strava.Activity
	recordsByActivityID       map[int64]compositeRecord
	filteredActivities        map[string][]*strava.Activity
	heartRateSettingsSource   SourceProvider
	performanceSettingsSource SourceProvider
	dataMutex                 sync.RWMutex
	cacheMutex                sync.RWMutex
	diagnostics               compositeDiagnostics
}

type sourceActivity struct {
	source   Source
	activity *strava.Activity
	match    activityMatchMetadata
}

type activityMatchMetadata struct {
	sportFamily string
	startTime   time.Time
}

type activityCluster struct {
	items []sourceActivity
}

type compositeRecord struct {
	Activity        *strava.Activity
	PrimaryProvider string
	PrimaryID       int64
	Sources         []ActivitySourceRef
	Confidence      string
	Conflicts       []MergeConflict
}

type compositeDiagnostics struct {
	MatchedActivities   int
	LocalOnlyActivities int
	ConflictCount       int
	ConflictSamples     []MergeConflict
	SourceSummaries     []map[string]any
}

func NewCompositeActivityProvider(sources []Source) *CompositeActivityProvider {
	cleanSources := make([]Source, 0, len(sources))
	for _, source := range sources {
		if source.Provider == nil {
			continue
		}
		source.Name = normalizeSourceName(source.Name)
		cleanSources = append(cleanSources, source)
	}

	provider := &CompositeActivityProvider{
		sources:             cleanSources,
		sourcePriority:      make(map[string]int, len(cleanSources)),
		filteredActivities:  make(map[string][]*strava.Activity),
		recordsByActivityID: make(map[int64]compositeRecord),
	}
	for index, source := range cleanSources {
		provider.sourcePriority[source.Name] = index
	}
	if len(cleanSources) > 0 {
		provider.athlete = cleanSources[0].Provider.GetAthlete()
		provider.heartRateSettingsSource = cleanSources[0].Provider
		provider.performanceSettingsSource = cleanSources[0].Provider
	}

	provider.rebuild()
	return provider
}

func (provider *CompositeActivityProvider) rebuild() {
	clusters := make([]activityCluster, 0)
	sourceSummaries := make([]map[string]any, 0, len(provider.sources))

	for _, source := range provider.sources {
		activities := source.Provider.GetActivitiesByYearAndActivityTypes(nil, allActivityTypes...)
		sourceDiagnostics := source.Provider.CacheDiagnostics()
		sourceSummaries = append(sourceSummaries, map[string]any{
			"provider":          source.Name,
			"athleteId":         source.Provider.ClientID(),
			"cacheRoot":         source.Provider.CacheRootPath(),
			"activities":        len(activities),
			"availableYearBins": sourceDiagnostics["availableYearBins"],
		})

		for _, activity := range activities {
			if activity == nil {
				continue
			}
			item := sourceActivity{
				source:   source,
				activity: activity,
				match:    activityMatchMetadataFor(activity),
			}
			bestIndex := -1
			for index := range clusters {
				if sourceActivitiesMatch(clusters[index].items[0], item) {
					bestIndex = index
					break
				}
			}
			if bestIndex >= 0 {
				clusters[bestIndex].items = append(clusters[bestIndex].items, item)
			} else {
				clusters = append(clusters, activityCluster{items: []sourceActivity{item}})
			}
		}
	}

	records := make([]compositeRecord, 0, len(clusters))
	conflictSamples := make([]MergeConflict, 0)
	conflictCount := 0
	matchedActivities := 0
	localOnlyActivities := 0

	for _, cluster := range clusters {
		record := provider.mergeCluster(cluster)
		if len(record.Sources) > 1 {
			matchedActivities++
		}
		if !recordHasSource(record, sourceStrava) {
			localOnlyActivities++
		}
		conflictCount += len(record.Conflicts)
		for _, conflict := range record.Conflicts {
			if len(conflictSamples) >= 12 {
				break
			}
			conflictSamples = append(conflictSamples, conflict)
		}
		records = append(records, record)
	}

	activities := make([]*strava.Activity, 0, len(records))
	recordsByID := make(map[int64]compositeRecord, len(records))
	activityByID := make(map[int64]*strava.Activity, len(records))
	for _, record := range records {
		activities = append(activities, record.Activity)
		recordsByID[record.Activity.Id] = record
		activityByID[record.Activity.Id] = record.Activity
	}
	sort.SliceStable(activities, func(i, j int) bool {
		return activitySortTime(activities[i]).After(activitySortTime(activities[j]))
	})

	provider.dataMutex.Lock()
	provider.activities = activities
	provider.activityByID = activityByID
	provider.recordsByActivityID = recordsByID
	provider.diagnostics = compositeDiagnostics{
		MatchedActivities:   matchedActivities,
		LocalOnlyActivities: localOnlyActivities,
		ConflictCount:       conflictCount,
		ConflictSamples:     conflictSamples,
		SourceSummaries:     sourceSummaries,
	}
	provider.dataMutex.Unlock()
}

func (provider *CompositeActivityProvider) mergeCluster(cluster activityCluster) compositeRecord {
	sort.SliceStable(cluster.items, func(i, j int) bool {
		left := provider.sourcePriority[cluster.items[i].source.Name]
		right := provider.sourcePriority[cluster.items[j].source.Name]
		return left < right
	})

	primary := cluster.items[0]
	activity := cloneActivity(primary.activity)
	activity.Stream = bestStream(cluster.items)

	for _, item := range cluster.items[1:] {
		activity = enrichMissingFields(activity, item.activity)
	}

	conflicts := make([]MergeConflict, 0)
	for _, item := range cluster.items[1:] {
		conflicts = append(conflicts, detectConflicts(primary.activity, item.activity, item.source.Name)...)
	}

	return compositeRecord{
		Activity:        activity,
		PrimaryProvider: primary.source.Name,
		PrimaryID:       primary.activity.Id,
		Sources:         sourceRefs(cluster.items),
		Confidence:      confidenceForCluster(cluster),
		Conflicts:       conflicts,
	}
}

func (provider *CompositeActivityProvider) GetDetailedActivity(activityID int64) *strava.DetailedActivity {
	record, ok := provider.record(activityID)
	if !ok {
		return nil
	}
	source := provider.sourceByName(record.PrimaryProvider)
	var detailed *strava.DetailedActivity
	if source != nil {
		detailed = source.Provider.GetDetailedActivity(record.PrimaryID)
	}
	if detailed == nil {
		detailed = record.Activity.ToStravaDetailedActivity()
	}
	return enrichDetailedActivity(detailed, record.Activity)
}

func (provider *CompositeActivityProvider) GetCachedDetailedActivity(activityID int64) *strava.DetailedActivity {
	record, ok := provider.record(activityID)
	if !ok {
		return nil
	}
	source := provider.sourceByName(record.PrimaryProvider)
	var detailed *strava.DetailedActivity
	if source != nil {
		detailed = source.Provider.GetCachedDetailedActivity(record.PrimaryID)
	}
	if detailed == nil {
		return provider.GetDetailedActivity(activityID)
	}
	return enrichDetailedActivity(detailed, record.Activity)
}

func (provider *CompositeActivityProvider) GetActivitiesByYearAndActivityTypes(year *int, activityTypes ...business.ActivityType) []*strava.Activity {
	cacheKey := buildFilterCacheKey(year, activityTypes...)
	provider.cacheMutex.RLock()
	if cached, ok := provider.filteredActivities[cacheKey]; ok {
		provider.cacheMutex.RUnlock()
		return cloneActivityPointers(cached)
	}
	provider.cacheMutex.RUnlock()

	filtered := filterActivitiesByYear(provider.getActivitiesSnapshot(), year)
	filtered = filterActivitiesByType(filtered, activityTypes...)

	provider.cacheMutex.Lock()
	provider.filteredActivities[cacheKey] = filtered
	provider.cacheMutex.Unlock()
	return cloneActivityPointers(filtered)
}

func (provider *CompositeActivityProvider) GetActivitiesByActivityTypeGroupByYear(activityTypes ...business.ActivityType) map[string][]*strava.Activity {
	return groupActivitiesByYear(filterActivitiesByType(provider.getActivitiesSnapshot(), activityTypes...))
}

func (provider *CompositeActivityProvider) GetActivitiesByActivityTypeGroupByActiveDays(activityTypes ...business.ActivityType) map[string]int {
	result := make(map[string]int)
	for _, activity := range filterActivitiesByType(provider.getActivitiesSnapshot(), activityTypes...) {
		if activity == nil {
			continue
		}
		date := strings.Split(activity.StartDateLocal, "T")[0]
		if date == "" {
			continue
		}
		result[date] += int(activity.Distance / 1000)
	}
	return result
}

func (provider *CompositeActivityProvider) GetAthlete() strava.Athlete {
	return provider.athlete
}

func (provider *CompositeActivityProvider) GetHeartRateZoneSettings() business.HeartRateZoneSettings {
	if provider.heartRateSettingsSource == nil {
		return business.HeartRateZoneSettings{}
	}
	return provider.heartRateSettingsSource.GetHeartRateZoneSettings()
}

func (provider *CompositeActivityProvider) SaveHeartRateZoneSettings(settings business.HeartRateZoneSettings) business.HeartRateZoneSettings {
	if provider.heartRateSettingsSource == nil {
		return settings
	}
	return provider.heartRateSettingsSource.SaveHeartRateZoneSettings(settings)
}

func (provider *CompositeActivityProvider) GetPerformanceSettings() business.AthletePerformanceSettings {
	if provider.performanceSettingsSource == nil {
		return business.AthletePerformanceSettings{}
	}
	return provider.performanceSettingsSource.GetPerformanceSettings()
}

func (provider *CompositeActivityProvider) SavePerformanceSettings(settings business.AthletePerformanceSettings) business.AthletePerformanceSettings {
	if provider.performanceSettingsSource == nil {
		return settings
	}
	return provider.performanceSettingsSource.SavePerformanceSettings(settings)
}

func (provider *CompositeActivityProvider) CacheDiagnostics() map[string]any {
	provider.dataMutex.RLock()
	activities := len(provider.activities)
	years := availableYearBins(provider.activities)
	diagnostics := provider.diagnostics
	provider.dataMutex.RUnlock()

	activeProviders := make([]string, 0, len(provider.sources))
	for _, source := range provider.sources {
		activeProviders = append(activeProviders, source.Name)
	}

	return map[string]any{
		"timestamp":         time.Now().UTC().Format(time.RFC3339),
		"provider":          "composite",
		"athleteId":         provider.ClientID(),
		"cacheRoot":         "composite",
		"activities":        activities,
		"availableYearBins": years,
		"composite": map[string]any{
			"active":              true,
			"activeProviders":     activeProviders,
			"sources":             diagnostics.SourceSummaries,
			"matchedActivities":   diagnostics.MatchedActivities,
			"localOnlyActivities": diagnostics.LocalOnlyActivities,
			"conflictCount":       diagnostics.ConflictCount,
			"conflictSamples":     diagnostics.ConflictSamples,
			"futureProviders":     []string{"ridewithgps", "tcx"},
		},
	}
}

func (provider *CompositeActivityProvider) ClientID() string {
	ids := make([]string, 0, len(provider.sources))
	for _, source := range provider.sources {
		ids = append(ids, fmt.Sprintf("%s:%s", source.Name, source.Provider.ClientID()))
	}
	return strings.Join(ids, "+")
}

func (provider *CompositeActivityProvider) CacheRootPath() string {
	roots := make([]string, 0, len(provider.sources))
	for _, source := range provider.sources {
		roots = append(roots, fmt.Sprintf("%s=%s", source.Name, source.Provider.CacheRootPath()))
	}
	return strings.Join(roots, ";")
}

func (provider *CompositeActivityProvider) Reload() {
	for _, source := range provider.sources {
		if reloadable, ok := source.Provider.(interface{ Reload() }); ok {
			reloadable.Reload()
		}
	}
	provider.cacheMutex.Lock()
	provider.filteredActivities = make(map[string][]*strava.Activity)
	provider.cacheMutex.Unlock()
	provider.rebuild()
}

func (provider *CompositeActivityProvider) record(activityID int64) (compositeRecord, bool) {
	provider.dataMutex.RLock()
	defer provider.dataMutex.RUnlock()
	record, ok := provider.recordsByActivityID[activityID]
	return record, ok
}

func (provider *CompositeActivityProvider) getActivitiesSnapshot() []*strava.Activity {
	provider.dataMutex.RLock()
	defer provider.dataMutex.RUnlock()
	return cloneActivityPointers(provider.activities)
}

func (provider *CompositeActivityProvider) sourceByName(name string) *Source {
	for index := range provider.sources {
		if provider.sources[index].Name == name {
			return &provider.sources[index]
		}
	}
	return nil
}

func activitiesMatch(left *strava.Activity, right *strava.Activity) bool {
	if left == nil || right == nil {
		return false
	}
	return sourceActivitiesMatch(
		sourceActivity{activity: left, match: activityMatchMetadataFor(left)},
		sourceActivity{activity: right, match: activityMatchMetadataFor(right)},
	)
}

func sourceActivitiesMatch(left, right sourceActivity) bool {
	if left.activity == nil || right.activity == nil {
		return false
	}
	if left.match.sportFamily != right.match.sportFamily {
		return false
	}
	leftTime := left.match.startTime
	rightTime := right.match.startTime
	if leftTime.IsZero() || rightTime.IsZero() {
		return false
	}
	if !startTimesCompatible(leftTime, rightTime) {
		return false
	}
	if !summaryMetricsCompatible(left.activity, right.activity) {
		return false
	}
	if !startLocationCompatible(left.activity.StartLatlng, right.activity.StartLatlng) {
		return false
	}
	return true
}

func activityMatchMetadataFor(activity *strava.Activity) activityMatchMetadata {
	if activity == nil {
		return activityMatchMetadata{}
	}
	return activityMatchMetadata{
		sportFamily: sportFamily(activity),
		startTime:   activitySortTime(activity),
	}
}

func startTimesCompatible(left, right time.Time) bool {
	delta := absDuration(left.Sub(right))
	if delta <= sameActivityStartTolerance {
		return true
	}
	for _, offset := range timezoneOffsetsForStartTimes(left, right) {
		if absDuration(delta-offset) <= sameActivityTimezoneOffsetTolerance {
			return true
		}
	}
	return false
}

func timezoneOffsetsForStartTimes(left, right time.Time) []time.Duration {
	if sameActivityTimezoneLocation == nil {
		return sameActivityFallbackTimezoneOffsets
	}

	offsets := make([]time.Duration, 0, 2)
	offsets = appendUniqueTimezoneOffset(offsets, timezoneOffsetForInstant(left, sameActivityTimezoneLocation))
	offsets = appendUniqueTimezoneOffset(offsets, timezoneOffsetForInstant(right, sameActivityTimezoneLocation))
	if len(offsets) == 0 {
		return sameActivityFallbackTimezoneOffsets
	}
	return offsets
}

func loadSameActivityTimezoneLocation() *time.Location {
	location, err := time.LoadLocation(sameActivityTimezoneName)
	if err != nil {
		return nil
	}
	return location
}

func timezoneOffsetForInstant(value time.Time, location *time.Location) time.Duration {
	_, offsetSeconds := value.In(location).Zone()
	return absDuration(time.Duration(offsetSeconds) * time.Second)
}

func appendUniqueTimezoneOffset(offsets []time.Duration, offset time.Duration) []time.Duration {
	if offset <= 0 {
		return offsets
	}
	for _, existing := range offsets {
		if existing == offset {
			return offsets
		}
	}
	return append(offsets, offset)
}

func absDuration(value time.Duration) time.Duration {
	if value < 0 {
		return -value
	}
	return value
}

func distanceCompatible(left, right float64) bool {
	if left <= 0 || right <= 0 {
		return false
	}
	delta := math.Abs(left - right)
	limit := math.Max(500, math.Max(left, right)*0.05)
	return delta <= limit
}

func summaryMetricsCompatible(left, right *strava.Activity) bool {
	if left == nil || right == nil {
		return false
	}
	if left.Distance > 0 && right.Distance > 0 {
		return distanceCompatible(left.Distance, right.Distance)
	}
	return durationCompatible(left.MovingTime, right.MovingTime)
}

func durationCompatible(left, right int) bool {
	if left <= 0 || right <= 0 {
		return false
	}
	delta := math.Abs(float64(left - right))
	limit := math.Max(120, math.Max(float64(left), float64(right))*0.10)
	return delta <= limit
}

func startLocationCompatible(left, right []float64) bool {
	if !validLatLng(left) || !validLatLng(right) {
		return true
	}
	return haversineMeters(left[0], left[1], right[0], right[1]) <= 1000
}

func detectConflicts(primary *strava.Activity, other *strava.Activity, source string) []MergeConflict {
	conflicts := make([]MergeConflict, 0)
	if primary == nil || other == nil {
		return conflicts
	}
	if primary.Distance > 0 && other.Distance > 0 {
		delta := math.Abs(primary.Distance - other.Distance)
		if delta > math.Max(250, math.Max(primary.Distance, other.Distance)*0.02) {
			conflicts = append(conflicts, MergeConflict{
				Field:   "distance",
				Primary: fmt.Sprintf("%.0f", primary.Distance),
				Other:   fmt.Sprintf("%.0f", other.Distance),
				Source:  source,
			})
		}
	}
	if primary.MovingTime > 0 && other.MovingTime > 0 {
		delta := math.Abs(float64(primary.MovingTime - other.MovingTime))
		if delta > math.Max(60, math.Max(float64(primary.MovingTime), float64(other.MovingTime))*0.05) {
			conflicts = append(conflicts, MergeConflict{
				Field:   "moving_time",
				Primary: strconv.Itoa(primary.MovingTime),
				Other:   strconv.Itoa(other.MovingTime),
				Source:  source,
			})
		}
	}
	if validLatLng(primary.StartLatlng) && validLatLng(other.StartLatlng) {
		delta := haversineMeters(primary.StartLatlng[0], primary.StartLatlng[1], other.StartLatlng[0], other.StartLatlng[1])
		if delta > 250 {
			conflicts = append(conflicts, MergeConflict{
				Field:   "start_latlng",
				Primary: fmt.Sprintf("%.0fm", 0.0),
				Other:   fmt.Sprintf("%.0fm", delta),
				Source:  source,
			})
		}
	}
	return conflicts
}

func bestStream(items []sourceActivity) *strava.Stream {
	var best *strava.Stream
	bestScore := 0
	for _, item := range items {
		score := streamScore(item.source.Name, item.activity.Stream)
		if score > bestScore {
			best = item.activity.Stream
			bestScore = score
		}
	}
	return best
}

func streamScore(source string, stream *strava.Stream) int {
	if stream == nil {
		return 0
	}
	score := 1
	if stream.LatLng != nil {
		score += len(stream.LatLng.Data) * 3
	}
	if stream.Altitude != nil {
		score += len(stream.Altitude.Data)
	}
	if stream.HeartRate != nil && len(stream.HeartRate.Data) > 0 {
		score += 3000
	}
	if stream.Cadence != nil && len(stream.Cadence.Data) > 0 {
		score += 1500
	}
	if stream.Watts != nil && len(stream.Watts.Data) > 0 {
		score += 3000
	}
	switch source {
	case sourceFIT:
		score += 500
	case sourceGPX:
		score += 250
	}
	return score
}

func enrichMissingFields(primary *strava.Activity, other *strava.Activity) *strava.Activity {
	if primary == nil || other == nil {
		return primary
	}
	if primary.AverageCadence == 0 {
		primary.AverageCadence = other.AverageCadence
	}
	if primary.AverageHeartrate == 0 {
		primary.AverageHeartrate = other.AverageHeartrate
	}
	if primary.MaxHeartrate == 0 {
		primary.MaxHeartrate = other.MaxHeartrate
	}
	if primary.AverageWatts == 0 {
		primary.AverageWatts = other.AverageWatts
	}
	if primary.WeightedAverageWatts == 0 {
		primary.WeightedAverageWatts = other.WeightedAverageWatts
	}
	if primary.Kilojoules == 0 {
		primary.Kilojoules = other.Kilojoules
	}
	if primary.ElevHigh == 0 {
		primary.ElevHigh = other.ElevHigh
	}
	if primary.TotalElevationGain == 0 {
		primary.TotalElevationGain = other.TotalElevationGain
	}
	if len(primary.StartLatlng) == 0 && len(other.StartLatlng) > 0 {
		primary.StartLatlng = other.StartLatlng
	}
	if primary.MaxSpeed == 0 {
		primary.MaxSpeed = other.MaxSpeed
	}
	if !primary.DeviceWatts {
		primary.DeviceWatts = other.DeviceWatts
	}
	return primary
}

func enrichDetailedActivity(detailed *strava.DetailedActivity, activity *strava.Activity) *strava.DetailedActivity {
	if detailed == nil || activity == nil {
		return detailed
	}
	enriched := *detailed
	if activity.Stream != nil {
		enriched.Stream = activity.Stream
	}
	if enriched.AverageCadence == 0 {
		enriched.AverageCadence = activity.AverageCadence
	}
	if enriched.AverageHeartrate == 0 {
		enriched.AverageHeartrate = activity.AverageHeartrate
	}
	if enriched.MaxHeartrate == 0 {
		enriched.MaxHeartrate = activity.MaxHeartrate
	}
	if enriched.AverageWatts == 0 {
		enriched.AverageWatts = activity.AverageWatts
	}
	if enriched.WeightedAverageWatts == 0 {
		enriched.WeightedAverageWatts = activity.WeightedAverageWatts
	}
	return &enriched
}

func cloneActivity(activity *strava.Activity) *strava.Activity {
	if activity == nil {
		return nil
	}
	cloned := *activity
	return &cloned
}

func sourceRefs(items []sourceActivity) []ActivitySourceRef {
	refs := make([]ActivitySourceRef, 0, len(items))
	for _, item := range items {
		refs = append(refs, ActivitySourceRef{
			Provider:       item.source.Name,
			ActivityID:     item.activity.Id,
			StartDateLocal: item.activity.StartDateLocal,
			Distance:       item.activity.Distance,
			MovingTime:     item.activity.MovingTime,
			HasStream:      item.activity.Stream != nil,
		})
	}
	return refs
}

func confidenceForCluster(cluster activityCluster) string {
	if len(cluster.items) <= 1 {
		return "single_source"
	}
	for _, item := range cluster.items[1:] {
		if !distanceCompatible(cluster.items[0].activity.Distance, item.activity.Distance) {
			return "medium"
		}
	}
	return "high"
}

func recordHasSource(record compositeRecord, sourceName string) bool {
	for _, source := range record.Sources {
		if source.Provider == sourceName {
			return true
		}
	}
	return false
}

func activitySortTime(activity *strava.Activity) time.Time {
	if activity == nil {
		return time.Time{}
	}
	if parsed, ok := helpers.ParseActivityDate(activity.StartDateLocal); ok {
		return parsed
	}
	if parsed, ok := helpers.ParseActivityDate(activity.StartDate); ok {
		return parsed
	}
	return time.Time{}
}

func sportFamily(activity *strava.Activity) string {
	sport := activity.SportType
	if sport == "" {
		sport = activity.Type
	}
	switch sport {
	case business.Ride.String(), business.GravelRide.String(), business.MountainBikeRide.String(), business.VirtualRide.String():
		return "ride"
	case business.Run.String(), business.TrailRun.String():
		return "run"
	case business.Hike.String(), business.Walk.String():
		return "walk"
	default:
		return sport
	}
}

func filterActivitiesByType(activities []*strava.Activity, activityTypes ...business.ActivityType) []*strava.Activity {
	if len(activityTypes) == 0 {
		return []*strava.Activity{}
	}
	filtered := make([]*strava.Activity, 0, len(activities))
	for _, activity := range activities {
		if activity == nil {
			continue
		}
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

func filterActivitiesByYear(activities []*strava.Activity, year *int) []*strava.Activity {
	if year == nil {
		return activities
	}
	filtered := make([]*strava.Activity, 0, len(activities))
	for _, activity := range activities {
		activityYear, err := strconv.Atoi(extractYear(activity.StartDateLocal))
		if err == nil && activityYear == *year {
			filtered = append(filtered, activity)
		}
	}
	return filtered
}

func groupActivitiesByYear(activities []*strava.Activity) map[string][]*strava.Activity {
	grouped := make(map[string][]*strava.Activity)
	for _, activity := range activities {
		if activity == nil {
			continue
		}
		year := extractYear(activity.StartDateLocal)
		if year == "" {
			year = extractYear(activity.StartDate)
		}
		if year != "" {
			grouped[year] = append(grouped[year], activity)
		}
	}
	return grouped
}

func availableYearBins(activities []*strava.Activity) []string {
	yearsSet := make(map[string]struct{})
	for _, activity := range activities {
		if activity == nil {
			continue
		}
		year := extractYear(activity.StartDateLocal)
		if year == "" {
			year = extractYear(activity.StartDate)
		}
		if year != "" {
			yearsSet[year] = struct{}{}
		}
	}
	years := make([]string, 0, len(yearsSet))
	for year := range yearsSet {
		years = append(years, year)
	}
	sort.Strings(years)
	return years
}

func extractYear(value string) string {
	if len(value) >= 4 {
		return value[:4]
	}
	return ""
}

func buildFilterCacheKey(year *int, activityTypes ...business.ActivityType) string {
	yearKey := "all"
	if year != nil {
		yearKey = strconv.Itoa(*year)
	}
	return fmt.Sprintf("%s:%v", yearKey, activityTypes)
}

func cloneActivityPointers(activities []*strava.Activity) []*strava.Activity {
	if len(activities) == 0 {
		return []*strava.Activity{}
	}
	cloned := make([]*strava.Activity, len(activities))
	copy(cloned, activities)
	return cloned
}

func normalizeSourceName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validLatLng(value []float64) bool {
	return len(value) >= 2 && value[0] >= -90 && value[0] <= 90 && value[1] >= -180 && value[1] <= 180
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMeters = 6371000
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
