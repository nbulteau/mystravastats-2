package api

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestAPIPackage_DoesNotImportInternalServices(t *testing.T) {
	// GIVEN
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("failed to list api go files: %v", err)
	}

	// WHEN
	assertNoForbiddenImportsInFiles(t, files, "mystravastats/internal/services", "api package")

	// THEN
	// No forbidden import was found in the api package.
}

func TestHexagonalInternalPackages_DoNotImportInternalServices(t *testing.T) {
	// GIVEN
	files := make([]string, 0)
	err := filepath.WalkDir("../internal", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			if path == "../internal/services" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk internal packages: %v", err)
	}

	// WHEN
	assertNoForbiddenImportsInFiles(t, files, "mystravastats/internal/services", "internal hexagonal packages")

	// THEN
	// No forbidden import was found outside legacy internal/services package.
}

func assertNoForbiddenImportsInFiles(t *testing.T, files []string, forbiddenPrefix string, scope string) {
	t.Helper()

	fileSet := token.NewFileSet()
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		parsedFile, err := parser.ParseFile(fileSet, file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("failed to parse %s: %v", file, err)
		}

		for _, importSpec := range parsedFile.Imports {
			importPath, err := strconv.Unquote(importSpec.Path.Value)
			if err != nil {
				t.Fatalf("failed to decode import path %q in %s: %v", importSpec.Path.Value, file, err)
			}

			if strings.HasPrefix(importPath, forbiddenPrefix) {
				t.Fatalf("forbidden import in %s (%s): %s", scope, file, importPath)
			}
		}
	}
}
