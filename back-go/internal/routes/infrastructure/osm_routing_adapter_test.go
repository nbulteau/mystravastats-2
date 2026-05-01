package infrastructure

import (
	"math"
	"mystravastats/internal/routes/application"
	routesDomain "mystravastats/internal/routes/domain"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestOSMRoutingAdapter_WhenDisabled_ReturnsDisabledHealthAndNoGeneratedLoops(t *testing.T) {
	// GIVEN
	t.Setenv("OSM_ROUTING_ENABLED", "false")
	adapter := NewOSMRoutingAdapter()
	request := application.RoutingEngineRequest{
		StartPoint:       routesDomain.Coordinates{Lat: 48.13, Lng: -1.63},
		DistanceTargetKm: 40.0,
		StartDirection:   "N",
		RouteType:        "RIDE",
		Limit:            4,
	}

	// WHEN
	health := adapter.HealthDetails()
	routes, err := adapter.GenerateTargetLoops(request)

	// THEN
	if err != nil {
		t.Fatalf("expected no error when adapter is disabled, got %v", err)
	}
	if len(routes) != 0 {
		t.Fatalf("expected no generated routes when adapter is disabled, got %d", len(routes))
	}
	if got := health["status"]; got != "disabled" {
		t.Fatalf("expected health status=disabled, got %v", got)
	}
	if got := health["enabled"]; got != false {
		t.Fatalf("expected enabled=false, got %v", got)
	}
}

func TestOSMRoutingAdapter_HealthDetails_ExposeSupportedRouteTypesFromExtractProfile(t *testing.T) {
	// GIVEN
	t.Setenv("OSM_ROUTING_ENABLED", "false")
	t.Setenv("OSM_ROUTING_EXTRACT_PROFILE", "/opt/foot.lua")
	adapter := NewOSMRoutingAdapter()

	// WHEN
	health := adapter.HealthDetails()

	// THEN
	if got := health["extractProfile"]; got != "/opt/foot.lua" {
		t.Fatalf("expected extractProfile=/opt/foot.lua, got %v", got)
	}
	routeTypes, ok := health["supportedRouteTypes"].([]string)
	if !ok {
		t.Fatalf("expected supportedRouteTypes to be []string, got %T", health["supportedRouteTypes"])
	}
	if len(routeTypes) != 3 || routeTypes[0] != "RUN" || routeTypes[1] != "TRAIL" || routeTypes[2] != "HIKE" {
		t.Fatalf("expected [RUN TRAIL HIKE], got %v", routeTypes)
	}
}

func TestOSMRoutingAdapter_HealthDetails_UsesOverridePathForEffectiveProfile(t *testing.T) {
	// GIVEN
	t.Setenv("OSM_ROUTING_ENABLED", "false")
	t.Setenv("OSM_ROUTING_PROFILE", "/opt/bicycle.lua")
	adapter := NewOSMRoutingAdapter()

	// WHEN
	health := adapter.HealthDetails()

	// THEN
	if got := health["effectiveProfile"]; got != "cycling" {
		t.Fatalf("expected effectiveProfile=cycling, got %v", got)
	}
	routeTypes, ok := health["supportedRouteTypes"].([]string)
	if !ok {
		t.Fatalf("expected supportedRouteTypes to be []string, got %T", health["supportedRouteTypes"])
	}
	if len(routeTypes) != 3 || routeTypes[0] != "RIDE" || routeTypes[1] != "MTB" || routeTypes[2] != "GRAVEL" {
		t.Fatalf("expected [RIDE MTB GRAVEL], got %v", routeTypes)
	}
}

func TestEvaluateAxisReuseOutsideStartZone_DetectsOppositeTraversalAwayFromStart(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000}
	points := [][]float64{
		{48.13000, -1.63000}, // start
		{48.15000, -1.63000}, // far north
		{48.15000, -1.62000}, // far east
		{48.15000, -1.63000}, // back on same far axis (reverse traversal)
		{48.13000, -1.63000}, // return start
	}

	// WHEN
	hasOpposite, maxReuse, oppositeRatio := evaluateAxisReuseOutsideStartZone(
		points,
		start,
		backtrackingStartZoneM,
		minOppositeReuseMeters,
	)

	// THEN
	if !hasOpposite {
		t.Fatalf("expected opposite traversal outside start zone to be detected")
	}
	if maxReuse < 2 {
		t.Fatalf("expected max axis reuse outside start zone >= 2, got %d", maxReuse)
	}
	if oppositeRatio <= 0 {
		t.Fatalf("expected opposite ratio > 0, got %.3f", oppositeRatio)
	}
}

func TestEvaluateAxisReuseOutsideStartZone_DetectsSameDirectionReuseAwayFromStart(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000}
	points := [][]float64{
		{48.13000, -1.63000}, // start
		{48.15600, -1.63000}, // far north
		{48.15600, -1.61800}, // far east
		{48.16000, -1.61200}, // farther east
		{48.16400, -1.62000}, // turn south-west
		{48.15600, -1.61800}, // back near prior axis
		{48.16000, -1.61200}, // same axis as above, same direction
		{48.13000, -1.63000}, // return start
	}

	// WHEN
	hasOpposite, maxReuse, oppositeRatio := evaluateAxisReuseOutsideStartZone(
		points,
		start,
		backtrackingStartZoneM,
		minOppositeReuseMeters,
	)

	// THEN
	if hasOpposite {
		t.Fatalf("expected no opposite traversal for same-direction reuse case")
	}
	if maxReuse < 2 {
		t.Fatalf("expected max axis reuse outside start zone >= 2, got %d", maxReuse)
	}
	if oppositeRatio != 0.0 {
		t.Fatalf("expected opposite ratio to stay 0, got %.3f", oppositeRatio)
	}
	if limit := outsideStartAxisReuseLimit("RIDE", false); maxReuse <= limit {
		t.Fatalf("expected reuse %d to exceed hard outside-start limit %d", maxReuse, limit)
	}
}

func TestEvaluateAxisReuseOutsideStartZone_LongSegmentCrossingHubBoundaryIsCounted(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000}
	points := [][]float64{
		{48.13000, -1.63000}, // start
		{48.17000, -1.63000}, // far north (~4.4km)
		{48.13000, -1.63000}, // retrace same axis back to start
	}

	// WHEN
	hasOpposite, maxReuse, oppositeRatio := evaluateAxisReuseOutsideStartZone(
		points,
		start,
		backtrackingStartZoneM,
		minOppositeReuseMeters,
	)

	// THEN
	if !hasOpposite {
		t.Fatalf("expected opposite traversal to be detected for long segment crossing start-zone boundary")
	}
	if maxReuse < 2 {
		t.Fatalf("expected max axis reuse outside start zone >= 2, got %d", maxReuse)
	}
	if oppositeRatio <= 0 {
		t.Fatalf("expected opposite ratio > 0, got %.3f", oppositeRatio)
	}
}

func TestEvaluateAxisReuseOutsideStartZone_KeepsLocalHubReuseAllowed(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000}
	points := [][]float64{
		{48.13000, -1.63000}, // start
		{48.13600, -1.63000}, // ~660m north (inside 2km hub)
		{48.13000, -1.63000}, // back
		{48.13600, -1.63000}, // same local axis again
		{48.13000, -1.63000}, // back
	}

	// WHEN
	hasOpposite, maxReuse, oppositeRatio := evaluateAxisReuseOutsideStartZone(
		points,
		start,
		backtrackingStartZoneM,
		minOppositeReuseMeters,
	)

	// THEN
	if hasOpposite {
		t.Fatalf("expected no opposite traversal outside hub for local reuse")
	}
	if maxReuse != 0 {
		t.Fatalf("expected no counted outside-start reuse for local hub traversal, got %d", maxReuse)
	}
	if oppositeRatio != 0.0 {
		t.Fatalf("expected opposite ratio 0 for local hub traversal, got %.3f", oppositeRatio)
	}
}

func TestOutsideStartAxisReusePolicy_IsAlwaysStrict(t *testing.T) {
	if got := outsideStartAxisReuseLimit("RIDE", false); got != 1 {
		t.Fatalf("expected RIDE limit to stay hard at 1, got %d", got)
	}
	if got := outsideStartAxisReuseLimit("MTB", false); got != 1 {
		t.Fatalf("expected MTB limit to stay hard at 1, got %d", got)
	}
	if got := outsideStartAxisReuseLimit("GRAVEL", true); got != 1 {
		t.Fatalf("expected strict GRAVEL limit to stay hard at 1, got %d", got)
	}
	if got := allowedOppositeOutsideStartRatio("RIDE", false); got != 0.0 {
		t.Fatalf("expected opposite overlap ratio to be forbidden, got %.3f", got)
	}
}

