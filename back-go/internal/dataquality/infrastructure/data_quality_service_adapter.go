package infrastructure

import (
	"fmt"
	"math"
	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"sort"
	"strings"
	"time"
)

const (
	maxTopIssues      = 5
	altitudeSpikeM    = 120.0
	altitudeSpikeSecs = 15
	defaultMaxSpeedMS = 35.0
	earthRadiusMeters = 6371e3
)

type DataQualityServiceAdapter struct{}

func NewDataQualityServiceAdapter() *DataQualityServiceAdapter {
	return &DataQualityServiceAdapter{}
}

func (adapter *DataQualityServiceAdapter) GetDataQualityReport() business.DataQualityReport {
	return CurrentProviderReport()
}

func (adapter *DataQualityServiceAdapter) ExcludeActivityFromStats(activityID int64, reason string) (business.DataQualityReport, error) {
	return excludeCurrentProviderActivityFromStats(activityID, reason)
}

func (adapter *DataQualityServiceAdapter) IncludeActivityInStats(activityID int64) (business.DataQualityReport, error) {
	return includeCurrentProviderActivityInStats(activityID)
}

func (adapter *DataQualityServiceAdapter) PreviewCorrection(issueID string) (business.DataQualityCorrectionPreview, error) {
	return CurrentProviderCorrectionPreview(issueID)
}

func (adapter *DataQualityServiceAdapter) PreviewSafeCorrections() business.DataQualityCorrectionPreview {
	return CurrentProviderSafeCorrectionPreview()
}

func (adapter *DataQualityServiceAdapter) ApplyCorrection(issueID string) (business.DataQualityReport, error) {
	return ApplyCurrentProviderCorrection(issueID)
}

func (adapter *DataQualityServiceAdapter) ApplySafeCorrections() (business.DataQualityReport, error) {
	return ApplyCurrentProviderSafeCorrections()
}

func (adapter *DataQualityServiceAdapter) RevertCorrection(correctionID string) (business.DataQualityReport, error) {
	return RevertCurrentProviderCorrection(correctionID)
}

func CurrentProviderReport() business.DataQualityReport {
	provider := activityprovider.Get()
	diagnostics := provider.CacheDiagnostics()
	source := strings.ToLower(fmt.Sprint(diagnostics["provider"]))
	sourcePath := provider.CacheRootPath()

	activities := provider.GetActivitiesByYearAndActivityTypes(nil, allActivityTypes()...)
	exclusions := loadExclusions(provider.CacheRootPath(), provider.ClientID())
	corrections := loadCorrections(provider.CacheRootPath(), provider.ClientID())
	correctedActivities := ApplyCorrectionsToActivities(activities, corrections)
	return AnalyzeActivitiesWithCorrections(source, sourcePath, correctedActivities, exclusions, corrections)
}

func AnalyzeLocalActivities(source string, sourcePath string, activities []*strava.Activity) business.DataQualityReport {
	return AnalyzeActivities(source, sourcePath, activities, []business.DataQualityExclusion{})
}

func AnalyzeActivities(source string, sourcePath string, activities []*strava.Activity, exclusions []business.DataQualityExclusion) business.DataQualityReport {
	return AnalyzeActivitiesWithCorrections(source, sourcePath, activities, exclusions, []business.DataQualityCorrection{})
}

func AnalyzeActivitiesWithCorrections(source string, sourcePath string, activities []*strava.Activity, exclusions []business.DataQualityExclusion, corrections []business.DataQualityCorrection) business.DataQualityReport {
	source = strings.ToLower(strings.TrimSpace(source))
	issues := make([]business.DataQualityIssue, 0)
	exclusionsByID := exclusionsByActivityID(exclusions)
	correctionsByIssue := activeCorrectionsByIssueID(corrections)
	for _, activity := range activities {
		activityIssues := analyzeActivity(source, sourcePath, activity)
		activityIssues = markIssueExclusions(activityIssues, exclusionsByID)
		activityIssues = markIssueCorrections(activityIssues, correctionsByIssue)
		issues = append(issues, activityIssues...)
	}

	sortIssues(issues)
	return business.DataQualityReport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Summary:     buildSummary(source, issues, exclusions, corrections),
		Issues:      issues,
		Exclusions:  sortedExclusions(exclusionsByID),
		Corrections: sortedCorrections(corrections),
	}
}

