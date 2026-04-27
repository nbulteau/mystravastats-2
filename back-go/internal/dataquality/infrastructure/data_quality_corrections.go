package infrastructure

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const altitudeSpikeInterpolationMaxNeighborDeltaM = 60.0

type dataQualityCorrectionFile struct {
	Corrections []business.DataQualityCorrection `json:"corrections"`
}

func CurrentProviderCorrectionPreview(issueID string) (business.DataQualityCorrectionPreview, error) {
	issueID = strings.TrimSpace(issueID)
	if issueID == "" {
		return business.DataQualityCorrectionPreview{}, fmt.Errorf("issueId must not be empty")
	}

	context := currentProviderCorrectionContext()
	for _, issue := range context.report.Issues {
		if issue.ID != issueID {
			continue
		}
		correction, warnings, blockingReasons, ok := buildCorrectionForIssue(context.activityByID[issue.ActivityID], issue)
		preview := newCorrectionPreview("single")
		preview.Warnings = warnings
		preview.BlockingReasons = blockingReasons
		if ok {
			preview.Corrections = []business.DataQualityCorrection{correction}
		}
		preview.Summary = summarizeCorrections(preview.Corrections, 0, len(preview.BlockingReasons))
		return preview, nil
	}

	return business.DataQualityCorrectionPreview{}, fmt.Errorf("issue %s not found", issueID)
}

func CurrentProviderSafeCorrectionPreview() business.DataQualityCorrectionPreview {
	context := currentProviderCorrectionContext()
	preview := newCorrectionPreview("safe_batch")
	manualReviewCount := 0
	unsupportedCount := 0

	for _, issue := range context.report.Issues {
		correction, warnings, blockingReasons, ok := buildCorrectionForIssue(context.activityByID[issue.ActivityID], issue)
		preview.Warnings = append(preview.Warnings, warnings...)
		if ok && correction.Safety == business.DataQualityCorrectionSafetySafe {
			preview.Corrections = append(preview.Corrections, correction)
			continue
		}
		if ok && correction.Safety == business.DataQualityCorrectionSafetyManual {
			manualReviewCount++
			continue
		}
		if len(blockingReasons) > 0 {
			unsupportedCount++
			preview.BlockingReasons = append(preview.BlockingReasons, blockingReasons...)
		}
	}

	preview.Corrections = dedupeCorrections(preview.Corrections)
	preview.Summary = summarizeCorrections(preview.Corrections, manualReviewCount, unsupportedCount)
	return preview
}

func ApplyCurrentProviderCorrection(issueID string) (business.DataQualityReport, error) {
	preview, err := CurrentProviderCorrectionPreview(issueID)
	if err != nil {
		return business.DataQualityReport{}, err
	}
	if len(preview.Corrections) == 0 {
		return business.DataQualityReport{}, fmt.Errorf("issue %s has no safe correction: %s", issueID, strings.Join(preview.BlockingReasons, "; "))
	}
	correction := preview.Corrections[0]
	if correction.Safety != business.DataQualityCorrectionSafetySafe {
		return business.DataQualityReport{}, fmt.Errorf("issue %s requires manual review", issueID)
	}
	if err := saveCurrentProviderCorrections([]business.DataQualityCorrection{correction}); err != nil {
		return business.DataQualityReport{}, err
	}
	return CurrentProviderReport(), nil
}

func ApplyCurrentProviderSafeCorrections() (business.DataQualityReport, error) {
	preview := CurrentProviderSafeCorrectionPreview()
	if len(preview.Corrections) == 0 {
		return CurrentProviderReport(), nil
	}
	if err := saveCurrentProviderCorrections(preview.Corrections); err != nil {
		return business.DataQualityReport{}, err
	}
	return CurrentProviderReport(), nil
}

func RevertCurrentProviderCorrection(correctionID string) (business.DataQualityReport, error) {
	correctionID = strings.TrimSpace(correctionID)
	if correctionID == "" {
		return business.DataQualityReport{}, fmt.Errorf("correctionId must not be empty")
	}

	provider := activityprovider.Get()
	corrections := loadCorrections(provider.CacheRootPath(), provider.ClientID())
	found := false
	now := time.Now().UTC().Format(time.RFC3339)
	for index := range corrections {
		if corrections[index].ID != correctionID {
			continue
		}
		corrections[index].Status = business.DataQualityCorrectionStatusReverted
		corrections[index].RevertedAt = now
		found = true
	}
	if !found {
		return business.DataQualityReport{}, fmt.Errorf("correction %s not found", correctionID)
	}
	if err := saveCorrections(provider.CacheRootPath(), provider.ClientID(), sortedCorrections(corrections)); err != nil {
		return business.DataQualityReport{}, err
	}
	return CurrentProviderReport(), nil
}

