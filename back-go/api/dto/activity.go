package dto

import "time"

type ActivityDto struct {
	Id                               int64   `json:"id"`
	Name                             string  `json:"name"`
	Type                             string  `json:"type"`
	Commute                          bool    `json:"commute"`
	Link                             string  `json:"link"`
	Distance                         int     `json:"distance"`
	ElapsedTime                      int     `json:"elapsedTime"`
	MovingTime                       int     `json:"movingTime"`
	TotalElevationGain               int     `json:"totalElevationGain"`
	AverageSpeed                     float64 `json:"averageSpeed"`
	AverageHeartrate                 int     `json:"averageHeartrate"`
	BestSpeedForDistanceFor1000m     float64 `json:"bestSpeedForDistanceFor1000m"`
	BestElevationForDistanceFor500m  float64 `json:"bestElevationForDistanceFor500m"`
	BestElevationForDistanceFor1000m float64 `json:"bestElevationForDistanceFor1000m"`
	Date                             string  `json:"date"`
	AverageWatts                     int     `json:"averageWatts"`
	WeightedAverageWatts             int     `json:"weightedAverageWatts"`
	BestPowerFor20Minutes            int     `json:"bestPowerFor20Minutes"`
	BestPowerFor60Minutes            int     `json:"bestPowerFor60Minutes"`
	FTP                              int     `json:"ftp"`
	BadgeEffortSeconds               int     `json:"badgeEffortSeconds,omitempty"`
}

type DetailedActivityDto struct {
	AverageCadence       int                    `json:"averageCadence"`
	AverageHeartrate     int                    `json:"averageHeartrate"`
	AverageWatts         int                    `json:"averageWatts"`
	AverageSpeed         float32                `json:"averageSpeed"`
	Calories             float64                `json:"calories"`
	Commute              bool                   `json:"commute"`
	DeviceWatts          bool                   `json:"deviceWatts"`
	Distance             float64                `json:"distance"`
	ElapsedTime          int                    `json:"elapsedTime"`
	ElevHigh             float64                `json:"elevHigh"`
	ID                   int64                  `json:"id"`
	Kilojoules           float64                `json:"kilojoules"`
	MaxHeartrate         int                    `json:"maxHeartrate"`
	MaxSpeed             float32                `json:"maxSpeed"`
	MaxWatts             int                    `json:"maxWatts"`
	MovingTime           int                    `json:"movingTime"`
	Name                 string                 `json:"name"`
	ActivityEfforts      []ActivityEffortDto    `json:"activityEfforts"`
	ActivityComparison   *ActivityComparisonDto `json:"activityComparison,omitempty"`
	StartDate            time.Time              `json:"startDate"`
	StartDateLocal       time.Time              `json:"startDateLocal"`
	StartLatlng          []float64              `json:"startLatlng"`
	Stream               *StreamDto             `json:"stream"`
	SufferScore          *float64               `json:"sufferScore"`
	TotalDescent         float64                `json:"totalDescent"`
	TotalElevationGain   int                    `json:"totalElevationGain"`
	Type                 string                 `json:"type"`
	WeightedAverageWatts int                    `json:"weightedAverageWatts"`
}

type ActivityComparisonDto struct {
	Status            string                          `json:"status"`
	Label             string                          `json:"label"`
	Criteria          ActivityComparisonCriteriaDto   `json:"criteria"`
	Baseline          ActivityComparisonBaselineDto   `json:"baseline"`
	Deltas            ActivityComparisonDeltasDto     `json:"deltas"`
	SimilarActivities []ActivityComparisonActivityDto `json:"similarActivities"`
	CommonSegments    []ActivityComparisonSegmentDto  `json:"commonSegments"`
}

type ActivityComparisonCriteriaDto struct {
	ActivityType string `json:"activityType"`
	Year         int    `json:"year"`
	SampleSize   int    `json:"sampleSize"`
}

type ActivityComparisonBaselineDto struct {
	Distance         float64 `json:"distance"`
	ElevationGain    float64 `json:"elevationGain"`
	MovingTime       int     `json:"movingTime"`
	AverageSpeed     float64 `json:"averageSpeed"`
	AverageHeartrate float64 `json:"averageHeartrate"`
	AverageWatts     float64 `json:"averageWatts"`
	AverageCadence   float64 `json:"averageCadence"`
}

type ActivityComparisonDeltasDto struct {
	Distance         float64 `json:"distance"`
	ElevationGain    float64 `json:"elevationGain"`
	MovingTime       int     `json:"movingTime"`
	AverageSpeed     float64 `json:"averageSpeed"`
	AverageSpeedPct  float64 `json:"averageSpeedPct"`
	AverageHeartrate float64 `json:"averageHeartrate"`
	AverageWatts     float64 `json:"averageWatts"`
	AverageCadence   float64 `json:"averageCadence"`
}

type ActivityComparisonActivityDto struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	Date             string  `json:"date"`
	Distance         float64 `json:"distance"`
	ElevationGain    float64 `json:"elevationGain"`
	MovingTime       int     `json:"movingTime"`
	AverageSpeed     float64 `json:"averageSpeed"`
	AverageHeartrate float64 `json:"averageHeartrate"`
	AverageWatts     float64 `json:"averageWatts"`
	AverageCadence   float64 `json:"averageCadence"`
	SimilarityScore  float64 `json:"similarityScore"`
}

type ActivityComparisonSegmentDto struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	MatchCount    int      `json:"matchCount"`
	ActivityIDs   []int64  `json:"activityIds"`
	ActivityNames []string `json:"activityNames"`
}

type ActivityEffortDto struct {
	ID            string   `json:"id"`
	Label         string   `json:"label"`
	Distance      float64  `json:"distance"`
	Seconds       int      `json:"seconds"`
	DeltaAltitude float64  `json:"deltaAltitude"`
	IdxStart      int      `json:"idxStart"`
	IdxEnd        int      `json:"idxEnd"`
	AveragePower  *float64 `json:"averagePower"`
	Description   string   `json:"description"`
}

type StreamDto struct {
	Distance       []float64   `json:"distance"`
	Time           []int       `json:"time"`
	Latlng         [][]float64 `json:"latlng,omitempty"`
	Heartrate      []int       `json:"heartrate,omitempty"`
	Moving         []bool      `json:"moving,omitempty"`
	Altitude       []float64   `json:"altitude,omitempty"`
	Watts          []float64   `json:"watts,omitempty"`
	VelocitySmooth []float64   `json:"velocitySmooth,omitempty"`
}
