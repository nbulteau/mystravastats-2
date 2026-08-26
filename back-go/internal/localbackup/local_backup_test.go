package localbackup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExportAndRestoreWhitelistedLocalData(t *testing.T) {
	cacheRoot := t.TempDir()
	athleteID := "athlete-1"
	directory := athleteDirectory(cacheRoot, athleteID)
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	goalsPath := filepath.Join(directory, "annual-goals-athlete-1.json")
	if err := os.WriteFile(goalsPath, []byte(`{"goals":{"2026:Ride":{"distance":1000}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "activities-athlete-1-2026.json"), []byte(`[{"id":1}]`), 0600); err != nil {
		t.Fatal(err)
	}

	bundle, err := Export(cacheRoot, athleteID)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(bundle.Files) != 1 || bundle.Files["annualGoals"] == nil {
		t.Fatalf("unexpected exported files: %v", bundle.Files)
	}

	if err := os.WriteFile(goalsPath, []byte(`{"goals":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := Restore(cacheRoot, athleteID, bundle)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if len(result.Restored) != 1 || result.Restored[0] != "annualGoals" {
		t.Fatalf("unexpected restore result: %#v", result)
	}
	data, err := os.ReadFile(goalsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(bundle.Files["annualGoals"]) {
		t.Fatalf("restored content = %s", data)
	}
	backup, err := os.ReadFile(goalsPath + ".bak")
	if err != nil || string(backup) != `{"goals":{}}` {
		t.Fatalf("backup = %s, err=%v", backup, err)
	}
}

func TestRestoreRejectsWrongAthleteUnknownKeysAndInvalidJSON(t *testing.T) {
	tests := []Bundle{
		{Version: Version, AthleteID: "other", Files: map[string]json.RawMessage{}},
		{Version: Version, AthleteID: "athlete-1", Files: map[string]json.RawMessage{"activities": json.RawMessage(`[]`)}},
		{Version: Version, AthleteID: "athlete-1", Files: map[string]json.RawMessage{"annualGoals": json.RawMessage(`{`)}},
	}
	for _, bundle := range tests {
		if _, err := Restore(t.TempDir(), "athlete-1", bundle); err == nil {
			t.Fatalf("expected restore rejection for %#v", bundle)
		}
	}
}
