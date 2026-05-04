package strava

import (
	"math"
)

type Stream struct {
	Distance       DistanceStream        `json:"distance"`
	Time           TimeStream            `json:"time"`
	LatLng         *LatLngStream         `json:"latlng,omitempty"`
	Cadence        *CadenceStream        `json:"cadence,omitempty"`
	HeartRate      *HeartRateStream      `json:"heartrate,omitempty"`
	Moving         *MovingStream         `json:"moving,omitempty"`
	Altitude       *AltitudeStream       `json:"altitude,omitempty"`
	Watts          *PowerStream          `json:"watts,omitempty"`
	VelocitySmooth *SmoothVelocityStream `json:"velocity_smooth,omitempty"`
	GradeSmooth    *SmoothGradeStream    `json:"grade_smooth,omitempty"`
}

type DistanceStream struct {
	Data         []float64 `json:"data"`
	OriginalSize int       `json:"original_size"`
	Resolution   string    `json:"resolution"`
	SeriesType   string    `json:"series_type"`
}

type TimeStream struct {
	Data         []int  `json:"data"`
	OriginalSize int    `json:"original_size"`
	Resolution   string `json:"resolution"`
	SeriesType   string `json:"series_type"`
}

type LatLngStream struct {
	Data         [][]float64 `json:"data"`
	OriginalSize int         `json:"original_size"`
	Resolution   string      `json:"resolution"`
	SeriesType   string      `json:"series_type"`
}

type PowerStream struct {
	Data         []float64 `json:"data"`
	OriginalSize int       `json:"original_size"`
	Resolution   string    `json:"resolution"`
	SeriesType   string    `json:"series_type"`
}

type AltitudeStream struct {
	Data         []float64 `json:"data"`
	OriginalSize int       `json:"original_size"`
	Resolution   string    `json:"resolution"`
	SeriesType   string    `json:"series_type"`
}

type CadenceStream struct {
	Data         []int  `json:"data"`
	OriginalSize int    `json:"original_size"`
	Resolution   string `json:"resolution"`
	SeriesType   string `json:"series_type"`
}

type HeartRateStream struct {
	Data         []int  `json:"data"`
	OriginalSize int    `json:"original_size"`
	Resolution   string `json:"resolution"`
	SeriesType   string `json:"series_type"`
}

type MovingStream struct {
	Data         []bool `json:"data"`
	OriginalSize int    `json:"original_size"`
	Resolution   string `json:"resolution"`
	SeriesType   string `json:"series_type"`
}

type SmoothVelocityStream struct {
	Data         []float64 `json:"data"`
	OriginalSize int       `json:"original_size"`
	Resolution   string    `json:"resolution"`
	SeriesType   string    `json:"series_type"`
}

type SmoothGradeStream struct {
	Data         []float64 `json:"data"`
	OriginalSize int       `json:"original_size"`
	Resolution   string    `json:"resolution"`
	SeriesType   string    `json:"series_type"`
}

type GpxPoint struct {
	Latitude  float64
	Longitude float64
}

