package application

import "mystravastats/domain/business"

type ListSegmentEffortsUseCase struct {
	reader SegmentsReader
}

func NewListSegmentEffortsUseCase(reader SegmentsReader) *ListSegmentEffortsUseCase {
	return &ListSegmentEffortsUseCase{
		reader: reader,
	}
}

func (uc *ListSegmentEffortsUseCase) Execute(
	year *int,
	metric *string,
	segmentID int64,
	from *string,
	to *string,
	activityTypes []business.ActivityType,
) []business.SegmentClimbAttempt {
	efforts := uc.reader.FindSegmentEffortsByYearMetricRangeAndTypes(year, metric, segmentID, from, to, activityTypes...)
	if efforts == nil {
		return []business.SegmentClimbAttempt{}
	}

	return efforts
}
