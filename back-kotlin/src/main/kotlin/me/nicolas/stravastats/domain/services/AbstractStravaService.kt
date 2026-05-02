package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import java.time.LocalDate

abstract class AbstractStravaService(
    protected val activityProvider: IActivityProvider
) {
    /**
     * Extract a sortable ISO day string (yyyy-MM-dd) from an activity date string.
     * Returns null if the value is blank or cannot be parsed.
     */
    protected fun extractSortableDay(value: String?): String? {
        if (value.isNullOrBlank()) return null
        val normalized = value.trim()
        if (normalized.length < 10) return null
        val day = normalized.substring(0, 10)
        return runCatching { LocalDate.parse(day).toString() }.getOrNull()
    }
}
