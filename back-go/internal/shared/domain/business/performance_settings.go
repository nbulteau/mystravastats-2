package business

type AthleteFtpSetting struct {
	EffectiveFrom string `json:"effectiveFrom"`
	Ftp           int    `json:"ftp"`
}

type AthletePerformanceSettings struct {
	FtpHistory []AthleteFtpSetting `json:"ftpHistory"`
	WeightKg   *float64            `json:"weightKg,omitempty"`
}
