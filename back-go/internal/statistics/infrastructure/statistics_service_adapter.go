package infrastructure

import (
	"mystravastats/domain/business"
	domainStatistics "mystravastats/domain/statistics"
	"mystravastats/internal/services"
)

// StatisticsServiceAdapter bridges the current internal/services layer
// to the hexagonal outbound port used by statistics use cases.
type StatisticsServiceAdapter struct{}

func NewStatisticsServiceAdapter() *StatisticsServiceAdapter {
	return &StatisticsServiceAdapter{}
}

func (adapter *StatisticsServiceAdapter) FindStatisticsByYearAndTypes(year *int, activityTypes ...business.ActivityType) []domainStatistics.Statistic {
	return services.FetchStatisticsByActivityTypeAndYear(year, activityTypes...)
}

func (adapter *StatisticsServiceAdapter) FindPersonalRecordsTimelineByYearMetricAndTypes(year *int, metric *string, activityTypes ...business.ActivityType) []business.PersonalRecordTimelineEntry {
	return services.FetchPersonalRecordsTimelineByActivityTypeAndYear(year, metric, activityTypes...)
}
