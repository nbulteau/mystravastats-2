package me.nicolas.stravastats.domain.business.strava


import com.fasterxml.jackson.annotation.JsonIgnoreProperties
import com.fasterxml.jackson.annotation.JsonProperty

@JsonIgnoreProperties(ignoreUnknown = true)
data class StravaAthlete(
    @param:JsonProperty("badge_type_id")
    val badgeTypeId: Int? = null,
    @param:JsonProperty("city")
    val city: String? = null,
    @param:JsonProperty("country")
    val country: String? = null,
    @param:JsonProperty("created_at")
    val createdAt: String? = null,
    @param:JsonProperty("firstname")
    val firstname: String? = null,
    @param:JsonProperty("follower")
    val follower: Any? = null,
    @param:JsonProperty("friend")
    val friend: Any? = null,
    @param:JsonProperty("id")
    val id: Long,
    @param:JsonProperty("lastname")
    val lastname: String? = null,
    @param:JsonProperty("premium")
    val premium: Boolean? = null,
    @param:JsonProperty("profile")
    val profile: String? = null,
    @param:JsonProperty("profile_medium")
    val profileMedium: String? = null,
    @param:JsonProperty("resource_state")
    val resourceState: Int? = null,
    @param:JsonProperty("sex")
    val sex: String? = null,
    @param:JsonProperty("state")
    val state: String? = null,
    @param:JsonProperty("summit")
    val summit: Boolean? = null,
    @param:JsonProperty("updated_at")
    val updatedAt: String? = null,
    @param:JsonProperty("username")
    val username: String? = null,
    @param:JsonProperty("athlete_type")
    val athleteType: Int? = null,
    @param:JsonProperty("bikes")
    val bikes: List<Bike>? = null,
    @param:JsonProperty("clubs")
    val clubs: List<Any>? = null,
    @param:JsonProperty("date_preference")
    val datePreference: String? = null,
    @param:JsonProperty("follower_count")
    val followerCount: Int? = null,
    @param:JsonProperty("friend_count")
    val friendCount: Int? = null,
    @param:JsonProperty("ftp")
    val ftp: Any? = null,
    @param:JsonProperty("measurement_preference")
    val measurementPreference: String? = null,
    @param:JsonProperty("mutual_friend_count")
    val mutualFriendCount: Int? = null,
    @param:JsonProperty("shoes")
    val shoes: List<Shoe>? = null,
    @param:JsonProperty("weight")
    val weight: Int? = null,
)