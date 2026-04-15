package infrastructure

import (
	"mystravastats/domain/business"
	"mystravastats/internal/segments/domain"
	"mystravastats/internal/services"
)

// SegmentServiceAdapter bridges the current internal/services layer
// to the hexagonal outbound ports used by segment use cases.
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
	return services.FetchSegmentClimbProgressionByActivityTypeAndYear(year, metric, targetType, targetId, activityTypes...)
}

func (adapter *SegmentServiceAdapter) FindSegmentsByYearMetricQueryRangeAndTypes(
	year *int,
	metric *string,
	query *string,
	from *string,
	to *string,
	activityTypes ...business.ActivityType,
) []business.SegmentClimbTargetSummary {
	return services.ListSegmentsByActivityTypeAndYear(year, metric, query, from, to, activityTypes...)
}

func (adapter *SegmentServiceAdapter) FindSegmentEffortsByYearMetricRangeAndTypes(
	year *int,
	metric *string,
	segmentID int64,
	from *string,
	to *string,
	activityTypes ...business.ActivityType,
) []business.SegmentClimbAttempt {
	return services.FetchSegmentEffortsByActivityTypeAndYear(year, metric, segmentID, from, to, activityTypes...)
}

func (adapter *SegmentServiceAdapter) FindSegmentSummaryByYearMetricRangeAndTypes(
	year *int,
	metric *string,
	segmentID int64,
	from *string,
	to *string,
	activityTypes ...business.ActivityType,
) *domain.SegmentSummary {
	summary := services.FetchSegmentSummaryByActivityTypeAndYear(year, metric, segmentID, from, to, activityTypes...)
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
