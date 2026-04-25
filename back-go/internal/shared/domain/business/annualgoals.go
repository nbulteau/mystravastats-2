package business

type AnnualGoalMetric string

const (
	AnnualGoalMetricDistanceKm        AnnualGoalMetric = "DISTANCE_KM"
	AnnualGoalMetricElevationMeters   AnnualGoalMetric = "ELEVATION_METERS"
	AnnualGoalMetricMovingTimeSeconds AnnualGoalMetric = "MOVING_TIME_SECONDS"
	AnnualGoalMetricActivities        AnnualGoalMetric = "ACTIVITIES"
	AnnualGoalMetricActiveDays        AnnualGoalMetric = "ACTIVE_DAYS"
	AnnualGoalMetricEddington         AnnualGoalMetric = "EDDINGTON"
)

type AnnualGoalStatus string

const (
	AnnualGoalStatusNotSet  AnnualGoalStatus = "NOT_SET"
	AnnualGoalStatusAhead   AnnualGoalStatus = "AHEAD"
	AnnualGoalStatusOnTrack AnnualGoalStatus = "ON_TRACK"
	AnnualGoalStatusBehind  AnnualGoalStatus = "BEHIND"
)

type AnnualGoalTargets struct {
	DistanceKm        *float64 `json:"distanceKm,omitempty"`
	ElevationMeters   *int     `json:"elevationMeters,omitempty"`
	MovingTimeSeconds *int     `json:"movingTimeSeconds,omitempty"`
	Activities        *int     `json:"activities,omitempty"`
	ActiveDays        *int     `json:"activeDays,omitempty"`
	Eddington         *int     `json:"eddington,omitempty"`
}

type AnnualGoalProgress struct {
	Metric                  AnnualGoalMetric `json:"metric"`
	Label                   string           `json:"label"`
	Unit                    string           `json:"unit"`
	Current                 float64          `json:"current"`
	Target                  float64          `json:"target"`
	ProgressPercent         float64          `json:"progressPercent"`
	ExpectedProgressPercent float64          `json:"expectedProgressPercent"`
	ProjectedEndOfYear      float64          `json:"projectedEndOfYear"`
	RequiredPace            float64          `json:"requiredPace"`
	RequiredPaceUnit        string           `json:"requiredPaceUnit"`
	Status                  AnnualGoalStatus `json:"status"`
}

type AnnualGoals struct {
	Year            int                  `json:"year"`
	ActivityTypeKey string               `json:"activityTypeKey"`
	Targets         AnnualGoalTargets    `json:"targets"`
	Progress        []AnnualGoalProgress `json:"progress"`
}
