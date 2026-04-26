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
