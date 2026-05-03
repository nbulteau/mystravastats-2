package me.nicolas.stravastats.adapters.strava

import kotlinx.coroutines.delay
import kotlinx.coroutines.runBlocking
import kotlin.time.Duration.Companion.milliseconds
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.errors.RateLimitExceededException
import me.nicolas.stravastats.domain.interfaces.IStravaApi
import me.nicolas.stravastats.domain.utils.BrowserUtils.openBrowser
import okhttp3.Headers
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.slf4j.Logger
import org.slf4j.LoggerFactory
import org.springframework.http.HttpStatus
import tools.jackson.databind.DatabindException
import tools.jackson.databind.DeserializationFeature
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import tools.jackson.module.kotlin.readValue
import java.io.File
import java.net.*
import java.nio.charset.StandardCharsets
import java.time.Instant
import java.time.LocalDateTime
import java.time.ZoneOffset
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter
import java.util.UUID
import java.util.concurrent.ThreadLocalRandom
import java.util.concurrent.atomic.AtomicLong
import kotlin.math.roundToLong

internal class StravaApi(clientId: String, clientSecret: String, stravaCache: String? = null) : IStravaApi {

    companion object {
        private const val QUOTA_EXCEED_LIMIT =
            "Quotas exceeded: Strava rate limitations (100 requests every 15 minutes, with up to 1,000 requests per day)"
        private const val MAX_RETRY_AFTER_MS = 120_000L
        private const val RETRY_JITTER_MS = 250L
        private const val RATE_LIMIT_EXHAUSTED_COOLDOWN_MS = 60_000L
        private const val RATE_LIMIT_WINDOW_BUFFER_MS = 1_000L
        private const val MAX_BLOCKING_WAIT_MS = 30_000L
        private const val TOKEN_REFRESH_BUFFER_SECONDS = 3_600L
    }

    private val logger: Logger = LoggerFactory.getLogger(StravaApi::class.java)

    private val objectMapper = JsonMapper.builder()
        .addModule(KotlinModule.Builder().build())
        .disable(DeserializationFeature.FAIL_ON_NULL_FOR_PRIMITIVES)
        .build()

    private val properties: StravaProperties = StravaProperties()

    private val okHttpClient: OkHttpClient = OkHttpClient.Builder().proxy(getProxyFromEnvironment()).build()

    private var accessToken: String? = null
    private val globalRateLimitUntilMs = AtomicLong(0L)
    private val tokenFile: File? = stravaCache?.let { File(it, ".strava-token.json") }

    private fun setAccessToken(accessToken: String) {
        this.accessToken = accessToken
    }

    init {
        setAccessToken(clientId, clientSecret)
    }

    override fun retrieveLoggedInAthlete(): StravaAthlete? {
        try {
            return doGetLoggedInAthlete()
        } catch (connectException: ConnectException) {
            throw RuntimeException("Unable to connect to Strava API : ${connectException.message}")
        }
    }

    override fun getActivities(year: Int): List<StravaActivity> {
        try {
            return doGetActivities(
                before = LocalDateTime.of(year + 1, 1, 1, 0, 0),
                after = LocalDateTime.of(year, 1, 1, 0, 0),
                failFastOnRateLimit = false
            )
        } catch (connectException: ConnectException) {
            throw RuntimeException("Unable to connect to Strava API : ${connectException.message}")
        }
    }

    override fun getActivitiesFailFastOnRateLimit(year: Int): List<StravaActivity> {
        try {
            return doGetActivities(
                before = LocalDateTime.of(year + 1, 1, 1, 0, 0),
                after = LocalDateTime.of(year, 1, 1, 0, 0),
                failFastOnRateLimit = true
            )
        } catch (connectException: ConnectException) {
            throw RuntimeException("Unable to connect to Strava API : ${connectException.message}")
        }
    }

    override fun getActivityStream(stravaActivity: StravaActivity): Stream? {
        try {
            return doGetActivityStream(stravaActivity, failFastOnRateLimit = false)
        } catch (connectException: ConnectException) {
            throw RuntimeException("Unable to connect to Strava API : ${connectException.message}")
        }
    }

