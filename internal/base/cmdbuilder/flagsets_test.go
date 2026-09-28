package cmdbuilder

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertFlagExists verifies a flag exists on the command with the expected default value.
func assertFlagExists(t *testing.T, cmd *cobra.Command, name, expectedDefault string) {
	t.Helper()
	f := cmd.Flags().Lookup(name)
	require.NotNil(t, f, "flag %q should exist", name)
	assert.Equal(t, expectedDefault, f.DefValue, "flag %q default", name)
}

// assertFlagType verifies a flag exists with the expected type.
func assertFlagType(t *testing.T, cmd *cobra.Command, name, expectedType string) {
	t.Helper()
	f := cmd.Flags().Lookup(name)
	require.NotNil(t, f, "flag %q should exist", name)
	assert.Equal(t, expectedType, f.Value.Type(), "flag %q type", name)
}

func TestWithWatchFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithWatchFlags().Build()

	assertFlagExists(t, cmd, "watch", "false")
	assertFlagType(t, cmd, "watch", "bool")
	// Verify shorthand
	f := cmd.Flags().Lookup("watch")
	assert.Equal(t, "w", f.Shorthand)

	assertFlagExists(t, cmd, "interval", "5s")
	assertFlagType(t, cmd, "interval", "duration")
}

func TestWithStandardInputFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithStandardInputFlags().Build()

	assertFlagExists(t, cmd, "interactive", "false")
	assertFlagType(t, cmd, "interactive", "bool")
	f := cmd.Flags().Lookup("interactive")
	assert.Equal(t, "i", f.Shorthand)

	assertFlagExists(t, cmd, "json", "")
	assertFlagType(t, cmd, "json", "string")

	assertFlagExists(t, cmd, "json-file", "")
	assertFlagType(t, cmd, "json-file", "string")
}

func TestWithDeleteFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithDeleteFlags().Build()

	assertFlagExists(t, cmd, "force", "false")
	assertFlagType(t, cmd, "force", "bool")
	f := cmd.Flags().Lookup("force")
	assert.Equal(t, "f", f.Shorthand)

	assertFlagExists(t, cmd, "now", "false")
	assertFlagType(t, cmd, "now", "bool")
}

func TestWithSafeDeleteFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithSafeDeleteFlags().Build()

	// Should include all delete flags
	assertFlagExists(t, cmd, "force", "false")
	assertFlagExists(t, cmd, "now", "false")

	// Plus safe-delete
	assertFlagExists(t, cmd, "safe-delete", "false")
	assertFlagType(t, cmd, "safe-delete", "bool")
}

func TestWithImmediateDeleteFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithImmediateDeleteFlags().Build()

	assertFlagExists(t, cmd, "force", "false")
	assertFlagType(t, cmd, "force", "bool")

	// No --now for resources that only support immediate deletion.
	assert.Nil(t, cmd.Flags().Lookup("now"))
}

func TestWithBuyConfirmFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithBuyConfirmFlags().Build()

	assertFlagExists(t, cmd, "yes", "false")
	assertFlagType(t, cmd, "yes", "bool")
	f := cmd.Flags().Lookup("yes")
	assert.Equal(t, "y", f.Shorthand)
}

func TestWithNoWaitFlag(t *testing.T) {
	cmd := NewCommand("test", "test").WithNoWaitFlag().Build()

	assertFlagExists(t, cmd, "no-wait", "false")
	assertFlagType(t, cmd, "no-wait", "bool")
}

func TestWithDateRangeFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithDateRangeFlags().Build()

	assertFlagExists(t, cmd, "start-date", "")
	assertFlagType(t, cmd, "start-date", "string")

	assertFlagExists(t, cmd, "end-date", "")
	assertFlagType(t, cmd, "end-date", "string")
}

func TestWithResourceTagFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithResourceTagFlags().Build()

	assertFlagExists(t, cmd, "resource-tags", "")
	assertFlagType(t, cmd, "resource-tags", "string")

	assertFlagExists(t, cmd, "resource-tags-file", "")
	assertFlagType(t, cmd, "resource-tags-file", "string")
}

func TestWithInteractiveFlag(t *testing.T) {
	cmd := NewCommand("test", "test").WithInteractiveFlag().Build()

	assertFlagExists(t, cmd, "interactive", "false")
	assertFlagType(t, cmd, "interactive", "bool")
	f := cmd.Flags().Lookup("interactive")
	assert.Equal(t, "i", f.Shorthand)
}

func TestWithJSONConfigFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithJSONConfigFlags().Build()

	assertFlagExists(t, cmd, "json", "")
	assertFlagType(t, cmd, "json", "string")

	assertFlagExists(t, cmd, "json-file", "")
	assertFlagType(t, cmd, "json-file", "string")
}

func TestWithMCRIPSecAddOnFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithMCRIPSecAddOnFlags().Build()

	assertFlagExists(t, cmd, "tunnel-count", "0")
	assertFlagType(t, cmd, "tunnel-count", "int")
}

func TestWithMCRUpdateIPSecAddOnFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithMCRUpdateIPSecAddOnFlags().Build()

	assertFlagExists(t, cmd, "tunnel-count", "0")
	assertFlagType(t, cmd, "tunnel-count", "int")
}

// Resource-specific flagset tests

func TestWithPortCreationFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithPortCreationFlags().Build()

	expectedFlags := []string{
		"name", "term", "marketplace-visibility", "diversity-zone",
		"cost-centre", "port-speed", "location-id", "promo-code",
		"resource-tags", "resource-tags-file",
	}

	for _, flag := range expectedFlags {
		f := cmd.Flags().Lookup(flag)
		assert.NotNil(t, f, "port creation flag %q should exist", flag)
	}
}

func TestWithPortFilterFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithPortFilterFlags().Build()

	expectedFlags := map[string]string{
		"location-id":      "int",
		"port-speed":       "int",
		"port-name":        "string",
		"include-inactive": "bool",
	}

	for name, expectedType := range expectedFlags {
		f := cmd.Flags().Lookup(name)
		assert.NotNil(t, f, "port filter flag %q should exist", name)
		if f != nil {
			assert.Equal(t, expectedType, f.Value.Type(), "port filter flag %q type", name)
		}
	}
}

func TestWithMCRCreateFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithMCRCreateFlags().Build()

	expectedFlags := []string{
		"name", "term", "port-speed", "location-id", "mcr-asn",
		"diversity-zone", "cost-centre", "marketplace-visibility",
		"promo-code", "resource-tags", "resource-tags-file",
	}

	for _, flag := range expectedFlags {
		f := cmd.Flags().Lookup(flag)
		assert.NotNil(t, f, "MCR creation flag %q should exist", flag)
	}

	expectedTypes := map[string]string{
		"term":                   "int",
		"port-speed":             "int",
		"location-id":            "int",
		"mcr-asn":                "int",
		"marketplace-visibility": "bool",
	}
	for name, expectedType := range expectedTypes {
		f := cmd.Flags().Lookup(name)
		if f != nil {
			assert.Equal(t, expectedType, f.Value.Type(), "MCR creation flag %q type", name)
		}
	}
}

func TestWithVXCCreateFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithVXCCreateFlags().Build()

	expectedFlags := []string{
		"name", "rate-limit", "term", "cost-centre",
		"a-end-uid", "b-end-uid", "a-end-vlan", "b-end-vlan",
		"a-end-inner-vlan", "b-end-inner-vlan",
		"a-end-partner-config", "b-end-partner-config",
		"a-end-vnic-index", "b-end-vnic-index",
		"promo-code", "service-key",
		"resource-tags", "resource-tags-file",
	}

	for _, flag := range expectedFlags {
		f := cmd.Flags().Lookup(flag)
		assert.NotNil(t, f, "VXC creation flag %q should exist", flag)
	}
}

func TestWithMVECreateFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithMVECreateFlags().Build()

	expectedFlags := []string{
		"name", "term", "location-id", "vendor-config", "vnics",
		"diversity-zone", "promo-code", "cost-centre",
		"resource-tags", "resource-tags-file",
	}

	for _, flag := range expectedFlags {
		f := cmd.Flags().Lookup(flag)
		assert.NotNil(t, f, "MVE creation flag %q should exist", flag)
	}
}

func TestWithPortLAGFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithPortLAGFlags().Build()

	// Should have all port creation flags plus lag-count
	assertFlagExists(t, cmd, "lag-count", "0")
	assertFlagType(t, cmd, "lag-count", "int")
	// Spot-check port creation flags are present too
	assert.NotNil(t, cmd.Flags().Lookup("name"))
	assert.NotNil(t, cmd.Flags().Lookup("port-speed"))
}

func TestWithPortUpdateFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithPortUpdateFlags().Build()

	expectedFlags := []string{"name", "marketplace-visibility", "cost-centre", "term"}
	for _, flag := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(flag), "port update flag %q should exist", flag)
	}
}

func TestWithMCRUpdateFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithMCRUpdateFlags().Build()

	expectedFlags := map[string]string{
		"name":                   "string",
		"cost-centre":            "string",
		"marketplace-visibility": "bool",
		"term":                   "int",
	}
	for name, expectedType := range expectedFlags {
		f := cmd.Flags().Lookup(name)
		assert.NotNil(t, f, "MCR update flag %q should exist", name)
		if f != nil {
			assert.Equal(t, expectedType, f.Value.Type(), "MCR update flag %q type", name)
		}
	}

	for _, flag := range []string{"name", "cost-centre", "marketplace-visibility", "term", "mcr-asn"} {
		assert.NotNil(t, cmd.Flags().Lookup(flag), "MCR update flag %q should exist", flag)
	}

	for _, intFlag := range []string{"term", "mcr-asn"} {
		if f := cmd.Flags().Lookup(intFlag); f != nil {
			assert.Equal(t, "int", f.Value.Type(), "%s should be an int flag", intFlag)
		}
	}
}

func TestWithMCRFilterFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithMCRFilterFlags().Build()

	expectedFlags := map[string]string{
		"location-id":      "int",
		"port-speed":       "int",
		"name":             "string",
		"include-inactive": "bool",
	}

	for name, expectedType := range expectedFlags {
		f := cmd.Flags().Lookup(name)
		assert.NotNil(t, f, "MCR filter flag %q should exist", name)
		if f != nil {
			assert.Equal(t, expectedType, f.Value.Type(), "MCR filter flag %q type", name)
		}
	}
}

func TestWithVXCUpdateFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithVXCUpdateFlags().Build()

	expectedFlags := []string{
		"name", "rate-limit", "term", "cost-centre",
		"a-end-uid", "b-end-uid", "a-end-vlan", "b-end-vlan",
		"shutdown", "is-approved", "a-vnic-index", "b-vnic-index",
	}
	for _, flag := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(flag), "VXC update flag %q should exist", flag)
	}
}

// TestVXCUpdateFlagNamesMatchRegistered guards against the update gate's flag
// list drifting from the flags WithVXCUpdateFlags actually registers.
func TestVXCUpdateFlagNamesMatchRegistered(t *testing.T) {
	cmd := NewCommand("test", "test").WithVXCUpdateFlags().Build()

	var registered []string
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		registered = append(registered, f.Name)
	})

	assert.ElementsMatch(t, VXCUpdateFlagNames, registered,
		"VXCUpdateFlagNames must match exactly the flags WithVXCUpdateFlags registers")
}

func TestWithVXCFilterFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithVXCFilterFlags().Build()

	expectedFlags := map[string]string{
		"name":             "string",
		"name-contains":    "string",
		"rate-limit":       "int",
		"a-end-uid":        "string",
		"b-end-uid":        "string",
		"status":           "string",
		"include-inactive": "bool",
	}

	for name, expectedType := range expectedFlags {
		f := cmd.Flags().Lookup(name)
		assert.NotNil(t, f, "VXC filter flag %q should exist", name)
		if f != nil {
			assert.Equal(t, expectedType, f.Value.Type(), "VXC filter flag %q type", name)
		}
	}
}

func TestWithMVEUpdateFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithMVEUpdateFlags().Build()

	expectedFlags := []string{"name", "cost-centre", "term", "vnics"}
	for _, flag := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(flag), "MVE update flag %q should exist", flag)
	}
}

func TestWithMVEFilterFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithMVEFilterFlags().Build()

	expectedFlags := map[string]string{
		"location-id":      "int",
		"vendor":           "string",
		"name":             "string",
		"include-inactive": "bool",
	}

	for name, expectedType := range expectedFlags {
		f := cmd.Flags().Lookup(name)
		assert.NotNil(t, f, "MVE filter flag %q should exist", name)
		if f != nil {
			assert.Equal(t, expectedType, f.Value.Type(), "MVE filter flag %q type", name)
		}
	}
}

func TestWithMVEImageFilterFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithMVEImageFilterFlags().Build()

	expectedFlags := map[string]string{
		"vendor":        "string",
		"product-code":  "string",
		"id":            "int",
		"version":       "string",
		"release-image": "bool",
	}

	for name, expectedType := range expectedFlags {
		f := cmd.Flags().Lookup(name)
		assert.NotNil(t, f, "MVE image filter flag %q should exist", name)
		if f != nil {
			assert.Equal(t, expectedType, f.Value.Type(), "MVE image filter flag %q type", name)
		}
	}
}

func TestWithMCRPrefixFilterListFlags(t *testing.T) {
	cmd := NewCommand("test", "test").WithMCRPrefixFilterListFlags().Build()

	expectedFlags := []string{"description", "address-family", "entries"}
	for _, flag := range expectedFlags {
		f := cmd.Flags().Lookup(flag)
		assert.NotNil(t, f, "MCR prefix filter list flag %q should exist", flag)
		if f != nil {
			assert.Equal(t, "string", f.Value.Type())
		}
	}
}

// Test that composite flagsets can be combined without conflicts

func TestFlagsetCombinations(t *testing.T) {
	t.Run("standard input with delete flags", func(t *testing.T) {
		cmd := NewCommand("test", "test").
			WithStandardInputFlags().
			WithDeleteFlags().
			Build()

		assertFlagExists(t, cmd, "interactive", "false")
		assertFlagExists(t, cmd, "json", "")
		assertFlagExists(t, cmd, "json-file", "")
		assertFlagExists(t, cmd, "force", "false")
		assertFlagExists(t, cmd, "now", "false")
	})

	t.Run("standard input with buy confirm and no-wait", func(t *testing.T) {
		cmd := NewCommand("test", "test").
			WithStandardInputFlags().
			WithBuyConfirmFlags().
			WithNoWaitFlag().
			Build()

		assertFlagExists(t, cmd, "interactive", "false")
		assertFlagExists(t, cmd, "json", "")
		assertFlagExists(t, cmd, "yes", "false")
		assertFlagExists(t, cmd, "no-wait", "false")
	})

	t.Run("watch with date range", func(t *testing.T) {
		cmd := NewCommand("test", "test").
			WithWatchFlags().
			WithDateRangeFlags().
			Build()

		assertFlagExists(t, cmd, "watch", "false")
		assertFlagExists(t, cmd, "interval", "5s")
		assertFlagExists(t, cmd, "start-date", "")
		assertFlagExists(t, cmd, "end-date", "")
	})
}
