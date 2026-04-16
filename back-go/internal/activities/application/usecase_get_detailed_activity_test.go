package application

import (
	"errors"
	activitiesDomain "mystravastats/internal/activities/domain"
	"mystravastats/internal/shared/domain/strava"
	"testing"
)

type detailedActivityReaderStub struct {
	activity *strava.DetailedActivity
	err      error
	calls    int
}

func (stub *detailedActivityReaderStub) FindDetailedActivityByID(_ int64) (*strava.DetailedActivity, error) {
	stub.calls++
	return stub.activity, stub.err
}

func TestGetDetailedActivityUseCase_Execute_ReturnsInvalidIDError(t *testing.T) {
	// GIVEN
	reader := &detailedActivityReaderStub{}
	useCase := NewGetDetailedActivityUseCase(reader)

	// WHEN
	activity, err := useCase.Execute(0)

	// THEN
	if !errors.Is(err, activitiesDomain.ErrInvalidActivityID) {
		t.Fatalf("expected ErrInvalidActivityID, got: %v", err)
	}
	if activity != nil {
		t.Fatalf("expected nil activity, got: %+v", activity)
	}
	if reader.calls != 0 {
		t.Fatalf("reader should not be called for invalid ids, got %d call(s)", reader.calls)
	}
}

func TestGetDetailedActivityUseCase_Execute_ReturnsNotFoundError(t *testing.T) {
	// GIVEN
	reader := &detailedActivityReaderStub{}
	useCase := NewGetDetailedActivityUseCase(reader)

	// WHEN
	activity, err := useCase.Execute(42)

	// THEN
	if !errors.Is(err, activitiesDomain.ErrDetailedActivityNotFound) {
		t.Fatalf("expected ErrDetailedActivityNotFound, got: %v", err)
	}
	if activity != nil {
		t.Fatalf("expected nil activity, got: %+v", activity)
	}
	if reader.calls != 1 {
		t.Fatalf("expected reader to be called once, got %d call(s)", reader.calls)
	}
}

func TestGetDetailedActivityUseCase_Execute_ReturnsDetailedActivity(t *testing.T) {
	// GIVEN
	expected := &strava.DetailedActivity{Id: 4242, Name: "test activity"}
	reader := &detailedActivityReaderStub{
		activity: expected,
	}
	useCase := NewGetDetailedActivityUseCase(reader)

	// WHEN
	activity, err := useCase.Execute(4242)

	// THEN
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if activity == nil || activity.Id != expected.Id {
		t.Fatalf("expected activity %d, got: %+v", expected.Id, activity)
	}
	if reader.calls != 1 {
		t.Fatalf("expected reader to be called once, got %d call(s)", reader.calls)
	}
}
