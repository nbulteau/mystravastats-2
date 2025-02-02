package dto

type DashboardDataDto struct {
	NbActivities           map[string]int     `json:"nbActivities"`
	TotalDistanceByYear    map[string]float64 `json:"totalDistanceByYear"`
	AverageDistanceByYear  map[string]float64 `json:"averageDistanceByYear"`
	MaxDistanceByYear      map[string]float64 `json:"maxDistanceByYear"`
	TotalElevationByYear   map[string]int     `json:"totalElevationByYear"`
	AverageElevationByYear map[string]int     `json:"averageElevationByYear"`
	MaxElevationByYear     map[string]int     `json:"maxElevationByYear"`
	AverageSpeedByYear     map[string]float64 `json:"averageSpeedByYear"`
	MaxSpeedByYear         map[string]float64 `json:"maxSpeedByYear"`
	AverageHeartRateByYear map[string]int     `json:"averageHeartRateByYear"`
	MaxHeartRateByYear     map[string]float64 `json:"maxHeartRateByYear"`
	AverageWattsByYear     map[string]float64 `json:"averageWattsByYear"`
	MaxWattsByYear         map[string]float64 `json:"maxWattsByYear"`
	AverageCadence         [][]int64          `json:"averageCadence"`
}

type EddingtonNumberDto struct {
	EddingtonNumber int   `json:"eddingtonNumber"`
	EddingtonList   []int `json:"eddingtonList"`
}

type CumulativeDataPerYearDto struct {
	Distance  map[string]map[string]float64 `json:"distance"`
	Elevation map[string]map[string]float64 `json:"elevation"`
}
