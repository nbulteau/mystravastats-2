package business

type PersonalRecordTimelineEntry struct {
	MetricKey     string
	MetricLabel   string
	ActivityDate  string
	Value         string
	PreviousValue *string
	Improvement   *string
	Activity      ActivityShort
}
