package me.nicolas.stravastats.api.dto


import com.fasterxml.jackson.annotation.JsonIgnoreProperties
import com.fasterxml.jackson.annotation.JsonProperty
import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.strava.Athlete

@Schema(description = "Athlete object", name = "Athlete")
@JsonIgnoreProperties(ignoreUnknown = true)
data class AthleteDto(
    @JsonProperty("badge_type_id")
    val badgeTypeId: Int,
    @JsonProperty("city")
    val city: String?,
    @JsonProperty("country")
    val country: String?,
    @JsonProperty("created_at")
    val createdAt: String?,
    @JsonProperty("firstname")
    val firstname: String?,
    @JsonProperty("id")
    val id: Long,
    @JsonProperty("lastname")
    val lastname: String?,
    @JsonProperty("premium")
    val premium: Boolean?,
    @JsonProperty("profile")
    val profile: String?,
    @JsonProperty("profile_medium")
    val profileMedium: String?,
    @JsonProperty("resource_state")
    val resourceState: Int?,
    @JsonProperty("sex")
    val sex: String?,
    @JsonProperty("state")
    val state: String?,
    @JsonProperty("summit")
    val summit: Boolean?,
    @JsonProperty("updated_at")
    val updatedAt: String?,
    @JsonProperty("username")
    val username: String?,
    @JsonProperty("athlete_type")
    val athleteType: Int?,
    @JsonProperty("date_preference")
    val datePreference: String?,
    @JsonProperty("follower_count")
    val followerCount: Int?,
    @JsonProperty("friend_count")
    val friendCount: Int?,
    @JsonProperty("measurement_preference")
    val measurementPreference: String?,
    @JsonProperty("mutual_friend_count")
    val mutualFriendCount: Int?,
    @JsonProperty("weight")
    val weight: Int?,
)

fun Athlete.toDto() = AthleteDto(
    badgeTypeId = 0,
    city = this.city,
    country = this.country,
    createdAt = this.createdAt,
    firstname = this.firstname,
    id = this.id,
    lastname = this.lastname,
    premium = this.premium,
    profile = this.profile,
    profileMedium = this.profileMedium,
    resourceState = this.resourceState,
    sex = this.sex,
    state = this.state,
    summit = this.summit,
    updatedAt = this.updatedAt,
    username = this.username,
    athleteType = this.athleteType,
    datePreference = this.datePreference,
    followerCount = this.followerCount,
    friendCount = this.friendCount,
    measurementPreference = this.measurementPreference,
    mutualFriendCount = this.mutualFriendCount,
    weight = this.weight
)