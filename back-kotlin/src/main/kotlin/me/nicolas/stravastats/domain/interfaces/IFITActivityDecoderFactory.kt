package me.nicolas.stravastats.domain.interfaces

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import java.io.File

fun interface IFITActivityDecoder {
    fun decodeActivity(fitFile: File): StravaActivity?
}

fun interface IFITActivityDecoderFactory {
    fun create(path: String): IFITActivityDecoder
}
