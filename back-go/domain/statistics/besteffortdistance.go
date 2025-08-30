package statistics

import (
	"fmt"
	"math"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

type BestEffortDistanceStatistic struct {
	name               string
	Activities         []*strava.Activity
	Distance           float64
	BestActivityEffort *business.ActivityEffort
}

func (stat *BestEffortDistanceStatistic) Value() string {
	if stat.BestActivityEffort != nil {
		return fmt.Sprintf("%s => %s", formatSeconds(stat.BestActivityEffort.Seconds), stat.BestActivityEffort.GetFormattedSpeed())
	}
	return "Not available"
}

func (stat *BestEffortDistanceStatistic) Label() string {
	return stat.name
}

func (stat *BestEffortDistanceStatistic) Activity() *business.ActivityShort {
	if stat.BestActivityEffort != nil {
		return &stat.BestActivityEffort.ActivityShort
	}
	return nil
}

func NewBestEffortDistanceStatistic(name string, activities []*strava.Activity, distance float64) *BestEffortDistanceStatistic {
	if distance <= 100 {
		panic("DistanceStream must be > 100 meters")
	}

	bestActivityEffort := FindBestActivityEffort(activities, distance)

	return &BestEffortDistanceStatistic{
		name:               name,
		Activities:         activities,
		Distance:           distance,
		BestActivityEffort: bestActivityEffort,
	}
}

func FindBestActivityEffort(activities []*strava.Activity, distance float64) *business.ActivityEffort {
	var bestEffort *business.ActivityEffort
	for _, activity := range activities {
		effort := BestTimeEffort(*activity, distance)
		if effort != nil && (bestEffort == nil || effort.Seconds < bestEffort.Seconds) {
			bestEffort = effort
		}
	}
	return bestEffort
}

func BestTimeEffort(activity strava.Activity, distance float64) *business.ActivityEffort {
	if activity.Stream == nil || activity.Stream.Altitude == nil || len(activity.Stream.Altitude.Data) == 0 {
		return nil
	}

	return BestTimeForDistance(activity.Id, activity.Name, activity.Type, activity.Stream, distance)
}

func BestTimeForDistance(id int64, name, activityType string, stream *strava.Stream, distance float64) *business.ActivityEffort {
	idxStart, idxEnd := 0, 0
	bestTime := math.MaxFloat64
	var bestEffort *business.ActivityEffort

	distances := stream.Distance.Data
	times := stream.Time
	altitudes := stream.Altitude

	nonNullWatts := buildNonNullWatts(stream.Watts)

	streamDataSize := len(distances)

	for idxEnd < streamDataSize {
		totalDistance := distances[idxEnd] - distances[idxStart]
		totalTime := times.Data[idxEnd] - times.Data[idxStart]
		totalAltitude := 0.0
		if idxEnd < len(altitudes.Data) && idxStart < len(altitudes.Data) {
			totalAltitude = altitudes.Data[idxEnd] - altitudes.Data[idxStart]
		}

		if totalDistance < distance-0.5 {
			idxEnd++
		} else {
			estimatedTimeForDistance := distance / totalDistance * float64(totalTime)
			if estimatedTimeForDistance < bestTime && estimatedTimeForDistance > 1 {
				bestTime = estimatedTimeForDistance
				averagePower := averagePower(nonNullWatts, idxStart, idxEnd)
				bestEffort = &business.ActivityEffort{
					Distance:      distance,
					Seconds:       int(bestTime),
					DeltaAltitude: totalAltitude,
					IdxStart:      idxStart,
					IdxEnd:        idxEnd,
					AveragePower:  averagePower,
					Label:         fmt.Sprintf("Best speed for %.0fm", distance),
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
