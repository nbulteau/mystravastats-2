package helpers

import (
	"log"
	"mystravastats/internal/platform/runtimeconfig"

	"github.com/joho/godotenv"
)

var StravaCachePath string

func init() {
	log.Println("🚀 Starting mystravasts...")
	StravaCachePath = loadEnvironmentVariables()
}

func loadEnvironmentVariables() string {
	if err := godotenv.Overload(); err != nil {
		log.Printf("Error loading environment variables: %v", err)
	}

	return runtimeconfig.StringValue("STRAVA_CACHE_PATH", "strava-cache")
}
