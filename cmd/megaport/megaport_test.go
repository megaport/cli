//go:build !js && !wasm

package megaport

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noHeaderTestItem is a minimal struct used to observe table header suppression.
type noHeaderTestItem struct {
	Name string `header:"NAME" json:"name"`
}

// TestNoHeaderFlagWiredThroughPersistentPreRunE verifies that --no-header is
// registered on the root command and that PersistentPreRunE propagates its
// value to the output package by observing that table headers are suppressed.
func TestNoHeaderFlagWiredThroughPersistentPreRunE(t *testing.T) {
	// Restore all output state, the backing var, and the cobra flag so
	// subsequent tests in this package are not affected.
	defer func() {
		output.ResetState()
		noHeader = false
		_ = rootCmd.PersistentFlags().Set("no-header", "false")
	}()

	// Run a no-auth command through the real rootCmd so PersistentPreRunE fires.
	rootCmd.SetArgs([]string{"version", "--no-header"})
	_ = output.CaptureOutput(func() {
		err := rootCmd.Execute()
		require.NoError(t, err)
	})

	// Assert via observable behavior: table output should have no header row.
	captured := output.CaptureOutput(func() {
		_ = output.PrintOutput([]noHeaderTestItem{{Name: "row1"}}, "table", true)
	})
	assert.False(t, strings.Contains(captured, "NAME"), "header row should be suppressed after --no-header")
	assert.True(t, strings.Contains(captured, "row1"), "data rows should still appear")
}

// TestNoPagerDefaultApplied verifies that a "no-pager" default persisted in
// the config file is read by applyDefaultSettings AND forwarded to the output
// package via output.ApplyOutputConfig in PersistentPreRunE. Both sides of
// the wiring are asserted so that removing either call causes the test to fail.
func TestNoPagerDefaultApplied(t *testing.T) {
	// Use an isolated config directory so this test doesn't touch the real one.
	dir := t.TempDir()
	t.Setenv("MEGAPORT_CONFIG_DIR", dir)

	// Write no-pager = true as a config default.
	mgr, err := config.NewConfigManager()
	require.NoError(t, err)
	require.NoError(t, mgr.SetDefault("no-pager", true))

	// Restore package-level state and cobra flag after the test.
	defer func() {
		output.ResetState()
		noPager = false
		_ = rootCmd.PersistentFlags().Set("no-pager", "false")
	}()

	// Fire a lightweight command through the real rootCmd so PersistentPreRunE runs.
	rootCmd.SetArgs([]string{"version"})
	_ = output.CaptureOutput(func() {
		err := rootCmd.Execute()
		require.NoError(t, err)
	})

	// Assert the flag var was set (applyDefaultSettings path).
	assert.True(t, noPager, "noPager package var should be true after config default is applied")
	// Assert the output package was notified (output.ApplyOutputConfig wiring path).
	assert.True(t, output.GetNoPager(), "output.GetNoPager() should be true after PersistentPreRunE wires NoPager via ApplyOutputConfig")
}

// TestTimeoutFlagRejectsNonPositive verifies that an explicit non-positive
// --timeout is rejected as a usage error through PersistentPreRunE, rather than
// silently falling back to a default.
func TestTimeoutFlagRejectsNonPositive(t *testing.T) {
	// Reset the shared persistent flag so the explicit value (and Changed state)
	// doesn't leak into sibling tests that run through rootCmd.
	defer func() {
		f := rootCmd.PersistentFlags().Lookup("timeout")
		_ = f.Value.Set("0s")
		f.Changed = false
	}()

	rootCmd.SetArgs([]string{"version", "--timeout", "0"})
	var execErr error
	_ = output.CaptureOutput(func() {
		execErr = rootCmd.Execute()
	})

	require.Error(t, execErr)
	assert.Equal(t, exitcodes.Usage, exitCodeFromError(execErr))
	assert.Contains(t, execErr.Error(), "--timeout must be greater than 0")
}

