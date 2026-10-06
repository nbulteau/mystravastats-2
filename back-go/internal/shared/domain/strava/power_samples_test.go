package strava

import (
	"encoding/json"
	"math"
	"testing"
)

func TestPowerSamplesRoundTrip(t *testing.T) {
	for _, raw := range []string{`[0,null,200]`, `[null,null]`, `[]`, `null`} {
		var samples PowerSamples
		if err := json.Unmarshal([]byte(raw), &samples); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(samples)
		if err != nil {
			t.Fatal(err)
		}
		if string(encoded) != raw {
			t.Fatalf("got %s, expected %s", encoded, raw)
		}
	}
	encoded, err := json.Marshal(PowerSamples{0, math.NaN(), math.Inf(1)})
	if err != nil || string(encoded) != `[0,null,null]` {
		t.Fatalf("got %s, %v", encoded, err)
	}
}
