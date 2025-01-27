package api

import (
	"encoding/json"
	"fmt"
	"mystravastats/api/dto"
	"mystravastats/domain"
	"mystravastats/domain/business"
	"mystravastats/domain/statistics"
	"mystravastats/domain/strava"
	"net/http"
	"strconv"
)

func getActivitiesByActivityType(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	year := getYearParam(r)
	activityType := getActivityTypeParam(r)
	activities := domain.FetchActivitiesByActivityTypeAndYear(activityType, year)
	activitiesDto := make([]dto.ActivityDto, len(activities))
	for i, activity := range activities {
		activitiesDto[i] = toDto(activity)
	}

	if err := json.NewEncoder(w).Encode(activitiesDto); err != nil {
		panic(err)
	}
}

func getActivityTypeParam(r *http.Request) business.ActivityType {
	var activityType business.ActivityType
	activityTypeStr := r.URL.Query().Get("activityType")
	if activityTypeStr != "" {
		activityType = business.ActivityTypes[activityTypeStr]
	}
	return activityType
}

func getYearParam(r *http.Request) *int {
	var year *int
	yearStr := r.URL.Query().Get("year")
	if yearStr != "" {
		y, _ := strconv.Atoi(yearStr)
		year = &y
	}
	return year
}

func toDto(activity strava.Activity) dto.ActivityDto {
	bestPowerFor20Minutes := statistics.CalculateBestPowerForTimeForActivity(activity, 20*60)
	bestPowerFor60Minutes := statistics.CalculateBestPowerForTimeForActivity(activity, 60*60)

	var ftp = ""
	if bestPowerFor60Minutes != nil {
		ftp = fmt.Sprintf("%d", bestPowerFor60Minutes.AveragePower)
	} else if bestPowerFor20Minutes != nil {
		ftp = fmt.Sprintf("%d", int(float64(*bestPowerFor20Minutes.AveragePower)*0.95))
	} else {
		ftp = ""
	}

	link := ""
	if activity.UploadId != 0 {
		link = fmt.Sprintf("https://www.strava.com/activities/%d", activity.Id)
	}

	bestPowerFor20MinutesStr := func() string {
		if bestPowerFor20Minutes != nil {
			return bestPowerFor20Minutes.GetFormattedPower()
		}
		return ""
	}()

	bestPowerFor60MinutesStr := func() string {
		if bestPowerFor60Minutes != nil {
			return bestPowerFor60Minutes.GetFormattedPower()
		}
		return ""
	}()

	return dto.ActivityDto{
		ID:                 activity.Id,
		Name:               activity.Name,
		Type:               activity.Type,
		Link:               link,
		Distance:           int(activity.Distance),
		ElapsedTime:        activity.ElapsedTime,
		TotalElevationGain: int(activity.TotalElevationGain),
		AverageSpeed:       activity.AverageSpeed,
		// TODO BestTimeForDistanceFor500m:      42.0, // activity.calculateBestTimeForDistance(500.0).getMSSpeed(),
		BestTimeForDistanceFor1000m:      42.0, // activity.calculateBestTimeForDistance(1000.0).getMSSpeed(),
		BestElevationForDistanceFor500m:  42.0, // activity.calculateBestElevationForDistance(500.0).getGradient(),
		BestElevationForDistanceFor1000m: 42.0, // activity.calculateBestElevationForDistance(1000.0).getGradient(),
		Date:                             activity.StartDateLocal,
		AverageWatts:                     int(activity.AverageWatts),
		WeightedAverageWatts:             strconv.Itoa(activity.WeightedAverageWatts),
		BestPowerFor20Minutes:            bestPowerFor20MinutesStr,
		BestPowerFor60Minutes:            bestPowerFor60MinutesStr,
		FTP:                              ftp,
	}
}
