package wowi

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ogri-la/strongbox-catalogue-builder-go/src/types"
)

// URLClassifier determines the type of a WowInterface URL
type URLClassifier struct{}

// NewURLClassifier creates a new URL classifier
func NewURLClassifier() *URLClassifier {
	return &URLClassifier{}
}

// ClassifyURL determines what type of page a URL represents
func (c *URLClassifier) ClassifyURL(rawURL string) URLType {
	u, err := url.Parse(rawURL)
	if err != nil {
		return URLTypeUnknown
	}

	// API file list (matches both v3 and v4)
	if rawURL == APIFileListV3 || rawURL == APIFileListV4 {
		return URLTypeAPIFileList
	}

	// API addon detail
	if strings.Contains(u.Path, "/filedetails/") && strings.HasSuffix(u.Path, ".json") {
		return URLTypeAPIDetail
	}

	// Addon detail page
	if strings.Contains(u.Path, "/downloads/info") {
		return URLTypeAddonDetail
	}

	// Category group pages
	for _, page := range CategoryGroupPages {
		if strings.Contains(u.Path, page) && len(u.Query()) == 0 {
			return URLTypeCategoryGroup
		}
	}

	// Category listing pages (have pagination parameters)
	if strings.Contains(u.Query().Get("page"), "") && u.Query().Get("page") != "" {
		return URLTypeCategoryListing
	}

	return URLTypeUnknown
}

// URLType represents different types of WowInterface URLs
type URLType int

const (
	URLTypeUnknown URLType = iota
	URLTypeCategoryGroup
	URLTypeCategoryListing
	URLTypeAddonDetail
	URLTypeAPIFileList
	URLTypeAPIDetail
)

// Parser handles parsing of different WowInterface content types
type Parser struct {
	classifier *URLClassifier
}

// NewParser creates a new parser
func NewParser() *Parser {
	return &Parser{
		classifier: NewURLClassifier(),
	}
}

// Parse parses content based on URL type
func (p *Parser) Parse(rawURL string, content []byte) (*types.ParseResult, error) {
	urlType := p.classifier.ClassifyURL(rawURL)

	switch urlType {
	case URLTypeCategoryGroup:
		return p.parseCategoryGroup(content)
	case URLTypeCategoryListing:
		return p.parseCategoryListing(rawURL, content)
	case URLTypeAddonDetail:
		return p.parseAddonDetail(rawURL, content)
	case URLTypeAPIFileList:
		return p.parseAPIFileList(content)
	case URLTypeAPIDetail:
		return p.parseAPIDetail(content)
	default:
		return nil, fmt.Errorf("unknown URL type for: %s", rawURL)
	}
}

// parseCategoryGroup extracts category links from a category group page
func (p *Parser) parseCategoryGroup(content []byte) (*types.ParseResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(content)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var urls []string

	doc.Find("div#colleft div.subcats div.subtitle a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		// Check if this is another category group page
		isGroupPage := false
		for _, page := range CategoryGroupPages {
			if strings.Contains(href, page) {
				urls = append(urls, Host+href)
				isGroupPage = true
				break
			}
		}

		if !isGroupPage {
			// Convert to listing page URL with sorting
			if catID := extractCategoryID(href); catID != "" {
				listingURL := fmt.Sprintf("%s/downloads/index.php?cid=%s&sb=dec_date&so=desc&pt=f&page=1", Host, catID)
				urls = append(urls, listingURL)
			}
		}
	})

	return &types.ParseResult{
		DownloadURLs: urls,
	}, nil
}

