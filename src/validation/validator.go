package validation

import (
	"encoding/json"
	"fmt"
	"os"
)

// ValidateCatalogueFile validates a catalogue JSON file
func ValidateCatalogueFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	return ValidateCatalogueJSON(data)
}

// ValidateCatalogueJSON validates catalogue JSON data
func ValidateCatalogueJSON(data []byte) error {
	var catalogueData map[string]any
	if err := json.Unmarshal(data, &catalogueData); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	return ValidateCatalogue(catalogueData)
}

// ValidateCatalogue validates a catalogue data structure
func ValidateCatalogue(data map[string]any) error {
	return SimpleValidateCatalogue(data)
}
