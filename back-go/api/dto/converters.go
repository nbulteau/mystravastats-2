package dto

import (
	"fmt"
	"math"
	"mystravastats/domain/badges"
	"mystravastats/domain/business"
	"mystravastats/domain/statistics"
	"mystravastats/domain/strava"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func FormatSpeed(speed float64, activityType string) string {
	if strings.EqualFold(activityType, "Run") || strings.EqualFold(activityType, "TrailRun") {
		if speed <= 0 {
			return "-/km"
		}
		secondsPerKm := 1000.0 / speed
		return fmt.Sprintf("%s/km", formatSeconds(secondsPerKm))
	}
	return fmt.Sprintf("%.02f km/h", speed*3.6)
}

func FormatGradient(gradient float64) string {
	return fmt.Sprintf("%.02f", gradient)
}

func formatSeconds(totalSeconds float64) string {
	secs := int(math.Round(totalSeconds))
	if secs < 0 {
		secs = 0
	}
	h := secs / 3600
	m := (secs % 3600) / 60
	s := secs % 60

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func ToActivityDto(activity strava.Activity) ActivityDto {
	bestPowerFor20Minutes := statistics.BestPowerForTime(activity, 20*60)
	bestPowerFor60Minutes := statistics.BestPowerForTime(activity, 60*60)

	var ftp = 0
	// FTP is defined as the highest power you can maintain in a quasi-steady state for approximately one hour without fatiguing.
	// If a 60-minute effort is not available, it is common practice to estimate FTP as 95% of a 20-minute effort.
	// https://www.trainingpeaks.com/blog/functional-threshold-power-ftp-what-is-it-and-how-to-test-it/
	// https://www.peakendurancesport.com/endurance-training/training-plans/functional-threshold-power-ftp/
	// https://www.cyclingnews.com/features/ask-cyclingnews-com-what-is-functional-threshold-power-ftp/
	// https://www.trainerroad.com/blog/what-is-functional-threshold-power/
	// https://www.strava.com/support/athlete/activities/what-is-functional-threshold-power-ftp
	if bestPowerFor60Minutes != nil {
		ftp = int(*bestPowerFor60Minutes.AveragePower)
	} else if bestPowerFor20Minutes != nil {
		ftp = int(float64(*bestPowerFor20Minutes.AveragePower) * 0.95)
	}

	link := ""
	if activity.UploadId != 0 {
		link = fmt.Sprintf("https://www.strava.com/activities/%d", activity.Id)
	}

	bestTimeForDistanceFor1000m := 0.0
	if bestTimeForDistance := statistics.BestTimeEffort(activity, 1000.0); bestTimeForDistance != nil {
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

	return ActivityDto{
		Id:                               activity.Id,
		Name:                             activity.Name,
		Type:                             activity.Type,
		Link:                             link,
		Distance:                         int(activity.Distance),
		ElapsedTime:                      activity.ElapsedTime,
		MovingTime:                       activity.MovingTime,
		TotalElevationGain:               int(activity.TotalElevationGain),
		AverageSpeed:                     activity.AverageSpeed,       // in m/s
		BestSpeedForDistanceFor1000m:     bestTimeForDistanceFor1000m, // in m/s
		BestElevationForDistanceFor500m:  bestElevationForDistanceFor500m,
		BestElevationForDistanceFor1000m: bestElevationForDistanceFor1000m,
		Date:                             activity.StartDateLocal,
		AverageWatts:                     int(activity.AverageWatts),
		WeightedAverageWatts:             activity.WeightedAverageWatts,
		BestPowerFor20Minutes: func() int {
			if bestPowerFor20Minutes != nil {
				return bestPowerFor20Minutes.GetPower()
			} else {
				return 0
			}
		}(),
		BestPowerFor60Minutes: func() int {
			if bestPowerFor60Minutes != nil {
				return bestPowerFor60Minutes.GetPower()
			} else {
				return 0
			}
		}(),
		FTP: ftp,
	}
}

func ToDetailedActivityDto(detailedActivity *strava.DetailedActivity) DetailedActivityDto {

	activityEfforts := BuildActivityEfforts(detailedActivity)

	return DetailedActivityDto{
		AverageCadence:       int(detailedActivity.AverageCadence),
		AverageHeartrate:     int(detailedActivity.AverageHeartrate),
		AverageWatts:         int(detailedActivity.AverageWatts),
		AverageSpeed:         float32(detailedActivity.AverageSpeed),
		Calories:             detailedActivity.Calories,
		Commute:              detailedActivity.Commute,
		DeviceWatts:          detailedActivity.DeviceWatts,
		Distance:             float64(detailedActivity.Distance),
		ElapsedTime:          detailedActivity.ElapsedTime,
		ElevHigh:             detailedActivity.ElevHigh,
		ID:                   detailedActivity.Id,
		Kilojoules:           detailedActivity.Kilojoules,
		MaxHeartrate:         int(detailedActivity.MaxHeartrate),
		MaxSpeed:             float32(detailedActivity.MaxSpeed),
		MaxWatts:             detailedActivity.MaxWatts,
		MovingTime:           detailedActivity.MovingTime,
		Name:                 detailedActivity.Name,
		ActivityEfforts:      toActivityEffortsDto(activityEfforts),
		StartDate:            parseTime(detailedActivity.StartDate),
		StartDateLocal:       parseTime(detailedActivity.StartDateLocal),
		StartLatlng:          detailedActivity.StartLatLng,
		Stream:               toStreamDto(detailedActivity.Stream),
		SufferScore:          detailedActivity.SufferScore,
		TotalDescent:         detailedActivity.ElevLow,
		TotalElevationGain:   detailedActivity.TotalElevationGain,
		Type:                 detailedActivity.Type,
		WeightedAverageWatts: detailedActivity.WeightedAverageWatts,
	}
}

func toActivityEffortsDto(efforts []business.ActivityEffort) []ActivityEffortDto {
	var effortsDto []ActivityEffortDto
	for _, effort := range efforts {
		effortsDto = append(effortsDto, ActivityEffortDto{
			ID:            uuid.New().String(),
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

func toStreamDto(stream *strava.Stream) *StreamDto {
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

	return &StreamDto{
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

func ToAthleteDto(athlete strava.Athlete) AthleteDto {
	return AthleteDto{
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

func ToStatisticDto(statistic statistics.Statistic) StatisticDto {
	if statistic.Activity() != nil {
		return StatisticDto{
			Label: statistic.Label(),
			Value: statistic.Value(),
			Activity: &ActivityShortDto{
				ID:   statistic.Activity().Id,
				Name: statistic.Activity().Name,
				Type: statistic.Activity().Type.String(),
			},
		}
	} else {
		return StatisticDto{
			Label: statistic.Label(),
			Value: statistic.Value(),
		}
	}
}

func ToDashboardDataDto(data business.DashboardData) DashboardDataDto {
	return DashboardDataDto{
		NbActivitiesByYear:     data.NbActivities,
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
		AverageCadenceByYear:   data.AverageCadence,
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

func BuildActivityEfforts(activity *strava.DetailedActivity) []business.ActivityEffort {
	var activityEfforts []business.ActivityEffort

	slopes := activity.Stream.ListSlopesDefault()
	//.filter { slope.type == SlopeType.ASCENT }
	ascentSlopes := make([]strava.Slope, 0, len(slopes))
	for _, slope := range slopes {
		if slope.Type == strava.ASCENT {
			ascentSlopes = append(ascentSlopes, slope)
		}
	}

	for index, s := range ascentSlopes {

		e := business.ActivityEffort{
			Distance:      s.Distance,
			Seconds:       s.Duration,
			DeltaAltitude: s.EndAltitude - s.StartAltitude,
			IdxStart:      s.StartIndex,
			IdxEnd:        s.EndIndex,
			AveragePower:  nil,
			Label:         fmt.Sprintf("Slope %.1d - max %.1f %%", index, s.MaxGrade),
			ActivityShort: business.ActivityShort{
				Id:   activity.Id,
				Name: activity.Name,
				Type: business.ActivityTypes[activity.Type],
			},
		}
		activityEfforts = append(activityEfforts, e)
	}

	// Add additional efforts based on specific criteria
	bestTimeFor1000m := calculateBestTimeForDistance(activity, 1000.0)
	if bestTimeFor1000m != nil {
		activityEfforts = append(activityEfforts, *bestTimeFor1000m)
	}
	bestTimeFor5000m := calculateBestTimeForDistance(activity, 5000.0)
	if bestTimeFor5000m != nil {
		activityEfforts = append(activityEfforts, *bestTimeFor5000m)
	}
	bestTimeFor10000m := calculateBestTimeForDistance(activity, 10000.0)
	if bestTimeFor10000m != nil {
		activityEfforts = append(activityEfforts, *bestTimeFor10000m)
	}
	bestDistanceFor1Hour := calculateBestDistanceForTime(activity, 3600)
	if bestDistanceFor1Hour != nil {
		activityEfforts = append(activityEfforts, *bestDistanceFor1Hour)
	}
	bestElevationFor500m := calculateBestElevationForDistance(activity, 500.0)
	if bestElevationFor500m != nil {
		activityEfforts = append(activityEfforts, *bestElevationFor500m)
	}
	bestElevationFor1000m := calculateBestElevationForDistance(activity, 1000.0)
	if bestElevationFor1000m != nil {
		activityEfforts = append(activityEfforts, *bestElevationFor1000m)
	}
	bestElevationFor10000m := calculateBestElevationForDistance(activity, 10000.0)
	if bestElevationFor10000m != nil {
		activityEfforts = append(activityEfforts, *bestElevationFor10000m)
	}

	for _, segmentEffort := range activity.SegmentEfforts {
		if segmentEffort.Segment.ClimbCategory > 2 || segmentEffort.Segment.Starred {
			activityEfforts = append(activityEfforts, toActivityEffort(&segmentEffort))
		}
	}

	return activityEfforts
}

func toActivityEffort(effort *strava.SegmentEffort) business.ActivityEffort {

	return business.ActivityEffort{
		Distance:      effort.Distance,
		Seconds:       effort.ElapsedTime,
		DeltaAltitude: effort.Segment.ElevationHigh - effort.Segment.ElevationLow,
		IdxStart:      effort.StartIndex,
		IdxEnd:        effort.EndIndex,
		AveragePower:  &effort.AverageWatts,
		Label:         effort.Segment.Name,
		ActivityShort: business.ActivityShort{
			Id:   effort.Id,
			Name: effort.Segment.Name,
			Type: business.ActivityTypes[effort.Segment.ActivityType],
		},
	}
}

func calculateBestElevationForDistance(activity *strava.DetailedActivity, f float64) *business.ActivityEffort {
	if activity.Stream == nil {
		return nil
	}
	return statistics.BestElevationForDistance(activity.Id, activity.Name, activity.Type, activity.Stream, f)
}

func calculateBestDistanceForTime(activity *strava.DetailedActivity, i int) *business.ActivityEffort {
	if activity.Stream == nil {
		return nil
	}
	return statistics.BestDistanceForTime(activity.Id, activity.Name, activity.Type, activity.Stream, i)
}

func calculateBestTimeForDistance(activity *strava.DetailedActivity, f float64) *business.ActivityEffort {

	if activity.Stream == nil {
		return nil
	}
	return statistics.BestTimeForDistance(activity.Id, activity.Name, activity.Type, activity.Stream, f)
}

func ToBadgeCheckResultDto(result business.BadgeCheckResult, activityTypes ...business.ActivityType) BadgeCheckResultDto {
	nbCheckedActivities := len(result.Activities)
	var activities []ActivityDto
	if nbCheckedActivities > 0 {
		activities = append(activities, ToActivityDto(*result.Activities[nbCheckedActivities-1]))
	}

	return BadgeCheckResultDto{
		Badge:               ToBadgeDto(result.Badge, activityTypes...),
		Activities:          activities,
		NbCheckedActivities: nbCheckedActivities,
	}
}

func ToBadgeDto(badge business.Badge, activityTypes ...business.ActivityType) BadgeDto {

	// TODO: handle case multiple activity types
	activityType := activityTypes[0]

	switch b := badge.(type) {
	case badges.DistanceBadge:
		return BadgeDto{
			Label:       b.Label,
			Description: strconv.FormatFloat(b.Distance, 'f', 0, 64),
			Type:        activityType.String() + "DistanceBadge",
		}
	case badges.ElevationBadge:
		return BadgeDto{
			Label:       b.Label,
			Description: strconv.FormatFloat(b.TotalElevationGain, 'f', 0, 64),
			Type:        activityType.String() + "ElevationBadge",
		}
	case badges.MovingTimeBadge:
		return BadgeDto{
			Label:       b.Label,
			Description: strconv.Itoa(b.MovingTime),
			Type:        activityType.String() + "MovingTimeBadge",
		}
	case badges.FamousClimbBadge:
		return BadgeDto{
			Label:       b.Label,
			Description: b.Name,
			Type:        activityType.String() + "FamousClimbBadge",
		}
	default:
		return BadgeDto{}
	}
}
