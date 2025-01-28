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
		"GetActivitiesByActivityType",
		"GET",
		"/api/activities",
		getActivitiesByActivityType,
	},
	Route{
		"GetStatisticsByActivityType",
		"GET",
		"/api/statistics",
		getStatisticsByActivityType,
	},
	Route{
		"GetAthlete",
		"GET",
		"/api/athletes/me",
		getAthlete,
	},
}
