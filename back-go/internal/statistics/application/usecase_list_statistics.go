package application

import (
	domainStatistics "mystravastats/domain/statistics"
	"mystravastats/internal/shared/domain/business"
)

type ListStatisticsUseCase struct {
	reader StatisticsReader
}

func NewListStatisticsUseCase(reader StatisticsReader) *ListStatisticsUseCase {
	return &ListStatisticsUseCase{
		reader: reader,
	}
}

func (uc *ListStatisticsUseCase) Execute(year *int, activityTypes []business.ActivityType) []domainStatistics.Statistic {
	statistics := uc.reader.FindStatisticsByYearAndTypes(year, activityTypes...)
	if statistics == nil {
		return []domainStatistics.Statistic{}
	}

	return statistics
}
