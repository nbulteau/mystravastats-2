package infrastructure

import (
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"testing"
)

func TestComputeMapPassages_CountsActivitiesInsteadOfGpsPoints(t *testing.T) {
	// GIVEN
	activities := []*strava.Activity{
		mapPassageTestActivity(1, business.Ride.String(), [][]float64{
			{48.0000, 2.0000},
			{48.0010, 2.0000},
			{48.0020, 2.0000},
			{48.0030, 2.0000},
		}),
		mapPassageTestActivity(2, business.Ride.String(), [][]float64{
			{48.0000, 2.0000},
			{48.0003, 2.0000},
			{48.0006, 2.0000},
			{48.0009, 2.0000},
			{48.0012, 2.0000},
			{48.0015, 2.0000},
			{48.0018, 2.0000},
			{48.0021, 2.0000},
			{48.0024, 2.0000},
			{48.0027, 2.0000},
			{48.0030, 2.0000},
		}),
	}

	// WHEN
	result := computeMapPassages(activities, nil)

	// THEN
	if result.IncludedActivities != 2 {
		t.Fatalf("expected 2 included activities, got %d", result.IncludedActivities)
	}
	if len(result.Segments) == 0 {
		t.Fatal("expected passage segments")
	}
	if result.Segments[0].PassageCount != 2 {
		t.Fatalf("expected max passage count 2, got %d", result.Segments[0].PassageCount)
	}
	if result.Segments[0].ActivityTypeCounts[business.Ride.String()] != 2 {
		t.Fatalf("expected ride type count 2, got %d", result.Segments[0].ActivityTypeCounts[business.Ride.String()])
	}
}

func TestComputeMapPassages_CountsOnePassagePerActivityPerCorridor(t *testing.T) {
	// GIVEN
	activities := []*strava.Activity{
		mapPassageTestActivity(1, business.Ride.String(), [][]float64{
			{48.0000, 2.0000},
			{48.0010, 2.0000},
			{48.0020, 2.0000},
			{48.0010, 2.0000},
			{48.0000, 2.0000},
		}),
	}

	// WHEN
	result := computeMapPassages(activities, nil)

	// THEN
	if result.IncludedActivities != 1 {
		t.Fatalf("expected 1 included activity, got %d", result.IncludedActivities)
	}
	for _, segment := range result.Segments {
		if segment.PassageCount != 1 {
			t.Fatalf("expected repeated corridor to count once, got %d", segment.PassageCount)
		}
	}
}

func TestComputeMapPassages_IgnoresExcludedActivities(t *testing.T) {
	// GIVEN
	activities := []*strava.Activity{
		mapPassageTestActivity(1, business.Ride.String(), [][]float64{
			{48.0000, 2.0000},
			{48.0010, 2.0000},
			{48.0020, 2.0000},
		}),
		mapPassageTestActivity(2, business.Ride.String(), [][]float64{
			{48.0000, 2.0000},
			{48.0010, 2.0000},
			{48.0020, 2.0000},
		}),
	}
	exclusions := map[int64]business.DataQualityExclusion{
		2: {ActivityID: 2},
	}

	// WHEN
	result := computeMapPassages(activities, exclusions)

	// THEN
	if result.ExcludedActivities != 1 {
		t.Fatalf("expected 1 excluded activity, got %d", result.ExcludedActivities)
	}
	if result.IncludedActivities != 1 {
		t.Fatalf("expected 1 included activity, got %d", result.IncludedActivities)
	}
	for _, segment := range result.Segments {
		if segment.PassageCount != 1 {
			t.Fatalf("expected excluded activity to be ignored, got passage count %d", segment.PassageCount)
		}
	}
}

func TestComputeMapPassagesWithOptions_FiltersAndCapsAllYearsPayload(t *testing.T) {
	// GIVEN
	activities := []*strava.Activity{
		mapPassageTestActivity(1, business.Ride.String(), [][]float64{
			{48.0000, 2.0000},
			{48.0010, 2.0000},
			{48.0020, 2.0000},
		}),
		mapPassageTestActivity(2, business.Ride.String(), [][]float64{
			{48.0000, 2.0000},
			{48.0010, 2.0000},
			{48.0020, 2.0000},
		}),
		mapPassageTestActivity(3, business.Ride.String(), [][]float64{
			{49.0000, 3.0000},
			{49.0010, 3.0000},
			{49.0020, 3.0000},
		}),
	}

	// WHEN
	result := computeMapPassagesWithOptions(activities, nil, mapPassageOptions{
		resolutionMeters: 250,
		minPassageCount:  2,
		maxSegments:      1,
	})

	// THEN
	if result.ResolutionMeters != 250 {
		t.Fatalf("expected resolution 250, got %d", result.ResolutionMeters)
	}
	if result.MinPassageCount != 2 {
		t.Fatalf("expected min passage count 2, got %d", result.MinPassageCount)
	}
	if len(result.Segments) != 1 {
		t.Fatalf("expected capped segment list of 1, got %d", len(result.Segments))
	}
	if result.Segments[0].PassageCount != 2 {
		t.Fatalf("expected repeated corridor passage count 2, got %d", result.Segments[0].PassageCount)
	}
	if result.OmittedSegments == 0 {
		t.Fatal("expected omitted segments to be reported")
	}
}

func mapPassageTestActivity(id int64, activityType string, coordinates [][]float64) *strava.Activity {
	return &strava.Activity{
		Id:   id,
		Type: activityType,
		Stream: &strava.Stream{
			LatLng: &strava.LatLngStream{Data: coordinates},
		},
	}
}
