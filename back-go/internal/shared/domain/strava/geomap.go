package strava

type GeoMap struct {
	Id              string  `json:"id"`
	Polyline        *string `json:"polyline,omitempty"`
	ResourceState   int     `json:"resource_state"`
	SummaryPolyline *string `json:"summary_polyline,omitempty"`
}
