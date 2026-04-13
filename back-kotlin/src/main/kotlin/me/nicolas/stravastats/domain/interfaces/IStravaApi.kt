package me.nicolas.stravastats.domain.interfaces

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import java.util.*

interface IStravaApi {
    fun retrieveLoggedInAthlete(): Optional<StravaAthlete>

    fun getActivities(year: Int): List<StravaActivity>

    fun getActivitiesFailFastOnRateLimit(year: Int): List<StravaActivity> = getActivities(year)

    fun getActivityStream(stravaActivity: StravaActivity): Stream?

    fun getDetailedActivity(activityId: Long): Optional<StravaDetailedActivity>
}
