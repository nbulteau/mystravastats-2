package dto

type StatisticDto struct {
	Label    string            `json:"label"`
	Value    string            `json:"value"`
	Activity *ActivityShortDto `json:"activity,omitempty"`
}

type ActivityShortDto struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type PersonalRecordTimelineDto struct {
	MetricKey     string           `json:"metricKey"`
	MetricLabel   string           `json:"metricLabel"`
	ActivityDate  string           `json:"activityDate"`
	Value         string           `json:"value"`
	PreviousValue *string          `json:"previousValue,omitempty"`
	Improvement   *string          `json:"improvement,omitempty"`
	Activity      ActivityShortDto `json:"activity"`
}

type SegmentClimbProgressionDto struct {
	Metric                  string                         `json:"metric"`
	TargetTypeFilter        string                         `json:"targetTypeFilter"`
	WeatherContextAvailable bool                           `json:"weatherContextAvailable"`
	Targets                 []SegmentClimbTargetSummaryDto `json:"targets"`
	SelectedTargetId        *int64                         `json:"selectedTargetId,omitempty"`
	SelectedTargetType      *string                        `json:"selectedTargetType,omitempty"`
	Attempts                []SegmentClimbAttemptDto       `json:"attempts"`
}

type SegmentClimbTargetSummaryDto struct {
	TargetId       int64   `json:"targetId"`
	TargetName     string  `json:"targetName"`
	TargetType     string  `json:"targetType"`
	ClimbCategory  int     `json:"climbCategory"`
	Distance       float64 `json:"distance"`
	AverageGrade   float64 `json:"averageGrade"`
	AttemptsCount  int     `json:"attemptsCount"`
	BestValue      string  `json:"bestValue"`
	LatestValue    string  `json:"latestValue"`
	Consistency    string  `json:"consistency"`
	AveragePacing  string  `json:"averagePacing"`
	CloseToPrCount int     `json:"closeToPrCount"`
	RecentTrend    string  `json:"recentTrend"`
}

type SegmentClimbAttemptDto struct {
	TargetId           int64            `json:"targetId"`
	TargetName         string           `json:"targetName"`
	TargetType         string           `json:"targetType"`
	ActivityDate       string           `json:"activityDate"`
	ElapsedTimeSeconds int              `json:"elapsedTimeSeconds"`
	SpeedKph           float64          `json:"speedKph"`
	Distance           float64          `json:"distance"`
	AverageGrade       float64          `json:"averageGrade"`
	ElevationGain      float64          `json:"elevationGain"`
	PrRank             *int             `json:"prRank,omitempty"`
	SetsNewPr          bool             `json:"setsNewPr"`
	CloseToPr          bool             `json:"closeToPr"`
	DeltaToPr          string           `json:"deltaToPr"`
	WeatherSummary     *string          `json:"weatherSummary,omitempty"`
	Activity           ActivityShortDto `json:"activity"`
}

type HeartRateZoneSettingsDto struct {
	MaxHr       *int `json:"maxHr,omitempty"`
	ThresholdHr *int `json:"thresholdHr,omitempty"`
	ReserveHr   *int `json:"reserveHr,omitempty"`
}

type ResolvedHeartRateZoneSettingsDto struct {
	MaxHr       int    `json:"maxHr"`
	ThresholdHr *int   `json:"thresholdHr,omitempty"`
	ReserveHr   *int   `json:"reserveHr,omitempty"`
	Method      string `json:"method"`
	Source      string `json:"source"`
}

type HeartRateZoneDistributionDto struct {
	Zone       string  `json:"zone"`
	Label      string  `json:"label"`
	Seconds    int     `json:"seconds"`
	Percentage float64 `json:"percentage"`
}

type HeartRateZoneActivitySummaryDto struct {
	Activity            ActivityShortDto               `json:"activity"`
	ActivityDate        string                         `json:"activityDate"`
	TotalTrackedSeconds int                            `json:"totalTrackedSeconds"`
	EasySeconds         int                            `json:"easySeconds"`
	HardSeconds         int                            `json:"hardSeconds"`
	EasyHardRatio       *float64                       `json:"easyHardRatio,omitempty"`
	Zones               []HeartRateZoneDistributionDto `json:"zones"`
}

type HeartRateZonePeriodSummaryDto struct {
	Period              string                         `json:"period"`
	TotalTrackedSeconds int                            `json:"totalTrackedSeconds"`
	EasySeconds         int                            `json:"easySeconds"`
	HardSeconds         int                            `json:"hardSeconds"`
	EasyHardRatio       *float64                       `json:"easyHardRatio,omitempty"`
	Zones               []HeartRateZoneDistributionDto `json:"zones"`
}

type HeartRateZoneAnalysisDto struct {
	Settings            HeartRateZoneSettingsDto          `json:"settings"`
	ResolvedSettings    *ResolvedHeartRateZoneSettingsDto `json:"resolvedSettings,omitempty"`
	HasHeartRateData    bool                              `json:"hasHeartRateData"`
	TotalTrackedSeconds int                               `json:"totalTrackedSeconds"`
	EasyHardRatio       *float64                          `json:"easyHardRatio,omitempty"`
	Zones               []HeartRateZoneDistributionDto    `json:"zones"`
	Activities          []HeartRateZoneActivitySummaryDto `json:"activities"`
	ByMonth             []HeartRateZonePeriodSummaryDto   `json:"byMonth"`
	ByYear              []HeartRateZonePeriodSummaryDto   `json:"byYear"`
}
