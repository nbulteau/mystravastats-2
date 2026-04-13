package me.nicolas.stravastats.adapters.strava


import io.ktor.http.*
import io.ktor.server.engine.*
import io.ktor.server.netty.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.stream.Stream
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
import java.net.*
import java.time.Instant
import java.time.LocalDateTime
import java.time.ZoneOffset
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter
import java.util.*
import java.util.concurrent.ThreadLocalRandom
import java.util.concurrent.atomic.AtomicLong

internal class StravaRateLimitException(message: String) : RuntimeException(message)

internal class StravaApi(clientId: String, clientSecret: String) : IStravaApi {

    companion object {
        private const val QUOTA_EXCEED_LIMIT =
            "Quotas exceeded: Strava rate limitations (100 requests every 15 minutes, with up to 1,000 requests per day)"
        private const val MAX_RETRY_AFTER_MS = 120_000L
        private const val RETRY_JITTER_MS = 250L
        private const val RATE_LIMIT_EXHAUSTED_COOLDOWN_MS = 60_000L
        private const val RATE_LIMIT_WINDOW_BUFFER_MS = 1_000L
        private const val MAX_BLOCKING_WAIT_MS = 30_000L
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

    private fun setAccessToken(accessToken: String) {
        this.accessToken = accessToken
    }

    init {
        setAccessToken(clientId, clientSecret)
    }

    override fun retrieveLoggedInAthlete(): Optional<StravaAthlete> {
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
            return doGetActivityStream(stravaActivity)
        } catch (connectException: ConnectException) {
            throw RuntimeException("Unable to connect to Strava API : ${connectException.message}")
        }
    }

