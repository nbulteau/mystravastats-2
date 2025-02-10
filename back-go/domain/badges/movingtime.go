package badges

import (
	"mystravastats/domain/strava"
)

type MovingTimeBadge struct {
	Label      string
	MovingTime int
}

func (m MovingTimeBadge) Check(activities []*strava.Activity) ([]*strava.Activity, bool) {
	var checkedActivities []*strava.Activity
	for _, activity := range activities {
		if activity.MovingTime >= m.MovingTime {
			checkedActivities = append(checkedActivities, activity)
		}
	}
	return checkedActivities, len(checkedActivities) > 0
}

func (m MovingTimeBadge) String() string {
	return m.Label
}

var (
	MovingTimeLevel1 = MovingTimeBadge{
		Label:      "Moving 1 hour",
		MovingTime: 3600,
	}
	MovingTimeLevel2 = MovingTimeBadge{
		Label:      "Moving 2 hours",
		MovingTime: 7200,
	}
	MovingTimeLevel3 = MovingTimeBadge{
		Label:      "Moving 3 hours",
		MovingTime: 10800,
	}
	MovingTimeLevel4 = MovingTimeBadge{
		Label:      "Moving 4 hours",
		MovingTime: 14400,
	}
	MovingTimeLevel5 = MovingTimeBadge{
		Label:      "Moving 5 hours",
		MovingTime: 18000,
	}
	MovingTimeLevel6 = MovingTimeBadge{
		Label:      "Moving 6 hours",
		MovingTime: 21600,
	}
	MovingTimeLevel7 = MovingTimeBadge{
		Label:      "Moving 7 hours",
		MovingTime: 25200,
	}
	MovingTimeBadgesSet = BadgeSet{
		Name:   "Run that distance",
		Badges: []Badge{MovingTimeLevel1, MovingTimeLevel2, MovingTimeLevel3, MovingTimeLevel4, MovingTimeLevel5, MovingTimeLevel6, MovingTimeLevel7},
	}
)
