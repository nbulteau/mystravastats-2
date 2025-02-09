package badges

import (
	"mystravastats/domain/strava"
)

type DistanceBadge struct {
	Label    string
	Distance float64
}

func (d DistanceBadge) Check(activities []*strava.Activity) ([]*strava.Activity, bool) {
	var checkedActivities []*strava.Activity
	for _, activity := range activities {
		if activity.Distance >= d.Distance {
			checkedActivities = append(checkedActivities, activity)
		}
	}
	return checkedActivities, len(checkedActivities) > 0
}

func (d DistanceBadge) String() string {
	return d.Label
}

var (
	DistanceRideLevel1 = DistanceBadge{
		Label:    "Hit the road 50 km",
		Distance: 50000,
	}
	DistanceRideLevel2 = DistanceBadge{
		Label:    "Hit the road 100 km",
		Distance: 100000,
	}
	DistanceRideLevel3 = DistanceBadge{
		Label:    "Hit the road 150 km",
		Distance: 150000,
	}
	DistanceRideLevel4 = DistanceBadge{
		Label:    "Hit the road 200 km",
		Distance: 200000,
	}
	DistanceRideLevel5 = DistanceBadge{
		Label:    "Hit the road 250 km",
		Distance: 250000,
	}
	DistanceRideLevel6 = DistanceBadge{
		Label:    "Hit the road 300 km",
		Distance: 300000,
	}
	DistanceRideBadgeSet = BadgeSet{
		Name:   "Hit the road",
		Badges: []Badge{DistanceRideLevel1, DistanceRideLevel2, DistanceRideLevel3, DistanceRideLevel4, DistanceRideLevel5, DistanceRideLevel6},
	}

	DistanceRunLevel1 = DistanceBadge{
		Label:    "Run that distance 10 km",
		Distance: 10000,
	}
	DistanceRunLevel2 = DistanceBadge{
		Label:    "Run that distance half Marathon",
		Distance: 21097,
	}
	DistanceRunLevel3 = DistanceBadge{
		Label:    "Run that distance 30 km",
		Distance: 30000,
	}
	DistanceRunLevel4 = DistanceBadge{
		Label:    "Run that distance Marathon",
		Distance: 42195,
	}
	DistanceRunBadgeSet = BadgeSet{
		Name:   "Run that distance",
		Badges: []Badge{DistanceRunLevel1, DistanceRunLevel2, DistanceRunLevel3, DistanceRunLevel4},
	}

	DistanceHikeLevel1 = DistanceBadge{
		Label:    "Hike that distance 10 km",
		Distance: 10000,
	}
	DistanceHikeLevel2 = DistanceBadge{
		Label:    "Hike that distance 15 km",
		Distance: 15000,
	}
	DistanceHikeLevel3 = DistanceBadge{
		Label:    "Hike that distance 20 km",
		Distance: 20000,
	}
	DistanceHikeLevel4 = DistanceBadge{
		Label:    "Hike that distance 25 km",
		Distance: 25000,
	}
	DistanceHikeLevel5 = DistanceBadge{
		Label:    "Hike that distance 30 km",
		Distance: 30000,
	}
	DistanceHikeLevel6 = DistanceBadge{
		Label:    "Hike that distance 35 km",
		Distance: 35000,
	}
	DistanceHikeBadgeSet = BadgeSet{
		Name:   "Hike that distance",
		Badges: []Badge{DistanceHikeLevel1, DistanceHikeLevel2, DistanceHikeLevel3, DistanceHikeLevel4, DistanceHikeLevel5, DistanceHikeLevel6},
	}
)
