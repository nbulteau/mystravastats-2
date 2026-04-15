package application

import "mystravastats/domain/business"

type ListSegmentsUseCase struct {
	reader SegmentsReader
}

func NewListSegmentsUseCase(reader SegmentsReader) *ListSegmentsUseCase {
	return &ListSegmentsUseCase{
		reader: reader,
	}
}

func (uc *ListSegmentsUseCase) Execute(
	year *int,
	metric *string,
	query *string,
	from *string,
	to *string,
	activityTypes []business.ActivityType,
) []business.SegmentClimbTargetSummary {
	segments := uc.reader.FindSegmentsByYearMetricQueryRangeAndTypes(year, metric, query, from, to, activityTypes...)
	if segments == nil {
		return []business.SegmentClimbTargetSummary{}
	}

	return segments
}