func TestComputeSurfaceBreakdown_ClassifiesPavedGravelTrailAndUnknown(t *testing.T) {
	// GIVEN
	route := osrmRoute{
		Distance: 2000.0,
		Legs: []osrmLeg{
			{
				Steps: []osrmStep{
					{Distance: 1000.0, Mode: "cycling"},
					{Distance: 500.0, Mode: "cycling", Classes: []string{"unpaved"}},
					{Distance: 300.0, Mode: "pushing bike"},
					{Distance: 200.0, Mode: "cycling", Classes: []string{"ferry"}},
				},
			},
		},
	}

	// WHEN
	breakdown := computeSurfaceBreakdown(route)
	pavedRatio, gravelRatio, trailRatio, unknownRatio := breakdown.normalizedRatios()

	// THEN
	if pavedRatio < 0.49 || pavedRatio > 0.51 {
		t.Fatalf("expected paved ratio around 0.50, got %.3f", pavedRatio)
	}
	if gravelRatio < 0.24 || gravelRatio > 0.26 {
		t.Fatalf("expected gravel ratio around 0.25, got %.3f", gravelRatio)
	}
	if trailRatio < 0.14 || trailRatio > 0.16 {
		t.Fatalf("expected trail ratio around 0.15, got %.3f", trailRatio)
	}
	if unknownRatio < 0.09 || unknownRatio > 0.11 {
		t.Fatalf("expected unknown ratio around 0.10, got %.3f", unknownRatio)
	}
}

func TestSurfaceMatchScore_AdaptsToRequestedRouteType(t *testing.T) {
	// GIVEN
	mixedBreakdown := routeSurfaceBreakdown{
		pavedM:  3500.0,
		gravelM: 5500.0,
		trailM:  1000.0,
	}
	trailBreakdown := routeSurfaceBreakdown{
		pavedM:  800.0,
		gravelM: 2900.0,
		trailM:  6300.0,
	}

	// WHEN
	gravelScore := surfaceMatchScore("GRAVEL", mixedBreakdown)
	rideScoreOnMixed := surfaceMatchScore("RIDE", mixedBreakdown)
	mtbScoreOnTrail := surfaceMatchScore("MTB", trailBreakdown)
	rideScoreOnTrail := surfaceMatchScore("RIDE", trailBreakdown)

	// THEN
	if gravelScore <= rideScoreOnMixed {
		t.Fatalf("expected gravel score to be higher than ride on mixed gravel profile, gravel=%.1f ride=%.1f", gravelScore, rideScoreOnMixed)
	}
	if mtbScoreOnTrail <= rideScoreOnTrail {
		t.Fatalf("expected mtb score to be higher than ride on trail-heavy profile, mtb=%.1f ride=%.1f", mtbScoreOnTrail, rideScoreOnTrail)
	}
}

func TestParseShapePolylineCoordinates_DecodesEncodedPolyline(t *testing.T) {
	// GIVEN
	encoded := "_p~iF~ps|U_ulLnnqC_mqNvxq`@"

	// WHEN
	points := parseShapePolylineCoordinates(encoded)

	// THEN
	if len(points) != 3 {
		t.Fatalf("expected 3 decoded points, got %d", len(points))
	}
	if points[0].Lat < 38.49 || points[0].Lat > 38.51 {
		t.Fatalf("unexpected first decoded latitude %.5f", points[0].Lat)
	}
}

func TestParseShapePolylineCoordinates_ExtractsTrackPointsFromGPX(t *testing.T) {
	// GIVEN
	gpx := `
<gpx version="1.1" creator="test">
  <trk><trkseg>
    <trkpt lat="48.1000" lon="-1.6000"></trkpt>
    <trkpt lat="48.1200" lon="-1.6200"></trkpt>
    <trkpt lat="48.1300" lon="-1.6300"></trkpt>
  </trkseg></trk>
</gpx>`

	// WHEN
	points := parseShapePolylineCoordinates(gpx)

	// THEN
	if len(points) != 3 {
		t.Fatalf("expected 3 GPX points, got %d", len(points))
	}
	if points[2].Lat < 48.129 || points[2].Lat > 48.131 {
		t.Fatalf("unexpected GPX last latitude %.5f", points[2].Lat)
	}
}

func TestBuildShapeRoadFirstWaypoints_ReturnsAnchoredLoopWithFarAnchors(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000}
	shape := []routesDomain.Coordinates{
		{Lat: 48.13000, Lng: -1.63000},
		{Lat: 48.14200, Lng: -1.62000},
		{Lat: 48.14800, Lng: -1.60000},
		{Lat: 48.13700, Lng: -1.59000},
		{Lat: 48.13000, Lng: -1.63000},
	}

	// WHEN
	roadFirstWaypoints := buildShapeRoadFirstWaypoints(start, shape)
	shapeFirstWaypoints := buildShapeLoopWaypoints(start, shape)

	// THEN
	if len(roadFirstWaypoints) < 3 {
		t.Fatalf("expected at least 3 road-first waypoints, got %d", len(roadFirstWaypoints))
	}
	first := roadFirstWaypoints[0]
	last := roadFirstWaypoints[len(roadFirstWaypoints)-1]
	if first.Lat != start.Lat || first.Lng != start.Lng || last.Lat != start.Lat || last.Lng != start.Lng {
		t.Fatalf("expected loop anchored to start, first=%+v last=%+v start=%+v", first, last, start)
	}
	if len(roadFirstWaypoints) > len(shapeFirstWaypoints)+1 {
		t.Fatalf("expected road-first waypoints to stay compact, road-first=%d shape-first=%d", len(roadFirstWaypoints), len(shapeFirstWaypoints))
	}
}

func TestBuildShapeSimplifiedWaypoints_KeepsSimpleShapeAnchors(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.1300, Lng: -1.6300}
	circle := prepareShapeForRouting(
		coordinatesFromLatLng(testCircleLatLng(start.Lat, start.Lng, 1000.0, 72)),
		start,
	)
	square := []routesDomain.Coordinates{
		destinationFromBearing(start, 1.0, 315.0),
		destinationFromBearing(start, 1.0, 45.0),
		destinationFromBearing(start, 1.0, 135.0),
		destinationFromBearing(start, 1.0, 225.0),
		destinationFromBearing(start, 1.0, 315.0),
	}
	star := coordinatesFromLatLng(testStarLatLng(start.Lat, start.Lng, 1000.0, 420.0))
	star = prepareShapeForRouting(star, start)

	// WHEN
	circleWaypoints := buildShapeSimplifiedWaypoints(circle[0], circle)
	circleShapeFirstWaypoints := buildShapeLoopWaypoints(circle[0], circle)
	squareWaypoints := buildShapeSimplifiedWaypoints(square[0], square)
	starWaypoints := buildShapeSimplifiedWaypoints(star[0], star)

	// THEN
	if len(circleWaypoints) < 7 {
		t.Fatalf("expected circle to keep enough anchors, got %d", len(circleWaypoints))
	}
	if len(circleWaypoints) >= len(circleShapeFirstWaypoints) {
		t.Fatalf("expected circle anchors to be simpler than shape-first, simplified=%d shape-first=%d", len(circleWaypoints), len(circleShapeFirstWaypoints))
	}
	if len(squareWaypoints) != len(square) {
		t.Fatalf("expected square corners to be preserved, got %d vs %d", len(squareWaypoints), len(square))
	}
	if len(starWaypoints) < 10 {
		t.Fatalf("expected star points to be preserved, got %d", len(starWaypoints))
	}
	for label, waypoints := range map[string][]routesDomain.Coordinates{
		"circle": circleWaypoints,
		"square": squareWaypoints,
		"star":   starWaypoints,
	} {
		first := waypoints[0]
		last := waypoints[len(waypoints)-1]
		if haversineDistanceMeters(first.Lat, first.Lng, last.Lat, last.Lng) > 120.0 {
			t.Fatalf("expected %s waypoints to close the loop, first=%+v last=%+v", label, first, last)
		}
	}
}