    override fun getDetailedActivity(activityId: Long): Optional<StravaDetailedActivity> {
        try {
            if (accessToken == null) {
                return Optional.empty()
            }
            return doGetActivity(activityId)
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

    private fun doGetLoggedInAthlete(): Optional<StravaAthlete> {

        val url = "https://www.strava.com/api/v3/athlete"

        val response = executeRequestWithRetry(
            requestBuilder = { Request.Builder().url(url).headers(buildRequestHeaders()).build() },
            operationName = "retrieve logged in athlete",
            maxAttempts = 6
        ) ?: return Optional.empty()

        response.use {
            if (response.isSuccessful) {
                try {
                    val json = response.body.string()
                    return Optional.of(objectMapper.readValue<StravaAthlete>(json))
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
                    throw RuntimeException("Invalid accessToken : $accessToken")
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

    private fun doGetActivityStream(stravaActivity: StravaActivity): Stream? {

        // uploadId = 0 => this is a manual stravaActivity without streams
        if (stravaActivity.uploadId == 0L) {
            return null
        }
        val url =
            "https://www.strava.com/api/v3/activities/${stravaActivity.id}/streams" + "?keys=time,distance,latlng,altitude,velocity_smooth,heartrate,cadence,watts,moving,grade_smooth&key_by_type=true"

        val response = executeRequestWithRetry(
            requestBuilder = { Request.Builder().url(url).headers(buildRequestHeaders()).build() },
            operationName = "retrieve stream for activity ${stravaActivity.id}",
            maxAttempts = 4
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
                    logger.info("Unable to load streams for stravaActivity : $stravaActivity")
                    throw RuntimeException("Something was wrong with Strava API for url $url : ${response.code} - ${response.code}")
                }
            }
        }
    }

    private fun doGetActivity(activityId: Long): Optional<StravaDetailedActivity> {
        val url = "https://www.strava.com/api/v3/activities/$activityId?include_all_efforts=true"

        val response = executeRequestWithRetry(
            requestBuilder = { Request.Builder().url(url).headers(buildRequestHeaders()).build() },
            operationName = "retrieve detailed activity $activityId",
            maxAttempts = 6
        ) ?: return Optional.empty()

        response.use {
            when {
                response.code >= HttpStatus.BAD_REQUEST.value() -> {
                    when (response.code) {
                        HttpStatus.NOT_FOUND.value() -> {
                            logger.warn("StravaActivity $activityId not found")
                            return Optional.empty()
                        }

                        else -> {
                            logger.error("Something was wrong with Strava API while getting stravaActivity ${response.request.url} : ${response.code} - ${response.body}")
                            return Optional.empty()
                        }
                    }
                }

                response.code == HttpStatus.OK.value() -> {
                    return try {
                        val json = response.body.string()
                        return Optional.of(objectMapper.readValue<StravaDetailedActivity>(json))
                    } catch (databindException: DatabindException) {
                        logger.info("Unable to load stravaActivity : $activityId - ${databindException.message}")
                        Optional.empty()
                    }
                }

                else -> {
                    logger.info("Unable to load stravaActivity : $activityId")
                    throw RuntimeException("Something was wrong with Strava API for url $url : ${response.code} - ${response.code}")
                }
            }
        }
    }

    private fun setAccessToken(clientId: String, clientSecret: String) {
        val url =
            "https://www.strava.com/api/v3/oauth/authorize?client_id=$clientId&response_type=code&redirect_uri=http://localhost:8090/exchange_token&approval_prompt=auto&scope=read_all,activity:read_all,profile:read_all"
        openBrowser(url)

        println()
        println("To grant MyStravaStats to read your Strava activities data: copy paste this URL in a browser")
        println(url)
        println()

        runBlocking {
            val channel = Channel<String>()

            val embeddedServer = embeddedServer(Netty, 8090) {
                routing {
                    get("/exchange_token") {
                        val authorizationCode = call.request.queryParameters["code"] ?: "no authorization code"
                        call.respondText(
                            buildResponseHtml(clientId),
                            ContentType.Text.Html
                        )
                        launch {
                            // Get an authorization token with the code
                            val accessToken = getAccessToken(clientId, clientSecret, authorizationCode)
                            channel.send(accessToken)

                        }
                    }
                }
            }.start(wait = false)

            logger.info("Waiting for your agreement to allow MyStravaStats to access to your Strava data ...")
            val accessTokenFromToken = channel.receive()
            logger.info("Access granted.")
            setAccessToken(accessTokenFromToken)

            // stop de web server
            logger.info("Stopping the web server ...")
            embeddedServer.stop(1000, 1000)
        }
    }

    private fun buildResponseHtml(clientId: String): String = """
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
                <p class="custom-class">Access granted to read activities of clientId: $clientId.</p>
                <p>You can now close this window.</p>
            </div>
        </body>
        </html>
    """.trimIndent()

    private fun getAccessToken(clientId: String, clientSecret: String, authorizationCode: String): String {

        val url = "${properties.url}/api/v3/oauth/token"

        val payload = mapOf(
            "client_id" to clientId,
            "client_secret" to clientSecret,
            "code" to authorizationCode,
            "grant_type" to "authorization_code"
        )
        val body = objectMapper.writeValueAsString(payload).toRequestBody("application/json".toMediaType())
        val request: Request = Request.Builder().url(url).post(body).build()

        okHttpClient.newCall(request).execute().use { response ->
            val responseBody = response.body.string()
            try {
                if (response.code == 200) {
                    val tokenPayload: Map<String, Any?> = objectMapper.readValue(responseBody)
                    val accessToken = tokenPayload["access_token"]?.toString().orEmpty()
                    if (accessToken.isBlank()) {
                        throw RuntimeException("Missing access_token in Strava response")
                    }
                    return accessToken
                } else {
                    throw RuntimeException("Something was wrong with Strava API for url $url (status=${response.code})")
                }
            } catch (ex: Exception) {
                logger.error("Something was wrong with Strava API for url $url. ${ex.cause?.message ?: ex.message}")
                throw RuntimeException("Something was wrong with Strava API for url $url. ${ex.cause?.message ?: ex.message}")
            }
        }
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
            if (!waitForGlobalRateLimitWindow(operationName)) {
                if (failFastOnRateLimit) {
                    throw StravaRateLimitException(
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
                throw StravaRateLimitException("strava rate limit reached (429) during '$operationName'")
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
            Thread.sleep(sleepMs)

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

    private fun waitForGlobalRateLimitWindow(operationName: String): Boolean {
        val waitMs = globalRateLimitUntilMs.get() - System.currentTimeMillis()
        if (waitMs <= 0) {
            return true
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
        Thread.sleep(waitMs)
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
