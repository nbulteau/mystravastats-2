package infrastructure

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"

	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
)

type dataQualityParityFixture struct {
	Cases []dataQualityParityCase `json:"cases"`
}

type dataQualityParityCase struct {
	Source     string             `json:"source"`
	SourcePath string             `json:"sourcePath"`
	Activities []*strava.Activity `json:"activities"`
}

type dataQualityParitySnapshot struct {
	Cases []dataQualityParityCaseSnapshot `json:"cases"`
}

type dataQualityParityCaseSnapshot struct {
	Source                string                           `json:"source"`
	Summary               dataQualitySummarySnapshot       `json:"summary"`
	Issues                []dataQualityIssueSnapshot       `json:"issues"`
	SafeCorrectionPreview dataQualityCorrectionSnapshotSet `json:"safeCorrectionPreview"`
}

type dataQualitySummarySnapshot struct {
	Status              string         `json:"status"`
	IssueCount          int            `json:"issueCount"`
	ImpactedActivities  int            `json:"impactedActivities"`
	ExcludedActivities  int            `json:"excludedActivities"`
	CorrectionCount     int            `json:"correctionCount"`
	SafeCorrectionCount int            `json:"safeCorrectionCount"`
	ManualReviewCount   int            `json:"manualReviewCount"`
	BySeverity          map[string]int `json:"bySeverity"`
	ByCategory          map[string]int `json:"byCategory"`
}

type dataQualityIssueSnapshot struct {
	ID                  string                               `json:"id"`
	ActivityID          int64                                `json:"activityId"`
	Severity            business.DataQualitySeverity         `json:"severity"`
	Category            business.DataQualityCategory         `json:"category"`
	Field               string                               `json:"field"`
	CorrectionAvailable bool                                 `json:"correctionAvailable"`
	CorrectionSafety    business.DataQualityCorrectionSafety `json:"correctionSafety,omitempty"`
	CorrectionType      business.DataQualityCorrectionType   `json:"correctionType,omitempty"`
}

type dataQualityCorrectionSnapshotSet struct {
	Summary     dataQualityCorrectionSummarySnapshot `json:"summary"`
	Corrections []dataQualityCorrectionSnapshot      `json:"corrections"`
}

type dataQualityCorrectionSummarySnapshot struct {
	SafeCorrectionCount       int      `json:"safeCorrectionCount"`
	ManualReviewCount         int      `json:"manualReviewCount"`
	UnsupportedIssueCount     int      `json:"unsupportedIssueCount"`
	ActivityCount             int      `json:"activityCount"`
	DistanceDeltaMeters       float64  `json:"distanceDeltaMeters"`
	ElevationDeltaMeters      float64  `json:"elevationDeltaMeters"`
	ModifiedFields            []string `json:"modifiedFields"`
	PotentiallyImpactsRecords bool     `json:"potentiallyImpactsRecords"`
}

type dataQualityCorrectionSnapshot struct {
	ID             string                               `json:"id"`
	IssueID        string                               `json:"issueId"`
	ActivityID     int64                                `json:"activityId"`
	Type           business.DataQualityCorrectionType   `json:"type"`
	Safety         business.DataQualityCorrectionSafety `json:"safety"`
	PointIndexes   []int                                `json:"pointIndexes,omitempty"`
	ModifiedFields []string                             `json:"modifiedFields"`
	Impact         dataQualityCorrectionImpactSnapshot  `json:"impact"`
}

type dataQualityCorrectionImpactSnapshot struct {
	DistanceMetersBefore  float64 `json:"distanceMetersBefore"`
	DistanceMetersAfter   float64 `json:"distanceMetersAfter"`
	ElevationMetersBefore float64 `json:"elevationMetersBefore"`
	ElevationMetersAfter  float64 `json:"elevationMetersAfter"`
	MaxSpeedBefore        float64 `json:"maxSpeedBefore"`
	MaxSpeedAfter         float64 `json:"maxSpeedAfter"`
	DistanceDeltaMeters   float64 `json:"distanceDeltaMeters"`
	ElevationDeltaMeters  float64 `json:"elevationDeltaMeters"`
}

