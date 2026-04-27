package me.nicolas.stravastats.api.controllers

import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import io.swagger.v3.oas.annotations.tags.Tag
import me.nicolas.stravastats.api.dto.ErrorResponseMessageDto
import me.nicolas.stravastats.api.dto.MapPassageSegmentDto
import me.nicolas.stravastats.api.dto.MapPassagesDto
import me.nicolas.stravastats.api.dto.MapTrackDto
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.services.dataQualityExcludedActivityIds
import me.nicolas.stravastats.domain.services.withDataQualityCorrections
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.springframework.http.MediaType
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController
import kotlin.math.PI
import kotlin.math.atan2
import kotlin.math.ceil
import kotlin.math.cos
import kotlin.math.floor
import kotlin.math.round
import kotlin.math.sin
import kotlin.math.sqrt

private const val MAP_PASSAGE_DEFAULT_RESOLUTION_METERS = 120
private const val MAP_PASSAGE_ALL_YEARS_RESOLUTION_METERS = 250
private const val MAP_PASSAGE_MAX_LEG_METERS = 2000.0
private const val MAP_PASSAGE_METERS_PER_DEGREE = 111320.0
private const val MAP_PASSAGE_EARTH_RADIUS_M = 6371e3
private const val MAP_PASSAGE_DEFAULT_MAX_SEGMENTS = 12000
private const val MAP_PASSAGE_ALL_YEARS_MAX_SEGMENTS = 5000

