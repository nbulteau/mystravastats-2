package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"mystravastats/domain/business"
	domainStatistics "mystravastats/domain/statistics"
	"mystravastats/domain/strava"
	activitiesApp "mystravastats/internal/activities/application"
	chartsApp "mystravastats/internal/charts/application"
	dashboardApp "mystravastats/internal/dashboard/application"
	dashboardDomain "mystravastats/internal/dashboard/domain"
	healthApp "mystravastats/internal/health/application"
	heartrateApp "mystravastats/internal/heartrate/application"
	routesApp "mystravastats/internal/routes/application"
	routesDomain "mystravastats/internal/routes/domain"
	segmentsApp "mystravastats/internal/segments/application"
	segmentsDomain "mystravastats/internal/segments/domain"
	statisticsApp "mystravastats/internal/statistics/application"

	"github.com/gorilla/mux"
)

type contractErrorResponse struct {
	Message     string `json:"message"`
	Description string `json:"description"`
	Code        int    `json:"code"`
}

type contractActivitiesReaderStub struct {
	activities []*strava.Activity
}

func (stub *contractActivitiesReaderStub) FindActivitiesByYearAndTypes(_ *int, _ ...business.ActivityType) []*strava.Activity {
	return stub.activities
}

type contractDetailedActivityReaderStub struct {
	activity *strava.DetailedActivity
	err      error
}

func (stub *contractDetailedActivityReaderStub) FindDetailedActivityByID(_ int64) (*strava.DetailedActivity, error) {
	return stub.activity, stub.err
}

type contractHealthReaderStub struct {
	details map[string]any
}

func (stub *contractHealthReaderStub) FindCacheHealthDetails() map[string]any {
	return stub.details
}

type contractStatistic struct {
	label string
	value string
}

func (stub *contractStatistic) Label() string {
	return stub.label
}

func (stub *contractStatistic) Value() string {
	return stub.value
}

func (stub *contractStatistic) Activity() *business.ActivityShort {
	return nil
}

type contractStatisticsReaderStub struct {
	statistics []domainStatistics.Statistic
}

func (stub *contractStatisticsReaderStub) FindStatisticsByYearAndTypes(_ *int, _ ...business.ActivityType) []domainStatistics.Statistic {
	return stub.statistics
}

type contractPersonalRecordsTimelineReaderStub struct {
	timeline []business.PersonalRecordTimelineEntry
}

func (stub *contractPersonalRecordsTimelineReaderStub) FindPersonalRecordsTimelineByYearMetricAndTypes(_ *int, _ *string, _ ...business.ActivityType) []business.PersonalRecordTimelineEntry {
	return stub.timeline
}

type contractHeartRateReaderStub struct {
	settings business.HeartRateZoneSettings
	analysis business.HeartRateZoneAnalysis
}

func (stub *contractHeartRateReaderStub) FindHeartRateZoneSettings() business.HeartRateZoneSettings {
	return stub.settings
}

func (stub *contractHeartRateReaderStub) SaveHeartRateZoneSettings(settings business.HeartRateZoneSettings) business.HeartRateZoneSettings {
	stub.settings = settings
	return settings
}

func (stub *contractHeartRateReaderStub) FindHeartRateZoneAnalysisByYearAndTypes(_ *int, _ ...business.ActivityType) business.HeartRateZoneAnalysis {
	return stub.analysis
}

type contractSegmentsReaderStub struct {
	progression business.SegmentClimbProgression
	summaries   []business.SegmentClimbTargetSummary
	efforts     []business.SegmentClimbAttempt
	summary     *segmentsDomain.SegmentSummary
}

func (stub *contractSegmentsReaderStub) FindSegmentClimbProgressionByYearMetricTargetAndTypes(_ *int, _ *string, _ *string, _ *int64, _ ...business.ActivityType) business.SegmentClimbProgression {
	return stub.progression
}

func (stub *contractSegmentsReaderStub) FindSegmentsByYearMetricQueryRangeAndTypes(_ *int, _ *string, _ *string, _ *string, _ *string, _ ...business.ActivityType) []business.SegmentClimbTargetSummary {
	return stub.summaries
}

func (stub *contractSegmentsReaderStub) FindSegmentEffortsByYearMetricRangeAndTypes(_ *int, _ *string, _ int64, _ *string, _ *string, _ ...business.ActivityType) []business.SegmentClimbAttempt {
	return stub.efforts
}

