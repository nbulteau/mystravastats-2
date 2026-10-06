package strava

import (
	"encoding/json"
	"math"
)

// PowerSamples preserves sample indexes. NaN is the explicit in-memory marker
// for an unavailable reading; JSON (including disk caches) uses null. Zero is
// always a measured value. Consumers must check validity before arithmetic.
type PowerSamples []float64

func (samples *PowerSamples) UnmarshalJSON(data []byte) error {
	var values []*float64
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	if values == nil {
		*samples = nil
		return nil
	}
	result := make(PowerSamples, len(values))
	for i, value := range values {
		result[i] = math.NaN()
		if value != nil {
			result[i] = *value
		}
	}
	*samples = result
	return nil
}

func (samples PowerSamples) MarshalJSON() ([]byte, error) {
	if samples == nil {
		return []byte("null"), nil
	}
	values := make([]*float64, len(samples))
	for i, value := range samples {
		if !math.IsNaN(value) && !math.IsInf(value, 0) {
			values[i] = &value
		}
	}
	return json.Marshal(values)
}
