//go:build js && wasm

package megaport

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/megaport/megaport-cli/internal/base/help"
	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/megaport/megaport-cli/internal/wasm"
	"github.com/spf13/cobra"
)

// ExecuteWithArgs runs the command tree against args and returns the error the
// host should surface as result.error, or nil. The error is returned only when
// it hasn't already reached the user another way: the JSON envelope branch and
// any path where output already streamed or was captured return nil, so the
// host doesn't double-render a message the terminal already shows.
func ExecuteWithArgs(args []string) error {
	// Reset all flags on the command tree so that flag values from a previous
	// execution don't leak into the current one. Cobra marks flags as "Changed"
	// after parsing, and this state persists when the same command tree is reused
	// across multiple WASM invocations.
	resetAllFlags(rootCmd)

	// Default to colored output each invocation. fatih/color derives its global
	// NoColor from isatty, always false under js/wasm, so it would otherwise strip
	// every colorized value/badge/status line while go-pretty still colors the
	// table chrome. Resetting here (not just in PersistentPreRunE) keeps paths that
	// skip PersistentPreRunE, e.g. --help, from inheriting a prior run's flag.
	// PersistentPreRunE re-applies --no-color for the command actually running.
	color.NoColor = false

	// Direct output to our WASM buffer
	rootCmd.SetOut(wasm.WasmOutputBuffer)
	rootCmd.SetErr(wasm.WasmOutputBuffer)

	// Enable traversal so a subcommand is still resolved when persistent
	// flags precede it (e.g. "--output json ports list"). This does not
	// relax unknown-flag handling; see enableTraversalForAllCommands.
	rootCmd.TraverseChildren = true

	// Enable subcommand traversal for ALL commands, not just root
	enableTraversalForAllCommands(rootCmd)

	// Properly handle args - if first arg is the program name, skip it
	argsToUse := args
	if len(args) > 0 && (args[0] == "megaport-cli" || args[0] == "./megaport-cli") {
		argsToUse = args[1:]
	}

	// Own the failure output. SilenceErrors stops cobra writing "Error: ..." to
	// the buffer before we can route it to result.error. Set unconditionally
	// because the command tree is reused across invocations.
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	// Set the args on the root command
	rootCmd.SetArgs(argsToUse)

	// Execute and capture errors. ExecuteC returns the command that ran so we
	// can resolve --output from the most specific source (local flag on the
	// executed command, used by WrapOutputFormatRunE, vs. the root persistent
	// flag, used by WrapRunE / WrapColorAwareRunE).
	executedCmd, err := rootCmd.ExecuteC()

	if err == nil {
		return nil
	}

	// A *CLIError in JSON mode already wrote the structured envelope via
	// PrintErrorJSON, so re-emit a clean one and return nil (routing it to
	// result.error too would stack a red line on machine-readable output).
	// Gate on *CLIError, not the flag: pre-wrapper errors (e.g. flag parse)
	// aren't *CLIError and still need routing.
	var cliErr *exitcodes.CLIError
	if errors.As(err, &cliErr) && resolveOutputFormat(executedCmd) == utils.FormatJSON {
		// Reset the buffer to remove any plain-text output written before
		// the error (commands may call output.PrintError etc. before
		// returning), then re-emit a single clean JSON envelope so the
		// buffer contains only valid JSON.
		wasm.ResetOutputBuffers()
		output.PrintErrorJSON(cliErr.Code, cliErr.Error())
		return nil
	}
	// When a live-output handler is registered the error may already have
	// streamed during execution (the RunE wrapper's PrintError, or cobra's own
	// error print). Streamed chunks cannot be retracted, so returning the error
	// as result.error too would double-render it in the terminal.
	if wasm.HasOutputHandler() {
		// Only surface the error when nothing streamed AND nothing was captured
		// (e.g. a flag-parse failure before any command ran): keep the usage
		// hint in output and return the error so the host renders it and emits
		// failure telemetry. Otherwise the error is already in the terminal or
		// the captured buffer, so return nil to avoid duplicating it.
		if !wasm.DidStreamOutput() && wasm.WasmOutputBuffer.String() == "" {
			fmt.Fprintf(wasm.WasmOutputBuffer, "Run 'megaport-cli --help' to see the list of available commands.\n")
			return err
		}
		return nil
	}

	// Capture path (no handler): clear anything written before the error and
	// keep only the usage hint in output. The failure message itself is
	// returned so the host can route it to result.error rather than burying it
	// in result.output uncolored.
	wasm.ResetOutputBuffers()
	fmt.Fprintf(wasm.WasmOutputBuffer, "Run 'megaport-cli --help' to see the list of available commands.\n")
	return err
}