    override fun getActivityStreamFailFastOnRateLimit(stravaActivity: StravaActivity): Stream? {
        try {
            return doGetActivityStream(stravaActivity, failFastOnRateLimit = true)
        } catch (connectException: ConnectException) {
            throw RuntimeException("Unable to connect to Strava API : ${connectException.message}")
        }
    }

    override fun getDetailedActivity(activityId: Long): StravaDetailedActivity? {
        try {
            if (accessToken == null) {
                return null
            }
            return doGetActivity(activityId, failFastOnRateLimit = false)
        } catch (connectException: ConnectException) {
            throw RuntimeException("Unable to connect to Strava API : ${connectException.message}")
        }
    }

    override fun getDetailedActivityFailFastOnRateLimit(activityId: Long): StravaDetailedActivity? {
        try {
            if (accessToken == null) {
                return null
            }
            return doGetActivity(activityId, failFastOnRateLimit = true)
        } catch (connectException: ConnectException) {
            throw RuntimeException("Unable to connect to Strava API : ${connectException.message}")
        }
    }

    private fun getProxyFromEnvironment(): Proxy? {
        var httpsProxy = System.getenv()["https_proxy"]
        if (httpsProxy == null) {
            httpsProxy = System.getenv()["HTTPS_PROXY"]
        }
        if (httpsProxy != null) {
            try {
                val proxyUrl = URI(httpsProxy).toURL()
                logger.info("Set http proxy : $proxyUrl")
                return Proxy(Proxy.Type.HTTP, InetSocketAddress(proxyUrl.host, proxyUrl.port))
            } catch (malformedURLException: MalformedURLException) {
                logger.info("Error configuring proxy : $malformedURLException")
            }
        } else {
            logger.info("No https proxy defined")
        }

        return null
    }

    private fun doGetLoggedInAthlete(): StravaAthlete? {

        val url = "https://www.strava.com/api/v3/athlete"

        val response = executeRequestWithRetry(
            requestBuilder = { Request.Builder().url(url).headers(buildRequestHeaders()).build() },
            operationName = "retrieve logged in athlete",
            maxAttempts = 6,
            failFastOnRateLimit = true,
        ) ?: return null

        response.use {
            if (response.isSuccessful) {
                try {
                    val json = response.body.string()
                    return objectMapper.readValue<StravaAthlete>(json)
                } catch (databindException: DatabindException) {
                    throw RuntimeException("Something was wrong with Strava API", databindException)
                }
            } else {
                throw RuntimeException("Something was wrong with Strava API for url $url : ${response.body.string()}")
            }
        }
    }

    private fun doGetActivities(
        before: LocalDateTime? = null,
        after: LocalDateTime,
        failFastOnRateLimit: Boolean = false
    ): List<StravaActivity> {

        val activities = mutableListOf<StravaActivity>()
        var page = 1
        // Use UTC as reference timezone so that date boundaries are consistent for all users
        var url = "https://www.strava.com/api/v3/athlete/activities?per_page=${properties.pageSize}"
        if (before != null) {
            url += "&before=${before.toEpochSecond(ZoneOffset.UTC)}"
        }
        url += "&after=${after.toEpochSecond(ZoneOffset.UTC)}"

        val requestHeaders = buildRequestHeaders()
        while (true) {
            val requestUrl = "$url&page=$page"
            val response = executeRequestWithRetry(
                requestBuilder = { Request.Builder().url(requestUrl).headers(requestHeaders).build() },
                operationName = "retrieve activities page=$page",
                maxAttempts = 6,
                failFastOnRateLimit = failFastOnRateLimit
            ) ?: break

            val result: List<StravaActivity>
            response.use {
                if (response.code == HttpStatus.UNAUTHORIZED.value()) {
                    throw RuntimeException("Invalid access token (HTTP 401 Unauthorized). Please re-authenticate.")
                }
                if (response.code >= HttpStatus.BAD_REQUEST.value()) {
                    throw RuntimeException("Something was wrong with Strava API for url $requestUrl : ${response.code} - ${response.body.string()}")
                }

                result = objectMapper.readValue(response.body.string())
                activities.addAll(result)
            }

            if (result.isEmpty()) {
                break
            }
            page++
        }

        return activities
    }

