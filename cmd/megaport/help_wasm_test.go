//go:build js && wasm

package megaport

import (
	"testing"

	"github.com/megaport/megaport-cli/internal/wasm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The WASM command tree persists across --help runs in one browser session.
// These tests run --help repeatedly and expect the same text every time.
func TestWasmHelp_RepeatedRunsPrintSameText(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"subcommand", []string{"megaport-cli", "ports", "list", "--help"}},
		{"group command", []string{"megaport-cli", "ports", "--help"}},
		{"root", []string{"megaport-cli", "--help"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wasm.ResetOutputBuffers()
			require.NoError(t, ExecuteWithArgs(tc.args))
			first := wasm.WasmOutputBuffer.String()
			require.NotEmpty(t, first)

			for i := 0; i < 3; i++ {
				wasm.ResetOutputBuffers()
				require.NoError(t, ExecuteWithArgs(tc.args))
				assert.Equal(t, first, wasm.WasmOutputBuffer.String(), "run %d differs from run 1", i+2)
			}
		})
	}
}
