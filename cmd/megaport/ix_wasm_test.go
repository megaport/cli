//go:build js && wasm

package megaport

import (
	"testing"

	"github.com/megaport/megaport-cli/internal/wasm"
	"github.com/stretchr/testify/assert"
)

// TestIXModuleRegistered verifies the ix module is wired into
// the WASM command tree (ESD-1285). Each command is invoked with --help so no
// network call is made; an unregistered command would instead surface an
// "unknown command" error.
func TestIXModuleRegistered(t *testing.T) {
	cases := []struct {
		name  string
		args  []string
		usage string
	}{
		{"ix", []string{"megaport-cli", "ix", "--help"}, "megaport-cli ix"},
		{"list", []string{"megaport-cli", "ix", "list", "--help"}, "megaport-cli ix list"},
		{"get", []string{"megaport-cli", "ix", "get", "--help"}, "megaport-cli ix get"},
		{"buy", []string{"megaport-cli", "ix", "buy", "--help"}, "megaport-cli ix buy"},
		{"update", []string{"megaport-cli", "ix", "update", "--help"}, "megaport-cli ix update"},
		{"delete", []string{"megaport-cli", "ix", "delete", "--help"}, "megaport-cli ix delete"},
		{"status", []string{"megaport-cli", "ix", "status", "--help"}, "megaport-cli ix status"},
		{"validate", []string{"megaport-cli", "ix", "validate", "--help"}, "megaport-cli ix validate"},
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
