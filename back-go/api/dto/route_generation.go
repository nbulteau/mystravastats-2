package dto

type RouteGenerationScoreDto struct {
	Global      float64 `json:"global"`
	Distance    float64 `json:"distance"`
	Elevation   float64 `json:"elevation"`
	Duration    float64 `json:"duration"`
	Direction   float64 `json:"direction"`
	Shape       float64 `json:"shape"`
	RoadFitness float64 `json:"roadFitness"`
}

type GeneratedRouteDto struct {
	RouteID              string                  `json:"routeId"`
	Title                string                  `json:"title"`
	VariantType          string                  `json:"variantType"`
	RouteType            string                  `json:"routeType,omitempty"`
	StartDirection       string                  `json:"startDirection,omitempty"`
	DistanceKm           float64                 `json:"distanceKm"`
	ElevationGainM       float64                 `json:"elevationGainM"`
	DurationSec          int                     `json:"durationSec"`
	EstimatedDurationSec int                     `json:"estimatedDurationSec"`
	Score                RouteGenerationScoreDto `json:"score"`
	Reasons              []string                `json:"reasons"`
	PreviewLatLng        [][]float64             `json:"previewLatLng"`
	Start                *RouteCoordinateDto     `json:"start,omitempty"`
	End                  *RouteCoordinateDto     `json:"end,omitempty"`
	ActivityID           *int64                  `json:"activityId,omitempty"`
	IsRoadGraphGenerated bool                    `json:"isRoadGraphGenerated"`
}

type RouteGenerationDiagnosticDto struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type GenerateRoutesResponseDto struct {
	Routes      []GeneratedRouteDto            `json:"routes"`
	Diagnostics []RouteGenerationDiagnosticDto `json:"diagnostics,omitempty"`
}