// parseCategoryListing extracts addon data and pagination URLs from a listing page
func (p *Parser) parseCategoryListing(rawURL string, content []byte) (*types.ParseResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(content)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var addonData []types.AddonData
	var urls []string

	// Extract pagination URLs
	doc.Find(".pagenav td.alt1 a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && !strings.Contains(href, "http") {
			urls = append(urls, Host+href)
		}
	})

	// Extract addon information
	doc.Find("#filepage div.file").Each(func(i int, s *goquery.Selection) {
		addon := types.AddonData{
			Source:   types.WowInterfaceSource,
			Filename: "listing.json",
			WoWI:     make(map[string]interface{}),
		}

		// Extract title and source ID
		s.Find("a[href*='fileinfo']").Each(func(j int, link *goquery.Selection) {
			href, exists := link.Attr("href")
			if exists {
				if sourceID := extractSourceIDFromHref(href); sourceID != "" {
					addon.SourceID = sourceID
					addon.Label = strings.TrimSpace(link.Text())
					addon.Name = slugify(addon.Label)
					addon.URL = Host + "/downloads/info" + sourceID
					urls = append(urls, addon.URL) // Add detail page URL
				}
			}
		})

		// Extract updated date
		s.Find("div.updated").Each(func(j int, date *goquery.Selection) {
			if dateStr := extractUpdatedDate(date.Text()); dateStr != "" {
				if parsedDate, err := parseWoWIDate(dateStr); err == nil {
					addon.UpdatedDate = &parsedDate
				}
			}
		})

		// Extract download count
		s.Find("div.downloads").Each(func(j int, downloads *goquery.Selection) {
			if count := extractDownloadCount(downloads.Text()); count > 0 {
				addon.DownloadCount = &count
			}
		})

		if addon.SourceID != "" {
			addonData = append(addonData, addon)
		}
	})

	return &types.ParseResult{
		AddonData:    addonData,
		DownloadURLs: urls,
	}, nil
}

