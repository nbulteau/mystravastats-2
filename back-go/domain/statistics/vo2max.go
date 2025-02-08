package statistics

import (
	"fmt"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

type VO2maxStatistic struct {
	BestEffortTimeStatistic
}

func NewVO2maxStatistic(activities []*strava.Activity) *VO2maxStatistic {
	return &VO2maxStatistic{
		BestEffortTimeStatistic: *NewBestEffortTimeStatistic("Best VO2max (6 min)", activities, 6*60),
	}
}

func (stat *VO2maxStatistic) Result(bestActivityEffort *business.ActivityEffort) string {
	return fmt.Sprintf("%s -- VO2max = %.2f km/h",
		stat.BestEffortTimeStatistic.Result(bestActivityEffort),
		calculateVO2max(bestActivityEffort.Distance, bestActivityEffort.Seconds))
}

func calculateVO2max(distance float64, seconds int) float64 {
	return distance / float64(seconds) * 3600 / 1000
}
