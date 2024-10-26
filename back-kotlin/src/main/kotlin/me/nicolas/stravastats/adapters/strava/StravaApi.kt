package me.nicolas.stravastats.adapters.strava

import com.fasterxml.jackson.core.JsonProcessingException
import com.fasterxml.jackson.databind.JsonMappingException
import com.fasterxml.jackson.module.kotlin.jacksonObjectMapper
import com.fasterxml.jackson.module.kotlin.readValue
import io.ktor.http.*
import io.ktor.server.engine.*
import io.ktor.server.netty.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking
import me.nicolas.stravastats.adapters.strava.business.Token
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.interfaces.IStravaApi
import okhttp3.Headers
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.slf4j.Logger
import org.slf4j.LoggerFactory
import org.springframework.http.HttpStatus
import java.awt.Desktop
import java.net.*
import java.time.LocalDateTime
import java.time.ZoneId
import java.util.*
import kotlin.system.exitProcess


internal class StravaApi(clientId: String, clientSecret: String) : IStravaApi {

    companion object {
        private const val QUOTA_EXCEED_LIMIT =
            "Quotas exceeded: Strava rate limitations (100 requests every 15 minutes, with up to 1,000 requests per day)"
    }

    private val logger: Logger = LoggerFactory.getLogger(StravaApi::class.java)

    private val objectMapper = jacksonObjectMapper()

    private val properties: StravaProperties = StravaProperties()

    private val okHttpClient: OkHttpClient = OkHttpClient.Builder().proxy(getupProxyFromEnvironment()).build()

