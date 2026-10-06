package statistics

import (
	"encoding/json"
	"mystravastats/internal/shared/domain/strava"
	"os"
	"testing"
)

func TestSharedPowerGaps(t *testing.T) {
	data, err := os.ReadFile("../../../test-fixtures/api/power-gaps.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Watts    strava.PowerSamples
		Expected *float64
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	ClearBestEffortCache()
	for _, tc := range cases {
		stream := syntheticStream([]float64{0, 10, 20, 30, 40, 50}, []int{0, 1, 2, 3, 4, 5}, []float64{100, 100, 100, 100, 100, 100})
		stream.Watts = &strava.PowerStream{Data: tc.Watts}
		// Persist and reload exactly as the activity cache does.
		encoded, err := json.Marshal(stream)
		if err != nil {
			t.Fatal(err)
		}
		path := t.TempDir() + "/stream.json"
		if err := os.WriteFile(path, encoded, 0600); err != nil {
			t.Fatal(err)
		}
		cached, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var restored strava.Stream
		if err := json.Unmarshal(cached, &restored); err != nil {
			t.Fatal(err)
		}
		activity := strava.Activity{Id: 991, Name: "Gap", Type: "Ride", Stream: &restored}
		timeEffort, distanceEffort := BestPowerForTime(activity, 2), BestPowerForDistance(activity, 20)
		if tc.Expected == nil {
			if timeEffort != nil || distanceEffort != nil {
				t.Fatalf("expected no complete window: %s", encoded)
			}
			continue
		}
		if timeEffort == nil || timeEffort.AveragePower == nil || *timeEffort.AveragePower != *tc.Expected {
			t.Fatalf("unexpected time effort: %+v", timeEffort)
		}
		if distanceEffort == nil || distanceEffort.AveragePower == nil || *distanceEffort.AveragePower != *tc.Expected {
			t.Fatalf("unexpected distance effort: %+v", distanceEffort)
		}
	}
}
