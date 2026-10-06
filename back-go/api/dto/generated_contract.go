// Code generated from docs/api/openapi.json. DO NOT EDIT.

package dto

type ContractApiError struct {
	Message     string `json:"message"`
	Code        int64  `json:"code"`
	Description string `json:"description"`
	RequestId   string `json:"requestId"`
}

type ContractRouteCoordinate struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type ContractRouteGenerationDiagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ContractSourceModeSelection struct {
	Mode *string `json:"mode,omitempty"`
	Path *string `json:"path,omitempty"`
}

type ContractOperationStatus struct {
	Success bool    `json:"success"`
	Message *string `json:"message,omitempty"`
}

type ContractActivitySummary struct {
	Id                               int64   `json:"id"`
	Name                             string  `json:"name"`
	Type                             string  `json:"type"`
	Commute                          bool    `json:"commute"`
	Link                             string  `json:"link"`
	Distance                         int64   `json:"distance"`
	ElapsedTime                      int64   `json:"elapsedTime"`
	MovingTime                       int64   `json:"movingTime"`
	TotalElevationGain               int64   `json:"totalElevationGain"`
	AverageSpeed                     float64 `json:"averageSpeed"`
	AverageHeartrate                 int64   `json:"averageHeartrate"`
	BestSpeedForDistanceFor1000m     float64 `json:"bestSpeedForDistanceFor1000m"`
	BestElevationForDistanceFor500m  float64 `json:"bestElevationForDistanceFor500m"`
	BestElevationForDistanceFor1000m float64 `json:"bestElevationForDistanceFor1000m"`
	Date                             string  `json:"date"`
	AverageWatts                     int64   `json:"averageWatts"`
	WeightedAverageWatts             int64   `json:"weightedAverageWatts"`
	BestPowerFor20Minutes            int64   `json:"bestPowerFor20Minutes"`
	BestPowerFor60Minutes            int64   `json:"bestPowerFor60Minutes"`
	BestPowerFor20minutes            int64   `json:"bestPowerFor20minutes"`
	BestPowerFor60minutes            int64   `json:"bestPowerFor60minutes"`
	Ftp                              int64   `json:"ftp"`
	BadgeEffortSeconds               *int64  `json:"badgeEffortSeconds,omitempty"`
}

type ContractRouteGenerationScore struct {
	Global      float64 `json:"global"`
	Distance    float64 `json:"distance"`
	Elevation   float64 `json:"elevation"`
	Duration    float64 `json:"duration"`
	Direction   float64 `json:"direction"`
	Shape       float64 `json:"shape"`
	RoadFitness float64 `json:"roadFitness"`
}

type ContractGeneratedRoute struct {
	RouteId              string                       `json:"routeId"`
	Title                string                       `json:"title"`
	VariantType          string                       `json:"variantType"`
	RouteType            *string                      `json:"routeType,omitempty"`
	DistanceKm           float64                      `json:"distanceKm"`
	ElevationGainM       float64                      `json:"elevationGainM"`
	DurationSec          int64                        `json:"durationSec"`
	EstimatedDurationSec int64                        `json:"estimatedDurationSec"`
	Score                ContractRouteGenerationScore `json:"score"`
	Reasons              []string                     `json:"reasons"`
	PreviewLatLng        [][]float64                  `json:"previewLatLng"`
	Start                *ContractRouteCoordinate     `json:"start,omitempty"`
	End                  *ContractRouteCoordinate     `json:"end,omitempty"`
	ActivityId           *int64                       `json:"activityId,omitempty"`
	IsRoadGraphGenerated bool                         `json:"isRoadGraphGenerated"`
}

type ContractGenerateRoutesResponse struct {
	Routes      []ContractGeneratedRoute            `json:"routes"`
	Diagnostics []ContractRouteGenerationDiagnostic `json:"diagnostics,omitempty"`
}

type ContractAthleteFtpSetting struct {
	EffectiveFrom string `json:"effectiveFrom"`
	Ftp           int64  `json:"ftp"`
}