func CurrentProviderCorrections() []business.DataQualityCorrection {
	provider := activityprovider.Get()
	return loadCorrections(provider.CacheRootPath(), provider.ClientID())
}

func ApplyCurrentProviderCorrections(activities []*strava.Activity) []*strava.Activity {
	provider := activityprovider.Get()
	return ApplyCorrectionsToActivities(activities, activeCorrections(loadCorrections(provider.CacheRootPath(), provider.ClientID())))
}

func ApplyCurrentProviderCorrectionsToDetailedActivity(activity *strava.DetailedActivity) *strava.DetailedActivity {
	if activity == nil {
		return nil
	}
	provider := activityprovider.Get()
	corrections := activeCorrections(loadCorrections(provider.CacheRootPath(), provider.ClientID()))
	if len(corrections) == 0 {
		return activity
	}
	relevant := correctionsByActivityID(corrections)[activity.Id]
	if len(relevant) == 0 {
		return activity
	}

	cloned := cloneDetailedActivity(activity)
	activityView := activityFromDetailed(cloned)
	applyCorrectionsToActivity(activityView, relevant)
	mergeActivityIntoDetailed(cloned, activityView)
	return cloned
}

func ApplyCorrectionsToActivities(activities []*strava.Activity, corrections []business.DataQualityCorrection) []*strava.Activity {
	if len(activities) == 0 {
		return []*strava.Activity{}
	}
	activeByActivityID := correctionsByActivityID(activeCorrections(corrections))
	if len(activeByActivityID) == 0 {
		return cloneDataQualityActivityPointers(activities)
	}

	result := make([]*strava.Activity, 0, len(activities))
	for _, activity := range activities {
		if activity == nil {
			continue
		}
		relevant := activeByActivityID[activity.Id]
		if len(relevant) == 0 {
			result = append(result, activity)
			continue
		}
		cloned := cloneActivity(activity)
		applyCorrectionsToActivity(cloned, relevant)
		result = append(result, cloned)
	}
	return result
}

func CorrectionSignature() string {
	provider := activityprovider.Get()
	correctionsFile := correctionsFilePath(provider.CacheRootPath(), provider.ClientID())
	info, err := os.Stat(correctionsFile)
	if err != nil {
		return "none"
	}
	return fmt.Sprintf("%d:%d", info.ModTime().UnixNano(), info.Size())
}

type correctionContext struct {
	report       business.DataQualityReport
	activityByID map[int64]*strava.Activity
}

func currentProviderCorrectionContext() correctionContext {
	provider := activityprovider.Get()
	diagnostics := provider.CacheDiagnostics()
	source := strings.ToLower(fmt.Sprint(diagnostics["provider"]))
	sourcePath := provider.CacheRootPath()
	activities := provider.GetActivitiesByYearAndActivityTypes(nil, allActivityTypes()...)
	exclusions := loadExclusions(provider.CacheRootPath(), provider.ClientID())
	corrections := loadCorrections(provider.CacheRootPath(), provider.ClientID())
	correctedActivities := ApplyCorrectionsToActivities(activities, corrections)
	activityByID := make(map[int64]*strava.Activity, len(correctedActivities))
	for _, activity := range correctedActivities {
		if activity != nil {
			activityByID[activity.Id] = activity
		}
	}
	return correctionContext{
		report:       AnalyzeActivitiesWithCorrections(source, sourcePath, correctedActivities, exclusions, corrections),
		activityByID: activityByID,
	}
}

func newCorrectionPreview(mode string) business.DataQualityCorrectionPreview {
	return business.DataQualityCorrectionPreview{
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		Mode:            mode,
		Corrections:     []business.DataQualityCorrection{},
		Warnings:        []string{},
		BlockingReasons: []string{},
	}
}

