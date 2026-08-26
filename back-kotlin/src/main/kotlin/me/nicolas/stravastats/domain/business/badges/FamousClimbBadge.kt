package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.GeoCoordinate

private const val FAMOUS_CLIMB_ACTIVITY_START_RADIUS_KM = 80.0
private const val FAMOUS_CLIMB_WAYPOINT_TOLERANCE_METERS = 500
private const val FAMOUS_CLIMB_LENGTH_TOLERANCE_RATIO = 0.35
private const val FAMOUS_CLIMB_LENGTH_TOLERANCE_MINIMUM_METERS = 750.0


data class FamousClimbBadge(
    override val label: String,
	val summitId: String = "",
	val variantId: String = "",
    val name: String,
    val topOfTheAscent: Int,
    val start: GeoCoordinate,
    val end: GeoCoordinate,
    val routeCheckpoints: List<GeoCoordinate> = emptyList(),
    val summitToleranceMeters: Int = 0,
    val length: Double,
    val totalAscent: Int,
    val averageGradient: Double,
    val difficulty: Int,
    val category: String,
    val minimumAltitude: Int = 0,
    val maximumGradient: Double = 0.0,
    val country: String = "",
    val massif: String = "",
    val sourceUrl: String = "",
) : Badge(label) {

    override fun check(activities: List<StravaActivity>): Pair<List<StravaActivity>, Boolean> {
        val filteredActivities = activities.filter { activity ->
            if (activity.startLatlng?.isNotEmpty() == true) {
                val distanceToStart = this.start.haversineInKM(activity.startLatlng[0], activity.startLatlng[1])
                val distanceToEnd = this.end.haversineInKM(activity.startLatlng[0], activity.startLatlng[1])
                distanceToStart < FAMOUS_CLIMB_ACTIVITY_START_RADIUS_KM || distanceToEnd < FAMOUS_CLIMB_ACTIVITY_START_RADIUS_KM
            } else {
                false
            }
        }.filter { activity -> matchQuality(activity) != null }

        return Pair(filteredActivities, filteredActivities.isNotEmpty())
    }

    internal fun matchQuality(stravaActivity: StravaActivity): Double? {
        val stream = stravaActivity.stream ?: return null
        val latLngStream = stream.latlng ?: return null
        val checkpointMatchIndices = routeCheckpointMatchIndices(latLngStream.data) ?: return null
        val distances = stream.distance.data
        val startIndices = mutableListOf<Int>()
        var fallbackMatch = false
        var scoredCandidate = false
        var bestFallbackQuality = Double.POSITIVE_INFINITY
        var bestQuality = Double.POSITIVE_INFINITY
        val referenceLengthMeters = length * 1000.0
        val resolvedSummitToleranceMeters = resolvedSummitToleranceMeters()
        val lengthToleranceMeters = maxOf(
            FAMOUS_CLIMB_LENGTH_TOLERANCE_MINIMUM_METERS,
            referenceLengthMeters * FAMOUS_CLIMB_LENGTH_TOLERANCE_RATIO,
        )

        for ((index, coords) in latLngStream.data.withIndex()) {
            if (coords.size < 2) {
                continue
            }
            if (this.start.haversineInM(coords[0], coords[1]) < FAMOUS_CLIMB_WAYPOINT_TOLERANCE_METERS) {
                startIndices += index
            }
            val endDistanceMeters = this.end.haversineInM(coords[0], coords[1])
            if (endDistanceMeters >= resolvedSummitToleranceMeters) {
                continue
            }

            for (startIndex in startIndices) {
                if (startIndex >= index) {
                    continue
                }
                if (!containsRouteCheckpoints(checkpointMatchIndices, startIndex, index)) {
                    continue
                }
                fallbackMatch = true
                val startCoords = latLngStream.data[startIndex]
                val startProximity = this.start.haversineInM(startCoords[0], startCoords[1]) /
                    FAMOUS_CLIMB_WAYPOINT_TOLERANCE_METERS.toDouble()
                val endProximity = endDistanceMeters / resolvedSummitToleranceMeters.toDouble()
                val proximityQuality = (startProximity + endProximity) / 2.0
                bestFallbackQuality = minOf(bestFallbackQuality, proximityQuality)
                if (referenceLengthMeters <= 0.0 || startIndex >= distances.size || index >= distances.size) {
                    continue
                }
                val candidateLengthMeters = distances[index] - distances[startIndex]
                if (!candidateLengthMeters.isFinite() || candidateLengthMeters <= 0.0) {
                    continue
                }
                scoredCandidate = true
                val lengthDifference = kotlin.math.abs(candidateLengthMeters - referenceLengthMeters)
                if (lengthDifference <= lengthToleranceMeters) {
                    val quality = lengthDifference / maxOf(referenceLengthMeters, 1.0) + proximityQuality * 0.01
                    bestQuality = minOf(bestQuality, quality)
                }
            }
        }

        if (bestQuality.isFinite()) {
            return bestQuality
        }
        if (fallbackMatch && (!scoredCandidate || referenceLengthMeters <= 0.0)) {
            return bestFallbackQuality
        }
        return null
    }

    private fun resolvedSummitToleranceMeters(): Int =
        summitToleranceMeters.takeIf { it > 0 } ?: FAMOUS_CLIMB_WAYPOINT_TOLERANCE_METERS

    private fun routeCheckpointMatchIndices(latLngData: List<List<Double>>): List<List<Int>>? {
        if (routeCheckpoints.isEmpty()) {
            return emptyList()
        }

        val checkpointMatchIndices = routeCheckpoints.map { mutableListOf<Int>() }
        latLngData.forEachIndexed { index, coords ->
            if (coords.size < 2) {
                return@forEachIndexed
            }
            routeCheckpoints.forEachIndexed { checkpointIndex, checkpoint ->
                if (checkpoint.haversineInM(coords[0], coords[1]) < FAMOUS_CLIMB_WAYPOINT_TOLERANCE_METERS) {
                    checkpointMatchIndices[checkpointIndex] += index
                }
            }
        }
        return checkpointMatchIndices.takeIf { matches -> matches.all { it.isNotEmpty() } }
    }

    private fun containsRouteCheckpoints(
        checkpointMatchIndices: List<List<Int>>,
        startIndex: Int,
        endIndex: Int,
    ): Boolean {
        var nextMinimumIndex = startIndex
        for (indices in checkpointMatchIndices) {
            val searchResult = indices.binarySearch(nextMinimumIndex)
            val position = if (searchResult >= 0) searchResult else -searchResult - 1
            if (position >= indices.size || indices[position] > endIndex) {
                return false
            }
            nextMinimumIndex = indices[position] + 1
        }
        return true
    }

    override fun toString() = name
}
