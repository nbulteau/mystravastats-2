package application

type MapTrack struct {
	ActivityID     int64       `json:"activityId"`
	ActivityName   string      `json:"activityName"`
	ActivityDate   string      `json:"activityDate"`
	ActivityType   string      `json:"activityType"`
	DistanceKm     float64     `json:"distanceKm"`
	ElevationGainM float64     `json:"elevationGainM"`
	Coordinates    [][]float64 `json:"coordinates"`
}