func buildCorrectionForIssue(activity *strava.Activity, issue business.DataQualityIssue) (business.DataQualityCorrection, []string, []string, bool) {
	if activity == nil || issue.ActivityID == 0 {
		return business.DataQualityCorrection{}, nil, []string{fmt.Sprintf("%s has no activity context", issue.ID)}, false
	}

	switch issue.Category {
	case business.DataQualityCategoryGPSGlitch:
		index, ok := findIsolatedGPSOutlier(activity)
		if !ok {
			return manualCorrection(issue, "GPS glitch is not isolated enough for a safe automatic fix"), nil, nil, true
		}
		return buildRemoveGPSPointCorrection(activity, issue, index), nil, nil, true
	case business.DataQualityCategoryAltitudeSpike:
		index, ok := findIsolatedAltitudeSpike(activity)
		if !ok {
			return manualCorrection(issue, "Altitude spike is not isolated enough for a safe automatic fix"), nil, nil, true
		}
		return buildSmoothAltitudeCorrection(activity, issue, index), nil, nil, true
	default:
		return business.DataQualityCorrection{}, nil, []string{fmt.Sprintf("%s cannot be corrected automatically", issue.ID)}, false
	}
}

func correctionSuggestionForIssue(activity *strava.Activity, issue business.DataQualityIssue) *business.DataQualityCorrectionSuggestion {
	correction, _, _, ok := buildCorrectionForIssue(activity, issue)
	if !ok {
		return &business.DataQualityCorrectionSuggestion{
			Available:   false,
			Safety:      business.DataQualityCorrectionSafetyUnsupported,
			Description: "No local non-destructive correction is available for this issue.",
		}
	}
	switch correction.Safety {
	case business.DataQualityCorrectionSafetySafe:
		return &business.DataQualityCorrectionSuggestion{
			Available:   true,
			Safety:      correction.Safety,
			Type:        correction.Type,
			Label:       correctionLabel(correction.Type),
			Description: correction.Reason,
		}
	case business.DataQualityCorrectionSafetyManual:
		return &business.DataQualityCorrectionSuggestion{
			Available:   true,
			Safety:      correction.Safety,
			Type:        correction.Type,
			Label:       "Manual review",
			Description: correction.Reason,
		}
	default:
		return nil
	}
}

func manualCorrection(issue business.DataQualityIssue, reason string) business.DataQualityCorrection {
	return business.DataQualityCorrection{
		ID:             correctionID(issue.ID, business.DataQualityCorrectionTypeRecalculateFromStream),
		IssueID:        issue.ID,
		Source:         issue.Source,
		ActivityID:     issue.ActivityID,
		ActivityName:   issue.ActivityName,
		ActivityType:   issue.ActivityType,
		Year:           issue.Year,
		Type:           business.DataQualityCorrectionTypeRecalculateFromStream,
		Safety:         business.DataQualityCorrectionSafetyManual,
		Status:         business.DataQualityCorrectionStatusActive,
		ModifiedFields: []string{},
		Reason:         reason,
	}
}

func buildRemoveGPSPointCorrection(activity *strava.Activity, issue business.DataQualityIssue, index int) business.DataQualityCorrection {
	correction := baseCorrection(activity, issue, business.DataQualityCorrectionTypeRemoveGPSPoint)
	correction.PointIndexes = []int{index}
	correction.ModifiedFields = []string{"stream.latlng", "stream.distance", "stream.velocitySmooth", "distance", "average_speed", "max_speed"}
	correction.Reason = fmt.Sprintf("Remove isolated GPS point %d and recompute distance and speed from remaining coordinates.", index)
	correction.Impact = impactForCorrection(activity, correction)
	return correction
}

func buildSmoothAltitudeCorrection(activity *strava.Activity, issue business.DataQualityIssue, index int) business.DataQualityCorrection {
	correction := baseCorrection(activity, issue, business.DataQualityCorrectionTypeSmoothAltitudeSpike)
	correction.PointIndexes = []int{index}
	correction.ModifiedFields = []string{"stream.altitude", "total_elevation_gain", "elev_high"}
	correction.Reason = fmt.Sprintf("Replace isolated altitude point %d by interpolation and recompute elevation gain.", index)
	correction.Impact = impactForCorrection(activity, correction)
	return correction
}

func baseCorrection(activity *strava.Activity, issue business.DataQualityIssue, correctionType business.DataQualityCorrectionType) business.DataQualityCorrection {
	return business.DataQualityCorrection{
		ID:             correctionID(issue.ID, correctionType),
		IssueID:        issue.ID,
		Source:         issue.Source,
		ActivityID:     activity.Id,
		ActivityName:   strings.TrimSpace(activity.Name),
		ActivityType:   activity.Type,
		Year:           extractIssueYear(activity),
		Type:           correctionType,
		Safety:         business.DataQualityCorrectionSafetySafe,
		Status:         business.DataQualityCorrectionStatusActive,
		ModifiedFields: []string{},
	}
}

