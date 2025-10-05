package wowi

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ogri-la/strongbox-catalogue-builder-go/src/types"
)

// Helper function to load test fixtures
func loadFixture(name string) ([]byte, error) {
	fixturePath := filepath.Join("..", "..", "test", "fixtures", name)
	return os.ReadFile(fixturePath)
}

func TestParseAddonDetailPage_MultipleDownloadsTabber(t *testing.T) {
	parser := NewParser()

	// Load fixture
	content, err := loadFixture("wowinterface--addon-detail--multiple-downloads--tabber.html")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	url := "https://www.wowinterface.com/downloads/info8149-BrokerPlayedTime.html"
	result, err := parser.parseAddonDetail(url, content)
	if err != nil {
		t.Fatalf("Failed to parse addon detail: %v", err)
	}

	if len(result.AddonData) != 1 {
		t.Fatalf("Expected 1 addon, got %d", len(result.AddonData))
	}

	addon := result.AddonData[0]

	// Verify basic fields
	if addon.Source != types.WowInterfaceSource {
		t.Errorf("Source = %s, want %s", addon.Source, types.WowInterfaceSource)
	}

	if addon.SourceID != "8149" {
		t.Errorf("SourceID = %s, want 8149", addon.SourceID)
	}

	if addon.Name != "broker-played-time" {
		t.Errorf("Name = %s, want broker-played-time", addon.Name)
	}

	if addon.Label != "Broker Played Time" {
		t.Errorf("Label = %s, want 'Broker Played Time'", addon.Label)
	}

	// Verify game tracks
	expectedTracks := map[types.GameTrack]bool{
		types.RetailTrack:       true,
		types.ClassicTrack:      true,
		types.ClassicTBCTrack:   true,
		types.ClassicWotLKTrack: true,
	}

	if len(addon.GameTrackSet) != len(expectedTracks) {
		t.Errorf("GameTrackSet length = %d, want %d", len(addon.GameTrackSet), len(expectedTracks))
	}

	for track := range expectedTracks {
		if !addon.GameTrackSet[track] {
			t.Errorf("Missing game track: %s", track)
		}
	}

	// Verify downloads
	if len(addon.LatestReleaseSet) == 0 {
		t.Error("Expected download releases, got none")
	}

	// Check that addon has retail game track (from Compatibility field, not download links)
	if !addon.GameTrackSet[types.RetailTrack] {
		t.Error("Expected retail game track, found none")
	}
}

func TestParseAddonDetailPage_SingleDownloadTabber(t *testing.T) {
	parser := NewParser()

	content, err := loadFixture("wowinterface--addon-detail--single-download--tabber.html")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	url := "https://www.wowinterface.com/downloads/info8149-IceHUD.html"
	result, err := parser.parseAddonDetail(url, content)
	if err != nil {
		t.Fatalf("Failed to parse addon detail: %v", err)
	}

	if len(result.AddonData) != 1 {
		t.Fatalf("Expected 1 addon, got %d", len(result.AddonData))
	}

	addon := result.AddonData[0]

	if addon.SourceID != "8149" {
		t.Errorf("SourceID = %s, want 8149", addon.SourceID)
	}

	if addon.Name != "icehud" {
		t.Errorf("Name = %s, want icehud", addon.Name)
	}

	if addon.Label != "IceHUD" {
		t.Errorf("Label = %s, want IceHUD", addon.Label)
	}

	// Compatibility field says "Plunderstorm (10.2.6)" which is retail
	if !addon.GameTrackSet[types.RetailTrack] {
		t.Error("Expected retail game track based on Compatibility field")
	}

	// Should have description
	if addon.Description == "" {
		t.Error("Expected description, got empty string")
	}
}

