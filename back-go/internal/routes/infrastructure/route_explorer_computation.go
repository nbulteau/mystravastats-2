package infrastructure

import (
	"container/heap"
	"fmt"
	"math"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
	"mystravastats/internal/helpers"
	routesDomain "mystravastats/internal/routes/domain"
	"sort"
	"strings"
	"time"
)

const (
	defaultRouteLimit   = 6
	maxRouteLimit       = 24
	previewPointMaxSize = 120
)

type routeCandidate struct {
	activity       *strava.Activity
	date           time.Time
	activityDate   string
	distanceKm     float64
	elevationGainM float64
	durationSec    int
	isLoop         bool
	start          *routesDomain.Coordinates
	end            *routesDomain.Coordinates
	startArea      string
	season         string
	previewLatLng  [][]float64
	shape          string
	shapeScore     float64
}

type remixCandidate struct {
	remix routesDomain.ShapeRemixRecommendation
	score float64
}

type routeScoringProfile struct {
	distanceWeight   float64
	elevationWeight  float64
	durationWeight   float64
	directionWeight  float64
	startPointWeight float64
}

func computeRouteExplorerFromActivities(
	activities []*strava.Activity,
	request routesDomain.RouteExplorerRequest,
) routesDomain.RouteExplorerResult {
	candidates := buildRouteCandidates(activities)
	if len(candidates) == 0 {
		return routesDomain.RouteExplorerResult{
			ClosestLoops:   []routesDomain.RouteRecommendation{},
			Variants:       []routesDomain.RouteRecommendation{},
			Seasonal:       []routesDomain.RouteRecommendation{},
			RoadGraphLoops: []routesDomain.RouteRecommendation{},
			ShapeMatches:   []routesDomain.RouteRecommendation{},
			ShapeRemixes:   []routesDomain.ShapeRemixRecommendation{},
		}
	}

	limit := normalizeRouteLimit(request.Limit)
	distanceTargetKm := resolveDistanceTargetKm(request.DistanceTargetKm, candidates)
	elevationTargetM := resolveElevationTargetM(request.ElevationTargetM, candidates)
	durationTargetSec := resolveDurationTargetSec(request.DurationTargetMin, candidates)
	routeType := normalizeRouteType(request.RouteType)
	startDirection := normalizeStartDirection(request.StartDirection)
	preferredStart := normalizePreferredStartPoint(request.StartPoint)
	scoringProfile := buildRouteScoringProfile(routeType, startDirection, preferredStart != nil)
	seasonFilter := normalizeSeason(request.Season)
	shapeFilter := normalizeShape(request.Shape)

	baseCandidates := candidates
	if seasonFilter != "" {
		baseCandidates = filterBySeason(baseCandidates, seasonFilter)
	}
	if len(baseCandidates) == 0 {
		baseCandidates = candidates
	}

	result := routesDomain.RouteExplorerResult{
		ClosestLoops: buildClosestLoopRecommendations(
			baseCandidates,
			distanceTargetKm,
			elevationTargetM,
			durationTargetSec,
			scoringProfile,
			preferredStart,
			startDirection,
			limit,
		),
		Variants: buildSmartVariants(
			baseCandidates,
			distanceTargetKm,
			elevationTargetM,
			durationTargetSec,
			scoringProfile,
			preferredStart,
			startDirection,
		),
		Seasonal: buildSeasonalRecommendations(
			candidates,
			seasonFilter,
			distanceTargetKm,
			elevationTargetM,
			durationTargetSec,
			scoringProfile,
			preferredStart,
			startDirection,
			limit,
		),
		RoadGraphLoops: buildRoadGraphRecommendations(
			baseCandidates,
			distanceTargetKm,
			elevationTargetM,
			durationTargetSec,
			scoringProfile,
			preferredStart,
			startDirection,
			limit,
		),
		ShapeMatches: buildShapeMatchRecommendations(
			baseCandidates,
			shapeFilter,
			distanceTargetKm,
			elevationTargetM,
			durationTargetSec,
			limit,
		),
		ShapeRemixes: []routesDomain.ShapeRemixRecommendation{},
	}

	if request.IncludeRemix {
		result.ShapeRemixes = buildShapeRemixRecommendations(
			baseCandidates,
			distanceTargetKm,
			elevationTargetM,
			durationTargetSec,
			limit,
		)
	}

	return result
}