func (stream *LatLngStream) isValidPoint(previous, current GpxPoint, threshold float64) bool {
	distance := haversine(previous.Latitude, previous.Longitude, current.Latitude, current.Longitude)
	return distance <= threshold
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371e3 // Earth radius in meters
	lat1Rad := toRadians(lat1)
	lat2Rad := toRadians(lat2)
	Δφ := toRadians(lat2 - lat1)
	Δλ := toRadians(lon2 - lon1)

	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(Δλ/2)*math.Sin(Δλ/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c // in meters
}

func toRadians(deg float64) float64 {
	return deg * math.Pi / 180
}

func (stream *LatLngStream) CorrectInconsistentGpxPoints(threshold float64) []GpxPoint {
	var correctedPoints []GpxPoint
	points := make([]GpxPoint, len(stream.Data))
	for i, point := range stream.Data {
		points[i] = GpxPoint{Latitude: point[0], Longitude: point[1]}
	}
	correctedPoints = append(correctedPoints, points[0])
	for i := 1; i < len(points); i++ {
		previous := correctedPoints[len(correctedPoints)-1]
		current := points[i]
		if stream.isValidPoint(previous, current, threshold) {
			correctedPoints = append(correctedPoints, current)
		} else {
			correctedLatitude := (previous.Latitude + current.Latitude) / 2
			correctedLongitude := (previous.Longitude + current.Longitude) / 2
			correctedPoints = append(correctedPoints, GpxPoint{Latitude: correctedLatitude, Longitude: correctedLongitude})
		}
	}
	return correctedPoints
}

type SlopeType int

const (
	ASCENT SlopeType = iota
	DESCENT
	PLATEAU
)

type Slope struct {
	Type          SlopeType
	StartIndex    int
	EndIndex      int
	StartAltitude float64
	EndAltitude   float64
	Grade         float64
	MaxGrade      float64
	Distance      float64
	Duration      int
	AverageSpeed  float64
}

// ListSlopesDefault provides a convenient wrapper with default parameters.
func (s *Stream) ListSlopesDefault() []Slope {
	return s.ListSlopes(3.0, 500.0, 3500.0, 25)
}

// ListSlopes extracts sustained ascent segments from the stream data.
// Parameters:
// - threshold: minimum smoothed grade in percent to enter a climb (default 3.0)
// - minDistance: minimum distance in meters for a segment (default 500.0)
// - climbIndex: minimum "distance × abs(grade)" for ascents (default 3500.0)
// - smoothingWindow: distance-based smoothing hint for grade samples (default 25)
func (s *Stream) ListSlopes(threshold, minDistance, climbIndex float64, smoothingWindow int) []Slope {
	slopes := make([]Slope, 0)

	// Check that we have the necessary data
	if s == nil || s.Altitude == nil || len(s.Altitude.Data) == 0 {
		return slopes
	}

	altitudeData := s.Altitude.Data
	distanceData := s.Distance.Data
	timeData := s.Time.Data

	// Ensure all lists have the same size
	dataSize := minInt(len(altitudeData), len(distanceData), len(timeData))
	if dataSize < 2 {
		return slopes
	}

	gradeSamples := s.gradePercentSamples(altitudeData, distanceData, dataSize)
	windowMeters := math.Max(100.0, float64(smoothingWindow)*10.0)
	smoothedGrades := smoothGradeByDistance(gradeSamples, distanceData, dataSize, windowMeters)

	return detectSustainedAscents(
		distanceData,
		altitudeData,
		timeData,
		smoothedGrades,
		dataSize,
		threshold,
		minDistance,
		climbIndex,
	)
}

func (s *Stream) gradePercentSamples(altitudes, distances []float64, dataSize int) []float64 {
	grades := make([]float64, dataSize)
	if s.GradeSmooth != nil && len(s.GradeSmooth.Data) >= dataSize {
		maxGrade := maxAbsGradeSmooth(s.GradeSmooth.Data[:dataSize])
		if maxGrade > 0 {
			ratioScale := maxGrade <= 1.0
			for i := 0; i < dataSize; i++ {
				grade := finiteOrZero(s.GradeSmooth.Data[i])
				if ratioScale {
					grade *= 100
				}
				grades[i] = grade
			}
			return grades
		}
	}

	for i := 1; i < dataSize; i++ {
		altitudeDiff := altitudes[i] - altitudes[i-1]
		distanceDiff := distances[i] - distances[i-1]
		if distanceDiff <= 0 || !isFiniteFloat(altitudeDiff) || !isFiniteFloat(distanceDiff) {
			grades[i] = 0
			continue
		}
		grades[i] = (altitudeDiff / distanceDiff) * 100
	}
	grades[0] = grades[1]
	return grades
}

func maxAbsGradeSmooth(grades []float64) float64 {
	maxAbs := 0.0
	for _, grade := range grades {
		if !isFiniteFloat(grade) {
			continue
		}
		maxAbs = math.Max(maxAbs, math.Abs(grade))
	}
	return maxAbs
}

func smoothGradeByDistance(grades, distances []float64, dataSize int, windowMeters float64) []float64 {
	smoothed := make([]float64, dataSize)
	if dataSize == 0 {
		return smoothed
	}

	halfWindow := windowMeters / 2
	for i := 0; i < dataSize; i++ {
		if !isFiniteFloat(distances[i]) {
			smoothed[i] = finiteOrZero(grades[i])
			continue
		}

		sum := 0.0
		count := 0
		for j := i; j >= 0; j-- {
			if !isFiniteFloat(distances[j]) {
				continue
			}
			if distances[i]-distances[j] > halfWindow {
				break
			}
			if isFiniteFloat(grades[j]) {
				sum += grades[j]
				count++
			}
		}
		for j := i + 1; j < dataSize; j++ {
			if !isFiniteFloat(distances[j]) {
				continue
			}
			if distances[j]-distances[i] > halfWindow {
				break
			}
			if isFiniteFloat(grades[j]) {
				sum += grades[j]
				count++
			}
		}

		if count == 0 {
			smoothed[i] = finiteOrZero(grades[i])
		} else {
			smoothed[i] = sum / float64(count)
		}
	}
	return smoothed
}

func detectSustainedAscents(
	distances []float64,
	altitudes []float64,
	times []int,
	grades []float64,
	dataSize int,
	threshold float64,
	minDistance float64,
	climbIndex float64,
) []Slope {
	if threshold <= 0 {
		threshold = 3.0
	}
	exitThreshold := math.Max(1.0, threshold*0.35)
	minAverageGrade := math.Max(exitThreshold, threshold*0.65)
	falseFlatDistance := math.Min(300.0, math.Max(150.0, minDistance*0.5))

	slopes := make([]Slope, 0)
	inClimb := false
	climbStartIndex := 0
	belowExitStartIndex := -1
	belowExitStartDistance := 0.0

	for i := 1; i < dataSize; i++ {
		grade := finiteOrZero(grades[i])
		if !inClimb {
			if grade >= threshold {
				climbStartIndex = i - 1
				inClimb = true
				belowExitStartIndex = -1
			}
			continue
		}

		if grade < exitThreshold {
			if belowExitStartIndex < 0 {
				belowExitStartIndex = i
				belowExitStartDistance = distances[i]
			}
			if isFiniteFloat(distances[i]) &&
				isFiniteFloat(belowExitStartDistance) &&
				distances[i]-belowExitStartDistance >= falseFlatDistance {
				endIndex := belowExitStartIndex - 1
				if slope, ok := buildClimbSlope(distances, altitudes, times, grades, climbStartIndex, endIndex, minDistance, minAverageGrade, climbIndex); ok {
					slopes = append(slopes, slope)
				}
				inClimb = false
				belowExitStartIndex = -1
			}
			continue
		}

		belowExitStartIndex = -1
	}

	if inClimb {
		if slope, ok := buildClimbSlope(distances, altitudes, times, grades, climbStartIndex, dataSize-1, minDistance, minAverageGrade, climbIndex); ok {
			slopes = append(slopes, slope)
		}
	}

	return slopes
}

func buildClimbSlope(
	distances []float64,
	altitudes []float64,
	times []int,
	grades []float64,
	startIndex int,
	endIndex int,
	minDistance float64,
	minAverageGrade float64,
	climbIndex float64,
) (Slope, bool) {
	if startIndex < 0 || endIndex <= startIndex || endIndex >= len(distances) || endIndex >= len(altitudes) || endIndex >= len(times) {
		return Slope{}, false
	}

	startAltitude := altitudes[startIndex]
	endAltitude := altitudes[endIndex]
	totalDistance := distances[endIndex] - distances[startIndex]
	totalDuration := times[endIndex] - times[startIndex]
	if !isFiniteFloat(startAltitude) || !isFiniteFloat(endAltitude) || !isFiniteFloat(totalDistance) || totalDistance <= 0 || totalDuration < 0 {
		return Slope{}, false
	}

	averageGrade := ((endAltitude - startAltitude) / totalDistance) * 100
	if !isFiniteFloat(averageGrade) ||
		totalDistance < minDistance ||
		averageGrade < minAverageGrade ||
		totalDistance*averageGrade < climbIndex {
		return Slope{}, false
	}

	maxGrade := averageGrade
	for i := startIndex + 1; i <= endIndex && i < len(grades); i++ {
		if isFiniteFloat(grades[i]) {
			maxGrade = math.Max(maxGrade, grades[i])
		}
	}

	averageSpeed := 0.0
	if totalDuration > 0 {
		averageSpeed = totalDistance / float64(totalDuration)
	}

	return Slope{
		Type:          ASCENT,
		StartIndex:    startIndex,
		EndIndex:      endIndex,
		StartAltitude: startAltitude,
		EndAltitude:   endAltitude,
		Grade:         averageGrade,
		MaxGrade:      maxGrade,
		Distance:      totalDistance,
		Duration:      totalDuration,
		AverageSpeed:  averageSpeed,
	}, true
}

func finiteOrZero(value float64) float64 {
	if !isFiniteFloat(value) {
		return 0
	}
	return value
}

func isFiniteFloat(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func minInt(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
