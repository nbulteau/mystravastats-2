package statistics

import (
	"fmt"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
	"time"
)

type Statistic interface {
	Label() string
	Value() string
	Activity() *business.ActivityShort
}

type BaseStatistic struct {
	name       string
	Activities []*strava.Activity
}

func (stat *BaseStatistic) Label() string {
	return stat.name
}

func (stat *BaseStatistic) Value() string {
	return stat.Value()
}

func (stat *BaseStatistic) Activity() *business.ActivityShort {
	return nil
}

type GlobalStatistic struct {
	BaseStatistic
	Function func([]*strava.Activity) string
}

func (stat *GlobalStatistic) Value() string {
	return stat.Function(stat.Activities)
}

func (stat *GlobalStatistic) Label() string {
	return stat.BaseStatistic.Label()
}

func (stat *GlobalStatistic) Activity() *business.ActivityShort {
	return nil
}

func NewGlobalStatistic(name string, activities []*strava.Activity, function func([]*strava.Activity) string) *GlobalStatistic {
	return &GlobalStatistic{
		BaseStatistic: BaseStatistic{name: name, Activities: activities},
		Function:      function,
	}
}

type ActivityStatistic struct {
	BaseStatistic
	activity *business.ActivityShort
}

func (stat *ActivityStatistic) Value() string {
	return fmt.Sprintf("%s - %v", stat.BaseStatistic.Label(), stat.Activity)
}

func (stat *ActivityStatistic) Label() string {
	return stat.BaseStatistic.Label()
}

func (stat *ActivityStatistic) Activity() *business.ActivityShort {
	return stat.activity
}

func NewActivityStatistic(name string, activities []*strava.Activity) *ActivityStatistic {
	return &ActivityStatistic{
		BaseStatistic: BaseStatistic{name: name, Activities: activities},
	}
}

func formatSeconds(seconds int) string {
	return time.Duration(seconds * int(time.Second)).String()
}

func averagePower(watts []float64, idxStart int, idxEnd int) float64 {
	averagePower := 0.0
	if watts != nil && len(watts) > 0 {
		sumPower := 0.0
		for i := idxStart; i <= idxEnd; i++ {
			sumPower += watts[i]
		}
		averagePower = sumPower / float64(idxEnd-idxStart+1)
	}
	return averagePower
}
