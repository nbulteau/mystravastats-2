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
