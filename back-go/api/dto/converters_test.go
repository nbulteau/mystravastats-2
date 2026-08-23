package dto

import (
	"math"
	"strings"
	"testing"

	"mystravastats/domain/badges"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
)

func TestToAthleteDto_NilDates(t *testing.T) {
	// GIVEN
	athlete := strava.Athlete{Id: 123}

	// WHEN
	dto := ToAthleteDto(athlete)

	// THEN
	if !dto.CreatedAt.IsZero() {
		t.Fatalf("expected CreatedAt to be zero value, got %v", dto.CreatedAt)
	}
	if !dto.UpdatedAt.IsZero() {
		t.Fatalf("expected UpdatedAt to be zero value, got %v", dto.UpdatedAt)
	}
}

func TestToAthleteDto_MapsFTP(t *testing.T) {
	ftp := any(250.0)
	athlete := strava.Athlete{
		Id:  123,
		Ftp: &ftp,
	}

	dto := ToAthleteDto(athlete)

	if dto.FTP != 250 {
		t.Fatalf("expected FTP=250, got %d", dto.FTP)
	}
}

func TestAthletePerformanceSettingsConverters_RoundTrip(t *testing.T) {
	weight := 72.5
	settings := business.AthletePerformanceSettings{
		WeightKg: &weight,
		FtpHistory: []business.AthleteFtpSetting{
			{EffectiveFrom: "2026-01-01", Ftp: 160},
		},
	}

	dto := ToAthletePerformanceSettingsDto(settings)
	if dto.WeightKg == nil || *dto.WeightKg != weight {
		t.Fatalf("expected weightKg=%f, got %+v", weight, dto.WeightKg)
	}
	if len(dto.FtpHistory) != 1 || dto.FtpHistory[0].Ftp != 160 {
		t.Fatalf("unexpected DTO history: %+v", dto.FtpHistory)
	}

	roundTrip := ToAthletePerformanceSettings(dto)
	if roundTrip.WeightKg == nil || *roundTrip.WeightKg != weight {
		t.Fatalf("expected round-trip weightKg=%f, got %+v", weight, roundTrip.WeightKg)
	}
	if len(roundTrip.FtpHistory) != 1 || roundTrip.FtpHistory[0].EffectiveFrom != "2026-01-01" {
		t.Fatalf("unexpected round-trip history: %+v", roundTrip.FtpHistory)
	}
}

func TestBuildActivityEfforts_NilStream(t *testing.T) {
	// GIVEN
	detailedActivity := &strava.DetailedActivity{Id: 42, Stream: nil}

	// WHEN
	efforts := BuildActivityEfforts(detailedActivity)

	// THEN
	if len(efforts) != 0 {
		t.Fatalf("expected no efforts when stream is nil, got %d", len(efforts))
	}
}

func TestBuildActivityEfforts_LabelsDetectedClimbs(t *testing.T) {
	// GIVEN
	detailedActivity := &strava.DetailedActivity{
		Id:   43,
		Name: "Irregular climb",
		Type: "Ride",
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{
				0, 100, 200, 300, 400, 500, 600, 700, 800, 900,
				1000, 1100, 1200, 1300, 1400, 1500, 1600, 1700, 1800,
			}},
			Time: strava.TimeStream{Data: []int{
				0, 40, 80, 120, 160, 200, 240, 280, 320, 360,
				400, 440, 480, 520, 560, 600, 640, 680, 720,
			}},
			Altitude: &strava.AltitudeStream{Data: []float64{
				100, 106, 112, 118, 124, 126, 125, 127, 133, 139,
				145, 151, 157, 163, 169, 175, 181, 187, 193,
			}},
		},
	}

	// WHEN
	efforts := BuildActivityEfforts(detailedActivity)

	// THEN
	for _, effort := range efforts {
		if strings.HasPrefix(effort.Label, "Climb 1 -") {
			if !strings.Contains(effort.Label, "D+") || strings.Contains(effort.Label, "Slope") {
				t.Fatalf("expected explicit climb label, got %q", effort.Label)
			}
			if effort.DeltaAltitude <= 0 {
				t.Fatalf("expected climb effort to have positive elevation, got %.1f", effort.DeltaAltitude)
			}
			return
		}
	}
	t.Fatalf("expected generated climb effort, got %#v", efforts)
}

