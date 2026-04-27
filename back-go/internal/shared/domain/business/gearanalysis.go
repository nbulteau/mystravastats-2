package business

type GearKind string

const (
	GearKindBike    GearKind = "BIKE"
	GearKindShoe    GearKind = "SHOE"
	GearKindUnknown GearKind = "UNKNOWN"
)

type GearAnalysis struct {
	Items      []GearAnalysisItem
	Unassigned GearAnalysisSummary
	Coverage   GearAnalysisCoverage
}

type GearAnalysisItem struct {
	ID                       string
	Name                     string
	Kind                     GearKind
	Retired                  bool
	Primary                  bool
	MaintenanceStatus        string
	MaintenanceLabel         string
	MaintenanceTasks         []GearMaintenanceTask
	MaintenanceHistory       []GearMaintenanceRecord
	Distance                 float64
	MovingTime               int
	ElevationGain            float64
	Activities               int
	AverageSpeed             float64
	FirstUsed                string
	LastUsed                 string
	LongestActivity          *ActivityShort
	BiggestElevationActivity *ActivityShort
	FastestActivity          *ActivityShort
	MonthlyDistance          []GearAnalysisPeriodPoint
}

type GearAnalysisSummary struct {
	Distance      float64
	MovingTime    int
	ElevationGain float64
	Activities    int
	AverageSpeed  float64
}

type GearAnalysisCoverage struct {
	TotalActivities      int
	AssignedActivities   int
	UnassignedActivities int
}

type GearAnalysisPeriodPoint struct {
	PeriodKey     string
	Value         float64
	ActivityCount int
}

type GearMaintenanceRecord struct {
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

type GearMaintenanceRecordRequest struct {
	GearID    string  `json:"gearId"`
	Component string  `json:"component"`
	Operation string  `json:"operation"`
	Date      string  `json:"date"`
	Distance  float64 `json:"distance"`
	Note      string  `json:"note,omitempty"`
}

type GearMaintenanceTask struct {
	Component         string                 `json:"component"`
	ComponentLabel    string                 `json:"componentLabel"`
	IntervalDistance  float64                `json:"intervalDistance"`
	IntervalMonths    int                    `json:"intervalMonths"`
	Status            string                 `json:"status"`
	StatusLabel       string                 `json:"statusLabel"`
	DistanceSince     float64                `json:"distanceSince"`
	DistanceRemaining float64                `json:"distanceRemaining"`
	NextDueDistance   float64                `json:"nextDueDistance"`
	MonthsSince       int                    `json:"monthsSince"`
	MonthsRemaining   int                    `json:"monthsRemaining"`
	LastMaintenance   *GearMaintenanceRecord `json:"lastMaintenance,omitempty"`
}