func TestParseAddonDetailPage_SingleDownloadSupportsAll(t *testing.T) {
	parser := NewParser()

	content, err := loadFixture("wowinterface--addon-detail--single-download--supports-all.html")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	url := "https://www.wowinterface.com/downloads/info11551-MapCoords.html"
	result, err := parser.parseAddonDetail(url, content)
	if err != nil {
		t.Fatalf("Failed to parse addon detail: %v", err)
	}

	if len(result.AddonData) != 1 {
		t.Fatalf("Expected 1 addon, got %d", len(result.AddonData))
	}

	addon := result.AddonData[0]

	if addon.SourceID != "11551" {
		t.Errorf("SourceID = %s, want 11551", addon.SourceID)
	}

	if addon.Name != "mapcoords" {
		t.Errorf("Name = %s, want mapcoords", addon.Name)
	}

	if addon.Label != "MapCoords" {
		t.Errorf("Label = %s, want MapCoords", addon.Label)
	}

	// This fixture should support all game tracks based on Clojure test
	expectedTracks := []types.GameTrack{
		types.RetailTrack,
		types.ClassicTrack,
		types.ClassicTBCTrack,
		types.ClassicWotLKTrack,
	}

	for _, track := range expectedTracks {
		if !addon.GameTrackSet[track] {
			t.Errorf("Missing expected game track: %s", track)
		}
	}
}

func TestParseAddonDetailPage_MultipleDownloadsNoTabber(t *testing.T) {
	parser := NewParser()

	content, err := loadFixture("wowinterface--addon-detail--multiple-downloads--no-tabber.html")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	url := "https://www.wowinterface.com/downloads/info25287-Skillet-Classic.html"
	result, err := parser.parseAddonDetail(url, content)
	if err != nil {
		t.Fatalf("Failed to parse addon detail: %v", err)
	}

	if len(result.AddonData) != 1 {
		t.Fatalf("Expected 1 addon, got %d", len(result.AddonData))
	}

	addon := result.AddonData[0]

	if addon.SourceID != "25287" {
		t.Errorf("SourceID = %s, want 25287", addon.SourceID)
	}

	if addon.Name != "skillet-classic" {
		t.Errorf("Name = %s, want skillet-classic", addon.Name)
	}

	if addon.Label != "Skillet-Classic" {
		t.Errorf("Label = %s, want Skillet-Classic", addon.Label)
	}

	// This should have multiple downloads
	if len(addon.LatestReleaseSet) < 1 {
		t.Errorf("Expected releases, got %d", len(addon.LatestReleaseSet))
	}

	// Game tracks come from Compatibility field, not download count
	// Just verify addon has game tracks
	if len(addon.GameTrackSet) == 0 {
		t.Error("Expected game tracks, got none")
	}
}

func TestParseAddonDetailPage_SupportsMultiple(t *testing.T) {
	parser := NewParser()

	content, err := loadFixture("wowinterface--addon-detail--supports-multiple.html")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	url := "https://www.wowinterface.com/downloads/info24870-BFAInvasionTimer.html"
	result, err := parser.parseAddonDetail(url, content)
	if err != nil {
		t.Fatalf("Failed to parse addon detail: %v", err)
	}

	if len(result.AddonData) != 1 {
		t.Fatalf("Expected 1 addon, got %d", len(result.AddonData))
	}

	addon := result.AddonData[0]

	if addon.SourceID != "24870" {
		t.Errorf("SourceID = %s, want 24870", addon.SourceID)
	}

	if addon.Name != "bfainvasiontimer" {
		t.Errorf("Name = %s, want bfainvasiontimer", addon.Name)
	}

	// Based on Clojure test, should support retail, classic, and classic-wotlk
	expectedTracks := []types.GameTrack{
		types.RetailTrack,
		types.ClassicTrack,
		types.ClassicWotLKTrack,
	}

	for _, track := range expectedTracks {
		if !addon.GameTrackSet[track] {
			t.Errorf("Missing expected game track: %s", track)
		}
	}
}

func TestParseAddonDetailPage_RemovedByAuthorRequest(t *testing.T) {
	parser := NewParser()

	content, err := loadFixture("wowinterface--addon-detail--removed-author-request.html")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	url := "https://www.wowinterface.com/downloads/info24906-AtlasWorldMapClassic.html"

	// This should be detected as a dead page and return an error or empty result
	result, err := parser.parseAddonDetail(url, content)

	// Either we get an error or empty results for removed addons
	if err == nil && len(result.AddonData) > 0 {
		// If we do get data, verify it's marked as unavailable somehow
		addon := result.AddonData[0]

		// We should detect that this addon is not available
		// The Clojure version has a `dead-page?` function
		if len(addon.LatestReleaseSet) > 0 {
			t.Error("Expected no releases for removed addon")
		}
	}
}