// parseAddonDetail extracts detailed addon information from an addon detail page
func (p *Parser) parseAddonDetail(rawURL string, content []byte) (*types.ParseResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(content)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Check if this is a removed/dead page
	pageText := doc.Text()
	if strings.Contains(pageText, "Removed per author's request") ||
		strings.Contains(pageText, "This file has been removed") ||
		strings.Contains(pageText, "File no longer available") {
		// Return empty result for removed addons - they should not be included in catalogue
		return &types.ParseResult{
			AddonData: []types.AddonData{},
		}, nil
	}

	addon := types.AddonData{
		Source:   types.WowInterfaceSource,
		Filename: "web-detail.json",
		URL:      rawURL,
		WoWI:     make(map[string]interface{}),
	}

	// Extract source ID from URL
	if sourceID := extractSourceIDFromURL(rawURL); sourceID != "" {
		addon.SourceID = sourceID
	} else {
		return nil, fmt.Errorf("could not extract source ID from URL: %s", rawURL)
	}

	// Extract title from meta tag
	doc.Find("meta[property='og:title']").Each(func(i int, s *goquery.Selection) {
		if title, exists := s.Attr("content"); exists {
			addon.Label = strings.TrimSpace(title)
			addon.Name = slugify(addon.Label)
		}
	})

	// Extract description
	doc.Find("div.postmessage").First().Each(func(i int, s *goquery.Selection) {
		addon.Description = cleanDescription(s.Text())
	})

	// Extract created date from info table
	doc.Find("td:contains('Created:')").Next().Each(func(i int, s *goquery.Selection) {
		dateStr := strings.TrimSpace(s.Text())
		if dateStr != "" {
			if parsedTime, err := parseWoWIDate(dateStr); err == nil {
				addon.CreatedDate = &parsedTime
			}
		}
	})

	// Extract categories first - we'll use them for game track inference and tags
	categorySet := make(map[string]bool)

	// Look for categories in the info table
	doc.Find("td:contains('Categories:')").Next().Each(func(i int, s *goquery.Selection) {
		s.Find("a").Each(func(j int, link *goquery.Selection) {
			category := strings.TrimSpace(link.Text())
			if category != "" {
				categorySet[category] = true
			}
		})
	})

	// Also check selected dropdown options as fallback
	doc.Find("select option[selected]").Each(func(i int, s *goquery.Selection) {
		category := strings.TrimSpace(s.Text())
		if category != "" && !strings.HasPrefix(category, "Choose") {
			// Clean up category text (remove leading dashes and spaces)
			category = strings.TrimLeft(category, "- ")
			categorySet[category] = true
		}
	})

	// Convert categories to tags (like Clojure version does)
	// Use replacement/supplement maps first, then split if no replacement
	addon.TagSet = make(map[string]bool)
	for category := range categorySet {
		tags := categoryToTagsWithMaps(category)
		for _, tag := range tags {
			if tag != "" {
				addon.TagSet[tag] = true
			}
		}
	}

	// Extract game tracks from compatibility info
	addon.GameTrackSet = make(map[types.GameTrack]bool)

	// Check #multitoc element for basic compatibility
	doc.Find("#multitoc").Each(func(i int, s *goquery.Selection) {
		compatText := s.Text()
		tracks := parseGameTracks(compatText)
		for _, track := range tracks {
			addon.GameTrackSet[track] = true
		}
	})

	// Also check detailed compatibility table
	doc.Find("td:contains('Compatibility:')").Next().Each(func(i int, s *goquery.Selection) {
		s.Find("div").Each(func(j int, div *goquery.Selection) {
			compatText := div.Text()
			tracks := parseGameTracks(compatText)
			for _, track := range tracks {
				addon.GameTrackSet[track] = true
			}
		})
	})

	// NOTE: We do NOT infer game tracks from categories because:
	// 1. Categories like "Classic - General" appear in dropdowns for ALL addons
	// 2. Only the explicit Compatibility field indicates actual game version support
	// 3. Inferring from categories causes false positives (retail addons marked as classic)

	// Extract latest releases and detect game tracks from download sections
	var releases []types.Release

	// Count download buttons to determine if this is a multi-version addon
	downloadButtonCount := doc.Find(".infobox div#downloadbutton").Length()
	isMultiVersion := downloadButtonCount > 1

	// Find all download sections - each has an #iconnew (or #icon) div followed by a #download div
	// The #iconnew div has a class that indicates the game version (tbc, wotlk, cata, etc.)
	// Try both #iconnew (multi-version addons) and #icon (simple addons)
	iconSelector := ".infobox div#iconnew, .infobox div#icon"
	doc.Find(iconSelector).Each(func(i int, iconDiv *goquery.Selection) {
		// Get the game track from the icon div class
		var gameTrack types.GameTrack
		if classAttr, exists := iconDiv.Attr("class"); exists {
			switch {
			case strings.Contains(classAttr, "cata"):
				gameTrack = types.ClassicCataTrack
			case strings.Contains(classAttr, "mists"):
				gameTrack = types.ClassicMistsTrack
			case strings.Contains(classAttr, "wotlk"):
				gameTrack = types.ClassicWotLKTrack
			case strings.Contains(classAttr, "tbc"):
				gameTrack = types.ClassicTBCTrack
			}
		}

		// For multi-version addons, we can trust the download link title
		// because each version has its own download button with accurate labels
		if isMultiVersion && gameTrack == "" {
			iconDiv.Find("a").Each(func(j int, a *goquery.Selection) {
				if title, exists := a.Attr("title"); exists {
					titleLower := strings.ToLower(title)
					if strings.Contains(titleLower, "wow classic") && !strings.Contains(titleLower, "burning crusade") &&
						!strings.Contains(titleLower, "wrath") && !strings.Contains(titleLower, "cataclysm") {
						gameTrack = types.ClassicTrack
					} else if strings.Contains(titleLower, "wow retail") {
						gameTrack = types.RetailTrack
					}
				}
			})
		}
		// Note: For single-version addons, we do NOT use the title because it's unreliable.
		// WoWInterface often shows "WoW Retail" even for classic-only addons.

		// Find the adjacent download div to get the actual download link
		downloadDiv := iconDiv.NextAll().Filter("#download").First()
		downloadDiv.Find("a").Each(func(j int, a *goquery.Selection) {
			if href, exists := a.Attr("href"); exists && strings.Contains(href, "downloads") {
				// Add game track to addon's supported tracks
				if gameTrack != "" {
					addon.GameTrackSet[gameTrack] = true
				}

				release := types.Release{
					DownloadURL: Host + href,
					GameTrack:   gameTrack,
				}
				releases = append(releases, release)
			}
		})
	})

	addon.LatestReleaseSet = releases

	// Default to retail if no game tracks found
	if len(addon.GameTrackSet) == 0 {
		addon.GameTrackSet = map[types.GameTrack]bool{types.RetailTrack: true}
	}

	return &types.ParseResult{
		AddonData: []types.AddonData{addon},
	}, nil
}

