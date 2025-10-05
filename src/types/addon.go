package types

import "time"

// GameTrack represents WoW game versions
type GameTrack string

const (
	RetailTrack       GameTrack = "retail"
	ClassicTrack      GameTrack = "classic"
	ClassicTBCTrack   GameTrack = "classic-tbc"
	ClassicWotLKTrack GameTrack = "classic-wotlk"
	ClassicCataTrack  GameTrack = "classic-cata"
	ClassicMistsTrack GameTrack = "classic-mists"
)

var AllGameTracks = []GameTrack{
	RetailTrack, ClassicTrack, ClassicTBCTrack,
	ClassicWotLKTrack, ClassicCataTrack, ClassicMistsTrack,
}

// Source represents an addon source
type Source string

const (
	WowInterfaceSource Source = "wowinterface"
	GitHubSource       Source = "github"
)

// Addon represents a WoW addon
// Note: keep fields alphabetised for deterministic JSON output
type Addon struct {
	CreatedDate   *time.Time  `json:"created-date,omitempty"`
	Description   string      `json:"description,omitempty"`
	DownloadCount *int        `json:"download-count,omitempty"`
	GameTrackList []GameTrack `json:"game-track-list"`
	Label         string      `json:"label"`
	Name          string      `json:"name"`
	Source        Source      `json:"source"`
	SourceID      string      `json:"source-id"`
	TagList       []string    `json:"tag-list,omitempty"`
	URL           string      `json:"url"`
	UpdatedDate   time.Time   `json:"updated-date"`
}

// AddonData represents parsed addon data that may be incomplete
type AddonData struct {
	Source           Source                 `json:"source"`
	SourceID         string                 `json:"source-id"`
	Filename         string                 `json:"filename"`
	Name             string                 `json:"name,omitempty"`
	Label            string                 `json:"label,omitempty"`
	Description      string                 `json:"description,omitempty"`
	UpdatedDate      *time.Time             `json:"updated-date,omitempty"`
	CreatedDate      *time.Time             `json:"created-date,omitempty"`
	DownloadCount    *int                   `json:"download-count,omitempty"`
	GameTrackSet     map[GameTrack]bool     `json:"game-track-set,omitempty"`
	TagSet           map[string]bool        `json:"tag-set,omitempty"`
	URL              string                 `json:"url,omitempty"`
	LatestReleaseSet []Release              `json:"latest-release-set,omitempty"`
	WoWI             map[string]interface{} `json:"wowi,omitempty"` // WowInterface specific data
}

// Release represents a downloadable release
type Release struct {
	DownloadURL string    `json:"download-url"`
	Version     string    `json:"version,omitempty"`
	GameTrack   GameTrack `json:"game-track,omitempty"`
}

// Catalogue represents the output catalogue structure
type Catalogue struct {
	Spec struct {
		Version int `json:"version"`
	} `json:"spec"`
	Datestamp        string  `json:"datestamp"`
	Total            int     `json:"total"`
	AddonSummaryList []Addon `json:"addon-summary-list"`
}

// DownloadResult represents the result of downloading content
type DownloadResult struct {
	URL      string
	Response []byte
	Error    error
}

// ParseResult represents the result of parsing downloaded content
type ParseResult struct {
	AddonData    []AddonData `json:"addon-data,omitempty"`
	DownloadURLs []string    `json:"download-urls,omitempty"`
	Error        error       `json:"-"`
}
