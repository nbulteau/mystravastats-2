package gpx

import (
	"encoding/xml"
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"mystravastats/internal/helpers"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"mystravastats/internal/shared/infrastructure/localrepository"
)

const firstSupportedYear = 2010

type GPXActivityProvider struct {
	gpxDirectory          string
	clientID              string
	stravaAthlete         strava.Athlete
	activities            []*strava.Activity
	activityByID          map[int64]*strava.Activity
	filteredActivities    map[string][]*strava.Activity
	heartRateZoneSettings business.HeartRateZoneSettings
	localStorageProvider  *localrepository.StravaRepository
	dataMutex             sync.RWMutex
	cacheMutex            sync.RWMutex
}

func NewGPXActivityProvider(gpxDirectory string) *GPXActivityProvider {
	resolvedDirectory := strings.TrimSpace(gpxDirectory)
	if resolvedDirectory == "" {
		resolvedDirectory = "."
	}
	resolvedDirectory = filepath.Clean(resolvedDirectory)

	clientID := deriveGPXClientID(resolvedDirectory)
	firstName := deriveFirstNameFromGPXDirectory(resolvedDirectory)
	athleteID := int64(hashStringToInt("athlete:" + clientID))

	localStorageProvider := localrepository.NewStravaRepository(resolvedDirectory)
	localStorageProvider.InitLocalStorageForClientId(clientID)

	provider := &GPXActivityProvider{
		gpxDirectory:         resolvedDirectory,
		clientID:             clientID,
		localStorageProvider: localStorageProvider,
		stravaAthlete: strava.Athlete{
			Id:        athleteID,
			Firstname: &firstName,
		},
		heartRateZoneSettings: localStorageProvider.LoadHeartRateZoneSettings(clientID),
	}

	loadedActivities := provider.loadActivitiesFromGPXDirectory()
	provider.replaceActivities(loadedActivities)

	log.Printf("Initialize GPXActivityProvider using %s ...", provider.gpxDirectory)
	log.Printf("✅ GPX mode ready with profile=%s and %d activities", provider.clientID, len(loadedActivities))

	return provider
}

func (provider *GPXActivityProvider) GetDetailedActivity(activityID int64) *strava.DetailedActivity {
	activity := provider.findActivityByID(activityID)
	if activity == nil {
		return nil
	}
	return activity.ToStravaDetailedActivity()
}

func (provider *GPXActivityProvider) GetCachedDetailedActivity(activityID int64) *strava.DetailedActivity {
	return provider.GetDetailedActivity(activityID)
}

func (provider *GPXActivityProvider) GetActivitiesByYearAndActivityTypes(year *int, activityTypes ...business.ActivityType) []*strava.Activity {
	cacheKey := buildFilterCacheKey(year, activityTypes...)
	provider.cacheMutex.RLock()
	if cachedActivities, ok := provider.filteredActivities[cacheKey]; ok {
		provider.cacheMutex.RUnlock()
		return cloneActivityPointers(cachedActivities)
	}
	provider.cacheMutex.RUnlock()

	filteredActivities := filterActivitiesByYear(provider.getActivitiesSnapshot(), year)
	filteredActivities = filterActivitiesByType(filteredActivities, activityTypes...)

	provider.cacheMutex.Lock()
	provider.filteredActivities[cacheKey] = filteredActivities
	provider.cacheMutex.Unlock()

	return cloneActivityPointers(filteredActivities)
}

func (provider *GPXActivityProvider) GetActivitiesByActivityTypeGroupByYear(activityTypes ...business.ActivityType) map[string][]*strava.Activity {
	filteredActivities := filterActivitiesByType(provider.getActivitiesSnapshot(), activityTypes...)
	return groupActivitiesByYear(filteredActivities)
}

func (provider *GPXActivityProvider) GetActivitiesByActivityTypeGroupByActiveDays(activityTypes ...business.ActivityType) map[string]int {
	filteredActivities := filterActivitiesByType(provider.getActivitiesSnapshot(), activityTypes...)
	result := make(map[string]int)
	for _, activity := range filteredActivities {
		date := extractSortableDay(activity.StartDateLocal)
		if date == "" {
			continue
		}
		result[date] += int(activity.Distance / 1000)
	}
	return result
}

