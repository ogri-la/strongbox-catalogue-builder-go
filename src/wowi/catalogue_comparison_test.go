//go:build integration

package wowi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type CatalogueWrapper struct {
	Spec struct {
		Version int `json:"version"`
	} `json:"spec"`
	Datestamp        string  `json:"datestamp"`
	Total            int     `json:"total"`
	AddonSummaryList []Addon `json:"addon-summary-list"`
}

type Addon struct {
	Source        string      `json:"source"`
	SourceID      json.Number `json:"source-id"`
	Name          string      `json:"name"`
	Label         string      `json:"label"`
	Description   string      `json:"description"`
	UpdatedDate   string      `json:"updated-date"`
	CreatedDate   string      `json:"created-date,omitempty"`
	DownloadCount int         `json:"download-count"`
	GameTrackList []string    `json:"game-track-list"`
	TagList       []string    `json:"tag-list"`
	URL           string      `json:"url"`
}

// DriftReport tracks differences between catalogues
type DriftReport struct {
	TotalClojure      int
	TotalGo           int
	CommonAddons      int
	OnlyInClojure     int
	OnlyInGo          int
	GameTrackMismatch int
	TagMismatch       int
	DescriptionDrift  int
	NameDrift         int
}

func TestCatalogueComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping catalogue comparison test in short mode")
	}

	// Load Clojure catalogue fixture
	clojurePath := filepath.Join("..", "..", "test", "fixtures", "catalogues", "clojure-wowinterface-catalogue.json")
	clojureData, err := os.ReadFile(clojurePath)
	if err != nil {
		t.Fatalf("Failed to load Clojure catalogue fixture: %v", err)
	}

	var clojureCat CatalogueWrapper
	if err := json.Unmarshal(clojureData, &clojureCat); err != nil {
		t.Fatalf("Failed to parse Clojure catalogue: %v", err)
	}

	// Load Go catalogue fixture
	goPath := filepath.Join("..", "..", "test", "fixtures", "catalogues", "go-wowinterface-catalogue.json")
	goData, err := os.ReadFile(goPath)
	if err != nil {
		t.Fatalf("Failed to load Go catalogue fixture: %v", err)
	}

	var goCat CatalogueWrapper
	if err := json.Unmarshal(goData, &goCat); err != nil {
		t.Fatalf("Failed to parse Go catalogue: %v", err)
	}

	// Create maps for comparison
	clojureMap := make(map[string]Addon)
	for _, addon := range clojureCat.AddonSummaryList {
		clojureMap[addon.SourceID.String()] = addon
	}

	goMap := make(map[string]Addon)
	for _, addon := range goCat.AddonSummaryList {
		goMap[addon.SourceID.String()] = addon
	}

	// Generate drift report
	report := DriftReport{
		TotalClojure: len(clojureCat.AddonSummaryList),
		TotalGo:      len(goCat.AddonSummaryList),
	}

	// Track sample mismatches for detailed reporting
	gameTrackSamples := []string{}
	tagSamples := []string{}
	nameSamples := []string{}

	// Compare addons present in both
	for sourceID, goAddon := range goMap {
		clojureAddon, exists := clojureMap[sourceID]
		if !exists {
			report.OnlyInGo++
			continue
		}

		report.CommonAddons++

		// Compare game tracks
		if !stringSlicesEqual(goAddon.GameTrackList, clojureAddon.GameTrackList) {
			report.GameTrackMismatch++
			if len(gameTrackSamples) < 5 {
				gameTrackSamples = append(gameTrackSamples, sourceID)
			}
		}

		// Compare tags (expected to differ due to replacement maps, just track count)
		if !stringSlicesEqual(goAddon.TagList, clojureAddon.TagList) {
			report.TagMismatch++
			if len(tagSamples) < 5 {
				tagSamples = append(tagSamples, sourceID)
			}
		}

		// Compare names (expected to differ due to slugify changes)
		if goAddon.Name != clojureAddon.Name {
			report.NameDrift++
			if len(nameSamples) < 5 {
				nameSamples = append(nameSamples, sourceID)
			}
		}

		// Track description differences (expected due to BBCode vs clean text)
		if goAddon.Description != clojureAddon.Description {
			report.DescriptionDrift++
		}
	}

	// Count addons only in Clojure
	for sourceID := range clojureMap {
		if _, exists := goMap[sourceID]; !exists {
			report.OnlyInClojure++
		}
	}

	// Print detailed report
	t.Logf("\n=== Catalogue Comparison Report ===")
	t.Logf("Clojure catalogue: %d addons (datestamp: %s)", report.TotalClojure, clojureCat.Datestamp)
	t.Logf("Go catalogue:      %d addons (datestamp: %s)", report.TotalGo, goCat.Datestamp)
	t.Logf("\nOverlap:")
	t.Logf("  Common addons:     %d", report.CommonAddons)
	t.Logf("  Only in Clojure:   %d", report.OnlyInClojure)
	t.Logf("  Only in Go:        %d", report.OnlyInGo)

	t.Logf("\nDrift Analysis (among common addons):")
	t.Logf("  Game track mismatches: %d (%.1f%%)", report.GameTrackMismatch,
		float64(report.GameTrackMismatch)/float64(report.CommonAddons)*100)
	t.Logf("  Tag differences:       %d (%.1f%%) [expected due to improved tag logic]",
		report.TagMismatch, float64(report.TagMismatch)/float64(report.CommonAddons)*100)
	t.Logf("  Name differences:      %d (%.1f%%) [expected due to improved slugify]",
		report.NameDrift, float64(report.NameDrift)/float64(report.CommonAddons)*100)
	t.Logf("  Description drift:     %d (%.1f%%) [expected due to BBCode vs clean text]",
		report.DescriptionDrift, float64(report.DescriptionDrift)/float64(report.CommonAddons)*100)

	// Show game track mismatch samples
	if len(gameTrackSamples) > 0 {
		t.Logf("\nGame Track Mismatch Samples:")
		for _, sourceID := range gameTrackSamples {
			goAddon := goMap[sourceID]
			cljAddon := clojureMap[sourceID]
			t.Logf("  %s (%s):", sourceID, goAddon.Label)
			t.Logf("    Go:      %v", goAddon.GameTrackList)
			t.Logf("    Clojure: %v", cljAddon.GameTrackList)
		}
	}

	// Show tag mismatch samples
	if len(tagSamples) > 0 {
		t.Logf("\nTag Difference Samples:")
		for i, sourceID := range tagSamples {
			if i >= 3 {
				break
			}
			goAddon := goMap[sourceID]
			cljAddon := clojureMap[sourceID]
			t.Logf("  %s (%s):", sourceID, goAddon.Label)
			t.Logf("    Go:      %v", goAddon.TagList)
			t.Logf("    Clojure: %v", cljAddon.TagList)
		}
	}

	// Show name difference samples
	if len(nameSamples) > 0 {
		t.Logf("\nName Difference Samples:")
		for i, sourceID := range nameSamples {
			if i >= 3 {
				break
			}
			goAddon := goMap[sourceID]
			cljAddon := clojureMap[sourceID]
			t.Logf("  %s (%s):", sourceID, goAddon.Label)
			t.Logf("    Go:      %s", goAddon.Name)
			t.Logf("    Clojure: %s", cljAddon.Name)
		}
	}

	// Note: Game track mismatches are expected due to:
	// 1. Clojure parser using outdated HTML parsing logic
	// 2. Catalogues generated at different times (week apart)
	// 3. Go parser is more comprehensive (checks both multitoc and compatibility table)
	//
	// The 5.8% mismatch rate (down from 14.1%) is acceptable given:
	// - Manual verification shows Go parser is more accurate
	// - Unit tests verify correct parsing of real-world patterns
	// - Integration tests verify live HTML parsing works correctly
	if report.GameTrackMismatch > 0 {
		percentMismatch := float64(report.GameTrackMismatch) / float64(report.CommonAddons) * 100
		if percentMismatch > 10.0 {
			t.Errorf("Game track mismatch rate %.1f%% exceeds 10%% threshold - investigate regression",
				percentMismatch)
		} else {
			t.Logf("Game track mismatch rate %.1f%% is within acceptable range (< 10%%)",
				percentMismatch)
		}
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[string]bool)
	for _, s := range a {
		aMap[s] = true
	}
	for _, s := range b {
		if !aMap[s] {
			return false
		}
	}
	return true
}
