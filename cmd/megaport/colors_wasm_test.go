//go:build js && wasm

package megaport

import (
	"testing"

	"github.com/fatih/color"
	"github.com/megaport/megaport-cli/internal/wasm"
	"github.com/stretchr/testify/assert"
)

// ESD-1593: fatih/color's global NoColor defaults to true under js/wasm (isatty
// is always false), which stripped every colorized value/badge/status line while
// go-pretty still colored the table chrome. These tests drive the real command
// path (ExecuteWithArgs -> PersistentPreRunE) so they fail if the color wiring is
// reverted. Each command reaches PersistentPreRunE and then returns on an offline
// usage error (invalid --output), so no network call is made.

// TestWasmColor_DefaultRunEnablesColor covers AC1: a normal run leaves color
// enabled (NoColor false) in the browser.
func TestWasmColor_DefaultRunEnablesColor(t *testing.T) {
	orig := color.NoColor
	defer func() { color.NoColor = orig }()
	color.NoColor = true // start from the hostile js/wasm default

	wasm.ResetOutputBuffers()
	_ = ExecuteWithArgs([]string{"megaport-cli", "ports", "list", "--output", "invalid"})

	assert.False(t, color.NoColor, "a default run should enable fatih color in the browser")
}

// TestWasmColor_NoColorFlagDisablesColor covers AC2: --no-color disables fatih
// color. This is the guard on the PersistentPreRunE flag sync; without it the
// per-invocation default (false) would leave color on.
func TestWasmColor_NoColorFlagDisablesColor(t *testing.T) {
	orig := color.NoColor
	defer func() { color.NoColor = orig }()
	color.NoColor = false

	wasm.ResetOutputBuffers()
	_ = ExecuteWithArgs([]string{"megaport-cli", "ports", "list", "--no-color", "--output", "invalid"})

	assert.True(t, color.NoColor, "--no-color should disable fatih color")
}

// TestWasmColor_NoStaleFlagAcrossRuns guards the per-invocation reset in
// ExecuteWithArgs: a --no-color run must not bleed into a later run that skips
// PersistentPreRunE (e.g. --help), which would otherwise keep color disabled.
func TestWasmColor_NoStaleFlagAcrossRuns(t *testing.T) {
	orig := color.NoColor
	defer func() { color.NoColor = orig }()

	wasm.ResetOutputBuffers()
	_ = ExecuteWithArgs([]string{"megaport-cli", "ports", "list", "--no-color", "--output", "invalid"})
	assert.True(t, color.NoColor, "sanity: --no-color run disables color")

	// --help returns before PersistentPreRunE runs, so only the ExecuteWithArgs
	// reset can clear the prior run's flag.
	wasm.ResetOutputBuffers()
	_ = ExecuteWithArgs([]string{"megaport-cli", "ports", "list", "--help"})
	assert.False(t, color.NoColor, "a subsequent non --no-color run must re-enable color")
}
