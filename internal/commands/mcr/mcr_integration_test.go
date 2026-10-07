//go:build integration && provisioning

package mcr

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/testutil"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stagingMCRLocationID is the staging location MCR lifecycle tests prefer. It
// has historically been MCR-capable, but it is only a preference: each test
// resolves its location through testutil.FindMCRTestLocation, which falls back
// to another active MCR-capable location (or skips) if this one ever stops
// advertising the speed under test.
const stagingMCRLocationID = 67

func generateUniqueID() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("failed to generate random bytes: %v", err))
	}
	return hex.EncodeToString(buf)
}

// captureTableOutput captures stdout with the output format forced to "table",
// since JSON mode routes success/info messages to stderr where CaptureOutput
// can't see them.
func captureTableOutput(f func()) string {
	output.SetOutputFormat("table")
	return output.CaptureOutput(f)
}

func integrationMCRBuyCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "buy"}
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().Bool("yes", false, "")
	cmd.Flags().Bool("no-wait", false, "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().Int("term", 0, "")
	cmd.Flags().Int("port-speed", 0, "")
	cmd.Flags().Int("location-id", 0, "")
	cmd.Flags().Int("mcr-asn", 0, "")
	cmd.Flags().String("diversity-zone", "", "")
	cmd.Flags().String("cost-centre", "", "")
	cmd.Flags().String("promo-code", "", "")
	cmd.Flags().Bool("marketplace-visibility", false, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	return cmd
}

func integrationMCRUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "update"}
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().String("cost-centre", "", "")
	cmd.Flags().Bool("marketplace-visibility", false, "")
	cmd.Flags().Int("term", 0, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	return cmd
}

func integrationMCRDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "delete"}
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().Bool("safe-delete", false, "")
	return cmd
}

func integrationMCRTagCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "update-tags"}
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	return cmd
}

// tagsFromListJSON parses the JSON output of list-tags (an array of {key,value}
// objects) into a map, so assertions can check key->value pairs instead of
// substring-matching the rendered blob.
func tagsFromListJSON(t *testing.T, out string) map[string]string {
	t.Helper()
	var tags []output.ResourceTag
	require.NoErrorf(t, json.Unmarshal([]byte(out), &tags), "parse list-tags JSON: %s", out)
	m := make(map[string]string, len(tags))
	for _, tag := range tags {
		m[tag.Key] = tag.Value
	}
	return m
}

func integrationMCRPrefixFilterCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "prefix-filter-list"}
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().String("description", "", "")
	cmd.Flags().String("address-family", "", "")
	cmd.Flags().String("entries", "", "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	return cmd
}

// parseCreatedUID pulls the MCR UID out of a "MCR created <uid>" success
// message.
func parseCreatedUID(out string) string {
	const marker = "MCR created "
	for _, line := range strings.Split(out, "\n") {
		if i := strings.Index(line, marker); i >= 0 {
			return strings.TrimSpace(line[i+len(marker):])
		}
	}
	return ""
}

// parsePrefixFilterListID pulls the numeric ID out of the
// "Prefix filter list created successfully - ID: <n>" success message.
func parsePrefixFilterListID(out string) string {
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, "created successfully") {
			continue
		}
		if i := strings.Index(line, "ID:"); i >= 0 {
			return strings.TrimSpace(line[i+len("ID:"):])
		}
	}
	return ""
}

// getMCRJSON retrieves an MCR as JSON and returns the decoded object.
func getMCRJSON(t *testing.T, uid string) map[string]interface{} {
	t.Helper()
	var err error
	out := output.CaptureOutput(func() {
		err = GetMCR(&cobra.Command{Use: "get"}, []string{uid}, true, "json")
	})
	require.NoError(t, err, "get MCR output: %s", out)

	var mcrs []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &mcrs), "MCR output should be valid JSON: %s", out)
	require.Len(t, mcrs, 1, "expected exactly one MCR")
	return mcrs[0]
}

