package infrastructure

import (
	"fmt"
	"math"
	"mystravastats/internal/routes/application"
	routesDomain "mystravastats/internal/routes/domain"
	"sort"
)

func buildRoutingHistoryBiasContext(request application.RoutingEngineRequest) routingHistoryBiasContext {
	profile := request.HistoryProfile
	if !request.HistoryBiasEnabled || profile == nil {
		return routingHistoryBiasContext{}
	}
	normalizedRouteType := normalizeHistoryRouteType(request.RouteType)
	profileRouteType := normalizeHistoryRouteType(profile.RouteType)
	if profileRouteType != normalizedRouteType {
		return routingHistoryBiasContext{}
	}
	maxAxisScore := maxPositiveMapScore(profile.AxisScores)
	maxZoneScore := maxPositiveMapScore(profile.ZoneScores)
	if maxAxisScore <= 0 && maxZoneScore <= 0 {
		return routingHistoryBiasContext{}
	}
	return routingHistoryBiasContext{
		enabled:             true,
		normalizedRouteType: normalizedRouteType,
		axisScores:          profile.AxisScores,
		zoneScores:          profile.ZoneScores,
		maxAxisScore:        maxAxisScore,
		maxZoneScore:        maxZoneScore,
	}
}

func sortAnchorsByHistoryReuse(
	anchors []routesDomain.Coordinates,
	start routesDomain.Coordinates,
	context routingHistoryBiasContext,
) []routesDomain.Coordinates {
	if !context.enabled || len(anchors) < 2 || context.maxZoneScore <= 0 {
		return anchors
	}
	type scoredAnchor struct {
		anchor routesDomain.Coordinates
		score  float64
		index  int
	}
	scoredAnchors := make([]scoredAnchor, 0, len(anchors))
	for index, anchor := range anchors {
		scoredAnchors = append(scoredAnchors, scoredAnchor{
			anchor: anchor,
			score:  historyAnchorReuseScore(anchor, start, context),
			index:  index,
		})
	}
	sort.SliceStable(scoredAnchors, func(i, j int) bool {
		left := scoredAnchors[i]
		right := scoredAnchors[j]
		if left.score != right.score {
			return left.score > right.score
		}
		return left.index < right.index
	})
	sortedAnchors := make([]routesDomain.Coordinates, 0, len(scoredAnchors))
	for _, entry := range scoredAnchors {
		sortedAnchors = append(sortedAnchors, entry.anchor)
	}
	return sortedAnchors
}

func historyAnchorReuseScore(
	anchor routesDomain.Coordinates,
	start routesDomain.Coordinates,
	context routingHistoryBiasContext,
) float64 {
	anchorZoneScore := normalizedHistoryZoneScore(anchor.Lat, anchor.Lng, context)
	midLat := (anchor.Lat + start.Lat) / 2.0
	midLng := (anchor.Lng + start.Lng) / 2.0
	midZoneScore := normalizedHistoryZoneScore(midLat, midLng, context)
	return clampUnit(anchorZoneScore*0.65 + midZoneScore*0.35)
}

func applyHistoryBiasToCandidate(
	candidate osrmRouteCandidate,
	start routesDomain.Coordinates,
	context routingHistoryBiasContext,
) osrmRouteCandidate {
	if !context.enabled {
		return candidate
	}
	corridorReuseScore := computeHistoryReuseScore(candidate.recommendation.PreviewLatLng, context)
	startZoneReuseScore := computeHistoryStartZoneReuseScore(candidate.recommendation.PreviewLatLng, start, context)
	reuseScore := clampUnit(corridorReuseScore*0.55 + startZoneReuseScore*0.45)
	candidate.historyReuseScore = reuseScore
	candidate.effectiveMatchScore = clampOSMScore(
		candidate.effectiveMatchScore +
			corridorReuseScore*historyReuseBonusWeight +
			startZoneReuseScore*historyStartZoneBonusWeight,
	)
	candidate.recommendation.Reasons = append(
		candidate.recommendation.Reasons,
		fmt.Sprintf(
			"History guidance (%s): %.0f%% corridor reuse / %.0f%% start-return reuse",
			context.normalizedRouteType,
			corridorReuseScore*100.0,
			startZoneReuseScore*100.0,
		),
	)
	return candidate
}

func computeHistoryReuseScore(points [][]float64, context routingHistoryBiasContext) float64 {
	if !context.enabled || len(points) < 2 {
		return 0.0
	}
	totalLengthM := 0.0
	axisWeighted := 0.0
	zoneWeighted := 0.0
	for index := 1; index < len(points); index++ {
		from := points[index-1]
		to := points[index]
		if len(from) < 2 || len(to) < 2 {
			continue
		}
		segmentLengthM := haversineDistanceMeters(from[0], from[1], to[0], to[1])
		if !isFinitePositive(segmentLengthM) || segmentLengthM < minHistorySegmentLengthM {
			continue
		}
		totalLengthM += segmentLengthM
		if context.maxAxisScore > 0 {
			axisID := historyAxisKey(from[0], from[1], to[0], to[1])
			axisWeighted += normalizedHistoryScore(context.axisScores[axisID], context.maxAxisScore) * segmentLengthM
		}
		if context.maxZoneScore > 0 {
			midLat := (from[0] + to[0]) / 2.0
			midLng := (from[1] + to[1]) / 2.0
			zoneID := historyZoneKey(midLat, midLng)
			zoneWeighted += normalizedHistoryScore(context.zoneScores[zoneID], context.maxZoneScore) * segmentLengthM
		}
	}
	if totalLengthM <= 0 {
		return 0.0
	}
	return blendHistoryReuseRatios(axisWeighted, zoneWeighted, totalLengthM, context)
}