func buildRouteCandidates(activities []*strava.Activity) []routeCandidate {
	candidates := make([]routeCandidate, 0, len(activities))

	for _, activity := range activities {
		if activity == nil || activity.Distance <= 0 {
			continue
		}

		dateRaw := helpers.FirstNonEmpty(activity.StartDateLocal, activity.StartDate)
		activityDate := helpers.ExtractSortableDay(dateRaw)
		activityTime, ok := helpers.ParseActivityDate(dateRaw)
		if !ok {
			activityTime = time.Time{}
		}

		distanceKm := activity.Distance / 1000.0
		elevationGainM := math.Max(activity.TotalElevationGain, 0)
		durationSec := activity.MovingTime
		if durationSec <= 0 {
			durationSec = activity.ElapsedTime
		}
		if durationSec <= 0 {
			durationSec = int(math.Round(distanceKm * 180.0)) // fallback ~20km/h
		}

		start, end := extractActivityCoordinates(activity)
		isLoop := detectLoop(start, end, distanceKm)
		preview := buildPreviewLatLng(activity, start, end, isLoop)
		shape, shapeScore := classifyShape(preview, isLoop)

		candidates = append(candidates, routeCandidate{
			activity:       activity,
			date:           activityTime,
			activityDate:   activityDate,
			distanceKm:     distanceKm,
			elevationGainM: elevationGainM,
			durationSec:    durationSec,
			isLoop:         isLoop,
			start:          start,
			end:            end,
			startArea:      formatStartArea(start),
			season:         seasonFromDate(activityTime),
			previewLatLng:  preview,
			shape:          shape,
			shapeScore:     shapeScore,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if !left.date.Equal(right.date) {
			return left.date.After(right.date)
		}
		return left.activity.Id > right.activity.Id
	})

	return candidates
}

func buildClosestLoopRecommendations(
	candidates []routeCandidate,
	targetDistanceKm float64,
	targetElevationM float64,
	targetDurationSec int,
	scoringProfile routeScoringProfile,
	preferredStart *routesDomain.Coordinates,
	startDirection string,
	limit int,
) []routesDomain.RouteRecommendation {
	loopCandidates := make([]routeCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.isLoop {
			loopCandidates = append(loopCandidates, candidate)
		}
	}
	if len(loopCandidates) == 0 {
		loopCandidates = candidates
	}

	type scored struct {
		candidate routeCandidate
		score     float64
	}
	scoredCandidates := make([]scored, 0, len(loopCandidates))
	for _, candidate := range loopCandidates {
		score := closenessScoreWithProfile(
			candidate,
			targetDistanceKm,
			targetElevationM,
			targetDurationSec,
			scoringProfile,
			preferredStart,
			startDirection,
		)
		scoredCandidates = append(scoredCandidates, scored{candidate: candidate, score: score})
	}

	sort.Slice(scoredCandidates, func(i, j int) bool {
		if scoredCandidates[i].score == scoredCandidates[j].score {
			return scoredCandidates[i].candidate.date.After(scoredCandidates[j].candidate.date)
		}
		return scoredCandidates[i].score > scoredCandidates[j].score
	})

	result := make([]routesDomain.RouteRecommendation, 0, minInt(limit, len(scoredCandidates)))
	for _, entry := range scoredCandidates {
		if len(result) >= limit {
			break
		}

		reasons := []string{
			fmt.Sprintf("Distance delta: %s", formatDistanceDelta(entry.candidate.distanceKm-targetDistanceKm)),
			fmt.Sprintf("Elevation delta: %s", formatElevationDelta(entry.candidate.elevationGainM-targetElevationM)),
		}
		if startDirection != "" {
			reasons = append(reasons, fmt.Sprintf("Departure direction: %s", startDirectionLabel(startDirection)))
		}
		if preferredStart != nil {
			reasons = append(reasons, fmt.Sprintf("Start proximity: %s", formatDistanceDelta(startDistanceKm(entry.candidate, preferredStart))))
		}

		result = append(result, toRouteRecommendation(
			entry.candidate,
			routesDomain.RouteVariantClosest,
			entry.score,
			reasons,
			false,
		))
	}

	return dedupeRouteRecommendations(result)
}

func buildSmartVariants(
	candidates []routeCandidate,
	targetDistanceKm float64,
	targetElevationM float64,
	targetDurationSec int,
	scoringProfile routeScoringProfile,
	preferredStart *routesDomain.Coordinates,
	startDirection string,
) []routesDomain.RouteRecommendation {
	shorter := pickBestVariant(candidates, func(candidate routeCandidate) bool {
		return candidate.distanceKm < targetDistanceKm*0.95
	}, targetDistanceKm, targetElevationM, targetDurationSec, scoringProfile, preferredStart, startDirection)

	longer := pickBestVariant(candidates, func(candidate routeCandidate) bool {
		return candidate.distanceKm > targetDistanceKm*1.05
	}, targetDistanceKm, targetElevationM, targetDurationSec, scoringProfile, preferredStart, startDirection)

	hillier := pickBestVariant(candidates, func(candidate routeCandidate) bool {
		if candidate.elevationGainM < math.Max(targetElevationM+120.0, targetElevationM*1.15) {
			return false
		}
		distanceDelta := math.Abs(candidate.distanceKm - targetDistanceKm)
		return distanceDelta <= math.Max(targetDistanceKm*0.45, 15.0)
	}, targetDistanceKm, targetElevationM, targetDurationSec, scoringProfile, preferredStart, startDirection)

	result := make([]routesDomain.RouteRecommendation, 0, 3)
	if shorter != nil {
		result = append(result, toRouteRecommendation(
			shorter.candidate,
			routesDomain.RouteVariantShorter,
			shorter.score,
			[]string{
				fmt.Sprintf("About %s shorter than your target", formatDistanceDelta(targetDistanceKm-shorter.candidate.distanceKm)),
				fmt.Sprintf("Estimated duration %s", formatDuration(shorter.candidate.durationSec)),
			},
			false,
		))
	}
	if longer != nil {
		result = append(result, toRouteRecommendation(
			longer.candidate,
			routesDomain.RouteVariantLonger,
			longer.score,
			[]string{
				fmt.Sprintf("About %s longer than your target", formatDistanceDelta(longer.candidate.distanceKm-targetDistanceKm)),
				fmt.Sprintf("Good endurance extension (+%s)", formatDurationDelta(longer.candidate.durationSec-targetDurationSec)),
			},
			false,
		))
	}
	if hillier != nil {
		result = append(result, toRouteRecommendation(
			hillier.candidate,
			routesDomain.RouteVariantHillier,
			hillier.score,
			[]string{
				fmt.Sprintf("+%s elevation vs target", formatElevationDelta(hillier.candidate.elevationGainM-targetElevationM)),
				"Climbing-focused variant",
			},
			false,
		))
	}

	return dedupeRouteRecommendations(result)
}

func buildSeasonalRecommendations(
	candidates []routeCandidate,
	seasonFilter string,
	targetDistanceKm float64,
	targetElevationM float64,
	targetDurationSec int,
	scoringProfile routeScoringProfile,
	preferredStart *routesDomain.Coordinates,
	startDirection string,
	limit int,
) []routesDomain.RouteRecommendation {
	season := seasonFilter
	if season == "" {
		season = seasonFromDate(time.Now())
	}

	filtered := filterBySeason(candidates, season)
	if len(filtered) == 0 {
		return []routesDomain.RouteRecommendation{}
	}

	type scored struct {
		candidate routeCandidate
		score     float64
	}
	scoredCandidates := make([]scored, 0, len(filtered))
	for _, candidate := range filtered {
		score := closenessScoreWithProfile(
			candidate,
			targetDistanceKm,
			targetElevationM,
			targetDurationSec,
			scoringProfile,
			preferredStart,
			startDirection,
		)
		scoredCandidates = append(scoredCandidates, scored{candidate: candidate, score: score})
	}

	sort.Slice(scoredCandidates, func(i, j int) bool {
		if scoredCandidates[i].score == scoredCandidates[j].score {
			return scoredCandidates[i].candidate.date.After(scoredCandidates[j].candidate.date)
		}
		return scoredCandidates[i].score > scoredCandidates[j].score
	})

	result := make([]routesDomain.RouteRecommendation, 0, minInt(limit, len(scoredCandidates)))
	for _, entry := range scoredCandidates {
		if len(result) >= limit {
			break
		}
		result = append(result, toRouteRecommendation(
			entry.candidate,
			routesDomain.RouteVariantSeasonal,
			entry.score,
			[]string{
				fmt.Sprintf("Seasonal fit: %s", seasonLabel(season)),
				"Similar profile to your historical rides in this season",
			},
			false,
		))
	}

	return dedupeRouteRecommendations(result)
}

type roadGraphNode struct {
	id  string
	lat float64
	lng float64
}

type roadGraphEdge struct {
	to       string
	distance float64
}

type roadGraph struct {
	nodes map[string]roadGraphNode
	edges map[string][]roadGraphEdge
}

func buildRoadGraphRecommendations(
	candidates []routeCandidate,
	targetDistanceKm float64,
	targetElevationM float64,
	targetDurationSec int,
	scoringProfile routeScoringProfile,
	preferredStart *routesDomain.Coordinates,
	startDirection string,
	limit int,
) []routesDomain.RouteRecommendation {
	if len(candidates) < 2 {
		return []routesDomain.RouteRecommendation{}
	}

	loopCandidates := make([]routeCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.isLoop && len(candidate.previewLatLng) >= 8 {
			loopCandidates = append(loopCandidates, candidate)
		}
	}
	if len(loopCandidates) < 2 {
		return []routesDomain.RouteRecommendation{}
	}

	sourceLimit := minInt(len(loopCandidates), 70)
	loopCandidates = loopCandidates[:sourceLimit]

	graph := buildRoadGraph(loopCandidates)
	if len(graph.nodes) == 0 {
		return []routesDomain.RouteRecommendation{}
	}

	elevationPerKm := estimateElevationPerKm(loopCandidates)
	durationPerKm := estimateDurationPerKm(loopCandidates)

	type scoredRecommendation struct {
		recommendation routesDomain.RouteRecommendation
		score          float64
	}
	scored := make([]scoredRecommendation, 0, sourceLimit)
	seenIDs := make(map[string]struct{})

	for i := 0; i < len(loopCandidates); i++ {
		left := loopCandidates[i]
		for j := i + 1; j < len(loopCandidates); j++ {
			right := loopCandidates[j]
			recommendation, score, ok := buildRoadGraphRecommendationFromPair(
				graph,
				left,
				right,
				targetDistanceKm,
				targetElevationM,
				targetDurationSec,
				scoringProfile,
				preferredStart,
				startDirection,
				elevationPerKm,
				durationPerKm,
			)
			if !ok {
				continue
			}
			if _, exists := seenIDs[recommendation.RouteID]; exists {
				continue
			}
			seenIDs[recommendation.RouteID] = struct{}{}
			scored = append(scored, scoredRecommendation{
				recommendation: recommendation,
				score:          score,
			})
			if len(scored) >= maxRouteLimit*4 {
				break
			}
		}
		if len(scored) >= maxRouteLimit*4 {
			break
		}
	}

	if len(scored) == 0 {
		return []routesDomain.RouteRecommendation{}
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].recommendation.RouteID < scored[j].recommendation.RouteID
		}
		return scored[i].score > scored[j].score
	})

	resultLimit := minInt(limit, 6)
	result := make([]routesDomain.RouteRecommendation, 0, resultLimit)
	for _, entry := range scored {
		if len(result) >= resultLimit {
			break
		}
		result = append(result, entry.recommendation)
	}
	return dedupeRouteRecommendations(result)
}

