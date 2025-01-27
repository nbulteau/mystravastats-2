package statistics

import (
	"fmt"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
	"time"
)

type Statistic interface {
	Value() string
	String() string
}

type BaseStatistic struct {
	Name       string
	Activities []strava.Activity
}

func (s *BaseStatistic) Value() string {
	// Implement the Value method
	return fmt.Sprintf("BaseStatistic: %s", s.Name)
}

func (s *BaseStatistic) String() string {
	return s.Value()
}

type GlobalStatistic struct {
	BaseStatistic
	FormatString string
	Function     func([]strava.Activity) float64
}

func (s *GlobalStatistic) Value() string {
	return fmt.Sprintf(s.FormatString, s.Function(s.Activities))
}

func NewGlobalStatistic(name string, activities []strava.Activity, formatString string, function func([]strava.Activity) float64) *GlobalStatistic {
	return &GlobalStatistic{
		BaseStatistic: BaseStatistic{Name: name, Activities: activities},
		FormatString:  formatString,
		Function:      function,
	}
}

type ActivityStatistic struct {
	BaseStatistic
	Activity *business.ActivityShort
}

func (s *ActivityStatistic) Value() string {
	if s.Activity != nil {
		return fmt.Sprintf("%s - %v", s.BaseStatistic.String(), s.Activity)
	}
	return "Not available"
}

func NewActivityStatistic(name string, activities []strava.Activity) *ActivityStatistic {
	return &ActivityStatistic{
		BaseStatistic: BaseStatistic{Name: name, Activities: activities},
	}
}

func formatSeconds(seconds int) string {
	return time.Duration(seconds * int(time.Second)).String()
}

func average(nums []int) *int {
	if len(nums) == 0 {
		return nil
	}
	sum := 0
	for _, num := range nums {
		sum += num
	}
	avg := sum / len(nums)
	return &avg
}
