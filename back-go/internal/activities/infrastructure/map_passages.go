package infrastructure

import (
	"math"
	application "mystravastats/internal/activities/application"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"sort"
)

const (
	mapPassageDefaultResolutionMeters  = 120
	mapPassageAllYearsResolutionMeters = 250
	mapPassageMaxLegMeters             = 2000.0
	mapPassageMetersPerDegree          = 111320.0
	mapPassageEarthRadiusM             = 6371e3
	mapPassageDefaultMaxSegments       = 12000
	mapPassageAllYearsMaxSegments      = 5000
)

type mapPassageCell struct {
	lat int
	lng int
}

type mapPassageEdge struct {
	start mapPassageCell
	end   mapPassageCell
}

type mapPassageAccumulator struct {
	passageCount       int
	activityTypeCounts map[string]int
}

type mapPassageOptions struct {
	resolutionMeters int
	minPassageCount  int
	maxSegments      int
}

func mapPassageOptionsForYear(year *int) mapPassageOptions {
	if year == nil {
		return mapPassageOptions{
			resolutionMeters: mapPassageAllYearsResolutionMeters,
			minPassageCount:  2,
			maxSegments:      mapPassageAllYearsMaxSegments,
		}
	}
	return defaultMapPassageOptions()
}

func defaultMapPassageOptions() mapPassageOptions {
	return mapPassageOptions{
		resolutionMeters: mapPassageDefaultResolutionMeters,
		minPassageCount:  1,
		maxSegments:      mapPassageDefaultMaxSegments,
	}
}

func computeMapPassages(activities []*strava.Activity, exclusions map[int64]business.DataQualityExclusion) application.MapPassagesResponse {
	return computeMapPassagesWithOptions(activities, exclusions, defaultMapPassageOptions())
}

func computeMapPassagesWithOptions(activities []*strava.Activity, exclusions map[int64]business.DataQualityExclusion, options mapPassageOptions) application.MapPassagesResponse {
	if options.resolutionMeters <= 0 {
		options.resolutionMeters = mapPassageDefaultResolutionMeters
	}
	if options.minPassageCount <= 0 {
		options.minPassageCount = 1
	}
	accumulators := make(map[mapPassageEdge]*mapPassageAccumulator)
	excludedActivities := 0
	missingStreamActivities := 0
	includedActivities := 0

	for _, activity := range activities {
		if activity == nil {
			missingStreamActivities++
			continue
		}
		if _, excluded := exclusions[activity.Id]; excluded {
			excludedActivities++
			continue
		}
		if activity.Stream == nil || activity.Stream.LatLng == nil {
			missingStreamActivities++
			continue
		}

		coordinates := validMapPassageCoordinates(activity.Stream.LatLng.Data)
		if len(coordinates) < 2 {
			missingStreamActivities++
			continue
		}

		activityEdges := mapPassageEdgesForActivity(coordinates, options.resolutionMeters)
		if len(activityEdges) == 0 {
			missingStreamActivities++
			continue
		}

		includedActivities++
		activityType := resolveMapTrackActivityType(activity)
		for edge := range activityEdges {
			accumulator := accumulators[edge]
			if accumulator == nil {
				accumulator = &mapPassageAccumulator{
					activityTypeCounts: map[string]int{},
				}
				accumulators[edge] = accumulator
			}
			accumulator.passageCount++
			accumulator.activityTypeCounts[activityType]++
		}
	}

	segments := make([]application.MapPassageSegment, 0, len(accumulators))
	omittedSegments := 0
	for edge, accumulator := range accumulators {
		if accumulator.passageCount < options.minPassageCount {
			omittedSegments++
			continue
		}
		start := mapPassageCellCenter(edge.start, options.resolutionMeters)
		end := mapPassageCellCenter(edge.end, options.resolutionMeters)
		edgeDistanceKm := mapPassageDistanceMeters(start[0], start[1], end[0], end[1]) / 1000.0
		segments = append(segments, application.MapPassageSegment{
			Coordinates:        [][]float64{start, end},
			PassageCount:       accumulator.passageCount,
			ActivityCount:      accumulator.passageCount,
			DistanceKm:         roundMapPassageDistance(edgeDistanceKm * float64(accumulator.passageCount)),
			ActivityTypeCounts: cloneMapPassageActivityTypeCounts(accumulator.activityTypeCounts),
		})
	}

	sort.SliceStable(segments, func(i, j int) bool {
		if segments[i].PassageCount != segments[j].PassageCount {
			return segments[i].PassageCount > segments[j].PassageCount
		}
		left := segments[i].Coordinates[0]
		right := segments[j].Coordinates[0]
		if left[0] != right[0] {
			return left[0] < right[0]
		}
		return left[1] < right[1]
	})

	if options.maxSegments > 0 && len(segments) > options.maxSegments {
		omittedSegments += len(segments) - options.maxSegments
		segments = segments[:options.maxSegments]
	}

	return application.MapPassagesResponse{
		Segments:                segments,
		IncludedActivities:      includedActivities,
		ExcludedActivities:      excludedActivities,
		MissingStreamActivities: missingStreamActivities,
		ResolutionMeters:        options.resolutionMeters,
		MinPassageCount:         options.minPassageCount,
		OmittedSegments:         omittedSegments,
	}
}

