package services

import (
	"log"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

func ExportCSV(year *int, activityTypes ...business.ActivityType) (string, error) {
	log.Printf("Get export CSV by activity (%s) type by year (%d)", activityTypes, year)

	csvData := ""
	sportType := business.Ride
	// export header
	switch sportType {
	case business.Ride:
		csvData += "id,athlete_id,name,type,start_date,elapsed_time,distance,average_speed,average_cadence,average_heartrate,max_heartrate,total_elevation_gain,average_watts,device_watts,commute,upload_id\n"
	default:
		csvData += "id,athlete_id,name,type,start_date,elapsed_time,distance,average_speed,average_cadence,average_heartrate,max_heartrate,total_elevation_gain\n"
	}

	// export activities
	activities := activityProvider.GetActivitiesByYearAndActivityTypes(year, activityTypes...)
	for _, activity := range activities {
		switch sportType {
		case business.Ride:
			csvData += exportRideActivity(activity)
		default:
			csvData += "Activity Data" // Placeholder for other activity types
		}
	}

	// export footer
	switch sportType {
	case business.Ride:
		csvData += "Total,," + business.Ride.String() + ",,,,"
	default:
		csvData += "Total,," + sportType.String() + ",,,,"
	}

	return csvData, nil
}

func exportRideActivity(activity *strava.Activity) string {
	return "Ride Activity Data" // Placeholder for actual CSV export logic
}
