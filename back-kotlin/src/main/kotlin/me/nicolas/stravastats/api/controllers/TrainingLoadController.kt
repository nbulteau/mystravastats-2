package me.nicolas.stravastats.api.controllers

import me.nicolas.stravastats.domain.business.training.Report
import me.nicolas.stravastats.domain.services.TrainingLoadService
import org.springframework.web.bind.annotation.*
import java.time.LocalDate

@RestController
@RequestMapping("/statistics/training-load")
class TrainingLoadController(private val service: TrainingLoadService) {
    @GetMapping
    fun getTrainingLoad(@RequestParam activityType: String, @RequestParam week: String): Report {
        require(Regex("\\d{4}-\\d{2}-\\d{2}").matches(week)) { "week must be YYYY-MM-DD" }
        val date=runCatching { LocalDate.parse(week) }.getOrElse { throw IllegalArgumentException("week must be YYYY-MM-DD") }
        return service.getReport(date,activityType.convertToActivityTypeSet())
    }
}
