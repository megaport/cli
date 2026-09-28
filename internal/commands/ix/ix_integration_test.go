//go:build integration

package ix

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/ports"
	"github.com/megaport/megaport-cli/internal/testutil"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateUniqueID(t *testing.T) string {
	t.Helper()
	buf := make([]byte, 4)
	_, err := rand.Read(buf)
	require.NoError(t, err, "failed to read crypto/rand entropy")
	return hex.EncodeToString(buf)
}

// extractCreatedUID parses "<ResourceType> created <uid>" from buy command output.
func extractCreatedUID(captured, resourceType string) (string, bool) {
	prefix := resourceType + " created "
	for _, line := range strings.Split(captured, "\n") {
		if idx := strings.Index(line, prefix); idx >= 0 {
			uid := strings.TrimSpace(line[idx+len(prefix):])
			if uid != "" {
				return uid, true
			}
		}
	}
	return "", false
}

func integrationListIXCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "list"}
	cmd.Flags().String("name", "", "")
	cmd.Flags().Int("asn", 0, "")
	cmd.Flags().Int("vlan", 0, "")
	cmd.Flags().String("network-service-type", "", "")
	cmd.Flags().Int("location-id", 0, "")
	cmd.Flags().Int("rate-limit", 0, "")
	cmd.Flags().Bool("include-inactive", false, "")
	cmd.Flags().Int("limit", 0, "")
	return cmd
}

func integrationBuyIXCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "buy"}
	cmd.Flags().BoolP("interactive", "i", false, "")
	cmd.Flags().Bool("no-wait", false, "")
	cmd.Flags().BoolP("yes", "y", false, "")
	cmd.Flags().String("product-uid", "", "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().String("network-service-type", "", "")
	cmd.Flags().Int("asn", 0, "")
	cmd.Flags().String("mac-address", "", "")
	cmd.Flags().Int("rate-limit", 0, "")
	cmd.Flags().Int("vlan", 0, "")
	cmd.Flags().Bool("shutdown", false, "")
	cmd.Flags().String("promo-code", "", "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	return cmd
}

func integrationDeleteIXCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "delete"}
	cmd.Flags().BoolP("force", "f", false, "")
	return cmd
}

func integrationUpdateIXCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "update"}
	cmd.Flags().BoolP("interactive", "i", false, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().Int("rate-limit", 0, "")
	cmd.Flags().String("cost-centre", "", "")
	cmd.Flags().Int("vlan", 0, "")
	cmd.Flags().String("mac-address", "", "")
	cmd.Flags().Int("asn", 0, "")
	cmd.Flags().String("password", "", "")
	cmd.Flags().Bool("public-graph", false, "")
	cmd.Flags().String("reverse-dns", "", "")
	cmd.Flags().String("a-end-product-uid", "", "")
	cmd.Flags().Bool("shutdown", false, "")
	return cmd
}

func integrationBuyPortCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "buy"}
	cmd.Flags().BoolP("interactive", "i", false, "")
	cmd.Flags().Bool("no-wait", false, "")
	cmd.Flags().BoolP("yes", "y", false, "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().Int("term", 0, "")
	cmd.Flags().Int("port-speed", 0, "")
	cmd.Flags().Int("location-id", 0, "")
	cmd.Flags().Bool("marketplace-visibility", false, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	return cmd
}

func integrationDeletePortCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "delete"}
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().Bool("safe-delete", false, "")
	return cmd
}

func integrationGetIXCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "get"}
	cmd.Flags().Bool("export", false, "")
	return cmd
}

func integrationStatusIXCmd() *cobra.Command {
	return &cobra.Command{Use: "status"}
}

// TestIntegration_IXReadOnly is a fast read-only smoke test against the
// configured environment (staging by default):
// list, then get + status on the first IX. Skips cleanly when the account has
// no IXs. Performs no mutation.
func TestIntegration_IXReadOnly(t *testing.T) {
	testutil.RequireSharedIntegrationClient(t)
	origFormat := output.GetOutputFormat()
	t.Cleanup(func() { output.SetOutputFormat(origFormat) })

	var listErr error
	listOut := output.CaptureOutput(func() {
		listErr = ListIXs(integrationListIXCmd(), nil, true, "json")
	})
	require.NoError(t, listErr)

	// JSON mode emits no output (not "[]") when the list is empty.
	if strings.TrimSpace(listOut) == "" {
		t.Skip("no IXs available on the account to test Get")
	}
	var ixs []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(listOut), &ixs), "ListIXs returned invalid JSON")
	if len(ixs) == 0 {
		t.Skip("no IXs available on the account to test Get")
	}

	first := ixs[0]
	assert.Contains(t, first, "uid", "IX should have a uid field")
	assert.Contains(t, first, "name", "IX should have a name field")

	uid, ok := first["uid"].(string)
	require.True(t, ok && uid != "", "uid must be a non-empty string")

	var getErr error
	getOut := output.CaptureOutput(func() {
		getErr = GetIX(integrationGetIXCmd(), []string{uid}, true, "json")
	})
	require.NoError(t, getErr)

	var gotList []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(getOut), &gotList), "GetIX JSON output must be valid")
	require.Len(t, gotList, 1)
	gotIX := gotList[0]
	assert.Equal(t, uid, gotIX["uid"])
	assert.Contains(t, gotIX, "name")
	assert.Contains(t, gotIX, "status")

	var statusErr error
	statusOut := output.CaptureOutput(func() {
		statusErr = GetIXStatus(integrationStatusIXCmd(), []string{uid}, true, "json")
	})
	require.NoError(t, statusErr)

	var statusList []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(statusOut), &statusList), "GetIXStatus JSON output must be valid")
	require.Len(t, statusList, 1)
	assert.Equal(t, uid, statusList[0]["uid"])
	assert.Contains(t, statusList[0], "status")
}

