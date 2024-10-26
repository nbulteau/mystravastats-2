package me.nicolas.stravastats.domain.services.activityproviders

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import org.springframework.data.domain.Page
import org.springframework.data.domain.Pageable
import java.util.*

interface IActivityProvider {

    fun athlete(): StravaAthlete

    fun listActivitiesPaginated(pageable: Pageable): Page<StravaActivity>

    fun getActivity(activityId: Long): Optional<StravaActivity>

    fun getDetailedActivity(activityId: Long): Optional<StravaDetailedActivity>

    fun getActivitiesByActivityTypeGroupByActiveDays(activityType: ActivityType): Map<String, Int>

    fun getActivitiesByActivityTypeByYearGroupByActiveDays(activityType: ActivityType, year: Int): Map<String, Int>

    fun getActivitiesByActivityTypeAndYear(activityType: ActivityType, year: Int? = null): List<StravaActivity>

    fun getActivitiesByActivityTypeGroupByYear(activityType: ActivityType): Map<String, List<StravaActivity>>
}