package statistics

import (
	"mystravastats/domain/strava"
	"testing"
)

func TestEddingtonStatistic_processEddingtonNumber(t *testing.T) {
	tests := []struct {
		name       string
		activities []*strava.Activity
		expected   int
		// Expected counts for detailed verification
		expectedCounts []int
	}{
		{
			name:           "No activities",
			activities:     []*strava.Activity{},
			expected:       0,
			expectedCounts: []int{},
		},
		{
			name: "Multiple activities same day",
			activities: []*strava.Activity{
				{StartDateLocal: "2023-01-01T10:00:00Z", Distance: 10000}, // 10 km
				{StartDateLocal: "2023-01-01T15:00:00Z", Distance: 5000},  // 5 km
			},
			expected:       1, // 1 day with 15 km
			expectedCounts: []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		},
		{
			name: "Eddington 5 example",
			activities: []*strava.Activity{
				{StartDateLocal: "2023-01-01T10:00:00Z", Distance: 10000}, // 10 km
				{StartDateLocal: "2023-01-02T10:00:00Z", Distance: 5000},  // 5 km
				{StartDateLocal: "2023-01-03T10:00:00Z", Distance: 7000},  // 7 km
				{StartDateLocal: "2023-01-04T10:00:00Z", Distance: 6000},  // 6 km
				{StartDateLocal: "2023-01-05T10:00:00Z", Distance: 8000},  // 8 km
				{StartDateLocal: "2023-01-06T10:00:00Z", Distance: 3000},  // 3 km
			},
			expected:       5, // 5 days with at least 5 km
			expectedCounts: []int{6, 6, 6, 5, 5, 4, 3, 2, 1, 1},
		},
		{
			name: "Activities on multiple days with different distances",
			activities: []*strava.Activity{
				{StartDateLocal: "2023-01-01T10:00:00Z", Distance: 20000}, // 20 km
				{StartDateLocal: "2023-01-02T10:00:00Z", Distance: 15000}, // 15 km
				{StartDateLocal: "2023-01-03T10:00:00Z", Distance: 10000}, // 10 km
				{StartDateLocal: "2023-01-04T10:00:00Z", Distance: 5000},  // 5 km
				{StartDateLocal: "2023-01-05T10:00:00Z", Distance: 2000},  // 2 km
			},
			expected:       4, // 4 days with at least 4 km
			expectedCounts: []int{5, 5, 4, 4, 4, 3, 3, 3, 3, 3, 2, 2, 2, 2, 2, 1, 1, 1, 1, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stat := &EddingtonStatistic{
				BaseStatistic: BaseStatistic{
					name:       "Eddington number",
					Activities: tt.activities,
				},
			}

			result := stat.processEddingtonNumber()

			// Check if the calculated Eddington number is correct
			if result != tt.expected {
				t.Errorf("processEddingtonNumber() = %v, expected %v", result, tt.expected)
			}

			// Check if empty activities case is handled correctly
			if len(tt.activities) == 0 && len(stat.counts) != 0 {
				t.Errorf("For empty activities, counts should be empty, got %v", stat.counts)
			}

			// Check if counts are calculated correctly when expected counts are provided
			if len(tt.expectedCounts) > 0 && len(stat.counts) > 0 {
				if len(stat.counts) != len(tt.expectedCounts) {
					t.Errorf("Length of counts mismatch: got %d, expected %d", len(stat.counts), len(tt.expectedCounts))
				} else {
					for i := range stat.counts {
						if stat.counts[i] != tt.expectedCounts[i] {
							t.Errorf("counts[%d] = %d, expected %d", i, stat.counts[i], tt.expectedCounts[i])
						}
					}
				}
			}
		})
	}
}