func TestBuildActivityEfforts_AddsPowerEfforts(t *testing.T) {
	// GIVEN
	detailedActivity := &strava.DetailedActivity{
		Id:   44,
		Name: "Power detail",
		Type: "Ride",
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 1000, 2000}},
			Time:     strava.TimeStream{Data: []int{0, 3600, 7200}},
			Altitude: &strava.AltitudeStream{Data: []float64{100, 110, 120}},
			Watts:    &strava.PowerStream{Data: []float64{180, 220, 260}},
		},
	}

	// WHEN
	efforts := BuildActivityEfforts(detailedActivity)

	// THEN
	if !hasEffortLabel(efforts, "Best Power for 1000 m") {
		t.Fatalf("expected best power distance effort, got %#v", efforts)
	}
	if !hasEffortLabel(efforts, "Best power for 1h0m0s") {
		t.Fatalf("expected best power time effort, got %#v", efforts)
	}
}

func hasEffortLabel(efforts []business.ActivityEffort, label string) bool {
	for _, effort := range efforts {
		if effort.Label == label {
			return true
		}
	}
	return false
}

func TestBuildActivityEfforts_DirectionAwareSegmentLabels(t *testing.T) {
	// GIVEN
	segmentName := "MURAILLE DE CHINE <Alpe d'Huez>"
	detailedActivity := &strava.DetailedActivity{
		Id:   42,
		Name: "Direction check",
		Type: "Ride",
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 100, 200, 300, 400, 500, 600}},
			Time:     strava.TimeStream{Data: []int{0, 10, 20, 30, 40, 50, 60}},
			Altitude: &strava.AltitudeStream{Data: []float64{100, 102, 105, 108, 106, 104, 102}},
		},
		SegmentEfforts: []strava.SegmentEffort{
			{
				Id:          1001,
				Name:        "Muraille montée",
				Distance:    300,
				ElapsedTime: 30,
				StartIndex:  0,
				EndIndex:    3,
				Segment: strava.Segment{
					Id:            9001,
					Name:          segmentName,
					ActivityType:  "Ride",
					AverageGrade:  8.0,
					ClimbCategory: 4,
					ElevationHigh: 108,
					ElevationLow:  100,
				},
			},
			{
				Id:          1002,
				Name:        "Muraille descente",
				Distance:    300,
				ElapsedTime: 20,
				StartIndex:  3,
				EndIndex:    6,
				Segment: strava.Segment{
					Id:            9002,
					Name:          segmentName,
					ActivityType:  "Ride",
					AverageGrade:  -7.5,
					ClimbCategory: 4,
					ElevationHigh: 108,
					ElevationLow:  100,
				},
			},
		},
	}

	// WHEN
	efforts := BuildActivityEfforts(detailedActivity)

	// THEN
	foundAscent := false
	foundDescent := false
	for _, effort := range efforts {
		if !strings.Contains(effort.Label, segmentName) {
			continue
		}
		if strings.Contains(effort.Label, "(ascent)") {
			foundAscent = true
			if effort.DeltaAltitude <= 0 {
				t.Fatalf("expected ascent delta altitude to be positive, got %.2f", effort.DeltaAltitude)
			}
		}
		if strings.Contains(effort.Label, "(descent)") {
			foundDescent = true
			if effort.DeltaAltitude >= 0 {
				t.Fatalf("expected descent delta altitude to be negative, got %.2f", effort.DeltaAltitude)
			}
		}
	}

	if !foundAscent {
		t.Fatalf("expected ascent segment effort label for %q", segmentName)
	}
	if !foundDescent {
		t.Fatalf("expected descent segment effort label for %q", segmentName)
	}
}

func TestToBadgeDto_UsesRepresentativeBadgeActivityTypeForRideVariants(t *testing.T) {
	dto := ToBadgeDto(badges.DistanceRideLevel1, business.GravelRide, business.MountainBikeRide, business.Ride)

	if dto.Type != "RideDistanceBadge" {
		t.Fatalf("expected RideDistanceBadge, got %q", dto.Type)
	}
}

func TestToBadgeDto_UsesRepresentativeBadgeActivityTypeForTrailRun(t *testing.T) {
	dto := ToBadgeDto(badges.DistanceRunLevel1, business.TrailRun)

	if dto.Type != "RunDistanceBadge" {
		t.Fatalf("expected RunDistanceBadge, got %q", dto.Type)
	}
}

