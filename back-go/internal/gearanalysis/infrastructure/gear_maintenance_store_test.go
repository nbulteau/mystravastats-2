package infrastructure

import (
	"testing"

	"mystravastats/internal/shared/domain/business"
)

func TestNormalizeGearMaintenanceRequest_AcceptsFreeFormComponent(t *testing.T) {
	request := business.GearMaintenanceRecordRequest{
		GearID:    " b123 ",
		Component: "Rear valve core",
		Operation: "",
		Date:      "2026-04-27T12:00:00Z",
		Distance:  3603000,
		Note:      " slow leak ",
	}

	normalized, err := normalizeGearMaintenanceRequest(request)
	if err != nil {
		t.Fatalf("expected free-form component to be accepted: %v", err)
	}

	if normalized.GearID != "b123" {
		t.Fatalf("unexpected gear id: %q", normalized.GearID)
	}
	if normalized.Component != "VALVE_CORE_REAR" {
		t.Fatalf("unexpected component key: %q", normalized.Component)
	}
	if normalized.Action != business.GearMaintenanceActionService {
		t.Fatalf("unexpected action: %q", normalized.Action)
	}
	if normalized.Operation != "Rear valve core serviced" {
		t.Fatalf("unexpected default operation: %q", normalized.Operation)
	}
	if normalized.Date != "2026-04-27" {
		t.Fatalf("unexpected date: %q", normalized.Date)
	}
	if normalized.Note != "slow leak" {
		t.Fatalf("unexpected note: %q", normalized.Note)
	}
}

func TestNormalizeGearMaintenanceRequest_DistinguishesReplacement(t *testing.T) {
	request := business.GearMaintenanceRecordRequest{
		GearID:    "b123",
		Component: "Chain",
		Action:    "replacement",
		Date:      "2026-04-27",
		Distance:  3603000,
	}

	normalized, err := normalizeGearMaintenanceRequest(request)
	if err != nil {
		t.Fatalf("expected replacement request to be accepted: %v", err)
	}

	if normalized.Component != "CHAIN" {
		t.Fatalf("unexpected component key: %q", normalized.Component)
	}
	if normalized.Action != business.GearMaintenanceActionReplacement {
		t.Fatalf("unexpected action: %q", normalized.Action)
	}
	if normalized.Operation != "Chain replaced" {
		t.Fatalf("unexpected default replacement operation: %q", normalized.Operation)
	}
}

func TestSaveGearMaintenanceRecords_PersistsFreeFormComponent(t *testing.T) {
	cacheRoot := t.TempDir()
	records := []business.GearMaintenanceRecord{
		{
			ID:        "gm-1",
			GearID:    "b123",
			GearName:  "Gravel bike",
			Component: "Rear valve core",
			Operation: "Rear valve core changed",
			Date:      "2026-04-27",
			Distance:  3603000,
			CreatedAt: "2026-04-27T12:00:00Z",
			UpdatedAt: "2026-04-27T12:00:00Z",
		},
	}

	if err := saveGearMaintenanceRecords(cacheRoot, "athlete-1", records); err != nil {
		t.Fatalf("failed to save maintenance records: %v", err)
	}

	loaded := loadGearMaintenanceRecords(cacheRoot, "athlete-1")
	if len(loaded) != 1 {
		t.Fatalf("expected one record, got %#v", loaded)
	}
	if loaded[0].Component != "VALVE_CORE_REAR" {
		t.Fatalf("unexpected component key: %q", loaded[0].Component)
	}
	if loaded[0].ComponentLabel != "Rear valve core" {
		t.Fatalf("unexpected component label: %q", loaded[0].ComponentLabel)
	}
	if loaded[0].Action != business.GearMaintenanceActionReplacement {
		t.Fatalf("expected changed operation to normalize as replacement, got %q", loaded[0].Action)
	}
}
