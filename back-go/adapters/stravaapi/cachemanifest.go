package stravaapi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	cacheManifestSchemaVersion = 1
	bestEffortAlgoVersion      = "best-effort-v1"
	warmupAlgoVersion          = "warmup-v1"
	defaultBestEffortFile      = "best-effort-cache.json"
	defaultWarmupSummariesFile = "warmup-summaries.json"
)

type cacheManifest struct {
	SchemaVersion   int                     `json:"schemaVersion"`
	AthleteID       string                  `json:"athleteId"`
	UpdatedAt       string                  `json:"updatedAt"`
	BestEffortCache cacheBestEffortManifest `json:"bestEffortCache"`
	Warmup          cacheWarmupManifest     `json:"warmup"`
}

type cacheBestEffortManifest struct {
	AlgoVersion     string `json:"algoVersion"`
	File            string `json:"file"`
	Entries         int    `json:"entries"`
	LastPersistedAt string `json:"lastPersistedAt,omitempty"`
}

type cacheWarmupManifest struct {
	AlgoVersion   string `json:"algoVersion"`
	File          string `json:"file"`
	Priority1     string `json:"priority1"`
	Priority2     string `json:"priority2"`
	Priority3     string `json:"priority3"`
	PreparedYears []int  `json:"preparedYears,omitempty"`
	LastRunAt     string `json:"lastRunAt,omitempty"`
}

type warmupSummariesFile struct {
	SchemaVersion    int                   `json:"schemaVersion"`
	AthleteID        string                `json:"athleteId"`
	GeneratedAt      string                `json:"generatedAt"`
	YearSummaries    []warmupYearSummary   `json:"yearSummaries"`
	MajorBestEfforts []warmupMetricSummary `json:"majorBestEfforts,omitempty"`
	AdvancedMetrics  []warmupMetricSummary `json:"advancedMetrics,omitempty"`
}

type warmupYearSummary struct {
	Year            int     `json:"year"`
	ActivityCount   int     `json:"activityCount"`
	TotalDistanceKM float64 `json:"totalDistanceKm"`
	TotalElevationM float64 `json:"totalElevationM"`
	ElapsedSeconds  int     `json:"elapsedSeconds"`
}

type warmupMetricSummary struct {
	ActivityGroup string `json:"activityGroup"`
	Metric        string `json:"metric"`
	Target        string `json:"target"`
	Value         string `json:"value"`
	ActivityID    int64  `json:"activityId,omitempty"`
}

func defaultCacheManifest(clientID string) cacheManifest {
	return cacheManifest{
		SchemaVersion: cacheManifestSchemaVersion,
		AthleteID:     clientID,
		UpdatedAt:     time.Now().UTC().Format(time.RFC3339),
		BestEffortCache: cacheBestEffortManifest{
			AlgoVersion: bestEffortAlgoVersion,
			File:        defaultBestEffortFile,
			Entries:     0,
		},
		Warmup: cacheWarmupManifest{
			AlgoVersion: warmupAlgoVersion,
			File:        defaultWarmupSummariesFile,
			Priority1:   "pending",
			Priority2:   "pending",
			Priority3:   "pending",
		},
	}
}

func loadCacheManifest(cacheRoot, clientID string) (cacheManifest, error) {
	path := cacheManifestPath(cacheRoot, clientID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultCacheManifest(clientID), nil
		}
		return cacheManifest{}, fmt.Errorf("read cache manifest: %w", err)
	}

	manifest := defaultCacheManifest(clientID)
	if err := json.Unmarshal(data, &manifest); err != nil {
		return defaultCacheManifest(clientID), fmt.Errorf("unmarshal cache manifest: %w", err)
	}

	if manifest.SchemaVersion == 0 {
		manifest.SchemaVersion = cacheManifestSchemaVersion
	}
	if manifest.AthleteID == "" {
		manifest.AthleteID = clientID
	}
	if manifest.BestEffortCache.AlgoVersion == "" {
		manifest.BestEffortCache.AlgoVersion = bestEffortAlgoVersion
	}
	if manifest.BestEffortCache.File == "" {
		manifest.BestEffortCache.File = defaultBestEffortFile
	}
	if manifest.Warmup.AlgoVersion == "" {
		manifest.Warmup.AlgoVersion = warmupAlgoVersion
	}
	if manifest.Warmup.File == "" {
		manifest.Warmup.File = defaultWarmupSummariesFile
	}
	if manifest.Warmup.Priority1 == "" {
		manifest.Warmup.Priority1 = "pending"
	}
	if manifest.Warmup.Priority2 == "" {
		manifest.Warmup.Priority2 = "pending"
	}
	if manifest.Warmup.Priority3 == "" {
		manifest.Warmup.Priority3 = "pending"
	}

	return manifest, nil
}

func saveCacheManifest(cacheRoot, clientID string, manifest cacheManifest) error {
	manifest.SchemaVersion = cacheManifestSchemaVersion
	manifest.AthleteID = clientID
	manifest.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if manifest.BestEffortCache.AlgoVersion == "" {
		manifest.BestEffortCache.AlgoVersion = bestEffortAlgoVersion
	}
	if manifest.BestEffortCache.File == "" {
		manifest.BestEffortCache.File = defaultBestEffortFile
	}
	if manifest.Warmup.AlgoVersion == "" {
		manifest.Warmup.AlgoVersion = warmupAlgoVersion
	}
	if manifest.Warmup.File == "" {
		manifest.Warmup.File = defaultWarmupSummariesFile
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cache manifest: %w", err)
	}

	return writeJSONAtomically(cacheManifestPath(cacheRoot, clientID), data)
}

func saveWarmupSummaries(cacheRoot, clientID string, file warmupSummariesFile, manifest cacheManifest) error {
	file.SchemaVersion = cacheManifestSchemaVersion
	file.AthleteID = clientID
	file.GeneratedAt = time.Now().UTC().Format(time.RFC3339)

	sort.Slice(file.YearSummaries, func(i, j int) bool {
		return file.YearSummaries[i].Year > file.YearSummaries[j].Year
	})

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal warmup summaries: %w", err)
	}

	return writeJSONAtomically(warmupSummariesPath(cacheRoot, clientID, manifest), data)
}

func cacheManifestPath(cacheRoot, clientID string) string {
	return filepath.Join(cacheRoot, fmt.Sprintf("strava-%s", clientID), "cache-manifest.json")
}

func bestEffortCachePath(cacheRoot, clientID string, manifest cacheManifest) string {
	file := manifest.BestEffortCache.File
	if file == "" {
		file = defaultBestEffortFile
	}
	return filepath.Join(cacheRoot, fmt.Sprintf("strava-%s", clientID), file)
}

func warmupSummariesPath(cacheRoot, clientID string, manifest cacheManifest) string {
	file := manifest.Warmup.File
	if file == "" {
		file = defaultWarmupSummariesFile
	}
	return filepath.Join(cacheRoot, fmt.Sprintf("strava-%s", clientID), file)
}

func writeJSONAtomically(path string, payload []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return fmt.Errorf("create directory for %s: %w", path, err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return fmt.Errorf("write temp file %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename temp file %s to %s: %w", tmp, path, err)
	}

	return nil
}
