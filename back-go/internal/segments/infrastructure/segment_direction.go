package infrastructure

import (
	"fmt"
	"math"
	"mystravastats/domain/strava"
	"strings"
)

type segmentDirection int

const (
	segmentDirectionUnknown segmentDirection = 0
	segmentDirectionAscent  segmentDirection = 1
	segmentDirectionDescent segmentDirection = -1
)

const (
	segmentDirectionMinAltitudeDeltaM = 3.0
	segmentDirectionMinGradePercent   = 0.5
)

var segmentDirectionLabelNormalizer = strings.NewReplacer(
	"é", "e",
	"è", "e",
	"ê", "e",
	"ë", "e",
	"à", "a",
	"â", "a",
	"ä", "a",
	"î", "i",
	"ï", "i",
	"ô", "o",
	"ö", "o",
	"ù", "u",
	"û", "u",
	"ü", "u",
	"ç", "c",
	"’", "'",
	"-", " ",
)

var ascentDirectionKeywords = []string{
	"montee",
	"ascent",
	"climb",
	"uphill",
}

var descentDirectionKeywords = []string{
	"descente",
	"descent",
	"downhill",
}

func resolveSegmentDirection(
	effort strava.SegmentEffort,
	activity *strava.Activity,
	detailedActivity *strava.DetailedActivity,
) segmentDirection {
	if direction := resolveDirectionFromAltitudeStream(detailedActivity, effort); direction != segmentDirectionUnknown {
		return direction
	}

	if direction := resolveDirectionFromLabels(activityNameOrEmpty(activity), effort.Name, effort.Segment.Name); direction != segmentDirectionUnknown {
		return direction
	}

	return resolveDirectionFromAverageGrade(effort.Segment.AverageGrade)
}

func splitAttemptsByDirection(
	attemptsByTarget map[int64][]segmentAttemptRaw,
) map[int64][]segmentAttemptRaw {
	split := make(map[int64][]segmentAttemptRaw, len(attemptsByTarget))

	for targetID, attempts := range attemptsByTarget {
		if targetID <= 0 {
			split[targetID] = append(split[targetID], attempts...)
			continue
		}

		hasAscent := false
		hasDescent := false
		for _, attempt := range attempts {
			switch attempt.direction {
			case segmentDirectionAscent:
				hasAscent = true
			case segmentDirectionDescent:
				hasDescent = true
			}
		}

		if !(hasAscent && hasDescent) {
			split[targetID] = append(split[targetID], attempts...)
			continue
		}

		for _, attempt := range attempts {
			direction := attempt.direction
			if direction == segmentDirectionUnknown {
				direction = resolveDirectionFromAverageGrade(attempt.averageGrade)
				if direction == segmentDirectionUnknown {
					direction = segmentDirectionAscent
				}
			}

			directionalTargetID := directionAwareTargetID(targetID, direction)
			attempt.targetId = directionalTargetID
			attempt.targetName = directionAwareTargetName(attempt.targetName, direction)
			split[directionalTargetID] = append(split[directionalTargetID], attempt)
		}
	}

	return split
}

func groupRawAttemptsByTarget(attempts []segmentAttemptRaw) map[int64][]segmentAttemptRaw {
	attemptsByTarget := make(map[int64][]segmentAttemptRaw)
	for _, attempt := range attempts {
		attemptsByTarget[attempt.targetId] = append(attemptsByTarget[attempt.targetId], attempt)
	}
	return attemptsByTarget
}

func resolveDirectionFromAltitudeStream(
	detailedActivity *strava.DetailedActivity,
	effort strava.SegmentEffort,
) segmentDirection {
	if detailedActivity == nil || detailedActivity.Stream == nil || detailedActivity.Stream.Altitude == nil {
		return segmentDirectionUnknown
	}
	altitudeData := detailedActivity.Stream.Altitude.Data
	if len(altitudeData) == 0 {
		return segmentDirectionUnknown
	}

	startIndex := effort.StartIndex
	endIndex := effort.EndIndex
	if startIndex < 0 || endIndex < 0 {
		return segmentDirectionUnknown
	}
	if startIndex >= len(altitudeData) || endIndex >= len(altitudeData) || startIndex == endIndex {
		return segmentDirectionUnknown
	}

	altitudeDelta := altitudeData[endIndex] - altitudeData[startIndex]
	if math.Abs(altitudeDelta) < segmentDirectionMinAltitudeDeltaM {
		return segmentDirectionUnknown
	}
	if altitudeDelta > 0 {
		return segmentDirectionAscent
	}
	return segmentDirectionDescent
}

func resolveDirectionFromLabels(labels ...string) segmentDirection {
	for _, label := range labels {
		normalized := normalizeDirectionLabel(label)
		if normalized == "" {
			continue
		}
		if hasAnyDirectionKeyword(normalized, descentDirectionKeywords) {
			return segmentDirectionDescent
		}
		if hasAnyDirectionKeyword(normalized, ascentDirectionKeywords) {
			return segmentDirectionAscent
		}
	}
	return segmentDirectionUnknown
}

func resolveDirectionFromAverageGrade(averageGrade float64) segmentDirection {
	if math.Abs(averageGrade) < segmentDirectionMinGradePercent {
		return segmentDirectionUnknown
	}
	if averageGrade > 0 {
		return segmentDirectionAscent
	}
	return segmentDirectionDescent
}

func directionAwareTargetID(targetID int64, direction segmentDirection) int64 {
	if targetID <= 0 {
		return targetID
	}
	switch direction {
	case segmentDirectionAscent:
		return -(targetID*10 + 1)
	case segmentDirectionDescent:
		return -(targetID*10 + 2)
	default:
		return targetID
	}
}

func directionAwareTargetName(targetName string, direction segmentDirection) string {
	switch direction {
	case segmentDirectionAscent:
		if strings.Contains(targetName, "(ascent)") {
			return targetName
		}
		return fmt.Sprintf("%s (ascent)", targetName)
	case segmentDirectionDescent:
		if strings.Contains(targetName, "(descent)") {
			return targetName
		}
		return fmt.Sprintf("%s (descent)", targetName)
	default:
		return targetName
	}
}

func activityNameOrEmpty(activity *strava.Activity) string {
	if activity == nil {
		return ""
	}
	return activity.Name
}

func normalizeDirectionLabel(label string) string {
	normalized := strings.ToLower(strings.TrimSpace(label))
	if normalized == "" {
		return ""
	}
	normalized = segmentDirectionLabelNormalizer.Replace(normalized)
	return strings.Join(strings.Fields(normalized), " ")
}

func hasAnyDirectionKeyword(label string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(label, keyword) {
			return true
		}
	}
	return false
}
