package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.adapters.localrepositories.fit.FITRepository
import me.nicolas.stravastats.adapters.localrepositories.gpx.GPXRepository
import me.nicolas.stravastats.adapters.localrepositories.strava.StravaRepository
import me.nicolas.stravastats.domain.RuntimeConfig
import me.nicolas.stravastats.domain.business.SourceMode
import me.nicolas.stravastats.domain.business.SourceModeEnvironmentVariable
import me.nicolas.stravastats.domain.business.SourceModePreview
import me.nicolas.stravastats.domain.business.SourceModePreviewError
import me.nicolas.stravastats.domain.business.SourceModePreviewRequest
import me.nicolas.stravastats.domain.business.SourceModeYearPreview
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import org.springframework.stereotype.Service
import java.io.File
import java.util.Locale

interface ISourceModeService {
    fun preview(request: SourceModePreviewRequest): SourceModePreview
}

@Service
class SourceModeService : ISourceModeService {
    override fun preview(request: SourceModePreviewRequest): SourceModePreview {
        val mode = normalizeMode(request.mode)
        val path = request.path.trim().ifEmpty { configuredPath(mode) }

        return enrichActivation(when (mode) {
            SourceMode.STRAVA -> previewStrava(path)
            SourceMode.FIT -> previewLocal(mode, "FIT_FILES_PATH", "fit", path) { FITRepository(path).loadActivitiesFromCache(it) }
            SourceMode.GPX -> previewLocal(mode, "GPX_FILES_PATH", "gpx", path) { GPXRepository(path).loadActivitiesFromCache(it) }
        })
    }

    private fun normalizeMode(raw: String): SourceMode {
        return when (raw.trim().uppercase(Locale.ROOT)) {
            "FIT" -> SourceMode.FIT
            "GPX" -> SourceMode.GPX
            else -> SourceMode.STRAVA
        }
    }

    private fun configuredPath(mode: SourceMode): String {
        return when (mode) {
            SourceMode.STRAVA -> RuntimeConfig.readConfigValue("STRAVA_CACHE_PATH") ?: "strava-cache"
            SourceMode.FIT -> RuntimeConfig.readConfigValue("FIT_FILES_PATH") ?: ""
            SourceMode.GPX -> RuntimeConfig.readConfigValue("GPX_FILES_PATH") ?: ""
        }
    }

    private fun activeMode(): SourceMode {
        return when {
            RuntimeConfig.readConfigValue("FIT_FILES_PATH") != null -> SourceMode.FIT
            RuntimeConfig.readConfigValue("GPX_FILES_PATH") != null -> SourceMode.GPX
            else -> SourceMode.STRAVA
        }
    }

    private fun enrichActivation(preview: SourceModePreview): SourceModePreview {
        val activeMode = activeMode()
        if (!preview.supported || preview.configKey.isBlank()) {
            return preview.copy(activeMode = activeMode, environment = emptyList())
        }

        return preview.copy(
            activeMode = activeMode,
            active = activeMode == preview.mode && !preview.restartNeeded,
            activationCommand = activationCommand(preview.mode, preview.configKey, preview.path),
            environment = sourceEnvironment(preview.mode, preview.configKey, preview.path),
        )
    }

    private fun activationCommand(mode: SourceMode, configKey: String, path: String): String {
        val trimmedPath = path.trim()
        if (trimmedPath.isBlank()) return ""

        val parts = mutableListOf("env")
        sourceUnsetKeys(mode).forEach { key ->
            parts.add("-u")
            parts.add(key)
        }
        val serverPort = RuntimeConfig.readConfigValue("SERVER_PORT")
            ?: RuntimeConfig.readConfigValue("PORT")
            ?: "8080"
        parts.add("$configKey=${shellQuote(trimmedPath)}")
        parts.add("SERVER_PORT=${shellQuote(serverPort)}")
        parts.add("./gradlew")
        parts.add("bootRun")
        return parts.joinToString(" ")
    }

    private fun sourceEnvironment(mode: SourceMode, configKey: String, path: String): List<SourceModeEnvironmentVariable> {
        return listOf(
            SourceModeEnvironmentVariable(
                key = configKey,
                value = path.trim(),
                required = true,
            ),
        ) + sourceUnsetKeys(mode).map { key ->
            SourceModeEnvironmentVariable(
                key = key,
                value = "",
                required = false,
            )
        }
    }