func TestIntegration_MCRLifecycle(t *testing.T) {
	testutil.RequireStagingForProvisioning(t)
	client := testutil.SetupIntegrationClient(t)
	// Restore login via t.Cleanup, not defer: defers run before t.Cleanup, so a
	// deferred restore would swap back the default login (wrong environment)
	// before the resource-deletion cleanups below get to run.
	t.Cleanup(testutil.LoginWithClient(t, client))

	// Action functions mutate the process-wide output format; restore it so
	// test order can't leak state between tests in this package.
	origFmt := output.GetOutputFormat()
	t.Cleanup(func() { output.SetOutputFormat(origFmt) })

	name := fmt.Sprintf("CLI-Test-MCR-%s", generateUniqueID())
	locationID := testutil.FindMCRTestLocation(t, client, 1000, stagingMCRLocationID)

	// Buy a new MCR using flags. BuyMCR waits for provisioning (no --no-wait),
	// so the MCR is ready for prefix-filter-list operations once it returns.
	buyCmd := integrationMCRBuyCmd()
	require.NoError(t, buyCmd.Flags().Set("name", name))
	require.NoError(t, buyCmd.Flags().Set("term", "1"))
	require.NoError(t, buyCmd.Flags().Set("port-speed", "1000"))
	require.NoError(t, buyCmd.Flags().Set("location-id", strconv.Itoa(locationID)))
	require.NoError(t, buyCmd.Flags().Set("marketplace-visibility", "false"))
	require.NoError(t, buyCmd.Flags().Set("yes", "true"))

	var buyErr error
	buyOut := captureTableOutput(func() { buyErr = BuyMCR(buyCmd, nil, true) })
	require.NoError(t, buyErr, "buy MCR output: %s", buyOut)

	mcrUID := parseCreatedUID(buyOut)
	// Register cleanup before asserting on mcrUID, so any created MCR is
	// deleted even if the UID parse fails.
	t.Cleanup(func() {
		if mcrUID == "" {
			return
		}
		delCmd := integrationMCRDeleteCmd()
		_ = delCmd.Flags().Set("force", "true")
		var delErr error
		out := captureTableOutput(func() { delErr = DeleteMCR(delCmd, []string{mcrUID}, true) })
		if delErr != nil {
			t.Logf("cleanup: delete MCR %s failed: %v; output: %s", mcrUID, delErr, out)
			return
		}
		t.Logf("cleanup: delete MCR %s: %s", mcrUID, out)
	})
	require.NotEmpty(t, mcrUID, "could not parse MCR UID from: %s", buyOut)

	// Get and verify the core fields are present.
	mcr := getMCRJSON(t, mcrUID)
	assert.Equal(t, mcrUID, mcr["uid"])
	assert.Equal(t, name, mcr["name"])
	assert.Contains(t, mcr, "provisioning_status")
	assert.NotEmpty(t, mcr["provisioning_status"])

	// Update the name and verify it took effect.
	newName := name + "-updated"
	updCmd := integrationMCRUpdateCmd()
	require.NoError(t, updCmd.Flags().Set("name", newName))
	var updErr error
	updOut := captureTableOutput(func() { updErr = UpdateMCR(updCmd, []string{mcrUID}, true) })
	require.NoError(t, updErr, "update MCR output: %s", updOut)

	assert.Equal(t, newName, getMCRJSON(t, mcrUID)["name"])

	// Resource tag round-trip (ESD-1392): set tags via update-tags, read them
	// back via list-tags, then clear them. Rides on the lifecycle MCR, so no
	// extra cleanup is needed.
	want := map[string]string{"env": "cli-integration", "owner": "esd-1392"}
	setTagsJSON, err := json.Marshal(want)
	require.NoError(t, err)
	setTagsCmd := integrationMCRTagCmd()
	require.NoError(t, setTagsCmd.Flags().Set("json", string(setTagsJSON)))
	require.NoError(t, setTagsCmd.Flags().Set("force", "true"))
	var setTagsErr error
	setTagsOut := captureTableOutput(func() { setTagsErr = UpdateMCRResourceTags(setTagsCmd, []string{mcrUID}, true) })
	require.NoError(t, setTagsErr, "update MCR tags output: %s", setTagsOut)

	var listTagsErr error
	listTagsOut := output.CaptureOutput(func() {
		listTagsErr = ListMCRResourceTags(&cobra.Command{Use: "list-tags"}, []string{mcrUID}, true, "json")
	})
	require.NoError(t, listTagsErr, "list MCR tags output: %s", listTagsOut)
	// Assert our tags round-tripped without requiring the map to contain only
	// them, so an API-injected tag can't make this flaky.
	got := tagsFromListJSON(t, listTagsOut)
	for k, v := range want {
		assert.Equalf(t, v, got[k], "tag %q should round-trip", k)
	}

	// Clear the tags so the MCR is left clean for the steps that follow.
	clearTagsCmd := integrationMCRTagCmd()
	require.NoError(t, clearTagsCmd.Flags().Set("json", "{}"))
	require.NoError(t, clearTagsCmd.Flags().Set("force", "true"))
	var clearTagsErr error
	clearTagsOut := captureTableOutput(func() { clearTagsErr = UpdateMCRResourceTags(clearTagsCmd, []string{mcrUID}, true) })
	require.NoError(t, clearTagsErr, "clear MCR tags output: %s", clearTagsOut)

	var verifyTagsErr error
	verifyTagsOut := output.CaptureOutput(func() {
		verifyTagsErr = ListMCRResourceTags(&cobra.Command{Use: "list-tags"}, []string{mcrUID}, true, "json")
	})
	require.NoError(t, verifyTagsErr, "list MCR tags after clear output: %s", verifyTagsOut)
	cleared := tagsFromListJSON(t, verifyTagsOut)
	for k := range want {
		assert.NotContainsf(t, cleared, k, "tag %q should be cleared", k)
	}

	// Prefix filter list lifecycle (create -> get -> update -> delete).
	createPFLJSON := `{
		"description": "Test Prefix Filter List",
		"addressFamily": "IPv4",
		"entries": [
			{"action": "permit", "prefix": "10.0.1.0/24", "ge": 25, "le": 32},
			{"action": "deny", "prefix": "10.0.2.0/24", "le": 25}
		]
	}`

	createCmd := integrationMCRPrefixFilterCmd()
	require.NoError(t, createCmd.Flags().Set("json", createPFLJSON))
	var createErr error
	createOut := captureTableOutput(func() {
		createErr = CreateMCRPrefixFilterList(createCmd, []string{mcrUID}, true)
	})
	require.NoError(t, createErr, "create prefix filter list output: %s", createOut)

	pflID := parsePrefixFilterListID(createOut)
	require.NotEmpty(t, pflID, "could not parse prefix filter list ID from: %s", createOut)

	// Best-effort cleanup. Registered after the MCR cleanup so it runs first
	// (cleanups run LIFO) — the list must be gone before the MCR is deleted.
	t.Cleanup(func() {
		var delErr error
		out := captureTableOutput(func() {
			delErr = DeleteMCRPrefixFilterList(&cobra.Command{Use: "delete"}, []string{mcrUID, pflID}, true)
		})
		if delErr != nil {
			t.Logf("cleanup: delete prefix filter list %s failed: %v; output: %s", pflID, delErr, out)
			return
		}
		t.Logf("cleanup: delete prefix filter list %s: %s", pflID, out)
	})

	// Get the prefix filter list and assert it exists with the expected content.
	var getPFLErr error
	getPFLOut := output.CaptureOutput(func() {
		getPFLErr = GetMCRPrefixFilterList(&cobra.Command{Use: "get"}, []string{mcrUID, pflID}, true, "json")
	})
	require.NoError(t, getPFLErr, "get prefix filter list output: %s", getPFLOut)
	assert.Contains(t, getPFLOut, "Test Prefix Filter List")
	assert.Contains(t, getPFLOut, "10.0.1.0/24")
	assert.Contains(t, getPFLOut, "10.0.2.0/24")

	// Update the prefix filter list, adding a new entry.
	updatePFLJSON := `{
		"description": "Test Prefix Filter List Updated",
		"addressFamily": "IPv4",
		"entries": [
			{"action": "permit", "prefix": "10.0.1.0/24", "ge": 25, "le": 32},
			{"action": "deny", "prefix": "10.0.2.0/24", "le": 25},
			{"action": "permit", "prefix": "192.168.0.0/16", "ge": 24, "le": 32}
		]
	}`
	updatePFLCmd := integrationMCRPrefixFilterCmd()
	require.NoError(t, updatePFLCmd.Flags().Set("json", updatePFLJSON))
	var updatePFLErr error
	updatePFLOut := captureTableOutput(func() {
		updatePFLErr = UpdateMCRPrefixFilterList(updatePFLCmd, []string{mcrUID, pflID}, true)
	})
	require.NoError(t, updatePFLErr, "update prefix filter list output: %s", updatePFLOut)

	// Verify the update landed.
	var verifyPFLErr error
	verifyPFLOut := output.CaptureOutput(func() {
		verifyPFLErr = GetMCRPrefixFilterList(&cobra.Command{Use: "get"}, []string{mcrUID, pflID}, true, "json")
	})
	require.NoError(t, verifyPFLErr, "verify prefix filter list output: %s", verifyPFLOut)
	assert.Contains(t, verifyPFLOut, "Test Prefix Filter List Updated")
	assert.Contains(t, verifyPFLOut, "192.168.0.0/16")

	// Delete the prefix filter list and assert it is gone.
	var deletePFLErr error
	deletePFLOut := captureTableOutput(func() {
		deletePFLErr = DeleteMCRPrefixFilterList(&cobra.Command{Use: "delete"}, []string{mcrUID, pflID}, true)
	})
	require.NoError(t, deletePFLErr, "delete prefix filter list output: %s", deletePFLOut)

	goneErr := GetMCRPrefixFilterList(&cobra.Command{Use: "get"}, []string{mcrUID, pflID}, true, "json")
	assert.Error(t, goneErr, "prefix filter list should no longer exist")
}