func analyzeActivity(source string, sourcePath string, activity *strava.Activity) []business.DataQualityIssue {
	if activity == nil {
		return []business.DataQualityIssue{
			newIssue(source, sourcePath, nil, business.DataQualitySeverityCritical, business.DataQualityCategoryInvalidValue, "activity", "Activity is nil", "", "Ignore this record and inspect the source file."),
		}
	}

	issues := make([]business.DataQualityIssue, 0)
	if activity.Distance <= 0 || invalidFloat(activity.Distance) {
		issues = append(issues, newIssue(source, sourcePath, activity, business.DataQualitySeverityCritical, business.DataQualityCategoryInvalidValue, "distance", "Activity distance is missing or invalid.", formatFloat(activity.Distance), "Check the source file or exclude the activity from statistics."))
	}
	if activity.ElapsedTime <= 0 {
		issues = append(issues, newIssue(source, sourcePath, activity, business.DataQualitySeverityCritical, business.DataQualityCategoryInvalidValue, "elapsed_time", "Elapsed time is missing or invalid.", fmt.Sprintf("%d", activity.ElapsedTime), "Check the source file timing data."))
	}
	if activity.MovingTime <= 0 {
		issues = append(issues, newIssue(source, sourcePath, activity, business.DataQualitySeverityWarning, business.DataQualityCategoryInvalidValue, "moving_time", "Moving time is missing or zero.", fmt.Sprintf("%d", activity.MovingTime), "Use elapsed time as fallback only if the activity has no pauses."))
	}
	if activity.MovingTime > activity.ElapsedTime && activity.ElapsedTime > 0 {
		issues = append(issues, newIssue(source, sourcePath, activity, business.DataQualitySeverityWarning, business.DataQualityCategoryInconsistentTime, "moving_time", "Moving time is greater than elapsed time.", fmt.Sprintf("%d > %d", activity.MovingTime, activity.ElapsedTime), "Prefer elapsed time or recompute moving time from stream data."))
	}
	if invalidFloat(activity.AverageSpeed) {
		issues = append(issues, newIssue(source, sourcePath, activity, business.DataQualitySeverityCritical, business.DataQualityCategoryInvalidValue, "average_speed", "Average speed is not serializable.", formatFloat(activity.AverageSpeed), "Sanitize NaN/Inf values before exposing the activity."))
	} else if activity.AverageSpeed > speedThreshold(activity.Type) {
		issues = append(issues, newIssue(source, sourcePath, activity, business.DataQualitySeverityWarning, business.DataQualityCategoryInvalidValue, "average_speed", "Average speed is unusually high.", fmt.Sprintf("%.1f km/h", activity.AverageSpeed*3.6), "Inspect GPS glitches or timing data before trusting speed statistics."))
	}
	if invalidFloat(activity.MaxSpeed) {
		issues = append(issues, newIssue(source, sourcePath, activity, business.DataQualitySeverityCritical, business.DataQualityCategoryInvalidValue, "max_speed", "Max speed is not serializable.", formatFloat(activity.MaxSpeed), "Sanitize NaN/Inf values before exposing the activity."))
	} else if activity.MaxSpeed > speedThreshold(activity.Type)*1.3 {
		issues = append(issues, newIssue(source, sourcePath, activity, business.DataQualitySeverityWarning, business.DataQualityCategoryInvalidValue, "max_speed", "Max speed is unusually high.", fmt.Sprintf("%.1f km/h", activity.MaxSpeed*3.6), "Inspect GPS glitches before trusting speed records."))
	}
	if invalidFloat(activity.TotalElevationGain) {
		issues = append(issues, newIssue(source, sourcePath, activity, business.DataQualitySeverityCritical, business.DataQualityCategoryInvalidValue, "total_elevation_gain", "Elevation gain is not serializable.", formatFloat(activity.TotalElevationGain), "Recompute elevation from altitude stream or SRTM."))
	}

	issues = append(issues, analyzeDerivedSpeed(source, sourcePath, activity)...)

	if activity.Stream == nil {
		if source == "strava" && activity.UploadId > 0 {
			issues = append(issues, newIssue(source, sourcePath, activity, business.DataQualitySeverityInfo, business.DataQualityCategoryMissingStream, "stream", "Detailed stream is missing from the local cache.", "", "Download missing streams from Strava when API access is available."))
			return issues
		}
		if source != "fit" && source != "gpx" {
			return issues
		}
		issues = append(issues, newIssue(source, sourcePath, activity, business.DataQualitySeverityWarning, business.DataQualityCategoryMissingStream, "stream", "Activity has no stream data.", "", "Open the source file and verify GPS/time streams are present."))
		return issues
	}

	issues = append(issues, analyzeStreamPresence(source, sourcePath, activity)...)
	issues = append(issues, analyzeGPSGlitch(source, sourcePath, activity)...)
	issues = append(issues, analyzeAltitudeSpike(source, sourcePath, activity)...)
	return annotateCorrectionSuggestions(activity, issues)
}