func (provider *GPXActivityProvider) GetAthlete() strava.Athlete {
	return provider.stravaAthlete
}

func (provider *GPXActivityProvider) GetHeartRateZoneSettings() business.HeartRateZoneSettings {
	provider.dataMutex.RLock()
	defer provider.dataMutex.RUnlock()

	return provider.heartRateZoneSettings
}

func (provider *GPXActivityProvider) SaveHeartRateZoneSettings(settings business.HeartRateZoneSettings) business.HeartRateZoneSettings {
	provider.dataMutex.Lock()
	provider.heartRateZoneSettings = settings
	provider.dataMutex.Unlock()

	provider.localStorageProvider.SaveHeartRateZoneSettings(provider.clientID, settings)
	return settings
}

func (provider *GPXActivityProvider) CacheDiagnostics() map[string]any {
	provider.dataMutex.RLock()
	activitiesCount := len(provider.activities)
	provider.dataMutex.RUnlock()

	yearsSet := make(map[string]struct{})
	for _, activity := range provider.getActivitiesSnapshot() {
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

	return map[string]any{
		"timestamp":         time.Now().UTC().Format(time.RFC3339),
		"provider":          "gpx",
		"gpxDirectory":      provider.gpxDirectory,
		"athleteId":         provider.clientID,
		"activities":        activitiesCount,
		"availableYearBins": years,
	}
}

func (provider *GPXActivityProvider) ClientID() string {
	return provider.clientID
}

func (provider *GPXActivityProvider) CacheRootPath() string {
	return provider.gpxDirectory
}

func (provider *GPXActivityProvider) loadActivitiesFromGPXDirectory() []*strava.Activity {
	start := time.Now()
	loadedActivities := make([]*strava.Activity, 0)

	for year := time.Now().Year(); year >= firstSupportedYear; year-- {
		yearDirectory := filepath.Join(provider.gpxDirectory, strconv.Itoa(year))
		yearEntries, err := os.ReadDir(yearDirectory)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				log.Printf("Unable to list GPX directory %s: %v", yearDirectory, err)
			}
			continue
		}

		for _, entry := range yearEntries {
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".gpx") {
				continue
			}

			filePath := filepath.Join(yearDirectory, entry.Name())
			activity, decodeErr := DecodeGPXActivity(filePath, provider.stravaAthlete.Id, year)
			if decodeErr != nil {
				log.Printf("Unable to decode GPX activity %s: %v", filePath, decodeErr)
				continue
			}
			loadedActivities = append(loadedActivities, activity)
		}
	}

	sort.SliceStable(loadedActivities, func(i, j int) bool {
		left, leftOK := helpers.ParseActivityDate(loadedActivities[i].StartDateLocal)
		right, rightOK := helpers.ParseActivityDate(loadedActivities[j].StartDateLocal)
		switch {
		case leftOK && rightOK:
			return left.After(right)
		case leftOK && !rightOK:
			return true
		case !leftOK && rightOK:
			return false
		default:
			return loadedActivities[i].StartDateLocal > loadedActivities[j].StartDateLocal
		}
	})

	log.Printf("Loaded %d GPX activities in %s", len(loadedActivities), time.Since(start))
	return loadedActivities
}

type gpxDocument struct {
	Tracks []gpxTrack `xml:"trk"`
}

type gpxTrack struct {
	Name     string       `xml:"name"`
	Type     string       `xml:"type"`
	Segments []gpxSegment `xml:"trkseg"`
}

type gpxSegment struct {
	Points []gpxTrackPoint `xml:"trkpt"`
}

type gpxTrackPoint struct {
	Latitude   float64       `xml:"lat,attr"`
	Longitude  float64       `xml:"lon,attr"`
	Elevation  string        `xml:"ele"`
	Time       string        `xml:"time"`
	Extensions gpxExtensions `xml:"extensions"`
}

type gpxExtensions struct {
	InnerXML string `xml:",innerxml"`
}

type parsedGPXPoint struct {
	latitude     float64
	longitude    float64
	elevation    float64
	hasElevation bool
	timestamp    time.Time
	heartRate    int
	cadence      int
	watts        float64
}