func TestIntegration_MCRJSONInputLifecycle(t *testing.T) {
	testutil.RequireStagingForProvisioning(t)
	client := testutil.SetupIntegrationClient(t)
	// Restore login via t.Cleanup, not defer: defers run before t.Cleanup, so a
	// deferred restore would swap back the default login (wrong environment)
	// before the resource-deletion cleanups below get to run.
	t.Cleanup(testutil.LoginWithClient(t, client))

	// Action functions mutate the process-wide output format; restore it so
	// test order can't leak state between tests in this package.
	origFmt := output.GetOutputFormat()
	t.Cleanup(func() { output.SetOutputFormat(origFmt) })

	name := fmt.Sprintf("CLI-JSON-MCR-%s", generateUniqueID())
	locationID := testutil.FindMCRTestLocation(t, client, 1000, stagingMCRLocationID)

	buyJSON := fmt.Sprintf(`{
		"name": "%s",
		"term": 1,
		"portSpeed": 1000,
		"locationId": %d,
		"marketplaceVisibility": false
	}`, name, locationID)

	buyCmd := integrationMCRBuyCmd()
	require.NoError(t, buyCmd.Flags().Set("json", buyJSON))

	var buyErr error
	buyOut := captureTableOutput(func() { buyErr = BuyMCR(buyCmd, nil, true) })
	require.NoError(t, buyErr, "buy MCR (JSON) output: %s", buyOut)

	mcrUID := parseCreatedUID(buyOut)
	// Register cleanup before asserting on mcrUID, so any created MCR is
	// deleted even if the UID parse fails.
	t.Cleanup(func() {
		if mcrUID == "" {
			return
		}
		delCmd := integrationMCRDeleteCmd()
		_ = delCmd.Flags().Set("force", "true")
		var delErr error
		out := captureTableOutput(func() { delErr = DeleteMCR(delCmd, []string{mcrUID}, true) })
		if delErr != nil {
			t.Logf("cleanup: delete MCR %s failed: %v; output: %s", mcrUID, delErr, out)
			return
		}
		t.Logf("cleanup: delete MCR %s: %s", mcrUID, out)
	})
	require.NotEmpty(t, mcrUID, "could not parse MCR UID from: %s", buyOut)

	mcr := getMCRJSON(t, mcrUID)
	assert.Equal(t, mcrUID, mcr["uid"])
	assert.Equal(t, name, mcr["name"])

	newName := name + "-updated"
	updCmd := integrationMCRUpdateCmd()
	require.NoError(t, updCmd.Flags().Set("name", newName))
	var updErr error
	updOut := captureTableOutput(func() { updErr = UpdateMCR(updCmd, []string{mcrUID}, true) })
	require.NoError(t, updErr, "update MCR output: %s", updOut)

	assert.Equal(t, newName, getMCRJSON(t, mcrUID)["name"])
}

