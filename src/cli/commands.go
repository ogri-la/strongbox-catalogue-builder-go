package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ogri-la/strongbox-catalogue-builder-go/src/catalogue"
	"github.com/ogri-la/strongbox-catalogue-builder-go/src/github"
	"github.com/ogri-la/strongbox-catalogue-builder-go/src/http"
	"github.com/ogri-la/strongbox-catalogue-builder-go/src/retry"
	"github.com/ogri-la/strongbox-catalogue-builder-go/src/types"
	"github.com/ogri-la/strongbox-catalogue-builder-go/src/validation"
	"github.com/ogri-la/strongbox-catalogue-builder-go/src/wowi"
)

// ScrapeConfig holds configuration for scraping
type ScrapeConfig struct {
	HTTPClient     http.HTTPClient
	Sources        []types.Source
	MaxWorkers     int
	WoWIAPIVersion wowi.APIVersion
}

// WriteConfig holds configuration for writing catalogues
type WriteConfig struct {
	Sources     []types.Source
	OutputFiles []string
}

// CommandHandler handles CLI commands
type CommandHandler struct {
	builder *catalogue.Builder
}

// NewCommandHandler creates a new command handler
func NewCommandHandler() *CommandHandler {
	return &CommandHandler{
		builder: catalogue.NewBuilder(),
	}
}

// Scrape executes the scrape command
func (h *CommandHandler) Scrape(ctx context.Context, config ScrapeConfig) error {
	slog.Info("starting scrape command", "sources", config.Sources)

	var allAddons []types.Addon
	var mu sync.Mutex

	// Process each source
	for _, source := range config.Sources {
		switch source {
		case types.WowInterfaceSource:
			addons, err := h.scrapeWowInterface(ctx, config.HTTPClient, config.MaxWorkers, config.WoWIAPIVersion)
			if err != nil {
				return fmt.Errorf("failed to scrape WowInterface: %w", err)
			}

			mu.Lock()
			allAddons = append(allAddons, addons...)
			mu.Unlock()

		case types.GitHubSource:
			addons, err := h.scrapeGitHub(ctx)
			if err != nil {
				return fmt.Errorf("failed to scrape GitHub: %w", err)
			}

			mu.Lock()
			allAddons = append(allAddons, addons...)
			mu.Unlock()

		default:
			slog.Warn("unsupported source", "source", source)
		}
	}

	// Build full catalogue with all sources
	fullCatalogue := h.builder.BuildCatalogue(allAddons, config.Sources)
	slog.Info("built catalogue", "total-addons", fullCatalogue.Total)

	// Create state directory
	stateDir := "state"
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Cutoff date for "short" catalogue: Dragonflight expansion (2022-11-28)
	cutoffDate := time.Date(2022, 11, 28, 0, 0, 0, 0, time.UTC)

	// Write source-specific catalogues
	for _, source := range config.Sources {
		sourceCatalogue := h.builder.FilterCatalogue(fullCatalogue, func(addon types.Addon) bool {
			return addon.Source == source
		})

		var filename string
		switch source {
		case types.WowInterfaceSource:
			filename = "wowinterface-catalogue.json"
		case types.GitHubSource:
			filename = "github-catalogue.json"
		default:
			continue
		}

		outputPath := filepath.Join(stateDir, filename)
		if err := h.writeCatalogue(sourceCatalogue, outputPath); err != nil {
			return err
		}
	}

	// Write full catalogue (all sources)
	fullPath := filepath.Join(stateDir, "full-catalogue.json")
	if err := h.writeCatalogue(fullCatalogue, fullPath); err != nil {
		return err
	}

	// Write short catalogue (maintained addons only)
	shortCatalogue := h.builder.ShortenCatalogue(fullCatalogue, cutoffDate)
	slog.Info("shortened catalogue", "original", fullCatalogue.Total, "maintained", shortCatalogue.Total, "cutoff", cutoffDate.Format("2006-01-02"))

	shortPath := filepath.Join(stateDir, "short-catalogue.json")
	if err := h.writeCatalogue(shortCatalogue, shortPath); err != nil {
		return err
	}

	return nil
}

// Write executes the write command (reads from state files)
func (h *CommandHandler) Write(ctx context.Context, config WriteConfig) error {
	slog.Info("starting write command", "sources", config.Sources)

	// For now, just create an empty catalogue since we don't have state file reading implemented
	// In a full implementation, this would read addon data from state files
	catalogue := h.builder.BuildCatalogue([]types.Addon{}, config.Sources)

	if len(config.OutputFiles) == 0 {
		return h.writeCatalogue(catalogue, "")
	}

	for _, outputFile := range config.OutputFiles {
		if err := h.writeCatalogue(catalogue, outputFile); err != nil {
			return err
		}
	}

	return nil
}

