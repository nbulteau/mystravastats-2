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

func TestSynchronize_ImportsFITInboxIntoYearDirectory(t *testing.T) {
	destinationDirectory := t.TempDir()
	inboxDirectory := filepath.Join(destinationDirectory, "_inbox")
	if err := os.MkdirAll(inboxDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inboxDirectory, "ride.fit"), []byte("fit"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := testService(destinationDirectory, t.TempDir(), func(filePath string, athleteID int64) (*strava.Activity, error) {
		return &strava.Activity{
			Id:             43,
			Type:           "Ride",
			StartDate:      "2026-06-04T08:00:00Z",
			StartDateLocal: "2026-06-04T10:00:00+02:00",
			Distance:       22_345,
			ElapsedTime:    2400,
		}, nil
	}, func() {})
	service.fitInboxPath = func() (string, bool) { return inboxDirectory, true }

	result := service.Synchronize("test")

	if result.FIT.SourceKind != "fit_inbox" {
		t.Fatalf("expected fit_inbox source, got %s", result.FIT.SourceKind)
	}
	if result.FIT.ImportedFiles != 1 {
		t.Fatalf("expected 1 imported inbox file, got %d", result.FIT.ImportedFiles)
	}
	if _, err := os.Stat(filepath.Join(destinationDirectory, "2026", "ride.fit")); err != nil {
		t.Fatalf("expected imported inbox FIT in year directory: %v", err)
	}
}

func TestSynchronize_RunsGarminSyncModuleBeforeImportingInbox(t *testing.T) {
	destinationDirectory := t.TempDir()
	inboxDirectory := t.TempDir()
	service := testService(destinationDirectory, t.TempDir(), func(filePath string, athleteID int64) (*strava.Activity, error) {
		return &strava.Activity{
			Id:             44,
			Type:           "Run",
			StartDate:      "2026-06-05T08:00:00Z",
			StartDateLocal: "2026-06-05T10:00:00+02:00",
			Distance:       5_000,
			ElapsedTime:    1500,
		}, nil
	}, func() {})
	service.fitInboxPath = func() (string, bool) { return inboxDirectory, true }
	service.garminSyncBin = func() (string, bool) { return "/usr/local/bin/garmin-fit-sync", true }
	service.runSyncModule = func(binPath string, inboxPath string, sourcePath string) (FITSyncModuleResult, error) {
		if binPath == "" || inboxPath != inboxDirectory {
			t.Fatalf("unexpected module args: bin=%s inbox=%s", binPath, inboxPath)
		}
		if err := os.WriteFile(filepath.Join(inboxPath, "run.fit"), []byte("fit"), 0o644); err != nil {
			t.Fatal(err)
		}
		return FITSyncModuleResult{
			Status:      "ok",
			Message:     "Copied 1 FIT file(s).",
			Backend:     "filesystem",
			InboxPath:   inboxPath,
			CopiedFiles: 1,
		}, nil
	}

	result := service.Synchronize("test")

	if result.FIT.SyncModule == nil || result.FIT.SyncModule.CopiedFiles != 1 {
		t.Fatalf("expected sync module diagnostics, got %#v", result.FIT.SyncModule)
	}
	if result.FIT.ImportedFiles != 1 {
		t.Fatalf("expected 1 imported module file, got %d", result.FIT.ImportedFiles)
	}
	if _, err := os.Stat(filepath.Join(destinationDirectory, "2026", "run.fit")); err != nil {
		t.Fatalf("expected imported module FIT in year directory: %v", err)
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
		fitInboxPath:     func() (string, bool) { return "", false },
		garminSourcePath: func() (string, bool) { return "", false },
		garminSyncBin:    func() (string, bool) { return "", false },
		runSyncModule:    runGarminFITSyncModule,
		reloadProvider:   reload,
		volumesRoot:      volumesRoot,
		now:              time.Now,
		lastResult: SyncResult{
			Status: "idle",
		},
	}
}
