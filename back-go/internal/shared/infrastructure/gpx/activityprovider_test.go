package gpx

import (
	"os"
	"path/filepath"
	"testing"

	"mystravastats/internal/shared/domain/business"
)

func TestDecodeGPXActivity_MapsTrackToActivityWithStreams(t *testing.T) {
	// GIVEN
	gpxFile := writeTestGPX(t, t.TempDir(), "2026", "ride.gpx", `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test" xmlns:gpxtpx="http://www.garmin.com/xmlschemas/TrackPointExtension/v1">
  <trk>
    <name>Morning Ride</name>
    <type>cycling</type>
    <trkseg>
      <trkpt lat="48.1000" lon="-1.6000">
        <ele>10</ele>
        <time>2026-04-01T08:00:00Z</time>
        <extensions><gpxtpx:hr>120</gpxtpx:hr><gpxtpx:cad>80</gpxtpx:cad><power>180</power></extensions>
      </trkpt>
      <trkpt lat="48.1010" lon="-1.6000">
        <ele>20</ele>
        <time>2026-04-01T08:01:00Z</time>
        <extensions><gpxtpx:hr>130</gpxtpx:hr><gpxtpx:cad>82</gpxtpx:cad><power>190</power></extensions>
      </trkpt>
    </trkseg>
  </trk>
</gpx>`)

	// WHEN
	activity, err := DecodeGPXActivity(gpxFile, 42, 2026)

	// THEN
	if err != nil {
		t.Fatalf("expected GPX activity to decode, got error: %v", err)
	}
	if activity.Name != "Morning Ride" {
		t.Fatalf("expected name Morning Ride, got %q", activity.Name)
	}
	if activity.SportType != business.Ride.String() {
		t.Fatalf("expected Ride sport type, got %q", activity.SportType)
	}
	if activity.Distance <= 0 {
		t.Fatalf("expected positive distance, got %f", activity.Distance)
	}
	if activity.TotalElevationGain != 10 {
		t.Fatalf("expected 10m elevation gain, got %f", activity.TotalElevationGain)
	}
	if activity.Stream == nil || activity.Stream.HeartRate == nil || activity.Stream.Cadence == nil || activity.Stream.Watts == nil {
		t.Fatalf("expected heart-rate, cadence and power streams, got %#v", activity.Stream)
	}
}

func TestGPXActivityProvider_FiltersActivitiesByYearAndType(t *testing.T) {
	// GIVEN
	root := t.TempDir()
	writeTestGPX(t, root, "2026", "run.gpx", `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test">
  <trk><name>Run</name><type>running</type><trkseg>
    <trkpt lat="48.1000" lon="-1.6000"><time>2026-01-01T08:00:00Z</time></trkpt>
    <trkpt lat="48.1010" lon="-1.6000"><time>2026-01-01T08:05:00Z</time></trkpt>
  </trkseg></trk>
</gpx>`)
	writeTestGPX(t, root, "2025", "ride.gpx", `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test">
  <trk><name>Ride</name><type>cycling</type><trkseg>
    <trkpt lat="48.1000" lon="-1.6000"><time>2025-01-01T08:00:00Z</time></trkpt>
    <trkpt lat="48.1010" lon="-1.6000"><time>2025-01-01T08:05:00Z</time></trkpt>
  </trkseg></trk>
</gpx>`)
	provider := NewGPXActivityProvider(root)
	year := 2026

	// WHEN
	activities := provider.GetActivitiesByYearAndActivityTypes(&year, business.Run)

	// THEN
	if len(activities) != 1 {
		t.Fatalf("expected one 2026 run activity, got %d", len(activities))
	}
	if activities[0].SportType != business.Run.String() {
		t.Fatalf("expected Run, got %s", activities[0].SportType)
	}
}

func writeTestGPX(t *testing.T, root string, year string, name string, content string) string {
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
