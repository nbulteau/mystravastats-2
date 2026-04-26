package infrastructure

import (
	"fmt"
	"math"
	"mystravastats/internal/helpers"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"sort"
	"strings"
)

type gearMetadata struct {
	name    string
	kind    business.GearKind
	retired bool
	primary bool
}

type gearAccumulator struct {
	item                 business.GearAnalysisItem
	monthly              map[string]*business.GearAnalysisPeriodPoint
	longestDistance      float64
	biggestElevationGain float64
	fastestSpeed         float64
}

func buildGearAnalysis(activities []*strava.Activity, athlete strava.Athlete) business.GearAnalysis {
	metadata := buildGearMetadata(athlete)
	itemsByID := make(map[string]*gearAccumulator)
	unassigned := business.GearAnalysisSummary{}
	coverage := business.GearAnalysisCoverage{TotalActivities: len(activities)}

	for _, activity := range activities {
		if activity == nil {
			continue
		}
		gearID := normalizedGearID(activity.GearId)
		if gearID == "" {
			coverage.UnassignedActivities++
			addToSummary(&unassigned, activity)
			continue
		}

		coverage.AssignedActivities++
		accumulator, ok := itemsByID[gearID]
		if !ok {
			meta := metadata[gearID]
			if meta.kind == "" {
				meta.kind = inferGearKind(gearID)
			}
			accumulator = &gearAccumulator{
				item: business.GearAnalysisItem{
					ID:      gearID,
					Name:    gearDisplayName(gearID, meta),
					Kind:    meta.kind,
					Retired: meta.retired,
					Primary: meta.primary,
				},
				monthly: make(map[string]*business.GearAnalysisPeriodPoint),
			}
			itemsByID[gearID] = accumulator
		}

		addActivityToGear(accumulator, activity)
	}

	items := make([]business.GearAnalysisItem, 0, len(itemsByID))
	for _, accumulator := range itemsByID {
		finalizeGearItem(accumulator)
		items = append(items, accumulator.item)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Distance == items[j].Distance {
			return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
		}
		return items[i].Distance > items[j].Distance
	})

	finalizeSummary(&unassigned)

	return business.GearAnalysis{
		Items:      items,
		Unassigned: unassigned,
		Coverage:   coverage,
	}
}

func buildGearMetadata(athlete strava.Athlete) map[string]gearMetadata {
	result := make(map[string]gearMetadata)
	for _, bike := range athlete.Bikes {
		result[bike.Id] = gearMetadata{
			name:    firstNonBlankString(bike.Nickname, bike.Name),
			kind:    business.GearKindBike,
			retired: boolValue(bike.Retired),
			primary: bike.Primary,
		}
	}
	for _, shoe := range athlete.Shoes {
		result[shoe.Id] = gearMetadata{
			name:    firstNonBlankString(shoe.Nickname, shoe.Name),
			kind:    business.GearKindShoe,
			retired: boolValue(shoe.Retired),
			primary: shoe.Primary,
		}
	}
	return result
}

func addActivityToGear(accumulator *gearAccumulator, activity *strava.Activity) {
	accumulator.item.Distance += finiteGearFloat(activity.Distance)
	accumulator.item.MovingTime += activity.MovingTime
	accumulator.item.ElevationGain += finiteGearFloat(activity.TotalElevationGain)
	accumulator.item.Activities++

	date := gearActivityDate(activity)
	if date != "" {
		if accumulator.item.FirstUsed == "" || date < accumulator.item.FirstUsed {
			accumulator.item.FirstUsed = date
		}
		if accumulator.item.LastUsed == "" || date > accumulator.item.LastUsed {
			accumulator.item.LastUsed = date
		}

		periodKey := gearActivityMonth(date)
		if periodKey != "" {
			point := accumulator.monthly[periodKey]
			if point == nil {
				point = &business.GearAnalysisPeriodPoint{PeriodKey: periodKey}
				accumulator.monthly[periodKey] = point
			}
			point.Value += finiteGearFloat(activity.Distance)
			point.ActivityCount++
		}
	}

	if activity.Distance > accumulator.longestDistance || accumulator.item.LongestActivity == nil {
		accumulator.longestDistance = activity.Distance
		accumulator.item.LongestActivity = gearActivityShort(activity)
	}

	if activity.TotalElevationGain > accumulator.biggestElevationGain || accumulator.item.BiggestElevationActivity == nil {
		accumulator.biggestElevationGain = activity.TotalElevationGain
		accumulator.item.BiggestElevationActivity = gearActivityShort(activity)
	}

	speed := gearActivitySpeed(activity)
	if speed > accumulator.fastestSpeed || accumulator.item.FastestActivity == nil {
		accumulator.fastestSpeed = speed
		accumulator.item.FastestActivity = gearActivityShort(activity)
	}
}

