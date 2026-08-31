//go:build js && wasm

package megaport

import (
	"testing"

	"github.com/megaport/megaport-cli/internal/wasm"
	"github.com/stretchr/testify/assert"
)

// ESD-1650: the command tree persists across WASM invocations in a browser
// session, so a naive SetHelpFunc that rebuilds a command's help text from its
// own (already mutated) cmd.Long re-colors and re-suffixes it on every call.
// These tests drive the real --help path (ExecuteWithArgs) so they fail if
// that regresses.

// TestWasmHelp_RepeatedInvocationsProduceIdenticalOutput covers the
// accumulation bug directly: calling --help on the same subcommand twice must
// yield byte-identical output, not growing/duplicated help text.
func TestWasmHelp_RepeatedInvocationsProduceIdenticalOutput(t *testing.T) {
	wasm.ResetOutputBuffers()
	_ = ExecuteWithArgs([]string{"megaport-cli", "ports", "list", "--help"})
	first := wasm.WasmOutputBuffer.String()
	assert.NotEmpty(t, first)

	wasm.ResetOutputBuffers()
	_ = ExecuteWithArgs([]string{"megaport-cli", "ports", "list", "--help"})
	second := wasm.WasmOutputBuffer.String()

	assert.Equal(t, first, second, "repeated --help calls must produce identical output, not accumulate coloring/suffixes")
}

// TestWasmHelp_RepeatedInvocationsAcrossManyCalls guards against slow growth
// that a two-call comparison might miss.
func TestWasmHelp_RepeatedInvocationsAcrossManyCalls(t *testing.T) {
	wasm.ResetOutputBuffers()
	_ = ExecuteWithArgs([]string{"megaport-cli", "ports", "list", "--help"})
	baseline := wasm.WasmOutputBuffer.String()
	assert.NotEmpty(t, baseline)

	for i := 0; i < 5; i++ {
		wasm.ResetOutputBuffers()
		_ = ExecuteWithArgs([]string{"megaport-cli", "ports", "list", "--help"})
		out := wasm.WasmOutputBuffer.String()
		assert.Equal(t, baseline, out, "help output must not drift across repeated invocations")
	}
}

// TestWasmHelp_RootCommandRepeatedInvocationsProduceIdenticalOutput covers the
// same guarantee for the root command's help, which is rebuilt from a static
// LongDesc string each time rather than a cached original.
func TestWasmHelp_RootCommandRepeatedInvocationsProduceIdenticalOutput(t *testing.T) {
	wasm.ResetOutputBuffers()
	_ = ExecuteWithArgs([]string{"megaport-cli", "--help"})
	first := wasm.WasmOutputBuffer.String()
	assert.NotEmpty(t, first)

	wasm.ResetOutputBuffers()
	_ = ExecuteWithArgs([]string{"megaport-cli", "--help"})
	second := wasm.WasmOutputBuffer.String()

	assert.Equal(t, first, second, "repeated root --help calls must produce identical output")
}
