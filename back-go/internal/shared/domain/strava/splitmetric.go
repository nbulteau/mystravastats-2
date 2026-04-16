package strava

type SplitsMetric struct {
	AverageSpeed              float64  `json:"average_speed"`
	AverageGradeAdjustedSpeed *float64 `json:"average_grade_adjusted_speed,omitempty"`
	AverageHeartRate          float64  `json:"average_heartrate"`
	Distance                  float64  `json:"distance"`
	ElapsedTime               int      `json:"elapsed_time"`
	ElevationDifference       float64  `json:"elevation_difference"`
	MovingTime                int      `json:"moving_time"`
	PaceZone                  int      `json:"pace_zone"`
	Split                     int      `json:"split"`
}
