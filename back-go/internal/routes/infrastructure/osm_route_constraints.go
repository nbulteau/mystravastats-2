package infrastructure

import (
	"math"
	"mystravastats/internal/routes/application"
	routesDomain "mystravastats/internal/routes/domain"
	"strings"
)

func startsNearRequestedStart(points [][]float64, start routesDomain.Coordinates, toleranceMeters float64) bool {
	if len(points) == 0 {
		return false
	}
	first := points[0]
	if len(first) < 2 {
		return false
	}
	return haversineDistanceMeters(first[0], first[1], start.Lat, start.Lng) <= toleranceMeters
}

func combinedDirectionPenalty(
	points [][]float64,
	start routesDomain.Coordinates,
	direction string,
	toleranceMeters float64,
) float64 {
	if strings.TrimSpace(direction) == "" {
		return 0.0
	}
	// We combine three direction signals:
	// - initial heading alignment (bearing-based)
	// - half-plane violations (did the route go too much in the opposite side)
	// - global lobe dominance (does the whole loop stay mostly in requested direction)
	// The max keeps enforcement robust in dense urban grids.
	// Bearing is intentionally softened because local street orientation near the
	// start can temporarily oppose the desired global direction.
	bearingPenalty := directionPenaltyFromPreview(points, direction)
	halfPlanePenalty := halfPlaneViolationRatio(points, start, direction, toleranceMeters)
	lobePenalty := directionalLobePenalty(points, start, direction)
	farOppositePenalty := farOppositeViolationRatio(points, start, direction, toleranceMeters)
	quadrantPenalty := directionalQuadrantPenalty(points, start, direction, toleranceMeters)
	return math.Max(
		math.Max(
			math.Max(bearingPenalty*0.65, halfPlanePenalty),
			math.Max(lobePenalty, farOppositePenalty),
		),
		quadrantPenalty,
	)
}

func halfPlaneViolationRatio(
	points [][]float64,
	start routesDomain.Coordinates,
	direction string,
	toleranceMeters float64,
) float64 {
	normalized := strings.ToUpper(strings.TrimSpace(direction))
	if normalized == "" || len(points) == 0 {
		return 0.0
	}
	latTolerance := toleranceMeters / 111320.0
	lngTolerance := toleranceMeters / math.Max(1000.0, 111320.0*math.Cos(degreesToRadians(start.Lat)))

	total := 0
	violations := 0
	for _, point := range points {
		if len(point) < 2 {
			continue
		}
		total++
		switch normalized {
		case "N":
			if point[0] < start.Lat-latTolerance {
				violations++
			}
		case "S":
			if point[0] > start.Lat+latTolerance {
				violations++
			}
		case "E":
			if point[1] < start.Lng-lngTolerance {
				violations++
			}
		case "W":
			if point[1] > start.Lng+lngTolerance {
				violations++
			}
		}
	}
	if total == 0 {
		return 0.0
	}
	return float64(violations) / float64(total)
}

func directionalLobePenalty(
	points [][]float64,
	start routesDomain.Coordinates,
	direction string,
) float64 {
	normalized := strings.ToUpper(strings.TrimSpace(direction))
	if normalized == "" || len(points) == 0 {
		return 0.0
	}

	desiredExtent := 0.0
	oppositeExtent := 0.0
	sumProjection := 0.0
	projectionCount := 0

	for _, point := range points {
		if len(point) < 2 {
			continue
		}
		projection, ok := directionProjectionMeters(point[0], point[1], start, normalized)
		if !ok {
			continue
		}
		if projection > desiredExtent {
			desiredExtent = projection
		}
		if projection < 0 && -projection > oppositeExtent {
			oppositeExtent = -projection
		}
		sumProjection += projection
		projectionCount++
	}

	if projectionCount == 0 {
		return 0.0
	}

	// Dominance asks: "how much of the route envelope is on requested side?"
	// 1.0 means full dominance on requested side, 0.5 is symmetric, 0 is opposite.
	dominancePenalty := 0.0
	totalExtent := desiredExtent + oppositeExtent
	if totalExtent > 1.0 {
		dominanceRatio := desiredExtent / totalExtent
		// Keep a clearer direction dominance in dense grids.
		dominancePenalty = clampUnit((0.68 - dominanceRatio) / 0.68)
	}

	// Average projection guard: route center of mass should not drift opposite.
	avgPenalty := 0.0
	if desiredExtent > 1.0 {
		avgProjection := sumProjection / float64(projectionCount)
		avgPenalty = clampUnit((-avgProjection) / math.Max(desiredExtent*0.25, 1.0))
	}

	return math.Max(dominancePenalty, avgPenalty)
}