func (stub *contractSegmentsReaderStub) FindSegmentSummaryByYearMetricRangeAndTypes(_ *int, _ *string, _ int64, _ *string, _ *string, _ ...business.ActivityType) *segmentsDomain.SegmentSummary {
	return stub.summary
}

type contractRoutesReaderStub struct {
	result routesDomain.RouteExplorerResult
}

func (stub *contractRoutesReaderStub) FindRouteExplorerByYearAndTypes(_ *int, _ routesDomain.RouteExplorerRequest, _ ...business.ActivityType) routesDomain.RouteExplorerResult {
	return stub.result
}

type contractChartsReaderStub struct {
	result []map[string]float64
}

func (stub *contractChartsReaderStub) FindDistanceByPeriod(_ *int, _ business.Period, _ ...business.ActivityType) []map[string]float64 {
	return stub.result
}

func (stub *contractChartsReaderStub) FindElevationByPeriod(_ *int, _ business.Period, _ ...business.ActivityType) []map[string]float64 {
	return stub.result
}

func (stub *contractChartsReaderStub) FindAverageSpeedByPeriod(_ *int, _ business.Period, _ ...business.ActivityType) []map[string]float64 {
	return stub.result
}

func (stub *contractChartsReaderStub) FindAverageCadenceByPeriod(_ *int, _ business.Period, _ ...business.ActivityType) []map[string]float64 {
	return stub.result
}

type contractDashboardReaderStub struct {
	dashboardData       business.DashboardData
	cumulativeDistance  map[string]map[string]float64
	cumulativeElevation map[string]map[string]float64
	heatmap             map[string]map[string]dashboardDomain.ActivityHeatmapDay
	eddington           business.EddingtonNumber
}

func (stub *contractDashboardReaderStub) FindDashboardData(_ ...business.ActivityType) business.DashboardData {
	return stub.dashboardData
}

func (stub *contractDashboardReaderStub) FindCumulativeDistancePerYear(_ ...business.ActivityType) map[string]map[string]float64 {
	return stub.cumulativeDistance
}

func (stub *contractDashboardReaderStub) FindCumulativeElevationPerYear(_ ...business.ActivityType) map[string]map[string]float64 {
	return stub.cumulativeElevation
}

func (stub *contractDashboardReaderStub) FindActivityHeatmap(_ ...business.ActivityType) map[string]map[string]dashboardDomain.ActivityHeatmapDay {
	return stub.heatmap
}

func (stub *contractDashboardReaderStub) FindEddingtonNumber(_ ...business.ActivityType) business.EddingtonNumber {
	return stub.eddington
}

func setTestContainer(t *testing.T, testContainer *container) {
	t.Helper()

	previousContainer := sharedContainer
	previousOnce := containerOnce

	containerOnce = sync.Once{}
	containerOnce.Do(func() {})
	sharedContainer = testContainer

	t.Cleanup(func() {
		sharedContainer = previousContainer
		containerOnce = previousOnce
	})
}

