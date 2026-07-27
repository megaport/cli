package mcr

import (
	"fmt"

	"github.com/megaport/megaport-cli/internal/base/cmdbuilder"
	"github.com/megaport/megaport-cli/internal/validation"
	"github.com/spf13/cobra"
)

// AddCommandsTo builds the mcr commands and adds them to the root command
func AddCommandsTo(rootCmd *cobra.Command) {
	// Create mcr parent command
	mcrCmd := cmdbuilder.NewCommand("mcr", "Manage MCRs in the Megaport API").
		WithLongDesc("Manage MCRs in the Megaport API.\n\nThis command groups all operations related to Megaport Cloud Routers (MCRs). MCRs are virtual routing appliances that run in the Megaport network, providing interconnection between your cloud environments and the Megaport fabric.").
		WithExample("megaport-cli mcr get [mcrUID]").
		WithExample("megaport-cli mcr list --location-id 67").
		WithExample("megaport-cli mcr buy").
		WithExample("megaport-cli mcr update [mcrUID]").
		WithExample("megaport-cli mcr delete [mcrUID]").
		WithImportantNote("With MCRs you can establish virtual cross-connects (VXCs) to cloud service providers").
		WithImportantNote("Create private network connections between different cloud regions").
		WithImportantNote("Implement hybrid cloud architectures with seamless connectivity").
		WithImportantNote("Peer with other networks using BGP routing").
		WithRootCmd(rootCmd).
		Build()

	get, buy, update, del, restore, lock, unlock, list, status, validate := buildMCRCommands(rootCmd)
	create, listPFL, getPFL, updatePFL, deletePFL := buildMCRPrefixFilterCommands(rootCmd)
	listTags, updateTags := buildMCRTagCommands()
	addIPSec, updateIPSec := buildMCRIPSecCommands(rootCmd)
	lookingGlass := buildMCRLookingGlassCommands(rootCmd)

	mcrCmd.AddCommand(
		get, buy, update, del, restore, lock, unlock,
		create, listPFL, getPFL, updatePFL, deletePFL,
		list, status, validate,
		listTags, updateTags,
		addIPSec, updateIPSec,
		lookingGlass,
	)
	rootCmd.AddCommand(mcrCmd)
}

