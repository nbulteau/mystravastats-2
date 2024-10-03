package me.nicolas.stravastats.adapter.strava

import org.springframework.boot.context.properties.ConfigurationProperties
import org.springframework.context.annotation.Configuration

@Configuration
@ConfigurationProperties(prefix = "strava")
class StravaProperties {

    var pagesize: Int = 150

    var url: String = "https://www.strava.com"
}