func validMapPassageCoordinates(coordinates [][]float64) [][]float64 {
	result := make([][]float64, 0, len(coordinates))
	for _, coordinate := range coordinates {
		if len(coordinate) < 2 || !isFiniteMapPassageCoordinate(coordinate[0], coordinate[1]) {
			continue
		}
		result = append(result, []float64{coordinate[0], coordinate[1]})
	}
	return result
}

func mapPassageEdgesForActivity(coordinates [][]float64, resolutionMeters int) map[mapPassageEdge]struct{} {
	cells := make([]mapPassageCell, 0, len(coordinates))
	appendCell := func(cell mapPassageCell) {
		if len(cells) > 0 && cells[len(cells)-1] == cell {
			return
		}
		cells = append(cells, cell)
	}

	for index := 1; index < len(coordinates); index++ {
		previous := coordinates[index-1]
		current := coordinates[index]
		distance := mapPassageDistanceMeters(previous[0], previous[1], current[0], current[1])
		if distance <= 0 || distance > mapPassageMaxLegMeters {
			continue
		}

		steps := int(math.Ceil(distance / float64(resolutionMeters)))
		if steps < 1 {
			steps = 1
		}
		for step := 0; step <= steps; step++ {
			ratio := float64(step) / float64(steps)
			lat := previous[0] + (current[0]-previous[0])*ratio
			lng := previous[1] + (current[1]-previous[1])*ratio
			appendCell(mapPassageCellForCoordinate(lat, lng, resolutionMeters))
		}
	}

	edges := make(map[mapPassageEdge]struct{})
	for index := 1; index < len(cells); index++ {
		if cells[index-1] == cells[index] {
			continue
		}
		edges[normalizeMapPassageEdge(cells[index-1], cells[index])] = struct{}{}
	}
	return edges
}

func mapPassageCellForCoordinate(lat float64, lng float64, resolutionMeters int) mapPassageCell {
	degrees := float64(resolutionMeters) / mapPassageMetersPerDegree
	return mapPassageCell{
		lat: int(math.Floor(lat / degrees)),
		lng: int(math.Floor(lng / degrees)),
	}
}

func mapPassageCellCenter(cell mapPassageCell, resolutionMeters int) []float64 {
	degrees := float64(resolutionMeters) / mapPassageMetersPerDegree
	return []float64{
		(float64(cell.lat) + 0.5) * degrees,
		(float64(cell.lng) + 0.5) * degrees,
	}
}

func normalizeMapPassageEdge(left mapPassageCell, right mapPassageCell) mapPassageEdge {
	if left.lat < right.lat || (left.lat == right.lat && left.lng <= right.lng) {
		return mapPassageEdge{start: left, end: right}
	}
	return mapPassageEdge{start: right, end: left}
}

func mapPassageDistanceMeters(lat1 float64, lon1 float64, lat2 float64, lon2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return mapPassageEarthRadiusM * c
}

func isFiniteMapPassageCoordinate(lat float64, lng float64) bool {
	return !math.IsNaN(lat) && !math.IsInf(lat, 0) && !math.IsNaN(lng) && !math.IsInf(lng, 0)
}

func roundMapPassageDistance(value float64) float64 {
	return math.Round(value*100) / 100
}

func cloneMapPassageActivityTypeCounts(source map[string]int) map[string]int {
	result := make(map[string]int, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
