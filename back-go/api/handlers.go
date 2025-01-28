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
	"time"
)

func getActivitiesByActivityType(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	year := getYearParam(r)
	activityType := getActivityTypeParam(r)

	activities := domain.FetchActivitiesByActivityTypeAndYear(activityType, year)
	activitiesDto := make([]dto.ActivityDto, len(activities))
	for i, activity := range activities {
		activitiesDto[i] = toActivityDto(activity)
	}

	if err := json.NewEncoder(w).Encode(activitiesDto); err != nil {
		panic(err)
	}
}

func getStatisticsByActivityType(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	year := getYearParam(r)
	activityType := getActivityTypeParam(r)

	statisticsForActivityTypeAndYear := domain.GetStatistics(activityType, year)

	if err := json.NewEncoder(w).Encode(statisticsForActivityTypeAndYear); err != nil {
		panic(err)
	}
}

func getAthlete(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	athlete := domain.FetchAthlete()

	// Convert to DTO
	athleteDto := toAthleteDto(athlete)

	if err := json.NewEncoder(w).Encode(athleteDto); err != nil {
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

func toAthleteDto(athlete strava.Athlete) dto.AthleteDto {
	return dto.AthleteDto{
		BadgeTypeId:           getIntValue(athlete.BadgeTypeId),
		City:                  getStringValue(athlete.City),
		Country:               getStringValue(athlete.Country),
		CreatedAt:             parseTime(athlete.CreatedAt),
		Firstname:             getStringValue(athlete.Firstname),
		Id:                    athlete.Id,
		Lastname:              getStringValue(athlete.Lastname),
		Premium:               getBoolValue(athlete.Premium),
		Profile:               getStringValue(athlete.Profile),
		ProfileMedium:         getStringValue(athlete.ProfileMedium),
		ResourceState:         getIntValue(athlete.ResourceState),
		Sex:                   getStringValue(athlete.Sex),
		State:                 getStringValue(athlete.State),
		Summit:                getBoolValue(athlete.Summit),
		UpdatedAt:             parseTime(athlete.UpdatedAt),
		Username:              getStringValue(athlete.Username),
		AthleteType:           getIntValue(athlete.AthleteType),
		DatePreference:        getStringValue(athlete.DatePreference),
		FollowerCount:         getIntValue(athlete.FollowerCount),
		FriendCount:           getIntValue(athlete.FriendCount),
		MeasurementPreference: getStringValue(athlete.MeasurementPreference),
		MutualFriendCount:     getIntValue(athlete.MutualFriendCount),
		Weight:                getIntValueFromFloat(athlete.Weight),
	}
}

func getIntValue(value *int) int {
	if value != nil {
		return *value
	}
	return 0
}

func getStringValue(value *string) string {
	if value != nil {
		return *value
	}
	return ""
}

func getBoolValue(value *bool) bool {
	if value != nil {
		return *value
	}
	return false
}

func parseTime(value *string) time.Time {
	if value != nil {
		parsedTime, _ := time.Parse(time.RFC3339, *value)
		return parsedTime
	}
	return time.Time{}
}

func getIntValueFromFloat(value *float64) int {
	if value != nil {
		return int(*value)
	}
	return 0
}

func toActivityDto(activity strava.Activity) dto.ActivityDto {
	bestPowerFor20Minutes := statistics.BestPowerForTime(activity, 20*60)
	bestPowerFor60Minutes := statistics.BestPowerForTime(activity, 60*60)

	var ftp string
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

	bestPowerFor20MinutesStr := ""
	if bestPowerFor20Minutes != nil {
		bestPowerFor20MinutesStr = bestPowerFor20Minutes.GetFormattedPower()
	}

	bestPowerFor60MinutesStr := ""
	if bestPowerFor60Minutes != nil {
		bestPowerFor60MinutesStr = bestPowerFor60Minutes.GetFormattedPower()
	}

	bestTimeForDistanceFor1000m := 0.0
	if bestTimeForDistance := statistics.BestActivityEffort(activity, 1000.0); bestTimeForDistance != nil {
		bestTimeForDistanceFor1000m = bestTimeForDistance.GetMSSpeed()
	}

	bestElevationForDistanceFor500m := 0.0
	if bestElevationForDistance := statistics.BestElevationEffort(activity, 500.0); bestElevationForDistance != nil {
		bestElevationForDistanceFor500m = bestElevationForDistance.GetGradient()
	}

	bestElevationForDistanceFor1000m := 0.0
	if bestElevationForDistance := statistics.BestElevationEffort(activity, 1000.0); bestElevationForDistance != nil {
		bestElevationForDistanceFor1000m = bestElevationForDistance.GetGradient()
	}

	return dto.ActivityDto{
		Id:                               activity.Id,
		Name:                             activity.Name,
		Type:                             activity.Type,
		Link:                             link,
		Distance:                         int(activity.Distance),
		ElapsedTime:                      activity.ElapsedTime,
		TotalElevationGain:               int(activity.TotalElevationGain),
		AverageSpeed:                     activity.AverageSpeed,
		BestTimeForDistanceFor1000m:      bestTimeForDistanceFor1000m,
		BestElevationForDistanceFor500m:  bestElevationForDistanceFor500m,
		BestElevationForDistanceFor1000m: bestElevationForDistanceFor1000m,
		Date:                             activity.StartDateLocal,
		AverageWatts:                     int(activity.AverageWatts),
		WeightedAverageWatts:             strconv.Itoa(activity.WeightedAverageWatts),
		BestPowerFor20Minutes:            bestPowerFor20MinutesStr,
		BestPowerFor60Minutes:            bestPowerFor60MinutesStr,
		FTP:                              ftp,
	}
}
