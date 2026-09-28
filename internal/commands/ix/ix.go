package ix

import (
	"github.com/megaport/megaport-cli/internal/base/cmdbuilder"
	"github.com/spf13/cobra"
)

// AddCommandsTo builds the ix commands and adds them to the root command
func AddCommandsTo(rootCmd *cobra.Command) {
	// Create ix parent command
	ixCmd := cmdbuilder.NewCommand("ix", "Manage Internet Exchanges (IXs) in the Megaport API").
		WithLongDesc("Manage Internet Exchanges (IXs) in the Megaport API.\n\nThis command groups all operations related to Megaport Internet Exchange connections. IXs allow you to connect to Internet Exchange points through the Megaport fabric.").
		WithExample("megaport-cli ix get [ixUID]").
		WithExample("megaport-cli ix list").
		WithExample("megaport-cli ix buy").
		WithExample("megaport-cli ix update [ixUID]").
		WithExample("megaport-cli ix delete [ixUID]").
		WithImportantNote("IXs allow you to connect to Internet Exchange points for peering").
		WithImportantNote("An IX is attached to an existing port via the product-uid flag").
		WithRootCmd(rootCmd).
		Build()

	// Create get IX command
	getIXCmd := cmdbuilder.NewCommand("get", "Get details for a single IX").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(GetIX).
		WithBoolFlag("export", false, "Output recreatable JSON config for use with buy --json (excludes read-only fields; productUid not available from API)").
		WithLongDesc("Get details for a single IX.\n\nThis command retrieves and displays detailed information for a single Internet Exchange (IX). You must provide the unique identifier (UID) of the IX you wish to retrieve.").
		WithExample("megaport-cli ix get a1b2c3d4-e5f6-7890-1234-567890abcdef").
		WithExample("megaport-cli ix get a1b2c3d4-e5f6-7890-1234-567890abcdef --export").
		WithImportantNote("The output includes the IX's UID, name, network service type, ASN, rate limit, VLAN, MAC address, and provisioning status").
		WithRootCmd(rootCmd).
		WithAliases([]string{"show"}).
		Build()

	// Create buy IX command
	buyIXCmd := cmdbuilder.NewCommand("buy", "Buy an IX through the Megaport API").
		WithColorAwareRunFunc(BuyIX).
		WithNoWaitFlag().
		WithBuyConfirmFlags().
		WithIXCreateFlags().
		WithStandardInputFlags().
		WithLongDesc("Buy an IX through the Megaport API.\n\nThis command allows you to purchase an IX by providing the necessary details.").
		WithDocumentedRequiredFlag("product-uid", "The UID of the port to attach the IX to").
		WithDocumentedRequiredFlag("name", "The name of the IX").
		WithDocumentedRequiredFlag("network-service-type", "The IX type/network service to connect to").
		WithDocumentedRequiredFlag("asn", "ASN (Autonomous System Number) for BGP peering").
		WithDocumentedRequiredFlag("mac-address", "MAC address for the IX interface").
		WithDocumentedRequiredFlag("rate-limit", "Rate limit in Mbps").
		WithDocumentedRequiredFlag("vlan", "VLAN ID for the IX connection").
		WithExample("megaport-cli ix buy --interactive").
		WithExample("megaport-cli ix buy --product-uid port-uid --name \"My IX\" --network-service-type \"Los Angeles IX\" --asn 65000 --mac-address \"00:11:22:33:44:55\" --rate-limit 1000 --vlan 100").
		WithExample("megaport-cli ix buy --json '{\"productUid\":\"port-uid\",\"productName\":\"My IX\",\"networkServiceType\":\"Los Angeles IX\",\"asn\":65000,\"macAddress\":\"00:11:22:33:44:55\",\"rateLimit\":1000,\"vlan\":100}'").
		WithExample("megaport-cli ix buy --json-file ./ix-config.json").
		WithJSONExample(`{
  "productUid": "port-uid-here",
  "productName": "My IX",
  "networkServiceType": "Los Angeles IX",
  "asn": 65000,
  "macAddress": "00:11:22:33:44:55",
  "rateLimit": 1000,
  "vlan": 100,
  "shutdown": false,
  "promoCode": "PROMO2025"
}`).
		WithImportantNote("The product-uid must correspond to a valid port in the Megaport API").
		WithImportantNote("Required flags (product-uid, name, network-service-type, asn, mac-address, rate-limit, vlan) can be skipped when using --interactive, --json, or --json-file").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("product-uid", "name", "network-service-type", "asn", "mac-address", "rate-limit", "vlan").
		Build()

	// Create update IX command
	updateIXCmd := cmdbuilder.NewCommand("update", "Update an existing IX").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(UpdateIX).
		WithStandardInputFlags().
		WithIXUpdateFlags().
		WithLongDesc("Update an existing Internet Exchange (IX).\n\nThis command allows you to update the details of an existing IX.").
		WithExample("megaport-cli ix update [ixUID] --interactive").
		WithExample("megaport-cli ix update [ixUID] --name \"Updated IX\" --rate-limit 2000").
		WithExample("megaport-cli ix update [ixUID] --json '{\"name\":\"Updated IX\",\"rateLimit\":2000}'").
		WithExample("megaport-cli ix update [ixUID] --json-file ./update-ix-config.json").
		WithJSONExample(`{
  "name": "Updated IX",
  "rateLimit": 2000,
  "costCentre": "Finance",
  "vlan": 200,
  "macAddress": "00:11:22:33:44:66",
  "asn": 65001
}`).
		WithImportantNote("The IX UID cannot be changed").
		WithImportantNote("Only specified fields will be updated; unspecified fields will remain unchanged").
		WithRootCmd(rootCmd).
		Build()

	// Create delete IX command
	deleteIXCmd := cmdbuilder.NewCommand("delete", "Delete an IX from your account").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(DeleteIX).
		WithImmediateDeleteFlags().
		WithLongDesc("Delete an IX from your account.\n\nDeletion is immediate.").
		WithExample("megaport-cli ix delete [ixUID]").
		WithExample("megaport-cli ix delete [ixUID] --force").
		WithImportantNote("Billing stops right away").
		WithRootCmd(rootCmd).
		WithAliases([]string{"rm"}).
		Build()

	// Create list IXs command
	listIXsCmd := cmdbuilder.NewCommand("list", "List all IXs with optional filters").
		WithOutputFormatRunFunc(ListIXs).
		WithLongDesc("List all IXs available in the Megaport API.\n\nThis command fetches and displays a list of IXs with details such as UID, name, network service type, ASN, rate limit, VLAN, and status. By default, only active IXs are shown.").
		WithIXFilterFlags().
		WithOptionalFlag("name", "Filter IXs by name (partial match)").
		WithOptionalFlag("asn", "Filter IXs by ASN").
		WithOptionalFlag("vlan", "Filter IXs by VLAN").
		WithOptionalFlag("network-service-type", "Filter IXs by network service type").
		WithOptionalFlag("location-id", "Filter IXs by location ID").
		WithOptionalFlag("rate-limit", "Filter IXs by rate limit in Mbps").
		WithOptionalFlag("include-inactive", "Include IXs in CANCELLED, DECOMMISSIONED, or DECOMMISSIONING states").
		WithExample("megaport-cli ix list").
		WithExample("megaport-cli ix list --name \"My IX\"").
		WithExample("megaport-cli ix list --asn 65000").
		WithExample("megaport-cli ix list --network-service-type \"Los Angeles IX\"").
		WithExample("megaport-cli ix list --include-inactive").
		WithIntFlag("limit", 0, "Maximum number of results to display (0 = unlimited)").
		WithRootCmd(rootCmd).
		WithAliases([]string{"ls"}).
		Build()

	// Create status IX command
	statusIXCmd := cmdbuilder.NewCommand("status", "Check the provisioning status of an IX").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(GetIXStatus).
		WithLongDesc("Check the provisioning status of an IX through the Megaport API.\n\nThis command retrieves only the essential status information for an Internet Exchange (IX) without all the details. It's useful for monitoring ongoing provisioning.").
		WithExample("megaport-cli ix status ix-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx").
		WithImportantNote("This is a lightweight command that only shows the IX's status without retrieving all details.").
		WithRootCmd(rootCmd).
		WithAliases([]string{"st"}).
		Build()

	// Create validate IX command
	validateIXCmd := cmdbuilder.NewCommand("validate", "Validate an IX order without purchasing").
		WithColorAwareRunFunc(ValidateIX).
		WithIXCreateFlags().
		WithStandardInputFlags().
		WithLongDesc("Validates an IX configuration against the Megaport API without creating the resource.\n\nUse this for dry-run validation before purchasing, or in CI pipelines to check configurations.").
		WithDocumentedRequiredFlag("product-uid", "The UID of the port to attach the IX to").
		WithDocumentedRequiredFlag("name", "The name of the IX").
		WithDocumentedRequiredFlag("network-service-type", "The IX type/network service to connect to").
		WithDocumentedRequiredFlag("asn", "ASN (Autonomous System Number) for BGP peering").
		WithDocumentedRequiredFlag("mac-address", "MAC address for the IX interface").
		WithDocumentedRequiredFlag("rate-limit", "Rate limit in Mbps").
		WithDocumentedRequiredFlag("vlan", "VLAN ID for the IX connection").
		WithExample(`megaport-cli ix validate --product-uid port-uid --name "My IX" --network-service-type "Los Angeles IX" --asn 65000 --mac-address "00:11:22:33:44:55" --rate-limit 1000 --vlan 100`).
		WithExample("megaport-cli ix validate --json-file ./ix-config.json").
		WithImportantNote("This command only validates the configuration — no resources are created and no charges are incurred").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("product-uid", "name", "network-service-type", "asn", "mac-address", "rate-limit", "vlan").
		Build()

	// Add commands to their parents
	ixCmd.AddCommand(
		getIXCmd,
		buyIXCmd,
		validateIXCmd,
		updateIXCmd,
		deleteIXCmd,
		listIXsCmd,
		statusIXCmd,
	)
	rootCmd.AddCommand(ixCmd)
}