type ContractAthletePerformanceSettings struct {
	FtpHistory []ContractAthleteFtpSetting `json:"ftpHistory"`
	WeightKg   *float64                    `json:"weightKg,omitempty"`
}

type ContractDataQualityIssue struct {
	Id                string  `json:"id"`
	Source            string  `json:"source"`
	ActivityId        *int64  `json:"activityId,omitempty"`
	ActivityName      *string `json:"activityName,omitempty"`
	Severity          string  `json:"severity"`
	Category          string  `json:"category"`
	Field             string  `json:"field"`
	Message           string  `json:"message"`
	ExcludedFromStats *bool   `json:"excludedFromStats,omitempty"`
}

type ContractDataQualitySummary struct {
	Status              string  `json:"status"`
	Provider            *string `json:"provider,omitempty"`
	IssueCount          int64   `json:"issueCount"`
	ImpactedActivities  int64   `json:"impactedActivities"`
	ExcludedActivities  int64   `json:"excludedActivities"`
	SafeCorrectionCount *int64  `json:"safeCorrectionCount,omitempty"`
	ManualReviewCount   *int64  `json:"manualReviewCount,omitempty"`
}

type ContractDataQualityReport struct {
	GeneratedAt *string                    `json:"generatedAt,omitempty"`
	Summary     ContractDataQualitySummary `json:"summary"`
	Issues      []ContractDataQualityIssue `json:"issues"`
}

type ContractHeartRateZoneSettings struct {
	MaxHr       *int64 `json:"maxHr,omitempty"`
	ThresholdHr *int64 `json:"thresholdHr,omitempty"`
	ReserveHr   *int64 `json:"reserveHr,omitempty"`
}

type ContractDataQualityExclusionRequest struct {
	Reason *string `json:"reason,omitempty"`
}

type ContractLocalDataBackup struct {
	Version    int64          `json:"version"`
	ExportedAt string         `json:"exportedAt"`
	AthleteId  string         `json:"athleteId"`
	Files      map[string]any `json:"files"`
}

type ContractLocalDataRestoreResult struct {
	Restored []string `json:"restored"`
}

type ContractStravaOAuthStartRequest struct {
	Path         *string `json:"path,omitempty"`
	ClientId     *string `json:"clientId,omitempty"`
	ClientSecret *string `json:"clientSecret,omitempty"`
	UseCache     *bool   `json:"useCache,omitempty"`
}

type ContractGenerateShapeRoutesRequest struct {
	ShapeInputType string                   `json:"shapeInputType"`
	ShapeData      string                   `json:"shapeData"`
	StartPoint     *ContractRouteCoordinate `json:"startPoint,omitempty"`
	RouteType      *string                  `json:"routeType,omitempty"`
	VariantCount   *int64                   `json:"variantCount,omitempty"`
}

type ContractEditGeneratedRouteRequest struct {
	RouteType     *string                   `json:"routeType,omitempty"`
	ControlPoints []ContractRouteCoordinate `json:"controlPoints"`
}

type ContractEditGeneratedRouteResponse struct {
	Route         *ContractGeneratedRoute             `json:"route,omitempty"`
	ControlPoints []ContractRouteCoordinate           `json:"controlPoints"`
	Diagnostics   []ContractRouteGenerationDiagnostic `json:"diagnostics"`
}

type ContractGearMaintenanceRequest struct {
	GearId    *string  `json:"gearId,omitempty"`
	Component *string  `json:"component,omitempty"`
	Action    *string  `json:"action,omitempty"`
	Operation *string  `json:"operation,omitempty"`
	Date      *string  `json:"date,omitempty"`
	Distance  *float64 `json:"distance,omitempty"`
	Note      *string  `json:"note,omitempty"`
}

type ContractSourceSyncResult struct {
	Status      string         `json:"status"`
	Reason      string         `json:"reason"`
	Message     string         `json:"message"`
	StartedAt   string         `json:"startedAt"`
	CompletedAt string         `json:"completedAt"`
	DurationMs  int64          `json:"durationMs"`
	Reloaded    bool           `json:"reloaded"`
	Fit         map[string]any `json:"fit"`
}

