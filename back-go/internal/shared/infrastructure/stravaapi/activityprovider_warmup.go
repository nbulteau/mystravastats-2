package stravaapi

import (
	"fmt"
	"log"
	"mystravastats/domain/statistics"
	"mystravastats/internal/helpers"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"sort"
	"time"
)

func (provider *StravaActivityProvider) loadPersistentCacheArtifacts() {
	manifest, err := loadCacheManifest(provider.cacheRoot, provider.clientId)
	if err != nil {
		log.Printf("Unable to load cache manifest: %v", err)
		manifest = defaultCacheManifest(provider.clientId)
	}

	entries, err := statistics.LoadBestEffortCacheFromDisk(bestEffortCachePath(provider.cacheRoot, provider.clientId, manifest))
	if err != nil {
		log.Printf("Unable to load best-effort cache: %v", err)
		statistics.ClearBestEffortCache()
		entries = 0
	}

	manifest.BestEffortCache.Entries = entries
	if entries > 0 && manifest.BestEffortCache.LastPersistedAt == "" {
		manifest.BestEffortCache.LastPersistedAt = time.Now().UTC().Format(time.RFC3339)
	}

	provider.manifestMutex.Lock()
	provider.cacheManifest = manifest
	provider.manifestMutex.Unlock()

	if err := saveCacheManifest(provider.cacheRoot, provider.clientId, manifest); err != nil {
		log.Printf("Unable to persist cache manifest: %v", err)
	}

	log.Printf("Loaded best-effort cache: %d entries", entries)
}

func (provider *StravaActivityProvider) launchBackgroundWarmup(reason string) {
	go provider.runWarmupPipeline(reason)
}

func (provider *StravaActivityProvider) runWarmupPipeline(reason string) {
	if !provider.warmupInProgress.CompareAndSwap(false, true) {
		return
	}
	defer provider.warmupInProgress.Store(false)

	activities := provider.getActivitiesSnapshot()
	if len(activities) == 0 {
		return
	}

	log.Printf("Warmup started (%s)", reason)

	warmupPayload := warmupSummariesFile{
		YearSummaries: provider.computeWarmupYearSummaries(activities),
	}
	preparedYears := extractPreparedYears(warmupPayload.YearSummaries)
	if err := provider.persistWarmupArtifacts(warmupPayload, "ready", "pending", "pending", preparedYears); err != nil {
		log.Printf("Warmup priority 1 failed: %v", err)
		_ = provider.persistWarmupArtifacts(warmupPayload, "failed", "pending", "pending", preparedYears)
		return
	}

	warmupPayload.MajorBestEfforts = provider.precomputeMajorBestEfforts(activities)
	if err := provider.persistWarmupArtifacts(warmupPayload, "ready", "ready", "pending", preparedYears); err != nil {
		log.Printf("Warmup priority 2 failed: %v", err)
		_ = provider.persistWarmupArtifacts(warmupPayload, "ready", "failed", "pending", preparedYears)
		return
	}

	warmupPayload.AdvancedMetrics = provider.precomputeAdvancedMetrics(activities)
	if err := provider.persistWarmupArtifacts(warmupPayload, "ready", "ready", "ready", preparedYears); err != nil {
		log.Printf("Warmup priority 3 failed: %v", err)
		_ = provider.persistWarmupArtifacts(warmupPayload, "ready", "ready", "failed", preparedYears)
		return
	}

	log.Printf("Warmup completed (%s)", reason)
}

func (provider *StravaActivityProvider) persistWarmupArtifacts(
	warmupPayload warmupSummariesFile,
	priority1 string,
	priority2 string,
	priority3 string,
	preparedYears []int,
) error {
	provider.manifestMutex.Lock()
	manifest := provider.cacheManifest

	entries, err := statistics.SaveBestEffortCacheToDisk(bestEffortCachePath(provider.cacheRoot, provider.clientId, manifest))
	if err != nil {
		provider.manifestMutex.Unlock()
		return err
	}

	manifest.BestEffortCache.Entries = entries
	manifest.BestEffortCache.LastPersistedAt = time.Now().UTC().Format(time.RFC3339)
	manifest.Warmup.Priority1 = priority1
	manifest.Warmup.Priority2 = priority2
	manifest.Warmup.Priority3 = priority3
	manifest.Warmup.PreparedYears = preparedYears
	manifest.Warmup.LastRunAt = time.Now().UTC().Format(time.RFC3339)

	if err := saveWarmupSummaries(provider.cacheRoot, provider.clientId, warmupPayload, manifest); err != nil {
		provider.manifestMutex.Unlock()
		return err
	}

	if err := saveCacheManifest(provider.cacheRoot, provider.clientId, manifest); err != nil {
		provider.manifestMutex.Unlock()
		return err
	}

	provider.cacheManifest = manifest
	provider.manifestMutex.Unlock()
	return nil
}

