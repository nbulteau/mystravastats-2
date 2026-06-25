package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import java.time.DayOfWeek
import java.time.LocalDate
import java.util.Locale
import kotlin.math.round

data class HikingBadge(
    override val label: String,
    val description: String,
    private val matcher: (List<StravaActivity>) -> List<StravaActivity>,
) : Badge(label) {

    override fun check(activities: List<StravaActivity>): Pair<List<StravaActivity>, Boolean> {
        val checkedActivities = matcher(activities)
        return Pair(checkedActivities, checkedActivities.isNotEmpty())
    }

    override fun toString() = label

    companion object {
        val summitDayBadge = HikingBadge(
            label = "Summit Day",
            description = "Reach a high point above 2000 m with at least 500 m of elevation gain.",
            matcher = ::summitDayActivities,
        )
        val backToBackHikingWeekendBadge = HikingBadge(
            label = "Back-to-back Hiking Weekend",
            description = "Record hikes on both Saturday and Sunday of the same weekend.",
            matcher = ::backToBackHikingWeekendActivities,
        )
        val highPointPRBadge = HikingBadge(
            label = "High Point PR",
            description = "Your highest recorded hiking point.",
            matcher = ::highPointPrActivities,
        )
        val newTrailBadge = HikingBadge(
            label = "New Trail",
            description = "Explore a hiking trailhead or route name not seen earlier in this badge scope.",
            matcher = ::newTrailActivities,
        )
        val hikingAdventureBadgeSet = BadgeSet(
            name = "Hiking adventures",
            badges = listOf(
                summitDayBadge,
                backToBackHikingWeekendBadge,
                highPointPRBadge,
                newTrailBadge,
            )
        )
    }
}

private fun summitDayActivities(activities: List<StravaActivity>): List<StravaActivity> =
    activities.filter { activity ->
        activity.elevHigh >= 2000 && activity.totalElevationGain >= 500
    }

private fun backToBackHikingWeekendActivities(activities: List<StravaActivity>): List<StravaActivity> {
    val activitiesByDate = activities.groupBy { activity -> activity.localDateOrNull() ?: return@groupBy null }
        .filterKeys { it != null }
        .mapKeys { it.key!! }

    val matched = linkedMapOf<Long, StravaActivity>()
    activitiesByDate.forEach { (date, saturdayActivities) ->
        if (date.dayOfWeek != DayOfWeek.SATURDAY) return@forEach
        val sunday = date.plusDays(1)
        if (sunday.dayOfWeek != DayOfWeek.SUNDAY) return@forEach
        val sundayActivities = activitiesByDate[sunday].orEmpty()
        if (sundayActivities.isEmpty()) return@forEach
        (saturdayActivities + sundayActivities).forEach { activity ->
            matched.putIfAbsent(activity.id, activity)
        }
    }
    return matched.values.toList()
}

private fun highPointPrActivities(activities: List<StravaActivity>): List<StravaActivity> {
    val bestActivity = activities
        .filter { activity -> activity.elevHigh.isFinite() && activity.elevHigh > 0 }
        .maxByOrNull { activity -> activity.elevHigh }
    return listOfNotNull(bestActivity)
}

private fun newTrailActivities(activities: List<StravaActivity>): List<StravaActivity> {
    val seenTrailKeys = mutableSetOf<String>()
    return activities.sortedWith(compareBy(
        { activity -> activity.localDateOrNull() ?: LocalDate.MAX },
        { activity -> activity.startDateLocal },
        { activity -> activity.id },
    )).filter { activity ->
        val key = activity.hikingTrailKey()
        key.isNotBlank() && seenTrailKeys.add(key)
    }
}

private fun StravaActivity.localDateOrNull(): LocalDate? {
    val dateText = startDateLocal.ifBlank { startDate }
    if (dateText.length < "yyyy-MM-dd".length) return null
    return runCatching { LocalDate.parse(dateText.take(10)) }.getOrNull()
}

private fun StravaActivity.hikingTrailKey(): String {
    val coordinates = startLatlng
    if (coordinates != null && coordinates.size >= 2 && coordinates[0].isValidCoordinate() && coordinates[1].isValidCoordinate()) {
        return "geo:${coordinates[0].roundedTrailCoordinate()}:${coordinates[1].roundedTrailCoordinate()}"
    }
    return name.lowercase(Locale.US)
        .map { char -> if (char.isLetterOrDigit()) char else ' ' }
        .joinToString("")
        .trim()
        .split(Regex("\\s+"))
        .filter { it.isNotBlank() }
        .joinToString(" ")
}

private fun Double.roundedTrailCoordinate(): String =
    "%.3f".format(Locale.US, round(this * 1000) / 1000)

private fun Double.isValidCoordinate(): Boolean =
    isFinite() && this != 0.0
