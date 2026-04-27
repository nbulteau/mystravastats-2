package api

import "net/http"

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Routes []Route

var routes = Routes{
	{Name: "GetHealthDetails", Method: "GET", Pattern: "/api/health/details", HandlerFunc: getHealthDetails},
	{Name: "GetDataQualityIssues", Method: "GET", Pattern: "/api/data-quality/issues", HandlerFunc: getDataQualityIssues},
	{Name: "PutDataQualityStatsExclusion", Method: "PUT", Pattern: "/api/data-quality/exclusions/{activityId}", HandlerFunc: putDataQualityStatsExclusion},
	{Name: "DeleteDataQualityStatsExclusion", Method: "DELETE", Pattern: "/api/data-quality/exclusions/{activityId}", HandlerFunc: deleteDataQualityStatsExclusion},
	{Name: "PostSourceModePreview", Method: "POST", Pattern: "/api/source-modes/preview", HandlerFunc: postSourceModePreview},
	{Name: "GetAthlete", Method: "GET", Pattern: "/api/athletes/me", HandlerFunc: getAthlete},
	{Name: "GetAthleteHeartRateZones", Method: "GET", Pattern: "/api/athletes/me/heart-rate-zones", HandlerFunc: getAthleteHeartRateZones},
	{Name: "PutAthleteHeartRateZones", Method: "PUT", Pattern: "/api/athletes/me/heart-rate-zones", HandlerFunc: putAthleteHeartRateZones},
	{Name: "GetActivitiesByActivityType", Method: "GET", Pattern: "/api/activities", HandlerFunc: getActivitiesByActivityType},
	{Name: "GetExportCSV", Method: "GET", Pattern: "/api/activities/csv", HandlerFunc: getExportCSV},
	{Name: "GetDetailedActivity", Method: "GET", Pattern: "/api/activities/{activityId}", HandlerFunc: getDetailedActivity},
	{Name: "GetStatisticsByActivityType", Method: "GET", Pattern: "/api/statistics", HandlerFunc: getStatisticsByActivityType},
	{Name: "GetPersonalRecordsTimelineByActivityType", Method: "GET", Pattern: "/api/statistics/personal-records-timeline", HandlerFunc: getPersonalRecordsTimelineByActivityType},
	{Name: "GetHeartRateZoneAnalysisByActivityType", Method: "GET", Pattern: "/api/statistics/heart-rate-zones", HandlerFunc: getHeartRateZoneAnalysisByActivityType},
	{Name: "GetSegmentClimbProgressionByActivityType", Method: "GET", Pattern: "/api/statistics/segment-climb-progression", HandlerFunc: getSegmentClimbProgressionByActivityType},
	{Name: "GetGearAnalysisByActivityType", Method: "GET", Pattern: "/api/gear-analysis", HandlerFunc: getGearAnalysisByActivityType},
	{Name: "GetSegmentsByActivityType", Method: "GET", Pattern: "/api/segments", HandlerFunc: getSegmentsByActivityType},
	{Name: "GetSegmentEffortsByActivityType", Method: "GET", Pattern: "/api/segments/{segmentId}/efforts", HandlerFunc: getSegmentEffortsByActivityType},
	{Name: "GetSegmentSummaryByActivityType", Method: "GET", Pattern: "/api/segments/{segmentId}/summary", HandlerFunc: getSegmentSummaryByActivityType},
	{Name: "GetRouteRecommendationsByActivityType", Method: "GET", Pattern: "/api/routes/recommendations", HandlerFunc: getRouteRecommendationsByActivityType},
	{Name: "GetRouteRecommendationGpxByActivityType", Method: "GET", Pattern: "/api/routes/recommendations/gpx", HandlerFunc: getRouteRecommendationGPXByActivityType},
	{Name: "GenerateTargetRoutesByActivityType", Method: "POST", Pattern: "/api/routes/generate/target", HandlerFunc: generateTargetRoutesByActivityType},
	{Name: "GenerateShapeRoutesByActivityType", Method: "POST", Pattern: "/api/routes/generate/shape", HandlerFunc: generateShapeRoutesByActivityType},
	{Name: "GetGeneratedRouteGpx", Method: "GET", Pattern: "/api/routes/{routeId}/gpx", HandlerFunc: getGeneratedRouteGPXByID},
	{Name: "GetMapsGPX", Method: "GET", Pattern: "/api/maps/gpx", HandlerFunc: getMapsGPX},
	{Name: "GetChartsDistanceByPeriod", Method: "GET", Pattern: "/api/charts/distance-by-period", HandlerFunc: getChartsDistanceByPeriod},
	{Name: "GetChartsElevationByPeriod", Method: "GET", Pattern: "/api/charts/elevation-by-period", HandlerFunc: getChartsElevationByPeriod},
	{Name: "GetChartsAverageSpeedByPeriod", Method: "GET", Pattern: "/api/charts/average-speed-by-period", HandlerFunc: getChartsAverageSpeedByPeriod},
	{Name: "GetChartsAverageCadenceByPeriod", Method: "GET", Pattern: "/api/charts/average-cadence-by-period", HandlerFunc: getChartsAverageCadenceByPeriod},
	{Name: "GetDashboard", Method: "GET", Pattern: "/api/dashboard", HandlerFunc: getDashboard},
	{Name: "GetDashboardCumulativeDataByYear", Method: "GET", Pattern: "/api/dashboard/cumulative-data-per-year", HandlerFunc: getDashboardCumulativeDataByYear},
	{Name: "GetDashboardEddingtonNumber", Method: "GET", Pattern: "/api/dashboard/eddington-number", HandlerFunc: getDashboardEddingtonNumber},
	{Name: "GetDashboardActivityHeatmap", Method: "GET", Pattern: "/api/dashboard/activity-heatmap", HandlerFunc: getDashboardActivityHeatmap},
	{Name: "GetDashboardAnnualGoals", Method: "GET", Pattern: "/api/dashboard/annual-goals", HandlerFunc: getDashboardAnnualGoals},
	{Name: "PutDashboardAnnualGoals", Method: "PUT", Pattern: "/api/dashboard/annual-goals", HandlerFunc: putDashboardAnnualGoals},
	{Name: "GetBadges", Method: "GET", Pattern: "/api/badges", HandlerFunc: getBadges},
}
