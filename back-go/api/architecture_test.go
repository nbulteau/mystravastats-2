package api

import (
	"go/parser"
	"go/token"
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

			if strings.HasPrefix(importPath, "mystravastats/internal/services") {
				// THEN
				t.Fatalf("forbidden import in api package (%s): %s", file, importPath)
			}
		}
	}

	// THEN
	// No forbidden import was found in the api package.
}
