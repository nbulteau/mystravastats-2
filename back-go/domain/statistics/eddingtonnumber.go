package statistics

import (
	"fmt"
	"mystravastats/domain/strava"
	"strings"
)

type EddingtonStatistic struct {
	BaseStatistic
	eddingtonNumber int
	counts          []int
}

func NewEddingtonStatistic(activities []*strava.Activity) *EddingtonStatistic {
	stat := &EddingtonStatistic{
		BaseStatistic: BaseStatistic{
			name:       "Eddington number",
			Activities: activities,
		},
	}
	stat.eddingtonNumber = stat.processEddingtonNumber()
	return stat
}

func (stat *EddingtonStatistic) Value() string {
	return fmt.Sprintf("%d km", stat.eddingtonNumber)
}

func (stat *EddingtonStatistic) String() string {
	return stat.Value()
}

func (stat *EddingtonStatistic) processEddingtonNumber() int {
	if len(stat.BaseStatistic.Activities) == 0 {
		stat.counts = []int{}
		return 0
	}

	activeDaysMap := make(map[string]int)
	for _, activity := range stat.BaseStatistic.Activities {
		date := strings.Split(activity.StartDateLocal, "T")[0]
		activeDaysMap[date] += int(activity.Distance / 1000)
	}

	maxDistance := 0
	for _, distance := range activeDaysMap {
		if distance > maxDistance {
			maxDistance = distance
		}
	}

	stat.counts = make([]int, maxDistance)
	for _, distance := range activeDaysMap {
		for day := distance; day > 0; day-- {
			stat.counts[day-1]++
		}
	}

	for day := len(stat.counts); day > 0; day-- {
		if stat.counts[day-1] >= day {
			return day
		}
	}

	return 0
}