// buildMCRCommands extracts the get, buy, update, delete, restore, list, and status command definitions.
func buildMCRCommands(rootCmd *cobra.Command) (get, buy, update, del, restore, lock, unlock, list, status, validate *cobra.Command) {
	// Create get MCR command
	get = cmdbuilder.NewCommand("get", "Get details for a single MCR").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(GetMCR).
		WithBoolFlag("export", false, "Output recreatable JSON config for use with buy --json (excludes read-only fields)").
		WithWatchFlags().
		WithLongDesc("Get details for a single MCR.\n\nThis command retrieves and displays detailed information for a single Megaport Cloud Router (MCR). You must provide the unique identifier (UID) of the MCR you wish to retrieve.").
		WithExample("megaport-cli mcr get a1b2c3d4-e5f6-7890-1234-567890abcdef").
		WithExample("megaport-cli mcr get a1b2c3d4-e5f6-7890-1234-567890abcdef --export").
		WithExample("megaport-cli mcr get a1b2c3d4-e5f6-7890-1234-567890abcdef --watch").
		WithExample("megaport-cli mcr get a1b2c3d4-e5f6-7890-1234-567890abcdef --watch --interval 10s").
		WithImportantNote("The output includes the MCR's UID, name, location ID, port speed, and provisioning status").
		WithRootCmd(rootCmd).
		WithAliases([]string{"show"}).
		Build()

	// Create buy MCR command
	buy = cmdbuilder.NewCommand("buy", "Buy an MCR through the Megaport API").
		WithArgs(cobra.NoArgs).
		WithColorAwareRunFunc(BuyMCR).
		WithNoWaitFlag().
		WithBuyConfirmFlags().
		WithMCRCreateFlags().
		WithStandardInputFlags().
		WithLongDesc("Buy an MCR through the Megaport API.\n\nThis command allows you to purchase an MCR by providing the necessary details.").
		WithDocumentedRequiredFlag("name", "The name of the MCR (1-64 characters)").
		WithDocumentedRequiredFlag("term", fmt.Sprintf("The term of the MCR (%s months)", validation.FormatIntSlice(validation.ValidContractTerms))).
		WithDocumentedRequiredFlag("port-speed", "The speed of the MCR (1000, 2500, 5000, 10000, 25000, 50000, or 100000 Mbps)").
		WithDocumentedRequiredFlag("location-id", "The ID of the location where the MCR will be provisioned").
		WithDocumentedRequiredFlag("marketplace-visibility", "Whether the MCR should be visible in the marketplace (true or false)").
		WithOptionalFlag("mcr-asn", "The ASN for the MCR (if not provided, a private ASN will be assigned)").
		WithOptionalFlag("diversity-zone", "The diversity zone for the MCR").
		WithOptionalFlag("cost-centre", "The cost centre for the MCR").
		WithOptionalFlag("promo-code", "A promotional code for the MCR").
		WithOptionalFlag("resource-tags", "JSON string of key-value pairs for resource tagging").
		WithIntFlag("ipsec-tunnel-count", 0, "IPSec tunnel count for an add-on (10, 20, or 30)").
		WithOptionalFlag("ipsec-tunnel-count", "IPSec tunnel count for an add-on (10, 20, or 30); omit to skip IPSec; set to 0 to include with API default (10)").
		WithExample("megaport-cli mcr buy --interactive").
		WithExample("megaport-cli mcr buy --name \"My MCR\" --term 12 --port-speed 5000 --location-id 123 --marketplace-visibility=true --mcr-asn 65000").
		WithExample("megaport-cli mcr buy --name \"My MCR\" --term 12 --port-speed 5000 --location-id 123 --marketplace-visibility=true --resource-tags '{\"env\":\"prod\",\"owner\":\"network-team\"}'").
		WithExample("megaport-cli mcr buy --json '{\"name\":\"My MCR\",\"term\":12,\"portSpeed\":5000,\"locationId\":123,\"mcrAsn\":65000,\"marketplaceVisibility\":true}'").
		WithExample("megaport-cli mcr buy --json-file ./mcr-config.json").
		WithJSONExample(`{
  "name": "My MCR",
  "term": 12,
  "portSpeed": 5000,
  "locationId": 123,
  "mcrAsn": 65000,
  "marketplaceVisibility": true,
  "diversityZone": "blue",
  "costCentre": "IT-Networking",
  "promoCode": "SUMMER2024",
  "tunnelCount": 10,
  "resourceTags": {
    "environment": "production",
    "department": "networking",
    "project": "cloud-migration",
    "owner": "john.doe@example.com"
  }
}`).
		WithImportantNote("The location_id must correspond to a valid location in the Megaport API").
		WithImportantNote("The port_speed must be one of the supported speeds (1000, 2500, 5000, or 10000 Mbps)").
		WithImportantNote("If mcr_asn is not provided, a private ASN will be automatically assigned").
		WithImportantNote("Resource tags allow you to categorize resources for organization and billing purposes").
		WithImportantNote("Required flags (name, term, port-speed, location-id, marketplace-visibility) can be skipped when using --interactive, --json, or --json-file").
		WithImportantNote("IPSec add-on (--ipsec-tunnel-count) is not prompted in interactive mode; use --ipsec-tunnel-count flag or include 'tunnelCount' in the JSON input").
		WithImportantNote("Use --marketplace-visibility=true or --marketplace-visibility=false (with an equals sign); a space-separated value is not consumed by the flag and is rejected as an unexpected argument").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("name", "term", "port-speed", "location-id", "marketplace-visibility").
		Build()

	update = cmdbuilder.NewCommand("update", "Update an existing MCR").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(UpdateMCR).
		WithStandardInputFlags().
		WithMCRUpdateFlags().
		WithLongDesc("Update an existing Megaport Cloud Router (MCR).\n\nThis command allows you to update the details of an existing MCR.").
		WithExample("megaport-cli mcr update [mcrUID] --interactive").
		WithExample("megaport-cli mcr update [mcrUID] --name \"Updated MCR\" --marketplace-visibility true --cost-centre \"Finance\"").
		WithExample("megaport-cli mcr update [mcrUID] --term 24").
		WithExample("megaport-cli mcr update [mcrUID] --json '{\"name\":\"Updated MCR\",\"marketplaceVisibility\":true,\"costCentre\":\"Finance\"}'").
		WithExample("megaport-cli mcr update [mcrUID] --mcr-asn 65020").
		WithExample("megaport-cli mcr update [mcrUID] --json '{\"name\":\"Updated MCR\",\"marketplaceVisibility\":true,\"costCentre\":\"Finance\",\"mcrAsn\":65020}'").
		WithExample("megaport-cli mcr update [mcrUID] --json-file ./update-mcr-config.json").
		WithJSONExample(`{
  "name": "Updated MCR",
  "marketplaceVisibility": true,
  "costCentre": "Finance",
  "contractTermMonths": 24,
  "mcrAsn": 65020
}`).
		WithImportantNote("The MCR UID cannot be changed").
		WithImportantNote("Only specified fields will be updated; unspecified fields will remain unchanged").
		WithImportantNote("Updating mcr-asn re-runs BGP setup on the MCR; existing peering sessions may flap during the change").
		WithImportantNote("Ensure the JSON file is correctly formatted").
		WithRootCmd(rootCmd).
		Build()

	// Create delete MCR command
	del = cmdbuilder.NewCommand("delete", "Delete an MCR from your account").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(DeleteMCR).
		WithImmediateSafeDeleteFlags().
		WithLongDesc("Delete an MCR from your account.\n\nMCRs are deleted immediately; deferred end-of-term cancellation is not supported by the API.").
		WithExample("megaport-cli mcr delete [mcrUID]").
		WithExample("megaport-cli mcr delete [mcrUID] --force").
		WithExample("megaport-cli mcr delete [mcrUID] --force --safe-delete").
		WithImportantNote("Deletion is immediate; billing stops right away").
		WithImportantNote("Deletion is final and cannot be undone").
		WithRootCmd(rootCmd).
		WithAliases([]string{"rm"}).
		Build()

	// Create restore MCR command
	restore = cmdbuilder.NewCommand("restore", "Restore a deleted MCR").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(RestoreMCR).
		WithLongDesc("Restore a previously deleted MCR.\n\nThis command allows you to restore a previously deleted MCR, provided it has not yet been fully decommissioned.").
		WithExample("megaport-cli mcr restore [mcrUID]").
		WithRootCmd(rootCmd).
		Build()

	// Create lock MCR command
	lock = cmdbuilder.NewCommand("lock", "Lock an MCR").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(LockMCR).
		WithLongDesc("Lock an MCR to prevent modifications.\n\nThis command locks a Megaport Cloud Router (MCR) to prevent any changes from being made to it. Use the unlock command to re-enable modifications.").
		WithExample("megaport-cli mcr lock [mcrUID]").
		WithRootCmd(rootCmd).
		Build()

	// Create unlock MCR command
	unlock = cmdbuilder.NewCommand("unlock", "Unlock an MCR").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(UnlockMCR).
		WithLongDesc("Unlock a previously locked MCR.\n\nThis command unlocks a Megaport Cloud Router (MCR) that was previously locked, allowing modifications to be made again.").
		WithExample("megaport-cli mcr unlock [mcrUID]").
		WithRootCmd(rootCmd).
		Build()

	// Create list MCRs command
	list = cmdbuilder.NewCommand("list", "List all MCRs with optional filters").
		WithOutputFormatRunFunc(ListMCRs).
		WithLongDesc("List all MCRs available in the Megaport API.\n\nThis command fetches and displays a list of MCRs with details such as MCR ID, name, location, speed, and status. By default, only active MCRs are shown. You can also filter by resource tags.").
		WithMCRFilterFlags().
		WithTagFilterFlags().
		WithOptionalFlag("location-id", "Filter MCRs by location ID").
		WithOptionalFlag("name", "Filter MCRs by name").
		WithOptionalFlag("port-speed", "Filter MCRs by port speed").
		WithOptionalFlag("include-inactive", "Include MCRs in CANCELLED, DECOMMISSIONED, or DECOMMISSIONING states").
		WithExample("megaport-cli mcr list").
		WithExample("megaport-cli mcr list --location-id 1").
		WithExample("megaport-cli mcr list --port-speed 10000").
		WithExample("megaport-cli mcr list --name \"My MCR\"").
		WithExample("megaport-cli mcr list --include-inactive").
		WithExample("megaport-cli mcr list --location-id 1 --port-speed 10000 --name \"My MCR\"").
		WithExample("megaport-cli mcr list --tag env=prod").
		WithExample("megaport-cli mcr list --tag env=prod --tag team=network").
		WithIntFlag("limit", 0, "Maximum number of results to display (0 = unlimited)").
		WithRootCmd(rootCmd).
		WithAliases([]string{"ls"}).
		Build()

	// Create status MCR command
	status = cmdbuilder.NewCommand("status", "Check the provisioning status of an MCR").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(GetMCRStatus).
		WithWatchFlags().
		WithLongDesc("Check the provisioning status of an MCR through the Megaport API.\n\nThis command retrieves only the essential status information for a Megaport Cloud Router (MCR) without all the details. It's useful for monitoring ongoing provisioning.").
		WithExample("megaport-cli mcr status mcr-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx").
		WithExample("megaport-cli mcr status mcr-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --watch").
		WithExample("megaport-cli mcr status mcr-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --watch --interval 10s").
		WithImportantNote("This is a lightweight command that only shows the MCR's status without retrieving all details.").
		WithRootCmd(rootCmd).
		WithAliases([]string{"st"}).
		Build()

	validate = cmdbuilder.NewCommand("validate", "Validate an MCR order without purchasing").
		WithArgs(cobra.NoArgs).
		WithColorAwareRunFunc(ValidateMCR).
		WithMCRCreateFlags().
		WithStandardInputFlags().
		WithLongDesc("Validates an MCR configuration against the Megaport API without creating the resource.\n\nUse this for dry-run validation before purchasing, or in CI pipelines to check configurations.").
		WithDocumentedRequiredFlag("name", "The name of the MCR (1-64 characters)").
		WithDocumentedRequiredFlag("term", fmt.Sprintf("The term of the MCR (%s months)", validation.FormatIntSlice(validation.ValidContractTerms))).
		WithDocumentedRequiredFlag("port-speed", "The speed of the MCR (1000, 2500, 5000, 10000, 25000, 50000, or 100000 Mbps)").
		WithDocumentedRequiredFlag("location-id", "The ID of the location where the MCR will be provisioned").
		WithDocumentedRequiredFlag("marketplace-visibility", "Whether the MCR should be visible in the marketplace (true or false)").
		WithExample(`megaport-cli mcr validate --name "My MCR" --term 12 --port-speed 5000 --location-id 123 --marketplace-visibility=true`).
		WithExample("megaport-cli mcr validate --json-file ./mcr-config.json").
		WithImportantNote("This command only validates the configuration — no resources are created and no charges are incurred").
		WithImportantNote("Use --marketplace-visibility=true or --marketplace-visibility=false (with an equals sign); a space-separated value is not consumed by the flag and is rejected as an unexpected argument").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("name", "term", "port-speed", "location-id", "marketplace-visibility").
		Build()

	return
}

