package infrastructure

import (
	domainStatistics "mystravastats/domain/statistics"
	"mystravastats/internal/shared/domain/business"
)

// StatisticsServiceAdapter computes statistics directly and still delegates
// personal-record timeline to legacy services during migration.
type StatisticsServiceAdapter struct{}

func NewStatisticsServiceAdapter() *StatisticsServiceAdapter {
	return &StatisticsServiceAdapter{}
}

func (adapter *StatisticsServiceAdapter) FindStatisticsByYearAndTypes(year *int, activityTypes ...business.ActivityType) []domainStatistics.Statistic {
	return computeStatisticsByYearAndTypes(year, activityTypes...)
}

func (adapter *StatisticsServiceAdapter) FindPersonalRecordsTimelineByYearMetricAndTypes(year *int, metric *string, activityTypes ...business.ActivityType) []business.PersonalRecordTimelineEntry {
	return computePersonalRecordsTimelineByYearMetricAndTypes(year, metric, activityTypes...)
}
