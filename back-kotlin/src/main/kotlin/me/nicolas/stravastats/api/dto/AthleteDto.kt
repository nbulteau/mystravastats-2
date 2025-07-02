package me.nicolas.stravastats.api.dto


import com.fasterxml.jackson.annotation.JsonIgnoreProperties
import com.fasterxml.jackson.annotation.JsonProperty
import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.strava.StravaAthlete

@Schema(description = "StravaAthlete object", name = "StravaAthlete")
@JsonIgnoreProperties(ignoreUnknown = true)
data class AthleteDto(
    @param:JsonProperty("badge_type_id")
    val badgeTypeId: Int,
    @param:JsonProperty("city")
    val city: String?,
    @param:JsonProperty("country")
    val country: String?,
    @param:JsonProperty("created_at")
    val createdAt: String?,
    @param:JsonProperty("firstname")
    val firstname: String?,
    @param:JsonProperty("id")
    val id: Long,
    @param:JsonProperty("lastname")
    val lastname: String?,
    @param:JsonProperty("premium")
    val premium: Boolean?,
    @param:JsonProperty("profile")
    val profile: String?,
    @param:JsonProperty("profile_medium")
    val profileMedium: String?,
    @param:JsonProperty("resource_state")
    val resourceState: Int?,
    @param:JsonProperty("sex")
    val sex: String?,
    @param:JsonProperty("state")
    val state: String?,
    @param:JsonProperty("summit")
    val summit: Boolean?,
    @param:JsonProperty("updated_at")
    val updatedAt: String?,
    @param:JsonProperty("username")
    val username: String?,
    @param:JsonProperty("athlete_type")
    val athleteType: Int?,
    @param:JsonProperty("date_preference")
    val datePreference: String?,
    @param:JsonProperty("follower_count")
    val followerCount: Int?,
    @param:JsonProperty("friend_count")
    val friendCount: Int?,
    @param:JsonProperty("measurement_preference")
    val measurementPreference: String?,
    @param:JsonProperty("mutual_friend_count")
    val mutualFriendCount: Int?,
    @param:JsonProperty("weight")
    val weight: Int?,
)

fun StravaAthlete.toDto() = AthleteDto(
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