    private fun doGetActivityStream(
        stravaActivity: StravaActivity,
        failFastOnRateLimit: Boolean = false,
    ): Stream? {

        // uploadId = 0 => this is a manual stravaActivity without streams
        if (stravaActivity.uploadId == 0L) {
            return null
        }
        val url =
            "https://www.strava.com/api/v3/activities/${stravaActivity.id}/streams" + "?keys=time,distance,latlng,altitude,velocity_smooth,heartrate,cadence,watts,moving,grade_smooth&key_by_type=true"

        val response = executeRequestWithRetry(
            requestBuilder = { Request.Builder().url(url).headers(buildRequestHeaders()).build() },
            operationName = "retrieve stream for activity ${stravaActivity.id}",
            maxAttempts = 4,
            failFastOnRateLimit = failFastOnRateLimit,
        ) ?: return null

        response.use {
            when {
                response.code >= HttpStatus.BAD_REQUEST.value() -> {
                    when (response.code) {
                        HttpStatus.NOT_FOUND.value() -> {
                            logger.warn("Stream not found for stravaActivity : ${stravaActivity.id}")
                            return null
                        }

                        else -> {
                            logger.error("Something was wrong with Strava API for url ${response.request.url} : ${response.code} - ${response.body}")
                            return null
                        }
                    }
                }

                response.code == HttpStatus.OK.value() -> {
                    return try {
                        val json = response.body.string()
                        return objectMapper.readValue<Stream>(json)
                    } catch (databindException: DatabindException) {
                        logger.error("Unable to load streams for stravaActivity : $stravaActivity: ${databindException.message}")
                        null
                    }
                }

                else -> {
                    val errorBody = response.body.string()
                    logger.info("Unable to load streams for stravaActivity : $stravaActivity")
                    throw RuntimeException("Something was wrong with Strava API for url $url : ${response.code} - $errorBody")
                }
            }
        }
    }

    private fun doGetActivity(
        activityId: Long,
        failFastOnRateLimit: Boolean = false,
    ): StravaDetailedActivity? {
        val url = "https://www.strava.com/api/v3/activities/$activityId?include_all_efforts=true"

        val response = executeRequestWithRetry(
            requestBuilder = { Request.Builder().url(url).headers(buildRequestHeaders()).build() },
            operationName = "retrieve detailed activity $activityId",
            maxAttempts = 6,
            failFastOnRateLimit = failFastOnRateLimit,
        ) ?: return null

        response.use {
            when {
                response.code >= HttpStatus.BAD_REQUEST.value() -> {
                    when (response.code) {
                        HttpStatus.NOT_FOUND.value() -> {
                            logger.warn("StravaActivity $activityId not found")
                            return null
                        }

                        else -> {
                            logger.error("Something was wrong with Strava API while getting stravaActivity ${response.request.url} : ${response.code} - ${response.body}")
                            return null
                        }
                    }
                }

                response.code == HttpStatus.OK.value() -> {
                    return try {
                        val json = response.body.string()
                        objectMapper.readValue<StravaDetailedActivity>(json)
                    } catch (databindException: DatabindException) {
                        logger.info("Unable to load stravaActivity : $activityId - ${databindException.message}")
                        null
                    }
                }

                else -> {
                    val errorBody = response.body.string()
                    logger.info("Unable to load stravaActivity : $activityId")
                    throw RuntimeException("Something was wrong with Strava API for url $url : ${response.code} - $errorBody")
                }
            }
        }
    }

    private fun setAccessToken(clientId: String, clientSecret: String) {
        if (usePersistedTokenIfAvailable(clientId, clientSecret)) {
            logger.info("Reused persisted Strava OAuth token")
            return
        }

        val redirectPort = 8090
        val redirectUri = "http://localhost:$redirectPort/exchange_token"
        val state = UUID.randomUUID().toString()
        val url =
            "${properties.url}/oauth/authorize?client_id=${encodeQueryParam(clientId)}&response_type=code" +
                "&redirect_uri=${encodeQueryParam(redirectUri)}&approval_prompt=auto" +
                "&scope=${encodeQueryParam("read_all,activity:read_all,profile:read_all")}" +
                "&state=${encodeQueryParam(state)}"
        openBrowser(url)

        println()
        println("To grant MyStravaStats to read your Strava activities data: copy paste this URL in a browser")
        println(url)
        println()

        logger.info("Waiting for your agreement to allow MyStravaStats to access to your Strava data ...")
        val authorizationCode = receiveOAuthCallback(redirectPort, state)
        logger.info("Access granted - exchanging authorization code for access token.")
        val tokenPayload = getAccessToken(clientId, clientSecret, authorizationCode)
        persistToken(tokenPayload)
        setAccessToken(tokenPayload.accessTokenOrThrow())
    }

