package business

type ActivityType int

var ActivityTypes = map[string]ActivityType{
	"Run":         Run,
	"Ride":        Ride,
	"InlineSkate": InlineSkate,
	"Hike":        Hike,
	"Commute":     Commute,
	"AlpineSki":   AlpineSki,
	"VirtualRide": VirtualRide,
}

const (
	Run ActivityType = iota
	Ride
	InlineSkate
	Hike
	Commute
	AlpineSki
	VirtualRide
)

func (a ActivityType) String() string {
	return [...]string{"Run", "Ride", "InlineSkate", "Hike", "Commute", "AlpineSki", "VirtualRide"}[a]
}
