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

// smooth applies a moving average filter to smooth the data
func smooth(data []float64, size int) []float64 {
	if size <= 0 {
		size = 5 // default value
	}

	smooth := make([]float64, len(data))

	// Copy first 'size' elements as-is
	for i := 0; i < size && i < len(data); i++ {
		smooth[i] = data[i]
	}

	// Apply smoothing to middle elements
	for i := size; i < len(data)-size; i++ {
		sum := 0.0
		for j := i - size; j <= i+size; j++ {
			sum += data[j]
		}
		smooth[i] = sum / float64(2*size+1)
	}

	// Copy last 'size' elements as-is
	for i := len(data) - size; i < len(data); i++ {
		if i >= 0 {
			smooth[i] = data[i]
		}
	}

	return smooth
}

// ListSlopesDefault provides a convenient wrapper with default parameters.
func (s *Stream) ListSlopesDefault() []Slope {
	return s.ListSlopes(3.0, 500.0, 3500.0, 50)
}

// ListSlopes extracts slope segments (ascents/descents/plateaus) from the stream data.
// Parameters:
// - threshold: minimum average grade in percent to classify ascent/descent (default 3.0)
// - minDistance: minimum distance in meters for a segment (default 500.0)
// - climbIndex: minimum "distance × abs(grade)" for ascents (default 3500.0)
// - smoothingWindow: window size for smoothing the raw data (default 20)
func (s *Stream) ListSlopes(threshold, minDistance, climbIndex float64, smoothingWindow int) []Slope {
	slopes := make([]Slope, 0)

	// Check that we have the necessary data
	if s.Altitude == nil || len(s.Altitude.Data) == 0 {
		return slopes
	}

	// Apply smoothing to raw GPS data to reduce noise
	rawAltitudeData := s.Altitude.Data
	rawDistanceData := s.Distance.Data
	timeData := s.Time.Data

	// Smooth altitude and distance data using the helper function
	smoothedAltitudeData := smooth(rawAltitudeData, smoothingWindow)
	smoothedDistanceData := smooth(rawDistanceData, smoothingWindow)

	// Ensure all lists have the same size
	dataSize := minInt(len(smoothedAltitudeData), len(smoothedDistanceData), len(timeData))
	if dataSize < 2 {
		return slopes
	}

	currentSlopeStartIndex := 0
	var currentSlopeType *SlopeType

	for i := 1; i < dataSize; i++ {
		altitudeDiff := smoothedAltitudeData[i] - smoothedAltitudeData[i-1]
		distanceDiff := smoothedDistanceData[i] - smoothedDistanceData[i-1]

		if distanceDiff == 0.0 {
			continue
		}

		grade := (altitudeDiff / distanceDiff) * 100 // Percentage grade

		var slopeType SlopeType
		switch {
		case grade >= threshold:
			slopeType = ASCENT
		case grade <= -threshold:
			slopeType = DESCENT
		default:
			slopeType = PLATEAU
		}

		// If the slope type changes or if we're at the last point
		if currentSlopeType == nil || *currentSlopeType != slopeType || i == dataSize-1 {
			// Create a slope segment if we have a previous segment
			if currentSlopeType != nil {
				endIndex := i - 1
				if i == dataSize-1 {
					endIndex = i
				}

				if endIndex > currentSlopeStartIndex {
					startAltitude := smoothedAltitudeData[currentSlopeStartIndex]
					endAltitude := smoothedAltitudeData[endIndex]
					totalDistance := smoothedDistanceData[endIndex] - smoothedDistanceData[currentSlopeStartIndex]
					totalDuration := timeData[endIndex] - timeData[currentSlopeStartIndex]

					// Calculate average grade for the segment using smoothed data
					var averageGrade float64
					if totalDistance > 0 {
						averageGrade = ((endAltitude - startAltitude) / totalDistance) * 100
					}

					// Apply specific criteria for ascents
					shouldIncludeSlope := false
					switch *currentSlopeType {
					case ASCENT:
						// For ascents, check all three criteria:
						// 1. Minimum distance (500m)
						// 2. Minimum average grade (3%)
						// 3. Minimum climb index (3500)
						calculatedClimbIndex := totalDistance * math.Abs(averageGrade)
						shouldIncludeSlope = totalDistance >= minDistance &&
							math.Abs(averageGrade) >= threshold &&
							calculatedClimbIndex >= climbIndex
					case DESCENT:
						// For descents, use basic distance criteria
						shouldIncludeSlope = totalDistance >= minDistance && math.Abs(averageGrade) >= threshold
					case PLATEAU:
						// For plateaus, use basic distance criteria
						shouldIncludeSlope = totalDistance >= minDistance
					}

					if shouldIncludeSlope {
						// Calculate the maximum grade for the segment using smoothed data
						maxGrade := 0.0
						for j := currentSlopeStartIndex; j < endIndex; j++ {
							segmentDistanceDiff := smoothedDistanceData[j+1] - smoothedDistanceData[j]
							if segmentDistanceDiff > 0 {
								segmentGrade := ((smoothedAltitudeData[j+1] - smoothedAltitudeData[j]) / segmentDistanceDiff) * 100
								// Update maxGrade based on the slope type
								// For ascents, we want the maximum grade, for descents, we want the minimum grade
								if *currentSlopeType == DESCENT {
									maxGrade = math.Min(maxGrade, segmentGrade)
								} else {
									maxGrade = math.Max(maxGrade, segmentGrade)
								}
							}
						}

						var averageSpeed float64
						if totalDuration > 0 {
							averageSpeed = totalDistance / float64(totalDuration)
						}

						slopes = append(slopes, Slope{
							Type:          *currentSlopeType,
							StartIndex:    currentSlopeStartIndex,
							EndIndex:      endIndex,
							StartAltitude: startAltitude,
							EndAltitude:   endAltitude,
							Grade:         averageGrade,
							MaxGrade:      maxGrade,
							Distance:      totalDistance,
							Duration:      totalDuration,
							AverageSpeed:  averageSpeed,
						})
					}
				}
			}

			// Start a new segment
			if i == dataSize-1 {
				currentSlopeStartIndex = i
			} else {
				currentSlopeStartIndex = i - 1
			}
			currentSlopeType = &slopeType
		}
	}

	return mergeConsecutiveSegments(slopes)
}

