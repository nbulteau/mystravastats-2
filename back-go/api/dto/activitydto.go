package dto

type ActivityDto struct {
	ID                               int64   `json:"id"`
	Name                             string  `json:"name"`
	Type                             string  `json:"type"`
	Link                             string  `json:"link"`
	Distance                         int     `json:"distance"`
	ElapsedTime                      int     `json:"elapsedTime"`
	TotalElevationGain               int     `json:"totalElevationGain"`
	AverageSpeed                     float64 `json:"averageSpeed"`
	BestTimeForDistanceFor1000m      float64 `json:"bestTimeForDistanceFor1000m"`
	BestElevationForDistanceFor500m  float64 `json:"bestElevationForDistanceFor500m"`
	BestElevationForDistanceFor1000m float64 `json:"bestElevationForDistanceFor1000m"`
	Date                             string  `json:"date"`
	AverageWatts                     int     `json:"averageWatts"`
	WeightedAverageWatts             string  `json:"weightedAverageWatts"`
	BestPowerFor20Minutes            string  `json:"bestPowerFor20Minutes"`
	BestPowerFor60Minutes            string  `json:"bestPowerFor60Minutes"`
	FTP                              string  `json:"ftp"`
}