func impactForCorrection(activity *strava.Activity, correction business.DataQualityCorrection) business.DataQualityCorrectionImpact {
	beforeDistance := activity.Distance
	beforeElevation := activity.TotalElevationGain
	beforeMaxSpeed := activity.MaxSpeed
	after := cloneActivity(activity)
	applyCorrectionToActivity(after, correction)
	return business.DataQualityCorrectionImpact{
		DistanceMetersBefore:  beforeDistance,
		DistanceMetersAfter:   after.Distance,
		ElevationMetersBefore: beforeElevation,
		ElevationMetersAfter:  after.TotalElevationGain,
		MaxSpeedBefore:        beforeMaxSpeed,
		MaxSpeedAfter:         after.MaxSpeed,
		DistanceDeltaMeters:   after.Distance - beforeDistance,
		ElevationDeltaMeters:  after.TotalElevationGain - beforeElevation,
	}
}

func correctionID(issueID string, correctionType business.DataQualityCorrectionType) string {
	normalized := strings.ToLower(strings.ReplaceAll(issueID, " ", "-"))
	return fmt.Sprintf("%s-%s", normalized, strings.ToLower(string(correctionType)))
}

func correctionLabel(correctionType business.DataQualityCorrectionType) string {
	switch correctionType {
	case business.DataQualityCorrectionTypeRemoveGPSPoint:
		return "Remove GPS point"
	case business.DataQualityCorrectionTypeSmoothAltitudeSpike:
		return "Smooth altitude spike"
	case business.DataQualityCorrectionTypeMaskInvalidValue:
		return "Mask invalid value"
	default:
		return "Recalculate from stream"
	}
}

func findIsolatedGPSOutlier(activity *strava.Activity) (int, bool) {
	stream := activity.Stream
	if stream == nil || stream.LatLng == nil || len(stream.LatLng.Data) < 3 || len(stream.Time.Data) < 3 {
		return 0, false
	}
	limit := minInt(len(stream.LatLng.Data), len(stream.Time.Data))
	threshold := speedThreshold(activity.Type)
	bestIndex := 0
	bestScore := 0.0
	for index := 1; index < limit-1; index++ {
		prevSpeed, okPrev := segmentSpeed(stream.LatLng.Data[index-1], stream.LatLng.Data[index], stream.Time.Data[index]-stream.Time.Data[index-1])
		nextSpeed, okNext := segmentSpeed(stream.LatLng.Data[index], stream.LatLng.Data[index+1], stream.Time.Data[index+1]-stream.Time.Data[index])
		stitchedSpeed, okStitched := segmentSpeed(stream.LatLng.Data[index-1], stream.LatLng.Data[index+1], stream.Time.Data[index+1]-stream.Time.Data[index-1])
		if !okPrev || !okNext || !okStitched {
			continue
		}
		if prevSpeed <= threshold || nextSpeed <= threshold || stitchedSpeed > threshold {
			continue
		}
		score := prevSpeed + nextSpeed - stitchedSpeed
		if score > bestScore {
			bestScore = score
			bestIndex = index
		}
	}
	return bestIndex, bestIndex > 0
}

func findIsolatedAltitudeSpike(activity *strava.Activity) (int, bool) {
	stream := activity.Stream
	if stream == nil || stream.Altitude == nil || len(stream.Altitude.Data) < 3 {
		return 0, false
	}
	limit := len(stream.Altitude.Data)
	if len(stream.Time.Data) > 0 {
		limit = minInt(limit, len(stream.Time.Data))
	}
	bestIndex := 0
	bestDelta := 0.0
	for index := 1; index < limit-1; index++ {
		prev := stream.Altitude.Data[index-1]
		current := stream.Altitude.Data[index]
		next := stream.Altitude.Data[index+1]
		prevDelta := math.Abs(current - prev)
		nextDelta := math.Abs(current - next)
		neighborDelta := math.Abs(next - prev)
		if prevDelta < altitudeSpikeM || nextDelta < altitudeSpikeM || neighborDelta > altitudeSpikeInterpolationMaxNeighborDeltaM {
			continue
		}
		if len(stream.Time.Data) > 0 && stream.Time.Data[index+1]-stream.Time.Data[index-1] > altitudeSpikeSecs*2 {
			continue
		}
		if prevDelta+nextDelta > bestDelta {
			bestDelta = prevDelta + nextDelta
			bestIndex = index
		}
	}
	return bestIndex, bestIndex > 0
}

