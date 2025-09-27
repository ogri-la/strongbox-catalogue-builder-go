package cli

import (
	"fmt"
	"log/slog"
	"os"
	"slices"

	"github.com/ogri-la/strongbox-catalogue-builder-go/src/types"
	flag "github.com/spf13/pflag"
)

// SubCommand represents CLI subcommands
type SubCommand string

const (
	ScrapeSubCommand SubCommand = "scrape"
	WriteSubCommand  SubCommand = "write"
)

var KnownSubCommands = []SubCommand{ScrapeSubCommand, WriteSubCommand}

// Flags holds all CLI flags and configuration
type Flags struct {
	SubCommand     SubCommand
	LogLevel       slog.Level
	ScrapeConfig   ScrapeConfig
	WriteConfig    WriteConfig
	ShowHelp       bool
	ShowVersion    bool
	MaxWorkers     int
}

// ParseFlags parses command line arguments and returns configuration
func ParseFlags(args []string, version string) (*Flags, error) {
	flags := &Flags{
		MaxWorkers: 5, // Default number of workers
	}

	// Global flags
	defaults := flag.NewFlagSet("strongbox-catalogue-builder", flag.ContinueOnError)
	defaults.BoolVarP(&flags.ShowHelp, "help", "h", false, "print this help and exit")
	defaults.BoolVarP(&flags.ShowVersion, "version", "V", false, "print program version and exit")

	var logLevelStr string
	defaults.StringVar(&logLevelStr, "log-level", "info", "verbosity level. one of: debug, info, warn, error")
	defaults.IntVar(&flags.MaxWorkers, "workers", 5, "number of concurrent workers")

	// Determine subcommand
	var subcommand string
	if len(args) > 1 {
		subcommand = args[1]
	}

	var flagset *flag.FlagSet
	scrapeConfig := ScrapeConfig{}
	writeConfig := WriteConfig{}

	switch subcommand {
	case string(ScrapeSubCommand):
		flagset = flag.NewFlagSet("scrape", flag.ExitOnError)
		flagset.StringArrayVar(&scrapeConfig.OutputFiles, "out", []string{}, "write results to file (default: stdout)")

		var sourcesStr []string
		flagset.StringArrayVar(&sourcesStr, "source", []string{"wowinterface"}, "sources to scrape")

		// Parse sources
		for _, sourceStr := range sourcesStr {
			switch sourceStr {
			case "wowinterface":
				scrapeConfig.Sources = append(scrapeConfig.Sources, types.WowInterfaceSource)
			case "github":
				scrapeConfig.Sources = append(scrapeConfig.Sources, types.GitHubSource)
			default:
				return nil, fmt.Errorf("unknown source: %s", sourceStr)
			}
		}

		flagset.AddFlagSet(defaults)

	case string(WriteSubCommand):
		flagset = flag.NewFlagSet("write", flag.ExitOnError)
		flagset.StringArrayVar(&writeConfig.OutputFiles, "out", []string{}, "write results to file (default: stdout)")

		var sourcesStr []string
		flagset.StringArrayVar(&sourcesStr, "source", []string{"wowinterface"}, "sources to include")

		// Parse sources
		for _, sourceStr := range sourcesStr {
			switch sourceStr {
			case "wowinterface":
				writeConfig.Sources = append(writeConfig.Sources, types.WowInterfaceSource)
			case "github":
				writeConfig.Sources = append(writeConfig.Sources, types.GitHubSource)
			default:
				return nil, fmt.Errorf("unknown source: %s", sourceStr)
			}
		}

		flagset.AddFlagSet(defaults)

	default:
		flagset = defaults
	}

	// Parse flags
	if err := flagset.Parse(args); err != nil {
		return nil, fmt.Errorf("failed to parse flags: %w", err)
	}

	// Handle help and version
	if flags.ShowHelp {
		printUsage(flagset)
		os.Exit(0)
	}

	if flags.ShowVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	// Validate subcommand
	if subcommand == "" || !slices.Contains(KnownSubCommands, SubCommand(subcommand)) {
		printUsage(flagset)
		return nil, fmt.Errorf("unknown subcommand: %s", subcommand)
	}

	// Parse log level
	logLevelMap := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}

	logLevel, exists := logLevelMap[logLevelStr]
	if !exists {
		return nil, fmt.Errorf("unknown log level: %s", logLevelStr)
	}

	// Assign parsed values
	flags.SubCommand = SubCommand(subcommand)
	flags.LogLevel = logLevel
	flags.ScrapeConfig = scrapeConfig
	flags.WriteConfig = writeConfig

	// Set max workers in configs
	flags.ScrapeConfig.MaxWorkers = flags.MaxWorkers

	return flags, nil
}

// printUsage prints usage information
func printUsage(flagset *flag.FlagSet) {
	fmt.Println("usage: strongbox-catalogue-builder <scrape|write> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  scrape    Scrape addon data from sources and generate catalogue")
	fmt.Println("  write     Generate catalogue from existing state files")
	fmt.Println()
	fmt.Println("Options:")
	flagset.PrintDefaults()
}