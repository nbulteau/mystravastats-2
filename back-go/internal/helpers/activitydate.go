package helpers

import (
	"strings"
	"time"
)

var stravaDateLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05.999999-0700",
	"2006-01-02T15:04:05-0700",
	"2006-01-02T15:04:05.000000Z07:00",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05.999999",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

func ParseActivityDate(value string) (time.Time, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, false
	}
	for _, layout := range stravaDateLayouts {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func ExtractSortableDay(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < 10 {
		return ""
	}
	candidate := trimmed[:10]
	if _, err := time.Parse("2006-01-02", candidate); err != nil {
		return ""
	}
	return candidate
}

func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func IsBeforeActivityDate(left, right string) bool {
	leftTime, leftOK := ParseActivityDate(left)
	rightTime, rightOK := ParseActivityDate(right)
	switch {
	case leftOK && rightOK:
		return leftTime.Before(rightTime)
	case leftOK && !rightOK:
		return true
	case !leftOK && rightOK:
		return false
	default:
		return left < right
	}
}
