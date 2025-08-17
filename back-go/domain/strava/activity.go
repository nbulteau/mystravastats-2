package strava

type AthleteRef struct {
	ID int `json:"id"`
}

type Activity struct {
	Athlete              AthleteRef `json:"athlete"`
	AverageSpeed         float64    `json:"average_speed"`
	AverageCadence       float64    `json:"average_cadence"`
	AverageHeartrate     float64    `json:"average_heartrate"`
	MaxHeartrate         float64    `json:"max_heartrate"`
	AverageWatts         float64    `json:"average_watts"`
	Commute              bool       `json:"commute"`
	Distance             float64    `json:"distance"`
	DeviceWatts          bool       `json:"device_watts"`
	ElapsedTime          int        `json:"elapsed_time"`
	ElevHigh             float64    `json:"elev_high"`
	Id                   int64      `json:"id"`
	Kilojoules           float64    `json:"kilojoules"`
	MaxSpeed             float64    `json:"max_speed"`
	MovingTime           int        `json:"moving_time"`
	Name                 string     `json:"name"`
	SportType            string     `json:"sport_type"`
	StartDate            string     `json:"start_date"`
	StartDateLocal       string     `json:"start_date_local"`
	StartLatlng          []float64  `json:"start_latlng"`
	TotalElevationGain   float64    `json:"total_elevation_gain"`
	Type                 string     `json:"type"`
	UploadId             int64      `json:"upload_id"`
	WeightedAverageWatts int        `json:"weighted_average_watts"`
	Stream               *Stream    `json:"stream"`
}

func (activity *Activity) ToStravaDetailedActivity() *DetailedActivity {
	return &DetailedActivity{
		AchievementCount:         0,
		Athlete:                  MetaActivity{Id: 0},
		AthleteCount:             1,
		AverageCadence:           activity.AverageCadence,
		AverageHeartrate:         activity.AverageHeartrate,
		AverageSpeed:             activity.AverageSpeed,
		AverageTemp:              0,
		AverageWatts:             activity.AverageWatts,
		Calories:                 0.0,
		CommentCount:             0,
		Commute:                  activity.Commute,
		Description:              nil,
		DeviceName:               nil,
		DeviceWatts:              activity.DeviceWatts,
		Distance:                 int(activity.Distance),
		ElapsedTime:              activity.ElapsedTime,
		ElevHigh:                 activity.ElevHigh,
		ElevLow:                  0.0,
		EmbedToken:               "",
		EndLatLng:                []float64{},
		ExternalId:               "",
		Flagged:                  false,
		FromAcceptedTag:          false,
		Gear:                     nil,
		GearId:                   nil,
		HasHeartRate:             true,
		HasKudoed:                false,
		HideFromHome:             false,
		Id:                       activity.Id,
		Kilojoules:               activity.Kilojoules,
		KudosCount:               0,
		LeaderboardOptOut:        false,
		Map:                      nil,
		Manual:                   false,
		MaxHeartrate:             activity.MaxHeartrate,
		MaxSpeed:                 activity.MaxSpeed,
		MaxWatts:                 0,
		MovingTime:               activity.MovingTime,
		Name:                     activity.Name,
		PrCount:                  0,
		ResourceState:            0,
		SegmentEfforts:           []SegmentEffort{},
		SegmentLeaderboardOptOut: false,
		SplitsMetric:             nil,
		SportType:                activity.SportType,
		StartDate:                activity.StartDate,
		StartDateLocal:           activity.StartDateLocal,
		StartLatLng:              activity.StartLatlng,
		SufferScore:              nil,
		Timezone:                 "",
		TotalElevationGain:       int(activity.TotalElevationGain),
		TotalPhotoCount:          0,
		Trainer:                  false,
		Type:                     activity.Type,
		UploadId:                 activity.UploadId,
		UtcOffset:                0,
		WeightedAverageWatts:     activity.WeightedAverageWatts,
		WorkoutType:              0,
		Stream:                   activity.Stream,
	}
}
