package dto

type StatisticDto struct {
	Label    string            `json:"label"`
	Value    string            `json:"value"`
	Activity *ActivityShortDto `json:"activity,omitempty"`
}

type ActivityShortDto struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}
