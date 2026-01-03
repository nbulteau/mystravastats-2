package statistics

import (
	"fmt"
	"mystravastats/domain/strava"
	"sort"
	"strings"
	"time"
)

// EddingtonStatistic computes the Eddington number for a set of activities.
// The Eddington number E is the largest number such that you have cycled at least E km on at least E days.
type EddingtonStatistic struct {
	BaseStatistic
	eddingtonNumber    int   // The computed Eddington number
	daysWithAtLeastXKm []int // daysWithAtLeastXKm[i] is the number of days with at least (i+1) km
}

// NewEddingtonStatistic creates a new EddingtonStatistic from a list of activities.
// If activities is nil it's treated as an empty slice.
func NewEddingtonStatistic(activities []*strava.Activity) *EddingtonStatistic {
	if activities == nil {
		activities = []*strava.Activity{}
	}

	stat := &EddingtonStatistic{
		BaseStatistic: BaseStatistic{
			name:       "Eddington number",
			Activities: activities,
		},
	}
	stat.eddingtonNumber = stat.processEddingtonNumber()
	return stat
}

// Value returns the Eddington number as a formatted string.
func (stat *EddingtonStatistic) Value() string {
	return fmt.Sprintf("%d km", stat.eddingtonNumber)
}

// String returns the string representation of the Eddington number.
func (stat *EddingtonStatistic) String() string {
	return stat.Value()
}

// processEddingtonNumber calculates the Eddington number based on the activities.
// Uses an optimized algorithm with sorting for better performance.
func (stat *EddingtonStatistic) processEddingtonNumber() int {
	if len(stat.BaseStatistic.Activities) == 0 {
		stat.daysWithAtLeastXKm = []int{}
		return 0
	}

	// Group total distance per day (in kilometers) with better date parsing
	activeDaysMap := make(map[string]int)
	for _, activity := range stat.BaseStatistic.Activities {
		if activity == nil || activity.Distance <= 0 {
			continue // Ignore nil or non-positive distance activities
		}

		// More robust date parsing
		date, err := parseActivityDate(activity.StartDateLocal)
		if err != nil {
			continue // Skip activities with invalid dates
		}

		// Note: integer kilometers (truncated). Adjust if you prefer rounding.
		km := int(activity.Distance / 1000)
		if km > 0 {
			activeDaysMap[date] += km
		}
	}

	if len(activeDaysMap) == 0 {
		stat.daysWithAtLeastXKm = []int{}
		return 0
	}

	// Convert to slice and sort for optimized algorithm
	dailyDistances := make([]int, 0, len(activeDaysMap))
	for _, distance := range activeDaysMap {
		dailyDistances = append(dailyDistances, distance)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(dailyDistances)))

	// Find Eddington number using optimized algorithm
	eddingtonNumber := 0
	for i, distance := range dailyDistances {
		days := i + 1
		if distance >= days {
			eddingtonNumber = days
		} else {
			break
		}
	}

	// Build daysWithAtLeastXKm array for compatibility (optimized)
	stat.buildDaysWithAtLeastXKmArray(dailyDistances)

	return eddingtonNumber
}

// parseActivityDate extracts date from activity start date string.
func parseActivityDate(startDateLocal string) (string, error) {
	s := strings.TrimSpace(startDateLocal)
	if s == "" {
		return "", fmt.Errorf("empty date string")
	}

	// Try several common layouts
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05", // fallback with space
		"2006-01-02",          // date only
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02"), nil
		}
	}

	// Fallback to splitting on 'T' or space if parsing failed
	if parts := strings.Split(s, "T"); len(parts) > 0 && parts[0] != "" {
		return parts[0], nil
	}
	if parts := strings.Split(s, " "); len(parts) > 0 && parts[0] != "" {
		return parts[0], nil
	}

	return "", fmt.Errorf("invalid date format: %s", startDateLocal)
}

// buildDaysWithAtLeastXKmArray constructs the daysWithAtLeastXKm array from sorted distances.
func (stat *EddingtonStatistic) buildDaysWithAtLeastXKmArray(sortedDistances []int) {
	if len(sortedDistances) == 0 {
		stat.daysWithAtLeastXKm = []int{}
		return
	}

	maxDistance := sortedDistances[0]
	if maxDistance <= 0 {
		stat.daysWithAtLeastXKm = []int{}
		return
	}

	// Use counting + cumulative sum to build the array in O(n + maxDistance)
	counts := make([]int, maxDistance+1) // index = km, counts[0] unused
	for _, d := range sortedDistances {
		if d <= 0 {
			continue
		}
		if d > maxDistance {
			d = maxDistance
		}
		counts[d]++
	}

	stat.daysWithAtLeastXKm = make([]int, maxDistance)
	cumulative := 0
	for km := maxDistance; km >= 1; km-- {
		cumulative += counts[km]
		stat.daysWithAtLeastXKm[km-1] = cumulative
	}
}