func DecodeGPXActivity(filePath string, athleteID int64, fallbackYear int) (*strava.Activity, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var document gpxDocument
	if err := xml.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	if len(document.Tracks) == 0 {
		return nil, errors.New("GPX file has no track")
	}

	points := flattenTrackPoints(document.Tracks)
	if len(points) < 2 {
		return nil, errors.New("GPX file must contain at least 2 track points")
	}

	startTime := firstGPXTimestamp(points)
	if startTime.IsZero() {
		startTime = fallbackGPXStartTime(filePath, fallbackYear)
	}

	stream, stats := buildGPXStream(points, startTime)
	if stream == nil || len(stream.Distance.Data) == 0 || len(stream.Time.Data) == 0 {
		return nil, errors.New("GPX file has no usable stream")
	}

	firstTrack := document.Tracks[0]
	sportType := mapGPXTypeToActivityType(firstTrack.Type)
	name := strings.TrimSpace(firstTrack.Name)
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	}

	elapsedTime := stats.elapsedTime
	if elapsedTime <= 0 && len(stream.Time.Data) > 0 {
		elapsedTime = stream.Time.Data[len(stream.Time.Data)-1]
	}
	if elapsedTime <= 0 {
		elapsedTime = len(stream.Time.Data) - 1
	}

	movingTime := stats.movingTime
	if movingTime <= 0 {
		movingTime = elapsedTime
	}

	averageSpeed := 0.0
	if elapsedTime > 0 {
		averageSpeed = stats.distanceMeters / float64(elapsedTime)
	}

	startDateUTC := startTime.UTC()
	startDateLocal := startDateUTC.In(time.Local)
	activityID := gpxActivityID(filePath, startDateUTC, sportType, stats.distanceMeters)

	return &strava.Activity{
		Athlete:              strava.AthleteRef{ID: int(athleteID)},
		AverageSpeed:         averageSpeed,
		AverageCadence:       averageInt(stats.cadenceData),
		AverageHeartrate:     averageInt(stats.heartRateData),
		MaxHeartrate:         float64(maxIntSlice(stats.heartRateData)),
		AverageWatts:         averageFloat(stats.powerData),
		Commute:              false,
		Distance:             stats.distanceMeters,
		DeviceWatts:          hasAnyFloat(stats.powerData),
		ElapsedTime:          elapsedTime,
		ElevHigh:             maxFloat64Slice(stats.altitudeData),
		Id:                   activityID,
		Kilojoules:           0.8604 * averageFloat(stats.powerData) * float64(elapsedTime) / 1000,
		MaxSpeed:             maxFloat64Slice(stats.velocityData),
		MovingTime:           movingTime,
		Name:                 name,
		SportType:            sportType,
		StartDate:            startDateUTC.Format(time.RFC3339),
		StartDateLocal:       startDateLocal.Format(time.RFC3339),
		StartLatlng:          []float64{points[0].latitude, points[0].longitude},
		TotalElevationGain:   stats.elevationGainMeters,
		Type:                 sportType,
		UploadId:             activityID,
		WeightedAverageWatts: int(math.Round(averageFloat(stats.powerData))),
		Stream:               stream,
	}, nil
}

func flattenTrackPoints(tracks []gpxTrack) []parsedGPXPoint {
	points := make([]parsedGPXPoint, 0)
	for _, track := range tracks {
		for _, segment := range track.Segments {
			for _, point := range segment.Points {
				parsedPoint := parsedGPXPoint{
					latitude:  point.Latitude,
					longitude: point.Longitude,
					timestamp: parseGPXTime(point.Time),
					heartRate: extensionInt(point.Extensions.InnerXML, "hr"),
					cadence:   extensionInt(point.Extensions.InnerXML, "cad"),
					watts:     float64(extensionInt(point.Extensions.InnerXML, "power")),
				}
				if elevation, ok := parseOptionalFloat(point.Elevation); ok {
					parsedPoint.elevation = elevation
					parsedPoint.hasElevation = true
				}
				if isCoordinateValid([]float64{parsedPoint.latitude, parsedPoint.longitude}) {
					points = append(points, parsedPoint)
				}
			}
		}
	}
	return points
}

