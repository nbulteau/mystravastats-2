package infrastructure

import (
	routesDomain "mystravastats/internal/routes/domain"
	"net/http"
	"time"
)

const (
	defaultOSMRoutingBaseURL    = "http://localhost:5000"
	defaultOSMRoutingV3Enabled  = true
	maxOSRMRoutingCalls         = 24
	startSnapToleranceMeters    = 900.0
	fallbackStartSnapTolerance  = 4000.0
	directionToleranceMeters    = 120.0
	backtrackingStartZoneM      = 2000.0
	minAxisSegmentLengthM       = 25.0
	minOppositeReuseMeters      = 120.0
	historyReuseBonusWeight     = 18.0
	historyStartZoneBonusWeight = 14.0
	historyAxisBiasWeight       = 0.75
	historyZoneBiasWeight       = 0.25
	shapeModeStrategyShapeFirst = "shape-first"
	shapeModeStrategyMapMatch   = "shape-map-match"
	shapeModeStrategyRoadSnap   = "shape-road-snap"
	shapeModeStrategyStitched   = "shape-stitched"
	shapeModeStrategySimplified = "shape-simplified"
	shapeModeStrategyRoadFirst  = "road-first"
	shapeModeStrategyBestEffort = "shape-best-effort"
	maxShapeTraceVariants       = 8
	maxShapeBestEffortVariants  = 8
	shapeCoverageSamplePoints   = 6
	shapeCoverageMaxSnapMeters  = 5000.0
	editControlSnapMaxMeters    = 900.0
	editMinControlSpacingMeters = 15.0
	defaultOSRMProfileFilePath  = "./osm/region.osrm.profile"
	fallbackOSRMProfilePath     = "../osm/region.osrm.profile"
)

type osrmRouteCandidate struct {
	recommendation      routesDomain.RouteRecommendation
	directionPenalty    float64
	backtrackingRatio   float64
	corridorOverlap     float64
	edgeReuseRatio      float64
	maxAxisReuseCount   int
	maxAxisReuseRatio   float64
	segmentDiversity    float64
	distanceDeltaRatio  float64
	pathRatio           float64
	historyReuseScore   float64
	effectiveMatchScore float64
}

type shapeRoutingStrategy struct {
	code       string
	label      string
	waypoints  []routesDomain.Coordinates
	bestEffort bool
}

type shapeRoutingVariant struct {
	label string
	shape []routesDomain.Coordinates
}

type routingDiagnosticError struct {
	diagnostic routesDomain.RouteGenerationDiagnostic
}

func (err *routingDiagnosticError) Error() string {
	return err.diagnostic.Message
}

func (err *routingDiagnosticError) Diagnostic() routesDomain.RouteGenerationDiagnostic {
	return err.diagnostic
}

type routingHistoryBiasContext struct {
	enabled             bool
	normalizedRouteType string
	axisScores          map[string]float64
	zoneScores          map[string]float64
	maxAxisScore        float64
	maxZoneScore        float64
}

type routeRelaxationLevel struct {
	name                  string
	maxDirectionPenalty   float64
	maxBacktrackingRatio  float64
	maxCorridorOverlap    float64
	maxEdgeReuseRatio     float64
	maxAxisReuseCount     int
	minSegmentDiversity   float64
	maxDistanceDeltaRatio float64
}

type routeSurfaceBreakdown struct {
	pavedM   float64
	gravelM  float64
	trailM   float64
	unknownM float64
}

type osrmRouteResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Routes  []osrmRoute `json:"routes"`
}

type osrmMatchResponse struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Matchings []osrmRoute `json:"matchings"`
}

type osrmNearestResponse struct {
	Code      string             `json:"code"`
	Message   string             `json:"message"`
	Waypoints []osrmNearestPoint `json:"waypoints"`
}

type osrmNearestPoint struct {
	Distance float64   `json:"distance"`
	Location []float64 `json:"location"`
}

type osrmRoute struct {
	Distance float64      `json:"distance"`
	Duration float64      `json:"duration"`
	Geometry osrmGeometry `json:"geometry"`
	Legs     []osrmLeg    `json:"legs"`
}

type osrmGeometry struct {
	Type        string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"`
}

type osrmLeg struct {
	Steps []osrmStep `json:"steps"`
}

type osrmStep struct {
	Distance  float64  `json:"distance"`
	Mode      string   `json:"mode"`
	Classes   []string `json:"classes"`
	Surface   string   `json:"surface"`
	TrackType string   `json:"tracktype"`
}

// OSMRoutingAdapter integrates a local OSRM endpoint as a routing engine.
type OSMRoutingAdapter struct {
	enabled               bool
	v3Enabled             bool
	debug                 bool
	baseURL               string
	timeout               time.Duration
	osrmClient            *osrmClient
	client                *http.Client
	profileOverride       string
	extractProfileEnv     string
	extractProfileCfgFile string
}