func TestParseAddonDetailPage_UnknownCompatibility(t *testing.T) {
	parser := NewParser()

	content, err := loadFixture("wowinterface--addon-detail--unknown-compatibility.html")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	url := "https://www.wowinterface.com/downloads/info12345-TestAddon.html"
	result, err := parser.parseAddonDetail(url, content)
	if err != nil {
		t.Fatalf("Failed to parse addon detail: %v", err)
	}

	if len(result.AddonData) != 1 {
		t.Fatalf("Expected 1 addon, got %d", len(result.AddonData))
	}

	addon := result.AddonData[0]

	// When compatibility is unknown, should default to retail
	if len(addon.GameTrackSet) == 0 {
		t.Error("Expected default game track, got none")
	}

	// Should at least have retail as default
	if !addon.GameTrackSet[types.RetailTrack] {
		t.Error("Expected retail track as default for unknown compatibility")
	}
}

func TestWoWIDateFormatting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Standard WowInterface date format",
			input:    "09-07-18 01:27 PM",
			expected: "2018-09-07T13:27:00Z",
		},
		{
			name:     "Another date example",
			input:    "03-22-24 04:59 PM",
			expected: "2024-03-22T16:59:00Z",
		},
		{
			name:     "Morning time",
			input:    "12-25-20 11:59 AM",
			expected: "2020-12-25T11:59:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parseWoWIDate(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse date: %v", err)
			}

			formatted := parsed.Format(time.RFC3339)
			if formatted != tt.expected {
				t.Errorf("Formatted date = %s, want %s", formatted, tt.expected)
			}
		})
	}
}

func TestGameTrackDetectionFromCompatibility(t *testing.T) {
	tests := []struct {
		name          string
		compatibility string
		expected      []types.GameTrack
	}{
		{
			name:          "Retail only",
			compatibility: "Shadowlands patch (9.0.5)",
			expected:      []types.GameTrack{types.RetailTrack},
		},
		{
			name:          "Classic only",
			compatibility: "Classic (1.13.7)",
			expected:      []types.GameTrack{types.ClassicTrack},
		},
		{
			name:          "TBC Classic",
			compatibility: "The Burning Crusade Classic (2.5.1)",
			expected:      []types.GameTrack{types.ClassicTBCTrack}, // TBC only, not vanilla
		},
		{
			name:          "WotLK Classic",
			compatibility: "WOTLK Patch (3.4.3)",
			expected:      []types.GameTrack{types.ClassicWotLKTrack},
		},
		{
			name:          "Multiple tracks",
			compatibility: "Compatible with Retail, Classic & TBC",
			expected:      []types.GameTrack{types.RetailTrack, types.ClassicTBCTrack, types.ClassicTrack},
		},
		{
			name:          "All game versions",
			compatibility: "Compatible with Retail, Classic & TBC Classic (1.13.7) WOTLK Patch (3.4.3)",
			expected:      []types.GameTrack{types.RetailTrack, types.ClassicTrack, types.ClassicTBCTrack, types.ClassicWotLKTrack},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGameTracks(tt.compatibility)

			if len(result) != len(tt.expected) {
				t.Errorf("parseGameTracks(%s) returned %d tracks, want %d", tt.compatibility, len(result), len(tt.expected))
			}

			resultMap := make(map[types.GameTrack]bool)
			for _, track := range result {
				resultMap[track] = true
			}

			for _, expectedTrack := range tt.expected {
				if !resultMap[expectedTrack] {
					t.Errorf("parseGameTracks(%s) missing expected track: %s", tt.compatibility, expectedTrack)
				}
			}
		})
	}
}

func TestParseCategoryGroup(t *testing.T) {
	parser := NewParser()

	content, err := loadFixture("wowinterface--landing.html")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	result, err := parser.parseCategoryGroup(content)
	if err != nil {
		t.Fatalf("Failed to parse category group: %v", err)
	}

	// Should have discovered category URLs
	if len(result.DownloadURLs) == 0 {
		t.Error("Expected download URLs from category group, got none")
	}

	// Verify we got some listing URLs
	foundListingURL := false
	for _, discoveredURL := range result.DownloadURLs {
		if len(discoveredURL) > 0 {
			foundListingURL = true
			break
		}
	}

	if !foundListingURL {
		t.Error("Expected to find category listing URLs")
	}
}

