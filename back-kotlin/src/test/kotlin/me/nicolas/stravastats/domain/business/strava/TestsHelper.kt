package me.nicolas.stravastats.domain.business.strava

import com.fasterxml.jackson.module.kotlin.jacksonObjectMapper
import java.io.File


fun loadColAgnelActivity(): StravaActivity {
    val objectMapper = jacksonObjectMapper()
    var url = Thread.currentThread().contextClassLoader.getResource("colagnel-activity.json")
    var jsonFile = File(url!!.path)
    val stravaActivity = objectMapper.readValue(jsonFile, StravaActivity::class.java)

    url = Thread.currentThread().contextClassLoader.getResource("colagnel-stream.json")
    jsonFile = File(url!!.path)
    stravaActivity.stream = objectMapper.readValue(jsonFile, Stream::class.java)

    return stravaActivity
}

fun loadActivity(name: String): StravaActivity {
    val objectMapper = jacksonObjectMapper()
    var url = Thread.currentThread().contextClassLoader.getResource(name)
    var jsonFile = File(url!!.path)
    val stravaActivity = objectMapper.readValue(jsonFile, StravaActivity::class.java)

    url = Thread.currentThread().contextClassLoader.getResource("stream-$name")
    jsonFile = File(url!!.path)
    stravaActivity.stream = objectMapper.readValue(jsonFile, Stream::class.java)

    return stravaActivity
}

fun loadZwiftActivity(): StravaActivity {
    val objectMapper = jacksonObjectMapper()
    var url = Thread.currentThread().contextClassLoader.getResource("zwift-activity.json")
    var jsonFile = File(url!!.path)
    val stravaActivity = objectMapper.readValue(jsonFile, StravaActivity::class.java)

    url = Thread.currentThread().contextClassLoader.getResource("zwift-stream.json")
    jsonFile = File(url!!.path)
    stravaActivity.stream = objectMapper.readValue(jsonFile, Stream::class.java)

    return stravaActivity
}







