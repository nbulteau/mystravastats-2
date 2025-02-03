package statistics

import (
	"fmt"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

type MaxSpeedStatistic struct {
	ActivityStatistic
	maxSpeed *float64
}

func NewMaxSpeedStatistic(activities []*strava.Activity) *MaxSpeedStatistic {
	stat := &MaxSpeedStatistic{
		ActivityStatistic: ActivityStatistic{
			BaseStatistic: BaseStatistic{
				name:       "Max speed",
				Activities: activities,
			},
		},
	}

	var maxSpeedActivity *strava.Activity
	for _, activity := range activities {
		if maxSpeedActivity == nil || activity.MaxSpeed > maxSpeedActivity.MaxSpeed {
			maxSpeedActivity = activity
		}
	}

	if maxSpeedActivity != nil {
		stat.activity = &business.ActivityShort{
			Id:   maxSpeedActivity.Id,
			Name: maxSpeedActivity.Name,
			Type: business.ActivityTypes[maxSpeedActivity.Type],
		}
		stat.maxSpeed = &maxSpeedActivity.MaxSpeed
	}

	return stat
}

func (stat *MaxSpeedStatistic) Value() string {
	if stat.maxSpeed != nil {
		return fmt.Sprintf("%.02f km/h", *stat.maxSpeed*3.6)
	}
	return "Not available"
}