func buildRoadGraphRecommendationFromPair(
	graph roadGraph,
	left routeCandidate,
	right routeCandidate,
	targetDistanceKm float64,
	targetElevationM float64,
	targetDurationSec int,
	scoringProfile routeScoringProfile,
	preferredStart *routesDomain.Coordinates,
	startDirection string,
	elevationPerKm float64,
	durationPerKm float64,
) (routesDomain.RouteRecommendation, float64, bool) {
	if len(left.previewLatLng) < 4 || len(right.previewLatLng) < 4 {
		return routesDomain.RouteRecommendation{}, 0, false
	}
	leftStart := left.previewLatLng[0]
	leftEnd := left.previewLatLng[len(left.previewLatLng)-1]
	rightStart := right.previewLatLng[0]
	rightEnd := right.previewLatLng[len(right.previewLatLng)-1]
	if len(leftStart) < 2 || len(leftEnd) < 2 || len(rightStart) < 2 || len(rightEnd) < 2 {
		return routesDomain.RouteRecommendation{}, 0, false
	}

	connectorA, okA := shortestGraphPath(graph, leftEnd, rightStart)
	if !okA {
		return routesDomain.RouteRecommendation{}, 0, false
	}
	connectorB, okB := shortestGraphPath(graph, rightEnd, leftStart)
	if !okB {
		return routesDomain.RouteRecommendation{}, 0, false
	}

	merged := mergePreviewLatLng(left.previewLatLng, connectorA)
	merged = mergePreviewLatLng(merged, right.previewLatLng)
	merged = mergePreviewLatLng(merged, connectorB)
	merged = sampleLatLng(merged, previewPointMaxSize)
	if len(merged) < 6 {
		return routesDomain.RouteRecommendation{}, 0, false
	}

	distanceMeters := pathDistanceMeters(merged)
	distanceKm := distanceMeters / 1000.0
	if distanceKm < 3.0 {
		return routesDomain.RouteRecommendation{}, 0, false
	}

	estimatedElevation := math.Max(0, distanceKm*elevationPerKm)
	estimatedDuration := int(math.Round(distanceKm * durationPerKm))
	synthetic := routeCandidate{
		distanceKm:     distanceKm,
		elevationGainM: estimatedElevation,
		durationSec:    estimatedDuration,
		isLoop:         true,
		start:          toCoordinates(leftStart),
		end:            toCoordinates(leftStart),
		startArea:      left.startArea,
		season:         left.season,
		previewLatLng:  merged,
		shape:          "LOOP",
		shapeScore:     0.85,
		activityDate:   left.activityDate,
		activity: &strava.Activity{
			Id:        0,
			Name:      fmt.Sprintf("Road-graph loop: %s + %s", left.activity.Name, right.activity.Name),
			Type:      left.activity.Type,
			SportType: left.activity.SportType,
			Commute:   left.activity.Commute,
		},
	}

	score := closenessScoreWithProfile(
		synthetic,
		targetDistanceKm,
		targetElevationM,
		targetDurationSec,
		scoringProfile,
		preferredStart,
		startDirection,
	)
	if score < 40.0 {
		return routesDomain.RouteRecommendation{}, 0, false
	}

	recommendation := toRouteRecommendation(
		synthetic,
		routesDomain.RouteVariantRoadGraph,
		score,
		[]string{
			"Generated on cache road-graph (beta)",
			fmt.Sprintf("Anchors: %s + %s", left.activity.Name, right.activity.Name),
			fmt.Sprintf("Estimated profile: %s / %s", formatDistanceDelta(distanceKm), formatElevationDelta(estimatedElevation)),
		},
		true,
	)
	recommendation.RouteID = fmt.Sprintf("road-graph-%d-%d", minInt64(left.activity.Id, right.activity.Id), maxInt64(left.activity.Id, right.activity.Id))
	return recommendation, score, true
}

func buildRoadGraph(candidates []routeCandidate) roadGraph {
	graph := roadGraph{
		nodes: make(map[string]roadGraphNode),
		edges: make(map[string][]roadGraphEdge),
	}

	for _, candidate := range candidates {
		if len(candidate.previewLatLng) < 2 {
			continue
		}
		for index := 0; index < len(candidate.previewLatLng)-1; index++ {
			left := candidate.previewLatLng[index]
			right := candidate.previewLatLng[index+1]
			if len(left) < 2 || len(right) < 2 {
				continue
			}
			leftID, leftNode, okLeft := quantizedRoadNode(left[0], left[1])
			rightID, rightNode, okRight := quantizedRoadNode(right[0], right[1])
			if !okLeft || !okRight || leftID == rightID {
				continue
			}
			graph.nodes[leftID] = leftNode
			graph.nodes[rightID] = rightNode
			distance := greatCircleDistanceMeters(
				&routesDomain.Coordinates{Lat: leftNode.lat, Lng: leftNode.lng},
				&routesDomain.Coordinates{Lat: rightNode.lat, Lng: rightNode.lng},
			)
			if distance <= 0 || math.IsInf(distance, 0) || math.IsNaN(distance) {
				continue
			}
			graph.edges[leftID] = append(graph.edges[leftID], roadGraphEdge{to: rightID, distance: distance})
			graph.edges[rightID] = append(graph.edges[rightID], roadGraphEdge{to: leftID, distance: distance})
		}
	}

	return graph
}

