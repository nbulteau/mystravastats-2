package me.nicolas.stravastats.domain.interfaces

import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.business.strava.Athlete
import me.nicolas.stravastats.domain.business.strava.DetailedActivity
import me.nicolas.stravastats.domain.business.strava.Stream
import java.time.LocalDateTime
import java.util.*

interface IStravaApi {
    fun retrieveLoggedInAthlete(): Optional<Athlete>

    fun getActivities(year: Int): List<Activity>

    fun getActivities(after: LocalDateTime): List<Activity>

    fun getActivityStream(activity: Activity): Optional<Stream>

    fun getDetailledActivity(activityId: Long): Optional<DetailedActivity>
}