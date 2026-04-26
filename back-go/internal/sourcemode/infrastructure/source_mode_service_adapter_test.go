package infrastructure

import (
	"os"
	"path/filepath"
	"testing"

	"mystravastats/internal/shared/domain/business"
)

func TestPreviewSourceMode_GPXValidatesYearFoldersAndFields(t *testing.T) {
	// GIVEN
	root := t.TempDir()
	writeSourceModeGPX(t, root, "2026", "ride.gpx", `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test">
  <trk><name>Ride</name><type>cycling</type><trkseg>
    <trkpt lat="48.1000" lon="-1.6000"><ele>10</ele><time>2026-01-01T08:00:00Z</time></trkpt>
    <trkpt lat="48.1010" lon="-1.6000"><ele>15</ele><time>2026-01-01T08:05:00Z</time></trkpt>
  </trkseg></trk>
</gpx>`)
	adapter := NewSourceModeServiceAdapter()

	// WHEN
	preview := adapter.PreviewSourceMode(business.SourceModePreviewRequest{
		Mode: "GPX",
		Path: root,
	})

	// THEN
	if !preview.Supported || !preview.Readable || !preview.ValidStructure {
		t.Fatalf("expected supported readable valid GPX preview, got %#v", preview)
	}
	if preview.FileCount != 1 || preview.ValidFileCount != 1 || preview.ActivityCount != 1 {
		t.Fatalf("expected one valid activity, got %#v", preview)
	}
	if len(preview.Years) != 1 || preview.Years[0].Year != "2026" {
		t.Fatalf("expected 2026 year preview, got %#v", preview.Years)
	}
	if preview.Errors == nil || preview.Recommendations == nil || preview.MissingFields == nil {
		t.Fatalf("expected JSON list fields to be normalized to empty slices, got %#v", preview)
	}
	if preview.ActiveMode != business.SourceModeStrava {
		t.Fatalf("expected default active mode STRAVA, got %s", preview.ActiveMode)
	}
	if preview.ActivationCommand == "" || preview.Environment == nil {
		t.Fatalf("expected activation assistant fields, got %#v", preview)
	}
	if len(preview.Environment) == 0 || preview.Environment[0].Key != "GPX_FILES_PATH" || preview.Environment[0].Value != root {
		t.Fatalf("expected GPX environment activation, got %#v", preview.Environment)
	}
}

func writeSourceModeGPX(t *testing.T, root string, year string, name string, content string) string {
	t.Helper()
	yearDirectory := filepath.Join(root, year)
	if err := os.MkdirAll(yearDirectory, 0o700); err != nil {
		t.Fatalf("failed to create year directory: %v", err)
	}
	filePath := filepath.Join(yearDirectory, name)
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write GPX fixture: %v", err)
	}
	return filePath
}
