package business

type FtpEstimate struct {
	Available      bool    `json:"available"`
	Ftp            int     `json:"ftp"`
	Method         string  `json:"method"`
	MethodLabel    string  `json:"methodLabel"`
	BestPower      int     `json:"bestPower"`
	Multiplier     float64 `json:"multiplier"`
	BasedOnSeconds int     `json:"basedOnSeconds"`
	Confidence     string  `json:"confidence"`
	Source         string  `json:"source"`
	SourceKind     string  `json:"sourceKind"`
	ActivityID     int64   `json:"activityId"`
	ActivityName   string  `json:"activityName"`
	ActivityType   string  `json:"activityType"`
	ActivityDate   string  `json:"activityDate"`
	WindowDays     int     `json:"windowDays"`
	ActivityCount  int     `json:"activityCount"`
}
