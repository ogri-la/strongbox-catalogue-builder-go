package cli

import (
	"fmt"
	"log/slog"
	"os"
	"slices"

	"github.com/ogri-la/strongbox-catalogue-builder-go/src/types"
	"github.com/ogri-la/strongbox-catalogue-builder-go/src/wowi"
	flag "github.com/spf13/pflag"
)

// SubCommand represents CLI subcommands
type SubCommand string

const (
	ScrapeSubCommand   SubCommand = "scrape"
	WriteSubCommand    SubCommand = "write"
	ValidateSubCommand SubCommand = "validate"
)

var KnownSubCommands = []SubCommand{ScrapeSubCommand, WriteSubCommand, ValidateSubCommand}

// Flags holds all CLI flags and configuration
type Flags struct {
	SubCommand   SubCommand
	LogLevel     slog.Level
	ScrapeConfig ScrapeConfig
	WriteConfig  WriteConfig
	ValidateFile string
	ShowHelp     bool
	ShowVersion  bool
	MaxWorkers   int
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
	apiVersionStr := "v4" // default

	var sourcesStr []string

	switch subcommand {
	case string(ScrapeSubCommand):
		flagset = flag.NewFlagSet("scrape", flag.ExitOnError)
		flagset.StringVar(&apiVersionStr, "wowi-api-version", "v4", "WowInterface API version (v3 or v4). v3 has more addons and UIDir data")
		flagset.StringArrayVar(&sourcesStr, "source", []string{"wowinterface"}, "sources to scrape")
		flagset.AddFlagSet(defaults)

	case string(WriteSubCommand):
		flagset = flag.NewFlagSet("write", flag.ExitOnError)
		flagset.StringArrayVar(&writeConfig.OutputFiles, "out", []string{}, "write results to file (default: stdout)")
		flagset.StringArrayVar(&sourcesStr, "source", []string{"wowinterface"}, "sources to include")
		flagset.AddFlagSet(defaults)

	case string(ValidateSubCommand):
		flagset = flag.NewFlagSet("validate", flag.ExitOnError)
		flagset.AddFlagSet(defaults)

	default:
		flagset = defaults
	}

	// Parse flags - skip program name and subcommand
	argsToparse := args[2:] // Skip program name and subcommand
	if err := flagset.Parse(argsToparse); err != nil {
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

	// Parse API version for scrape command
	if subcommand == string(ScrapeSubCommand) {
		switch apiVersionStr {
		case "v3":
			scrapeConfig.WoWIAPIVersion = wowi.APIVersionV3
		case "v4":
			scrapeConfig.WoWIAPIVersion = wowi.APIVersionV4
		default:
			return nil, fmt.Errorf("unknown API version: %s (must be v3 or v4)", apiVersionStr)
		}
	}

	// Parse sources after flags are parsed
	if len(sourcesStr) > 0 {
		for _, sourceStr := range sourcesStr {
			switch sourceStr {
			case "wowinterface":
				if subcommand == string(ScrapeSubCommand) {
					scrapeConfig.Sources = append(scrapeConfig.Sources, types.WowInterfaceSource)
				} else if subcommand == string(WriteSubCommand) {
					writeConfig.Sources = append(writeConfig.Sources, types.WowInterfaceSource)
				}
			case "github":
				if subcommand == string(ScrapeSubCommand) {
					scrapeConfig.Sources = append(scrapeConfig.Sources, types.GitHubSource)
				} else if subcommand == string(WriteSubCommand) {
					writeConfig.Sources = append(writeConfig.Sources, types.GitHubSource)
				}
			default:
				return nil, fmt.Errorf("unknown source: %s", sourceStr)
			}
		}
	}

	// Assign parsed values
	flags.SubCommand = SubCommand(subcommand)
	flags.LogLevel = logLevel
	flags.ScrapeConfig = scrapeConfig
	flags.WriteConfig = writeConfig

	// Set max workers in configs
	flags.ScrapeConfig.MaxWorkers = flags.MaxWorkers

	// Parse validate file from remaining args
	if subcommand == string(ValidateSubCommand) {
		remainingArgs := flagset.Args()
		if len(remainingArgs) < 1 {
			return nil, fmt.Errorf("validate command requires a catalogue file path")
		}
		flags.ValidateFile = remainingArgs[0]
	}

	return flags, nil
}

// printUsage prints usage information
func printUsage(flagset *flag.FlagSet) {
	fmt.Println("usage: strongbox-catalogue-builder <scrape|write|validate> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  scrape           Scrape addon data and write catalogues to state/ directory")
	fmt.Println("  write            Generate catalogues from existing state files")
	fmt.Println("  validate <file>  Validate a catalogue JSON file")
	fmt.Println()
	fmt.Println("Options:")
	flagset.PrintDefaults()
}
