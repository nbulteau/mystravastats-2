package me.nicolas.stravastats.adapters.localrepositories.fit

import com.garmin.fit.Sport
import com.garmin.fit.SubSport
import me.nicolas.stravastats.domain.business.ActivityType
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test

class FITActivityTypeTest {
    @Test
    fun `extractFITActivityType uses sport and sub sport`() {
        val cases = listOf(
            Case("generic cycling", Sport.CYCLING, SubSport.GENERIC, ActivityType.Ride),
            Case("mountain cycling", Sport.CYCLING, SubSport.MOUNTAIN, ActivityType.MountainBikeRide),
            Case("gravel cycling", Sport.CYCLING, SubSport.GRAVEL_CYCLING, ActivityType.GravelRide),
            Case("mixed surface cycling", Sport.CYCLING, SubSport.MIXED_SURFACE, ActivityType.GravelRide),
            Case("virtual cycling", Sport.CYCLING, SubSport.VIRTUAL_ACTIVITY, ActivityType.VirtualRide),
            Case("trail running", Sport.RUNNING, SubSport.TRAIL, ActivityType.TrailRun),
            Case("walking", Sport.WALKING, SubSport.GENERIC, ActivityType.Walk),
            Case("hiking", Sport.HIKING, SubSport.GENERIC, ActivityType.Hike),
            Case("unknown fallback", Sport.INVALID, SubSport.INVALID, ActivityType.Ride),
        )

        cases.forEach { case ->
            assertEquals(
                case.expected.name,
                extractFITActivityType(case.sport, case.subSport),
                case.name,
            )
        }
    }

    private data class Case(
        val name: String,
        val sport: Sport,
        val subSport: SubSport,
        val expected: ActivityType,
    )
}