// TestInvalidOutputFormatRejected verifies that an unknown --output value is
// rejected as a usage error through PersistentPreRunE and routed through
// FinishPreRunError (so it carries a usage exit code), not silently accepted.
func TestInvalidOutputFormatRejected(t *testing.T) {
	t.Setenv("MEGAPORT_CONFIG_DIR", t.TempDir())
	defer func() {
		output.ResetState()
		f := rootCmd.PersistentFlags().Lookup("output")
		_ = f.Value.Set("table")
		f.Changed = false
	}()

	rootCmd.SetArgs([]string{"version", "--output", "bogus"})
	var execErr error
	_ = output.CaptureOutput(func() {
		execErr = rootCmd.Execute()
	})

	require.Error(t, execErr)
	assert.Equal(t, exitcodes.Usage, exitCodeFromError(execErr))
	assert.Contains(t, execErr.Error(), "invalid output format")
}

// TestNegativeMaxRetriesRejected verifies that a negative --max-retries is
// rejected as a usage error through PersistentPreRunE and routed through
// FinishPreRunError.
func TestNegativeMaxRetriesRejected(t *testing.T) {
	t.Setenv("MEGAPORT_CONFIG_DIR", t.TempDir())
	defer func() {
		output.ResetState()
		f := rootCmd.PersistentFlags().Lookup("max-retries")
		_ = f.Value.Set("3") // restore the default; IntVar binding resets utils.MaxRetries too
		f.Changed = false
	}()

	rootCmd.SetArgs([]string{"version", "--max-retries=-1"})
	var execErr error
	_ = output.CaptureOutput(func() {
		execErr = rootCmd.Execute()
	})

	require.Error(t, execErr)
	assert.Equal(t, exitcodes.Usage, exitCodeFromError(execErr))
	assert.Contains(t, execErr.Error(), "--max-retries must be >= 0")
}

// TestApplyDefaultSettings_WarnsOnConfigLoadFailure verifies that when
// NewConfigManager fails (e.g. the configured config dir cannot be created),
// applyDefaultSettings returns a warning message instead of silently skipping.
func TestApplyDefaultSettings_WarnsOnConfigLoadFailure(t *testing.T) {
	// Create a temp file, then point MEGAPORT_CONFIG_DIR at a subpath of it.
	// os.MkdirAll will fail because the parent is a regular file, forcing
	// NewConfigManager to return an error.
	parent := filepath.Join(t.TempDir(), "not-a-dir")
	require.NoError(t, os.WriteFile(parent, []byte("x"), 0600))
	t.Setenv("MEGAPORT_CONFIG_DIR", filepath.Join(parent, "child"))

	// Guard against drift in NewConfigManager: if it ever stops returning an
	// error for this scenario, the warning assertion below would pass vacuously.
	if _, err := config.NewConfigManager(); err == nil {
		t.Fatal("expected NewConfigManager to fail when config dir cannot be created")
	}

	warnings := applyDefaultSettings(rootCmd)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "Could not load saved default settings")
}