// scrapeWowInterface handles WowInterface-specific scraping logic
func (h *CommandHandler) scrapeWowInterface(ctx context.Context, client http.HTTPClient, maxWorkers int, apiVersion wowi.APIVersion) ([]types.Addon, error) {
	slog.Info("scraping WowInterface", "mode", "API + HTML detail pages", "api_version", apiVersion)

	parser := wowi.NewParser()

	// Track processed URLs and addon data
	processedURLs := make(map[string]bool)
	addonDataMap := make(map[string][]types.AddonData) // sourceID -> []AddonData

	var mu sync.Mutex
	var wg sync.WaitGroup
	var inFlight atomic.Int32 // Track URLs currently being processed

	// Create worker pool with larger buffer to handle API file list
	// v3 API has ~7971 addons, each generating 2 URLs = ~16k URLs
	urlChan := make(chan string, 20000)

	// Start periodic queue status logger
	stopLogger := make(chan bool)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				queueDepth := len(urlChan)
				processing := inFlight.Load()
				if queueDepth > 0 || processing > 0 {
					slog.Info("queue status", "pending_urls", queueDepth, "processing", processing, "workers", maxWorkers)
				}
			case <-stopLogger:
				return
			}
		}
	}()

	// Start workers
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for url := range urlChan {
				inFlight.Add(1)
				if err := h.processURL(ctx, client, parser, url, &mu, processedURLs, addonDataMap, urlChan); err != nil {
					slog.Error("failed to process URL", "url", url, "error", err)
				}
				inFlight.Add(-1)
			}
		}()
	}

	// Start with initial URL (API filelist only - HTML detail pages discovered from there)
	for _, url := range wowi.StartingURLs(apiVersion) {
		urlChan <- url
	}

	// Monitor queue and close when all work is done
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			<-ticker.C
			queueDepth := len(urlChan)
			processing := inFlight.Load()

			// We're done when queue is empty AND nothing is being processed
			if queueDepth == 0 && processing == 0 {
				slog.Info("all URLs processed, finishing scrape")
				close(urlChan)
				return
			}
		}
	}()

	wg.Wait()
	close(stopLogger)

	// Convert addon data to final addons
	var addons []types.Addon
	mu.Lock()
	for sourceID, dataList := range addonDataMap {
		if addon, err := h.builder.MergeAddonData(dataList); err == nil && addon != nil {
			addons = append(addons, *addon)
		} else if err != nil {
			slog.Error("failed to merge addon data", "source-id", sourceID, "error", err)
		}
	}
	mu.Unlock()

	slog.Info("completed WowInterface scraping", "addons", len(addons))
	return addons, nil
}

// scrapeGitHub handles GitHub-specific scraping logic
func (h *CommandHandler) scrapeGitHub(ctx context.Context) ([]types.Addon, error) {
	slog.Info("scraping GitHub catalogue")

	parser := github.NewParser()
	addons, err := parser.BuildCatalogue()
	if err != nil {
		return nil, fmt.Errorf("failed to build GitHub catalogue: %w", err)
	}

	slog.Info("completed GitHub scraping", "addons", len(addons))
	return addons, nil
}

// processURL processes a single URL and adds results to the data structures
func (h *CommandHandler) processURL(
	ctx context.Context,
	client http.HTTPClient,
	parser *wowi.Parser,
	url string,
	mu *sync.Mutex,
	processedURLs map[string]bool,
	addonDataMap map[string][]types.AddonData,
	urlChan chan<- string,
) error {
	// Check if already processed
	mu.Lock()
	if processedURLs[url] {
		mu.Unlock()
		return nil
	}
	processedURLs[url] = true
	mu.Unlock()

	slog.Debug("processing URL", "url", url)

	// Download content with retry logic
	retryConfig := retry.DefaultConfig()
	resp, err := retry.WithRetry(ctx, client, url, retryConfig)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("non-200 status code %d for %s", resp.StatusCode, url)
	}

	// Parse content
	result, err := parser.Parse(url, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", url, err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Add new URLs to process (both API and HTML detail pages)
	for _, newURL := range result.DownloadURLs {
		if !processedURLs[newURL] {
			// Block until we can send - we don't want to skip URLs
			urlChan <- newURL
		}
	}

	// Store addon data
	for _, addonData := range result.AddonData {
		if addonData.SourceID != "" {
			addonDataMap[addonData.SourceID] = append(addonDataMap[addonData.SourceID], addonData)
		}
	}

	return nil
}

// Validate executes the validate command
func (h *CommandHandler) Validate(ctx context.Context, cataloguePath string) error {
	slog.Info("validating catalogue", "file", cataloguePath)

	if err := validation.ValidateCatalogueFile(cataloguePath); err != nil {
		slog.Error("validation failed", "file", cataloguePath, "error", err)
		return err
	}

	slog.Info("validation successful", "file", cataloguePath)
	return nil
}

// writeCatalogue writes a catalogue to a file or stdout
func (h *CommandHandler) writeCatalogue(catalogue types.Catalogue, outputFile string) error {
	jsonData, err := json.MarshalIndent(catalogue, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal catalogue: %w", err)
	}

	if outputFile == "" {
		// Write to stdout
		fmt.Println(string(jsonData))
		return nil
	}

	// Write to file
	if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write catalogue to %s: %w", outputFile, err)
	}
	slog.Info("wrote catalogue", "file", outputFile, "addons", catalogue.Total)

	// Validate the catalogue after writing
	if err := validation.ValidateCatalogueFile(outputFile); err != nil {
		slog.Error("catalogue validation failed after write", "file", outputFile, "error", err)
		return fmt.Errorf("catalogue validation failed: %w", err)
	}
	slog.Info("catalogue validated", "file", outputFile)

	return nil
}
