package infrastructure

import (
	"mystravastats/domain/badges"
	"path/filepath"
	"testing"
)

func TestNationalFamousClimbCatalogs(t *testing.T) {
	tests := []struct {
		name          string
		expectedSides int
	}{
		{name: "france", expectedSides: 297},
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
			for _, badge := range badgeSet.Badges {
				climb, ok := badge.(badges.FamousClimbBadge)
				if !ok {
					t.Fatalf("unexpected badge type %T", badge)
				}
				if climb.Country == "" || climb.Massif == "" {
					t.Fatalf("missing geography for %q", climb.Label)
				}
			}
			if test.name == "france" {
				assertMadeleineVariantCheckpoint(t, badgeSet, "Col de la Madeleine from La Chambre, par la D213")
				assertMadeleineVariantCheckpoint(t, badgeSet, "Col de la Madeleine from La Chambre, via Montgellafrey")
			}
		})
	}
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
