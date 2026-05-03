package me.nicolas.stravastats.api.controllers

import jakarta.servlet.http.HttpServletRequest
import me.nicolas.stravastats.domain.business.SourceModePreview
import me.nicolas.stravastats.domain.business.SourceModePreviewRequest
import me.nicolas.stravastats.domain.business.StravaOAuthStartRequest
import me.nicolas.stravastats.domain.business.StravaOAuthStartResult
import me.nicolas.stravastats.domain.services.ISourceModeService
import org.springframework.http.HttpStatus
import org.springframework.http.MediaType
import org.springframework.http.ResponseEntity
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController
import org.springframework.web.server.ResponseStatusException

@RestController
@RequestMapping("/source-modes", produces = [MediaType.APPLICATION_JSON_VALUE])
class SourceModeController(
    private val sourceModeService: ISourceModeService,
) {
    @PostMapping("/preview")
    fun preview(@RequestBody request: SourceModePreviewRequest): SourceModePreview {
        return sourceModeService.preview(request)
    }

    @PostMapping("/strava/oauth/start")
    fun startStravaOAuth(
        @RequestBody request: StravaOAuthStartRequest,
        servletRequest: HttpServletRequest,
    ): StravaOAuthStartResult {
        return try {
            sourceModeService.startStravaOAuth(request, localStravaOAuthCallbackUrl(servletRequest))
        } catch (exception: IllegalArgumentException) {
            throw ResponseStatusException(HttpStatus.BAD_REQUEST, exception.message.orEmpty(), exception)
        }
    }

    @GetMapping("/strava/oauth/callback", produces = [MediaType.TEXT_HTML_VALUE])
    fun completeStravaOAuth(
        @RequestParam state: String?,
        @RequestParam code: String?,
        @RequestParam scope: String?,
        @RequestParam error: String?,
    ): ResponseEntity<String> {
        return try {
            ResponseEntity.ok(sourceModeService.completeStravaOAuth(state, code, scope, error))
        } catch (exception: IllegalArgumentException) {
            ResponseEntity.badRequest().body(sourceModeService.stravaOAuthHtml("Authorization failed", exception.message.orEmpty()))
        } catch (exception: Exception) {
            ResponseEntity.status(502).body(sourceModeService.stravaOAuthHtml("Authorization failed", exception.message.orEmpty()))
        }
    }

    private fun localStravaOAuthCallbackUrl(request: HttpServletRequest): String {
        return "http://127.0.0.1:${request.serverPort}/api/source-modes/strava/oauth/callback"
    }
}
