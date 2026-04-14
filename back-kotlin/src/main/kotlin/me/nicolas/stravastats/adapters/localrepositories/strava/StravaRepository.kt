package me.nicolas.stravastats.adapters.localrepositories.strava


import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.HeartRateZoneSettings
import me.nicolas.stravastats.domain.interfaces.ILocalStorageProvider
import me.nicolas.stravastats.domain.services.ActivityHelper.filterByActivityTypes
import org.slf4j.LoggerFactory
import tools.jackson.databind.DeserializationFeature
import tools.jackson.databind.ObjectWriter
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import java.io.File
import java.io.FileInputStream
import java.nio.file.Files
import java.util.Collections
import java.util.Properties
import kotlin.io.path.name

internal class StravaRepository(stravaCache: String) : ILocalStorageProvider {

    private val logger = LoggerFactory.getLogger(StravaRepository::class.java)

    private val objectMapper = JsonMapper.builder()
        .addModule(KotlinModule.Builder().build())
        .disable(DeserializationFeature.FAIL_ON_NULL_FOR_PRIMITIVES)
        .build()

    private val writer: ObjectWriter = objectMapper.writer()

    private val prettyWriter = objectMapper.writerWithDefaultPrettyPrinter()

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
            try {
                activities = objectMapper.readValue(yearActivitiesJsonFile, Array<StravaActivity>::class.java)
                    .toList()
                    .filterByActivityTypes()
            } catch (exception: Exception) {
                logger.error("Unable to load activities from cache for year $year")
                return emptyList()
            }
            logger.info("${activities.size} activities loaded for year $year from cache")

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

        // Return empty set if the directory does not exist yet
        if (!yearActivitiesDirectory.exists()) {
            return emptySet()
        }

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

    /**
     * Update Strava authentication in the ".strava" file.
     */
    override fun updateStravaAuthentication(stravaCache: String, clientId: String, clientSecret: String, useCache: Boolean) {
        val cacheDirectory = File(stravaCache)
        val file = File(cacheDirectory, ".strava")
        val properties = Properties()

        // Load existing properties if file exists
        if (file.exists()) {
            FileInputStream(file).use { properties.load(it) }
        }

        // Update properties
        properties["clientId"] = clientId
        properties["clientSecret"] = clientSecret
        properties["useCache"] = useCache.toString()

        // Save properties to file
        try {
            file.outputStream().use { properties.store(it, null) }
            logger.info("Updated Strava authentication file: useCache=$useCache")
        } catch (e: Exception) {
            logger.error("Failed to update Strava authentication file", e)
        }
    }

    override fun loadHeartRateZoneSettings(clientId: String): HeartRateZoneSettings {
        val activitiesDirectory = File(cacheDirectory, "strava-$clientId")
        val settingsFile = File(activitiesDirectory, "heart-rate-zones-$clientId.json")
        if (!settingsFile.exists()) {
            return HeartRateZoneSettings()
        }

        return runCatching {
            objectMapper.readValue(settingsFile, HeartRateZoneSettings::class.java)
        }.getOrElse { exception ->
            logger.error("Unable to read heart-rate zone settings from ${settingsFile.absolutePath}", exception)
            HeartRateZoneSettings()
        }
    }

    override fun saveHeartRateZoneSettings(clientId: String, settings: HeartRateZoneSettings) {
        val activitiesDirectory = File(cacheDirectory, "strava-$clientId")
        activitiesDirectory.mkdirs()
        val settingsFile = File(activitiesDirectory, "heart-rate-zones-$clientId.json")

        prettyWriter.writeValue(settingsFile, settings)
    }

    private fun loadActivitiesStreams(activities: List<StravaActivity>, activitiesDirectory: File) {
        if (activities.isEmpty()) {
            return
        }

        val failures = Collections.synchronizedList(mutableListOf<Pair<Long, Exception>>())
        activities.parallelStream().forEach { activity ->
            val streamFile = File(activitiesDirectory, "stream-${activity.id}")
            if (streamFile.exists()) {
                try {
                    activity.stream = objectMapper.readValue(streamFile, Stream::class.java)
                } catch (exception: Exception) {
                    failures.add(activity.id to exception)
                }
            }
        }

        failures.forEach { (activityId, exception) ->
            logger.error("Unable to load stream from cache for activity $activityId", exception)
        }
    }
}
