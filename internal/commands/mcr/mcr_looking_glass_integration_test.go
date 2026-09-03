//go:build integration

package mcr

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/testutil"
	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// firstMCRUIDForLookingGlass lists MCRs via the CLI and returns the first
// UID, skipping the test when the account has none.
func firstMCRUIDForLookingGlass(t *testing.T) string {
	t.Helper()

	var listErr error
	listOut := output.CaptureOutput(func() {
		listErr = ListMCRs(readOnlyListMCRsCmd(), nil, true, "json")
	})
	require.NoError(t, listErr)

	if strings.TrimSpace(listOut) == "" {
		t.Skip("no MCRs on the account")
	}
	var mcrList []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(listOut), &mcrList), "ListMCRs returned invalid JSON")
	if len(mcrList) == 0 {
		t.Skip("no MCRs on the account")
	}

	uid, ok := mcrList[0]["uid"].(string)
	require.True(t, ok && uid != "", "uid must be a non-empty string")
	return uid
}

// firstBGPPeerIP looks for a BGP peer IP on any VXC attached to mcrUID, by
// reading associated VXCs straight from the SDK. Returns "" when none is
// configured.
func firstBGPPeerIP(t *testing.T, mcrUID string) string {
	t.Helper()

	client := testutil.SharedIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	mcr, err := client.MCRService.GetMCR(ctx, mcrUID)
	require.NoError(t, err)

	for _, vxc := range mcr.AssociatedVXCs {
		if vxc.Resources == nil || vxc.Resources.CSPConnection == nil {
			continue
		}
		for _, conn := range vxc.Resources.CSPConnection.CSPConnection {
			vRouter, ok := conn.(*megaport.CSPConnectionVirtualRouter)
			if !ok {
				continue
			}
			for _, iface := range vRouter.Interfaces {
				for _, bgp := range iface.BGPConnections {
					if bgp.PeerIpAddress != "" {
						return bgp.PeerIpAddress
					}
				}
			}
		}
	}
	return ""
}

// TestIntegration_MCRLookingGlassReadOnly is a fast read-only smoke test
// against the configured environment (staging by default): ip-routes and
// bgp-routes on the account's first MCR, plus bgp-neighbor-routes when a BGP
// peer is configured on one of its VXCs. Skips cleanly when the account has
// no MCRs, or no BGP peer for the neighbor-routes case. Performs no mutation.
func TestIntegration_MCRLookingGlassReadOnly(t *testing.T) {
	testutil.RequireSharedIntegrationClient(t)
	origFormat := output.GetOutputFormat()
	t.Cleanup(func() { output.SetOutputFormat(origFormat) })

	mcrUID := firstMCRUIDForLookingGlass(t)

	t.Run("ip-routes", func(t *testing.T) {
		var err error
		out := output.CaptureOutput(func() {
			err = ListLookingGlassIPRoutes(routeFlagsCmd("", ""), []string{mcrUID}, true, "json")
		})
		require.NoError(t, err)
		var routes []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(out), &routes), "ip-routes JSON output must be valid")
	})

	t.Run("bgp-routes", func(t *testing.T) {
		var err error
		out := output.CaptureOutput(func() {
			err = ListLookingGlassBGPRoutes(routeFlagsCmd("", ""), []string{mcrUID}, true, "json")
		})
		require.NoError(t, err)
		var routes []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(out), &routes), "bgp-routes JSON output must be valid")
	})

	t.Run("bgp-neighbor-routes", func(t *testing.T) {
		peerIP := firstBGPPeerIP(t, mcrUID)
		if peerIP == "" {
			t.Skip("no BGP peer configured on any VXC attached to the MCR")
		}

		var err error
		out := output.CaptureOutput(func() {
			err = ListLookingGlassBGPNeighborRoutes(routeFlagsCmd("", ""), []string{mcrUID, peerIP, "received"}, true, "json")
		})
		require.NoError(t, err)
		var routes []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(out), &routes), "bgp-neighbor-routes JSON output must be valid")
		if len(routes) > 0 {
			assert.Contains(t, routes[0], "prefix")
		}
	})
}