// TestIntegration_IXLifecycle exercises the full create → get → status → update → delete path.
// Discovers a valid IX network-service-type from the IXP catalog and a matching location.
// Expected runtime: up to ~30 minutes (three provisioning waits of up to 10 min each: port buy, IX buy, IX update).
func TestIntegration_IXLifecycle(t *testing.T) {
	testutil.RequireStagingForProvisioning(t)
	client := testutil.SetupIntegrationClient(t)
	t.Cleanup(testutil.LoginWithClient(t, client))
	origFormat := output.GetOutputFormat()
	// Ensure table format so PrintResourceCreated writes to stdout, not stderr.
	output.SetOutputFormat("table")
	t.Cleanup(func() { output.SetOutputFormat(origFormat) })

	// Discover a valid IX network-service-type from the global IXP catalog,
	// then find a Megaport location in the same metro to host the test port.
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	ixps, err := client.IXService.ListIXPs(ctx, nil)
	require.NoError(t, err, "failed to list IXPs")
	if len(ixps) == 0 {
		t.Skip("no IXPs available on staging")
	}

	allLocs, err := client.LocationService.ListLocationsV3(ctx)
	require.NoError(t, err, "failed to list locations")

	var networkServiceType string
	var locationID int
	for _, ixp := range ixps {
		if ixp == nil || ixp.Name == "" {
			continue
		}
		matching := client.LocationService.FilterLocationsByMetroV3(ctx, ixp.Metro, allLocs)
		if len(matching) == 0 {
			continue
		}
		networkServiceType = ixp.Name
		locationID = matching[0].ID
		break
	}
	if networkServiceType == "" || locationID == 0 {
		t.Skip("no IXP with a matching staging location found")
	}

	// Create the port that will serve as the A-end of the IX.
	const portSpeed = 1000
	portName := fmt.Sprintf("CLI-Test-Port-IX-%s", generateUniqueID(t))
	pCmd := integrationBuyPortCmd()
	require.NoError(t, pCmd.Flags().Set("name", portName))
	require.NoError(t, pCmd.Flags().Set("term", "1"))
	require.NoError(t, pCmd.Flags().Set("port-speed", fmt.Sprintf("%d", portSpeed)))
	require.NoError(t, pCmd.Flags().Set("location-id", fmt.Sprintf("%d", locationID)))
	require.NoError(t, pCmd.Flags().Set("marketplace-visibility", "false"))
	require.NoError(t, pCmd.Flags().Set("yes", "true"))

	var portErr error
	portOut := output.CaptureOutput(func() {
		portErr = ports.BuyPort(pCmd, nil, true)
	})
	require.NoError(t, portErr, "failed to create test port")

	portUID, ok := extractCreatedUID(portOut, "Port")

	// Port cleanup runs last (registered first — t.Cleanup is LIFO).
	// Registered before require.True so it runs even if UID extraction fails.
	t.Cleanup(func() {
		if portUID == "" {
			t.Errorf("cleanup: port UID is empty, staged port may have been leaked")
			return
		}
		output.SetOutputFormat("table")
		delPortCmd := integrationDeletePortCmd()
		if err := delPortCmd.Flags().Set("force", "true"); err != nil {
			t.Errorf("cleanup: failed to set --force flag on port delete: %v", err)
			return
		}
		output.CaptureOutput(func() {
			if err := ports.DeletePort(delPortCmd, []string{portUID}, true); err != nil {
				t.Errorf("cleanup: failed to delete test port %s: %v", portUID, err)
			}
		})
	})
	require.True(t, ok, "could not extract port UID from output:\n%s", portOut)

	// Probe for a supported rate limit — each metro/IXP may only accept specific speeds,
	// so validating before buy avoids environment-unrelated failures.
	candidateRateLimits := []int{1000, 100, 500, 10000}
	rateLimit := 0
	probeCtx, probeCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer probeCancel()
	for _, candidate := range candidateRateLimits {
		probeReq := &megaport.BuyIXRequest{
			ProductUID:         portUID,
			Name:               "probe",
			NetworkServiceType: networkServiceType,
			ASN:                12345,
			MACAddress:         "00:11:22:33:44:55",
			RateLimit:          candidate,
			VLAN:               100,
		}
		if err := client.IXService.ValidateIXOrder(probeCtx, probeReq); err == nil {
			rateLimit = candidate
			break
		}
	}
	if rateLimit == 0 {
		t.Skipf("no candidate rate limit validated for IX type %q at location %d", networkServiceType, locationID)
	}

	// Buy the IX.
	ixName := fmt.Sprintf("CLI-Test-IX-%s", generateUniqueID(t))
	buyCmd := integrationBuyIXCmd()
	require.NoError(t, buyCmd.Flags().Set("product-uid", portUID))
	require.NoError(t, buyCmd.Flags().Set("name", ixName))
	require.NoError(t, buyCmd.Flags().Set("network-service-type", networkServiceType))
	require.NoError(t, buyCmd.Flags().Set("asn", "12345"))
	require.NoError(t, buyCmd.Flags().Set("mac-address", "00:11:22:33:44:55"))
	require.NoError(t, buyCmd.Flags().Set("rate-limit", fmt.Sprintf("%d", rateLimit)))
	require.NoError(t, buyCmd.Flags().Set("vlan", "100"))
	require.NoError(t, buyCmd.Flags().Set("yes", "true"))

	var buyErr error
	buyOut := output.CaptureOutput(func() {
		buyErr = BuyIX(buyCmd, nil, true)
	})
	if buyErr != nil {
		errMsg := buyErr.Error()
		if strings.Contains(errMsg, "not available") || strings.Contains(errMsg, "not supported") {
			t.Skipf("IX type %q not available at location %d on staging: %v", networkServiceType, locationID, buyErr)
		}
		require.NoError(t, buyErr, "failed to create test IX")
	}

	ixUID, ok := extractCreatedUID(buyOut, "IX")

	// IX cleanup runs first (registered second — LIFO).
	// Registered before require.True so it runs even if UID extraction fails.
	t.Cleanup(func() {
		if ixUID == "" {
			t.Errorf("cleanup: IX UID is empty, staged IX may have been leaked")
			return
		}
		output.SetOutputFormat("table")
		delCmd := integrationDeleteIXCmd()
		if err := delCmd.Flags().Set("force", "true"); err != nil {
			t.Errorf("cleanup: failed to set --force flag on IX delete: %v", err)
			return
		}
		output.CaptureOutput(func() {
			if err := DeleteIX(delCmd, []string{ixUID}, true); err != nil {
				t.Errorf("cleanup: failed to delete test IX %s: %v", ixUID, err)
			}
		})
	})
	require.True(t, ok, "could not extract IX UID from buy output:\n%s", buyOut)

	// Verify IX fields via GetIX.
	var getErr error
	getOut := output.CaptureOutput(func() {
		getErr = GetIX(integrationGetIXCmd(), []string{ixUID}, true, "json")
	})
	require.NoError(t, getErr)

	var gotList []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(getOut), &gotList))
	require.Len(t, gotList, 1)
	gotIX := gotList[0]
	assert.Equal(t, ixUID, gotIX["uid"])
	assert.Equal(t, ixName, gotIX["name"])
	assert.Contains(t, gotIX, "status")

	// Status as a sub-test to avoid an extra IX.
	t.Run("Status", func(t *testing.T) {
		var statusErr error
		statusOut := output.CaptureOutput(func() {
			statusErr = GetIXStatus(integrationStatusIXCmd(), []string{ixUID}, true, "json")
		})
		require.NoError(t, statusErr)

		var statusList []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(statusOut), &statusList))
		require.Len(t, statusList, 1)
		assert.Equal(t, ixUID, statusList[0]["uid"])
		assert.Contains(t, statusList[0], "status")
	})

	// Update the IX name. Force table format so CaptureOutput captures update messages.
	output.SetOutputFormat("table")
	updatedName := ixName + "-upd"
	updCmd := integrationUpdateIXCmd()
	require.NoError(t, updCmd.Flags().Set("name", updatedName))
	var updateErr error
	output.CaptureOutput(func() {
		updateErr = UpdateIX(updCmd, []string{ixUID}, true)
	})
	require.NoError(t, updateErr)

	// Verify the name was updated.
	var getErr2 error
	getOut2 := output.CaptureOutput(func() {
		getErr2 = GetIX(integrationGetIXCmd(), []string{ixUID}, true, "json")
	})
	require.NoError(t, getErr2)

	var gotList2 []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(getOut2), &gotList2))
	require.Len(t, gotList2, 1)
	assert.Equal(t, updatedName, gotList2[0]["name"])
}
