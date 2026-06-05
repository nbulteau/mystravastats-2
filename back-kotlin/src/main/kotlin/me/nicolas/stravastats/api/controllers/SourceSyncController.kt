package me.nicolas.stravastats.api.controllers

import me.nicolas.stravastats.domain.business.SourceSyncResult
import me.nicolas.stravastats.domain.services.ISourceSyncService
import org.springframework.http.MediaType
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/source-sync", produces = [MediaType.APPLICATION_JSON_VALUE])
class SourceSyncController(
    private val sourceSyncService: ISourceSyncService,
) {
    @PostMapping("/synchronize")
    fun synchronize(): SourceSyncResult {
        return sourceSyncService.synchronize("manual")
    }
}