func TestBuildShapeStitchedWaypoints_ReturnsCompactAnchoredLoop(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.1300, Lng: -1.6300}
	circle := prepareShapeForRouting(
		coordinatesFromLatLng(testCircleLatLng(start.Lat, start.Lng, 1000.0, 72)),
		start,
	)
	shapeFirstWaypoints := buildShapeLoopWaypoints(circle[0], circle)

	// WHEN
	stitchedWaypoints := buildShapeStitchedWaypoints(circle[0], circle)

	// THEN
	if len(stitchedWaypoints) < 8 {
		t.Fatalf("expected stitched waypoints to keep enough contour anchors, got %d", len(stitchedWaypoints))
	}
	if len(stitchedWaypoints) >= len(shapeFirstWaypoints) {
		t.Fatalf("expected stitched waypoints to stay more compact than dense shape-first, stitched=%d shape-first=%d", len(stitchedWaypoints), len(shapeFirstWaypoints))
	}
	first := stitchedWaypoints[0]
	last := stitchedWaypoints[len(stitchedWaypoints)-1]
	if haversineDistanceMeters(first.Lat, first.Lng, last.Lat, last.Lng) > 120.0 {
		t.Fatalf("expected stitched waypoints to close the loop, first=%+v last=%+v", first, last)
	}
}

func TestStitchOSRMRoutes_MergesSegmentsAndDropsDuplicateJoin(t *testing.T) {
	// GIVEN
	segments := []osrmRoute{
		{
			Distance: 100.0,
			Duration: 20.0,
			Geometry: osrmGeometry{Type: "LineString", Coordinates: [][]float64{
				{-1.6300, 48.1300},
				{-1.6200, 48.1300},
			}},
		},
		{
			Distance: 150.0,
			Duration: 30.0,
			Geometry: osrmGeometry{Type: "LineString", Coordinates: [][]float64{
				{-1.6200, 48.1300},
				{-1.6100, 48.1400},
			}},
		},
	}

	// WHEN
	stitched, ok := stitchOSRMRoutes(segments)

	// THEN
	if !ok {
		t.Fatalf("expected stitched route to be valid")
	}
	if stitched.Distance != 250.0 || stitched.Duration != 50.0 {
		t.Fatalf("expected summed distance/duration, got distance=%.1f duration=%.1f", stitched.Distance, stitched.Duration)
	}
	if len(stitched.Geometry.Coordinates) != 3 {
		t.Fatalf("expected duplicate join coordinate to be dropped, got %d coordinates", len(stitched.Geometry.Coordinates))
	}
}

func TestFetchOSRMNearestRoadTraceRoute_RoutesBetweenSnappedAnchors(t *testing.T) {
	// GIVEN
	routeCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch {
		case strings.HasPrefix(request.URL.Path, "/nearest/v1/cycling/"):
			rawCoordinate := strings.TrimPrefix(request.URL.Path, "/nearest/v1/cycling/")
			lng, lat := parseOSRMTestCoordinate(t, rawCoordinate)
			writeOSRMTestJSON(response, http.StatusOK, `{"code":"Ok","waypoints":[{"location":[`+
				formatOSRMTestFloat(lng)+`,`+formatOSRMTestFloat(lat)+`],"distance":5.0}]}`)
		case strings.HasPrefix(request.URL.Path, "/route/v1/cycling/"):
			routeCalls++
			rawCoordinates := strings.TrimPrefix(request.URL.Path, "/route/v1/cycling/")
			parts := strings.Split(rawCoordinates, ";")
			if len(parts) != 2 {
				t.Fatalf("expected two route coordinates, got %q", rawCoordinates)
			}
			startLng, startLat := parseOSRMTestCoordinate(t, parts[0])
			endLng, endLat := parseOSRMTestCoordinate(t, parts[1])
			midLng := (startLng+endLng)/2.0 + 0.0002
			midLat := (startLat+endLat)/2.0 + 0.0002
			writeOSRMTestJSON(response, http.StatusOK, `{"code":"Ok","routes":[{"distance":100.0,"duration":20.0,"geometry":{"type":"LineString","coordinates":[[`+
				formatOSRMTestFloat(startLng)+`,`+formatOSRMTestFloat(startLat)+`],[`+
				formatOSRMTestFloat(midLng)+`,`+formatOSRMTestFloat(midLat)+`],[`+
				formatOSRMTestFloat(endLng)+`,`+formatOSRMTestFloat(endLat)+`]]},"legs":[{"steps":[{"distance":100.0,"mode":"cycling"}]}]}]}`)
		default:
			writeOSRMTestJSON(response, http.StatusNotFound, `{"code":"NotFound"}`)
		}
	}))
	defer server.Close()

	adapter := &OSMRoutingAdapter{
		baseURL: server.URL,
		client:  server.Client(),
	}
	shape := []routesDomain.Coordinates{
		{Lat: 48.1300, Lng: -1.6300},
		{Lat: 48.1310, Lng: -1.6200},
		{Lat: 48.1300, Lng: -1.6100},
	}

	// WHEN
	route, ok := adapter.fetchOSRMNearestRoadTraceRoute("cycling", shape)

	// THEN
	if !ok {
		t.Fatalf("expected nearest-road trace route to be valid")
	}
	if routeCalls != 3 {
		t.Fatalf("expected one OSRM route call per snapped segment, got %d", routeCalls)
	}
	if route.Distance <= 0 || route.Duration <= 0 {
		t.Fatalf("expected positive distance and duration, got distance=%.1f duration=%.1f", route.Distance, route.Duration)
	}
	if len(route.Geometry.Coordinates) <= len(shape)+1 {
		t.Fatalf("expected routed geometry points beyond snapped anchors, got %d", len(route.Geometry.Coordinates))
	}
	if route.Geometry.Coordinates[0][0] != shape[0].Lng || route.Geometry.Coordinates[0][1] != shape[0].Lat {
		t.Fatalf("expected OSRM lon/lat coordinates, got %+v", route.Geometry.Coordinates[0])
	}
}

func writeOSRMTestJSON(response http.ResponseWriter, status int, body string) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_, _ = response.Write([]byte(body))
}

func parseOSRMTestCoordinate(t *testing.T, raw string) (float64, float64) {
	t.Helper()
	parts := strings.Split(raw, ",")
	if len(parts) != 2 {
		t.Fatalf("expected lon,lat coordinate, got %q", raw)
	}
	lng, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		t.Fatalf("failed to parse lng from %q: %v", raw, err)
	}
	lat, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		t.Fatalf("failed to parse lat from %q: %v", raw, err)
	}
	return lng, lat
}

func formatOSRMTestFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 6, 64)
}

func TestProjectShapePolylineToStart_PreservesMapPlacedShapeAroundStart(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.1300, Lng: -1.6300}
	shape := coordinatesFromLatLng(testCircleLatLng(start.Lat, start.Lng, 1000.0, 72))
	targetDistanceKm := polylineDistanceKmFromCoordinates(shape)

	// WHEN
	projected := projectShapePolylineToStart(shape, start, targetDistanceKm)
	projectedCenter, _ := shapeCenterAndRadius(projected)

	// THEN
	if len(projected) != len(shape) {
		t.Fatalf("expected projected shape to preserve point count, got %d vs %d", len(projected), len(shape))
	}
	firstPointDistance := haversineDistanceMeters(start.Lat, start.Lng, projected[0].Lat, projected[0].Lng)
	if firstPointDistance < 900.0 {
		t.Fatalf("expected first sketch point to remain on the drawn contour, got %.1fm from start", firstPointDistance)
	}
	centerDrift := haversineDistanceMeters(start.Lat, start.Lng, projectedCenter.Lat, projectedCenter.Lng)
	if centerDrift > 30.0 {
		t.Fatalf("expected map-placed shape center to stay near start, drift=%.1fm", centerDrift)
	}
}

func TestProjectShapePolylineToStart_RecentersRemoteShapeByCenter(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.1300, Lng: -1.6300}
	remoteShape := coordinatesFromLatLng(testCircleLatLng(45.1885, 5.7245, 1000.0, 72))
	targetDistanceKm := polylineDistanceKmFromCoordinates(remoteShape)

	// WHEN
	projected := projectShapePolylineToStart(remoteShape, start, targetDistanceKm)
	projectedCenter, _ := shapeCenterAndRadius(projected)

	// THEN
	centerDrift := haversineDistanceMeters(start.Lat, start.Lng, projectedCenter.Lat, projectedCenter.Lng)
	if centerDrift > 30.0 {
		t.Fatalf("expected remote shape to be recentered around start, drift=%.1fm", centerDrift)
	}
	firstPointDistance := haversineDistanceMeters(start.Lat, start.Lng, projected[0].Lat, projected[0].Lng)
	if firstPointDistance < 900.0 {
		t.Fatalf("expected recentered shape to preserve its contour radius, got %.1fm", firstPointDistance)
	}
}

