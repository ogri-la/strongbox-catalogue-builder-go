package validation

import (
	"net/url"
	"time"

	"github.com/Oudwins/zog"
	"github.com/ogri-la/strongbox-catalogue-builder-go/src/types"
)

// ValidGameTracks contains all valid game track values
var ValidGameTracks = []string{
	string(types.RetailTrack),
	string(types.ClassicTrack),
	string(types.ClassicTBCTrack),
	string(types.ClassicWotLKTrack),
	string(types.ClassicCataTrack),
	string(types.ClassicMistsTrack),
}

// ValidSources contains all valid source values
var ValidSources = []string{
	string(types.WowInterfaceSource),
	string(types.GitHubSource),
}

// isValidSource checks if a string is a valid source
func isValidSource(val any) bool {
	str, ok := val.(string)
	if !ok {
		return false
	}
	for _, valid := range ValidSources {
		if str == valid {
			return true
		}
	}
	return false
}

// isValidGameTrack checks if a string is a valid game track
func isValidGameTrack(val any) bool {
	str, ok := val.(string)
	if !ok {
		return false
	}
	for _, valid := range ValidGameTracks {
		if str == valid {
			return true
		}
	}
	return false
}

// isValidURL checks if a string is a valid URL
func isValidURL(val any) bool {
	str, ok := val.(string)
	if !ok {
		return false
	}
	if str == "" {
		return false
	}
	_, err := url.Parse(str)
	return err == nil
}

// isValidURLPtr checks if a string pointer is a valid URL
func isValidURLPtr(val *string, ctx zog.Ctx) bool {
	if val == nil {
		return false
	}
	if *val == "" {
		return false
	}
	_, err := url.Parse(*val)
	return err == nil
}

// isValidDateString checks if a string is a valid date
func isValidDateString(val any) bool {
	str, ok := val.(string)
	if !ok {
		return false
	}
	// Accept both RFC3339 and YYYY-MM-DD formats
	_, err1 := time.Parse(time.RFC3339, str)
	_, err2 := time.Parse("2006-01-02", str)
	return err1 == nil || err2 == nil
}

// isValidDateStringPtr checks if a string pointer is a valid date
func isValidDateStringPtr(val *string, ctx zog.Ctx) bool {
	if val == nil {
		return false
	}
	// Accept both RFC3339 and YYYY-MM-DD formats
	_, err1 := time.Parse(time.RFC3339, *val)
	_, err2 := time.Parse("2006-01-02", *val)
	return err1 == nil || err2 == nil
}

// AddonSchema validates an Addon structure (using PascalCase field names)
var AddonSchema = zog.Struct(zog.Schema{
	"Source":        zog.String().Required().OneOf(ValidSources, zog.Message("source must be one of: wowinterface, github")),
	"SourceId":      zog.String().Required().Min(1, zog.Message("source-id must be a non-empty string")),
	"Name":          zog.String().Required().Min(1, zog.Message("name must be a non-empty string")),
	"Label":         zog.String().Required().Min(1, zog.Message("label must be a non-empty string")),
	"Description":   zog.String().Optional(),
	"UpdatedDate":   zog.String().Required().TestFunc(isValidDateStringPtr, zog.Message("updated-date must be a valid RFC3339 or YYYY-MM-DD timestamp")),
	"CreatedDate":   zog.String().Optional().TestFunc(isValidDateStringPtr, zog.Message("created-date must be a valid RFC3339 or YYYY-MM-DD timestamp")),
	"DownloadCount": zog.Int().Optional().GTE(0, zog.Message("download-count must be a non-negative integer")),
	"GameTrackList": zog.Slice(
		zog.String().OneOf(ValidGameTracks, zog.Message("invalid game track")),
	).Required(zog.Message("game-track-list is required")),
	"TagList": zog.Slice(zog.String()).Optional(),
	"Url":     zog.String().Required().TestFunc(isValidURLPtr, zog.Message("url must be a valid URL")),
})

// totalMatchesAddonCountPtr validates that total equals addon count
func totalMatchesAddonCountPtr(val any, ctx zog.Ctx) bool {
	data, ok := val.(map[string]any)
	if !ok {
		return false
	}
	total, ok := data["Total"].(int)
	if !ok {
		// Try float64 which is how JSON numbers are unmarshaled
		totalFloat, ok := data["Total"].(float64)
		if !ok {
			return false
		}
		total = int(totalFloat)
	}
	addonList, ok := data["AddonSummaryList"].([]any)
	if !ok {
		return false
	}
	return total == len(addonList)
}

// CatalogueSchema validates a Catalogue structure
var CatalogueSchema = zog.Struct(zog.Schema{
	"Spec": zog.Struct(zog.Schema{
		"Version": zog.Int().Required().GTE(1, zog.Message("spec version must be >= 1")),
	}).Required(),
	"Datestamp":        zog.String().Required().TestFunc(isValidDateStringPtr, zog.Message("datestamp must be a valid date string")),
	"Total":            zog.Int().Required().GTE(0, zog.Message("total must be a non-negative integer")),
	"AddonSummaryList": zog.Slice(AddonSchema).Required(),
}).TestFunc(totalMatchesAddonCountPtr, zog.Message("total must equal the number of addons in addon-summary-list"))
