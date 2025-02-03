package api

import (
	"fmt"
	"mystravastats/api/dto"
	"mystravastats/domain/business"
	"mystravastats/domain/statistics"
	"mystravastats/domain/strava"
	"strconv"
	"time"
)

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

func toDetailedActivityDto(activity strava.DetailedActivity) dto.DetailedActivityDto {

	activityEfforts := activity.BuildActivityEfforts()

	return dto.DetailedActivityDto{
		AverageCadence:       int(activity.AverageCadence),
		AverageHeartrate:     int(activity.AverageHeartrate),
		AverageWatts:         int(activity.AverageWatts),
		AverageSpeed:         float32(activity.AverageSpeed),
		Calories:             activity.Calories,
		Commute:              activity.Commute,
		DeviceWatts:          activity.DeviceWatts,
		Distance:             float64(activity.Distance),
		ElapsedTime:          activity.ElapsedTime,
		ElevHigh:             activity.ElevHigh,
		ID:                   activity.Id,
		Kilojoules:           activity.Kilojoules,
		MaxHeartrate:         int(activity.MaxHeartrate),
		MaxSpeed:             float32(activity.MaxSpeed),
		MaxWatts:             activity.MaxWatts,
		MovingTime:           activity.MovingTime,
		Name:                 activity.Name,
		ActivityEfforts:      toActivityEffortsDto(activityEfforts),
		StartDate:            parseTime(activity.StartDate),
		StartDateLocal:       parseTime(activity.StartDateLocal),
		StartLatlng:          activity.StartLatLng,
		Stream:               toStreamDto(activity.Stream),
		SufferScore:          activity.SufferScore,
		TotalDescent:         activity.ElevLow,
		TotalElevationGain:   activity.TotalElevationGain,
		Type:                 activity.Type,
		WeightedAverageWatts: activity.WeightedAverageWatts,
	}
}

func toActivityEffortsDto(efforts []business.ActivityEffort) []dto.ActivityEffortDto {
	var effortsDto []dto.ActivityEffortDto
	for _, effort := range efforts {
		effortsDto = append(effortsDto, dto.ActivityEffortDto{
			ID:            strconv.FormatInt(effort.ActivityShort.Id, 10),
			Label:         effort.Label,
			Distance:      effort.Distance,
			Seconds:       effort.Seconds,
			DeltaAltitude: effort.DeltaAltitude,
			IdxStart:      effort.IdxStart,
			IdxEnd:        effort.IdxEnd,
			AveragePower:  effort.AveragePower,
			Description:   effort.GetDescription(),
		})
	}
	return effortsDto
}

func toStreamDto(stream *strava.Stream) *dto.StreamDto {
	if stream == nil {
		return nil
	}

	var Latlng [][]*float64
	if stream.LatLng != nil {
		Latlng = make([][]*float64, len(stream.LatLng.Data))
		if stream.LatLng.Data != nil {
			for i, latlng := range stream.LatLng.Data {
				Latlng[i] = []*float64{&latlng[0], &latlng[1]}
			}
		}
	}

	var moving []*bool
	if stream.Moving != nil {
		moving = make([]*bool, len(stream.Moving.Data))
		if stream.Moving.Data != nil {
			for i, m := range stream.Moving.Data {
				moving[i] = &m
			}
		}
	}

	var altitude []*float64
	if stream.Altitude != nil {
		altitude = make([]*float64, len(stream.Altitude.Data))
		if stream.Altitude.Data != nil {
			for i, a := range stream.Altitude.Data {
				altitude[i] = &a
			}
		}
	}

	var watts []*float64
	if stream.Watts != nil {
		watts = make([]*float64, len(stream.Watts.Data))
		if stream.Watts.Data != nil {
			for i, w := range stream.Watts.Data {
				watts[i] = &w
			}
		}
	}

	var velocitySmooth []*float64
	if stream.VelocitySmooth != nil {
		velocitySmooth = make([]*float64, len(stream.VelocitySmooth.Data))
		if stream.VelocitySmooth.Data != nil {
			for i, v := range stream.VelocitySmooth.Data {
				velocitySmooth[i] = &v
			}
		}
	}

	return &dto.StreamDto{
		Distance:       stream.Distance.Data,
		Time:           stream.Time.Data,
		Latlng:         Latlng,
		Moving:         moving,
		Altitude:       altitude,
		Watts:          watts,
		VelocitySmooth: velocitySmooth,
	}
}

func parseTime(value string) time.Time {
	parsedTime, _ := time.Parse(time.RFC3339, value)
	return parsedTime
}

func toAthleteDto(athlete strava.Athlete) dto.AthleteDto {
	return dto.AthleteDto{
		BadgeTypeId:           getIntValue(athlete.BadgeTypeId),
		City:                  getStringValue(athlete.City),
		Country:               getStringValue(athlete.Country),
		CreatedAt:             parseTime(*athlete.CreatedAt),
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
		UpdatedAt:             parseTime(*athlete.UpdatedAt),
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

func toStatisticDto(statistic statistics.Statistic) dto.StatisticDto {
	if statistic.Activity() != nil {
		return dto.StatisticDto{
			Label: statistic.Label(),
			Value: statistic.Value(),
			Activity: &dto.ActivityShortDto{
				ID:   statistic.Activity().Id,
				Name: statistic.Activity().Name,
				Type: statistic.Activity().Type.String(),
			},
		}
	} else {
		return dto.StatisticDto{
			Label: statistic.Label(),
			Value: statistic.Value(),
		}
	}
}

func toDashboardDataDto(data business.DashboardData) dto.DashboardDataDto {
	return dto.DashboardDataDto{
		NbActivities:           data.NbActivities,
		TotalDistanceByYear:    data.TotalDistanceByYear,
		AverageDistanceByYear:  data.AverageDistanceByYear,
		MaxDistanceByYear:      data.MaxDistanceByYear,
		TotalElevationByYear:   data.TotalElevationByYear,
		AverageElevationByYear: data.AverageElevationByYear,
		MaxElevationByYear:     data.MaxElevationByYear,
		AverageSpeedByYear:     data.AverageSpeedByYear,
		MaxSpeedByYear:         data.MaxSpeedByYear,
		AverageHeartRateByYear: data.AverageHeartRateByYear,
		MaxHeartRateByYear:     data.MaxHeartRateByYear,
		AverageWattsByYear:     data.AverageWattsByYear,
		MaxWattsByYear:         data.MaxWattsByYear,
		AverageCadence:         data.AverageCadence,
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

func getIntValueFromFloat(value *float64) int {
	if value != nil {
		return int(*value)
	}
	return 0
}
