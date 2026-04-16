package domain

import "mystravastats/domain/business"

type RouteVariantType string

const (
	RouteVariantClosest  RouteVariantType = "CLOSE_MATCH"
	RouteVariantShorter  RouteVariantType = "SHORTER"
	RouteVariantLonger   RouteVariantType = "LONGER"
	RouteVariantHillier  RouteVariantType = "HILLIER"
	RouteVariantSeasonal RouteVariantType = "SEASONAL"
	RouteVariantShape    RouteVariantType = "SHAPE_MATCH"
	RouteVariantShapeMix RouteVariantType = "SHAPE_REMIX"
)

type Coordinates struct {
	Lat float64
	Lng float64
}

type RouteRecommendation struct {
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
	StartDirection    *string
	RouteType         *string
	Season            *string
	Limit             int
	Shape             *string
	IncludeRemix      bool
}

type RouteExplorerResult struct {
	ClosestLoops []RouteRecommendation
	Variants     []RouteRecommendation
	Seasonal     []RouteRecommendation
	ShapeMatches []RouteRecommendation
	ShapeRemixes []ShapeRemixRecommendation
}
