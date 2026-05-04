package me.nicolas.stravastats.domain.business.strava.stream

import com.fasterxml.jackson.annotation.JsonIgnoreProperties
import com.fasterxml.jackson.annotation.JsonProperty
import me.nicolas.stravastats.domain.business.Slope
import me.nicolas.stravastats.domain.business.SlopeType
import kotlin.math.abs
import kotlin.math.max
import kotlin.math.min

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

    // Method to list sustained ascent segments as Slope objects.
    fun listSlopes(
        threshold: Double = 3.0, // Minimum smoothed grade to enter a climb (3%)
        minDistance: Double = 500.0, // Minimum distance for significant ascents (500m)
        climbIndex: Double = 3500.0, // Minimum climb index (distance × grade)
        smoothingWindow: Int = 25, // Distance-based smoothing hint for grade samples
    ): List<Slope> {
        // Check that we have the necessary data
        if (!hasAltitudeStream() || altitude == null) {
            return emptyList()
        }

        val altitudeData = altitude.data
        val distanceData = distance.data
        val timeData = time.data

        // Ensure all lists have the same size
        val dataSize = minOf(altitudeData.size, distanceData.size, timeData.size)
        if (dataSize < 2) {
            return emptyList()
        }

        val gradeSamples = gradePercentSamples(altitudeData, distanceData, dataSize)
        val windowMeters = max(100.0, smoothingWindow.toDouble() * 10.0)
        val smoothedGrades = smoothGradeByDistance(gradeSamples, distanceData, dataSize, windowMeters)

        return detectSustainedAscents(
            distanceData = distanceData,
            altitudeData = altitudeData,
            timeData = timeData,
            grades = smoothedGrades,
            dataSize = dataSize,
            threshold = threshold,
            minDistance = minDistance,
            climbIndex = climbIndex,
        )
    }

    private fun gradePercentSamples(
        altitudeData: List<Double>,
        distanceData: List<Double>,
        dataSize: Int,
    ): List<Double> {
        if (gradeSmooth != null && gradeSmooth.data.size >= dataSize) {
            val maxGrade = maxAbsGradeSmooth(gradeSmooth.data, dataSize)
            if (maxGrade > 0.0) {
                val ratioScale = maxGrade <= 1.0
                return (0 until dataSize).map { index ->
                    val grade = gradeSmooth.data[index].toDouble().finiteOrZero()
                    if (ratioScale) grade * 100 else grade
                }
            }
        }

        val grades = MutableList(dataSize) { 0.0 }
        for (index in 1 until dataSize) {
            val altitudeDiff = altitudeData[index] - altitudeData[index - 1]
            val distanceDiff = distanceData[index] - distanceData[index - 1]
            grades[index] = if (distanceDiff > 0 && altitudeDiff.isUsable() && distanceDiff.isUsable()) {
                (altitudeDiff / distanceDiff) * 100
            } else {
                0.0
            }
        }
        grades[0] = grades[1]
        return grades
    }

    private fun maxAbsGradeSmooth(grades: List<Float>, dataSize: Int): Double {
        var maxAbs = 0.0
        for (index in 0 until dataSize) {
            val grade = grades[index].toDouble()
            if (grade.isUsable()) {
                maxAbs = max(maxAbs, abs(grade))
            }
        }
        return maxAbs
    }

    private fun smoothGradeByDistance(
        grades: List<Double>,
        distances: List<Double>,
        dataSize: Int,
        windowMeters: Double,
    ): List<Double> {
        val smoothed = MutableList(dataSize) { 0.0 }
        val halfWindow = windowMeters / 2
        for (index in 0 until dataSize) {
            if (!distances[index].isUsable()) {
                smoothed[index] = grades[index].finiteOrZero()
                continue
            }

            var sum = 0.0
            var count = 0
            var beforeIndex = index
            while (beforeIndex >= 0) {
                if (distances[beforeIndex].isUsable()) {
                    if (distances[index] - distances[beforeIndex] > halfWindow) {
                        break
                    }
                    if (grades[beforeIndex].isUsable()) {
                        sum += grades[beforeIndex]
                        count += 1
                    }
                }
                beforeIndex -= 1
            }
            var afterIndex = index + 1
            while (afterIndex < dataSize) {
                if (distances[afterIndex].isUsable()) {
                    if (distances[afterIndex] - distances[index] > halfWindow) {
                        break
                    }
                    if (grades[afterIndex].isUsable()) {
                        sum += grades[afterIndex]
                        count += 1
                    }
                }
                afterIndex += 1
            }

            smoothed[index] = if (count == 0) grades[index].finiteOrZero() else sum / count
        }
        return smoothed
    }

    private fun detectSustainedAscents(
        distanceData: List<Double>,
        altitudeData: List<Double>,
        timeData: List<Int>,
        grades: List<Double>,
        dataSize: Int,
        threshold: Double,
        minDistance: Double,
        climbIndex: Double,
    ): List<Slope> {
        val enterThreshold = if (threshold > 0) threshold else 3.0
        val exitThreshold = max(1.0, enterThreshold * 0.35)
        val minAverageGrade = max(exitThreshold, enterThreshold * 0.65)
        val falseFlatDistance = min(300.0, max(150.0, minDistance * 0.5))

        val slopes = mutableListOf<Slope>()
        var inClimb = false
        var climbStartIndex = 0
        var belowExitStartIndex = -1
        var belowExitStartDistance = 0.0

        for (index in 1 until dataSize) {
            val grade = grades[index].finiteOrZero()
            if (!inClimb) {
                if (grade >= enterThreshold) {
                    climbStartIndex = index - 1
                    inClimb = true
                    belowExitStartIndex = -1
                }
                continue
            }

            if (grade < exitThreshold) {
                if (belowExitStartIndex < 0) {
                    belowExitStartIndex = index
                    belowExitStartDistance = distanceData[index]
                }
                if (
                    distanceData[index].isUsable() &&
                    belowExitStartDistance.isUsable() &&
                    distanceData[index] - belowExitStartDistance >= falseFlatDistance
                ) {
                    val endIndex = belowExitStartIndex - 1
                    buildClimbSlope(
                        distanceData = distanceData,
                        altitudeData = altitudeData,
                        timeData = timeData,
                        grades = grades,
                        startIndex = climbStartIndex,
                        endIndex = endIndex,
                        minDistance = minDistance,
                        minAverageGrade = minAverageGrade,
                        climbIndex = climbIndex,
                    )?.let { slopes.add(it) }
                    inClimb = false
                    belowExitStartIndex = -1
                }
                continue
            }

            belowExitStartIndex = -1
        }

        if (inClimb) {
            buildClimbSlope(
                distanceData = distanceData,
                altitudeData = altitudeData,
                timeData = timeData,
                grades = grades,
                startIndex = climbStartIndex,
                endIndex = dataSize - 1,
                minDistance = minDistance,
                minAverageGrade = minAverageGrade,
                climbIndex = climbIndex,
            )?.let { slopes.add(it) }
        }

        return slopes
    }

    private fun buildClimbSlope(
        distanceData: List<Double>,
        altitudeData: List<Double>,
        timeData: List<Int>,
        grades: List<Double>,
        startIndex: Int,
        endIndex: Int,
        minDistance: Double,
        minAverageGrade: Double,
        climbIndex: Double,
    ): Slope? {
        if (
            startIndex < 0 ||
            endIndex <= startIndex ||
            endIndex >= distanceData.size ||
            endIndex >= altitudeData.size ||
            endIndex >= timeData.size
        ) {
            return null
        }

        val startAltitude = altitudeData[startIndex]
        val endAltitude = altitudeData[endIndex]
        val totalDistance = distanceData[endIndex] - distanceData[startIndex]
        val totalDuration = timeData[endIndex] - timeData[startIndex]
        if (
            !startAltitude.isUsable() ||
            !endAltitude.isUsable() ||
            !totalDistance.isUsable() ||
            totalDistance <= 0 ||
            totalDuration < 0
        ) {
            return null
        }

        val averageGrade = ((endAltitude - startAltitude) / totalDistance) * 100
        if (
            !averageGrade.isUsable() ||
            totalDistance < minDistance ||
            averageGrade < minAverageGrade ||
            totalDistance * averageGrade < climbIndex
        ) {
            return null
        }

        var maxGrade = averageGrade
        for (index in (startIndex + 1)..endIndex) {
            if (index < grades.size && grades[index].isUsable()) {
                maxGrade = max(maxGrade, grades[index])
            }
        }

        val averageSpeed = if (totalDuration > 0) totalDistance / totalDuration else 0.0
        return Slope(
            type = SlopeType.ASCENT,
            startIndex = startIndex,
            endIndex = endIndex,
            startAltitude = startAltitude,
            endAltitude = endAltitude,
            grade = averageGrade,
            maxGrade = maxGrade,
            distance = totalDistance,
            duration = totalDuration,
            averageSpeed = averageSpeed,
        )
    }

    private fun Double.finiteOrZero() = if (isUsable()) this else 0.0

    private fun Double.isUsable() = !isNaN() && !isInfinite()
}
