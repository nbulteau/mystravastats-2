package me.nicolas.stravastats.domain.business.strava

import com.fasterxml.jackson.annotation.JsonProperty

data class StravaSegmentEffort(
    val achievements: List<Achievement>,
    val activity: MetaActivity,
    val athlete: MetaAthlete,
    @param:JsonProperty("average_cadence")
    val averageCadence: Double,
    @param:JsonProperty("average_heartrate")
    val averageHeartRate: Double,
    @param:JsonProperty("average_watts")
    val averageWatts: Double,
    @param:JsonProperty("device_watts")
    val deviceWatts: Boolean,
    val distance: Double,
    @param:JsonProperty("elapsed_time")
    val elapsedTime: Int,
    @param:JsonProperty("end_index")
    val endIndex: Int,
    val hidden: Boolean,
    val id: Long,
    @param:JsonProperty("kom_rank")
    val komRank: Int?,
    @param:JsonProperty("max_heartrate")
    val maxHeartRate: Double,
    @param:JsonProperty("moving_time")
    val movingTime: Int,
    val name: String,
    @param:JsonProperty("pr_rank")
    val prRank: Int?,
    @param:JsonProperty("resource_state")
    val resourceState: Int,
    val segment: Segment,
    @param:JsonProperty("start_date")
    val startDate: String,
    @param:JsonProperty("start_date_local")
    val startDateLocal: String,
    @param:JsonProperty("start_index")
    val startIndex: Int,
    val visibility: String?,
) {
    override fun equals(other: Any?): Boolean {
        if (this === other) return true
        if (javaClass != other?.javaClass) return false

        other as StravaSegmentEffort

        if (endIndex != other.endIndex) return false
        if (name != other.name) return false
        return startIndex == other.startIndex
    }

    override fun hashCode(): Int {
        var result = endIndex
        result = 31 * result + name.hashCode()
        result = 31 * result + startIndex
        return result
    }


}