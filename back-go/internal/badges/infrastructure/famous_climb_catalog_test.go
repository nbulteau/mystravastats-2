package infrastructure

import (
	"mystravastats/domain/badges"
	"path/filepath"
	"strings"
	"testing"
)

func TestNationalFamousClimbCatalogs(t *testing.T) {
	tests := []struct {
		name          string
		expectedSides int
	}{
		{name: "france", expectedSides: 317},
		{name: "suisse", expectedSides: 48},
		{name: "italie", expectedSides: 78},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join("..", "..", "..", "famous-climb", test.name+".json")
			badgeSet := loadBadgeSet(test.name, path)
			if len(badgeSet.Badges) != test.expectedSides {
				t.Fatalf("expected %d %s climb sides, got %d", test.expectedSides, test.name, len(badgeSet.Badges))
			}
			labels := make(map[string]struct{}, len(badgeSet.Badges))
			corsicaSides := 0
			for _, badge := range badgeSet.Badges {
				climb, ok := badge.(badges.FamousClimbBadge)
				if !ok {
					t.Fatalf("unexpected badge type %T", badge)
				}
				if climb.Country == "" || climb.Massif == "" {
					t.Fatalf("missing geography for %q", climb.Label)
				}
				if _, duplicate := labels[climb.Label]; duplicate {
					t.Fatalf("duplicate climb alternative %q", climb.Label)
				}
				labels[climb.Label] = struct{}{}
				if climb.Length <= 0 || climb.TotalAscent <= 0 || climb.AverageGradient <= 0 || climb.Difficulty < 0 {
					t.Fatalf("invalid climb metrics for %q", climb.Label)
				}
				estimatedAscent := climb.Length * climb.AverageGradient * 10
				if estimatedAscent < float64(climb.TotalAscent)*0.75 || estimatedAscent > float64(climb.TotalAscent)*1.25 {
					t.Fatalf("average gradient is inconsistent with length and ascent for %q", climb.Label)
				}
				if climb.MinimumAltitude < 0 || climb.MaximumGradient < 0 || climb.MaximumGradient > 30 {
					t.Fatalf("invalid optional climb metrics for %q", climb.Label)
				}
				if climb.MaximumGradient > 0 && climb.MaximumGradient+0.1 < climb.AverageGradient {
					t.Fatalf("maximum gradient is below average gradient for %q", climb.Label)
				}
				if !validClimbCategory(climb.Category) {
					t.Fatalf("invalid category %q for %q", climb.Category, climb.Label)
				}
				if !validCoordinate(climb.Start.Latitude, climb.Start.Longitude) || !validCoordinate(climb.End.Latitude, climb.End.Longitude) {
					t.Fatalf("invalid coordinates for %q", climb.Label)
				}
				for _, checkpoint := range climb.RouteCheckpoints {
					if !validCoordinate(checkpoint.Latitude, checkpoint.Longitude) {
						t.Fatalf("invalid route checkpoint for %q", climb.Label)
					}
				}
				directDistance := climb.Start.HaversineInKM(climb.End.Latitude, climb.End.Longitude)
				if directDistance > climb.Length+0.5 {
					t.Fatalf("direct distance %.2f km exceeds published length %.2f km for %q", directDistance, climb.Length, climb.Label)
				}
				if climb.SourceURL != "" && !strings.HasPrefix(climb.SourceURL, "https://") {
					t.Fatalf("invalid source URL for %q", climb.Label)
				}
				if climb.Massif == "Corse" {
					corsicaSides++
				}
			}
			if test.name == "france" {
				if corsicaSides != 29 {
					t.Fatalf("expected 29 Corsican climb sides, got %d", corsicaSides)
				}
				assertMadeleineVariantCheckpoint(t, badgeSet, "Col de la Madeleine from La Chambre, par la D213")
				assertMadeleineVariantCheckpoint(t, badgeSet, "Col de la Madeleine from La Chambre, via Montgellafrey")
				assertClimbClassification(t, badgeSet, "Alpe d'Huez from Le Bourg-d'Oisans", "HC", 994)
			}
		})
	}
}

func validClimbCategory(category string) bool {
	switch category {
	case "HC", "1", "2", "3", "4":
		return true
	default:
		return false
	}
}

func validCoordinate(latitude, longitude float64) bool {
	return latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180
}

func assertClimbClassification(t *testing.T, badgeSet badges.BadgeSet, label, category string, difficulty int) {
	t.Helper()
	for _, badge := range badgeSet.Badges {
		climb, ok := badge.(badges.FamousClimbBadge)
		if ok && climb.Label == label {
			if climb.Category != category || climb.Difficulty != difficulty {
				t.Fatalf("expected %q to be category %s with difficulty %d, got %s and %d", label, category, difficulty, climb.Category, climb.Difficulty)
			}
			return
		}
	}
	t.Fatalf("missing climb alternative %q", label)
}

func assertMadeleineVariantCheckpoint(t *testing.T, badgeSet badges.BadgeSet, label string) {
	t.Helper()
	for _, badge := range badgeSet.Badges {
		climb, ok := badge.(badges.FamousClimbBadge)
		if ok && climb.Label == label {
			if len(climb.RouteCheckpoints) != 1 {
				t.Fatalf("expected one route checkpoint for %q, got %d", label, len(climb.RouteCheckpoints))
			}
			return
		}
	}
	t.Fatalf("missing Madeleine variant %q", label)
}
