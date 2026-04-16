package domain

import "mystravastats/internal/shared/domain/business"

type SegmentSummary struct {
	Metric         string
	Segment        business.SegmentClimbTargetSummary
	PersonalRecord *business.SegmentClimbAttempt
	TopEfforts     []business.SegmentClimbAttempt
	Attempts       []business.SegmentClimbAttempt
}
