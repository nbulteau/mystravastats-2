package me.nicolas.stravastats.domain

import me.nicolas.stravastats.domain.services.activityproviders.FitActivityProvider
import me.nicolas.stravastats.domain.services.activityproviders.GpxActivityProvider
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.activityproviders.StravaActivityProvider
import org.springframework.boot.ApplicationArguments
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration


@Configuration
class ActivityProviderConfig(
    private val args: ApplicationArguments,
) {

    @Bean
    fun activityProvider(): IActivityProvider {
        val stravaCache: String? = args.getOptionValues("stravaCache")?.get(0)
        val fitCache: String? = args.getOptionValues("fitCache")?.get(0)
        val gpxCache: String? = args.getOptionValues("gpxCache")?.get(0)

        // FIT Files and GPX Files are not supported yet

        return if (fitCache == null && gpxCache == null) {
            if (stravaCache == null) {
                StravaActivityProvider()
            } else {
                StravaActivityProvider(stravaCache)
            }
        } else if (fitCache != null) {
            FitActivityProvider(fitCache)
        } else if (gpxCache != null)  {
            GpxActivityProvider(gpxCache)
        } else {
            throw IllegalArgumentException("No cache provided")
        }
    }
}