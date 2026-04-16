package me.nicolas.stravastats.domain.services.activityproviders

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.HeartRateZoneSettings
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import org.springframework.data.domain.Page
import org.springframework.data.domain.Pageable

data class ActivityProviderCacheIdentity(
    val cacheRoot: String,
    val athleteId: String,
)

interface IActivityProvider {

    fun athlete(): StravaAthlete

    fun listActivitiesPaginated(pageable: Pageable): Page<StravaActivity>

    fun getActivity(activityId: Long): StravaActivity?

    fun getDetailedActivity(activityId: Long): StravaDetailedActivity?

    fun getCachedDetailedActivity(activityId: Long): StravaDetailedActivity? = null

    fun getActivitiesByActivityTypeGroupByActiveDays(activityTypes: Set<ActivityType>): Map<String, Int>

    fun getActivitiesByActivityTypeByYearGroupByActiveDays(activityTypes: Set<ActivityType>, year: Int): Map<String, Int>

    fun getActivitiesByActivityTypeAndYear(activityTypes: Set<ActivityType>, year: Int? = null): List<StravaActivity>

    fun getActivitiesByActivityTypeGroupByYear(activityTypes: Set<ActivityType>): Map<String, List<StravaActivity>>

    fun getHeartRateZoneSettings(): HeartRateZoneSettings = HeartRateZoneSettings()

    fun saveHeartRateZoneSettings(settings: HeartRateZoneSettings): HeartRateZoneSettings = settings

    fun getCacheDiagnostics(): Map<String, Any?> = mapOf(
        "available" to false,
        "reason" to "cache diagnostics not supported by this activity provider",
    )

    fun cacheIdentity(): ActivityProviderCacheIdentity? = null
}
