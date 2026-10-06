package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.AthletePerformanceSettings
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.stream.PowerTimeline
import me.nicolas.stravastats.domain.business.training.*
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.springframework.stereotype.Service
import java.time.LocalDate
import kotlin.math.abs

@Service
class TrainingLoadService(private val provider: IActivityProvider) {
    fun getReport(week: LocalDate, types: Set<ActivityType>): Report = TrainingReports.build(
        week, provider.getActivitiesByActivityTypeAndYear(types).withoutDataQualityExcludedStats(provider), provider.getPerformanceSettings(),
    )
}

object TrainingReports {
    private fun parseDate(raw: String): LocalDate? = runCatching { LocalDate.parse(raw.take(10)) }.getOrNull()
    private fun clean(value: Double) = if(value.isFinite() && value >= 0) value else 0.0
    fun build(day: LocalDate, activities: List<StravaActivity>, settings: AthletePerformanceSettings): Report {
        val monday = day.minusDays(day.dayOfWeek.value.toLong()-1)
        var undated = 0
        val rows = activities.mapNotNull { activity ->
            val local = parseDate(activity.startDateLocal.ifEmpty { activity.startDate })
            if(local == null) { undated++; null }
            else if(local < monday.minusDays(28) || local >= monday.plusDays(7)) null
            else calculate(activity,local.toString(),settings)
        }.sortedWith(compareByDescending<LoadActivity> { it.date }.thenByDescending { it.activityId })
        val weeks = (0..4).map { offset ->
            val start = monday.minusDays(offset*7L)
            LoadWeek(period(start,start.plusDays(6),rows), (0..6).map { period(start.plusDays(it.toLong()),start.plusDays(it.toLong()),rows) })
        }
        return Report("power-duration-v1",PowerTimeline.MAX_GAP_SECONDS,undated,weeks,rows)
    }
    private fun calculate(a: StravaActivity, date: String, settings: AthletePerformanceSettings): LoadActivity {
        val row = LoadActivity(activityId=a.id,name=a.name,date=date,sport=a.type,source=if(a.deviceWatts) "measured" else "estimated",
            movingSeconds=maxOf(0,a.movingTime),elapsedSeconds=maxOf(0,a.elapsedTime),distanceMeters=clean(a.distance),elevationMeters=clean(a.totalElevationGain))
        if(a.stream?.watts?.data.isNullOrEmpty()) row.source="unavailable"
        for(entry in settings.ftpHistory) {
            if(Regex("\\d{4}-\\d{2}-\\d{2}").matches(entry.effectiveFrom) && parseDate(entry.effectiveFrom)!=null && entry.ftp>0 && entry.effectiveFrom<=date && (row.ftpEffectiveFrom==null || entry.effectiveFrom>=row.ftpEffectiveFrom!!)) {
                row.ftp=entry.ftp;row.ftpEffectiveFrom=entry.effectiveFrom
            }
        }
        if(a.type !in setOf("Ride","VirtualRide","MountainBikeRide","GravelRide","Commute")) {row.reason="unsupported-sport";return row}
        val stream=a.stream
        if(stream?.watts?.data.isNullOrEmpty()) {row.source="unavailable";row.reason="missing-power";return row}
        val timeline=PowerTimeline(stream.watts.data,stream.time.data)
        if(!timeline.valid) {row.reason="invalid-time";return row}
        val times=timeline.times
        row.recordedSeconds=(times.last()-times.first()).toDouble()
        for(i in 0 until times.lastIndex) if(timeline.intervalValid(i)) {
            val seconds=(times[i+1]-times[i]).toDouble()
            row.coveredSeconds+=seconds
            row.ftp?.let { ftp ->
                val power=timeline.watts[i]!!
                if(power<=ftp*0.9) row.aerobicSeconds+=seconds
                else if(power<=ftp*1.2) row.thresholdSeconds+=seconds
                else row.highIntensitySeconds+=seconds
            }
        }
        row.best5MinutePower=timeline.best(300)?.average
        row.normalizedPower=timeline.normalized()
        row.reason=when {
            row.coveredSeconds<row.recordedSeconds -> "power-gaps"
            row.recordedSeconds<30 -> "too-short"
            times.first()!=0 || a.elapsedTime<=0 || abs(times.last().toDouble()-a.elapsedTime)>1 -> "incomplete-activity"
            row.ftp==null -> "missing-dated-ftp"
            row.normalizedPower==null -> "missing-power"
            else -> {
                val intensity=row.normalizedPower!!/row.ftp!!
                row.intensityFactor=intensity;row.load=row.coveredSeconds/3600*intensity*intensity*100
                "available"
            }
        }
        return row
    }
    private fun period(start: LocalDate,end: LocalDate,rows: List<LoadActivity>): LoadPeriod {
        val p=LoadPeriod(startDate=start.toString(),endDate=end.toString())
        for(row in rows.filter { it.date>=p.startDate && it.date<=p.endDate }) {
            p.activityCount++;p.movingSeconds+=row.movingSeconds;p.distanceMeters+=row.distanceMeters;p.elevationMeters+=row.elevationMeters
            p.coveredSeconds+=row.coveredSeconds;p.recordedSeconds+=row.recordedSeconds
            row.load?.let { load ->
                p.scoredCount++
                if(row.source=="measured") p.measuredLoad=(p.measuredLoad ?: 0.0)+load else p.estimatedLoad=(p.estimatedLoad ?: 0.0)+load
            }
            if(row.source=="measured") {
                p.aerobicSeconds+=row.aerobicSeconds;p.thresholdSeconds+=row.thresholdSeconds;p.highIntensitySeconds+=row.highIntensitySeconds
                row.best5MinutePower?.let { power -> if(p.best5MinutePower==null || power>p.best5MinutePower!!) {p.best5MinutePower=power;p.best5MinuteActivityId=row.activityId} }
            }
        }
        return p
    }
}
