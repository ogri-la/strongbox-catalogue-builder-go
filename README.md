# Strongbox Catalogue Builder (Go)

A Go translation of the Clojure-based strongbox-catalogue-builder, designed to scrape WowInterface.com and generate catalogues of World of Warcraft addons.

## Architecture

This project follows a modular, test-driven design with clear separation of concerns:

### Directory Structure

```
src/
├── types/          # Core types and data structures
├── http/           # HTTP client with mockable interface
├── cache/          # HTTP caching transport
├── wowi/           # WowInterface-specific parsing logic
├── catalogue/      # Catalogue generation and manipulation
└── cli/            # Command-line interface and handlers
```

### Key Design Principles

- **Pure Functions**: Most logic is implemented as pure functions for easy testing
- **Mockable Interfaces**: Side-effecting operations (HTTP, file I/O) use interfaces for testability
- **Modular Structure**: Clear separation between parsing, HTTP, caching, and CLI concerns
- **Testability**: Each module has comprehensive unit tests

## Features

- **WowInterface Scraping**: Parses HTML category pages, listing pages, and addon detail pages
- **API Integration**: Consumes WowInterface JSON API for addon metadata
- **HTTP Caching**: File-based caching to avoid redundant requests
- **Concurrent Processing**: Worker pool pattern for efficient scraping
- **Catalogue Generation**: Merges data from multiple sources into standardized format
- **CLI Interface**: Commands for scraping and writing catalogues

## Usage

### Build

```bash
go mod tidy
go build -o strongbox-catalogue-builder
```

### Run Tests

```bash
go test ./src/...
```

### Scrape and Generate Catalogue

```bash
# Scrape WowInterface and output to stdout
./strongbox-catalogue-builder scrape

# Scrape and save to file
./strongbox-catalogue-builder scrape --out catalogue.json

# Generate from existing state files
./strongbox-catalogue-builder write --out catalogue.json
```

### Command Options

```bash
# Show help
./strongbox-catalogue-builder --help
./strongbox-catalogue-builder scrape --help

# Set log level
./strongbox-catalogue-builder scrape --log-level debug

# Control concurrency
./strongbox-catalogue-builder scrape --workers 10
```

## Testing

The codebase emphasizes testability:

- **HTTP Client**: Uses interface with mock implementation for testing HTTP interactions
- **Pure Functions**: Parser functions are pure and easily unit testable
- **Builder Pattern**: Catalogue generation uses pure functions with predictable inputs/outputs
- **Dependency Injection**: External dependencies are injected for easy mocking

Example test structure:

```go
func TestParser_ParseAddonDetail(t *testing.T) {
    parser := NewParser()
    htmlContent := `<html>...</html>`

    result, err := parser.parseAddonDetail("https://example.com", []byte(htmlContent))

    // Assert results...
}
```

## Comparison to Original

This Go version maintains the core functionality of the Clojure original while adapting to Go idioms:

- **Queue-based Workers** → **Channel-based Worker Pools**
- **Multimethods** → **Interface-based Polymorphism**
- **Clojure Specs** → **Struct Types with Validation**
- **Leiningen** → **Go Modules**
- **Ring/HTTP** → **net/http with Custom Transport**

## Implementation Status

✅ Core types and interfaces
✅ HTTP client with caching
✅ WowInterface parsing (HTML & API)
✅ Catalogue generation and merging
✅ CLI interface and commands
✅ Unit tests for key functionality
🚧 State file persistence (simplified for now)
🚧 Complete error handling and retry logic
🚧 Full compatibility with original catalogue format

## Dependencies

- `github.com/PuerkitoBio/goquery` - HTML parsing (jQuery-like selectors)
- `github.com/spf13/pflag` - Command-line flag parsing
- `github.com/lmittmann/tint` - Structured logging with color
- `github.com/santhosh-tekuri/jsonschema/v5` - JSON schema validation (if needed)

## License

Follows the same license as the original Clojure project.