func analyzeStreamPresence(source string, sourcePath string, activity *strava.Activity) []business.DataQualityIssue {
	issues := make([]business.DataQualityIssue, 0)
	stream := activity.Stream
	if len(stream.Distance.Data) == 0 {
		issues = append(issues, newIssue(source, sourcePath, activity, business.DataQualitySeverityCritical, business.DataQualityCategoryMissingStreamField, "stream.distance", "Distance stream field is missing.", "", "Recompute distance from GPS points when possible."))
	}
	if len(stream.Time.Data) == 0 {
		issues = append(issues, newIssue(source, sourcePath, activity, business.DataQualitySeverityCritical, business.DataQualityCategoryMissingStreamField, "stream.time", "Time stream field is missing.", "", "The activity cannot be checked for speed glitches without time data."))
	}
	if requiresRouteStream(activity) && (stream.LatLng == nil || len(stream.LatLng.Data) == 0) {
		issues = append(issues, newIssue(source, sourcePath, activity, business.DataQualitySeverityWarning, business.DataQualityCategoryMissingStreamField, "stream.latlng", "GPS trace stream field is missing.", "", "Map and route-based analysis will be unavailable."))
	}
	if stream.Altitude == nil || len(stream.Altitude.Data) == 0 {
		issues = append(issues, newIssue(source, sourcePath, activity, business.DataQualitySeverityWarning, business.DataQualityCategoryMissingStreamField, "stream.altitude", "Altitude stream field is missing.", "", "Use SRTM elevation fallback or recompute D+."))
	}
	if stream.HeartRate == nil && activity.AverageHeartrate > 0 {
		issues = append(issues, newIssue(source, sourcePath, activity, business.DataQualitySeverityInfo, business.DataQualityCategoryStreamDataCoverage, "stream.heartrate", "Average heart rate exists but heart-rate samples are not available.", formatFloat(activity.AverageHeartrate), "Heart-rate charts will be incomplete."))
	}
	if stream.Watts == nil && activity.AverageWatts > 0 && (source != "strava" || activity.DeviceWatts) {
		issues = append(issues, newIssue(source, sourcePath, activity, business.DataQualitySeverityInfo, business.DataQualityCategoryStreamDataCoverage, "stream.watts", "Average power exists but power samples are not available.", formatFloat(activity.AverageWatts), "Power charts will be incomplete."))
	}

	if stream.LatLng != nil && len(stream.Time.Data) > 0 && absInt(len(stream.LatLng.Data)-len(stream.Time.Data)) > 1 {
		issues = append(issues, newIssue(source, sourcePath, activity, business.DataQualitySeverityWarning, business.DataQualityCategoryMissingStreamField, "stream.latlng", "GPS and time stream fields have inconsistent sizes.", fmt.Sprintf("gps=%d time=%d", len(stream.LatLng.Data), len(stream.Time.Data)), "Trim or resample streams before detailed analysis."))
	}
	if stream.Altitude != nil && len(stream.Time.Data) > 0 && absInt(len(stream.Altitude.Data)-len(stream.Time.Data)) > 1 {
		issues = append(issues, newIssue(source, sourcePath, activity, business.DataQualitySeverityWarning, business.DataQualityCategoryMissingStreamField, "stream.altitude", "Altitude and time stream fields have inconsistent sizes.", fmt.Sprintf("altitude=%d time=%d", len(stream.Altitude.Data), len(stream.Time.Data)), "Trim or resample streams before elevation analysis."))
	}
	return issues
}

