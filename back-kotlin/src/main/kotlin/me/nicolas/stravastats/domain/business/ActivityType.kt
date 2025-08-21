package me.nicolas.stravastats.domain.business

enum class ActivityType {
    AlpineSki,
    GravelRide,
    Hike,
    InlineSkate,
    MountainBikeRide,
    Ride,
    Run,
    Commute,
    TrailRun,
    VirtualRide
}

val rideActivities = setOf(ActivityType.Ride, ActivityType.GravelRide, ActivityType.MountainBikeRide, ActivityType.VirtualRide, ActivityType.Commute)

val runActivities = setOf(ActivityType.Run, ActivityType.TrailRun)