func TestParseCategoryListing(t *testing.T) {
	parser := NewParser()

	content, err := loadFixture("wowinterface--listing.html")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	url := "https://www.wowinterface.com/downloads/index.php?cid=160&sb=dec_date&so=desc&pt=f&page=1"
	result, err := parser.parseCategoryListing(url, content)
	if err != nil {
		t.Fatalf("Failed to parse category listing: %v", err)
	}

	// Should have parsed some addons from the listing
	if len(result.AddonData) == 0 {
		t.Error("Expected addon data from category listing, got none")
	}

	// Should have discovered addon detail URLs
	if len(result.DownloadURLs) == 0 {
		t.Error("Expected download URLs from category listing, got none")
	}

	// Verify addon data has required fields
	for i, addon := range result.AddonData {
		if addon.SourceID == "" {
			t.Errorf("Addon %d missing SourceID", i)
		}
		if addon.Name == "" {
			t.Errorf("Addon %d missing Name", i)
		}
		if addon.Source != types.WowInterfaceSource {
			t.Errorf("Addon %d has wrong source: %s", i, addon.Source)
		}
	}
}

// Test API fixtures
func TestParseAPIDetail_Addon21651(t *testing.T) {
	parser := NewParser()

	content, err := loadFixture("api-21651.json")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	result, err := parser.parseAPIDetail(content)
	if err != nil {
		t.Fatalf("Failed to parse API detail: %v", err)
	}

	if len(result.AddonData) != 1 {
		t.Fatalf("Expected 1 addon, got %d", len(result.AddonData))
	}

	addon := result.AddonData[0]

	// Verify basic fields from API
	if addon.SourceID != "21651" {
		t.Errorf("SourceID = %s, want 21651", addon.SourceID)
	}

	if addon.Label != "$old!it" {
		t.Errorf("Label = %s, want $old!it", addon.Label)
	}

	// Slugified name removes all symbols and punctuation
	if addon.Name != "old-it" {
		t.Errorf("Name = %s, want old-it", addon.Name)
	}

	// API returns BBCode description
	if addon.Description == "" {
		t.Error("Expected description from API, got empty")
	}

	// Check download count
	if addon.DownloadCount == nil || *addon.DownloadCount != 1187 {
		t.Errorf("DownloadCount = %v, want 1187", addon.DownloadCount)
	}

	// Check URL construction
	expectedURL := "https://www.wowinterface.com/downloads/info21651"
	if addon.URL != expectedURL {
		t.Errorf("URL = %s, want %s", addon.URL, expectedURL)
	}

	// Check timestamp is in UTC (2012-09-20T11:32:21Z)
	if addon.UpdatedDate == nil {
		t.Error("Expected UpdatedDate, got nil")
	} else {
		if addon.UpdatedDate.Location() != time.UTC {
			t.Errorf("UpdatedDate not in UTC: %v", addon.UpdatedDate.Location())
		}
		expectedTime := time.Date(2012, 9, 20, 11, 32, 21, 0, time.UTC)
		if !addon.UpdatedDate.Equal(expectedTime) {
			t.Errorf("UpdatedDate = %v, want %v", addon.UpdatedDate, expectedTime)
		}
	}
}

