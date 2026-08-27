// Code generated from docs/api/openapi.json. DO NOT EDIT.

package dto

type ContractApiError struct {
	Message     string  `json:"message"`
	Code        int64   `json:"code"`
	Description *string `json:"description,omitempty"`
	Path        *string `json:"path,omitempty"`
	RequestId   *string `json:"requestId,omitempty"`
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
	Mode string  `json:"mode"`
	Path *string `json:"path,omitempty"`
}

type ContractOperationStatus struct {
	Success bool    `json:"success"`
	Message *string `json:"message,omitempty"`
}

type ContractActivitySummary struct {
	Id       int64    `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Date     *string  `json:"date,omitempty"`
	Distance *float64 `json:"distance,omitempty"`
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
	"revertDataQualityCorrection":       {Method: "DELETE", Path: "/api/data-quality/corrections/{correctionId}"},
	"applyDataQualityCorrection":        {Method: "POST", Path: "/api/data-quality/corrections/{issueId}"},
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
}
