package utils

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/validation"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
)

const (
	FormatTable      = "table"
	FormatJSON       = "json"
	FormatCSV        = "csv"
	FormatXML        = "xml"
	FormatGoTemplate = "go-template"

	// StatusDecommissioning is used for filtering inactive resources. The SDK
	// exports STATUS_CANCELLED and STATUS_DECOMMISSIONED but not this one.
	StatusDecommissioning = "DECOMMISSIONING"
)

// These persistent-flag variables are written by cobra during flag parsing
// on the main goroutine (via PersistentFlags().<T>Var(&utils.X, ...)) and
// read later while the command runs. They are intentionally plain types
// rather than sync/atomic wrappers because pflag binds to *string, *bool
// and *int — switching to atomics would require per-flag pflag.Value shims
// and break every test that assigns these directly (see auth/login tests).
//
// The safety argument rests on a strict happens-before ordering *per
// invocation*:
//  1. cmd.Execute() parses flags and writes each variable on the main goroutine.
//  2. Only after parsing completes does cobra invoke PersistentPreRunE /
//     RunE, which may spawn background work (spinners, API retry loops).
//  3. Nothing writes these variables again until the command returns and
//     all spawned work has been joined, so concurrent readers within a
//     single invocation always observe the parsed value.
//
// In the long-lived WASM process each subsequent command re-enters this
// sequence: flags are re-parsed on the main goroutine only after the
// previous invocation has fully returned, so the per-invocation invariant
// holds across re-entries too.
//
// If a future caller needs to mutate any of these at runtime (for example:
// re-login with a new profile mid-command, toggle LogHTTP after a
// suspicious response, or adjust MaxRetries between API calls), introduce
// an atomic snapshot rather than mutating the globals — the set-once
// invariant is what makes plain variables acceptable here, and it applies
// equally to every variable in this block.
var (
	// Env is the target environment (prod, dev, staging).
	Env string

	// ProfileOverride is the config profile name selected via --profile.
	ProfileOverride string

	// NoRetry disables automatic retry on transient API failures. Set via --no-retry flag.
	NoRetry bool

	// MaxRetries overrides the default retry count. Set via --max-retries flag.
	MaxRetries int

	// LogHTTP enables raw HTTP request/response logging to stderr. Set via --log-http flag.
	LogHTTP bool

	// BaseURL overrides the API base URL (e.g. http://localhost:8080). Set via --base-url flag.
	BaseURL string

	// TokenURL overrides the OAuth token endpoint. Use with --base-url when the token server
	// is not one of the three standard Megaport auth hosts. Set via --token-url flag.
	TokenURL string

	ValidFormats     = []string{FormatTable, FormatJSON, FormatCSV, FormatXML, FormatGoTemplate}
	ValidFormatsWASM = []string{FormatTable, FormatJSON, FormatCSV, FormatXML}
)

func ShouldDisableColors() bool {
	// Check if NO_COLOR environment variable is set (standard for disabling color)
	_, noColorEnv := os.LookupEnv("NO_COLOR")

	// Check raw command line args - more reliable with --help
	for _, arg := range os.Args {
		if arg == "--no-color" || arg == "-no-color" {
			return true
		}
	}

	return noColorEnv
}

func GetCurrentEnv() string {
	if Env == "" {
		return "production" // Default to production if not specified
	}
	return Env
}

// applyTemplateFilter reads the --template persistent flag and calls output.SetTemplateString.
// Called by all RunE wrappers so the template string is always set before PrintOutput is invoked.
func applyTemplateFilter(cmd *cobra.Command) {
	tmplStr, err := cmd.Root().PersistentFlags().GetString("template")
	if err != nil {
		tmplStr = ""
	}
	output.SetTemplateString(tmplStr)
}

// applyQueryFilter reads the --query persistent flag and calls output.SetOutputQuery.
// Returns the query string so RunE wrappers can validate it against the selected output format.
func applyQueryFilter(cmd *cobra.Command) string {
	queryStr, err := cmd.Root().PersistentFlags().GetString("query")
	if err != nil {
		queryStr = ""
	}
	output.SetOutputQuery(queryStr)
	return queryStr
}

