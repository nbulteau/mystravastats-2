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
		"GetAthlete",
		"GET",
		"/api/athletes/me",
		getAthlete,
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
		"GetBadges",
		"GET",
		"/api/badges",
		getBadges,
	},
}
