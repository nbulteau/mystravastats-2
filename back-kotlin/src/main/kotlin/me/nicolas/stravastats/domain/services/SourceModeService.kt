package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.RuntimeConfig
import me.nicolas.stravastats.domain.business.SourceMode
import me.nicolas.stravastats.domain.business.SourceModeEnvironmentVariable
import me.nicolas.stravastats.domain.business.SourceModePreview
import me.nicolas.stravastats.domain.business.SourceModePreviewError
import me.nicolas.stravastats.domain.business.SourceModePreviewRequest
import me.nicolas.stravastats.domain.business.SourceModeYearPreview
import me.nicolas.stravastats.domain.business.StravaOAuthStartRequest
import me.nicolas.stravastats.domain.business.StravaOAuthStartResult
import me.nicolas.stravastats.domain.business.StravaOAuthStatus
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.interfaces.ISourcePreviewRepositoryFactory
import org.springframework.stereotype.Service
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import tools.jackson.module.kotlin.readValue
import java.io.File
import java.net.URI
import java.net.URLEncoder
import java.net.http.HttpClient
import java.net.http.HttpRequest
import java.net.http.HttpResponse
import java.nio.charset.StandardCharsets
import java.security.SecureRandom
import java.time.Instant
import java.util.Properties
import java.util.Locale
import java.util.concurrent.ConcurrentHashMap

interface ISourceModeService {
    fun preview(request: SourceModePreviewRequest): SourceModePreview
    fun startStravaOAuth(request: StravaOAuthStartRequest, callbackUrl: String = ""): StravaOAuthStartResult
    fun completeStravaOAuth(state: String?, code: String?, scope: String?, error: String?): String
    fun stravaOAuthHtml(title: String, message: String): String
}

