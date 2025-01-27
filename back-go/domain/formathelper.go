package domain

import (
	"fmt"
	"time"
)

var inDateTimeFormatter = "2006-01-02T15:04:05Z"
var outDateTimeFormatter = "Mon 02 January 2006 - 15:04"
var dateFormatter = "Mon 02 January 2006"

func formatDate(dateStr string) string {
	t, err := time.Parse(inDateTimeFormatter, dateStr)
	if err != nil {
		return ""
	}
	return t.Format(outDateTimeFormatter)
}

func formatSeconds(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	if hours == 0 {
		if minutes == 0 {
			return fmt.Sprintf("%02ds", secs)
		}
		return fmt.Sprintf("%02dm %02ds", minutes, secs)
	}

	if minutes == 0 && secs == 0 {
		return fmt.Sprintf("%2dh", hours)
	}

	return fmt.Sprintf("%02dh %02dm %02ds", hours, minutes, secs)
}

func formatSecondsFloat(seconds float64) string {
	minutes := int(seconds) / 60
	secs := int(seconds) % 60
	hundredths := int((seconds - float64(minutes*60+secs)) * 100)

	if hundredths == 100 {
		secs++
		if secs == 60 {
			secs = 0
			minutes++
		}
	}

	return fmt.Sprintf("%d'%02d", minutes, secs)
}

func formatSpeed(speed float64, activityType string) string {
	if activityType == "Run" {
		return fmt.Sprintf("%s/km", formatSecondsFloat(1000/speed))
	}
	return fmt.Sprintf("%.02f km/h", speed*3.6)
}
