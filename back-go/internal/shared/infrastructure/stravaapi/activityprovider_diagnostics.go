package stravaapi

import (
	"os"
	"time"

	"mystravastats/domain/statistics"
)

func (provider *StravaActivityProvider) CacheDiagnostics() map[string]any {
	provider.manifestMutex.Lock()
	manifest := provider.cacheManifest
	provider.manifestMutex.Unlock()

	bestEffortPath := bestEffortCachePath(provider.cacheRoot, provider.clientId, manifest)
	warmupPath := warmupSummariesPath(provider.cacheRoot, provider.clientId, manifest)
	manifestPath := cacheManifestPath(provider.cacheRoot, provider.clientId)

	return map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"athleteId": provider.clientId,
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
				"preparedYears": manifest.Warmup.PreparedYears,
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
