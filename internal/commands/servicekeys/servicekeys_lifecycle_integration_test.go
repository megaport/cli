//go:build integration && provisioning

package servicekeys

import (
	"context"
	crypto_rand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/testutil"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// integrationLocationID is the staging data center service key tests prefer for
// the throwaway port a key needs as its product. FindPortTestLocation falls back
// to another orderable location when it is out of capacity.
const integrationLocationID = 67

func uniqueSuffix(t *testing.T) string {
	t.Helper()
	buf := make([]byte, 4)
	_, err := crypto_rand.Read(buf)
	require.NoError(t, err, "failed to read crypto/rand entropy")
	return hex.EncodeToString(buf)
}

// findPortUIDByName looks up the port with the given name and returns its UID,
// with ok=false if no live (non-decommissioned) port matches. It returns the
// list error rather than aborting, so it is safe to call from a t.Cleanup
// callback, where a FailNow would mask the original failure.
func findPortUIDByName(client *megaport.Client, name string) (uid string, ok bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ports, err := client.PortService.ListPorts(ctx)
	if err != nil {
		return "", false, err
	}
	for _, p := range ports {
		if p == nil || p.Name != name || p.ProvisioningStatus == "DECOMMISSIONED" || p.ProvisioningStatus == "CANCELLED" {
			continue
		}
		return p.UID, true, nil
	}
	return "", false, nil
}

// provisionPortForServiceKey buys a 1G port on staging via the SDK and registers
// its teardown. A service key must reference an existing product (port), so the
// lifecycle test provisions a real one and deletes it in t.Cleanup even on failure.
func provisionPortForServiceKey(t *testing.T, client *megaport.Client) string {
	t.Helper()
	portName := fmt.Sprintf("CLI-Test-SvcKey-Port-%s", uniqueSuffix(t))

	// Safety net registered before BuyPort: if the buy partially succeeds (e.g.
	// the port is created but provisioning times out and BuyPort returns an
	// error), this still removes the port. It resolves the UID by the unique
	// name at cleanup time and is best-effort, so it is harmless when the port
	// was never created or is already gone.
	t.Cleanup(func() {
		uid, ok, err := findPortUIDByName(client, portName)
		if err != nil {
			t.Logf("cleanup: could not list ports to find %s: %v", portName, err)
			return
		}
		if !ok {
			return
		}
		delCtx, delCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer delCancel()
		if _, err := client.PortService.DeletePort(delCtx, &megaport.DeletePortRequest{
			PortID:    uid,
			DeleteNow: true,
		}); err != nil {
			t.Errorf("cleanup: failed to delete port %s (%s): %v", uid, portName, err)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()

	locationID := testutil.FindPortTestLocation(t, client, 1000, integrationLocationID)
	resp, err := client.PortService.BuyPort(ctx, &megaport.BuyPortRequest{
		Name:                  portName,
		Term:                  1,
		PortSpeed:             1000,
		LocationId:            locationID,
		MarketPlaceVisibility: false,
		WaitForProvision:      true,
		WaitForTime:           10 * time.Minute,
	})
	require.NoError(t, err, "failed to provision port for service key test")
	require.NotEmpty(t, resp.TechnicalServiceUIDs, "buy port returned no technical service UIDs")
	portUID := resp.TechnicalServiceUIDs[0]

	return portUID
}

// serviceKeyUIDForProduct finds the key just created against productUID by its
// unique description. CreateServiceKey prints the UID but does not return it, so
// the test recovers it from the SDK list filtered by product.
func serviceKeyUIDForProduct(t *testing.T, client *megaport.Client, productUID, description string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	resp, err := client.ServiceKeyService.ListServiceKeys(ctx, &megaport.ListServiceKeysRequest{ProductUID: &productUID})
	require.NoError(t, err, "SDK ListServiceKeys failed")
	require.NotNil(t, resp)
	for _, k := range resp.ServiceKeys {
		if k != nil && k.Description == description {
			require.NotEmpty(t, k.Key, "service key UID should not be empty")
			return k.Key
		}
	}
	t.Fatalf("created service key with description %q not found for product %s", description, productUID)
	return ""
}

func serviceKeyFromSDK(t *testing.T, client *megaport.Client, keyUID string) *megaport.ServiceKey {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	sk, err := client.ServiceKeyService.GetServiceKey(ctx, keyUID)
	require.NoErrorf(t, err, "SDK GetServiceKey failed for %s", keyUID)
	require.NotNil(t, sk)
	return sk
}

// createServiceKeyViaCLI drives the CLI CreateServiceKey action against portUID
// with the given description plus any extra flags, then recovers the new key's
// UID via the SDK. It centralizes the set-flags-create-recover boilerplate the
// lifecycle tests share so each test body reads as its assertions.
func createServiceKeyViaCLI(t *testing.T, client *megaport.Client, portUID, description string, extra map[string]string) string {
	t.Helper()
	cmd := newCreateServiceKeyCmdLifecycle()
	require.NoError(t, cmd.Flags().Set("product-uid", portUID))
	require.NoError(t, cmd.Flags().Set("description", description))
	for name, value := range extra {
		require.NoErrorf(t, cmd.Flags().Set(name, value), "failed to set --%s", name)
	}
	require.NoError(t, CreateServiceKey(cmd, nil, true), "CreateServiceKey failed")
	return serviceKeyUIDForProduct(t, client, portUID, description)
}

func newCreateServiceKeyCmdLifecycle() *cobra.Command {
	cmd := &cobra.Command{Use: "create"}
	cmd.Flags().String("product-uid", "", "")
	cmd.Flags().Int("product-id", 0, "")
	cmd.Flags().Bool("single-use", false, "")
	cmd.Flags().String("description", "", "")
	cmd.Flags().String("start-date", "", "")
	cmd.Flags().String("end-date", "", "")
	cmd.Flags().Int("max-speed", 0, "")
	cmd.Flags().Bool("active", false, "")
	cmd.Flags().Bool("pre-approved", false, "")
	cmd.Flags().Int("vlan", 0, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	cmd.Flags().Bool("interactive", false, "")
	return cmd
}

func newUpdateServiceKeyCmdLifecycle() *cobra.Command {
	cmd := &cobra.Command{Use: "update"}
	cmd.Flags().String("product-uid", "", "")
	cmd.Flags().Int("product-id", 0, "")
	cmd.Flags().Bool("single-use", false, "")
	cmd.Flags().String("description", "", "")
	cmd.Flags().Bool("active", false, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	cmd.Flags().Bool("interactive", false, "")
	return cmd
}

// TestIntegration_ServiceKeyLifecycle provisions a port, then exercises the full
// create/get/update path of the service key CLI actions against it. Only the port
// is torn down in t.Cleanup; service keys have no delete API and are left attached
// to the (deleted) port. This test carries the extra `provisioning` build tag so
// the nightly read-only job (which builds only `-tags integration`) never runs it;
// it runs in the manual provisioning job.
func TestIntegration_ServiceKeyLifecycle(t *testing.T) {
	testutil.RequireStagingForProvisioning(t)
	client := testutil.SetupIntegrationClient(t)
	defer testutil.LoginWithClient(t, client)()

	portUID := provisionPortForServiceKey(t, client)
	t.Logf("provisioned port %s for service key lifecycle", portUID)

	description := fmt.Sprintf("CLI-Test-Key-%s", uniqueSuffix(t))

	createCmd := newCreateServiceKeyCmdLifecycle()
	require.NoError(t, createCmd.Flags().Set("product-uid", portUID))
	require.NoError(t, createCmd.Flags().Set("description", description))
	require.NoError(t, createCmd.Flags().Set("max-speed", "1000"))

	require.NoError(t, CreateServiceKey(createCmd, nil, true), "CreateServiceKey failed")

	keyUID := serviceKeyUIDForProduct(t, client, portUID, description)
	t.Logf("created service key %s", keyUID)

	getCmd := &cobra.Command{Use: "get"}
	getOut := output.CaptureOutput(func() {
		require.NoError(t, GetServiceKey(getCmd, []string{keyUID}, true, "json"))
	})

	var got []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(getOut), &got), "get output should be valid JSON")
	require.NotEmpty(t, got)
	assert.Equal(t, keyUID, got[0]["key_uid"])
	assert.Equal(t, portUID, got[0]["product_uid"])
	assert.Equal(t, description, got[0]["description"])

	updateCmd := newUpdateServiceKeyCmdLifecycle()
	require.NoError(t, updateCmd.Flags().Set("active", "true"))
	require.NoError(t, UpdateServiceKey(updateCmd, []string{keyUID}, true), "UpdateServiceKey failed")

	updated := serviceKeyFromSDK(t, client, keyUID)
	assert.True(t, updated.Active, "service key should be active after update")
}

// TestIntegration_ServiceKeyVLANScopedLifecycle provisions a port and creates a
// single-use, VLAN-scoped service key against it, then asserts via the SDK that
// the VLAN and single-use scope round-trip. Like TestIntegration_ServiceKeyLifecycle
// it tears down only the port: service keys have no delete API, so the key is left
// attached to the (deleted) port.
func TestIntegration_ServiceKeyVLANScopedLifecycle(t *testing.T) {
	testutil.RequireStagingForProvisioning(t)
	client := testutil.SetupIntegrationClient(t)
	defer testutil.LoginWithClient(t, client)()

	portUID := provisionPortForServiceKey(t, client)
	t.Logf("provisioned port %s for VLAN-scoped service key", portUID)

	const vlan = 100
	description := fmt.Sprintf("CLI-Test-VLANKey-%s", uniqueSuffix(t))

	keyUID := createServiceKeyViaCLI(t, client, portUID, description, map[string]string{
		"single-use": "true",
		"vlan":       strconv.Itoa(vlan),
		"max-speed":  "1000",
	})
	t.Logf("created VLAN-scoped service key %s", keyUID)

	sk := serviceKeyFromSDK(t, client, keyUID)
	assert.Equal(t, vlan, sk.VLAN, "VLAN should round-trip via SDK")
	assert.True(t, sk.SingleUse, "service key should be single-use")
	assert.Equal(t, portUID, sk.ProductUID, "service key should be scoped to the provisioned port")
}

// TestIntegration_ServiceKeyUpdateMatrix creates a service key, confirms create
// round-trips its fields, normalizes it to a known active+single-use baseline,
// then runs single-field updates and asserts via the SDK that the unspecified
// active/single-use sibling survives. This is the regression guard for ESD-1417,
// where an unset bool reset active/single-use. max-speed is also checked as a
// preserved sibling. description is NOT preserved by an update: the update request
// carries no description field, so any update clears it. That is asserted as a
// documented limitation rather than treated as a preservation invariant.
func TestIntegration_ServiceKeyUpdateMatrix(t *testing.T) {
	testutil.RequireStagingForProvisioning(t)
	client := testutil.SetupIntegrationClient(t)
	defer testutil.LoginWithClient(t, client)()

	portUID := provisionPortForServiceKey(t, client)
	t.Logf("provisioned port %s for service key update matrix", portUID)

	const (
		vlan = 200
		// Full 1G port speed: a guaranteed-valid rate the API won't normalize,
		// while still non-zero so a reset-to-zero regression is caught.
		maxSpeed = 1000
	)
	description := fmt.Sprintf("CLI-Test-MatrixKey-%s", uniqueSuffix(t))

	keyUID := createServiceKeyViaCLI(t, client, portUID, description, map[string]string{
		"single-use": "true",
		"vlan":       strconv.Itoa(vlan),
		"max-speed":  strconv.Itoa(maxSpeed),
	})
	t.Logf("created service key %s for update matrix", keyUID)

	// Create must round-trip the fields we set. active is not asserted here:
	// create omits it (json omitempty) and staging defaults a new key to
	// active=true, so the baseline below normalizes active explicitly.
	created := serviceKeyFromSDK(t, client, keyUID)
	require.True(t, created.SingleUse, "create must round-trip single-use")
	require.Equal(t, vlan, created.VLAN, "create must round-trip vlan")
	require.Equal(t, maxSpeed, created.MaxSpeed, "create must round-trip max-speed")
	require.Equal(t, description, created.Description, "create must round-trip description")

	// updateField runs a single-flag update and returns the re-fetched key.
	updateField := func(flag, value string) *megaport.ServiceKey {
		t.Helper()
		cmd := newUpdateServiceKeyCmdLifecycle()
		require.NoError(t, cmd.Flags().Set(flag, value))
		require.NoErrorf(t, UpdateServiceKey(cmd, []string{keyUID}, true), "%s-only update failed", flag)
		return serviceKeyFromSDK(t, client, keyUID)
	}

	// Normalize to active=true, single-use=true regardless of the API's
	// create-time defaults (staging defaults a new key to active=true). Each
	// guard below then starts with its sibling at a non-zero value, so an
	// ESD-1417 reset-to-false regression cannot pass trivially.
	setup := newUpdateServiceKeyCmdLifecycle()
	require.NoError(t, setup.Flags().Set("active", "true"))
	require.NoError(t, setup.Flags().Set("single-use", "true"))
	require.NoError(t, UpdateServiceKey(setup, []string{keyUID}, true), "baseline normalize update failed")
	base := serviceKeyFromSDK(t, client, keyUID)
	require.True(t, base.Active, "baseline: active must be true")
	require.True(t, base.SingleUse, "baseline: single-use must be true")
	require.Equal(t, maxSpeed, base.MaxSpeed, "baseline: max-speed must survive the update")
	// Known limitation: an update clears the description. The update request has
	// no description field (and the CLI update has no --description flag), so any
	// update wipes the create-time description. Pinned here so the behavior is
	// captured; round-tripping description on update is a follow-up.
	assert.Empty(t, base.Description, "an update is expected to clear the description (known limitation)")

	// Guard 1: update active only (true -> false). single-use is left unspecified
	// and must survive at true; the ESD-1417 reset-to-false bug would clear it.
	afterActive := updateField("active", "false")
	assert.False(t, afterActive.Active, "active should be false after the active-only update")
	assert.True(t, afterActive.SingleUse, "single-use must survive an active-only update (ESD-1417)")

	// Restore active=true so guard 2 runs with the active sibling non-zero. This
	// re-run of the active update doubles as a second single-use-survives check.
	restored := updateField("active", "true")
	require.True(t, restored.Active, "active should be true after restore")
	require.True(t, restored.SingleUse, "single-use must survive the active restore (ESD-1417)")

	// Guard 2: update single-use only (true -> false). active is left unspecified
	// and must survive at true (the other ESD-1417 direction).
	afterSingleUse := updateField("single-use", "false")
	assert.False(t, afterSingleUse.SingleUse, "single-use should be false after the single-use-only update")
	assert.True(t, afterSingleUse.Active, "active must survive a single-use-only update (ESD-1417)")
	// VLAN intentionally not asserted here: dropping single-use can legitimately
	// clear the VLAN scope, so it is not a sibling-preservation invariant.
}
