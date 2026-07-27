//go:build integration && provisioning

package ports

import (
	"context"
	crypto_rand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/megaport/megaport-cli/internal/testutil"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// integrationLocationID is the staging data center port lifecycle tests prefer.
// ID 67 is the canonical example location across the CLI's README, example flag
// strings, and the previous binary-invocation integration suite, and it has
// historically advertised the 1G/10G port speeds these tests exercise. It is
// only a preference: each test resolves its location through
// testutil.FindPortTestLocation, which falls back to another active location (or
// skips) if 67 ever stops advertising the speed under test.
const integrationLocationID = 67

// These tests use t.Parallel(); see testutil.RequireSharedIntegrationClient
// for why a process-wide sync.Once-guarded login function is used here
// instead of the per-test save/restore pattern in testutil.LoginWithClient.
//
// State assertions go through the SDK directly (testutil.SharedIntegrationClient)
// rather than via output.CaptureOutput on GetPort/ListPorts. CaptureOutput
// swaps the global os.Stdout for a tmpfile while it runs; with t.Parallel(),
// concurrent action goroutines (especially their spinner goroutines, which
// write asynchronously) end up writing into another test's tmpfile or into a
// just-closed file descriptor. The result is polluted or empty captures.
// Side-effecting actions (BuyPort, BuyLAGPort, UpdatePort, DeletePort) still
// run end-to-end through the CLI code paths; we only skip capture and read
// the API state directly for verification.

// integrationBuyResponses lets parallel tests retrieve the BuyPortResponse
// for their port without scraping stdout. The init() hook below wraps
// buyPortFunc so the SDK response is stored under the request name; tests
// read the UID back from this map.
var integrationBuyResponses sync.Map // key: request.Name (string), value: *megaport.BuyPortResponse

func init() {
	base := buyPortFunc
	buyPortFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error) {
		resp, err := base(ctx, client, req)
		if err == nil && resp != nil && req != nil && req.Name != "" {
			integrationBuyResponses.Store(req.Name, resp)
		}
		return resp, err
	}
}

func generateUniqueID(t *testing.T) string {
	t.Helper()
	buf := make([]byte, 4)
	_, err := crypto_rand.Read(buf)
	require.NoError(t, err, "failed to read crypto/rand entropy")
	return hex.EncodeToString(buf)
}

func newBuyPortCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "buy"}
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().Bool("no-wait", false, "")
	cmd.Flags().Bool("yes", false, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().Int("term", 0, "")
	cmd.Flags().Int("port-speed", 0, "")
	cmd.Flags().Int("location-id", 0, "")
	cmd.Flags().Bool("marketplace-visibility", false, "")
	cmd.Flags().String("diversity-zone", "", "")
	cmd.Flags().String("cost-centre", "", "")
	cmd.Flags().String("promo-code", "", "")
	cmd.Flags().String("resource-tags", "", "")
	cmd.Flags().Bool("cost-confirm", true, "")
	return cmd
}

func newBuyLAGPortCmd() *cobra.Command {
	cmd := newBuyPortCmd()
	cmd.Flags().Int("lag-count", 0, "")
	return cmd
}

func newDeletePortCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "delete"}
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().Bool("now", false, "")
	cmd.Flags().Bool("safe-delete", false, "")
	return cmd
}

func newUpdatePortTagsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "update-tags"}
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	cmd.Flags().String("tags", "", "")
	cmd.Flags().String("tags-file", "", "")
	cmd.Flags().String("resource-tags", "", "")
	return cmd
}

// cleanupStatusTimeout bounds how long registerPortCleanup will poll the SDK
// for the port to enter DECOMMISSIONING/DECOMMISSIONED after DeletePort.
// DeletePort only submits the cancellation request; the API can take a few
// seconds to reflect the new provisioning_status. Sixty seconds is well
// above observed transitions on staging without making a stuck cleanup drag
// the test run out by minutes.
const (
	cleanupStatusTimeout  = 60 * time.Second
	cleanupStatusInterval = 2 * time.Second
)