func quantizedRoadNode(lat float64, lng float64) (string, roadGraphNode, bool) {
	if !isValidCoordinate(lat, lng) {
		return "", roadGraphNode{}, false
	}
	quantizedLat := roundFloat(lat, 4)
	quantizedLng := roundFloat(lng, 4)
	id := fmt.Sprintf("%.4f,%.4f", quantizedLat, quantizedLng)
	return id, roadGraphNode{
		id:  id,
		lat: quantizedLat,
		lng: quantizedLng,
	}, true
}

func shortestGraphPath(graph roadGraph, from []float64, to []float64) ([][]float64, bool) {
	if len(from) < 2 || len(to) < 2 {
		return nil, false
	}
	startID, _, okStart := quantizedRoadNode(from[0], from[1])
	endID, _, okEnd := quantizedRoadNode(to[0], to[1])
	if !okStart || !okEnd {
		return nil, false
	}
	if startID == endID {
		startNode, exists := graph.nodes[startID]
		if !exists {
			return nil, false
		}
		return [][]float64{{startNode.lat, startNode.lng}}, true
	}
	if _, exists := graph.nodes[startID]; !exists {
		return nil, false
	}
	if _, exists := graph.nodes[endID]; !exists {
		return nil, false
	}

	distances := map[string]float64{startID: 0}
	parents := map[string]string{}
	visited := map[string]struct{}{}
	queue := &roadPathPriorityQueue{{nodeID: startID, distance: 0}}
	heap.Init(queue)

	for queue.Len() > 0 {
		current := heap.Pop(queue).(roadPathState)
		if _, seen := visited[current.nodeID]; seen {
			continue
		}
		visited[current.nodeID] = struct{}{}
		if current.nodeID == endID {
			break
		}
		for _, edge := range graph.edges[current.nodeID] {
			nextDistance := current.distance + edge.distance
			known, exists := distances[edge.to]
			if !exists || nextDistance < known {
				distances[edge.to] = nextDistance
				parents[edge.to] = current.nodeID
				heap.Push(queue, roadPathState{
					nodeID:   edge.to,
					distance: nextDistance,
				})
			}
		}
	}

	if _, exists := distances[endID]; !exists {
		return nil, false
	}

	ids := make([]string, 0, 32)
	cursor := endID
	ids = append(ids, cursor)
	for cursor != startID {
		parent, ok := parents[cursor]
		if !ok {
			return nil, false
		}
		cursor = parent
		ids = append(ids, cursor)
	}
	reverseStrings(ids)

	points := make([][]float64, 0, len(ids))
	for _, id := range ids {
		node := graph.nodes[id]
		points = append(points, []float64{node.lat, node.lng})
	}
	return points, true
}

type roadPathState struct {
	nodeID   string
	distance float64
}

type roadPathPriorityQueue []roadPathState

func (queue roadPathPriorityQueue) Len() int { return len(queue) }
func (queue roadPathPriorityQueue) Less(i, j int) bool {
	return queue[i].distance < queue[j].distance
}
func (queue roadPathPriorityQueue) Swap(i, j int) { queue[i], queue[j] = queue[j], queue[i] }
func (queue *roadPathPriorityQueue) Push(item interface{}) {
	*queue = append(*queue, item.(roadPathState))
}
func (queue *roadPathPriorityQueue) Pop() interface{} {
	old := *queue
	last := len(old) - 1
	item := old[last]
	*queue = old[:last]
	return item
}

func reverseStrings(values []string) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}

func pathDistanceMeters(points [][]float64) float64 {
	if len(points) < 2 {
		return 0
	}
	total := 0.0
	for index := 0; index < len(points)-1; index++ {
		from := toCoordinates(points[index])
		to := toCoordinates(points[index+1])
		total += greatCircleDistanceMeters(from, to)
	}
	return total
}

func estimateElevationPerKm(candidates []routeCandidate) float64 {
	values := make([]float64, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.distanceKm <= 0 {
			continue
		}
		values = append(values, candidate.elevationGainM/candidate.distanceKm)
	}
	return medianFloat(values, 12.0)
}

func estimateDurationPerKm(candidates []routeCandidate) float64 {
	values := make([]float64, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.distanceKm <= 0 || candidate.durationSec <= 0 {
			continue
		}
		values = append(values, float64(candidate.durationSec)/candidate.distanceKm)
	}
	return medianFloat(values, 190.0)
}

func minInt64(left, right int64) int64 {
	if left < right {
		return left
	}
	return right
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}

func buildShapeMatchRecommendations(
	candidates []routeCandidate,
	shapeFilter string,
	targetDistanceKm float64,
	targetElevationM float64,
	targetDurationSec int,
	limit int,
) []routesDomain.RouteRecommendation {
	if shapeFilter == "" {
		return []routesDomain.RouteRecommendation{}
	}

	type scored struct {
		candidate routeCandidate
		score     float64
	}
	scoredCandidates := make([]scored, 0, len(candidates))
	for _, candidate := range candidates {
		if !shapeMatches(candidate, shapeFilter) {
			continue
		}
		closeness := closenessScore(candidate, targetDistanceKm, targetElevationM, targetDurationSec)
		shapeScore := candidate.shapeScore * 100.0
		score := (shapeScore * 0.65) + (closeness * 0.35)
		scoredCandidates = append(scoredCandidates, scored{candidate: candidate, score: score})
	}
	if len(scoredCandidates) == 0 {
		return []routesDomain.RouteRecommendation{}
	}

	sort.Slice(scoredCandidates, func(i, j int) bool {
		if scoredCandidates[i].score == scoredCandidates[j].score {
			return scoredCandidates[i].candidate.date.After(scoredCandidates[j].candidate.date)
		}
		return scoredCandidates[i].score > scoredCandidates[j].score
	})

	result := make([]routesDomain.RouteRecommendation, 0, minInt(limit, len(scoredCandidates)))
	for _, entry := range scoredCandidates {
		if len(result) >= limit {
			break
		}
		result = append(result, toRouteRecommendation(
			entry.candidate,
			routesDomain.RouteVariantShape,
			entry.score,
			[]string{
				fmt.Sprintf("Shape match: %s", strings.ToLower(strings.ReplaceAll(shapeFilter, "_", " "))),
				fmt.Sprintf("Route geometry confidence %.0f%%", entry.candidate.shapeScore*100.0),
			},
			false,
		))
	}

	return dedupeRouteRecommendations(result)
}