// mergeConsecutiveSegments merges consecutive slope segments of the same type into a single segment.
// Also merges small slopes with different types when they are between two slopes of the same type.
// This is useful to reduce noise and provide a cleaner representation of the activity's slopes.
func mergeConsecutiveSegments(slopes []Slope) []Slope {
	if len(slopes) == 0 {
		return []Slope{}
	}

	// First pass: merge consecutive segments of the same type
	mergedSlopes := make([]Slope, 0)
	currentSlope := slopes[0]

	for i := 1; i < len(slopes); i++ {
		slope := slopes[i]

		// Check if we can merge with the current slope
		if currentSlope.Type == slope.Type {
			// Merge with the current slope
			totalDistance := currentSlope.Distance + slope.Distance
			currentSlope = Slope{
				Type:          currentSlope.Type,
				StartIndex:    currentSlope.StartIndex,
				EndIndex:      slope.EndIndex,
				StartAltitude: currentSlope.StartAltitude,
				EndAltitude:   slope.EndAltitude,
				Grade:         (currentSlope.Grade*currentSlope.Distance + slope.Grade*slope.Distance) / totalDistance,
				MaxGrade:      math.Max(currentSlope.MaxGrade, slope.MaxGrade),
				Distance:      totalDistance,
				Duration:      currentSlope.Duration + slope.Duration,
				AverageSpeed:  (currentSlope.AverageSpeed*currentSlope.Distance + slope.AverageSpeed*slope.Distance) / totalDistance,
			}
		} else {
			// Cannot merge, add the current slope to results and start a new one
			mergedSlopes = append(mergedSlopes, currentSlope)
			currentSlope = slope
		}
	}

	// Remember to add the last slope
	mergedSlopes = append(mergedSlopes, currentSlope)

	// Second pass: merge small slopes between two slopes of the same type
	return mergeSmallIntermediateSlopes(mergedSlopes)
}

// mergeSmallIntermediateSlopes identifies small intermediate slopes and merges them with surrounding slopes of the same type.
// A small slope is considered for merging if it's shorter than 500 m and is between two slopes of the same type.
func mergeSmallIntermediateSlopes(slopes []Slope) []Slope {
	if len(slopes) < 3 {
		return slopes
	}

	result := make([]Slope, 0)
	i := 0

	for i < len(slopes) {
		// Check if we have a pattern: slope1 - smallSlope - slope2 where slope1.type == slope2.type
		if i < len(slopes)-2 &&
			slopes[i].Type == slopes[i+2].Type &&
			slopes[i+1].Type != slopes[i].Type &&
			slopes[i+1].Distance < 500.0 { // Small slope threshold: 500 m

			// Merge all three slopes into one
			slope1 := slopes[i]
			smallSlope := slopes[i+1]
			slope2 := slopes[i+2]

			totalDistance := slope1.Distance + smallSlope.Distance + slope2.Distance
			totalDuration := slope1.Duration + smallSlope.Duration + slope2.Duration

			mergedSlope := Slope{
				Type:          slope1.Type, // Use the type of the surrounding slopes
				StartIndex:    slope1.StartIndex,
				EndIndex:      slope2.EndIndex,
				StartAltitude: slope1.StartAltitude,
				EndAltitude:   slope2.EndAltitude,
				Grade:         (slope1.Grade*slope1.Distance + smallSlope.Grade*smallSlope.Distance + slope2.Grade*slope2.Distance) / totalDistance,
				MaxGrade:      math.Max(math.Max(slope1.MaxGrade, smallSlope.MaxGrade), slope2.MaxGrade),
				Distance:      totalDistance,
				Duration:      totalDuration,
				AverageSpeed:  (slope1.AverageSpeed*slope1.Distance + smallSlope.AverageSpeed*smallSlope.Distance + slope2.AverageSpeed*slope2.Distance) / totalDistance,
			}

			result = append(result, mergedSlope)
			i += 3 // Skip the next two slopes as they've been merged
		} else {
			result = append(result, slopes[i])
			i += 1
		}
	}

	return result
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
