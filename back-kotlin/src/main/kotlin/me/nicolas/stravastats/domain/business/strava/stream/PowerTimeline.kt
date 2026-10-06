package me.nicolas.stravastats.domain.business.strava.stream

import kotlin.math.abs
import kotlin.math.pow

/** Left-held readings; no extrapolation; missing endpoints or gaps >10 s break coverage. */
class PowerTimeline(val watts: List<Int?>, val times: List<Int>) {
    companion object { const val MAX_GAP_SECONDS = 10 }
    val valid = times.size >= 2 && times.all { it >= 0 } && times.zipWithNext().all { (a,b) -> b > a }
    private val energy = DoubleArray(times.size)
    private val covered = DoubleArray(times.size)
    init {
        if (valid) for (i in 0 until times.lastIndex) {
            energy[i+1] = energy[i]; covered[i+1] = covered[i]
            if (intervalValid(i)) {
                val dt = (times[i+1]-times[i]).toDouble()
                energy[i+1] += watts[i]!! * dt; covered[i+1] += dt
            }
        }
    }
    fun intervalValid(i: Int) = valid && i >= 0 && i+1 < times.size && i+1 < watts.size &&
        times[i+1]-times[i] <= MAX_GAP_SECONDS && watts[i]?.let { it >= 0 } == true && watts[i+1]?.let { it >= 0 } == true
    private fun upper(t: Double): Int {
        var lo=0; var hi=times.size
        while(lo<hi) { val mid=(lo+hi)/2; if(times[mid].toDouble() <= t) lo=mid+1 else hi=mid }
        return lo
    }
    private fun integral(t: Double): Pair<Double,Double> {
        val i=upper(t)-1
        val dt=if(intervalValid(i)) t-times[i] else 0.0
        return energy[i]+(watts.getOrNull(i) ?: 0)*dt to covered[i]+dt
    }
    fun average(start: Double,end: Double): Double? {
        if(!valid || !start.isFinite() || !end.isFinite() || start<times.first() || end>times.last() || end<=start) return null
        val (a,c)=integral(start); val (b,d)=integral(end)
        return if(abs(d-c-(end-start))>1e-7) null else (b-a)/(end-start)
    }
    fun averageIndices(start: Int,end: Int): Double? =
        if(start<0 || end>=times.size || end<=start) null else average(times[start].toDouble(),times[end].toDouble())
    data class Window(val start: Double,val end: Double,val average: Double,val startIndex: Int,val endIndex: Int)
    fun best(seconds: Int): Window? {
        if(!valid || seconds<=0) return null
        var best: Window?=null
        for(t in times) for(start in listOf(t.toDouble(),t.toDouble()-seconds)) {
            val end=start+seconds
            val avg=average(start,end) ?: continue
            val previous=best
            if(previous==null || avg>previous.average || (avg==previous.average && start<previous.start)) {
                val upperEnd=upper(end)-1
                best=Window(start,end,avg,upper(start)-1,if(times[upperEnd].toDouble()==end) upperEnd else upperEnd+1)
            }
        }
        return best
    }
    fun valueAt(values: List<Double>,t: Double): Double {
        val i=upper(t)-1
        if(i !in values.indices) return 0.0
        if(times[i].toDouble()==t) return values[i]
        if(i+1 !in values.indices) return 0.0
        val f=(t-times[i])/(times[i+1]-times[i]);return values[i]+f*(values[i+1]-values[i])
    }
    fun normalized(): Double? {
        if(!valid) return null
        val start=times.first().toDouble()+30; val end=times.last().toDouble()
        if(start>end || average(times.first().toDouble(),end)==null) return null
        if(start==end) return average(start-30,start)
        val points=(listOf(start,end)+times.flatMap { listOf(it.toDouble(),it.toDouble()+30) }.filter { it>start && it<end }).distinct().sorted()
        var integral=0.0
        for(i in 1 until points.size) {
            val a=average(points[i-1]-30,points[i-1])!!;val b=average(points[i]-30,points[i])!!
            val fourth=(a.pow(4)+a.pow(3)*b+a*a*b*b+a*b.pow(3)+b.pow(4))/5
            integral+=(points[i]-points[i-1])*fourth
        }
        return (integral/(end-start)).pow(0.25)
    }
}
