package me.nicolas.stravastats

import com.fasterxml.jackson.module.kotlin.jacksonObjectMapper
import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import java.io.File

class TestHelper {
    companion object {
        fun loadActivities(): List<Activity> {
            val url = Thread.currentThread().contextClassLoader.getResource("activities.json")
            val jsonFile = File(url?.path ?: "")
            return jacksonObjectMapper().readValue(jsonFile, Array<Activity>::class.java).toList()
        }

        val activity = Activity(
            athlete = AthleteRef(id = 12345),
            averageSpeed = 5.5,
            averageCadence = 80.0,
            averageHeartrate = 150.0,
            maxHeartrate = 180.0,
            averageWatts = 200.0,
            commute = false,
            distance = 10000.0,
            deviceWatts = true,
            elapsedTime = 3600,
            elevHigh = 500.0,
            id = 67890,
            kilojoules = 500.0,
            maxSpeed = 10.0,
            movingTime = 3500,
            name = "Morning Run",
            startDate = "2023-10-01T08:00:00Z",
            startDateLocal = "2023-10-01T10:00:00+02:00",
            startLatlng = listOf(48.8566, 2.3522),
            totalElevationGain = 100.0,
            type = "Run",
            uploadId = 1234567890,
            weightedAverageWatts = 210
        )
    }

}