package me.nicolas.stravastats.api.controllers

import me.nicolas.stravastats.domain.business.DataQualityExclusionRequest
import me.nicolas.stravastats.domain.business.DataQualityCorrectionPreview
import me.nicolas.stravastats.domain.business.DataQualityReport
import me.nicolas.stravastats.domain.services.IDataQualityService
import org.springframework.web.bind.annotation.DeleteMapping
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PathVariable
import org.springframework.web.bind.annotation.PostMapping
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

    @GetMapping("/corrections/preview/{issueId}")
    fun previewCorrection(@PathVariable issueId: String): DataQualityCorrectionPreview {
        return dataQualityService.previewCorrection(issueId)
    }

    @GetMapping("/corrections/safe/preview")
    fun previewSafeCorrections(): DataQualityCorrectionPreview {
        return dataQualityService.previewSafeCorrections()
    }

    @PostMapping("/corrections/safe")
    fun applySafeCorrections(): DataQualityReport {
        return dataQualityService.applySafeCorrections()
    }

    @PostMapping("/corrections/{issueId}")
    fun applyCorrection(@PathVariable issueId: String): DataQualityReport {
        return dataQualityService.applyCorrection(issueId)
    }

    @DeleteMapping("/corrections/{correctionId}")
    fun revertCorrection(@PathVariable correctionId: String): DataQualityReport {
        return dataQualityService.revertCorrection(correctionId)
    }
}