// buildMCRPrefixFilterCommands extracts the prefix filter list command definitions.
func buildMCRPrefixFilterCommands(rootCmd *cobra.Command) (create, listPFL, getPFL, updatePFL, deletePFL *cobra.Command) {
	// Create prefix filter list command
	create = cmdbuilder.NewCommand("create-prefix-filter-list", "Create a prefix filter list on an MCR").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(CreateMCRPrefixFilterList).
		WithStandardInputFlags().
		WithLongDesc("Create a prefix filter list on an MCR.\n\nThis command allows you to create a new prefix filter list on an MCR. Prefix filter lists are used to control which routes are accepted or advertised by the MCR.").
		WithMCRPrefixFilterListFlags().
		WithExample("megaport-cli mcr create-prefix-filter-list [mcrUID] --interactive").
		WithExample("megaport-cli mcr create-prefix-filter-list [mcrUID] --description \"My prefix list\" --address-family \"IPv4\" --entries '[{\"action\":\"permit\",\"prefix\":\"10.0.0.0/8\",\"ge\":24,\"le\":32}]'").
		WithExample("megaport-cli mcr create-prefix-filter-list [mcrUID] --json '{\"description\":\"My prefix list\",\"addressFamily\":\"IPv4\",\"entries\":[{\"action\":\"permit\",\"prefix\":\"10.0.0.0/8\",\"ge\":24,\"le\":32}]}'").
		WithExample("megaport-cli mcr create-prefix-filter-list [mcrUID] --json-file ./prefix-list-config.json").
		WithJSONExample(`{
  "description": "My prefix list",
  "addressFamily": "IPv4",
  "entries": [
    {
      "action": "permit",
      "prefix": "10.0.0.0/8",
      "ge": 24,
      "le": 32
    },
    {
      "action": "deny",
      "prefix": "0.0.0.0/0"
    }
  ]
}`).
		WithImportantNote("The address_family must be either \"IPv4\" or \"IPv6\"").
		WithImportantNote("The entries must be a valid JSON array of prefix filter entries").
		WithImportantNote("The ge and le values are optional but must be within the range of the prefix length").
		WithImportantNote("Required flags (description, address-family, entries) can be skipped when using --interactive, --json, or --json-file").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("description", "address-family", "entries").
		Build()

	// List prefix filter lists command
	listPFL = cmdbuilder.NewCommand("list-prefix-filter-lists", "List all prefix filter lists for a specific MCR").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(ListMCRPrefixFilterLists).
		WithLongDesc("List all prefix filter lists for a specific MCR.\n\nThis command retrieves and displays a list of all prefix filter lists configured on the specified MCR.").
		WithExample("megaport-cli mcr list-prefix-filter-lists [mcrUID]").
		WithRootCmd(rootCmd).
		Build()

	// Get prefix filter list command
	getPFL = cmdbuilder.NewCommand("get-prefix-filter-list", "Get details for a single prefix filter list on an MCR").
		WithArgs(cobra.ExactArgs(2)).
		WithOutputFormatRunFunc(GetMCRPrefixFilterList).
		WithLongDesc("Get details for a single prefix filter list on an MCR.\n\nThis command retrieves and displays detailed information about a specific prefix filter list on the specified MCR.").
		WithExample("megaport-cli mcr get-prefix-filter-list [mcrUID] [prefixFilterListID]").
		WithRootCmd(rootCmd).
		Build()

	// Update prefix filter list command
	updatePFL = cmdbuilder.NewCommand("update-prefix-filter-list", "Update a prefix filter list on an MCR").
		WithArgs(cobra.ExactArgs(2)).
		WithColorAwareRunFunc(UpdateMCRPrefixFilterList).
		WithStandardInputFlags().
		WithMCRPrefixFilterListFlags().
		WithLongDesc("Update a prefix filter list on an MCR.\n\nThis command allows you to update the details of an existing prefix filter list on an MCR. You can use this command to modify the description, address family, or entries in the list.").
		WithExample("megaport-cli mcr update-prefix-filter-list [mcrUID] [prefixFilterListID] --interactive").
		WithExample("megaport-cli mcr update-prefix-filter-list [mcrUID] [prefixFilterListID] --description \"Updated prefix list\" --entries '[{\"action\":\"permit\",\"prefix\":\"10.0.0.0/8\",\"ge\":24,\"le\":32}]'").
		WithExample("megaport-cli mcr update-prefix-filter-list [mcrUID] [prefixFilterListID] --json '{\"description\":\"Updated prefix list\",\"entries\":[{\"action\":\"permit\",\"prefix\":\"10.0.0.0/8\",\"ge\":24,\"le\":32}]}'").
		WithExample("megaport-cli mcr update-prefix-filter-list [mcrUID] [prefixFilterListID] --json-file ./update-prefix-list.json").
		WithJSONExample(`{
  "description": "Updated prefix list",
  "addressFamily": "IPv4",
  "entries": [
    {
      "action": "permit",
      "prefix": "10.0.0.0/8",
      "ge": 24,
      "le": 32
    },
    {
      "action": "deny",
      "prefix": "0.0.0.0/0"
    }
  ]
}`).
		WithRootCmd(rootCmd).
		Build()

	// Delete prefix filter list command
	deletePFL = cmdbuilder.NewCommand("delete-prefix-filter-list", "Delete a prefix filter list on an MCR").
		WithArgs(cobra.ExactArgs(2)).
		WithColorAwareRunFunc(DeleteMCRPrefixFilterList).
		WithDeleteFlags().
		WithLongDesc("Delete a prefix filter list on an MCR.\n\nThis command allows you to delete a prefix filter list from the specified MCR.").
		WithExample("megaport-cli mcr delete-prefix-filter-list [mcrUID] [prefixFilterListID]").
		WithExample("megaport-cli mcr delete-prefix-filter-list [mcrUID] [prefixFilterListID] --force").
		WithRootCmd(rootCmd).
		Build()

	return
}

