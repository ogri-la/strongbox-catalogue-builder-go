package catalogue

import (
	"testing"
	"time"

	"github.com/ogri-la/strongbox-catalogue-builder-go/src/types"
)

func TestBuilder_MergeAddonData(t *testing.T) {
	builder := NewBuilder()

	// Create test addon data with different priorities
	listingData := types.AddonData{
		Source:       types.WowInterfaceSource,
		SourceID:     "12345",
		Filename:     "listing.json",
		Name:         "test-addon",
		Label:        "Test Addon",
		DownloadCount: intPtr(100),
		GameTrackSet: map[types.GameTrack]bool{
			types.RetailTrack: true,
		},
	}

	webDetailData := types.AddonData{
		Source:      types.WowInterfaceSource,
		SourceID:    "12345",
		Filename:    "web-detail.json",
		Description: "A test addon for unit testing",
		URL:         "https://www.wowinterface.com/downloads/info12345",
		GameTrackSet: map[types.GameTrack]bool{
			types.RetailTrack:  true,
			types.ClassicTrack: true,
		},
	}

	apiDetailData := types.AddonData{
		Source:   types.WowInterfaceSource,
		SourceID: "12345",
		Filename: "api-detail.json",
		UpdatedDate: timePtr(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)),
		TagSet: map[string]bool{
			"bags": true,
			"inventory": true,
		},
	}

	tests := []struct {
		name        string
		addonData   []types.AddonData
		expectNil   bool
		expectError bool
		checkFunc   func(*testing.T, *types.Addon)
	}{
		{
			name:        "Empty addon data",
			addonData:   []types.AddonData{},
			expectNil:   true,
			expectError: false,
		},
		{
			name:        "Single addon data",
			addonData:   []types.AddonData{listingData},
			expectNil:   true, // No updated date, so should be nil
			expectError: false,
		},
		{
			name:      "Full merge with all data types",
			addonData: []types.AddonData{listingData, webDetailData, apiDetailData},
			checkFunc: func(t *testing.T, addon *types.Addon) {
				if addon.SourceID != "12345" {
					t.Errorf("SourceID = %s, want 12345", addon.SourceID)
				}
				if addon.Name != "test-addon" {
					t.Errorf("Name = %s, want test-addon", addon.Name)
				}
				if addon.Label != "Test Addon" {
					t.Errorf("Label = %s, want Test Addon", addon.Label)
				}
				if addon.Description != "A test addon for unit testing" {
					t.Errorf("Description = %s, want 'A test addon for unit testing'", addon.Description)
				}
				if addon.URL != "https://www.wowinterface.com/downloads/info12345" {
					t.Errorf("URL = %s, want wowinterface URL", addon.URL)
				}
				if *addon.DownloadCount != 100 {
					t.Errorf("DownloadCount = %d, want 100", *addon.DownloadCount)
				}
				if len(addon.GameTrackList) != 2 {
					t.Errorf("GameTrackList length = %d, want 2", len(addon.GameTrackList))
				}
				if len(addon.TagList) != 2 {
					t.Errorf("TagList length = %d, want 2", len(addon.TagList))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := builder.MergeAddonData(tt.addonData)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectNil && result != nil {
				t.Error("Expected nil result but got non-nil")
			}
			if !tt.expectNil && result == nil {
				t.Error("Expected non-nil result but got nil")
			}

			if result != nil && tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

func TestBuilder_BuildCatalogue(t *testing.T) {
	builder := NewBuilder()

	addon1 := types.Addon{
		Source:        types.WowInterfaceSource,
		SourceID:      "12345",
		Name:          "adibags",
		Label:         "AdiBags",
		UpdatedDate:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		GameTrackList: []types.GameTrack{types.RetailTrack},
	}

	addon2 := types.Addon{
		Source:        types.GitHubSource,
		SourceID:      "67890",
		Name:          "deadly-boss-mods",
		Label:         "Deadly Boss Mods",
		UpdatedDate:   time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
		GameTrackList: []types.GameTrack{types.RetailTrack, types.ClassicTrack},
	}

	tests := []struct {
		name           string
		addons         []types.Addon
		sources        []types.Source
		expectedTotal  int
		expectedSource types.Source
	}{
		{
			name:          "All addons, no source filter",
			addons:        []types.Addon{addon1, addon2},
			sources:       []types.Source{},
			expectedTotal: 2,
		},
		{
			name:           "Filter by WowInterface source",
			addons:         []types.Addon{addon1, addon2},
			sources:        []types.Source{types.WowInterfaceSource},
			expectedTotal:  1,
			expectedSource: types.WowInterfaceSource,
		},
		{
			name:           "Filter by GitHub source",
			addons:         []types.Addon{addon1, addon2},
			sources:        []types.Source{types.GitHubSource},
			expectedTotal:  1,
			expectedSource: types.GitHubSource,
		},
		{
			name:          "Empty addon list",
			addons:        []types.Addon{},
			sources:       []types.Source{},
			expectedTotal: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.BuildCatalogue(tt.addons, tt.sources)

			if result.Spec.Version != 2 {
				t.Errorf("Spec.Version = %d, want 2", result.Spec.Version)
			}

			if result.Total != tt.expectedTotal {
				t.Errorf("Total = %d, want %d", result.Total, tt.expectedTotal)
			}

			if len(result.AddonSummaryList) != tt.expectedTotal {
				t.Errorf("AddonSummaryList length = %d, want %d", len(result.AddonSummaryList), tt.expectedTotal)
			}

			if tt.expectedTotal > 0 && tt.expectedSource != "" {
				if result.AddonSummaryList[0].Source != tt.expectedSource {
					t.Errorf("First addon source = %s, want %s", result.AddonSummaryList[0].Source, tt.expectedSource)
				}
			}

			// Check that datestamp is set
			if result.Datestamp == "" {
				t.Error("Datestamp should not be empty")
			}
		})
	}
}

func TestBuilder_ShortenCatalogue(t *testing.T) {
	builder := NewBuilder()

	oldAddon := types.Addon{
		Source:      types.WowInterfaceSource,
		SourceID:    "12345",
		Name:        "old-addon",
		UpdatedDate: time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	newAddon := types.Addon{
		Source:      types.WowInterfaceSource,
		SourceID:    "67890",
		Name:        "new-addon",
		UpdatedDate: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	catalogue := types.Catalogue{
		Spec: struct {
			Version int `json:"version"`
		}{Version: 2},
		Datestamp:        "2024-01-01",
		Total:            2,
		AddonSummaryList: []types.Addon{oldAddon, newAddon},
	}

	cutoffDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	result := builder.ShortenCatalogue(catalogue, cutoffDate)

	if result.Total != 1 {
		t.Errorf("Shortened catalogue total = %d, want 1", result.Total)
	}

	if len(result.AddonSummaryList) != 1 {
		t.Errorf("Shortened catalogue addon list length = %d, want 1", len(result.AddonSummaryList))
	}

	if result.AddonSummaryList[0].Name != "new-addon" {
		t.Errorf("Remaining addon name = %s, want new-addon", result.AddonSummaryList[0].Name)
	}
}

func TestBuilder_FilterCatalogue(t *testing.T) {
	builder := NewBuilder()

	addon1 := types.Addon{
		Source:        types.WowInterfaceSource,
		SourceID:      "12345",
		Name:          "adibags",
		GameTrackList: []types.GameTrack{types.RetailTrack},
	}

	addon2 := types.Addon{
		Source:        types.WowInterfaceSource,
		SourceID:      "67890",
		Name:          "classic-addon",
		GameTrackList: []types.GameTrack{types.ClassicTrack},
	}

	catalogue := types.Catalogue{
		Spec: struct {
			Version int `json:"version"`
		}{Version: 2},
		Datestamp:        "2024-01-01",
		Total:            2,
		AddonSummaryList: []types.Addon{addon1, addon2},
	}

	// Filter for retail addons only
	retailFilter := func(addon types.Addon) bool {
		for _, track := range addon.GameTrackList {
			if track == types.RetailTrack {
				return true
			}
		}
		return false
	}

	result := builder.FilterCatalogue(catalogue, retailFilter)

	if result.Total != 1 {
		t.Errorf("Filtered catalogue total = %d, want 1", result.Total)
	}

	if len(result.AddonSummaryList) != 1 {
		t.Errorf("Filtered catalogue addon list length = %d, want 1", len(result.AddonSummaryList))
	}

	if result.AddonSummaryList[0].Name != "adibags" {
		t.Errorf("Remaining addon name = %s, want adibags", result.AddonSummaryList[0].Name)
	}
}

// Helper functions for tests
func intPtr(i int) *int {
	return &i
}

func timePtr(t time.Time) *time.Time {
	return &t
}