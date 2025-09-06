package services

import (
	"fmt"
	"log"
	"mystravastats/domain/business"
	"mystravastats/domain/statistics"
	"mystravastats/domain/strava"
	"strings"
	"time"
)

func ExportCSV(year *int, activityTypes ...business.ActivityType) string {
	log.Printf("Get export CSV by activity (%s) type by year (%d)", activityTypes, *year)

	activities := activityProvider.GetActivitiesByYearAndActivityTypes(year, activityTypes...)

	switch activityTypes[0] {
	case business.Ride, business.VirtualRide, business.MountainBikeRide, business.GravelRide, business.Commute:
		return rideExport(activities)
	case business.Run, business.TrailRun:
		return runExport(activities)
	case business.Hike:
		return hikeExport(activities)
	case business.AlpineSki:
		return alpineSkiExport(activities)
	case business.InlineSkate:
		return inlineSkateExport(activities)
	default:
		log.Printf("Unsupported activity type: %s", activityTypes[0])
		return ""
	}
}

func rideExport(activities []*strava.Activity) string {
	var csvData strings.Builder

	// Generate header
	csvData.WriteString(generateRideHeader())
	csvData.WriteString("\n")

	// Generate activities
	for _, activity := range activities {
		csvData.WriteString(generateRideActivity(activity))
		csvData.WriteString("\n")
	}

	// Generate footer
	csvData.WriteString(generateRideFooter(len(activities)))

	return csvData.String()
}

func runExport(activities []*strava.Activity) string {
	var csvData strings.Builder

	// Generate header
	csvData.WriteString(generateRunHeader())
	csvData.WriteString("\n")

	// Generate activities
	for _, activity := range activities {
		csvData.WriteString(generateRunActivity(activity))
		csvData.WriteString("\n")
	}

	// Generate footer
	csvData.WriteString(generateRunFooter(len(activities)))

	return csvData.String()
}

func hikeExport(activities []*strava.Activity) string {
	var csvData strings.Builder

	// Generate header
	csvData.WriteString(generateHikeHeader())
	csvData.WriteString("\n")

	// Generate activities
	for _, activity := range activities {
		csvData.WriteString(generateHikeActivity(activity))
		csvData.WriteString("\n")
	}

	// Generate footer
	csvData.WriteString(generateHikeFooter(len(activities)))

	return csvData.String()
}

func alpineSkiExport(activities []*strava.Activity) string {
	var csvData strings.Builder

	// Generate header
	csvData.WriteString(generateAlpineSkiHeader())
	csvData.WriteString("\n")

	// Generate activities
	for _, activity := range activities {
		csvData.WriteString(generateAlpineSkiActivity(activity))
		csvData.WriteString("\n")
	}

	// Generate footer
	csvData.WriteString(generateAlpineSkiFooter(len(activities)))

	return csvData.String()
}

func inlineSkateExport(activities []*strava.Activity) string {
	var csvData strings.Builder

	// Generate header
	csvData.WriteString(generateInlineSkateHeader())
	csvData.WriteString("\n")

	// Generate activities
	for _, activity := range activities {
		csvData.WriteString(generateInlineSkateActivity(activity))
		csvData.WriteString("\n")
	}

	// Generate footer
	csvData.WriteString(generateInlineSkateFooter(len(activities)))

	return csvData.String()
}

func generateRideHeader() string {
	headers := []string{
		"Date",
		"Description",
		"DistanceStream (km)",
		"TimeStream",
		"TimeStream (seconds)",
		"Average speed (km/h)",
		"Best 250m (km/h)",
		"Best 500m (km/h)",
		"Best 1000m (km/h)",
		"Best 5km (km/h)",
		"Best 10km (km/h)",
		"Best 20km (km/h)",
		"Best 50km (km/h)",
		"Best 100km (km/h)",
		"Best 30 min (km/h)",
		"Best 1 h (km/h)",
		"Best 2 h (km/h)",
		"Best 3 h (km/h)",
		"Best 4 h (km/h)",
		"Best 5 h (km/h)",
		"Max gradient for 250 m (%)",
		"Max gradient for 500 m (%)",
		"Max gradient for 1000 m (%)",
		"Max gradient for 5 km (%)",
		"Max gradient for 10 km (%)",
		"Max gradient for 20 km (%)",
	}
	return writeCSVLine(headers)
}