    private fun sourceUnsetKeys(mode: SourceMode): List<String> {
        return when (mode) {
            SourceMode.STRAVA -> listOf("FIT_FILES_PATH", "GPX_FILES_PATH")
            SourceMode.GPX -> listOf("FIT_FILES_PATH")
            SourceMode.FIT -> listOf("GPX_FILES_PATH")
        }
    }

    private fun shellQuote(value: String): String {
        if (value.isEmpty()) return "''"
        return "'${value.replace("'", "'\\''")}'"
    }

    private fun previewLocal(
        mode: SourceMode,
        configKey: String,
        extension: String,
        path: String,
        loadActivities: (Int) -> List<StravaActivity>,
    ): SourceModePreview {
        val configuredPath = RuntimeConfig.readConfigValue(configKey).orEmpty()
        val configured = configuredPath.isNotBlank()
        val errors = mutableListOf<SourceModePreviewError>()
        val recommendations = mutableListOf<String>()

        if (path.isBlank()) {
            return SourceModePreview(
                mode = mode,
                path = path,
                configKey = configKey,
                supported = true,
                configured = configured,
                readable = false,
                validStructure = false,
                restartNeeded = true,
                fileCount = 0,
                validFileCount = 0,
                invalidFileCount = 0,
                activityCount = 0,
                years = emptyList(),
                missingFields = listOf("activities"),
                errors = listOf(SourceModePreviewError(message = "path is required")),
                recommendations = listOf("Set $configKey to a local $mode directory."),
            )
        }

        val directory = File(path)
        if (!directory.exists() || !directory.isDirectory) {
            return SourceModePreview(
                mode = mode,
                path = path,
                configKey = configKey,
                supported = true,
                configured = configured,
                readable = false,
                validStructure = false,
                restartNeeded = activeMode() != mode || configuredPath != path,
                fileCount = 0,
                validFileCount = 0,
                invalidFileCount = 0,
                activityCount = 0,
                years = emptyList(),
                missingFields = listOf("activities"),
                errors = listOf(SourceModePreviewError(path = path, message = "directory is not readable")),
                recommendations = listOf("Choose the parent directory containing year folders such as 2025/ and 2026/."),
            )
        }

        val fieldStats = SourceFieldStats()
        val years = directory
            .listFiles()
            .orEmpty()
            .filter { it.isDirectory && it.name.matches(Regex("\\d{4}")) }
            .mapNotNull { yearDirectory ->
                val year = yearDirectory.name.toIntOrNull() ?: return@mapNotNull null
                val fileCount = yearDirectory
                    .listFiles { file -> file.isFile && file.extension.lowercase(Locale.ROOT) == extension }
                    .orEmpty()
                    .size
                if (fileCount == 0) return@mapNotNull null
                val activities = runCatching { loadActivities(year) }.getOrElse {
                    errors.add(SourceModePreviewError(path = yearDirectory.absolutePath, message = it.message ?: "unable to parse files"))
                    emptyList()
                }
                activities.forEach { activity -> fieldStats.add(activity) }
                SourceModeYearPreview(
                    year = yearDirectory.name,
                    fileCount = fileCount,
                    validFileCount = activities.size,
                    activityCount = activities.size,
                )
            }
            .sortedByDescending { it.year }

        val fileCount = years.sumOf { it.fileCount }
        val validFileCount = years.sumOf { it.validFileCount }
        val activityCount = years.sumOf { it.activityCount }
        val invalidFileCount = (fileCount - validFileCount).coerceAtLeast(0)

        if (years.isEmpty()) {
            recommendations.add("Use year folders such as 2025/ and 2026/ under the selected directory.")
        }
        if (activityCount > 0) {
            recommendations.add("Set $configKey=$path to use this source.")
        }
        if (activeMode() != mode || configuredPath != path) {
            recommendations.add("Restart the backend after changing the source mode.")
        }

        return SourceModePreview(
            mode = mode,
            path = path,
            configKey = configKey,
            supported = true,
            configured = configured,
            readable = true,
            validStructure = years.isNotEmpty(),
            restartNeeded = activeMode() != mode || configuredPath != path,
            fileCount = fileCount,
            validFileCount = validFileCount,
            invalidFileCount = invalidFileCount,
            activityCount = activityCount,
            years = years,
            missingFields = fieldStats.missingFields(activityCount),
            errors = errors.take(8),
            recommendations = recommendations,
        )
    }

