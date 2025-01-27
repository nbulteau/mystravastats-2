package statistics

import (
	"fmt"
	"math"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

type BestEffortDistanceStatistic struct {
	Name               string
	Activities         []strava.Activity
	Distance           float64
	BestActivityEffort *business.ActivityEffort
}

func NewBestEffortDistanceStatistic(name string, activities []strava.Activity, distance float64) *BestEffortDistanceStatistic {
	if distance <= 100 {
		panic("DistanceStream must be > 100 meters")
	}

	bestActivityEffort := findBestActivityEffort(activities, distance)

	return &BestEffortDistanceStatistic{
		Name:               name,
		Activities:         activities,
		Distance:           distance,
		BestActivityEffort: bestActivityEffort,
	}
}

func findBestActivityEffort(activities []strava.Activity, distance float64) *business.ActivityEffort {
	var bestEffort *business.ActivityEffort
	for _, activity := range activities {
		effort := BestActivityEffort(activity, distance)
		if effort != nil && (bestEffort == nil || effort.Seconds < bestEffort.Seconds) {
			bestEffort = effort
		}
	}
	return bestEffort
}

func BestActivityEffort(activity strava.Activity, distance float64) *business.ActivityEffort {
	if activity.Stream == nil || len(activity.Stream.Altitude.Data) == 0 {
		return nil
	}
	return bestActivityEffortForDistance(activity.Id, activity.Name, activity.Type, *activity.Stream, distance)
}

func bestActivityEffortForDistance(id int64, name, activityType string, stream strava.Stream, distance float64) *business.ActivityEffort {
	idxStart, idxEnd := 0, 0
	bestTime := math.MaxFloat64
	var bestEffort *business.ActivityEffort

	distances := stream.Distance.Data
	times := stream.Time
	altitudes := stream.Altitude
	nonNullWatts := stream.Watts

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
				averagePower := average(nonNullWatts.Data[idxStart : idxEnd+1])
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

func (stat *BestEffortDistanceStatistic) Value() string {
	if stat.BestActivityEffort != nil {
		return fmt.Sprintf("%s => %s", formatSeconds(stat.BestActivityEffort.Seconds), stat.BestActivityEffort.GetFormattedSpeed())
	}
	return "Not available"
}
