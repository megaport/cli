//go:build !js && !wasm

package megaport

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/megaport/megaport-cli/internal/base/help"
	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/config"
	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/megaport/megaport-cli/internal/wasm"
	"github.com/spf13/cobra"
)

func init() {
	// Initialize common components
	InitializeCommon()

	// Structurally tag flag-parsing failures (unknown flag, unknown shorthand
	// flag, malformed value) as usage errors at the point cobra generates them,
	// instead of pattern-matching the message text later in exitCodeFromError.
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return exitcodes.NewUsageError(err)
	})

	// Apply non-WASM specific initialization
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		defaultWarnings := applyDefaultSettings(cmd)

		// Auto-disable color when stdout is not a TTY (piped output)
		if !cmd.Flags().Changed("no-color") && !output.IsTerminal() {
			noColor = true
			_ = cmd.Flags().Set("no-color", "true")
		}
		format := strings.ToLower(outputFormat)
		validFmt := false
		for _, vf := range utils.ValidFormats {
			if format == vf {
				validFmt = true
				break
			}
		}
		if !validFmt {
			return utils.FinishPreRunError(cmd, args, exitcodes.NewUsageError(fmt.Errorf("invalid output format: %s. Must be one of: %s",
				outputFormat, strings.Join(utils.ValidFormats, ", "))))
		}
		output.SetOutputFormat(format)

		verbosity := "normal"
		if quiet {
			verbosity = "quiet"
		} else if verbose {
			verbosity = "verbose"
		}
		cfg := output.GetOutputConfig()
		cfg.NoHeader = noHeader
		cfg.NoPager = noPager
		cfg.Verbosity = verbosity
		cfg.Format = format // normalized to lower-case above
		output.ApplyOutputConfig(cfg)

		// Emit config-default warnings now that output format and verbosity are
		// configured, so PrintWarning routes to stderr under --output json and
		// is suppressed under --quiet.
		for _, w := range defaultWarnings {
			output.PrintWarning("%s", noColor, w)
		}

		// Validate retry flags
		if utils.MaxRetries < 0 {
			return utils.FinishPreRunError(cmd, args, exitcodes.NewUsageError(fmt.Errorf("--max-retries must be >= 0, got %d", utils.MaxRetries)))
		}

		// Reject an explicit non-positive --timeout; omitting it uses the default.
		if err := utils.ValidateTimeoutFlag(cmd); err != nil {
			return utils.FinishPreRunError(cmd, args, exitcodes.NewUsageError(err))
		}

		// Run cobra's own required-flags check here, ahead of when cobra would
		// normally run it after PersistentPreRunE, so a missing required flag
		// gets tagged as a typed usage error instead of falling back to
		// isCobraUsageError's message-substring match.
		if err := cmd.ValidateRequiredFlags(); err != nil {
			return utils.FinishPreRunError(cmd, args, exitcodes.NewUsageError(err))
		}

		return nil
	}

	// Store the original help function so we can call it when needed
	originalHelpFunc := rootCmd.HelpFunc()

	// Define the help text configuration once to avoid duplication
	getRootHelpBuilder := func(disableColor bool) *help.CommandHelpBuilder {
		return &help.CommandHelpBuilder{
			CommandName: "megaport-cli",
			ShortDesc:   "A CLI tool to interact with the Megaport API",
			LongDesc:    "Megaport CLI provides a command line interface to interact with the Megaport API.\n\nThe CLI allows you to manage Megaport resources such as ports, VXCs, MCRs, MVEs, service keys, and more.",
			OptionalFlags: map[string]string{
				"--no-color":    "Disable colored output",
				"--no-header":   "Suppress table and CSV column headers (useful for scripting)",
				"--no-pager":    "Disable pager for long table output",
				"--output":      "Output format (table, json, csv, xml, go-template)",
				"--template":    "Go template string for --output go-template",
				"--help":        "Show help for any command",
				"--env":         "Environment to use (production, staging, development)",
				"--quiet":       "Suppress informational output, only show errors and data",
				"--verbose":     "Show additional debug information",
				"--no-retry":    "Disable automatic retry on transient API failures",
				"--max-retries": "Maximum number of retries for transient API failures (default 3)",
			},
			Examples: []string{
				"megaport-cli ports list",
				"megaport-cli vxc buy --interactive",
				"megaport-cli mcr get [mcrUID]",
				"megaport-cli locations list",
			},
			ImportantNotes: []string{
				"Use the --help flag with any command to see specific usage information",
				"Authentication is handled via the MEGAPORT_ACCESS_KEY and MEGAPORT_SECRET_KEY environment variables",
				"By default, the CLI connects to the Megaport production environment",
				"Set the MEGAPORT_ENVIRONMENT environment variable to connect to a different environment",
			},
			DisableColor: disableColor,
		}
	}

	// Create a help function that runs the help.CommandHelpBuilder with the current noColor setting
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		var isColorDisabled bool

		if noColor {
			isColorDisabled = true
		}

		// Check raw command line args directly
		for _, arg := range os.Args {
			if arg == "--no-color" || arg == "-no-color" {
				isColorDisabled = true
				break
			}
		}

		// Also check environment
		if _, exists := os.LookupEnv("NO_COLOR"); exists {
			isColorDisabled = true
		}

		// Auto-disable color when stdout is not a TTY
		if !output.IsTerminal() {
			isColorDisabled = true
		}

		// For the root command, regenerate the help text completely
		if cmd == rootCmd {
			rootHelp := getRootHelpBuilder(isColorDisabled)
			cmd.Long = rootHelp.Build(rootCmd)
		} else if cmd.Long != "" {
			// For non-root commands, modify the existing help text only if there is a Long description
			helpBuilder := &help.CommandHelpBuilder{
				CommandName:  cmd.UseLine(),
				ShortDesc:    cmd.Short,
				LongDesc:     cmd.Long,
				DisableColor: isColorDisabled,
			}
			cmd.Long = helpBuilder.Build(rootCmd)
		}

		// Call the original help function
		originalHelpFunc(cmd, args)
	})

	// Set the initial root command help text
	rootCmd.Long = getRootHelpBuilder(false).Build(rootCmd)

	// Register all commands from the modules
	moduleRegistry.RegisterAll(rootCmd)
}