// enforceQueryFormatGuard returns a usage error if --query is set and the
// resolved output format is not json. The caller must pass in the already-
// resolved format so this function does not re-derive it; this avoids a
// discrepancy when a subcommand defines a local --output flag that shadows
// the root persistent flag.
func enforceQueryFormatGuard(cmd *cobra.Command, queryStr, format string) error {
	if queryStr == "" {
		return nil
	}
	if format != FormatJSON {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		return exitcodes.NewUsageError(fmt.Errorf("--query flag requires --output json"))
	}
	return nil
}

// applyFieldsFilter reads the --fields persistent flag and calls output.SetOutputFields.
// Called by all RunE wrappers so the filter is always consistent regardless of wrapper used.
func applyFieldsFilter(cmd *cobra.Command) {
	fieldsStr, err := cmd.Root().PersistentFlags().GetString("fields")
	if err != nil {
		fieldsStr = ""
	}
	if fieldsStr != "" {
		var fields []string
		for _, f := range strings.Split(fieldsStr, ",") {
			if f = strings.TrimSpace(f); f != "" {
				fields = append(fields, f)
			}
		}
		output.SetOutputFields(fields)
	} else {
		output.SetOutputFields(nil)
	}
}

// finishWithError centralizes RunE error handling so a failing command can
// never exit non-zero while printing nothing. cobra has SilenceErrors set, so
// the wrapper owns the error output: in json mode it emits one structured
// envelope and returns the bare error; in every other format it prints the
// error to stderr (unless an action already showed one via output.PrintError,
// tracked by the output.ErrorEmitted latch) and returns a verbose wrapped error.
func finishWithError(cmd *cobra.Command, args []string, err error, format string, noColor bool, tokenPresent bool) error {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	original := err
	err = wrapSessionExpiredError(err, tokenPresent)
	code := classifyError(err)
	if format == FormatJSON {
		output.PrintErrorJSON(code, err.Error())
		cmd.Root().SilenceErrors = true
		return exitcodes.New(code, err)
	}
	if !output.ErrorEmitted() {
		output.PrintError("%s", noColor, err.Error())
	} else if err != original {
		// An action already printed (and, under a live-output handler, streamed)
		// the raw error, latching ErrorEmitted so the block above is skipped. But
		// wrapSessionExpiredError rewrote it to carry SessionExpiredMarker, and a
		// streamed chunk can't be retracted; without emitting again the marker
		// reaches neither the stream nor the capture buffer. Re-emit so a
		// streaming host still sees the re-auth signal.
		output.PrintError("%s", noColor, err.Error())
	}
	wrapped := fmt.Errorf("error running %s command\n\nError: %v\nCommand: %s\nArguments: %v\n\nFor more information, use the --help flag",
		cmd.Name(), err, cmd.Name(), args)
	return exitcodes.New(code, wrapped)
}

// FinishPreRunError routes a PreRunE / PersistentPreRunE validation failure
// through the same single-envelope path RunE errors use. Without it these
// errors bypass finishWithError, so under --output json cobra prints plain text
// plus a usage block instead of the structured envelope every other error path
// emits. Call sites must return the error this returns: finishWithError sets
// SilenceUsage/SilenceErrors on the command so cobra neither re-prints the error
// nor dumps usage, and the embedded *CLIError carries the exit code (mapped by
// the native Execute; surfaced by the WASM ExecuteWithArgs envelope branch).
func FinishPreRunError(cmd *cobra.Command, args []string, err error) error {
	// A prior command in the long-lived WASM process may have left the latch
	// set; reset so the non-json path always surfaces this error on stderr.
	output.ResetErrorEmitted()
	tokenPresent := sessionTokenPresent()
	rawFormat, _ := cmd.Root().PersistentFlags().GetString("output")
	format := strings.ToLower(rawFormat)
	syncOutputFormat(format)
	noColor := resolveNoColor(cmd)
	return finishWithError(cmd, args, err, format, noColor, tokenPresent)
}

