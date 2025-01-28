package statistics

import (
	"fmt"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

type CooperStatistic struct {
	BestEffortTimeStatistic
}

func NewCooperStatistic(activities []strava.Activity) *CooperStatistic {
	return &CooperStatistic{
		BestEffortTimeStatistic: *NewBestEffortTimeStatistic("Best Cooper (12 min)", activities, 12*60),
	}
}

func (cs *CooperStatistic) Result(bestActivityEffort business.ActivityEffort) string {
	vo2max := calculateVo2max(bestActivityEffort)
	return fmt.Sprintf("%s -- VO2 max = %.2f ml/kg/min", cs.BestEffortTimeStatistic.Result(&bestActivityEffort), vo2max)
}

func calculateVo2max(activityEffort business.ActivityEffort) float64 {
	return (activityEffort.Distance - 504.9) / 44.73
}
