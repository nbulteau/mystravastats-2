package business

type SegmentClimbProgression struct {
	Metric                  string
	TargetTypeFilter        string
	WeatherContextAvailable bool
	Targets                 []SegmentClimbTargetSummary
	SelectedTargetId        *int64
	SelectedTargetType      *string
	Attempts                []SegmentClimbAttempt
}

type SegmentClimbTargetSummary struct {
	TargetId       int64
	TargetName     string
	TargetType     string
	ClimbCategory  int
	Distance       float64
	AverageGrade   float64
	AttemptsCount  int
	BestValue      string
	LatestValue    string
	Consistency    string
	AveragePacing  string
	CloseToPrCount int
	RecentTrend    string
}

type SegmentClimbAttempt struct {
	TargetId           int64
	TargetName         string
	TargetType         string
	ActivityDate       string
	ElapsedTimeSeconds int
	SpeedKph           float64
	Distance           float64
	AverageGrade       float64
	ElevationGain      float64
	PrRank             *int
	SetsNewPr          bool
	CloseToPr          bool
	DeltaToPr          string
	WeatherSummary     *string
	Activity           ActivityShort
}
