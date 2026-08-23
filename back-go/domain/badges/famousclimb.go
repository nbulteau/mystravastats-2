package badges

import (
	"math"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"sort"
)

const (
	famousClimbActivityStartRadiusKM = 80.0
	famousClimbWaypointToleranceInM  = 500
	famousClimbLengthToleranceRatio  = 0.35
	famousClimbLengthToleranceMinM   = 750.0
)

type FamousClimb struct {
	Name           string
	Country        string
	Massif         string
	TopOfTheAscent int
	GeoCoordinate  business.GeoCoordinate
	Alternatives   []Alternative
}

type Alternative struct {
	Name                  string
	GeoCoordinate         business.GeoCoordinate
	RouteCheckpoints      []business.GeoCoordinate
	SummitToleranceMeters int
	Length                float64
	TotalAscent           int
	MinimumAltitude       int
	MaximumGradient       float64
	Difficulty            int
	Category              string
	AverageGradient       float64
	SourceURL             string
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
	Label                 string
	Name                  string
	Country               string
	Massif                string
	SourceURL             string
	TopOfTheAscent        int
	Start                 business.GeoCoordinate
	End                   business.GeoCoordinate
	RouteCheckpoints      []business.GeoCoordinate
	SummitToleranceMeters int
	Length                float64
	TotalAscent           int
	MinimumAltitude       int
	MaximumGradient       float64
	AverageGradient       float64
	Difficulty            int
	Category              string
}

func (f FamousClimbBadge) Check(activities []*strava.Activity) ([]*strava.Activity, bool) {
	var filteredActivities []*strava.Activity
	for _, activity := range activities {
		if len(activity.StartLatlng) > 0 {
			distanceToStart := f.Start.HaversineInKM(activity.StartLatlng[0], activity.StartLatlng[1])
			distanceToEnd := f.End.HaversineInKM(activity.StartLatlng[0], activity.StartLatlng[1])
			if distanceToStart < famousClimbActivityStartRadiusKM || distanceToEnd < famousClimbActivityStartRadiusKM {
				if _, matched := f.matchQuality(activity); matched {
					filteredActivities = append(filteredActivities, activity)
				}
			}
		}
	}
	return filteredActivities, len(filteredActivities) > 0
}

func (f FamousClimbBadge) checkAscentDirection(activity *strava.Activity) bool {
	_, matched := f.matchQuality(activity)
	return matched
}

func (f FamousClimbBadge) matchQuality(activity *strava.Activity) (float64, bool) {
	if activity.Stream == nil || activity.Stream.LatLng == nil {
		return 0, false
	}

	latLngData := activity.Stream.LatLng.Data
	checkpointMatchIndices, checkpointsAvailable := f.routeCheckpointMatchIndices(latLngData)
	if !checkpointsAvailable {
		return 0, false
	}
	startIndices := make([]int, 0)
	fallbackMatch := false
	scoredCandidate := false
	bestFallbackQuality := math.MaxFloat64
	bestQuality := math.MaxFloat64
	referenceLengthMeters := f.Length * 1000
	summitToleranceMeters := f.summitToleranceMeters()
	lengthToleranceMeters := math.Max(famousClimbLengthToleranceMinM, referenceLengthMeters*famousClimbLengthToleranceRatio)
	distances := activity.Stream.Distance.Data

	for index, coords := range latLngData {
		if len(coords) < 2 {
			continue
		}
		if f.Start.HaversineInM(coords[0], coords[1]) < famousClimbWaypointToleranceInM {
			startIndices = append(startIndices, index)
		}
		endDistanceMeters := f.End.HaversineInM(coords[0], coords[1])
		if endDistanceMeters >= summitToleranceMeters {
			continue
		}

		for _, startIndex := range startIndices {
			if startIndex >= index {
				continue
			}
			if !containsRouteCheckpoints(checkpointMatchIndices, startIndex, index) {
				continue
			}
			fallbackMatch = true
			startCoords := latLngData[startIndex]
			startProximity := float64(f.Start.HaversineInM(startCoords[0], startCoords[1])) / famousClimbWaypointToleranceInM
			endProximity := float64(endDistanceMeters) / float64(summitToleranceMeters)
			proximityQuality := (startProximity + endProximity) / 2
			bestFallbackQuality = math.Min(bestFallbackQuality, proximityQuality)
			if referenceLengthMeters <= 0 || startIndex >= len(distances) || index >= len(distances) {
				continue
			}
			candidateLengthMeters := distances[index] - distances[startIndex]
			if math.IsNaN(candidateLengthMeters) || math.IsInf(candidateLengthMeters, 0) || candidateLengthMeters <= 0 {
				continue
			}
			scoredCandidate = true
			lengthDifference := math.Abs(candidateLengthMeters - referenceLengthMeters)
			if lengthDifference <= lengthToleranceMeters {
				quality := lengthDifference/math.Max(referenceLengthMeters, 1) + proximityQuality*0.01
				bestQuality = math.Min(bestQuality, quality)
			}
		}
	}

	if bestQuality < math.MaxFloat64 {
		return bestQuality, true
	}
	if fallbackMatch && (!scoredCandidate || referenceLengthMeters <= 0) {
		return bestFallbackQuality, true
	}
	return 0, false
}

func (f FamousClimbBadge) summitToleranceMeters() int {
	if f.SummitToleranceMeters > 0 {
		return f.SummitToleranceMeters
	}
	return famousClimbWaypointToleranceInM
}

func (f FamousClimbBadge) routeCheckpointMatchIndices(latLngData [][]float64) ([][]int, bool) {
	if len(f.RouteCheckpoints) == 0 {
		return nil, true
	}

	checkpointMatchIndices := make([][]int, len(f.RouteCheckpoints))
	for index, coords := range latLngData {
		if len(coords) < 2 {
			continue
		}
		for checkpointIndex, checkpoint := range f.RouteCheckpoints {
			if checkpoint.HaversineInM(coords[0], coords[1]) < famousClimbWaypointToleranceInM {
				checkpointMatchIndices[checkpointIndex] = append(checkpointMatchIndices[checkpointIndex], index)
			}
		}
	}
	for _, indices := range checkpointMatchIndices {
		if len(indices) == 0 {
			return nil, false
		}
	}
	return checkpointMatchIndices, true
}

func containsRouteCheckpoints(checkpointMatchIndices [][]int, startIndex, endIndex int) bool {
	nextMinimumIndex := startIndex
	for _, indices := range checkpointMatchIndices {
		position := sort.SearchInts(indices, nextMinimumIndex)
		if position >= len(indices) || indices[position] > endIndex {
			return false
		}
		nextMinimumIndex = indices[position] + 1
	}
	return true
}

func (f FamousClimbBadge) String() string {
	return f.Name
}