func segmentSpeed(previous []float64, current []float64, seconds int) (float64, bool) {
	if len(previous) < 2 || len(current) < 2 || seconds <= 0 {
		return 0, false
	}
	distance := haversineMeters(previous[0], previous[1], current[0], current[1])
	return distance / float64(seconds), true
}

func applyCorrectionsToActivity(activity *strava.Activity, corrections []business.DataQualityCorrection) {
	sort.SliceStable(corrections, func(i, j int) bool {
		return corrections[i].AppliedAt < corrections[j].AppliedAt
	})
	for _, correction := range corrections {
		applyCorrectionToActivity(activity, correction)
	}
}

func applyCorrectionToActivity(activity *strava.Activity, correction business.DataQualityCorrection) {
	if activity == nil || correction.Status == business.DataQualityCorrectionStatusReverted {
		return
	}
	switch correction.Type {
	case business.DataQualityCorrectionTypeRemoveGPSPoint:
		if len(correction.PointIndexes) == 0 {
			return
		}
		removeGPSPoint(activity, correction.PointIndexes[0])
	case business.DataQualityCorrectionTypeSmoothAltitudeSpike:
		if len(correction.PointIndexes) == 0 {
			return
		}
		smoothAltitudePoint(activity, correction.PointIndexes[0])
	}
}

func removeGPSPoint(activity *strava.Activity, index int) {
	stream := activity.Stream
	if stream == nil || stream.LatLng == nil || index <= 0 || index >= len(stream.LatLng.Data)-1 {
		return
	}
	originalSize := len(stream.LatLng.Data)
	stream.LatLng.Data = removeFloat64Row(stream.LatLng.Data, index)
	stream.LatLng.OriginalSize = len(stream.LatLng.Data)
	if len(stream.Time.Data) == originalSize {
		stream.Time.Data = removeInt(stream.Time.Data, index)
		stream.Time.OriginalSize = len(stream.Time.Data)
	}
	if stream.Altitude != nil && len(stream.Altitude.Data) == originalSize {
		stream.Altitude.Data = removeFloat64(stream.Altitude.Data, index)
		stream.Altitude.OriginalSize = len(stream.Altitude.Data)
	}
	if stream.Moving != nil && len(stream.Moving.Data) == originalSize {
		stream.Moving.Data = removeBool(stream.Moving.Data, index)
		stream.Moving.OriginalSize = len(stream.Moving.Data)
	}
	if stream.HeartRate != nil && len(stream.HeartRate.Data) == originalSize {
		stream.HeartRate.Data = removeInt(stream.HeartRate.Data, index)
		stream.HeartRate.OriginalSize = len(stream.HeartRate.Data)
	}
	if stream.Watts != nil && len(stream.Watts.Data) == originalSize {
		stream.Watts.Data = removeFloat64(stream.Watts.Data, index)
		stream.Watts.OriginalSize = len(stream.Watts.Data)
	}
	if stream.Cadence != nil && len(stream.Cadence.Data) == originalSize {
		stream.Cadence.Data = removeInt(stream.Cadence.Data, index)
		stream.Cadence.OriginalSize = len(stream.Cadence.Data)
	}
	recomputeDistanceAndSpeed(activity)
	recomputeElevation(activity)
}

func smoothAltitudePoint(activity *strava.Activity, index int) {
	stream := activity.Stream
	if stream == nil || stream.Altitude == nil || index <= 0 || index >= len(stream.Altitude.Data)-1 {
		return
	}
	stream.Altitude.Data[index] = (stream.Altitude.Data[index-1] + stream.Altitude.Data[index+1]) / 2
	recomputeElevation(activity)
}

func recomputeDistanceAndSpeed(activity *strava.Activity) {
	stream := activity.Stream
	if stream == nil || stream.LatLng == nil || len(stream.LatLng.Data) == 0 {
		return
	}
	distances := make([]float64, len(stream.LatLng.Data))
	for index := 1; index < len(stream.LatLng.Data); index++ {
		previous := stream.LatLng.Data[index-1]
		current := stream.LatLng.Data[index]
		if len(previous) < 2 || len(current) < 2 {
			distances[index] = distances[index-1]
			continue
		}
		distances[index] = distances[index-1] + haversineMeters(previous[0], previous[1], current[0], current[1])
	}
	stream.Distance.Data = distances
	stream.Distance.OriginalSize = len(distances)
	activity.Distance = distances[len(distances)-1]

	movingTime := activity.MovingTime
	if movingTime <= 0 {
		movingTime = activity.ElapsedTime
	}
	if movingTime > 0 {
		activity.AverageSpeed = activity.Distance / float64(movingTime)
	}
	recomputeVelocity(activity)
}

