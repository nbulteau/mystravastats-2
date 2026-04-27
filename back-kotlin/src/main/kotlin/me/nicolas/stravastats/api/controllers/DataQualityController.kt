package me.nicolas.stravastats.api.controllers

import me.nicolas.stravastats.domain.business.DataQualityExclusionRequest
import me.nicolas.stravastats.domain.business.DataQualityReport
import me.nicolas.stravastats.domain.services.IDataQualityService
import org.springframework.web.bind.annotation.DeleteMapping
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PathVariable
import org.springframework.web.bind.annotation.PutMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/data-quality")
class DataQualityController(
    private val dataQualityService: IDataQualityService,
) {
    @GetMapping("/issues")
    fun getIssues(): DataQualityReport = dataQualityService.getReport()

    @PutMapping("/exclusions/{activityId}")
    fun excludeActivityFromStats(
        @PathVariable activityId: Long,
        @RequestBody(required = false) request: DataQualityExclusionRequest?,
    ): DataQualityReport {
        return dataQualityService.excludeActivityFromStats(activityId, request?.reason)
    }

    @DeleteMapping("/exclusions/{activityId}")
    fun includeActivityInStats(@PathVariable activityId: Long): DataQualityReport {
        return dataQualityService.includeActivityInStats(activityId)
    }
}
