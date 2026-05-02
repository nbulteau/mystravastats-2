//go:build ignore

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/tormoder/fit"
)

type sample struct {
	seconds  int
	lat      float64
	lng      float64
	altitude float64
	distance float64
	speed    float64
	hr       uint8
	cadence  uint8
	power    uint16
}

func main() {
	out := flag.String("out", filepath.Join("test-fixtures", "source-modes", "fit", "2026", "smoke-ride.fit"), "output FIT fixture path")
	flag.Parse()

	if err := writeFITFixture(*out); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "generate FIT fixture: %v\n", err)
		os.Exit(1)
	}
}

func writeFITFixture(out string) error {
	start := time.Date(2026, time.January, 15, 7, 30, 0, 0, time.UTC)
	points := []sample{
		{seconds: 0, lat: 48.1173, lng: -1.6778, altitude: 36, distance: 0, speed: 0, hr: 128, cadence: 80, power: 165},
		{seconds: 75, lat: 48.1181, lng: -1.6748, altitude: 38, distance: 342.2, speed: 4.56, hr: 130, cadence: 82, power: 172},
		{seconds: 150, lat: 48.1190, lng: -1.6717, altitude: 41, distance: 695.4, speed: 4.71, hr: 134, cadence: 84, power: 188},
		{seconds: 225, lat: 48.1201, lng: -1.6689, altitude: 43, distance: 1003.1, speed: 4.10, hr: 140, cadence: 83, power: 205},
		{seconds: 300, lat: 48.1210, lng: -1.6662, altitude: 42, distance: 1306.7, speed: 4.05, hr: 136, cadence: 81, power: 174},
	}

	header := fit.NewHeader(fit.V20, false)
	file, err := fit.NewFile(fit.FileTypeActivity, header)
	if err != nil {
		return err
	}
	file.FileId.Manufacturer = fit.ManufacturerDevelopment
	file.FileId.Product = 1
	file.FileId.SerialNumber = 2601002
	file.FileId.TimeCreated = start

	activity, err := file.Activity()
	if err != nil {
		return err
	}

	startEvent := fit.NewEventMsg()
	startEvent.Timestamp = start
	startEvent.Event = fit.EventTimer
	startEvent.EventType = fit.EventTypeStart
	activity.Events = append(activity.Events, startEvent)

	for _, point := range points {
		record := fit.NewRecordMsg()
		record.Timestamp = start.Add(time.Duration(point.seconds) * time.Second)
		record.PositionLat = fit.NewLatitudeDegrees(point.lat)
		record.PositionLong = fit.NewLongitudeDegrees(point.lng)
		record.Distance = scaledUint32(point.distance, 100)
		record.Altitude = scaledAltitude16(point.altitude)
		record.EnhancedAltitude = scaledAltitude32(point.altitude)
		record.Speed = scaledUint16(point.speed, 1000)
		record.EnhancedSpeed = scaledUint32(point.speed, 1000)
		record.HeartRate = point.hr
		record.Cadence = point.cadence
		record.Power = point.power
		activity.Records = append(activity.Records, record)
	}

	last := points[len(points)-1]
	session := fit.NewSessionMsg()
	session.Timestamp = start.Add(time.Duration(last.seconds) * time.Second)
	session.Event = fit.EventSession
	session.EventType = fit.EventTypeStop
	session.StartTime = start
	session.StartPositionLat = fit.NewLatitudeDegrees(points[0].lat)
	session.StartPositionLong = fit.NewLongitudeDegrees(points[0].lng)
	session.EndPositionLat = fit.NewLatitudeDegrees(last.lat)
	session.EndPositionLong = fit.NewLongitudeDegrees(last.lng)
	session.Sport = fit.SportCycling
	session.SubSport = fit.SubSportRoad
	session.TotalElapsedTime = uint32(last.seconds * 1000)
	session.TotalTimerTime = uint32(last.seconds * 1000)
	session.TotalMovingTime = uint32(last.seconds * 1000)
	session.TotalDistance = scaledUint32(last.distance, 100)
	session.AvgSpeed = scaledUint16(last.distance/float64(last.seconds), 1000)
	session.EnhancedAvgSpeed = scaledUint32(last.distance/float64(last.seconds), 1000)
	session.MaxSpeed = scaledUint16(maxSpeed(points), 1000)
	session.EnhancedMaxSpeed = scaledUint32(maxSpeed(points), 1000)
	session.AvgHeartRate = uint8(math.Round(avgInt(points, func(point sample) int { return int(point.hr) })))
	session.MaxHeartRate = uint8(maxInt(points, func(point sample) int { return int(point.hr) }))
	session.AvgCadence = uint8(math.Round(avgInt(points, func(point sample) int { return int(point.cadence) })))
	session.MaxCadence = uint8(maxInt(points, func(point sample) int { return int(point.cadence) }))
	session.AvgPower = uint16(math.Round(avgInt(points, func(point sample) int { return int(point.power) })))
	session.MaxPower = uint16(maxInt(points, func(point sample) int { return int(point.power) }))
	session.TotalAscent = 7
	session.TotalDescent = 1
	session.EnhancedMinAltitude = scaledAltitude32(minAltitude(points))
	session.EnhancedMaxAltitude = scaledAltitude32(maxAltitude(points))
	session.MinAltitude = scaledAltitude16(minAltitude(points))
	session.MaxAltitude = scaledAltitude16(maxAltitude(points))
	activity.Sessions = append(activity.Sessions, session)

	stopEvent := fit.NewEventMsg()
	stopEvent.Timestamp = session.Timestamp
	stopEvent.Event = fit.EventTimer
	stopEvent.EventType = fit.EventTypeStopAll
	activity.Events = append(activity.Events, stopEvent)

	activityMsg := fit.NewActivityMsg()
	activityMsg.Timestamp = session.Timestamp
	activityMsg.TotalTimerTime = uint32(last.seconds * 1000)
	activityMsg.NumSessions = 1
	activityMsg.Event = fit.EventActivity
	activityMsg.EventType = fit.EventTypeStop
	activity.Activity = activityMsg

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	output, err := os.Create(out)
	if err != nil {
		return err
	}
	defer output.Close()

	return fit.Encode(output, file, binary.LittleEndian)
}

func scaledUint16(value float64, scale float64) uint16 {
	return uint16(math.Round(value * scale))
}

func scaledUint32(value float64, scale float64) uint32 {
	return uint32(math.Round(value * scale))
}

func scaledAltitude16(meters float64) uint16 {
	return scaledUint16(meters+500, 5)
}

func scaledAltitude32(meters float64) uint32 {
	return scaledUint32(meters+500, 5)
}

func maxSpeed(points []sample) float64 {
	max := 0.0
	for _, point := range points {
		if point.speed > max {
			max = point.speed
		}
	}
	return max
}

func minAltitude(points []sample) float64 {
	min := points[0].altitude
	for _, point := range points[1:] {
		if point.altitude < min {
			min = point.altitude
		}
	}
	return min
}

func maxAltitude(points []sample) float64 {
	max := points[0].altitude
	for _, point := range points[1:] {
		if point.altitude > max {
			max = point.altitude
		}
	}
	return max
}

func avgInt(points []sample, value func(sample) int) float64 {
	total := 0
	for _, point := range points {
		total += value(point)
	}
	return float64(total) / float64(len(points))
}

func maxInt(points []sample, value func(sample) int) int {
	max := value(points[0])
	for _, point := range points[1:] {
		if current := value(point); current > max {
			max = current
		}
	}
	return max
}
