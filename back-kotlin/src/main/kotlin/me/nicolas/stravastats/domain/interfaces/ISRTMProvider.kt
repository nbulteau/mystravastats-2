package me.nicolas.stravastats.domain.interfaces

interface ISRTMProvider {

    fun getElevation(latitudeLongitudeList: List<List<Double>>): List<Double>

    fun isAvailable(): Boolean
}