func buildShapeRemixRecommendations(
	candidates []routeCandidate,
	targetDistanceKm float64,
	targetElevationM float64,
	targetDurationSec int,
	limit int,
) []routesDomain.ShapeRemixRecommendation {
	eligible := make([]routeCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.start == nil || candidate.end == nil {
			continue
		}
		if len(candidate.previewLatLng) < 2 {
			continue
		}
		eligible = append(eligible, candidate)
	}

	if len(eligible) < 2 {
		return []routesDomain.ShapeRemixRecommendation{}
	}

	sort.Slice(eligible, func(i, j int) bool {
		return eligible[i].date.After(eligible[j].date)
	})

	maxSource := minInt(len(eligible), 140)
	eligible = eligible[:maxSource]

	pairs := make([]remixCandidate, 0, maxSource)
	seen := make(map[string]struct{})
	for i := 0; i < len(eligible); i++ {
		for j := i + 1; j < len(eligible); j++ {
			left := eligible[i]
			right := eligible[j]
			if left.activity.Id == right.activity.Id {
				continue
			}

			remix, score, ok := buildRemixFromPair(left, right, targetDistanceKm, targetElevationM, targetDurationSec)
			if !ok {
				continue
			}
			if score < 40.0 {
				continue
			}
			if _, exists := seen[remix.ID]; exists {
				continue
			}
			seen[remix.ID] = struct{}{}
			pairs = append(pairs, remixCandidate{
				remix: remix,
				score: score,
			})
		}
	}

	if len(pairs) == 0 {
		return []routesDomain.ShapeRemixRecommendation{}
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].score == pairs[j].score {
			return pairs[i].remix.ID < pairs[j].remix.ID
		}
		return pairs[i].score > pairs[j].score
	})

	result := make([]routesDomain.ShapeRemixRecommendation, 0, minInt(limit, len(pairs)))
	for _, candidate := range pairs {
		if len(result) >= limit {
			break
		}
		result = append(result, candidate.remix)
	}

	return result
}

func buildRemixFromPair(
	left routeCandidate,
	right routeCandidate,
	targetDistanceKm float64,
	targetElevationM float64,
	targetDurationSec int,
) (routesDomain.ShapeRemixRecommendation, float64, bool) {
	connectorA := greatCircleDistanceMeters(left.end, right.start)
	connectorB := greatCircleDistanceMeters(right.end, left.start)
	totalConnector := connectorA + connectorB
	if totalConnector > 7000 {
		return routesDomain.ShapeRemixRecommendation{}, 0, false
	}

	distanceKm := left.distanceKm + right.distanceKm + (totalConnector/1000.0)*0.25
	elevationGainM := left.elevationGainM + right.elevationGainM
	durationSec := left.durationSec + right.durationSec + int(totalConnector/6.0)

	mergedPreview := mergePreviewLatLng(left.previewLatLng, right.previewLatLng)
	shape, shapeScore := classifyShape(mergedPreview, true)

	scoreCandidate := routeCandidate{
		distanceKm:     distanceKm,
		elevationGainM: elevationGainM,
		durationSec:    durationSec,
		shapeScore:     shapeScore,
	}
	closeness := closenessScore(scoreCandidate, targetDistanceKm, targetElevationM, targetDurationSec)
	score := (closeness * 0.7) + (shapeScore * 100.0 * 0.3)

	idLeft, idRight := left.activity.Id, right.activity.Id
	if idLeft > idRight {
		idLeft, idRight = idRight, idLeft
	}
	remixID := fmt.Sprintf("remix-%d-%d", idLeft, idRight)
	components := []business.ActivityShort{
		toActivityShort(left.activity),
		toActivityShort(right.activity),
	}

	return routesDomain.ShapeRemixRecommendation{
		ID:             remixID,
		Shape:          shape,
		DistanceKm:     roundFloat(distanceKm, 2),
		ElevationGainM: roundFloat(elevationGainM, 0),
		DurationSec:    durationSec,
		MatchScore:     roundFloat(score, 1),
		Reasons: []string{
			fmt.Sprintf("Synthetic loop from %s + %s", left.activity.Name, right.activity.Name),
			fmt.Sprintf("Connector cost: %.1f km", totalConnector/1000.0),
		},
		Components:    components,
		PreviewLatLng: mergedPreview,
		Experimental:  true,
	}, score, true
}

func pickBestVariant(
	candidates []routeCandidate,
	filter func(routeCandidate) bool,
	targetDistanceKm float64,
	targetElevationM float64,
	targetDurationSec int,
	scoringProfile routeScoringProfile,
	preferredStart *routesDomain.Coordinates,
	startDirection string,
) *struct {
	candidate routeCandidate
	score     float64
} {
	var bestCandidate *routeCandidate
	bestScore := -1.0
	for _, candidate := range candidates {
		if !filter(candidate) {
			continue
		}
		score := closenessScoreWithProfile(
			candidate,
			targetDistanceKm,
			targetElevationM,
			targetDurationSec,
			scoringProfile,
			preferredStart,
			startDirection,
		)
		if score > bestScore {
			candidateCopy := candidate
			bestCandidate = &candidateCopy
			bestScore = score
		}
	}
	if bestCandidate == nil {
		return nil
	}
	return &struct {
		candidate routeCandidate
		score     float64
	}{
		candidate: *bestCandidate,
		score:     bestScore,
	}
}

func toRouteRecommendation(
	candidate routeCandidate,
	variantType routesDomain.RouteVariantType,
	score float64,
	reasons []string,
	experimental bool,
) routesDomain.RouteRecommendation {
	shape := candidate.shape
	if shape == "" {
		shape = "UNKNOWN"
	}
	shapeScore := candidate.shapeScore * 100.0
	return routesDomain.RouteRecommendation{
		RouteID:        routeRecommendationID(candidate, variantType),
		Activity:       toActivityShort(candidate.activity),
		ActivityDate:   candidate.activityDate,
		DistanceKm:     roundFloat(candidate.distanceKm, 2),
		ElevationGainM: roundFloat(candidate.elevationGainM, 0),
		DurationSec:    candidate.durationSec,
		IsLoop:         candidate.isLoop,
		Start:          candidate.start,
		End:            candidate.end,
		StartArea:      candidate.startArea,
		Season:         candidate.season,
		VariantType:    variantType,
		MatchScore:     roundFloat(score, 1),
		Reasons:        reasons,
		PreviewLatLng:  candidate.previewLatLng,
		Shape:          &shape,
		ShapeScore:     &shapeScore,
		Experimental:   experimental,
	}
}

func routeRecommendationID(candidate routeCandidate, variantType routesDomain.RouteVariantType) string {
	if candidate.activity != nil && candidate.activity.Id > 0 {
		return fmt.Sprintf("route-%d-%s", candidate.activity.Id, strings.ToLower(string(variantType)))
	}
	if candidate.activityDate != "" {
		return fmt.Sprintf("route-%s-%s", candidate.activityDate, strings.ToLower(string(variantType)))
	}
	return fmt.Sprintf("route-%d-%s", time.Now().UnixNano(), strings.ToLower(string(variantType)))
}

func toActivityShort(activity *strava.Activity) business.ActivityShort {
	if activity == nil {
		return business.ActivityShort{}
	}
	if activity.Commute {
		return business.ActivityShort{
			Id:   activity.Id,
			Name: activity.Name,
			Type: business.Commute,
		}
	}
	if activityType, ok := business.ActivityTypes[helpers.FirstNonEmpty(activity.SportType, activity.Type)]; ok {
		return business.ActivityShort{
			Id:   activity.Id,
			Name: activity.Name,
			Type: activityType,
		}
	}
	return business.ActivityShort{
		Id:   activity.Id,
		Name: activity.Name,
		Type: business.Ride,
	}
}

