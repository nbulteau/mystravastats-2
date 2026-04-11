package me.nicolas.stravastats.domain.services.activityproviders

import me.nicolas.stravastats.adapters.srtm.SRTMProvider
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.stream.AltitudeStream
import org.slf4j.LoggerFactory

private val logger = LoggerFactory.getLogger("ActivityProviderUtils")

/**
 * For each activity whose stream has no altitude data, retrieves elevation from SRTM
 * and injects an [AltitudeStream] into the activity.
 */
internal fun List<StravaActivity>.processAltitudeStreamIfMissing(srtmProvider: SRTMProvider): List<StravaActivity> {
    logger.debug("Processing altitude stream for activities that are missing it")

    return this.map { activity ->
        if (activity.stream != null && activity.stream?.altitude == null) {
            logger.info("Enriching altitude stream for activity: ${activity.name}")
            val data = srtmProvider.getElevation(activity.stream?.latlng?.data ?: emptyList())
            activity.setStreamAltitude(AltitudeStream(data))
        } else {
            activity
        }
    }
}