// parseAPIFileList parses the WowInterface API file list
func (p *Parser) parseAPIFileList(content []byte) (*types.ParseResult, error) {
	var apiData []map[string]interface{}
	if err := json.Unmarshal(content, &apiData); err != nil {
		return nil, fmt.Errorf("failed to parse API JSON: %w", err)
	}

	if len(apiData) == 0 {
		return &types.ParseResult{}, nil
	}

	// Detect API version by checking field names in first item
	isV3 := false
	if _, hasUID := apiData[0]["UID"]; hasUID {
		isV3 = true
	}

	var addonData []types.AddonData
	var urls []string
	apiHost := GetAPIHost(APIVersionV4)
	if isV3 {
		apiHost = GetAPIHost(APIVersionV3)
	}

	for _, item := range apiData {
		var addon types.AddonData
		if isV3 {
			addon = parseAPIFileListItemV3(item)
		} else {
			addon = parseAPIFileListItemV4(item)
		}

		if addon.SourceID != "" {
			addonData = append(addonData, addon)
			// Add URLs for detail pages
			urls = append(urls, fmt.Sprintf("%s/downloads/info%s", Host, addon.SourceID))
			urls = append(urls, fmt.Sprintf("%s/filedetails/%s.json", apiHost, addon.SourceID))
		}
	}

	return &types.ParseResult{
		AddonData:    addonData,
		DownloadURLs: urls,
	}, nil
}

// parseAPIFileListItemV3 parses a v3 API file list item
// v3 fields: UID, UIName, UIAuthorName, UIDate, UICATID, UICompatibility (array of objects), UIDir (addon folders), etc.
func parseAPIFileListItemV3(item map[string]interface{}) types.AddonData {
	addon := types.AddonData{
		Source:       types.WowInterfaceSource,
		Filename:     "api-filelist-v3.json",
		GameTrackSet: make(map[types.GameTrack]bool),
		WoWI:         item,
	}

	// UID -> SourceID
	if uid, ok := item["UID"].(string); ok {
		addon.SourceID = uid
	}

	// UIName -> Label
	if name, ok := item["UIName"].(string); ok {
		addon.Label = name
		addon.Name = slugify(name)
	}

	// UIDate -> UpdatedDate
	if date, ok := item["UIDate"].(float64); ok {
		updateTime := time.Unix(int64(date)/1000, 0).UTC()
		addon.UpdatedDate = &updateTime
	}

	// UICompatibility -> GameTrackSet (v3 has array of {version, name} objects)
	if compat, ok := item["UICompatibility"].([]interface{}); ok {
		for _, c := range compat {
			if compatObj, ok := c.(map[string]interface{}); ok {
				if version, ok := compatObj["version"].(string); ok {
					if track := gameVersionToGameTrack(version); track != "" {
						addon.GameTrackSet[track] = true
					}
				}
			}
		}
	}

	// UIDir is available in v3 (addon folder names) - store in WoWI data

	return addon
}

// parseAPIFileListItemV4 parses a v4 API file list item
// v4 fields: id, title, author, lastUpdate, categoryId, gameVersions (array of strings), checksum, etc.
func parseAPIFileListItemV4(item map[string]interface{}) types.AddonData {
	addon := types.AddonData{
		Source:       types.WowInterfaceSource,
		Filename:     "api-filelist-v4.json",
		GameTrackSet: make(map[types.GameTrack]bool),
		WoWI:         item,
	}

	// id -> SourceID
	if id, ok := item["id"].(float64); ok {
		addon.SourceID = strconv.Itoa(int(id))
	}

	// title -> Label
	if title, ok := item["title"].(string); ok {
		addon.Label = title
		addon.Name = slugify(title)
	}

	// lastUpdate -> UpdatedDate
	if lastUpdate, ok := item["lastUpdate"].(float64); ok {
		updateTime := time.Unix(int64(lastUpdate)/1000, 0).UTC()
		addon.UpdatedDate = &updateTime
	}

	// gameVersions -> GameTrackSet (v4 has simple string array)
	if gameVersions, ok := item["gameVersions"].([]interface{}); ok {
		for _, version := range gameVersions {
			if versionStr, ok := version.(string); ok {
				if track := gameVersionToGameTrack(versionStr); track != "" {
					addon.GameTrackSet[track] = true
				}
			}
		}
	}

	return addon
}

