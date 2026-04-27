package application

import "mystravastats/internal/shared/domain/business"

type ExcludeActivityFromStatsUseCase struct {
	writer DataQualityWriter
}

func NewExcludeActivityFromStatsUseCase(writer DataQualityWriter) *ExcludeActivityFromStatsUseCase {
	return &ExcludeActivityFromStatsUseCase{writer: writer}
}

func (uc *ExcludeActivityFromStatsUseCase) Execute(activityID int64, reason string) (business.DataQualityReport, error) {
	return uc.writer.ExcludeActivityFromStats(activityID, reason)
}

type IncludeActivityInStatsUseCase struct {
	writer DataQualityWriter
}

func NewIncludeActivityInStatsUseCase(writer DataQualityWriter) *IncludeActivityInStatsUseCase {
	return &IncludeActivityInStatsUseCase{writer: writer}
}

func (uc *IncludeActivityInStatsUseCase) Execute(activityID int64) (business.DataQualityReport, error) {
	return uc.writer.IncludeActivityInStats(activityID)
}
