package statistics

import (
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
	"mystravastats/internal/helpers"
)

type MaxMovingTimeStatistic struct {
	ActivityStatistic
	maxMovingTime *int
}

func NewMaxMovingTimeStatistic(activities []*strava.Activity) *MaxMovingTimeStatistic {
	stat := &MaxMovingTimeStatistic{
		ActivityStatistic: ActivityStatistic{
			BaseStatistic: BaseStatistic{
				name:       "Max moving time",
				Activities: activities,
			},
		},
	}

	var maxMovingTimeActivity *strava.Activity
	for _, activity := range activities {
		if maxMovingTimeActivity == nil || activity.MovingTime > maxMovingTimeActivity.MovingTime {
			maxMovingTimeActivity = activity
		}
	}

	if maxMovingTimeActivity != nil {
		stat.activity = &business.ActivityShort{
			Id:   maxMovingTimeActivity.Id,
			Name: maxMovingTimeActivity.Name,
			Type: business.ActivityTypes[maxMovingTimeActivity.Type],
		}
		stat.maxMovingTime = &maxMovingTimeActivity.MovingTime
	}

	return stat
}

func (stat *MaxMovingTimeStatistic) Value() string {
	if stat.maxMovingTime != nil {
		return helpers.FormatSeconds(*stat.maxMovingTime)
	}
	return "Not available"
}
