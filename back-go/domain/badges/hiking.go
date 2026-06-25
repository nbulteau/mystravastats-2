package badges

import (
	"fmt"
	"math"
	"mystravastats/internal/shared/domain/strava"
	"sort"
	"strings"
	"time"
	"unicode"
)

type HikingBadge struct {
	Label       string
	Description string
	matcher     func(activities []*strava.Activity) []*strava.Activity
}

func (h HikingBadge) Check(activities []*strava.Activity) ([]*strava.Activity, bool) {
	if h.matcher == nil {
		return nil, false
	}
	checkedActivities := h.matcher(activities)
	return checkedActivities, len(checkedActivities) > 0
}

func (h HikingBadge) String() string {
	return h.Label
}

var (
	SummitDayBadge = HikingBadge{
		Label:       "Summit Day",
		Description: "Reach a high point above 2000 m with at least 500 m of elevation gain.",
		matcher:     summitDayActivities,
	}
	BackToBackHikingWeekendBadge = HikingBadge{
		Label:       "Back-to-back Hiking Weekend",
		Description: "Record hikes on both Saturday and Sunday of the same weekend.",
		matcher:     backToBackHikingWeekendActivities,
	}
	HighPointPRBadge = HikingBadge{
		Label:       "High Point PR",
		Description: "Your highest recorded hiking point.",
		matcher:     highPointPRActivities,
	}
	NewTrailBadge = HikingBadge{
		Label:       "New Trail",
		Description: "Explore a hiking trailhead or route name not seen earlier in this badge scope.",
		matcher:     newTrailActivities,
	}
	HikingAdventureBadgeSet = BadgeSet{
		Name: "Hiking adventures",
		Badges: []Badge{
			SummitDayBadge,
			BackToBackHikingWeekendBadge,
			HighPointPRBadge,
			NewTrailBadge,
		},
	}
)

func summitDayActivities(activities []*strava.Activity) []*strava.Activity {
	var matched []*strava.Activity
	for _, activity := range activities {
		if activity == nil {
			continue
		}
		if activity.ElevHigh >= 2000 && activity.TotalElevationGain >= 500 {
			matched = append(matched, activity)
		}
	}
	return matched
}

func backToBackHikingWeekendActivities(activities []*strava.Activity) []*strava.Activity {
	byDate := map[time.Time][]*strava.Activity{}
	for _, activity := range activities {
		date, ok := activityLocalDate(activity)
		if !ok {
			continue
		}
		byDate[date] = append(byDate[date], activity)
	}

	var matched []*strava.Activity
	seenIDs := map[int64]struct{}{}
	for date, dayActivities := range byDate {
		if date.Weekday() != time.Saturday {
			continue
		}
		nextDay := date.AddDate(0, 0, 1)
		if nextDay.Weekday() != time.Sunday {
			continue
		}
		sundayActivities := byDate[nextDay]
		if len(sundayActivities) == 0 {
			continue
		}
		matched = appendUniqueActivities(matched, seenIDs, dayActivities...)
		matched = appendUniqueActivities(matched, seenIDs, sundayActivities...)
	}
	return matched
}

func highPointPRActivities(activities []*strava.Activity) []*strava.Activity {
	var best *strava.Activity
	for _, activity := range activities {
		if activity == nil || !isFinitePositive(activity.ElevHigh) {
			continue
		}
		if best == nil || activity.ElevHigh > best.ElevHigh {
			best = activity
		}
	}
	if best == nil {
		return nil
	}
	return []*strava.Activity{best}
}

func newTrailActivities(activities []*strava.Activity) []*strava.Activity {
	sortedActivities := sortedActivitiesByDate(activities)
	seenTrailKeys := map[string]struct{}{}
	var matched []*strava.Activity
	for _, activity := range sortedActivities {
		key := hikingTrailKey(activity)
		if key == "" {
			continue
		}
		if _, exists := seenTrailKeys[key]; exists {
			continue
		}
		seenTrailKeys[key] = struct{}{}
		matched = append(matched, activity)
	}
	return matched
}

func appendUniqueActivities(
	matched []*strava.Activity,
	seenIDs map[int64]struct{},
	activities ...*strava.Activity,
) []*strava.Activity {
	for _, activity := range activities {
		if activity == nil {
			continue
		}
		if _, exists := seenIDs[activity.Id]; exists {
			continue
		}
		seenIDs[activity.Id] = struct{}{}
		matched = append(matched, activity)
	}
	return matched
}

func sortedActivitiesByDate(activities []*strava.Activity) []*strava.Activity {
	sortedActivities := append([]*strava.Activity{}, activities...)
	sort.SliceStable(sortedActivities, func(left, right int) bool {
		leftDate, leftOK := activityLocalDate(sortedActivities[left])
		rightDate, rightOK := activityLocalDate(sortedActivities[right])
		if leftOK && rightOK {
			return leftDate.Before(rightDate)
		}
		if leftOK != rightOK {
			return leftOK
		}
		return activityDateText(sortedActivities[left]) < activityDateText(sortedActivities[right])
	})
	return sortedActivities
}

func activityLocalDate(activity *strava.Activity) (time.Time, bool) {
	if activity == nil {
		return time.Time{}, false
	}
	dateText := activityDateText(activity)
	if len(dateText) < len("2006-01-02") {
		return time.Time{}, false
	}
	parsedDate, err := time.Parse("2006-01-02", dateText[:10])
	return parsedDate, err == nil
}

func activityDateText(activity *strava.Activity) string {
	if activity == nil {
		return ""
	}
	if strings.TrimSpace(activity.StartDateLocal) != "" {
		return activity.StartDateLocal
	}
	return activity.StartDate
}

func hikingTrailKey(activity *strava.Activity) string {
	if activity == nil {
		return ""
	}
	if len(activity.StartLatlng) >= 2 &&
		isFiniteCoordinate(activity.StartLatlng[0]) &&
		isFiniteCoordinate(activity.StartLatlng[1]) {
		return fmt.Sprintf("geo:%.3f:%.3f", activity.StartLatlng[0], activity.StartLatlng[1])
	}
	return normalizeTrailName(activity.Name)
}

func normalizeTrailName(name string) string {
	lowerName := strings.ToLower(strings.TrimSpace(name))
	if lowerName == "" {
		return ""
	}
	var builder strings.Builder
	for _, char := range lowerName {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			builder.WriteRune(char)
			continue
		}
		builder.WriteRune(' ')
	}
	return strings.Join(strings.Fields(builder.String()), " ")
}

func isFiniteCoordinate(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value != 0
}

func isFinitePositive(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value > 0
}
