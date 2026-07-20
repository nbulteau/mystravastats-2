package me.nicolas.stravastats.adapters.localrepositories.fit

import com.garmin.fit.Sport
import com.garmin.fit.SubSport
import me.nicolas.stravastats.domain.business.ActivityType
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test

class FITActivityTypeTest {
    @Test
    fun `extractFITActivityClassification uses sport and sub sport`() {
        val cases = listOf(
            Case("generic cycling", Sport.CYCLING, SubSport.GENERIC, FITActivityClassification(ActivityType.Ride.name)),
            Case("mountain cycling", Sport.CYCLING, SubSport.MOUNTAIN, FITActivityClassification(ActivityType.MountainBikeRide.name)),
            Case("e-bike mountain cycling", Sport.CYCLING, SubSport.E_BIKE_MOUNTAIN, FITActivityClassification(ActivityType.MountainBikeRide.name)),
            Case("gravel cycling", Sport.CYCLING, SubSport.GRAVEL_CYCLING, FITActivityClassification(ActivityType.GravelRide.name)),
            Case("mixed surface cycling", Sport.CYCLING, SubSport.MIXED_SURFACE, FITActivityClassification(ActivityType.GravelRide.name)),
            Case("virtual cycling", Sport.CYCLING, SubSport.VIRTUAL_ACTIVITY, FITActivityClassification(ActivityType.VirtualRide.name)),
            Case("indoor cycling", Sport.CYCLING, SubSport.INDOOR_CYCLING, FITActivityClassification(ActivityType.VirtualRide.name)),
            Case("fitness equipment indoor cycling", Sport.FITNESS_EQUIPMENT, SubSport.INDOOR_CYCLING, FITActivityClassification(ActivityType.VirtualRide.name)),
            Case("cycling commute", Sport.CYCLING, SubSport.COMMUTING, FITActivityClassification(ActivityType.Commute.name, ActivityType.Ride.name, true)),
            Case("generic running", Sport.RUNNING, SubSport.GENERIC, FITActivityClassification(ActivityType.Run.name)),
            Case("trail running", Sport.RUNNING, SubSport.TRAIL, FITActivityClassification(ActivityType.TrailRun.name)),
            Case("fitness equipment treadmill", Sport.FITNESS_EQUIPMENT, SubSport.TREADMILL, FITActivityClassification(ActivityType.Run.name)),
            Case("walking", Sport.WALKING, SubSport.GENERIC, FITActivityClassification(ActivityType.Walk.name)),
            Case("fitness equipment indoor walking", Sport.FITNESS_EQUIPMENT, SubSport.INDOOR_WALKING, FITActivityClassification(ActivityType.Walk.name)),
            Case("hiking", Sport.HIKING, SubSport.GENERIC, FITActivityClassification(ActivityType.Hike.name)),
            Case("mountaineering", Sport.MOUNTAINEERING, SubSport.GENERIC, FITActivityClassification(ActivityType.Hike.name)),
            Case("alpine skiing", Sport.ALPINE_SKIING, SubSport.GENERIC, FITActivityClassification(ActivityType.AlpineSki.name)),
            Case("inline skating", Sport.INLINE_SKATING, SubSport.GENERIC, FITActivityClassification(ActivityType.InlineSkate.name)),
            Case("generic e-biking", Sport.E_BIKING, SubSport.GENERIC, FITActivityClassification(ActivityType.Ride.name)),
            Case("virtual e-biking", Sport.E_BIKING, SubSport.VIRTUAL_ACTIVITY, FITActivityClassification(ActivityType.VirtualRide.name)),
            Case("unknown fallback", Sport.INVALID, SubSport.INVALID, FITActivityClassification(ActivityType.Ride.name)),
        )

        cases.forEach { case ->
            assertEquals(
                case.expected,
                extractFITActivityClassification(case.sport, case.subSport),
                case.name,
            )
        }

        val coveredTypes = cases.map { case -> case.expected.type }.toSet()
        ActivityType.entries.forEach { activityType ->
            assertEquals(true, coveredTypes.contains(activityType.name), "FIT mapping coverage for ${activityType.name}")
        }
    }

    private data class Case(
        val name: String,
        val sport: Sport,
        val subSport: SubSport,
        val expected: FITActivityClassification,
    )
}