type ContractDetailedActivity struct {
	AverageCadence       int64                         `json:"averageCadence"`
	AverageHeartrate     int64                         `json:"averageHeartrate"`
	AverageWatts         int64                         `json:"averageWatts"`
	AverageSpeed         float64                       `json:"averageSpeed"`
	Calories             float64                       `json:"calories"`
	Commute              bool                          `json:"commute"`
	DeviceWatts          bool                          `json:"deviceWatts"`
	Distance             float64                       `json:"distance"`
	ElapsedTime          int64                         `json:"elapsedTime"`
	ElevHigh             float64                       `json:"elevHigh"`
	Id                   int64                         `json:"id"`
	Link                 string                        `json:"link"`
	Kilojoules           float64                       `json:"kilojoules"`
	MaxHeartrate         int64                         `json:"maxHeartrate"`
	MaxSpeed             float64                       `json:"maxSpeed"`
	MaxWatts             int64                         `json:"maxWatts"`
	MovingTime           int64                         `json:"movingTime"`
	Name                 string                        `json:"name"`
	ActivityEfforts      []ContractActivityEffort      `json:"activityEfforts"`
	StravaSegmentEfforts []ContractStravaSegmentEffort `json:"stravaSegmentEfforts"`
	ActivityComparison   *ContractActivityComparison   `json:"activityComparison,omitempty"`
	StartDate            string                        `json:"startDate"`
	StartDateLocal       string                        `json:"startDateLocal"`
	StartLatlng          []float64                     `json:"startLatlng"`
	Source               *ContractActivitySource       `json:"source,omitempty"`
	Stream               *ContractActivityStream       `json:"stream"`
	SufferScore          *float64                      `json:"sufferScore"`
	TotalDescent         float64                       `json:"totalDescent"`
	TotalElevationGain   int64                         `json:"totalElevationGain"`
	Type                 string                        `json:"type"`
	SportType            string                        `json:"sportType"`
	WeightedAverageWatts int64                         `json:"weightedAverageWatts"`
}

type ContractActivitySource struct {
	PrimaryProvider string                           `json:"primaryProvider"`
	PrimaryId       int64                            `json:"primaryId"`
	StreamProvider  *string                          `json:"streamProvider,omitempty"`
	MergeConfidence *string                          `json:"mergeConfidence,omitempty"`
	Sources         []ContractActivitySourceRef      `json:"sources,omitempty"`
	Conflicts       []ContractActivitySourceConflict `json:"conflicts,omitempty"`
	FieldSources    map[string]string                `json:"fieldSources,omitempty"`
}

type ContractActivitySourceRef struct {
	Provider       string  `json:"provider"`
	ActivityId     int64   `json:"activityId"`
	StartDateLocal string  `json:"startDateLocal"`
	Distance       float64 `json:"distance"`
	MovingTime     int64   `json:"movingTime"`
	HasStream      bool    `json:"hasStream"`
}

type ContractActivitySourceConflict struct {
	Field   string `json:"field"`
	Primary string `json:"primary"`
	Other   string `json:"other"`
	Source  string `json:"source"`
}

type ContractActivityComparison struct {
	Status            string                               `json:"status"`
	Label             string                               `json:"label"`
	Criteria          ContractActivityComparisonCriteria   `json:"criteria"`
	Baseline          ContractActivityComparisonBaseline   `json:"baseline"`
	Deltas            ContractActivityComparisonDeltas     `json:"deltas"`
	SimilarActivities []ContractActivityComparisonActivity `json:"similarActivities"`
	CommonSegments    []ContractActivityComparisonSegment  `json:"commonSegments"`
}

type ContractActivityComparisonCriteria struct {
	ActivityType string `json:"activityType"`
	Year         int64  `json:"year"`
	SampleSize   int64  `json:"sampleSize"`
}

type ContractActivityComparisonBaseline struct {
	Distance         float64 `json:"distance"`
	ElevationGain    float64 `json:"elevationGain"`
	MovingTime       int64   `json:"movingTime"`
	AverageSpeed     float64 `json:"averageSpeed"`
	AverageHeartrate float64 `json:"averageHeartrate"`
	AverageWatts     float64 `json:"averageWatts"`
	AverageCadence   float64 `json:"averageCadence"`
}

