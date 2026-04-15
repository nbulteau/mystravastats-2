package application

import (
	"mystravastats/domain/business"
	segmentsDomain "mystravastats/internal/segments/domain"
)

type GetSegmentSummaryUseCase struct {
	reader SegmentsReader
}

func NewGetSegmentSummaryUseCase(reader SegmentsReader) *GetSegmentSummaryUseCase {
	return &GetSegmentSummaryUseCase{
		reader: reader,
	}
}

func (uc *GetSegmentSummaryUseCase) Execute(
	year *int,
	metric *string,
	segmentID int64,
	from *string,
	to *string,
	activityTypes []business.ActivityType,
) *segmentsDomain.SegmentSummary {
	return uc.reader.FindSegmentSummaryByYearMetricRangeAndTypes(year, metric, segmentID, from, to, activityTypes...)
}
