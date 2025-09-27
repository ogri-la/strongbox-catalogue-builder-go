package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/ogri-la/strongbox-catalogue-builder-go/src/catalogue"
	"github.com/ogri-la/strongbox-catalogue-builder-go/src/http"
	"github.com/ogri-la/strongbox-catalogue-builder-go/src/types"
	"github.com/ogri-la/strongbox-catalogue-builder-go/src/wowi"
)

// ScrapeConfig holds configuration for scraping
type ScrapeConfig struct {
	HTTPClient  http.HTTPClient
	OutputFiles []string
	Sources     []types.Source
	MaxWorkers  int
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
			addons, err := h.scrapeWowInterface(ctx, config.HTTPClient, config.MaxWorkers)
			if err != nil {
				return fmt.Errorf("failed to scrape WowInterface: %w", err)
			}

			mu.Lock()
			allAddons = append(allAddons, addons...)
			mu.Unlock()

		default:
			slog.Warn("unsupported source", "source", source)
		}
	}

	// Build catalogue
	catalogue := h.builder.BuildCatalogue(allAddons, config.Sources)
	slog.Info("built catalogue", "total-addons", catalogue.Total)

	// Write output
	return h.writeCatalogue(catalogue, config.OutputFiles)
}

// Write executes the write command (reads from state files)
func (h *CommandHandler) Write(ctx context.Context, config WriteConfig) error {
	slog.Info("starting write command", "sources", config.Sources)

	// For now, just create an empty catalogue since we don't have state file reading implemented
	// In a full implementation, this would read addon data from state files
	catalogue := h.builder.BuildCatalogue([]types.Addon{}, config.Sources)

	return h.writeCatalogue(catalogue, config.OutputFiles)
}

// scrapeWowInterface handles WowInterface-specific scraping logic
func (h *CommandHandler) scrapeWowInterface(ctx context.Context, client http.HTTPClient, maxWorkers int) ([]types.Addon, error) {
	slog.Info("scraping WowInterface")

	parser := wowi.NewParser()

	// Track processed URLs and addon data
	processedURLs := make(map[string]bool)
	addonDataMap := make(map[string][]types.AddonData) // sourceID -> []AddonData

	var mu sync.Mutex
	var wg sync.WaitGroup

	// Create worker pool
	urlChan := make(chan string, 100)

	// Start workers
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for url := range urlChan {
				if err := h.processURL(ctx, client, parser, url, &mu, processedURLs, addonDataMap, urlChan); err != nil {
					slog.Error("failed to process URL", "url", url, "error", err)
				}
			}
		}()
	}

	// Start with initial URLs
	for _, url := range wowi.StartingURLs() {
		urlChan <- url
	}

	// Close channel when done (this would need more sophisticated coordination in a real implementation)
	go func() {
		// In a real implementation, you'd wait for all URLs to be processed
		// This is simplified for the example
		time.Sleep(30 * time.Second) // Give enough time for initial scraping
		close(urlChan)
	}()

	wg.Wait()

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

	// Download content
	resp, err := client.Get(ctx, url)
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

	// Add new URLs to process
	for _, newURL := range result.DownloadURLs {
		if !processedURLs[newURL] {
			select {
			case urlChan <- newURL:
			default:
				// Channel full, skip this URL to avoid blocking
				slog.Warn("URL channel full, skipping URL", "url", newURL)
			}
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

// writeCatalogue writes the catalogue to the specified output files
func (h *CommandHandler) writeCatalogue(catalogue types.Catalogue, outputFiles []string) error {
	jsonData, err := json.MarshalIndent(catalogue, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal catalogue: %w", err)
	}

	if len(outputFiles) == 0 {
		// Write to stdout
		fmt.Println(string(jsonData))
		return nil
	}

	// Write to files
	for _, outputFile := range outputFiles {
		if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
			return fmt.Errorf("failed to write catalogue to %s: %w", outputFile, err)
		}
		slog.Info("wrote catalogue", "file", outputFile, "addons", catalogue.Total)
	}

	return nil
}