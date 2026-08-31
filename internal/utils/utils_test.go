package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/validation"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStderr captures what the function writes to os.Stderr.
// Not parallel-safe: it redirects the global os.Stderr via os.Pipe.
// Do not call t.Parallel() in tests that use this helper.
// The read end is drained concurrently to prevent pipe-buffer deadlocks when
// fn() writes more data than the OS pipe buffer can hold.
func captureStderr(t *testing.T, fn func()) (result string) {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	defer func() { os.Stderr = old }()

	// Drain the read end concurrently so fn() cannot block on a full pipe buffer.
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = io.Copy(&buf, r)
	}()

	// Use defers for cleanup so runtime.Goexit (from t.Fatal/require.*) inside
	// fn() does not leave the pipe open and the goroutine blocked forever.
	// result is set after the goroutine drains the pipe, via the named return.
	defer func() {
		_ = w.Close() // signal EOF to the goroutine
		<-done        // wait for all data to be read
		_ = r.Close()
		result = buf.String()
	}()

	fn()
	return
}

// buildJSONChild builds a minimal cobra root+child command tree with the flags
// needed by WrapOutputFormatRunE and WrapColorAwareRunE, with --output pre-set
// to the given format. Returns only the child command.
func buildJSONChild(format string) *cobra.Command {
	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().Bool("no-color", false, "")
	root.PersistentFlags().String("fields", "", "")
	root.PersistentFlags().String("query", "", "")
	root.PersistentFlags().String("template", "", "")
	child := &cobra.Command{Use: "list"}
	child.Flags().String("output", format, "")
	root.AddCommand(child)
	return child
}

func TestShouldDisableColors(t *testing.T) {
	origArgs := os.Args
	origNoColorEnv, origNoColorEnvSet := os.LookupEnv("NO_COLOR")
	defer func() {
		os.Args = origArgs
		if origNoColorEnvSet {
			os.Setenv("NO_COLOR", origNoColorEnv)
		} else {
			os.Unsetenv("NO_COLOR")
		}
	}()

	t.Run("returns true when NO_COLOR env is set", func(t *testing.T) {
		os.Args = []string{"cmd"}
		t.Setenv("NO_COLOR", "1")
		assert.True(t, ShouldDisableColors())
	})

	t.Run("returns true when --no-color arg is present", func(t *testing.T) {
		os.Unsetenv("NO_COLOR")
		os.Args = []string{"cmd", "--no-color"}
		assert.True(t, ShouldDisableColors())
	})

	t.Run("returns false when neither is set", func(t *testing.T) {
		os.Unsetenv("NO_COLOR")
		os.Args = []string{"cmd"}
		assert.False(t, ShouldDisableColors())
	})
}

func TestGetCurrentEnv(t *testing.T) {
	origEnv := Env
	defer func() { Env = origEnv }()

	t.Run("returns production by default", func(t *testing.T) {
		Env = ""
		assert.Equal(t, "production", GetCurrentEnv())
	})

	t.Run("returns configured env if set", func(t *testing.T) {
		Env = "staging"
		assert.Equal(t, "staging", GetCurrentEnv())
	})
}

