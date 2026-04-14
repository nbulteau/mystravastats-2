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
	Route{
		"GetHealthDetails",
		"GET",
		"/api/health/details",
		getHealthDetails,
	},
	Route{
		"GetAthlete",
		"GET",
		"/api/athletes/me",
		getAthlete,
	},
	Route{
		"GetAthleteHeartRateZones",
		"GET",
		"/api/athletes/me/heart-rate-zones",
		getAthleteHeartRateZones,
	},
	Route{
		"PutAthleteHeartRateZones",
		"PUT",
		"/api/athletes/me/heart-rate-zones",
		putAthleteHeartRateZones,
	},
	Route{
		"GetActivitiesByActivityType",
		"GET",
		"/api/activities",
		getActivitiesByActivityType,
	},
	Route{
		"GetExportCSV",
		"GET",
		"/api/activities/csv",
		getExportCSV,
	},
	Route{
		"GetDetailedActivity",
		"GET",
		"/api/activities/{activityId}",
		getDetailedActivity,
	},
	Route{
		"GetStatisticsByActivityType",
		"GET",
		"/api/statistics",
		getStatisticsByActivityType,
	},
	Route{
		"GetPersonalRecordsTimelineByActivityType",
		"GET",
		"/api/statistics/personal-records-timeline",
		getPersonalRecordsTimelineByActivityType,
	},
	Route{
		"GetHeartRateZoneAnalysisByActivityType",
		"GET",
		"/api/statistics/heart-rate-zones",
		getHeartRateZoneAnalysisByActivityType,
	},
	Route{
		"GetMapsGPX",
		"GET",
		"/api/maps/gpx",
		getMapsGPX,
	},
	Route{
		"GetChartsDistanceByPeriod",
		"GET",
		"/api/charts/distance-by-period",
		getChartsDistanceByPeriod,
	},
	Route{
		"GetChartsElevationByPeriod",
		"GET",
		"/api/charts/elevation-by-period",
		getChartsElevationByPeriod,
	},
	Route{
		"GetChartsAverageSpeedByPeriod",
		"GET",
		"/api/charts/average-speed-by-period",
		getChartsAverageSpeedByPeriod,
	},
	Route{
		"GetChartsAverageCadenceByPeriod",
		"GET",
		"/api/charts/average-cadence-by-period",
		getChartsAverageCadenceByPeriod,
	},
	Route{
		"GetDashboard",
		"GET",
		"/api/dashboard",
		getDashboard,
	},
	Route{
		"GetDashboardCumulativeDataByYear",
		"GET",
		"/api/dashboard/cumulative-data-per-year",
		getDashboardCumulativeDataByYear,
	},
	Route{
		"GetDashboardEddingtonNumber",
		"GET",
		"/api/dashboard/eddington-number",
		getDashboardEddingtonNumber,
	},
	Route{
		"GetDashboardActivityHeatmap",
		"GET",
		"/api/dashboard/activity-heatmap",
		getDashboardActivityHeatmap,
	},
	Route{
		"GetBadges",
		"GET",
		"/api/badges",
		getBadges,
	},
}
