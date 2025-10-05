package validation

import (
	"fmt"
)

// SimpleValidateCatalogue validates a catalogue using simple custom logic
func SimpleValidateCatalogue(data map[string]any) error {
	// Validate spec
	spec, ok := data["spec"].(map[string]any)
	if !ok {
		return fmt.Errorf("validation failed: spec is required and must be an object")
	}

	version, ok := spec["version"]
	if !ok {
		return fmt.Errorf("validation failed: spec.version is required")
	}

	versionInt, ok := getInt(version)
	if !ok || versionInt < 1 {
		return fmt.Errorf("validation failed: spec.version must be an integer >= 1")
	}

	// Validate datestamp
	datestamp, ok := data["datestamp"].(string)
	if !ok {
		return fmt.Errorf("validation failed: datestamp is required and must be a string")
	}

	if !isValidDateString(datestamp) {
		return fmt.Errorf("validation failed: datestamp must be a valid date string (RFC3339 or YYYY-MM-DD)")
	}

	// Validate total
	total, ok := getInt(data["total"])
	if !ok || total < 0 {
		return fmt.Errorf("validation failed: total is required and must be a non-negative integer")
	}

	// Validate addon-summary-list
	addonListRaw, ok := data["addon-summary-list"]
	if !ok {
		return fmt.Errorf("validation failed: addon-summary-list is required")
	}

	addonList, ok := addonListRaw.([]any)
	if !ok {
		return fmt.Errorf("validation failed: addon-summary-list must be an array")
	}

	// Validate total matches addon count
	if total != len(addonList) {
		return fmt.Errorf("validation failed: total (%d) must equal the number of addons in addon-summary-list (%d)", total, len(addonList))
	}

	// Validate each addon
	for i, addonRaw := range addonList {
		addon, ok := addonRaw.(map[string]any)
		if !ok {
			return fmt.Errorf("validation failed: addon-summary-list[%d] must be an object", i)
		}

		if err := validateAddon(addon, i); err != nil {
			return err
		}
	}

	return nil
}

func validateAddon(addon map[string]any, index int) error {
	prefix := fmt.Sprintf("addon-summary-list[%d]", index)

	// Required fields
	source, ok := addon["source"].(string)
	if !ok {
		return fmt.Errorf("validation failed: %s.source is required and must be a string", prefix)
	}

	if !isValidSource(source) {
		return fmt.Errorf("validation failed: %s.source must be one of: wowinterface, github", prefix)
	}

	sourceID, ok := addon["source-id"].(string)
	if !ok {
		return fmt.Errorf("validation failed: %s.source-id is required and must be a string", prefix)
	}

	if len(sourceID) == 0 {
		return fmt.Errorf("validation failed: %s.source-id must be a non-empty string", prefix)
	}

	name, ok := addon["name"].(string)
	if !ok {
		return fmt.Errorf("validation failed: %s.name is required and must be a string", prefix)
	}

	if len(name) == 0 {
		return fmt.Errorf("validation failed: %s.name must be a non-empty string", prefix)
	}

	label, ok := addon["label"].(string)
	if !ok {
		return fmt.Errorf("validation failed: %s.label is required and must be a string", prefix)
	}

	if len(label) == 0 {
		return fmt.Errorf("validation failed: %s.label must be a non-empty string", prefix)
	}

	updatedDate, ok := addon["updated-date"].(string)
	if !ok {
		return fmt.Errorf("validation failed: %s.updated-date is required and must be a string", prefix)
	}

	if !isValidDateString(updatedDate) {
		return fmt.Errorf("validation failed: %s.updated-date must be a valid RFC3339 or YYYY-MM-DD timestamp", prefix)
	}

	urlStr, ok := addon["url"].(string)
	if !ok {
		return fmt.Errorf("validation failed: %s.url is required and must be a string", prefix)
	}

	if !isValidURL(urlStr) {
		return fmt.Errorf("validation failed: %s.url must be a valid URL", prefix)
	}

	gameTrackList, ok := addon["game-track-list"]
	if !ok {
		return fmt.Errorf("validation failed: %s.game-track-list is required", prefix)
	}

	// game-track-list must be present but can be null or [] (both mean unclassified)
	if gameTrackList == nil {
		// null is acceptable - treat as empty list
		gameTrackList = []any{}
	}

	gameTrackArr, ok := gameTrackList.([]any)
	if !ok {
		return fmt.Errorf("validation failed: %s.game-track-list must be an array", prefix)
	}

	// Empty game-track-list is allowed - indicates addons that need classification
	for j, track := range gameTrackArr {
		trackStr, ok := track.(string)
		if !ok {
			return fmt.Errorf("validation failed: %s.game-track-list[%d] must be a string", prefix, j)
		}

		if !isValidGameTrack(trackStr) {
			return fmt.Errorf("validation failed: %s.game-track-list[%d] must be a valid game track", prefix, j)
		}
	}

	// Optional fields
	if createdDate, ok := addon["created-date"].(string); ok {
		if !isValidDateString(createdDate) {
			return fmt.Errorf("validation failed: %s.created-date must be a valid RFC3339 or YYYY-MM-DD timestamp", prefix)
		}
	}

	if downloadCount, ok := addon["download-count"]; ok {
		count, ok := getInt(downloadCount)
		if !ok || count < 0 {
			return fmt.Errorf("validation failed: %s.download-count must be a non-negative integer", prefix)
		}
	}

	return nil
}

func getInt(val any) (int, bool) {
	switch v := val.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	case int64:
		return int(v), true
	default:
		return 0, false
	}
}
