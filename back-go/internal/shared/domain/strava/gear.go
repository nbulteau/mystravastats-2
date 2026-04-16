package strava

type Gear struct {
	Distance          int64   `json:"distance"`
	Id                string  `json:"id"`
	ConvertedDistance float64 `json:"converted_distance"`
	Name              string  `json:"name"`
	Nickname          string  `json:"nickname"`
	Primary           bool    `json:"primary"`
	Retired           bool    `json:"retired"`
}
