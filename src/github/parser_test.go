package github

import (
	"os"
	"testing"
	"time"

	"github.com/ogri-la/strongbox-catalogue-builder-go/src/types"
)

func TestParseCSV(t *testing.T) {
	csvContent, err := os.ReadFile("test/fixtures/github-catalogue--dummy.csv")
	if err != nil {
		t.Fatalf("Failed to read fixture: %v", err)
	}

	parser := NewParser()
	addons, err := parser.ParseCSV(string(csvContent))
	if err != nil {
		t.Fatalf("ParseCSV failed: %v", err)
	}

	if len(addons) != 5 {
		t.Fatalf("Expected 5 addons, got %d", len(addons))
	}

	// Test first addon - has description and single flavor
	addon1 := addons[0]
	if addon1.Name != "premade-applicants-filter" {
		t.Errorf("Expected name 'premade-applicants-filter', got '%s'", addon1.Name)
	}
	if addon1.Label != "premade-applicants-filter" {
		t.Errorf("Expected label 'premade-applicants-filter', got '%s'", addon1.Label)
	}
	if addon1.Source != "github" {
		t.Errorf("Expected source 'github', got '%s'", addon1.Source)
	}
	if addon1.SourceID != "0xbs/premade-applicants-filter" {
		t.Errorf("Expected source-id '0xbs/premade-applicants-filter', got '%s'", addon1.SourceID)
	}
	if addon1.URL != "https://github.com/0xbs/premade-applicants-filter" {
		t.Errorf("Expected URL 'https://github.com/0xbs/premade-applicants-filter', got '%s'", addon1.URL)
	}
	if addon1.Description != "Allows filtering of premade applicants using advanced filter expressions." {
		t.Errorf("Expected description, got '%s'", addon1.Description)
	}
	if addon1.DownloadCount == nil || *addon1.DownloadCount != 34076 {
		t.Errorf("Expected download count 34076, got %v", addon1.DownloadCount)
	}
	if len(addon1.TagList) != 0 {
		t.Errorf("Expected empty tag list, got %v", addon1.TagList)
	}

	expectedDate, _ := time.Parse(time.RFC3339, "2021-12-26T09:40:18Z")
	if !addon1.UpdatedDate.Equal(expectedDate) {
		t.Errorf("Expected updated date %v, got %v", expectedDate, addon1.UpdatedDate)
	}

	if len(addon1.GameTrackList) != 1 || addon1.GameTrackList[0] != types.RetailTrack {
		t.Errorf("Expected game tracks [retail], got %v", addon1.GameTrackList)
	}

	// Test second addon - no description, multiple flavors
	addon2 := addons[1]
	if addon2.Name != "arenaleaveconfirmer" {
		t.Errorf("Expected name 'arenaleaveconfirmer', got '%s'", addon2.Name)
	}
	if addon2.Label != "ArenaLeaveConfirmer" {
		t.Errorf("Expected label 'ArenaLeaveConfirmer', got '%s'", addon2.Label)
	}
	if addon2.Description != "" {
		t.Errorf("Expected empty description, got '%s'", addon2.Description)
	}
	if addon2.DownloadCount == nil || *addon2.DownloadCount != 12345 {
		t.Errorf("Expected download count 12345, got %v", addon2.DownloadCount)
	}

	// Game tracks should be sorted alphabetically
	expectedTracks := []types.GameTrack{types.ClassicTrack, types.ClassicTBCTrack, types.RetailTrack}
	if len(addon2.GameTrackList) != len(expectedTracks) {
		t.Errorf("Expected %d game tracks, got %d", len(expectedTracks), len(addon2.GameTrackList))
	}
	for i, track := range expectedTracks {
		if addon2.GameTrackList[i] != track {
			t.Errorf("Expected game track %s at position %d, got %s", track, i, addon2.GameTrackList[i])
		}
	}

	// Test fifth addon - has description with comma and high download count
	addon5 := addons[4]
	if addon5.Name != "chatcleaner" {
		t.Errorf("Expected name 'chatcleaner', got '%s'", addon5.Name)
	}
	expectedDesc := "Makes system chat messages prettier and tidier, and reduces the need for multiple chat windows."
	if addon5.Description != expectedDesc {
		t.Errorf("Expected description '%s', got '%s'", expectedDesc, addon5.Description)
	}
	if addon5.DownloadCount == nil || *addon5.DownloadCount != 541101 {
		t.Errorf("Expected download count 541101, got %v", addon5.DownloadCount)
	}
}

func TestGuessGameTrack(t *testing.T) {
	tests := []struct {
		name     string
		flavor   string
		expected types.GameTrack
	}{
		{"mainline", "mainline", types.RetailTrack},
		{"retail", "retail", types.RetailTrack},
		{"classic", "classic", types.ClassicTrack},
		{"vanilla", "vanilla", types.ClassicTrack},
		{"bcc", "bcc", types.ClassicTBCTrack},
		{"tbc", "tbc", types.ClassicTBCTrack},
		{"wrath", "wrath", types.ClassicWotLKTrack},
		{"wotlk", "wotlk", types.ClassicWotLKTrack},
		{"cata", "cata", types.ClassicCataTrack},
		{"cataclysm", "cataclysm", types.ClassicCataTrack},
		{"mists", "mists", types.ClassicMistsTrack},
		{"mop", "mop", types.ClassicMistsTrack},
		{"unknown", "unknown", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := guessGameTrack(tt.flavor)
			if result != tt.expected {
				t.Errorf("guessGameTrack(%s) = %s, expected %s", tt.flavor, result, tt.expected)
			}
		})
	}
}