// parseAPIDetail parses WowInterface API addon detail (supports both v3 and v4)
func (p *Parser) parseAPIDetail(content []byte) (*types.ParseResult, error) {
	var apiData []map[string]interface{}
	if err := json.Unmarshal(content, &apiData); err != nil {
		return nil, fmt.Errorf("failed to parse API JSON: %w", err)
	}

	if len(apiData) == 0 {
		return &types.ParseResult{}, nil
	}

	item := apiData[0] // API returns array but should only have one item

	// Detect API version
	isV3 := false
	if _, hasUID := item["UID"]; hasUID {
		isV3 = true
	}

	var addon types.AddonData
	if isV3 {
		addon = parseAPIDetailItemV3(item)
	} else {
		addon = parseAPIDetailItemV4(item)
	}

	return &types.ParseResult{
		AddonData: []types.AddonData{addon},
	}, nil
}

// parseAPIDetailItemV3 parses a v3 API detail item
// v3 detail fields: UID, UIName, UIMD5, UIFileName, UIDownload, UIDescription, UIChangeLog, etc.
func parseAPIDetailItemV3(item map[string]interface{}) types.AddonData {
	addon := types.AddonData{
		Source:   types.WowInterfaceSource,
		Filename: "api-detail-v3.json",
		WoWI:     item,
	}

	// UID -> SourceID
	if uid, ok := item["UID"].(string); ok {
		addon.SourceID = uid
	}

	// UIName -> Label
	if name, ok := item["UIName"].(string); ok {
		addon.Label = name
		addon.Name = slugify(name)
	}

	return addon
}

// parseAPIDetailItemV4 parses a v4 API detail item
// v4 detail fields: id, title, checksum, fileName, downloadUri, description, changeLog, images, etc.
func parseAPIDetailItemV4(item map[string]interface{}) types.AddonData {
	addon := types.AddonData{
		Source:       types.WowInterfaceSource,
		Filename:     "api-detail-v4.json",
		GameTrackSet: make(map[types.GameTrack]bool),
		TagSet:       make(map[string]bool),
		WoWI:         item,
	}

	// id -> SourceID
	if id, ok := item["id"].(float64); ok {
		idStr := strconv.Itoa(int(id))
		addon.SourceID = idStr
		addon.URL = fmt.Sprintf("https://www.wowinterface.com/downloads/info%s", idStr)
	}

	// title -> Label
	if title, ok := item["title"].(string); ok {
		addon.Label = title
		addon.Name = slugify(title)
	}

	// description
	if desc, ok := item["description"].(string); ok {
		addon.Description = cleanDescription(desc)
	}

	// downloads -> DownloadCount
	if downloads, ok := item["downloads"].(float64); ok {
		count := int(downloads)
		addon.DownloadCount = &count
	}

	// lastUpdate (milliseconds since epoch) -> UpdatedDate
	if lastUpdate, ok := item["lastUpdate"].(float64); ok {
		timestamp := time.Unix(0, int64(lastUpdate)*int64(time.Millisecond)).UTC()
		addon.UpdatedDate = &timestamp
	}

	// categoryId -> tags (map category IDs to tag names)
	if categoryID, ok := item["categoryId"].(float64); ok {
		// You'd need to map category IDs to tag names
		// For now, we'll just skip this as it requires category mapping
		_ = categoryID
	}

	return addon
}

// Utility functions for parsing

var sourceIDRegex = regexp.MustCompile(`id=(\d+)`)
var sourceIDFromURLRegex = regexp.MustCompile(`info(\d+)`)
var categoryIDRegex = regexp.MustCompile(`\d+`)
var downloadCountRegex = regexp.MustCompile(`\d+`)

