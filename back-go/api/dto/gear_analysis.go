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
	MaintenanceTasks         []GearMaintenanceTaskDto     `json:"maintenanceTasks"`
	MaintenanceHistory       []GearMaintenanceRecordDto   `json:"maintenanceHistory"`
	Distance                 float64                      `json:"distance"`
	TotalDistance            float64                      `json:"totalDistance"`
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

type GearMaintenanceRecordDto struct {
	ID             string  `json:"id"`
	GearID         string  `json:"gearId"`
	GearName       string  `json:"gearName"`
	Component      string  `json:"component"`
	ComponentLabel string  `json:"componentLabel"`
	Operation      string  `json:"operation"`
	Date           string  `json:"date"`
	Distance       float64 `json:"distance"`
	Note           string  `json:"note,omitempty"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

type GearMaintenanceRecordRequestDto struct {
	GearID    string  `json:"gearId"`
	Component string  `json:"component"`
	Operation string  `json:"operation"`
	Date      string  `json:"date"`
	Distance  float64 `json:"distance"`
	Note      string  `json:"note,omitempty"`
}

type GearMaintenanceTaskDto struct {
	Component         string                    `json:"component"`
	ComponentLabel    string                    `json:"componentLabel"`
	IntervalDistance  float64                   `json:"intervalDistance"`
	IntervalMonths    int                       `json:"intervalMonths"`
	Status            string                    `json:"status"`
	StatusLabel       string                    `json:"statusLabel"`
	DistanceSince     float64                   `json:"distanceSince"`
	DistanceRemaining float64                   `json:"distanceRemaining"`
	NextDueDistance   float64                   `json:"nextDueDistance"`
	MonthsSince       int                       `json:"monthsSince"`
	MonthsRemaining   int                       `json:"monthsRemaining"`
	LastMaintenance   *GearMaintenanceRecordDto `json:"lastMaintenance,omitempty"`
}
