package infrastructure

import (
	"fmt"
	"math"
	"strings"
)

func computeSurfaceBreakdown(route osrmRoute) routeSurfaceBreakdown {
	breakdown := routeSurfaceBreakdown{}
	for _, leg := range route.Legs {
		for _, step := range leg.Steps {
			distance := math.Max(0.0, step.Distance)
			if distance <= 0 {
				continue
			}
			switch classifySurfaceBucket(step) {
			case "paved":
				breakdown.pavedM += distance
			case "gravel":
				breakdown.gravelM += distance
			case "trail":
				breakdown.trailM += distance
			default:
				breakdown.unknownM += distance
			}
		}
	}
	// If no step-level data is available, keep an explicit "unknown" fallback.
	if breakdown.totalDistanceM() <= 0 && route.Distance > 0 {
		breakdown.unknownM = route.Distance
	}
	return breakdown
}

func mergeSurfaceBreakdowns(left routeSurfaceBreakdown, right routeSurfaceBreakdown) routeSurfaceBreakdown {
	return routeSurfaceBreakdown{
		pavedM:   left.pavedM + right.pavedM,
		gravelM:  left.gravelM + right.gravelM,
		trailM:   left.trailM + right.trailM,
		unknownM: left.unknownM + right.unknownM,
	}
}

func classifySurfaceBucket(step osrmStep) string {
	mode := strings.ToLower(strings.TrimSpace(step.Mode))
	if strings.Contains(mode, "pushing") || mode == "foot" || mode == "walking" {
		return "trail"
	}
	classes := make(map[string]struct{}, len(step.Classes))
	for _, rawClass := range step.Classes {
		normalized := normalizeClassToken(rawClass)
		if normalized == "" {
			continue
		}
		classes[normalized] = struct{}{}
	}
	if _, hasFerry := classes["ferry"]; hasFerry {
		return "unknown"
	}
	surfaceValue := normalizeTagValue(step.Surface, "surface")
	if surfaceValue == "" {
		surfaceValue = extractTagValueFromClasses(step.Classes, "surface")
	}
	if bucket, ok := surfaceBucketFromSurfaceTag(surfaceValue); ok {
		return bucket
	}

	trackTypeValue := normalizeTagValue(step.TrackType, "tracktype")
	if trackTypeValue == "" {
		trackTypeValue = extractTagValueFromClasses(step.Classes, "tracktype")
	}
	if bucket, ok := surfaceBucketFromTrackType(trackTypeValue); ok {
		return bucket
	}

	if hasAnyClass(classes, "path", "track", "steps", "bridleway", "cycleway_unpaved") {
		return "trail"
	}
	if hasAnyClass(
		classes,
		"tracktype_grade1", "tracktype=grade1", "tracktype:grade1",
		"grade1",
		"asphalt", "paved", "concrete", "concrete:lanes", "concrete:plates",
		"paving_stones", "sett", "cobblestone", "metal", "wood",
	) {
		return "paved"
	}
	if hasAnyClass(
		classes,
		"tracktype_grade2", "tracktype=grade2", "tracktype:grade2",
		"tracktype_grade3", "tracktype=grade3", "tracktype:grade3",
		"grade2", "grade3",
	) {
		return "gravel"
	}
	if hasAnyClass(
		classes,
		"tracktype_grade4", "tracktype=grade4", "tracktype:grade4",
		"tracktype_grade5", "tracktype=grade5", "tracktype:grade5",
		"grade4", "grade5",
	) {
		return "trail"
	}
	if hasAnyClass(classes, "unpaved", "gravel", "dirt", "ground", "earth", "compacted", "fine_gravel", "sand", "mud") {
		return "gravel"
	}
	if mode == "cycling" || mode == "driving" || mode == "running" {
		return "paved"
	}
	return "unknown"
}

func hasAnyClass(classes map[string]struct{}, keys ...string) bool {
	for _, key := range keys {
		if _, exists := classes[normalizeClassToken(key)]; exists {
			return true
		}
	}
	return false
}

func normalizeClassToken(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	normalized = strings.ReplaceAll(normalized, " ", "_")
	return normalized
}

func normalizeTagValue(raw string, key string) string {
	normalized := normalizeClassToken(raw)
	if normalized == "" {
		return ""
	}
	keyNormalized := normalizeClassToken(key)
	if keyNormalized == "" {
		return normalized
	}
	prefixes := []string{
		keyNormalized + "=",
		keyNormalized + ":",
		keyNormalized + "_",
		keyNormalized + "-",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(normalized, prefix) && len(normalized) > len(prefix) {
			return strings.Trim(normalized[len(prefix):], "_-:")
		}
	}
	return normalized
}

func extractTagValueFromClasses(rawClasses []string, key string) string {
	keyNormalized := normalizeClassToken(key)
	if keyNormalized == "" {
		return ""
	}
	prefixes := []string{
		keyNormalized + "=",
		keyNormalized + ":",
		keyNormalized + "_",
		keyNormalized + "-",
	}
	for _, rawClass := range rawClasses {
		normalized := normalizeClassToken(rawClass)
		if normalized == "" {
			continue
		}
		for _, prefix := range prefixes {
			if strings.HasPrefix(normalized, prefix) && len(normalized) > len(prefix) {
				return strings.Trim(normalized[len(prefix):], "_-:")
			}
		}
	}
	return ""
}

