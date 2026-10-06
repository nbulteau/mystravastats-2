package statistics

import (
	"encoding/json"
	"math"
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

func TestTimedPowerEffortUsesSharedWindows(t *testing.T) {
	data, err := os.ReadFile("../../../test-fixtures/api/power-timing.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name    string
		Times   []int
		Watts   strava.PowerSamples
		Seconds int
		Best    *float64
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			distances, altitudes := make([]float64, len(tc.Times)), make([]float64, len(tc.Times))
			for i, v := range tc.Times {
				distances[i] = float64(v) * 2
				altitudes[i] = 100
			}
			stream := syntheticStream(distances, tc.Times, altitudes)
			stream.Watts = &strava.PowerStream{Data: tc.Watts}
			got := BestPowerForTime(strava.Activity{Id: 995, Type: "Ride", Stream: stream}, tc.Seconds)
			if tc.Best == nil {
				if got != nil {
					t.Fatal("expected no complete window")
				}
				return
			}
			if got == nil || got.AveragePower == nil {
				t.Fatal("missing complete window")
			}
			if math.Abs(*got.AveragePower-*tc.Best) > 1e-7 || got.Seconds != tc.Seconds || math.Abs(got.Distance-float64(tc.Seconds)*2) > 1e-7 {
				t.Fatalf("unexpected exact duration effort: %+v", got)
			}
		})
	}
}

func TestBestPowerAcrossActivitiesUsesPowerNotDistance(t *testing.T) {
	activities := []*strava.Activity{}
	for i, power := range []float64{200, 300} {
		stream := syntheticStream([]float64{0, float64(100 / (i + 1)), float64(200 / (i + 1))}, []int{0, 5, 10}, []float64{100, 100, 100})
		stream.Watts = &strava.PowerStream{Data: []float64{power, power, power}}
		activities = append(activities, &strava.Activity{Id: int64(996 + i), Type: "Ride", Stream: stream})
	}
	result := calculateBestPowerForTime(activities, 10)
	if result == nil || result.ActivityShort.Id != 997 {
		t.Fatalf("highest power must win: %+v", result)
	}
}