type ContractActivityComparisonDeltas struct {
	Distance         float64 `json:"distance"`
	ElevationGain    float64 `json:"elevationGain"`
	MovingTime       int64   `json:"movingTime"`
	AverageSpeed     float64 `json:"averageSpeed"`
	AverageSpeedPct  float64 `json:"averageSpeedPct"`
	AverageHeartrate float64 `json:"averageHeartrate"`
	AverageWatts     float64 `json:"averageWatts"`
	AverageCadence   float64 `json:"averageCadence"`
}

type ContractActivityComparisonActivity struct {
	Id               int64   `json:"id"`
	Name             string  `json:"name"`
	Date             string  `json:"date"`
	Distance         float64 `json:"distance"`
	ElevationGain    float64 `json:"elevationGain"`
	MovingTime       int64   `json:"movingTime"`
	AverageSpeed     float64 `json:"averageSpeed"`
	AverageHeartrate float64 `json:"averageHeartrate"`
	AverageWatts     float64 `json:"averageWatts"`
	AverageCadence   float64 `json:"averageCadence"`
	SimilarityScore  float64 `json:"similarityScore"`
}

type ContractActivityComparisonSegment struct {
	Id            int64    `json:"id"`
	Name          string   `json:"name"`
	MatchCount    int64    `json:"matchCount"`
	ActivityIds   []int64  `json:"activityIds"`
	ActivityNames []string `json:"activityNames"`
}

type ContractActivityEffort struct {
	Id            string   `json:"id"`
	Label         string   `json:"label"`
	Distance      float64  `json:"distance"`
	Seconds       int64    `json:"seconds"`
	DeltaAltitude float64  `json:"deltaAltitude"`
	ElevationGain *float64 `json:"elevationGain,omitempty"`
	ElevationLoss *float64 `json:"elevationLoss,omitempty"`
	IdxStart      int64    `json:"idxStart"`
	IdxEnd        int64    `json:"idxEnd"`
	AveragePower  *float64 `json:"averagePower"`
	Description   string   `json:"description"`
}

type ContractStravaSegmentEffort struct {
	AverageCadence   float64               `json:"averageCadence"`
	AverageHeartRate float64               `json:"averageHeartRate"`
	AverageWatts     float64               `json:"averageWatts"`
	DeviceWatts      bool                  `json:"deviceWatts"`
	Distance         float64               `json:"distance"`
	ElapsedTime      int64                 `json:"elapsedTime"`
	EndIndex         int64                 `json:"endIndex"`
	Hidden           bool                  `json:"hidden"`
	Id               int64                 `json:"id"`
	KomRank          *int64                `json:"komRank,omitempty"`
	MaxHeartRate     float64               `json:"maxHeartRate"`
	MovingTime       int64                 `json:"movingTime"`
	Name             string                `json:"name"`
	PrRank           *int64                `json:"prRank,omitempty"`
	ResourceState    int64                 `json:"resourceState"`
	Segment          ContractStravaSegment `json:"segment"`
	StartDate        string                `json:"startDate"`
	StartDateLocal   string                `json:"startDateLocal"`
	StartIndex       int64                 `json:"startIndex"`
	Visibility       *string               `json:"visibility,omitempty"`
	Achievements     []ContractAchievement `json:"achievements,omitempty"`
	Activity         *int64                `json:"activity,omitempty"`
	Athlete          *int64                `json:"athlete,omitempty"`
}

type ContractStravaSegment struct {
	ActivityType  string    `json:"activityType"`
	AverageGrade  float64   `json:"averageGrade"`
	City          *string   `json:"city,omitempty"`
	ClimbCategory int64     `json:"climbCategory"`
	Country       *string   `json:"country,omitempty"`
	Distance      float64   `json:"distance"`
	ElevationHigh float64   `json:"elevationHigh"`
	ElevationLow  float64   `json:"elevationLow"`
	EndLatLng     []float64 `json:"endLatLng"`
	Hazardous     bool      `json:"hazardous"`
	Id            int64     `json:"id"`
	MaximumGrade  float64   `json:"maximumGrade"`
	Name          string    `json:"name"`
	IsPrivate     bool      `json:"isPrivate"`
	ResourceState int64     `json:"resourceState"`
	Starred       bool      `json:"starred"`
	StartLatLng   []float64 `json:"startLatLng"`
	State         *string   `json:"state,omitempty"`
}

