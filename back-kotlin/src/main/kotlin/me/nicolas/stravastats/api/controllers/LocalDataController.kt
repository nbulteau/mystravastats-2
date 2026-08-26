package me.nicolas.stravastats.api.controllers

import me.nicolas.stravastats.domain.services.ILocalDataBackupService
import me.nicolas.stravastats.domain.services.LocalDataBackup
import me.nicolas.stravastats.domain.services.LocalDataRestoreResult
import org.springframework.http.HttpHeaders
import org.springframework.http.ResponseEntity
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/local-data")
class LocalDataController(
    private val service: ILocalDataBackupService,
) {
    @GetMapping("/backup")
    fun backup(): ResponseEntity<LocalDataBackup> {
        val backup = service.export()
        return ResponseEntity.ok()
            .header(HttpHeaders.CONTENT_DISPOSITION, "attachment; filename=\"mystravastats-local-data-${backup.athleteId}.json\"")
            .body(backup)
    }

    @PostMapping("/restore")
    fun restore(@RequestBody backup: LocalDataBackup): LocalDataRestoreResult = service.restore(backup)
}
