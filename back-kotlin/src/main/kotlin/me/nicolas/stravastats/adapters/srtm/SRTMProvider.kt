package me.nicolas.stravastats.adapters.srtm

import me.nicolas.stravastats.adapters.srtm.business.Point
import me.nicolas.stravastats.adapters.srtm.business.SRTMFile
import me.nicolas.stravastats.domain.interfaces.ISRTMProvider
import me.nicolas.stravastats.domain.services.ActivityHelper.smooth
import org.slf4j.LoggerFactory
import java.io.File
import java.nio.file.Path
import kotlin.math.abs
import kotlin.math.floor


/**
 * Get elevation using Shuttle Radar Topography Mission (SRTM) level 1 files
 */
class SRTMProvider(
    private val cachePath: Path = Path.of("srtm30m")
) : ISRTMProvider {

    private val logger = LoggerFactory.getLogger(SRTMProvider::class.java)
    /**
     * SRTM files cache
     */
    private val srtmTilesCache = mutableMapOf<String, SRTMFile>()

    /**
     * Missing SRTM files to prevent multiple
     */
    private val missingSRTMFiles = mutableSetOf<String>()

    override fun getElevation(latitudeLongitudeList: List<List<Double>>): List<Double> {

        return latitudeLongitudeList.map { latlong ->
            if (latlong[0] == 0.0 && latlong[1] == 0.0) {
                0.0
            } else {
                getElevation(latlong[0], latlong[1])
            }
        }.smooth()
    }

    override fun isAvailable(): Boolean {
        return cachePath.toFile().exists()
    }

    private fun getElevation(latitude: Double, longitude: Double): Double {

        // get SRTM tile
        val srtmFileName = getTileFileName(latitude, longitude)

        // SRTM file is missing
        if (missingSRTMFiles.contains(srtmFileName)) {
            return 0.0
        }

        var srtmFile: SRTMFile? = null
        if (srtmTilesCache.contains(srtmFileName)) {
            srtmFile = srtmTilesCache[srtmFileName]!!
        } else {
            try {
                srtmFile = SRTMFile(File(cachePath.toFile(), "$srtmFileName.hgt"))
                srtmTilesCache[srtmFileName] = srtmFile
            } catch (instantiationException: InstantiationException) {
                logger.info("Download $srtmFileName.hgt from https://dwtkns.com/srtm30m/")
                missingSRTMFiles.add(srtmFileName)
            }
        }

        val point = Point(latitude, longitude)
        return if (srtmFile?.contains(point) == true) {
            srtmFile.getElevation(point).elevation
        } else {
            0.0
        }
    }

    private fun getTileFileName(lat: Double, lng: Double): String {
        val latitude = lat.toDegreesMinutesAndSeconds().first
        val latitudeCardinal = if (lat >= 0) "N" else "S"

        val longitude = lng.toDegreesMinutesAndSeconds().first + 1
        val longitudeCardinal = if (lng >= 0) "E" else "W"

        return "$latitudeCardinal$latitude$longitudeCardinal${longitude.toString().padStart(3, '0')}"
    }

    private fun Double.toDegreesMinutesAndSeconds(): Triple<Int, Int, Int> {
        val absolute = abs(this)
        val degrees = floor(absolute)
        val minutesNotTruncated = (absolute - degrees) * 60
        val minutes = floor(minutesNotTruncated)
        val seconds = floor((minutesNotTruncated - minutes) * 60)

        return Triple(degrees.toInt(), minutes.toInt(), seconds.toInt())
    }
}
