package me.nicolas.stravastats.domain.business.strava


import com.fasterxml.jackson.annotation.JsonIgnoreProperties
import com.fasterxml.jackson.annotation.JsonProperty

@JsonIgnoreProperties(ignoreUnknown = true)
data class Athlete(
    @JsonProperty("badge_type_id")
    val badgeTypeId: Int? = null,
    @JsonProperty("city")
    val city: String? = null,
    @JsonProperty("country")
    val country: String? = null,
    @JsonProperty("created_at")
    val createdAt: String? = null,
    @JsonProperty("firstname")
    val firstname: String? = null,
    @JsonProperty("follower")
    val follower: Any? = null,
    @JsonProperty("friend")
    val friend: Any? = null,
    @JsonProperty("id")
    val id: Long,
    @JsonProperty("lastname")
    val lastname: String? = null,
    @JsonProperty("premium")
    val premium: Boolean? = null,
    @JsonProperty("profile")
    val profile: String? = null,
    @JsonProperty("profile_medium")
    val profileMedium: String? = null,
    @JsonProperty("resource_state")
    val resourceState: Int? = null,
    @JsonProperty("sex")
    val sex: String? = null,
    @JsonProperty("state")
    val state: String? = null,
    @JsonProperty("summit")
    val summit: Boolean? = null,
    @JsonProperty("updated_at")
    val updatedAt: String? = null,
    @JsonProperty("username")
    val username: String? = null,
    @JsonProperty("athlete_type")
    val athleteType: Int? = null,
    @JsonProperty("bikes")
    val bikes: List<Bike>? = null,
    @JsonProperty("clubs")
    val clubs: List<Any>? = null,
    @JsonProperty("date_preference")
    val datePreference: String? = null,
    @JsonProperty("follower_count")
    val followerCount: Int? = null,
    @JsonProperty("friend_count")
    val friendCount: Int? = null,
    @JsonProperty("ftp")
    val ftp: Any? = null,
    @JsonProperty("measurement_preference")
    val measurementPreference: String? = null,
    @JsonProperty("mutual_friend_count")
    val mutualFriendCount: Int? = null,
    @JsonProperty("shoes")
    val shoes: List<Shoe>? = null,
    @JsonProperty("weight")
    val weight: Int? = null,
)