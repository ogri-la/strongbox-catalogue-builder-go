package wowi

const (
	Host = "https://www.wowinterface.com"

	// API v3 endpoints
	APIHostV3     = "https://api.mmoui.com/v3/game/WOW"
	APIFileListV3 = "https://api.mmoui.com/v3/game/WOW/filelist.json"

	// API v4 endpoints (default)
	APIHostV4     = "https://api.mmoui.com/v4/game/WOW"
	APIFileListV4 = "https://api.mmoui.com/v4/game/WOW/filelist.json"
)

// APIVersion represents the WowInterface API version
type APIVersion string

const (
	APIVersionV3 APIVersion = "v3"
	APIVersionV4 APIVersion = "v4"
)

// GetAPIHost returns the API host for the given version
func GetAPIHost(version APIVersion) string {
	if version == APIVersionV3 {
		return APIHostV3
	}
	return APIHostV4
}

// GetAPIFileList returns the file list URL for the given version
func GetAPIFileList(version APIVersion) string {
	if version == APIVersionV3 {
		return APIFileListV3
	}
	return APIFileListV4
}

// CategoryGroupPages - deprecated, no longer used for addon discovery
// Kept for URL classification only
var CategoryGroupPages = []string{}

// StartingURLs returns the initial URL to begin scraping
// Addons are discovered from the API filelist, then HTML detail pages are scraped for each
func StartingURLs(apiVersion APIVersion) []string {
	return []string{GetAPIFileList(apiVersion)}
}