func TestPrepareShapeForRouting_RotatesClosedShapeToNearestContourPoint(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.1300, Lng: -1.6300}
	shape := coordinatesFromLatLng(testCircleLatLng(start.Lat, start.Lng, 1000.0, 72))

	// WHEN
	routed := prepareShapeForRouting(shape, start)

	// THEN
	if len(routed) != len(shape) {
		t.Fatalf("expected routed shape to preserve point count, got %d vs %d", len(routed), len(shape))
	}
	firstPointDistance := haversineDistanceMeters(start.Lat, start.Lng, routed[0].Lat, routed[0].Lng)
	if firstPointDistance < 900.0 {
		t.Fatalf("expected routing to start on the drawn contour, got %.1fm from start", firstPointDistance)
	}
	closureDistance := haversineDistanceMeters(
		routed[0].Lat,
		routed[0].Lng,
		routed[len(routed)-1].Lat,
		routed[len(routed)-1].Lng,
	)
	if closureDistance > 120.0 {
		t.Fatalf("expected closed routing shape, got closure distance %.1fm", closureDistance)
	}
}

func TestBuildShapeBestEffortRoutingStrategies_ReturnsFallbackWaypoints(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.1300, Lng: -1.6300}
	shape := prepareShapeForRouting(
		coordinatesFromLatLng(testCircleLatLng(start.Lat, start.Lng, 1000.0, 72)),
		start,
	)

	// WHEN
	strategies := buildShapeBestEffortRoutingStrategies(shape[0], shape)

	// THEN
	if len(strategies) < 2 {
		t.Fatalf("expected simplified and envelope fallback strategies, got %d", len(strategies))
	}
	for _, strategy := range strategies {
		if !strategy.bestEffort {
			t.Fatalf("expected strategy %s to be marked best-effort", strategy.label)
		}
		if len(strategy.waypoints) < 3 {
			t.Fatalf("expected strategy %s to keep at least 3 waypoints, got %d", strategy.label, len(strategy.waypoints))
		}
	}
}

func TestToRouteCandidateBestEffort_KeepsHighlyRetracedShapeRoute(t *testing.T) {
	// GIVEN
	adapter := NewOSMRoutingAdapter()
	start := routesDomain.Coordinates{Lat: 48.1300, Lng: -1.6300}
	route := osrmRoute{
		Distance: 9000.0,
		Duration: 1800.0,
		Geometry: osrmGeometry{
			Coordinates: [][]float64{
				{start.Lng, start.Lat},
				{start.Lng, start.Lat + 0.035},
				{start.Lng, start.Lat},
				{start.Lng, start.Lat + 0.035},
				{start.Lng, start.Lat},
			},
		},
	}
	request := application.RoutingEngineRequest{
		StartPoint:       start,
		DistanceTargetKm: 4.0,
		RouteType:        "RIDE",
		ShapePolyline:    "[[48.13,-1.63],[48.15,-1.63],[48.13,-1.63]]",
		Limit:            1,
	}
	rejectCounts := map[string]int{}
	if _, ok := adapter.toRouteCandidate(request, route, 0, rejectCounts); ok {
		t.Fatalf("expected normal candidate conversion to reject excessive retrace")
	}

	// WHEN
	candidate, ok := adapter.toRouteCandidateBestEffort(request, route, 0, rejectCounts)

	// THEN
	if !ok {
		t.Fatalf("expected best-effort candidate conversion to keep the route")
	}
	foundReason := false
	for _, reason := range candidate.recommendation.Reasons {
		if strings.Contains(reason, "shape best effort") {
			foundReason = true
			break
		}
	}
	if !foundReason {
		t.Fatalf("expected best-effort reason, got %+v", candidate.recommendation.Reasons)
	}
}

func TestShapeModeMatchScore_RoadFirstPenalizesLowShapeSimilarity(t *testing.T) {
	// GIVEN
	baseMatch := 78.0

	// WHEN
	highScore, highDriftPenalty := shapeModeMatchScore(
		baseMatch,
		0.72,
		0.0,
		0.0,
		0.0,
		0.0,
		shapeModeStrategyRoadFirst,
	)
	lowScore, lowDriftPenalty := shapeModeMatchScore(
		baseMatch,
		0.38,
		0.0,
		0.0,
		0.0,
		0.0,
		shapeModeStrategyRoadFirst,
	)

	// THEN
	if lowDriftPenalty <= highDriftPenalty {
		t.Fatalf("expected low similarity to trigger stronger road-first drift penalty, high=%.2f low=%.2f", highDriftPenalty, lowDriftPenalty)
	}
	if lowScore >= highScore {
		t.Fatalf("expected low similarity to reduce road-first score, high=%.2f low=%.2f", highScore, lowScore)
	}
}

func TestShapeModeMatchScore_LowSimilarityPrefersShapeFirstOverRoadFirst(t *testing.T) {
	// GIVEN
	baseMatch := 82.0
	shapeScore := 0.40

	// WHEN
	shapeFirstScore, shapeFirstPenalty := shapeModeMatchScore(
		baseMatch,
		shapeScore,
		0.0,
		0.0,
		0.0,
		0.0,
		shapeModeStrategyShapeFirst,
	)
	roadFirstScore, roadFirstPenalty := shapeModeMatchScore(
		baseMatch,
		shapeScore,
		0.0,
		0.0,
		0.0,
		0.0,
		shapeModeStrategyRoadFirst,
	)

	// THEN
	if roadFirstPenalty <= shapeFirstPenalty {
		t.Fatalf("expected road-first to carry higher drift penalty on low-similarity routes, shape-first=%.2f road-first=%.2f", shapeFirstPenalty, roadFirstPenalty)
	}
	if roadFirstScore >= shapeFirstScore {
		t.Fatalf("expected shape-first score to stay above road-first on low-similarity route, shape-first=%.2f road-first=%.2f", shapeFirstScore, roadFirstScore)
	}
}

func TestShapeSimilarityScore_PenalizesAnchoredShapeDrift(t *testing.T) {
	// GIVEN
	shape := testCircleLatLng(48.1300, -1.6300, 1000.0, 96)
	matchingRoute := testCircleLatLng(48.1300, -1.6300, 1000.0, 96)
	shiftedRoute := testCircleLatLng(48.1300, -1.6460, 1000.0, 96)

	// WHEN
	matchingScore := shapeSimilarityScore(matchingRoute, shape)
	shiftedScore := shapeSimilarityScore(shiftedRoute, shape)

	// THEN
	if matchingScore < 0.95 {
		t.Fatalf("expected matching circle to keep high shape score, got %.3f", matchingScore)
	}
	if shiftedScore > 0.62 {
		t.Fatalf("expected shifted circle to be rejected-level similarity, got %.3f", shiftedScore)
	}
}

func TestShapeSimilarityScore_PenalizesOrderedPathMismatch(t *testing.T) {
	// GIVEN
	shape := testCircleLatLng(48.1300, -1.6300, 1000.0, 96)
	zigzagRoute := [][]float64{
		{48.1300, -1.6300},
		{48.1390, -1.6400},
		{48.1210, -1.6380},
		{48.1390, -1.6250},
		{48.1210, -1.6220},
		{48.1300, -1.6300},
	}

	// WHEN
	score := shapeSimilarityScore(zigzagRoute, shape)

	// THEN
	if score > 0.56 {
		t.Fatalf("expected zigzag route to fail shape-first similarity floor, got %.3f", score)
	}
}