// buildMCRTagCommands extracts the tag command definitions.
func buildMCRTagCommands() (listTags, updateTags *cobra.Command) {
	// Add list-tags command
	listTags = cmdbuilder.NewCommand("list-tags", "List resource tags on a specific MCR").
		WithLongDesc("Lists all resource tags associated with a specific MCR").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(ListMCRResourceTags).
		WithExample("megaport-cli mcr list-tags mcr-abc123").
		Build()

	// Add update-tags command
	updateTags = cmdbuilder.NewCommand("update-tags", "Update resource tags on a specific MCR").
		WithLongDesc("Update resource tags associated with a specific MCR. Tags can be provided via interactive prompts, JSON string, or JSON file.").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(UpdateMCRResourceTags).
		WithStandardInputFlags().
		WithBoolFlag("force", false, "Skip the confirmation prompt").
		WithExample("megaport-cli mcr update-tags mcr-abc123 --interactive").
		WithExample("megaport-cli mcr update-tags mcr-abc123 --json '{\"env\":\"production\",\"team\":\"network\"}'").
		WithExample("megaport-cli mcr update-tags mcr-abc123 --json-file ./tags.json").
		WithJSONExample(`{
  "environment": "production",
  "owner": "network-team",
  "project": "cloud-migration"
}`).
		WithImportantNote("All existing tags will be replaced with the provided tags. To clear all tags, provide an empty tag set.").
		Build()

	return
}

