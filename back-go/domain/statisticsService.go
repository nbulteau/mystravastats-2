package domain

import (
	"fmt"
	"log"
	"mystravastats/domain/business"
	"mystravastats/domain/statistics"
	"mystravastats/domain/strava"
)

func FetchStatisticsByActivityTypeAndYear(activityType business.ActivityType, year *int) []statistics.Statistic {
	log.Printf("Compute statistics for %s for %v", activityType, &year)

	filteredActivities := activityProvider.GetActivitiesByActivityTypeAndYear(activityType, year)

	switch activityType {
	case business.Ride:
		return computeRideStatistics(filteredActivities)
	case business.RideWithCommute:
		return computeRideStatistics(filteredActivities)
	case business.VirtualRide:
		return computeVirtualRideStatistics(filteredActivities)
	case business.Commute:
		return computeCommuteStatistics(filteredActivities)
	case business.Run:
		return computeRunStatistics(filteredActivities)
	case business.InlineSkate:
		return computeInlineSkateStatistics(filteredActivities)
	case business.Hike:
		return computeHikeStatistics(filteredActivities)
	case business.AlpineSki:
		return computeAlpineSkiStatistics(filteredActivities)
	default:
		return nil
	}
}

func computeRunStatistics(runActivities []*strava.Activity) []statistics.Statistic {
	allStatistics := computeCommonStats(runActivities)

	allStatistics = append(allStatistics, []statistics.Statistic{
		statistics.NewCooperStatistic(runActivities),
		//statistics.NewVVO2maxStatistic(runActivities),
		statistics.NewBestEffortDistanceStatistic("Best 200 m", runActivities, 200.0),
		statistics.NewBestEffortDistanceStatistic("Best 400 m", runActivities, 400.0),
		statistics.NewBestEffortDistanceStatistic("Best 1000 m", runActivities, 1000.0),
		statistics.NewBestEffortDistanceStatistic("Best 5000 m", runActivities, 5000.0),
		statistics.NewBestEffortDistanceStatistic("Best 10000 m", runActivities, 10000.0),
		statistics.NewBestEffortDistanceStatistic("Best half Marathon", runActivities, 21097.0),
		statistics.NewBestEffortDistanceStatistic("Best Marathon", runActivities, 42195.0),
		statistics.NewBestEffortTimeStatistic("Best 1 h", runActivities, 60*60),
		statistics.NewBestEffortTimeStatistic("Best 2 h", runActivities, 2*60*60),
		statistics.NewBestEffortTimeStatistic("Best 3 h", runActivities, 3*60*60),
		statistics.NewBestEffortTimeStatistic("Best 4 h", runActivities, 4*60*60),
		statistics.NewBestEffortTimeStatistic("Best 5 h", runActivities, 5*60*60),
		statistics.NewBestEffortTimeStatistic("Best 6 h", runActivities, 6*60*60),
	}...)

	return allStatistics
}