func TestGetHealthDetails_Returns200AndBody(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	setTestContainer(t, &container{
		getCacheHealthDetailsUseCase: healthApp.NewGetCacheHealthDetailsUseCase(&contractHealthReaderStub{
			details: map[string]any{"status": "ok"},
		}),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/health/details", nil)
	recorder := httptest.NewRecorder()

	getHealthDetails(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if response["status"] != "ok" {
		t.Fatalf("expected status=ok, got %+v", response)
	}
}

func TestGetActivitiesByActivityType_Returns200AndArray(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	year := 2025
	activities := []*strava.Activity{
		{
			Id:             123,
			Name:           "Morning Ride",
			SportType:      "Ride",
			Type:           "Ride",
			StartDate:      "2025-01-02T10:00:00Z",
			StartDateLocal: "2025-01-02T10:00:00Z",
		},
	}

	setTestContainer(t, &container{
		listActivitiesUseCase: activitiesApp.NewListActivitiesUseCase(&contractActivitiesReaderStub{
			activities: activities,
		}),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/activities?year=2025&activityType=Ride", nil)
	recorder := httptest.NewRecorder()

	getActivitiesByActivityType(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response []map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if len(response) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(response))
	}
	if got := int64(response[0]["id"].(float64)); got != 123 {
		t.Fatalf("expected id=123, got %d", got)
	}
	if got := response[0]["name"]; got != "Morning Ride" {
		t.Fatalf("expected name=Morning Ride, got %v", got)
	}
	if year != 2025 {
		t.Fatalf("sanity check failed on year forwarding setup")
	}
}

func TestGetDetailedActivity_InvalidID_Returns400(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	request := httptest.NewRequest(http.MethodGet, "/api/activities/invalid", nil)
	request = mux.SetURLVars(request, map[string]string{"activityId": "invalid"})
	recorder := httptest.NewRecorder()

	getDetailedActivity(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}

	var response contractErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if response.Message != "Invalid request parameters" {
		t.Fatalf("expected message 'Invalid request parameters', got %q", response.Message)
	}
}

func TestGetDetailedActivity_NotFound_Returns404(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	setTestContainer(t, &container{
		getDetailedActivityUseCase: activitiesApp.NewGetDetailedActivityUseCase(&contractDetailedActivityReaderStub{
			activity: nil,
			err:      nil,
		}),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/activities/42", nil)
	request = mux.SetURLVars(request, map[string]string{"activityId": "42"})
	recorder := httptest.NewRecorder()

	getDetailedActivity(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", recorder.Code)
	}

	var response contractErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if response.Message != "Resource not found" {
		t.Fatalf("expected message 'Resource not found', got %q", response.Message)
	}
}

func TestGetChartsDistanceByPeriod_WithoutYear_Returns400(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	request := httptest.NewRequest(http.MethodGet, "/api/charts/distance-by-period?activityType=Ride&period=MONTHS", nil)
	recorder := httptest.NewRecorder()

	getChartsDistanceByPeriod(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}

	var response contractErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if response.Description != "year is required" {
		t.Fatalf("expected description 'year is required', got %q", response.Description)
	}
}

func TestGetBadges_InvalidBadgeSet_Returns400(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	request := httptest.NewRequest(http.MethodGet, "/api/badges?activityType=Ride&badgeSet=INVALID", nil)
	recorder := httptest.NewRecorder()

	getBadges(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}

	var response contractErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if response.Message != "Invalid request parameters" {
		t.Fatalf("expected message 'Invalid request parameters', got %q", response.Message)
	}
}

func TestGetStatisticsByActivityType_Returns200AndArray(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	setTestContainer(t, &container{
		listStatisticsUseCase: statisticsApp.NewListStatisticsUseCase(&contractStatisticsReaderStub{
			statistics: []domainStatistics.Statistic{
				&contractStatistic{label: "Total distance", value: "120.00 km"},
			},
		}),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/statistics?year=2025&activityType=Ride", nil)
	recorder := httptest.NewRecorder()

	getStatisticsByActivityType(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response []map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if len(response) != 1 {
		t.Fatalf("expected 1 statistic, got %d", len(response))
	}
	if got := response[0]["label"]; got != "Total distance" {
		t.Fatalf("expected label 'Total distance', got %v", got)
	}
}

func TestGetPersonalRecordsTimelineByActivityType_Returns200AndArray(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	setTestContainer(t, &container{
		listPersonalRecordsTimelineUseCase: statisticsApp.NewListPersonalRecordsTimelineUseCase(&contractPersonalRecordsTimelineReaderStub{
			timeline: []business.PersonalRecordTimelineEntry{
				{
					MetricKey:    "best-distance-1h",
					MetricLabel:  "Best 1 h",
					ActivityDate: "2025-08-04",
					Value:        "29.77 km",
					Activity:     business.ActivityShort{Id: 100, Name: "Ride", Type: business.Ride},
				},
			},
		}),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/statistics/personal-records-timeline?year=2025&activityType=Ride", nil)
	recorder := httptest.NewRecorder()

	getPersonalRecordsTimelineByActivityType(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response []map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if len(response) != 1 {
		t.Fatalf("expected 1 timeline event, got %d", len(response))
	}
	if got := response[0]["metricKey"]; got != "best-distance-1h" {
		t.Fatalf("expected metricKey 'best-distance-1h', got %v", got)
	}
}

func TestGetHeartRateZoneAnalysisByActivityType_Returns200(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	setTestContainer(t, &container{
		getHeartRateZoneAnalysisUseCase: heartrateApp.NewGetHeartRateZoneAnalysisUseCase(&contractHeartRateReaderStub{
			analysis: business.HeartRateZoneAnalysis{
				HasHeartRateData: true,
				Zones:            []business.HeartRateZoneDistribution{},
				Activities:       []business.HeartRateZoneActivitySummary{},
				ByMonth:          []business.HeartRateZonePeriodSummary{},
				ByYear:           []business.HeartRateZonePeriodSummary{},
			},
		}),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/statistics/heart-rate-zones?year=2025&activityType=Ride", nil)
	recorder := httptest.NewRecorder()

	getHeartRateZoneAnalysisByActivityType(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if got := response["hasHeartRateData"]; got != true {
		t.Fatalf("expected hasHeartRateData=true, got %v", got)
	}
}

func TestGetSegmentClimbProgressionByActivityType_InvalidTargetID_Returns400(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	request := httptest.NewRequest(http.MethodGet, "/api/statistics/segment-climb-progression?activityType=Ride&targetId=bad", nil)
	recorder := httptest.NewRecorder()

	getSegmentClimbProgressionByActivityType(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestGetSegmentClimbProgressionByActivityType_Returns200(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	setTestContainer(t, &container{
		getSegmentClimbProgressionUseCase: segmentsApp.NewGetSegmentClimbProgressionUseCase(&contractSegmentsReaderStub{
			progression: business.SegmentClimbProgression{
				Metric: "TIME",
				Targets: []business.SegmentClimbTargetSummary{
					{TargetId: 10, TargetName: "Test Segment"},
				},
				Attempts: []business.SegmentClimbAttempt{},
			},
		}),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/statistics/segment-climb-progression?activityType=Ride", nil)
	recorder := httptest.NewRecorder()

	getSegmentClimbProgressionByActivityType(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if got := response["metric"]; got != "TIME" {
		t.Fatalf("expected metric TIME, got %v", got)
	}
}

func TestGetSegmentsByActivityType_InvalidFromDate_Returns400(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	request := httptest.NewRequest(http.MethodGet, "/api/segments?activityType=Ride&from=2025-99-99", nil)
	recorder := httptest.NewRecorder()

	getSegmentsByActivityType(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestGetSegmentEffortsByActivityType_InvalidSegmentID_Returns400(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	request := httptest.NewRequest(http.MethodGet, "/api/segments/bad/efforts?activityType=Ride", nil)
	request = mux.SetURLVars(request, map[string]string{"segmentId": "bad"})
	recorder := httptest.NewRecorder()

	getSegmentEffortsByActivityType(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestGetSegmentSummaryByActivityType_NotFound_Returns404(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	setTestContainer(t, &container{
		getSegmentSummaryUseCase: segmentsApp.NewGetSegmentSummaryUseCase(&contractSegmentsReaderStub{
			summary: nil,
		}),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/segments/10/summary?activityType=Ride", nil)
	request = mux.SetURLVars(request, map[string]string{"segmentId": "10"})
	recorder := httptest.NewRecorder()

	getSegmentSummaryByActivityType(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", recorder.Code)
	}
}

func TestGetChartsAverageSpeedByPeriod_Returns200(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	year := 2025
	setTestContainer(t, &container{
		getAverageSpeedByPeriodUseCase: chartsApp.NewGetAverageSpeedByPeriodUseCase(&contractChartsReaderStub{
			result: []map[string]float64{{"01": 26.5}},
		}),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/charts/average-speed-by-period?year=2025&activityType=Ride&period=MONTHS", nil)
	recorder := httptest.NewRecorder()

	getChartsAverageSpeedByPeriod(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response []map[string]float64
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if len(response) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(response))
	}
	if year != 2025 {
		t.Fatalf("sanity check failed on year forwarding setup")
	}
}

func TestGetChartsElevationByPeriod_InvalidPeriod_Returns400(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	request := httptest.NewRequest(http.MethodGet, "/api/charts/elevation-by-period?year=2025&activityType=Ride&period=INVALID", nil)
	recorder := httptest.NewRecorder()

	getChartsElevationByPeriod(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestGetDashboard_Returns200(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	setTestContainer(t, &container{
		getDashboardDataUseCase: dashboardApp.NewGetDashboardDataUseCase(&contractDashboardReaderStub{
			dashboardData: business.DashboardData{
				NbActivities: map[string]int{"2025": 42},
			},
		}),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/dashboard?activityType=Ride", nil)
	recorder := httptest.NewRecorder()

	getDashboard(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if _, exists := response["nbActivitiesByYear"]; !exists {
		t.Fatalf("expected nbActivitiesByYear in response, got %+v", response)
	}
}

func TestGetDashboardCumulativeDataByYear_Returns200(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	setTestContainer(t, &container{
		getCumulativeDataPerYearUseCase: dashboardApp.NewGetCumulativeDataPerYearUseCase(&contractDashboardReaderStub{
			cumulativeDistance:  map[string]map[string]float64{"2025": {"01-01": 10}},
			cumulativeElevation: map[string]map[string]float64{"2025": {"01-01": 100}},
		}),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/dashboard/cumulative-data-per-year?activityType=Ride", nil)
	recorder := httptest.NewRecorder()

	getDashboardCumulativeDataByYear(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestGetDashboardActivityHeatmap_Returns200(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	setTestContainer(t, &container{
		getActivityHeatmapUseCase: dashboardApp.NewGetActivityHeatmapUseCase(&contractDashboardReaderStub{
			heatmap: map[string]map[string]dashboardDomain.ActivityHeatmapDay{
				"2025": {"01-01": {DistanceKm: 10.5, ActivityCount: 1}},
			},
		}),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/dashboard/activity-heatmap?activityType=Ride", nil)
	recorder := httptest.NewRecorder()

	getDashboardActivityHeatmap(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestGetDashboardEddingtonNumber_Returns200(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	setTestContainer(t, &container{
		getEddingtonNumberUseCase: dashboardApp.NewGetEddingtonNumberUseCase(&contractDashboardReaderStub{
			eddington: business.EddingtonNumber{Number: 55, List: []int{60, 58, 55}},
		}),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/dashboard/eddington-number?activityType=Ride", nil)
	recorder := httptest.NewRecorder()

	getDashboardEddingtonNumber(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if got := int(response["eddingtonNumber"].(float64)); got != 55 {
		t.Fatalf("expected eddingtonNumber=55, got %d", got)
	}
}

func TestGenerateTargetRoutesByActivityType_ReturnsGeneratedRoutesAndCachesForGPX(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	generatedRouteCache.mu.Lock()
	generatedRouteCache.items = map[string]generatedRouteCacheEntry{}
	generatedRouteCache.mu.Unlock()

	setTestContainer(t, &container{
		getRouteExplorerUseCase: routesApp.NewGetRouteExplorerUseCase(&contractRoutesReaderStub{
			result: routesDomain.RouteExplorerResult{
				RoadGraphLoops: []routesDomain.RouteRecommendation{
					{
						RouteID:        "generated-loop-1",
						Activity:       business.ActivityShort{Id: 1234, Name: "Generated loop", Type: business.Ride},
						DistanceKm:     42.1,
						ElevationGainM: 860,
						DurationSec:    7200,
						VariantType:    routesDomain.RouteVariantRoadGraph,
						MatchScore:     91.4,
						Reasons:        []string{"Road-graph generated loop"},
						PreviewLatLng:  [][]float64{{45.0, 6.0}, {45.01, 6.02}, {45.0, 6.0}},
					},
				},
			},
		}),
	})

	request := httptest.NewRequest(http.MethodPost, "/api/routes/generate/target?activityType=Ride&year=2025", strings.NewReader(`{
	  "startPoint": {"lat": 45.1, "lng": 6.1},
	  "routeType": "RIDE",
	  "startDirection": "N",
	  "distanceTargetKm": 42,
	  "elevationTargetM": 900,
	  "variantCount": 3
	}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	generateTargetRoutesByActivityType(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (%s)", recorder.Code, recorder.Body.String())
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	routes, ok := response["routes"].([]any)
	if !ok || len(routes) == 0 {
		t.Fatalf("expected routes array, got %+v", response)
	}

	gpxRequest := httptest.NewRequest(http.MethodGet, "/api/routes/generated-loop-1/gpx", nil)
	gpxRequest = mux.SetURLVars(gpxRequest, map[string]string{"routeId": "generated-loop-1"})
	gpxRecorder := httptest.NewRecorder()
	getGeneratedRouteGPXByID(gpxRecorder, gpxRequest)

	if gpxRecorder.Code != http.StatusOK {
		t.Fatalf("expected gpx status 200, got %d (%s)", gpxRecorder.Code, gpxRecorder.Body.String())
	}
	if !strings.Contains(gpxRecorder.Body.String(), "<gpx") {
		t.Fatalf("expected GPX payload, got %s", gpxRecorder.Body.String())
	}
}

func TestGenerateShapeRoutesByActivityType_InvalidShapeInputType_Returns400(t *testing.T) {
	// GIVEN
	// WHEN
	// THEN
	request := httptest.NewRequest(http.MethodPost, "/api/routes/generate/shape?activityType=Ride", strings.NewReader(`{
	  "shapeInputType": "invalid",
	  "shapeData": "[[45.0,6.0],[45.1,6.1]]",
	  "routeType": "RIDE"
	}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	generateShapeRoutesByActivityType(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}

	var response contractErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if response.Message != "Invalid request body" {
		t.Fatalf("expected message 'Invalid request body', got %q", response.Message)
	}
}