// buildMCRIPSecCommands extracts the IPSec add-on command definitions.
func buildMCRIPSecCommands(rootCmd *cobra.Command) (addIPSec, updateIPSec *cobra.Command) {
	addIPSec = cmdbuilder.NewCommand("add-ipsec-addon", "Add an IPSec add-on to an existing MCR").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(AddMCRIPSecAddOn).
		WithStandardInputFlags().
		WithMCRIPSecAddOnFlags().
		WithLongDesc("Add an IPSec add-on to an existing MCR.\n\nThis command provisions an IPSec add-on on the specified MCR. IPSec add-ons enable encrypted tunnel termination on the MCR.").
		WithOptionalFlag("tunnel-count", "Number of IPSec tunnels (10, 20, or 30); omit or set to 0 to use the API default of 10").
		WithExample("megaport-cli mcr add-ipsec-addon [mcrUID] --tunnel-count 10").
		WithExample("megaport-cli mcr add-ipsec-addon [mcrUID] --tunnel-count 20").
		WithExample("megaport-cli mcr add-ipsec-addon [mcrUID] --json '{\"tunnelCount\":10}'").
		WithExample("megaport-cli mcr add-ipsec-addon [mcrUID] --interactive").
		WithJSONExample(`{
  "tunnelCount": 10
}`).
		WithImportantNote("Valid tunnel counts are 10, 20, or 30. Omit --tunnel-count or set to 0 to use the API default (10).").
		WithImportantNote("You must provide one of: --tunnel-count, --interactive, --json, or --json-file").
		WithRootCmd(rootCmd).
		Build()

	updateIPSec = cmdbuilder.NewCommand("update-ipsec-addon", "Update or disable an IPSec add-on on an MCR").
		WithArgs(cobra.ExactArgs(2)).
		WithColorAwareRunFunc(UpdateMCRIPSecAddOn).
		WithStandardInputFlags().
		WithMCRUpdateIPSecAddOnFlags().
		WithLongDesc("Update or disable an existing IPSec add-on on an MCR.\n\nThis command updates the tunnel count on an existing IPSec add-on. Set tunnel-count to 0 to disable the IPSec add-on.").
		WithDocumentedRequiredFlag("tunnel-count", "New tunnel count (10, 20, or 30); set to 0 to disable IPSec").
		WithExample("megaport-cli mcr update-ipsec-addon [mcrUID] [addOnUID] --tunnel-count 20").
		WithExample("megaport-cli mcr update-ipsec-addon [mcrUID] [addOnUID] --tunnel-count 0").
		WithExample("megaport-cli mcr update-ipsec-addon [mcrUID] [addOnUID] --json '{\"tunnelCount\":30}'").
		WithExample("megaport-cli mcr update-ipsec-addon [mcrUID] [addOnUID] --interactive").
		WithJSONExample(`{
  "tunnelCount": 30
}`).
		WithImportantNote("Valid tunnel counts are 10, 20, or 30. Set to 0 to disable the IPSec add-on.").
		WithImportantNote("--tunnel-count can be skipped when using --interactive, --json, or --json-file").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("tunnel-count").
		Build()

	return
}

