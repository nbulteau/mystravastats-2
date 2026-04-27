package infrastructure

import (
	"math"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"testing"
)

func TestAnalyzeLocalActivitiesDetectsMissingStreamsAndGpsGlitch(t *testing.T) {
	activity := &strava.Activity{
		Id:             123,
		Name:           "Suspicious ride",
		Type:           "Ride",
		SportType:      "Ride",
		Distance:       10000,
		ElapsedTime:    600,
		MovingTime:     600,
		StartDateLocal: "2026-04-26T08:00:00Z",
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 10, 20}},
			Time:     strava.TimeStream{Data: []int{0, 1, 2}},
			LatLng: &strava.LatLngStream{Data: [][]float64{
				{48.0, -1.0},
				{49.0, -1.0},
				{49.0001, -1.0},
			}},
		},
	}

	report := AnalyzeLocalActivities("fit", "/tmp/fit", []*strava.Activity{activity})

	if report.Summary.Status != "warning" {
		t.Fatalf("expected warning status, got %s", report.Summary.Status)
	}
	if report.Summary.ImpactedActivities != 1 {
		t.Fatalf("expected one impacted activity, got %d", report.Summary.ImpactedActivities)
	}
	if got := report.Summary.ByCategory[string(business.DataQualityCategoryGPSGlitch)]; got != 1 {
		t.Fatalf("expected one GPS glitch issue, got %d", got)
	}
	if got := report.Summary.ByCategory[string(business.DataQualityCategoryMissingStreamField)]; got != 1 {
		t.Fatalf("expected one missing stream field issue, got %d", got)
	}
}

func TestAnalyzeActivitiesClassifiesStreamCoverageSeparately(t *testing.T) {
	report := AnalyzeLocalActivities("strava", "strava-cache", []*strava.Activity{
		{
			Id:             11,
			Name:           "Power meter ride",
			Type:           "Ride",
			SportType:      "Ride",
			Distance:       10000,
			ElapsedTime:    1800,
			MovingTime:     1800,
			AverageWatts:   180,
			DeviceWatts:    true,
			StartDateLocal: "2026-04-26T08:00:00Z",
			Stream:         completeStream(),
		},
		{
			Id:             12,
			Name:           "Estimated power ride",
			Type:           "Ride",
			SportType:      "Ride",
			Distance:       10000,
			ElapsedTime:    1800,
			MovingTime:     1800,
			AverageWatts:   160,
			DeviceWatts:    false,
			StartDateLocal: "2026-04-26T08:00:00Z",
			Stream:         completeStream(),
		},
	})

	if got := report.Summary.ByCategory[string(business.DataQualityCategoryStreamDataCoverage)]; got != 1 {
		t.Fatalf("expected one stream coverage issue, got %d", got)
	}
}

func TestAnalyzeActivitiesDetectsDownloadableStravaMissingStream(t *testing.T) {
	report := AnalyzeLocalActivities("strava", "strava-cache", []*strava.Activity{
		{
			Id:             99,
			Name:           "Uncached stream ride",
			Type:           "Ride",
			SportType:      "Ride",
			Distance:       10000,
			ElapsedTime:    1800,
			MovingTime:     1800,
			StartDateLocal: "2026-04-26T08:00:00Z",
			UploadId:       12345,
		},
	})

	if got := report.Summary.ByCategory[string(business.DataQualityCategoryMissingStream)]; got != 1 {
		t.Fatalf("expected one missing stream issue, got %d", got)
	}
	if report.Summary.Status != "ok" {
		t.Fatalf("expected info-only missing stream to keep ok status, got %s", report.Summary.Status)
	}
}

func TestAnalyzeLocalActivitiesDetectsStravaSummaryAnomalies(t *testing.T) {
	report := AnalyzeLocalActivities("strava", "strava-cache", []*strava.Activity{
		{
			Id:             1,
			Name:           "Fast suspicious ride",
			Type:           "Ride",
			SportType:      "Ride",
			Distance:       100000,
			ElapsedTime:    1200,
			MovingTime:     1200,
			StartDateLocal: "2026-04-26T08:00:00Z",
		},
	})

	if report.Summary.Status != "warning" {
		t.Fatalf("expected warning status, got %s", report.Summary.Status)
	}
	if got := report.Summary.ByCategory[string(business.DataQualityCategoryInvalidValue)]; got != 1 {
		t.Fatalf("expected one invalid value issue, got %d", got)
	}
}

func TestAnalyzeActivitiesMarksExcludedIssues(t *testing.T) {
	activity := &strava.Activity{
		Id:             42,
		Name:           "Excluded suspicious ride",
		Type:           "Ride",
		SportType:      "Ride",
		Distance:       100000,
		ElapsedTime:    1200,
		MovingTime:     1200,
		StartDateLocal: "2026-04-26T08:00:00Z",
	}

	report := AnalyzeActivities("strava", "strava-cache", []*strava.Activity{activity}, []business.DataQualityExclusion{
		{ActivityID: 42, Source: "STRAVA", ExcludedAt: "2026-04-26T09:00:00Z"},
	})

	if report.Summary.ExcludedActivities != 1 {
		t.Fatalf("expected one excluded activity, got %d", report.Summary.ExcludedActivities)
	}
	if len(report.Issues) != 1 || !report.Issues[0].ExcludedFromStats {
		t.Fatalf("expected excluded issue, got %+v", report.Issues)
	}
}

