package wowi

const (
	Host        = "https://www.wowinterface.com"
	APIHost     = "https://api.mmoui.com/v4/game/WOW"
	APIFileList = "https://api.mmoui.com/v4/game/WOW/filelist.json"
)

// CategoryGroupPages are the main category pages to start scraping from
var CategoryGroupPages = []string{
	"/downloads/index.php",
	"/addons.php",
	"/downloads/cat39.html",  // Class & Role Specific
	"/downloads/cat109.html", // Info, Plug-in Bars
	"/downloads/cat23.html",  // Stand-Alone addons
	"/downloads/cat28.html",  // Compilations
	"/downloads/cat158.html", // WoW Classic
	"/downloads/cat144.html", // Utilities
	"/downloads/cat145.html", // Optional
}

// StartingURLs returns the initial URLs to begin scraping
func StartingURLs() []string {
	urls := []string{APIFileList}
	for _, page := range CategoryGroupPages {
		if page == "/downloads/index.php" || page == "/addons.php" {
			urls = append(urls, Host+page)
		}
	}
	return urls
}