type ContractActivityStream struct {
	Distance       []float64   `json:"distance"`
	Time           []int64     `json:"time"`
	Latlng         [][]float64 `json:"latlng,omitempty"`
	Heartrate      []int64     `json:"heartrate,omitempty"`
	Cadence        []int64     `json:"cadence,omitempty"`
	Moving         []bool      `json:"moving,omitempty"`
	Altitude       []float64   `json:"altitude,omitempty"`
	Watts          []*float64  `json:"watts,omitempty"`
	VelocitySmooth []float64   `json:"velocitySmooth,omitempty"`
}

type ContractStatistic struct {
	Label    string                 `json:"label"`
	Value    string                 `json:"value"`
	Activity *ContractActivityShort `json:"activity,omitempty"`
}

type ContractActivityShort struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type ContractPersonalRecordTimeline struct {
	MetricKey     string                `json:"metricKey"`
	MetricLabel   string                `json:"metricLabel"`
	ActivityDate  string                `json:"activityDate"`
	Value         string                `json:"value"`
	PreviousValue *string               `json:"previousValue,omitempty"`
	Improvement   *string               `json:"improvement,omitempty"`
	Activity      ContractActivityShort `json:"activity"`
}

type ContractSegmentClimbProgression struct {
	Metric                  string                              `json:"metric"`
	TargetTypeFilter        string                              `json:"targetTypeFilter"`
	WeatherContextAvailable bool                                `json:"weatherContextAvailable"`
	Targets                 []ContractSegmentClimbTargetSummary `json:"targets"`
	SelectedTargetId        *int64                              `json:"selectedTargetId,omitempty"`
	SelectedTargetType      *string                             `json:"selectedTargetType,omitempty"`
	Attempts                []ContractSegmentClimbAttempt       `json:"attempts"`
}

type ContractSegmentClimbTargetSummary struct {
	TargetId       int64   `json:"targetId"`
	TargetName     string  `json:"targetName"`
	TargetType     string  `json:"targetType"`
	ClimbCategory  int64   `json:"climbCategory"`
	Distance       float64 `json:"distance"`
	AverageGrade   float64 `json:"averageGrade"`
	AttemptsCount  int64   `json:"attemptsCount"`
	BestValue      string  `json:"bestValue"`
	LatestValue    string  `json:"latestValue"`
	Consistency    string  `json:"consistency"`
	AveragePacing  string  `json:"averagePacing"`
	CloseToPrCount int64   `json:"closeToPrCount"`
	RecentTrend    string  `json:"recentTrend"`
}

type ContractSegmentClimbAttempt struct {
	TargetId           int64                 `json:"targetId"`
	TargetName         string                `json:"targetName"`
	TargetType         string                `json:"targetType"`
	ActivityDate       string                `json:"activityDate"`
	ElapsedTimeSeconds int64                 `json:"elapsedTimeSeconds"`
	MovingTimeSeconds  int64                 `json:"movingTimeSeconds"`
	SpeedKph           float64               `json:"speedKph"`
	Distance           float64               `json:"distance"`
	AverageGrade       float64               `json:"averageGrade"`
	ElevationGain      float64               `json:"elevationGain"`
	AveragePowerWatts  float64               `json:"averagePowerWatts"`
	AverageHeartRate   float64               `json:"averageHeartRate"`
	PrRank             *int64                `json:"prRank,omitempty"`
	PersonalRank       *int64                `json:"personalRank,omitempty"`
	SetsNewPr          bool                  `json:"setsNewPr"`
	CloseToPr          bool                  `json:"closeToPr"`
	DeltaToPr          string                `json:"deltaToPr"`
	WeatherSummary     *string               `json:"weatherSummary,omitempty"`
	Activity           ContractActivityShort `json:"activity"`
}

