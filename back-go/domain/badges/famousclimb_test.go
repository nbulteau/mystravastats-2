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

func TestFamousClimbBadgeCheck_DoesNotMatchAFullRideDetourBetweenWaypoints(t *testing.T) {
	badge := FamousClimbBadge{
		Name:   "La Hourquette d'Ancizan",
		Label:  "La Hourquette d'Ancizan from Payolle",
		Start:  business.GeoCoordinate{Latitude: 42.943493, Longitude: 0.278038},
		End:    business.GeoCoordinate{Latitude: 42.899975, Longitude: 0.305761},
		Length: 10,
	}
	activity := &strava.Activity{
		StartLatlng: []float64{42.943493, 0.278038},
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 52700}},
			LatLng: &strava.LatLngStream{Data: [][]float64{
				{42.943493, 0.278038},
				{42.899975, 0.305761},
			}},
		},
	}

	if _, matched := badge.Check([]*strava.Activity{activity}); matched {
		t.Fatal("expected a 52.7 km detour to be rejected for the 10 km Payolle ascent")
	}
}

func TestFamousClimbBadgeCheck_RequiresVariantRouteCheckpoint(t *testing.T) {
	badge := FamousClimbBadge{
		Name:   "Col de la Madeleine",
		Label:  "Col de la Madeleine from La Chambre, par la D213",
		Start:  business.GeoCoordinate{Latitude: 45.3597, Longitude: 6.29929},
		End:    business.GeoCoordinate{Latitude: 45.4352186, Longitude: 6.3756008},
		Length: 19.6,
		RouteCheckpoints: []business.GeoCoordinate{
			{Latitude: 45.386825, Longitude: 6.331231},
		},
	}
	activity := &strava.Activity{
		StartLatlng: []float64{45.3597, 6.29929},
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 10000, 19600}},
			LatLng: &strava.LatLngStream{Data: [][]float64{
				{45.3597, 6.29929},
				{45.391775, 6.319134}, // Montgellafrey, not the D213 checkpoint.
				{45.4352186, 6.3756008},
			}},
		},
	}

	if _, matched := badge.Check([]*strava.Activity{activity}); matched {
		t.Fatal("expected the Montgellafrey route not to match the D213 variant")
	}
}

func TestBadgeSetCheck_AssignsOneActivityToOnlyOneVariantOfSameClimb(t *testing.T) {
	start := business.GeoCoordinate{Latitude: 45.3597, Longitude: 6.29929}
	end := business.GeoCoordinate{Latitude: 45.4352186, Longitude: 6.3756008}
	activity := &strava.Activity{
		Id:          9336871903,
		StartLatlng: []float64{start.Latitude, start.Longitude},
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 19700}},
			LatLng: &strava.LatLngStream{Data: [][]float64{
				{start.Latitude, start.Longitude},
				{end.Latitude, end.Longitude},
			}},
		},
	}
	badgeSet := BadgeSet{Name: "france", Badges: []Badge{
		FamousClimbBadge{Name: "Col de la Madeleine", Label: "D213", Start: start, End: end, Length: 19.6},
		FamousClimbBadge{Name: "Col de la Madeleine", Label: "Montgellafrey", Start: start, End: end, Length: 19.8},
	}}

	results := badgeSet.Check([]*strava.Activity{activity})
	completedCount := 0
	assignedActivityCount := 0
	for _, result := range results {
		if result.IsCompleted {
			completedCount++
		}
		assignedActivityCount += len(result.Activities)
	}
	if completedCount != 1 || assignedActivityCount != 1 {
		t.Fatalf("expected one activity to earn exactly one variant, got completed=%d assigned=%d", completedCount, assignedActivityCount)
	}
}