func TestComputeSurfaceBreakdown_UsesSurfaceAndTracktypeTagsWhenAvailable(t *testing.T) {
	// GIVEN
	route := osrmRoute{
		Distance: 5000.0,
		Legs: []osrmLeg{
			{
				Steps: []osrmStep{
					{Distance: 1000.0, Mode: "cycling", Classes: []string{"surface=asphalt"}},
					{Distance: 1000.0, Mode: "cycling", Classes: []string{"surface:fine_gravel"}},
					{Distance: 1000.0, Mode: "cycling", Classes: []string{"tracktype=grade4"}},
					{Distance: 1000.0, Mode: "cycling", Surface: "surface=concrete:lanes"},
					{Distance: 1000.0, Mode: "cycling", TrackType: "tracktype=grade3"},
				},
			},
		},
	}

	// WHEN
	breakdown := computeSurfaceBreakdown(route)
	pavedRatio, gravelRatio, trailRatio, unknownRatio := breakdown.normalizedRatios()

	// THEN
	if pavedRatio < 0.39 || pavedRatio > 0.41 {
		t.Fatalf("expected paved ratio around 0.40, got %.3f", pavedRatio)
	}
	if gravelRatio < 0.39 || gravelRatio > 0.41 {
		t.Fatalf("expected gravel ratio around 0.40, got %.3f", gravelRatio)
	}
	if trailRatio < 0.19 || trailRatio > 0.21 {
		t.Fatalf("expected trail ratio around 0.20, got %.3f", trailRatio)
	}
	if unknownRatio != 0.0 {
		t.Fatalf("expected unknown ratio to be 0.0, got %.3f", unknownRatio)
	}
}

func testCircleLatLng(centerLat float64, centerLng float64, radiusMeters float64, pointCount int) [][]float64 {
	cosLat := math.Cos(degreesToRadians(centerLat))
	points := make([][]float64, 0, pointCount+1)
	for index := 0; index <= pointCount; index++ {
		angle := 2.0 * math.Pi * float64(index) / float64(pointCount)
		lat := centerLat + math.Sin(angle)*radiusMeters/111320.0
		lng := centerLng + math.Cos(angle)*radiusMeters/(111320.0*cosLat)
		points = append(points, []float64{lat, lng})
	}
	return points
}

func testStarLatLng(centerLat float64, centerLng float64, outerRadiusMeters float64, innerRadiusMeters float64) [][]float64 {
	cosLat := math.Cos(degreesToRadians(centerLat))
	points := make([][]float64, 0, 11)
	for index := 0; index <= 10; index++ {
		radius := outerRadiusMeters
		if index%2 == 1 {
			radius = innerRadiusMeters
		}
		angle := -math.Pi/2.0 + float64(index)*math.Pi/5.0
		lat := centerLat + math.Sin(angle)*radius/111320.0
		lng := centerLng + math.Cos(angle)*radius/(111320.0*cosLat)
		points = append(points, []float64{lat, lng})
	}
	return points
}

func coordinatesFromLatLng(points [][]float64) []routesDomain.Coordinates {
	coordinates := make([]routesDomain.Coordinates, 0, len(points))
	for _, point := range points {
		if len(point) < 2 {
			continue
		}
		coordinates = append(coordinates, routesDomain.Coordinates{Lat: point[0], Lng: point[1]})
	}
	return coordinates
}

func TestRespectsHalfPlaneDirection_NorthRejectsPointsSouthOfStart(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000}
	points := [][]float64{
		{48.13010, -1.63000},
		{48.13050, -1.62000},
		{48.12840, -1.61000}, // south of start by more than tolerance
	}

	// WHEN
	ok := respectsHalfPlaneDirection(points, start, "N", 120.0)

	// THEN
	if ok {
		t.Fatal("expected north direction filter to reject points south of start")
	}
}

func TestHasOppositeEdgeTraversal_DetectsBacktracking(t *testing.T) {
	// GIVEN
	points := [][]float64{
		{48.13000, -1.63000},
		{48.13100, -1.62900},
		{48.13200, -1.62800},
		{48.13100, -1.62900}, // traverses previous edge in reverse
	}

	// WHEN
	hasBacktracking := hasOppositeEdgeTraversal(points)

	// THEN
	if !hasBacktracking {
		t.Fatal("expected opposite-direction edge traversal to be detected")
	}
}

func TestHasMinimumSegmentDiversity_RejectsOverusedSegments(t *testing.T) {
	// GIVEN
	points := [][]float64{
		{48.13000, -1.63000},
		{48.13100, -1.62900},
		{48.13200, -1.62800},
		{48.13300, -1.62700},
		{48.13200, -1.62800},
		{48.13300, -1.62700},
		{48.13200, -1.62800},
	}

	// WHEN
	ok := hasMinimumSegmentDiversity(points, "RIDE")

	// THEN
	if ok {
		t.Fatal("expected diversity filter to reject a route reusing the same segment too often")
	}
}

func TestBuildOSRMScoringProfile_CalibratesWeightsByRouteType(t *testing.T) {
	// GIVEN
	rideProfile := buildOSRMScoringProfile("RIDE", true, false)
	hikeProfile := buildOSRMScoringProfile("HIKE", true, false)

	// WHEN
	hikeElevationHigher := hikeProfile.elevationWeight > rideProfile.elevationWeight
	hikeDistanceLower := hikeProfile.distanceWeight < rideProfile.distanceWeight

	// THEN
	if !hikeElevationHigher {
		t.Fatalf("expected hike profile elevation weight to be higher than ride, ride=%f hike=%f", rideProfile.elevationWeight, hikeProfile.elevationWeight)
	}
	if !hikeDistanceLower {
		t.Fatalf("expected hike profile distance weight to be lower than ride, ride=%f hike=%f", rideProfile.distanceWeight, hikeProfile.distanceWeight)
	}
}

func TestOSRMMatchScore_PenalizesOppositeDirection(t *testing.T) {
	// GIVEN
	elevationTarget := 600.0
	request := application.RoutingEngineRequest{
		StartPoint:       routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000},
		DistanceTargetKm: 40.0,
		ElevationTargetM: &elevationTarget,
		StartDirection:   "N",
		RouteType:        "RIDE",
		Limit:            4,
	}
	northPoints := [][]float64{
		{48.13000, -1.63000},
		{48.15000, -1.63000},
		{48.13000, -1.62000},
		{48.13000, -1.63000},
	}
	southPoints := [][]float64{
		{48.13000, -1.63000},
		{48.11000, -1.63000},
		{48.13000, -1.62000},
		{48.13000, -1.63000},
	}

	// WHEN
	northScore := osrmMatchScore(request, 40.0, 600.0, northPoints)
	southScore := osrmMatchScore(request, 40.0, 600.0, southPoints)

	// THEN
	if northScore <= southScore {
		t.Fatalf("expected north-aligned route to score higher than south-aligned route, north=%f south=%f", northScore, southScore)
	}
}

func TestSelectCandidatesWithRelaxation_WhenStrictFails_ThenRelaxedCandidateCanStillBeSelected(t *testing.T) {
	// GIVEN
	request := application.RoutingEngineRequest{
		StartPoint:       routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000},
		DistanceTargetKm: 40.0,
		StartDirection:   "N",
		RouteType:        "RIDE",
		Limit:            3,
	}
	candidates := []osrmRouteCandidate{
		{
			recommendation: routesDomain.RouteRecommendation{
				RouteID:       "candidate-relaxed",
				MatchScore:    88.0,
				PreviewLatLng: [][]float64{{48.13, -1.63}, {48.16, -1.62}, {48.13, -1.63}},
			},
			directionPenalty:    0.24, // strict rejects (>0.18), balanced accepts (<=0.28)
			backtrackingRatio:   0.04,
			segmentDiversity:    0.40,
			distanceDeltaRatio:  0.12,
			effectiveMatchScore: 82.0,
		},
	}
	rejectCounts := map[string]int{}

	// WHEN
	recommendations := selectCandidatesWithRelaxation(request, candidates, rejectCounts)

	// THEN
	if len(recommendations) != 1 {
		t.Fatalf("expected 1 recommendation after relaxation, got %d", len(recommendations))
	}
	if recommendations[0].RouteID != "candidate-relaxed" {
		t.Fatalf("expected candidate-relaxed to be selected, got %s", recommendations[0].RouteID)
	}
}

func TestDirectionalLobePenalty_PenalizesRoutesDominatingOppositeDirection(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000}
	northDominant := [][]float64{
		{48.13000, -1.63000},
		{48.14600, -1.62500},
		{48.14200, -1.61800},
		{48.13100, -1.62200},
		{48.13000, -1.63000},
	}
	southDominant := [][]float64{
		{48.13000, -1.63000},
		{48.11200, -1.62500},
		{48.11000, -1.61700},
		{48.12600, -1.62000},
		{48.13000, -1.63000},
	}

	// WHEN
	northPenalty := directionalLobePenalty(northDominant, start, "N")
	southPenalty := directionalLobePenalty(southDominant, start, "N")

	// THEN
	if northPenalty >= southPenalty {
		t.Fatalf("expected north-dominant route to have lower lobe penalty, north=%.3f south=%.3f", northPenalty, southPenalty)
	}
}