func TestDataQualityLocalActivityFixtureParity(t *testing.T) {
	fixturePath := dataQualityFixturePath(t, "local-activity-anomalies.json")
	expectedPath := dataQualityFixturePath(t, "expected-local-activity-anomalies.snapshot.json")

	fixture := readDataQualityJSON[dataQualityParityFixture](t, fixturePath)
	actual := dataQualitySnapshotFromFixture(fixture)
	if os.Getenv("UPDATE_DATA_QUALITY_SNAPSHOT") == "1" {
		writeDataQualitySnapshot(t, expectedPath, actual)
	}
	expected := readDataQualityJSON[dataQualityParitySnapshot](t, expectedPath)

	if !reflect.DeepEqual(expected, actual) {
		expectedJSON := mustMarshalDataQualityJSON(t, expected)
		actualJSON := mustMarshalDataQualityJSON(t, actual)
		t.Fatalf("data quality parity snapshot mismatch\nexpected:\n%s\nactual:\n%s", expectedJSON, actualJSON)
	}
}

func dataQualitySnapshotFromFixture(fixture dataQualityParityFixture) dataQualityParitySnapshot {
	snapshot := dataQualityParitySnapshot{
		Cases: make([]dataQualityParityCaseSnapshot, 0, len(fixture.Cases)),
	}
	for _, fixtureCase := range fixture.Cases {
		report := AnalyzeLocalActivities(fixtureCase.Source, fixtureCase.SourcePath, fixtureCase.Activities)
		preview := dataQualitySafeCorrectionPreviewForReport(report, fixtureCase.Activities)
		snapshot.Cases = append(snapshot.Cases, dataQualityParityCaseSnapshot{
			Source:                fixtureCase.Source,
			Summary:               snapshotSummary(report.Summary),
			Issues:                snapshotIssues(report.Issues),
			SafeCorrectionPreview: snapshotCorrections(preview),
		})
	}
	return snapshot
}

func dataQualitySafeCorrectionPreviewForReport(report business.DataQualityReport, activities []*strava.Activity) business.DataQualityCorrectionPreview {
	activityByID := make(map[int64]*strava.Activity, len(activities))
	for _, activity := range activities {
		if activity != nil {
			activityByID[activity.Id] = activity
		}
	}

	preview := newCorrectionPreview("safe_batch")
	manualReviewCount := 0
	unsupportedCount := 0
	for _, issue := range report.Issues {
		correction, warnings, blockingReasons, ok := buildCorrectionForIssue(activityByID[issue.ActivityID], issue)
		preview.Warnings = append(preview.Warnings, warnings...)
		if ok && correction.Safety == business.DataQualityCorrectionSafetySafe {
			preview.Corrections = append(preview.Corrections, correction)
			continue
		}
		if ok && correction.Safety == business.DataQualityCorrectionSafetyManual {
			manualReviewCount++
			continue
		}
		if len(blockingReasons) > 0 {
			unsupportedCount++
			preview.BlockingReasons = append(preview.BlockingReasons, blockingReasons...)
		}
	}
	preview.Corrections = dedupeCorrections(preview.Corrections)
	preview.Summary = summarizeCorrections(preview.Corrections, manualReviewCount, unsupportedCount)
	return preview
}

func snapshotSummary(summary business.DataQualitySummary) dataQualitySummarySnapshot {
	return dataQualitySummarySnapshot{
		Status:              summary.Status,
		IssueCount:          summary.IssueCount,
		ImpactedActivities:  summary.ImpactedActivities,
		ExcludedActivities:  summary.ExcludedActivities,
		CorrectionCount:     summary.CorrectionCount,
		SafeCorrectionCount: summary.SafeCorrectionCount,
		ManualReviewCount:   summary.ManualReviewCount,
		BySeverity:          summary.BySeverity,
		ByCategory:          summary.ByCategory,
	}
}

