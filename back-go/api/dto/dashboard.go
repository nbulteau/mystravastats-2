package dto

type DashboardDataDto struct {
	NbActivitiesByYear        map[string]int     `json:"nbActivitiesByYear"`
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
	AverageCadenceByYear      [][]int64          `json:"averageCadenceByYear"`
}

type EddingtonNumberDto struct {
	EddingtonNumber int   `json:"eddingtonNumber"`
	EddingtonList   []int `json:"eddingtonList"`
}

type CumulativeDataPerYearDto struct {
	Distance  map[string]map[string]float64 `json:"distance"`
	Elevation map[string]map[string]float64 `json:"elevation"`
}

type AnnualGoalTargetsDto struct {
	DistanceKm      *float64 `json:"distanceKm"`
	ElevationMeters *int     `json:"elevationMeters"`
	Activities      *int     `json:"activities"`
	ActiveDays      *int     `json:"activeDays"`
	Eddington       *int     `json:"eddington"`
}

type AnnualGoalProgressDto struct {
	Metric                  string               `json:"metric"`
	Label                   string               `json:"label"`
	Unit                    string               `json:"unit"`
	Current                 float64              `json:"current"`
	Target                  float64              `json:"target"`
	ProgressPercent         float64              `json:"progressPercent"`
	ExpectedProgressPercent float64              `json:"expectedProgressPercent"`
	ProjectedEndOfYear      float64              `json:"projectedEndOfYear"`
	RequiredPace            float64              `json:"requiredPace"`
	RequiredPaceUnit        string               `json:"requiredPaceUnit"`
	RequiredWeeklyPace      float64              `json:"requiredWeeklyPace"`
	Last30Days              float64              `json:"last30Days"`
	Last30DaysWeeklyPace    float64              `json:"last30DaysWeeklyPace"`
	WeeklyPaceGap           float64              `json:"weeklyPaceGap"`
	SuggestedTarget         *float64             `json:"suggestedTarget,omitempty"`
	Monthly                 []AnnualGoalMonthDto `json:"monthly"`
	Status                  string               `json:"status"`
}

type AnnualGoalMonthDto struct {
	Month              int     `json:"month"`
	Value              float64 `json:"value"`
	Cumulative         float64 `json:"cumulative"`
	ExpectedCumulative float64 `json:"expectedCumulative"`
}

type AnnualGoalsDto struct {
	Year            int                     `json:"year"`
	ActivityTypeKey string                  `json:"activityTypeKey"`
	Targets         AnnualGoalTargetsDto    `json:"targets"`
	Progress        []AnnualGoalProgressDto `json:"progress"`
}
