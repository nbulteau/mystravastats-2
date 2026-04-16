package strava

type Achievement struct {
	EffortCount int    `json:"effort_count"`
	Rank        int    `json:"rank"`
	Type        string `json:"type"`
	TypeId      int    `json:"type_id"`
}
