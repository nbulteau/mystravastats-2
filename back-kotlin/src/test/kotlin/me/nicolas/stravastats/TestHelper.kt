package me.nicolas.stravastats

import com.fasterxml.jackson.module.kotlin.jacksonObjectMapper
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.ActivityType

import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import java.io.File

class TestHelper {
    companion object {
        fun loadActivities(): List<StravaActivity> {
            val url = Thread.currentThread().contextClassLoader.getResource("activities.json")
            val jsonFile = File(url?.path ?: "")
            val activities = jacksonObjectMapper().readValue(jsonFile, Array<StravaActivity>::class.java)

            return activities.sortedBy { it.startDate }.reversed()
        }

        fun run2020Activities() = loadActivities().getFilteredActivitiesByActivityTypeAndYear(ActivityType.Run, 2020)

        fun hike2020Activities() = loadActivities().getFilteredActivitiesByActivityTypeAndYear(ActivityType.Hike, 2020)

        fun ride2020Activities() = loadActivities().getFilteredActivitiesByActivityTypeAndYear(ActivityType.Ride, 2020)

        fun run2023Activities() = loadActivities().getFilteredActivitiesByActivityTypeAndYear(ActivityType.Run, 2023)


        val stravaActivity = StravaActivity(
            athlete = AthleteRef(id = 12345),
            averageSpeed = 5.5,
            averageCadence = 80.0,
            averageHeartrate = 150.0,
            maxHeartrate = 180.0,
            averageWatts = 200,
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

        val stravaAthlete = StravaAthlete(
            badgeTypeId = 1,
            city = "Paris",
            country = "France",
            createdAt = "2023-01-01T00:00:00Z",
            firstname = "John",
            follower = null,
            friend = null,
            id = 123456,
            lastname = "Doe",
            premium = true,
            profile = "http://example.com/profile.jpg",
            profileMedium = "http://example.com/profile_medium.jpg",
            resourceState = 2,
            sex = "M",
            state = "Ile-de-France",
            summit = false,
            updatedAt = "2023-01-01T00:00:00Z",
            username = "john.doe",
            athleteType = 1,
            bikes = emptyList(),
            clubs = emptyList(),
            datePreference = "Europe/Paris",
            followerCount = 100,
            friendCount = 50,
            ftp = null,
            measurementPreference = "meters",
            mutualFriendCount = 10,
            shoes = emptyList(),
            weight = 70
        )

        private fun List<StravaActivity>.getFilteredActivitiesByActivityTypeAndYear(activityType: ActivityType, year: Int): List<StravaActivity> {

            val filteredActivities = this
                .filter { activity -> activity.startDateLocal.subSequence(0, 4).toString().toInt() == year }
                .filter { activity -> (activity.type == activityType.name) && !activity.commute }

            return filteredActivities
        }
    }



}