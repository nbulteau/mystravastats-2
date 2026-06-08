package composite

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"mystravastats/internal/shared/domain/business"
	fitprovider "mystravastats/internal/shared/infrastructure/fit"
	gpxprovider "mystravastats/internal/shared/infrastructure/gpx"
	"mystravastats/internal/shared/infrastructure/stravaapi"
)

func TestCompositeSharedSourceModeFixtures(t *testing.T) {
	expected := readSharedCompositeExpected(t)
	fixtureRoot := copySharedSourceModeFixtures(t)
	provider := NewCompositeActivityProvider([]Source{
		{Name: "strava", Provider: stravaapi.NewStravaActivityProvider(filepath.Join(fixtureRoot, "strava"), "0")},
		{Name: "fit", Provider: fitprovider.NewFITActivityProvider(filepath.Join(fixtureRoot, "fit"))},
		{Name: "gpx", Provider: gpxprovider.NewGPXActivityProvider(filepath.Join(fixtureRoot, "gpx"))},
	})
	year := expected.Year

	activities := provider.GetActivitiesByYearAndActivityTypes(&year, business.Ride)
	if len(activities) != expected.ActivityCount {
		t.Fatalf("expected %d composite activities, got %d", expected.ActivityCount, len(activities))
	}
	if activities[0].Id != expected.ActivityID {
		t.Fatalf("expected composite activity id=%d, got %d", expected.ActivityID, activities[0].Id)
	}
	details := provider.GetDetailedActivity(activities[0].Id)
	if details == nil || details.Source == nil {
		t.Fatalf("expected detailed source provenance, got %#v", details)
	}
	if details.Source.PrimaryProvider != expected.PrimaryProvider {
		t.Fatalf("expected primary provider %q, got %q", expected.PrimaryProvider, details.Source.PrimaryProvider)
	}
	if details.Source.StreamProvider != expected.StreamProvider {
		t.Fatalf("expected stream provider %q, got %q", expected.StreamProvider, details.Source.StreamProvider)
	}
	if len(details.Source.Sources) != expected.SourceCount {
		t.Fatalf("expected %d source refs, got %d", expected.SourceCount, len(details.Source.Sources))
	}
	sourceProviders := make([]string, 0, len(details.Source.Sources))
	for _, source := range details.Source.Sources {
		sourceProviders = append(sourceProviders, source.Provider)
	}
	if !slices.Equal(sourceProviders, expected.SourceProviders) {
		t.Fatalf("expected source providers %#v, got %#v", expected.SourceProviders, sourceProviders)
	}
	if len(details.Source.Conflicts) != len(expected.Conflicts) {
		t.Fatalf("expected %d conflicts, got %d", len(expected.Conflicts), len(details.Source.Conflicts))
	}
	for index, expectedConflict := range expected.Conflicts {
		actualConflict := details.Source.Conflicts[index]
		if actualConflict.Field != expectedConflict.Field || actualConflict.Source != expectedConflict.Source {
			t.Fatalf("expected conflict %#v, got %#v", expectedConflict, actualConflict)
		}
	}

	diagnostics := provider.CacheDiagnostics()
	compositeDetails := diagnostics["composite"].(map[string]any)
	assertDiagnosticNumber(t, compositeDetails, "matchedActivities", expected.MatchedActivities)
	assertDiagnosticNumber(t, compositeDetails, "localOnlyActivities", expected.LocalOnlyActivities)
	assertDiagnosticNumber(t, compositeDetails, "conflictCount", expected.ConflictCount)
}

func sharedSourceModeFixtureRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "test-fixtures", "source-modes"))
}

func copySharedSourceModeFixtures(t *testing.T) string {
	t.Helper()
	sourceRoot := sharedSourceModeFixtureRoot(t)
	destinationRoot := filepath.Join(t.TempDir(), "source-modes")
	if err := copyDirectory(sourceRoot, destinationRoot); err != nil {
		t.Fatalf("copy source-mode fixtures: %v", err)
	}
	return destinationRoot
}

func copyDirectory(sourceRoot string, destinationRoot string) error {
	return filepath.WalkDir(sourceRoot, func(sourcePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relativePath, err := filepath.Rel(sourceRoot, sourcePath)
		if err != nil {
			return err
		}
		destinationPath := filepath.Join(destinationRoot, relativePath)
		if entry.IsDir() {
			return os.MkdirAll(destinationPath, 0o755)
		}
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return err
		}
		return os.WriteFile(destinationPath, data, 0o644)
	})
}

func readSharedCompositeExpected(t *testing.T) sharedCompositeExpected {
	t.Helper()
	path := filepath.Join(sharedSourceModeFixtureRoot(t), "composite-expected.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read composite expected fixture: %v", err)
	}
	var expected sharedCompositeExpected
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatalf("decode composite expected fixture: %v", err)
	}
	return expected
}

func assertDiagnosticNumber(t *testing.T, diagnostics map[string]any, key string, expected int) {
	t.Helper()
	actual, ok := diagnostics[key].(int)
	if !ok {
		t.Fatalf("expected diagnostic %s to be int, got %#v", key, diagnostics[key])
	}
	if actual != expected {
		t.Fatalf("expected diagnostic %s=%d, got %d", key, expected, actual)
	}
}

type sharedCompositeExpected struct {
	Year                int                      `json:"year"`
	ActivityType        string                   `json:"activityType"`
	ActivityCount       int                      `json:"activityCount"`
	ActivityID          int64                    `json:"activityId"`
	PrimaryProvider     string                   `json:"primaryProvider"`
	StreamProvider      string                   `json:"streamProvider"`
	SourceCount         int                      `json:"sourceCount"`
	SourceProviders     []string                 `json:"sourceProviders"`
	MatchedActivities   int                      `json:"matchedActivities"`
	LocalOnlyActivities int                      `json:"localOnlyActivities"`
	ConflictCount       int                      `json:"conflictCount"`
	Conflicts           []sharedExpectedConflict `json:"conflicts"`
}

type sharedExpectedConflict struct {
	Field  string `json:"field"`
	Source string `json:"source"`
}
