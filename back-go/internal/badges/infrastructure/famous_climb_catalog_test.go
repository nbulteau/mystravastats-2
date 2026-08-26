package infrastructure

import (
	"bytes"
	"encoding/json"
	"mystravastats/domain/badges"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNationalCatalogMirrorsAndCoverageReport(t *testing.T) {
	for _, name := range []string{"france", "suisse", "italie", "espagne", "andorre"} {
		goCatalog, err := os.ReadFile(filepath.Join("..", "..", "..", "famous-climb", name+".json"))
		if err != nil {
			t.Fatalf("read Go %s catalog: %v", name, err)
		}
		kotlinCatalog, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "back-kotlin", "famous-climb", name+".json"))
		if err != nil {
			t.Fatalf("read Kotlin %s catalog: %v", name, err)
		}
		if !bytes.Equal(goCatalog, kotlinCatalog) {
			t.Fatalf("Go and Kotlin %s catalogs differ", name)
		}
	}

	reportData, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "docs", "data-sources", "climb-catalog-coverage.json"))
	if err != nil {
		t.Fatalf("read coverage report: %v", err)
	}
	var report struct {
		SummitIdentityCount int `json:"summitIdentityCount"`
		VariantCount        int `json:"variantCount"`
	}
	if err := json.Unmarshal(reportData, &report); err != nil {
		t.Fatalf("decode coverage report: %v", err)
	}
	if report.SummitIdentityCount != 394 || report.VariantCount != 766 {
		t.Fatalf("unexpected coverage report: %#v", report)
	}
}

var officialClimbSourceDomains = []string{
	"mycols.app",
	"cols-cyclisme.com",
	"bigcycling.eu",
	"climbfinder.com",
	"cyclinglocations.com",
}

func TestNationalFamousClimbCatalogs(t *testing.T) {
	tests := []struct {
		name          string
		country       string
		expectedSides int
	}{
		{name: "france", country: "FR", expectedSides: 508},
		{name: "suisse", country: "CH", expectedSides: 47},
		{name: "italie", country: "IT", expectedSides: 78},
		{name: "espagne", country: "ES", expectedSides: 127},
		{name: "andorre", country: "AD", expectedSides: 6},
	}
	allLabels := make(map[string]string)
	allVariantIDs := make(map[string]string)

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
				if climb.Country != test.country || climb.Massif == "" {
					t.Fatalf("missing geography for %q", climb.Label)
				}
				if climb.SummitID == "" || climb.VariantID == "" {
					t.Fatalf("missing stable identity for %q", climb.Label)
				}
				if !strings.HasPrefix(climb.VariantID, climb.SummitID+"--") {
					t.Fatalf("variant id %q is not attached to summit %q", climb.VariantID, climb.SummitID)
				}
				if existingCatalog, duplicate := allVariantIDs[climb.VariantID]; duplicate {
					t.Fatalf("variant id %q is duplicated in %s and %s", climb.VariantID, existingCatalog, test.name)
				}
				allVariantIDs[climb.VariantID] = test.name
				if _, duplicate := labels[climb.Label]; duplicate {
					t.Fatalf("duplicate climb alternative %q", climb.Label)
				}
				labels[climb.Label] = struct{}{}
				if existingCatalog, duplicate := allLabels[climb.Label]; duplicate {
					t.Fatalf("climb alternative %q is duplicated in %s and %s", climb.Label, existingCatalog, test.name)
				}
				allLabels[climb.Label] = test.name
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
				if climb.SummitToleranceMeters < 0 || climb.SummitToleranceMeters > 500 {
					t.Fatalf("invalid summit tolerance for %q", climb.Label)
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
				if climb.SourceURL == "" {
					t.Fatalf("missing source URL for %q", climb.Label)
				}
				if !hasOfficialClimbSource(climb.SourceURL) {
					t.Fatalf("non-official source URL %q for %q", climb.SourceURL, climb.Label)
				}
				if test.country == "ES" && (!validSpanishCoordinate(climb.Start.Latitude, climb.Start.Longitude) || !validSpanishCoordinate(climb.End.Latitude, climb.End.Longitude)) {
					t.Fatalf("implausible Spanish coordinates for %q", climb.Label)
				}
				if climb.Massif == "Corse" {
					corsicaSides++
				}
			}
			if test.name == "france" {
				if corsicaSides != 38 {
					t.Fatalf("expected 38 Corsican climb sides, got %d", corsicaSides)
				}
				assertMadeleineVariantCheckpoint(t, badgeSet, "Col de la Madeleine from La Chambre, via Montgellafrey")
				assertClimbClassification(t, badgeSet, "Alpe d'Huez from Le Bourg-d'Oisans", "HC", 979)
				assertClimbClassification(t, badgeSet, "Col d'Anelle from Saint-Étienne-de-Tinée", "2", 480)
				assertClimbClassification(t, badgeSet, "Col du Galibier from La Grave, via le col du Lautaret", "1", 807)
				assertClimbStart(t, badgeSet, "Col des Saisies from Flumet (D1212 / D218B), via Crest-Voland", 45.82128, 6.53094)
			}
			if test.name == "espagne" {
				assertClimbClassification(t, badgeSet, "Puerto Camacho from Los Tablones", "2", 597)
			}
		})
	}
}

func hasOfficialClimbSource(sourceURL string) bool {
	parsedURL, err := url.Parse(sourceURL)
	if err != nil {
		return false
	}
	host := strings.TrimPrefix(strings.ToLower(parsedURL.Hostname()), "www.")
	for _, domain := range officialClimbSourceDomains {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

func assertClimbStart(t *testing.T, badgeSet badges.BadgeSet, label string, latitude, longitude float64) {
	t.Helper()
	for _, badge := range badgeSet.Badges {
		climb, ok := badge.(badges.FamousClimbBadge)
		if ok && climb.Label == label {
			if climb.Start.Latitude != latitude || climb.Start.Longitude != longitude {
				t.Fatalf("expected %q to start at %.5f, %.5f, got %.5f, %.5f", label, latitude, longitude, climb.Start.Latitude, climb.Start.Longitude)
			}
			return
		}
	}
	t.Fatalf("missing climb alternative %q", label)
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

func validSpanishCoordinate(latitude, longitude float64) bool {
	return latitude >= 27 && latitude <= 44.5 && longitude >= -19 && longitude <= 5
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