    /**
     * Opens a temporary [ServerSocket] on [port], accepts a single HTTP request from the
     * Strava OAuth redirect, extracts the "code" query parameter, returns a confirmation
     * HTML page to the browser, then closes the socket.
     *
     * This replaces the previously embedded Ktor server and requires no additional
     * HTTP-server dependency.
     */
    private fun receiveOAuthCallback(port: Int, expectedState: String): String {
        // 5-minute timeout: if the user does not authorise in time, we fail loudly.
        ServerSocket(port, 1, InetAddress.getLoopbackAddress()).use { serverSocket ->
            serverSocket.soTimeout = 5 * 60 * 1_000
            serverSocket.accept().use { socket ->
                // Only the first line of the HTTP request is needed:
                // "GET /exchange_token?state=&code=<CODE>&scope=... HTTP/1.1"
                val requestLine = socket.getInputStream().bufferedReader().readLine()
                    ?: throw RuntimeException("Empty HTTP request received on OAuth callback port $port")

                val requestTarget = requestLine.substringAfter("GET ", "").substringBefore(" ")
                val query = URI(requestTarget).rawQuery.orEmpty()
                val params = parseQueryParams(query)

                if (params["state"] != expectedState) {
                    throw RuntimeException("OAuth state mismatch. Please retry Strava authorization.")
                }
                params["error"]?.takeIf { it.isNotBlank() }?.let { error ->
                    throw RuntimeException("Strava OAuth failed: $error")
                }

                val code = params["code"].orEmpty()

                if (code.isBlank()) {
                    throw RuntimeException("No authorization code in OAuth callback. Request line: $requestLine")
                }

                // Send a minimal HTTP 200 response with the confirmation page.
                val htmlBytes = buildResponseHtml().toByteArray(Charsets.UTF_8)
                socket.getOutputStream().apply {
                    write(
                        ("HTTP/1.1 200 OK\r\n" +
                            "Content-Type: text/html; charset=UTF-8\r\n" +
                            "Content-Length: ${htmlBytes.size}\r\n" +
                            "Connection: close\r\n" +
                            "\r\n").toByteArray(Charsets.US_ASCII)
                    )
                    write(htmlBytes)
                    flush()
                }

                return code
            }
        }
    }

    private fun buildResponseHtml(): String = """
        <!DOCTYPE html>
        <html lang="en">
        <head>
            <meta charset="UTF-8">
            <meta name="viewport" content="width=device-width, initial-scale=1.0">
            <title>Access Granted</title>
            <style>
                body {
                    font-family: Arial, sans-serif;
                    background-color: #f4f4f4;
                    display: flex;
                    justify-content: center;
                    align-items: center;
                    height: 100vh;
                    margin: 0;
                }
                .container {
                    background-color: #fff;
                    padding: 20px;
                    border-radius: 8px;
                    box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
                    text-align: center;
                }
                .custom-class {
                    color: #007bff;
                    font-weight: bold;
                }
                h1 {
                    color: #333;
                }
                p {
                    color: #666;
                }
            </style>
        </head>
        <body>
            <div class="container">
                <h1>Access Granted</h1>
                <p class="custom-class">Access granted to MyStravaStats.</p>
                <p>You can now close this window.</p>
            </div>
        </body>
        </html>
    """.trimIndent()

    private fun getAccessToken(clientId: String, clientSecret: String, authorizationCode: String): Map<String, Any?> {
        return requestToken(
            mapOf(
                "client_id" to clientId,
                "client_secret" to clientSecret,
                "code" to authorizationCode,
                "grant_type" to "authorization_code",
            )
        )
    }

    private fun refreshAccessToken(clientId: String, clientSecret: String, refreshToken: String): Map<String, Any?> {
        return requestToken(
            mapOf(
                "client_id" to clientId,
                "client_secret" to clientSecret,
                "grant_type" to "refresh_token",
                "refresh_token" to refreshToken,
            )
        )
    }