// syncOutputFormat aligns the output package's format with the format the
// wrapper resolved from the flags. This matters in the long-lived WASM process,
// where a stale "json" left by a previous command would otherwise make
// PrintError no-op and swallow a non-json command's error. An empty format
// (no --output flag) falls back to table.
func syncOutputFormat(format string) {
	if format == "" {
		format = FormatTable
	}
	output.SetOutputFormat(format)
}

// resolveNoColor reads the --no-color persistent flag, defaulting to false
// (color enabled) when the flag is absent.
func resolveNoColor(cmd *cobra.Command) bool {
	noColor, err := cmd.Root().PersistentFlags().GetBool("no-color")
	if err != nil {
		return false
	}
	return noColor
}

// WrapRunE wraps a RunE function to set SilenceUsage to true if an error occurs and formats the error message.
func WrapRunE(runE func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		output.ResetErrorEmitted()
		tokenPresent := sessionTokenPresent()
		applyFieldsFilter(cmd)
		applyTemplateFilter(cmd)
		rawFormat, _ := cmd.Root().PersistentFlags().GetString("output")
		format := strings.ToLower(rawFormat)
		syncOutputFormat(format)
		noColor := resolveNoColor(cmd)
		if format == FormatGoTemplate && output.GetTemplateString() == "" {
			return finishWithError(cmd, args, exitcodes.NewUsageError(fmt.Errorf("--template is required when --output go-template is used")), format, noColor, tokenPresent)
		}
		if err := enforceQueryFormatGuard(cmd, applyQueryFilter(cmd), format); err != nil {
			return finishWithError(cmd, args, err, format, noColor, tokenPresent)
		}
		if err := runE(cmd, args); err != nil {
			return finishWithError(cmd, args, err, format, noColor, tokenPresent)
		}
		return nil
	}
}

// WrapColorAwareRunE combines WrapRunE and WrapCommandFunc to handle both error formatting
// and passing the noColor flag to command functions.
func WrapColorAwareRunE(fn func(cmd *cobra.Command, args []string, noColor bool) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		output.ResetErrorEmitted()
		tokenPresent := sessionTokenPresent()
		applyFieldsFilter(cmd)
		applyTemplateFilter(cmd)
		rawFormat, _ := cmd.Root().PersistentFlags().GetString("output")
		format := strings.ToLower(rawFormat)
		syncOutputFormat(format)
		noColor := resolveNoColor(cmd)
		if format == FormatGoTemplate && output.GetTemplateString() == "" {
			return finishWithError(cmd, args, exitcodes.NewUsageError(fmt.Errorf("--template is required when --output go-template is used")), format, noColor, tokenPresent)
		}
		if err := enforceQueryFormatGuard(cmd, applyQueryFilter(cmd), format); err != nil {
			return finishWithError(cmd, args, err, format, noColor, tokenPresent)
		}
		if err := fn(cmd, args, noColor); err != nil {
			return finishWithError(cmd, args, err, format, noColor, tokenPresent)
		}
		return nil
	}
}

// WrapOutputFormatRunE handles both output format and color settings in command functions.
// This wrapper takes a function that needs both output format and noColor parameters.
func WrapOutputFormatRunE(fn func(cmd *cobra.Command, args []string, noColor bool, format string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		output.ResetErrorEmitted()
		tokenPresent := sessionTokenPresent()
		noColor := resolveNoColor(cmd)

		// Get output format value from command
		rawFormat, err := cmd.Flags().GetString("output")
		if err != nil {
			rawFormat = FormatTable // Default to table format if flag not found
		}
		// Lowercase to match WrapRunE / root PersistentPreRunE, so "--output JSON"
		// is accepted everywhere "--output json" is.
		format := strings.ToLower(rawFormat)

		// Validate format
		validFormat := false
		for _, f := range ValidFormats {
			if format == f {
				validFormat = true
				break
			}
		}
		if !validFormat {
			// An invalid format is itself a usage error. Sync the output package
			// to table first so finishWithError surfaces it on stderr rather than
			// suppressing it under a stale json format.
			syncOutputFormat(FormatTable)
			return finishWithError(cmd, args, exitcodes.NewUsageError(fmt.Errorf("invalid output format: %s. Must be one of: %v", format, ValidFormats)), FormatTable, noColor, tokenPresent)
		}

		syncOutputFormat(format)

		applyTemplateFilter(cmd)
		if format == FormatGoTemplate && output.GetTemplateString() == "" {
			return finishWithError(cmd, args, exitcodes.NewUsageError(fmt.Errorf("--template is required when --output go-template is used")), format, noColor, tokenPresent)
		}

		applyFieldsFilter(cmd)
		// Pass the already-resolved format so enforceQueryFormatGuard does not
		// re-read --output from the root persistent flags (which could differ
		// if the subcommand defines its own local --output override).
		if err := enforceQueryFormatGuard(cmd, applyQueryFilter(cmd), format); err != nil {
			return finishWithError(cmd, args, err, format, noColor, tokenPresent)
		}

		if err := fn(cmd, args, noColor, format); err != nil {
			return finishWithError(cmd, args, err, format, noColor, tokenPresent)
		}
		return nil
	}
}