func surfaceBucketFromSurfaceTag(surface string) (string, bool) {
	normalized := normalizeTagValue(surface, "surface")
	switch normalized {
	case "":
		return "", false
	case "asphalt", "paved", "concrete", "concrete_lanes", "concrete_plates",
		"concrete:lanes", "concrete:plates", "paving_stones", "sett",
		"cobblestone", "metal", "wood", "chipseal":
		return "paved", true
	case "unpaved", "gravel", "fine_gravel", "compacted", "dirt",
		"ground", "earth", "pebblestone", "sand", "mud", "clay":
		return "gravel", true
	case "path", "trail", "steps", "grass", "woodchips":
		return "trail", true
	default:
		return "", false
	}
}

func surfaceBucketFromTrackType(trackType string) (string, bool) {
	normalized := normalizeTagValue(trackType, "tracktype")
	switch normalized {
	case "":
		return "", false
	case "grade1":
		return "paved", true
	case "grade2", "grade3":
		return "gravel", true
	case "grade4", "grade5":
		return "trail", true
	default:
		return "", false
	}
}

func (breakdown routeSurfaceBreakdown) totalDistanceM() float64 {
	return breakdown.pavedM + breakdown.gravelM + breakdown.trailM + breakdown.unknownM
}

func (breakdown routeSurfaceBreakdown) normalizedRatios() (float64, float64, float64, float64) {
	total := breakdown.totalDistanceM()
	if total <= 0 {
		return 0, 0, 0, 1
	}
	return breakdown.pavedM / total, breakdown.gravelM / total, breakdown.trailM / total, breakdown.unknownM / total
}

func (breakdown routeSurfaceBreakdown) pathRatio() float64 {
	_, gravel, trail, _ := breakdown.normalizedRatios()
	return clampUnit(gravel + trail)
}

func formatSurfaceBreakdown(breakdown routeSurfaceBreakdown) string {
	paved, gravel, trail, unknown := breakdown.normalizedRatios()
	return fmt.Sprintf(
		"paved %.0f%%, gravel %.0f%%, trail %.0f%%, unknown %.0f%%",
		paved*100.0,
		gravel*100.0,
		trail*100.0,
		unknown*100.0,
	)
}

func surfaceMatchScore(routeType string, breakdown routeSurfaceBreakdown) float64 {
	paved, gravel, trail, unknown := breakdown.normalizedRatios()
	pathRatio := clampUnit(gravel + trail)

	targetPaved := 0.60
	targetGravel := 0.25
	targetTrail := 0.15
	switch strings.ToUpper(strings.TrimSpace(routeType)) {
	case "RIDE":
		targetPaved, targetGravel, targetTrail = 0.92, 0.06, 0.02
	case "GRAVEL":
		// Gravel contract:
		// - minimum 25% paths (gravel + trail)
		// - no hard upper bound once this minimum is reached
		shortfall := math.Max(0.0, 0.25-pathRatio)
		pavedExcess := math.Max(0.0, paved-0.75)
		penalty := shortfall*220.0 + pavedExcess*36.0 + unknown*22.0
		return clampOSMScore(100.0 - penalty)
	case "MTB":
		// MTB should prefer paths as much as possible.
		pavedExcess := math.Max(0.0, paved-0.20)
		score := 28.0 + pathRatio*74.0 - unknown*24.0 - pavedExcess*48.0
		return clampOSMScore(score)
	case "RUN":
		targetPaved, targetGravel, targetTrail = 0.50, 0.25, 0.25
	case "TRAIL", "HIKE":
		targetPaved, targetGravel, targetTrail = 0.12, 0.28, 0.60
	}

	penalty := math.Abs(paved-targetPaved)*85.0 +
		math.Abs(gravel-targetGravel)*78.0 +
		math.Abs(trail-targetTrail)*92.0 +
		unknown*35.0
	return clampOSMScore(100.0 - penalty)
}

func surfaceScoreWeight(routeType string) float64 {
	switch strings.ToUpper(strings.TrimSpace(routeType)) {
	case "RIDE":
		return 1.10
	case "GRAVEL":
		return 1.25
	case "MTB":
		return 1.70
	case "TRAIL", "HIKE":
		return 1.40
	default:
		return 0.45
	}
}

func pathPreferenceBonus(routeType string, pathRatio float64) float64 {
	normalizedType := strings.ToUpper(strings.TrimSpace(routeType))
	switch normalizedType {
	case "RIDE":
		// Road rides should avoid off-road sections as much as possible.
		return (0.10 - pathRatio) * 35.0
	case "MTB":
		// Strongly reward path-heavy candidates for MTB.
		return (pathRatio - 0.50) * 60.0
	case "GRAVEL":
		// Encourage higher path ratio once the 25% minimum is reached.
		return (pathRatio - 0.25) * 30.0
	default:
		return 0.0
	}
}