func TestSyntheticLoopWaypoints_WithNorthDirection_StayInForwardHemisphere(t *testing.T) {
	// GIVEN
	adapter := NewOSMRoutingAdapter()
	start := routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000}

	// WHEN
	waypoints := adapter.syntheticLoopWaypoints(start, 6.0, 0.0, "N", "RIDE", 0)

	// THEN
	if len(waypoints) < 3 {
		t.Fatalf("expected generated waypoints, got %d", len(waypoints))
	}
	for index, waypoint := range waypoints {
		// start/end are expected to be at start point; we only assert intermediate points.
		if index == 0 || index == len(waypoints)-1 {
			continue
		}
		if waypoint.Lat < start.Lat-0.0005 {
			t.Fatalf("expected directional waypoint %d to stay north-oriented, startLat=%.5f gotLat=%.5f", index, start.Lat, waypoint.Lat)
		}
	}
}

func TestBuildRouteRelaxationLevels_WhenDirectionStrict_ThenUsesStricterDirectionThresholds(t *testing.T) {
	// GIVEN
	regular := buildRouteRelaxationLevels("RIDE", true, false, 40.0)
	strict := buildRouteRelaxationLevels("RIDE", true, true, 40.0)

	// WHEN
	regularStrictMax := regular[0].maxDirectionPenalty
	strictStrictMax := strict[0].maxDirectionPenalty
	regularFallbackMax := regular[len(regular)-1].maxDirectionPenalty
	strictFallbackMax := strict[len(strict)-1].maxDirectionPenalty

	// THEN
	if strictStrictMax >= regularStrictMax {
		t.Fatalf("expected strict mode to tighten strict-level direction threshold, strict=%f regular=%f", strictStrictMax, regularStrictMax)
	}
	if strictFallbackMax >= regularFallbackMax {
		t.Fatalf("expected strict mode to tighten fallback-level direction threshold, strict=%f regular=%f", strictFallbackMax, regularFallbackMax)
	}
}

func TestBuildRouteRelaxationLevels_UsesNativeUltraThresholds(t *testing.T) {
	// GIVEN
	levels := buildRouteRelaxationLevels("RIDE", false, false, 40.0)

	// WHEN
	fallback := levels[len(levels)-1]

	// THEN
	if fallback.maxEdgeReuseRatio > 0.10 {
		t.Fatalf("expected native ultra fallback edge-reuse threshold to stay <= 0.10, got %f", fallback.maxEdgeReuseRatio)
	}
	if fallback.maxBacktrackingRatio > 0.03 {
		t.Fatalf("expected native ultra fallback backtracking threshold to stay <= 0.03, got %f", fallback.maxBacktrackingRatio)
	}
}

func TestBuildRouteRelaxationLevels_WhenLongDistance_ThenAxisReuseCapsAreHigher(t *testing.T) {
	// GIVEN
	shortDistanceLevels := buildRouteRelaxationLevels("RIDE", true, false, 30.0)
	longDistanceLevels := buildRouteRelaxationLevels("RIDE", true, false, 130.0)

	// WHEN
	shortFallbackAxisCap := shortDistanceLevels[len(shortDistanceLevels)-1].maxAxisReuseCount
	longFallbackAxisCap := longDistanceLevels[len(longDistanceLevels)-1].maxAxisReuseCount

	// THEN
	if longFallbackAxisCap <= shortFallbackAxisCap {
		t.Fatalf("expected long-distance fallback axis cap to be higher, long=%d short=%d", longFallbackAxisCap, shortFallbackAxisCap)
	}
}

func TestHalfPlaneViolationRatio_WhenPointsCrossForbiddenHalfPlane_ThenPenaltyIsPositive(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000}
	points := [][]float64{
		{48.13010, -1.63000},
		{48.12980, -1.62900},
		{48.12890, -1.62800}, // south of start for a north direction request
	}

	// WHEN
	penalty := halfPlaneViolationRatio(points, start, "N", 120.0)

	// THEN
	if penalty <= 0 {
		t.Fatalf("expected positive half-plane violation penalty, got %.3f", penalty)
	}
}

func TestFarOppositeViolationRatio_WhenRouteExcursionsGoFarOpposite_ThenPenaltyIsPositive(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000}
	mostlyNorthWithLocalOscillation := [][]float64{
		{48.13000, -1.63000},
		{48.13220, -1.62950},
		{48.12995, -1.62980}, // local oscillation around start, below guard band
		{48.13500, -1.62850},
		{48.13800, -1.62700},
		{48.13000, -1.63000},
	}
	farSouthExcursion := [][]float64{
		{48.13000, -1.63000},
		{48.13300, -1.62950},
		{48.13600, -1.62800},
		{48.12100, -1.62720}, // far opposite
		{48.11850, -1.62680}, // far opposite
		{48.13450, -1.62830},
		{48.13000, -1.63000},
	}

	// WHEN
	cleanPenalty := farOppositeViolationRatio(mostlyNorthWithLocalOscillation, start, "N", 120.0)
	oppositePenalty := farOppositeViolationRatio(farSouthExcursion, start, "N", 120.0)

	// THEN
	if cleanPenalty != 0.0 {
		t.Fatalf("expected local oscillation to be ignored by far-opposite metric, got %.3f", cleanPenalty)
	}
	if oppositePenalty <= 0.0 {
		t.Fatalf("expected far opposite excursion penalty to be positive, got %.3f", oppositePenalty)
	}
}

func TestCombinedDirectionPenalty_WhenFarOppositeExcursion_ThenPenaltyIncreases(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000}
	northDominant := [][]float64{
		{48.13000, -1.63000},
		{48.13220, -1.62950},
		{48.13450, -1.62840},
		{48.13680, -1.62710},
		{48.13300, -1.62830},
		{48.13000, -1.63000},
	}
	northWithFarSouthExcursion := [][]float64{
		{48.13000, -1.63000},
		{48.13220, -1.62950},
		{48.13600, -1.62800},
		{48.12100, -1.62720}, // far opposite
		{48.11850, -1.62680}, // far opposite
		{48.13500, -1.62820},
		{48.13000, -1.63000},
	}

	// WHEN
	cleanPenalty := combinedDirectionPenalty(northDominant, start, "N", 120.0)
	excursionPenalty := combinedDirectionPenalty(northWithFarSouthExcursion, start, "N", 120.0)

	// THEN
	if excursionPenalty <= cleanPenalty {
		t.Fatalf("expected far opposite excursion to increase combined direction penalty, clean=%.3f excursion=%.3f", cleanPenalty, excursionPenalty)
	}
}

func TestDirectionalQuadrantPenalty_PenalizesOppositeQuadrantMajority(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000}
	northMajority := [][]float64{
		{48.13000, -1.63000},
		{48.13700, -1.62980},
		{48.14100, -1.62830},
		{48.13800, -1.62740},
		{48.13300, -1.62860},
		{48.13000, -1.63000},
	}
	southMajority := [][]float64{
		{48.13000, -1.63000},
		{48.12700, -1.62970},
		{48.12100, -1.62820},
		{48.11800, -1.62740},
		{48.12400, -1.62850},
		{48.13000, -1.63000},
	}

	// WHEN
	northPenalty := directionalQuadrantPenalty(northMajority, start, "N", 120.0)
	southPenalty := directionalQuadrantPenalty(southMajority, start, "N", 120.0)

	// THEN
	if northPenalty >= southPenalty {
		t.Fatalf("expected north-majority route to have lower quadrant penalty, north=%.3f south=%.3f", northPenalty, southPenalty)
	}
	if southPenalty <= 0.0 {
		t.Fatalf("expected opposite-quadrant majority to trigger positive penalty, got %.3f", southPenalty)
	}
}

func TestCombinedDirectionPenalty_WhenQuadrantMajorityIsOpposite_ThenPenaltyIncreases(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000}
	northMajority := [][]float64{
		{48.13000, -1.63000},
		{48.13700, -1.62980},
		{48.14100, -1.62830},
		{48.13800, -1.62740},
		{48.13300, -1.62860},
		{48.13000, -1.63000},
	}
	southMajority := [][]float64{
		{48.13000, -1.63000},
		{48.12700, -1.62970},
		{48.12100, -1.62820},
		{48.11800, -1.62740},
		{48.12400, -1.62850},
		{48.13000, -1.63000},
	}

	// WHEN
	northPenalty := combinedDirectionPenalty(northMajority, start, "N", 120.0)
	southPenalty := combinedDirectionPenalty(southMajority, start, "N", 120.0)

	// THEN
	if southPenalty <= northPenalty {
		t.Fatalf("expected opposite-quadrant majority to increase combined penalty, north=%.3f south=%.3f", northPenalty, southPenalty)
	}
}