func TestToBadgeDto_UsesRepresentativeBadgeActivityTypeForWalk(t *testing.T) {
	dto := ToBadgeDto(badges.DistanceHikeLevel1, business.Walk)

	if dto.Type != "HikeDistanceBadge" {
		t.Fatalf("expected HikeDistanceBadge, got %q", dto.Type)
	}
}

func TestToBadgeDto_UsesHikingBadgeTypeForWalk(t *testing.T) {
	dto := ToBadgeDto(badges.SummitDayBadge, business.Walk)

	if dto.Type != "HikeHikingBadge" {
		t.Fatalf("expected HikeHikingBadge, got %q", dto.Type)
	}
	if dto.Label != "Summit Day" {
		t.Fatalf("expected Summit Day label, got %q", dto.Label)
	}
	if dto.Description == "" {
		t.Fatalf("expected hiking badge description")
	}
}

func TestToActivityDto_SanitizesNonFiniteSummaryValues(t *testing.T) {
	// GIVEN
	activity := strava.Activity{
		Id:                 51,
		Name:               "Non-finite summary",
		Type:               "Ride",
		Distance:           math.NaN(),
		TotalElevationGain: math.Inf(1),
		AverageSpeed:       math.Inf(-1),
		AverageHeartrate:   math.NaN(),
		AverageWatts:       math.Inf(1),
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 100}},
			Time:     strava.TimeStream{Data: []int{0, 10}},
			Altitude: &strava.AltitudeStream{Data: []float64{100, 110}},
		},
	}

	// WHEN
	dto := ToActivityDto(activity)

	// THEN
	if dto.Distance != 0 || dto.TotalElevationGain != 0 || dto.AverageSpeed != 0 || dto.AverageHeartrate != 0 || dto.AverageWatts != 0 {
		t.Fatalf("expected non-finite summary values to be zeroed, got %#v", dto)
	}
}

func TestFiniteEffortPower_SanitizesNonFiniteAveragePower(t *testing.T) {
	// GIVEN
	averagePower := math.NaN()
	effort := &business.ActivityEffort{AveragePower: &averagePower}

	// WHEN / THEN
	if got := finiteEffortPower(effort); got != 0 {
		t.Fatalf("expected non-finite average power to be zeroed, got %d", got)
	}
}

func TestBuildActivityEfforts_NaNAltitudeFallsBackToSegmentDelta(t *testing.T) {
	// GIVEN
	detailedActivity := &strava.DetailedActivity{
		Id:   52,
		Name: "NaN direction check",
		Type: "Ride",
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 100, 200}},
			Time:     strava.TimeStream{Data: []int{0, 10, 20}},
			Altitude: &strava.AltitudeStream{Data: []float64{100, math.NaN(), 110}},
		},
		SegmentEfforts: []strava.SegmentEffort{
			{
				Id:          2001,
				Name:        "NaN climb",
				Distance:    200,
				ElapsedTime: 20,
				StartIndex:  0,
				EndIndex:    2,
				Segment: strava.Segment{
					Id:            9901,
					Name:          "NaN climb segment",
					ActivityType:  "Ride",
					AverageGrade:  5.0,
					ClimbCategory: 4,
					ElevationHigh: 120,
					ElevationLow:  100,
				},
			},
		},
	}

	// WHEN
	efforts := BuildActivityEfforts(detailedActivity)

	// THEN
	for _, effort := range efforts {
		if !strings.Contains(effort.Label, "NaN climb segment") {
			continue
		}
		if math.IsNaN(effort.DeltaAltitude) || math.IsInf(effort.DeltaAltitude, 0) {
			t.Fatalf("expected finite delta altitude, got %.2f", effort.DeltaAltitude)
		}
		if effort.DeltaAltitude <= 0 {
			t.Fatalf("expected fallback ascent delta altitude to remain positive, got %.2f", effort.DeltaAltitude)
		}
		return
	}

	t.Fatalf("expected segment effort to be present")
}

