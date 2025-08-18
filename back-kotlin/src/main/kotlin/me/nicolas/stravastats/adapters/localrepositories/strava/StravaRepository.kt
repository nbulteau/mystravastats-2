package me.nicolas.stravastats.adapters.localrepositories.strava

import com.fasterxml.jackson.core.util.DefaultPrettyPrinter
import com.fasterxml.jackson.databind.ObjectWriter
import com.fasterxml.jackson.module.kotlin.jacksonObjectMapper
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.interfaces.ILocalStorageProvider
import me.nicolas.stravastats.domain.services.ActivityHelper.filterByActivityTypes
import org.slf4j.LoggerFactory
import java.io.File
import java.io.FileInputStream
import java.nio.file.Files
import java.util.*
import kotlin.io.path.name

internal class StravaRepository(stravaCache: String) : ILocalStorageProvider {

    private val logger = LoggerFactory.getLogger(StravaRepository::class.java)

    private val objectMapper = jacksonObjectMapper()

    private val writer: ObjectWriter = objectMapper.writer()

    private val prettyWriter: ObjectWriter = objectMapper.writer(DefaultPrettyPrinter())

    private val cacheDirectory = File(stravaCache)

    override fun initLocalStorageForClientId(clientId: String) {
        val activitiesDirectory = File(cacheDirectory, "strava-$clientId")
        if (!activitiesDirectory.exists()) {
            activitiesDirectory.mkdirs()
        }
    }

    override fun loadAthleteFromCache(clientId: String): StravaAthlete {
        val activitiesDirectory = File(cacheDirectory, "strava-$clientId")
        val athleteJsonFile = File(activitiesDirectory, "athlete-$clientId.json")

        return if (athleteJsonFile.exists()) {
            objectMapper.readValue(athleteJsonFile, StravaAthlete::class.java)
        } else {
            logger.warn("No stravaAthlete found in cache")
            StravaAthlete(id = clientId.toLong(), username = "Unknown")
        }
    }

    override fun saveAthleteToCache(clientId: String, stravaAthlete: StravaAthlete) {
        val activitiesDirectory = File(cacheDirectory, "strava-$clientId")
        activitiesDirectory.mkdirs()
        prettyWriter.writeValue(File(activitiesDirectory, "athlete-$clientId.json"), stravaAthlete)
    }

    override fun loadActivitiesFromCache(clientId: String, year: Int): List<StravaActivity> {
        var activities = emptyList<StravaActivity>()
        val activitiesDirectory = File(cacheDirectory, "strava-$clientId")
        val yearActivitiesDirectory = File(activitiesDirectory, "strava-$clientId-$year")
        val yearActivitiesJsonFile = File(yearActivitiesDirectory, "activities-$clientId-$year.json")

        if (yearActivitiesJsonFile.exists()) {
            logger.info("Load activities from cache for year $year")
            activities = objectMapper.readValue(yearActivitiesJsonFile, Array<StravaActivity>::class.java)
                .toList()
                .filterByActivityTypes()
            logger.info("${activities.size} activities loaded")

            // Load activities streams
            loadActivitiesStreams(activities, yearActivitiesDirectory)
        }

        return activities
    }

    override fun isLocalCacheExistForYear(clientId: String, year: Int): Boolean {
        val activitiesDirectory = File(cacheDirectory, "strava-$clientId")
        val yearActivitiesDirectory = File(activitiesDirectory, "strava-$clientId-$year")
        val yearActivitiesJsonFile = File(yearActivitiesDirectory, "activities-$clientId-$year.json")

        return yearActivitiesJsonFile.exists()
    }

    override fun getLocalCacheLastModified(clientId: String, year: Int): Long {
        val activitiesDirectory = File(cacheDirectory, "strava-$clientId")
        val yearActivitiesDirectory = File(activitiesDirectory, "strava-$clientId-$year")
        val yearActivitiesJsonFile = File(yearActivitiesDirectory, "activities-$clientId-$year.json")

        return yearActivitiesJsonFile.lastModified()
    }

