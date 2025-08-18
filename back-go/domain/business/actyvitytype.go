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
	"Commute":          Commute,
	"AlpineSki":        AlpineSki,
	"VirtualRide":      VirtualRide,
}

const (
	Run ActivityType = iota
	TrailRun
	Ride
	GravelRide
	MountainBikeRide
	InlineSkate
	Hike
	Commute
	AlpineSki
	VirtualRide
)

func (a ActivityType) String() string {
	return [...]string{"Run", "TrailRun", "Ride", "GravelRide", "MountainBikeRide", "InlineSkate", "Hike", "Commute", "AlpineSki", "VirtualRide"}[a]
}
