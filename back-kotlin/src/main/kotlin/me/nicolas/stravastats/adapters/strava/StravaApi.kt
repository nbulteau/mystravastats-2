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
import me.nicolas.stravastats.domain.utils.BrowserUtils.openBrowser
import okhttp3.Headers
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.slf4j.Logger
import org.slf4j.LoggerFactory
import org.springframework.http.HttpStatus
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
                before = LocalDateTime.of(year + 1, 1, 1, 0, 0),
                after = LocalDateTime.of(year, 1, 1, 0, 0)
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
                logger.info("Error configuring proxy : $malformedURLException")
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
                    val json = response.body.string()
                    return Optional.of(objectMapper.readValue(json, StravaAthlete::class.java))
                } catch (jsonMappingException: JsonMappingException) {
                    throw RuntimeException("Something was wrong with Strava API", jsonMappingException)
                }
            } else {
                throw RuntimeException("Something was wrong with Strava API for url $url : ${response.body.string()}")
            }
        }
    }

    private fun doGetActivities(before: LocalDateTime? = null, after: LocalDateTime): List<StravaActivity> {

        val activities = mutableListOf<StravaActivity>()
        var page = 1
        var url = "https://www.strava.com/api/v3/athlete/activities?per_page=${properties.pageSize}"
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
                    quotaExceedLimit()
                }
                result = objectMapper.readValue(response.body.string())

                activities.addAll(result)
            }
        } while (result.isNotEmpty())

        return activities
    }

    private fun quotaExceedLimit(): Nothing {
        logger.info(QUOTA_EXCEED_LIMIT)
        exitProcess(-1)
    }

    private fun doGetActivityStream(stravaActivity: StravaActivity): Stream? {

        // uploadId = 0 => this is a manual stravaActivity without streams
        if (stravaActivity.uploadId == 0L) {
            return null
        }
        val url =
            "https://www.strava.com/api/v3/activities/${stravaActivity.id}/streams" + "?keys=time,distance,latlng,altitude,velocity_smooth,heartrate,cadence,watts,moving,grade_smooth&key_by_type=true"

        val request: Request = Request.Builder().url(url).headers(buildRequestHeaders()).build()

        okHttpClient.newCall(request).execute().use { response ->
            when {
                response.code >= HttpStatus.BAD_REQUEST.value() -> {
                    when (response.code) {
                        HttpStatus.TOO_MANY_REQUESTS.value() -> {
                            logger.error("Unable to load streams for stravaActivity : ${stravaActivity.id} - $QUOTA_EXCEED_LIMIT")
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
                        return objectMapper.readValue(json, Stream::class.java)
                    } catch (jsonProcessingException: JsonProcessingException) {
                        logger.error("Unable to load streams for stravaActivity : $stravaActivity: ${jsonProcessingException.message}")
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

        val request: Request = Request.Builder().url(url).headers(buildRequestHeaders()).build()

        okHttpClient.newCall(request).execute().use { response ->
            when {
                response.code >= HttpStatus.BAD_REQUEST.value() -> {
                    when (response.code) {
                        HttpStatus.TOO_MANY_REQUESTS.value() -> {
                            logger.error("Unable to load stravaActivity : $activityId - $QUOTA_EXCEED_LIMIT")
                            return Optional.empty()
                        }

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
                        return Optional.of(objectMapper.readValue(json, StravaDetailedActivity::class.java))
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
                            buildResponseHtml(clientId),
                            ContentType.Text.Html
                        )
                        launch {
                            // Get an authorization token with the code
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
                    return objectMapper.readValue(response.body.string(), Token::class.java)
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
}
