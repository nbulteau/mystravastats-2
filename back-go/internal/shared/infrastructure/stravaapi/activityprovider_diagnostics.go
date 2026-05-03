package stravaapi

import (
	"os"
	"sort"
	"time"

	"mystravastats/domain/statistics"
	"mystravastats/internal/shared/domain/strava"
)

func (provider *StravaActivityProvider) CacheDiagnostics() map[string]any {
	provider.manifestMutex.Lock()
	manifest := provider.cacheManifest
	provider.manifestMutex.Unlock()

	bestEffortPath := bestEffortCachePath(provider.cacheRoot, provider.clientId, manifest)
	warmupPath := warmupSummariesPath(provider.cacheRoot, provider.clientId, manifest)
	manifestPath := cacheManifestPath(provider.cacheRoot, provider.clientId)
	activities := provider.getActivitiesSnapshot()
	rateLimitUntilUnix := provider.rateLimitUntilUnix.Load()

	return map[string]any{
		"timestamp":         time.Now().UTC().Format(time.RFC3339),
		"provider":          "strava",
		"athleteId":         provider.clientId,
		"cacheRoot":         provider.cacheRoot,
		"activities":        len(activities),
		"availableYearBins": availableYearBins(activities),
		"refresh": map[string]any{
			"backgroundInProgress": provider.backgroundRefresh.Load(),
			"warmupInProgress":     provider.warmupInProgress.Load(),
		},
		"rateLimit": map[string]any{
			"active":       provider.isStravaRateLimitedNow(),
			"untilEpochMs": rateLimitUntilUnix * 1000,
		},
		"manifest": map[string]any{
			"schemaVersion": manifest.SchemaVersion,
			"updatedAt":     manifest.UpdatedAt,
			"bestEffortCache": map[string]any{
				"algoVersion":      manifest.BestEffortCache.AlgoVersion,
				"entriesPersisted": manifest.BestEffortCache.Entries,
				"entriesInMemory":  statistics.BestEffortCacheSize(),
				"file":             manifest.BestEffortCache.File,
				"lastPersistedAt":  manifest.BestEffortCache.LastPersistedAt,
			},
			"warmup": map[string]any{
				"algoVersion":   manifest.Warmup.AlgoVersion,
				"file":          manifest.Warmup.File,
				"priority1":     manifest.Warmup.Priority1,
				"priority2":     manifest.Warmup.Priority2,
				"priority3":     manifest.Warmup.Priority3,
				"preparedYears": normalizePreparedYears(manifest.Warmup.PreparedYears),
				"lastRunAt":     manifest.Warmup.LastRunAt,
			},
		},
		"files": map[string]any{
			"manifest":        fileDetails(manifestPath),
			"bestEffortCache": fileDetails(bestEffortPath),
			"warmupSummaries": fileDetails(warmupPath),
		},
	}
}

func availableYearBins(activities []*strava.Activity) []string {
	yearsSet := make(map[string]struct{})
	for _, activity := range activities {
		if activity == nil {
			continue
		}
		year := extractActivityYear(activity.StartDateLocal)
		if year == "" {
			year = extractActivityYear(activity.StartDate)
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

func extractActivityYear(value string) string {
	if len(value) >= 4 {
		return value[:4]
	}
	return ""
}

func fileDetails(path string) map[string]any {
	info, err := os.Stat(path)
	if err != nil {
		return map[string]any{
			"path":   path,
			"exists": false,
		}
	}
	return map[string]any{
		"path":         path,
		"exists":       true,
		"sizeBytes":    info.Size(),
		"lastModified": info.ModTime().UTC().Format(time.RFC3339),
	}
}
