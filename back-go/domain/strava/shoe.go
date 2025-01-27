package strava

type Shoe struct {
	Distance          int     `json:"distance"`
	Id                string  `json:"id"`
	Name              string  `json:"name"`
	Nickname          *string `json:"nickname,omitempty"`
	Retired           *bool   `json:"retired,omitempty"`
	ConvertedDistance float64 `json:"converted_distance"`
	Primary           bool    `json:"primary"`
	ResourceState     int     `json:"resource_state"`
}