func TestToDetailedActivityDto_SanitizesNonFiniteValues(t *testing.T) {
	// GIVEN
	sufferScore := math.NaN()
	detailedActivity := &strava.DetailedActivity{
		Id:                 53,
		Name:               "Non-finite detail",
		Type:               "Ride",
		AverageCadence:     math.NaN(),
		AverageHeartrate:   math.Inf(1),
		AverageWatts:       math.Inf(-1),
		AverageSpeed:       math.NaN(),
		Calories:           math.Inf(1),
		Distance:           math.NaN(),
		ElevHigh:           math.NaN(),
		Kilojoules:         math.Inf(1),
		MaxHeartrate:       math.NaN(),
		MaxSpeed:           math.Inf(1),
		StartLatLng:        []float64{math.NaN(), math.Inf(1)},
		SufferScore:        &sufferScore,
		TotalElevationGain: math.NaN(),
		Stream: &strava.Stream{
			Distance:       strava.DistanceStream{Data: []float64{0, math.NaN()}},
			Time:           strava.TimeStream{Data: []int{0, 1}},
			Altitude:       &strava.AltitudeStream{Data: []float64{100, math.Inf(1)}},
			Watts:          &strava.PowerStream{Data: []float64{200, math.NaN()}},
			VelocitySmooth: &strava.SmoothVelocityStream{Data: []float64{5, math.Inf(-1)}},
		},
	}

	// WHEN
	dto := ToDetailedActivityDto(detailedActivity)

	// THEN
	if dto.AverageCadence != 0 || dto.AverageHeartrate != 0 || dto.AverageWatts != 0 {
		t.Fatalf("expected non-finite integer metrics to be zeroed, got cadence=%d hr=%d watts=%d", dto.AverageCadence, dto.AverageHeartrate, dto.AverageWatts)
	}
	if dto.AverageSpeed != 0 || dto.Calories != 0 || dto.Distance != 0 || dto.ElevHigh != 0 || dto.Kilojoules != 0 {
		t.Fatalf("expected non-finite float metrics to be zeroed, got %#v", dto)
	}
	if dto.SufferScore != nil {
		t.Fatalf("expected non-finite optional suffer score to be removed")
	}
	if dto.StartLatlng[0] != 0 || dto.StartLatlng[1] != 0 {
		t.Fatalf("expected start lat/lng to be sanitized, got %#v", dto.StartLatlng)
	}
	if dto.Stream == nil || dto.Stream.Distance[1] != 0 || dto.Stream.Altitude[1] != 0 || dto.Stream.Watts[1] != 0 || dto.Stream.VelocitySmooth[1] != 0 {
		t.Fatalf("expected stream values to be sanitized, got %#v", dto.Stream)
	}
}

func TestToDetailedActivityDto_ExposesStravaSegmentEfforts(t *testing.T) {
	// GIVEN
	prRank := 2
	detailedActivity := &strava.DetailedActivity{
		Id:        54,
		Name:      "Segment detail",
		Type:      "Ride",
		SportType: "GravelRide",
		SegmentEfforts: []strava.SegmentEffort{
			{
				Id:               1001,
				Name:             "Local sprint",
				Distance:         520.5,
				ElapsedTime:      75,
				MovingTime:       74,
				StartIndex:       10,
				EndIndex:         85,
				AverageWatts:     315.7,
				AverageHeartRate: 165,
				DeviceWatts:      true,
				PrRank:           &prRank,
				Segment: strava.Segment{
					Id:            9001,
					Name:          "Local sprint segment",
					ActivityType:  "Ride",
					AverageGrade:  4.2,
					Distance:      520.5,
					ElevationHigh: 120,
					ElevationLow:  98,
					Starred:       true,
				},
			},
		},
	}

	// WHEN
	dto := ToDetailedActivityDto(detailedActivity)

	// THEN
	if len(dto.StravaSegmentEfforts) != 1 {
		t.Fatalf("expected one Strava segment effort, got %d", len(dto.StravaSegmentEfforts))
	}
	if dto.Type != "Ride" || dto.SportType != "GravelRide" {
		t.Fatalf("expected type and sport type to be mapped, got type=%q sportType=%q", dto.Type, dto.SportType)
	}
	effort := dto.StravaSegmentEfforts[0]
	if effort.Name != "Local sprint" || effort.Segment.Name != "Local sprint segment" {
		t.Fatalf("expected segment names to be mapped, got %#v", effort)
	}
	if effort.StartIndex != 10 || effort.EndIndex != 85 {
		t.Fatalf("expected stream indexes to be mapped, got start=%d end=%d", effort.StartIndex, effort.EndIndex)
	}
	if effort.AverageWatts != 315.7 || !effort.DeviceWatts || effort.PrRank == nil || *effort.PrRank != 2 {
		t.Fatalf("expected power and rank fields to be mapped, got %#v", effort)
	}
}