func TestWrapRunE(t *testing.T) {
	t.Run("success returns nil", func(t *testing.T) {
		wrapped := WrapRunE(func(cmd *cobra.Command, args []string) error {
			return nil
		})
		cmd := &cobra.Command{Use: "test"}
		err := wrapped(cmd, []string{})
		assert.NoError(t, err)
		// SilenceUsage should not be set on success.
		assert.False(t, cmd.SilenceUsage)
	})

	t.Run("error returns formatted error and silences usage", func(t *testing.T) {
		wrapped := WrapRunE(func(cmd *cobra.Command, args []string) error {
			return errors.New("something went wrong")
		})
		cmd := &cobra.Command{Use: "test"}
		err := wrapped(cmd, []string{"arg1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error running test command")
		assert.Contains(t, err.Error(), "something went wrong")
		assert.Contains(t, err.Error(), "arg1")
		assert.True(t, cmd.SilenceUsage)
		assert.True(t, cmd.SilenceErrors)

		var cliErr *exitcodes.CLIError
		require.True(t, errors.As(err, &cliErr))
		assert.Equal(t, exitcodes.General, cliErr.Code)
	})

	t.Run("error message includes command name and args", func(t *testing.T) {
		wrapped := WrapRunE(func(cmd *cobra.Command, args []string) error {
			return errors.New("fail")
		})
		cmd := &cobra.Command{Use: "deploy"}
		err := wrapped(cmd, []string{"a", "b"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Command: deploy")
		assert.Contains(t, err.Error(), "[a b]")
	})

	t.Run("query flag without json output returns usage error", func(t *testing.T) {
		wrapped := WrapRunE(func(cmd *cobra.Command, args []string) error {
			return nil
		})
		root := &cobra.Command{Use: "root"}
		root.PersistentFlags().String("query", "", "")
		root.PersistentFlags().String("fields", "", "")
		root.PersistentFlags().String("output", "table", "")
		child := &cobra.Command{Use: "version"}
		root.AddCommand(child)
		require.NoError(t, root.PersistentFlags().Set("query", "[*].uid"))

		err := wrapped(child, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--query flag requires --output json")

		var cliErr *exitcodes.CLIError
		require.True(t, errors.As(err, &cliErr))
		assert.Equal(t, exitcodes.Usage, cliErr.Code)
	})

	t.Run("query flag with json output passes format guard", func(t *testing.T) {
		called := false
		wrapped := WrapRunE(func(cmd *cobra.Command, args []string) error {
			called = true
			return nil
		})
		root := &cobra.Command{Use: "root"}
		root.PersistentFlags().String("query", "", "")
		root.PersistentFlags().String("fields", "", "")
		root.PersistentFlags().String("output", "table", "")
		child := &cobra.Command{Use: "version"}
		root.AddCommand(child)
		require.NoError(t, root.PersistentFlags().Set("query", "[*].uid"))
		require.NoError(t, root.PersistentFlags().Set("output", "json"))

		err := wrapped(child, []string{})
		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("go-template without --template returns usage error", func(t *testing.T) {
		wrapped := WrapRunE(func(cmd *cobra.Command, args []string) error {
			return nil
		})
		root := &cobra.Command{Use: "root"}
		root.PersistentFlags().String("fields", "", "")
		root.PersistentFlags().String("query", "", "")
		root.PersistentFlags().String("template", "", "")
		root.PersistentFlags().String("output", "go-template", "")
		child := &cobra.Command{Use: "list"}
		root.AddCommand(child)

		err := wrapped(child, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--template is required")

		var cliErr *exitcodes.CLIError
		require.True(t, errors.As(err, &cliErr))
		assert.Equal(t, exitcodes.Usage, cliErr.Code)
	})
}

func TestWrapColorAwareRunE(t *testing.T) {
	t.Run("extracts noColor flag correctly when true", func(t *testing.T) {
		var capturedNoColor bool
		wrapped := WrapColorAwareRunE(func(cmd *cobra.Command, args []string, noColor bool) error {
			capturedNoColor = noColor
			return nil
		})

		root := &cobra.Command{Use: "root"}
		root.PersistentFlags().Bool("no-color", false, "")
		child := &cobra.Command{Use: "child"}
		root.AddCommand(child)
		require.NoError(t, root.PersistentFlags().Set("no-color", "true"))

		err := wrapped(child, []string{})
		assert.NoError(t, err)
		assert.True(t, capturedNoColor)
	})

	t.Run("defaults noColor to false when flag missing", func(t *testing.T) {
		var capturedNoColor bool
		wrapped := WrapColorAwareRunE(func(cmd *cobra.Command, args []string, noColor bool) error {
			capturedNoColor = noColor
			return nil
		})

		// Command without the no-color flag on root.
		cmd := &cobra.Command{Use: "orphan"}
		err := wrapped(cmd, []string{})
		assert.NoError(t, err)
		assert.False(t, capturedNoColor)
	})

	t.Run("error is formatted", func(t *testing.T) {
		wrapped := WrapColorAwareRunE(func(cmd *cobra.Command, args []string, noColor bool) error {
			return errors.New("color error")
		})
		root := &cobra.Command{Use: "root"}
		root.PersistentFlags().Bool("no-color", false, "")
		child := &cobra.Command{Use: "child"}
		root.AddCommand(child)

		err := wrapped(child, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error running child command")
		assert.Contains(t, err.Error(), "color error")
		assert.True(t, child.SilenceUsage)
	})

	t.Run("query flag without json output returns usage error", func(t *testing.T) {
		wrapped := WrapColorAwareRunE(func(cmd *cobra.Command, args []string, noColor bool) error {
			return nil
		})
		root := &cobra.Command{Use: "root"}
		root.PersistentFlags().Bool("no-color", false, "")
		root.PersistentFlags().String("query", "", "")
		root.PersistentFlags().String("fields", "", "")
		root.PersistentFlags().String("output", "table", "")
		child := &cobra.Command{Use: "status"}
		root.AddCommand(child)
		require.NoError(t, root.PersistentFlags().Set("query", "[*].uid"))

		err := wrapped(child, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--query flag requires --output json")

		var cliErr *exitcodes.CLIError
		require.True(t, errors.As(err, &cliErr))
		assert.Equal(t, exitcodes.Usage, cliErr.Code)
	})

	t.Run("query flag with json output passes format guard", func(t *testing.T) {
		called := false
		wrapped := WrapColorAwareRunE(func(cmd *cobra.Command, args []string, noColor bool) error {
			called = true
			return nil
		})
		root := &cobra.Command{Use: "root"}
		root.PersistentFlags().Bool("no-color", false, "")
		root.PersistentFlags().String("query", "", "")
		root.PersistentFlags().String("fields", "", "")
		root.PersistentFlags().String("output", "table", "")
		child := &cobra.Command{Use: "status"}
		root.AddCommand(child)
		require.NoError(t, root.PersistentFlags().Set("query", "[*].uid"))
		require.NoError(t, root.PersistentFlags().Set("output", "json"))

		err := wrapped(child, []string{})
		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("go-template without --template returns usage error", func(t *testing.T) {
		wrapped := WrapColorAwareRunE(func(cmd *cobra.Command, args []string, noColor bool) error {
			return nil
		})
		root := &cobra.Command{Use: "root"}
		root.PersistentFlags().Bool("no-color", false, "")
		root.PersistentFlags().String("fields", "", "")
		root.PersistentFlags().String("query", "", "")
		root.PersistentFlags().String("template", "", "")
		root.PersistentFlags().String("output", "go-template", "")
		child := &cobra.Command{Use: "list"}
		root.AddCommand(child)

		err := wrapped(child, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--template is required")

		var cliErr *exitcodes.CLIError
		require.True(t, errors.As(err, &cliErr))
		assert.Equal(t, exitcodes.Usage, cliErr.Code)
	})
}

func TestWrapOutputFormatRunE(t *testing.T) {
	t.Run("extracts output format and noColor correctly", func(t *testing.T) {
		var capturedFormat string
		var capturedNoColor bool
		wrapped := WrapOutputFormatRunE(func(cmd *cobra.Command, args []string, noColor bool, format string) error {
			capturedFormat = format
			capturedNoColor = noColor
			return nil
		})

		root := &cobra.Command{Use: "root"}
		root.PersistentFlags().Bool("no-color", false, "")
		child := &cobra.Command{Use: "list"}
		child.Flags().String("output", "json", "")
		root.AddCommand(child)
		require.NoError(t, root.PersistentFlags().Set("no-color", "true"))

		err := wrapped(child, []string{})
		assert.NoError(t, err)
		assert.Equal(t, "json", capturedFormat)
		assert.True(t, capturedNoColor)
	})

	t.Run("defaults to table format when flag missing", func(t *testing.T) {
		var capturedFormat string
		wrapped := WrapOutputFormatRunE(func(cmd *cobra.Command, args []string, noColor bool, format string) error {
			capturedFormat = format
			return nil
		})

		// No output flag defined at all.
		cmd := &cobra.Command{Use: "test"}
		err := wrapped(cmd, []string{})
		assert.NoError(t, err)
		assert.Equal(t, FormatTable, capturedFormat)
	})

	t.Run("invalid output format returns error with usage exit code", func(t *testing.T) {
		wrapped := WrapOutputFormatRunE(func(cmd *cobra.Command, args []string, noColor bool, format string) error {
			return nil
		})

		root := &cobra.Command{Use: "root"}
		root.PersistentFlags().Bool("no-color", false, "")
		child := &cobra.Command{Use: "list"}
		child.Flags().String("output", "", "")
		root.AddCommand(child)
		require.NoError(t, child.Flags().Set("output", "yaml"))

		err := wrapped(child, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid output format: yaml")

		var cliErr *exitcodes.CLIError
		require.True(t, errors.As(err, &cliErr))
		assert.Equal(t, exitcodes.Usage, cliErr.Code)
	})

	t.Run("query flag with non-json format returns usage error", func(t *testing.T) {
		wrapped := WrapOutputFormatRunE(func(cmd *cobra.Command, args []string, noColor bool, format string) error {
			return nil
		})

		root := &cobra.Command{Use: "root"}
		root.PersistentFlags().Bool("no-color", false, "")
		root.PersistentFlags().String("query", "", "")
		root.PersistentFlags().String("fields", "", "")
		// --output is registered as a local flag on the child to match production
		// usage (WrapOutputFormatRunE reads it via cmd.Flags(), not PersistentFlags).
		child := &cobra.Command{Use: "list"}
		child.Flags().String("output", "table", "")
		root.AddCommand(child)
		require.NoError(t, root.PersistentFlags().Set("query", "[*].uid"))
		// output is "table" (the default set above); guard should reject it.

		err := wrapped(child, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--query flag requires --output json")

		var cliErr *exitcodes.CLIError
		require.True(t, errors.As(err, &cliErr))
		assert.Equal(t, exitcodes.Usage, cliErr.Code)
	})

	t.Run("function error is formatted", func(t *testing.T) {
		wrapped := WrapOutputFormatRunE(func(cmd *cobra.Command, args []string, noColor bool, format string) error {
			return errors.New("inner failure")
		})

		root := &cobra.Command{Use: "root"}
		root.PersistentFlags().Bool("no-color", false, "")
		child := &cobra.Command{Use: "get"}
		child.Flags().String("output", "table", "")
		root.AddCommand(child)

		err := wrapped(child, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error running get command")
		assert.Contains(t, err.Error(), "inner failure")
	})

	t.Run("go-template without --template returns usage error", func(t *testing.T) {
		wrapped := WrapOutputFormatRunE(func(cmd *cobra.Command, args []string, noColor bool, format string) error {
			return nil
		})

		root := &cobra.Command{Use: "root"}
		root.PersistentFlags().Bool("no-color", false, "")
		root.PersistentFlags().String("fields", "", "")
		root.PersistentFlags().String("query", "", "")
		root.PersistentFlags().String("template", "", "")
		child := &cobra.Command{Use: "list"}
		child.Flags().String("output", "go-template", "")
		root.AddCommand(child)

		err := wrapped(child, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--template is required")

		var cliErr *exitcodes.CLIError
		require.True(t, errors.As(err, &cliErr))
		assert.Equal(t, exitcodes.Usage, cliErr.Code)
	})

	t.Run("uppercase output format is accepted and lowercased", func(t *testing.T) {
		for _, raw := range []string{"JSON", "Json", "CSV", "XML", "Table"} {
			var capturedFormat string
			wrapped := WrapOutputFormatRunE(func(cmd *cobra.Command, args []string, noColor bool, format string) error {
				capturedFormat = format
				return nil
			})

			root := &cobra.Command{Use: "root"}
			root.PersistentFlags().Bool("no-color", false, "")
			root.PersistentFlags().String("fields", "", "")
			root.PersistentFlags().String("query", "", "")
			root.PersistentFlags().String("template", "", "")
			child := &cobra.Command{Use: "list"}
			child.Flags().String("output", raw, "")
			root.AddCommand(child)

			err := wrapped(child, []string{})
			assert.NoError(t, err, "format %q should be accepted", raw)
			assert.Equal(t, strings.ToLower(raw), capturedFormat, "format %q should be lowercased", raw)
		}
	})
}

func TestRequireYesForJSONBuy(t *testing.T) {
	newCmd := func() *cobra.Command {
		cmd := &cobra.Command{Use: "buy"}
		cmd.Flags().String("json", "", "")
		cmd.Flags().String("json-file", "", "")
		cmd.Flags().Bool("yes", false, "")
		return cmd
	}

	t.Run("json without yes is a usage error", func(t *testing.T) {
		cmd := newCmd()
		require.NoError(t, cmd.Flags().Set("json", `{"name":"test"}`))

		yes, err := RequireYesForJSONBuy(cmd)
		assert.False(t, yes)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--yes is required to confirm a purchase when using --json or --json-file")
		var cliErr *exitcodes.CLIError
		require.ErrorAs(t, err, &cliErr)
		assert.Equal(t, exitcodes.Usage, cliErr.Code)
	})

	t.Run("json-file without yes is a usage error", func(t *testing.T) {
		cmd := newCmd()
		require.NoError(t, cmd.Flags().Set("json-file", "./config.json"))

		yes, err := RequireYesForJSONBuy(cmd)
		assert.False(t, yes)
		require.Error(t, err)
	})

	t.Run("json with yes succeeds", func(t *testing.T) {
		cmd := newCmd()
		require.NoError(t, cmd.Flags().Set("json", `{"name":"test"}`))
		require.NoError(t, cmd.Flags().Set("yes", "true"))

		yes, err := RequireYesForJSONBuy(cmd)
		assert.True(t, yes)
		assert.NoError(t, err)
	})

	t.Run("no json and no yes succeeds, leaving confirmation to the caller", func(t *testing.T) {
		cmd := newCmd()

		yes, err := RequireYesForJSONBuy(cmd)
		assert.False(t, yes)
		assert.NoError(t, err)
	})
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		wantCode int
	}{
		// Auth patterns
		{"auth - error logging in", "failed to log in: bad creds", exitcodes.Authentication},
		{"auth - access key not provided", "access key not provided", exitcodes.Authentication},
		{"auth - secret key not provided", "secret key not provided", exitcodes.Authentication},
		{"auth - authentication failed", "authentication failed for user", exitcodes.Authentication},
		{"auth - Authorize", "failed to Authorize request", exitcodes.Authentication},

		// Usage patterns
		{"usage - invalid output format", "invalid output format: yaml", exitcodes.Usage},
		{"usage - required flag", "required flag \"name\" not set", exitcodes.Usage},
		{"usage - not set when not using interactive", "required flag \"term\" not set when not using interactive or JSON input", exitcodes.Usage},
		{"usage - at least one field", "at least one field must be updated", exitcodes.Usage},
		{"usage - at least one of these flags", "at least one of these flags must be set: name, term", exitcodes.Usage},
		{"usage - invalid location ID", "invalid location ID: abc", exitcodes.Usage},
		{"usage - invalid ID combo", "invalid port ID provided", exitcodes.Usage},
		{"usage - failed to parse JSON", "failed to parse JSON: invalid character 'h' in literal true", exitcodes.Usage},
		{"usage - failed to parse JSON file", "failed to parse JSON file: unexpected end of JSON input", exitcodes.Usage},

		// API patterns
		{"api - error listing", "failed to list ports", exitcodes.API},
		{"api - error getting", "failed to get VXC details", exitcodes.API},
		{"api - error creating", "failed to create MCR", exitcodes.API},
		{"api - error updating", "failed to update port name", exitcodes.API},
		{"api - error deleting", "failed to delete service", exitcodes.API},
		{"api - error buying", "failed to buy port", exitcodes.API},
		{"api - error modifying", "failed to modify VXC", exitcodes.API},
		{"api - failed to retrieve", "failed to retrieve port info", exitcodes.API},
		{"api - failed to buy", "failed to buy VXC", exitcodes.API},
		{"api - failed to validate", "failed to validate order", exitcodes.API},
		{"api - API failure", "API failure: 500 internal server error", exitcodes.API},

		// General/unknown
		{"general - unknown error", "something unexpected happened", exitcodes.General},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := classifyError(errors.New(tt.errMsg))
			assert.Equal(t, tt.wantCode, code)
		})
	}
}

func TestClassifyError_CLIErrorPassthrough(t *testing.T) {
	// A CLIError already carrying a code should have that code preserved.
	inner := exitcodes.New(exitcodes.Cancelled, fmt.Errorf("cancelled by user"))
	assert.Equal(t, exitcodes.Cancelled, classifyError(inner))

	// Works through wrapping too.
	wrapped := fmt.Errorf("outer: %w", inner)
	assert.Equal(t, exitcodes.Cancelled, classifyError(wrapped))
}

func TestClassifyError_SDKErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantCode   int
	}{
		{"401 -> Authentication", 401, exitcodes.Authentication},
		{"403 -> Authentication", 403, exitcodes.Authentication},
		{"404 -> API", 404, exitcodes.API},
		{"422 -> API", 422, exitcodes.API},
		{"429 -> API", 429, exitcodes.API},
		{"500 -> API", 500, exitcodes.API},
		{"502 -> API", 502, exitcodes.API},
		{"503 -> API", 503, exitcodes.API},
		{"200 -> General (no match)", 200, exitcodes.General},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := classifyError(makeAPIError(tt.statusCode, ""))
			assert.Equal(t, tt.wantCode, code)
		})
	}
}

func TestWrapSessionExpiredError_RequiresAccessToken(t *testing.T) {
	// With tokenPresent false, a 401 stays an ordinary auth error rather than
	// being reclassified as session-expired, on every build target.
	err := wrapSessionExpiredError(makeAPIError(401, ""), false)
	assert.Equal(t, exitcodes.Authentication, classifyError(err))
	assert.NotContains(t, err.Error(), SessionExpiredMarker)
}

func TestClassifyError_ValidationError(t *testing.T) {
	// A raw ValidationError must map to the usage exit code. Its message is
	// "Invalid <field>: ..." (capital I), which the lowercase "invalid" heuristic
	// misses, so the type check carries it.
	verr := validation.NewValidationError("name", "", "must not be empty")
	assert.Equal(t, exitcodes.Usage, classifyError(verr))

	// errors.As traverses wrapping, so a wrapped ValidationError classifies the same.
	wrapped := fmt.Errorf("validation failed for port: %w", verr)
	assert.Equal(t, exitcodes.Usage, classifyError(wrapped))
}

func TestWrapRunE_CancelledError(t *testing.T) {
	// WrapRunE must preserve the Cancelled code when the action function
	// returns a CLIError that already carries it.
	wrapped := WrapRunE(func(cmd *cobra.Command, args []string) error {
		return exitcodes.NewCancelledError(fmt.Errorf("cancelled by user"))
	})
	cmd := &cobra.Command{Use: "test"}
	err := wrapped(cmd, []string{})
	require.Error(t, err)

	var cliErr *exitcodes.CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, exitcodes.Cancelled, cliErr.Code)
}

func TestWrapColorAwareRunE_CancelledError(t *testing.T) {
	wrapped := WrapColorAwareRunE(func(cmd *cobra.Command, args []string, noColor bool) error {
		return exitcodes.NewCancelledError(fmt.Errorf("cancelled by user"))
	})
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("no-color", false, "")
	err := wrapped(cmd, []string{})
	require.Error(t, err)

	var cliErr *exitcodes.CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, exitcodes.Cancelled, cliErr.Code)
}

func TestWrapRunE_AuthError(t *testing.T) {
	wrapped := WrapRunE(func(cmd *cobra.Command, args []string) error {
		return errors.New("failed to log in: invalid credentials")
	})
	cmd := &cobra.Command{Use: "test"}
	err := wrapped(cmd, []string{})
	require.Error(t, err)

	var cliErr *exitcodes.CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, exitcodes.Authentication, cliErr.Code)
}

func TestWrapRunE_APIError(t *testing.T) {
	wrapped := WrapRunE(func(cmd *cobra.Command, args []string) error {
		return errors.New("failed to list ports: connection refused")
	})
	cmd := &cobra.Command{Use: "test"}
	err := wrapped(cmd, []string{})
	require.Error(t, err)

	var cliErr *exitcodes.CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, exitcodes.API, cliErr.Code)
}

func TestWrapRunE_ValidationError(t *testing.T) {
	// A command that fails with a ValidationError must exit with the usage code,
	// end to end through the wrapper, not just at classifyError.
	wrapped := WrapRunE(func(cmd *cobra.Command, args []string) error {
		return validation.NewValidationError("name", "", "must not be empty")
	})
	cmd := &cobra.Command{Use: "test"}
	err := wrapped(cmd, []string{})
	require.Error(t, err)

	var cliErr *exitcodes.CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}

// parseErrorJSON unmarshals a JSON error envelope from s.
func parseErrorJSON(t *testing.T, s string) (code int, errType, message string) {
	t.Helper()
	var env struct {
		Error struct {
			Code    int    `json:"code"`
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal([]byte(s), &env), "stderr was not valid JSON: %q", s)
	return env.Error.Code, env.Error.Type, env.Error.Message
}

func TestWrapRunE_JSONErrorOutput(t *testing.T) {
	wrapped := WrapRunE(func(cmd *cobra.Command, args []string) error {
		return errors.New("inner api error")
	})
	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().String("fields", "", "")
	root.PersistentFlags().String("query", "", "")
	root.PersistentFlags().String("template", "", "")
	root.PersistentFlags().String("output", "json", "")
	child := &cobra.Command{Use: "list"}
	root.AddCommand(child)

	stderr := captureStderr(t, func() {
		err := wrapped(child, []string{})
		require.Error(t, err)
		// In JSON mode the error is not verbose-wrapped.
		assert.Equal(t, "inner api error", err.Error())
	})

	code, errType, msg := parseErrorJSON(t, stderr)
	assert.Equal(t, exitcodes.General, code)
	assert.Equal(t, "general_error", errType)
	assert.Equal(t, "inner api error", msg)
}

func TestWrapColorAwareRunE_JSONErrorOutput(t *testing.T) {
	wrapped := WrapColorAwareRunE(func(cmd *cobra.Command, args []string, noColor bool) error {
		return errors.New("auth failure")
	})
	// WrapColorAwareRunE reads --output from root persistent flags.
	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().Bool("no-color", false, "")
	root.PersistentFlags().String("fields", "", "")
	root.PersistentFlags().String("query", "", "")
	root.PersistentFlags().String("template", "", "")
	root.PersistentFlags().String("output", "json", "")
	child := &cobra.Command{Use: "list"}
	root.AddCommand(child)

	stderr := captureStderr(t, func() {
		err := wrapped(child, []string{})
		require.Error(t, err)
		assert.Equal(t, "auth failure", err.Error())
	})

	code, _, msg := parseErrorJSON(t, stderr)
	assert.Equal(t, exitcodes.General, code)
	assert.Equal(t, "auth failure", msg)
}

// buildTableChild builds a root+child tree with --output preset to a non-json
// format on the root persistent flags, as WrapColorAwareRunE reads it.
func buildTableChild(format string) *cobra.Command {
	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().Bool("no-color", false, "")
	root.PersistentFlags().String("fields", "", "")
	root.PersistentFlags().String("query", "", "")
	root.PersistentFlags().String("template", "", "")
	root.PersistentFlags().String("output", format, "")
	child := &cobra.Command{Use: "list"}
	root.AddCommand(child)
	return child
}

// A failing command must print its error to stderr even when the action returns
// the error silently: cobra has SilenceErrors set, so the wrapper owns the print.
func TestWrapColorAwareRunE_PrintsSilentErrorToStderr(t *testing.T) {
	output.SetOutputFormat("table")
	t.Cleanup(func() { output.ResetState() })

	wrapped := WrapColorAwareRunE(func(cmd *cobra.Command, args []string, noColor bool) error {
		return errors.New("connection blew up")
	})
	child := buildTableChild("table")

	stderr := captureStderr(t, func() {
		err := wrapped(child, []string{})
		require.Error(t, err)
	})
	assert.Contains(t, stderr, "connection blew up", "wrapper must surface the error on stderr")
}

// When an action already showed the error via output.PrintError, the wrapper
// must not print it a second time.
func TestWrapColorAwareRunE_NoDoublePrintWhenActionSelfPrints(t *testing.T) {
	output.SetOutputFormat("table")
	t.Cleanup(func() { output.ResetState() })

	wrapped := WrapColorAwareRunE(func(cmd *cobra.Command, args []string, noColor bool) error {
		output.PrintError("custom failure", noColor)
		return errors.New("custom failure")
	})
	child := buildTableChild("table")

	stderr := captureStderr(t, func() {
		err := wrapped(child, []string{})
		require.Error(t, err)
	})
	assert.Equal(t, 1, strings.Count(stderr, "custom failure"), "error must appear exactly once on stderr")
}

// Invalid output format is a usage error that must reach stderr, not vanish.
func TestWrapOutputFormatRunE_InvalidFormatPrintsToStderr(t *testing.T) {
	output.SetOutputFormat("table")
	t.Cleanup(func() { output.ResetState() })

	wrapped := WrapOutputFormatRunE(func(cmd *cobra.Command, args []string, noColor bool, format string) error {
		return nil
	})
	child := buildJSONChild("bogus")

	stderr := captureStderr(t, func() {
		err := wrapped(child, []string{})
		require.Error(t, err)
	})
	assert.Contains(t, stderr, "invalid output format", "usage error must reach stderr")
}

func TestWrapOutputFormatRunE_JSONErrorOutput(t *testing.T) {
	wrapped := WrapOutputFormatRunE(func(cmd *cobra.Command, args []string, noColor bool, format string) error {
		return errors.New("failed to get port: not found")
	})
	child := buildJSONChild("json")

	stderr := captureStderr(t, func() {
		err := wrapped(child, []string{})
		require.Error(t, err)
		assert.Equal(t, "failed to get port: not found", err.Error())
	})

	code, errType, msg := parseErrorJSON(t, stderr)
	assert.Equal(t, exitcodes.API, code)
	assert.Equal(t, "api_error", errType)
	assert.Equal(t, "failed to get port: not found", msg)
}

// buildPreRunChild builds a minimal root+child tree carrying the persistent
// flags FinishPreRunError reads (--output, --no-color), with --output pre-set.
func buildPreRunChild(format string) *cobra.Command {
	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().String("output", format, "")
	root.PersistentFlags().Bool("no-color", false, "")
	child := &cobra.Command{Use: "buy"}
	root.AddCommand(child)
	return child
}

func TestFinishPreRunError(t *testing.T) {
	t.Run("json mode emits the structured envelope and silences usage", func(t *testing.T) {
		child := buildPreRunChild("json")
		var err error
		stderr := captureStderr(t, func() {
			err = FinishPreRunError(child, []string{}, exitcodes.NewUsageError(errors.New("bad flag")))
		})

		require.Error(t, err)
		assert.True(t, child.SilenceUsage, "usage must be silenced so cobra does not dump the usage block")
		assert.True(t, child.SilenceErrors)

		var cliErr *exitcodes.CLIError
		require.True(t, errors.As(err, &cliErr))
		assert.Equal(t, exitcodes.Usage, cliErr.Code)

		code, errType, msg := parseErrorJSON(t, stderr)
		assert.Equal(t, exitcodes.Usage, code)
		assert.Equal(t, "usage_error", errType)
		assert.Equal(t, "bad flag", msg)
	})

	t.Run("non-json mode prints to stderr and returns a wrapped usage error", func(t *testing.T) {
		child := buildPreRunChild("table")
		var err error
		stderr := captureStderr(t, func() {
			err = FinishPreRunError(child, []string{"arg1"}, exitcodes.NewUsageError(errors.New("bad flag")))
		})

		require.Error(t, err)
		assert.True(t, child.SilenceUsage)
		assert.Contains(t, stderr, "bad flag", "the error should surface on stderr in non-json mode")
		assert.NotContains(t, stderr, "\"error\"", "non-json mode must not emit the JSON envelope")

		var cliErr *exitcodes.CLIError
		require.True(t, errors.As(err, &cliErr))
		assert.Equal(t, exitcodes.Usage, cliErr.Code)
		assert.Contains(t, err.Error(), "error running buy command")
	})

	t.Run("missing --output flag defaults to table without panicking", func(t *testing.T) {
		root := &cobra.Command{Use: "root"}
		child := &cobra.Command{Use: "buy"}
		root.AddCommand(child)

		var err error
		stderr := captureStderr(t, func() {
			err = FinishPreRunError(child, []string{}, exitcodes.NewUsageError(errors.New("bad flag")))
		})

		require.Error(t, err)
		assert.Contains(t, stderr, "bad flag")

		var cliErr *exitcodes.CLIError
		require.True(t, errors.As(err, &cliErr))
		assert.Equal(t, exitcodes.Usage, cliErr.Code)
	})
}
