package strava

import "testing"

func TestListSlopesDefaultKeepsIrregularClimbTogether(t *testing.T) {
	stream := Stream{
		Distance: DistanceStream{Data: []float64{
			0, 100, 200, 300, 400, 500, 600, 700, 800, 900,
			1000, 1100, 1200, 1300, 1400, 1500, 1600, 1700, 1800,
		}},
		Time: TimeStream{Data: []int{
			0, 40, 80, 120, 160, 200, 240, 280, 320, 360,
			400, 440, 480, 520, 560, 600, 640, 680, 720,
		}},
		Altitude: &AltitudeStream{Data: []float64{
			100, 106, 112, 118, 124, 126, 125, 127, 133, 139,
			145, 151, 157, 163, 169, 175, 181, 187, 193,
		}},
	}

	slopes := stream.ListSlopesDefault()

	if len(slopes) != 1 {
		t.Fatalf("expected one sustained climb, got %d: %#v", len(slopes), slopes)
	}
	climb := slopes[0]
	if climb.Type != ASCENT {
		t.Fatalf("expected ascent slope, got %v", climb.Type)
	}
	if climb.StartIndex > 1 {
		t.Fatalf("expected climb to start near the first ramp, got start index %d", climb.StartIndex)
	}
	if climb.EndIndex < 17 {
		t.Fatalf("expected climb to include the resumed ramp after the false flat, got end index %d", climb.EndIndex)
	}
	if climb.Distance < 1600 {
		t.Fatalf("expected climb distance to cover the irregular ascent, got %.1f", climb.Distance)
	}
	if gain := climb.EndAltitude - climb.StartAltitude; gain < 80 {
		t.Fatalf("expected climb gain to include both ramps, got %.1f", gain)
	}
}

func TestListSlopesDefaultAcceptsRatioGradeSmooth(t *testing.T) {
	stream := Stream{
		Distance: DistanceStream{Data: []float64{
			0, 100, 200, 300, 400, 500, 600, 700, 800, 900,
			1000, 1100, 1200,
		}},
		Time: TimeStream{Data: []int{
			0, 40, 80, 120, 160, 200, 240, 280, 320, 360,
			400, 440, 480,
		}},
		Altitude: &AltitudeStream{Data: []float64{
			100, 106, 112, 118, 124, 130, 136, 142, 148, 154,
			160, 166, 172,
		}},
		GradeSmooth: &SmoothGradeStream{Data: []float64{
			0, 0.06, 0.06, 0.06, 0.06, 0.06, 0.06, 0.06, 0.06, 0.06,
			0.06, 0.06, 0.06,
		}},
	}

	slopes := stream.ListSlopesDefault()

	if len(slopes) != 1 {
		t.Fatalf("expected ratio grade_smooth to produce one climb, got %d: %#v", len(slopes), slopes)
	}
	if slopes[0].MaxGrade < 5 {
		t.Fatalf("expected grade_smooth ratio to be normalized to percent, got max grade %.1f", slopes[0].MaxGrade)
	}
}

func TestListSlopesDefaultFallsBackToAltitudeWhenGradeSmoothHasNoSignal(t *testing.T) {
	stream := Stream{
		Distance: DistanceStream{Data: []float64{
			0, 100, 200, 300, 400, 500, 600, 700, 800, 900,
			1000, 1100, 1200,
		}},
		Time: TimeStream{Data: []int{
			0, 40, 80, 120, 160, 200, 240, 280, 320, 360,
			400, 440, 480,
		}},
		Altitude: &AltitudeStream{Data: []float64{
			100, 106, 112, 118, 124, 130, 136, 142, 148, 154,
			160, 166, 172,
		}},
		GradeSmooth: &SmoothGradeStream{Data: []float64{
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		}},
	}

	slopes := stream.ListSlopesDefault()

	if len(slopes) != 1 {
		t.Fatalf("expected altitude fallback to produce one climb, got %d: %#v", len(slopes), slopes)
	}
}
