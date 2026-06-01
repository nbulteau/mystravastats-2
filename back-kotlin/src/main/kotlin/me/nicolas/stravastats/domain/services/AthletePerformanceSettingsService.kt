package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.AthletePerformanceSettings
import me.nicolas.stravastats.domain.business.normalize
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.springframework.stereotype.Service

interface IAthletePerformanceSettingsService {
    fun getSettings(): AthletePerformanceSettings
    fun updateSettings(settings: AthletePerformanceSettings): AthletePerformanceSettings
}

@Service
internal class AthletePerformanceSettingsService(
    activityProvider: IActivityProvider,
) : IAthletePerformanceSettingsService, AbstractStravaService(activityProvider) {

    override fun getSettings(): AthletePerformanceSettings {
        return activityProvider.getPerformanceSettings().normalize()
    }

    override fun updateSettings(settings: AthletePerformanceSettings): AthletePerformanceSettings {
        return activityProvider.savePerformanceSettings(settings.normalize())
    }
}
