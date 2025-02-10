package badges

import (
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

type FamousClimb struct {
	Name           string
	TopOfTheAscent int
	GeoCoordinate  business.GeoCoordinate
	Alternatives   []Alternative
}

type Alternative struct {
	Name            string
	GeoCoordinate   business.GeoCoordinate
	Length          float64
	TotalAscent     int
	Difficulty      int
	AverageGradient float64
}

func NewFamousClimb(name string, topOfTheAscent int, geoCoordinate business.GeoCoordinate, alternatives []Alternative) FamousClimb {
	return FamousClimb{
		Name:           name,
		TopOfTheAscent: topOfTheAscent,
		GeoCoordinate:  geoCoordinate,
		Alternatives:   alternatives,
	}
}

type FamousClimbBadge struct {
	Label           string
	Name            string
	TopOfTheAscent  int
	Start           business.GeoCoordinate
	End             business.GeoCoordinate
	Length          float64
	TotalAscent     int
	AverageGradient float64
	Difficulty      int
}

func (f FamousClimbBadge) Check(activities []*strava.Activity) ([]*strava.Activity, bool) {
	var filteredActivities []*strava.Activity
	for _, activity := range activities {
		if len(activity.StartLatlng) > 0 {
			if f.Start.HaversineInKM(activity.StartLatlng[0], activity.StartLatlng[1]) < 50 {
				if f.check(activity, f.Start) && f.check(activity, f.End) {
					filteredActivities = append(filteredActivities, activity)
				}
			}
		}
	}
	return filteredActivities, len(filteredActivities) > 0
}

func (f FamousClimbBadge) check(activity *strava.Activity, geoCoordinateToCheck business.GeoCoordinate) bool {
	if activity.Stream != nil && activity.Stream.LatLng != nil {
		for _, coords := range activity.Stream.LatLng.Data {
			if geoCoordinateToCheck.Match(coords[0], coords[1]) {
				return true
			}
		}
	}
	return false
}

func (f FamousClimbBadge) String() string {
	return f.Name
}
