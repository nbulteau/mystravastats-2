package helpers

import (
	"os"
)

// GetStravaCachePath returns the path to the cache directory for Strava API data
func GetStravaCachePath() string {
	return os.Getenv("STRAVA_CACHE_PATH")
}