func buildPreviewLatLng(
	activity *strava.Activity,
	start *routesDomain.Coordinates,
	end *routesDomain.Coordinates,
	isLoop bool,
) [][]float64 {
	if activity != nil && activity.Stream != nil && activity.Stream.LatLng != nil && len(activity.Stream.LatLng.Data) > 0 {
		return sampleLatLng(activity.Stream.LatLng.Data, previewPointMaxSize)
	}

	fallback := make([][]float64, 0, 3)
	if start != nil {
		fallback = append(fallback, []float64{start.Lat, start.Lng})
	}
	if end != nil && (start == nil || start.Lat != end.Lat || start.Lng != end.Lng) {
		fallback = append(fallback, []float64{end.Lat, end.Lng})
	}
	if isLoop && start != nil && end != nil {
		fallback = append(fallback, []float64{start.Lat, start.Lng})
	}
	return fallback
}

func extractActivityCoordinates(activity *strava.Activity) (*routesDomain.Coordinates, *routesDomain.Coordinates) {
	start := toCoordinates(activity.StartLatlng)
	var end *routesDomain.Coordinates
	if activity.Stream != nil && activity.Stream.LatLng != nil && len(activity.Stream.LatLng.Data) > 0 {
		last := activity.Stream.LatLng.Data[len(activity.Stream.LatLng.Data)-1]
		end = toCoordinates(last)
	}
	return start, end
}

func toCoordinates(values []float64) *routesDomain.Coordinates {
	if len(values) < 2 {
		return nil
	}
	lat, lng := values[0], values[1]
	if !isValidCoordinate(lat, lng) {
		return nil
	}
	return &routesDomain.Coordinates{
		Lat: lat,
		Lng: lng,
	}
}

func detectLoop(start *routesDomain.Coordinates, end *routesDomain.Coordinates, distanceKm float64) bool {
	if start == nil || end == nil {
		return false
	}
	distance := greatCircleDistanceMeters(start, end)
	threshold := math.Max(250.0, distanceKm*1000.0*0.08)
	return distance <= threshold
}

func sampleLatLng(raw [][]float64, maxPoints int) [][]float64 {
	valid := make([][]float64, 0, len(raw))
	for _, point := range raw {
		if len(point) < 2 {
			continue
		}
		lat, lng := point[0], point[1]
		if !isValidCoordinate(lat, lng) {
			continue
		}
		valid = append(valid, []float64{lat, lng})
	}
	if len(valid) <= maxPoints {
		return valid
	}
	if maxPoints <= 1 {
		return valid[:1]
	}

	sampled := make([][]float64, 0, maxPoints)
	step := float64(len(valid)-1) / float64(maxPoints-1)
	lastIndex := -1
	for i := 0; i < maxPoints; i++ {
		index := int(math.Round(float64(i) * step))
		if index >= len(valid) {
			index = len(valid) - 1
		}
		if index == lastIndex {
			continue
		}
		lastIndex = index
		sampled = append(sampled, valid[index])
	}
	return sampled
}

func classifyShape(preview [][]float64, isLoop bool) (string, float64) {
	if len(preview) < 2 {
		if isLoop {
			return "LOOP", 0.55
		}
		return "POINT_TO_POINT", 0.35
	}

	if looksLikeFigureEight(preview) {
		return "FIGURE_EIGHT", 0.84
	}
	if looksLikeOutAndBack(preview) {
		return "OUT_AND_BACK", 0.82
	}
	if isLoop {
		return "LOOP", 0.78
	}

	start := &routesDomain.Coordinates{Lat: preview[0][0], Lng: preview[0][1]}
	end := &routesDomain.Coordinates{Lat: preview[len(preview)-1][0], Lng: preview[len(preview)-1][1]}

	latDelta := end.Lat - start.Lat
	lngDelta := end.Lng - start.Lng
	if math.Abs(latDelta) > math.Abs(lngDelta)*1.35 {
		if latDelta >= 0 {
			return "NORTHBOUND", 0.68
		}
		return "SOUTHBOUND", 0.68
	}
	if math.Abs(lngDelta) > math.Abs(latDelta)*1.35 {
		if lngDelta >= 0 {
			return "EASTBOUND", 0.68
		}
		return "WESTBOUND", 0.68
	}

	return "POINT_TO_POINT", 0.62
}

func looksLikeOutAndBack(preview [][]float64) bool {
	if len(preview) < 6 {
		return false
	}
	start := &routesDomain.Coordinates{Lat: preview[0][0], Lng: preview[0][1]}
	end := &routesDomain.Coordinates{Lat: preview[len(preview)-1][0], Lng: preview[len(preview)-1][1]}
	if greatCircleDistanceMeters(start, end) > 320 {
		return false
	}

	maxDistance := 0.0
	maxIndex := 0
	for i, point := range preview {
		current := &routesDomain.Coordinates{Lat: point[0], Lng: point[1]}
		distance := greatCircleDistanceMeters(start, current)
		if distance > maxDistance {
			maxDistance = distance
			maxIndex = i
		}
	}
	if maxDistance < 900 {
		return false
	}

	progress := float64(maxIndex) / float64(len(preview)-1)
	return progress >= 0.25 && progress <= 0.75
}

func looksLikeFigureEight(preview [][]float64) bool {
	if len(preview) < 10 {
		return false
	}
	start := &routesDomain.Coordinates{Lat: preview[0][0], Lng: preview[0][1]}
	end := &routesDomain.Coordinates{Lat: preview[len(preview)-1][0], Lng: preview[len(preview)-1][1]}
	if greatCircleDistanceMeters(start, end) > 360 {
		return false
	}

	center := centroid(preview)
	mid := preview[len(preview)/2]
	midCoord := &routesDomain.Coordinates{Lat: mid[0], Lng: mid[1]}
	centerCoord := &routesDomain.Coordinates{Lat: center[0], Lng: center[1]}
	return greatCircleDistanceMeters(midCoord, centerCoord) <= 180
}

func centroid(points [][]float64) []float64 {
	if len(points) == 0 {
		return []float64{0, 0}
	}
	sumLat := 0.0
	sumLng := 0.0
	for _, point := range points {
		sumLat += point[0]
		sumLng += point[1]
	}
	return []float64{sumLat / float64(len(points)), sumLng / float64(len(points))}
}

func shapeMatches(candidate routeCandidate, filter string) bool {
	if filter == "" {
		return true
	}
	if filter == "LOOP" {
		return candidate.isLoop || candidate.shape == "LOOP"
	}
	return candidate.shape == filter
}

