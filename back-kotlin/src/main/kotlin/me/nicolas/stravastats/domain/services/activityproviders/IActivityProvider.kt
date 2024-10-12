package me.nicolas.stravastats.domain.services.activityproviders

import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.business.strava.ActivityType
import me.nicolas.stravastats.domain.business.strava.Athlete
import me.nicolas.stravastats.domain.business.strava.DetailedActivity
import org.springframework.data.domain.Page
import org.springframework.data.domain.Pageable
import java.util.*

interface IActivityProvider {

    fun athlete(): Athlete

    fun listActivitiesPaginated(pageable: Pageable): Page<Activity>

    fun getActivity(activityId: Long): Optional<Activity>

    fun getDetailedActivity(activityId: Long): Optional<DetailedActivity>

    fun getActivitiesByActivityTypeGroupByActiveDays(activityType: ActivityType): Map<String, Int>

    fun getActivitiesByActivityTypeByYearGroupByActiveDays(activityType: ActivityType, year: Int): Map<String, Int>

    fun getFilteredActivitiesByActivityTypeAndYear(activityType: ActivityType, year: Int? = null): List<Activity>

    fun getActivitiesByActivityTypeGroupByYear(activityType: ActivityType): Map<String, List<Activity>>
}