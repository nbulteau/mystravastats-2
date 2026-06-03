package infrastructure

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

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

func TestPreviewSourceMode_StravaReportsOAuthEnrollmentStatus(t *testing.T) {
	// GIVEN
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".strava"), []byte("clientId=12345\nclientSecret=secret\nuseCache=false\n"), 0o600); err != nil {
		t.Fatalf("failed to write .strava: %v", err)
	}
	token := `{
  "access_token": "access",
  "refresh_token": "refresh",
  "expires_at": ` + strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10) + `,
  "scope": "read_all,activity:read_all,profile:read_all",
  "athlete": { "id": 42, "firstname": "Ada", "lastname": "Lovelace" }
}`
	if err := os.WriteFile(filepath.Join(root, ".strava-token.json"), []byte(token), 0o600); err != nil {
		t.Fatalf("failed to write token: %v", err)
	}
	adapter := NewSourceModeServiceAdapter()

	// WHEN
	preview := adapter.PreviewSourceMode(business.SourceModePreviewRequest{
		Mode: "STRAVA",
		Path: root,
	})

	// THEN
	if preview.StravaOAuth == nil {
		t.Fatal("expected Strava OAuth status")
	}
	if preview.StravaOAuth.Status != "ready" {
		t.Fatalf("expected ready OAuth status, got %#v", preview.StravaOAuth)
	}
	if !preview.StravaOAuth.CredentialsPresent || !preview.StravaOAuth.TokenPresent || !preview.StravaOAuth.TokenReadable {
		t.Fatalf("expected credentials and token to be detected, got %#v", preview.StravaOAuth)
	}
	if preview.StravaOAuth.AthleteID != "42" || preview.StravaOAuth.AthleteName != "Ada Lovelace" {
		t.Fatalf("expected athlete metadata, got %#v", preview.StravaOAuth)
	}
	if !strings.Contains(preview.StravaOAuth.SetupCommand, "setup-strava-oauth.mjs") {
		t.Fatalf("expected setup command, got %q", preview.StravaOAuth.SetupCommand)
	}
}

func TestWriteSourceModeEnv_SavesFITAndUnsetsGPX(t *testing.T) {
	root := t.TempDir()
	previousCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to read cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(previousCwd); err != nil {
			t.Fatalf("failed to restore cwd: %v", err)
		}
	}()

	if err := os.WriteFile(".env", []byte("STRAVA_CACHE_PATH=\"strava-cache\"\nexport GPX_FILES_PATH=\"/old/gpx\"\nOPEN_BROWSER=false\n"), 0o600); err != nil {
		t.Fatalf("failed to seed .env: %v", err)
	}

	envPath, err := writeSourceModeEnv(business.SourceModeFIT, "FIT_FILES_PATH", "/new fit")
	if err != nil {
		t.Fatalf("failed to write source mode env: %v", err)
	}

	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read .env: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "FIT_FILES_PATH=\"/new fit\"") {
		t.Fatalf("expected FIT path to be saved, got:\n%s", text)
	}
	if strings.Contains(text, "GPX_FILES_PATH") {
		t.Fatalf("expected GPX path to be removed, got:\n%s", text)
	}
	if !strings.Contains(text, "STRAVA_CACHE_PATH=\"strava-cache\"") || !strings.Contains(text, "OPEN_BROWSER=false") {
		t.Fatalf("expected unrelated config to be preserved, got:\n%s", text)
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