func computeRideStatistics(rideActivities []*strava.Activity) []statistics.Statistic {
	allStatistics := computeCommonStats(rideActivities)

	allStatistics = append(allStatistics, []statistics.Statistic{
		//statistics.NewMaxSpeedStatistic(rideActivities),
		//statistics.NewMaxMovingTimeStatistic(rideActivities),
		statistics.NewBestEffortDistanceStatistic("Best 250 m", rideActivities, 250.0),
		statistics.NewBestEffortDistanceStatistic("Best 500 m", rideActivities, 500.0),
		statistics.NewBestEffortDistanceStatistic("Best 1000 m", rideActivities, 1000.0),
		statistics.NewBestEffortDistanceStatistic("Best 5 km", rideActivities, 5000.0),
		statistics.NewBestEffortDistanceStatistic("Best 10 km", rideActivities, 10000.0),
		statistics.NewBestEffortDistanceStatistic("Best 20 km", rideActivities, 20000.0),
		statistics.NewBestEffortDistanceStatistic("Best 50 km", rideActivities, 50000.0),
		statistics.NewBestEffortDistanceStatistic("Best 100 km", rideActivities, 100000.0),
		statistics.NewBestEffortTimeStatistic("Best 30 min", rideActivities, 30*60),
		statistics.NewBestEffortTimeStatistic("Best 1 h", rideActivities, 60*60),
		statistics.NewBestEffortTimeStatistic("Best 2 h", rideActivities, 2*60*60),
		statistics.NewBestEffortTimeStatistic("Best 3 h", rideActivities, 3*60*60),
		statistics.NewBestEffortTimeStatistic("Best 4 h", rideActivities, 4*60*60),
		statistics.NewBestEffortTimeStatistic("Best 5 h", rideActivities, 5*60*60),
		statistics.NewBestElevationDistanceStatistic("Max gradient for 250 m", rideActivities, 250.0),
		statistics.NewBestElevationDistanceStatistic("Max gradient for 500 m", rideActivities, 500.0),
		statistics.NewBestElevationDistanceStatistic("Max gradient for 1000 m", rideActivities, 1000.0),
		statistics.NewBestElevationDistanceStatistic("Max gradient for 5 km", rideActivities, 5000.0),
		statistics.NewBestElevationDistanceStatistic("Max gradient for 10 km", rideActivities, 10000.0),
		statistics.NewBestElevationDistanceStatistic("Max gradient for 20 km", rideActivities, 20000.0),
	}...)

	return allStatistics
}

func computeVirtualRideStatistics(rideActivities []*strava.Activity) []statistics.Statistic {
	allStatistics := computeCommonStats(rideActivities)

	allStatistics = append(allStatistics, []statistics.Statistic{
		//statistics.NewMaxSpeedStatistic(rideActivities),
		//statistics.NewMaxMovingTimeStatistic(rideActivities),
		//statistics.NewMaxAveragePowerStatistic(rideActivities),
		//statistics.NewMaxWeightedAveragePowerStatistic(rideActivities),
		statistics.NewBestEffortDistanceStatistic("Best 250 m", rideActivities, 250.0),
		statistics.NewBestEffortDistanceStatistic("Best 500 m", rideActivities, 500.0),
		statistics.NewBestEffortDistanceStatistic("Best 1000 m", rideActivities, 1000.0),
		statistics.NewBestEffortDistanceStatistic("Best 5 km", rideActivities, 5000.0),
		statistics.NewBestEffortDistanceStatistic("Best 10 km", rideActivities, 10000.0),
		statistics.NewBestEffortDistanceStatistic("Best 20 km", rideActivities, 20000.0),
		statistics.NewBestEffortDistanceStatistic("Best 50 km", rideActivities, 50000.0),
		statistics.NewBestEffortDistanceStatistic("Best 100 km", rideActivities, 100000.0),
		statistics.NewBestEffortTimeStatistic("Best 30 min", rideActivities, 30*60),
		statistics.NewBestEffortTimeStatistic("Best 1 h", rideActivities, 60*60),
		statistics.NewBestEffortTimeStatistic("Best 2 h", rideActivities, 2*60*60),
		statistics.NewBestEffortTimeStatistic("Best 3 h", rideActivities, 3*60*60),
		statistics.NewBestEffortTimeStatistic("Best 4 h", rideActivities, 4*60*60),
		statistics.NewBestEffortPowerStatistic("Best average power for 20 min", rideActivities, 20*60),
		statistics.NewBestEffortPowerStatistic("Best average power for 1 h", rideActivities, 60*60),
	}...)

	return allStatistics
}

