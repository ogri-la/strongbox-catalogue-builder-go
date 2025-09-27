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

	// API file list
	if rawURL == APIFileList {
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
		addon.Description = strings.TrimSpace(s.Text())
	})

	// Extract game tracks from compatibility info
	doc.Find("#multitoc").Each(func(i int, s *goquery.Selection) {
		compatText := s.Text()
		tracks := parseGameTracks(compatText)
		addon.GameTrackSet = make(map[types.GameTrack]bool)
		for _, track := range tracks {
			addon.GameTrackSet[track] = true
		}
	})

	// Extract latest releases
	var releases []types.Release
	doc.Find(".infobox div#download a").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			release := types.Release{
				DownloadURL: Host + href,
			}

			// Try to determine game track from link text or title
			if title, exists := s.Attr("title"); exists {
				if track := parseGameTrackFromText(title); track != "" {
					release.GameTrack = track
				}
			}

			releases = append(releases, release)
		}
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

	var addonData []types.AddonData
	var urls []string

	for _, item := range apiData {
		addon := types.AddonData{
			Source:       types.WowInterfaceSource,
			Filename:     "api-filelist.json",
			GameTrackSet: make(map[types.GameTrack]bool),
			WoWI:         item, // Store raw API data
		}

		if id, ok := item["id"].(float64); ok {
			addon.SourceID = strconv.Itoa(int(id))
		}

		if title, ok := item["title"].(string); ok {
			addon.Label = title
			addon.Name = slugify(title)
		}

		if lastUpdate, ok := item["lastUpdate"].(float64); ok {
			updateTime := time.Unix(int64(lastUpdate), 0)
			addon.UpdatedDate = &updateTime
		}

		if gameVersions, ok := item["gameVersions"].([]interface{}); ok {
			for _, version := range gameVersions {
				if versionStr, ok := version.(string); ok {
					if track := gameVersionToGameTrack(versionStr); track != "" {
						addon.GameTrackSet[track] = true
					}
				}
			}
		}

		if addon.SourceID != "" {
			addonData = append(addonData, addon)
			// Add URLs for detail pages
			urls = append(urls, fmt.Sprintf("%s/downloads/info%s", Host, addon.SourceID))
			urls = append(urls, fmt.Sprintf("%s/filedetails/%s.json", APIHost, addon.SourceID))
		}
	}

	return &types.ParseResult{
		AddonData:    addonData,
		DownloadURLs: urls,
	}, nil
}

// parseAPIDetail parses WowInterface API addon detail
func (p *Parser) parseAPIDetail(content []byte) (*types.ParseResult, error) {
	var apiData []map[string]interface{}
	if err := json.Unmarshal(content, &apiData); err != nil {
		return nil, fmt.Errorf("failed to parse API JSON: %w", err)
	}

	if len(apiData) == 0 {
		return &types.ParseResult{}, nil
	}

	item := apiData[0] // API returns array but should only have one item
	addon := types.AddonData{
		Source:   types.WowInterfaceSource,
		Filename: "api-detail.json",
		WoWI:     item,
	}

	if id, ok := item["id"].(float64); ok {
		addon.SourceID = strconv.Itoa(int(id))
	}

	if title, ok := item["title"].(string); ok {
		addon.Label = title
		addon.Name = slugify(title)
	}

	return &types.ParseResult{
		AddonData: []types.AddonData{addon},
	}, nil
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
	return time.Parse("01-02-06 03:04 PM", dateStr)
}

func slugify(s string) string {
	// Simple slugify implementation
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return strings.ToLower(s)
}

func parseGameTracks(text string) []types.GameTrack {
	var tracks []types.GameTrack
	text = strings.ToLower(text)

	if strings.Contains(text, "retail") {
		tracks = append(tracks, types.RetailTrack)
	}
	if strings.Contains(text, "classic") && !strings.Contains(text, "tbc") && !strings.Contains(text, "wrath") {
		tracks = append(tracks, types.ClassicTrack)
	}
	if strings.Contains(text, "tbc") || strings.Contains(text, "burning crusade") {
		tracks = append(tracks, types.ClassicTBCTrack)
	}
	if strings.Contains(text, "wrath") || strings.Contains(text, "wotlk") {
		tracks = append(tracks, types.ClassicWotLKTrack)
	}
	if strings.Contains(text, "cata") {
		tracks = append(tracks, types.ClassicCataTrack)
	}
	if strings.Contains(text, "mists") {
		tracks = append(tracks, types.ClassicMistsTrack)
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