func TestToStreamDto_MapsValues(t *testing.T) {
	// GIVEN
	stream := &strava.Stream{
		Distance: strava.DistanceStream{Data: []float64{1, 2}},
		Time:     strava.TimeStream{Data: []int{10, 20}},
		LatLng: &strava.LatLngStream{Data: [][]float64{
			{10.1, 20.2},
			{30.3, 40.4},
		}},
		Moving:         &strava.MovingStream{Data: []bool{true, false}},
		Altitude:       &strava.AltitudeStream{Data: []float64{100.5, 101.5}},
		Cadence:        &strava.CadenceStream{Data: []int{82, 84}},
		Watts:          &strava.PowerStream{Data: []float64{210.0, 220.0}},
		VelocitySmooth: &strava.SmoothVelocityStream{Data: []float64{8.5, 8.8}},
	}

	// WHEN
	dto := toStreamDto(stream)

	// THEN
	if dto == nil {
		t.Fatal("expected non-nil dto")
	}

	if dto.Moving[0] != true || dto.Moving[1] != false {
		t.Fatalf("unexpected moving values: %v %v", dto.Moving[0], dto.Moving[1])
	}

	if dto.Altitude[0] != 100.5 || dto.Altitude[1] != 101.5 {
		t.Fatalf("unexpected altitude values: %.2f %.2f", dto.Altitude[0], dto.Altitude[1])
	}

	if dto.Watts[0] != 210.0 || dto.Watts[1] != 220.0 {
		t.Fatalf("unexpected watts values: %.1f %.1f", dto.Watts[0], dto.Watts[1])
	}

	if dto.Cadence[0] != 82 || dto.Cadence[1] != 84 {
		t.Fatalf("unexpected cadence values: %d %d", dto.Cadence[0], dto.Cadence[1])
	}

	if dto.VelocitySmooth[0] != 8.5 || dto.VelocitySmooth[1] != 8.8 {
		t.Fatalf("unexpected velocity values: %.1f %.1f", dto.VelocitySmooth[0], dto.VelocitySmooth[1])
	}

	if dto.Latlng[0][0] != 10.1 || dto.Latlng[0][1] != 20.2 {
		t.Fatalf("unexpected first latlng values: %.1f %.1f", dto.Latlng[0][0], dto.Latlng[0][1])
	}
	if dto.Latlng[1][0] != 30.3 || dto.Latlng[1][1] != 40.4 {
		t.Fatalf("unexpected second latlng values: %.1f %.1f", dto.Latlng[1][0], dto.Latlng[1][1])
	}
}

func TestComputeFamousClimbEffortSeconds_UsesSegmentDurationNotActivityDuration(t *testing.T) {
	// GIVEN
	badge := badges.FamousClimbBadge{
		Start: business.GeoCoordinate{Latitude: 45.2178751, Longitude: 6.4750846},
		End:   business.GeoCoordinate{Latitude: 45.2026999, Longitude: 6.4446143},
	}

	activity := &strava.Activity{
		MovingTime: 12000,
		Stream: &strava.Stream{
			Time: strava.TimeStream{Data: []int{0, 100, 700, 1200}},
			LatLng: &strava.LatLngStream{Data: [][]float64{
				{45.1000000, 6.3000000},
				{45.2178751, 6.4750846}, // start at t=100
				{45.2100000, 6.4600000},
				{45.2026999, 6.4446143}, // summit at t=1200
			}},
		},
	}

	// WHEN
	effortSeconds, ok := computeFamousClimbEffortSeconds(activity, badge)

	// THEN
	if !ok {
		t.Fatalf("expected effort duration to be detected")
	}
	if effortSeconds != 1100 {
		t.Fatalf("expected effortSeconds=1100, got %d", effortSeconds)
	}
}