func snapshotIssues(issues []business.DataQualityIssue) []dataQualityIssueSnapshot {
	result := make([]dataQualityIssueSnapshot, 0, len(issues))
	for _, issue := range issues {
		item := dataQualityIssueSnapshot{
			ID:         issue.ID,
			ActivityID: issue.ActivityID,
			Severity:   issue.Severity,
			Category:   issue.Category,
			Field:      issue.Field,
		}
		if issue.Correction != nil && issue.Correction.Available {
			item.CorrectionAvailable = true
			item.CorrectionSafety = issue.Correction.Safety
			item.CorrectionType = issue.Correction.Type
		}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ActivityID != result[j].ActivityID {
			return result[i].ActivityID < result[j].ActivityID
		}
		if result[i].Category != result[j].Category {
			return result[i].Category < result[j].Category
		}
		if result[i].Field != result[j].Field {
			return result[i].Field < result[j].Field
		}
		return result[i].ID < result[j].ID
	})
	return result
}

func snapshotCorrections(preview business.DataQualityCorrectionPreview) dataQualityCorrectionSnapshotSet {
	corrections := make([]dataQualityCorrectionSnapshot, 0, len(preview.Corrections))
	for _, correction := range preview.Corrections {
		corrections = append(corrections, dataQualityCorrectionSnapshot{
			ID:             correction.ID,
			IssueID:        correction.IssueID,
			ActivityID:     correction.ActivityID,
			Type:           correction.Type,
			Safety:         correction.Safety,
			PointIndexes:   correction.PointIndexes,
			ModifiedFields: append([]string{}, correction.ModifiedFields...),
			Impact: dataQualityCorrectionImpactSnapshot{
				DistanceMetersBefore:  roundDataQualityFloat(correction.Impact.DistanceMetersBefore),
				DistanceMetersAfter:   roundDataQualityFloat(correction.Impact.DistanceMetersAfter),
				ElevationMetersBefore: roundDataQualityFloat(correction.Impact.ElevationMetersBefore),
				ElevationMetersAfter:  roundDataQualityFloat(correction.Impact.ElevationMetersAfter),
				MaxSpeedBefore:        roundDataQualityFloat(correction.Impact.MaxSpeedBefore),
				MaxSpeedAfter:         roundDataQualityFloat(correction.Impact.MaxSpeedAfter),
				DistanceDeltaMeters:   roundDataQualityFloat(correction.Impact.DistanceDeltaMeters),
				ElevationDeltaMeters:  roundDataQualityFloat(correction.Impact.ElevationDeltaMeters),
			},
		})
	}
	return dataQualityCorrectionSnapshotSet{
		Summary: dataQualityCorrectionSummarySnapshot{
			SafeCorrectionCount:       preview.Summary.SafeCorrectionCount,
			ManualReviewCount:         preview.Summary.ManualReviewCount,
			UnsupportedIssueCount:     preview.Summary.UnsupportedIssueCount,
			ActivityCount:             preview.Summary.ActivityCount,
			DistanceDeltaMeters:       roundDataQualityFloat(preview.Summary.DistanceDeltaMeters),
			ElevationDeltaMeters:      roundDataQualityFloat(preview.Summary.ElevationDeltaMeters),
			ModifiedFields:            append([]string{}, preview.Summary.ModifiedFields...),
			PotentiallyImpactsRecords: preview.Summary.PotentiallyImpactsRecords,
		},
		Corrections: corrections,
	}
}

func dataQualityFixturePath(t *testing.T, name string) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to resolve current test path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../../.."))
	return filepath.Join(repoRoot, "test-fixtures", "data-quality", name)
}

func readDataQualityJSON[T any](t *testing.T, path string) T {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return value
}

func writeDataQualitySnapshot(t *testing.T, path string, snapshot dataQualityParitySnapshot) {
	t.Helper()
	data := mustMarshalDataQualityJSON(t, snapshot)
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustMarshalDataQualityJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal data quality snapshot: %v", err)
	}
	return data
}

func roundDataQualityFloat(value float64) float64 {
	return math.Round(value*1000) / 1000
}
