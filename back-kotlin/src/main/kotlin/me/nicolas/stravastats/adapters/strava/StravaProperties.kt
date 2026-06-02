package me.nicolas.stravastats.adapters.strava

import me.nicolas.stravastats.domain.RuntimeConfig
import org.springframework.boot.context.properties.ConfigurationProperties
import org.springframework.context.annotation.Configuration

@Configuration
@ConfigurationProperties(prefix = "strava")
class StravaProperties {

    var pageSize: Int = 200

    var url: String = "https://www.strava.com"

    var apiBaseUrl: String = RuntimeConfig.stravaApiBaseUrl()

    fun apiUrl(path: String): String {
        return "${apiBaseUrl.trimEnd('/')}/${path.trimStart('/')}"
    }
}