func TestAnalyzeLocalActivitiesAddsSafeCorrectionSuggestions(t *testing.T) {
	activity := &strava.Activity{
		Id:                 77,
		Name:               "Correctable ride",
		Type:               "Ride",
		SportType:          "Ride",
		Distance:           1000,
		ElapsedTime:        600,
		MovingTime:         600,
		TotalElevationGain: 250,
		StartDateLocal:     "2026-04-26T08:00:00Z",
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 500, 1000}},
			Time:     strava.TimeStream{Data: []int{0, 5, 10}},
			LatLng: &strava.LatLngStream{Data: [][]float64{
				{48.0, -1.0},
				{49.0, -1.0},
				{48.0001, -1.0},
			}},
			Altitude: &strava.AltitudeStream{Data: []float64{100, 320, 101}},
		},
	}

	report := AnalyzeLocalActivities("fit", "/tmp/fit", []*strava.Activity{activity})

	gpsIssue := findDataQualityIssue(report.Issues, business.DataQualityCategoryGPSGlitch)
	if gpsIssue == nil || gpsIssue.Correction == nil || gpsIssue.Correction.Safety != business.DataQualityCorrectionSafetySafe {
		t.Fatalf("expected GPS glitch to expose a safe correction, got %+v", gpsIssue)
	}
	altitudeIssue := findDataQualityIssue(report.Issues, business.DataQualityCategoryAltitudeSpike)
	if altitudeIssue == nil || altitudeIssue.Correction == nil || altitudeIssue.Correction.Safety != business.DataQualityCorrectionSafetySafe {
		t.Fatalf("expected altitude spike to expose a safe correction, got %+v", altitudeIssue)
	}
	if report.Summary.SafeCorrectionCount != 2 {
		t.Fatalf("expected two safe corrections, got %d", report.Summary.SafeCorrectionCount)
	}
}

func TestAnalyzeLocalActivitiesAddsSafeScalarCorrectionSuggestions(t *testing.T) {
	activity := &strava.Activity{
		Id:                 88,
		Name:               "Non serializable ride",
		Type:               "Ride",
		SportType:          "Ride",
		Distance:           math.NaN(),
		AverageSpeed:       math.NaN(),
		MaxSpeed:           math.Inf(1),
		ElapsedTime:        120,
		MovingTime:         120,
		TotalElevationGain: math.NaN(),
		StartDateLocal:     "2026-04-26T08:00:00Z",
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 0, 0}},
			Time:     strava.TimeStream{Data: []int{0, 60, 120}},
			LatLng: &strava.LatLngStream{Data: [][]float64{
				{48.0, -1.0},
				{48.001, -1.0},
				{48.002, -1.0},
			}},
			Altitude: &strava.AltitudeStream{Data: []float64{50, 60, 70}},
		},
	}

	report := AnalyzeLocalActivities("fit", "/tmp/fit", []*strava.Activity{activity})

	for _, field := range []string{"distance", "average_speed", "max_speed", "total_elevation_gain"} {
		issue := findDataQualityIssueByField(report.Issues, business.DataQualityCategoryInvalidValue, field)
		if issue == nil {
			t.Fatalf("expected invalid value issue for %s, got %+v", field, report.Issues)
		}
		if issue.Correction == nil || issue.Correction.Safety != business.DataQualityCorrectionSafetySafe {
			t.Fatalf("expected %s to expose a safe correction, got %+v", field, issue)
		}
	}
	if report.Summary.SafeCorrectionCount != 4 {
		t.Fatalf("expected four safe scalar corrections, got %d", report.Summary.SafeCorrectionCount)
	}
}

func findDataQualityIssue(issues []business.DataQualityIssue, category business.DataQualityCategory) *business.DataQualityIssue {
	for index := range issues {
		if issues[index].Category == category {
			return &issues[index]
		}
	}
	return nil
}

func findDataQualityIssueByField(issues []business.DataQualityIssue, category business.DataQualityCategory, field string) *business.DataQualityIssue {
	for index := range issues {
		if issues[index].Category == category && issues[index].Field == field {
			return &issues[index]
		}
	}
	return nil
}

func completeStream() *strava.Stream {
	return &strava.Stream{
		Distance: strava.DistanceStream{Data: []float64{0, 5000, 10000}},
		Time:     strava.TimeStream{Data: []int{0, 900, 1800}},
		LatLng: &strava.LatLngStream{Data: [][]float64{
			{48.0, -1.0},
			{48.01, -1.0},
			{48.02, -1.0},
		}},
		Altitude: &strava.AltitudeStream{Data: []float64{50, 60, 70}},
	}
}
