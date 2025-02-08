package statistics

import (
	"fmt"
	"math"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

type BestElevationDistanceStatistic struct {
	name               string
	Activities         []*strava.Activity
	Distance           float64
	BestActivityEffort *business.ActivityEffort
}

func (stat *BestElevationDistanceStatistic) Value() string {
	if stat.BestActivityEffort != nil {
		return fmt.Sprintf("%s => %s", formatSeconds(stat.BestActivityEffort.Seconds), stat.BestActivityEffort.GetFormattedGradient())
	}
	return "Not available"
}

func (stat *BestElevationDistanceStatistic) Label() string {
	return stat.name
}

func (stat *BestElevationDistanceStatistic) Activity() *business.ActivityShort {
	if stat.BestActivityEffort != nil {
		return &stat.BestActivityEffort.ActivityShort
	}
	return nil
}

func NewBestElevationDistanceStatistic(name string, activities []*strava.Activity, distance float64) *BestElevationDistanceStatistic {
	if distance <= 100 {
		panic("DistanceStream must be > 100 meters")
	}

	bestActivityEffort := findBestElevationEffortForDistance(activities, distance)

	return &BestElevationDistanceStatistic{
		name:               name,
		Activities:         activities,
		Distance:           distance,
		BestActivityEffort: bestActivityEffort,
	}
}

func findBestElevationEffortForDistance(activities []*strava.Activity, distance float64) *business.ActivityEffort {
	var bestEffort *business.ActivityEffort
	for _, activity := range activities {
		effort := BestElevationEffort(*activity, distance)
		if effort != nil && (bestEffort == nil || effort.DeltaAltitude > bestEffort.DeltaAltitude) {
			bestEffort = effort
		}
	}
	return bestEffort
}

func BestElevationEffort(activity strava.Activity, distance float64) *business.ActivityEffort {
	if activity.Stream == nil || len(activity.Stream.Altitude.Data) == 0 {
		return nil
	}
	return BestElevationForDistance(activity.Id, activity.Name, activity.Type, activity.Stream, distance)
}

func BestElevationForDistance(id int64, name, activityType string, stream *strava.Stream, distance float64) *business.ActivityEffort {
	idxStart := 0
	idxEnd := 0
	bestElevation := -math.MaxFloat64
	var bestEffort *business.ActivityEffort

	distances := stream.Distance
	times := stream.Time
	altitudes := stream.Altitude

	nonNullWatts := buildNonNullWatts(stream.Watts)

	streamDataSize := len(distances.Data)

	for idxEnd < streamDataSize {
		totalDistance := distances.Data[idxEnd] - distances.Data[idxStart]
		totalAltitude := altitudes.Data[idxEnd] - altitudes.Data[idxStart]
		totalTime := times.Data[idxEnd] - times.Data[idxStart]

		if totalDistance < distance-0.5 {
			idxEnd++
		} else {
			if totalAltitude > bestElevation {
				bestElevation = totalAltitude
				averagePower := averagePower(nonNullWatts, idxStart, idxEnd)
				bestEffort = &business.ActivityEffort{
					Distance:      distance,
					Seconds:       totalTime,
					DeltaAltitude: bestElevation,
					IdxStart:      idxStart,
					IdxEnd:        idxEnd,
					AveragePower:  &averagePower,
					Label:         "Best gradient for " + fmt.Sprintf("%dm", int(distance)),
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
