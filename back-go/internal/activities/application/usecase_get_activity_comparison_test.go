package application

import (
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"testing"
)

type activityComparisonReaderStub struct {
	activities       []*strava.Activity
	cachedDetails    map[int64]*strava.DetailedActivity
	receivedYear     *int
	receivedTypes    []business.ActivityType
	cachedDetailHits []int64
}

func (stub *activityComparisonReaderStub) FindActivitiesByYearAndTypes(year *int, activityTypes ...business.ActivityType) []*strava.Activity {
	stub.receivedYear = year
	stub.receivedTypes = append([]business.ActivityType(nil), activityTypes...)
	return stub.activities
}

func (stub *activityComparisonReaderStub) FindCachedDetailedActivityByID(activityID int64) *strava.DetailedActivity {
	stub.cachedDetailHits = append(stub.cachedDetailHits, activityID)
	if stub.cachedDetails == nil {
		return nil
	}
	return stub.cachedDetails[activityID]
}

func TestGetActivityComparisonUseCase_Execute_ComparesSimilarActivitiesAndCommonSegments(t *testing.T) {
	// GIVEN
	target := comparisonDetailedActivity(100, "Target", "2025-05-10T09:00:00Z", 50_000, 500, 6.9)
	target.SegmentEfforts = []strava.SegmentEffort{
		comparisonSegmentEffort(10, "Shared climb"),
		comparisonSegmentEffort(20, "Only target"),
	}
	reader := &activityComparisonReaderStub{
		activities: []*strava.Activity{
			comparisonActivity(1, "Close A", "2025-05-01T09:00:00Z", 51_000, 520, 6.3),
			comparisonActivity(2, "Close B", "2025-06-01T09:00:00Z", 48_000, 460, 6.2),
			comparisonActivity(3, "Too far", "2025-07-01T09:00:00Z", 110_000, 1_800, 5.0),
			comparisonActivity(100, "Target", "2025-05-10T09:00:00Z", 50_000, 500, 6.9),
		},
		cachedDetails: map[int64]*strava.DetailedActivity{
			1: {Id: 1, SegmentEfforts: []strava.SegmentEffort{comparisonSegmentEffort(10, "Shared climb")}},
			2: {Id: 2, SegmentEfforts: []strava.SegmentEffort{comparisonSegmentEffort(30, "Other segment")}},
		},
	}
	useCase := NewGetActivityComparisonUseCase(reader)

	// WHEN
	comparison := useCase.Execute(target)

	// THEN
	if comparison == nil {
		t.Fatal("expected comparison")
	}
	if reader.receivedYear == nil || *reader.receivedYear != 2025 {
		t.Fatalf("expected same-season year 2025, got %v", reader.receivedYear)
	}
	if len(reader.receivedTypes) != 1 || reader.receivedTypes[0] != business.Ride {
		t.Fatalf("expected same sport Ride, got %#v", reader.receivedTypes)
	}
	if comparison.Criteria.SampleSize != 2 {
		t.Fatalf("expected two similar activities, got %d", comparison.Criteria.SampleSize)
	}
	if comparison.Status != "faster" {
		t.Fatalf("expected faster status, got %q", comparison.Status)
	}
	if len(comparison.SimilarActivities) != 2 {
		t.Fatalf("expected two ranked activities, got %d", len(comparison.SimilarActivities))
	}
	if comparison.SimilarActivities[0].ID != 1 {
		t.Fatalf("expected closest activity first, got id=%d", comparison.SimilarActivities[0].ID)
	}
	if len(comparison.CommonSegments) != 1 {
		t.Fatalf("expected one common segment, got %#v", comparison.CommonSegments)
	}
	if comparison.CommonSegments[0].ID != 10 || comparison.CommonSegments[0].MatchCount != 1 {
		t.Fatalf("unexpected common segment: %#v", comparison.CommonSegments[0])
	}
}

func TestGetActivityComparisonUseCase_Execute_ReturnsInsufficientDataWhenNothingSimilar(t *testing.T) {
	// GIVEN
	target := comparisonDetailedActivity(100, "Target", "2025-05-10T09:00:00Z", 50_000, 500, 6.9)
	reader := &activityComparisonReaderStub{
		activities: []*strava.Activity{
			comparisonActivity(3, "Too far", "2025-07-01T09:00:00Z", 110_000, 1_800, 5.0),
		},
	}
	useCase := NewGetActivityComparisonUseCase(reader)

	// WHEN
	comparison := useCase.Execute(target)

	// THEN
	if comparison == nil {
		t.Fatal("expected comparison")
	}
	if comparison.Status != "insufficient-data" {
		t.Fatalf("expected insufficient-data, got %q", comparison.Status)
	}
	if comparison.Criteria.SampleSize != 0 {
		t.Fatalf("expected no similar activities, got %d", comparison.Criteria.SampleSize)
	}
}

func TestGetActivityComparisonUseCase_Execute_KeepsFlatActivitiesWithSmallElevationDelta(t *testing.T) {
	// GIVEN
	target := comparisonDetailedActivity(100, "Flat target", "2025-05-10T09:00:00Z", 40_000, 0, 6.8)
	reader := &activityComparisonReaderStub{
		activities: []*strava.Activity{
			comparisonActivity(4, "Flat close", "2025-05-03T09:00:00Z", 40_800, 35, 6.7),
		},
	}
	useCase := NewGetActivityComparisonUseCase(reader)

	// WHEN
	comparison := useCase.Execute(target)

	// THEN
	if comparison == nil {
		t.Fatal("expected comparison")
	}
	if comparison.Criteria.SampleSize != 1 {
		t.Fatalf("expected flat activity to remain comparable, got sample size %d", comparison.Criteria.SampleSize)
	}
	if comparison.SimilarActivities[0].ID != 4 {
		t.Fatalf("expected flat close activity, got id=%d", comparison.SimilarActivities[0].ID)
	}
}

func comparisonDetailedActivity(id int64, name string, date string, distance float64, elevation float64, speed float64) *strava.DetailedActivity {
	return &strava.DetailedActivity{
		Id:                 id,
		Name:               name,
		Type:               "Ride",
		SportType:          "Ride",
		StartDate:          date,
		StartDateLocal:     date,
		Distance:           distance,
		TotalElevationGain: elevation,
		MovingTime:         int(distance / speed),
		AverageSpeed:       speed,
		AverageHeartrate:   140,
		AverageWatts:       210,
		AverageCadence:     82,
	}
}

func comparisonActivity(id int64, name string, date string, distance float64, elevation float64, speed float64) *strava.Activity {
	return &strava.Activity{
		Id:                 id,
		Name:               name,
		Type:               "Ride",
		SportType:          "Ride",
		StartDate:          date,
		StartDateLocal:     date,
		Distance:           distance,
		TotalElevationGain: elevation,
		MovingTime:         int(distance / speed),
		AverageSpeed:       speed,
		AverageHeartrate:   138,
		AverageWatts:       190,
		AverageCadence:     80,
	}
}

func comparisonSegmentEffort(segmentID int64, name string) strava.SegmentEffort {
	return strava.SegmentEffort{
		Segment: strava.Segment{
			Id:   segmentID,
			Name: name,
		},
	}
}