func TestComputeFamousClimbEffortSeconds_RejectsDescentOrder(t *testing.T) {
	// GIVEN
	badge := badges.FamousClimbBadge{
		Start: business.GeoCoordinate{Latitude: 45.2178751, Longitude: 6.4750846},
		End:   business.GeoCoordinate{Latitude: 45.2026999, Longitude: 6.4446143},
	}

	activity := &strava.Activity{
		Stream: &strava.Stream{
			Time: strava.TimeStream{Data: []int{0, 400, 900}},
			LatLng: &strava.LatLngStream{Data: [][]float64{
				{45.2026999, 6.4446143}, // summit first
				{45.2100000, 6.4600000},
				{45.2178751, 6.4750846}, // start after summit
			}},
		},
	}

	// WHEN
	_, ok := computeFamousClimbEffortSeconds(activity, badge)

	// THEN
	if ok {
		t.Fatalf("expected descent-only order to be rejected")
	}
}

func TestToBadgeCheckResultDto_ExposesClimbPosterDetails(t *testing.T) {
	badge := badges.FamousClimbBadge{
		Label:           "Test col from valley",
		Name:            "Test col",
		Country:         "FR",
		Massif:          "Alpes",
		SourceURL:       "https://example.test/test-col",
		TopOfTheAscent:  1850,
		Start:           business.GeoCoordinate{Latitude: 45.1000, Longitude: 6.1000},
		End:             business.GeoCoordinate{Latitude: 45.2000, Longitude: 6.2000},
		Length:          12.4,
		TotalAscent:     980,
		Difficulty:      800,
		MinimumAltitude: 870,
		MaximumGradient: 12.5,
		AverageGradient: 7.9,
		Category:        "1",
	}
	activity := &strava.Activity{
		Id:             42,
		StartDateLocal: "2026-07-14T08:00:00Z",
		MovingTime:     3600,
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 1000, 7000, 13400}},
			Time:     strava.TimeStream{Data: []int{0, 100, 650, 1200}},
			LatLng: &strava.LatLngStream{Data: [][]float64{
				{45.0000, 6.0000},
				{45.1000, 6.1000},
				{45.1500, 6.1500},
				{45.2000, 6.2000},
			}},
			Altitude:  &strava.AltitudeStream{Data: []float64{90, 100, 120, 135}},
			Watts:     &strava.PowerStream{Data: []float64{0, 200, 220, 240}},
			HeartRate: &strava.HeartRateStream{Data: []int{0, 140, 150, 160}},
		},
	}
	slowerActivity := *activity
	slowerActivity.Id = 43
	slowerActivity.StartDateLocal = "2025-06-12T08:00:00Z"
	slowerStream := *activity.Stream
	slowerStream.Time = strava.TimeStream{Data: []int{0, 100, 750, 1400}}
	slowerActivity.Stream = &slowerStream

	result := business.BadgeCheckResult{
		Badge:       badge,
		Activities:  []*strava.Activity{&slowerActivity, activity},
		IsCompleted: true,
	}
	dto := ToBadgeCheckResultDto(result, business.Ride)

	if dto.ClimbDetails == nil {
		t.Fatal("expected climb poster details")
	}
	details := dto.ClimbDetails
	if details.Name != "Test col" || details.SummitAltitude != 1850 || details.MinimumAltitude != 870 || details.LengthKm != 12.4 || details.Difficulty != 800 {
		t.Fatalf("unexpected climb metadata: %#v", details)
	}
	if details.Country != "FR" || details.Massif != "Alpes" || details.SourceURL != "https://example.test/test-col" {
		t.Fatalf("unexpected climb geography or source: %#v", details)
	}
	if details.SummitCoordinate.Latitude != 45.2 || details.SummitCoordinate.Longitude != 6.2 || details.StartCoordinate.Latitude != 45.1 || details.StartCoordinate.Longitude != 6.1 {
		t.Fatalf("unexpected climb map coordinates: summit=%#v start=%#v", details.SummitCoordinate, details.StartCoordinate)
	}
	if details.AscentCount != 2 || details.BestAscent == nil || details.BestAscent.DurationSeconds != 1100 || details.BestAscent.ActivityID != 42 || details.BestAscent.Date != "2026-07-14T08:00:00Z" {
		t.Fatalf("unexpected climb ascent summary: count=%d best=%#v", details.AscentCount, details.BestAscent)
	}
	if len(details.Ascents) != 2 || details.Ascents[0].ActivityID != 42 || details.Ascents[1].ActivityID != 43 {
		t.Fatalf("expected every climb ascent ordered newest first, got %#v", details.Ascents)
	}
	best := details.Ascents[0]
	if best.ActivityName != activity.Name || best.VAMMetersPerHour == nil || *best.VAMMetersPerHour != 3207 || best.AverageSpeedKph == nil || *best.AverageSpeedKph != 40.6 {
		t.Fatalf("unexpected computed climb performance: %#v", best)
	}
	if best.AveragePowerWatts == nil || *best.AveragePowerWatts != 220 || best.AverageHeartRateBpm == nil || *best.AverageHeartRateBpm != 150 {
		t.Fatalf("unexpected climb sensor averages: %#v", best)
	}
	if len(details.Profile) != 3 || details.Profile[0].DistanceKm != 0 || details.Profile[2].DistanceKm != 12.4 {
		t.Fatalf("unexpected climb profile: %#v", details.Profile)
	}
	if details.MaximumGradient == nil || *details.MaximumGradient != 12.5 {
		t.Fatalf("expected the 12.5%% reference maximum gradient, got %#v", details.MaximumGradient)
	}
}

