//go:build js && wasm

package megaport

import (
	"testing"

	"github.com/megaport/megaport-cli/internal/wasm"
	"github.com/stretchr/testify/assert"
)

// TestNATGatewayModuleRegistered verifies the nat-gateway module is wired into
// the WASM command tree (ESD-1284). Each command is invoked with --help so no
// network call is made; an unregistered command would instead surface an
// "unknown command" error. Cases are help-only by design so they share the
// global rootCmd without leaking flag state between subtests.
func TestNATGatewayModuleRegistered(t *testing.T) {
	cases := []struct {
		name string
		args []string
		// usage is the command path cobra echoes in the "Usage:" block; it
		// tracks the command structure rather than drift-prone help prose.
		usage string
	}{
		{"nat-gateway", []string{"megaport-cli", "nat-gateway", "--help"}, "megaport-cli nat-gateway"},
		{"list", []string{"megaport-cli", "nat-gateway", "list", "--help"}, "megaport-cli nat-gateway list"},
		{"get", []string{"megaport-cli", "nat-gateway", "get", "--help"}, "megaport-cli nat-gateway get"},
		{"list-sessions", []string{"megaport-cli", "nat-gateway", "list-sessions", "--help"}, "megaport-cli nat-gateway list-sessions"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wasm.ResetOutputBuffers()
			_ = ExecuteWithArgs(tc.args)
			out := wasm.GetCapturedOutput()
			assert.NotContains(t, out, "unknown command", "%v should be a registered command", tc.args)
			assert.Contains(t, out, tc.usage, "%v help should show its usage path", tc.args)
		})
	}
}