type gpxStreamStats struct {
	distanceMeters      float64
	elevationGainMeters float64
	elapsedTime         int
	movingTime          int
	altitudeData        []float64
	velocityData        []float64
	heartRateData       []int
	cadenceData         []int
	powerData           []float64
}

func buildGPXStream(points []parsedGPXPoint, startTime time.Time) (*strava.Stream, gpxStreamStats) {
	if len(points) == 0 {
		return nil, gpxStreamStats{}
	}

	distanceData := make([]float64, 0, len(points))
	timeData := make([]int, 0, len(points))
	latLngData := make([][]float64, 0, len(points))
	altitudeData := make([]float64, 0, len(points))
	velocityData := make([]float64, 0, len(points))
	gradeData := make([]float64, 0, len(points))
	movingData := make([]bool, 0, len(points))
	cadenceData := make([]int, 0, len(points))
	heartRateData := make([]int, 0, len(points))
	powerData := make([]float64, 0, len(points))

	stats := gpxStreamStats{}
	previous := points[0]
	previousTime := resolvePointTime(previous, startTime, 0)
	lastElapsedSeconds := 0

	for index, point := range points {
		pointTime := resolvePointTime(point, startTime, index)
		if pointTime.Before(previousTime) {
			pointTime = previousTime
		}

		deltaDistance := 0.0
		deltaSeconds := 0
		elevationDelta := 0.0
		if index > 0 {
			deltaDistance = haversineMeters(previous.latitude, previous.longitude, point.latitude, point.longitude)
			deltaSeconds = int(math.Round(pointTime.Sub(previousTime).Seconds()))
			if deltaSeconds < 0 {
				deltaSeconds = 0
			}
			if point.hasElevation && previous.hasElevation {
				elevationDelta = point.elevation - previous.elevation
				if elevationDelta > 0 {
					stats.elevationGainMeters += elevationDelta
				}
			}
		}

		stats.distanceMeters += deltaDistance
		distanceData = append(distanceData, stats.distanceMeters)
		latLngData = append(latLngData, []float64{point.latitude, point.longitude})

		elapsedSeconds := int(math.Round(pointTime.Sub(startTime).Seconds()))
		if elapsedSeconds < lastElapsedSeconds {
			elapsedSeconds = lastElapsedSeconds
		}
		lastElapsedSeconds = elapsedSeconds
		timeData = append(timeData, elapsedSeconds)
		stats.elapsedTime = elapsedSeconds

		if point.hasElevation {
			altitudeData = append(altitudeData, point.elevation)
		} else {
			altitudeData = append(altitudeData, 0)
		}

		speed := 0.0
		if deltaSeconds > 0 {
			speed = deltaDistance / float64(deltaSeconds)
		}
		velocityData = append(velocityData, speed)

		grade := 0.0
		if deltaDistance > 0 {
			grade = elevationDelta / deltaDistance
		}
		gradeData = append(gradeData, grade)

		moving := deltaDistance > 0.5
		movingData = append(movingData, moving)
		if moving {
			stats.movingTime += deltaSeconds
		}

		cadenceData = append(cadenceData, point.cadence)
		heartRateData = append(heartRateData, point.heartRate)
		powerData = append(powerData, point.watts)

		previous = point
		previousTime = pointTime
	}

	stats.altitudeData = altitudeData
	stats.velocityData = velocityData
	stats.heartRateData = heartRateData
	stats.cadenceData = cadenceData
	stats.powerData = powerData

	stream := &strava.Stream{
		Distance: strava.DistanceStream{
			Data:         distanceData,
			OriginalSize: len(distanceData),
			Resolution:   "high",
			SeriesType:   "distance",
		},
		Time: strava.TimeStream{
			Data:         timeData,
			OriginalSize: len(timeData),
			Resolution:   "high",
			SeriesType:   "distance",
		},
		LatLng: &strava.LatLngStream{
			Data:         latLngData,
			OriginalSize: len(latLngData),
			Resolution:   "high",
			SeriesType:   "distance",
		},
		Moving: &strava.MovingStream{
			Data:         movingData,
			OriginalSize: len(movingData),
			Resolution:   "high",
			SeriesType:   "distance",
		},
	}

	if hasAnyFloat(altitudeData) {
		stream.Altitude = &strava.AltitudeStream{
			Data:         altitudeData,
			OriginalSize: len(altitudeData),
			Resolution:   "high",
			SeriesType:   "distance",
		}
	}
	if hasAnyFloat(velocityData) {
		stream.VelocitySmooth = &strava.SmoothVelocityStream{
			Data:         velocityData,
			OriginalSize: len(velocityData),
			Resolution:   "high",
			SeriesType:   "distance",
		}
	}
	if hasAnyFloat(gradeData) {
		stream.GradeSmooth = &strava.SmoothGradeStream{
			Data:         gradeData,
			OriginalSize: len(gradeData),
			Resolution:   "high",
			SeriesType:   "distance",
		}
	}
	if hasAnyInt(cadenceData) {
		stream.Cadence = &strava.CadenceStream{
			Data:         cadenceData,
			OriginalSize: len(cadenceData),
			Resolution:   "high",
			SeriesType:   "distance",
		}
	}
	if hasAnyInt(heartRateData) {
		stream.HeartRate = &strava.HeartRateStream{
			Data:         heartRateData,
			OriginalSize: len(heartRateData),
			Resolution:   "high",
			SeriesType:   "distance",
		}
	}
	if hasAnyFloat(powerData) {
		stream.Watts = &strava.PowerStream{
			Data:         powerData,
			OriginalSize: len(powerData),
			Resolution:   "high",
			SeriesType:   "distance",
		}
	}

	return stream, stats
}

