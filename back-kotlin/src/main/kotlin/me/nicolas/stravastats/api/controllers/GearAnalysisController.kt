package me.nicolas.stravastats.api.controllers

import me.nicolas.stravastats.api.dto.GearAnalysisDto
import me.nicolas.stravastats.api.dto.GearMaintenanceRecordDto
import me.nicolas.stravastats.api.dto.GearMaintenanceRecordRequestDto
import me.nicolas.stravastats.api.dto.toDomain
import me.nicolas.stravastats.api.dto.toDto
import me.nicolas.stravastats.domain.services.IGearAnalysisService
import org.springframework.http.HttpStatus
import org.springframework.web.bind.annotation.DeleteMapping
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PathVariable
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.ResponseStatus
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

    @PostMapping("/maintenance")
    @ResponseStatus(HttpStatus.CREATED)
    fun saveMaintenanceRecord(@RequestBody request: GearMaintenanceRecordRequestDto): GearMaintenanceRecordDto {
        return gearAnalysisService.saveMaintenanceRecord(request.toDomain()).toDto()
    }

    @DeleteMapping("/maintenance/{recordId}")
    @ResponseStatus(HttpStatus.NO_CONTENT)
    fun deleteMaintenanceRecord(@PathVariable recordId: String) {
        gearAnalysisService.deleteMaintenanceRecord(recordId)
    }
}
