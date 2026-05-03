package me.nicolas.stravastats.domain.business

enum class SourceMode {
    STRAVA,
    FIT,
    GPX,
}

data class SourceModePreviewRequest(
    val mode: String = "STRAVA",
    val path: String = "",
)

data class StravaOAuthStartRequest(
    val path: String = "",
    val clientId: String = "",
    val clientSecret: String = "",
    val useCache: Boolean = false,
)

data class StravaOAuthStartResult(
    val status: String,
    val message: String,
    val authorizeUrl: String = "",
    val settingsUrl: String,
    val callbackDomain: String,
    val oauthCallbackUrl: String,
    val credentialsFile: String,
    val tokenFile: String,
    val cacheOnly: Boolean,
)

data class SourceModeYearPreview(
    val year: String,
    val fileCount: Int,
    val validFileCount: Int,
    val activityCount: Int,
)

data class SourceModePreviewError(
    val path: String = "",
    val message: String,
)

data class SourceModeEnvironmentVariable(
    val key: String,
    val value: String,
    val required: Boolean,
)

data class StravaOAuthStatus(
    val status: String,
    val message: String,
    val settingsUrl: String,
    val callbackDomain: String,
    val oauthCallbackUrl: String,
    val setupCommand: String,
    val credentialsFile: String,
    val tokenFile: String,
    val credentialsFilePresent: Boolean,
    val credentialsPresent: Boolean,
    val clientIdPresent: Boolean,
    val clientSecretPresent: Boolean,
    val cacheOnly: Boolean,
    val tokenPresent: Boolean,
    val tokenReadable: Boolean,
    val accessTokenPresent: Boolean,
    val refreshTokenPresent: Boolean,
    val tokenExpired: Boolean,
    val tokenExpiresAt: String,
    val athleteId: String,
    val athleteName: String,
    val scopesVerified: Boolean,
    val grantedScopes: List<String>,
    val requiredScopes: List<String>,
    val missingScopes: List<String>,
    val tokenError: String,
)

data class SourceModePreview(
    val mode: SourceMode,
    val path: String,
    val configKey: String,
    val supported: Boolean,
    val activeMode: SourceMode = SourceMode.STRAVA,
    val active: Boolean = false,
    val configured: Boolean,
    val readable: Boolean,
    val validStructure: Boolean,
    val restartNeeded: Boolean,
    val activationCommand: String = "",
    val fileCount: Int,
    val validFileCount: Int,
    val invalidFileCount: Int,
    val activityCount: Int,
    val years: List<SourceModeYearPreview>,
    val missingFields: List<String>,
    val environment: List<SourceModeEnvironmentVariable> = emptyList(),
    val errors: List<SourceModePreviewError>,
    val recommendations: List<String>,
    val stravaOAuth: StravaOAuthStatus? = null,
)
