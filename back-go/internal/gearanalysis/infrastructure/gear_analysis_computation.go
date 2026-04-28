package infrastructure

import (
	"fmt"
	"math"
	"mystravastats/internal/helpers"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"sort"
	"strings"
	"time"
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

type gearMaintenanceRule struct {
	component        string
	label            string
	intervalDistance float64
	intervalMonths   int
}

var bikeMaintenanceRules = []gearMaintenanceRule{
	{component: "CHAIN", label: "Chain", intervalDistance: 1500 * 1000},
	{component: "CASSETTE", label: "Cassette", intervalDistance: 5000 * 1000},
	{component: "BRAKE_PADS_FRONT", label: "Front brake pads", intervalDistance: 1800 * 1000},
	{component: "BRAKE_PADS_REAR", label: "Rear brake pads", intervalDistance: 1800 * 1000},
	{component: "BRAKE_BLEED", label: "Brake bleed", intervalMonths: 12},
	{component: "TIRE_FRONT", label: "Front tire", intervalDistance: 3500 * 1000},
	{component: "TIRE_REAR", label: "Rear tire", intervalDistance: 3500 * 1000},
	{component: "TUBELESS_FRONT", label: "Front tubeless sealant", intervalMonths: 4},
	{component: "TUBELESS_REAR", label: "Rear tubeless sealant", intervalMonths: 4},
	{component: "BOTTOM_BRACKET", label: "Bottom bracket", intervalDistance: 8000 * 1000},
	{component: "BEARINGS", label: "Bearings", intervalDistance: 6000 * 1000},
	{component: "DRIVETRAIN", label: "Drivetrain", intervalDistance: 5000 * 1000},
}

func buildGearAnalysis(activities []*strava.Activity, lifetimeActivities []*strava.Activity, athlete strava.Athlete, maintenanceRecords []business.GearMaintenanceRecord) business.GearAnalysis {
	metadata := buildGearMetadata(athlete)
	totalDistanceByGearID := totalGearDistances(lifetimeActivities)
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
		finalizeGearItem(accumulator, totalDistanceByGearID)
		items = append(items, accumulator.item)
	}
	applyGearMaintenance(items, maintenanceRecords)

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

func totalGearDistances(activities []*strava.Activity) map[string]float64 {
	result := make(map[string]float64)
	for _, activity := range activities {
		if activity == nil {
			continue
		}
		gearID := normalizedGearID(activity.GearId)
		if gearID == "" {
			continue
		}
		result[gearID] += finiteGearFloat(activity.Distance)
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

func finalizeGearItem(accumulator *gearAccumulator, totalDistanceByGearID map[string]float64) {
	totalDistance := totalDistanceByGearID[accumulator.item.ID]
	if totalDistance <= 0 {
		totalDistance = accumulator.item.Distance
	}
	accumulator.item.TotalDistance = roundGearValue(totalDistance)
	if accumulator.item.MovingTime > 0 {
		accumulator.item.AverageSpeed = roundGearValue(accumulator.item.Distance / float64(accumulator.item.MovingTime))
	}
	accumulator.item.MaintenanceStatus, accumulator.item.MaintenanceLabel = gearMaintenance(accumulator.item.Kind, accumulator.item.TotalDistance)
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

func applyGearMaintenance(items []business.GearAnalysisItem, records []business.GearMaintenanceRecord) {
	recordsByGear := gearMaintenanceRecordsByGear(records)
	now := time.Now().UTC()
	for index := range items {
		history := recordsByGear[items[index].ID]
		items[index].MaintenanceHistory = history
		if items[index].Kind != business.GearKindBike {
			continue
		}
		items[index].MaintenanceTasks = buildGearMaintenanceTasks(items[index], history, now)
		items[index].MaintenanceStatus, items[index].MaintenanceLabel = summarizeGearMaintenance(items[index].MaintenanceTasks)
	}
}

func buildGearMaintenanceTasks(item business.GearAnalysisItem, records []business.GearMaintenanceRecord, now time.Time) []business.GearMaintenanceTask {
	recordsByComponent := make(map[string][]business.GearMaintenanceRecord)
	for _, record := range records {
		recordsByComponent[record.Component] = append(recordsByComponent[record.Component], record)
	}

	tasks := make([]business.GearMaintenanceTask, 0, len(bikeMaintenanceRules))
	for _, rule := range bikeMaintenanceRules {
		componentRecords := maintenanceRecordsForRule(recordsByComponent, rule.component)
		var last *business.GearMaintenanceRecord
		if len(componentRecords) > 0 {
			last = &componentRecords[0]
		}

		task := business.GearMaintenanceTask{
			Component:        rule.component,
			ComponentLabel:   rule.label,
			IntervalDistance: rule.intervalDistance,
			IntervalMonths:   rule.intervalMonths,
			Status:           "OK",
			StatusLabel:      "OK",
		}
		if last == nil {
			task.Status = "DUE"
			task.StatusLabel = "No service logged"
			tasks = append(tasks, task)
			continue
		}

		task.LastMaintenance = last
		if rule.intervalDistance > 0 {
			odometerDistance := gearMaintenanceOdometer(item)
			task.DistanceSince = roundGearValue(math.Max(0, odometerDistance-last.Distance))
			task.NextDueDistance = roundGearValue(last.Distance + rule.intervalDistance)
			task.DistanceRemaining = roundGearValue(math.Max(0, task.NextDueDistance-odometerDistance))
			ratio := task.DistanceSince / rule.intervalDistance
			if ratio >= 1 {
				task.Status = "OVERDUE"
				task.StatusLabel = fmt.Sprintf("%.0f km overdue", math.Ceil((task.DistanceSince-rule.intervalDistance)/1000))
			} else if ratio >= 0.85 {
				task.Status = "SOON"
				task.StatusLabel = fmt.Sprintf("%.0f km left", math.Ceil(task.DistanceRemaining/1000))
			}
		}

		if rule.intervalMonths > 0 {
			monthsSince, ok := monthsSinceMaintenance(last.Date, now)
			if ok {
				task.MonthsSince = monthsSince
				task.MonthsRemaining = maxInt(0, rule.intervalMonths-monthsSince)
				timeStatus := "OK"
				timeLabel := "OK"
				if monthsSince >= rule.intervalMonths {
					timeStatus = "OVERDUE"
					timeLabel = fmt.Sprintf("%d months overdue", monthsSince-rule.intervalMonths)
				} else if float64(monthsSince)/float64(rule.intervalMonths) >= 0.80 {
					timeStatus = "SOON"
					timeLabel = fmt.Sprintf("%d months left", task.MonthsRemaining)
				}
				if maintenanceStatusRank(timeStatus) > maintenanceStatusRank(task.Status) {
					task.Status = timeStatus
					task.StatusLabel = timeLabel
				}
			}
		}

		tasks = append(tasks, task)
	}
	return tasks
}

func maintenanceRecordsForRule(recordsByComponent map[string][]business.GearMaintenanceRecord, component string) []business.GearMaintenanceRecord {
	records := recordsByComponent[component]
	if len(records) == 0 && (component == "TIRE_FRONT" || component == "TIRE_REAR") {
		return recordsByComponent["TIRES"]
	}
	return records
}

func summarizeGearMaintenance(tasks []business.GearMaintenanceTask) (string, string) {
	if len(tasks) == 0 {
		return "OK", "OK"
	}
	counts := map[string]int{}
	worst := "OK"
	for _, task := range tasks {
		counts[task.Status]++
		if maintenanceStatusRank(task.Status) > maintenanceStatusRank(worst) {
			worst = task.Status
		}
	}
	switch worst {
	case "OVERDUE":
		return worst, fmt.Sprintf("%d overdue", counts[worst])
	case "DUE":
		return worst, fmt.Sprintf("%d due", counts[worst])
	case "SOON":
		return worst, fmt.Sprintf("%d soon", counts[worst])
	default:
		return "OK", "OK"
	}
}

func gearMaintenanceRecordsByGear(records []business.GearMaintenanceRecord) map[string][]business.GearMaintenanceRecord {
	result := make(map[string][]business.GearMaintenanceRecord)
	for _, record := range records {
		gearID := strings.TrimSpace(record.GearID)
		if gearID == "" {
			continue
		}
		result[gearID] = append(result[gearID], record)
	}
	for gearID := range result {
		sort.Slice(result[gearID], func(i, j int) bool {
			left := result[gearID][i]
			right := result[gearID][j]
			if left.Date == right.Date {
				return left.CreatedAt > right.CreatedAt
			}
			return left.Date > right.Date
		})
	}
	return result
}

func monthsSinceMaintenance(value string, now time.Time) (int, bool) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) >= 10 {
		trimmed = trimmed[:10]
	}
	date, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return 0, false
	}
	months := (now.Year()-date.Year())*12 + int(now.Month()-date.Month())
	if now.Day() < date.Day() {
		months--
	}
	return maxInt(0, months), true
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
			return "OVERDUE", "800+ km"
		}
		if distanceKm >= 600 {
			return "SOON", "600+ km"
		}
	case business.GearKindBike:
		if distanceKm >= 5000 {
			return "OVERDUE", "5000+ km"
		}
		if distanceKm >= 3000 {
			return "SOON", "3000+ km"
		}
	}
	return "OK", "OK"
}

func gearMaintenanceOdometer(item business.GearAnalysisItem) float64 {
	if item.TotalDistance > 0 {
		return item.TotalDistance
	}
	return item.Distance
}

func maintenanceStatusRank(status string) int {
	switch status {
	case "OVERDUE":
		return 3
	case "DUE":
		return 2
	case "SOON":
		return 1
	default:
		return 0
	}
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
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