func TestParseAPIDetail_Addon25078(t *testing.T) {
	parser := NewParser()

	content, err := loadFixture("api-25078.json")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	result, err := parser.parseAPIDetail(content)
	if err != nil {
		t.Fatalf("Failed to parse API detail: %v", err)
	}

	if len(result.AddonData) != 1 {
		t.Fatalf("Expected 1 addon, got %d", len(result.AddonData))
	}

	addon := result.AddonData[0]

	if addon.SourceID != "25078" {
		t.Errorf("SourceID = %s, want 25078", addon.SourceID)
	}

	if addon.Label != "Better Vendor Price" {
		t.Errorf("Label = %s, want 'Better Vendor Price'", addon.Label)
	}

	if addon.Name != "better-vendor-price" {
		t.Errorf("Name = %s, want better-vendor-price", addon.Name)
	}

	// Check download count
	if addon.DownloadCount == nil || *addon.DownloadCount != 83214 {
		t.Errorf("DownloadCount = %v, want 83214", addon.DownloadCount)
	}

	// Check timestamp in UTC (2025-08-06T05:20:20Z)
	if addon.UpdatedDate == nil {
		t.Error("Expected UpdatedDate, got nil")
	} else {
		if addon.UpdatedDate.Location() != time.UTC {
			t.Errorf("UpdatedDate not in UTC: %v", addon.UpdatedDate.Location())
		}
	}
}

func TestParseAPIDetail_Addon24657(t *testing.T) {
	parser := NewParser()

	content, err := loadFixture("api-24657.json")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	result, err := parser.parseAPIDetail(content)
	if err != nil {
		t.Fatalf("Failed to parse API detail: %v", err)
	}

	if len(result.AddonData) != 1 {
		t.Fatalf("Expected 1 addon, got %d", len(result.AddonData))
	}

	addon := result.AddonData[0]

	if addon.SourceID != "24657" {
		t.Errorf("SourceID = %s, want 24657", addon.SourceID)
	}

	if addon.Label != "[Delete]" {
		t.Errorf("Label = %s, want [Delete]", addon.Label)
	}

	// Clean slug without brackets
	if addon.Name != "delete" {
		t.Errorf("Name = %s, want delete", addon.Name)
	}

	// Empty description in API is OK
	if addon.Description != "" {
		t.Logf("Description = %s (empty is expected for this addon)", addon.Description)
	}
}

// Test HTML fixtures for integration test addons
func TestParseAddonDetail_Addon21651_HTML(t *testing.T) {
	parser := NewParser()

	content, err := loadFixture("addon-21651.html")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	url := "https://www.wowinterface.com/downloads/info21651"
	result, err := parser.parseAddonDetail(url, content)
	if err != nil {
		t.Fatalf("Failed to parse addon detail: %v", err)
	}

	if len(result.AddonData) != 1 {
		t.Fatalf("Expected 1 addon, got %d", len(result.AddonData))
	}

	addon := result.AddonData[0]

	if addon.SourceID != "21651" {
		t.Errorf("SourceID = %s, want 21651", addon.SourceID)
	}

	// HTML should provide tags that API lacks
	if len(addon.TagSet) == 0 {
		t.Error("Expected tags from HTML, got none")
	}

	// HTML should provide clean description (not BBCode)
	if addon.Description == "" {
		t.Error("Expected description from HTML, got empty")
	}

	// HTML should provide created-date that API lacks
	if addon.CreatedDate == nil {
		t.Error("Expected CreatedDate from HTML, got nil")
	} else {
		// Should be in UTC
		if addon.CreatedDate.Location() != time.UTC {
			t.Errorf("CreatedDate not in UTC: %v", addon.CreatedDate.Location())
		}
	}
}

func TestParseAddonDetail_Addon25078_HTML(t *testing.T) {
	parser := NewParser()

	content, err := loadFixture("addon-25078.html")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	url := "https://www.wowinterface.com/downloads/info25078"
	result, err := parser.parseAddonDetail(url, content)
	if err != nil {
		t.Fatalf("Failed to parse addon detail: %v", err)
	}

	if len(result.AddonData) != 1 {
		t.Fatalf("Expected 1 addon, got %d", len(result.AddonData))
	}

	addon := result.AddonData[0]

	if addon.SourceID != "25078" {
		t.Errorf("SourceID = %s, want 25078", addon.SourceID)
	}

	// This addon should have multiple tags
	if len(addon.TagSet) == 0 {
		t.Error("Expected tags from HTML, got none")
	}

	// Should have game tracks
	if len(addon.GameTrackSet) == 0 {
		t.Error("Expected game tracks, got none")
	}

	// Should have releases
	if len(addon.LatestReleaseSet) == 0 {
		t.Error("Expected releases, got none")
	}
}

