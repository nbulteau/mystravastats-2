package application

import (
	segmentsDomain "mystravastats/internal/segments/domain"
	"mystravastats/internal/shared/domain/business"
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