func computeAlpineSkiStatistics(filteredActivities []*strava.Activity) []statistics.Statistic {
	allStatistics := computeCommonStats(filteredActivities)

	allStatistics = append(allStatistics, []statistics.Statistic{
		//statistics.NewMaxSpeedStatistic(filteredActivities),
		//statistics.NewMaxMovingTimeStatistic(filteredActivities),
		statistics.NewBestEffortDistanceStatistic("Best 250 m", filteredActivities, 250.0),
		statistics.NewBestEffortDistanceStatistic("Best 500 m", filteredActivities, 500.0),
		statistics.NewBestEffortDistanceStatistic("Best 1000 m", filteredActivities, 1000.0),
		statistics.NewBestEffortDistanceStatistic("Best 5 km", filteredActivities, 5000.0),
		statistics.NewBestEffortDistanceStatistic("Best 10 km", filteredActivities, 10000.0),
		statistics.NewBestEffortDistanceStatistic("Best 20 km", filteredActivities, 20000.0),
		statistics.NewBestEffortDistanceStatistic("Best 50 km", filteredActivities, 50000.0),
		statistics.NewBestEffortDistanceStatistic("Best 100 km", filteredActivities, 100000.0),
		statistics.NewBestEffortTimeStatistic("Best 30 min", filteredActivities, 30*60),
		statistics.NewBestEffortTimeStatistic("Best 1 h", filteredActivities, 60*60),
		statistics.NewBestEffortTimeStatistic("Best 2 h", filteredActivities, 2*60*60),
		statistics.NewBestEffortTimeStatistic("Best 3 h", filteredActivities, 3*60*60),
		statistics.NewBestEffortTimeStatistic("Best 4 h", filteredActivities, 4*60*60),
		statistics.NewBestEffortTimeStatistic("Best 5 h", filteredActivities, 5*60*60),
	}...)

	return allStatistics
}

func computeCommuteStatistics(commuteActivities []*strava.Activity) []statistics.Statistic {
	allStatistics := computeCommonStats(commuteActivities)

	allStatistics = append(allStatistics, []statistics.Statistic{
		//statistics.NewMaxSpeedStatistic(commuteActivities),
		//statistics.NewMaxMovingTimeStatistic(commuteActivities),
		statistics.NewBestEffortDistanceStatistic("Best 250 m", commuteActivities, 250.0),
		statistics.NewBestEffortDistanceStatistic("Best 500 m", commuteActivities, 500.0),
		statistics.NewBestEffortDistanceStatistic("Best 1000 m", commuteActivities, 1000.0),
		statistics.NewBestEffortDistanceStatistic("Best 5 km", commuteActivities, 5000.0),
		statistics.NewBestEffortDistanceStatistic("Best 10 km", commuteActivities, 10000.0),
		statistics.NewBestEffortTimeStatistic("Best 30 min", commuteActivities, 30*60),
		statistics.NewBestEffortTimeStatistic("Best 1 h", commuteActivities, 60*60),
		statistics.NewBestElevationDistanceStatistic("Max gradient for 250 m", commuteActivities, 250.0),
		statistics.NewBestElevationDistanceStatistic("Max gradient for 500 m", commuteActivities, 500.0),
		statistics.NewBestElevationDistanceStatistic("Max gradient for 1000 m", commuteActivities, 1000.0),
	}...)

	return allStatistics
}

func computeHikeStatistics(hikeActivities []*strava.Activity) []statistics.Statistic {
	allStatistics := computeCommonStats(hikeActivities)

	/*
		statistics.NewBestDayStatistic("Max elevation in a day", hikeActivities, "%s => %.02f m", func() (string, float64) {
			maxElevation := 0.0
			allStatistics = append(allStatistics, []statistics.Statistic{
				statistics.NewBestDayStatistic("Max distance in a day", hikeActivities, "%s => %.02f km", func() (string, float64) {
					maxDistance := 0.0
					var maxDate string
					activityMap := make(map[string]float64)
					for _, activity := range hikeActivities {
						date := activity.StartDateLocal[:10]
						activityMap[date] += activity.Distance / 1000
					}
					for date, distance := range activityMap {
						if distance > maxDistance {
							maxDistance = distance
							maxDate = date
						}
					}
					return maxDate, maxDistance
				}),
				var maxDate string
				activityMap := make(map[string]float64)
				for _, activity := range hikeActivities {
				date := activity.StartDateLocal[:10]
				activityMap[date] += activity.TotalElevationGain
			}
				for date, elevation := range activityMap {
				if elevation > maxElevation {
				maxElevation = elevation
				maxDate = date
			}
			}
				return maxDate, maxElevation
			}),
		}...)
	*/
	return allStatistics
}

