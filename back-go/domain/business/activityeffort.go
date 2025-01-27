package business

import (
	"fmt"
	"strconv"
	"time"
)

type ActivityEffort struct {
	Distance      float64
	Seconds       int
	DeltaAltitude float64
	IdxStart      int
	IdxEnd        int
	AveragePower  *int
	Label         string
	ActivityShort ActivityShort
}

func (ae ActivityEffort) GetFormattedSpeed() string {
	speed := ae.GetSpeed()
	if ae.ActivityShort.Type == Run {
		return fmt.Sprintf("%s/km", speed)
	}
	return fmt.Sprintf("%s km/h", speed)
}

func (ae ActivityEffort) GetSpeed() string {
	if ae.ActivityShort.Type == Run {
		return formatSeconds(ae.Seconds * 1000 / int(ae.Distance))
	}
	return fmt.Sprintf("%.02f", ae.Distance/float64(ae.Seconds)*3600/1000)
}

func (ae ActivityEffort) GetMSSpeed() float64 {
	speed := ae.Distance / float64(ae.Seconds)
	formattedSpeed, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", speed), 64)
	return formattedSpeed
}

func (ae ActivityEffort) GetFormattedGradient() string {
	return fmt.Sprintf("%.02f%%", ae.GetGradient())
}

func (ae ActivityEffort) GetFormattedPower() string {
	if ae.AveragePower != nil {
		return fmt.Sprintf("%d W", *ae.AveragePower)
	}
	return ""
}

func (ae ActivityEffort) GetGradient() float64 {
	return 100 * ae.DeltaAltitude / ae.Distance
}

func (ae ActivityEffort) GetDescription() string {
	return fmt.Sprintf("%s:<ul><li>Distance : %.1f km</li><li>Time : %s</li><li>Speed : %s</li><li>Gradient: %.02f%%</li><li>Power: %s</li></ul>",
		ae.Label,
		ae.Distance/1000,
		formatSeconds(ae.Seconds),
		ae.GetFormattedSpeed(),
		ae.GetGradient(),
		ae.GetFormattedPower())
}

func formatSeconds(seconds int) string {
	return time.Duration(seconds * int(time.Second)).String()
}

func NewActivityEffort(distance float64, seconds int, deltaAltitude float64, idxStart int, idxEnd int, averagePower *int, label string, activityShort ActivityShort) ActivityEffort {
	return ActivityEffort{
		Distance:      distance,
		Seconds:       seconds,
		DeltaAltitude: deltaAltitude,
		IdxStart:      idxStart,
		IdxEnd:        idxEnd,
		AveragePower:  averagePower,
		Label:         label,
		ActivityShort: activityShort,
	}
}
