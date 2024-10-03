package me.nicolas.stravastats.adapter.strava

import com.fasterxml.jackson.core.JsonProcessingException
import com.fasterxml.jackson.databind.JsonMappingException
import com.fasterxml.jackson.module.kotlin.jacksonObjectMapper
import com.fasterxml.jackson.module.kotlin.readValue
import io.ktor.http.*
import io.ktor.server.application.*
import io.ktor.server.engine.*
import io.ktor.server.netty.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking
import me.nicolas.stravastats.adapter.strava.business.Token
import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.business.strava.Athlete
import me.nicolas.stravastats.domain.business.strava.DetailledActivity
import me.nicolas.stravastats.domain.business.strava.Stream
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


internal class StravaApi(clientId: String, clientSecret: String, private val properties: StravaProperties) :
    IStravaApi {

    private val logger: Logger = LoggerFactory.getLogger(StravaApi::class.java)

    private val objectMapper = jacksonObjectMapper()

    private val okHttpClient: OkHttpClient = OkHttpClient.Builder().proxy(getupProxyFromEnvironment()).build()

    private var accessToken: String? = null

    private fun setAccessToken(accessToken: String) {
        this.accessToken = accessToken
    }

    init {
        setAccessToken(clientId, clientSecret)
    }

    override fun retrieveLoggedInAthlete(): Optional<Athlete> {
        try {
            return doGetLoggedInAthlete()
        } catch (connectException: ConnectException) {
            throw RuntimeException("Unable to connect to Strava API : ${connectException.message}")
        }
    }

    override fun getActivities(year: Int): List<Activity> {
        try {
            return doGetActivities(
                before = LocalDateTime.of(year, 12, 31, 23, 59), after = LocalDateTime.of(year, 1, 1, 0, 0)
            )
        } catch (connectException: ConnectException) {
            throw RuntimeException("Unable to connect to Strava API : ${connectException.message}")
        }
    }

    override fun getActivityStream(activity: Activity): Optional<Stream> {
        try {
            return doGetActivityStream(activity)
        } catch (connectException: ConnectException) {
            throw RuntimeException("Unable to connect to Strava API : ${connectException.message}")
        }
    }

    override fun getActivities(after: LocalDateTime): List<Activity> {
        try {
            return doGetActivities(after = after)
        } catch (connectException: ConnectException) {
            throw RuntimeException("Unable to connect to Strava API : ${connectException.message}")
        }
    }

    override fun getActivity(activityId: Long): Optional<DetailledActivity> {
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

    private fun doGetLoggedInAthlete(): Optional<Athlete> {

        val url = "https://www.strava.com/api/v3/athlete"

        val request = Request.Builder().url(url).headers(buildRequestHeaders()).build()

        okHttpClient.newCall(request).execute().use { response ->
            if (response.isSuccessful) {
                try {
                    val json = response.body?.string()
                    return if (json != null) {
                        Optional.of(objectMapper.readValue(json, Athlete::class.java))
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

    private fun doGetActivities(before: LocalDateTime? = null, after: LocalDateTime): List<Activity> {

        val activities = mutableListOf<Activity>()
        var page = 1
        var url = "https://www.strava.com/api/v3/athlete/activities?per_page=${properties.pagesize}"
        if (before != null) {
            url += "&before=${before.atZone(ZoneId.of("Europe/Paris")).toEpochSecond()}"
        }
        url += "&after=${after.atZone(ZoneId.of("Europe/Paris")).toEpochSecond()}"

        val requestHeaders = buildRequestHeaders()
        do {
            val request = Request.Builder().url("$url&page=${page++}").headers(requestHeaders).build()

            val result: List<Activity>
            okHttpClient.newCall(request).execute().use { response ->
                if (response.code == 401) {
                    logger.info("Invalid accessToken : $accessToken")
                    exitProcess(-1)
                }
                result = objectMapper.readValue(response.body?.string() ?: "")

                activities.addAll(result)
            }
        } while (result.isNotEmpty())

        return activities
    }

    private fun doGetActivityStream(activity: Activity): Optional<Stream> {

        // uploadId = 0 => this is a manual activity without streams
        if (activity.uploadId == 0L) {
            return Optional.empty()
        }
        val url =
            "https://www.strava.com/api/v3/activities/${activity.id}/streams" + "?keys=time,distance,latlng,altitude,moving,watts&key_by_type=true"

        val request: Request = Request.Builder().url(url).headers(buildRequestHeaders()).build()

        okHttpClient.newCall(request).execute().use { response ->
            when {
                response.code >= HttpStatus.BAD_REQUEST.value() -> {
                    logger.info("Unable to load streams for activity : ${activity.id}")
                    when (response.code) {
                        HttpStatus.TOO_MANY_REQUESTS.value() -> {
                            logger.info(
                                "Strava API usage is limited on a per-application basis using both a 15-minute " + "and daily request limit." + "The default rate limit allows 100 requests every 15 minutes, " + "with up to 1,000 requests per day."
                            )
                            throw RuntimeException("Something was wrong with Strava API : 429 Too Many Requests")
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
                        logger.info("Unable to load streams for activity : $activity")
                        Optional.empty()
                    }
                }

                else -> {
                    logger.info("Unable to load streams for activity : $activity")
                    throw RuntimeException("Something was wrong with Strava API for url $url : ${response.code} - ${response.code}")
                }
            }
        }
    }

    private fun doGetActivity(activityId: Long): Optional<DetailledActivity> {
        val url = "https://www.strava.com/api/v3/activities/$activityId?include_all_efforts=true"

        val request: Request = Request.Builder().url(url).headers(buildRequestHeaders()).build()

        okHttpClient.newCall(request).execute().use { response ->
            when {
                response.code >= HttpStatus.BAD_REQUEST.value() -> {
                    logger.info("Unable to load activity : $activityId")
                    when (response.code) {
                        HttpStatus.TOO_MANY_REQUESTS.value() -> {
                            logger.info(
                                "Strava API usage is limited on a per-application basis using both a 15-minute " + "and daily request limit." + "The default rate limit allows 100 requests every 15 minutes, " + "with up to 1,000 requests per day."
                            )
                            throw RuntimeException("Something was wrong with Strava API : 429 Too Many Requests")
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
                            Optional.of(objectMapper.readValue(json, DetailledActivity::class.java))
                        } else {
                            Optional.empty()
                        }
                    } catch (jsonProcessingException: JsonProcessingException) {
                        logger.info("Unable to load activity : $activityId - ${jsonProcessingException.message}")
                        Optional.empty()
                    }
                }

                else -> {
                    logger.info("Unable to load activity : $activityId")
                    throw RuntimeException("Something was wrong with Strava API for url $url : ${response.code} - ${response.code}")
                }
            }
        }
    }

    private fun setAccessToken(clientId: String, clientSecret: String) {
        val url =
            "https://www.strava.com/api/v3/oauth/authorize" + "?client_id=$clientId" + "&response_type=code" + "&redirect_uri=http://localhost:8090/exchange_token" + "&approval_prompt=auto" + "&scope=read_all,activity:read_all,profile:read_all"
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
                    throw RuntimeException("Something was wrong with Strava API for url $url : ${response.body}")
                }
            } catch (ex: Exception) {
                throw RuntimeException("Something was wrong with Strava API for url $url", ex)
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
                os.contains("win") -> Runtime.getRuntime().exec(arrayOf("rundll32 url.dll,FileProtocolHandler", url))
                os.contains("mac") -> Runtime.getRuntime().exec(arrayOf("open", url))
                os.contains("nix") || os.contains("nux") -> Runtime.getRuntime().exec(arrayOf("xdg-open", url))
                else -> println("Unsupported operating system. Cannot open the browser.")
            }
        } catch (e: Exception) {
            println("Failed to open the browser: ${e.message}")
        }
    }
}
