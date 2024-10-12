package me.nicolas.stravastats.adapters.strava.business


import com.fasterxml.jackson.annotation.JsonProperty
import me.nicolas.stravastats.domain.business.strava.Athlete

data class Token(
    @JsonProperty("access_token")
    val accessToken: String,
    @JsonProperty("athlete")
    val athlete: Athlete,
    @JsonProperty("expires_at")
    val expiresAt: Int,
    @JsonProperty("expires_in")
    val expiresIn: Int,
    @JsonProperty("refresh_token")
    val refreshToken: String,
    @JsonProperty("token_type")
    val tokenType: String,
)