// RequireYesForJSONBuy enforces that a JSON-mode purchase (--json or
// --json-file) also passes --yes, since JSON mode has no interactive
// confirmation prompt to fall back on. Returns the resolved --yes value so
// callers can reuse it for their own confirmation-prompt branch.
func RequireYesForJSONBuy(cmd *cobra.Command) (bool, error) {
	jsonStr, _ := cmd.Flags().GetString("json")
	jsonFile, _ := cmd.Flags().GetString("json-file")
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes && (jsonStr != "" || jsonFile != "") {
		return false, exitcodes.NewUsageError(fmt.Errorf("--yes is required to confirm a purchase when using --json or --json-file"))
	}
	return yes, nil
}

// classifyError inspects an error message to determine the appropriate exit code.
func classifyError(err error) int {
	// Preserve exit codes already set by action functions
	var cliErr *exitcodes.CLIError
	if errors.As(err, &cliErr) {
		return cliErr.Code
	}

	// A validation failure is caller error, so map it to the usage code. Type
	// matching avoids depending on the "Invalid %s" message wording.
	var validationErr *validation.ValidationError
	if errors.As(err, &validationErr) {
		return exitcodes.Usage
	}

	// Type-safe SDK error inspection first
	var apiErr *megaport.ErrorResponse
	if errors.As(err, &apiErr) {
		switch apiErr.Response.StatusCode {
		case 401, 403:
			return exitcodes.Authentication
		case 404, 422, 429, 500, 502, 503:
			return exitcodes.API
		}
	}

	msg := err.Error()

	// Authentication errors
	authPatterns := []string{
		"failed to log in",
		"access key not provided",
		"secret key not provided",
		"authentication",
		"Authorize",
	}
	for _, p := range authPatterns {
		if strings.Contains(msg, p) {
			return exitcodes.Authentication
		}
	}

	// Usage/validation errors
	usagePatterns := []string{
		"invalid output format",
		"required flag",
		"not set when not using interactive",
		"at least one field must be updated",
		"at least one of these flags",
		"invalid location ID",
		// A malformed --json argument is caller error, not an API failure.
		// Matched before the "failed to parse" API pattern below.
		"failed to parse JSON",
	}
	for _, p := range usagePatterns {
		if strings.Contains(msg, p) {
			return exitcodes.Usage
		}
	}
	// "invalid" + "ID" pattern
	if strings.Contains(msg, "invalid") && strings.Contains(msg, "ID") {
		return exitcodes.Usage
	}

	// API errors
	apiPatterns := []string{
		"failed to list",
		"failed to get",
		"failed to create",
		"failed to update",
		"failed to delete",
		"failed to buy",
		"failed to modify",
		"failed to retrieve",
		"failed to validate",
		"failed to restore",
		"failed to lock",
		"failed to unlock",
		"failed to set",
		"failed to print",
		"failed to build",
		"failed to marshal",
		"failed to parse",
		"failed to read",
		"API failure",
	}
	for _, p := range apiPatterns {
		if strings.Contains(msg, p) {
			return exitcodes.API
		}
	}

	return exitcodes.General
}