func recomputeVelocity(activity *strava.Activity) {
	stream := activity.Stream
	if stream == nil || stream.LatLng == nil || len(stream.LatLng.Data) == 0 || len(stream.Time.Data) == 0 {
		return
	}
	limit := minInt(len(stream.LatLng.Data), len(stream.Time.Data))
	velocity := make([]float64, len(stream.LatLng.Data))
	maxSpeed := 0.0
	for index := 1; index < limit; index++ {
		speed, ok := segmentSpeed(stream.LatLng.Data[index-1], stream.LatLng.Data[index], stream.Time.Data[index]-stream.Time.Data[index-1])
		if !ok {
			continue
		}
		velocity[index] = speed
		if speed > maxSpeed {
			maxSpeed = speed
		}
	}
	if stream.VelocitySmooth == nil {
		stream.VelocitySmooth = &strava.SmoothVelocityStream{}
	}
	stream.VelocitySmooth.Data = velocity
	stream.VelocitySmooth.OriginalSize = len(velocity)
	stream.VelocitySmooth.Resolution = stream.Distance.Resolution
	stream.VelocitySmooth.SeriesType = "time"
	activity.MaxSpeed = maxSpeed
}

func recomputeElevation(activity *strava.Activity) {
	stream := activity.Stream
	if stream == nil || stream.Altitude == nil || len(stream.Altitude.Data) == 0 {
		return
	}
	totalGain := 0.0
	elevHigh := stream.Altitude.Data[0]
	for index, altitude := range stream.Altitude.Data {
		if altitude > elevHigh {
			elevHigh = altitude
		}
		if index == 0 {
			continue
		}
		delta := altitude - stream.Altitude.Data[index-1]
		if delta > 0 {
			totalGain += delta
		}
	}
	activity.TotalElevationGain = totalGain
	activity.ElevHigh = elevHigh
}

func cloneActivity(activity *strava.Activity) *strava.Activity {
	if activity == nil {
		return nil
	}
	cloned := *activity
	cloned.StartLatlng = cloneFloat64Slice(activity.StartLatlng)
	cloned.Stream = cloneStream(activity.Stream)
	return &cloned
}

func cloneDetailedActivity(activity *strava.DetailedActivity) *strava.DetailedActivity {
	cloned := *activity
	cloned.StartLatLng = cloneFloat64Slice(activity.StartLatLng)
	cloned.EndLatLng = cloneFloat64Slice(activity.EndLatLng)
	cloned.Stream = cloneStream(activity.Stream)
	return &cloned
}

func cloneStream(stream *strava.Stream) *strava.Stream {
	if stream == nil {
		return nil
	}
	cloned := *stream
	cloned.Distance.Data = cloneFloat64Slice(stream.Distance.Data)
	cloned.Time.Data = cloneIntSlice(stream.Time.Data)
	if stream.LatLng != nil {
		latlng := *stream.LatLng
		latlng.Data = cloneFloat64Rows(stream.LatLng.Data)
		cloned.LatLng = &latlng
	}
	if stream.Cadence != nil {
		cadence := *stream.Cadence
		cadence.Data = cloneIntSlice(stream.Cadence.Data)
		cloned.Cadence = &cadence
	}
	if stream.HeartRate != nil {
		heartRate := *stream.HeartRate
		heartRate.Data = cloneIntSlice(stream.HeartRate.Data)
		cloned.HeartRate = &heartRate
	}
	if stream.Moving != nil {
		moving := *stream.Moving
		moving.Data = cloneBoolSlice(stream.Moving.Data)
		cloned.Moving = &moving
	}
	if stream.Altitude != nil {
		altitude := *stream.Altitude
		altitude.Data = cloneFloat64Slice(stream.Altitude.Data)
		cloned.Altitude = &altitude
	}
	if stream.Watts != nil {
		watts := *stream.Watts
		watts.Data = cloneFloat64Slice(stream.Watts.Data)
		cloned.Watts = &watts
	}
	if stream.VelocitySmooth != nil {
		velocity := *stream.VelocitySmooth
		velocity.Data = cloneFloat64Slice(stream.VelocitySmooth.Data)
		cloned.VelocitySmooth = &velocity
	}
	if stream.GradeSmooth != nil {
		grade := *stream.GradeSmooth
		grade.Data = cloneFloat64Slice(stream.GradeSmooth.Data)
		cloned.GradeSmooth = &grade
	}
	return &cloned
}

