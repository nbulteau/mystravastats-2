package strava

type Segment struct {
	ActivityType  string    `json:"activity_type"`
	AverageGrade  float64   `json:"average_grade"`
	City          *string   `json:"city,omitempty"`
	ClimbCategory int       `json:"climb_category"`
	Country       *string   `json:"country,omitempty"`
	Distance      float64   `json:"distance"`
	ElevationHigh float64   `json:"elevation_high"`
	ElevationLow  float64   `json:"elevation_low"`
	EndLatLng     []float64 `json:"end_latlng"`
	Hazardous     bool      `json:"hazardous"`
	Id            int64     `json:"id"`
	MaximumGrade  float64   `json:"maximum_grade"`
	Name          string    `json:"name"`
	IsPrivate     bool      `json:"private"`
	ResourceState int       `json:"resource_state"`
	Starred       bool      `json:"starred"`
	StartLatLng   []float64 `json:"start_latlng"`
	State         *string   `json:"state,omitempty"`
}
