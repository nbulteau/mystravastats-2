package business

type DashboardData struct {
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
