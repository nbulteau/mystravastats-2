package me.nicolas.stravastats.domain.business.strava.stream

import com.fasterxml.jackson.annotation.JsonIgnoreProperties
import com.fasterxml.jackson.annotation.JsonProperty
import me.nicolas.stravastats.domain.business.Slope
import me.nicolas.stravastats.domain.business.SlopeType
import me.nicolas.stravastats.domain.services.ActivityHelper.smooth

@JsonIgnoreProperties(ignoreUnknown = true)
data class Stream(
    val distance: DistanceStream,
    val time: TimeStream,
    val latlng: LatLngStream? = null,
    val cadence: CadenceStream? = null,
    val heartrate: HeartRateStream? = null,
    val moving: MovingStream? = null,
    val altitude: AltitudeStream? = null,
    val watts: PowerStream? = null,
    @param:JsonProperty("velocity_smooth")
    val velocitySmooth: SmoothVelocityStream? = null,
    @param:JsonProperty("grade_smooth")
    val gradeSmooth: SmoothGradeStream? = null,
) {
    fun hasLatLngStream() = latlng != null
    fun hasPowerStream() = watts != null
    fun hasAltitudeStream() = altitude != null
    fun hasMovingStream() = moving != null
    fun hasVelocitySmoothStream() = velocitySmooth != null
    fun hasGradeSmoothStream() = gradeSmooth != null
    fun hasHeartRateStream() = heartrate != null
    fun hasCadenceStream() = cadence != null

    // Method to list all slope segments as Slope objects with smoothed data
    fun listSlopes(
        threshold: Double = 3.0, // Minimum average grade for ascents (3%)
        minDistance: Double = 500.0, // Minimum distance for significant ascents (500m)
        climbIndex: Double = 3500.0, // Minimum climb index (distance × grade)
        smoothingWindow: Int = 20, // Size of a smoothing window for raw GPS data
    ): List<Slope> {
        val slopes = mutableListOf<Slope>()

        // Check that we have the necessary data
        if (!hasAltitudeStream() || altitude == null) {
            return slopes
        }

        // Apply smoothing to raw GPS data to reduce noise
        val rawAltitudeData = altitude.data
        val rawDistanceData = distance.data
        val timeData = time.data

        // Smooth altitude and distance data using the helper function
        val smoothedAltitudeData = rawAltitudeData.smooth(smoothingWindow)
        val smoothedDistanceData = rawDistanceData.smooth(smoothingWindow)

        // Ensure all lists have the same size
        val dataSize = minOf(smoothedAltitudeData.size, smoothedDistanceData.size, timeData.size)
        if (dataSize < 2) {
            return slopes
        }

        var currentSlopeStartIndex = 0
        var currentSlopeType: SlopeType? = null

        for (i in 1 until dataSize) {
            val altitudeDiff = smoothedAltitudeData[i] - smoothedAltitudeData[i - 1]
            val distanceDiff = smoothedDistanceData[i] - smoothedDistanceData[i - 1]

            if (distanceDiff == 0.0) continue

            val grade = (altitudeDiff / distanceDiff) * 100 // Percentage grade

            val slopeType = when {
                grade >= threshold -> SlopeType.ASCENT
                grade <= -threshold -> SlopeType.DESCENT
                else -> SlopeType.PLATEAU
            }

            // If the slope type changes or if we're at the last point
            if (currentSlopeType != slopeType || i == dataSize - 1) {
                // Create a slope segment if we have a previous segment
                currentSlopeType?.let { type ->
                    val endIndex = if (i == dataSize - 1) i else i - 1

                    if (endIndex > currentSlopeStartIndex) {
                        val startAltitude = smoothedAltitudeData[currentSlopeStartIndex]
                        val endAltitude = smoothedAltitudeData[endIndex]
                        val totalDistance =
                            smoothedDistanceData[endIndex] - smoothedDistanceData[currentSlopeStartIndex]
                        val totalDuration = timeData[endIndex] - timeData[currentSlopeStartIndex]

                        // Calculate average grade for the segment using smoothed data
                        val averageGrade = if (totalDistance > 0) {
                            ((endAltitude - startAltitude) / totalDistance) * 100
                        } else {
                            0.0
                        }

                        // Apply specific criteria for ascents
                        val shouldIncludeSlope = when (type) {
                            SlopeType.ASCENT -> {
                                // For ascents, check all three criteria:
                                // 1. Minimum distance (500m)
                                // 2. Minimum average grade (3%)
                                // 3. Minimum climb index (3500)
                                val calculatedClimbIndex = totalDistance * kotlin.math.abs(averageGrade)
                                totalDistance >= minDistance &&
                                        kotlin.math.abs(averageGrade) >= threshold &&
                                        calculatedClimbIndex >= climbIndex
                            }

                           SlopeType.DESCENT -> {
                                // For descents, use basic distance criteria
                                totalDistance >= minDistance && kotlin.math.abs(averageGrade) >= threshold
                            }

                           SlopeType.PLATEAU -> {
                                // For plateaus, use basic distance criteria
                                totalDistance >= minDistance
                            }
                        }

                        if (shouldIncludeSlope) {
                            // Calculate the maximum grade for the segment using smoothed data
                            var maxGrade = 0.0
                            for (j in currentSlopeStartIndex until endIndex) {
                                val segmentDistanceDiff = smoothedDistanceData[j + 1] - smoothedDistanceData[j]
                                if (segmentDistanceDiff > 0) {
                                    val segmentGrade =
                                        ((smoothedAltitudeData[j + 1] - smoothedAltitudeData[j]) / segmentDistanceDiff) * 100
                                    maxGrade = if(currentSlopeType == SlopeType.DESCENT) {
                                        minOf(maxGrade, segmentGrade)
                                    } else {
                                        maxOf(maxGrade, segmentGrade)
                                    }
                                }
                            }

                            val averageSpeed = if (totalDuration > 0) {
                                totalDistance / totalDuration
                            } else {
                                0.0
                            }

                            slopes.add(
                                Slope(
                                    type = type,
                                    startIndex = currentSlopeStartIndex,
                                    endIndex = endIndex,
                                    startAltitude = startAltitude,
                                    endAltitude = endAltitude,
                                    grade = averageGrade,
                                    maxGrade = maxGrade,
                                    distance = totalDistance,
                                    duration = totalDuration,
                                    averageSpeed = averageSpeed
                                )
                            )
                        }
                    }
                }

                // Start a new segment
                currentSlopeStartIndex = if (i == dataSize - 1) i else i - 1
                currentSlopeType = slopeType
            }
        }

        return mergeConsecutiveSegments(slopes)
    }



    /**
     * Merges consecutive slope segments of the same type into a single segment.
     * Also merges small slopes with different types when they are between two slopes of the same type.
     * This is useful to reduce noise and provide a cleaner representation of the activity's slopes.
     */
    private fun mergeConsecutiveSegments(slopes: List<Slope>): List<Slope> {
        if (slopes.isEmpty()) return emptyList()

        // First pass: merge consecutive segments of the same type
        val mergedSlopes = mutableListOf<Slope>()
        var currentSlope = slopes[0]

        for (i in 1 until slopes.size) {
            val slope = slopes[i]

            // Check if we can merge with the current slope
            if (currentSlope.type == slope.type) {
                // Merge with the current slope
                currentSlope = Slope(
                    type = currentSlope.type,
                    startIndex = currentSlope.startIndex,
                    endIndex = slope.endIndex,
                    startAltitude = currentSlope.startAltitude,
                    endAltitude = slope.endAltitude,
                    grade = (currentSlope.grade * currentSlope.distance + slope.grade * slope.distance) / (currentSlope.distance + slope.distance),
                    maxGrade = maxOf(currentSlope.maxGrade, slope.maxGrade),
                    distance = currentSlope.distance + slope.distance,
                    duration = currentSlope.duration + slope.duration,
                    averageSpeed = (currentSlope.averageSpeed * currentSlope.distance + slope.averageSpeed * slope.distance) / (currentSlope.distance + slope.distance)
                )
            } else {
                // Cannot merge, add the current slope to results and start a new one
                mergedSlopes.add(currentSlope)
                currentSlope = slope
            }
        }

        // Remember to add the last slope
        mergedSlopes.add(currentSlope)

        // Second pass: merge small slopes between two slopes of the same type
        return mergeSmallIntermediateSlopes(mergedSlopes)
    }

    /**
     * Identifies small intermediate slopes and merges them with surrounding slopes of the same type.
     * A small slope is considered for merging if it's shorter than 500m and is between two slopes of the same type.
     */
    private fun mergeSmallIntermediateSlopes(slopes: List<Slope>): List<Slope> {
        if (slopes.size < 3) return slopes

        val result = mutableListOf<Slope>()
        var i = 0

        while (i < slopes.size) {
            // Check if we have a pattern: slope1 - smallSlope - slope2 where slope1.type == slope2.type
            if (i < slopes.size - 2 &&
                slopes[i].type == slopes[i + 2].type &&
                slopes[i + 1].type != slopes[i].type &&
                slopes[i + 1].distance < 500.0) { // Small slope threshold: 500m

                // Merge all three slopes into one
                val slope1 = slopes[i]
                val smallSlope = slopes[i + 1]
                val slope2 = slopes[i + 2]

                val totalDistance = slope1.distance + smallSlope.distance + slope2.distance
                val totalDuration = slope1.duration + smallSlope.duration + slope2.duration

                val mergedSlope = Slope(
                    type = slope1.type, // Use the type of the surrounding slopes
                    startIndex = slope1.startIndex,
                    endIndex = slope2.endIndex,
                    startAltitude = slope1.startAltitude,
                    endAltitude = slope2.endAltitude,
                    grade = (slope1.grade * slope1.distance + smallSlope.grade * smallSlope.distance + slope2.grade * slope2.distance) / totalDistance,
                    maxGrade = maxOf(slope1.maxGrade, smallSlope.maxGrade, slope2.maxGrade),
                    distance = totalDistance,
                    duration = totalDuration,
                    averageSpeed = (slope1.averageSpeed * slope1.distance + smallSlope.averageSpeed * smallSlope.distance + slope2.averageSpeed * slope2.distance) / totalDistance
                )

                result.add(mergedSlope)
                i += 3 // Skip the next two slopes as they've been merged
            } else {
                result.add(slopes[i])
                i += 1
            }
        }

        return result
    }
}