func generateRideActivity(activity *strava.Activity) string {
	data := []string{
		formatDate(activity.StartDateLocal),
		strings.TrimSpace(activity.Name),
		fmt.Sprintf("%.02f", activity.Distance/1000),
		formatSeconds(activity.ElapsedTime),
		fmt.Sprintf("%d", activity.ElapsedTime),
		processAverageSpeed(activity),
		calculateBestTimeForDistance(activity, 250.0),
		calculateBestTimeForDistance(activity, 500.0),
		calculateBestTimeForDistance(activity, 1000.0),
		calculateBestTimeForDistance(activity, 5000.0),
		calculateBestTimeForDistance(activity, 10000.0),
		calculateBestTimeForDistance(activity, 20000.0),
		calculateBestTimeForDistance(activity, 50000.0),
		calculateBestTimeForDistance(activity, 100000.0),
		calculateBestDistanceForTime(activity, 30*60),
		calculateBestDistanceForTime(activity, 60*60),
		calculateBestDistanceForTime(activity, 2*60*60),
		calculateBestDistanceForTime(activity, 3*60*60),
		calculateBestDistanceForTime(activity, 4*60*60),
		calculateBestDistanceForTime(activity, 5*60*60),
		calculateBestElevationForDistance(activity, 250.0),
		calculateBestElevationForDistance(activity, 500.0),
		calculateBestElevationForDistance(activity, 1000.0),
		calculateBestElevationForDistance(activity, 5000.0),
		calculateBestElevationForDistance(activity, 10000.0),
		calculateBestElevationForDistance(activity, 20000.0),
	}
	return writeCSVLine(data)
}

func generateRideFooter(activitiesCount int) string {
	lastRow := activitiesCount + 1
	formula := fmt.Sprintf(";;=SOMME($C2:$C%d);;=SOMME($E2:$E%d);=MAX($F2:$F%d);=MAX($G2:$G%d);=MAX($H2:$H%d);=MAX($I2:$I%d);=MAX($J2:$J%d);=MAX($K2:$K%d);=MAX($L2:$L%d);=MAX($M2:$M%d);=MAX($N2:$N%d);=MAX($O2:$O%d);=MAX($P2:$P%d);=MAX($Q2:$Q%d);=MAX($R2:$R%d);=MAX($S2:$S%d);=MAX($T2:$T%d);=MAX($U2:$U%d);=MAX($V2:$V%d);=MAX($W2:$W%d);=MAX($X2:$X%d);=MAX($Y2:$Y%d);=MAX($Z2:$Z%d)",
		lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow)
	return writeCSVLine([]string{formula})
}

func generateRunHeader() string {
	headers := []string{
		"Date",
		"Description",
		"DistanceStream (km)",
		"TimeStream",
		"TimeStream (seconds)",
		"Average speed (min/km)",
		"Best 200m (min/km)",
		"Best 400m (min/km)",
		"Best 1000m (min/km)",
		"Best 5000m (min/km)",
		"Best 10000m (min/km)",
		"Best half Marathon (min/km)",
		"Best Marathon (min/km)",
		"Best 30 min (min/km)",
		"Best 1 h (min/km)",
		"Best 2 h (min/km)",
		"Best 3 h (min/km)",
		"Best 4 h (min/km)",
		"Best 5 h (min/km)",
		"Best vVO2max = 6 min (min/km)",
	}
	return writeCSVLine(headers)
}