// applyDefaultSettings reads saved defaults from config and applies them to
// cmd's flags. It returns a list of warning messages to emit later (after the
// caller has configured output format and verbosity) so that warnings are
// routed and suppressed correctly under --output json / --quiet.
func applyDefaultSettings(cmd *cobra.Command) []string {
	manager, err := config.NewConfigManager()
	if err != nil {
		return []string{fmt.Sprintf("Could not load saved default settings: %v", err)}
	}

	var failed []string

	applyBool := func(flag string, target *bool) {
		if cmd.Flags().Changed(flag) {
			return
		}
		val, exists := manager.GetDefault(flag)
		if !exists {
			return
		}
		// Type mismatches (e.g. a string where a bool is expected) are treated
		// as absent so a stray manual edit to config.json does not block the CLI.
		boolVal, ok := val.(bool)
		if !ok {
			return
		}
		if setErr := cmd.Flags().Set(flag, fmt.Sprintf("%t", boolVal)); setErr != nil {
			failed = append(failed, flag)
			return
		}
		if target != nil {
			*target = boolVal
		}
	}

	applyString := func(flag string) {
		if cmd.Flags().Changed(flag) {
			return
		}
		val, exists := manager.GetDefault(flag)
		if !exists {
			return
		}
		strVal, ok := val.(string)
		if !ok {
			return
		}
		if setErr := cmd.Flags().Set(flag, strVal); setErr != nil {
			failed = append(failed, flag)
		}
	}

	// Capture which flags the user set on the CLI before applying defaults,
	// because cmd.Flags().Set (called inside applyBool) marks a flag as Changed
	// regardless of origin — we can't distinguish CLI-set from config-set after
	// the fact.
	cliQuiet := cmd.Flags().Changed("quiet")
	cliVerbose := cmd.Flags().Changed("verbose")

	applyBool("no-color", &noColor)
	applyString("output")
	applyBool("quiet", &quiet)
	applyBool("verbose", &verbose)
	applyBool("no-pager", &noPager)

	var warnings []string
	if len(failed) > 0 {
		warnings = append(warnings, fmt.Sprintf("Could not apply saved defaults for: %s", strings.Join(failed, ", ")))
	}

	// --quiet and --verbose are mutually exclusive (see MarkFlagsMutuallyExclusive).
	// A saved default can combine with a CLI flag (or a manually-edited config
	// can set both) and bypass cobra's validation. Resolve by preferring the
	// explicitly-set flag; if both are from config, drop quiet so unexpected
	// output surfaces a problem to the user rather than silently suppressing it.
	if quiet && verbose {
		dropped := "quiet"
		switch {
		case cliQuiet && !cliVerbose:
			verbose = false
			_ = cmd.Flags().Set("verbose", "false")
			dropped = "verbose"
		case cliVerbose && !cliQuiet:
			quiet = false
			_ = cmd.Flags().Set("quiet", "false")
		default:
			quiet = false
			_ = cmd.Flags().Set("quiet", "false")
		}
		warnings = append(warnings, fmt.Sprintf("Saved defaults set both --quiet and --verbose; dropping --%s", dropped))
	}

	return warnings
}