type ContractSegmentSummary struct {
	Metric         string                            `json:"metric"`
	Segment        ContractSegmentClimbTargetSummary `json:"segment"`
	PersonalRecord *ContractSegmentClimbAttempt      `json:"personalRecord,omitempty"`
	TopEfforts     []ContractSegmentClimbAttempt     `json:"topEfforts"`
}

type ContractResolvedHeartRateZoneSettings struct {
	MaxHr       int64  `json:"maxHr"`
	ThresholdHr *int64 `json:"thresholdHr,omitempty"`
	ReserveHr   *int64 `json:"reserveHr,omitempty"`
	Method      string `json:"method"`
	Source      string `json:"source"`
}

type ContractHeartRateZoneDistribution struct {
	Zone       string  `json:"zone"`
	Label      string  `json:"label"`
	Seconds    int64   `json:"seconds"`
	Percentage float64 `json:"percentage"`
}

type ContractHeartRateZoneActivitySummary struct {
	Activity            ContractActivityShort               `json:"activity"`
	ActivityDate        string                              `json:"activityDate"`
	TotalTrackedSeconds int64                               `json:"totalTrackedSeconds"`
	EasySeconds         int64                               `json:"easySeconds"`
	HardSeconds         int64                               `json:"hardSeconds"`
	EasyHardRatio       *float64                            `json:"easyHardRatio,omitempty"`
	Zones               []ContractHeartRateZoneDistribution `json:"zones"`
}

type ContractHeartRateZonePeriodSummary struct {
	Period              string                              `json:"period"`
	TotalTrackedSeconds int64                               `json:"totalTrackedSeconds"`
	EasySeconds         int64                               `json:"easySeconds"`
	HardSeconds         int64                               `json:"hardSeconds"`
	EasyHardRatio       *float64                            `json:"easyHardRatio,omitempty"`
	Zones               []ContractHeartRateZoneDistribution `json:"zones"`
}

type ContractHeartRateZoneAnalysis struct {
	Settings            ContractHeartRateZoneSettings          `json:"settings"`
	ResolvedSettings    *ContractResolvedHeartRateZoneSettings `json:"resolvedSettings,omitempty"`
	HasHeartRateData    bool                                   `json:"hasHeartRateData"`
	TotalTrackedSeconds int64                                  `json:"totalTrackedSeconds"`
	EasyHardRatio       *float64                               `json:"easyHardRatio,omitempty"`
	Zones               []ContractHeartRateZoneDistribution    `json:"zones"`
	Activities          []ContractHeartRateZoneActivitySummary `json:"activities"`
	ByMonth             []ContractHeartRateZonePeriodSummary   `json:"byMonth"`
	ByYear              []ContractHeartRateZonePeriodSummary   `json:"byYear"`
}

type ContractFtpEstimate struct {
	Available      bool    `json:"available"`
	Ftp            int64   `json:"ftp"`
	Method         string  `json:"method"`
	MethodLabel    string  `json:"methodLabel"`
	BestPower      int64   `json:"bestPower"`
	Multiplier     float64 `json:"multiplier"`
	BasedOnSeconds int64   `json:"basedOnSeconds"`
	Confidence     string  `json:"confidence"`
	Source         string  `json:"source"`
	SourceKind     string  `json:"sourceKind"`
	ActivityId     int64   `json:"activityId"`
	ActivityName   string  `json:"activityName"`
	ActivityType   string  `json:"activityType"`
	ActivityDate   string  `json:"activityDate"`
	WindowDays     int64   `json:"windowDays"`
	ActivityCount  int64   `json:"activityCount"`
}

type ContractAchievement struct {
	EffortCount int64  `json:"effortCount"`
	Rank        int64  `json:"rank"`
	Type        string `json:"type"`
	TypeId      int64  `json:"typeId"`
}