@RestController
@RequestMapping("/maps")
@Schema(description = "Maps controller", name = "MapsController")
@Tag(name = "Maps", description = "Maps endpoints")
class MapsController(
    private val stravaProxy: IActivityProvider,
) {
    @Operation(
        description = "Get the GPX coordinates for a specific stravaActivity type and year",
        summary = "Get the GPX coordinates for a specific stravaActivity type and year",
        responses = [ApiResponse(
            responseCode = "200", description = "GPX coordinates found", content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE, schema = Schema(implementation = List::class)
            )]
        ), ApiResponse(
            responseCode = "404", description = "GPX coordinates not found", content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = ErrorResponseMessageDto::class)
            )]
        )],
    )
    @GetMapping("/gpx")
    fun getGPX(
        @RequestParam(required = true) activityType: String,
        @RequestParam(required = false) year: Int?,
    ): List<MapTrackDto> {
        val activityTypes = activityType.convertToActivityTypeSet()

        val activities = stravaProxy.getActivitiesByActivityTypeAndYear(activityTypes, year)
            .withDataQualityCorrections(stravaProxy)

        // Keep enough points to render smooth tracks while avoiding huge payloads.
        val step = year?.let { 10 } ?: 100

        return activities.mapNotNull { activity ->
            val coordinates = sampleCoordinates(activity.stream?.latlng?.data, step)
            if (coordinates.size < 2) {
                return@mapNotNull null
            }

            MapTrackDto(
                activityId = activity.id,
                activityName = activity.name,
                activityDate = activity.startDateLocal,
                activityType = resolveMapTrackActivityType(activity),
                distanceKm = activity.distance / 1000.0,
                elevationGainM = activity.totalElevationGain,
                coordinates = coordinates,
            )
        }
    }

    @Operation(
        description = "Get aggregated passage density corridors for a specific activity type and year",
        summary = "Get map passage density corridors",
    )
    @GetMapping("/passages")
    fun getPassages(
        @RequestParam(required = true) activityType: String,
        @RequestParam(required = false) year: Int?,
    ): MapPassagesDto {
        val activityTypes = activityType.convertToActivityTypeSet()
        val activities = stravaProxy.getActivitiesByActivityTypeAndYear(activityTypes, year)
            .withDataQualityCorrections(stravaProxy)
        val excludedIds = dataQualityExcludedActivityIds(stravaProxy)
        return buildPassages(
            activities = activities.filterNot { activity -> excludedIds.contains(activity.id) },
            excludedActivities = activities.count { activity -> excludedIds.contains(activity.id) },
            options = passageOptionsForYear(year),
        )
    }

    private fun sampleCoordinates(
        latlng: List<List<Double>>?,
        step: Int,
    ): List<List<Double>> {
        if (latlng.isNullOrEmpty()) {
            return emptyList()
        }

        val sampled = latlng
            .filter { pair ->
                pair.size >= 2
                    && pair[0].isFinite()
                    && pair[1].isFinite()
            }
            .filterIndexed { index, _ -> index % step == 0 }
            .map { pair -> listOf(pair[0], pair[1]) }
            .toMutableList()

        val last = latlng.lastOrNull()
        if (
            last != null
            && last.size >= 2
            && last[0].isFinite()
            && last[1].isFinite()
        ) {
            val shouldAppendLast = sampled.isEmpty()
                || sampled.last()[0] != last[0]
                || sampled.last()[1] != last[1]
            if (shouldAppendLast) {
                sampled.add(listOf(last[0], last[1]))
            }
        }
        return sampled
    }

    private fun resolveMapTrackActivityType(activity: StravaActivity): String {
        if (activity.commute) {
            return ActivityType.Commute.name
        }

        return activity.sportType.takeIf { it.isNotBlank() }
            ?: activity.type.takeIf { it.isNotBlank() }
            ?: ActivityType.Ride.name
    }

    private fun buildPassages(
        activities: List<StravaActivity>,
        excludedActivities: Int,
        options: PassageOptions,
    ): MapPassagesDto {
        val accumulators = mutableMapOf<PassageEdge, PassageAccumulator>()
        var includedActivities = 0
        var missingStreamActivities = 0

        activities.forEach { activity ->
            val latlng = activity.stream?.latlng?.data
            val coordinates = validPassageCoordinates(latlng)
            if (coordinates.size < 2) {
                missingStreamActivities++
                return@forEach
            }

            val activityEdges = passageEdgesForActivity(coordinates, options.resolutionMeters)
            if (activityEdges.isEmpty()) {
                missingStreamActivities++
                return@forEach
            }

            includedActivities++
            val resolvedActivityType = resolveMapTrackActivityType(activity)
            activityEdges.forEach { edge ->
                val accumulator = accumulators.getOrPut(edge) { PassageAccumulator() }
                accumulator.passageCount++
                accumulator.activityTypeCounts[resolvedActivityType] = (accumulator.activityTypeCounts[resolvedActivityType] ?: 0) + 1
            }
        }

        var omittedSegments = 0
        val segments = accumulators.mapNotNull { (edge, accumulator) ->
            if (accumulator.passageCount < options.minPassageCount) {
                omittedSegments++
                return@mapNotNull null
            }
            val start = edge.start.center(options.resolutionMeters)
            val end = edge.end.center(options.resolutionMeters)
            val edgeDistanceKm = passageDistanceMeters(start[0], start[1], end[0], end[1]) / 1000.0
            MapPassageSegmentDto(
                coordinates = listOf(start, end),
                passageCount = accumulator.passageCount,
                activityCount = accumulator.passageCount,
                distanceKm = roundPassageDistance(edgeDistanceKm * accumulator.passageCount),
                activityTypeCounts = accumulator.activityTypeCounts.toSortedMap(),
            )
        }.sortedWith(
            compareByDescending<MapPassageSegmentDto> { it.passageCount }
                .thenBy { it.coordinates.first()[0] }
                .thenBy { it.coordinates.first()[1] }
        ).let { sortedSegments ->
            if (options.maxSegments > 0 && sortedSegments.size > options.maxSegments) {
                omittedSegments += sortedSegments.size - options.maxSegments
                sortedSegments.take(options.maxSegments)
            } else {
                sortedSegments
            }
        }

        return MapPassagesDto(
            segments = segments,
            includedActivities = includedActivities,
            excludedActivities = excludedActivities,
            missingStreamActivities = missingStreamActivities,
            resolutionMeters = options.resolutionMeters,
            minPassageCount = options.minPassageCount,
            omittedSegments = omittedSegments,
        )
    }

    private fun validPassageCoordinates(latlng: List<List<Double>>?): List<List<Double>> {
        if (latlng.isNullOrEmpty()) {
            return emptyList()
        }
        return latlng
            .filter { coordinate ->
                coordinate.size >= 2
                    && coordinate[0].isFinite()
                    && coordinate[1].isFinite()
            }
            .map { coordinate -> listOf(coordinate[0], coordinate[1]) }
    }

    private fun passageEdgesForActivity(coordinates: List<List<Double>>, resolutionMeters: Int): Set<PassageEdge> {
        val cells = mutableListOf<PassageCell>()
        fun appendCell(cell: PassageCell) {
            if (cells.lastOrNull() == cell) {
                return
            }
            cells += cell
        }

        for (index in 1 until coordinates.size) {
            val previous = coordinates[index - 1]
            val current = coordinates[index]
            val distance = passageDistanceMeters(previous[0], previous[1], current[0], current[1])
            if (distance <= 0.0 || distance > MAP_PASSAGE_MAX_LEG_METERS) {
                continue
            }

            val steps = maxOf(1, ceil(distance / resolutionMeters).toInt())
            for (step in 0..steps) {
                val ratio = step.toDouble() / steps.toDouble()
                val latitude = previous[0] + (current[0] - previous[0]) * ratio
                val longitude = previous[1] + (current[1] - previous[1]) * ratio
                appendCell(PassageCell.from(latitude, longitude, resolutionMeters))
            }
        }

        return cells.zipWithNext()
            .filter { (left, right) -> left != right }
            .map { (left, right) -> PassageEdge.normalized(left, right) }
            .toSet()
    }

    private fun passageDistanceMeters(lat1: Double, lon1: Double, lat2: Double, lon2: Double): Double {
        val lat1Rad = lat1 * PI / 180
        val lat2Rad = lat2 * PI / 180
        val deltaLat = (lat2 - lat1) * PI / 180
        val deltaLon = (lon2 - lon1) * PI / 180

        val a = sin(deltaLat / 2) * sin(deltaLat / 2) +
            cos(lat1Rad) * cos(lat2Rad) * sin(deltaLon / 2) * sin(deltaLon / 2)
        val c = 2 * atan2(sqrt(a), sqrt(1 - a))
        return MAP_PASSAGE_EARTH_RADIUS_M * c
    }

    private fun roundPassageDistance(value: Double): Double {
        return round(value * 100.0) / 100.0
    }

    private data class PassageCell(
        val lat: Int,
        val lng: Int,
    ) {
        fun center(resolutionMeters: Int): List<Double> {
            val degrees = resolutionMeters / MAP_PASSAGE_METERS_PER_DEGREE
            return listOf(
                (lat.toDouble() + 0.5) * degrees,
                (lng.toDouble() + 0.5) * degrees,
            )
        }

        companion object {
            fun from(latitude: Double, longitude: Double, resolutionMeters: Int): PassageCell {
                val degrees = resolutionMeters / MAP_PASSAGE_METERS_PER_DEGREE
                return PassageCell(
                    lat = floor(latitude / degrees).toInt(),
                    lng = floor(longitude / degrees).toInt(),
                )
            }
        }
    }

    private data class PassageEdge(
        val start: PassageCell,
        val end: PassageCell,
    ) {
        companion object {
            fun normalized(left: PassageCell, right: PassageCell): PassageEdge {
                return if (left.lat < right.lat || (left.lat == right.lat && left.lng <= right.lng)) {
                    PassageEdge(left, right)
                } else {
                    PassageEdge(right, left)
                }
            }
        }
    }

    private data class PassageAccumulator(
        var passageCount: Int = 0,
        val activityTypeCounts: MutableMap<String, Int> = mutableMapOf(),
    )

    private data class PassageOptions(
        val resolutionMeters: Int,
        val minPassageCount: Int,
        val maxSegments: Int,
    )

    private fun passageOptionsForYear(year: Int?): PassageOptions {
        return if (year == null) {
            PassageOptions(
                resolutionMeters = MAP_PASSAGE_ALL_YEARS_RESOLUTION_METERS,
                minPassageCount = 2,
                maxSegments = MAP_PASSAGE_ALL_YEARS_MAX_SEGMENTS,
            )
        } else {
            PassageOptions(
                resolutionMeters = MAP_PASSAGE_DEFAULT_RESOLUTION_METERS,
                minPassageCount = 1,
                maxSegments = MAP_PASSAGE_DEFAULT_MAX_SEGMENTS,
            )
        }
    }
}
