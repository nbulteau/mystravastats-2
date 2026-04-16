package application

import (
	segmentsDomain "mystravastats/internal/segments/domain"
	"mystravastats/internal/shared/domain/business"
)

// SegmentsReader is an outbound port used by segment-related use cases.
// Infrastructure adapters implement this interface.
type SegmentsReader interface {
	FindSegmentClimbProgressionByYearMetricTargetAndTypes(
		year *int,
		metric *string,
		targetType *string,
		targetId *int64,
		activityTypes ...business.ActivityType,
	) business.SegmentClimbProgression
	FindSegmentsByYearMetricQueryRangeAndTypes(
		year *int,
		metric *string,
		query *string,
		from *string,
		to *string,
		activityTypes ...business.ActivityType,
	) []business.SegmentClimbTargetSummary
	FindSegmentEffortsByYearMetricRangeAndTypes(
		year *int,
		metric *string,
		segmentID int64,
		from *string,
		to *string,
		activityTypes ...business.ActivityType,
	) []business.SegmentClimbAttempt
	FindSegmentSummaryByYearMetricRangeAndTypes(
		year *int,
		metric *string,
		segmentID int64,
		from *string,
		to *string,
		activityTypes ...business.ActivityType,
	) *segmentsDomain.SegmentSummary
}
