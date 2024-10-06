package me.nicolas.stravastats.adapter.localstorage

import com.fasterxml.jackson.core.util.DefaultPrettyPrinter
import com.fasterxml.jackson.databind.ObjectWriter
import com.fasterxml.jackson.module.kotlin.jacksonObjectMapper
import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.business.strava.Athlete
import me.nicolas.stravastats.domain.business.strava.DetailedActivity
import me.nicolas.stravastats.domain.business.strava.Stream
import me.nicolas.stravastats.domain.interfaces.ILocalStorageProvider
import me.nicolas.stravastats.domain.services.ActivityHelper.filterActivities
import org.slf4j.LoggerFactory
import java.io.File
import java.nio.file.Files
import kotlin.io.path.name

internal class LocalStorageProvider : ILocalStorageProvider {

    private val logger = LoggerFactory.getLogger(LocalStorageProvider::class.java)

    private val objectMapper = jacksonObjectMapper()

    private val writer: ObjectWriter = objectMapper.writer()

    private val prettyWriter: ObjectWriter = objectMapper.writer(DefaultPrettyPrinter())

    private val cacheDirectory = File("strava-cache")

    override fun loadAthleteFromCache(clientId: String): Athlete? {
        var athlete: Athlete? = null

        val activitiesDirectory = File(cacheDirectory, "strava-$clientId")
        val athleteJsonFile = File(activitiesDirectory, "athlete-$clientId.json")

        if (athleteJsonFile.exists()) {
            athlete = objectMapper.readValue(athleteJsonFile, Athlete::class.java)
        }

        return athlete
    }

    override fun saveAthleteToCache(clientId: String, athlete: Athlete) {
        val activitiesDirectory = File(cacheDirectory,"strava-$clientId")
        activitiesDirectory.mkdirs()
        prettyWriter.writeValue(File(activitiesDirectory, "athlete-$clientId.json"), athlete)
    }

    override fun loadActivitiesFromCache(clientId: String, year: Int): List<Activity> {
        var activities = emptyList<Activity>()
        val activitiesDirectory = File(cacheDirectory,"strava-$clientId")
        val yearActivitiesDirectory = File(activitiesDirectory, "strava-$clientId-$year")
        val yearActivitiesJsonFile = File(yearActivitiesDirectory, "activities-$clientId-$year.json")

        if (yearActivitiesJsonFile.exists()) {
            logger.info("Load activities from cache for year $year")
            activities = objectMapper.readValue(yearActivitiesJsonFile, Array<Activity>::class.java)
                .toList()
                .filterActivities()
            logger.info("${activities.size} activities loaded")

            // Load activities streams
            loadActivitiesStreams(activities, yearActivitiesDirectory)
        }

        return activities
    }

    override fun isLocalCacheExistForYear(clientId: String, year: Int): Boolean {
        val activitiesDirectory = File(cacheDirectory,"strava-$clientId")
        val yearActivitiesDirectory = File(activitiesDirectory, "strava-$clientId-$year")
        val yearActivitiesJsonFile = File(yearActivitiesDirectory, "activities-$clientId-$year.json")

        return yearActivitiesJsonFile.exists()
    }

    override fun saveActivitiesToCache(clientId: String, year: Int, activities: List<Activity>) {
        val activitiesDirectory = File(cacheDirectory,"strava-$clientId")
        val yearActivitiesDirectory = File(activitiesDirectory, "strava-$clientId-$year")
        yearActivitiesDirectory.mkdirs()

        prettyWriter.writeValue(
            File(yearActivitiesDirectory, "activities-$clientId-$year.json"),
            activities
        )
    }

    override fun loadDetailedActivityFromCache(clientId: String, year: Int, activityId: Long): DetailedActivity? {
        var detailedActivity: DetailedActivity? = null
        val activitiesDirectory = File(cacheDirectory,"strava-$clientId")
        val yearActivitiesDirectory = File(activitiesDirectory, "strava-${clientId}-$year")
        val detailedActivityFile = File(yearActivitiesDirectory, "activity-${activityId}")

        if (detailedActivityFile.exists()) {
            detailedActivity = objectMapper.readValue(detailedActivityFile, DetailedActivity::class.java)
        }

        return detailedActivity
    }

    override fun saveDetailedActivityToCache(clientId: String, year: Int, detailedActivity: DetailedActivity) {
        val activitiesDirectory = File(cacheDirectory,"strava-$clientId")
        val yearActivitiesDirectory = File(activitiesDirectory, "strava-${clientId}-$year")
        val detailedActivityFile = File(yearActivitiesDirectory, "activity-${detailedActivity.id}")

        writer.writeValue(detailedActivityFile, detailedActivity)
    }

    override fun loadActivitiesStreamsFromCache(clientId: String, year: Int, activity: Activity): Stream? {
        val activitiesDirectory = File(cacheDirectory,"strava-$clientId")
        val yearActivitiesDirectory = File(activitiesDirectory, "strava-${clientId}-$year")

        val streamFile = File(yearActivitiesDirectory, "stream-${activity.id}")

        return if (streamFile.exists()) {
            objectMapper.readValue(streamFile, Stream::class.java)
        } else {
            null
        }
    }

    override fun saveActivitiesStreamsToCache(clientId: String, year: Int, activity: Activity, stream: Stream) {
        val activitiesDirectory = File(cacheDirectory,"strava-$clientId")
        val yearActivitiesDirectory = File(activitiesDirectory, "strava-${clientId}-$year")

        val streamFile = File(yearActivitiesDirectory, "stream-${activity.id}")

        writer.writeValue(streamFile, stream)
    }

    override fun buildStreamIdsSet(clientId: String, year: Int): Set<Long> {

        val activitiesDirectory = File(cacheDirectory,"strava-$clientId")
        val yearActivitiesDirectory = File(activitiesDirectory, "strava-$clientId-$year")

        return Files.walk(yearActivitiesDirectory.toPath())
            .filter { Files.isRegularFile(it) }
            .filter { it.name.startsWith("stream-") }
            .map { it.name.substringAfter("stream-").toLong() }
            .toList().toSet()
    }

    private fun loadActivitiesStreams(activities: List<Activity>, activitiesDirectory: File) {
        activities.forEach { activity ->
            val streamFile = File(activitiesDirectory, "stream-${activity.id}")
            if (streamFile.exists()) {
                activity.stream = objectMapper.readValue(streamFile, Stream::class.java)
            }
        }
    }
}