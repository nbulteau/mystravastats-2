package fit

import (
	"math"
	"mystravastats/internal/shared/domain/strava"
	"slices"
)

type fitPowerMetrics struct {
	averageWatts         float64
	weightedAverageWatts int
	kilojoules           float64
	hasDeviceWatts       bool
}

func computeFITPowerMetrics(sessionAveragePower float64, stream *strava.Stream, elapsedTime int) fitPowerMetrics {
	var watts []float64
	var times []int
	if stream != nil {
		times = stream.Time.Data
		if stream.Watts != nil {
			watts = stream.Watts.Data
		}
	}
	timeline := strava.NewPowerTimeline(watts, times)
	result := fitPowerMetrics{hasDeviceWatts: sessionAveragePower > 0 || slices.ContainsFunc(watts, func(v float64) bool { return isFinite(v) && v > 0 && v < float64(fitInvalidUint16) })}
	// Session summaries are trusted; stream-derived energy uses only its actual span.
	if isFinite(sessionAveragePower) && sessionAveragePower > 0 {
		result.averageWatts = sessionAveragePower
		result.weightedAverageWatts = int(math.Round(sessionAveragePower))
		result.kilojoules = sessionAveragePower * float64(maxInt(elapsedTime, 0)) / 1000
	} else if timeline.Valid {
		duration := float64(times[len(times)-1] - times[0])
		if average := timeline.Average(float64(times[0]), float64(times[len(times)-1])); average != nil {
			result.averageWatts = *average
			result.kilojoules = *average * duration / 1000
		}
		if normalized := timeline.Normalized(); normalized != nil {
			result.weightedAverageWatts = int(math.Round(*normalized))
		}
	}
	return result
}
