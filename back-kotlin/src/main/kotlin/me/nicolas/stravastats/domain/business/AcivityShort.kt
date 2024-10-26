package me.nicolas.stravastats.domain.business

data class ActivityShort(
    val id: Long,
    val name: String,
    val type: ActivityType,
) {
    constructor(id: Long, name: String, type: String) : this(id, name, ActivityType.valueOf(type))
}
