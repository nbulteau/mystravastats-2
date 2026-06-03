package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.EddingtonNumber

@Schema(description = "Eddington number object", name = "EddingtonNumber")
data class EddingtonNumberDto(
    @param:Schema(description = "Eddington number")
    val eddingtonNumber: Int,
    @param:Schema(description = "Eddington list")
    val eddingtonList: List<Int>,
    @param:Schema(description = "Eddington scope")
    val scope: String,
    @param:Schema(description = "Eddington metric")
    val metric: String,
    @param:Schema(description = "Eddington basis")
    val basis: String,
    @param:Schema(description = "Eddington threshold unit")
    val unit: String,
    @param:Schema(description = "Next Eddington target")
    val nextTarget: Int,
    @param:Schema(description = "Items already qualifying for the next target")
    val qualifyingCount: Int,
    @param:Schema(description = "Additional qualifying items required for the next target")
    val missingCount: Int,
    @param:Schema(description = "Days already qualifying for the next target")
    val qualifyingDays: Int,
    @param:Schema(description = "Additional qualifying days required for the next target")
    val missingDays: Int,
)

fun EddingtonNumber.toDto() = EddingtonNumberDto(
    eddingtonNumber = this.eddingtonNumber,
    eddingtonList = this.eddingtonList,
    scope = this.scope.apiValue,
    metric = this.metric.apiValue,
    basis = this.basis.apiValue,
    unit = this.unit,
    nextTarget = this.nextTarget,
    qualifyingCount = this.qualifyingCount,
    missingCount = this.missingCount,
    qualifyingDays = this.qualifyingDays,
    missingDays = this.missingDays,
)