type ContractTrainingLoadActivity struct {
	ActivityId           int64    `json:"activityId"`
	Name                 string   `json:"name"`
	Date                 string   `json:"date"`
	Sport                string   `json:"sport"`
	Source               string   `json:"source"`
	Reason               string   `json:"reason"`
	Load                 *float64 `json:"load"`
	Ftp                  *int64   `json:"ftp"`
	FtpEffectiveFrom     *string  `json:"ftpEffectiveFrom"`
	NormalizedPower      *float64 `json:"normalizedPower"`
	IntensityFactor      *float64 `json:"intensityFactor"`
	Best5MinutePower     *float64 `json:"best5MinutePower"`
	MovingSeconds        int64    `json:"movingSeconds"`
	ElapsedSeconds       int64    `json:"elapsedSeconds"`
	RecordedSeconds      float64  `json:"recordedSeconds"`
	CoveredSeconds       float64  `json:"coveredSeconds"`
	DistanceMeters       float64  `json:"distanceMeters"`
	ElevationMeters      float64  `json:"elevationMeters"`
	AerobicSeconds       float64  `json:"aerobicSeconds"`
	ThresholdSeconds     float64  `json:"thresholdSeconds"`
	HighIntensitySeconds float64  `json:"highIntensitySeconds"`
}

type ContractTrainingLoadPeriod struct {
	StartDate             string   `json:"startDate"`
	EndDate               string   `json:"endDate"`
	ActivityCount         int64    `json:"activityCount"`
	ScoredCount           int64    `json:"scoredCount"`
	MeasuredLoad          *float64 `json:"measuredLoad"`
	EstimatedLoad         *float64 `json:"estimatedLoad"`
	MovingSeconds         int64    `json:"movingSeconds"`
	DistanceMeters        float64  `json:"distanceMeters"`
	ElevationMeters       float64  `json:"elevationMeters"`
	CoveredSeconds        float64  `json:"coveredSeconds"`
	RecordedSeconds       float64  `json:"recordedSeconds"`
	AerobicSeconds        float64  `json:"aerobicSeconds"`
	ThresholdSeconds      float64  `json:"thresholdSeconds"`
	HighIntensitySeconds  float64  `json:"highIntensitySeconds"`
	Best5MinutePower      *float64 `json:"best5MinutePower"`
	Best5MinuteActivityId *int64   `json:"best5MinuteActivityId"`
}

type ContractTrainingLoadWeek struct {
	Summary ContractTrainingLoadPeriod   `json:"summary"`
	Days    []ContractTrainingLoadPeriod `json:"days"`
}

type ContractTrainingReport struct {
	Method            string                         `json:"method"`
	MaxGapSeconds     int64                          `json:"maxGapSeconds"`
	UndatedActivities int64                          `json:"undatedActivities"`
	Weeks             []ContractTrainingLoadWeek     `json:"weeks"`
	Activities        []ContractTrainingLoadActivity `json:"activities"`
}

type ContractOperation struct {
	Method string
	Path   string
}