// registerPortCleanup schedules a best-effort delete of the given port. The
// cleanup runs even when the test fails, ensuring no orphaned resources on
// staging. DeletePort writes its progress to the real stdout (interleaved
// with other parallel tests' cleanups, which is harmless). The post-delete
// status check polls the SDK directly to avoid CaptureOutput's stdout swap.
// The package-level login function installed by
// testutil.RequireSharedIntegrationClient remains active for the cleanup
// callback (no per-test restore happens).
func registerPortCleanup(t *testing.T, uid string) {
	t.Helper()
	t.Cleanup(func() {
		delCmd := newDeletePortCmd()
		require.NoError(t, delCmd.Flags().Set("now", "true"))
		require.NoError(t, delCmd.Flags().Set("force", "true"))

		if err := DeletePort(delCmd, []string{uid}, true); err != nil {
			t.Errorf("cleanup: failed to delete port %s: %v", uid, err)
			return
		}

		client := testutil.SharedIntegrationClient(t)
		deadline := time.Now().Add(cleanupStatusTimeout)
		var lastStatus string
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			port, err := client.PortService.GetPort(ctx, uid)
			cancel()
			if err != nil {
				t.Logf("cleanup: SDK GetPort after delete returned %v (port may already be gone)", err)
				return
			}
			lastStatus = port.ProvisioningStatus
			if strings.Contains(lastStatus, "DECOMMISSIONING") || strings.Contains(lastStatus, "DECOMMISSIONED") {
				return
			}
			if time.Now().After(deadline) {
				t.Errorf("expected port %s to reach DECOMMISSIONING or DECOMMISSIONED within %s, last status %q", uid, cleanupStatusTimeout, lastStatus)
				return
			}
			time.Sleep(cleanupStatusInterval)
		}
	})
}

// portFromSDK reads the port via the shared SDK client. Used for parallel-safe
// state assertions instead of scraping GetPort's stdout output.
func portFromSDK(t *testing.T, uid string) *megaport.Port {
	t.Helper()
	client := testutil.SharedIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	port, err := client.PortService.GetPort(ctx, uid)
	require.NoErrorf(t, err, "SDK GetPort failed for %s", uid)
	require.NotNilf(t, port, "SDK GetPort returned nil for %s", uid)
	return port
}

// portsBySDKNameFilter lists all ports via the shared SDK client and returns
// those whose name contains the given substring (case-insensitive, matching
// the ListPorts action's client-side filter). Used for parallel-safe
// list assertions.
func portsBySDKNameFilter(t *testing.T, nameSubstring string) []*megaport.Port {
	t.Helper()
	client := testutil.SharedIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	all, err := client.PortService.ListPorts(ctx)
	require.NoError(t, err, "SDK ListPorts failed")
	needle := strings.ToLower(nameSubstring)
	var out []*megaport.Port
	for _, p := range all {
		if p == nil {
			continue
		}
		if strings.Contains(strings.ToLower(p.Name), needle) {
			out = append(out, p)
		}
	}
	return out
}

// runBuyPort and runBuyLAGPort intentionally do not wrap BuyPort/BuyLAGPort
// in output.CaptureOutput. Both actions block until LIVE
// (WaitForProvision=true), which can take tens of seconds on staging, and
// CaptureOutput would hold output.stdoutMu for the entire wait — defeating
// t.Parallel(). Their spinner goroutines also write asynchronously, which
// races with any concurrent CaptureOutput's stdout swap. The UID is
// recovered from the response recorded by the init() hook on buyPortFunc
// instead of scraping stdout.
//
// portName is passed explicitly (rather than read from --name) so that JSON
// buys, which encode the name inside the payload rather than on a cobra
// flag, work the same way as flag-driven buys. It must match the Name in
// the final BuyPortRequest the action sends to the SDK.
//
// Cleanup is registered here — immediately after the buy succeeds and before
// uidFromBuyResponse asserts on the response — so that any created resources
// are deleted even if UID extraction subsequently fails.
func runBuyPort(t *testing.T, cmd *cobra.Command, portName string) string {
	t.Helper()
	require.NotEmpty(t, portName, "portName must be provided to runBuyPort")
	require.NoErrorf(t, BuyPort(cmd, nil, true), "BuyPort failed for %q", portName)
	registerBuyCleanups(t, portName)
	return uidFromBuyResponse(t, portName)
}

