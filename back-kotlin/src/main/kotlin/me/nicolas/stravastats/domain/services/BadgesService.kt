package me.nicolas.stravastats.domain.services


import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.badges.*
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import tools.jackson.databind.DeserializationFeature
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import java.nio.file.Files
import java.nio.file.Path

interface IBadgesService {
    fun getGeneralBadges(activityTypes: Set<ActivityType>, year: Int?): List<BadgeCheckResult>

    fun getFamousBadges(activityTypes: Set<ActivityType>, year: Int?): List<BadgeCheckResult>
}

@Service
internal class BadgesService(
    activityProvider: IActivityProvider,
) : IBadgesService, AbstractStravaService(activityProvider) {

    private val logger = LoggerFactory.getLogger(ActivityService::class.java)

    private val objectMapper = JsonMapper.builder()
        .addModule(KotlinModule.Builder().build())
        .disable(DeserializationFeature.FAIL_ON_NULL_FOR_PRIMITIVES)
        .build()

    private val alpes: BadgeSet = loadBadgeSet("alpes", "famous-climb/alpes.json")

    private val pyrenees: BadgeSet = loadBadgeSet("pyrenees", "famous-climb/pyrenees.json")

    override fun getGeneralBadges(activityTypes: Set<ActivityType>, year: Int?): List<BadgeCheckResult> {
        logger.info("Checking general badges for $activityTypes in ${year ?: "all years"}")

        val activities = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)

        // TODO: handle case multiple activity types
        return when (activityTypes.firstOrNull()) {
            ActivityType.Ride -> {
                DistanceBadge.rideBadgeSet.check(activities) +
                        ElevationBadge.rideBadgeSet.check(activities) +
                        MovingTimeBadge.movingTimeBadgesSet.check(activities)
            }

            ActivityType.Hike -> {
                DistanceBadge.hikeBadgeSet.check(activities) +
                        ElevationBadge.hikeBadgeSet.check(activities) +
                        MovingTimeBadge.movingTimeBadgesSet.check(activities)
            }

            ActivityType.Run -> {
                DistanceBadge.runBadgeSet.check(activities) +
                        ElevationBadge.runBadgeSet.check(activities) +
                        MovingTimeBadge.movingTimeBadgesSet.check(activities)
            }

            else -> emptyList()
        }
    }

    override fun getFamousBadges(activityTypes: Set<ActivityType>, year: Int?): List<BadgeCheckResult> {
        logger.info("Checking famous badges for $activityTypes in ${year ?: "all years"}")

        val activities = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)

        // TODO: handle case multiple activity types
        return when (activityTypes.firstOrNull()) {
            ActivityType.Ride -> {
                alpes.check(activities) + pyrenees.check(activities)
            }

            else -> emptyList()
        }
    }

    private fun loadBadgeSet(name: String, climbsJsonFilePath: String): BadgeSet {
        val famousClimbs = loadFamousClimbs(climbsJsonFilePath)
        val famousClimbBadgeList = famousClimbs.flatMap { famousClimb ->
            famousClimb.alternatives.map { alternative ->
                FamousClimbBadge(
                    name = famousClimb.name,
                    label = "${famousClimb.name} from ${alternative.name}",
                    topOfTheAscent = famousClimb.topOfTheAscent,
                    start = famousClimb.geoCoordinate,
                    end = alternative.geoCoordinate,
                    difficulty = alternative.difficulty,
                    length = alternative.length,
                    totalAscent = alternative.totalAscent,
                    averageGradient = alternative.averageGradient
                )
            }
        }

        return BadgeSet(name, famousClimbBadgeList)
    }

    private fun loadFamousClimbs(climbsJsonFilePath: String): List<FamousClimb> {
        try {
            val filePath = resolveBadgeJsonFile(climbsJsonFilePath)
            if (filePath != null) {
                return objectMapper.readValue(filePath.toFile(), Array<FamousClimb>::class.java).toList()
            }

            val classpathResource = this::class.java.classLoader.getResourceAsStream(climbsJsonFilePath)
            if (classpathResource != null) {
                classpathResource.use { input ->
                    return objectMapper.readValue(input, Array<FamousClimb>::class.java).toList()
                }
            }

            logger.warn("Badge file not found for '$climbsJsonFilePath'. Badges set will be empty.")
            return emptyList()
        } catch (exception: Exception) {
            logger.error("Something was wrong while reading BadgeSet '$climbsJsonFilePath' : ${exception.message}")
            return emptyList()
        }
    }

    private fun resolveBadgeJsonFile(climbsJsonFilePath: String): Path? {
        val candidates = mutableListOf(
            Path.of(climbsJsonFilePath),
            Path.of("strava-cache").resolve(climbsJsonFilePath),
            Path.of("back-kotlin").resolve(climbsJsonFilePath),
        )

        val stravaCachePath = System.getenv("STRAVA_CACHE_PATH")
        if (!stravaCachePath.isNullOrBlank()) {
            candidates.add(Path.of(stravaCachePath).resolve(climbsJsonFilePath))
        }

        return candidates.firstOrNull { Files.isRegularFile(it) }
    }
}
