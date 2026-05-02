package me.nicolas.stravastats.domain.interfaces

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.stream.Stream

fun interface IStravaApiFactory {
    fun create(clientId: String, clientSecret: String): IStravaApi
}


interface IStravaApi {
    fun retrieveLoggedInAthlete(): StravaAthlete?

    fun getActivities(year: Int): List<StravaActivity>

    fun getActivitiesFailFastOnRateLimit(year: Int): List<StravaActivity> = getActivities(year)

    fun getActivityStream(stravaActivity: StravaActivity): Stream?

    fun getActivityStreamFailFastOnRateLimit(stravaActivity: StravaActivity): Stream? = getActivityStream(stravaActivity)

    fun getDetailedActivity(activityId: Long): StravaDetailedActivity?

    fun getDetailedActivityFailFastOnRateLimit(activityId: Long): StravaDetailedActivity? =
        getDetailedActivity(activityId)
}