func TestSelectCandidatesWithRelaxation_PrioritizesLowerBacktracking(t *testing.T) {
	// GIVEN
	request := application.RoutingEngineRequest{
		StartPoint:       routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000},
		DistanceTargetKm: 40.0,
		StartDirection:   "N",
		RouteType:        "RIDE",
		Limit:            1,
	}
	candidates := []osrmRouteCandidate{
		{
			recommendation:      routesDomain.RouteRecommendation{RouteID: "high-backtracking", MatchScore: 95.0},
			directionPenalty:    0.10,
			backtrackingRatio:   0.22,
			segmentDiversity:    0.60,
			distanceDeltaRatio:  0.10,
			effectiveMatchScore: 94.0,
		},
		{
			recommendation:      routesDomain.RouteRecommendation{RouteID: "low-backtracking", MatchScore: 90.0},
			directionPenalty:    0.10,
			backtrackingRatio:   0.02,
			segmentDiversity:    0.60,
			distanceDeltaRatio:  0.10,
			effectiveMatchScore: 88.0,
		},
	}
	rejectCounts := map[string]int{}

	// WHEN
	recommendations := selectCandidatesWithRelaxation(request, candidates, rejectCounts)

	// THEN
	if len(recommendations) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(recommendations))
	}
	if recommendations[0].RouteID != "low-backtracking" {
		t.Fatalf("expected low-backtracking route to be selected first, got %s", recommendations[0].RouteID)
	}
}

func TestSelectCandidatesWithRelaxation_ShapeModePrioritizesArtFitBeforeStrictness(t *testing.T) {
	// GIVEN
	request := application.RoutingEngineRequest{
		StartPoint:       routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000},
		DistanceTargetKm: 12.0,
		RouteType:        "RIDE",
		ShapePolyline:    "[[48.13,-1.63],[48.14,-1.62],[48.13,-1.63]]",
		Limit:            1,
	}
	highShapeScore := 0.82
	lowShapeScore := 0.44
	candidates := []osrmRouteCandidate{
		{
			recommendation: routesDomain.RouteRecommendation{
				RouteID:    "strict-low-art-fit",
				MatchScore: 92.0,
				ShapeScore: &lowShapeScore,
			},
			backtrackingRatio:   0.0005,
			corridorOverlap:     0.0010,
			edgeReuseRatio:      0.005,
			maxAxisReuseCount:   1,
			segmentDiversity:    0.70,
			effectiveMatchScore: 91.0,
		},
		{
			recommendation: routesDomain.RouteRecommendation{
				RouteID:    "relaxed-high-art-fit",
				MatchScore: 78.0,
				ShapeScore: &highShapeScore,
			},
			backtrackingRatio:   0.0060,
			corridorOverlap:     0.0010,
			edgeReuseRatio:      0.005,
			maxAxisReuseCount:   1,
			segmentDiversity:    0.70,
			effectiveMatchScore: 70.0,
		},
	}
	rejectCounts := map[string]int{}

	// WHEN
	recommendations := selectCandidatesWithRelaxation(request, candidates, rejectCounts)

	// THEN
	if len(recommendations) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(recommendations))
	}
	if recommendations[0].RouteID != "relaxed-high-art-fit" {
		t.Fatalf("expected relaxed-high-art-fit route to be selected first, got %s", recommendations[0].RouteID)
	}
	foundArtFitReason := false
	for _, reason := range recommendations[0].Reasons {
		if reason == "Selection priority: art-fit first" {
			foundArtFitReason = true
			break
		}
	}
	if !foundArtFitReason {
		t.Fatalf("expected art-fit selection reason, got %+v", recommendations[0].Reasons)
	}
}

func TestSelectCandidatesWithRelaxation_ShapeModeUsesArtFitSoftBeforeEmergency(t *testing.T) {
	// GIVEN
	request := application.RoutingEngineRequest{
		StartPoint:       routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000},
		DistanceTargetKm: 12.0,
		RouteType:        "RIDE",
		ShapePolyline:    "[[48.13,-1.63],[48.14,-1.62],[48.13,-1.63]]",
		Limit:            1,
	}
	shapeScore := 0.57
	candidates := []osrmRouteCandidate{
		{
			recommendation: routesDomain.RouteRecommendation{
				RouteID:    "shape-needs-soft-art-fit",
				MatchScore: 70.0,
				ShapeScore: &shapeScore,
			},
			backtrackingRatio:   0.17,
			corridorOverlap:     0.49,
			edgeReuseRatio:      0.12,
			maxAxisReuseCount:   2,
			segmentDiversity:    0.70,
			distanceDeltaRatio:  0.02,
			effectiveMatchScore: 20.0,
		},
	}
	rejectCounts := map[string]int{}

	// WHEN
	recommendations := selectCandidatesWithRelaxation(request, candidates, rejectCounts)

	// THEN
	if len(recommendations) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(recommendations))
	}
	if recommendations[0].RouteID != "shape-needs-soft-art-fit" {
		t.Fatalf("expected shape-needs-soft-art-fit route to be selected, got %s", recommendations[0].RouteID)
	}
	foundArtFitReason := false
	foundSoftProfile := false
	for _, reason := range recommendations[0].Reasons {
		if reason == "Selection priority: art-fit first" {
			foundArtFitReason = true
		}
		if reason == "Selection profile: best-effort-soft (art-fit first)" {
			foundSoftProfile = true
		}
	}
	if !foundArtFitReason || !foundSoftProfile {
		t.Fatalf("expected art-fit soft selection reasons, got %+v", recommendations[0].Reasons)
	}
}

func TestSelectCandidatesWithRelaxation_WhenDirectionRequested_PrioritizesDirectionBeforeScore(t *testing.T) {
	// GIVEN
	request := application.RoutingEngineRequest{
		StartPoint:       routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000},
		DistanceTargetKm: 40.0,
		StartDirection:   "N",
		RouteType:        "RIDE",
		Limit:            1,
	}
	candidates := []osrmRouteCandidate{
		{
			recommendation:      routesDomain.RouteRecommendation{RouteID: "better-score", MatchScore: 96.0},
			directionPenalty:    0.12,
			backtrackingRatio:   0.0005,
			corridorOverlap:     0.0010,
			edgeReuseRatio:      0.005,
			maxAxisReuseCount:   2,
			segmentDiversity:    0.70,
			distanceDeltaRatio:  0.02,
			effectiveMatchScore: 95.0,
		},
		{
			recommendation:      routesDomain.RouteRecommendation{RouteID: "better-direction", MatchScore: 84.0},
			directionPenalty:    0.10,
			backtrackingRatio:   0.0005,
			corridorOverlap:     0.0010,
			edgeReuseRatio:      0.005,
			maxAxisReuseCount:   2,
			segmentDiversity:    0.70,
			distanceDeltaRatio:  0.02,
			effectiveMatchScore: 72.0,
		},
	}
	rejectCounts := map[string]int{}

	// WHEN
	recommendations := selectCandidatesWithRelaxation(request, candidates, rejectCounts)

	// THEN
	if len(recommendations) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(recommendations))
	}
	if recommendations[0].RouteID != "better-direction" {
		t.Fatalf("expected direction-aligned route first, got %s", recommendations[0].RouteID)
	}
}

