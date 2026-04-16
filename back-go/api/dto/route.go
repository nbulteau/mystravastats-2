package dto

type RouteCoordinateDto struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type RouteRecommendationDto struct {
	RouteID        string              `json:"routeId"`
	Activity       ActivityShortDto    `json:"activity"`
	ActivityDate   string              `json:"activityDate"`
	DistanceKm     float64             `json:"distanceKm"`
	ElevationGainM float64             `json:"elevationGainM"`
	DurationSec    int                 `json:"durationSec"`
	IsLoop         bool                `json:"isLoop"`
	Start          *RouteCoordinateDto `json:"start,omitempty"`
	End            *RouteCoordinateDto `json:"end,omitempty"`
	StartArea      string              `json:"startArea"`
	Season         string              `json:"season"`
	VariantType    string              `json:"variantType"`
	MatchScore     float64             `json:"matchScore"`
	Reasons        []string            `json:"reasons"`
	PreviewLatLng  [][]float64         `json:"previewLatLng"`
	Shape          *string             `json:"shape,omitempty"`
	ShapeScore     *float64            `json:"shapeScore,omitempty"`
	Experimental   bool                `json:"experimental"`
}

type ShapeRemixRecommendationDto struct {
	ID             string             `json:"id"`
	Shape          string             `json:"shape"`
	DistanceKm     float64            `json:"distanceKm"`
	ElevationGainM float64            `json:"elevationGainM"`
	DurationSec    int                `json:"durationSec"`
	MatchScore     float64            `json:"matchScore"`
	Reasons        []string           `json:"reasons"`
	Components     []ActivityShortDto `json:"components"`
	PreviewLatLng  [][]float64        `json:"previewLatLng"`
	Experimental   bool               `json:"experimental"`
}

type RouteExplorerResultDto struct {
	ClosestLoops   []RouteRecommendationDto      `json:"closestLoops"`
	Variants       []RouteRecommendationDto      `json:"variants"`
	Seasonal       []RouteRecommendationDto      `json:"seasonal"`
	RoadGraphLoops []RouteRecommendationDto      `json:"roadGraphLoops"`
	ShapeMatches   []RouteRecommendationDto      `json:"shapeMatches"`
	ShapeRemixes   []ShapeRemixRecommendationDto `json:"shapeRemixes"`
}
