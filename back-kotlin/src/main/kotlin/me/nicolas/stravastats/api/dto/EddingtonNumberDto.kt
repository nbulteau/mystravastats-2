package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.EddingtonNumber

@Schema(description = "Eddington number object", name = "EddingtonNumber")
data class EddingtonNumberDto(
    @Schema(description = "Eddington number")
    val eddingtonNumber: Int,
    @Schema(description = "Eddington list")
    val eddingtonList: List<Int>,
)

fun EddingtonNumber.toDto() = EddingtonNumberDto(this.eddingtonNumber, this.eddingtonList)