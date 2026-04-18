package domain

import "mystravastats/internal/shared/domain/business"

type RouteVariantType string

const (
	RouteVariantClosest   RouteVariantType = "CLOSE_MATCH"
	RouteVariantShorter   RouteVariantType = "SHORTER"
	RouteVariantLonger    RouteVariantType = "LONGER"
	RouteVariantHillier   RouteVariantType = "HILLIER"
	RouteVariantSeasonal  RouteVariantType = "SEASONAL"
	RouteVariantRoadGraph RouteVariantType = "ROAD_GRAPH"
	RouteVariantShape     RouteVariantType = "SHAPE_MATCH"
	RouteVariantShapeMix  RouteVariantType = "SHAPE_REMIX"
)

type Coordinates struct {
	Lat float64
	Lng float64
}

type RouteRecommendation struct {
	RouteID        string
	Activity       business.ActivityShort
	ActivityDate   string
	DistanceKm     float64
	ElevationGainM float64
	DurationSec    int
	IsLoop         bool
	Start          *Coordinates
	End            *Coordinates
	StartArea      string
	Season         string
	VariantType    RouteVariantType
	MatchScore     float64
	Reasons        []string
	PreviewLatLng  [][]float64
	Shape          *string
	ShapeScore     *float64
	Experimental   bool
}

type ShapeRemixRecommendation struct {
	ID             string
	Shape          string
	DistanceKm     float64
	ElevationGainM float64
	DurationSec    int
	MatchScore     float64
	Reasons        []string
	Components     []business.ActivityShort
	PreviewLatLng  [][]float64
	Experimental   bool
}

type RouteExplorerRequest struct {
	DistanceTargetKm  *float64
	ElevationTargetM  *float64
	DurationTargetMin *int
	StartPoint        *Coordinates
	StartDirection    *string
	TargetMode        *string
	CustomWaypoints   []Coordinates
	RouteType         *string
	Season            *string
	Limit             int
	Shape             *string
	ShapePolyline     *string
	IncludeRemix      bool
}

type RouteExplorerResult struct {
	ClosestLoops   []RouteRecommendation
	Variants       []RouteRecommendation
	Seasonal       []RouteRecommendation
	RoadGraphLoops []RouteRecommendation
	ShapeMatches   []RouteRecommendation
	ShapeRemixes   []ShapeRemixRecommendation
}
