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

type DataQualityIssue struct {
	ID                string              `json:"id"`
	Source            string              `json:"source"`
	ActivityID        int64               `json:"activityId,omitempty"`
	ActivityName      string              `json:"activityName,omitempty"`
	ActivityType      string              `json:"activityType,omitempty"`
	Year              string              `json:"year,omitempty"`
	FilePath          string              `json:"filePath,omitempty"`
	Severity          DataQualitySeverity `json:"severity"`
	Category          DataQualityCategory `json:"category"`
	Field             string              `json:"field"`
	Message           string              `json:"message"`
	RawValue          string              `json:"rawValue,omitempty"`
	Suggestion        string              `json:"suggestion,omitempty"`
	ExcludedFromStats bool                `json:"excludedFromStats"`
	ExcludedAt        string              `json:"excludedAt,omitempty"`
}

type DataQualitySummary struct {
	Status             string             `json:"status"`
	Provider           string             `json:"provider"`
	IssueCount         int                `json:"issueCount"`
	ImpactedActivities int                `json:"impactedActivities"`
	ExcludedActivities int                `json:"excludedActivities"`
	BySeverity         map[string]int     `json:"bySeverity"`
	ByCategory         map[string]int     `json:"byCategory"`
	TopIssues          []DataQualityIssue `json:"topIssues"`
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
	GeneratedAt string                 `json:"generatedAt"`
	Summary     DataQualitySummary     `json:"summary"`
	Issues      []DataQualityIssue     `json:"issues"`
	Exclusions  []DataQualityExclusion `json:"exclusions"`
}

type DataQualityExclusionRequest struct {
	Reason string `json:"reason"`
}
