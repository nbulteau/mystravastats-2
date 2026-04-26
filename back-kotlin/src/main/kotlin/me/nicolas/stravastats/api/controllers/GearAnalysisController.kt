package me.nicolas.stravastats.api.controllers

import me.nicolas.stravastats.api.dto.GearAnalysisDto
import me.nicolas.stravastats.api.dto.toDto
import me.nicolas.stravastats.domain.services.IGearAnalysisService
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/gear-analysis")
class GearAnalysisController(
    private val gearAnalysisService: IGearAnalysisService,
) {
    @GetMapping
    fun getGearAnalysis(
        @RequestParam(required = true) activityType: String,
        @RequestParam(required = false) year: Int?,
    ): GearAnalysisDto {
        val activityTypes = activityType.convertToActivityTypeSet()
        return gearAnalysisService.getGearAnalysis(activityTypes, year).toDto()
    }
}