    private fun requestToken(payloadValues: Map<String, String>): Map<String, Any?> {
        val url = "${properties.url}/api/v3/oauth/token"

        val payload = payloadValues.entries.joinToString("&") { (key, value) ->
            "${encodeQueryParam(key)}=${encodeQueryParam(value)}"
        }
        val body = payload.toRequestBody("application/x-www-form-urlencoded".toMediaType())

        var lastException: Exception? = null
        var backoffMs = 1_000L
        val maxAttempts = 3

        for (attempt in 1..maxAttempts) {
            try {
                val request: Request = Request.Builder().url(url).post(body).build()

                okHttpClient.newCall(request).execute().use { response ->
                    val responseBody = response.body.string()
                    try {
                        if (response.code == 200) {
                            val tokenPayload: Map<String, Any?> = objectMapper.readValue(responseBody)
                            tokenPayload.accessTokenOrThrow()
                            return tokenPayload
                        } else {
                            throw RuntimeException("Something was wrong with Strava API for url $url (status=${response.code})")
                        }
                    } catch (ex: Exception) {
                        logger.error("Something was wrong with Strava API for url $url. ${ex.cause?.message ?: ex.message}")
                        throw RuntimeException("Something was wrong with Strava API for url $url. ${ex.cause?.message ?: ex.message}")
                    }
                }
            } catch (ex: Exception) {
                lastException = ex
                if (attempt < maxAttempts) {
                    logger.warn("Token request failed (attempt $attempt/$maxAttempts): ${ex.message}, retrying in ${backoffMs}ms")
                    runBlocking { delay(backoffMs.milliseconds) }
                    backoffMs *= 2
                }
            }
        }

        throw RuntimeException("Failed to get token after $maxAttempts attempts: ${lastException?.message}", lastException)
    }

    private fun usePersistedTokenIfAvailable(clientId: String, clientSecret: String): Boolean {
        val file = tokenFile ?: return false
        if (!file.exists()) {
            return false
        }

        return try {
            val tokenPayload: Map<String, Any?> = objectMapper.readValue(file)
            val accessToken = tokenPayload["access_token"]?.toString().orEmpty()
            val expiresAt = tokenPayload["expires_at"].asLong()
            val now = Instant.now().epochSecond

            if (accessToken.isNotBlank() && expiresAt > now + TOKEN_REFRESH_BUFFER_SECONDS) {
                setAccessToken(accessToken)
                true
            } else {
                val refreshToken = tokenPayload["refresh_token"]?.toString().orEmpty()
                if (refreshToken.isBlank()) {
                    false
                } else {
                    val refreshedToken = refreshAccessToken(clientId, clientSecret, refreshToken)
                    persistToken(refreshedToken)
                    setAccessToken(refreshedToken.accessTokenOrThrow())
                    true
                }
            }
        } catch (exception: Exception) {
            logger.warn("Unable to use persisted Strava OAuth token: ${exception.message}")
            false
        }
    }

    private fun persistToken(tokenPayload: Map<String, Any?>) {
        val file = tokenFile ?: return
        try {
            file.parentFile?.mkdirs()
            val persisted = tokenPayload + ("created_at" to Instant.now().toString())
            objectMapper.writeValue(file, persisted)
            file.setReadable(false, false)
            file.setReadable(true, true)
            file.setWritable(false, false)
            file.setWritable(true, true)
            file.setExecutable(false, false)
        } catch (exception: Exception) {
            throw RuntimeException("Unable to persist Strava OAuth token to ${file.absolutePath}", exception)
        }
    }

    private fun Map<String, Any?>.accessTokenOrThrow(): String {
        val accessToken = this["access_token"]?.toString().orEmpty()
        if (accessToken.isBlank()) {
            throw RuntimeException("Missing access_token in Strava response")
        }
        return accessToken
    }

    private fun Any?.asLong(): Long {
        return when (this) {
            is Number -> this.toDouble().roundToLong()
            is String -> this.toLongOrNull() ?: 0L
            else -> 0L
        }
    }

    private fun parseQueryParams(query: String): Map<String, String> {
        if (query.isBlank()) {
            return emptyMap()
        }
        return query.split("&")
            .filter { it.isNotBlank() }
            .mapNotNull { pair ->
                val key = pair.substringBefore("=", "")
                if (key.isBlank()) return@mapNotNull null
                val value = pair.substringAfter("=", "")
                URLDecoder.decode(key, StandardCharsets.UTF_8) to URLDecoder.decode(value, StandardCharsets.UTF_8)
            }
            .toMap()
    }