func runBuyLAGPort(t *testing.T, cmd *cobra.Command, portName string) string {
	t.Helper()
	require.NotEmpty(t, portName, "portName must be provided to runBuyLAGPort")
	require.NoErrorf(t, BuyLAGPort(cmd, nil, true), "BuyLAGPort failed for %q", portName)
	registerBuyCleanups(t, portName)
	return uidFromBuyResponse(t, portName)
}

// registerBuyCleanups registers a cleanup for every UID in the buy response
// stored under portName. It is called immediately after BuyPort/BuyLAGPort
// returns so that created resources are cleaned up even if uidFromBuyResponse
// subsequently fails (e.g. TechnicalServiceUIDs is unexpectedly empty).
func registerBuyCleanups(t *testing.T, portName string) {
	t.Helper()
	v, ok := integrationBuyResponses.Load(portName)
	if !ok {
		return
	}
	resp, ok := v.(*megaport.BuyPortResponse)
	if !ok {
		return
	}
	for _, uid := range resp.TechnicalServiceUIDs {
		if uid != "" {
			registerPortCleanup(t, uid)
		}
	}
}

// uidFromBuyResponse returns the first technical service UID recorded for the
// given port name. Tests rely on init()'s buyPortFunc wrapper to populate
// integrationBuyResponses with the SDK response.
func uidFromBuyResponse(t *testing.T, name string) string {
	t.Helper()
	v, ok := integrationBuyResponses.Load(name)
	require.Truef(t, ok, "no buy response recorded for port %q", name)
	resp, ok := v.(*megaport.BuyPortResponse)
	require.Truef(t, ok, "buy response for %q has unexpected type %T", name, v)
	require.NotEmptyf(t, resp.TechnicalServiceUIDs, "buy response for %q has no technical service UIDs", name)
	return resp.TechnicalServiceUIDs[0]
}

// runUpdatePortName and runUpdatePortWithFlag also skip CaptureOutput.
// UpdatePort blocks on WaitForUpdate=true and uses spinners. The test only
// needs to know the call succeeded; no stdout scraping is required.
func runUpdatePortName(t *testing.T, uid, newName string) {
	t.Helper()
	cmd := newUpdatePortCmd()
	require.NoError(t, cmd.Flags().Set("name", newName))
	require.NoErrorf(t, UpdatePort(cmd, []string{uid}, true), "UpdatePort failed for %s", uid)
}

func runUpdatePortWithFlag(t *testing.T, uid, flagName, flagValue string) {
	t.Helper()
	cmd := newUpdatePortCmd()
	require.NoError(t, cmd.Flags().Set(flagName, flagValue))
	require.NoErrorf(t, UpdatePort(cmd, []string{uid}, true), "UpdatePort failed for %s", uid)
}

// portTagsFromSDK reads a port's resource tags via the shared SDK client. This
// is the same call the list-tags command makes underneath; it is used instead
// of capturing the command's stdout because these tests run under t.Parallel()
// (see the package comment on CaptureOutput).
func portTagsFromSDK(t *testing.T, uid string) map[string]string {
	t.Helper()
	client := testutil.SharedIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tags, err := client.PortService.ListPortResourceTags(ctx, uid)
	require.NoErrorf(t, err, "SDK ListPortResourceTags failed for %s", uid)
	return tags
}

