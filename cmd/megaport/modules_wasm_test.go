//go:build js && wasm

package megaport

import (
	"testing"

	"github.com/megaport/megaport-cli/internal/wasm"
	"github.com/stretchr/testify/assert"
)

// TestReadOnlyModulesRegistered verifies the read-only status, topology, and
// product modules are wired into the WASM command tree (ESD-1283). Each is
// invoked with --help so no network call is made; an unregistered command would
// instead surface an "unknown command" error. Cases are help-only by design so
// they share the global rootCmd without leaking flag state between subtests.
func TestReadOnlyModulesRegistered(t *testing.T) {
	cases := []struct {
		name string
		args []string
		// usage must appear in the help output. Some rows include the "Usage:"
		// heading because an unknown subcommand prints its parent's help.
		usage string
	}{
		{"status", []string{"megaport-cli", "status", "--help"}, "megaport-cli status"},
		{"topology", []string{"megaport-cli", "topology", "--help"}, "megaport-cli topology"},
		{"product", []string{"megaport-cli", "product", "--help"}, "megaport-cli product"},
		{"product list", []string{"megaport-cli", "product", "list", "--help"}, "megaport-cli product list"},
		{"product get-type", []string{"megaport-cli", "product", "get-type", "--help"}, "megaport-cli product get-type"},
		{"product availability", []string{"megaport-cli", "product", "availability", "--help"}, "Usage:\n  megaport-cli product availability"},
		{"product pricing port", []string{"megaport-cli", "product", "pricing", "port", "--help"}, "Usage:\n  megaport-cli product pricing port"},
		{"product pricing vxc", []string{"megaport-cli", "product", "pricing", "vxc", "--help"}, "Usage:\n  megaport-cli product pricing vxc"},
		{"product pricing mcr", []string{"megaport-cli", "product", "pricing", "mcr", "--help"}, "Usage:\n  megaport-cli product pricing mcr"},
		{"product pricing mve", []string{"megaport-cli", "product", "pricing", "mve", "--help"}, "Usage:\n  megaport-cli product pricing mve"},
		{"product pricing ix", []string{"megaport-cli", "product", "pricing", "ix", "--help"}, "Usage:\n  megaport-cli product pricing ix"},
		{"product pricing nat-gateway", []string{"megaport-cli", "product", "pricing", "nat-gateway", "--help"}, "Usage:\n  megaport-cli product pricing nat-gateway"},
		{"product pricing ip-address", []string{"megaport-cli", "product", "pricing", "ip-address", "--help"}, "Usage:\n  megaport-cli product pricing ip-address"},
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

// TestAccountPartnerAdminModulesRegistered verifies the users, managed-account,
// and billing-market modules are wired into the WASM command tree (ESD-1287).
// Like the read-only check above, each case is help-only so no network call is
// made and subtests can share the global rootCmd without leaking flag state.
func TestAccountPartnerAdminModulesRegistered(t *testing.T) {
	cases := []struct {
		name string
		args []string
		// usage anchors on the "Usage:" block so a parent's "Example usage"
		// prose can't satisfy the match for a removed subcommand.
		usage string
	}{
		{"users", []string{"megaport-cli", "users", "--help"}, "Usage:\n  megaport-cli users"},
		{"users list", []string{"megaport-cli", "users", "list", "--help"}, "Usage:\n  megaport-cli users list"},
		{"managed-account", []string{"megaport-cli", "managed-account", "--help"}, "Usage:\n  megaport-cli managed-account"},
		{"managed-account list", []string{"megaport-cli", "managed-account", "list", "--help"}, "Usage:\n  megaport-cli managed-account list"},
		{"billing-market", []string{"megaport-cli", "billing-market", "--help"}, "Usage:\n  megaport-cli billing-market"},
		{"billing-market get", []string{"megaport-cli", "billing-market", "get", "--help"}, "Usage:\n  megaport-cli billing-market get"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wasm.ResetOutputBuffers()
			_ = ExecuteWithArgs(tc.args)
			out := wasm.GetCapturedOutput()
			// An unregistered command falls back to root help, which lacks
			// this command's Usage: block, so Contains is the real check.
			assert.Contains(t, out, tc.usage, "%v should be registered and show its usage path", tc.args)
		})
	}
}