func extractSourceIDFromHref(href string) string {
	matches := sourceIDRegex.FindStringSubmatch(href)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func extractSourceIDFromURL(url string) string {
	matches := sourceIDFromURLRegex.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func extractCategoryID(href string) string {
	return categoryIDRegex.FindString(href)
}

func extractUpdatedDate(text string) string {
	if strings.HasPrefix(text, "Updated ") {
		return strings.TrimSpace(strings.TrimPrefix(text, "Updated "))
	}
	return ""
}

func extractDownloadCount(text string) int {
	countStr := downloadCountRegex.FindString(text)
	if count, err := strconv.Atoi(countStr); err == nil {
		return count
	}
	return 0
}

func parseWoWIDate(dateStr string) (time.Time, error) {
	// WowInterface uses format: "09-07-18 01:27 PM"
	t, err := time.Parse("01-02-06 03:04 PM", dateStr)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

func slugify(s string) string {
	// Create a clean, readable slug suitable for identifying addons
	// 1. Lowercase
	// 2. Split on any non-alphanumeric characters (spaces, punctuation, symbols)
	// 3. Filter out empty parts
	// 4. Join with hyphens
	// 5. Trim to 250 characters

	// Lowercase
	s = strings.ToLower(s)

	// Split on any non-alphanumeric character (keeps only letters and numbers)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	parts := re.Split(s, -1)

	// Filter out empty parts
	var filtered []string
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}

	// Join with hyphen
	result := strings.Join(filtered, "-")

	// Trim to 250 characters
	if len(result) > 250 {
		result = result[:250]
	}

	return result
}

func parseGameTracks(text string) []types.GameTrack {
	var tracks []types.GameTrack
	text = strings.ToLower(text)

	// Look for retail
	if strings.Contains(text, "retail") || strings.Contains(text, "wow retail") ||
		strings.Contains(text, "shadowlands") || strings.Contains(text, "dragonflight") ||
		strings.Contains(text, "plunderstorm") || strings.Contains(text, "10.") ||
		strings.Contains(text, "9.") || strings.Contains(text, "8.") {
		tracks = append(tracks, types.RetailTrack)
	}

	// Look for classic variants (order matters - check specific first, then generic)
	if strings.Contains(text, "mists") {
		tracks = append(tracks, types.ClassicMistsTrack)
	}
	if strings.Contains(text, "cata") {
		tracks = append(tracks, types.ClassicCataTrack)
	}
	if strings.Contains(text, "wrath") || strings.Contains(text, "wotlk") || strings.Contains(text, "lich king") || strings.Contains(text, "3.4.") {
		tracks = append(tracks, types.ClassicWotLKTrack)
	}
	if strings.Contains(text, "tbc") || strings.Contains(text, "burning crusade") || strings.Contains(text, "2.5.") {
		tracks = append(tracks, types.ClassicTBCTrack)
	}

	// Classic (vanilla) - ONLY add if "classic" appears without expansion modifiers
	// "The Burning Crusade Classic" should NOT add vanilla classic
	// "Classic (1.13.2)" SHOULD add vanilla classic
	if strings.Contains(text, "classic") {
		// Check for standalone classic (no expansion keywords adjacent to it)
		// Patterns like "tbc classic" or "burning crusade classic" should NOT add vanilla
		hasExpansionModifier := strings.Contains(text, "tbc classic") ||
			strings.Contains(text, "wrath classic") ||
			strings.Contains(text, "wotlk classic") ||
			strings.Contains(text, "cata classic") ||
			strings.Contains(text, "burning crusade classic") ||
			strings.Contains(text, "lich king classic") ||
			strings.Contains(text, "cataclysm classic") ||
			strings.Contains(text, "mists classic")

		// Only add vanilla classic if there's no expansion modifier
		if !hasExpansionModifier {
			// Also check it's not just an expansion mention with "classic" in the name
			if !strings.Contains(text, "tbc") && !strings.Contains(text, "wrath") &&
				!strings.Contains(text, "wotlk") && !strings.Contains(text, "cata") &&
				!strings.Contains(text, "mists") {
				tracks = append(tracks, types.ClassicTrack)
			} else if strings.Contains(text, "& classic") || strings.Contains(text, ", classic") ||
				strings.Contains(text, "classic &") || strings.Contains(text, "classic,") {
				// Patterns like "retail & classic" or "tbc, classic" mean vanilla IS included
				tracks = append(tracks, types.ClassicTrack)
			}
		}
	}

	// Handle "Compatible with Retail, Classic & TBC" pattern specifically
	if strings.Contains(text, "retail") && strings.Contains(text, "classic") && strings.Contains(text, "tbc") {
		// This pattern typically means all three: retail, classic (vanilla), and tbc
		found := make(map[types.GameTrack]bool)
		for _, track := range tracks {
			found[track] = true
		}
		if !found[types.ClassicTrack] {
			tracks = append(tracks, types.ClassicTrack)
		}
	}

	return tracks
}

func parseGameTrackFromText(text string) types.GameTrack {
	tracks := parseGameTracks(text)
	if len(tracks) > 0 {
		return tracks[0]
	}
	return ""
}

func parseGameTracksFromCategory(category string) []types.GameTrack {
	var tracks []types.GameTrack
	categoryLower := strings.ToLower(category)

	// Direct category name mappings based on WowInterface categories
	switch {
	case strings.Contains(categoryLower, "the burning crusade classic"):
		tracks = append(tracks, types.ClassicTBCTrack)
	case strings.Contains(categoryLower, "wotlk classic"):
		tracks = append(tracks, types.ClassicWotLKTrack)
	case strings.Contains(categoryLower, "cataclysm classic"):
		tracks = append(tracks, types.ClassicCataTrack)
	case strings.Contains(categoryLower, "classic - general"):
		// Classic general usually means vanilla + other classics
		tracks = append(tracks, types.ClassicTrack)
	case strings.Contains(categoryLower, "addons for wow classic"):
		tracks = append(tracks, types.ClassicTrack)
	}

	return tracks
}

func gameVersionToGameTrack(version string) types.GameTrack {
	if len(version) < 2 {
		return types.RetailTrack
	}

	prefix := version[:2]
	switch prefix {
	case "1.":
		return types.ClassicTrack
	case "2.":
		return types.ClassicTBCTrack
	case "3.":
		return types.ClassicWotLKTrack
	case "4.":
		return types.ClassicCataTrack
	case "5.":
		return types.ClassicMistsTrack
	default:
		return types.RetailTrack
	}
}

// categoryToTags converts a WowInterface category string to one or more tags
// Following the Clojure implementation:
// 1. Split on " & ", ", ", or ": " to handle compound categories
// 2. For each part: lowercase, trim, and replace spaces with hyphens
func categoryToTags(category string) []string {
	if category == "" {
		return nil
	}

	// Split on " & ", ", ", or ": " (matching Clojure regex: ( & |, |: )+)
	var parts []string
	current := category

	// Replace separators with a unique delimiter, then split
	current = strings.ReplaceAll(current, " & ", "|||")
	current = strings.ReplaceAll(current, ", ", "|||")
	current = strings.ReplaceAll(current, ": ", "|||")
	parts = strings.Split(current, "|||")

	// Convert each part to a tag
	var tags []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// Lowercase and replace spaces with hyphens
		tag := strings.ToLower(part)
		tag = strings.ReplaceAll(tag, " ", "-")
		tags = append(tags, tag)
	}

	return tags
}

// cleanDescription processes description text to extract a meaningful first line.
// Matches Clojure implementation: splits into lines, removes decorative lines,
// skips common leading header words, returns first high-quality line.
// Falls back to first non-decorative line if no high-quality line found.
func cleanDescription(text string) string {
	if text == "" {
		return ""
	}

	// Split into lines
	lines := strings.Split(text, "\n")

	// First pass: find a high-quality description line
	var fallback string
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip decorative lines (matches Clojure's pure-non-alpha-numeric?)
		if isPureNonAlphanumeric(line) {
			continue
		}

		// Skip common leading header words that add no value
		if shouldSkipLeadingLine(line) {
			continue
		}

		// Remember first non-decorative line as fallback
		if fallback == "" {
			fallback = line
		}

		// Skip low-quality descriptions (version numbers, single words, etc.)
		if isLowQualityDescription(line) {
			continue
		}

		// Found a good quality line - limit to reasonable length
		const maxLength = 1000
		if len(line) > maxLength {
			return line[:maxLength]
		}
		return line
	}

	// No high-quality line found, use fallback (something is better than nothing)
	// BUT: don't use fallback if it's a known junk word
	if fallback != "" {
		fallbackLower := strings.ToLower(fallback)
		// Don't return known junk as description
		junkWords := []string{"null", "undefined", "n/a", "none", "unknown"}
		isJunk := false
		for _, junk := range junkWords {
			if fallbackLower == junk {
				isJunk = true
				break
			}
		}

		if !isJunk {
			const maxLength = 1000
			if len(fallback) > maxLength {
				return fallback[:maxLength]
			}
			return fallback
		}
	}

	return ""
}

// isLowQualityDescription returns true if the description is too short,
// contains only version numbers, dates, or other non-descriptive content.
func isLowQualityDescription(s string) bool {
	// Minimum length threshold
	if len(s) < 15 {
		return true
	}

	// Must contain at least one space (multiple words)
	if !strings.Contains(s, " ") {
		return true
	}

	lower := strings.ToLower(s)

	// Exact match low-quality words (these should never be returned in Go)
	exactBadWords := []string{
		"null", "undefined", "n/a", "none", "unknown",
	}
	for _, word := range exactBadWords {
		if lower == word {
			return true
		}
	}

	// Check for "AddonName by AuthorName" pattern (common lazy description)
	// e.g., "BigWigs by Funkydude"
	if strings.Contains(lower, " by ") {
		// Typically 3 words: "Name by Author"
		words := strings.Fields(s)
		if len(words) == 3 {
			return true
		}
	}

	// Prefix-based low-quality patterns
	lowQualityPrefixes := []string{
		"update:", "updated:", "new:", "news:",
	}
	for _, pattern := range lowQualityPrefixes {
		if strings.HasPrefix(lower, pattern) {
			return true
		}
	}

	// Check if it starts with version number patterns
	// e.g., "1.0", "10.1.5 UPDATE:", "0.8.2", "v1.2.3"
	if len(s) > 0 && (s[0] >= '0' && s[0] <= '9' || s[0] == 'v' || s[0] == 'V') {
		// Simple version pattern: starts with digit or v, contains dots
		if strings.Contains(s[:min(10, len(s))], ".") {
			return true
		}
	}

	// Check for date patterns: MM/DD/YYYY or YYYY-MM-DD
	if len(s) >= 10 {
		prefix := s[:10]
		// MM/DD/YYYY
		if len(prefix) == 10 && prefix[2] == '/' && prefix[5] == '/' {
			return true
		}
		// YYYY-MM-DD
		if len(prefix) == 10 && prefix[4] == '-' && prefix[7] == '-' {
			return true
		}
	}

	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// isPureNonAlphanumeric returns true if string contains only non-alphanumeric characters.
// Matches Clojure's pure-non-alpha-numeric? function with regex ^[\W_]*$
func isPureNonAlphanumeric(s string) bool {
	if s == "" {
		return true
	}
	// \W matches non-word characters (opposite of \w which is [a-zA-Z0-9_])
	// So [\W_] matches anything that's not alphanumeric
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

// shouldSkipLeadingLine returns true if the line starts with common header words
// that add no value (user's TODO list of words to filter).
func shouldSkipLeadingLine(line string) bool {
	lower := strings.ToLower(line)

	// List of prefixes to skip (from user's TODO)
	skipPrefixes := []string{
		// Heading words
		"about", "description", "general description", "general", "what", "info",
		"information", "credits", "features", "intro", "introduction", "note",
		"overview", "preamble", "purpose", "synopsis", "summary",

		// Donation/support
		"donate", "donation", "paypal", "support", "patreon",

		// Meta/status words
		"discontinued", "important", "news", "update", "updated", "urgent", "warning",

		// Locale
		"english", "engb", "enus",

		// Greetings
		"hello", "hey", "hi",

		// Special phrases
		"special thanks", "special note",
		"what is it", "what does it do", "what is", "what it is", "what's this",
	}

	for _, prefix := range skipPrefixes {
		// Check if line starts with prefix (possibly followed by punctuation/whitespace)
		if strings.HasPrefix(lower, prefix) {
			// Make sure it's actually a prefix, not part of a word
			// e.g., "about this addon" should match, "aboutface" should not
			if len(line) == len(prefix) {
				return true
			}
			nextChar := lower[len(prefix)]
			// Allow any non-alphanumeric character after prefix
			if !((nextChar >= 'a' && nextChar <= 'z') || (nextChar >= '0' && nextChar <= '9')) {
				return true
			}
		}
	}

	return false
}