func (provider *StravaActivityProvider) computeWarmupYearSummaries(activities []*strava.Activity) []warmupYearSummary {
	yearly := map[int]*warmupYearSummary{}
	allYears := warmupYearSummary{Year: 0}

	for _, activity := range activities {
		if activity == nil {
			continue
		}
		year := resolveActivityYear(activity)
		summary := yearly[year]
		if summary == nil {
			summary = &warmupYearSummary{Year: year}
			yearly[year] = summary
		}
		summary.ActivityCount++
		summary.TotalDistanceKM += activity.Distance / 1000
		summary.TotalElevationM += activity.TotalElevationGain
		summary.ElapsedSeconds += activity.ElapsedTime

		allYears.ActivityCount++
		allYears.TotalDistanceKM += activity.Distance / 1000
		allYears.TotalElevationM += activity.TotalElevationGain
		allYears.ElapsedSeconds += activity.ElapsedTime
	}

	summaries := make([]warmupYearSummary, 0, len(yearly)+1)
	summaries = append(summaries, allYears)
	for _, summary := range yearly {
		summaries = append(summaries, *summary)
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Year > summaries[j].Year
	})
	return summaries
}

func extractPreparedYears(summaries []warmupYearSummary) []int {
	years := make([]int, 0, len(summaries))
	for _, summary := range summaries {
		years = append(years, summary.Year)
	}
	sort.Slice(years, func(i, j int) bool {
		return years[i] > years[j]
	})
	return years
}

func (provider *StravaActivityProvider) precomputeMajorBestEfforts(activities []*strava.Activity) []warmupMetricSummary {
	rideActivities := filterActivitiesByGroup(activities, "ride")
	runActivities := filterActivitiesByGroup(activities, "run")

	metrics := []warmupMetricSummary{}
	metrics = append(metrics, computeBestTimeDistanceMetric("ride", rideActivities, 1000)...)
	metrics = append(metrics, computeBestTimeDistanceMetric("ride", rideActivities, 5000)...)
	metrics = append(metrics, computeBestDistanceTimeMetric("ride", rideActivities, 20*60)...)
	metrics = append(metrics, computeBestDistanceTimeMetric("ride", rideActivities, 60*60)...)
	metrics = append(metrics, computeBestTimeDistanceMetric("run", runActivities, 1000)...)
	metrics = append(metrics, computeBestTimeDistanceMetric("run", runActivities, 5000)...)
	metrics = append(metrics, computeBestDistanceTimeMetric("run", runActivities, 20*60)...)
	metrics = append(metrics, computeBestDistanceTimeMetric("run", runActivities, 60*60)...)
	return metrics
}

func (provider *StravaActivityProvider) precomputeAdvancedMetrics(activities []*strava.Activity) []warmupMetricSummary {
	rideActivities := filterActivitiesByGroup(activities, "ride")

	metrics := []warmupMetricSummary{}
	metrics = append(metrics, computeBestElevationMetric("ride", rideActivities, 1000)...)
	metrics = append(metrics, computeBestElevationMetric("ride", rideActivities, 5000)...)
	metrics = append(metrics, computeBestPowerMetric("ride", rideActivities, 20*60)...)
	metrics = append(metrics, computeBestPowerMetric("ride", rideActivities, 60*60)...)
	return metrics
}

func filterActivitiesByGroup(activities []*strava.Activity, group string) []*strava.Activity {
	filtered := make([]*strava.Activity, 0)
	for _, activity := range activities {
		if activity == nil {
			continue
		}
		sportType := activity.SportType
		if sportType == "" {
			sportType = activity.Type
		}

		switch group {
		case "run":
			if sportType == business.Run.String() || sportType == business.TrailRun.String() {
				filtered = append(filtered, activity)
			}
		case "ride":
			if sportType == business.Ride.String() ||
				sportType == business.GravelRide.String() ||
				sportType == business.MountainBikeRide.String() ||
				sportType == business.VirtualRide.String() {
				filtered = append(filtered, activity)
			}
		}
	}
	return filtered
}

