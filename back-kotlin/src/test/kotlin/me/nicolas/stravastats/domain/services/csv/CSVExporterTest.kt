package me.nicolas.stravastats.domain.services.csv

import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import org.junit.jupiter.api.Test
import java.io.File
import kotlin.test.assertTrue

class CSVExporterTest {

    @Test
    fun `ride csv includes enriched columns and data quality flags`() {
        val clientId = "csv-export-test-${System.nanoTime()}"
        val outputFile = File("$clientId-Ride-all-years.csv")
        val activity = StravaActivity(
            athlete = AthleteRef(id = 1),
            averageSpeed = 5.5,
            commute = true,
            distance = 10_000.0,
            elapsedTime = 1_800,
            id = 42,
            maxSpeed = 8.0f,
            movingTime = 1_700,
            name = "Morning; ride \"easy\"",
            startDate = "2026-04-26T06:30:00Z",
            startDateLocal = "2026-04-26T08:30:00Z",
            startLatlng = null,
            totalElevationGain = 250.0,
            type = "Ride",
            uploadId = 99,
            gearId = "b123",
        )

        try {
            val csv = RideCSVExporter().export(clientId, listOf(activity), null)

            assertTrue(csv.contains("Activity ID"))
            assertTrue(csv.contains("Gear ID"))
            assertTrue(csv.contains("\"Morning; ride \"\"easy\"\"\""))
            assertTrue(csv.contains("https://www.strava.com/activities/42"))
            assertTrue(csv.contains("missing_stream"))
        } finally {
            outputFile.delete()
        }
    }
}
