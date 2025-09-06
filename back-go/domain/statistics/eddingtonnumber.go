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
// Returns nil if activities slice is nil.
func NewEddingtonStatistic(activities []*strava.Activity) *EddingtonStatistic {
	if activities == nil {
		return nil
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

	// Build daysWithAtLeastXKm array for compatibility
	stat.buildDaysWithAtLeastXKmArray(dailyDistances)

	return eddingtonNumber
}

// parseActivityDate extracts date from activity start date string.
func parseActivityDate(startDateLocal string) (string, error) {
	if startDateLocal == "" {
		return "", fmt.Errorf("empty date string")
	}

	// Try parsing ISO format first
	if t, err := time.Parse(time.RFC3339, startDateLocal); err == nil {
		return t.Format("2006-01-02"), nil
	}

	// Fallback to simple string split
	parts := strings.Split(startDateLocal, "T")
	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("invalid date format: %s", startDateLocal)
	}

	return parts[0], nil
}

// buildDaysWithAtLeastXKmArray constructs the daysWithAtLeastXKm array from sorted distances.
func (stat *EddingtonStatistic) buildDaysWithAtLeastXKmArray(sortedDistances []int) {
	if len(sortedDistances) == 0 {
		stat.daysWithAtLeastXKm = []int{}
		return
	}

	maxDistance := sortedDistances[0]
	stat.daysWithAtLeastXKm = make([]int, maxDistance)

	// Count days with at least X km using sorted array
	for km := 1; km <= maxDistance; km++ {
		count := 0
		for _, distance := range sortedDistances {
			if distance >= km {
				count++
			} else {
				break // Since array is sorted in descending order
			}
		}
		stat.daysWithAtLeastXKm[km-1] = count
	}
}
