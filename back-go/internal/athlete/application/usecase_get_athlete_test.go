package application

import (
	"mystravastats/domain/strava"
	"testing"
)

type athleteReaderStub struct {
	athlete strava.Athlete
}

func (stub *athleteReaderStub) FindAthlete() strava.Athlete {
	return stub.athlete
}

func TestGetAthleteUseCase_Execute_ReturnsAthlete(t *testing.T) {
	// GIVEN
	expected := strava.Athlete{Id: 42}
	reader := &athleteReaderStub{athlete: expected}
	useCase := NewGetAthleteUseCase(reader)

	// WHEN
	result := useCase.Execute()

	// THEN
	if result.Id != expected.Id {
		t.Fatalf("expected athlete id %d, got %d", expected.Id, result.Id)
	}
}
