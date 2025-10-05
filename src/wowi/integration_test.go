//go:build integration

package wowi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	httpclient "github.com/ogri-la/strongbox-catalogue-builder-go/src/http"
	"github.com/ogri-la/strongbox-catalogue-builder-go/src/types"
)

func TestLiveWoWInterfaceData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create HTTP client without caching for integration test
	client := httpclient.NewRealHTTPClient(http.DefaultTransport, "strongbox-catalogue-builder 1.0.0-test (https://github.com/ogri-la/strongbox-catalogue-builder-go)")
	parser := NewParser()

	ctx := context.Background()

	t.Run("API File List", func(t *testing.T) {
		testAPIFileList(t, ctx, client, parser)
	})

	t.Run("Category Listing", func(t *testing.T) {
		testCategoryListing(t, ctx, client, parser)
	})

	t.Run("Addon Details", func(t *testing.T) {
		testAddonDetails(t, ctx, client, parser)
	})
}

func testAPIFileList(t *testing.T, ctx context.Context, client httpclient.HTTPClient, parser *Parser) {
	t.Log("Testing API file list parsing...")

	resp, err := client.Get(ctx, GetAPIFileList(APIVersionV4))
	if err != nil {
		t.Fatalf("Failed to fetch API file list: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("API returned status %d", resp.StatusCode)
	}

	result, err := parser.parseAPIFileList(resp.Body)
	if err != nil {
		t.Fatalf("Failed to parse API file list: %v", err)
	}

	if len(result.AddonData) == 0 {
		t.Fatal("No addons found in API file list")
	}

	t.Logf("Found %d addons in API file list", len(result.AddonData))

	// Validate first few addons
	for i, addon := range result.AddonData {
		if i >= 5 { // Just check first 5
			break
		}

		validateAddonData(t, addon, fmt.Sprintf("API addon %d", i))
	}

	// Check that download URLs were generated
	if len(result.DownloadURLs) == 0 {
		t.Error("No download URLs generated from API file list")
	}

	t.Logf("Generated %d download URLs", len(result.DownloadURLs))
}

func testCategoryListing(t *testing.T, ctx context.Context, client httpclient.HTTPClient, parser *Parser) {
	t.Log("Testing category listing parsing...")

	// Test a specific category page
	categoryURL := Host + "/downloads/index.php?cid=160&page=1" // Class & Role Specific

	resp, err := client.Get(ctx, categoryURL)
	if err != nil {
		t.Fatalf("Failed to fetch category listing: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("Category page returned status %d", resp.StatusCode)
	}

	result, err := parser.parseCategoryListing(categoryURL, resp.Body)
	if err != nil {
		t.Fatalf("Failed to parse category listing: %v", err)
	}

	if len(result.DownloadURLs) == 0 {
		t.Fatal("No addon URLs found in category listing")
	}

	t.Logf("Found %d addon URLs in category listing", len(result.DownloadURLs))

	// Validate URLs are properly formed
	for i, url := range result.DownloadURLs {
		if i >= 3 { // Just check first 3
			break
		}

		if !strings.HasPrefix(url, Host+"/downloads/info") {
			t.Logf("Non-addon URL found (may be pagination): %s", url)
		}
	}
}

func testAddonDetails(t *testing.T, ctx context.Context, client httpclient.HTTPClient, parser *Parser) {
	t.Log("Testing addon detail parsing...")

	// Test known popular addon IDs that should be stable
	testAddonIDs := []string{
		"8149",  // Broker Played Time
		"11551", // MapCoords
		"23145", // AdiBags
		"24939", // WeakAuras 2
		"20415", // BigWigs
		"21333", // TellMeWhen
		"19468", // Details! Damage Meter
		"11431", // ElvUI
		"5547",  // Deadly Boss Mods
		"4501",  // Bartender4
	}

	successCount := 0
	for i, addonID := range testAddonIDs {
		t.Run(fmt.Sprintf("Addon_%s", addonID), func(t *testing.T) {
			addonURL := fmt.Sprintf("%s/downloads/info%s", Host, addonID)

			resp, err := client.Get(ctx, addonURL)
			if err != nil {
				t.Logf("Failed to fetch addon %s: %v", addonID, err)
				return
			}

			if resp.StatusCode != 200 {
				t.Logf("Addon %s returned status %d", addonID, resp.StatusCode)
				return
			}

			result, err := parser.parseAddonDetail(addonURL, resp.Body)
			if err != nil {
				t.Errorf("Failed to parse addon %s: %v", addonID, err)
				return
			}

			if len(result.AddonData) == 0 {
				t.Logf("Addon %s returned no data (possibly removed)", addonID)
				return
			}

			addon := result.AddonData[0]
			validateAddonData(t, addon, fmt.Sprintf("addon %s", addonID))

			// Additional validation for detail pages
			if addon.SourceID != addonID {
				t.Errorf("Addon %s: SourceID mismatch, got %s", addonID, addon.SourceID)
			}

			// Should have some game tracks
			if len(addon.GameTrackSet) == 0 {
				t.Errorf("Addon %s: No game tracks detected", addonID)
			}

			t.Logf("Addon %s (%s): %d game tracks, %d releases",
				addon.Name, addon.Label, len(addon.GameTrackSet), len(addon.LatestReleaseSet))

			successCount++
		})

		if i >= 4 { // Test first 5 addons only to keep test reasonable
			break
		}
	}

	if successCount == 0 {
		t.Fatal("No addons were successfully parsed")
	}

	t.Logf("Successfully parsed %d out of 5 tested addons", successCount)
}

func validateAddonData(t *testing.T, addon types.AddonData, context string) {
	t.Helper()

	if addon.Source != types.WowInterfaceSource {
		t.Errorf("%s: Invalid source, got %s", context, addon.Source)
	}

	if addon.SourceID == "" {
		t.Errorf("%s: Missing SourceID", context)
	}

	if addon.Name == "" {
		t.Errorf("%s: Missing Name", context)
	}

	if addon.Label == "" {
		t.Errorf("%s: Missing Label", context)
	}

	// Check for common transcription issues
	if strings.Contains(addon.Name, " ") {
		t.Errorf("%s: Name contains spaces (should be slugified): %s", context, addon.Name)
	}

	if strings.Contains(addon.Name, "'") || strings.Contains(addon.Name, "\"") {
		t.Errorf("%s: Name contains quotes: %s", context, addon.Name)
	}

	// Check for encoding issues
	if strings.Contains(addon.Label, "�") {
		t.Errorf("%s: Label contains encoding errors: %s", context, addon.Label)
	}

	if strings.Contains(addon.Description, "�") {
		t.Errorf("%s: Description contains encoding errors", context)
	}

	// Validate game tracks are known values
	for track := range addon.GameTrackSet {
		switch track {
		case types.RetailTrack, types.ClassicTrack, types.ClassicTBCTrack,
			types.ClassicWotLKTrack, types.ClassicCataTrack, types.ClassicMistsTrack:
			// Valid tracks
		default:
			t.Errorf("%s: Unknown game track: %s", context, track)
		}
	}

	// Validate releases have proper structure
	for _, release := range addon.LatestReleaseSet {
		if release.GameTrack == "" {
			t.Errorf("%s: Release missing GameTrack", context)
		}

		if release.DownloadURL == "" {
			t.Errorf("%s: Release missing DownloadURL", context)
		}

		if release.Version == "" {
			t.Logf("%s: Release missing Version (may be normal for some releases)", context)
		}
	}
}
