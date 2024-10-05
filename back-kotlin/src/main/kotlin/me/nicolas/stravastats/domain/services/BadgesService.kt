package me.nicolas.stravastats.domain.services

import com.fasterxml.jackson.databind.JsonMappingException
import com.fasterxml.jackson.module.kotlin.jacksonObjectMapper
import me.nicolas.stravastats.domain.business.badges.*
import me.nicolas.stravastats.domain.business.strava.ActivityType
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import java.nio.file.Path

interface IBadgesService {
    fun getGeneralBadges(activityType: ActivityType, year: Int?): List<BadgeCheckResult>

    fun getFamousBadges(activityType: ActivityType, year: Int?): List<BadgeCheckResult>
}

@Service
internal class BadgesService(
    stravaProxy: StravaProxy,
) : IBadgesService, AbstractStravaService(stravaProxy) {

    private val logger = LoggerFactory.getLogger(ActivityService::class.java)

    private val objectMapper = jacksonObjectMapper()

    private val alpes: BadgeSet = loadBadgeSet("alpes", "famous-climb/alpes.json")

    private val pyrenees: BadgeSet = loadBadgeSet("pyrenees", "famous-climb/pyrenees.json")

    override fun getGeneralBadges(activityType: ActivityType, year: Int?): List<BadgeCheckResult> {
        logger.info("Checking general badges for $activityType in $year")

        val activities = stravaProxy.getFilteredActivitiesByActivityTypeAndYear(activityType, year)

        return when (activityType) {
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

    override fun getFamousBadges(activityType: ActivityType, year: Int?): List<BadgeCheckResult> {
        logger.info("Checking famous badges for $activityType in $year")

        val activities = stravaProxy.getFilteredActivitiesByActivityTypeAndYear(activityType, year)

        return when (activityType) {
            ActivityType.Ride -> {
                alpes.check(activities) + pyrenees.check(activities)
            }

            else -> emptyList()
        }
    }

    private fun loadBadgeSet(name: String, climbsJsonFilePath: String): BadgeSet {
        var famousClimbBadgeList: List<Badge>

        try {
            val url = Path.of(climbsJsonFilePath).toUri().toURL()
            val famousClimbs = objectMapper.readValue(url, Array<FamousClimb>::class.java).toList()
            famousClimbBadgeList = famousClimbs.flatMap { famousClimb ->
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
            }.toList()
        } catch (jsonMappingException: JsonMappingException) {
            println("Something was wrong while reading BadgeSet : ${jsonMappingException.message}")
            famousClimbBadgeList = emptyList()
        }

        return BadgeSet(name, famousClimbBadgeList)
    }
}
