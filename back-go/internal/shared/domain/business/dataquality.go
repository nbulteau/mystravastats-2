package business

type DataQualitySeverity string

const (
	DataQualitySeverityInfo     DataQualitySeverity = "info"
	DataQualitySeverityWarning  DataQualitySeverity = "warning"
	DataQualitySeverityCritical DataQualitySeverity = "critical"
)

type DataQualityCategory string

const (
	DataQualityCategoryInvalidFile        DataQualityCategory = "INVALID_FILE"
	DataQualityCategoryMissingStream      DataQualityCategory = "MISSING_STREAM"
	DataQualityCategoryMissingStreamField DataQualityCategory = "MISSING_STREAM_FIELD"
	DataQualityCategoryStreamDataCoverage DataQualityCategory = "STREAM_DATA_COVERAGE"
	DataQualityCategoryInvalidValue       DataQualityCategory = "INVALID_VALUE"
	DataQualityCategoryInconsistentTime   DataQualityCategory = "INCONSISTENT_TIME"
	DataQualityCategoryGPSGlitch          DataQualityCategory = "GPS_GLITCH"
	DataQualityCategoryAltitudeSpike      DataQualityCategory = "ALTITUDE_SPIKE"
	DataQualityCategoryFallbackValue      DataQualityCategory = "FALLBACK_VALUE"
)

type DataQualityCorrectionSafety string

const (
	DataQualityCorrectionSafetySafe        DataQualityCorrectionSafety = "safe"
	DataQualityCorrectionSafetyManual      DataQualityCorrectionSafety = "manual"
	DataQualityCorrectionSafetyUnsupported DataQualityCorrectionSafety = "unsupported"
)

type DataQualityCorrectionStatus string

const (
	DataQualityCorrectionStatusActive   DataQualityCorrectionStatus = "active"
	DataQualityCorrectionStatusReverted DataQualityCorrectionStatus = "reverted"
)

type DataQualityCorrectionType string

const (
	DataQualityCorrectionTypeRemoveGPSPoint        DataQualityCorrectionType = "REMOVE_GPS_POINT"
	DataQualityCorrectionTypeSmoothAltitudeSpike   DataQualityCorrectionType = "SMOOTH_ALTITUDE_SPIKE"
	DataQualityCorrectionTypeMaskInvalidValue      DataQualityCorrectionType = "MASK_INVALID_VALUE"
	DataQualityCorrectionTypeRecalculateFromStream DataQualityCorrectionType = "RECALCULATE_FROM_STREAM"
)

type DataQualityCorrectionSuggestion struct {
	Available   bool                        `json:"available"`
	Safety      DataQualityCorrectionSafety `json:"safety"`
	Type        DataQualityCorrectionType   `json:"type,omitempty"`
	Label       string                      `json:"label,omitempty"`
	Description string                      `json:"description,omitempty"`
}

type DataQualityIssue struct {
	ID                  string                           `json:"id"`
	Source              string                           `json:"source"`
	ActivityID          int64                            `json:"activityId,omitempty"`
	ActivityName        string                           `json:"activityName,omitempty"`
	ActivityType        string                           `json:"activityType,omitempty"`
	Year                string                           `json:"year,omitempty"`
	FilePath            string                           `json:"filePath,omitempty"`
	Severity            DataQualitySeverity              `json:"severity"`
	Category            DataQualityCategory              `json:"category"`
	Field               string                           `json:"field"`
	Message             string                           `json:"message"`
	RawValue            string                           `json:"rawValue,omitempty"`
	Suggestion          string                           `json:"suggestion,omitempty"`
	ExcludedFromStats   bool                             `json:"excludedFromStats"`
	ExcludedAt          string                           `json:"excludedAt,omitempty"`
	Corrected           bool                             `json:"corrected"`
	CorrectionAppliedAt string                           `json:"correctionAppliedAt,omitempty"`
	Correction          *DataQualityCorrectionSuggestion `json:"correction,omitempty"`
}

