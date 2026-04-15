package domain

type CumulativeDataPerYear struct {
	Distance  map[string]map[string]float64
	Elevation map[string]map[string]float64
}
