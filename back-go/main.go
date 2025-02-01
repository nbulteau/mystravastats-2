package main

import (
	"fmt"
	"github.com/rs/cors"
	"log"
	"mystravastats/adapters/localrepository"
	"mystravastats/api"
	"net/http"
)

func testStravaRepository() {
	// Initialize the repository with a cache directory
	stravaCache := "./strava-cache"
	repo := localrepository.NewStravaRepository(stravaCache)

	// Example client ID and year
	clientId := "41902"
	year := 2023

	// Initialize local storage for the client ID
	repo.InitLocalStorageForClientId(clientId)

	// Load athlete from cache
	athlete := repo.LoadAthleteFromCache(clientId)
	fmt.Printf("Loaded athlete: %+v\n", athlete)

	// Load activities from cache
	activities := repo.LoadActivitiesFromCache(clientId, year)
	fmt.Printf("Loaded %d activities for year %d\n", len(activities), year)

	// Load detailed activity from cache
	if len(activities) > 0 {
		activityId := activities[0].Id
		detailedActivity := repo.LoadDetailedActivityFromCache(clientId, year, activityId)
		if detailedActivity != nil {
			fmt.Printf("Loaded detailed activity: %+v\n", detailedActivity)
		} else {
			fmt.Printf("No detailed activity found for activity ID %d\n", activityId)
		}
	}

	// Load activity streams from cache
	if len(activities) > 0 {
		activity := activities[0]
		stream := repo.LoadActivitiesStreamsFromCache(clientId, year, activity)
		if stream != nil {
			fmt.Printf("Loaded stream for activity ID %d: %+v\n", activity.Id, stream)
		} else {
			fmt.Printf("No stream found for activity ID %d\n", activity.Id)
		}
	}
}

func main() {

	// Test the Strava repository
	//testStravaRepository()

	// Create a new CORS handler
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*"}, // Allow any port on localhost
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	handler := c.Handler(api.NewRouter())

	log.Fatal(http.ListenAndServe(":8080", handler))
}
