package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider

abstract class AbstractStravaService(
    protected val activityProvider: IActivityProvider
)