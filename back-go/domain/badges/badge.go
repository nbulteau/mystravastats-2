package badges

import (
	"mystravastats/internal/shared/domain/strava"
)

type Badge interface {
	Check(activities []*strava.Activity) (checkedActivities []*strava.Activity, isCompleted bool)
	String() string
}
