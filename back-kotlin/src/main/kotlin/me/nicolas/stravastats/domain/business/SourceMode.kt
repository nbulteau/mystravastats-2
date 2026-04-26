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
)
