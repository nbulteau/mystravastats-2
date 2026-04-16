package application

import "mystravastats/internal/shared/domain/business"

type GetSegmentClimbProgressionUseCase struct {
	reader SegmentsReader
}

func NewGetSegmentClimbProgressionUseCase(reader SegmentsReader) *GetSegmentClimbProgressionUseCase {
	return &GetSegmentClimbProgressionUseCase{
		reader: reader,
	}
}

func (uc *GetSegmentClimbProgressionUseCase) Execute(
	year *int,
	metric *string,
	targetType *string,
	targetID *int64,
	activityTypes []business.ActivityType,
) business.SegmentClimbProgression {
	return uc.reader.FindSegmentClimbProgressionByYearMetricTargetAndTypes(year, metric, targetType, targetID, activityTypes...)
}
