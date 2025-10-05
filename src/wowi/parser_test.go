package wowi

import (
	"strings"
	"testing"

	"github.com/ogri-la/strongbox-catalogue-builder-go/src/types"
)

func TestURLClassifier_ClassifyURL(t *testing.T) {
	classifier := NewURLClassifier()

	tests := []struct {
		name     string
		url      string
		expected URLType
	}{
		{
			name:     "API file list",
			url:      "https://api.mmoui.com/v4/game/WOW/filelist.json",
			expected: URLTypeAPIFileList,
		},
		{
			name:     "API addon detail",
			url:      "https://api.mmoui.com/v4/game/WOW/filedetails/12345.json",
			expected: URLTypeAPIDetail,
		},
		{
			name:     "Addon detail page",
			url:      "https://www.wowinterface.com/downloads/info12345",
			expected: URLTypeAddonDetail,
		},
		{
			name:     "Category group page (deprecated)",
			url:      "https://www.wowinterface.com/addons.php",
			expected: URLTypeUnknown, // Category groups no longer used for discovery
		},
		{
			name:     "Category listing page",
			url:      "https://www.wowinterface.com/downloads/index.php?cid=160&page=1",
			expected: URLTypeCategoryListing,
		},
		{
			name:     "Unknown URL",
			url:      "https://example.com/unknown",
			expected: URLTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.ClassifyURL(tt.url)
			if result != tt.expected {
				t.Errorf("ClassifyURL(%s) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestExtractSourceIDFromHref(t *testing.T) {
	tests := []struct {
		name     string
		href     string
		expected string
	}{
		{
			name:     "Valid href with source ID",
			href:     "fileinfo.php?s=c33edd26881a6a6509fd43e9a871809c&id=23145",
			expected: "23145",
		},
		{
			name:     "Another valid href",
			href:     "fileinfo.php?id=12345&other=param",
			expected: "12345",
		},
		{
			name:     "No ID in href",
			href:     "fileinfo.php?s=some-hash",
			expected: "",
		},
		{
			name:     "Empty href",
			href:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSourceIDFromHref(tt.href)
			if result != tt.expected {
				t.Errorf("extractSourceIDFromHref(%s) = %s, want %s", tt.href, result, tt.expected)
			}
		})
	}
}

func TestExtractSourceIDFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "Valid addon detail URL",
			url:      "https://www.wowinterface.com/downloads/info23145",
			expected: "23145",
		},
		{
			name:     "Another valid URL",
			url:      "https://www.wowinterface.com/downloads/info12345",
			expected: "12345",
		},
		{
			name:     "No ID in URL",
			url:      "https://www.wowinterface.com/downloads/",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSourceIDFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("extractSourceIDFromURL(%s) = %s, want %s", tt.url, result, tt.expected)
			}
		})
	}
}

