package business

import "mystravastats/domain/strava"

type BadgeSetEnum string

const (
	GENERAL BadgeSetEnum = "GENERAL"
	FAMOUS  BadgeSetEnum = "FAMOUS"
)

type BadgeCheckResult struct {
	Badge       Badge             `json:"badge"`
	Activities  []strava.Activity `json:"activities"`
	IsCompleted bool              `json:"isCompleted"`
}

type Badge interface {
	Check(activities []strava.Activity) (checkedActivities []strava.Activity, isCompleted bool)
	String() string
}

type BaseBadge struct {
	Label string
}

func (b BaseBadge) String() string {
	return b.Label
}
