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

func TestFamousClimbBadgeCheck_UsesTighterSummitToleranceForGlandonFromAllemond(t *testing.T) {
	start := business.GeoCoordinate{Latitude: 45.12809, Longitude: 6.0456}
	glandon := business.GeoCoordinate{Latitude: 45.2396101, Longitude: 6.1754635}
	badge := FamousClimbBadge{
		Name:                  "Col du Glandon",
		Label:                 "Col du Glandon from Allemond (Barrage du Verney)",
		Start:                 start,
		End:                   glandon,
		SummitToleranceMeters: 100,
		Length:                25.2,
	}
	passesOnlyOnCroixDeFerRoad := &strava.Activity{
		StartLatlng: []float64{start.Latitude, start.Longitude},
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 25200}},
			LatLng: &strava.LatLngStream{Data: [][]float64{
				{start.Latitude, start.Longitude},
				{45.238498, 6.175907}, // 128 m from Glandon, on the shared Croix-de-Fer road.
			}},
		},
	}
	visitsGlandonAfterCroixDeFer := &strava.Activity{
		StartLatlng: []float64{start.Latitude, start.Longitude},
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 27600, 31200}},
			LatLng: &strava.LatLngStream{Data: [][]float64{
				{start.Latitude, start.Longitude},
				{45.2274902, 6.2033309},
				{glandon.Latitude, glandon.Longitude},
			}},
		},
	}

	if _, matched := badge.Check([]*strava.Activity{passesOnlyOnCroixDeFerRoad}); matched {
		t.Fatal("expected the shared road to Croix-de-Fer not to earn the Glandon badge")
	}
	if _, matched := badge.Check([]*strava.Activity{visitsGlandonAfterCroixDeFer}); !matched {
		t.Fatal("expected a real Glandon visit after Croix-de-Fer to earn the Glandon badge")
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

func TestBadgeSetCheck_AssignsSaisiesActivityToEasternFlumetVariant(t *testing.T) {
	summit := business.GeoCoordinate{Latitude: 45.76102, Longitude: 6.53341}
	mainStart := business.GeoCoordinate{Latitude: 45.81808, Longitude: 6.51646}
	eastStart := business.GeoCoordinate{Latitude: 45.82128, Longitude: 6.53094}
	activity := &strava.Activity{
		Id:          19593558916,
		StartLatlng: []float64{summit.Latitude, summit.Longitude},
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 13778, 27556}},
			LatLng: &strava.LatLngStream{Data: [][]float64{
				{summit.Latitude, summit.Longitude},
				{45.821362, 6.531259}, // Turnaround recorded on 4 August 2026.
				{summit.Latitude, summit.Longitude},
			}},
		},
	}
	const eastLabel = "Col des Saisies from Flumet (D1212 / D218B), via Crest-Voland"
	badgeSet := BadgeSet{Name: "france", Badges: []Badge{
		FamousClimbBadge{Name: "Col des Saisies", Label: "Col des Saisies from Flumet via Le Planay", Start: mainStart, End: summit, Length: 14.8},
		FamousClimbBadge{Name: "Col des Saisies", Label: eastLabel, Start: eastStart, End: summit, Length: 13.106},
	}}

	results := badgeSet.Check([]*strava.Activity{activity})
	for _, result := range results {
		climb := result.Badge.(FamousClimbBadge)
		if climb.Label == eastLabel {
			if !result.IsCompleted || len(result.Activities) != 1 {
				t.Fatalf("expected activity %d to earn the eastern Flumet variant", activity.Id)
			}
			continue
		}
		if result.IsCompleted || len(result.Activities) != 0 {
			t.Fatalf("expected activity %d not to earn %q", activity.Id, climb.Label)
		}
	}
}
