package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/lmittmann/tint"
	"github.com/ogri-la/strongbox-catalogue-builder-go/src/cache"
	"github.com/ogri-la/strongbox-catalogue-builder-go/src/cli"
	httpClient "github.com/ogri-la/strongbox-catalogue-builder-go/src/http"
)

var APP_VERSION = "unreleased"
var APP_LOC = "https://github.com/ogri-la/strongbox-catalogue-builder-go"

func main() {
	// Parse command line flags
	flags, err := cli.ParseFlags(os.Args, APP_VERSION)
	if err != nil {
		slog.Error("failed to parse flags", "error", err)
		os.Exit(1)
	}

	// Setup logging
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level: flags.LogLevel,
	})))

	// Get working directory
	cwd, err := os.Getwd()
	if err != nil {
		slog.Error("failed to get current working directory", "error", err)
		os.Exit(1)
	}

	// Setup cache
	cacheDir := filepath.Join(cwd, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		slog.Error("failed to create cache directory", "error", err)
		os.Exit(1)
	}

	cacheConfig := cache.CacheConfig{
		Directory:       cacheDir,
		DefaultTTLHours: 24,
		SearchTTLHours:  2,
	}

	// Setup HTTP client with caching
	cachingTransport := cache.NewFileCachingTransport(cacheConfig, http.DefaultTransport)
	userAgent := userAgent()
	client := httpClient.NewRealHTTPClient(cachingTransport, userAgent)

	// Create command handler
	handler := cli.NewCommandHandler()
	ctx := context.Background()

	// Execute command
	switch flags.SubCommand {
	case cli.ScrapeSubCommand:
		config := flags.ScrapeConfig
		config.HTTPClient = client

		if err := handler.Scrape(ctx, config); err != nil {
			slog.Error("scrape command failed", "error", err)
			os.Exit(1)
		}

	case cli.WriteSubCommand:
		if err := handler.Write(ctx, flags.WriteConfig); err != nil {
			slog.Error("write command failed", "error", err)
			os.Exit(1)
		}

	default:
		slog.Error("unknown subcommand", "subcommand", flags.SubCommand)
		os.Exit(1)
	}
}

func userAgent() string {
	return "strongbox-catalogue-builder-go/" + APP_VERSION + " (" + APP_LOC + ")"
}