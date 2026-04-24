package me.nicolas.stravastats.api.configuration

import me.nicolas.stravastats.domain.RuntimeConfig
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration
import org.springframework.web.cors.CorsConfiguration
import org.springframework.web.cors.UrlBasedCorsConfigurationSource
import org.springframework.web.filter.CorsFilter

@Configuration
class CorsConfig {

    @Bean
    fun corsFilter(): CorsFilter {
        val config = CorsConfiguration()
        config.allowedOrigins = RuntimeConfig.corsAllowedOrigins()
        config.allowedMethods = RuntimeConfig.corsAllowedMethods()
        config.allowedHeaders = RuntimeConfig.corsAllowedHeaders()
        config.allowCredentials = true

        val source = UrlBasedCorsConfigurationSource()
        source.registerCorsConfiguration("/**", config)

        return CorsFilter(source)
    }
}