func analyzeDerivedSpeed(source string, sourcePath string, activity *strava.Activity) []business.DataQualityIssue {
	movingTime := activity.MovingTime
	if movingTime <= 0 {
		movingTime = activity.ElapsedTime
	}
	if movingTime <= 0 || activity.Distance <= 0 {
		return nil
	}
	speed := activity.Distance / float64(movingTime)
	threshold := speedThreshold(activity.Type)
	if speed > threshold {
		return []business.DataQualityIssue{
			newIssue(source, sourcePath, activity, business.DataQualitySeverityWarning, business.DataQualityCategoryInvalidValue, "average_speed", "Computed average speed is unusually high.", fmt.Sprintf("%.1f km/h", speed*3.6), "Inspect GPS glitches or timing data before trusting speed statistics."),
		}
	}
	return nil
}

func analyzeGPSGlitch(source string, sourcePath string, activity *strava.Activity) []business.DataQualityIssue {
	stream := activity.Stream
	if stream == nil || stream.LatLng == nil || len(stream.LatLng.Data) < 2 || len(stream.Time.Data) < 2 {
		return nil
	}

	limit := minInt(len(stream.LatLng.Data), len(stream.Time.Data))
	threshold := speedThreshold(activity.Type)
	maxSpeed := 0.0
	maxIndex := 0
	for i := 1; i < limit; i++ {
		previous := stream.LatLng.Data[i-1]
		current := stream.LatLng.Data[i]
		if len(previous) < 2 || len(current) < 2 {
			continue
		}
		deltaSeconds := stream.Time.Data[i] - stream.Time.Data[i-1]
		if deltaSeconds <= 0 {
			continue
		}
		distance := haversineMeters(previous[0], previous[1], current[0], current[1])
		speed := distance / float64(deltaSeconds)
		if speed > maxSpeed {
			maxSpeed = speed
			maxIndex = i
		}
	}

	if maxSpeed > threshold {
		return []business.DataQualityIssue{
			newIssue(source, sourcePath, activity, business.DataQualitySeverityWarning, business.DataQualityCategoryGPSGlitch, "stream.latlng", "GPS trace contains an impossible speed jump.", fmt.Sprintf("%.1f km/h near point %d", maxSpeed*3.6, maxIndex), "Mark the segment as suspicious or smooth/remove the point locally."),
		}
	}
	return nil
}

