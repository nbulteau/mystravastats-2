package dto

type ActivityDto struct {
	ID                               int64   `json:"id"`
	Name                             string  `json:"name"`
	Type                             string  `json:"type"`
	Link                             string  `json:"link"`
	Distance                         int     `json:"distance"`
	ElapsedTime                      int     `json:"elapsed_time"`
	TotalElevationGain               int     `json:"total_elevation_gain"`
	AverageSpeed                     float64 `json:"average_speed"`
	BestTimeForDistanceFor1000m      float64 `json:"best_time_for_distance_for_1000m"`
	BestElevationForDistanceFor500m  float64 `json:"best_elevation_for_distance_for_500m"`
	BestElevationForDistanceFor1000m float64 `json:"best_elevation_for_distance_for_1000m"`
	Date                             string  `json:"date"`
	AverageWatts                     int     `json:"average_watts"`
	WeightedAverageWatts             string  `json:"weighted_average_watts"`
	BestPowerFor20Minutes            string  `json:"best_power_for_20_minutes"`
	BestPowerFor60Minutes            string  `json:"best_power_for_60_minutes"`
	FTP                              string  `json:"ftp"`
}
