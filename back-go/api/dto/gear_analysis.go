package dto

type GearAnalysisDto struct {
	Items      []GearAnalysisItemDto   `json:"items"`
	Unassigned GearAnalysisSummaryDto  `json:"unassigned"`
	Coverage   GearAnalysisCoverageDto `json:"coverage"`
}

type GearAnalysisItemDto struct {
	ID                       string                       `json:"id"`
	Name                     string                       `json:"name"`
	Kind                     string                       `json:"kind"`
	Retired                  bool                         `json:"retired"`
	Primary                  bool                         `json:"primary"`
	MaintenanceStatus        string                       `json:"maintenanceStatus"`
	MaintenanceLabel         string                       `json:"maintenanceLabel"`
	Distance                 float64                      `json:"distance"`
	MovingTime               int                          `json:"movingTime"`
	ElevationGain            float64                      `json:"elevationGain"`
	Activities               int                          `json:"activities"`
	AverageSpeed             float64                      `json:"averageSpeed"`
	FirstUsed                string                       `json:"firstUsed"`
	LastUsed                 string                       `json:"lastUsed"`
	LongestActivity          *ActivityShortDto            `json:"longestActivity,omitempty"`
	BiggestElevationActivity *ActivityShortDto            `json:"biggestElevationActivity,omitempty"`
	FastestActivity          *ActivityShortDto            `json:"fastestActivity,omitempty"`
	MonthlyDistance          []GearAnalysisPeriodPointDto `json:"monthlyDistance"`
}

type GearAnalysisSummaryDto struct {
	Distance      float64 `json:"distance"`
	MovingTime    int     `json:"movingTime"`
	ElevationGain float64 `json:"elevationGain"`
	Activities    int     `json:"activities"`
	AverageSpeed  float64 `json:"averageSpeed"`
}

type GearAnalysisCoverageDto struct {
	TotalActivities      int `json:"totalActivities"`
	AssignedActivities   int `json:"assignedActivities"`
	UnassignedActivities int `json:"unassignedActivities"`
}

type GearAnalysisPeriodPointDto struct {
	PeriodKey     string  `json:"periodKey"`
	Value         float64 `json:"value"`
	ActivityCount int     `json:"activityCount"`
}