func farOppositeViolationRatio(
	points [][]float64,
	start routesDomain.Coordinates,
	direction string,
	toleranceMeters float64,
) float64 {
	normalized := strings.ToUpper(strings.TrimSpace(direction))
	if normalized == "" || len(points) == 0 {
		return 0.0
	}

	guardBand := math.Max(toleranceMeters*1.8, 220.0)
	total := 0
	violations := 0

	for _, point := range points {
		if len(point) < 2 {
			continue
		}
		projection, ok := directionProjectionMeters(point[0], point[1], start, normalized)
		if !ok {
			continue
		}
		if math.Abs(projection) < guardBand {
			// Ignore local oscillations around start/return hub.
			continue
		}
		total++
		if projection < -guardBand {
			violations++
		}
	}
	if total == 0 {
		return 0.0
	}
	return float64(violations) / float64(total)
}

func directionalQuadrantPenalty(
	points [][]float64,
	start routesDomain.Coordinates,
	direction string,
	toleranceMeters float64,
) float64 {
	normalized := strings.ToUpper(strings.TrimSpace(direction))
	if normalized == "" || len(points) < 2 {
		return 0.0
	}

	// Ignore local oscillations around start and focus on dominant travel zones.
	guardBand := math.Max(toleranceMeters*1.2, 160.0)
	desiredMeters := 0.0
	oppositeMeters := 0.0

	for index := 0; index < len(points)-1; index++ {
		from := points[index]
		to := points[index+1]
		if len(from) < 2 || len(to) < 2 {
			continue
		}
		segmentMeters := haversineDistanceMeters(from[0], from[1], to[0], to[1])
		if segmentMeters < 12.0 {
			continue
		}
		midLat := (from[0] + to[0]) / 2.0
		midLng := (from[1] + to[1]) / 2.0
		projection, ok := directionProjectionMeters(midLat, midLng, start, normalized)
		if !ok {
			continue
		}
		if math.Abs(projection) < guardBand {
			continue
		}
		if projection >= 0 {
			desiredMeters += segmentMeters
		} else {
			oppositeMeters += segmentMeters
		}
	}

	totalMeters := desiredMeters + oppositeMeters
	if totalMeters <= 0 {
		return 0.0
	}
	desiredRatio := desiredMeters / totalMeters
	// Keep at least ~62% of routed distance in requested quadrant.
	return clampUnit((0.62 - desiredRatio) / 0.62)
}

func directionProjectionMeters(
	lat float64,
	lng float64,
	start routesDomain.Coordinates,
	normalizedDirection string,
) (float64, bool) {
	latMeters := (lat - start.Lat) * 111320.0
	lngMeters := (lng - start.Lng) * 111320.0 * math.Cos(degreesToRadians(start.Lat))
	switch normalizedDirection {
	case "N":
		return latMeters, true
	case "S":
		return -latMeters, true
	case "E":
		return lngMeters, true
	case "W":
		return -lngMeters, true
	default:
		return 0.0, false
	}
}

func clampUnit(value float64) float64 {
	if value <= 0 {
		return 0
	}
	if value >= 1 {
		return 1
	}
	return value
}

type pathSegment struct {
	startLat float64
	startLng float64
	endLat   float64
	endLng   float64
	midLat   float64
	midLng   float64
	lengthM  float64
	bearing  float64
}

func corridorOverlapRatio(points [][]float64) float64 {
	if len(points) < 4 {
		return 0.0
	}
	sampled := samplePolylinePoints(points, 260)
	segments := buildPathSegments(sampled)
	if len(segments) < 2 {
		return 0.0
	}

	flagged := make([]bool, len(segments))
	for i := 0; i < len(segments); i++ {
		// Skip only immediate neighbors to avoid counting normal local curvature as overlap.
		for j := 0; j < i-1; j++ {
			if segmentsLikelySameCorridor(segments[i], segments[j]) {
				flagged[i] = true
				flagged[j] = true
			}
		}
	}
	overlapped := 0
	for _, value := range flagged {
		if value {
			overlapped++
		}
	}
	return float64(overlapped) / float64(len(segments))
}