// resolveOutputFormat returns the --output value for the command that ran.
// It checks the executed command's local --output flag first (used by
// WrapOutputFormatRunE commands that shadow the persistent root flag), then
// falls back to the root persistent flag (used by WrapRunE /
// WrapColorAwareRunE commands).
func resolveOutputFormat(cmd *cobra.Command) string {
	var raw string
	if cmd != nil {
		// Only use the local --output flag if the user explicitly set it.
		// WrapOutputFormatRunE commands always register a local --output flag
		// whose default is "table"; reading it unconditionally would mask the
		// user's intent expressed via the root persistent flag.
		if f := cmd.Flags().Lookup("output"); f != nil && f.Changed {
			raw = f.Value.String()
		}
	}
	if raw == "" {
		raw, _ = rootCmd.PersistentFlags().GetString("output")
	}
	return strings.ToLower(raw)
}

func EnsureRootCommandOutput(writer io.Writer) {
	rootCmd.SetOut(writer)
	rootCmd.SetErr(writer)
}

func init() {
	// Initialize common components (uses WASM-specific registerModules that excludes config)
	InitializeCommon()

	// Remove the auto-generated completion command since it's not supported in WASM
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Store the original help function so we can call it when needed
	originalHelpFunc := rootCmd.HelpFunc()

	// Define the help text configuration once to avoid duplication
	getRootHelpBuilder := func(disableColor bool) *help.CommandHelpBuilder {
		return &help.CommandHelpBuilder{
			CommandName: "megaport-cli",
			ShortDesc:   "A CLI tool to interact with the Megaport API",
			LongDesc:    "Megaport CLI provides a command line interface to interact with the Megaport API.\n\nThe CLI allows you to manage Megaport resources such as ports, VXCs, MCRs, MVEs, service keys, and more.",
			OptionalFlags: map[string]string{
				"--no-color":  "Disable colored output",
				"--no-header": "Suppress table and CSV column headers (useful for scripting)",
				"--no-pager":  "Disable pager for long table output (no-op in browser version)",
				"--output":    "Output format (table, json, csv, xml, go-template)",
				"--template":  "Go template string for --output go-template",
				"--help":      "Show help for any command",
				"--env":       "Environment to use (production, staging, development)",
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
				"Set the MEGAPORT_ENDPOINT environment variable to connect to a different environment",
			},
			DisableColor: disableColor,
		}
	}

	// Create a help function that runs the help.CommandHelpBuilder with the current noColor setting
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		// For the root command, regenerate the help text completely
		if cmd == rootCmd {
			rootHelp := getRootHelpBuilder(noColor)
			cmd.Long = rootHelp.Build(rootCmd)
		} else if cmd.Long != "" {
			// For non-root commands, modify the existing help text only if there is a Long description
			helpBuilder := &help.CommandHelpBuilder{
				CommandName:  cmd.UseLine(),
				ShortDesc:    cmd.Short,
				LongDesc:     cmd.Long,
				DisableColor: noColor,
			}
			cmd.Long = helpBuilder.Build(rootCmd)
		}

		// Call the original help function
		originalHelpFunc(cmd, args)
	})

	// Set the initial root command help text
	rootCmd.Long = getRootHelpBuilder(false).Build(rootCmd)

	// Configure Cobra to show proper error messages for unknown commands
	rootCmd.SilenceErrors = false
	rootCmd.SilenceUsage = false

	moduleRegistry.RegisterAll(rootCmd)
}