    private fun encodeQueryParam(value: String): String {
        return URLEncoder.encode(value, StandardCharsets.UTF_8)
    }

    private fun buildRequestHeaders() =
        Headers.Builder().set("Accept", "application/json").set("Content-Type", "application/json")
            .set("Authorization", "Bearer $accessToken").build()

    private fun executeRequestWithRetry(
        requestBuilder: () -> Request,
        operationName: String,
        maxAttempts: Int,
        failFastOnRateLimit: Boolean = false,
    ): okhttp3.Response? {
        var attempt = 1
        var backoffMs = 1_000L
        val maxBackoffMs = 30_000L

        while (attempt <= maxAttempts) {
            if (!waitForGlobalRateLimitWindow(operationName, failFastOnRateLimit)) {
                if (failFastOnRateLimit) {
                    throw RateLimitExceededException(
                        "strava rate limit reached (cooldown active) during '$operationName'"
                    )
                }
                return null
            }

            val request = requestBuilder()
            val response = okHttpClient.newCall(request).execute()
            if (response.code != HttpStatus.TOO_MANY_REQUESTS.value()) {
                return response
            }

            val retryAfterDelayMs = parseRetryAfterMillis(response.header("Retry-After"), backoffMs)
            val headerBasedDelayMs = computeDelayFromRateLimitHeaders(response)
            val retryDelayMs = maxOf(retryAfterDelayMs, headerBasedDelayMs ?: 0L)
            response.close()
            pushGlobalRateLimit(retryDelayMs, "429 during '$operationName'")

            if (failFastOnRateLimit) {
                throw RateLimitExceededException("strava rate limit reached (429) during '$operationName'")
            }

            if (retryDelayMs > MAX_BLOCKING_WAIT_MS) {
                logger.warn(
                    "Skipping '{}' retries because Strava cooldown is {} ms (> {} ms).",
                    operationName,
                    retryDelayMs,
                    MAX_BLOCKING_WAIT_MS
                )
                return null
            }

            if (attempt == maxAttempts) {
                pushGlobalRateLimit(maxOf(retryDelayMs, RATE_LIMIT_EXHAUSTED_COOLDOWN_MS), "rate limit retries exhausted")
                logger.error(
                    "Unable to complete '{}' after {} attempts due to 429. {} Cooldown={} ms",
                    operationName,
                    maxAttempts,
                    QUOTA_EXCEED_LIMIT,
                    retryDelayMs
                )
                return null
            }

            val sleepMs = addJitter(retryDelayMs)
            logger.warn(
                "Strava API rate limit (429) during '{}', retry in {} ms (attempt {}/{})",
                operationName,
                sleepMs,
                attempt,
                maxAttempts
            )
            runBlocking { delay(sleepMs.milliseconds) }

            backoffMs = (backoffMs * 2).coerceAtMost(maxBackoffMs)
            attempt++
        }

        return null
    }

    private fun parseRetryAfterMillis(retryAfterHeader: String?, fallbackMs: Long): Long {
        if (retryAfterHeader.isNullOrBlank()) {
            return fallbackMs
        }

        retryAfterHeader.toLongOrNull()?.let { seconds ->
            if (seconds > 0) {
                return (seconds * 1_000L).coerceAtMost(MAX_RETRY_AFTER_MS)
            }
        }

        return try {
            val retryAt = ZonedDateTime.parse(retryAfterHeader, DateTimeFormatter.RFC_1123_DATE_TIME).toInstant()
            val waitMs = retryAt.toEpochMilli() - System.currentTimeMillis()
            if (waitMs > 0) waitMs.coerceAtMost(MAX_RETRY_AFTER_MS) else fallbackMs
        } catch (_: Exception) {
            fallbackMs
        }
    }

    private fun computeDelayFromRateLimitHeaders(response: okhttp3.Response): Long? {
        val delays = mutableListOf<Long>()

        collectWindowDelay(
            response = response,
            limitHeader = "X-RateLimit-Limit",
            usageHeader = "X-RateLimit-Usage",
            delays = delays
        )
        collectWindowDelay(
            response = response,
            limitHeader = "X-ReadRateLimit-Limit",
            usageHeader = "X-ReadRateLimit-Usage",
            delays = delays
        )

        return delays.maxOrNull()
    }

