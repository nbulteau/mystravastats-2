package strava

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

func TestSharedPowerTiming(t *testing.T) {
	data, err := os.ReadFile("../../../../../test-fixtures/api/power-timing.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name                      string
		Times                     []int
		Watts                     PowerSamples
		Seconds                   int
		Average, Best, Normalized *float64
		Covered                   float64
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			p := NewPowerTimeline(tc.Watts, tc.Times)
			var avg, best *float64
			if len(tc.Times) > 0 {
				avg = p.Average(float64(tc.Times[0]), float64(tc.Times[len(tc.Times)-1]))
			}
			if w := p.Best(tc.Seconds); w != nil {
				best = &w.Average
			}
			check := func(got, want *float64) {
				t.Helper()
				if (got == nil) != (want == nil) {
					t.Fatalf("got %v want %v", got, want)
				}
				if got != nil && math.Abs(*got-*want) > 1e-7 {
					t.Fatalf("got %f want %f", *got, *want)
				}
			}
			check(avg, tc.Average)
			check(best, tc.Best)
			check(p.Normalized(), tc.Normalized)
			covered := 0.0
			for i := 0; i < len(tc.Times)-1; i++ {
				if p.IntervalValid(i) {
					covered += float64(tc.Times[i+1] - tc.Times[i])
				}
			}
			if covered != tc.Covered {
				t.Fatalf("coverage got %f want %f", covered, tc.Covered)
			}
		})
	}
}
