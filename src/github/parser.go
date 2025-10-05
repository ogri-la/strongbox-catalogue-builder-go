package github

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/ogri-la/strongbox-catalogue-builder-go/src/types"
	"sort"
)

const (
	CatalogueURL = "https://raw.githubusercontent.com/ogri-la/github-wow-addon-catalogue-go/master/addons.csv"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

// BuildCatalogue downloads and parses the Github addon catalogue CSV
func (p *Parser) BuildCatalogue() ([]types.Addon, error) {
	resp, err := http.Get(CatalogueURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download catalogue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return p.ParseCSV(string(body))
}

// ParseCSV parses the CSV content and returns a list of addons
func (p *Parser) ParseCSV(csvContent string) ([]types.Addon, error) {
	reader := csv.NewReader(strings.NewReader(csvContent))

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Create header index map
	headerIndex := make(map[string]int)
	for i, col := range header {
		headerIndex[col] = i
	}

	var addons []types.Addon

	// Read rows
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row: %w", err)
		}

		addon, err := p.parseCSVRow(record, headerIndex)
		if err != nil {
			// Skip invalid rows but log them
			continue
		}

		addons = append(addons, addon)
	}

	return addons, nil
}

func (p *Parser) parseCSVRow(record []string, headerIndex map[string]int) (types.Addon, error) {
	getField := func(name string) string {
		if idx, ok := headerIndex[name]; ok && idx < len(record) {
			return strings.TrimSpace(record[idx])
		}
		return ""
	}

	name := getField("name")
	if name == "" {
		return types.Addon{}, fmt.Errorf("name is required")
	}

	fullName := getField("full_name")
	if fullName == "" {
		return types.Addon{}, fmt.Errorf("full_name is required")
	}

	url := getField("url")
	if url == "" {
		return types.Addon{}, fmt.Errorf("url is required")
	}

	description := getField("description")

	// Parse updated date
	var updatedDate time.Time
	lastUpdated := getField("last_updated")
	if lastUpdated != "" {
		var err error
		updatedDate, err = time.Parse(time.RFC3339, lastUpdated)
		if err != nil {
			return types.Addon{}, fmt.Errorf("failed to parse last_updated: %w", err)
		}
	}

	// Parse flavors (game tracks)
	// Initialize as empty slice (not nil) so it marshals to [] instead of null
	gameTrackList := []types.GameTrack{}
	flavors := getField("flavors")
	if flavors != "" {
		flavorList := strings.Split(flavors, ",")
		for _, flavor := range flavorList {
			flavor = strings.TrimSpace(flavor)
			if track := guessGameTrack(flavor); track != "" {
				gameTrackList = append(gameTrackList, track)
			}
		}
	}

	// Sort game tracks alphabetically for deterministic output
	// Empty lists are allowed - they indicate addons that need classification
	sort.Slice(gameTrackList, func(i, j int) bool {
		return string(gameTrackList[i]) < string(gameTrackList[j])
	})

	downloadCount := 0

	// Create slugified name - replace underscores with hyphens for consistency with Clojure version
	slugifiedName := strings.ReplaceAll(slug.Make(name), "_", "-")

	addon := types.Addon{
		CreatedDate:   nil,
		Description:   description,
		DownloadCount: &downloadCount,
		GameTrackList: gameTrackList,
		Label:         name,
		Name:          slugifiedName,
		Source:        "github",
		SourceID:      fullName,
		TagList:       []string{},
		URL:           url,
		UpdatedDate:   updatedDate,
	}

	return addon, nil
}

// guessGameTrack maps flavor names to game tracks
func guessGameTrack(flavor string) types.GameTrack {
	flavor = strings.ToLower(strings.TrimSpace(flavor))

	switch flavor {
	case "mainline", "retail":
		return types.RetailTrack
	case "classic", "vanilla":
		return types.ClassicTrack
	case "bcc", "tbc":
		return types.ClassicTBCTrack
	case "wrath", "wotlk":
		return types.ClassicWotLKTrack
	case "cata", "cataclysm":
		return types.ClassicCataTrack
	case "mists", "mop":
		return types.ClassicMistsTrack
	default:
		return ""
	}
}