func generateRunActivity(activity *strava.Activity) string {
	data := []string{
		formatDate(activity.StartDateLocal),
		strings.TrimSpace(activity.Name),
		fmt.Sprintf("%.02f", activity.Distance/1000),
		formatSeconds(activity.ElapsedTime),
		fmt.Sprintf("%d", activity.ElapsedTime),
		processAverageSpeed(activity),
		calculateBestTimeForDistance(activity, 200.0),
		calculateBestTimeForDistance(activity, 400.0),
		calculateBestTimeForDistance(activity, 1000.0),
		calculateBestTimeForDistance(activity, 5000.0),
		calculateBestTimeForDistance(activity, 10000.0),
		calculateBestTimeForDistance(activity, 21097.0),
		calculateBestTimeForDistance(activity, 42195.0),
		calculateBestDistanceForTime(activity, 30*60),
		calculateBestDistanceForTime(activity, 60*60),
		calculateBestDistanceForTime(activity, 2*60*60),
		calculateBestDistanceForTime(activity, 3*60*60),
		calculateBestDistanceForTime(activity, 4*60*60),
		calculateBestDistanceForTime(activity, 5*60*60),
		calculateBestDistanceForTime(activity, 12*60),
	}
	return writeCSVLine(data)
}

func generateRunFooter(activitiesCount int) string {
	lastRow := activitiesCount + 1
	formula := fmt.Sprintf(";;=SOMME($C2:$C%d);;=SOMME($E2:$E%d);", lastRow, lastRow)
	return writeCSVLine([]string{formula})
}

func generateHikeHeader() string {
	headers := []string{
		"Date",
		"Description",
		"DistanceStream (km)",
		"TimeStream",
		"TimeStream (seconds)",
		"Average speed (km/h)",
		"Elevation (m)",
		"Highest point (m)",
		"Best 1000m (km/h)",
		"Best 1 h (km/h)",
		"Max gradient for 250 m (%)",
		"Max gradient for 500 m (%)",
		"Max gradient for 1000 m (%)",
		"Max gradient for 5 km (%)",
		"Max gradient for 10 km (%)",
	}
	return writeCSVLine(headers)
}

func generateHikeActivity(activity *strava.Activity) string {
	data := []string{
		formatDate(activity.StartDateLocal),
		strings.TrimSpace(activity.Name),
		fmt.Sprintf("%.02f", activity.Distance/1000),
		formatSeconds(activity.ElapsedTime),
		fmt.Sprintf("%d", activity.ElapsedTime),
		processAverageSpeed(activity),
		fmt.Sprintf("%.0f", activity.TotalElevationGain),
		fmt.Sprintf("%.0f", activity.ElevHigh),
		calculateBestTimeForDistance(activity, 1000.0),
		calculateBestDistanceForTime(activity, 60*60),
		calculateBestElevationForDistance(activity, 250.0),
		calculateBestElevationForDistance(activity, 500.0),
		calculateBestElevationForDistance(activity, 1000.0),
		calculateBestElevationForDistance(activity, 5000.0),
		calculateBestElevationForDistance(activity, 10000.0),
	}
	return writeCSVLine(data)
}

func generateHikeFooter(activitiesCount int) string {
	lastRow := activitiesCount + 1
	formula := fmt.Sprintf(";;=SOMME($C2:$C%d);;=SOMME($E2:$E%d);=MAX($F2:$F%d);=MAX($G2:$G%d);=MAX($H2:$H%d);=MAX($I2:$I%d);=MAX($J2:$J%d);=MAX($K2:$K%d);=MAX($L2:$L%d);=MAX($M2:$M%d)",
		lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow)
	return writeCSVLine([]string{formula})
}

func generateInlineSkateHeader() string {
	headers := []string{
		"Date",
		"Description",
		"DistanceStream (km)",
		"TimeStream",
		"TimeStream (seconds)",
		"Average speed (km/h)",
		"Best 200m (km/h)",
		"Best 400m (km/h)",
		"Best 1000m (km/h)",
		"Best 10000m (km/h)",
		"Best half Marathon (km/h)",
		"Best Marathon (km/h)",
		"Best 30 min (km/h)",
		"Best 1 h (km/h)",
		"Best 2 h (km/h)",
		"Best 3 h (km/h)",
		"Best 4 h (km/h)",
		"Best 5 h (km/h)",
	}
	return writeCSVLine(headers)
}

