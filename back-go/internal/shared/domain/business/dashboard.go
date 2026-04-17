package business

type DashboardData struct {
	NbActivities              map[string]int     `json:"nbActivities"`
	ActiveDaysByYear          map[string]int     `json:"activeDaysByYear"`
	ConsistencyByYear         map[string]float64 `json:"consistencyByYear"`
	MovingTimeByYear          map[string]int     `json:"movingTimeByYear"`
	TotalDistanceByYear       map[string]float64 `json:"totalDistanceByYear"`
	AverageDistanceByYear     map[string]float64 `json:"averageDistanceByYear"`
	MaxDistanceByYear         map[string]float64 `json:"maxDistanceByYear"`
	TotalElevationByYear      map[string]int     `json:"totalElevationByYear"`
	AverageElevationByYear    map[string]int     `json:"averageElevationByYear"`
	MaxElevationByYear        map[string]int     `json:"maxElevationByYear"`
	ElevationEfficiencyByYear map[string]float64 `json:"elevationEfficiencyByYear"`
	AverageSpeedByYear        map[string]float64 `json:"averageSpeedByYear"`
	MaxSpeedByYear            map[string]float64 `json:"maxSpeedByYear"`
	AverageHeartRateByYear    map[string]int     `json:"averageHeartRateByYear"`
	MaxHeartRateByYear        map[string]float64 `json:"maxHeartRateByYear"`
	AverageWattsByYear        map[string]float64 `json:"averageWattsByYear"`
	MaxWattsByYear            map[string]float64 `json:"maxWattsByYear"`
	AverageCadence            [][]int64          `json:"averageCadence"`
}
