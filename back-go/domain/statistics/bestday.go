package statistics

import (
	"fmt"
	"mystravastats/domain/helpers"
	"mystravastats/domain/strava"
	"time"
)

type BestDayStatistic struct {
	BaseStatistic
	formatString string
	function     func([]*strava.Activity) *Pair
}

type Pair struct {
	Date  string
	Value float64
}

func NewBestDayStatistic(name string, activities []*strava.Activity, formatString string, function func([]*strava.Activity) *Pair) *BestDayStatistic {
	return &BestDayStatistic{
		BaseStatistic: BaseStatistic{
			name:       name,
			Activities: activities,
		},
		formatString: formatString,
		function:     function,
	}
}

func (stat *BestDayStatistic) Value() string {
	pair := stat.function(stat.BaseStatistic.Activities)
	if pair != nil {
		date, _ := time.Parse("2006-01-02", pair.Date)
		return fmt.Sprintf(stat.formatString, pair.Value, date.Format(helpers.DateFormatter))
	}
	return "Not available"
}

func (stat *BestDayStatistic) String() string {
	return stat.Value()
}