func samplePolylinePoints(points [][]float64, maxPoints int) [][]float64 {
	if len(points) <= maxPoints || maxPoints <= 0 {
		return points
	}
	step := int(math.Ceil(float64(len(points)) / float64(maxPoints)))
	if step < 1 {
		step = 1
	}
	sampled := make([][]float64, 0, maxPoints+1)
	lastIndex := len(points) - 1
	for index := 0; index < len(points); index += step {
		sampled = append(sampled, points[index])
	}
	lastSample := sampled[len(sampled)-1]
	lastPoint := points[lastIndex]
	if len(lastSample) < 2 || len(lastPoint) < 2 || lastSample[0] != lastPoint[0] || lastSample[1] != lastPoint[1] {
		sampled = append(sampled, lastPoint)
	}
	return sampled
}

func buildPathSegments(points [][]float64) []pathSegment {
	segments := make([]pathSegment, 0, len(points))
	for index := 0; index < len(points)-1; index++ {
		left := points[index]
		right := points[index+1]
		if len(left) < 2 || len(right) < 2 {
			continue
		}
		lengthM := haversineDistanceMeters(left[0], left[1], right[0], right[1])
		if lengthM < 12.0 {
			continue
		}
		segments = append(segments, pathSegment{
			startLat: left[0],
			startLng: left[1],
			endLat:   right[0],
			endLng:   right[1],
			midLat:   (left[0] + right[0]) / 2.0,
			midLng:   (left[1] + right[1]) / 2.0,
			lengthM:  lengthM,
			bearing:  osrmBearingDegrees(left[0], left[1], right[0], right[1]),
		})
	}
	return segments
}

func segmentsLikelySameCorridor(left pathSegment, right pathSegment) bool {
	const midpointToleranceMeters = 50.0
	const endpointToleranceMeters = 80.0

	midpointDistance := haversineDistanceMeters(left.midLat, left.midLng, right.midLat, right.midLng)
	if midpointDistance > midpointToleranceMeters {
		return false
	}
	leftToRightStart := haversineDistanceMeters(left.startLat, left.startLng, right.startLat, right.startLng)
	leftToRightEnd := haversineDistanceMeters(left.startLat, left.startLng, right.endLat, right.endLng)
	rightToLeftStart := haversineDistanceMeters(left.endLat, left.endLng, right.startLat, right.startLng)
	rightToLeftEnd := haversineDistanceMeters(left.endLat, left.endLng, right.endLat, right.endLng)
	if math.Min(leftToRightStart, leftToRightEnd) > endpointToleranceMeters ||
		math.Min(rightToLeftStart, rightToLeftEnd) > endpointToleranceMeters {
		return false
	}
	bearingDiff := math.Abs(left.bearing - right.bearing)
	if bearingDiff > 180.0 {
		bearingDiff = 360.0 - bearingDiff
	}
	if bearingDiff > 22.0 && bearingDiff < 158.0 {
		return false
	}
	maxLength := math.Max(left.lengthM, right.lengthM)
	minLength := math.Min(left.lengthM, right.lengthM)
	if minLength <= 0 || maxLength/minLength > 6.0 {
		return false
	}
	return true
}

type axisTraversal struct {
	axisID    string
	isForward bool
}

type axisUsageSummary struct {
	totalTraversals      int
	uniqueAxisCount      int
	conflictingAxisCount int
	reusedTraversals     int
	maxAxisReuseCount    int
}

func evaluateAxisUsage(points [][]float64) axisUsageSummary {
	traversals := extractAxisTraversals(points)
	if len(traversals) == 0 {
		return axisUsageSummary{}
	}

	axisCounts := make(map[string]int, len(traversals))
	axisDirections := make(map[string]uint8, len(traversals))
	maxReuse := 0

	for _, traversal := range traversals {
		axisCounts[traversal.axisID]++
		if axisCounts[traversal.axisID] > maxReuse {
			maxReuse = axisCounts[traversal.axisID]
		}
		mask := axisDirections[traversal.axisID]
		if traversal.isForward {
			mask |= 0b01
		} else {
			mask |= 0b10
		}
		axisDirections[traversal.axisID] = mask
	}

	conflicting := 0
	reused := 0
	for axisID, count := range axisCounts {
		if axisDirections[axisID] == 0b11 {
			conflicting++
		}
		if count > 1 {
			reused += count - 1
		}
	}

	return axisUsageSummary{
		totalTraversals:      len(traversals),
		uniqueAxisCount:      len(axisCounts),
		conflictingAxisCount: conflicting,
		reusedTraversals:     reused,
		maxAxisReuseCount:    maxReuse,
	}
}

