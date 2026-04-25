package business

import "testing"

func TestRepresentativeBadgeActivityType_ResolvesSportFamilies(t *testing.T) {
	tests := []struct {
		name     string
		input    []ActivityType
		expected ActivityType
		ok       bool
	}{
		{
			name:     "cycling family",
			input:    []ActivityType{GravelRide, MountainBikeRide, Ride, VirtualRide, Commute},
			expected: Ride,
			ok:       true,
		},
		{
			name:     "running family",
			input:    []ActivityType{TrailRun, Run},
			expected: Run,
			ok:       true,
		},
		{
			name:     "hiking family",
			input:    []ActivityType{Walk, Hike},
			expected: Hike,
			ok:       true,
		},
		{
			name:  "unsupported family",
			input: []ActivityType{AlpineSki},
			ok:    false,
		},
		{
			name:  "mixed families",
			input: []ActivityType{Ride, Run},
			ok:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, ok := RepresentativeBadgeActivityType(tt.input...)
			if ok != tt.ok {
				t.Fatalf("expected ok=%v, got %v", tt.ok, ok)
			}
			if ok && actual != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, actual)
			}
		})
	}
}