// prefixEntry is the expected shape of one prefix-filter-list entry, used for
// SDK-level assertions.
type prefixEntry struct {
	action string
	prefix string
	ge     *int
	le     *int
}

// canonicalPrefix normalizes an IP prefix so textual differences (IPv6 case,
// zero compression) can't cause spurious mismatches when pairing entries.
func canonicalPrefix(t *testing.T, p string) string {
	t.Helper()
	parsed, err := netip.ParsePrefix(p)
	if err != nil {
		return p
	}
	return parsed.Masked().String()
}

// getPrefixFilterListViaSDK reads a prefix filter list straight from the SDK so
// assertions check authoritative API state rather than scraped CLI output.
func getPrefixFilterListViaSDK(t *testing.T, client *megaport.Client, mcrUID string, pflID int) *megaport.MCRPrefixFilterList {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	list, err := client.MCRService.GetMCRPrefixFilterList(ctx, mcrUID, pflID)
	require.NoErrorf(t, err, "SDK get prefix filter list %d", pflID)
	require.NotNil(t, list, "SDK returned nil prefix filter list")
	return list
}

// assertPrefixEntries checks the SDK-returned entries match the expected set.
// Prefixes are paired in canonical form so IPv6 normalization is tolerated;
// order is not assumed. ge/le are compared exactly, on the assumption the API
// echoes them back unchanged; staging normalizing them would surface here.
func assertPrefixEntries(t *testing.T, got []*megaport.MCRPrefixListEntry, want []prefixEntry) {
	t.Helper()
	require.Len(t, got, len(want), "entry count")

	byPrefix := make(map[string]*megaport.MCRPrefixListEntry, len(got))
	for _, e := range got {
		byPrefix[canonicalPrefix(t, e.Prefix)] = e
	}
	for _, w := range want {
		e, ok := byPrefix[canonicalPrefix(t, w.prefix)]
		if !assert.Truef(t, ok, "expected an entry for prefix %s", w.prefix) {
			continue
		}
		assert.Equalf(t, w.action, e.Action, "action for %s", w.prefix)
		assert.Equalf(t, w.ge, e.Ge, "ge for %s", w.prefix)
		assert.Equalf(t, w.le, e.Le, "le for %s", w.prefix)
	}
}