func analyzeAltitudeSpike(source string, sourcePath string, activity *strava.Activity) []business.DataQualityIssue {
	stream := activity.Stream
	if stream == nil || stream.Altitude == nil || len(stream.Altitude.Data) < 2 {
		return nil
	}
	limit := len(stream.Altitude.Data)
	if len(stream.Time.Data) > 0 {
		limit = minInt(limit, len(stream.Time.Data))
	}

	maxDelta := 0.0
	maxIndex := 0
	for i := 1; i < limit; i++ {
		delta := math.Abs(stream.Altitude.Data[i] - stream.Altitude.Data[i-1])
		if delta > maxDelta {
			maxDelta = delta
			maxIndex = i
		}
		if delta < altitudeSpikeM {
			continue
		}
		if len(stream.Time.Data) == 0 || stream.Time.Data[i]-stream.Time.Data[i-1] <= altitudeSpikeSecs {
			return []business.DataQualityIssue{
				newIssue(source, sourcePath, activity, business.DataQualitySeverityWarning, business.DataQualityCategoryAltitudeSpike, "stream.altitude", "Altitude stream contains a sharp spike.", fmt.Sprintf("%.0f m near point %d", maxDelta, maxIndex), "Smooth altitude locally or recompute elevation from SRTM."),
			}
		}
	}
	return nil
}

func buildSummary(provider string, issues []business.DataQualityIssue, exclusions []business.DataQualityExclusion, corrections []business.DataQualityCorrection) business.DataQualitySummary {
	bySeverity := map[string]int{
		string(business.DataQualitySeverityCritical): 0,
		string(business.DataQualitySeverityWarning):  0,
		string(business.DataQualitySeverityInfo):     0,
	}
	byCategory := make(map[string]int)
	impactedActivities := make(map[int64]struct{})
	safeCorrectionCount := 0
	manualReviewCount := 0

	for _, issue := range issues {
		bySeverity[string(issue.Severity)]++
		byCategory[string(issue.Category)]++
		if issue.ActivityID != 0 {
			impactedActivities[issue.ActivityID] = struct{}{}
		}
		if issue.Correction == nil || !issue.Correction.Available {
			continue
		}
		if issue.Correction.Safety == business.DataQualityCorrectionSafetySafe {
			safeCorrectionCount++
		} else if issue.Correction.Safety == business.DataQualityCorrectionSafetyManual {
			manualReviewCount++
		}
	}

	status := "ok"
	if provider == "" {
		status = "not_applicable"
	} else if bySeverity[string(business.DataQualitySeverityCritical)] > 0 {
		status = "critical"
	} else if bySeverity[string(business.DataQualitySeverityWarning)] > 0 {
		status = "warning"
	}

	topIssueCount := minInt(len(issues), maxTopIssues)
	topIssues := make([]business.DataQualityIssue, topIssueCount)
	copy(topIssues, issues[:topIssueCount])

	return business.DataQualitySummary{
		Status:              status,
		Provider:            provider,
		IssueCount:          len(issues),
		ImpactedActivities:  len(impactedActivities),
		ExcludedActivities:  len(exclusionsByActivityID(exclusions)),
		CorrectionCount:     len(activeCorrections(corrections)),
		SafeCorrectionCount: safeCorrectionCount,
		ManualReviewCount:   manualReviewCount,
		BySeverity:          bySeverity,
		ByCategory:          byCategory,
		TopIssues:           topIssues,
	}
}

func markIssueExclusions(issues []business.DataQualityIssue, exclusions map[int64]business.DataQualityExclusion) []business.DataQualityIssue {
	for index := range issues {
		if exclusion, excluded := exclusions[issues[index].ActivityID]; excluded {
			issues[index].ExcludedFromStats = true
			issues[index].ExcludedAt = exclusion.ExcludedAt
		}
	}
	return issues
}

func markIssueCorrections(issues []business.DataQualityIssue, corrections map[string]business.DataQualityCorrection) []business.DataQualityIssue {
	for index := range issues {
		if correction, corrected := corrections[issues[index].ID]; corrected {
			issues[index].Corrected = true
			issues[index].CorrectionAppliedAt = correction.AppliedAt
		}
	}
	return issues
}

