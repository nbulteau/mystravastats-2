package me.nicolas.stravastats

import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.ArraySchema
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import io.swagger.v3.oas.annotations.tags.Tag
import me.nicolas.stravastats.api.dto.PersonalRecordTimelineDto
import me.nicolas.stravastats.domain.business.PersonalRecordTimelineEntry
import me.nicolas.stravastats.domain.business.strava.Achievement
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.Bike
import me.nicolas.stravastats.domain.business.strava.Gear
import me.nicolas.stravastats.domain.business.strava.GeoCoordinate
import me.nicolas.stravastats.domain.business.strava.GeoMap
import me.nicolas.stravastats.domain.business.strava.MetaActivity
import me.nicolas.stravastats.domain.business.strava.MetaAthlete
import me.nicolas.stravastats.domain.business.strava.Segment
import me.nicolas.stravastats.domain.business.strava.Shoe
import me.nicolas.stravastats.domain.business.strava.SplitsMetric
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.StravaSegmentEffort
import me.nicolas.stravastats.domain.business.strava.stream.AltitudeStream
import me.nicolas.stravastats.domain.business.strava.stream.CadenceStream
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.HeartRateStream
import me.nicolas.stravastats.domain.business.strava.stream.LatLngStream
import me.nicolas.stravastats.domain.business.strava.stream.MovingStream
import me.nicolas.stravastats.domain.business.strava.stream.PowerStream
import me.nicolas.stravastats.domain.business.strava.stream.SmoothGradeStream
import me.nicolas.stravastats.domain.business.strava.stream.SmoothVelocityStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import me.nicolas.stravastats.domain.business.badges.Alternative
import me.nicolas.stravastats.domain.business.badges.FamousClimb
import org.springframework.aot.hint.annotation.RegisterReflectionForBinding
import org.springframework.boot.autoconfigure.SpringBootApplication
import org.springframework.boot.runApplication

@SpringBootApplication
@RegisterReflectionForBinding(
    classes = [
        Achievement::class,
        AthleteRef::class,
        Bike::class,
        Gear::class,
        GeoCoordinate::class,
        GeoMap::class,
        MetaActivity::class,
        MetaAthlete::class,
        Segment::class,
        Shoe::class,
        SplitsMetric::class,
        StravaActivity::class,
        StravaAthlete::class,
        StravaDetailedActivity::class,
        StravaSegmentEffort::class,
        Stream::class,
        AltitudeStream::class,
        CadenceStream::class,
        DistanceStream::class,
        HeartRateStream::class,
        LatLngStream::class,
        MovingStream::class,
        PowerStream::class,
        SmoothGradeStream::class,
        SmoothVelocityStream::class,
        TimeStream::class,
        FamousClimb::class,
        Alternative::class,
        Operation::class,
        ArraySchema::class,
        Content::class,
        Schema::class,
        Schema.AccessMode::class,
        Schema.AdditionalPropertiesValue::class,
        Schema.RequiredMode::class,
        Schema.SchemaResolution::class,
        ApiResponse::class,
        Tag::class,
        PersonalRecordTimelineEntry::class,
        PersonalRecordTimelineDto::class,
    ]
)
class StravastatsApplication

fun main(args: Array<String>) {
    runApplication<StravastatsApplication>(*args)
}
