package services

type ActivityType int

const (
	Run ActivityType = iota
	Ride
	RideWithCommute
	InlineSkate
	Hike
	Commute
	AlpineSki
	VirtualRide
)

var ActivityTypes = map[string]ActivityType{
	"Run":             Run,
	"Ride":            Ride,
	"RideWithCommute": RideWithCommute,
	"InlineSkate":     InlineSkate,
	"Hike":            Hike,
	"Commute":         Commute,
	"AlpineSki":       AlpineSki,
	"VirtualRide":     VirtualRide,
}

func (a ActivityType) String() string {
	return [...]string{"Run", "Ride", "RideWithCommute", "InlineSkate", "Hike", "Commute", "AlpineSki", "VirtualRide"}[a]
}