    override fun saveActivitiesToCache(clientId: String, year: Int, activities: List<StravaActivity>) {
        val activitiesDirectory = File(cacheDirectory, "strava-$clientId")
        val yearActivitiesDirectory = File(activitiesDirectory, "strava-$clientId-$year")
        yearActivitiesDirectory.mkdirs()

        prettyWriter.writeValue(
            File(yearActivitiesDirectory, "activities-$clientId-$year.json"),
            activities
        )
    }

    override fun loadDetailedActivityFromCache(clientId: String, year: Int, activityId: Long): StravaDetailedActivity? {
        var stravaDetailedActivity: StravaDetailedActivity? = null
        val activitiesDirectory = File(cacheDirectory, "strava-$clientId")
        val yearActivitiesDirectory = File(activitiesDirectory, "strava-${clientId}-$year")
        val detailedActivityFile = File(yearActivitiesDirectory, "stravaActivity-${activityId}")

        if (detailedActivityFile.exists()) {
            stravaDetailedActivity = objectMapper.readValue(detailedActivityFile, StravaDetailedActivity::class.java)
        }

        return stravaDetailedActivity
    }

    override fun saveDetailedActivityToCache(clientId: String, year: Int, stravaDetailedActivity: StravaDetailedActivity) {
        val activitiesDirectory = File(cacheDirectory, "strava-$clientId")
        val yearActivitiesDirectory = File(activitiesDirectory, "strava-${clientId}-$year")
        val detailedActivityFile = File(yearActivitiesDirectory, "stravaActivity-${stravaDetailedActivity.id}")

        writer.writeValue(detailedActivityFile, stravaDetailedActivity)
    }

    override fun loadActivitiesStreamsFromCache(clientId: String, year: Int, stravaActivity: StravaActivity): Stream? {
        val activitiesDirectory = File(cacheDirectory, "strava-$clientId")
        val yearActivitiesDirectory = File(activitiesDirectory, "strava-${clientId}-$year")

        val streamFile = File(yearActivitiesDirectory, "stream-${stravaActivity.id}")

        return if (streamFile.exists()) {
            objectMapper.readValue(streamFile, Stream::class.java)
        } else {
            null
        }
    }

    override fun saveActivitiesStreamsToCache(clientId: String, year: Int, stravaActivity: StravaActivity, stream: Stream) {
        val activitiesDirectory = File(cacheDirectory, "strava-$clientId")
        val yearActivitiesDirectory = File(activitiesDirectory, "strava-${clientId}-$year")

        val streamFile = File(yearActivitiesDirectory, "stream-${stravaActivity.id}")

        writer.writeValue(streamFile, stream)
    }

    override fun buildStreamIdsSet(clientId: String, year: Int): Set<Long> {

        val activitiesDirectory = File(cacheDirectory, "strava-$clientId")
        val yearActivitiesDirectory = File(activitiesDirectory, "strava-$clientId-$year")

        return Files.walk(yearActivitiesDirectory.toPath())
            .filter { Files.isRegularFile(it) }
            .filter { it.name.startsWith("stream-") }
            .map { it.name.substringAfter("stream-").toLong() }
            .toList().toSet()
    }

    /**
     * Read Strava authentication from the ".strava" file.
     * The file must contain two properties: clientId and clientSecret.
     * @return a Triple with clientId, clientSecret and useCache
     */
    override fun readStravaAuthentication(stravaCache: String): Triple<String?, String?, Boolean?> {
        val cacheDirectory = File(stravaCache)
        val file = File(cacheDirectory, ".strava")
        val properties = Properties()

        if (file.exists()) {
            FileInputStream(file).use { properties.load(it) }
        } else {
            logger.error("File .strava not found")
        }

        return Triple(
            properties["clientId"]?.toString(),
            properties["clientSecret"]?.toString(),
            properties["useCache"]?.toString()?.toBoolean()
        )
    }

    private fun loadActivitiesStreams(activities: List<StravaActivity>, activitiesDirectory: File) {
        activities.forEach { activity ->
            val streamFile = File(activitiesDirectory, "stream-${activity.id}")
            if (streamFile.exists()) {
                activity.stream = objectMapper.readValue(streamFile, Stream::class.java)
            }
        }
    }
}