// runPortTagRoundTrip exercises update-tags then list-tags on an existing
// lifecycle port: it sets two tags via the update-tags command, reads them back
// through the SDK, then clears them and confirms they are gone. It rides on the
// port the caller already provisioned, so it needs no separate cleanup.
func runPortTagRoundTrip(t *testing.T, uid string) {
	t.Helper()
	want := map[string]string{"env": "cli-integration", "owner": "esd-1392"}
	tagJSON, err := json.Marshal(want)
	require.NoError(t, err)

	setCmd := newUpdatePortTagsCmd()
	require.NoError(t, setCmd.Flags().Set("json", string(tagJSON)))
	require.NoError(t, setCmd.Flags().Set("force", "true"))
	require.NoErrorf(t, UpdatePortResourceTags(setCmd, []string{uid}, true), "UpdatePortResourceTags failed for %s", uid)

	// Assert our tags round-tripped without requiring the map to contain only
	// them, so an API-injected tag can't make this flaky.
	got := portTagsFromSDK(t, uid)
	for k, v := range want {
		assert.Equalf(t, v, got[k], "tag %q should round-trip", k)
	}

	clearCmd := newUpdatePortTagsCmd()
	require.NoError(t, clearCmd.Flags().Set("json", "{}"))
	require.NoError(t, clearCmd.Flags().Set("force", "true"))
	require.NoErrorf(t, UpdatePortResourceTags(clearCmd, []string{uid}, true), "clearing port tags failed for %s", uid)

	cleared := portTagsFromSDK(t, uid)
	for k := range want {
		assert.NotContainsf(t, cleared, k, "tag %q should be cleared", k)
	}
}

func TestIntegration_PortLifecycle(t *testing.T) {
	t.Parallel()
	testutil.RequireStagingForProvisioning(t)
	testutil.RequireSharedIntegrationClient(t)

	portName := fmt.Sprintf("CLI-Test-Port-%s", generateUniqueID(t))
	locationID := testutil.FindPortTestLocation(t, testutil.SharedIntegrationClient(t), 1000, integrationLocationID)

	buyCmd := newBuyPortCmd()
	require.NoError(t, buyCmd.Flags().Set("name", portName))
	require.NoError(t, buyCmd.Flags().Set("term", "1"))
	require.NoError(t, buyCmd.Flags().Set("port-speed", "1000"))
	require.NoError(t, buyCmd.Flags().Set("location-id", fmt.Sprintf("%d", locationID)))
	require.NoError(t, buyCmd.Flags().Set("marketplace-visibility", "false"))
	require.NoError(t, buyCmd.Flags().Set("yes", "true"))

	uid := runBuyPort(t, buyCmd, portName)
	t.Logf("Created port with UID: %s", uid)

	port := portFromSDK(t, uid)
	assert.Equal(t, uid, port.UID)
	assert.Equal(t, portName, port.Name)
	assert.NotEmpty(t, port.ProvisioningStatus, "provisioning_status should be populated")

	listed := portsBySDKNameFilter(t, portName)
	require.NotEmpty(t, listed, "newly created port should appear in list filtered by name %q", portName)
	found := false
	for _, p := range listed {
		if p.UID == uid {
			found = true
			break
		}
	}
	assert.Truef(t, found, "uid %s not found in list filtered by name %q; got %d port(s)", uid, portName, len(listed))

	newName := portName + "-Updated"
	runUpdatePortName(t, uid, newName)

	updated := portFromSDK(t, uid)
	assert.Equal(t, newName, updated.Name)

	runPortTagRoundTrip(t, uid)
}

func TestIntegration_LAGPortLifecycle(t *testing.T) {
	t.Parallel()
	testutil.RequireStagingForProvisioning(t)
	testutil.RequireSharedIntegrationClient(t)

	portName := fmt.Sprintf("CLI-Test-LAG-%s", generateUniqueID(t))
	locationID := testutil.FindPortTestLocation(t, testutil.SharedIntegrationClient(t), 10000, integrationLocationID)

	buyCmd := newBuyLAGPortCmd()
	require.NoError(t, buyCmd.Flags().Set("name", portName))
	require.NoError(t, buyCmd.Flags().Set("term", "1"))
	require.NoError(t, buyCmd.Flags().Set("port-speed", "10000"))
	require.NoError(t, buyCmd.Flags().Set("location-id", fmt.Sprintf("%d", locationID)))
	require.NoError(t, buyCmd.Flags().Set("lag-count", "1"))
	require.NoError(t, buyCmd.Flags().Set("marketplace-visibility", "false"))
	require.NoError(t, buyCmd.Flags().Set("yes", "true"))

	uid := runBuyLAGPort(t, buyCmd, portName)
	t.Logf("Created LAG port with UID: %s", uid)

	port := portFromSDK(t, uid)
	assert.Equal(t, uid, port.UID)
	assert.Equal(t, portName, port.Name)
	assert.NotEmpty(t, port.ProvisioningStatus)

	newName := portName + "-Updated"
	runUpdatePortName(t, uid, newName)

	updated := portFromSDK(t, uid)
	assert.Equal(t, newName, updated.Name)
}

