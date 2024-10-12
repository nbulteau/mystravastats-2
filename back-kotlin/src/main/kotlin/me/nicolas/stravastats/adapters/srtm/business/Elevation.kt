package me.nicolas.stravastats.adapters.srtm.business

/**
 *
 * The Elevation object contains elevation data along with the actual point data came from.
 *
 */
data class Elevation(val elevation: Double, val point: Point)