// buildMCRLookingGlassCommands extracts the Looking Glass diagnostic command definitions.
func buildMCRLookingGlassCommands(rootCmd *cobra.Command) (lookingGlass *cobra.Command) {
	lookingGlass = cmdbuilder.NewCommand("looking-glass", "MCR Looking Glass diagnostic commands").
		WithLongDesc("MCR Looking Glass diagnostic commands.\n\nThe Looking Glass provides visibility into traffic routing on your MCR, helping you troubleshoot connections by showing the status of protocols and routing tables.").
		WithExample("megaport-cli mcr looking-glass ip-routes [mcrUID]").
		WithExample("megaport-cli mcr looking-glass bgp-routes [mcrUID]").
		WithExample("megaport-cli mcr looking-glass bgp-sessions [mcrUID]").
		WithImportantNote("Looking Glass commands are read-only diagnostic tools").
		WithImportantNote("Use --ip flag to filter routes by IP address or prefix").
		WithRootCmd(rootCmd).
		Build()

	lgIPRoutesCmd := cmdbuilder.NewCommand("ip-routes", "List IP routes from the MCR routing table").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(ListLookingGlassIPRoutes).
		WithLongDesc("List IP routes from the MCR Looking Glass.\n\nThis command retrieves all routes (BGP, static, connected, local) from the MCR's routing table. You can filter by protocol or IP address/prefix.").
		WithFlag("protocol", "", "Filter by protocol (BGP, STATIC, CONNECTED, LOCAL)").
		WithFlag("ip", "", "Filter by IP address or prefix (e.g., 10.0.0.0/8 or 192.168.1.1)").
		WithExample("megaport-cli mcr looking-glass ip-routes [mcrUID]").
		WithExample("megaport-cli mcr looking-glass ip-routes [mcrUID] --protocol BGP").
		WithExample("megaport-cli mcr looking-glass ip-routes [mcrUID] --ip 10.0.0.0/8").
		WithExample("megaport-cli mcr looking-glass ip-routes [mcrUID] --protocol STATIC --ip 192.168.0.0/16").
		WithImportantNote("Protocol values: BGP, STATIC, CONNECTED, LOCAL").
		WithRootCmd(rootCmd).
		Build()

	lgBGPRoutesCmd := cmdbuilder.NewCommand("bgp-routes", "List BGP routes with full BGP attributes").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(ListLookingGlassBGPRoutes).
		WithLongDesc("List BGP routes from the MCR Looking Glass.\n\nThis command retrieves routes learned via BGP with full BGP attributes including AS path, local preference, MED, communities, and origin.").
		WithFlag("ip", "", "Filter by IP address or prefix (e.g., 10.0.0.0/8 or 192.168.1.1)").
		WithExample("megaport-cli mcr looking-glass bgp-routes [mcrUID]").
		WithExample("megaport-cli mcr looking-glass bgp-routes [mcrUID] --ip 10.0.0.0/8").
		WithImportantNote("Shows BGP-specific attributes like AS path, local preference, MED, and communities").
		WithRootCmd(rootCmd).
		Build()

	lgBGPSessionsCmd := cmdbuilder.NewCommand("bgp-sessions", "List BGP sessions configured on the MCR").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(ListLookingGlassBGPSessions).
		WithLongDesc("List all BGP sessions configured on the MCR.\n\nThis command shows the status of all BGP peering sessions including neighbor address, ASN, session status, uptime, and prefix counts.").
		WithExample("megaport-cli mcr looking-glass bgp-sessions [mcrUID]").
		WithImportantNote("Session status can be UP, DOWN, or UNKNOWN").
		WithImportantNote("Use the session ID from this output with bgp-neighbor-routes command").
		WithRootCmd(rootCmd).
		Build()

	lgBGPNeighborRoutesCmd := cmdbuilder.NewCommand("bgp-neighbor-routes", "List routes advertised to or received from a BGP neighbor").
		WithArgs(cobra.ExactArgs(3)).
		WithOutputFormatRunFunc(ListLookingGlassBGPNeighborRoutes).
		WithLongDesc("List routes advertised to or received from a specific BGP neighbor.\n\nThis command shows routes that are either being advertised to a neighbor or received from a neighbor. Use the session ID from 'bgp-sessions' command.").
		WithFlag("ip", "", "Filter by IP address or prefix (e.g., 10.0.0.0/8 or 192.168.1.1)").
		WithExample("megaport-cli mcr looking-glass bgp-neighbor-routes [mcrUID] [sessionID] advertised").
		WithExample("megaport-cli mcr looking-glass bgp-neighbor-routes [mcrUID] [sessionID] received").
		WithExample("megaport-cli mcr looking-glass bgp-neighbor-routes [mcrUID] [sessionID] received --ip 10.0.0.0/8").
		WithImportantNote("Direction must be 'advertised' or 'received'").
		WithImportantNote("Get session ID from 'mcr looking-glass bgp-sessions' command").
		WithRootCmd(rootCmd).
		Build()

	lgPingCmd := cmdbuilder.NewCommand("ping", "Run a ping from the MCR to a destination").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(LookingGlassPing).
		WithLongDesc("Run an ICMP ping from the MCR Looking Glass to a destination address.\n\nThis command starts a ping operation on the MCR, polls until it completes, then prints the result including RTT statistics and packet loss.").
		WithFlag("destination", "", "Destination IP address to ping").
		WithRequiredFlag("destination", "Destination IP address to ping").
		WithFlag("source", "", "Source IP address to ping from").
		WithIntFlag("packet-count", 0, "Number of packets to send (1-60)").
		WithIntFlag("packet-size", 0, "Packet size in bytes (1-9186)").
		WithExample("megaport-cli mcr looking-glass ping [mcrUID] --destination 8.8.8.8").
		WithExample("megaport-cli mcr looking-glass ping [mcrUID] --destination 8.8.8.8 --source 10.0.0.1").
		WithExample("megaport-cli mcr looking-glass ping [mcrUID] --destination 8.8.8.8 --packet-count 10 --packet-size 128").
		WithRootCmd(rootCmd).
		Build()

	lgTracerouteCmd := cmdbuilder.NewCommand("traceroute", "Run a traceroute from the MCR to a destination").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(LookingGlassTraceroute).
		WithLongDesc("Run a traceroute from the MCR Looking Glass to a destination address.\n\nThis command starts a traceroute operation on the MCR, polls until it completes, then prints the hops.").
		WithFlag("destination", "", "Destination IP address to traceroute to").
		WithRequiredFlag("destination", "Destination IP address to traceroute to").
		WithFlag("source", "", "Source IP address to traceroute from").
		WithExample("megaport-cli mcr looking-glass traceroute [mcrUID] --destination 8.8.8.8").
		WithExample("megaport-cli mcr looking-glass traceroute [mcrUID] --destination 8.8.8.8 --source 10.0.0.1").
		WithRootCmd(rootCmd).
		Build()

	lookingGlass.AddCommand(
		lgIPRoutesCmd,
		lgBGPRoutesCmd,
		lgBGPSessionsCmd,
		lgBGPNeighborRoutesCmd,
		lgPingCmd,
		lgTracerouteCmd,
	)

	return
}
