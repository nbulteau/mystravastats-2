package badges

import (
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

const (
	famousClimbActivityStartRadiusKM = 80.0
	famousClimbWaypointToleranceInM  = 500
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
	Category        string
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
	Category        string
}

func (f FamousClimbBadge) Check(activities []*strava.Activity) ([]*strava.Activity, bool) {
	var filteredActivities []*strava.Activity
	for _, activity := range activities {
		if len(activity.StartLatlng) > 0 {
			distanceToStart := f.Start.HaversineInKM(activity.StartLatlng[0], activity.StartLatlng[1])
			distanceToEnd := f.End.HaversineInKM(activity.StartLatlng[0], activity.StartLatlng[1])
			if distanceToStart < famousClimbActivityStartRadiusKM || distanceToEnd < famousClimbActivityStartRadiusKM {
				if f.checkAscentDirection(activity) {
					filteredActivities = append(filteredActivities, activity)
				}
			}
		}
	}
	return filteredActivities, len(filteredActivities) > 0
}

func (f FamousClimbBadge) checkAscentDirection(activity *strava.Activity) bool {
	if activity.Stream == nil || activity.Stream.LatLng == nil {
		return false
	}

	seenStart := false
	for _, coords := range activity.Stream.LatLng.Data {
		if len(coords) < 2 {
			continue
		}

		if !seenStart {
			if f.Start.HaversineInM(coords[0], coords[1]) < famousClimbWaypointToleranceInM {
				seenStart = true
			}
			continue
		}

		if f.End.HaversineInM(coords[0], coords[1]) < famousClimbWaypointToleranceInM {
			return true
		}
	}

	return false
}

func (f FamousClimbBadge) String() string {
	return f.Name
}
