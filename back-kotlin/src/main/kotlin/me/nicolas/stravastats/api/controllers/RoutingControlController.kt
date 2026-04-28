package me.nicolas.stravastats.api.controllers

import me.nicolas.stravastats.domain.services.routing.IOsrmControlService
import me.nicolas.stravastats.domain.services.routing.OsrmControlResult
import org.springframework.http.MediaType
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/routing/osrm", produces = [MediaType.APPLICATION_JSON_VALUE])
class RoutingControlController(
    private val osrmControlService: IOsrmControlService,
) {
    @PostMapping("/start")
    fun startOsrm(): OsrmControlResult = osrmControlService.startOsrm()
}
