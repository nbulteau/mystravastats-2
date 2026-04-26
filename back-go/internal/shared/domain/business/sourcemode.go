package business

type SourceMode string

const (
	SourceModeStrava SourceMode = "STRAVA"
	SourceModeFIT    SourceMode = "FIT"
	SourceModeGPX    SourceMode = "GPX"
)

type SourceModePreviewRequest struct {
	Mode string `json:"mode"`
	Path string `json:"path"`
}

type SourceModeYearPreview struct {
	Year           string `json:"year"`
	FileCount      int    `json:"fileCount"`
	ValidFileCount int    `json:"validFileCount"`
	ActivityCount  int    `json:"activityCount"`
}

type SourceModePreviewError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type SourceModeEnvironmentVariable struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Required bool   `json:"required"`
}

type SourceModePreview struct {
	Mode              SourceMode                      `json:"mode"`
	ActiveMode        SourceMode                      `json:"activeMode"`
	Path              string                          `json:"path"`
	ConfigKey         string                          `json:"configKey"`
	Supported         bool                            `json:"supported"`
	Active            bool                            `json:"active"`
	Configured        bool                            `json:"configured"`
	Readable          bool                            `json:"readable"`
	ValidStructure    bool                            `json:"validStructure"`
	RestartNeeded     bool                            `json:"restartNeeded"`
	ActivationCommand string                          `json:"activationCommand"`
	FileCount         int                             `json:"fileCount"`
	ValidFileCount    int                             `json:"validFileCount"`
	InvalidFileCount  int                             `json:"invalidFileCount"`
	ActivityCount     int                             `json:"activityCount"`
	Years             []SourceModeYearPreview         `json:"years"`
	MissingFields     []string                        `json:"missingFields"`
	Environment       []SourceModeEnvironmentVariable `json:"environment"`
	Errors            []SourceModePreviewError        `json:"errors"`
	Recommendations   []string                        `json:"recommendations"`
}
