package catalogue

import (
	"sort"
	"time"

	"github.com/ogri-la/strongbox-catalogue-builder-go/src/types"
)

// Builder handles building catalogues from addon data
type Builder struct{}

// NewBuilder creates a new catalogue builder
func NewBuilder() *Builder {
	return &Builder{}
}

// MergeAddonData merges multiple AddonData items for the same addon into a single Addon
// This is a pure function that follows the merge strategy from the Clojure version
func (b *Builder) MergeAddonData(addonDataList []types.AddonData) (*types.Addon, error) {
	if len(addonDataList) == 0 {
		return nil, nil
	}

	// Sort by filename priority: listing < web-detail < api-detail
	sort.Slice(addonDataList, func(i, j int) bool {
		return b.getFilePriority(addonDataList[i].Filename) < b.getFilePriority(addonDataList[j].Filename)
	})

	// Start with empty addon and merge data in priority order
	merged := &types.Addon{
		Source:   addonDataList[0].Source,
		SourceID: addonDataList[0].SourceID,
	}

	gameTrackSet := make(map[types.GameTrack]bool)
	tagSet := make(map[string]bool)

	for _, data := range addonDataList {
		// Merge basic fields (later entries override earlier ones)
		if data.Name != "" {
			merged.Name = data.Name
		}
		if data.Label != "" {
			merged.Label = data.Label
		}
		if data.Description != "" {
			merged.Description = data.Description
		}
		if data.URL != "" {
			merged.URL = data.URL
		}

		// Merge dates (prefer non-zero values)
		if data.UpdatedDate != nil && !data.UpdatedDate.IsZero() {
			merged.UpdatedDate = *data.UpdatedDate
		}
		if data.CreatedDate != nil && !data.CreatedDate.IsZero() {
			merged.CreatedDate = data.CreatedDate
		}

		// Merge download count (prefer non-zero values)
		if data.DownloadCount != nil && *data.DownloadCount > 0 {
			merged.DownloadCount = data.DownloadCount
		}

		// Accumulate game tracks
		for track := range data.GameTrackSet {
			gameTrackSet[track] = true
		}

		// Accumulate tags
		for tag := range data.TagSet {
			tagSet[tag] = true
		}
	}

	// Convert sets to sorted slices
	merged.GameTrackList = b.gameTrackSetToSortedSlice(gameTrackSet)
	merged.TagList = b.stringSetToSortedSlice(tagSet)

	// Apply defaults and validation
	if merged.UpdatedDate.IsZero() {
		return nil, nil // Invalid addon without update date
	}

	if len(merged.GameTrackList) == 0 {
		merged.GameTrackList = []types.GameTrack{types.RetailTrack} // Default to retail
	}

	return merged, nil
}

// BuildCatalogue creates a catalogue from a list of addons
func (b *Builder) BuildCatalogue(addons []types.Addon, sources []types.Source) types.Catalogue {
	var filteredAddons []types.Addon

	// Filter by sources if specified
	if len(sources) > 0 {
		sourceMap := make(map[types.Source]bool)
		for _, source := range sources {
			sourceMap[source] = true
		}

		for _, addon := range addons {
			if sourceMap[addon.Source] {
				filteredAddons = append(filteredAddons, addon)
			}
		}
	} else {
		filteredAddons = addons
	}

	// Sort addons by source-id for stable, deterministic output
	// source-id changes less frequently than name (which can vary with slugification)
	sort.Slice(filteredAddons, func(i, j int) bool {
		return filteredAddons[i].SourceID < filteredAddons[j].SourceID
	})

	return types.Catalogue{
		Spec: struct {
			Version int `json:"version"`
		}{Version: 2},
		Datestamp:        b.currentDateStamp(),
		Total:            len(filteredAddons),
		AddonSummaryList: filteredAddons,
	}
}

// ShortenCatalogue filters out unmaintained addons (similar to Clojure version)
func (b *Builder) ShortenCatalogue(catalogue types.Catalogue, cutoffDate time.Time) types.Catalogue {
	var maintainedAddons []types.Addon

	for _, addon := range catalogue.AddonSummaryList {
		if addon.UpdatedDate.After(cutoffDate) {
			maintainedAddons = append(maintainedAddons, addon)
		}
	}

	return types.Catalogue{
		Spec:             catalogue.Spec,
		Datestamp:        catalogue.Datestamp,
		Total:            len(maintainedAddons),
		AddonSummaryList: maintainedAddons,
	}
}

// FilterCatalogue filters addons by a predicate function
func (b *Builder) FilterCatalogue(catalogue types.Catalogue, predicate func(types.Addon) bool) types.Catalogue {
	var filteredAddons []types.Addon

	for _, addon := range catalogue.AddonSummaryList {
		if predicate(addon) {
			filteredAddons = append(filteredAddons, addon)
		}
	}

	return types.Catalogue{
		Spec:             catalogue.Spec,
		Datestamp:        catalogue.Datestamp,
		Total:            len(filteredAddons),
		AddonSummaryList: filteredAddons,
	}
}

// Private helper methods

// getFilePriority returns priority for merge order (lower = higher priority)
func (b *Builder) getFilePriority(filename string) int {
	switch {
	case filename == "listing.json":
		return 0 // lowest priority
	case filename == "web-detail.json":
		return 1 // medium priority
	case filename == "api-detail.json":
		return 2 // highest priority
	case filename == "api-filelist.json":
		return 2 // same as api-detail
	default:
		return 0 // default to lowest priority
	}
}

// gameTrackSetToSortedSlice converts a set to a sorted slice
func (b *Builder) gameTrackSetToSortedSlice(trackSet map[types.GameTrack]bool) []types.GameTrack {
	tracks := make([]types.GameTrack, 0, len(trackSet))
	for track := range trackSet {
		tracks = append(tracks, track)
	}

	// Sort by the order defined in types.AllGameTracks
	trackOrder := make(map[types.GameTrack]int)
	for i, track := range types.AllGameTracks {
		trackOrder[track] = i
	}

	sort.Slice(tracks, func(i, j int) bool {
		return trackOrder[tracks[i]] < trackOrder[tracks[j]]
	})

	return tracks
}

// stringSetToSortedSlice converts a string set to a sorted slice
func (b *Builder) stringSetToSortedSlice(stringSet map[string]bool) []string {
	strings := make([]string, 0, len(stringSet))
	for str := range stringSet {
		strings = append(strings, str)
	}
	sort.Strings(strings)
	return strings
}

// currentDateStamp returns current date in YYYY-MM-DD format
func (b *Builder) currentDateStamp() string {
	return time.Now().Format("2006-01-02")
}