func extractAxisTraversals(points [][]float64) []axisTraversal {
	if len(points) < 3 {
		return []axisTraversal{}
	}

	traversals := make([]axisTraversal, 0, len(points)-1)
	for index := 0; index < len(points)-1; index++ {
		left := points[index]
		right := points[index+1]
		if len(left) < 2 || len(right) < 2 {
			continue
		}
		fromID := quantizedPointKey(left[0], left[1])
		toID := quantizedPointKey(right[0], right[1])
		if fromID == "" || toID == "" || fromID == toID {
			continue
		}
		traversals = append(traversals, axisTraversal{
			axisID:    canonicalEdgeKey(fromID, toID),
			isForward: fromID < toID,
		})
	}
	return traversals
}

func evaluateAxisReuseOutsideStartZone(
	points [][]float64,
	start routesDomain.Coordinates,
	startZoneMeters float64,
	minOppositeMeters float64,
) (bool, int, float64) {
	if len(points) < 2 {
		return false, 0, 0.0
	}

	type localAxisUsage struct {
		count         int
		directionMask uint8
		forwardMeters float64
		reverseMeters float64
	}

	axisUsage := make(map[string]localAxisUsage, len(points))
	maxReuseOutsideStart := 0
	outsideTotalMeters := 0.0

	for index := 0; index < len(points)-1; index++ {
		left := points[index]
		right := points[index+1]
		if len(left) < 2 || len(right) < 2 {
			continue
		}

		midLat := (left[0] + right[0]) / 2.0
		midLng := (left[1] + right[1]) / 2.0
		midDistance := haversineDistanceMeters(midLat, midLng, start.Lat, start.Lng)
		if midDistance <= startZoneMeters {
			// Reuse around start/finish hub is allowed.
			// Midpoint classification avoids exempting long segments that
			// cross the hub boundary and then retrace outside it.
			continue
		}

		fromID := quantizedPointKey(left[0], left[1])
		toID := quantizedPointKey(right[0], right[1])
		if fromID == "" || toID == "" || fromID == toID {
			continue
		}

		axisID := canonicalEdgeKey(fromID, toID)
		segmentMeters := haversineDistanceMeters(left[0], left[1], right[0], right[1])
		if segmentMeters < minAxisSegmentLengthM {
			continue
		}
		current := axisUsage[axisID]
		current.count++
		if fromID < toID {
			current.directionMask |= 0b01
			current.forwardMeters += segmentMeters
		} else {
			current.directionMask |= 0b10
			current.reverseMeters += segmentMeters
		}
		axisUsage[axisID] = current
		outsideTotalMeters += segmentMeters
		if current.count > maxReuseOutsideStart {
			maxReuseOutsideStart = current.count
		}
	}

	oppositeMeters := 0.0
	for _, usage := range axisUsage {
		if usage.directionMask == 0b11 {
			oppositeMeters += math.Min(usage.forwardMeters, usage.reverseMeters)
		}
	}
	if outsideTotalMeters <= 0 {
		return false, maxReuseOutsideStart, 0.0
	}
	oppositeRatio := oppositeMeters / outsideTotalMeters
	// Ignore tiny opposite-direction artifacts caused by local snap/geometry noise.
	minimum := math.Max(minOppositeReuseMeters, minOppositeMeters)
	return oppositeMeters >= minimum, maxReuseOutsideStart, clampUnit(oppositeRatio)
}

func (summary axisUsageSummary) oppositeTraversalRatio() float64 {
	if summary.totalTraversals == 0 {
		return 0.0
	}
	return float64(summary.conflictingAxisCount) / float64(summary.totalTraversals)
}

func (summary axisUsageSummary) reuseRatio() float64 {
	if summary.totalTraversals == 0 {
		return 0.0
	}
	return float64(summary.reusedTraversals) / float64(summary.totalTraversals)
}

func (summary axisUsageSummary) segmentDiversityRatio() float64 {
	if summary.totalTraversals == 0 {
		return 0.0
	}
	return float64(summary.uniqueAxisCount) / float64(summary.totalTraversals)
}

func (summary axisUsageSummary) maxAxisReuseRatio() float64 {
	if summary.totalTraversals == 0 {
		return 0.0
	}
	return float64(summary.maxAxisReuseCount) / float64(summary.totalTraversals)
}

func edgeReuseRatio(points [][]float64) float64 {
	return evaluateAxisUsage(points).reuseRatio()
}