func TestIntegration_PortJSONInputLifecycle(t *testing.T) {
	t.Parallel()
	testutil.RequireStagingForProvisioning(t)
	testutil.RequireSharedIntegrationClient(t)

	portName := fmt.Sprintf("CLI-Test-Port-JSON-%s", generateUniqueID(t))
	locationID := testutil.FindPortTestLocation(t, testutil.SharedIntegrationClient(t), 1000, integrationLocationID)

	buyPayload := map[string]any{
		"name":                  portName,
		"term":                  1,
		"portSpeed":             1000,
		"locationId":            locationID,
		"marketPlaceVisibility": false,
	}
	buyJSON, err := json.Marshal(buyPayload)
	require.NoError(t, err)

	buyCmd := newBuyPortCmd()
	require.NoError(t, buyCmd.Flags().Set("json", string(buyJSON)))

	uid := runBuyPort(t, buyCmd, portName)
	t.Logf("Created port (JSON input) with UID: %s", uid)

	port := portFromSDK(t, uid)
	assert.Equal(t, uid, port.UID)
	assert.Equal(t, portName, port.Name)
	assert.NotEmpty(t, port.ProvisioningStatus)

	newName := portName + "-Updated-JSON"
	updatePayload, err := json.Marshal(map[string]string{"name": newName})
	require.NoError(t, err)

	runUpdatePortWithFlag(t, uid, "json", string(updatePayload))

	updated := portFromSDK(t, uid)
	assert.Equal(t, newName, updated.Name)
}

func TestIntegration_PortJSONFileLifecycle(t *testing.T) {
	t.Parallel()
	testutil.RequireStagingForProvisioning(t)
	testutil.RequireSharedIntegrationClient(t)

	portName := fmt.Sprintf("CLI-Test-Port-JSONFile-%s", generateUniqueID(t))
	locationID := testutil.FindPortTestLocation(t, testutil.SharedIntegrationClient(t), 1000, integrationLocationID)

	buyPayload := map[string]any{
		"name":                  portName,
		"term":                  1,
		"portSpeed":             1000,
		"locationId":            locationID,
		"marketPlaceVisibility": false,
	}
	buyJSON, err := json.MarshalIndent(buyPayload, "", "  ")
	require.NoError(t, err)

	buyFile := filepath.Join(t.TempDir(), "port-buy.json")
	require.NoError(t, os.WriteFile(buyFile, buyJSON, 0o600))

	buyCmd := newBuyPortCmd()
	require.NoError(t, buyCmd.Flags().Set("json-file", buyFile))

	uid := runBuyPort(t, buyCmd, portName)
	t.Logf("Created port (JSON file) with UID: %s", uid)

	port := portFromSDK(t, uid)
	assert.Equal(t, uid, port.UID)
	assert.Equal(t, portName, port.Name)
	assert.NotEmpty(t, port.ProvisioningStatus)

	newName := portName + "-Updated-TempJSON"
	updatePayload, err := json.MarshalIndent(map[string]any{
		"name":                  newName,
		"marketPlaceVisibility": true,
	}, "", "  ")
	require.NoError(t, err)

	updateFile := filepath.Join(t.TempDir(), "port-update.json")
	require.NoError(t, os.WriteFile(updateFile, updatePayload, 0o600))

	runUpdatePortWithFlag(t, uid, "json-file", updateFile)

	updated := portFromSDK(t, uid)
	assert.Equal(t, newName, updated.Name)
}

