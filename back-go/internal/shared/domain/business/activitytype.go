package business

type ActivityType int

var ActivityTypes = map[string]ActivityType{
	"Run":              Run,
	"TrailRun":         TrailRun,
	"Ride":             Ride,
	"GravelRide":       GravelRide,
	"MountainBikeRide": MountainBikeRide,
	"InlineSkate":      InlineSkate,
	"Hike":             Hike,
	"Walk":             Walk,
	"Commute":          Commute,
	"AlpineSki":        AlpineSki,
	"VirtualRide":      VirtualRide,
}

var badgeActivityTypeFamilies = []struct {
	representative ActivityType
	members        map[ActivityType]struct{}
}{
	{
		representative: Ride,
		members: map[ActivityType]struct{}{
			Ride: {}, GravelRide: {}, MountainBikeRide: {}, VirtualRide: {}, Commute: {},
		},
	},
	{
		representative: Run,
		members: map[ActivityType]struct{}{
			Run: {}, TrailRun: {},
		},
	},
	{
		representative: Hike,
		members: map[ActivityType]struct{}{
			Hike: {}, Walk: {},
		},
	},
}

const (
	Run ActivityType = iota
	TrailRun
	Ride
	GravelRide
	MountainBikeRide
	InlineSkate
	Hike
	Walk
	Commute
	AlpineSki
	VirtualRide
)

func (a ActivityType) String() string {
	return [...]string{"Run", "TrailRun", "Ride", "GravelRide", "MountainBikeRide", "InlineSkate", "Hike", "Walk", "Commute", "AlpineSki", "VirtualRide"}[a]
}

func RepresentativeBadgeActivityType(activityTypes ...ActivityType) (ActivityType, bool) {
	if len(activityTypes) == 0 {
		return 0, false
	}

	for _, family := range badgeActivityTypeFamilies {
		if allActivityTypesInFamily(activityTypes, family.members) {
			return family.representative, true
		}
	}

	return 0, false
}

func allActivityTypesInFamily(activityTypes []ActivityType, members map[ActivityType]struct{}) bool {
	for _, activityType := range activityTypes {
		if _, ok := members[activityType]; !ok {
			return false
		}
	}
	return true
}