func computeInlineSkateStatistics(inlineSkateActivities []*strava.Activity) []statistics.Statistic {
	allStatistics := computeCommonStats(inlineSkateActivities)

	allStatistics = append(allStatistics, []statistics.Statistic{
		statistics.NewBestEffortDistanceStatistic("Best 200 m", inlineSkateActivities, 200.0),
		statistics.NewBestEffortDistanceStatistic("Best 400 m", inlineSkateActivities, 400.0),
		statistics.NewBestEffortDistanceStatistic("Best 1000 m", inlineSkateActivities, 1000.0),
		statistics.NewBestEffortDistanceStatistic("Best 10000 m", inlineSkateActivities, 10000.0),
		statistics.NewBestEffortDistanceStatistic("Best half Marathon", inlineSkateActivities, 21097.0),
		statistics.NewBestEffortDistanceStatistic("Best Marathon", inlineSkateActivities, 42195.0),
		statistics.NewBestEffortTimeStatistic("Best 1 h", inlineSkateActivities, 60*60),
		statistics.NewBestEffortTimeStatistic("Best 2 h", inlineSkateActivities, 2*60*60),
		statistics.NewBestEffortTimeStatistic("Best 3 h", inlineSkateActivities, 3*60*60),
		statistics.NewBestEffortTimeStatistic("Best 4 h", inlineSkateActivities, 4*60*60),
	}...)

	return allStatistics
}

func computeCommonStats(activities []*strava.Activity) []statistics.Statistic {
	return []statistics.Statistic{
		statistics.NewGlobalStatistic("Nb activities", activities, func(activities []strava.Activity) string {
			return fmt.Sprintf("%d", len(activities))
		}),
		statistics.NewGlobalStatistic("Nb actives days", activities, func(activities []strava.Activity) string {
			activeDays := make(map[string]struct{})
			for _, activity := range activities {
				date := activity.StartDateLocal[:10]
				activeDays[date] = struct{}{}
			}
			return fmt.Sprintf("%d", len(activeDays))
		}),
		//statistics.NewMaxStreakStatistic(activities),
		statistics.NewGlobalStatistic("Total distance", activities, func(activities []strava.Activity) string {
			totalDistance := 0.0
			for _, activity := range activities {
				totalDistance += activity.Distance
			}

			return fmt.Sprintf("%.2f km", totalDistance/1000)
		}),
		statistics.NewGlobalStatistic("Total elevation", activities, func(activities []strava.Activity) string {
			totalElevation := 0.0
			for _, activity := range activities {
				totalElevation += activity.TotalElevationGain
			}
			return fmt.Sprintf("%.2f m", totalElevation)
		}),
		statistics.NewGlobalStatistic("Km by activity", activities, func(activities []strava.Activity) string {
			totalDistance := 0.0
			for _, activity := range activities {
				totalDistance += activity.Distance
			}
			return fmt.Sprintf("%.2f km", totalDistance/float64(len(activities))/1000)
		}),
		//statistics.NewMaxDistanceStatistic(activities),
		//statistics.NewMaxDistanceInADayStatistic(activities),
		//statistics.NewMaxElevationStatistic(activities),
		//statistics.NewMaxElevationInADayStatistic(activities),
		//statistics.NewHighestPointStatistic(activities),
		//statistics.NewMaxMovingTimeStatistic(activities),
		//statistics.NewMostActiveMonthStatistic(activities),
		//statistics.NewEddingtonStatistic(activities),
	}
}
