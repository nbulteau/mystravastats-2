package strava

type MetaAthlete struct {
	Id int64 `json:"id"`
}

type SegmentEffort struct {
	Achievements     []Achievement `json:"achievements"`
	Activity         MetaActivity  `json:"activity"`
	Athlete          MetaAthlete   `json:"athlete"`
	AverageCadence   float64       `json:"average_cadence"`
	AverageHeartRate float64       `json:"average_heartrate"`
	AverageWatts     float64       `json:"average_watts"`
	DeviceWatts      bool          `json:"device_watts"`
	Distance         float64       `json:"distance"`
	ElapsedTime      int           `json:"elapsed_time"`
	EndIndex         int           `json:"end_index"`
	Hidden           bool          `json:"hidden"`
	Id               int64         `json:"id"`
	KomRank          *int          `json:"kom_rank,omitempty"`
	MaxHeartRate     float64       `json:"max_heartrate"`
	MovingTime       int           `json:"moving_time"`
	Name             string        `json:"name"`
	PrRank           *int          `json:"pr_rank,omitempty"`
	ResourceState    int           `json:"resource_state"`
	Segment          Segment       `json:"segment"`
	StartDate        string        `json:"start_date"`
	StartDateLocal   string        `json:"start_date_local"`
	StartIndex       int           `json:"start_index"`
	Visibility       *string       `json:"visibility,omitempty"`
}
