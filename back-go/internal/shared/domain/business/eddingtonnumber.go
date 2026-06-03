package business

type EddingtonScope string
type EddingtonMetric string
type EddingtonBasis string

const (
	EddingtonScopeLifetime        EddingtonScope = "lifetime"
	EddingtonScopeYear            EddingtonScope = "year"
	EddingtonScopeRolling12Months EddingtonScope = "rolling-12-months"

	EddingtonMetricDistance  EddingtonMetric = "distance"
	EddingtonMetricElevation EddingtonMetric = "elevation"

	EddingtonBasisDays       EddingtonBasis = "days"
	EddingtonBasisActivities EddingtonBasis = "activities"
)

type EddingtonNumber struct {
	Number          int
	List            []int
	Scope           EddingtonScope
	Metric          EddingtonMetric
	Basis           EddingtonBasis
	Unit            string
	NextTarget      int
	QualifyingCount int
	MissingCount    int
	QualifyingDays  int
	MissingDays     int
}
