package statistics

import (
	"fmt"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

type BestEffortPowerStatistic struct {
	Name               string
	Activities         []strava.Activity
	Seconds            int
	BestActivityEffort *business.ActivityEffort
	Activity           *business.ActivityShort
}

func NewBestEffortPowerStatistic(name string, activities []strava.Activity, seconds int) *BestEffortPowerStatistic {
	if seconds <= 10 {
		panic("DistanceStream must be > 10 seconds")
	}

	bestActivityEffort := calculateBestPowerForTime(activities, seconds)
	var activity *business.ActivityShort
	if bestActivityEffort != nil {
		activity = &bestActivityEffort.ActivityShort
	}

	return &BestEffortPowerStatistic{
		Name:               name,
		Activities:         activities,
		Seconds:            seconds,
		BestActivityEffort: bestActivityEffort,
		Activity:           activity,
	}
}

func (b *BestEffortPowerStatistic) Value() string {
	if b.BestActivityEffort != nil && b.BestActivityEffort.AveragePower != nil {
		return fmt.Sprintf("%d W", *b.BestActivityEffort.AveragePower)
	}
	return "Not available"
}

func calculateBestPowerForTime(activities []strava.Activity, seconds int) *business.ActivityEffort {
	var bestEffort *business.ActivityEffort
	for _, activity := range activities {
		effort := BestPowerForTime(activity, seconds)
		if effort != nil && (bestEffort == nil || effort.Distance > bestEffort.Distance) {
			bestEffort = effort
		}
	}
	return bestEffort
}

func BestPowerForTime(a strava.Activity, seconds int) *business.ActivityEffort {
	if a.Stream == nil || len(a.Stream.Altitude.Data) == 0 {
		return nil
	}
	return bestPowerForTimeForTime(a.Id, a.Name, a.Type, a.Stream, seconds)
}

func bestPowerForTimeForTime(id int64, name, activityType string, stream *strava.Stream, seconds int) *business.ActivityEffort {
	altitudes := stream.Altitude
	watts := stream.Watts
	if watts == nil || len(watts.Data) == 0 {
		return nil
	}

	nonNullWatts := make([]int, len(watts.Data))
	for i, watt := range watts.Data {
		if watt == 0 {
			nonNullWatts[i] = 0
		} else {
			nonNullWatts[i] = watt
		}
	}

	idxStart, idxEnd, maxPower := 0, 0, 0
	var bestEffort *business.ActivityEffort

	distances := stream.Distance
	times := stream.Time
	streamDataSize := len(distances.Data)

	currentPower := 0

	for idxEnd < streamDataSize {
		totalDistance := distances.Data[idxEnd] - distances.Data[idxStart]
		totalAltitude := 0.0
		if len((*altitudes).Data) > 0 {
			totalAltitude = (*altitudes).Data[idxEnd] - (*altitudes).Data[idxStart]
		}

		currentPower += nonNullWatts[idxEnd]
		totalTime := times.Data[idxEnd] - times.Data[idxStart]

		if totalTime < seconds {
			idxEnd++
		} else {
			if currentPower > maxPower {
				maxPower = currentPower
				averagePower := currentPower / (idxEnd - idxStart + 1)
				bestEffort = &business.ActivityEffort{
					Distance:      totalDistance,
					Seconds:       seconds,
					DeltaAltitude: totalAltitude,
					IdxStart:      idxStart,
					IdxEnd:        idxEnd,
					AveragePower:  &averagePower,
					Label:         fmt.Sprintf("Best power for %s", formatSeconds(seconds)),
					ActivityShort: business.ActivityShort{
						Id:   id,
						Name: name,
						Type: business.ActivityTypes[activityType],
					},
				}
			}
			currentPower -= nonNullWatts[idxStart]
			idxStart++
			idxEnd++
		}
	}

	return bestEffort
}
