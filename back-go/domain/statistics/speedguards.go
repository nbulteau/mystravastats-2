package statistics

import (
	"math"

	"mystravastats/internal/shared/domain/strava"
)

const defaultPlausibleMaxSpeedMS = 35.0

func invalidSpeedSegmentPrefix(distances []float64, times []int, streamDataSize int, activityType string) []int {
	if streamDataSize < 2 {
		return make([]int, streamDataSize+1)
	}
	invalidSegments := make([]int, streamDataSize)
	threshold := plausibleSpeedThreshold(activityType)
	for i := 1; i < streamDataSize; i++ {
		invalidSegments[i] = invalidSegments[i-1]
		deltaTime := times[i] - times[i-1]
		deltaDistance := distances[i] - distances[i-1]
		if deltaTime <= 0 || deltaDistance < 0 {
			invalidSegments[i]++
			continue
		}
		speed := deltaDistance / float64(deltaTime)
		if speed > threshold {
			invalidSegments[i]++
		}
	}
	return invalidSegments
}

func hasInvalidSpeedSegment(invalidSegments []int, idxStart int, idxEnd int) bool {
	if idxStart < 0 || idxEnd <= idxStart || idxEnd >= len(invalidSegments) {
		return false
	}
	return invalidSegments[idxEnd] > invalidSegments[idxStart]
}

func IsPlausibleActivityMaxSpeed(activity *strava.Activity) bool {
	if activity == nil || math.IsNaN(activity.MaxSpeed) || math.IsInf(activity.MaxSpeed, 0) || activity.MaxSpeed <= 0 {
		return false
	}
	return activity.MaxSpeed <= plausibleSpeedThreshold(activity.Type)*1.3
}

func plausibleSpeedThreshold(activityType string) float64 {
	switch activityType {
	case "Run", "TrailRun":
		return 12
	case "Hike", "Walk":
		return 7
	case "AlpineSki":
		return 45
	default:
		return defaultPlausibleMaxSpeedMS
	}
}