// TestIntegration_MCRIPv6PrefixFilterLifecycle mirrors the IPv4 prefix-filter
// coverage in TestIntegration_MCRLifecycle but for an IPv6 list: buy an MCR,
// create an IPv6 prefix filter list, then get/update/delete it. Each step is
// driven through the CLI action functions and asserted against the SDK.
//
// Routing/BGP scope (ESD-1531): the CLI exposes no standalone command to
// configure static IP routes or BGP peering on an MCR, and neither does the
// SDK's MCRService. Those are set through a VXC's A-end MCR partner config:
// static IP routes are exercised by the VXC MCR-vrouter integration test, while
// BGP peering is configured the same way but is not yet covered by an
// integration test. The MCR looking-glass route/BGP commands are read-only
// diagnostics. So prefix filter lists are the only standalone MCR routing config
// to cover here, and there is no TestIntegration_MCRRouteConfigLifecycle. See
// docs/INTEGRATION_TESTING.md.
func TestIntegration_MCRIPv6PrefixFilterLifecycle(t *testing.T) {
	testutil.RequireStagingForProvisioning(t)
	client := testutil.SetupIntegrationClient(t)
	// Restore login via t.Cleanup, not defer: defers run before t.Cleanup, so a
	// deferred restore would swap back the default login before the
	// resource-deletion cleanups below get to run.
	t.Cleanup(testutil.LoginWithClient(t, client))

	// Action functions mutate the process-wide output format; restore it so
	// test order can't leak state between tests in this package.
	origFmt := output.GetOutputFormat()
	t.Cleanup(func() { output.SetOutputFormat(origFmt) })

	name := fmt.Sprintf("CLI-Test-MCR-IPv6-%s", generateUniqueID())
	locationID := testutil.FindMCRTestLocation(t, client, 1000, stagingMCRLocationID)

	// Buy a new MCR using flags. BuyMCR waits for provisioning (no --no-wait),
	// so the MCR is ready for prefix-filter-list operations once it returns.
	buyCmd := integrationMCRBuyCmd()
	require.NoError(t, buyCmd.Flags().Set("name", name))
	require.NoError(t, buyCmd.Flags().Set("term", "1"))
	require.NoError(t, buyCmd.Flags().Set("port-speed", "1000"))
	require.NoError(t, buyCmd.Flags().Set("location-id", strconv.Itoa(locationID)))
	require.NoError(t, buyCmd.Flags().Set("marketplace-visibility", "false"))
	require.NoError(t, buyCmd.Flags().Set("yes", "true"))

	var buyErr error
	buyOut := captureTableOutput(func() { buyErr = BuyMCR(buyCmd, nil, true) })
	require.NoError(t, buyErr, "buy MCR output: %s", buyOut)

	mcrUID := parseCreatedUID(buyOut)
	// Register cleanup before asserting on mcrUID, so any created MCR is
	// deleted even if the UID parse fails.
	t.Cleanup(func() {
		if mcrUID == "" {
			return
		}
		delCmd := integrationMCRDeleteCmd()
		_ = delCmd.Flags().Set("force", "true")
		var delErr error
		out := captureTableOutput(func() { delErr = DeleteMCR(delCmd, []string{mcrUID}, true) })
		if delErr != nil {
			t.Logf("cleanup: delete MCR %s failed: %v; output: %s", mcrUID, delErr, out)
			return
		}
		t.Logf("cleanup: delete MCR %s: %s", mcrUID, out)
	})
	require.NotEmpty(t, mcrUID, "could not parse MCR UID from: %s", buyOut)

	// Create an IPv6 prefix filter list via the CLI. Uses RFC 3849
	// documentation prefixes (2001:db8::/32) so the entries are realistic but
	// never collide with real address space. ge is set above each prefix
	// length to avoid the ge==prefix-length edge case.
	createPFLJSON := `{
		"description": "Test IPv6 Prefix Filter List",
		"addressFamily": "IPv6",
		"entries": [
			{"action": "permit", "prefix": "2001:db8::/32", "ge": 48, "le": 64},
			{"action": "deny", "prefix": "2001:db8:abcd::/48", "ge": 56, "le": 64}
		]
	}`

	createCmd := integrationMCRPrefixFilterCmd()
	require.NoError(t, createCmd.Flags().Set("json", createPFLJSON))
	var createErr error
	createOut := captureTableOutput(func() {
		createErr = CreateMCRPrefixFilterList(createCmd, []string{mcrUID}, true)
	})
	require.NoError(t, createErr, "create IPv6 prefix filter list output: %s", createOut)

	pflID := parsePrefixFilterListID(createOut)
	require.NotEmpty(t, pflID, "could not parse prefix filter list ID from: %s", createOut)

	// Best-effort cleanup, registered right after parsing the ID so a later
	// failure can't leak the list. Registered after the MCR cleanup so it runs
	// first (cleanups run LIFO), ensuring the list is gone before the MCR.
	t.Cleanup(func() {
		var delErr error
		out := captureTableOutput(func() {
			delErr = DeleteMCRPrefixFilterList(&cobra.Command{Use: "delete"}, []string{mcrUID, pflID}, true)
		})
		if delErr != nil {
			t.Logf("cleanup: delete prefix filter list %s failed: %v; output: %s", pflID, delErr, out)
			return
		}
		t.Logf("cleanup: delete prefix filter list %s: %s", pflID, out)
	})

	pflIDInt, err := strconv.Atoi(pflID)
	require.NoErrorf(t, err, "prefix filter list ID %q should be numeric", pflID)

	// Assert the created list via the SDK (authoritative read).
	created := getPrefixFilterListViaSDK(t, client, mcrUID, pflIDInt)
	assert.Equal(t, "IPv6", created.AddressFamily, "address family should be IPv6")
	assert.Equal(t, "Test IPv6 Prefix Filter List", created.Description)
	assertPrefixEntries(t, created.Entries, []prefixEntry{
		{action: "permit", prefix: "2001:db8::/32", ge: megaport.PtrTo(48), le: megaport.PtrTo(64)},
		{action: "deny", prefix: "2001:db8:abcd::/48", ge: megaport.PtrTo(56), le: megaport.PtrTo(64)},
	})

	// Update the list via the CLI, adding a third IPv6 entry.
	updatePFLJSON := `{
		"description": "Test IPv6 Prefix Filter List Updated",
		"addressFamily": "IPv6",
		"entries": [
			{"action": "permit", "prefix": "2001:db8::/32", "ge": 48, "le": 64},
			{"action": "deny", "prefix": "2001:db8:abcd::/48", "ge": 56, "le": 64},
			{"action": "permit", "prefix": "2001:db8:1234::/48", "ge": 56, "le": 64}
		]
	}`
	updateCmd := integrationMCRPrefixFilterCmd()
	require.NoError(t, updateCmd.Flags().Set("json", updatePFLJSON))
	var updateErr error
	updateOut := captureTableOutput(func() {
		updateErr = UpdateMCRPrefixFilterList(updateCmd, []string{mcrUID, pflID}, true)
	})
	require.NoError(t, updateErr, "update IPv6 prefix filter list output: %s", updateOut)

	// Assert the update landed via the SDK.
	updated := getPrefixFilterListViaSDK(t, client, mcrUID, pflIDInt)
	assert.Equal(t, "IPv6", updated.AddressFamily, "address family should remain IPv6")
	assert.Equal(t, "Test IPv6 Prefix Filter List Updated", updated.Description)
	assertPrefixEntries(t, updated.Entries, []prefixEntry{
		{action: "permit", prefix: "2001:db8::/32", ge: megaport.PtrTo(48), le: megaport.PtrTo(64)},
		{action: "deny", prefix: "2001:db8:abcd::/48", ge: megaport.PtrTo(56), le: megaport.PtrTo(64)},
		{action: "permit", prefix: "2001:db8:1234::/48", ge: megaport.PtrTo(56), le: megaport.PtrTo(64)},
	})

	// Delete via the CLI and assert it is gone via the SDK.
	var deleteErr error
	deleteOut := captureTableOutput(func() {
		deleteErr = DeleteMCRPrefixFilterList(&cobra.Command{Use: "delete"}, []string{mcrUID, pflID}, true)
	})
	require.NoError(t, deleteErr, "delete IPv6 prefix filter list output: %s", deleteOut)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	_, getErr := client.MCRService.GetMCRPrefixFilterList(ctx, mcrUID, pflIDInt)
	assert.Error(t, getErr, "IPv6 prefix filter list should no longer exist")
}
