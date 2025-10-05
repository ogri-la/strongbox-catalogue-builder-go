//go:build integration

package github

import (
	"testing"
)

func TestBuildCatalogue(t *testing.T) {
	parser := NewParser()
	addons, err := parser.BuildCatalogue()
	if err != nil {
		t.Fatalf("BuildCatalogue failed: %v", err)
	}

	if len(addons) == 0 {
		t.Errorf("Expected at least some addons, got 0")
	}

	// Verify first addon has expected structure
	if len(addons) > 0 {
		addon := addons[0]
		if addon.Source != "github" {
			t.Errorf("Expected source 'github', got '%s'", addon.Source)
		}
		if addon.SourceID == "" {
			t.Errorf("Expected non-empty source-id")
		}
		if addon.Name == "" {
			t.Errorf("Expected non-empty name")
		}
		if addon.URL == "" {
			t.Errorf("Expected non-empty URL")
		}
	}

	t.Logf("Successfully fetched %d GitHub addons", len(addons))
}