type DataQualitySummary struct {
	Status              string             `json:"status"`
	Provider            string             `json:"provider"`
	IssueCount          int                `json:"issueCount"`
	ImpactedActivities  int                `json:"impactedActivities"`
	ExcludedActivities  int                `json:"excludedActivities"`
	CorrectionCount     int                `json:"correctionCount"`
	SafeCorrectionCount int                `json:"safeCorrectionCount"`
	ManualReviewCount   int                `json:"manualReviewCount"`
	BySeverity          map[string]int     `json:"bySeverity"`
	ByCategory          map[string]int     `json:"byCategory"`
	TopIssues           []DataQualityIssue `json:"topIssues"`
}

type DataQualityExclusion struct {
	ActivityID   int64  `json:"activityId"`
	Source       string `json:"source"`
	ActivityName string `json:"activityName,omitempty"`
	ActivityType string `json:"activityType,omitempty"`
	Year         string `json:"year,omitempty"`
	Reason       string `json:"reason,omitempty"`
	ExcludedAt   string `json:"excludedAt"`
}

type DataQualityReport struct {
	GeneratedAt string                  `json:"generatedAt"`
	Summary     DataQualitySummary      `json:"summary"`
	Issues      []DataQualityIssue      `json:"issues"`
	Exclusions  []DataQualityExclusion  `json:"exclusions"`
	Corrections []DataQualityCorrection `json:"corrections"`
}

type DataQualityExclusionRequest struct {
	Reason string `json:"reason"`
}

type DataQualityCorrectionImpact struct {
	DistanceMetersBefore  float64 `json:"distanceMetersBefore,omitempty"`
	DistanceMetersAfter   float64 `json:"distanceMetersAfter,omitempty"`
	ElevationMetersBefore float64 `json:"elevationMetersBefore,omitempty"`
	ElevationMetersAfter  float64 `json:"elevationMetersAfter,omitempty"`
	MaxSpeedBefore        float64 `json:"maxSpeedBefore,omitempty"`
	MaxSpeedAfter         float64 `json:"maxSpeedAfter,omitempty"`
	DistanceDeltaMeters   float64 `json:"distanceDeltaMeters"`
	ElevationDeltaMeters  float64 `json:"elevationDeltaMeters"`
}

type DataQualityCorrection struct {
	ID             string                      `json:"id"`
	IssueID        string                      `json:"issueId"`
	Source         string                      `json:"source"`
	ActivityID     int64                       `json:"activityId"`
	ActivityName   string                      `json:"activityName,omitempty"`
	ActivityType   string                      `json:"activityType,omitempty"`
	Year           string                      `json:"year,omitempty"`
	Type           DataQualityCorrectionType   `json:"type"`
	Safety         DataQualityCorrectionSafety `json:"safety"`
	Status         DataQualityCorrectionStatus `json:"status"`
	PointIndexes   []int                       `json:"pointIndexes,omitempty"`
	ModifiedFields []string                    `json:"modifiedFields"`
	Impact         DataQualityCorrectionImpact `json:"impact"`
	Reason         string                      `json:"reason,omitempty"`
	AppliedAt      string                      `json:"appliedAt,omitempty"`
	RevertedAt     string                      `json:"revertedAt,omitempty"`
}

type DataQualityCorrectionBatchSummary struct {
	SafeCorrectionCount       int      `json:"safeCorrectionCount"`
	ManualReviewCount         int      `json:"manualReviewCount"`
	UnsupportedIssueCount     int      `json:"unsupportedIssueCount"`
	ActivityCount             int      `json:"activityCount"`
	DistanceDeltaMeters       float64  `json:"distanceDeltaMeters"`
	ElevationDeltaMeters      float64  `json:"elevationDeltaMeters"`
	ModifiedFields            []string `json:"modifiedFields"`
	PotentiallyImpactsRecords bool     `json:"potentiallyImpactsRecords"`
}

type DataQualityCorrectionPreview struct {
	GeneratedAt     string                            `json:"generatedAt"`
	Mode            string                            `json:"mode"`
	Summary         DataQualityCorrectionBatchSummary `json:"summary"`
	Corrections     []DataQualityCorrection           `json:"corrections"`
	Warnings        []string                          `json:"warnings"`
	BlockingReasons []string                          `json:"blockingReasons"`
}
