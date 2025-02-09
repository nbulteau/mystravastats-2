package business

import (
	"math"
)

type GeoCoordinate struct {
	Latitude  float64
	Longitude float64
}

const (
	equatorialEarthRadius = 6378.1370
	d2r                   = math.Pi / 180.0
)

// HaversineInM calculates the distance (in meters) between two points on Earth using their latitude and longitude.
func (g GeoCoordinate) HaversineInM(lat2, long2 float64) int {
	return int(1000.0 * g.HaversineInKM(lat2, long2))
}

// HaversineInKM calculates the distance (in kilometers) between two points on Earth using their latitude and longitude.
func (g GeoCoordinate) HaversineInKM(lat2, long2 float64) float64 {
	long := (long2 - g.Longitude) * d2r
	lat := (lat2 - g.Latitude) * d2r
	a := math.Sin(lat/2.0)*math.Sin(lat/2.0) + math.Cos(g.Latitude*d2r)*math.Cos(lat2*d2r)*math.Sin(long/2.0)*math.Sin(long/2.0)
	c := 2.0 * math.Atan2(math.Sqrt(a), math.Sqrt(1.0-a))

	return equatorialEarthRadius * c
}

// Match checks if the distance from the geo location is less than 250 meters.
func (g GeoCoordinate) Match(latitude, longitude float64) bool {
	return g.HaversineInM(latitude, longitude) < 250
}
