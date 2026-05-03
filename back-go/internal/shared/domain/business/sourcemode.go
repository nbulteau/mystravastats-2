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

type StravaOAuthStartRequest struct {
	Path         string `json:"path"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	UseCache     bool   `json:"useCache"`
}

type StravaOAuthStartResult struct {
	Status           string `json:"status"`
	Message          string `json:"message"`
	AuthorizeURL     string `json:"authorizeUrl"`
	SettingsURL      string `json:"settingsUrl"`
	CallbackDomain   string `json:"callbackDomain"`
	OAuthCallbackURL string `json:"oauthCallbackUrl"`
	CredentialsFile  string `json:"credentialsFile"`
	TokenFile        string `json:"tokenFile"`
	CacheOnly        bool   `json:"cacheOnly"`
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

type StravaOAuthStatus struct {
	Status                 string   `json:"status"`
	Message                string   `json:"message"`
	SettingsURL            string   `json:"settingsUrl"`
	CallbackDomain         string   `json:"callbackDomain"`
	OAuthCallbackURL       string   `json:"oauthCallbackUrl"`
	SetupCommand           string   `json:"setupCommand"`
	CredentialsFile        string   `json:"credentialsFile"`
	TokenFile              string   `json:"tokenFile"`
	CredentialsFilePresent bool     `json:"credentialsFilePresent"`
	CredentialsPresent     bool     `json:"credentialsPresent"`
	ClientIDPresent        bool     `json:"clientIdPresent"`
	ClientSecretPresent    bool     `json:"clientSecretPresent"`
	CacheOnly              bool     `json:"cacheOnly"`
	TokenPresent           bool     `json:"tokenPresent"`
	TokenReadable          bool     `json:"tokenReadable"`
	AccessTokenPresent     bool     `json:"accessTokenPresent"`
	RefreshTokenPresent    bool     `json:"refreshTokenPresent"`
	TokenExpired           bool     `json:"tokenExpired"`
	TokenExpiresAt         string   `json:"tokenExpiresAt"`
	AthleteID              string   `json:"athleteId"`
	AthleteName            string   `json:"athleteName"`
	ScopesVerified         bool     `json:"scopesVerified"`
	GrantedScopes          []string `json:"grantedScopes"`
	RequiredScopes         []string `json:"requiredScopes"`
	MissingScopes          []string `json:"missingScopes"`
	TokenError             string   `json:"tokenError"`
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
	StravaOAuth       *StravaOAuthStatus              `json:"stravaOAuth,omitempty"`
}
