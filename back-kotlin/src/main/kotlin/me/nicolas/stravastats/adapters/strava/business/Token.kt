package me.nicolas.stravastats.adapters.strava.business


import com.fasterxml.jackson.annotation.JsonProperty
import me.nicolas.stravastats.domain.business.strava.StravaAthlete

data class Token(
    @param:JsonProperty("access_token")
    val accessToken: String,
    @param:JsonProperty("athlete")
    val athlete: StravaAthlete,
    @param:JsonProperty("expires_at")
    val expiresAt: Int,
    @param:JsonProperty("expires_in")
    val expiresIn: Int,
    @param:JsonProperty("refresh_token")
    val refreshToken: String,
    @param:JsonProperty("token_type")
    val tokenType: String,
)