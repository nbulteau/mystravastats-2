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

type targetDiagnosticsParityFixture struct {
	Cases []targetDiagnosticsParityCase `json:"cases"`
}

type targetDiagnosticsParityCase struct {
	Name          string   `json:"name"`
	Reasons       []string `json:"reasons"`
	ExpectedCodes []string `json:"expectedCodes"`
}

func TestTargetDiagnosticsParityFixture_SuccessDiagnosticsCodes(t *testing.T) {
	fixture := loadTargetDiagnosticsParityFixture(t)
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

			gotDiagnostics := buildSuccessfulTargetDiagnostics(routes)
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

func loadTargetDiagnosticsParityFixture(t *testing.T) targetDiagnosticsParityFixture {
	t.Helper()
	fixturePath, err := findRouteDiagnosticsFixtureFile("test-fixtures/routes/target-diagnostics-parity.json")
	if err != nil {
		t.Fatalf("failed to locate diagnostics parity fixture file: %v", err)
	}
	payload, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read diagnostics parity fixture file %q: %v", fixturePath, err)
	}
	var fixture targetDiagnosticsParityFixture
	if err := json.Unmarshal(payload, &fixture); err != nil {
		t.Fatalf("failed to decode diagnostics parity fixture file %q: %v", fixturePath, err)
	}
	return fixture
}

func findRouteDiagnosticsFixtureFile(relativePath string) (string, error) {
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