func minSegmentDiversityRatio(routeType string) float64 {
	switch strings.ToUpper(strings.TrimSpace(routeType)) {
	case "MTB":
		return 0.55
	case "GRAVEL":
		return 0.54
	case "RUN":
		return 0.35
	case "TRAIL":
		return 0.46
	case "HIKE":
		return 0.40
	case "WALK":
		return 0.42
	default:
		return 0.32
	}
}

func segmentDiversityRatio(points [][]float64) float64 {
	return evaluateAxisUsage(points).segmentDiversityRatio()
}

func distanceShortfallRatio(distanceKm float64, targetKm float64) float64 {
	if targetKm <= 0 {
		return 0
	}
	shortfall := targetKm - distanceKm
	if shortfall <= 0 {
		return 0
	}
	return shortfall / math.Max(targetKm, 1.0)
}

func distanceOvershootRatio(distanceKm float64, targetKm float64) float64 {
	if targetKm <= 0 {
		return 0
	}
	overshoot := distanceKm - targetKm
	if overshoot <= 0 {
		return 0
	}
	return overshoot / math.Max(targetKm, 1.0)
}

func outsideStartAxisReuseLimit(routeType string, strict bool) int {
	_ = strict
	_ = routeType
	// P0-02 policy: outside start/finish zone, an axis cannot be reused.
	return 1
}

func allowedOppositeOutsideStartRatio(routeType string, strict bool) float64 {
	_ = strict
	_ = routeType
	// P0-02 policy: opposite-direction overlap is forbidden outside start zone.
	return 0.0
}

func minimumOppositeReuseMetersForRequest(routeType string, strict bool, distanceTargetKm float64) float64 {
	_ = strict
	base := math.Max(minOppositeReuseMeters, distanceTargetKm*6.0)
	switch strings.ToUpper(strings.TrimSpace(routeType)) {
	case "MTB", "TRAIL", "HIKE":
		return math.Max(base, 320.0)
	case "GRAVEL":
		return math.Max(base, 280.0)
	default:
		return math.Max(base, 240.0)
	}
}

func requiredPathRatioForRequest(routeType string, strict bool) float64 {
	normalized := strings.ToUpper(strings.TrimSpace(routeType))
	_ = strict
	if normalized != "GRAVEL" {
		return 0.0
	}
	// Gravel contract: keep a 25% path target; fallback to Ride handles impossible cases.
	return 0.25
}

func meetsMinimumDistance(distanceKm float64, targetKm float64) bool {
	if targetKm <= 0.0 {
		return true
	}
	// Keep a small tolerance for geometry simplification / snapping noise.
	toleranceKm := math.Max(0.25, targetKm*0.02)
	return distanceKm+toleranceKm >= targetKm
}

func fallbackRouteTypes(routeType string) []string {
	switch strings.ToUpper(strings.TrimSpace(routeType)) {
	case "MTB":
		return []string{"GRAVEL", "RIDE"}
	case "GRAVEL":
		return []string{"RIDE"}
	case "RIDE":
		return nil
	default:
		// Conservative default for unsupported types.
		return []string{"RIDE"}
	}
}

type osrmScoringProfile struct {
	distanceWeight  float64
	elevationWeight float64
	directionWeight float64
	diversityWeight float64
}

func osrmMatchScore(
	request application.RoutingEngineRequest,
	distanceKm float64,
	elevationGainM float64,
	points [][]float64,
) float64 {
	hasElevationTarget := request.ElevationTargetM != nil && *request.ElevationTargetM > 0
	hasDirection := strings.TrimSpace(request.StartDirection) != ""
	profile := buildOSRMScoringProfile(request.RouteType, hasElevationTarget, hasDirection)

	distanceComponent := distanceShortfallRatio(distanceKm, request.DistanceTargetKm) +
		distanceOvershootRatio(distanceKm, request.DistanceTargetKm)*0.15
	elevationComponent := 0.0
	if hasElevationTarget {
		elevationComponent = math.Abs(elevationGainM-*request.ElevationTargetM) / math.Max(*request.ElevationTargetM, 150.0)
	}
	directionComponent := 0.0
	if hasDirection {
		directionComponent = directionPenaltyFromPreview(points, request.StartDirection)
	}
	diversityComponent := 1.0 - segmentDiversityRatio(points)

	weighted := distanceComponent*profile.distanceWeight +
		elevationComponent*profile.elevationWeight +
		directionComponent*profile.directionWeight +
		diversityComponent*profile.diversityWeight

	return clampOSMScore(100.0 - weighted*100.0)
}

