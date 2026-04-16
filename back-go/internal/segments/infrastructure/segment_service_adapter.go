package infrastructure

import (
	"mystravastats/internal/segments/domain"
	"mystravastats/internal/shared/domain/business"
)

// SegmentServiceAdapter computes segment read models directly from provider data.
type SegmentServiceAdapter struct{}

func NewSegmentServiceAdapter() *SegmentServiceAdapter {
	return &SegmentServiceAdapter{}
}

func (adapter *SegmentServiceAdapter) FindSegmentClimbProgressionByYearMetricTargetAndTypes(
	year *int,
	metric *string,
	targetType *string,
	targetId *int64,
	activityTypes ...business.ActivityType,
) business.SegmentClimbProgression {
	return computeSegmentClimbProgressionByYearMetricTargetAndTypes(year, metric, targetType, targetId, activityTypes...)
}

func (adapter *SegmentServiceAdapter) FindSegmentsByYearMetricQueryRangeAndTypes(
	year *int,
	metric *string,
	query *string,
	from *string,
	to *string,
	activityTypes ...business.ActivityType,
) []business.SegmentClimbTargetSummary {
	return computeSegmentsByYearMetricQueryRangeAndTypes(year, metric, query, from, to, activityTypes...)
}

func (adapter *SegmentServiceAdapter) FindSegmentEffortsByYearMetricRangeAndTypes(
	year *int,
	metric *string,
	segmentID int64,
	from *string,
	to *string,
	activityTypes ...business.ActivityType,
) []business.SegmentClimbAttempt {
	return computeSegmentEffortsByYearMetricRangeAndTypes(year, metric, segmentID, from, to, activityTypes...)
}

func (adapter *SegmentServiceAdapter) FindSegmentSummaryByYearMetricRangeAndTypes(
	year *int,
	metric *string,
	segmentID int64,
	from *string,
	to *string,
	activityTypes ...business.ActivityType,
) *domain.SegmentSummary {
	summary := computeSegmentSummaryByYearMetricRangeAndTypes(year, metric, segmentID, from, to, activityTypes...)
	if summary == nil {
		return nil
	}

	return &domain.SegmentSummary{
		Metric:         summary.Metric,
		Segment:        summary.Segment,
		PersonalRecord: summary.PersonalRecord,
		TopEfforts:     summary.TopEfforts,
		Attempts:       summary.Attempts,
	}
}