func activityFromDetailed(activity *strava.DetailedActivity) *strava.Activity {
	return &strava.Activity{
		AverageSpeed:         activity.AverageSpeed,
		AverageCadence:       activity.AverageCadence,
		AverageHeartrate:     activity.AverageHeartrate,
		MaxHeartrate:         activity.MaxHeartrate,
		AverageWatts:         activity.AverageWatts,
		Commute:              activity.Commute,
		Distance:             activity.Distance,
		DeviceWatts:          activity.DeviceWatts,
		ElapsedTime:          activity.ElapsedTime,
		ElevHigh:             activity.ElevHigh,
		GearId:               activity.GearId,
		Id:                   activity.Id,
		Kilojoules:           activity.Kilojoules,
		MaxSpeed:             activity.MaxSpeed,
		MovingTime:           activity.MovingTime,
		Name:                 activity.Name,
		SportType:            activity.SportType,
		StartDate:            activity.StartDate,
		StartDateLocal:       activity.StartDateLocal,
		StartLatlng:          cloneFloat64Slice(activity.StartLatLng),
		TotalElevationGain:   activity.TotalElevationGain,
		Type:                 activity.Type,
		UploadId:             activity.UploadId,
		WeightedAverageWatts: activity.WeightedAverageWatts,
		Stream:               activity.Stream,
	}
}

func mergeActivityIntoDetailed(detailed *strava.DetailedActivity, activity *strava.Activity) {
	detailed.AverageSpeed = activity.AverageSpeed
	detailed.Distance = activity.Distance
	detailed.ElevHigh = activity.ElevHigh
	detailed.MaxSpeed = activity.MaxSpeed
	detailed.TotalElevationGain = activity.TotalElevationGain
	detailed.Stream = activity.Stream
}

func cloneFloat64Slice(values []float64) []float64 {
	if values == nil {
		return nil
	}
	cloned := make([]float64, len(values))
	copy(cloned, values)
	return cloned
}

func cloneIntSlice(values []int) []int {
	if values == nil {
		return nil
	}
	cloned := make([]int, len(values))
	copy(cloned, values)
	return cloned
}

func cloneBoolSlice(values []bool) []bool {
	if values == nil {
		return nil
	}
	cloned := make([]bool, len(values))
	copy(cloned, values)
	return cloned
}

func cloneFloat64Rows(values [][]float64) [][]float64 {
	if values == nil {
		return nil
	}
	cloned := make([][]float64, len(values))
	for index := range values {
		cloned[index] = cloneFloat64Slice(values[index])
	}
	return cloned
}

func removeFloat64(values []float64, index int) []float64 {
	if index < 0 || index >= len(values) {
		return values
	}
	result := make([]float64, 0, len(values)-1)
	result = append(result, values[:index]...)
	result = append(result, values[index+1:]...)
	return result
}

func removeFloat64Row(values [][]float64, index int) [][]float64 {
	if index < 0 || index >= len(values) {
		return values
	}
	result := make([][]float64, 0, len(values)-1)
	result = append(result, values[:index]...)
	result = append(result, values[index+1:]...)
	return result
}

func removeInt(values []int, index int) []int {
	if index < 0 || index >= len(values) {
		return values
	}
	result := make([]int, 0, len(values)-1)
	result = append(result, values[:index]...)
	result = append(result, values[index+1:]...)
	return result
}

func removeBool(values []bool, index int) []bool {
	if index < 0 || index >= len(values) {
		return values
	}
	result := make([]bool, 0, len(values)-1)
	result = append(result, values[:index]...)
	result = append(result, values[index+1:]...)
	return result
}

func saveCurrentProviderCorrections(corrections []business.DataQualityCorrection) error {
	provider := activityprovider.Get()
	existing := loadCorrections(provider.CacheRootPath(), provider.ClientID())
	byID := make(map[string]business.DataQualityCorrection, len(existing)+len(corrections))
	for _, correction := range existing {
		byID[correction.ID] = correction
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, correction := range corrections {
		if correction.ID == "" {
			continue
		}
		correction.Status = business.DataQualityCorrectionStatusActive
		correction.AppliedAt = now
		correction.RevertedAt = ""
		byID[correction.ID] = correction
	}
	merged := make([]business.DataQualityCorrection, 0, len(byID))
	for _, correction := range byID {
		merged = append(merged, correction)
	}
	return saveCorrections(provider.CacheRootPath(), provider.ClientID(), sortedCorrections(merged))
}

func loadCorrections(cacheRoot string, clientID string) []business.DataQualityCorrection {
	correctionsFile := correctionsFilePath(cacheRoot, clientID)
	data, err := os.ReadFile(correctionsFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Unable to read data quality corrections from %s: %v", correctionsFile, err)
		}
		return []business.DataQualityCorrection{}
	}
	var payload dataQualityCorrectionFile
	if err := json.Unmarshal(data, &payload); err != nil {
		log.Printf("Unable to parse data quality corrections from %s: %v", correctionsFile, err)
		return []business.DataQualityCorrection{}
	}
	return payload.Corrections
}

