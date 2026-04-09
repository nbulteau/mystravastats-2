package helpers

import (
	"fmt"
	"math"
	"time"
)

const roundingThreshold = 0.995
const roundingEpsilon = 1e-9

var inDateTimeFormatter = "2006-01-02T15:04:05Z"
var outDateTimeFormatter = "Mon 02 January 2006 - 15:04"
var DateFormatter = "Mon 02 January 2006"

func formatDate(dateStr string) string {
	t, err := time.Parse(inDateTimeFormatter, dateStr)
	if err != nil {
		return ""
	}
	return t.Format(outDateTimeFormatter)
}

func FormatSeconds(seconds int) string {
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

func FormatSecondsFloat(seconds float64) string {
	roundedSeconds := int(math.Round(seconds))
	if roundedSeconds > 0 && roundedSeconds != int(seconds) {
		fractional := seconds - math.Floor(seconds)
		if fractional+roundingEpsilon < roundingThreshold {
			roundedSeconds--
		}
	}

	minutes := roundedSeconds / 60
	secs := roundedSeconds % 60

	return fmt.Sprintf("%d'%02d", minutes, secs)
}