func finalizeGearItem(accumulator *gearAccumulator) {
	if accumulator.item.MovingTime > 0 {
		accumulator.item.AverageSpeed = roundGearValue(accumulator.item.Distance / float64(accumulator.item.MovingTime))
	}
	accumulator.item.MaintenanceStatus, accumulator.item.MaintenanceLabel = gearMaintenance(accumulator.item.Kind, accumulator.item.Distance)
	accumulator.item.Distance = roundGearValue(accumulator.item.Distance)
	accumulator.item.ElevationGain = roundGearValue(accumulator.item.ElevationGain)

	monthly := make([]business.GearAnalysisPeriodPoint, 0, len(accumulator.monthly))
	for _, point := range accumulator.monthly {
		point.Value = roundGearValue(point.Value)
		monthly = append(monthly, *point)
	}
	sort.Slice(monthly, func(i, j int) bool {
		return monthly[i].PeriodKey < monthly[j].PeriodKey
	})
	accumulator.item.MonthlyDistance = monthly
}

func addToSummary(summary *business.GearAnalysisSummary, activity *strava.Activity) {
	summary.Distance += finiteGearFloat(activity.Distance)
	summary.MovingTime += activity.MovingTime
	summary.ElevationGain += finiteGearFloat(activity.TotalElevationGain)
	summary.Activities++
}

func finalizeSummary(summary *business.GearAnalysisSummary) {
	if summary.MovingTime > 0 {
		summary.AverageSpeed = roundGearValue(summary.Distance / float64(summary.MovingTime))
	}
	summary.Distance = roundGearValue(summary.Distance)
	summary.ElevationGain = roundGearValue(summary.ElevationGain)
}

func gearActivityShort(activity *strava.Activity) *business.ActivityShort {
	if activity == nil {
		return nil
	}
	activityType := business.Ride
	if activity.Commute {
		activityType = business.Commute
	} else if resolved, ok := business.ActivityTypes[helpers.FirstNonEmpty(activity.SportType, activity.Type)]; ok {
		activityType = resolved
	}
	return &business.ActivityShort{
		Id:   activity.Id,
		Name: activity.Name,
		Type: activityType,
	}
}

func gearActivityDate(activity *strava.Activity) string {
	if activity == nil {
		return ""
	}
	return helpers.FirstNonEmpty(activity.StartDateLocal, activity.StartDate)
}

func gearActivityMonth(date string) string {
	trimmed := strings.TrimSpace(date)
	if len(trimmed) < 7 {
		return ""
	}
	return trimmed[:7]
}

func gearActivitySpeed(activity *strava.Activity) float64 {
	if activity == nil {
		return 0
	}
	if activity.AverageSpeed > 0 {
		return activity.AverageSpeed
	}
	if activity.MovingTime > 0 {
		return activity.Distance / float64(activity.MovingTime)
	}
	return 0
}

func normalizedGearID(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func gearDisplayName(id string, metadata gearMetadata) string {
	name := strings.TrimSpace(metadata.name)
	if name != "" {
		return name
	}
	if strings.HasPrefix(id, "b") {
		return fmt.Sprintf("Bike %s", id)
	}
	if strings.HasPrefix(id, "g") {
		return fmt.Sprintf("Shoes %s", id)
	}
	return fmt.Sprintf("Gear %s", id)
}

func inferGearKind(id string) business.GearKind {
	if strings.HasPrefix(id, "b") {
		return business.GearKindBike
	}
	if strings.HasPrefix(id, "g") {
		return business.GearKindShoe
	}
	return business.GearKindUnknown
}

func gearMaintenance(kind business.GearKind, distanceMeters float64) (string, string) {
	distanceKm := distanceMeters / 1000.0
	switch kind {
	case business.GearKindShoe:
		if distanceKm >= 800 {
			return "REVIEW", "800+ km"
		}
		if distanceKm >= 600 {
			return "WATCH", "600+ km"
		}
	case business.GearKindBike:
		if distanceKm >= 5000 {
			return "REVIEW", "5000+ km"
		}
		if distanceKm >= 3000 {
			return "WATCH", "3000+ km"
		}
	}
	return "OK", "OK"
}

func firstNonBlankString(optional *string, fallback string) string {
	if optional != nil && strings.TrimSpace(*optional) != "" {
		return strings.TrimSpace(*optional)
	}
	return strings.TrimSpace(fallback)
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func finiteGearFloat(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return value
}

func roundGearValue(value float64) float64 {
	return math.Round(finiteGearFloat(value)*10) / 10
}