func saveCorrections(cacheRoot string, clientID string, corrections []business.DataQualityCorrection) error {
	athleteDirectory := exclusionsDirectory(cacheRoot, clientID)
	if err := os.MkdirAll(athleteDirectory, dataQualitySecureDirMode); err != nil {
		return fmt.Errorf("unable to create data quality directory: %w", err)
	}
	data, err := json.MarshalIndent(dataQualityCorrectionFile{Corrections: corrections}, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to encode data quality corrections: %w", err)
	}
	if err := os.WriteFile(correctionsFilePath(cacheRoot, clientID), data, dataQualitySecureFileMode); err != nil {
		return fmt.Errorf("unable to write data quality corrections: %w", err)
	}
	return nil
}

func correctionsFilePath(cacheRoot string, clientID string) string {
	return filepath.Join(exclusionsDirectory(cacheRoot, clientID), fmt.Sprintf("data-quality-corrections-%s.json", clientID))
}

func activeCorrections(corrections []business.DataQualityCorrection) []business.DataQualityCorrection {
	result := make([]business.DataQualityCorrection, 0, len(corrections))
	for _, correction := range corrections {
		if correction.Status == business.DataQualityCorrectionStatusReverted {
			continue
		}
		result = append(result, correction)
	}
	return result
}

func correctionsByActivityID(corrections []business.DataQualityCorrection) map[int64][]business.DataQualityCorrection {
	result := make(map[int64][]business.DataQualityCorrection)
	for _, correction := range corrections {
		if correction.ActivityID <= 0 {
			continue
		}
		result[correction.ActivityID] = append(result[correction.ActivityID], correction)
	}
	return result
}

func sortedCorrections(corrections []business.DataQualityCorrection) []business.DataQualityCorrection {
	result := append([]business.DataQualityCorrection{}, corrections...)
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Status != result[j].Status {
			return result[i].Status < result[j].Status
		}
		if result[i].Year != result[j].Year {
			return result[i].Year > result[j].Year
		}
		if result[i].ActivityID != result[j].ActivityID {
			return result[i].ActivityID < result[j].ActivityID
		}
		return result[i].ID < result[j].ID
	})
	return result
}

func dedupeCorrections(corrections []business.DataQualityCorrection) []business.DataQualityCorrection {
	byID := make(map[string]business.DataQualityCorrection, len(corrections))
	for _, correction := range corrections {
		byID[correction.ID] = correction
	}
	result := make([]business.DataQualityCorrection, 0, len(byID))
	for _, correction := range byID {
		result = append(result, correction)
	}
	return sortedCorrections(result)
}

func summarizeCorrections(corrections []business.DataQualityCorrection, manualReviewCount int, unsupportedIssueCount int) business.DataQualityCorrectionBatchSummary {
	activityIDs := make(map[int64]struct{})
	modifiedFields := make(map[string]struct{})
	distanceDelta := 0.0
	elevationDelta := 0.0
	for _, correction := range corrections {
		activityIDs[correction.ActivityID] = struct{}{}
		distanceDelta += correction.Impact.DistanceDeltaMeters
		elevationDelta += correction.Impact.ElevationDeltaMeters
		for _, field := range correction.ModifiedFields {
			modifiedFields[field] = struct{}{}
		}
	}
	fields := make([]string, 0, len(modifiedFields))
	for field := range modifiedFields {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return business.DataQualityCorrectionBatchSummary{
		SafeCorrectionCount:       len(corrections),
		ManualReviewCount:         manualReviewCount,
		UnsupportedIssueCount:     unsupportedIssueCount,
		ActivityCount:             len(activityIDs),
		DistanceDeltaMeters:       distanceDelta,
		ElevationDeltaMeters:      elevationDelta,
		ModifiedFields:            fields,
		PotentiallyImpactsRecords: len(corrections) > 0,
	}
}
