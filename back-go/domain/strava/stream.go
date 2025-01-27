package strava

import "math"

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
	Data         []int  `json:"data"`
	OriginalSize int    `json:"original_size"`
	Resolution   string `json:"resolution"`
	SeriesType   string `json:"series_type"`
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
	Data         []float32 `json:"data"`
	OriginalSize int       `json:"original_size"`
	Resolution   string    `json:"resolution"`
	SeriesType   string    `json:"series_type"`
}

type SmoothGradeStream struct {
	Data         []float32 `json:"data"`
	OriginalSize int       `json:"original_size"`
	Resolution   string    `json:"resolution"`
	SeriesType   string    `json:"series_type"`
}

func NewAltitudeStream(data []float64) *AltitudeStream {
	return &AltitudeStream{
		Data:         data,
		OriginalSize: len(data),
		Resolution:   "high",
		SeriesType:   "distance",
	}
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
	φ1 := toRadians(lat1)
	φ2 := toRadians(lat2)
	Δφ := toRadians(lat2 - lat1)
	Δλ := toRadians(lon2 - lon1)

	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) +
		math.Cos(φ1)*math.Cos(φ2)*
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
