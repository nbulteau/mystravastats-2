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

// ListSlopes extracts slope segments (ascents/descents/plateaus) from the stream data.
// Parameters:
// - threshold: minimum average grade in percent to classify ascent/descent (default 3.0)
// - minDistance: minimum distance in meters for a segment (default 500.0)
// - climbIndex: minimum "distance × abs(grade)" for ascents (default 3500.0)
// - smoothingWindow: window size for smoothing the raw data (default 20)
func (s *Stream) ListSlopes(threshold, minDistance, climbIndex float64, smoothingWindow int) []Slope {
	slopes := make([]Slope, 0)

	// Ensure required streams are present
	if s == nil || s.Altitude == nil {
		return slopes
	}

	rawAltitude := s.Altitude.Data
	rawDistance := s.Distance.Data
	timeData := s.Time.Data

	if len(rawAltitude) == 0 || len(rawDistance) == 0 || len(timeData) == 0 {
		return slopes
	}

	// Smooth altitude and distance series
	smAltitude := smoothFloat(rawAltitude, smoothingWindow)
	smDistance := smoothFloat(rawDistance, smoothingWindow)

	// Ensure all arrays have a common usable size
	dataSize := minInt(len(smAltitude), len(smDistance), len(timeData))
	if dataSize < 2 {
		return slopes
	}

	currentSlopeStartIndex := 0
	var currentSlopeType *SlopeType

	// Classify a grade value into a slope type
	classify := func(grade float64) SlopeType {
		if grade >= threshold {
			return ASCENT
		}
		if grade <= -threshold {
			return DESCENT
		}
		return PLATEAU
	}

	for i := 1; i < dataSize; i++ {
		altDiff := smAltitude[i] - smAltitude[i-1]
		distDiff := smDistance[i] - smDistance[i-1]
		if distDiff == 0 {
			continue
		}

		grade := (altDiff / distDiff) * 100.0
		sType := classify(grade)

		// Close the current segment if the type changes, or we reached the last point
		if currentSlopeType == nil || *currentSlopeType != sType || i == dataSize-1 {
			// If there is a previous segment to close
			if currentSlopeType != nil {
				endIndex := i - 1
				if i == dataSize-1 {
					endIndex = i
				}
				if endIndex > currentSlopeStartIndex {
					startAlt := smAltitude[currentSlopeStartIndex]
					endAlt := smAltitude[endIndex]
					totalDistance := smDistance[endIndex] - smDistance[currentSlopeStartIndex]
					totalDuration := timeData[endIndex] - timeData[currentSlopeStartIndex]

					averageGrade := 0.0
					if totalDistance > 0 {
						averageGrade = ((endAlt - startAlt) / totalDistance) * 100.0
					}

					// Inclusion criteria depending on the slope type
					shouldInclude := false
					switch *currentSlopeType {
					case ASCENT:
						calculatedClimbIndex := totalDistance * math.Abs(averageGrade)
						shouldInclude = totalDistance >= minDistance &&
							math.Abs(averageGrade) >= threshold &&
							calculatedClimbIndex >= climbIndex
					case DESCENT:
						shouldInclude = totalDistance >= minDistance &&
							math.Abs(averageGrade) >= threshold
					case PLATEAU:
						shouldInclude = totalDistance >= minDistance
					}

					if shouldInclude {
						// Compute maximum grade within the segment using smoothed data
						maxGrade := 0.0
						for j := currentSlopeStartIndex; j < endIndex; j++ {
							segDist := smDistance[j+1] - smDistance[j]
							if segDist > 0 {
								segGrade := ((smAltitude[j+1] - smAltitude[j]) / segDist) * 100.0
								if g := math.Abs(segGrade); g > maxGrade {
									maxGrade = g
								}
							}
						}

						avgSpeed := 0.0
						if totalDuration > 0 {
							avgSpeed = totalDistance / float64(totalDuration)
						}

						slopes = append(slopes, Slope{
							Type:          *currentSlopeType,
							StartIndex:    currentSlopeStartIndex,
							EndIndex:      endIndex,
							StartAltitude: startAlt,
							EndAltitude:   endAlt,
							Grade:         averageGrade,
							MaxGrade:      maxGrade,
							Distance:      totalDistance,
							Duration:      totalDuration,
							AverageSpeed:  avgSpeed,
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
			tmp := sType
			currentSlopeType = &tmp
		}
	}

	return mergeConsecutiveSegments(slopes)
}

// ListSlopesDefault provides a convenient wrapper with default parameters.
func (s *Stream) ListSlopesDefault() []Slope {
	return s.ListSlopes(3.0, 500.0, 3500.0, 20)
}

// smoothFloat applies a moving average smoothing with a centered window.
// Edges are handled with truncated windows.
func smoothFloat(data []float64, window int) []float64 {
	n := len(data)
	if n == 0 || window <= 1 {
		dup := make([]float64, n)
		copy(dup, data)
		return dup
	}
	if window > n {
		window = n
	}

	// Prefix sums for O(1) window sums
	prefix := make([]float64, n+1)
	for i := 0; i < n; i++ {
		prefix[i+1] = prefix[i] + data[i]
	}

	out := make([]float64, n)
	half := window / 2
	for i := 0; i < n; i++ {
		start := i - half
		end := i + half
		// For even window sizes, this yields a window size of ~window
		if start < 0 {
			start = 0
		}
		if end >= n {
			end = n - 1
		}
		count := float64(end - start + 1)
		sum := prefix[end+1] - prefix[start]
		out[i] = sum / count
	}
	return out
}

// mergeConsecutiveSegments merges adjacent segments of the same type.
// Grades and average speeds are distance-weighted; max grade is the maximum of both.
func mergeConsecutiveSegments(slopes []Slope) []Slope {
	if len(slopes) == 0 {
		return nil
	}

	merged := make([]Slope, 0, len(slopes))
	current := slopes[0]

	for i := 1; i < len(slopes); i++ {
		s := slopes[i]
		if current.Type == s.Type {
			totalDistance := current.Distance + s.Distance

			weightedGrade := 0.0
			weightedSpeed := 0.0
			if totalDistance > 0 {
				weightedGrade = (current.Grade*current.Distance + s.Grade*s.Distance) / totalDistance
				weightedSpeed = (current.AverageSpeed*current.Distance + s.AverageSpeed*s.Distance) / totalDistance
			}

			if s.MaxGrade > current.MaxGrade {
				current.MaxGrade = s.MaxGrade
			}
			current.EndIndex = s.EndIndex
			current.EndAltitude = s.EndAltitude
			current.Distance = totalDistance
			current.Duration += s.Duration
			current.Grade = weightedGrade
			current.AverageSpeed = weightedSpeed
		} else {
			merged = append(merged, current)
			current = s
		}
	}
	merged = append(merged, current)
	return merged
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
