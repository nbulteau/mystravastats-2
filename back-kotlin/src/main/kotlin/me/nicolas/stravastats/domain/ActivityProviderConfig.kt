package me.nicolas.stravastats.domain

import me.nicolas.stravastats.domain.services.activityproviders.FitActivityProvider
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

        return if (fitCache == null) {
            if (stravaCache == null) {
                StravaActivityProvider()
            } else {
                StravaActivityProvider(stravaCache)
            }
        } else {
            FitActivityProvider(fitCache)
        }
    }
}