func generateInlineSkateActivity(activity *strava.Activity) string {
	data := []string{
		formatDate(activity.StartDateLocal),
		strings.TrimSpace(activity.Name),
		fmt.Sprintf("%.02f", activity.Distance/1000),
		formatSeconds(activity.ElapsedTime),
		fmt.Sprintf("%d", activity.ElapsedTime),
		processAverageSpeed(activity),
		calculateBestTimeForDistance(activity, 200.0),
		calculateBestTimeForDistance(activity, 400.0),
		calculateBestTimeForDistance(activity, 1000.0),
		calculateBestTimeForDistance(activity, 10000.0),
		calculateBestTimeForDistance(activity, 21097.0),
		calculateBestTimeForDistance(activity, 42195.0),
		calculateBestDistanceForTime(activity, 30*60),
		calculateBestDistanceForTime(activity, 60*60),
		calculateBestDistanceForTime(activity, 2*60*60),
		calculateBestDistanceForTime(activity, 3*60*60),
		calculateBestDistanceForTime(activity, 4*60*60),
		calculateBestDistanceForTime(activity, 5*60*60),
	}
	return writeCSVLine(data)
}

func generateInlineSkateFooter(activitiesCount int) string {
	lastRow := activitiesCount + 1
	formula := fmt.Sprintf(";;=SOMME($C2:$C%d);;=SOMME($E2:$E%d);=MAX($F2:$F%d);=MAX($G2:$G%d);=MAX($H2:$H%d);=MAX($I2:$I%d);=MAX($J2:$J%d);=MAX($K2:$K%d);=MAX($L2:$L%d);=MAX($M2:$M%d);=MAX($N2:$N%d);=MAX($O2:$O%d);=MAX($P2:$P%d);=MAX($Q2:$Q%d);=MAX($R2:$R%d)",
		lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow)
	return writeCSVLine([]string{formula})
}

func generateAlpineSkiHeader() string {
	headers := []string{
		"Date",
		"Description",
		"DistanceStream (km)",
		"TimeStream",
		"TimeStream (seconds)",
		"Average speed (km/h)",
		"Best 250m (km/h)",
		"Best 500m (km/h)",
		"Best 1000m (km/h)",
		"Best 5km (km/h)",
		"Best 10km (km/h)",
		"Best 20km (km/h)",
		"Best 50km (km/h)",
		"Best 100km (km/h)",
		"Best 30 min (km/h)",
		"Best 1 h (km/h)",
		"Best 2 h (km/h)",
		"Best 3 h (km/h)",
		"Best 4 h (km/h)",
		"Best 5 h (km/h)",
		"Max gradient for 250 m (%)",
		"Max gradient for 500 m (%)",
		"Max gradient for 1000 m (%)",
		"Max gradient for 5 km (%)",
		"Max gradient for 10 km (%)",
		"Max gradient for 20 km (%)",
	}
	return writeCSVLine(headers)
}