// TestIntegration_LAGPortUpdateLifecycle exercises the LAG port update path
// that TestIntegration_LAGPortLifecycle (name-only) does not.
//
// Immutable via the update action: the LAG member count (--lag-count) and the
// port speed (--port-speed). Neither is a field on megaport.ModifyPortRequest,
// and neither has an update flag (see WithPortUpdateFlags and
// processFlagUpdatePortInput), so `ports update` cannot change them post-buy;
// changing either means a buy/replace, not an update. Mutable via the update
// action: name, marketplace visibility, cost centre, and contract term. Name
// is already covered by TestIntegration_LAGPortLifecycle, and a term change can
// apply at renewal rather than immediately, so this test asserts the two
// attributes that flip and read back deterministically on a LAG primary:
// marketplace visibility and cost centre.
func TestIntegration_LAGPortUpdateLifecycle(t *testing.T) {
	t.Parallel()
	testutil.RequireStagingForProvisioning(t)
	testutil.RequireSharedIntegrationClient(t)

	portName := fmt.Sprintf("CLI-Test-LAG-Update-%s", generateUniqueID(t))
	locationID := testutil.FindPortTestLocation(t, testutil.SharedIntegrationClient(t), 10000, integrationLocationID)

	buyCmd := newBuyLAGPortCmd()
	require.NoError(t, buyCmd.Flags().Set("name", portName))
	require.NoError(t, buyCmd.Flags().Set("term", "1"))
	require.NoError(t, buyCmd.Flags().Set("port-speed", "10000"))
	require.NoError(t, buyCmd.Flags().Set("location-id", fmt.Sprintf("%d", locationID)))
	require.NoError(t, buyCmd.Flags().Set("lag-count", "1"))
	require.NoError(t, buyCmd.Flags().Set("marketplace-visibility", "false"))
	require.NoError(t, buyCmd.Flags().Set("yes", "true"))

	uid := runBuyLAGPort(t, buyCmd, portName)
	t.Logf("Created LAG port with UID: %s", uid)

	// Baseline read. We assert the buy landed the values the updates move away
	// from, so each assertion below proves a real change rather than passing
	// trivially. LAGPrimary/speed are captured to confirm the attribute updates
	// leave the LAG topology untouched.
	port := portFromSDK(t, uid)
	assert.Equal(t, uid, port.UID)
	assert.Equal(t, portName, port.Name)
	require.False(t, port.MarketplaceVisibility, "buy set marketplace visibility to false")
	require.Equal(t, 10000, port.PortSpeed, "buy requested a 10G LAG port")
	initialLAGPrimary := port.LAGPrimary
	t.Logf("LAG port baseline: lagPrimary=%t lagId=%d", port.LAGPrimary, port.LAGID)

	// Update 1: marketplace visibility false -> true.
	runUpdatePortWithFlag(t, uid, "marketplace-visibility", "true")
	afterMV := portFromSDK(t, uid)
	assert.True(t, afterMV.MarketplaceVisibility, "marketplace visibility should be true after update")

	// Update 2: set a cost centre and read it back.
	costCentre := "CLI-ESD-1528-" + generateUniqueID(t)
	runUpdatePortWithFlag(t, uid, "cost-centre", costCentre)
	afterCC := portFromSDK(t, uid)
	assert.Equal(t, costCentre, afterCC.CostCentre, "cost centre should round-trip after update")
	// The cost-centre-only update must not clobber the earlier visibility flip.
	assert.True(t, afterCC.MarketplaceVisibility, "marketplace visibility should persist across the cost-centre update")

	// LAG count and speed are immutable via the update action; confirm the
	// attribute updates left the LAG topology and speed untouched.
	assert.Equal(t, initialLAGPrimary, afterCC.LAGPrimary, "LAG primary status should be unchanged by attribute updates")
	assert.Equal(t, 10000, afterCC.PortSpeed, "port speed should be unchanged by attribute updates")

	// Delete and decommissioning are asserted by the cleanup registered in
	// runBuyLAGPort: registerPortCleanup deletes the port and polls the SDK
	// until it reaches DECOMMISSIONING/DECOMMISSIONED.
}