func TestSelectCandidatesWithRelaxation_WhenAllLevelsReject_ThenBestEffortStillReturnsCandidate(t *testing.T) {
	// GIVEN
	request := application.RoutingEngineRequest{
		StartPoint:       routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000},
		DistanceTargetKm: 40.0,
		StartDirection:   "N",
		RouteType:        "RIDE",
		Limit:            1,
	}
	candidates := []osrmRouteCandidate{
		{
			recommendation:      routesDomain.RouteRecommendation{RouteID: "needs-best-effort", MatchScore: 78.0},
			directionPenalty:    0.45, // reject configured levels, accepted by directional safety net (<= 0.52)
			backtrackingRatio:   0.12, // reject fallback (0.015), accepted by directional best-effort (<= 0.18)
			corridorOverlap:     0.08, // reject fallback (0.018), accepted by directional best-effort (<= 0.14)
			segmentDiversity:    0.02, // reject all configured levels
			distanceDeltaRatio:  0.24, // reject strict/balanced/relaxed/fallback, accepted by directional best-effort
			effectiveMatchScore: 75.0,
		},
	}
	rejectCounts := map[string]int{}

	// WHEN
	recommendations := selectCandidatesWithRelaxation(request, candidates, rejectCounts)

	// THEN
	if len(recommendations) != 1 {
		t.Fatalf("expected 1 recommendation from best-effort fallback, got %d", len(recommendations))
	}
	if recommendations[0].RouteID != "needs-best-effort" {
		t.Fatalf("expected needs-best-effort to be selected, got %s", recommendations[0].RouteID)
	}
	if len(recommendations[0].Reasons) == 0 {
		t.Fatalf("expected selection reason to include best-effort profile")
	}
}

func TestSelectCandidatesWithRelaxation_WhenDirectionalStrictProfileRejectsAll_ThenDirectionalBestEffortCanReturnRoute(t *testing.T) {
	// GIVEN
	request := application.RoutingEngineRequest{
		StartPoint:         routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000},
		DistanceTargetKm:   40.0,
		StartDirection:     "N",
		RouteType:          "RIDE",
		StrictBacktracking: true,
		Limit:              1,
	}
	candidates := []osrmRouteCandidate{
		{
			recommendation:      routesDomain.RouteRecommendation{RouteID: "directional-safety-net", MatchScore: 73.0},
			directionPenalty:    0.34,
			backtrackingRatio:   0.17,
			corridorOverlap:     0.13,
			edgeReuseRatio:      0.12,
			segmentDiversity:    0.18,
			distanceDeltaRatio:  0.23,
			effectiveMatchScore: 68.0,
		},
	}
	rejectCounts := map[string]int{}

	// WHEN
	recommendations := selectCandidatesWithRelaxation(request, candidates, rejectCounts)

	// THEN
	if len(recommendations) != 1 {
		t.Fatalf("expected 1 recommendation from directional best-effort, got %d", len(recommendations))
	}
	if recommendations[0].RouteID != "directional-safety-net" {
		t.Fatalf("expected directional-safety-net route to be selected, got %s", recommendations[0].RouteID)
	}
}

func TestCorridorOverlapRatio_DetectsNearParallelOutAndBackCorridor(t *testing.T) {
	// GIVEN
	points := [][]float64{
		{48.13000, -1.63000},
		{48.13000, -1.62000},
		{48.13020, -1.62000},
		{48.13020, -1.63000},
		{48.13000, -1.63000},
	}

	// WHEN
	overlapRatio := corridorOverlapRatio(points)

	// THEN
	if overlapRatio <= 0.0 {
		t.Fatalf("expected a positive corridor overlap ratio, got %.3f", overlapRatio)
	}
}

func TestEdgeReuseRatio_WhenLoopReusesSameAxis_ThenPenaltyIsPositive(t *testing.T) {
	// GIVEN
	points := [][]float64{
		{48.13000, -1.63000},
		{48.13200, -1.62800},
		{48.13400, -1.62600},
		{48.13200, -1.62800},
		{48.13400, -1.62600},
		{48.13600, -1.62400},
	}

	// WHEN
	reuse := edgeReuseRatio(points)

	// THEN
	if reuse <= 0.0 {
		t.Fatalf("expected edge reuse ratio to be positive, got %.3f", reuse)
	}
}

func TestSortAnchorsByHistoryReuse_PrioritizesMostUsedZones(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 45.10, Lng: 6.10}
	highReuseAnchor := routesDomain.Coordinates{Lat: 45.30, Lng: 6.30}
	lowReuseAnchor := routesDomain.Coordinates{Lat: 45.32, Lng: 6.32}
	highReuseZone := historyZoneKey(highReuseAnchor.Lat, highReuseAnchor.Lng)
	lowReuseZone := historyZoneKey(lowReuseAnchor.Lat, lowReuseAnchor.Lng)
	context := routingHistoryBiasContext{
		enabled:      true,
		zoneScores:   map[string]float64{highReuseZone: 10_000.0, lowReuseZone: 200.0},
		maxZoneScore: 10_000.0,
	}

	// WHEN
	sorted := sortAnchorsByHistoryReuse(
		[]routesDomain.Coordinates{highReuseAnchor, lowReuseAnchor},
		start,
		context,
	)

	// THEN
	if len(sorted) != 2 {
		t.Fatalf("expected 2 anchors, got %d", len(sorted))
	}
	if sorted[0] != highReuseAnchor {
		t.Fatalf("expected high reuse anchor first, got %+v", sorted[0])
	}
}

func TestApplyHistoryBiasToCandidate_RewardsKnownHistoryReuse(t *testing.T) {
	// GIVEN
	points := [][]float64{
		{45.0000, 6.0000},
		{45.0200, 6.0000},
		{45.0000, 6.0000},
	}
	axisA := historyAxisKey(points[0][0], points[0][1], points[1][0], points[1][1])
	axisB := historyAxisKey(points[1][0], points[1][1], points[2][0], points[2][1])
	zoneA := historyZoneKey((points[0][0]+points[1][0])/2.0, (points[0][1]+points[1][1])/2.0)
	zoneB := historyZoneKey((points[1][0]+points[2][0])/2.0, (points[1][1]+points[2][1])/2.0)
	request := application.RoutingEngineRequest{
		RouteType:          "RIDE",
		HistoryBiasEnabled: true,
		HistoryProfile: &application.RoutingHistoryProfile{
			RouteType:  "RIDE",
			AxisScores: map[string]float64{axisA: 8_000.0, axisB: 8_000.0},
			ZoneScores: map[string]float64{zoneA: 6_000.0, zoneB: 6_000.0},
		},
	}
	context := buildRoutingHistoryBiasContext(request)
	candidate := osrmRouteCandidate{
		recommendation:      routesDomain.RouteRecommendation{PreviewLatLng: points},
		effectiveMatchScore: 80.0,
	}

	// WHEN
	biased := applyHistoryBiasToCandidate(candidate, routesDomain.Coordinates{Lat: 45.0000, Lng: 6.0000}, context)

	// THEN
	if biased.historyReuseScore <= 0.0 {
		t.Fatalf("expected positive history reuse score, got %.3f", biased.historyReuseScore)
	}
	if biased.effectiveMatchScore <= candidate.effectiveMatchScore {
		t.Fatalf("expected effective score bonus, before=%.2f after=%.2f", candidate.effectiveMatchScore, biased.effectiveMatchScore)
	}
	if len(biased.recommendation.Reasons) == 0 {
		t.Fatalf("expected history bias reason to be appended")
	}
}

func TestSelectCandidatesWithRelaxation_PrioritizesHigherHistoryReuseWhenMetricsTie(t *testing.T) {
	// GIVEN
	request := application.RoutingEngineRequest{
		StartPoint:       routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000},
		DistanceTargetKm: 40.0,
		RouteType:        "RIDE",
		Limit:            1,
	}
	candidates := []osrmRouteCandidate{
		{
			recommendation:      routesDomain.RouteRecommendation{RouteID: "high-history", MatchScore: 90.0},
			directionPenalty:    0.0,
			backtrackingRatio:   0.0,
			corridorOverlap:     0.0,
			edgeReuseRatio:      0.0,
			maxAxisReuseCount:   1,
			segmentDiversity:    0.90,
			distanceDeltaRatio:  0.02,
			historyReuseScore:   0.85,
			effectiveMatchScore: 88.0,
		},
		{
			recommendation:      routesDomain.RouteRecommendation{RouteID: "low-history", MatchScore: 90.0},
			directionPenalty:    0.0,
			backtrackingRatio:   0.0,
			corridorOverlap:     0.0,
			edgeReuseRatio:      0.0,
			maxAxisReuseCount:   1,
			segmentDiversity:    0.90,
			distanceDeltaRatio:  0.02,
			historyReuseScore:   0.10,
			effectiveMatchScore: 88.0,
		},
	}
	rejectCounts := map[string]int{}

	// WHEN
	recommendations := selectCandidatesWithRelaxation(request, candidates, rejectCounts)

	// THEN
	if len(recommendations) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(recommendations))
	}
	if recommendations[0].RouteID != "high-history" {
		t.Fatalf("expected high-history route first, got %s", recommendations[0].RouteID)
	}
}