    private var accessToken: String? = null

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
                before = LocalDateTime.of(year, 12, 31, 23, 59), after = LocalDateTime.of(year, 1, 1, 0, 0)
            )
        } catch (connectException: ConnectException) {
            throw RuntimeException("Unable to connect to Strava API : ${connectException.message}")
        }
    }

    override fun getActivityStream(stravaActivity: StravaActivity): Optional<Stream> {
        try {
            return doGetActivityStream(stravaActivity)
        } catch (connectException: ConnectException) {
            throw RuntimeException("Unable to connect to Strava API : ${connectException.message}")
        }
    }

    override fun getActivities(after: LocalDateTime): List<StravaActivity> {
        try {
            return doGetActivities(after = after)
        } catch (connectException: ConnectException) {
            throw RuntimeException("Unable to connect to Strava API : ${connectException.message}")
        }
    }

    override fun getDetailedActivity(activityId: Long): Optional<StravaDetailedActivity> {
        try {
            return doGetActivity(activityId)
        } catch (connectException: ConnectException) {
            throw RuntimeException("Unable to connect to Strava API : ${connectException.message}")
        }
    }

    private fun getupProxyFromEnvironment(): Proxy? {
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
                logger.info("Error configuring proxy : malformedURLException")
            }
        } else {
            logger.info("No https proxy defined")
        }

        return null
    }

    private fun doGetLoggedInAthlete(): Optional<StravaAthlete> {

        val url = "https://www.strava.com/api/v3/athlete"

        val request = Request.Builder().url(url).headers(buildRequestHeaders()).build()

        okHttpClient.newCall(request).execute().use { response ->
            if (response.isSuccessful) {
                try {
                    val json = response.body?.string()
                    return if (json != null) {
                        Optional.of(objectMapper.readValue(json, StravaAthlete::class.java))
                    } else {
                        Optional.empty()
                    }
                } catch (jsonMappingException: JsonMappingException) {
                    throw RuntimeException("Something was wrong with Strava API", jsonMappingException)
                }
            } else {
                throw RuntimeException("Something was wrong with Strava API for url $url : ${response.body}")
            }
        }
    }

    private fun doGetActivities(before: LocalDateTime? = null, after: LocalDateTime): List<StravaActivity> {

        val activities = mutableListOf<StravaActivity>()
        var page = 1
        var url = "https://www.strava.com/api/v3/athlete/activities?per_page=${properties.pagesize}"
        if (before != null) {
            url += "&before=${before.atZone(ZoneId.of("Europe/Paris")).toEpochSecond()}"
        }
        url += "&after=${after.atZone(ZoneId.of("Europe/Paris")).toEpochSecond()}"

        val requestHeaders = buildRequestHeaders()
        do {
            val request = Request.Builder().url("$url&page=${page++}").headers(requestHeaders).build()

            val result: List<StravaActivity>
            okHttpClient.newCall(request).execute().use { response ->
                if (response.code == 401) {
                    logger.info("Invalid accessToken : $accessToken")
                    exitProcess(-1)
                }
                if (response.code == 429) {
                    logger.info(QUOTA_EXCEED_LIMIT)
                    exitProcess(-1)
                }
                result = objectMapper.readValue(response.body?.string() ?: "")

                activities.addAll(result)
            }
        } while (result.isNotEmpty())

        return activities
    }

    private fun doGetActivityStream(stravaActivity: StravaActivity): Optional<Stream> {

        // uploadId = 0 => this is a manual stravaActivity without streams
        if (stravaActivity.uploadId == 0L) {
            return Optional.empty()
        }
        val url =
            "https://www.strava.com/api/v3/activities/${stravaActivity.id}/streams" + "?keys=time,distance,latlng,altitude,velocity_smooth,heartrate,cadence,watts,moving,grade_smooth&key_by_type=true"

        val request: Request = Request.Builder().url(url).headers(buildRequestHeaders()).build()

        okHttpClient.newCall(request).execute().use { response ->
            when {
                response.code >= HttpStatus.BAD_REQUEST.value() -> {
                    logger.info("Unable to load streams for stravaActivity : ${stravaActivity.id}")
                    when (response.code) {
                        HttpStatus.TOO_MANY_REQUESTS.value() -> {
                            logger.info(QUOTA_EXCEED_LIMIT)
                            return Optional.empty()
                        }

                        else -> {
                            logger.info("Something was wrong with Strava API for url ${response.request.url} : ${response.code} - ${response.body}")
                            return Optional.empty()
                        }
                    }
                }

                response.code == HttpStatus.OK.value() -> {
                    return try {
                        val json = response.body?.string()
                        return if (json != null) {
                            Optional.of(objectMapper.readValue(json, Stream::class.java))
                        } else {
                            Optional.empty()
                        }
                    } catch (jsonProcessingException: JsonProcessingException) {
                        logger.info("Unable to load streams for stravaActivity : $stravaActivity")
                        Optional.empty()
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

        val request: Request = Request.Builder().url(url).headers(buildRequestHeaders()).build()

        okHttpClient.newCall(request).execute().use { response ->
            when {
                response.code >= HttpStatus.BAD_REQUEST.value() -> {
                    logger.info("Unable to load stravaActivity : $activityId")
                    when (response.code) {
                        HttpStatus.TOO_MANY_REQUESTS.value() -> {
                            logger.info(QUOTA_EXCEED_LIMIT)
                            return Optional.empty()
                        }

                        HttpStatus.NOT_FOUND.value() -> {
                            logger.warn("StravaActivity $activityId not found")
                            return Optional.empty()
                        }

                        else -> {
                            logger.info("Something was wrong with Strava API while getting stravaActivity ${response.request.url} : ${response.code} - ${response.body}")
                            return Optional.empty()
                        }
                    }
                }

                response.code == HttpStatus.OK.value() -> {
                    return try {
                        val json = response.body?.string()
                        return if (json != null) {
                            Optional.of(objectMapper.readValue(json, StravaDetailedActivity::class.java))
                        } else {
                            Optional.empty()
                        }
                    } catch (jsonProcessingException: JsonProcessingException) {
                        logger.info("Unable to load stravaActivity : $activityId - ${jsonProcessingException.message}")
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
                            "Access granted to read activities of clientId: $clientId.",
                            ContentType.Text.Html
                        )
                        launch {
                            // Get authorisation token with the code
                            val token = getToken(clientId, clientSecret, authorizationCode)
                            channel.send(token.accessToken)

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

    private fun getToken(clientId: String, clientSecret: String, authorizationCode: String): Token {

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
            try {
                if (response.code == 200) {
                    return objectMapper.readValue(response.body?.string() ?: "", Token::class.java)
                } else {
                    throw RuntimeException("Something was wrong with Strava API for url $url")
                }
            } catch (ex: Exception) {
                logger.error("Something was wrong with Strava API for url $url. ${ex.cause?.message ?: ex.message}")
                throw RuntimeException("Something was wrong with Strava API for url $url. ${ex.cause?.message ?: ex.message}")
            }
        }
    }

    private fun buildRequestHeaders() =
        Headers.Builder().set("Accept", "application/json").set("ContentType", "application/json")
            .set("Authorization", "Bearer $accessToken").build()

    private fun openBrowser(url: String) {
        try {
            if (Desktop.isDesktopSupported()) {
                val desktop = Desktop.getDesktop()
                if (desktop.isSupported(Desktop.Action.BROWSE)) {
                    desktop.browse(URI(url))
                    return
                }
            }
            // Fallback to using Runtime exec
            val os = System.getProperty("os.name").lowercase(Locale.getDefault())
            when {
                os.contains("win") -> Runtime.getRuntime().exec(arrayOf("rundll32", "url.dll,FileProtocolHandler", url))
                os.contains("mac") -> Runtime.getRuntime().exec(arrayOf("open", url))
                os.contains("nix") || os.contains("nux") -> {
                    val process = Runtime.getRuntime().exec(arrayOf("xdg-open", url))
                    val exitCode = process.waitFor()
                    if (exitCode != 0) {
                        println("Failed to open the browser with xdg-open. Exit code: $exitCode")
                    }
                }

                else -> println("Unsupported operating system. Cannot open the browser.")
            }
        } catch (e: Exception) {
            println("Failed to open the browser: ${e.message}")
        }
    }
}
