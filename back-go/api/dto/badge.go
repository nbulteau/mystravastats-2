package dto

type BadgeCheckResultDto struct {
	Badge               BadgeDto      `json:"badge"`
	Activities          []ActivityDto `json:"activities"`
	NbCheckedActivities int           `json:"nbCheckedActivities"`
}

type BadgeDto struct {
	Label       string `json:"label"`
	Description string `json:"description"`
	Type        string `json:"type"`
}
