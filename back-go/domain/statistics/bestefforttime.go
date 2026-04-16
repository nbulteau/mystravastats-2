package statistics

import (
	"fmt"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
)

type BestEffortTimeStatistic struct {
	ActivityStatistic
	seconds            int
	bestActivityEffort *business.ActivityEffort
}

func NewBestEffortTimeStatistic(name string, activities []*strava.Activity, seconds int) *BestEffortTimeStatistic {

	bestActivityEffort := findBestDistanceEffortForTime(activities, seconds)
	var activity *business.ActivityShort
	if bestActivityEffort != nil {
		activity = &bestActivityEffort.ActivityShort
	}

	return &BestEffortTimeStatistic{
		ActivityStatistic: ActivityStatistic{
			BaseStatistic: BaseStatistic{name: name, Activities: activities},
			activity:      activity,
		},
		seconds:            seconds,
		bestActivityEffort: bestActivityEffort,
	}
}

func (b *BestEffortTimeStatistic) Value() string {
	if b.bestActivityEffort != nil {
		if b.bestActivityEffort.Distance > 1000 {
			return fmt.Sprintf("%.2f km => %s", b.bestActivityEffort.Distance/1000, b.bestActivityEffort.GetFormattedSpeed())
		}
		return fmt.Sprintf("%.0f m => %s", b.bestActivityEffort.Distance, b.bestActivityEffort.GetFormattedSpeed())
	}
	return "Not available"
}

func (b *BestEffortTimeStatistic) Result(bestActivityEffort *business.ActivityEffort) string {
	if bestActivityEffort.Distance > 1000 {
		return fmt.Sprintf("%.2f km => %s", bestActivityEffort.Distance/1000, bestActivityEffort.GetFormattedSpeed())
	}
	return fmt.Sprintf("%.0f m => %s", bestActivityEffort.Distance, bestActivityEffort.GetFormattedSpeed())
}

func findBestDistanceEffortForTime(activities []*strava.Activity, seconds int) *business.ActivityEffort {
	var bestEffort *business.ActivityEffort
	for _, activity := range activities {
		effort := BestDistanceEffort(*activity, seconds)
		if effort != nil && (bestEffort == nil || effort.Distance > bestEffort.Distance) {
			bestEffort = effort
		}
	}
	return bestEffort
}

func BestDistanceEffort(activity strava.Activity, seconds int) *business.ActivityEffort {
	if activity.Stream == nil || activity.Stream.Altitude == nil || len(activity.Stream.Altitude.Data) == 0 {
		return nil
	}
	return getOrComputeBestEffort(
		activity.Id,
		"best-distance-time",
		effortSecondsTarget(seconds),
		activity.Stream,
		func() *business.ActivityEffort {
			return BestDistanceForTime(activity.Id, activity.Name, activity.Type, activity.Stream, seconds)
		},
	)
}

func BestDistanceForTime(id int64, name, activityType string, stream *strava.Stream, seconds int) *business.ActivityEffort {
	var idxStart, idxEnd int
	var maxDist float64
	var bestEffort *business.ActivityEffort

	distances := stream.Distance.Data
	times := stream.Time.Data
	altitudes := stream.Altitude.Data
	if len(distances) == 0 || len(times) == 0 || len(altitudes) == 0 {
		return nil
	}

	nonNullWatts := buildNonNullWatts(stream.Watts)

	streamDataSize := len(distances)
	if len(times) < streamDataSize {
		streamDataSize = len(times)
	}
	if len(altitudes) < streamDataSize {
		streamDataSize = len(altitudes)
	}
	if streamDataSize < 2 {
		return nil
	}

	for idxEnd < streamDataSize {
		totalDistance := distances[idxEnd] - distances[idxStart]
		totalTime := times[idxEnd] - times[idxStart]
		totalAltitude := altitudes[idxEnd] - altitudes[idxStart]

		if totalTime < seconds {
			idxEnd++
		} else {
			if totalTime <= 0 {
				idxStart++
				continue
			}
			estimatedDistanceForTime := totalDistance / float64(totalTime) * float64(seconds)
			if estimatedDistanceForTime > maxDist {
				maxDist = estimatedDistanceForTime
				averagePower := averagePower(nonNullWatts, idxStart, idxEnd)

				bestEffort = &business.ActivityEffort{
					Distance:      maxDist,
					Seconds:       seconds,
					DeltaAltitude: totalAltitude,
					IdxStart:      idxStart,
					IdxEnd:        idxEnd,
					AveragePower:  averagePower,
					Label:         fmt.Sprintf("Best distance for %s", formatSeconds(seconds)),
					ActivityShort: business.ActivityShort{
						Id:   id,
						Name: name,
						Type: business.ActivityTypes[activityType],
					},
				}
			}
			idxStart++
		}
	}

	return bestEffort
}

func buildNonNullWatts(watts *strava.PowerStream) []float64 {
	var nonNullWatts []float64
	if watts != nil && len(watts.Data) > 0 {
		nonNullWatts = make([]float64, len(watts.Data))
		for i, watt := range watts.Data {
			if watt == 0 {
				nonNullWatts[i] = 0
			} else {
				nonNullWatts[i] = watt
			}
		}
	}
	return nonNullWatts
}