func TestParseWoWIDate(t *testing.T) {
	tests := []struct {
		name        string
		dateStr     string
		expectError bool
	}{
		{
			name:        "Valid WowInterface date",
			dateStr:     "09-07-18 01:27 PM",
			expectError: false,
		},
		{
			name:        "Another valid date",
			dateStr:     "12-25-20 11:59 AM",
			expectError: false,
		},
		{
			name:        "Invalid date format",
			dateStr:     "2018-09-07 13:27:00",
			expectError: true,
		},
		{
			name:        "Empty string",
			dateStr:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseWoWIDate(tt.dateStr)
			if tt.expectError && err == nil {
				t.Errorf("parseWoWIDate(%s) expected error but got none", tt.dateStr)
			}
			if !tt.expectError && err != nil {
				t.Errorf("parseWoWIDate(%s) unexpected error: %v", tt.dateStr, err)
			}
			if !tt.expectError && err == nil {
				// Verify the result is a valid time
				if result.IsZero() {
					t.Errorf("parseWoWIDate(%s) returned zero time", tt.dateStr)
				}
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple addon name",
			input:    "AdiBags",
			expected: "adibags",
		},
		{
			name:     "Addon name with spaces",
			input:    "Deadly Boss Mods",
			expected: "deadly-boss-mods",
		},
		{
			name:     "Complex addon name with brackets",
			input:    "BigWigs [LittleWigs]",
			expected: "bigwigs-littlewigs", // Clean slug without punctuation
		},
		{
			name:     "Name with numbers",
			input:    "Grid2",
			expected: "grid2",
		},
		{
			name:     "Name with symbols and punctuation",
			input:    "Addon++_Name!",
			expected: "addon-name", // All symbols/punctuation removed
		},
		{
			name:     "$old!it test case",
			input:    "$old!it",
			expected: "old-it", // $ and ! removed
		},
		{
			name:     "Brackets only",
			input:    "[Delete]",
			expected: "delete", // Clean slug without brackets
		},
		{
			name:     "Mixed symbols",
			input:    "%^& Off",
			expected: "off", // All symbols removed
		},
		{
			name:     "Leading numbers",
			input:    "123 Addon",
			expected: "123-addon",
		},
		{
			name:     "Multiple consecutive separators",
			input:    "Addon   --  Name",
			expected: "addon-name", // Consecutive separators become single hyphen
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := slugify(tt.input)
			if result != tt.expected {
				t.Errorf("slugify(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseGameTracks(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []types.GameTrack
	}{
		{
			name:     "Retail only",
			text:     "Compatible with retail",
			expected: []types.GameTrack{types.RetailTrack},
		},
		{
			name:     "Classic only",
			text:     "Compatible with classic",
			expected: []types.GameTrack{types.ClassicTrack},
		},
		{
			name:     "TBC classic",
			text:     "Compatible with TBC classic",
			expected: []types.GameTrack{types.ClassicTBCTrack},
		},
		{
			name:     "Wrath classic",
			text:     "Compatible with wrath classic",
			expected: []types.GameTrack{types.ClassicWotLKTrack},
		},
		{
			name:     "Multiple tracks",
			text:     "Compatible with retail, classic, and TBC",
			expected: []types.GameTrack{types.RetailTrack, types.ClassicTBCTrack, types.ClassicTrack},
		},
		{
			name:     "Retail, Classic & TBC (real-world example)",
			text:     "Compatible with Retail, Classic & TBC",
			expected: []types.GameTrack{types.RetailTrack, types.ClassicTrack, types.ClassicTBCTrack},
		},
		{
			name:     "Cataclysm Classic only",
			text:     "Cataclysm Classic (4.4.2)",
			expected: []types.GameTrack{types.ClassicCataTrack},
		},
		{
			name:     "Classic with version number",
			text:     "Classic (1.15.2)",
			expected: []types.GameTrack{types.ClassicTrack},
		},
		{
			name:     "The Burning Crusade Classic (full name)",
			text:     "The Burning Crusade Classic (2.5.4)",
			expected: []types.GameTrack{types.ClassicTBCTrack},
		},
		{
			name:     "No tracks mentioned",
			text:     "This is just some text",
			expected: []types.GameTrack{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGameTracks(tt.text)
			if len(result) != len(tt.expected) {
				t.Errorf("parseGameTracks(%s) returned %d tracks, want %d\nGot: %v\nWant: %v",
					tt.text, len(result), len(tt.expected), result, tt.expected)
				return
			}

			// Convert to maps for order-independent comparison
			resultMap := make(map[types.GameTrack]bool)
			for _, track := range result {
				resultMap[track] = true
			}
			expectedMap := make(map[types.GameTrack]bool)
			for _, track := range tt.expected {
				expectedMap[track] = true
			}

			// Check all expected tracks are present
			for track := range expectedMap {
				if !resultMap[track] {
					t.Errorf("parseGameTracks(%s) missing expected track: %s\nGot: %v\nWant: %v",
						tt.text, track, result, tt.expected)
				}
			}

			// Check no unexpected tracks are present
			for track := range resultMap {
				if !expectedMap[track] {
					t.Errorf("parseGameTracks(%s) has unexpected track: %s\nGot: %v\nWant: %v",
						tt.text, track, result, tt.expected)
				}
			}
		})
	}
}

func TestGameVersionToGameTrack(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected types.GameTrack
	}{
		{
			name:     "Classic version",
			version:  "1.13.2",
			expected: types.ClassicTrack,
		},
		{
			name:     "TBC version",
			version:  "2.5.1",
			expected: types.ClassicTBCTrack,
		},
		{
			name:     "Wrath version",
			version:  "3.4.0",
			expected: types.ClassicWotLKTrack,
		},
		{
			name:     "Cata version",
			version:  "4.3.4",
			expected: types.ClassicCataTrack,
		},
		{
			name:     "Mists version",
			version:  "5.4.8",
			expected: types.ClassicMistsTrack,
		},
		{
			name:     "Retail version",
			version:  "10.2.5",
			expected: types.RetailTrack,
		},
		{
			name:     "Short version",
			version:  "1",
			expected: types.RetailTrack,
		},
		{
			name:     "Empty version",
			version:  "",
			expected: types.RetailTrack,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gameVersionToGameTrack(tt.version)
			if result != tt.expected {
				t.Errorf("gameVersionToGameTrack(%s) = %s, want %s", tt.version, result, tt.expected)
			}
		})
	}
}

func TestParseAPIFileList(t *testing.T) {
	parser := NewParser()

	// Sample API response data
	jsonData := `[
		{
			"id": 23145,
			"title": "AdiBags",
			"lastUpdate": 1640995200,
			"gameVersions": ["10.2.5", "1.13.2"]
		},
		{
			"id": 12345,
			"title": "Deadly Boss Mods",
			"lastUpdate": 1640995300,
			"gameVersions": ["10.2.5"]
		}
	]`

	result, err := parser.parseAPIFileList([]byte(jsonData))
	if err != nil {
		t.Fatalf("parseAPIFileList() unexpected error: %v", err)
	}

	if len(result.AddonData) != 2 {
		t.Errorf("parseAPIFileList() returned %d addons, want 2", len(result.AddonData))
	}

	// Check first addon
	addon1 := result.AddonData[0]
	if addon1.SourceID != "23145" {
		t.Errorf("First addon SourceID = %s, want 23145", addon1.SourceID)
	}
	if addon1.Label != "AdiBags" {
		t.Errorf("First addon Label = %s, want AdiBags", addon1.Label)
	}
	if addon1.Name != "adibags" {
		t.Errorf("First addon Name = %s, want adibags", addon1.Name)
	}

	// Check that URLs were generated
	if len(result.DownloadURLs) == 0 {
		t.Error("parseAPIFileList() generated no download URLs")
	}
}

func TestParseAPIDetail(t *testing.T) {
	parser := NewParser()

	// Sample API detail response (based on actual WowInterface API)
	jsonData := `[{
		"id": 25078,
		"categoryId": 20,
		"version": "v1.22.0",
		"lastUpdate": 1754440820000,
		"checksum": "77429fa58f1a4e5201e82d2d04afb4bc",
		"fileName": "BetterVendorPrice-v1.22.0.zip",
		"downloadUri": "https://cdn.wowinterface.com/downloads/getfile.php?id=25078",
		"title": "Better Vendor Price",
		"author": "MooreaTv",
		"description": "A helpful addon for pricing",
		"downloads": 83214,
		"downloadsMonthly": 33,
		"favorites": 188
	}]`

	result, err := parser.parseAPIDetail([]byte(jsonData))
	if err != nil {
		t.Fatalf("parseAPIDetail() unexpected error: %v", err)
	}

	if len(result.AddonData) != 1 {
		t.Fatalf("parseAPIDetail() returned %d addons, want 1", len(result.AddonData))
	}

	addon := result.AddonData[0]

	// Check basic fields
	if addon.SourceID != "25078" {
		t.Errorf("SourceID = %s, want 25078", addon.SourceID)
	}

	if addon.Label != "Better Vendor Price" {
		t.Errorf("Label = %s, want 'Better Vendor Price'", addon.Label)
	}

	if addon.Name != "better-vendor-price" {
		t.Errorf("Name = %s, want better-vendor-price", addon.Name)
	}

	if addon.Source != types.WowInterfaceSource {
		t.Errorf("Source = %s, want %s", addon.Source, types.WowInterfaceSource)
	}

	// Filename depends on API version detected
	if addon.Filename != "api-detail-v4.json" && addon.Filename != "api-detail-v3.json" {
		t.Errorf("Filename = %s, want api-detail-v4.json or api-detail-v3.json", addon.Filename)
	}

	// Check that WoWI data was stored
	if addon.WoWI == nil {
		t.Error("Expected WoWI data to be stored, got nil")
	}

	// Verify some WoWI fields were captured
	if author, ok := addon.WoWI["author"].(string); !ok || author != "MooreaTv" {
		t.Errorf("WoWI author = %v, want MooreaTv", addon.WoWI["author"])
	}
}

func TestParseAPIDetail_EmptyArray(t *testing.T) {
	parser := NewParser()

	jsonData := `[]`

	result, err := parser.parseAPIDetail([]byte(jsonData))
	if err != nil {
		t.Fatalf("parseAPIDetail() unexpected error: %v", err)
	}

	if len(result.AddonData) != 0 {
		t.Errorf("parseAPIDetail() returned %d addons for empty array, want 0", len(result.AddonData))
	}
}

func TestParseAPIDetail_InvalidJSON(t *testing.T) {
	parser := NewParser()

	jsonData := `{invalid json`

	_, err := parser.parseAPIDetail([]byte(jsonData))
	if err == nil {
		t.Error("parseAPIDetail() expected error for invalid JSON, got nil")
	}
}

func TestCleanDescription(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Simple description",
			input:    "A simple addon that does cool things.",
			expected: "A simple addon that does cool things.",
		},
		{
			name:     "Multi-line with decorative separator",
			input:    "==========\nThis addon helps you.\nMore details here.",
			expected: "This addon helps you.",
		},
		{
			name:     "Skip 'About' prefix",
			input:    "About: This is an awesome addon.\nIt does things.",
			expected: "It does things.",
		},
		{
			name:     "Skip 'Description' prefix",
			input:    "Description\nHelps manage your inventory.",
			expected: "Helps manage your inventory.",
		},
		{
			name:     "Skip 'What is it' phrase",
			input:    "What is it?\nA utility addon for tracking quests.",
			expected: "A utility addon for tracking quests.",
		},
		{
			name:     "Skip donation message",
			input:    "Donate via PayPal!\nThis addon improves your UI.",
			expected: "This addon improves your UI.",
		},
		{
			name:     "Skip greeting",
			input:    "Hello!\nWelcome to my addon.",
			expected: "Welcome to my addon.",
		},
		{
			name:     "Skip 'Discontinued' warning",
			input:    "Discontinued: No longer maintained\nShows damage meters.",
			expected: "Shows damage meters.",
		},
		{
			name:     "Skip multiple leading lines",
			input:    "===\nIntro:\nHello there!\nThis addon tracks achievements.",
			expected: "This addon tracks achievements.",
		},
		{
			name:     "Truncate long description",
			input:    strings.Repeat("A", 1500),
			expected: strings.Repeat("A", 1000),
		},
		{
			name:     "Skip 'Overview:' line",
			input:    "Overview:\nManage your bags efficiently.",
			expected: "Manage your bags efficiently.",
		},
		{
			name:     "Don't skip 'aboutface' (not a prefix match)",
			input:    "aboutface is a military command.",
			expected: "aboutface is a military command.",
		},
		{
			name:     "All lines are decorative",
			input:    "===\n---\n***",
			expected: "",
		},
		{
			name:     "Skip 'Important' message",
			input:    "Important: Read the documentation\nProvides DPS tracking.",
			expected: "Provides DPS tracking.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanDescription(tt.input)
			if result != tt.expected {
				t.Errorf("cleanDescription() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsPureNonAlphanumeric(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: true,
		},
		{
			name:     "Only equals signs",
			input:    "========",
			expected: true,
		},
		{
			name:     "Only dashes",
			input:    "--------",
			expected: true,
		},
		{
			name:     "Mixed punctuation",
			input:    "=== *** ---",
			expected: true,
		},
		{
			name:     "Contains letters",
			input:    "=== text ===",
			expected: false,
		},
		{
			name:     "Contains numbers",
			input:    "--- 123 ---",
			expected: false,
		},
		{
			name:     "Single letter",
			input:    "a",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPureNonAlphanumeric(tt.input)
			if result != tt.expected {
				t.Errorf("isPureNonAlphanumeric(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsLowQualityDescription(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Too short - single word",
			input:    "null",
			expected: true,
		},
		{
			name:     "Too short - few chars",
			input:    "1.3",
			expected: true,
		},
		{
			name:     "Version number with update",
			input:    "10.1.5 UPDATE:",
			expected: true,
		},
		{
			name:     "Version number simple",
			input:    "1.0 release",
			expected: true,
		},
		{
			name:     "Version with v prefix",
			input:    "v1.2.3 fixes",
			expected: true,
		},
		{
			name:     "Date MM/DD/YYYY",
			input:    "05/03/2025 - Updated",
			expected: true,
		},
		{
			name:     "Date YYYY-MM-DD",
			input:    "2025-05-03 - New version",
			expected: true,
		},
		{
			name:     "Null exact match",
			input:    "null",
			expected: true,
		},
		{
			name:     "Update prefix",
			input:    "UPDATE: Fixed bugs",
			expected: true,
		},
		{
			name:     "No spaces (single word)",
			input:    "SomeLongSingleWord",
			expected: true,
		},
		{
			name:     "Good quality - normal description",
			input:    "This addon helps you manage your inventory efficiently.",
			expected: false,
		},
		{
			name:     "Good quality - minimum length with spaces",
			input:    "Tracks your DPS",
			expected: false,
		},
		{
			name:     "Good quality - sentence",
			input:    "A simple raid frame addon.",
			expected: false,
		},
		{
			name:     "Number in middle is ok",
			input:    "This is version 2.0 of the addon.",
			expected: false,
		},
		{
			name:     "AddonName by AuthorName pattern",
			input:    "BigWigs by Funkydude",
			expected: true,
		},
		{
			name:     "'by' in longer description is ok",
			input:    "This addon was created by the author to solve problems.",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLowQualityDescription(tt.input)
			if result != tt.expected {
				t.Errorf("isLowQualityDescription(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCleanDescriptionWithQualityFilter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Skip version, use next line",
			input:    "1.3\nThis addon provides raid utilities.",
			expected: "This addon provides raid utilities.",
		},
		{
			name:     "Skip multiple bad lines",
			input:    "null\n10.1.5 UPDATE:\nManages your action bars effectively.",
			expected: "Manages your action bars effectively.",
		},
		{
			name:     "Skip date, use description",
			input:    "05/03/2025 - V3.1\nProvides enhanced tooltips.",
			expected: "Provides enhanced tooltips.",
		},
		{
			name:     "Fallback to short line if nothing better",
			input:    "hud",
			expected: "hud",
		},
		{
			name:     "Don't use 'null' as fallback",
			input:    "null",
			expected: "",
		},
		{
			name:     "Don't use 'undefined' as fallback",
			input:    "===\nundefined",
			expected: "",
		},
		{
			name:     "Use good line even if bad lines exist later",
			input:    "Tracks gathering nodes on your map.\n1.0\nv2.3",
			expected: "Tracks gathering nodes on your map.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanDescription(tt.input)
			if result != tt.expected {
				t.Errorf("cleanDescription() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestShouldSkipLeadingLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "About prefix",
			input:    "About this addon",
			expected: true,
		},
		{
			name:     "Description prefix",
			input:    "Description: Tracks DPS",
			expected: true,
		},
		{
			name:     "Donate message",
			input:    "Donate to support development",
			expected: true,
		},
		{
			name:     "Hello greeting",
			input:    "Hello, welcome!",
			expected: true,
		},
		{
			name:     "What is it question",
			input:    "What is it?",
			expected: true,
		},
		{
			name:     "Discontinued notice",
			input:    "Discontinued - no updates",
			expected: true,
		},
		{
			name:     "Not a prefix (word boundary)",
			input:    "aboutface command",
			expected: false,
		},
		{
			name:     "Normal content",
			input:    "This addon helps you manage inventory",
			expected: false,
		},
		{
			name:     "Contains but doesn't start with skip word",
			input:    "An overview of features",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSkipLeadingLine(tt.input)
			if result != tt.expected {
				t.Errorf("shouldSkipLeadingLine(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
