package me.nicolas.stravastats.domain.services.activityproviders

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.stream.AltitudeStream
import me.nicolas.stravastats.domain.interfaces.ISRTMProvider
import org.slf4j.LoggerFactory
import java.time.LocalDate

private val logger = LoggerFactory.getLogger("ActivityProviderUtils")

/**
 * Extracts the year from a Strava ISO-8601-like date string (e.g. "2024-06-15T10:30:00Z").
 * Returns [fallback] if the string is too short or the year prefix is not a valid integer.
 */
internal fun resolveYearFromDateString(dateStr: String, fallback: Int = LocalDate.now().year): Int =
    dateStr.take(4).toIntOrNull() ?: fallback

/**
 * For each activity whose stream has no altitude data, retrieves elevation from SRTM
 * and injects an [AltitudeStream] into the activity.
 */
internal fun List<StravaActivity>.processAltitudeStreamIfMissing(srtmProvider: ISRTMProvider): List<StravaActivity> {
    logger.debug("Processing altitude stream for activities that are missing it")

    return this.map { activity ->
        val stream = activity.stream
        if (stream != null && stream.altitude == null) {
            logger.info("Enriching altitude stream for activity: ${activity.name}")
            val data = srtmProvider.getElevation(stream.latlng?.data ?: emptyList())
            activity.setStreamAltitude(AltitudeStream(data))
        } else {
            activity
        }
    }
}
