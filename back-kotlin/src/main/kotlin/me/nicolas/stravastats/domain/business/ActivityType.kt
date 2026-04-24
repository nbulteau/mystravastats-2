package me.nicolas.stravastats.domain.business

enum class ActivityType {
    AlpineSki,
    GravelRide,
    Hike,
    InlineSkate,
    MountainBikeRide,
    Ride,
    Run,
    Walk,
    Commute,
    TrailRun,
    VirtualRide
}

val rideActivities = setOf(ActivityType.Ride, ActivityType.GravelRide, ActivityType.MountainBikeRide, ActivityType.VirtualRide, ActivityType.Commute)

val runActivities = setOf(ActivityType.Run, ActivityType.TrailRun)

val hikeActivities = setOf(ActivityType.Hike, ActivityType.Walk)

fun Set<ActivityType>.representativeBadgeActivityType(): ActivityType? {
    if (isEmpty()) {
        return null
    }

    return when {
        all { it in rideActivities } -> ActivityType.Ride
        all { it in runActivities } -> ActivityType.Run
        all { it in hikeActivities } -> ActivityType.Hike
        else -> null
    }
}
