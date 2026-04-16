package application

import (
	domainStatistics "mystravastats/domain/statistics"
	"mystravastats/internal/shared/domain/business"
)

// StatisticsReader is an outbound port used by statistics use cases.
// Infrastructure adapters implement this interface.
type StatisticsReader interface {
	FindStatisticsByYearAndTypes(year *int, activityTypes ...business.ActivityType) []domainStatistics.Statistic
}

// PersonalRecordsTimelineReader is an outbound port used by
// personal-records-timeline use cases.
type PersonalRecordsTimelineReader interface {
	FindPersonalRecordsTimelineByYearMetricAndTypes(year *int, metric *string, activityTypes ...business.ActivityType) []business.PersonalRecordTimelineEntry
}
