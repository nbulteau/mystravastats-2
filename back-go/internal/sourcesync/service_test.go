package sourcesync

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"mystravastats/internal/shared/domain/strava"
)

func TestSynchronize_ImportsGarminFITIntoYearDirectory(t *testing.T) {
	volumesRoot := t.TempDir()
	sourceDirectory := filepath.Join(volumesRoot, "GARMIN", "GARMIN", "ACTIVITY")
	destinationDirectory := t.TempDir()
	if err := os.MkdirAll(sourceDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	sourceFile := filepath.Join(sourceDirectory, "ride.fit")
	if err := os.WriteFile(sourceFile, []byte("fit"), 0o644); err != nil {
		t.Fatal(err)
	}

	reloadCount := 0
	service := testService(destinationDirectory, volumesRoot, func(filePath string, athleteID int64) (*strava.Activity, error) {
		return &strava.Activity{
			Id:             42,
			Type:           "Ride",
			StartDate:      "2026-06-03T08:00:00Z",
			StartDateLocal: "2026-06-03T10:00:00+02:00",
			Distance:       12_345,
			ElapsedTime:    1800,
		}, nil
	}, func() {
		reloadCount++
	})

	result := service.Synchronize("test")

	if result.Status != "completed" {
		t.Fatalf("expected completed synchronization, got %s", result.Status)
	}
	if result.FIT.Status != "imported" {
		t.Fatalf("expected FIT import, got %s", result.FIT.Status)
	}
	if result.FIT.ImportedFiles != 1 {
		t.Fatalf("expected 1 imported file, got %d", result.FIT.ImportedFiles)
	}
	if reloadCount != 1 || !result.Reloaded {
		t.Fatalf("expected provider reload after import, reloadCount=%d reloaded=%t", reloadCount, result.Reloaded)
	}
	importedPath := filepath.Join(destinationDirectory, "2026", "ride.fit")
	if _, err := os.Stat(importedPath); err != nil {
		t.Fatalf("expected imported FIT in year directory: %v", err)
	}
}

func TestSynchronize_SkipsAlreadyImportedFIT(t *testing.T) {
	volumesRoot := t.TempDir()
	sourceDirectory := filepath.Join(volumesRoot, "GARMIN", "GARMIN", "ACTIVITY")
	destinationDirectory := t.TempDir()
	if err := os.MkdirAll(sourceDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDirectory, "ride.fit"), []byte("fit"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := testService(destinationDirectory, volumesRoot, func(filePath string, athleteID int64) (*strava.Activity, error) {
		return &strava.Activity{
			Type:        "Ride",
			StartDate:   "2026-06-03T08:00:00Z",
			Distance:    12_345,
			ElapsedTime: 1800,
		}, nil
	}, func() {})

	firstResult := service.Synchronize("test")
	secondResult := service.Synchronize("test")

	if firstResult.FIT.ImportedFiles != 1 {
		t.Fatalf("expected first import to copy 1 file, got %d", firstResult.FIT.ImportedFiles)
	}
	if secondResult.FIT.Status != "up_to_date" {
		t.Fatalf("expected second import to be up to date, got %s", secondResult.FIT.Status)
	}
	if secondResult.FIT.AlreadyPresentFiles != 1 {
		t.Fatalf("expected already present count=1, got %d", secondResult.FIT.AlreadyPresentFiles)
	}
}

func TestSynchronize_ReportsNoDeviceWhenGarminDirectoryIsMissing(t *testing.T) {
	service := testService(t.TempDir(), t.TempDir(), func(filePath string, athleteID int64) (*strava.Activity, error) {
		t.Fatalf("decode should not be called without a source directory")
		return nil, nil
	}, func() {
		t.Fatalf("reload should not be called without imports")
	})

	result := service.Synchronize("test")

	if result.Status != "skipped" {
		t.Fatalf("expected skipped synchronization, got %s", result.Status)
	}
	if result.FIT.Status != "no_device" {
		t.Fatalf("expected no_device FIT status, got %s", result.FIT.Status)
	}
}

func testService(
	destinationDirectory string,
	volumesRoot string,
	decode func(filePath string, athleteID int64) (*strava.Activity, error),
	reload func(),
) *Service {
	return &Service{
		decodeFIT:        decode,
		fitDestination:   func() (string, bool) { return destinationDirectory, true },
		garminSourcePath: func() (string, bool) { return "", false },
		reloadProvider:   reload,
		volumesRoot:      volumesRoot,
		now:              time.Now,
		lastResult: SyncResult{
			Status: "idle",
		},
	}
}