func filterBySeason(candidates []routeCandidate, season string) []routeCandidate {
	if season == "" {
		return candidates
	}
	filtered := make([]routeCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.season == season {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

func closenessScore(
	candidate routeCandidate,
	targetDistanceKm float64,
	targetElevationM float64,
	targetDurationSec int,
) float64 {
	return closenessScoreWithProfile(
		candidate,
		targetDistanceKm,
		targetElevationM,
		targetDurationSec,
		buildRouteScoringProfile("", "", false),
		nil,
		"",
	)
}

func closenessScoreWithProfile(
	candidate routeCandidate,
	targetDistanceKm float64,
	targetElevationM float64,
	targetDurationSec int,
	scoringProfile routeScoringProfile,
	preferredStart *routesDomain.Coordinates,
	startDirection string,
) float64 {
	scoringProfile = normalizeScoringProfile(scoringProfile)
	distanceComponent := math.Abs(candidate.distanceKm-targetDistanceKm) / math.Max(targetDistanceKm, 1.0)
	elevationComponent := math.Abs(candidate.elevationGainM-targetElevationM) / math.Max(targetElevationM, 200.0)
	durationComponent := math.Abs(float64(candidate.durationSec-targetDurationSec)) / math.Max(float64(targetDurationSec), 1800.0)
	directionComponent := directionPenaltyComponent(candidate, startDirection)
	startPointComponent := startPointPenaltyComponent(candidate, preferredStart)
	weighted := distanceComponent*scoringProfile.distanceWeight +
		elevationComponent*scoringProfile.elevationWeight +
		durationComponent*scoringProfile.durationWeight +
		directionComponent*scoringProfile.directionWeight +
		startPointComponent*scoringProfile.startPointWeight
	score := 100.0 - weighted*100.0
	return math.Max(0, score)
}

func buildRouteScoringProfile(routeType string, startDirection string, hasPreferredStart bool) routeScoringProfile {
	normalizedType := strings.ToUpper(strings.TrimSpace(routeType))
	distanceWeight := 0.52
	elevationWeight := 0.30
	durationWeight := 0.18

	switch normalizedType {
	case "MTB":
		distanceWeight = 0.44
		elevationWeight = 0.39
		durationWeight = 0.17
	case "GRAVEL":
		distanceWeight = 0.48
		elevationWeight = 0.34
		durationWeight = 0.18
	case "RUN":
		distanceWeight = 0.45
		elevationWeight = 0.22
		durationWeight = 0.33
	case "TRAIL":
		distanceWeight = 0.36
		elevationWeight = 0.40
		durationWeight = 0.24
	case "HIKE":
		distanceWeight = 0.30
		elevationWeight = 0.45
		durationWeight = 0.25
	}

	directionWeight := 0.0
	if startDirection != "" {
		switch normalizedType {
		case "MTB":
			directionWeight = 0.10
		case "GRAVEL":
			directionWeight = 0.09
		case "RUN":
			directionWeight = 0.10
		case "TRAIL":
			directionWeight = 0.12
		case "HIKE":
			directionWeight = 0.12
		default:
			directionWeight = 0.08
		}
	}

	startPointWeight := 0.0
	if hasPreferredStart {
		switch normalizedType {
		case "RUN", "TRAIL", "HIKE":
			startPointWeight = 0.22
		case "MTB", "GRAVEL":
			startPointWeight = 0.16
		default:
			startPointWeight = 0.14
		}
	}

	core := math.Max(0.05, 1.0-directionWeight-startPointWeight)
	return normalizeScoringProfile(routeScoringProfile{
		distanceWeight:   distanceWeight * core,
		elevationWeight:  elevationWeight * core,
		durationWeight:   durationWeight * core,
		directionWeight:  directionWeight,
		startPointWeight: startPointWeight,
	})
}

func normalizeScoringProfile(profile routeScoringProfile) routeScoringProfile {
	total := profile.distanceWeight + profile.elevationWeight + profile.durationWeight + profile.directionWeight + profile.startPointWeight
	if total <= 0 {
		return routeScoringProfile{
			distanceWeight:   0.5,
			elevationWeight:  0.3,
			durationWeight:   0.2,
			directionWeight:  0.0,
			startPointWeight: 0.0,
		}
	}
	return routeScoringProfile{
		distanceWeight:   profile.distanceWeight / total,
		elevationWeight:  profile.elevationWeight / total,
		durationWeight:   profile.durationWeight / total,
		directionWeight:  profile.directionWeight / total,
		startPointWeight: profile.startPointWeight / total,
	}
}

func normalizeRouteType(value *string) string {
	if value == nil {
		return ""
	}
	switch strings.ToUpper(strings.TrimSpace(*value)) {
	case "RIDE", "MTB", "GRAVEL", "RUN", "TRAIL", "HIKE":
		return strings.ToUpper(strings.TrimSpace(*value))
	default:
		return ""
	}
}

func normalizeStartDirection(value *string) string {
	if value == nil {
		return ""
	}
	switch strings.ToUpper(strings.TrimSpace(*value)) {
	case "N", "S", "E", "W":
		return strings.ToUpper(strings.TrimSpace(*value))
	default:
		return ""
	}
}

func normalizePreferredStartPoint(value *routesDomain.Coordinates) *routesDomain.Coordinates {
	if value == nil {
		return nil
	}
	if !isValidCoordinate(value.Lat, value.Lng) {
		return nil
	}
	return &routesDomain.Coordinates{
		Lat: value.Lat,
		Lng: value.Lng,
	}
}

func startDirectionLabel(value string) string {
	switch value {
	case "N":
		return "North"
	case "S":
		return "South"
	case "E":
		return "East"
	case "W":
		return "West"
	default:
		return "Any"
	}
}

func directionPenaltyComponent(candidate routeCandidate, startDirection string) float64 {
	if startDirection == "" {
		return 0.0
	}
	bearing, ok := initialBearingDegrees(candidate)
	if !ok {
		return 1.0
	}
	target := targetBearingDegrees(startDirection)
	diff := math.Abs(bearing - target)
	if diff > 180.0 {
		diff = 360.0 - diff
	}
	return diff / 180.0
}

func startPointPenaltyComponent(candidate routeCandidate, preferredStart *routesDomain.Coordinates) float64 {
	if preferredStart == nil {
		return 0.0
	}
	distanceKm := startDistanceKm(candidate, preferredStart)
	if distanceKm <= 0 {
		return 0.0
	}
	// Above 15 km from preferred start we consider the route a poor match.
	return math.Min(1.0, distanceKm/15.0)
}

func startDistanceKm(candidate routeCandidate, preferredStart *routesDomain.Coordinates) float64 {
	if preferredStart == nil || candidate.start == nil {
		return 0.0
	}
	distanceMeters := greatCircleDistanceMeters(candidate.start, preferredStart)
	return distanceMeters / 1000.0
}

func initialBearingDegrees(candidate routeCandidate) (float64, bool) {
	if len(candidate.previewLatLng) < 2 {
		return 0, false
	}
	start := candidate.previewLatLng[0]
	if len(start) < 2 {
		return 0, false
	}
	for idx := 1; idx < len(candidate.previewLatLng); idx++ {
		next := candidate.previewLatLng[idx]
		if len(next) < 2 {
			continue
		}
		if distanceBetweenPointsMeters(start[0], start[1], next[0], next[1]) < 35.0 {
			continue
		}
		return bearingDegrees(start[0], start[1], next[0], next[1]), true
	}
	last := candidate.previewLatLng[len(candidate.previewLatLng)-1]
	if len(last) < 2 {
		return 0, false
	}
	return bearingDegrees(start[0], start[1], last[0], last[1]), true
}

func targetBearingDegrees(startDirection string) float64 {
	switch startDirection {
	case "N":
		return 0.0
	case "E":
		return 90.0
	case "S":
		return 180.0
	case "W":
		return 270.0
	default:
		return 0.0
	}
}

func bearingDegrees(lat1, lng1, lat2, lng2 float64) float64 {
	lat1r := toRadians(lat1)
	lat2r := toRadians(lat2)
	deltaLng := toRadians(lng2 - lng1)
	y := math.Sin(deltaLng) * math.Cos(lat2r)
	x := math.Cos(lat1r)*math.Sin(lat2r) - math.Sin(lat1r)*math.Cos(lat2r)*math.Cos(deltaLng)
	bearing := math.Atan2(y, x) * 180.0 / math.Pi
	if bearing < 0 {
		bearing += 360.0
	}
	return bearing
}

func distanceBetweenPointsMeters(lat1, lng1, lat2, lng2 float64) float64 {
	return greatCircleDistanceMeters(
		&routesDomain.Coordinates{Lat: lat1, Lng: lng1},
		&routesDomain.Coordinates{Lat: lat2, Lng: lng2},
	)
}

func resolveDistanceTargetKm(target *float64, candidates []routeCandidate) float64 {
	if target != nil && *target > 0 {
		return *target
	}
	values := make([]float64, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.distanceKm > 0 {
			values = append(values, candidate.distanceKm)
		}
	}
	return medianFloat(values, 45.0)
}

func resolveElevationTargetM(target *float64, candidates []routeCandidate) float64 {
	if target != nil && *target > 0 {
		return *target
	}
	values := make([]float64, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.elevationGainM > 0 {
			values = append(values, candidate.elevationGainM)
		}
	}
	return medianFloat(values, 600.0)
}

func resolveDurationTargetSec(targetMin *int, candidates []routeCandidate) int {
	if targetMin != nil && *targetMin > 0 {
		return *targetMin * 60
	}
	values := make([]float64, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.durationSec > 0 {
			values = append(values, float64(candidate.durationSec))
		}
	}
	return int(math.Round(medianFloat(values, 2.5*3600)))
}

func medianFloat(values []float64, fallback float64) float64 {
	if len(values) == 0 {
		return fallback
	}
	sort.Float64s(values)
	middle := len(values) / 2
	if len(values)%2 == 0 {
		return (values[middle-1] + values[middle]) / 2.0
	}
	return values[middle]
}

func normalizeRouteLimit(limit int) int {
	if limit <= 0 {
		return defaultRouteLimit
	}
	if limit > maxRouteLimit {
		return maxRouteLimit
	}
	return limit
}

func normalizeSeason(value *string) string {
	if value == nil {
		return ""
	}
	switch strings.ToUpper(strings.TrimSpace(*value)) {
	case "WINTER":
		return "WINTER"
	case "SPRING":
		return "SPRING"
	case "SUMMER":
		return "SUMMER"
	case "AUTUMN", "FALL":
		return "AUTUMN"
	default:
		return ""
	}
}

func seasonFromDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	switch value.Month() {
	case time.December, time.January, time.February:
		return "WINTER"
	case time.March, time.April, time.May:
		return "SPRING"
	case time.June, time.July, time.August:
		return "SUMMER"
	default:
		return "AUTUMN"
	}
}

func seasonLabel(season string) string {
	switch season {
	case "WINTER":
		return "Winter"
	case "SPRING":
		return "Spring"
	case "SUMMER":
		return "Summer"
	case "AUTUMN":
		return "Autumn"
	default:
		return "All seasons"
	}
}

func normalizeShape(value *string) string {
	if value == nil {
		return ""
	}
	normalized := strings.ToUpper(strings.TrimSpace(*value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	switch normalized {
	case "LOOP", "OUT_AND_BACK", "POINT_TO_POINT", "FIGURE_EIGHT",
		"NORTHBOUND", "SOUTHBOUND", "EASTBOUND", "WESTBOUND":
		return normalized
	default:
		return ""
	}
}

func formatStartArea(start *routesDomain.Coordinates) string {
	if start == nil {
		return "Unknown start"
	}
	return fmt.Sprintf("%.2f, %.2f", start.Lat, start.Lng)
}

func formatDistanceDelta(deltaKm float64) string {
	return fmt.Sprintf("%.1f km", math.Abs(deltaKm))
}

func formatElevationDelta(deltaM float64) string {
	return fmt.Sprintf("%.0f m", math.Abs(deltaM))
}

func formatDurationDelta(deltaSec int) string {
	seconds := deltaSec
	if seconds < 0 {
		seconds = -seconds
	}
	return formatDuration(seconds)
}

func formatDuration(durationSec int) string {
	if durationSec <= 0 {
		return "0m"
	}
	hours := durationSec / 3600
	minutes := (durationSec % 3600) / 60
	if hours > 0 {
		return fmt.Sprintf("%dh%02dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func mergePreviewLatLng(left [][]float64, right [][]float64) [][]float64 {
	merged := make([][]float64, 0, len(left)+len(right))
	merged = append(merged, left...)
	if len(right) > 0 {
		if len(merged) == 0 || !samePoint(merged[len(merged)-1], right[0]) {
			merged = append(merged, right[0])
		}
		merged = append(merged, right[1:]...)
	}
	return sampleLatLng(merged, previewPointMaxSize)
}

func samePoint(left []float64, right []float64) bool {
	if len(left) < 2 || len(right) < 2 {
		return false
	}
	return left[0] == right[0] && left[1] == right[1]
}

func isValidCoordinate(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

func roundFloat(value float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(value*pow) / pow
}

func dedupeRouteRecommendations(recommendations []routesDomain.RouteRecommendation) []routesDomain.RouteRecommendation {
	if len(recommendations) == 0 {
		return []routesDomain.RouteRecommendation{}
	}

	seen := make(map[string]struct{}, len(recommendations))
	result := make([]routesDomain.RouteRecommendation, 0, len(recommendations))
	for _, recommendation := range recommendations {
		key := strings.TrimSpace(recommendation.RouteID)
		if key == "" && recommendation.Activity.Id != 0 {
			key = fmt.Sprintf("activity-%d", recommendation.Activity.Id)
		}
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, recommendation)
	}
	return result
}

func greatCircleDistanceMeters(
	left *routesDomain.Coordinates,
	right *routesDomain.Coordinates,
) float64 {
	if left == nil || right == nil {
		return math.MaxFloat64
	}
	const earthRadiusMeters = 6371000.0
	lat1 := toRadians(left.Lat)
	lat2 := toRadians(right.Lat)
	deltaLat := toRadians(right.Lat - left.Lat)
	deltaLng := toRadians(right.Lng - left.Lng)

	sinLat := math.Sin(deltaLat / 2)
	sinLng := math.Sin(deltaLng / 2)
	a := sinLat*sinLat + math.Cos(lat1)*math.Cos(lat2)*sinLng*sinLng
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusMeters * c
}

func toRadians(value float64) float64 {
	return value * math.Pi / 180.0
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
