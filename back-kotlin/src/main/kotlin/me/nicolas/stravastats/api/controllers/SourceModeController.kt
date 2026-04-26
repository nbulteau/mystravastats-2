package me.nicolas.stravastats.api.controllers

import me.nicolas.stravastats.domain.business.SourceModePreview
import me.nicolas.stravastats.domain.business.SourceModePreviewRequest
import me.nicolas.stravastats.domain.services.ISourceModeService
import org.springframework.http.MediaType
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/source-modes", produces = [MediaType.APPLICATION_JSON_VALUE])
class SourceModeController(
    private val sourceModeService: ISourceModeService,
) {
    @PostMapping("/preview")
    fun preview(@RequestBody request: SourceModePreviewRequest): SourceModePreview {
        return sourceModeService.preview(request)
    }
}
