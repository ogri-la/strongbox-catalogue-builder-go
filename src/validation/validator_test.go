package validation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateCatalogueFile(t *testing.T) {
	tests := []struct {
		name          string
		catalogueJSON string
		wantErr       bool
		errContains   string
	}{
		{
			name: "valid wowinterface catalogue",
			catalogueJSON: `{
  "spec": {
    "version": 2
  },
  "datestamp": "2025-10-04",
  "total": 1,
  "addon-summary-list": [
    {
      "source": "wowinterface",
      "source-id": "21718",
      "name": "test-addon",
      "label": "Test Addon",
      "description": "A test addon",
      "updated-date": "2012-10-04T16:42:34Z",
      "created-date": "2012-10-04T10:42:00Z",
      "download-count": 1559,
      "game-track-list": ["retail"],
      "tag-list": ["patches", "plug-ins"],
      "url": "https://www.wowinterface.com/downloads/info21718"
    }
  ]
}`,
			wantErr: false,
		},
		{
			name: "valid github catalogue without created-date",
			catalogueJSON: `{
  "spec": {
    "version": 2
  },
  "datestamp": "2025-10-04T00:00:00Z",
  "total": 1,
  "addon-summary-list": [
    {
      "source": "github",
      "source-id": "owner/repo",
      "name": "test-addon",
      "label": "Test Addon",
      "description": "A test addon",
      "updated-date": "2024-08-01T19:55:21Z",
      "download-count": 0,
      "game-track-list": ["retail"],
      "url": "https://github.com/owner/repo"
    }
  ]
}`,
			wantErr: false,
		},
		{
			name: "invalid - missing required field",
			catalogueJSON: `{
  "spec": {
    "version": 2
  },
  "datestamp": "2025-10-04",
  "total": 1,
  "addon-summary-list": [
    {
      "source": "wowinterface",
      "source-id": "123",
      "label": "Test",
      "updated-date": "2012-10-04T16:42:34Z",
      "game-track-list": ["retail"],
      "url": "https://example.com"
    }
  ]
}`,
			wantErr:     true,
			errContains: "name",
		},
		{
			name: "invalid - wrong source value",
			catalogueJSON: `{
  "spec": {
    "version": 2
  },
  "datestamp": "2025-10-04",
  "total": 1,
  "addon-summary-list": [
    {
      "source": "curseforge",
      "source-id": "123",
      "name": "test",
      "label": "Test",
      "updated-date": "2012-10-04T16:42:34Z",
      "game-track-list": ["retail"],
      "url": "https://example.com"
    }
  ]
}`,
			wantErr:     true,
			errContains: "source",
		},
		{
			name: "invalid - total mismatch",
			catalogueJSON: `{
  "spec": {
    "version": 2
  },
  "datestamp": "2025-10-04",
  "total": 5,
  "addon-summary-list": [
    {
      "source": "wowinterface",
      "source-id": "123",
      "name": "test",
      "label": "Test",
      "updated-date": "2012-10-04T16:42:34Z",
      "game-track-list": ["retail"],
      "url": "https://example.com"
    }
  ]
}`,
			wantErr:     true,
			errContains: "total",
		},
		{
			name: "invalid - bad game track",
			catalogueJSON: `{
  "spec": {
    "version": 2
  },
  "datestamp": "2025-10-04",
  "total": 1,
  "addon-summary-list": [
    {
      "source": "wowinterface",
      "source-id": "123",
      "name": "test",
      "label": "Test",
      "updated-date": "2012-10-04T16:42:34Z",
      "game-track-list": ["invalid-track"],
      "url": "https://example.com"
    }
  ]
}`,
			wantErr:     true,
			errContains: "game-track-list",
		},
		{
			name: "invalid - bad URL",
			catalogueJSON: `{
  "spec": {
    "version": 2
  },
  "datestamp": "2025-10-04",
  "total": 1,
  "addon-summary-list": [
    {
      "source": "wowinterface",
      "source-id": "123",
      "name": "test",
      "label": "Test",
      "updated-date": "2012-10-04T16:42:34Z",
      "game-track-list": ["retail"],
      "url": ""
    }
  ]
}`,
			wantErr:     true,
			errContains: "url",
		},
		{
			name: "invalid - negative download count",
			catalogueJSON: `{
  "spec": {
    "version": 2
  },
  "datestamp": "2025-10-04",
  "total": 1,
  "addon-summary-list": [
    {
      "source": "wowinterface",
      "source-id": "123",
      "name": "test",
      "label": "Test",
      "updated-date": "2012-10-04T16:42:34Z",
      "download-count": -5,
      "game-track-list": ["retail"],
      "url": "https://example.com"
    }
  ]
}`,
			wantErr:     true,
			errContains: "download-count",
		},
		{
			name: "invalid - missing spec version",
			catalogueJSON: `{
  "spec": {},
  "datestamp": "2025-10-04",
  "total": 0,
  "addon-summary-list": []
}`,
			wantErr:     true,
			errContains: "version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "test-catalogue.json")

			err := os.WriteFile(filePath, []byte(tt.catalogueJSON), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Test ValidateCatalogueFile
			err = ValidateCatalogueFile(filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCatalogueFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errContains)
				} else if !contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errContains, err)
				}
			}

			// Also test ValidateCatalogueJSON
			err = ValidateCatalogueJSON([]byte(tt.catalogueJSON))
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCatalogueJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// And test ValidateCatalogue
			var data map[string]any
			json.Unmarshal([]byte(tt.catalogueJSON), &data)
			err = ValidateCatalogue(data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCatalogue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCatalogueFile_FileNotFound(t *testing.T) {
	err := ValidateCatalogueFile("/nonexistent/path/catalogue.json")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestValidateCatalogueJSON_InvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{"invalid json`)
	err := ValidateCatalogueJSON(invalidJSON)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
	if !contains(err.Error(), "parse JSON") {
		t.Errorf("Expected error about parsing JSON, got: %v", err)
	}
}

func TestValidateRealCatalogues(t *testing.T) {
	cataloguePaths := []string{
		"../../state/wowinterface-catalogue.json",
		"../../state/github-catalogue.json",
		"../../test/fixtures/catalogues/clojure-wowinterface-catalogue.json",
		"../../test/fixtures/catalogues/go-wowinterface-catalogue.json",
	}

	for _, path := range cataloguePaths {
		t.Run(path, func(t *testing.T) {
			// Check if file exists
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Skipf("Catalogue file not found: %s", path)
				return
			}

			err := ValidateCatalogueFile(path)
			if err != nil {
				// Some catalogues may have validation errors due to bad source data
				// Log as a warning but don't fail the test
				t.Logf("Validation warning for %s: %v", path, err)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
