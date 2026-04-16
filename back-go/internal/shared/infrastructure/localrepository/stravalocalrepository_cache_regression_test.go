package localrepository

import (
	"fmt"
	"mystravastats/internal/shared/domain/strava"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var activitiesFilePattern = regexp.MustCompile(`^activities-(.+)-([0-9]{4})\.json$`)

func TestStravaRepository_ReadsRealCacheJsonSamples(t *testing.T) {
	// GIVEN
	cacheRoot, ok := findRealStravaCacheRoot()
	if !ok {
		t.Skip("real strava-cache directory not found")
	}

	// WHEN
	activitiesFiles, err := filepath.Glob(filepath.Join(cacheRoot, "strava-*", "strava-*-*", "activities-*-*.json"))
	if err != nil {
		t.Fatalf("failed to enumerate activities files: %v", err)
	}
	if len(activitiesFiles) == 0 {
		t.Skip("no cached activities JSON file found in strava-cache")
	}

	// THEN: Verify we can read and parse the repository
	sourceActivitiesFile := activitiesFiles[0]
	clientID, year, err := extractClientIDAndYearFromActivitiesFile(sourceActivitiesFile)
	if err != nil {
		t.Fatalf("failed to parse activities file metadata from %s: %v", sourceActivitiesFile, err)
	}

	tempCache := t.TempDir()
	targetYearDir := filepath.Join(tempCache, fmt.Sprintf("strava-%s", clientID), fmt.Sprintf("strava-%s-%d", clientID, year))
	if err := os.MkdirAll(targetYearDir, os.ModePerm); err != nil {
		t.Fatalf("failed to create temp cache year dir: %v", err)
	}
	targetActivitiesFile := filepath.Join(targetYearDir, filepath.Base(sourceActivitiesFile))
	if err := copyFile(sourceActivitiesFile, targetActivitiesFile); err != nil {
		t.Fatalf("failed to copy activities sample: %v", err)
	}

	repository := NewStravaRepository(tempCache)

	t.Run("detailed activity sample", func(t *testing.T) {
		detailedFiles, err := filepath.Glob(filepath.Join(filepath.Dir(sourceActivitiesFile), "stravaActivity-*"))
		if err != nil {
			t.Fatalf("failed to enumerate detailed activities: %v", err)
		}
		if len(detailedFiles) == 0 {
			t.Skip("no detailed activity JSON file found in selected cache year")
		}

		sourceDetailed := detailedFiles[0]
		targetDetailed := filepath.Join(targetYearDir, filepath.Base(sourceDetailed))
		if err := copyFile(sourceDetailed, targetDetailed); err != nil {
			t.Fatalf("failed to copy detailed activity sample: %v", err)
		}

		activityIDStr := strings.TrimPrefix(filepath.Base(sourceDetailed), "stravaActivity-")
		activityID, err := strconv.ParseInt(activityIDStr, 10, 64)
		if err != nil {
			t.Fatalf("failed to parse detailed activity id %q: %v", activityIDStr, err)
		}

		detailed := repository.LoadDetailedActivityFromCache(clientID, year, activityID)
		if detailed == nil {
			t.Fatalf("expected detailed activity to be loaded from cache")
		}
		if detailed.Id != activityID {
			t.Fatalf("detailed activity id mismatch: got %d, want %d", detailed.Id, activityID)
		}
	})

	t.Run("stream sample", func(t *testing.T) {
		streamFiles, err := filepath.Glob(filepath.Join(filepath.Dir(sourceActivitiesFile), "stream-*"))
		if err != nil {
			t.Fatalf("failed to enumerate stream files: %v", err)
		}
		if len(streamFiles) == 0 {
			t.Skip("no stream JSON file found in selected cache year")
		}

		sourceStream := streamFiles[0]
		targetStream := filepath.Join(targetYearDir, filepath.Base(sourceStream))
		if err := copyFile(sourceStream, targetStream); err != nil {
			t.Fatalf("failed to copy stream sample: %v", err)
		}

		activityIDStr := strings.TrimPrefix(filepath.Base(sourceStream), "stream-")
		activityID, err := strconv.ParseInt(activityIDStr, 10, 64)
		if err != nil {
			t.Fatalf("failed to parse stream activity id %q: %v", activityIDStr, err)
		}

		stream := repository.LoadActivitiesStreamsFromCache(clientID, year, strava.Activity{Id: activityID})
		if stream == nil {
			t.Fatalf("expected stream to be loaded from cache")
		}
		if len(stream.Distance.Data) == 0 || len(stream.Time.Data) == 0 {
			t.Fatalf("expected parsed stream to contain distance and time samples")
		}
	})
}

func findRealStravaCacheRoot() (string, bool) {
	candidates := []string{
		"../../../strava-cache",
		"../../strava-cache",
		"../strava-cache",
		"strava-cache",
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, true
		}
	}
	return "", false
}

func extractClientIDAndYearFromActivitiesFile(path string) (string, int, error) {
	base := filepath.Base(path)
	matches := activitiesFilePattern.FindStringSubmatch(base)
	if len(matches) != 3 {
		return "", 0, fmt.Errorf("invalid activities filename: %s", base)
	}
	year, err := strconv.Atoi(matches[2])
	if err != nil {
		return "", 0, fmt.Errorf("invalid year in activities filename %s: %w", base, err)
	}
	return matches[1], year, nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, os.ModePerm)
}
