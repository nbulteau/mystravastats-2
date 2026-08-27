package me.nicolas.stravastats.domain.services.routing

import me.nicolas.stravastats.domain.business.Coordinates
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import tools.jackson.module.kotlin.readValue
import java.net.URI
import java.net.http.HttpClient
import java.net.http.HttpRequest
import java.net.http.HttpResponse
import java.time.Duration
import java.util.Locale

internal class OsrmClient(
    private val baseUrl: String,
    private val timeoutMs: Int,
) {
    private val mapper = JsonMapper.builder().addModule(KotlinModule.Builder().build()).build()
    private val httpClient = HttpClient.newBuilder().connectTimeout(Duration.ofMillis(timeoutMs.toLong())).build()

    fun healthStatus(): Int = send("$baseUrl/").statusCode()

    fun routes(profile: String, waypoints: List<Coordinates>, continueStraight: Boolean): List<OsrmRoute> {
        require(waypoints.size >= 2) { "at least 2 waypoints are required" }
        val coordinates = waypoints.joinToString(";") { "%.6f,%.6f".format(Locale.US, it.lng, it.lat) }
        val response = send("$baseUrl/route/v1/$profile/$coordinates?alternatives=true&steps=true&overview=full&geometries=geojson&continue_straight=$continueStraight")
        requireSuccess(response, "route")
        val payload = mapper.readValue<OsrmRouteResponse>(response.body())
        if (!payload.code.equals("ok", ignoreCase = true)) {
            throw IllegalStateException("OSRM route API returned code ${payload.code}: ${payload.message}")
        }
        return payload.routes
    }

    fun match(profile: String, shape: List<Coordinates>): List<OsrmRoute> {
        require(shape.size >= 2) { "at least 2 shape points are required for map matching" }
        val sampled = sampleCoordinatesByDistance(shape, maxPoints = 48, minSpacingMeters = 45.0)
        require(sampled.size >= 2) { "at least 2 sampled shape points are required for map matching" }
        val coordinates = sampled.joinToString(";") { "%.6f,%.6f".format(Locale.US, it.lng, it.lat) }
        var lastError: Throwable? = null
        for (radiusMeters in listOf(35.0, 80.0, 160.0, 320.0)) {
            val radiuses = sampled.joinToString(";") { "%.0f".format(Locale.US, radiusMeters) }
            try {
                val response = send("$baseUrl/match/v1/$profile/$coordinates?steps=true&overview=full&geometries=geojson&gaps=ignore&tidy=true&radiuses=$radiuses")
                requireSuccess(response, "match")
                val payload = mapper.readValue<OsrmMatchResponse>(response.body())
                if (!payload.code.equals("ok", ignoreCase = true)) {
                    lastError = IllegalStateException("OSRM match API returned code ${payload.code}")
                    if (!payload.code.equals("NoMatch", ignoreCase = true)) throw lastError
                    continue
                }
                val routes = payload.matchings.filter { it.distance > 0.0 && (it.geometry?.coordinates?.size ?: 0) >= 2 }
                if (routes.isNotEmpty()) return routes
                lastError = IllegalStateException("OSRM match API returned no valid matchings")
            } catch (error: InterruptedException) {
                Thread.currentThread().interrupt()
                throw error
            } catch (error: RuntimeException) {
                lastError = error
            } catch (error: java.io.IOException) {
                lastError = error
            }
        }
        throw IllegalStateException(lastError?.message ?: "OSRM match API returned no route")
    }

    fun nearest(profile: String, point: Coordinates): Pair<Coordinates, Double>? {
        val coordinate = "%.6f,%.6f".format(Locale.US, point.lng, point.lat)
        val response = runCatching { send("$baseUrl/nearest/v1/$profile/$coordinate?number=1") }.getOrElse { return null }
        if (response.statusCode() !in 200..299) return null
        val payload = runCatching { mapper.readValue<OsrmNearestResponse>(response.body()) }.getOrElse { return null }
        if (!payload.code.equals("ok", ignoreCase = true)) return null
        val waypoint = payload.waypoints.firstOrNull()?.takeIf { it.location.size >= 2 } ?: return null
        return Coordinates(lat = waypoint.location[1], lng = waypoint.location[0]) to waypoint.distance
    }

    private fun send(url: String): HttpResponse<String> = httpClient.send(
        HttpRequest.newBuilder().uri(URI.create(url)).timeout(Duration.ofMillis(timeoutMs.toLong())).GET().build(),
        HttpResponse.BodyHandlers.ofString(),
    )

    private fun requireSuccess(response: HttpResponse<String>, operation: String) {
        if (response.statusCode() !in 200..299) {
            throw IllegalStateException("OSRM $operation API returned status ${response.statusCode()}")
        }
    }
}
