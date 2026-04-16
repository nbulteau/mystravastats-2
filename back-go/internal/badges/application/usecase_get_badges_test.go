package application

import (
	"mystravastats/internal/shared/domain/business"
	"testing"
)

type badgesReaderStub struct {
	generalCalls int
	famousCalls  int
	general      []business.BadgeCheckResult
	famous       []business.BadgeCheckResult
}

func (stub *badgesReaderStub) FindGeneralBadges(_ *int, _ ...business.ActivityType) []business.BadgeCheckResult {
	stub.generalCalls++
	return stub.general
}

func (stub *badgesReaderStub) FindFamousBadges(_ *int, _ ...business.ActivityType) []business.BadgeCheckResult {
	stub.famousCalls++
	return stub.famous
}

func TestGetBadgesUseCase_Execute_WithGeneralFilter(t *testing.T) {
	// GIVEN
	reader := &badgesReaderStub{general: []business.BadgeCheckResult{{}}}
	useCase := NewGetBadgesUseCase(reader)
	set := business.GENERAL

	// WHEN
	result := useCase.Execute(nil, &set, []business.ActivityType{business.Ride})

	// THEN
	if len(result) != 1 {
		t.Fatalf("expected 1 badge, got %d", len(result))
	}
	if reader.generalCalls != 1 || reader.famousCalls != 0 {
		t.Fatalf("unexpected call counts: general=%d famous=%d", reader.generalCalls, reader.famousCalls)
	}
}

func TestGetBadgesUseCase_Execute_WithNoFilter_MergesBothSets(t *testing.T) {
	// GIVEN
	reader := &badgesReaderStub{
		general: []business.BadgeCheckResult{{}, {}},
		famous:  []business.BadgeCheckResult{{}},
	}
	useCase := NewGetBadgesUseCase(reader)

	// WHEN
	result := useCase.Execute(nil, nil, []business.ActivityType{business.Ride})

	// THEN
	if len(result) != 3 {
		t.Fatalf("expected 3 badges, got %d", len(result))
	}
	if reader.generalCalls != 1 || reader.famousCalls != 1 {
		t.Fatalf("unexpected call counts: general=%d famous=%d", reader.generalCalls, reader.famousCalls)
	}
}
