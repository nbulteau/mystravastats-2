package badges

import (
	"testing"

	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
)

func TestFamousClimbBadgeCheck_AllowsActivitiesStartingFarFromClimb(t *testing.T) {
	// GIVEN
	badge := FamousClimbBadge{
		Name:   "Col du Télégraphe",
		Label:  "Col du Télégraphe from Saint Michel de Maurienne",
		Start:  business.GeoCoordinate{Latitude: 45.2178751, Longitude: 6.4750846},
		End:    business.GeoCoordinate{Latitude: 45.2026999, Longitude: 6.4446143},
		Length: 11.8,
	}

	// WHEN
	activity := &strava.Activity{
		StartLatlng: []float64{45.1885, 5.7245}, // Grenoble area, far from Télégraphe start.
		Stream: &strava.Stream{
			LatLng: &strava.LatLngStream{
				Data: [][]float64{
					{45.2178751, 6.4750846}, // Saint-Michel-de-Maurienne
					{45.2026999, 6.4446143}, // Col du Télégraphe
				},
			},
		},
	}

	// THEN
	activities, matched := badge.Check([]*strava.Activity{activity})
	if !matched {
		t.Fatalf("expected Télégraphe badge to match when both climb points are in stream")
	}
	if len(activities) != 1 {
		t.Fatalf("expected exactly one matched activity, got %d", len(activities))
	}
}

func TestFamousClimbBadgeCheck_AllowsWaypointWithinFiveHundredMeters(t *testing.T) {
	// GIVEN
	badge := FamousClimbBadge{
		Name:   "Col du Télégraphe",
		Label:  "Col du Télégraphe from Saint Michel de Maurienne",
		Start:  business.GeoCoordinate{Latitude: 45.2178751, Longitude: 6.4750846},
		End:    business.GeoCoordinate{Latitude: 45.2026999, Longitude: 6.4446143},
		Length: 11.8,
	}

	// WHEN
	activity := &strava.Activity{
		StartLatlng: []float64{45.2178751, 6.4750846},
		Stream: &strava.Stream{
			LatLng: &strava.LatLngStream{
				Data: [][]float64{
					{45.2178751, 6.4750846}, // Saint-Michel-de-Maurienne
					{45.2058, 6.4446143},    // ~340m from Télégraphe summit
				},
			},
		},
	}

	// THEN
	_, matched := badge.Check([]*strava.Activity{activity})
	if !matched {
		t.Fatalf("expected Télégraphe badge to match with a stream point within 500m of summit")
	}
}

func TestFamousClimbBadgeCheck_DoesNotMatchDescentOnly(t *testing.T) {
	// GIVEN
	badge := FamousClimbBadge{
		Name:   "Col du Télégraphe",
		Label:  "Col du Télégraphe from Saint Michel de Maurienne",
		Start:  business.GeoCoordinate{Latitude: 45.2178751, Longitude: 6.4750846},
		End:    business.GeoCoordinate{Latitude: 45.2026999, Longitude: 6.4446143},
		Length: 11.8,
	}

	// WHEN
	activity := &strava.Activity{
		StartLatlng: []float64{45.2026999, 6.4446143},
		Stream: &strava.Stream{
			LatLng: &strava.LatLngStream{
				Data: [][]float64{
					{45.2026999, 6.4446143}, // summit first
					{45.2178751, 6.4750846}, // valley after => descent
				},
			},
		},
	}

	// THEN
	_, matched := badge.Check([]*strava.Activity{activity})
	if matched {
		t.Fatalf("expected Télégraphe descent-only activity to NOT match badge")
	}
}