func TestParseAddonDetail_Addon24637_MultiGameTracks(t *testing.T) {
	// Test addon with multiple game version downloads (retail, classic, tbc, wotlk, cata)
	htmlPath := "test/fixtures/addon-24637-multi-game-tracks.html"
	htmlContent, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("Failed to read fixture file: %v", err)
	}

	parser := NewParser()
	result, err := parser.parseAddonDetail("https://www.wowinterface.com/downloads/info24637", htmlContent)
	if err != nil {
		t.Fatalf("parseAddonDetail failed: %v", err)
	}

	if len(result.AddonData) == 0 {
		t.Fatal("Expected addon data, got none")
	}

	addon := result.AddonData[0]

	// Basic validation
	if addon.SourceID != "24637" {
		t.Errorf("Expected SourceID 24637, got %s", addon.SourceID)
	}

	if addon.Label != "MaxDps Rotation Helper" {
		t.Errorf("Expected label 'MaxDps Rotation Helper', got %s", addon.Label)
	}

	// Critical: Should detect ALL 5 game tracks from download sections
	expectedTracks := []types.GameTrack{
		types.RetailTrack,
		types.ClassicTrack,
		types.ClassicTBCTrack,
		types.ClassicWotLKTrack,
		types.ClassicCataTrack,
	}

	for _, track := range expectedTracks {
		if !addon.GameTrackSet[track] {
			t.Errorf("Missing expected game track: %s\nFound tracks: %v", track, addon.GameTrackSet)
		}
	}

	if len(addon.GameTrackSet) != len(expectedTracks) {
		t.Errorf("Expected %d game tracks, got %d: %v",
			len(expectedTracks), len(addon.GameTrackSet), addon.GameTrackSet)
	}

	// Should have 5 releases (one for each game version)
	if len(addon.LatestReleaseSet) < 5 {
		t.Errorf("Expected at least 5 releases, got %d", len(addon.LatestReleaseSet))
	}

	// Each release should have a game track assigned
	for i, release := range addon.LatestReleaseSet {
		if release.GameTrack == "" {
			t.Errorf("Release %d missing GameTrack: %+v", i, release)
		}
	}
}

func TestParseAddonDetail_Addon25551_ClassicOnly(t *testing.T) {
	// Test classic-only addon that should NOT have retail track
	// This addon has Compatibility: "Classic Patch (1.13.4)"
	htmlPath := "test/fixtures/addon-25551-classic-only.html"
	htmlContent, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("Failed to read fixture file: %v", err)
	}

	parser := NewParser()
	result, err := parser.parseAddonDetail("https://www.wowinterface.com/downloads/info25551", htmlContent)
	if err != nil {
		t.Fatalf("parseAddonDetail failed: %v", err)
	}

	if len(result.AddonData) == 0 {
		t.Fatal("Expected addon data, got none")
	}

	addon := result.AddonData[0]

	// Should ONLY have classic track
	if !addon.GameTrackSet[types.ClassicTrack] {
		t.Errorf("Missing classic track. Found: %v", addon.GameTrackSet)
	}

	if addon.GameTrackSet[types.RetailTrack] {
		t.Errorf("Should NOT have retail track for classic-only addon. Found: %v", addon.GameTrackSet)
	}

	if len(addon.GameTrackSet) != 1 {
		t.Errorf("Expected exactly 1 game track (classic), got %d: %v",
			len(addon.GameTrackSet), addon.GameTrackSet)
	}
}

func TestParseAddonDetail_Addon24657_HTML(t *testing.T) {
	parser := NewParser()

	content, err := loadFixture("addon-24657.html")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	url := "https://www.wowinterface.com/downloads/info24657"
	result, err := parser.parseAddonDetail(url, content)
	if err != nil {
		t.Fatalf("Failed to parse addon detail: %v", err)
	}

	if len(result.AddonData) != 1 {
		t.Fatalf("Expected 1 addon, got %d", len(result.AddonData))
	}

	addon := result.AddonData[0]

	if addon.SourceID != "24657" {
		t.Errorf("SourceID = %s, want 24657", addon.SourceID)
	}

	// Even with empty API description, HTML might have tags
	if len(addon.TagSet) > 0 {
		t.Logf("Found %d tags from HTML", len(addon.TagSet))
	}
}