func computeBestTimeDistanceMetric(group string, activities []*strava.Activity, distance float64) []warmupMetricSummary {
	best := findBestTimeDistanceEffort(activities, distance)
	if best == nil {
		return nil
	}
	return []warmupMetricSummary{{
		ActivityGroup: group,
		Metric:        "best-time-distance",
		Target:        statistics.EffortDistanceTarget(distance),
		Value:         fmt.Sprintf("%s => %s", helpers.FormatSeconds(best.Seconds), best.GetFormattedSpeed()),
		ActivityID:    best.ActivityShort.Id,
	}}
}

func computeBestDistanceTimeMetric(group string, activities []*strava.Activity, seconds int) []warmupMetricSummary {
	best := findBestDistanceTimeEffort(activities, seconds)
	if best == nil {
		return nil
	}
	distanceLabel := fmt.Sprintf("%.0f m", best.Distance)
	if best.Distance >= 1000 {
		distanceLabel = fmt.Sprintf("%.2f km", best.Distance/1000)
	}

	return []warmupMetricSummary{{
		ActivityGroup: group,
		Metric:        "best-distance-time",
		Target:        statistics.EffortSecondsTarget(seconds),
		Value:         fmt.Sprintf("%s => %s", distanceLabel, best.GetFormattedSpeed()),
		ActivityID:    best.ActivityShort.Id,
	}}
}

func computeBestPowerMetric(group string, activities []*strava.Activity, seconds int) []warmupMetricSummary {
	best := findBestPowerEffort(activities, seconds)
	if best == nil || best.AveragePower == nil {
		return nil
	}
	return []warmupMetricSummary{{
		ActivityGroup: group,
		Metric:        "best-power-time",
		Target:        statistics.EffortSecondsTarget(seconds),
		Value:         fmt.Sprintf("%.0f W", *best.AveragePower),
		ActivityID:    best.ActivityShort.Id,
	}}
}

func computeBestElevationMetric(group string, activities []*strava.Activity, distance float64) []warmupMetricSummary {
	best := findBestElevationEffort(activities, distance)
	if best == nil {
		return nil
	}
	return []warmupMetricSummary{{
		ActivityGroup: group,
		Metric:        "best-elevation-distance",
		Target:        statistics.EffortDistanceTarget(distance),
		Value:         fmt.Sprintf("%s => %s", helpers.FormatSeconds(best.Seconds), best.GetFormattedGradient()),
		ActivityID:    best.ActivityShort.Id,
	}}
}

func findBestTimeDistanceEffort(activities []*strava.Activity, distance float64) *business.ActivityEffort {
	var best *business.ActivityEffort
	for _, activity := range activities {
		effort := statistics.BestTimeEffort(*activity, distance)
		if effort != nil && (best == nil || effort.Seconds < best.Seconds) {
			best = effort
		}
	}
	return best
}

func findBestDistanceTimeEffort(activities []*strava.Activity, seconds int) *business.ActivityEffort {
	var best *business.ActivityEffort
	for _, activity := range activities {
		effort := statistics.BestDistanceEffort(*activity, seconds)
		if effort != nil && (best == nil || effort.Distance > best.Distance) {
			best = effort
		}
	}
	return best
}

func findBestPowerEffort(activities []*strava.Activity, seconds int) *business.ActivityEffort {
	var best *business.ActivityEffort
	for _, activity := range activities {
		effort := statistics.BestPowerForTime(*activity, seconds)
		if effort != nil && (best == nil || effort.Distance > best.Distance) {
			best = effort
		}
	}
	return best
}

func findBestElevationEffort(activities []*strava.Activity, distance float64) *business.ActivityEffort {
	var best *business.ActivityEffort
	for _, activity := range activities {
		effort := statistics.BestElevationEffort(*activity, distance)
		if effort != nil && (best == nil || effort.DeltaAltitude > best.DeltaAltitude) {
			best = effort
		}
	}
	return best
}