// TestApplyDefaultSettings_ResolvesQuietVerboseConflict verifies that when
// saved defaults would enable both --quiet and --verbose (which are declared
// mutually exclusive on rootCmd), applyDefaultSettings drops one and returns
// a warning. Covers the case where neither flag was passed on the CLI, so
// both get applied from config.
func TestApplyDefaultSettings_ResolvesQuietVerboseConflict(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MEGAPORT_CONFIG_DIR", dir)

	mgr, err := config.NewConfigManager()
	require.NoError(t, err)
	require.NoError(t, mgr.SetDefault("quiet", true))
	require.NoError(t, mgr.SetDefault("verbose", true))

	defer func() {
		output.ResetState()
		quiet = false
		verbose = false
		_ = rootCmd.PersistentFlags().Set("quiet", "false")
		_ = rootCmd.PersistentFlags().Set("verbose", "false")
	}()

	warnings := applyDefaultSettings(rootCmd)

	assert.True(t, verbose, "verbose should remain set (safer default — unexpected output surfaces problems)")
	assert.False(t, quiet, "quiet should be dropped to resolve conflict")
	require.NotEmpty(t, warnings)
	var found bool
	for _, w := range warnings {
		if strings.Contains(w, "--quiet and --verbose") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected a conflict-resolution warning, got %v", warnings)
}

// TestApplyDefaultSettings_CLIVerboseOverridesConfigQuiet verifies that when
// the user passes --verbose on the CLI and a saved default also sets quiet,
// the CLI flag wins. Regression test for a bug where the conflict resolver
// read cmd.Flags().Changed *after* applying defaults, which always reported
// true and caused the CLI-provided flag to be dropped instead of the config one.
func TestApplyDefaultSettings_CLIVerboseOverridesConfigQuiet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MEGAPORT_CONFIG_DIR", dir)

	mgr, err := config.NewConfigManager()
	require.NoError(t, err)
	require.NoError(t, mgr.SetDefault("quiet", true))

	defer func() {
		output.ResetState()
		resetVerbosityFlags(t)
	}()

	// Simulate the user passing --verbose on the CLI.
	require.NoError(t, rootCmd.PersistentFlags().Set("verbose", "true"))
	verbose = true

	_ = applyDefaultSettings(rootCmd)

	assert.True(t, verbose, "CLI-set --verbose should win over config quiet")
	assert.False(t, quiet, "config-sourced quiet should be dropped when CLI set verbose")
}

// resetVerbosityFlags resets the quiet/verbose package vars, their cobra flag
// values, and — critically — their Changed state so the mutually-exclusive
// group check does not fire on subsequent tests that share the global rootCmd.
func resetVerbosityFlags(t *testing.T) {
	t.Helper()
	quiet = false
	verbose = false
	for _, name := range []string{"quiet", "verbose"} {
		require.NoError(t, rootCmd.PersistentFlags().Set(name, "false"))
		if f := rootCmd.PersistentFlags().Lookup(name); f != nil {
			f.Changed = false
		}
	}
}

// TestQuietFlagWiredThroughPersistentPreRunE verifies that --quiet sets
// Verbosity to "quiet" in the output config via ApplyOutputConfig.
func TestQuietFlagWiredThroughPersistentPreRunE(t *testing.T) {
	defer func() {
		output.ResetState()
		resetVerbosityFlags(t)
	}()
	rootCmd.SetArgs([]string{"version", "--quiet"})
	_ = output.CaptureOutput(func() {
		err := rootCmd.Execute()
		require.NoError(t, err)
	})
	assert.True(t, output.IsQuiet(), "IsQuiet should be true after --quiet flag wired via ApplyOutputConfig")
}

// TestVerboseFlagWiredThroughPersistentPreRunE verifies that --verbose sets
// Verbosity to "verbose" in the output config via ApplyOutputConfig.
func TestVerboseFlagWiredThroughPersistentPreRunE(t *testing.T) {
	defer func() {
		output.ResetState()
		resetVerbosityFlags(t)
	}()
	rootCmd.SetArgs([]string{"version", "--verbose"})
	_ = output.CaptureOutput(func() {
		err := rootCmd.Execute()
		require.NoError(t, err)
	})
	assert.True(t, output.IsVerbose(), "IsVerbose should be true after --verbose flag wired via ApplyOutputConfig")
}

