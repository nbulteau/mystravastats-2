package dto

type BadgeCheckResultDto struct {
	Badge               BadgeDto         `json:"badge"`
	Activities          []ActivityDto    `json:"activities"`
	NbCheckedActivities int              `json:"nbCheckedActivities"`
	ClimbDetails        *ClimbDetailsDto `json:"climbDetails,omitempty"`
}

type BadgeDto struct {
	Label       string `json:"label"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Category    string `json:"category,omitempty"`
}

// ClimbDetailsDto contains the print-ready data attached to a famous-climb badge.
// BestAscent stays nil when no matching activity has usable timing data.
type ClimbDetailsDto struct {
	Name            string                 `json:"name"`
	Country         string                 `json:"country"`
	Massif          string                 `json:"massif"`
	SourceURL       string                 `json:"sourceUrl,omitempty"`
	SummitAltitude  int                    `json:"summitAltitude"`
	MinimumAltitude int                    `json:"minimumAltitude"`
	LengthKm        float64                `json:"lengthKm"`
	TotalAscent     int                    `json:"totalAscent"`
	Difficulty      int                    `json:"difficulty"`
	AverageGradient float64                `json:"averageGradient"`
	MaximumGradient *float64               `json:"maximumGradient,omitempty"`
	Profile         []ClimbProfilePointDto `json:"profile"`
	AscentCount     int                    `json:"ascentCount"`
	BestAscent      *ClimbAscentDto        `json:"bestAscent,omitempty"`
}

type ClimbProfilePointDto struct {
	DistanceKm float64 `json:"distanceKm"`
	Elevation  float64 `json:"elevation"`
}

type ClimbAscentDto struct {
	ActivityID      int64  `json:"activityId"`
	Date            string `json:"date"`
	DurationSeconds int    `json:"durationSeconds"`
}
