package badges

import (
	"mystravastats/domain/strava"
	"strconv"
)

type ElevationBadge struct {
	Label              string
	TotalElevationGain float64
}

func (e ElevationBadge) Check(activities []*strava.Activity) ([]*strava.Activity, bool) {
	var checkedActivities []*strava.Activity
	for _, activity := range activities {
		if activity.TotalElevationGain >= e.TotalElevationGain {
			checkedActivities = append(checkedActivities, activity)
		}
	}
	return checkedActivities, len(checkedActivities) > 0
}

func (e ElevationBadge) String() string {
	return e.Label + "\n" + strconv.FormatFloat(e.TotalElevationGain, 'f', 2, 64) + " m"
}

var (
	ElevationRideLevel1 = ElevationBadge{
		Label:              "Ride that climb 1000 m",
		TotalElevationGain: 1000,
	}
	ElevationRideLevel2 = ElevationBadge{
		Label:              "Ride that climb 1500 m",
		TotalElevationGain: 1500,
	}
	ElevationRideLevel3 = ElevationBadge{
		Label:              "Ride that climb 2000 m",
		TotalElevationGain: 2000,
	}
	ElevationRideLevel4 = ElevationBadge{
		Label:              "Ride that climb 2500 m",
		TotalElevationGain: 2500,
	}
	ElevationRideLevel5 = ElevationBadge{
		Label:              "Ride that climb 3000 m",
		TotalElevationGain: 3000,
	}
	ElevationRideLevel6 = ElevationBadge{
		Label:              "Ride that climb 3500 m",
		TotalElevationGain: 3500,
	}
	ElevationRideBadgeSet = BadgeSet{
		Name:   "Run that climb",
		Badges: []Badge{ElevationRideLevel1, ElevationRideLevel2, ElevationRideLevel3, ElevationRideLevel4, ElevationRideLevel5, ElevationRideLevel6},
	}

	ElevationRunLevel1 = ElevationBadge{
		Label:              "Run that climb",
		TotalElevationGain: 250,
	}
	ElevationRunLevel2 = ElevationBadge{
		Label:              "Run that climb",
		TotalElevationGain: 500,
	}
	ElevationRunLevel3 = ElevationBadge{
		Label:              "Run that climb",
		TotalElevationGain: 1000,
	}
	ElevationRunLevel4 = ElevationBadge{
		Label:              "Run that climb",
		TotalElevationGain: 1500,
	}
	ElevationRunLevel5 = ElevationBadge{
		Label:              "Run that climb",
		TotalElevationGain: 2000,
	}
	ElevationRunBadgeSet = BadgeSet{
		Name:   "Run that climb",
		Badges: []Badge{ElevationRunLevel1, ElevationRunLevel2, ElevationRunLevel3, ElevationRunLevel4, ElevationRunLevel5},
	}

	ElevationHikeLevel1 = ElevationBadge{
		Label:              "Hike that climb 1000 m",
		TotalElevationGain: 1000,
	}
	ElevationHikeLevel2 = ElevationBadge{
		Label:              "Hike that climb 1500 m",
		TotalElevationGain: 1500,
	}
	ElevationHikeLevel3 = ElevationBadge{
		Label:              "Hike that climb 2000 m",
		TotalElevationGain: 2000,
	}
	ElevationHikeLevel4 = ElevationBadge{
		Label:              "Hike that climb 2500 m",
		TotalElevationGain: 2500,
	}
	ElevationHikeLevel5 = ElevationBadge{
		Label:              "Hike that climb 3000 m",
		TotalElevationGain: 3000,
	}
	ElevationHikeBadgeSet = BadgeSet{
		Name:   "Run that climb",
		Badges: []Badge{ElevationHikeLevel1, ElevationHikeLevel2, ElevationHikeLevel3, ElevationHikeLevel4, ElevationHikeLevel5},
	}
)
