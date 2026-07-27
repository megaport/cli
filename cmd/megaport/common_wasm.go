//go:build js && wasm

package megaport

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/base/registry"
	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/spf13/cobra"
)

// Common variables and declarations for WASM builds
var (
	noColor      bool
	noHeader     bool
	noPager      bool
	outputFormat string
	quiet        bool
	verbose      bool

	// rootCmd is the root command for the CLI
	rootCmd = &cobra.Command{
		Use:           "megaport-cli",
		Short:         "A CLI tool to interact with the Megaport API",
		SilenceUsage:  true,  // Don't show usage on errors
		SilenceErrors: false, // Do show errors
		// Long will be set by the help builder later
		// RunE (not Run) so an unknown command surfaces as a returned error:
		// ExecuteWithArgs routes that to result.error, which the host renders as a
		// failure and uses to emit failure telemetry. Printing the message to the
		// output buffer instead (the old behavior) left result.error empty, so the
		// host treated the failure as success (ESD-1666).
		RunE: func(cmd *cobra.Command, args []string) error {
			// If we reach here with args, it means an unknown command was used
			if len(args) > 0 {
				// Special messages for commands not available in WASM
				switch args[0] {
				case "config":
					return fmt.Errorf("the 'config' command is not available in the browser version\n\nConfiguration is managed through the web UI login form.\nUse the --env flag to specify environment: megaport-cli ports list --env staging")
				case "completion":
					return fmt.Errorf("the 'completion' command is not available in the browser version\n\nShell completion is not applicable in a browser environment")
				case "generate-docs":
					return fmt.Errorf("the 'generate-docs' command is not available in the browser version\n\nDocumentation generation is a development tool, not available in WASM")
				case "version":
					return fmt.Errorf("the 'version' command is not available in the browser version\n\nVersion information is managed by the web application")
				}
				// Error for other unknown commands
				return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
			}
			// No args - just show help
			return cmd.Help()
		},
	}

	// moduleRegistry holds all command modules
	moduleRegistry *registry.Registry
)

// InitializeCommon performs initialization steps common to all platforms
func InitializeCommon() {
	// Initialize module registry
	moduleRegistry = registry.NewRegistry()

	// Register all modules (WASM version excludes config)
	registerModules()

	// Setup persistent flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", utils.FormatTable,
		"Output format (table, json, csv, xml, go-template; go-template not supported in browser version)")
	rootCmd.PersistentFlags().String("template", "", "Go template string for --output go-template (not supported in browser version)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colorful output")
	rootCmd.PersistentFlags().StringVar(&utils.Env, "env", "", "Environment to use (prod, dev, or staging)")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress informational output, only show errors and data")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show additional debug information")
	rootCmd.PersistentFlags().Duration("timeout", 0, "Timeout for the operation (e.g., 30s, 2m, 5m); must be positive. Omit to use each command's built-in default (see the command's own help)")
	rootCmd.PersistentFlags().String("fields", "", "Comma-separated list of fields to include in output (e.g., uid,name,status); use an unknown name to list available fields")
	rootCmd.PersistentFlags().String("query", "", "JMESPath query to filter JSON output (requires --output json)")
	rootCmd.PersistentFlags().BoolVar(&utils.NoRetry, "no-retry", false, "Disable automatic retry on transient API failures")
	rootCmd.PersistentFlags().IntVar(&utils.MaxRetries, "max-retries", 3, "Maximum number of retries for transient API failures")
	rootCmd.PersistentFlags().BoolVar(&noHeader, "no-header", false, "Suppress table and CSV column headers (useful for scripting)")
	rootCmd.PersistentFlags().BoolVar(&noPager, "no-pager", false, "Disable pager for long table output (no-op in browser version)")
	rootCmd.MarkFlagsMutuallyExclusive("quiet", "verbose")
	rootCmd.SuggestionsMinimumDistance = 2

	// Validate retry flags in WASM builds too.
	existingPreRunE := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// fatih/color's global NoColor defaults to true under js/wasm (isatty is
		// always false), which strips every color.*String/colorizeValue/colorizeStatus
		// while go-pretty still colors the table chrome. xterm.js renders ANSI, so
		// mirror the --no-color flag here to keep coloring all-or-nothing.
		color.NoColor = noColor

		verbosity := "normal"
		if quiet {
			verbosity = "quiet"
		} else if verbose {
			verbosity = "verbose"
		}
		format := strings.ToLower(outputFormat)
		validFmt := false
		for _, vf := range utils.ValidFormatsWASM {
			if format == vf {
				validFmt = true
				break
			}
		}
		if !validFmt {
			// Type the error as a usage CLIError so ExecuteWithArgs emits the JSON
			// envelope under --output json, matching the native root and the shared
			// conditional-requirement validators.
			return exitcodes.NewUsageError(fmt.Errorf("invalid output format: %s. Must be one of: %s",
				outputFormat, strings.Join(utils.ValidFormatsWASM, ", ")))
		}
		cfg := output.GetOutputConfig()
		cfg.NoHeader = noHeader
		cfg.NoPager = noPager // no-op in WASM pager; keeps flag wiring symmetric with native
		cfg.Verbosity = verbosity
		cfg.Format = format
		output.ApplyOutputConfig(cfg)
		if utils.MaxRetries < 0 {
			return exitcodes.NewUsageError(fmt.Errorf("--max-retries must be >= 0, got %d", utils.MaxRetries))
		}
		if err := utils.ValidateTimeoutFlag(cmd); err != nil {
			return exitcodes.NewUsageError(err)
		}
		if existingPreRunE != nil {
			return existingPreRunE(cmd, args)
		}
		return nil
	}
}
