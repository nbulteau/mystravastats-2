package strava

type Bike struct {
	Distance          int     `json:"distance"`           // Distance covered by the bike, in meters
	Id                string  `json:"id"`                 // Unique identifier of the bike
	Name              string  `json:"name"`               // Name of the bike
	Nickname          *string `json:"nickname,omitempty"` // Optional nickname for the bike
	Retired           *bool   `json:"retired,omitempty"`  // Indicates if the bike is retired (optional)
	ConvertedDistance float64 `json:"converted_distance"` // Distance converted to another unit (e.g., kilometers)
	Primary           bool    `json:"primary"`            // True if this is the primary bike
	ResourceState     int     `json:"resource_state"`     // Level of detail about the bike resource
}
