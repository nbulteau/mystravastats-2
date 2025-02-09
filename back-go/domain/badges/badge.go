package badges

import (
	"mystravastats/domain/strava"
)

type Badge interface {
	Check(activities []*strava.Activity) (checkedActivities []*strava.Activity, isCompleted bool)
	String() string
}
