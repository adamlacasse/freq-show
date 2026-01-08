package api

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestOpenAPISpecIsValid(t *testing.T) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	specPath := filepath.Clean(filepath.Join("..", "..", "..", "..", "specs", "openapi.yaml"))
	doc, err := loader.LoadFromFile(specPath)
	if err != nil {
		t.Fatalf("failed to load OpenAPI spec at %s: %v", specPath, err)
	}

	if err := doc.Validate(context.Background()); err != nil {
		t.Fatalf("OpenAPI spec validation failed: %v", err)
	}
}
