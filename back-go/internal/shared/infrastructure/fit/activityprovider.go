package fit

import (
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

	fitparser "github.com/tormoder/fit"
)

const firstSupportedYear = 2010

type FITActivityProvider struct {
	fitDirectory          string
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

func NewFITActivityProvider(fitDirectory string) *FITActivityProvider {
	resolvedDirectory := strings.TrimSpace(fitDirectory)
	if resolvedDirectory == "" {
		resolvedDirectory = "."
	}
	resolvedDirectory = filepath.Clean(resolvedDirectory)

	clientID := deriveFITClientID(resolvedDirectory)
	firstName := deriveFirstNameFromFITDirectory(resolvedDirectory)
	athleteID := int64(hashStringToInt("athlete:" + clientID))

	localStorageProvider := localrepository.NewStravaRepository(resolvedDirectory)
	localStorageProvider.InitLocalStorageForClientId(clientID)

	provider := &FITActivityProvider{
		fitDirectory:         resolvedDirectory,
		clientID:             clientID,
		localStorageProvider: localStorageProvider,
		stravaAthlete: strava.Athlete{
			Id:        athleteID,
			Firstname: &firstName,
		},
		heartRateZoneSettings: localStorageProvider.LoadHeartRateZoneSettings(clientID),
	}

	loadedActivities := provider.loadActivitiesFromFITDirectory()
	provider.replaceActivities(loadedActivities)

	log.Printf("Initialize FITActivityProvider using %s ...", provider.fitDirectory)
	log.Printf("✅ FIT mode ready with profile=%s and %d activities", provider.clientID, len(loadedActivities))

	return provider
}

func (provider *FITActivityProvider) GetDetailedActivity(activityID int64) *strava.DetailedActivity {
	activity := provider.findActivityByID(activityID)
	if activity == nil {
		return nil
	}
	return activity.ToStravaDetailedActivity()
}

func (provider *FITActivityProvider) GetCachedDetailedActivity(activityID int64) *strava.DetailedActivity {
	return provider.GetDetailedActivity(activityID)
}

func (provider *FITActivityProvider) GetActivitiesByYearAndActivityTypes(year *int, activityTypes ...business.ActivityType) []*strava.Activity {
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

func (provider *FITActivityProvider) GetActivitiesByActivityTypeGroupByYear(activityTypes ...business.ActivityType) map[string][]*strava.Activity {
	filteredActivities := filterActivitiesByType(provider.getActivitiesSnapshot(), activityTypes...)
	return groupActivitiesByYear(filteredActivities)
}

func (provider *FITActivityProvider) GetActivitiesByActivityTypeGroupByActiveDays(activityTypes ...business.ActivityType) map[string]int {
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

func (provider *FITActivityProvider) GetAthlete() strava.Athlete {
	return provider.stravaAthlete
}

func (provider *FITActivityProvider) GetHeartRateZoneSettings() business.HeartRateZoneSettings {
	provider.dataMutex.RLock()
	defer provider.dataMutex.RUnlock()

	return provider.heartRateZoneSettings
}

func (provider *FITActivityProvider) SaveHeartRateZoneSettings(settings business.HeartRateZoneSettings) business.HeartRateZoneSettings {
	provider.dataMutex.Lock()
	provider.heartRateZoneSettings = settings
	provider.dataMutex.Unlock()

	provider.localStorageProvider.SaveHeartRateZoneSettings(provider.clientID, settings)
	return settings
}

func (provider *FITActivityProvider) CacheDiagnostics() map[string]any {
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
		"provider":          "fit",
		"fitDirectory":      provider.fitDirectory,
		"athleteId":         provider.clientID,
		"activities":        activitiesCount,
		"availableYearBins": years,
	}
}

func (provider *FITActivityProvider) ClientID() string {
	return provider.clientID
}

func (provider *FITActivityProvider) CacheRootPath() string {
	return provider.fitDirectory
}

func (provider *FITActivityProvider) loadActivitiesFromFITDirectory() []*strava.Activity {
	start := time.Now()
	loadedActivities := make([]*strava.Activity, 0)

	for year := time.Now().Year(); year >= firstSupportedYear; year-- {
		yearDirectory := filepath.Join(provider.fitDirectory, strconv.Itoa(year))
		yearEntries, err := os.ReadDir(yearDirectory)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				log.Printf("Unable to list FIT directory %s: %v", yearDirectory, err)
			}
			continue
		}

		for _, entry := range yearEntries {
			if entry.IsDir() {
				continue
			}
			if !strings.EqualFold(filepath.Ext(entry.Name()), ".fit") {
				continue
			}

			filePath := filepath.Join(yearDirectory, entry.Name())
			activity, decodeErr := decodeFITActivity(filePath, provider.stravaAthlete.Id)
			if decodeErr != nil {
				log.Printf("Unable to decode FIT activity %s: %v", filePath, decodeErr)
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

	log.Printf("Loaded %d FIT activities in %s", len(loadedActivities), time.Since(start))
	return loadedActivities
}

func decodeFITActivity(filePath string, athleteID int64) (*strava.Activity, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Unable to close FIT file %s: %v", filePath, err)
		}
	}(file)

	decodedFile, err := fitparser.Decode(file)
	if err != nil {
		return nil, err
	}
	activityFile, err := decodedFile.Activity()
	if err != nil {
		return nil, err
	}
	if activityFile == nil {
		return nil, errors.New("FIT file is not an activity file")
	}

	session := firstSession(activityFile.Sessions)
	if session == nil {
		return nil, errors.New("FIT activity has no session message")
	}

	stream := buildStreamFromFITRecords(activityFile.Records, session.StartTime)
	sportType := mapFITSportToActivityType(session.Sport, session.SubSport)
	startDate := resolveActivityStartDate(session.StartTime, activityFile.Records)

	averageSpeed := session.GetEnhancedAvgSpeedScaled()
	if averageSpeed <= 0 {
		averageSpeed = session.GetAvgSpeedScaled()
	}

	maxSpeed := session.GetEnhancedMaxSpeedScaled()
	if maxSpeed <= 0 {
		maxSpeed = session.GetMaxSpeedScaled()
	}

	distance := session.GetTotalDistanceScaled()
	if distance <= 0 && stream != nil && len(stream.Distance.Data) > 0 {
		distance = stream.Distance.Data[len(stream.Distance.Data)-1]
	}

	elapsedTime := int(math.Round(session.GetTotalElapsedTimeScaled()))
	if elapsedTime <= 0 {
		elapsedTime = int(math.Round(session.GetTotalTimerTimeScaled()))
	}
	if elapsedTime <= 0 && stream != nil && len(stream.Time.Data) > 0 {
		elapsedTime = stream.Time.Data[len(stream.Time.Data)-1]
	}

	movingTime := int(math.Round(session.GetTotalMovingTimeScaled()))
	if movingTime <= 0 {
		movingTime = int(math.Round(session.GetTotalTimerTimeScaled()))
	}
	if movingTime <= 0 {
		movingTime = elapsedTime
	}

	averageCadence := float64(asUint8(session.GetAvgCadence()))
	averageHeartRate := float64(session.AvgHeartRate)
	maxHeartRate := float64(session.MaxHeartRate)
	powerMetrics := computeFITPowerMetrics(float64(session.AvgPower), stream, elapsedTime)

	totalElevationGain := float64(session.TotalAscent)
	if totalElevationGain <= 0 && stream != nil && stream.Altitude != nil {
		totalElevationGain = computeTotalAscent(stream.Altitude.Data)
	}

	elevHigh := session.GetEnhancedMaxAltitudeScaled()
	if elevHigh <= 0 {
		elevHigh = session.GetMaxAltitudeScaled()
	}
	if elevHigh <= 0 && stream != nil && stream.Altitude != nil {
		elevHigh = maxFloat64Slice(stream.Altitude.Data)
	}

	startDateUTC := startDate.UTC()
	startDateLocal := startDate.In(time.Local)
	startLatlng := extractStartLatLng(session, stream)

	activityID := fitActivityID(filePath, startDateUTC, sportType, distance)
	name := fmt.Sprintf("%s - %s", sportType, startDateLocal.Format("2006-01-02 15:04:05"))

	return &strava.Activity{
		Athlete:              strava.AthleteRef{ID: int(athleteID)},
		AverageSpeed:         averageSpeed,
		AverageCadence:       averageCadence,
		AverageHeartrate:     averageHeartRate,
		MaxHeartrate:         maxHeartRate,
		AverageWatts:         powerMetrics.averageWatts,
		Commute:              false,
		Distance:             distance,
		DeviceWatts:          powerMetrics.hasDeviceWatts,
		ElapsedTime:          elapsedTime,
		ElevHigh:             elevHigh,
		Id:                   activityID,
		Kilojoules:           powerMetrics.kilojoules,
		MaxSpeed:             maxSpeed,
		MovingTime:           movingTime,
		Name:                 name,
		SportType:            sportType,
		StartDate:            startDateUTC.Format(time.RFC3339),
		StartDateLocal:       startDateLocal.Format(time.RFC3339),
		StartLatlng:          startLatlng,
		TotalElevationGain:   totalElevationGain,
		Type:                 sportType,
		UploadId:             activityID,
		WeightedAverageWatts: powerMetrics.weightedAverageWatts,
		Stream:               stream,
	}, nil
}

type fitPowerMetrics struct {
	averageWatts         float64
	weightedAverageWatts int
	kilojoules           float64
	hasDeviceWatts       bool
}

func computeFITPowerMetrics(sessionAveragePower float64, stream *strava.Stream, elapsedTime int) fitPowerMetrics {
	samples := fitPowerSamples(stream)
	streamAverageWatts := averageFITPower(samples)

	averageWatts := 0.0
	if sessionAveragePower > 0 {
		averageWatts = sessionAveragePower
	} else {
		averageWatts = streamAverageWatts
	}

	weightedAverageWatts := 0
	if sessionAveragePower > 0 {
		weightedAverageWatts = int(math.Round(sessionAveragePower))
	} else {
		weightedAverageWatts = int(math.Round(normalizedFITPower(samples)))
	}

	return fitPowerMetrics{
		averageWatts:         averageWatts,
		weightedAverageWatts: weightedAverageWatts,
		kilojoules:           0.8604 * averageWatts * float64(maxInt(elapsedTime, 0)) / 1000,
		hasDeviceWatts:       sessionAveragePower > 0 || len(samples) > 0,
	}
}

func fitPowerSamples(stream *strava.Stream) []float64 {
	if stream == nil || stream.Watts == nil {
		return nil
	}

	samples := make([]float64, 0, len(stream.Watts.Data))
	hasPositivePower := false
	for _, watts := range stream.Watts.Data {
		if !isFinite(watts) || watts < 0 {
			continue
		}
		if watts > 0 {
			hasPositivePower = true
		}
		samples = append(samples, watts)
	}
	if !hasPositivePower {
		return nil
	}
	return samples
}

func averageFITPower(samples []float64) float64 {
	if len(samples) == 0 {
		return 0
	}

	total := 0.0
	for _, sample := range samples {
		total += sample
	}
	return total / float64(len(samples))
}

func normalizedFITPower(samples []float64) float64 {
	if len(samples) == 0 {
		return 0
	}

	const rollingWindowSeconds = 30
	if len(samples) < rollingWindowSeconds {
		return averageFITPower(samples)
	}

	rollingTotal := 0.0
	for _, sample := range samples[:rollingWindowSeconds] {
		rollingTotal += sample
	}

	fourthPowerTotal := math.Pow(rollingTotal/rollingWindowSeconds, 4)
	rollingCount := 1
	for index := rollingWindowSeconds; index < len(samples); index++ {
		rollingTotal += samples[index] - samples[index-rollingWindowSeconds]
		fourthPowerTotal += math.Pow(rollingTotal/rollingWindowSeconds, 4)
		rollingCount++
	}

	return math.Pow(fourthPowerTotal/float64(rollingCount), 0.25)
}

func buildStreamFromFITRecords(records []*fitparser.RecordMsg, startTime time.Time) *strava.Stream {
	if len(records) == 0 {
		return nil
	}

	var firstTimestamp time.Time
	for _, record := range records {
		if record != nil && !record.Timestamp.IsZero() {
			firstTimestamp = record.Timestamp
			break
		}
	}
	if firstTimestamp.IsZero() {
		if startTime.IsZero() {
			firstTimestamp = time.Now().UTC()
		} else {
			firstTimestamp = startTime
		}
	}

	distanceData := make([]float64, 0, len(records))
	timeData := make([]int, 0, len(records))
	coordinates := make([][]float64, 0, len(records))
	altitudeData := make([]float64, 0, len(records))
	velocityData := make([]float64, 0, len(records))
	gradeData := make([]float64, 0, len(records))
	movingData := make([]bool, 0, len(records))
	cadenceData := make([]int, 0, len(records))
	heartRateData := make([]int, 0, len(records))
	powerData := make([]float64, 0, len(records))

	lastDistance := 0.0
	lastElapsedSeconds := 0
	for index, record := range records {
		if record == nil {
			continue
		}

		distance := record.GetDistanceScaled()
		if !isFinite(distance) || distance < 0 {
			distance = lastDistance
		}
		if distance < lastDistance {
			distance = lastDistance
		}
		lastDistance = distance
		distanceData = append(distanceData, distance)

		recordTimestamp := record.Timestamp
		if recordTimestamp.IsZero() {
			recordTimestamp = firstTimestamp.Add(time.Duration(index) * time.Second)
		}
		elapsedSeconds := int(math.Round(recordTimestamp.Sub(firstTimestamp).Seconds()))
		if elapsedSeconds < lastElapsedSeconds {
			elapsedSeconds = lastElapsedSeconds
		}
		lastElapsedSeconds = elapsedSeconds
		timeData = append(timeData, elapsedSeconds)

		coordinates = append(coordinates, extractCoordinate(record))

		altitude := record.GetEnhancedAltitudeScaled()
		if !isFinite(altitude) || altitude < -1000 || altitude > 12000 {
			altitude = record.GetAltitudeScaled()
		}
		if !isFinite(altitude) || altitude < -1000 || altitude > 12000 {
			altitude = 0
		}
		altitudeData = append(altitudeData, altitude)

		speed := record.GetEnhancedSpeedScaled()
		if !isFinite(speed) || speed < 0 {
			speed = record.GetSpeedScaled()
		}
		if !isFinite(speed) || speed < 0 {
			speed = 0
		}
		velocityData = append(velocityData, speed)
		movingData = append(movingData, speed > 0.1)

		grade := record.GetGradeScaled()
		if !isFinite(grade) {
			grade = 0
		}
		gradeData = append(gradeData, grade)

		cadence := int(record.Cadence)
		fractionalCadence := int(math.Round(record.GetCadence256Scaled()))
		if fractionalCadence > cadence {
			cadence = fractionalCadence
		}
		if cadence < 0 {
			cadence = 0
		}
		cadenceData = append(cadenceData, cadence)

		heartRate := int(record.HeartRate)
		if heartRate < 0 {
			heartRate = 0
		}
		heartRateData = append(heartRateData, heartRate)

		power := float64(record.Power)
		if !isFinite(power) || power < 0 {
			power = 0
		}
		powerData = append(powerData, power)
	}

	if len(distanceData) == 0 || len(timeData) == 0 {
		return nil
	}

	var latLngStream *strava.LatLngStream
	normalizedCoordinates, hasValidCoordinates := normalizeCoordinates(coordinates)
	if hasValidCoordinates {
		latLngStream = &strava.LatLngStream{
			Data:         normalizedCoordinates,
			OriginalSize: len(normalizedCoordinates),
			Resolution:   "high",
			SeriesType:   "distance",
		}
	}

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
		LatLng: latLngStream,
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

	if hasAnyFloat(altitudeData) {
		stream.Altitude = &strava.AltitudeStream{
			Data:         altitudeData,
			OriginalSize: len(altitudeData),
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

	stream.Moving = &strava.MovingStream{
		Data:         movingData,
		OriginalSize: len(movingData),
		Resolution:   "high",
		SeriesType:   "distance",
	}

	return stream
}

func extractCoordinate(record *fitparser.RecordMsg) []float64 {
	if record == nil {
		return []float64{0, 0}
	}
	if record.PositionLat.Invalid() || record.PositionLong.Invalid() {
		return []float64{0, 0}
	}
	lat := record.PositionLat.Degrees()
	lng := record.PositionLong.Degrees()
	if !isFinite(lat) || !isFinite(lng) {
		return []float64{0, 0}
	}
	return []float64{lat, lng}
}

func normalizeCoordinates(coordinates [][]float64) ([][]float64, bool) {
	if len(coordinates) == 0 {
		return nil, false
	}

	normalized := make([][]float64, len(coordinates))
	copy(normalized, coordinates)

	firstValidIndex := -1
	for index, coordinate := range normalized {
		if isCoordinateValid(coordinate) {
			firstValidIndex = index
			break
		}
	}
	if firstValidIndex < 0 {
		return nil, false
	}

	firstValid := normalized[firstValidIndex]
	for i := 0; i < firstValidIndex; i++ {
		normalized[i] = []float64{firstValid[0], firstValid[1]}
	}

	lastValid := firstValid
	index := firstValidIndex + 1
	for index < len(normalized) {
		if isCoordinateValid(normalized[index]) {
			lastValid = normalized[index]
			index++
			continue
		}

		nextValidIndex := -1
		for lookAhead := index + 1; lookAhead < len(normalized); lookAhead++ {
			if isCoordinateValid(normalized[lookAhead]) {
				nextValidIndex = lookAhead
				break
			}
		}

		fillCoordinate := []float64{lastValid[0], lastValid[1]}
		if nextValidIndex > 0 {
			next := normalized[nextValidIndex]
			fillCoordinate = []float64{
				(lastValid[0] + next[0]) / 2,
				(lastValid[1] + next[1]) / 2,
			}
		}
		normalized[index] = fillCoordinate
		lastValid = fillCoordinate
		index++
	}

	return normalized, true
}

func firstSession(sessions []*fitparser.SessionMsg) *fitparser.SessionMsg {
	for _, session := range sessions {
		if session != nil {
			return session
		}
	}
	return nil
}

func resolveActivityStartDate(startTime time.Time, records []*fitparser.RecordMsg) time.Time {
	if !startTime.IsZero() {
		return startTime
	}
	for _, record := range records {
		if record != nil && !record.Timestamp.IsZero() {
			return record.Timestamp
		}
	}
	return time.Now().UTC()
}

func mapFITSportToActivityType(sport fitparser.Sport, subSport fitparser.SubSport) string {
	switch sport {
	case fitparser.SportCycling:
		switch subSport {
		case fitparser.SubSportMountain:
			return business.MountainBikeRide.String()
		case fitparser.SubSportGravelCycling:
			return business.GravelRide.String()
		case fitparser.SubSportVirtualActivity:
			return business.VirtualRide.String()
		default:
			return business.Ride.String()
		}
	case fitparser.SportRunning:
		switch subSport {
		case fitparser.SubSportTrail:
			return business.TrailRun.String()
		default:
			return business.Run.String()
		}
	case fitparser.SportHiking:
		return business.Hike.String()
	case fitparser.SportAlpineSkiing:
		return business.AlpineSki.String()
	case fitparser.SportInlineSkating:
		return business.InlineSkate.String()
	case fitparser.SportEBiking:
		return business.VirtualRide.String()
	default:
		return business.Ride.String()
	}
}

func deriveFITClientID(fitDirectory string) string {
	base := strings.TrimSpace(filepath.Base(fitDirectory))
	base = strings.ToLower(base)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "fit-local"
	}
	base = strings.ReplaceAll(base, " ", "-")
	return base
}

func deriveFirstNameFromFITDirectory(fitDirectory string) string {
	base := strings.TrimSpace(filepath.Base(fitDirectory))
	if strings.HasPrefix(strings.ToLower(base), "fit-") && len(base) > 4 {
		return base[4:]
	}
	if base != "" && base != "." {
		return base
	}
	return "FIT User"
}

func fitActivityID(filePath string, startDate time.Time, sportType string, distanceMeters float64) int64 {
	identity := fmt.Sprintf("%s|%s|%s|%.3f", filePath, startDate.UTC().Format(time.RFC3339), sportType, distanceMeters)
	return int64(hashStringToInt(identity))
}

func hashStringToInt(value string) int {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(value))
	return int(hasher.Sum32())
}

func extractStartLatLng(session *fitparser.SessionMsg, stream *strava.Stream) []float64 {
	if session != nil && !session.StartPositionLat.Invalid() && !session.StartPositionLong.Invalid() {
		return []float64{session.StartPositionLat.Degrees(), session.StartPositionLong.Degrees()}
	}
	if stream != nil && stream.LatLng != nil && len(stream.LatLng.Data) > 0 {
		first := stream.LatLng.Data[0]
		if isCoordinateValid(first) {
			return []float64{first[0], first[1]}
		}
	}
	return nil
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

	if len(activitiesByYear) == 0 {
		return activitiesByYear
	}

	minYear, _ := strconv.Atoi(minYearKey(activitiesByYear))
	maxYear, _ := strconv.Atoi(maxYearKey(activitiesByYear))
	for year := minYear; year <= maxYear; year++ {
		key := strconv.Itoa(year)
		if _, exists := activitiesByYear[key]; !exists {
			activitiesByYear[key] = []*strava.Activity{}
		}
	}

	return activitiesByYear
}

func minYearKey(values map[string][]*strava.Activity) string {
	minKey := ""
	for key := range values {
		if minKey == "" || key < minKey {
			minKey = key
		}
	}
	return minKey
}

func maxYearKey(values map[string][]*strava.Activity) string {
	maxKey := ""
	for key := range values {
		if maxKey == "" || key > maxKey {
			maxKey = key
		}
	}
	return maxKey
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

func (provider *FITActivityProvider) findActivityByID(activityID int64) *strava.Activity {
	provider.dataMutex.RLock()
	defer provider.dataMutex.RUnlock()
	if provider.activityByID == nil {
		return nil
	}
	return provider.activityByID[activityID]
}

func (provider *FITActivityProvider) replaceActivities(activities []*strava.Activity) {
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

func (provider *FITActivityProvider) getActivitiesSnapshot() []*strava.Activity {
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

func asUint8(value interface{}) uint8 {
	switch typed := value.(type) {
	case uint8:
		return typed
	case uint16:
		return uint8(typed)
	case int:
		if typed < 0 {
			return 0
		}
		if typed > 255 {
			return 255
		}
		return uint8(typed)
	case float64:
		if typed < 0 {
			return 0
		}
		if typed > 255 {
			return 255
		}
		return uint8(math.Round(typed))
	default:
		return 0
	}
}

func computeTotalAscent(altitudes []float64) float64 {
	total := 0.0
	for index := 1; index < len(altitudes); index++ {
		delta := altitudes[index] - altitudes[index-1]
		if delta > 0 {
			total += delta
		}
	}
	return total
}

func maxFloat64Slice(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	maximum := values[0]
	for _, value := range values[1:] {
		if value > maximum {
			maximum = value
		}
	}
	return maximum
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