    private fun previewStrava(path: String): SourceModePreview {
        val configuredPath = RuntimeConfig.readConfigValue("STRAVA_CACHE_PATH") ?: "strava-cache"
        val directory = File(path)
        val configured = RuntimeConfig.readConfigValue("STRAVA_CACHE_PATH") != null
        val restartNeeded = activeMode() != SourceMode.STRAVA || configuredPath != path
        if (!directory.exists() || !directory.isDirectory) {
            return SourceModePreview(
                mode = SourceMode.STRAVA,
                path = path,
                configKey = "STRAVA_CACHE_PATH",
                supported = true,
                configured = configured,
                readable = false,
                validStructure = false,
                restartNeeded = restartNeeded,
                fileCount = 0,
                validFileCount = 0,
                invalidFileCount = 0,
                activityCount = 0,
                years = emptyList(),
                missingFields = emptyList(),
                errors = listOf(SourceModePreviewError(path = path, message = "directory is not readable")),
                recommendations = listOf("Choose the Strava cache directory containing the .strava file."),
            )
        }

        val repository = StravaRepository(path)
        val (clientId, _, useCache) = repository.readStravaAuthentication(path)
        if (clientId.isNullOrBlank()) {
            return SourceModePreview(
                mode = SourceMode.STRAVA,
                path = path,
                configKey = "STRAVA_CACHE_PATH",
                supported = true,
                configured = configured,
                readable = true,
                validStructure = false,
                restartNeeded = restartNeeded,
                fileCount = 0,
                validFileCount = 0,
                invalidFileCount = 0,
                activityCount = 0,
                years = emptyList(),
                missingFields = emptyList(),
                errors = listOf(SourceModePreviewError(path = File(directory, ".strava").absolutePath, message = ".strava file is missing or invalid")),
                recommendations = listOf("Configure Strava credentials or switch to FIT/GPX local mode."),
            )
        }

        val athleteDirectory = File(directory, "strava-$clientId")
        val years = athleteDirectory
            .listFiles()
            .orEmpty()
            .filter { it.isDirectory && it.name.startsWith("strava-$clientId-") }
            .mapNotNull { yearDirectory ->
                val year = yearDirectory.name.removePrefix("strava-$clientId-")
                if (!year.matches(Regex("\\d{4}"))) return@mapNotNull null
                val activitiesFile = File(yearDirectory, "activities-$clientId-$year.json")
                if (!activitiesFile.exists()) return@mapNotNull null
                val activityCount = repository.loadActivitiesFromCache(clientId, year.toInt()).size
                SourceModeYearPreview(year = year, fileCount = 1, validFileCount = 1, activityCount = activityCount)
            }
            .sortedByDescending { it.year }
        val recommendations = mutableListOf<String>()
        if (useCache == true) recommendations.add("Strava cache-only mode is enabled.")
        if (restartNeeded) recommendations.add("Restart the backend after changing STRAVA_CACHE_PATH or switching source mode.")

        return SourceModePreview(
            mode = SourceMode.STRAVA,
            path = path,
            configKey = "STRAVA_CACHE_PATH",
            supported = true,
            configured = configured,
            readable = true,
            validStructure = true,
            restartNeeded = restartNeeded,
            fileCount = years.size,
            validFileCount = years.size,
            invalidFileCount = 0,
            activityCount = years.sumOf { it.activityCount },
            years = years,
            missingFields = emptyList(),
            errors = emptyList(),
            recommendations = recommendations,
        )
    }

    private class SourceFieldStats {
        private var heartRate = 0
        private var power = 0
        private var cadence = 0
        private var elevation = 0
        private var trace = 0

        fun add(activity: StravaActivity) {
            val stream = activity.stream ?: return
            if (stream.heartrate != null) heartRate++
            if (stream.watts != null) power++
            if (stream.cadence != null) cadence++
            if (stream.altitude != null) elevation++
            if (stream.latlng != null) trace++
        }

        fun missingFields(activityCount: Int): List<String> {
            if (activityCount <= 0) return listOf("activities")
            val missing = mutableListOf<String>()
            if (trace == 0) missing.add("trace")
            if (elevation == 0) missing.add("elevation")
            if (heartRate == 0) missing.add("heartRate")
            if (power == 0) missing.add("power")
            if (cadence == 0) missing.add("cadence")
            return missing
        }
    }
}
