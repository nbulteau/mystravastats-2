package statistics

import (
	"fmt"
	"mystravastats/domain/strava"
	"time"
)

type MaxStreakStatistic struct {
	BaseStatistic
	maxStreak int
}

func NewMaxStreakStatistic(activities []*strava.Activity) *MaxStreakStatistic {
	stat := &MaxStreakStatistic{
		BaseStatistic: BaseStatistic{
			name:       "Max streak",
			Activities: activities,
		},
	}

	var maxLen int

	if len(activities) > 0 {
		// sort activities by date
		for i := 0; i < len(activities); i++ {
			for j := i + 1; j < len(activities); j++ {
				if activities[i].StartDateLocal < activities[j].StartDateLocal {
					activities[i], activities[j] = activities[j], activities[i]
				}
			}
		}

		lastDate, _ := time.Parse("2006-01-02", activities[0].StartDateLocal[:10])
		firstDate, _ := time.Parse("2006-01-02", activities[len(activities)-1].StartDateLocal[:10])
		firstEpochDay := firstDate.Unix() / 86400

		activeDaysSet := make(map[int]bool)
		for _, activity := range activities {
			date, _ := time.Parse("2006-01-02", activity.StartDateLocal[:10])
			activeDaysSet[int(date.Unix()/86400-firstEpochDay)] = true
		}

		days := int(lastDate.Unix()/86400 - firstDate.Unix()/86400)
		activeDays := make([]bool, days)
		for i := 0; i < days; i++ {
			activeDays[i] = activeDaysSet[i]
		}

		var currLen int
		for k := 0; k < days; k++ {
			if activeDays[k] {
				currLen++
			} else {
				if currLen > maxLen {
					maxLen = currLen
				}
				currLen = 0
			}
		}
	}

	stat.maxStreak = maxLen
	return stat
}

func (stat *MaxStreakStatistic) Value() string {
	return fmt.Sprintf("%d", stat.maxStreak)
}

func (stat *MaxStreakStatistic) String() string {
	return stat.Value()
}
