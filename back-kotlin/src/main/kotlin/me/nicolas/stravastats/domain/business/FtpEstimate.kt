package me.nicolas.stravastats.domain.business

data class FtpEstimate(
    val available: Boolean = false,
    val ftp: Int = 0,
    val method: String = "",
    val methodLabel: String = "",
    val bestPower: Int = 0,
    val multiplier: Double = 0.0,
    val basedOnSeconds: Int = 0,
    val confidence: String = "unavailable",
    val source: String = "",
    val sourceKind: String = "none",
    val activityId: Long = 0,
    val activityName: String = "",
    val activityType: String = "",
    val activityDate: String = "",
    val windowDays: Int = 180,
    val activityCount: Int = 0,
)
