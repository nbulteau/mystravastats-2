package business

type DashboardData struct {
	NbActivities                      map[string]int     `json:"nbActivities"`
	ActiveDaysByYear                  map[string]int     `json:"activeDaysByYear"`
	ConsistencyByYear                 map[string]float64 `json:"consistencyByYear"`
	MovingTimeByYear                  map[string]int     `json:"movingTimeByYear"`
	TotalDistanceByYear               map[string]float64 `json:"totalDistanceByYear"`
	AverageDistanceByYear             map[string]float64 `json:"averageDistanceByYear"`
	MaxDistanceByYear                 map[string]float64 `json:"maxDistanceByYear"`
	MaxDistanceDateByYear             map[string]string  `json:"maxDistanceDateByYear"`
	AverageDistanceByActiveDayByYear  map[string]float64 `json:"averageDistanceByActiveDayByYear"`
	MaxDistanceByActiveDayByYear      map[string]float64 `json:"maxDistanceByActiveDayByYear"`
	MaxDistanceByActiveDayDateByYear  map[string]string  `json:"maxDistanceByActiveDayDateByYear"`
	TotalElevationByYear              map[string]int     `json:"totalElevationByYear"`
	AverageElevationByYear            map[string]int     `json:"averageElevationByYear"`
	MaxElevationByYear                map[string]int     `json:"maxElevationByYear"`
	MaxElevationDateByYear            map[string]string  `json:"maxElevationDateByYear"`
	AverageElevationByActiveDayByYear map[string]int     `json:"averageElevationByActiveDayByYear"`
	MaxElevationByActiveDayByYear     map[string]int     `json:"maxElevationByActiveDayByYear"`
	MaxElevationByActiveDayDateByYear map[string]string  `json:"maxElevationByActiveDayDateByYear"`
	ElevationEfficiencyByYear         map[string]float64 `json:"elevationEfficiencyByYear"`
	AverageSpeedByYear                map[string]float64 `json:"averageSpeedByYear"`
	MaxSpeedByYear                    map[string]float64 `json:"maxSpeedByYear"`
	MaxSpeedDateByYear                map[string]string  `json:"maxSpeedDateByYear"`
	AverageHeartRateByYear            map[string]int     `json:"averageHeartRateByYear"`
	MaxHeartRateByYear                map[string]float64 `json:"maxHeartRateByYear"`
	MaxHeartRateDateByYear            map[string]string  `json:"maxHeartRateDateByYear"`
	AverageWattsByYear                map[string]float64 `json:"averageWattsByYear"`
	MaxWattsByYear                    map[string]float64 `json:"maxWattsByYear"`
	MaxWattsDateByYear                map[string]string  `json:"maxWattsDateByYear"`
	DeviceAverageWattsByYear          map[string]float64 `json:"deviceAverageWattsByYear"`
	DeviceMaxWattsByYear              map[string]float64 `json:"deviceMaxWattsByYear"`
	DeviceMaxWattsDateByYear          map[string]string  `json:"deviceMaxWattsDateByYear"`
	AverageCadence                    [][]int64          `json:"averageCadence"`
}
