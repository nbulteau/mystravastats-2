package business

type HeartRateZoneSettings struct {
	MaxHr       *int `json:"maxHr,omitempty"`
	ThresholdHr *int `json:"thresholdHr,omitempty"`
	ReserveHr   *int `json:"reserveHr,omitempty"`
}

type HeartRateZoneMethod string

const (
	HeartRateZoneMethodMax       HeartRateZoneMethod = "MAX"
	HeartRateZoneMethodThreshold HeartRateZoneMethod = "THRESHOLD"
	HeartRateZoneMethodReserve   HeartRateZoneMethod = "RESERVE"
)

type HeartRateZoneSource string

const (
	HeartRateZoneSourceAthleteSettings HeartRateZoneSource = "ATHLETE_SETTINGS"
	HeartRateZoneSourceDerivedFromData HeartRateZoneSource = "DERIVED_FROM_DATA"
)

type ResolvedHeartRateZoneSettings struct {
	MaxHr       int                 `json:"maxHr"`
	ThresholdHr *int                `json:"thresholdHr,omitempty"`
	ReserveHr   *int                `json:"reserveHr,omitempty"`
	Method      HeartRateZoneMethod `json:"method"`
	Source      HeartRateZoneSource `json:"source"`
}

type HeartRateZoneDistribution struct {
	Zone       string  `json:"zone"`
	Label      string  `json:"label"`
	Seconds    int     `json:"seconds"`
	Percentage float64 `json:"percentage"`
}

type HeartRateZoneActivitySummary struct {
	Activity            ActivityShort               `json:"activity"`
	ActivityDate        string                      `json:"activityDate"`
	TotalTrackedSeconds int                         `json:"totalTrackedSeconds"`
	EasySeconds         int                         `json:"easySeconds"`
	HardSeconds         int                         `json:"hardSeconds"`
	EasyHardRatio       *float64                    `json:"easyHardRatio,omitempty"`
	Zones               []HeartRateZoneDistribution `json:"zones"`
}

type HeartRateZonePeriodSummary struct {
	Period              string                      `json:"period"`
	TotalTrackedSeconds int                         `json:"totalTrackedSeconds"`
	EasySeconds         int                         `json:"easySeconds"`
	HardSeconds         int                         `json:"hardSeconds"`
	EasyHardRatio       *float64                    `json:"easyHardRatio,omitempty"`
	Zones               []HeartRateZoneDistribution `json:"zones"`
}

type HeartRateZoneAnalysis struct {
	Settings            HeartRateZoneSettings          `json:"settings"`
	ResolvedSettings    *ResolvedHeartRateZoneSettings `json:"resolvedSettings,omitempty"`
	HasHeartRateData    bool                           `json:"hasHeartRateData"`
	TotalTrackedSeconds int                            `json:"totalTrackedSeconds"`
	EasyHardRatio       *float64                       `json:"easyHardRatio,omitempty"`
	Zones               []HeartRateZoneDistribution    `json:"zones"`
	Activities          []HeartRateZoneActivitySummary `json:"activities"`
	ByMonth             []HeartRateZonePeriodSummary   `json:"byMonth"`
	ByYear              []HeartRateZonePeriodSummary   `json:"byYear"`
}