func generateAlpineSkiActivity(activity *strava.Activity) string {
	data := []string{
		formatDate(activity.StartDateLocal),
		strings.TrimSpace(activity.Name),
		fmt.Sprintf("%.02f", activity.Distance/1000),
		formatSeconds(activity.ElapsedTime),
		fmt.Sprintf("%d", activity.ElapsedTime),
		processAverageSpeed(activity),
		calculateBestTimeForDistance(activity, 250.0),
		calculateBestTimeForDistance(activity, 500.0),
		calculateBestTimeForDistance(activity, 1000.0),
		calculateBestTimeForDistance(activity, 5000.0),
		calculateBestTimeForDistance(activity, 10000.0),
		calculateBestTimeForDistance(activity, 20000.0),
		calculateBestTimeForDistance(activity, 50000.0),
		calculateBestTimeForDistance(activity, 100000.0),
		calculateBestDistanceForTime(activity, 30*60),
		calculateBestDistanceForTime(activity, 60*60),
		calculateBestDistanceForTime(activity, 2*60*60),
		calculateBestDistanceForTime(activity, 3*60*60),
		calculateBestDistanceForTime(activity, 4*60*60),
		calculateBestDistanceForTime(activity, 5*60*60),
		calculateBestElevationForDistance(activity, 250.0),
		calculateBestElevationForDistance(activity, 500.0),
		calculateBestElevationForDistance(activity, 1000.0),
		calculateBestElevationForDistance(activity, 5000.0),
		calculateBestElevationForDistance(activity, 10000.0),
		calculateBestElevationForDistance(activity, 20000.0),
	}
	return writeCSVLine(data)
}

func generateAlpineSkiFooter(activitiesCount int) string {
	lastRow := activitiesCount + 1
	formula := fmt.Sprintf(";;=SOMME($C2:$C%d);;=SOMME($E2:$E%d);=MAX($F2:$F%d);=MAX($G2:$G%d);=MAX($H2:$H%d);=MAX($I2:$I%d);=MAX($J2:$J%d);=MAX($K2:$K%d);=MAX($L2:$L%d);=MAX($M2:$M%d);=MAX($N2:$N%d);=MAX($O2:$O%d);=MAX($P2:$P%d);=MAX($Q2:$Q%d);=MAX($R2:$R%d);=MAX($S2:$S%d);=MAX($T2:$T%d);=MAX($U2:$U%d);=MAX($V2:$V%d);=MAX($W2:$W%d);=MAX($X2:$X%d);=MAX($Y2:$Y%d);=MAX($Z2:$Z%d)",
		lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow, lastRow)
	return writeCSVLine([]string{formula})
}

func writeCSVLine(fields []string) string {
	return strings.Join(fields, ",")
}

// Helper functions - these would need to be implemented based on your activity model
func formatSeconds(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

func processAverageSpeed(activity *strava.Activity) string {
	if activity.ElapsedTime > 0 {
		speed := (activity.Distance / 1000) / (float64(activity.ElapsedTime) / 3600)
		return fmt.Sprintf("%.02f", speed)
	}
	return ""
}

func formatDate(dateStr string) string {
	// Input format: "2006-01-02T15:04:05Z"
	inLayout := "2006-01-02T15:04:05Z"

	// Output format: "Mon 02 January 2006-15:04"
	outLayout := "Mon 02 January 2006 - 15:04"

	parsedTime, err := time.Parse(inLayout, dateStr)
	if err != nil {
		return ""
	}

	return parsedTime.Format(outLayout)
}

func calculateBestElevationForDistance(activity *strava.Activity, f float64) string {
	if activity.Stream == nil {
		return "Not available"
	}
	bestElevationForDistance := statistics.BestElevationForDistance(activity.Id, activity.Name, activity.Type, activity.Stream, f)
	if bestElevationForDistance == nil {
		return "Not available"
	}
	return bestElevationForDistance.GetFormattedGradient()
}

func calculateBestDistanceForTime(activity *strava.Activity, i int) string {
	if activity.Stream == nil {
		return "Not available"
	}
	bestDistanceForTime := statistics.BestDistanceForTime(activity.Id, activity.Name, activity.Type, activity.Stream, i)
	if bestDistanceForTime == nil {
		return "Not available"
	}
	return fmt.Sprintf("%.1f km", bestDistanceForTime.Distance/1000)
}

func calculateBestTimeForDistance(activity *strava.Activity, f float64) string {

	if activity.Stream == nil {
		return "Not available"
	}
	bestTimeForDistance := statistics.BestTimeForDistance(activity.Id, activity.Name, activity.Type, activity.Stream, f)
	if bestTimeForDistance == nil {
		return "Not available"
	}
	return bestTimeForDistance.GetFormattedSpeed()
}
