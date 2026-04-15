package domain

type ActivityHeatmapActivity struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	DistanceKm     float64 `json:"distanceKm"`
	ElevationGainM float64 `json:"elevationGainM"`
	DurationSec    int     `json:"durationSec"`
}

type ActivityHeatmapDay struct {
	DistanceKm     float64                   `json:"distanceKm"`
	ElevationGainM float64                   `json:"elevationGainM"`
	DurationSec    int                       `json:"durationSec"`
	ActivityCount  int                       `json:"activityCount"`
	Activities     []ActivityHeatmapActivity `json:"activities"`
}