// ExecuteWithArgs adds all child commands to the root command and executes with given args.
func ExecuteWithArgs(args []string) {
	// Direct output to our WASM buffer
	rootCmd.SetOut(wasm.WasmOutputBuffer)
	rootCmd.SetErr(wasm.WasmOutputBuffer)

	// Use cobra's SetArgs instead of mutating the global os.Args
	rootCmd.SetArgs(args[1:]) // Skip program name (args[0])

	// Execute and capture errors
	err := rootCmd.Execute()

	if err != nil {
		fmt.Fprintf(wasm.WasmOutputBuffer, "Error executing command: %v\n", err)
	}
}

func EnsureRootCommandOutput(writer io.Writer) {
	rootCmd.SetOut(writer)
	rootCmd.SetErr(writer)
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(exitCodeFromError(err))
	}
}

func exitCodeFromError(err error) int {
	// Check for explicitly typed CLIError (from wrappers or cmdbuilder)
	var cliErr *exitcodes.CLIError
	if errors.As(err, &cliErr) {
		return cliErr.Code
	}

	// Cobra usage errors (unknown command/flag, wrong arg count)
	msg := err.Error()
	if isCobraUsageError(msg) {
		return exitcodes.Usage
	}

	// PersistentPreRunE format validation
	if strings.Contains(msg, "invalid output format") {
		return exitcodes.Usage
	}

	return exitcodes.General
}

// isCobraUsageError is a defensive fallback matching cobra's usage-error text.
// Flag-parse and missing-required-flag errors are already typed as usage errors
// at the source (SetFlagErrorFunc and the ValidateRequiredFlags check), so the
// matches for those are a backstop for any that arrive untyped; "unknown command"
// and the arg-count validators have no such hook and rely on this match.
func isCobraUsageError(msg string) bool {
	cobraPatterns := []string{
		"unknown command",
		"unknown flag",
		"unknown shorthand flag",
		// Covers every cobra arg-count validator: ExactArgs, MinimumNArgs,
		// MaximumNArgs, and RangeArgs all phrase their error as "... arg(s) ...".
		"arg(s)",
		"required flag(s)",
	}
	for _, p := range cobraPatterns {
		if strings.Contains(msg, p) {
			return true
		}
	}
	return false
}
