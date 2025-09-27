package wowi

import (
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
			name:     "Category group page",
			url:      "https://www.wowinterface.com/addons.php",
			expected: URLTypeCategoryGroup,
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
			name:     "Complex addon name",
			input:    "BigWigs [LittleWigs]",
			expected: "bigwigs-littlewigs",
		},
		{
			name:     "Name with numbers",
			input:    "Grid2",
			expected: "grid2",
		},
		{
			name:     "Name with special characters",
			input:    "Addon++_Name!",
			expected: "addon-name",
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
			expected: []types.GameTrack{types.RetailTrack, types.ClassicTBCTrack},
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
				t.Errorf("parseGameTracks(%s) returned %d tracks, want %d", tt.text, len(result), len(tt.expected))
			}
			for i, track := range result {
				if i >= len(tt.expected) || track != tt.expected[i] {
					t.Errorf("parseGameTracks(%s) = %v, want %v", tt.text, result, tt.expected)
					break
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