func firstGPXTimestamp(points []parsedGPXPoint) time.Time {
	for _, point := range points {
		if !point.timestamp.IsZero() {
			return point.timestamp
		}
	}
	return time.Time{}
}

func resolvePointTime(point parsedGPXPoint, startTime time.Time, index int) time.Time {
	if !point.timestamp.IsZero() {
		return point.timestamp
	}
	return startTime.Add(time.Duration(index) * time.Second)
}

func fallbackGPXStartTime(filePath string, fallbackYear int) time.Time {
	if info, err := os.Stat(filePath); err == nil && !info.ModTime().IsZero() {
		return info.ModTime().UTC()
	}
	if fallbackYear < firstSupportedYear {
		fallbackYear = time.Now().Year()
	}
	return time.Date(fallbackYear, time.January, 1, 0, 0, 0, 0, time.UTC)
}

func parseGPXTime(value string) time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z0700",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func parseOptionalFloat(value string) (float64, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || !isFinite(parsed) {
		return 0, false
	}
	return parsed, true
}

func extensionInt(rawXML string, localName string) int {
	if strings.TrimSpace(rawXML) == "" {
		return 0
	}
	decoder := xml.NewDecoder(strings.NewReader("<root>" + rawXML + "</root>"))
	decoder.Strict = false
	for {
		token, err := decoder.Token()
		if err != nil {
			return 0
		}
		start, ok := token.(xml.StartElement)
		if !ok || !strings.EqualFold(start.Name.Local, localName) {
			continue
		}
		var text string
		if err := decoder.DecodeElement(&text, &start); err != nil {
			return 0
		}
		parsed, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil || !isFinite(parsed) {
			return 0
		}
		return int(math.Round(parsed))
	}
}

func mapGPXTypeToActivityType(gpxType string) string {
	switch strings.ToLower(strings.TrimSpace(gpxType)) {
	case "cycling", "biking", "bike", "ride":
		return business.Ride.String()
	case "running", "run":
		return business.Run.String()
	case "trailrun", "trail_run", "trail running":
		return business.TrailRun.String()
	case "walking", "walk":
		return business.Walk.String()
	case "hiking", "hike":
		return business.Hike.String()
	case "ski", "skiing", "alpineski", "alpine skiing":
		return business.AlpineSki.String()
	case "inlineskate", "inline skating", "skate":
		return business.InlineSkate.String()
	default:
		return business.Ride.String()
	}
}

func deriveGPXClientID(gpxDirectory string) string {
	base := strings.TrimSpace(filepath.Base(gpxDirectory))
	base = strings.ToLower(base)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "gpx-local"
	}
	base = strings.ReplaceAll(base, " ", "-")
	return base
}

