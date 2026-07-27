//go:build js && wasm

package megaport

import (
	"testing"

	"github.com/megaport/megaport-cli/internal/wasm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteWithArgs_RejectsUnknownFlags is the ESD-1634 regression test:
// the WASM entrypoint must report an unknown flag as an error, matching the
// native build, instead of silently dropping it and either proceeding or
// failing later with a misleading "required flag not set" message. Each case
// also carries --help: flag parsing fails before Cobra ever checks --help, so
// this has no effect while the fix holds, but it keeps the case side-effect
// free (no real network call or order placed) if the rejection ever
// regresses and the args are actually executed.
func TestExecuteWithArgs_RejectsUnknownFlags(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"unknown flag on a leaf command", []string{"megaport-cli", "locations", "list", "--help", "--totally-bogus-flag"}},
		{"unknown flag before the subcommand", []string{"megaport-cli", "--totally-bogus-flag", "locations", "list", "--help"}},
		{"unknown flag between subcommand levels", []string{"megaport-cli", "locations", "--totally-bogus-flag", "list", "--help"}},
		{"ticket repro: typo'd boolean flag on a buy command", []string{"megaport-cli", "ports", "buy", "--help", "--marketplace-visability", "false"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wasm.ResetOutputBuffers()
			err := ExecuteWithArgs(tc.args)
			require.Error(t, err, "%v should return an error", tc.args)
			assert.Contains(t, err.Error(), "unknown flag", "%v should report the unknown flag", tc.args)
			assert.NotContains(t, err.Error(), "required flag(s)", "%v should not fall through to a required-flag error", tc.args)
			// The failure must land in result.error (the returned error), not be
			// buried in result.output; only the usage hint stays in output.
			assert.NotContains(t, wasm.GetCapturedOutput(), "unknown flag", "%v error text must not stay in output", tc.args)
		})
	}
}

// TestExecuteWithArgs_ValidTraversalStillResolves guards the other half of
// ESD-1634: removing the dead UnknownFlags allowlist must not break
// subcommand resolution when a persistent flag precedes the subcommand, or
// when help is requested at a nested command. Cases are help-only so no
// network call is made.
func TestExecuteWithArgs_ValidTraversalStillResolves(t *testing.T) {
	cases := []struct {
		name string
		args []string
		// usage anchors on the "Usage:" block so a parent's "Example usage"
		// prose (e.g. the root help's Examples list) can't satisfy the match
		// if traversal actually failed and root help was shown instead.
		usage string
	}{
		{"persistent flag before subcommand", []string{"megaport-cli", "--no-color", "locations", "list", "--help"}, "Usage:\n  megaport-cli locations list"},
		{"persistent flag between subcommand levels", []string{"megaport-cli", "locations", "--no-color", "list", "--help"}, "Usage:\n  megaport-cli locations list"},
		{"deeply nested subcommand", []string{"megaport-cli", "ports", "buy", "--help"}, "Usage:\n  megaport-cli ports buy"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wasm.ResetOutputBuffers()
			_ = ExecuteWithArgs(tc.args)
			out := wasm.GetCapturedOutput()
			assert.NotContains(t, out, "unknown flag", "%v should not be treated as an unknown flag", tc.args)
			assert.NotContains(t, out, "unknown command", "%v should be a registered command", tc.args)
			assert.Contains(t, out, tc.usage, "%v help should show its usage path", tc.args)
		})
	}
}