var ContractOperations = map[string]ContractOperation{
	"listActivities":                    {Method: "GET", Path: "/api/activities"},
	"getActivity":                       {Method: "GET", Path: "/api/activities/{activityId}"},
	"exportActivitiesCsv":               {Method: "GET", Path: "/api/activities/csv"},
	"getCurrentAthlete":                 {Method: "GET", Path: "/api/athletes/me"},
	"getFtpEstimate":                    {Method: "GET", Path: "/api/athletes/me/ftp-estimate"},
	"getHeartRateZones":                 {Method: "GET", Path: "/api/athletes/me/heart-rate-zones"},
	"updateHeartRateZones":              {Method: "PUT", Path: "/api/athletes/me/heart-rate-zones"},
	"getPerformanceSettings":            {Method: "GET", Path: "/api/athletes/me/performance-settings"},
	"updatePerformanceSettings":         {Method: "PUT", Path: "/api/athletes/me/performance-settings"},
	"getBadges":                         {Method: "GET", Path: "/api/badges"},
	"getAverageCadenceChart":            {Method: "GET", Path: "/api/charts/average-cadence-by-period"},
	"getAverageSpeedChart":              {Method: "GET", Path: "/api/charts/average-speed-by-period"},
	"getDistanceChart":                  {Method: "GET", Path: "/api/charts/distance-by-period"},
	"getElevationChart":                 {Method: "GET", Path: "/api/charts/elevation-by-period"},
	"getDashboard":                      {Method: "GET", Path: "/api/dashboard"},
	"getActivityHeatmap":                {Method: "GET", Path: "/api/dashboard/activity-heatmap"},
	"getCumulativeData":                 {Method: "GET", Path: "/api/dashboard/cumulative-data-per-year"},
	"getEddingtonNumber":                {Method: "GET", Path: "/api/dashboard/eddington-number"},
	"revertDataQualityCorrection":       {Method: "DELETE", Path: "/api/data-quality/corrections/{id}"},
	"applyDataQualityCorrection":        {Method: "POST", Path: "/api/data-quality/corrections/{id}"},
	"previewDataQualityCorrection":      {Method: "GET", Path: "/api/data-quality/corrections/preview/{issueId}"},
	"applySafeDataQualityCorrections":   {Method: "POST", Path: "/api/data-quality/corrections/safe"},
	"previewSafeDataQualityCorrections": {Method: "GET", Path: "/api/data-quality/corrections/safe/preview"},
	"includeActivityInStatistics":       {Method: "DELETE", Path: "/api/data-quality/exclusions/{activityId}"},
	"excludeActivityFromStatistics":     {Method: "PUT", Path: "/api/data-quality/exclusions/{activityId}"},
	"getDataQualityIssues":              {Method: "GET", Path: "/api/data-quality/issues"},
	"getGearAnalysis":                   {Method: "GET", Path: "/api/gear-analysis"},
	"createGearMaintenance":             {Method: "POST", Path: "/api/gear-analysis/maintenance"},
	"deleteGearMaintenance":             {Method: "DELETE", Path: "/api/gear-analysis/maintenance/{recordId}"},
	"getHealthDetails":                  {Method: "GET", Path: "/api/health/details"},
	"getLocalDataBackup":                {Method: "GET", Path: "/api/local-data/backup"},
	"restoreLocalData":                  {Method: "POST", Path: "/api/local-data/restore"},
	"getMapTracks":                      {Method: "GET", Path: "/api/maps/gpx"},
	"getMapPassages":                    {Method: "GET", Path: "/api/maps/passages"},
	"editGeneratedRoute":                {Method: "POST", Path: "/api/routes/{routeId}/edit"},
	"exportGeneratedRouteGpx":           {Method: "GET", Path: "/api/routes/{routeId}/gpx"},
	"generateShapeRoutes":               {Method: "POST", Path: "/api/routes/generate/shape"},
	"getRouteRecommendations":           {Method: "GET", Path: "/api/routes/recommendations"},
	"exportRouteRecommendationGpx":      {Method: "GET", Path: "/api/routes/recommendations/gpx"},
	"startOsrm":                         {Method: "POST", Path: "/api/routing/osrm/start"},
	"listSegments":                      {Method: "GET", Path: "/api/segments"},
	"listSegmentEfforts":                {Method: "GET", Path: "/api/segments/{segmentId}/efforts"},
	"getSegmentSummary":                 {Method: "GET", Path: "/api/segments/{segmentId}/summary"},
	"applySourceMode":                   {Method: "POST", Path: "/api/source-modes/apply"},
	"previewSourceMode":                 {Method: "POST", Path: "/api/source-modes/preview"},
	"completeStravaOAuth":               {Method: "GET", Path: "/api/source-modes/strava/oauth/callback"},
	"startStravaOAuth":                  {Method: "POST", Path: "/api/source-modes/strava/oauth/start"},
	"synchronizeSources":                {Method: "POST", Path: "/api/source-sync/synchronize"},
	"getStatistics":                     {Method: "GET", Path: "/api/statistics"},
	"getHeartRateZoneAnalysis":          {Method: "GET", Path: "/api/statistics/heart-rate-zones"},
	"getPersonalRecordsTimeline":        {Method: "GET", Path: "/api/statistics/personal-records-timeline"},
	"getSegmentClimbProgression":        {Method: "GET", Path: "/api/statistics/segment-climb-progression"},
	"getTrainingLoad":                   {Method: "GET", Path: "/api/statistics/training-load"},
}
