package api

import (
	"net/http/httptest"
	"testing"

	"mystravastats/internal/shared/domain/business"
)

func TestGetActivityTypeParam_AcceptsWalk(t *testing.T) {
	// GIVEN
	request := httptest.NewRequest("GET", "/api/statistics?activityType=Walk", nil)

	// WHEN
	activityTypes, err := getActivityTypeParam(request)

	// THEN
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(activityTypes) != 1 || activityTypes[0] != business.Walk {
		t.Fatalf("expected [Walk], got %v", activityTypes)
	}
}

func TestDefaultRouteGenerationActivityTypes_IncludesWalk(t *testing.T) {
	// GIVEN / WHEN
	activityTypes := defaultRouteGenerationActivityTypes()

	// THEN
	foundWalk := false
	for _, activityType := range activityTypes {
		if activityType == business.Walk {
			foundWalk = true
			break
		}
	}
	if !foundWalk {
		t.Fatalf("expected default route activity types to include Walk, got %v", activityTypes)
	}
}
