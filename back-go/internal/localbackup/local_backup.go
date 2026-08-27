package localbackup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"mystravastats/internal/platform/safefile"
)

const Version = 1
const secureDirectoryMode = 0700
const secureFileMode = 0600

type Bundle struct {
	Version    int                        `json:"version"`
	ExportedAt string                     `json:"exportedAt"`
	AthleteID  string                     `json:"athleteId"`
	Files      map[string]json.RawMessage `json:"files"`
}

type RestoreResult struct {
	Restored []string `json:"restored"`
}

var fileNames = map[string]func(string) string{
	"dataQualityCorrections": func(id string) string { return fmt.Sprintf("data-quality-corrections-%s.json", id) },
	"dataQualityExclusions":  func(id string) string { return fmt.Sprintf("data-quality-exclusions-%s.json", id) },
	"gearMaintenance":        func(id string) string { return fmt.Sprintf("gear-maintenance-%s.json", id) },
	"heartRateZones":         func(id string) string { return fmt.Sprintf("heart-rate-zones-%s.json", id) },
	"performanceSettings":    func(id string) string { return fmt.Sprintf("performance-settings-%s.json", id) },
}

func Export(cacheRoot string, athleteID string) (Bundle, error) {
	bundle := Bundle{
		Version:    Version,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		AthleteID:  athleteID,
		Files:      make(map[string]json.RawMessage),
	}
	for key, name := range fileNames {
		data, err := os.ReadFile(filepath.Join(athleteDirectory(cacheRoot, athleteID), name(athleteID)))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return Bundle{}, fmt.Errorf("read %s: %w", key, err)
		}
		if !json.Valid(data) {
			return Bundle{}, fmt.Errorf("local data %s is not valid JSON", key)
		}
		bundle.Files[key] = data
	}
	return bundle, nil
}

func Restore(cacheRoot string, athleteID string, bundle Bundle) (RestoreResult, error) {
	if bundle.Version != Version {
		return RestoreResult{}, fmt.Errorf("unsupported backup version %d", bundle.Version)
	}
	if bundle.AthleteID != athleteID {
		return RestoreResult{}, fmt.Errorf("backup belongs to athlete %q, current athlete is %q", bundle.AthleteID, athleteID)
	}
	keys := make([]string, 0, len(bundle.Files))
	for key, content := range bundle.Files {
		if _, allowed := fileNames[key]; !allowed {
			return RestoreResult{}, fmt.Errorf("unsupported local data key %q", key)
		}
		if !json.Valid(content) {
			return RestoreResult{}, fmt.Errorf("backup entry %s is not valid JSON", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	directory := athleteDirectory(cacheRoot, athleteID)
	if err := os.MkdirAll(directory, secureDirectoryMode); err != nil {
		return RestoreResult{}, fmt.Errorf("create local data directory: %w", err)
	}
	for _, key := range keys {
		path := filepath.Join(directory, fileNames[key](athleteID))
		if err := safefile.WithLock(path, func() error {
			return safefile.WriteFile(path, bundle.Files[key], secureFileMode)
		}); err != nil {
			return RestoreResult{}, fmt.Errorf("restore %s: %w", key, err)
		}
	}
	return RestoreResult{Restored: keys}, nil
}

func athleteDirectory(cacheRoot string, athleteID string) string {
	return filepath.Join(cacheRoot, fmt.Sprintf("strava-%s", athleteID))
}
