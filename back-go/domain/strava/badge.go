package strava

type BadgeSetEnum string

const (
	GENERAL BadgeSetEnum = "GENERAL"
	FAMOUS  BadgeSetEnum = "FAMOUS"
)

type BadgeCheckResult struct {
	Badge       Badge      `json:"badge"`
	Activities  []Activity `json:"activities"`
	IsCompleted bool       `json:"isCompleted"`
}

type Badge interface {
	Check(activities []Activity) (checkedActivities []Activity, isCompleted bool)
	String() string
}

type BaseBadge struct {
	Label string
}

func (b BaseBadge) String() string {
	return b.Label
}