func annotateCorrectionSuggestions(activity *strava.Activity, issues []business.DataQualityIssue) []business.DataQualityIssue {
	for index := range issues {
		issues[index].Correction = correctionSuggestionForIssue(activity, issues[index])
	}
	return issues
}

func activeCorrectionsByIssueID(corrections []business.DataQualityCorrection) map[string]business.DataQualityCorrection {
	result := make(map[string]business.DataQualityCorrection)
	for _, correction := range activeCorrections(corrections) {
		if correction.IssueID == "" {
			continue
		}
		result[correction.IssueID] = correction
	}
	return result
}

func newIssue(source string, sourcePath string, activity *strava.Activity, severity business.DataQualitySeverity, category business.DataQualityCategory, field string, message string, rawValue string, suggestion string) business.DataQualityIssue {
	activityID := int64(0)
	activityName := ""
	activityType := ""
	year := ""
	if activity != nil {
		activityID = activity.Id
		activityName = strings.TrimSpace(activity.Name)
		activityType = activity.Type
		year = extractIssueYear(activity)
	}
	return business.DataQualityIssue{
		ID:           issueID(source, activityID, category, field),
		Source:       strings.ToUpper(source),
		ActivityID:   activityID,
		ActivityName: activityName,
		ActivityType: activityType,
		Year:         year,
		FilePath:     sourcePath,
		Severity:     severity,
		Category:     category,
		Field:        field,
		Message:      message,
		RawValue:     rawValue,
		Suggestion:   suggestion,
	}
}

func issueID(source string, activityID int64, category business.DataQualityCategory, field string) string {
	return fmt.Sprintf("%s-%d-%s-%s", strings.ToLower(source), activityID, category, strings.ReplaceAll(field, ".", "-"))
}

func allActivityTypes() []business.ActivityType {
	return []business.ActivityType{
		business.Run,
		business.TrailRun,
		business.Ride,
		business.GravelRide,
		business.MountainBikeRide,
		business.InlineSkate,
		business.Hike,
		business.Walk,
		business.Commute,
		business.AlpineSki,
		business.VirtualRide,
	}
}

func speedThreshold(activityType string) float64 {
	switch activityType {
	case "Run", "TrailRun":
		return 12
	case "Hike", "Walk":
		return 7
	case "AlpineSki":
		return 45
	default:
		return defaultMaxSpeedMS
	}
}

func requiresRouteStream(activity *strava.Activity) bool {
	if activity == nil {
		return false
	}
	activityType := strings.TrimSpace(activity.Type)
	sportType := strings.TrimSpace(activity.SportType)
	return activityType != "VirtualRide" && sportType != "VirtualRide"
}

func sortIssues(issues []business.DataQualityIssue) {
	sort.SliceStable(issues, func(i, j int) bool {
		left := severityRank(issues[i].Severity)
		right := severityRank(issues[j].Severity)
		if left != right {
			return left < right
		}
		if issues[i].Year != issues[j].Year {
			return issues[i].Year > issues[j].Year
		}
		return issues[i].ActivityName < issues[j].ActivityName
	})
}

func severityRank(severity business.DataQualitySeverity) int {
	switch severity {
	case business.DataQualitySeverityCritical:
		return 0
	case business.DataQualitySeverityWarning:
		return 1
	default:
		return 2
	}
}

func extractIssueYear(activity *strava.Activity) string {
	for _, value := range []string{activity.StartDateLocal, activity.StartDate} {
		if len(value) >= 4 {
			return value[:4]
		}
	}
	return ""
}

func invalidFloat(value float64) bool {
	return math.IsNaN(value) || math.IsInf(value, 0)
}

func formatFloat(value float64) string {
	if math.IsNaN(value) {
		return "NaN"
	}
	if math.IsInf(value, 1) {
		return "+Inf"
	}
	if math.IsInf(value, -1) {
		return "-Inf"
	}
	return fmt.Sprintf("%.2f", value)
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
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

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