func deriveFirstNameFromGPXDirectory(gpxDirectory string) string {
	base := strings.TrimSpace(filepath.Base(gpxDirectory))
	if strings.HasPrefix(strings.ToLower(base), "gpx-") && len(base) > 4 {
		return base[4:]
	}
	if base != "" && base != "." {
		return base
	}
	return "GPX User"
}

func gpxActivityID(filePath string, startDate time.Time, sportType string, distanceMeters float64) int64 {
	identity := fmt.Sprintf("%s|%s|%s|%.3f", filePath, startDate.UTC().Format(time.RFC3339), sportType, distanceMeters)
	return int64(hashStringToInt(identity))
}

func hashStringToInt(value string) int {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(value))
	return int(hasher.Sum32())
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
		if activity == nil {
			continue
		}
		activityYear, err := strconv.Atoi(extractYear(activity.StartDateLocal))
		if err != nil {
			continue
		}
		if activityYear == *year {
			filtered = append(filtered, activity)
		}
	}
	return filtered
}

func groupActivitiesByYear(activities []*strava.Activity) map[string][]*strava.Activity {
	activitiesByYear := make(map[string][]*strava.Activity)
	for _, activity := range activities {
		if activity == nil {
			continue
		}
		year := extractYear(activity.StartDateLocal)
		if year == "" {
			year = extractYear(activity.StartDate)
		}
		if year == "" {
			continue
		}
		activitiesByYear[year] = append(activitiesByYear[year], activity)
	}
	return activitiesByYear
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

func (provider *GPXActivityProvider) findActivityByID(activityID int64) *strava.Activity {
	provider.dataMutex.RLock()
	defer provider.dataMutex.RUnlock()
	if provider.activityByID == nil {
		return nil
	}
	return provider.activityByID[activityID]
}

func (provider *GPXActivityProvider) replaceActivities(activities []*strava.Activity) {
	provider.dataMutex.Lock()
	provider.activities = activities
	provider.activityByID = make(map[int64]*strava.Activity, len(activities))
	for _, activity := range activities {
		if activity == nil {
			continue
		}
		provider.activityByID[activity.Id] = activity
	}
	provider.dataMutex.Unlock()

	provider.cacheMutex.Lock()
	provider.filteredActivities = make(map[string][]*strava.Activity)
	provider.cacheMutex.Unlock()
}

func (provider *GPXActivityProvider) getActivitiesSnapshot() []*strava.Activity {
	provider.dataMutex.RLock()
	defer provider.dataMutex.RUnlock()

	snapshot := make([]*strava.Activity, len(provider.activities))
	copy(snapshot, provider.activities)
	return snapshot
}

func extractYear(value string) string {
	if len(value) >= 4 {
		return value[:4]
	}
	return ""
}

func extractSortableDay(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < 10 {
		return ""
	}
	day := trimmed[:10]
	if _, err := time.Parse("2006-01-02", day); err != nil {
		return ""
	}
	return day
}

func isCoordinateValid(coordinate []float64) bool {
	if len(coordinate) < 2 {
		return false
	}
	lat := coordinate[0]
	lng := coordinate[1]
	if !isFinite(lat) || !isFinite(lng) {
		return false
	}
	return !(lat == 0 && lng == 0)
}

func hasAnyFloat(values []float64) bool {
	for _, value := range values {
		if isFinite(value) && math.Abs(value) > 0 {
			return true
		}
	}
	return false
}

func hasAnyInt(values []int) bool {
	for _, value := range values {
		if value > 0 {
			return true
		}
	}
	return false
}

func averageInt(values []int) float64 {
	sum := 0
	count := 0
	for _, value := range values {
		if value > 0 {
			sum += value
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return float64(sum) / float64(count)
}

func averageFloat(values []float64) float64 {
	sum := 0.0
	count := 0
	for _, value := range values {
		if value > 0 && isFinite(value) {
			sum += value
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func maxIntSlice(values []int) int {
	maximum := 0
	for _, value := range values {
		if value > maximum {
			maximum = value
		}
	}
	return maximum
}

func maxFloat64Slice(values []float64) float64 {
	maximum := 0.0
	for _, value := range values {
		if value > maximum && isFinite(value) {
			maximum = value
		}
	}
	return maximum
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMeters = 6371e3
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusMeters * c
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