func computeHistoryStartZoneReuseScore(
	points [][]float64,
	start routesDomain.Coordinates,
	context routingHistoryBiasContext,
) float64 {
	if !context.enabled || len(points) < 2 {
		return 0.0
	}
	totalLengthM := 0.0
	axisWeighted := 0.0
	zoneWeighted := 0.0
	for index := 1; index < len(points); index++ {
		from := points[index-1]
		to := points[index]
		if len(from) < 2 || len(to) < 2 {
			continue
		}
		segmentLengthM := haversineDistanceMeters(from[0], from[1], to[0], to[1])
		if !isFinitePositive(segmentLengthM) || segmentLengthM < minHistorySegmentLengthM {
			continue
		}
		midLat := (from[0] + to[0]) / 2.0
		midLng := (from[1] + to[1]) / 2.0
		if haversineDistanceMeters(midLat, midLng, start.Lat, start.Lng) > backtrackingStartZoneM {
			continue
		}
		totalLengthM += segmentLengthM
		if context.maxAxisScore > 0 {
			axisID := historyAxisKey(from[0], from[1], to[0], to[1])
			axisWeighted += normalizedHistoryScore(context.axisScores[axisID], context.maxAxisScore) * segmentLengthM
		}
		if context.maxZoneScore > 0 {
			zoneID := historyZoneKey(midLat, midLng)
			zoneWeighted += normalizedHistoryScore(context.zoneScores[zoneID], context.maxZoneScore) * segmentLengthM
		}
	}
	if totalLengthM <= 0 {
		return 0.0
	}
	return blendHistoryReuseRatios(axisWeighted, zoneWeighted, totalLengthM, context)
}

func blendHistoryReuseRatios(
	axisWeighted float64,
	zoneWeighted float64,
	totalLengthM float64,
	context routingHistoryBiasContext,
) float64 {
	hasAxisScores := context.maxAxisScore > 0 && len(context.axisScores) > 0
	hasZoneScores := context.maxZoneScore > 0 && len(context.zoneScores) > 0
	axisReuseRatio := 0.0
	zoneReuseRatio := 0.0
	if hasAxisScores {
		axisReuseRatio = axisWeighted / totalLengthM
	}
	if hasZoneScores {
		zoneReuseRatio = zoneWeighted / totalLengthM
	}
	switch {
	case hasAxisScores && hasZoneScores:
		return clampUnit(axisReuseRatio*historyAxisBiasWeight + zoneReuseRatio*historyZoneBiasWeight)
	case hasAxisScores:
		return clampUnit(axisReuseRatio)
	case hasZoneScores:
		return clampUnit(zoneReuseRatio)
	default:
		return 0.0
	}
}

func normalizedHistoryZoneScore(lat float64, lng float64, context routingHistoryBiasContext) float64 {
	if !context.enabled || context.maxZoneScore <= 0 {
		return 0.0
	}
	zoneID := historyZoneKey(lat, lng)
	return normalizedHistoryScore(context.zoneScores[zoneID], context.maxZoneScore)
}

func normalizedHistoryScore(score float64, maxScore float64) float64 {
	if !isFinitePositive(score) || !isFinitePositive(maxScore) {
		return 0.0
	}
	return clampUnit(score / maxScore)
}

func maxPositiveMapScore(scores map[string]float64) float64 {
	maxScore := 0.0
	for _, score := range scores {
		if isFinitePositive(score) && score > maxScore {
			maxScore = score
		}
	}
	return maxScore
}

func quantizedPointKey(lat float64, lng float64) string {
	return fmt.Sprintf("%.5f:%.5f", lat, lng)
}

func canonicalEdgeKey(a string, b string) string {
	if a < b {
		return a + "|" + b
	}
	return b + "|" + a
}

func haversineDistanceMeters(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
	const earthRadiusMeters = 6371000.0
	dLat := degreesToRadians(lat2 - lat1)
	dLng := degreesToRadians(lng2 - lng1)
	sinLat := math.Sin(dLat / 2.0)
	sinLng := math.Sin(dLng / 2.0)
	a := sinLat*sinLat + math.Cos(degreesToRadians(lat1))*math.Cos(degreesToRadians(lat2))*sinLng*sinLng
	c := 2.0 * math.Atan2(math.Sqrt(a), math.Sqrt(1.0-a))
	return earthRadiusMeters * c
}