    private fun collectWindowDelay(
        response: okhttp3.Response,
        limitHeader: String,
        usageHeader: String,
        delays: MutableList<Long>,
    ) {
        val limits = parseRateLimitTuple(response.header(limitHeader)) ?: return
        val usage = parseRateLimitTuple(response.header(usageHeader)) ?: return
        val nowMs = System.currentTimeMillis()

        if (usage.second >= limits.second) {
            delays.add(millisUntilNextUtcMidnight(nowMs))
        }
        if (usage.first >= limits.first) {
            delays.add(millisUntilNextQuarterHourUtc(nowMs))
        }
    }

    private fun parseRateLimitTuple(value: String?): Pair<Int, Int>? {
        if (value.isNullOrBlank()) {
            return null
        }

        val tokens = value.split(",")
            .map { it.trim() }
            .filter { it.isNotEmpty() }
        if (tokens.size != 2) {
            return null
        }

        val shortTerm = tokens[0].toIntOrNull() ?: return null
        val daily = tokens[1].toIntOrNull() ?: return null
        return shortTerm to daily
    }

    private fun millisUntilNextQuarterHourUtc(nowMs: Long): Long {
        val now = ZonedDateTime.ofInstant(Instant.ofEpochMilli(nowMs), ZoneOffset.UTC)
        val quarterStartMinute = (now.minute / 15) * 15
        val nextQuarter = now
            .withMinute(quarterStartMinute)
            .withSecond(0)
            .withNano(0)
            .plusMinutes(15)

        return (nextQuarter.toInstant().toEpochMilli() - nowMs + RATE_LIMIT_WINDOW_BUFFER_MS)
            .coerceAtLeast(RATE_LIMIT_WINDOW_BUFFER_MS)
    }

    private fun millisUntilNextUtcMidnight(nowMs: Long): Long {
        val now = ZonedDateTime.ofInstant(Instant.ofEpochMilli(nowMs), ZoneOffset.UTC)
        val nextMidnight = now.toLocalDate().plusDays(1).atStartOfDay(ZoneOffset.UTC)
        return (nextMidnight.toInstant().toEpochMilli() - nowMs + RATE_LIMIT_WINDOW_BUFFER_MS)
            .coerceAtLeast(RATE_LIMIT_WINDOW_BUFFER_MS)
    }

    private fun pushGlobalRateLimit(delayMs: Long, reason: String) {
        if (delayMs <= 0) {
            return
        }

        val newDeadline = System.currentTimeMillis() + delayMs
        while (true) {
            val current = globalRateLimitUntilMs.get()
            if (newDeadline <= current) {
                return
            }
            if (globalRateLimitUntilMs.compareAndSet(current, newDeadline)) {
                logger.warn("Applying Strava global cooldown of {} ms ({})", delayMs, reason)
                return
            }
        }
    }

    private fun waitForGlobalRateLimitWindow(
        operationName: String,
        failFastOnRateLimit: Boolean,
    ): Boolean {
        val waitMs = globalRateLimitUntilMs.get() - System.currentTimeMillis()
        if (waitMs <= 0) {
            return true
        }
        if (failFastOnRateLimit) {
            logger.warn(
                "Skipping '{}' immediately because Strava cooldown is active for {} ms (fail-fast mode)",
                operationName,
                waitMs
            )
            return false
        }
        if (waitMs > MAX_BLOCKING_WAIT_MS) {
            logger.warn(
                "Skipping '{}' because Strava cooldown is still active for {} ms",
                operationName,
                waitMs
            )
            return false
        }

        logger.info(
            "Waiting {} ms before '{}' because Strava cooldown is active",
            waitMs,
            operationName
        )
        runBlocking { delay(waitMs.milliseconds) }
        return true
    }

    private fun addJitter(delayMs: Long): Long {
        if (delayMs <= 0) {
            return delayMs
        }
        val jitter = ThreadLocalRandom.current().nextLong(0, RETRY_JITTER_MS + 1)
        return (delayMs + jitter).coerceAtMost(MAX_RETRY_AFTER_MS)
    }
}