@Service
class SourceModeService(
    private val repositoryFactory: ISourcePreviewRepositoryFactory,
) : ISourceModeService {
    private val objectMapper = JsonMapper.builder()
        .addModule(KotlinModule.Builder().build())
        .build()
    private val httpClient: HttpClient = HttpClient.newBuilder().build()
    private val oauthSessions = ConcurrentHashMap<String, StravaOAuthSession>()

    override fun preview(request: SourceModePreviewRequest): SourceModePreview {
        val mode = normalizeMode(request.mode)
        val path = request.path.trim().ifEmpty { configuredPath(mode) }

        return enrichActivation(when (mode) {
            SourceMode.STRAVA -> previewStrava(path)
            SourceMode.FIT -> previewLocal(mode, "FIT_FILES_PATH", "fit", path) {
                repositoryFactory.createFitRepository(path).loadActivitiesFromCache(it)
            }
            SourceMode.GPX -> previewLocal(mode, "GPX_FILES_PATH", "gpx", path) {
                repositoryFactory.createGpxRepository(path).loadActivitiesFromCache(it)
            }
        })
    }

    override fun startStravaOAuth(request: StravaOAuthStartRequest, callbackUrl: String): StravaOAuthStartResult {
        cleanupStravaOAuthSessions()
        val path = request.path.trim().ifBlank { "strava-cache" }
        val existingCredentials = readStravaCredentials(path)
        val clientId = request.clientId.trim().ifBlank { existingCredentials.clientId }
        val clientSecret = request.clientSecret.trim().ifBlank { existingCredentials.clientSecret }
        val useCache = request.useCache

        require(clientId.isNotBlank()) { "clientId is required" }
        require(clientId.matches(Regex("\\d+"))) { "clientId must be numeric" }
        require(useCache || clientSecret.isNotBlank()) { "clientSecret is required for live OAuth" }

        writeStravaCredentials(path, clientId, clientSecret, useCache)
        val resolvedCallbackUrl = callbackUrl.ifBlank { stravaOAuthCallbackUrl() }
        val result = StravaOAuthStartResult(
            status = if (useCache) "cache_only" else "credentials_saved",
            message = if (useCache) "Strava cache-only mode saved." else "Strava credentials saved.",
            settingsUrl = STRAVA_SETTINGS_URL,
            callbackDomain = "127.0.0.1",
            oauthCallbackUrl = resolvedCallbackUrl,
            credentialsFile = File(path, ".strava").absolutePath,
            tokenFile = File(path, ".strava-token.json").absolutePath,
            cacheOnly = useCache,
        )
        if (useCache) {
            return result
        }

        val state = newStravaOAuthState()
        oauthSessions[state] = StravaOAuthSession(
            path = path,
            clientId = clientId,
            clientSecret = clientSecret,
            tokenFile = File(path, ".strava-token.json"),
            createdAt = Instant.now(),
        )
        return result.copy(
            status = "oauth_started",
            message = "Open Strava authorization to finish OAuth.",
            authorizeUrl = stravaAuthorizeUrl(clientId, resolvedCallbackUrl, state),
        )
    }

    override fun completeStravaOAuth(state: String?, code: String?, scope: String?, error: String?): String {
        val session = oauthSessions[state.orEmpty()]
            ?: throw IllegalArgumentException("OAuth session is missing or expired.")
        if (session.createdAt.plusSeconds(STRAVA_OAUTH_SESSION_TTL_SECONDS).isBefore(Instant.now())) {
            oauthSessions.remove(state.orEmpty())
            throw IllegalArgumentException("OAuth session expired. Restart Strava enrollment from MyStravaStats.")
        }
        if (!error.isNullOrBlank()) {
            oauthSessions.remove(state.orEmpty())
            throw IllegalArgumentException("Strava OAuth failed: $error")
        }
        val authorizationCode = code.orEmpty()
        require(authorizationCode.isNotBlank()) { "Strava did not return an authorization code." }
        val missingScopes = missingStravaOAuthScopes(scope.orEmpty())
        require(missingScopes.isEmpty()) { "Missing required scope(s): ${missingScopes.joinToString(", ")}" }

        val tokenPayload = exchangeStravaOAuthCode(session.clientId, session.clientSecret, authorizationCode).toMutableMap()
        tokenPayload["scope"] = scope?.takeIf { it.isNotBlank() } ?: STRAVA_OAUTH_SCOPE
        tokenPayload["athlete"] = fetchStravaAthlete(tokenPayload["access_token"]?.toString().orEmpty())
        tokenPayload["created_at"] = Instant.now().toString()
        writePrivateJson(session.tokenFile, tokenPayload)
        oauthSessions.remove(state.orEmpty())
        return stravaOAuthHtml("Access granted", "Strava OAuth token saved. You can close this window.")
    }

    override fun stravaOAuthHtml(title: String, message: String): String {
        val escapedTitle = htmlEscape(title)
        return """
            <!doctype html>
            <html lang="en">
            <head><meta charset="utf-8"><title>$escapedTitle</title></head>
            <body style="font-family: system-ui, sans-serif; margin: 40px; line-height: 1.4;">
              <h1>$escapedTitle</h1>
              <p>${htmlEscape(message)}</p>
            </body>
            </html>
        """.trimIndent()
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
        val missingOAuthStatus = inspectStravaOAuth(path, clientId = null, clientSecret = null, useCache = false)
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
                stravaOAuth = missingOAuthStatus,
            )
        }

        val repository = repositoryFactory.createStravaRepository(path)
        val (clientId, clientSecret, useCache) = repository.readStravaAuthentication(path)
        val oauthStatus = inspectStravaOAuth(path, clientId, clientSecret, useCache == true)
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
                stravaOAuth = oauthStatus,
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
        if (useCache != true) {
            if (File(directory, ".strava-token.json").exists()) {
                recommendations.add("Strava OAuth token is available for refresh.")
            } else {
                recommendations.add("Run node scripts/setup-strava-oauth.mjs to create .strava-token.json before live Strava refresh.")
            }
        }
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
            stravaOAuth = oauthStatus,
        )
    }

    private fun inspectStravaOAuth(path: String, clientId: String?, clientSecret: String?, useCache: Boolean): StravaOAuthStatus {
        val trimmedPath = path.trim()
        val credentialsFile = File(trimmedPath, ".strava")
        val tokenFile = File(trimmedPath, ".strava-token.json")
        val clientIdPresent = !clientId.isNullOrBlank()
        val clientSecretPresent = !clientSecret.isNullOrBlank()
        val baseStatus = StravaOAuthStatus(
            status = "needs_credentials",
            message = "Create a Strava app, then run the local setup assistant with Client ID and Client Secret.",
            settingsUrl = STRAVA_SETTINGS_URL,
            callbackDomain = "127.0.0.1",
            oauthCallbackUrl = "http://127.0.0.1:8090/exchange_token",
            setupCommand = stravaSetupCommand(trimmedPath),
            credentialsFile = credentialsFile.absolutePath,
            tokenFile = tokenFile.absolutePath,
            credentialsFilePresent = credentialsFile.exists(),
            credentialsPresent = clientIdPresent && clientSecretPresent,
            clientIdPresent = clientIdPresent,
            clientSecretPresent = clientSecretPresent,
            cacheOnly = useCache,
            tokenPresent = false,
            tokenReadable = false,
            accessTokenPresent = false,
            refreshTokenPresent = false,
            tokenExpired = false,
            tokenExpiresAt = "",
            athleteId = "",
            athleteName = "",
            scopesVerified = false,
            grantedScopes = emptyList(),
            requiredScopes = REQUIRED_STRAVA_SCOPES,
            missingScopes = emptyList(),
            tokenError = "",
        )

        if (!baseStatus.credentialsPresent) {
            return baseStatus
        }
        if (useCache) {
            return baseStatus.copy(
                status = "cache_only",
                message = "Strava cache-only mode is enabled; OAuth token is not required until live refresh is re-enabled.",
            )
        }
        if (!tokenFile.exists()) {
            return baseStatus.copy(
                status = "needs_token",
                message = "Credentials are present; run the OAuth assistant to create .strava-token.json.",
            )
        }

        val tokenPayload = runCatching { objectMapper.readValue<Map<String, Any?>>(tokenFile) }.getOrElse { exception ->
            return baseStatus.copy(
                status = "token_unreadable",
                tokenPresent = true,
                tokenError = exception.message.orEmpty(),
                message = "The OAuth token file exists but cannot be read.",
            )
        }

        val accessTokenPresent = tokenPayload["access_token"]?.toString()?.isNotBlank() == true
        val refreshTokenPresent = tokenPayload["refresh_token"]?.toString()?.isNotBlank() == true
        val expiresAtEpoch = tokenPayload["expires_at"].toLongOrNull()
        val expiresAt = expiresAtEpoch?.let { Instant.ofEpochSecond(it) }
        val tokenExpired = expiresAt?.isBefore(Instant.now().plusSeconds(120)) == true
        val grantedScopes = splitStravaScopes(tokenPayload["scope"]?.toString().orEmpty())
        val missingScopes = if (grantedScopes.isNotEmpty()) REQUIRED_STRAVA_SCOPES.filterNot { it in grantedScopes } else emptyList()
        val athlete = tokenPayload["athlete"] as? Map<*, *>
        val athleteId = athlete?.get("id")?.toString().orEmpty()
        val athleteName = athleteDisplayName(
            username = athlete?.get("username")?.toString().orEmpty(),
            firstName = athlete?.get("firstname")?.toString().orEmpty(),
            lastName = athlete?.get("lastname")?.toString().orEmpty(),
        )
        val populatedStatus = baseStatus.copy(
            tokenPresent = true,
            tokenReadable = true,
            accessTokenPresent = accessTokenPresent,
            refreshTokenPresent = refreshTokenPresent,
            tokenExpired = tokenExpired,
            tokenExpiresAt = expiresAt?.toString().orEmpty(),
            athleteId = athleteId,
            athleteName = athleteName,
            scopesVerified = grantedScopes.isNotEmpty(),
            grantedScopes = grantedScopes,
            missingScopes = missingScopes,
        )
        val (status, message) = stravaOAuthStatusMessage(populatedStatus)
        return populatedStatus.copy(status = status, message = message)
    }

    private fun stravaSetupCommand(path: String): String {
        return if (path.isBlank()) {
            "node scripts/setup-strava-oauth.mjs"
        } else {
            "node scripts/setup-strava-oauth.mjs --cache ${shellQuote(path)}"
        }
    }

    private fun splitStravaScopes(scope: String): List<String> {
        return scope.split(",", " ", "\t", "\n")
            .map { it.trim() }
            .filter { it.isNotEmpty() }
            .distinct()
            .sorted()
    }

    private fun Any?.toLongOrNull(): Long? {
        return when (this) {
            is Number -> this.toLong()
            is String -> this.toLongOrNull()
            else -> null
        }
    }

    private fun athleteDisplayName(username: String, firstName: String, lastName: String): String {
        return listOf(firstName.trim(), lastName.trim()).filter { it.isNotBlank() }.joinToString(" ")
            .ifBlank { username.trim() }
    }

    private fun stravaOAuthStatusMessage(status: StravaOAuthStatus): Pair<String, String> {
        if (!status.accessTokenPresent || !status.refreshTokenPresent) {
            return "token_incomplete" to "The token file is incomplete; run the OAuth assistant again."
        }
        if (status.missingScopes.isNotEmpty()) {
            return "scope_incomplete" to "The token is missing Strava scopes; run the OAuth assistant and accept every requested permission."
        }
        if (status.tokenExpired) {
            return if (status.refreshTokenPresent) {
                "refreshable" to "The access token is expired, but the backend can refresh it on next live Strava call."
            } else {
                "token_expired" to "The access token is expired and no refresh token is available."
            }
        }
        if (!status.scopesVerified) {
            return "ready_unverified_scopes" to "OAuth token is available; scopes are not recorded in the token file."
        }
        return "ready" to "Strava credentials and OAuth token are ready."
    }

    private fun readStravaCredentials(path: String): StravaCredentials {
        val file = File(path, ".strava")
        if (!file.exists()) return StravaCredentials()
        val properties = Properties()
        file.inputStream().use { properties.load(it) }
        return StravaCredentials(
            clientId = properties.getProperty("clientId").orEmpty().trim(),
            clientSecret = properties.getProperty("clientSecret").orEmpty().trim(),
            useCache = properties.getProperty("useCache").orEmpty().trim().equals("true", ignoreCase = true),
        )
    }

    private fun writeStravaCredentials(path: String, clientId: String, clientSecret: String, useCache: Boolean) {
        val directory = File(path)
        directory.mkdirs()
        val file = File(directory, ".strava")
        file.writeText("clientId=$clientId\nclientSecret=$clientSecret\nuseCache=$useCache\n")
        file.setReadable(false, false)
        file.setReadable(true, true)
        file.setWritable(false, false)
        file.setWritable(true, true)
        file.setExecutable(false, false)
    }

    private fun stravaOAuthCallbackUrl(): String {
        val port = RuntimeConfig.readConfigValue("SERVER_PORT")
            ?: RuntimeConfig.readConfigValue("PORT")
            ?: "8080"
        return "http://127.0.0.1:$port/api/source-modes/strava/oauth/callback"
    }

    private fun stravaAuthorizeUrl(clientId: String, callbackUrl: String, state: String): String {
        val params = mapOf(
            "client_id" to clientId,
            "response_type" to "code",
            "redirect_uri" to callbackUrl,
            "approval_prompt" to "auto",
            "scope" to STRAVA_OAUTH_SCOPE,
            "state" to state,
        ).entries.joinToString("&") { (key, value) ->
            "${encodeQueryParam(key)}=${encodeQueryParam(value)}"
        }
        return "$STRAVA_AUTHORIZE_URL?$params"
    }

    private fun exchangeStravaOAuthCode(clientId: String, clientSecret: String, code: String): Map<String, Any?> {
        val formBody = mapOf(
            "client_id" to clientId,
            "client_secret" to clientSecret,
            "code" to code,
            "grant_type" to "authorization_code",
        ).toFormBody()
        val request = HttpRequest.newBuilder(URI.create(STRAVA_TOKEN_URL))
            .header("Content-Type", "application/x-www-form-urlencoded")
            .POST(HttpRequest.BodyPublishers.ofString(formBody))
            .build()
        val response = httpClient.send(request, HttpResponse.BodyHandlers.ofString())
        if (response.statusCode() !in 200..299) {
            throw RuntimeException("Strava token exchange failed (${response.statusCode()}): ${response.body()}")
        }
        val tokenPayload: Map<String, Any?> = objectMapper.readValue(response.body())
        if (tokenPayload["access_token"]?.toString().isNullOrBlank() || tokenPayload["refresh_token"]?.toString().isNullOrBlank()) {
            throw RuntimeException("Strava token response is missing access_token or refresh_token")
        }
        return tokenPayload
    }

    private fun fetchStravaAthlete(accessToken: String): Map<String, Any?> {
        val request = HttpRequest.newBuilder(URI.create(STRAVA_ATHLETE_URL))
            .header("Authorization", "Bearer $accessToken")
            .GET()
            .build()
        val response = httpClient.send(request, HttpResponse.BodyHandlers.ofString())
        if (response.statusCode() !in 200..299) {
            throw RuntimeException("Unable to validate Strava athlete (${response.statusCode()}): ${response.body()}")
        }
        return objectMapper.readValue(response.body())
    }

    private fun writePrivateJson(file: File, payload: Map<String, Any?>) {
        file.parentFile?.mkdirs()
        objectMapper.writerWithDefaultPrettyPrinter().writeValue(file, payload)
        file.setReadable(false, false)
        file.setReadable(true, true)
        file.setWritable(false, false)
        file.setWritable(true, true)
        file.setExecutable(false, false)
    }

    private fun Map<String, String>.toFormBody(): String {
        return entries.joinToString("&") { (key, value) ->
            "${encodeQueryParam(key)}=${encodeQueryParam(value)}"
        }
    }

    private fun encodeQueryParam(value: String): String = URLEncoder.encode(value, StandardCharsets.UTF_8)

    private fun newStravaOAuthState(): String {
        val bytes = ByteArray(24)
        SecureRandom().nextBytes(bytes)
        return bytes.joinToString("") { "%02x".format(it.toInt() and 0xff) }
    }

    private fun missingStravaOAuthScopes(scope: String): List<String> {
        if (scope.isBlank()) return emptyList()
        val granted = scope.split(",").map { it.trim() }.filter { it.isNotBlank() }.toSet()
        return REQUIRED_STRAVA_SCOPES.filterNot { it in granted }
    }

    private fun cleanupStravaOAuthSessions() {
        val now = Instant.now()
        oauthSessions.entries.removeIf { (_, session) ->
            session.createdAt.plusSeconds(STRAVA_OAUTH_SESSION_TTL_SECONDS).isBefore(now)
        }
    }

    private fun htmlEscape(value: String): String {
        return value
            .replace("&", "&amp;")
            .replace("<", "&lt;")
            .replace(">", "&gt;")
            .replace("\"", "&quot;")
            .replace("'", "&#39;")
    }

    private data class StravaCredentials(
        val clientId: String = "",
        val clientSecret: String = "",
        val useCache: Boolean = false,
    )

    private data class StravaOAuthSession(
        val path: String,
        val clientId: String,
        val clientSecret: String,
        val tokenFile: File,
        val createdAt: Instant,
    )

    companion object {
        private const val STRAVA_SETTINGS_URL = "https://www.strava.com/settings/api"
        private const val STRAVA_AUTHORIZE_URL = "https://www.strava.com/oauth/authorize"
        private const val STRAVA_TOKEN_URL = "https://www.strava.com/oauth/token"
        private const val STRAVA_ATHLETE_URL = "https://www.strava.com/api/v3/athlete"
        private const val STRAVA_OAUTH_SCOPE = "read_all,activity:read_all,profile:read_all"
        private const val STRAVA_OAUTH_SESSION_TTL_SECONDS = 600L
        private val REQUIRED_STRAVA_SCOPES = listOf("read_all", "activity:read_all", "profile:read_all")
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