func TestExitCodeFromError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		// CLIError with each code
		{"CLIError general", exitcodes.New(exitcodes.General, errors.New("unknown")), exitcodes.General},
		{"CLIError usage", exitcodes.NewUsageError(errors.New("bad flag")), exitcodes.Usage},
		{"CLIError auth", exitcodes.NewAuthError(errors.New("no creds")), exitcodes.Authentication},
		{"CLIError api", exitcodes.NewAPIError(errors.New("500")), exitcodes.API},
		{"CLIError cancelled", exitcodes.NewCancelledError(errors.New("cancelled by user")), exitcodes.Cancelled},

		// Cobra-style errors
		{"cobra unknown command", errors.New(`unknown command "foo" for "megaport-cli"`), exitcodes.Usage},
		{"cobra unknown flag", errors.New(`unknown flag: --bogus`), exitcodes.Usage},
		{"cobra unknown shorthand flag", errors.New(`unknown shorthand flag: 'x' in -x`), exitcodes.Usage},
		{"cobra accepts at most", errors.New(`accepts at most 1 arg(s), received 2`), exitcodes.Usage},
		{"cobra exact args", errors.New(`accepts 1 arg(s), received 0`), exitcodes.Usage},
		{"cobra minimum n args", errors.New(`requires at least 2 arg(s), only received 1`), exitcodes.Usage},
		{"cobra required flag(s)", errors.New(`required flag(s) "name" not set`), exitcodes.Usage},

		// PersistentPreRunE format validation
		{"invalid output format", errors.New("invalid output format: yaml"), exitcodes.Usage},

		// Unknown errors
		{"unknown error", errors.New("something unexpected"), exitcodes.General},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantCode, exitCodeFromError(tt.err))
		})
	}
}

// TestExitCodeFromError_CobraArgValidators drives cobra's own argument
// validators so the asserted exit code is checked against the exact error
// strings cobra produces, not a hand-copied approximation.
func TestExitCodeFromError_CobraArgValidators(t *testing.T) {
	tests := []struct {
		name string
		args cobra.PositionalArgs
		argv []string
	}{
		{"ExactArgs too few", cobra.ExactArgs(1), []string{}},
		{"ExactArgs too many", cobra.ExactArgs(1), []string{"a", "b"}},
		{"MinimumNArgs", cobra.MinimumNArgs(2), []string{"a"}},
		{"MaximumNArgs", cobra.MaximumNArgs(1), []string{"a", "b"}},
		{"RangeArgs", cobra.RangeArgs(1, 2), []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:           "get",
				Args:          tt.args,
				RunE:          func(cmd *cobra.Command, args []string) error { return nil },
				SilenceUsage:  true,
				SilenceErrors: true,
			}
			cmd.SetArgs(tt.argv)
			err := cmd.Execute()
			require.Error(t, err)
			assert.Equal(t, exitcodes.Usage, exitCodeFromError(err), "cobra error: %v", err)
		})
	}
}

// TestUnknownFlagIsTypedUsageError verifies that rootCmd.SetFlagErrorFunc
// tags a flag-parse failure as a typed *exitcodes.CLIError at the point
// cobra generates it, rather than relying on exitCodeFromError's message
// substring match.
func TestUnknownFlagIsTypedUsageError(t *testing.T) {
	rootCmd.SetArgs([]string{"version", "--totally-bogus-flag"})
	var execErr error
	_ = output.CaptureOutput(func() {
		execErr = rootCmd.Execute()
	})

	require.Error(t, execErr)
	var cliErr *exitcodes.CLIError
	require.True(t, errors.As(execErr, &cliErr), "expected a typed *exitcodes.CLIError, got %T: %v", execErr, execErr)
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}

// TestMissingRequiredFlagIsTypedUsageError verifies that the proactive
// cmd.ValidateRequiredFlags() check added to PersistentPreRunE tags a
// missing-required-flag error as a typed *exitcodes.CLIError ahead of
// cobra's own (redundant) ValidateRequiredFlags call.
func TestMissingRequiredFlagIsTypedUsageError(t *testing.T) {
	t.Setenv("MEGAPORT_CONFIG_DIR", t.TempDir())
	defer output.ResetState()

	rootCmd.SetArgs([]string{"billing-market", "set"})
	var execErr error
	_ = output.CaptureOutput(func() {
		execErr = rootCmd.Execute()
	})

	require.Error(t, execErr)
	var cliErr *exitcodes.CLIError
	require.True(t, errors.As(execErr, &cliErr), "expected a typed *exitcodes.CLIError, got %T: %v", execErr, execErr)
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}
