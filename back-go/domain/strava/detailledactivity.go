package strava

import (
	"mystravastats/domain/business"
)

type MetaActivity struct {
	Id int64 `json:"id"`
}

type DetailedActivity struct {
	AchievementCount         int             `json:"achievement_count"`
	Athlete                  MetaActivity    `json:"athlete"`
	AthleteCount             int             `json:"athlete_count"`
	AverageCadence           float64         `json:"average_cadence"`
	AverageHeartrate         float64         `json:"average_heartrate"`
	AverageSpeed             float64         `json:"average_speed"`
	AverageTemp              int             `json:"average_temp"`
	AverageWatts             float64         `json:"average_watts"`
	Calories                 float64         `json:"calories"`
	CommentCount             int             `json:"comment_count"`
	Commute                  bool            `json:"commute"`
	Description              *string         `json:"description,omitempty"`
	DeviceName               *string         `json:"device_name,omitempty"`
	DeviceWatts              bool            `json:"device_watts"`
	Distance                 int             `json:"distance"`
	ElapsedTime              int             `json:"elapsed_time"`
	ElevHigh                 float64         `json:"elev_high"`
	ElevLow                  float64         `json:"elev_low"`
	EmbedToken               string          `json:"embed_token"`
	EndLatLng                []float64       `json:"end_latlng"`
	ExternalId               string          `json:"external_id"`
	Flagged                  bool            `json:"flagged"`
	FromAcceptedTag          bool            `json:"from_accepted_tag"`
	Gear                     *Gear           `json:"gear,omitempty"`
	GearId                   *string         `json:"gear_id,omitempty"`
	HasHeartRate             bool            `json:"has_heartrate"`
	HasKudoed                bool            `json:"has_kudoed"`
	HideFromHome             bool            `json:"hide_from_home"`
	Id                       int64           `json:"id"`
	Kilojoules               float64         `json:"kilojoules"`
	KudosCount               int             `json:"kudos_count"`
	LeaderboardOptOut        bool            `json:"leaderboard_opt_out"`
	Map                      *GeoMap         `json:"map,omitempty"`
	Manual                   bool            `json:"manual"`
	MaxHeartrate             float64         `json:"max_heartrate"`
	MaxSpeed                 float64         `json:"max_speed"`
	MaxWatts                 int             `json:"max_watts"`
	MovingTime               int             `json:"moving_time"`
	Name                     string          `json:"name"`
	PrCount                  int             `json:"pr_count"`
	IsPrivate                bool            `json:"private"`
	ResourceState            int             `json:"resource_state"`
	SegmentEfforts           []SegmentEffort `json:"segment_efforts"`
	SegmentLeaderboardOptOut bool            `json:"segment_leaderboard_opt_out"`
	SplitsMetric             []SplitsMetric  `json:"splits_metric"`
	SportType                string          `json:"sport_type"`
	StartDate                string          `json:"start_date"`
	StartDateLocal           string          `json:"start_date_local"`
	StartLatLng              []float64       `json:"start_latlng"`
	SufferScore              *float64        `json:"suffer_score,omitempty"`
	Timezone                 string          `json:"timezone"`
	TotalElevationGain       int             `json:"total_elevation_gain"`
	TotalPhotoCount          int             `json:"total_photo_count"`
	Trainer                  bool            `json:"trainer"`
	Type                     string          `json:"type"`
	UploadId                 int64           `json:"upload_id"`
	UtcOffset                int             `json:"utc_offset"`
	WeightedAverageWatts     int             `json:"weighted_average_watts"`
	WorkoutType              int             `json:"workout_type"`
	Stream                   *Stream         `json:"stream,omitempty"`
}

func (activity *DetailedActivity) BuildActivityEfforts() []business.ActivityEffort {
	var activityEfforts []business.ActivityEffort

	for _, segmentEffort := range activity.SegmentEfforts {
		if segmentEffort.Segment.ClimbCategory > 2 || segmentEffort.Segment.Starred {
			activityEfforts = append(activityEfforts, segmentEffort.toActivityEffort())
		}
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

	return activityEfforts
}

func calculateBestElevationForDistance(activity *DetailedActivity, f float64) *business.ActivityEffort {
	if activity.Stream == nil {
		return nil
	}
	// TODO: Implement
	return nil
}

func calculateBestDistanceForTime(activity *DetailedActivity, i int) *business.ActivityEffort {
	if activity.Stream == nil {
		return nil
	}
	// TODO: Implement
	return nil
}

func calculateBestTimeForDistance(activity *DetailedActivity, f float64) *business.ActivityEffort {

	if activity.Stream == nil {
		return nil
	}
	// TODO: Implement
	return nil
}