func TestFindFamousClimbBounds_SelectsOccurrenceClosestToReferenceLength(t *testing.T) {
	badge := badges.FamousClimbBadge{
		Start:  business.GeoCoordinate{Latitude: 45.1, Longitude: 6.1},
		End:    business.GeoCoordinate{Latitude: 45.2, Longitude: 6.2},
		Length: 10,
	}
	activity := &strava.Activity{Stream: &strava.Stream{
		Distance: strava.DistanceStream{Data: []float64{0, 42000, 50000, 55000, 60000}},
		LatLng: &strava.LatLngStream{Data: [][]float64{
			{45.1, 6.1},
			{45.0, 6.0},
			{45.1, 6.1},
			{45.15, 6.15},
			{45.2, 6.2},
		}},
	}}

	bounds, matched := findFamousClimbBounds(activity, badge)
	if !matched {
		t.Fatal("expected the repeated climb occurrence to match")
	}
	if bounds.startIndex != 2 || bounds.endIndex != 4 {
		t.Fatalf("expected the 10 km occurrence, got %#v", bounds)
	}
}

func TestFindFamousClimbBounds_RejectsImplausiblyLongRouteBetweenWaypoints(t *testing.T) {
	badge := badges.FamousClimbBadge{
		Start:  business.GeoCoordinate{Latitude: 45.1, Longitude: 6.1},
		End:    business.GeoCoordinate{Latitude: 45.2, Longitude: 6.2},
		Length: 10,
	}
	activity := &strava.Activity{Stream: &strava.Stream{
		Distance: strava.DistanceStream{Data: []float64{0, 52000}},
		LatLng: &strava.LatLngStream{Data: [][]float64{
			{45.1, 6.1},
			{45.2, 6.2},
		}},
	}}

	if _, matched := findFamousClimbBounds(activity, badge); matched {
		t.Fatal("expected a 52 km detour to be rejected for a 10 km climb")
	}
}

func TestToBadgeCheckResultDto_AcceptsPublishedMaximumGradientAboveComputedCeiling(t *testing.T) {
	badge := badges.FamousClimbBadge{
		Label:           "Steep col from valley",
		Name:            "Steep col",
		TopOfTheAscent:  1500,
		Length:          10,
		TotalAscent:     1000,
		MaximumGradient: 21,
		AverageGradient: 10,
		Difficulty:      1000,
		Category:        "HC",
	}
	dto := ToBadgeCheckResultDto(business.BadgeCheckResult{Badge: badge}, business.Ride)
	if dto.ClimbDetails == nil || dto.ClimbDetails.MaximumGradient == nil || *dto.ClimbDetails.MaximumGradient != 21 {
		t.Fatalf("expected the published 21%% maximum gradient, got %#v", dto.ClimbDetails)
	}
}

func TestComputeClimbMaximumGradient_IgnoresShortAltitudeSpike(t *testing.T) {
	activity := &strava.Activity{Stream: &strava.Stream{
		Distance: strava.DistanceStream{Data: []float64{0, 100, 500}},
		Altitude: &strava.AltitudeStream{Data: []float64{100, 160, 140}},
	}}

	maximumGradient := computeClimbMaximumGradient(activity, famousClimbBounds{startIndex: 0, endIndex: 2})

	if maximumGradient == nil || *maximumGradient != 8 {
		t.Fatalf("expected a 500 m rolling maximum of 8%%, got %#v", maximumGradient)
	}
}
