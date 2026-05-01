package api

import (
	"encoding/json"
	"fmt"
	"mystravastats/api/dto"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type generationDiagnosticsParityFixture struct {
	Cases []generationDiagnosticsParityCase `json:"cases"`
}

type generationDiagnosticsParityCase struct {
	Name          string   `json:"name"`
	Reasons       []string `json:"reasons"`
	ExpectedCodes []string `json:"expectedCodes"`
}

func TestGenerationDiagnosticsParityFixture_SuccessDiagnosticsCodes(t *testing.T) {
	fixture := loadGenerationDiagnosticsParityFixture(t)
	if len(fixture.Cases) == 0 {
		t.Fatal("expected at least one diagnostics parity case")
	}

	for _, testCase := range fixture.Cases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			routes := []dto.GeneratedRouteDto{
				{
					RouteID: "generated-parity-route",
					Reasons: append([]string(nil), testCase.Reasons...),
				},
			}

			gotDiagnostics := buildSuccessfulGenerationDiagnostics(routes)
			gotCodes := make([]string, 0, len(gotDiagnostics))
			for _, diagnostic := range gotDiagnostics {
				gotCodes = append(gotCodes, diagnostic.Code)
			}

			if !reflect.DeepEqual(gotCodes, testCase.ExpectedCodes) {
				t.Fatalf("diagnostic code mismatch for case %q: got %v want %v", testCase.Name, gotCodes, testCase.ExpectedCodes)
			}
		})
	}
}

func loadGenerationDiagnosticsParityFixture(t *testing.T) generationDiagnosticsParityFixture {
	t.Helper()
	fixturePath, err := findRouteFixtureFile("test-fixtures/routes/target-diagnostics-parity.json")
	if err != nil {
		t.Fatalf("failed to locate diagnostics parity fixture file: %v", err)
	}
	payload, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read diagnostics parity fixture file %q: %v", fixturePath, err)
	}
	var fixture generationDiagnosticsParityFixture
	if err := json.Unmarshal(payload, &fixture); err != nil {
		t.Fatalf("failed to decode diagnostics parity fixture file %q: %v", fixturePath, err)
	}
	return fixture
}

func findRouteFixtureFile(relativePath string) (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	visited := map[string]struct{}{}
	for {
		if _, seen := visited[currentDir]; seen {
			break
		}
		visited[currentDir] = struct{}{}

		candidate := filepath.Join(currentDir, filepath.FromSlash(relativePath))
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break
		}
		currentDir = parent
	}

	return "", fmt.Errorf("unable to find %q from current directory tree", relativePath)
}
