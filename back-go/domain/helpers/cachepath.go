package helpers

import (
	"log"
	"os"

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

	cachePath := os.Getenv("STRAVA_CACHE_PATH")
	if cachePath == "" {
		cachePath = "strava-cache" // default value if environment variable is not set
	}

	return cachePath
}