func buildOSRMScoringProfile(routeType string, hasElevationTarget bool, hasDirection bool) osrmScoringProfile {
	profile := osrmScoringProfile{
		distanceWeight:  0.70,
		elevationWeight: 0.22,
		directionWeight: 0.06,
		diversityWeight: 0.02,
	}

	switch strings.ToUpper(strings.TrimSpace(routeType)) {
	case "MTB":
		profile = osrmScoringProfile{distanceWeight: 0.36, elevationWeight: 0.29, directionWeight: 0.07, diversityWeight: 0.28}
	case "GRAVEL":
		profile = osrmScoringProfile{distanceWeight: 0.44, elevationWeight: 0.26, directionWeight: 0.06, diversityWeight: 0.24}
	case "RUN":
		profile = osrmScoringProfile{distanceWeight: 0.56, elevationWeight: 0.17, directionWeight: 0.13, diversityWeight: 0.14}
	case "TRAIL":
		profile = osrmScoringProfile{distanceWeight: 0.34, elevationWeight: 0.28, directionWeight: 0.10, diversityWeight: 0.28}
	case "HIKE":
		profile = osrmScoringProfile{distanceWeight: 0.30, elevationWeight: 0.35, directionWeight: 0.09, diversityWeight: 0.26}
	case "WALK":
		profile = osrmScoringProfile{distanceWeight: 0.33, elevationWeight: 0.28, directionWeight: 0.10, diversityWeight: 0.29}
	}

	if !hasElevationTarget {
		profile.distanceWeight += profile.elevationWeight * 0.70
		profile.diversityWeight += profile.elevationWeight * 0.30
		profile.elevationWeight = 0.0
	}
	if !hasDirection {
		profile.distanceWeight += profile.directionWeight * 0.60
		profile.diversityWeight += profile.directionWeight * 0.40
		profile.directionWeight = 0.0
	}

	return normalizeOSRMScoringProfile(profile)
}

func normalizeOSRMScoringProfile(profile osrmScoringProfile) osrmScoringProfile {
	total := profile.distanceWeight + profile.elevationWeight + profile.directionWeight + profile.diversityWeight
	if total <= 0 {
		return osrmScoringProfile{
			distanceWeight:  0.72,
			elevationWeight: 0.20,
			directionWeight: 0.04,
			diversityWeight: 0.04,
		}
	}
	return osrmScoringProfile{
		distanceWeight:  profile.distanceWeight / total,
		elevationWeight: profile.elevationWeight / total,
		directionWeight: profile.directionWeight / total,
		diversityWeight: profile.diversityWeight / total,
	}
}

func directionPenaltyFromPreview(points [][]float64, startDirection string) float64 {
	initialBearing, ok := initialBearingFromPreview(points)
	if !ok {
		return 1.0
	}
	targetBearing, ok := targetBearingFromDirection(startDirection)
	if !ok {
		return 0.0
	}
	diff := math.Abs(initialBearing - targetBearing)
	if diff > 180 {
		diff = 360 - diff
	}
	return diff / 180.0
}

func initialBearingFromPreview(points [][]float64) (float64, bool) {
	if len(points) < 2 {
		return 0, false
	}
	start := points[0]
	if len(start) < 2 {
		return 0, false
	}
	for index := 1; index < len(points); index++ {
		next := points[index]
		if len(next) < 2 {
			continue
		}
		if haversineDistanceMeters(start[0], start[1], next[0], next[1]) < 35.0 {
			continue
		}
		return osrmBearingDegrees(start[0], start[1], next[0], next[1]), true
	}
	last := points[len(points)-1]
	if len(last) < 2 {
		return 0, false
	}
	return osrmBearingDegrees(start[0], start[1], last[0], last[1]), true
}

func targetBearingFromDirection(direction string) (float64, bool) {
	switch strings.ToUpper(strings.TrimSpace(direction)) {
	case "N":
		return 0, true
	case "E":
		return 90, true
	case "S":
		return 180, true
	case "W":
		return 270, true
	default:
		return 0, false
	}
}

func osrmBearingDegrees(lat1, lng1, lat2, lng2 float64) float64 {
	lat1r := degreesToRadians(lat1)
	lat2r := degreesToRadians(lat2)
	deltaLng := degreesToRadians(lng2 - lng1)
	y := math.Sin(deltaLng) * math.Cos(lat2r)
	x := math.Cos(lat1r)*math.Sin(lat2r) - math.Sin(lat1r)*math.Cos(lat2r)*math.Cos(deltaLng)
	bearing := math.Atan2(y, x) * 180.0 / math.Pi
	if bearing < 0 {
		bearing += 360
	}
	return bearing
}
