package vxc

import (
	"fmt"

	"github.com/megaport/megaport-cli/internal/base/cmdbuilder"
	"github.com/megaport/megaport-cli/internal/validation"
	"github.com/spf13/cobra"
)

// AddCommandsTo builds the vxc commands and adds them to the root command
func AddCommandsTo(rootCmd *cobra.Command) {
	// Create vxc parent command
	vxcCmd := cmdbuilder.NewCommand("vxc", "Manage VXCs in the Megaport API").
		WithLongDesc("Manage VXCs in the Megaport API.\n\nThis command groups all operations related to Virtual Cross Connects (VXCs). VXCs are virtual point-to-point connections between two ports or devices on the Megaport network. You can use the subcommands to perform actions such as retrieving details, purchasing, updating, and deleting VXCs.").
		WithExample("megaport-cli vxc list").
		WithExample("megaport-cli vxc get [vxcUID]").
		WithExample("megaport-cli vxc buy").
		WithExample("megaport-cli vxc update [vxcUID]").
		WithExample("megaport-cli vxc delete [vxcUID]").
		WithRootCmd(rootCmd).
		Build()

	// Create list VXCs command
	listVXCsCmd := cmdbuilder.NewCommand("list", "List all VXCs with optional filters").
		WithOutputFormatRunFunc(ListVXCs).
		WithVXCFilterFlags().
		WithTagFilterFlags().
		WithLongDesc("List all VXCs available in the Megaport API.\n\nThis command retrieves all Virtual Cross Connects (VXCs) associated with your account. You can filter results by name, rate limit, A-End UID, B-End UID, status, or resource tags.").
		WithOptionalFlag("name", "Filter VXCs by name (case-sensitive partial match)").
		WithOptionalFlag("name-contains", "Filter VXCs by name (case-sensitive partial match; takes precedence over --name)").
		WithOptionalFlag("rate-limit", "Filter VXCs by rate limit in Mbps").
		WithOptionalFlag("a-end-uid", "Filter VXCs by A-End product UID").
		WithOptionalFlag("b-end-uid", "Filter VXCs by B-End product UID").
		WithOptionalFlag("status", "Filter VXCs by status (comma-separated, e.g. LIVE,CONFIGURED)").
		WithOptionalFlag("include-inactive", "Include inactive VXCs in the list").
		WithExample("megaport-cli vxc list").
		WithExample("megaport-cli vxc list --name \"My VXC\"").
		WithExample("megaport-cli vxc list --a-end-uid port-abc123").
		WithExample("megaport-cli vxc list --status LIVE,CONFIGURED").
		WithExample("megaport-cli vxc list --include-inactive").
		WithExample("megaport-cli vxc list --tag env=prod").
		WithExample("megaport-cli vxc list --tag env=prod --tag team=network").
		WithIntFlag("limit", 0, "Maximum number of results to display (0 = unlimited)").
		WithRootCmd(rootCmd).
		WithAliases([]string{"ls"}).
		Build()

	// Create get VXC command
	getVXCCmd := cmdbuilder.NewCommand("get", "Get details for a single VXC").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(GetVXC).
		WithBoolFlag("export", false, "Output recreatable JSON config for use with buy --json (excludes read-only fields; partner configs not available from API)").
		WithWatchFlags().
		WithLongDesc("Get details for a single VXC through the Megaport API.\n\nThis command retrieves detailed information for a single Virtual Cross Connect (VXC). You must provide the unique identifier (UID) of the VXC you wish to retrieve.").
		WithExample("megaport-cli vxc get vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx").
		WithExample("megaport-cli vxc get vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --export").
		WithExample("megaport-cli vxc get vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --watch").
		WithExample("megaport-cli vxc get vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --watch --interval 10s").
		WithImportantNote("The output includes the VXC's UID, name, rate limit, A-End and B-End details, status, and cost centre.").
		WithRootCmd(rootCmd).
		WithAliases([]string{"show"}).
		Build()

	// Create status VXC command
	statusVXCCmd := cmdbuilder.NewCommand("status", "Check the provisioning status of a VXC").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(GetVXCStatus).
		WithWatchFlags().
		WithLongDesc("Check the provisioning status of a VXC through the Megaport API.\n\nThis command retrieves only the essential status information for a Virtual Cross Connect (VXC) without all the details. It's useful for monitoring ongoing provisioning.").
		WithExample("megaport-cli vxc status vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx").
		WithExample("megaport-cli vxc status vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --watch").
		WithExample("megaport-cli vxc status vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --watch --interval 10s").
		WithImportantNote("This is a lightweight command that only shows the VXC's status without retrieving all details.").
		WithRootCmd(rootCmd).
		WithAliases([]string{"st"}).
		Build()

	// Create buy VXC command
	buyVXCCmd := cmdbuilder.NewCommand("buy", "Purchase a new VXC").
		WithColorAwareRunFunc(BuyVXC).
		WithInteractiveFlag().
		WithNoWaitFlag().
		WithBuyConfirmFlags().
		WithVXCCreateFlags().
		WithJSONConfigFlags().
		WithLongDesc("Purchase a new Megaport Virtual Cross Connect (VXC) through the Megaport API.\n\nThis command allows you to create a VXC by providing the necessary details.").
		WithDocumentedRequiredFlag("name", "Name of the VXC").
		WithDocumentedRequiredFlag("rate-limit", "Bandwidth in Mbps").
		WithDocumentedRequiredFlag("term", fmt.Sprintf("Contract term in months (%s)", validation.FormatIntSlice(validation.ValidContractTerms))).
		WithDocumentedRequiredFlag("a-end-uid", "UID of the A-End product").
		WithDocumentedRequiredFlag("b-end-uid", "UID of the B-End product (if not using partner configuration)").
		WithDocumentedRequiredFlag("a-end-vlan", "VLAN for A-End ("+validation.VLANHelpText()+")").
		WithDocumentedRequiredFlag("b-end-vlan", "VLAN for B-End ("+validation.VLANHelpText()+")").
		WithExample("megaport-cli vxc buy --interactive").
		WithExample("megaport-cli vxc buy --name \"My VXC\" --rate-limit 1000 --term 12 --a-end-uid port-123 --b-end-uid port-456 --a-end-vlan 100 --b-end-vlan 200").
		WithExample("megaport-cli vxc buy --name \"My VXC\" --rate-limit 1000 --term 12 --a-end-uid port-123 --b-end-uid port-456 --a-end-vlan 100 --b-end-vlan 200 --resource-tags '{\"environment\":\"production\",\"team\":\"networking\"}'").
		WithExample("megaport-cli vxc buy --json '{\"vxcName\":\"My VXC\",\"rateLimit\":1000,\"term\":12,\"portUid\":\"port-123\",\"aEndConfiguration\":{\"vlan\":100},\"bEndConfiguration\":{\"productUID\":\"port-456\",\"vlan\":200},\"resourceTags\":{\"environment\":\"production\",\"owner\":\"network-team\"}}'").
		WithExample("megaport-cli vxc buy --name \"IPsec VXC\" --rate-limit 1000 --term 12 --a-end-uid port-123 --b-end-uid port-456 --a-end-partner-config '{\"connectType\":\"VROUTER\",\"interfaces\":[{\"interfaceType\":\"ipSecTunnel\",\"ipSecTunnelOptions\":{\"sourceIpAddress\":\"192.0.2.1\",\"destinationIpAddress\":\"198.51.100.1\",\"preSharedKey\":\"<your-psk>\",\"phase1Lifetime\":28800,\"phase2Lifetime\":3600}}]}'").
		WithExample("megaport-cli vxc buy --json-file ./vxc-config.json").
		WithJSONExample(`{
  "vxcName": "My VXC",
  "rateLimit": 1000, 
  "term": 12,
  "portUid": "port-123",
  "aEndConfiguration": {
    "vlan": 100
  },
  "bEndConfiguration": {
    "productUID": "port-456",
    "vlan": 200
  },
  "costCentre": "IT Department",
  "resourceTags": {
    "environment": "production",
    "owner": "network-team",
    "project": "cloud-migration"
  }
}`).
		WithImportantNote("To order an MCR IPsec tunnel, set interfaceType \"ipSecTunnel\" on the vRouter interface and provide ipSecTunnelOptions; treat the preSharedKey as a secret and avoid committing populated config files to source control").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("name", "rate-limit", "term", "a-end-uid").
		Build()

	// Create update VXC command
	updateVXCCmd := cmdbuilder.NewCommand("update", "Update an existing Virtual Cross Connect (VXC)").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(UpdateVXC).
		WithBoolFlag("interactive", false, "Use interactive mode").
		WithVXCUpdateFlags().
		WithJSONConfigFlags().
		WithLongDesc("Update an existing Virtual Cross Connect (VXC) through the Megaport API.\n\nThis command allows you to update an existing VXC by providing the necessary details.").
		WithOptionalFlag("name", "New name for the VXC (1-64 characters)").
		WithOptionalFlag("rate-limit", "New bandwidth in Mbps (50 - 10000)").
		WithOptionalFlag("term", fmt.Sprintf("New contract term in months (%s)", validation.FormatIntSlice(validation.ValidContractTerms))).
		WithOptionalFlag("cost-centre", "New cost centre for billing").
		WithOptionalFlag("shutdown", "Whether to shut down the VXC (true/false)").
		WithOptionalFlag("a-end-vlan", "New VLAN for A-End ("+validation.VLANHelpText()+")").
		WithOptionalFlag("b-end-vlan", "New VLAN for B-End ("+validation.VLANHelpText()+")").
		WithOptionalFlag("a-end-inner-vlan", "New inner VLAN for A-End ("+validation.InnerVLANHelpText()+", only for QinQ)").
		WithOptionalFlag("b-end-inner-vlan", "New inner VLAN for B-End ("+validation.InnerVLANHelpText()+", only for QinQ)").
		WithOptionalFlag("a-end-uid", "New A-End product UID").
		WithOptionalFlag("b-end-uid", "New B-End product UID").
		WithOptionalFlag("a-end-partner-config", "JSON string with A-End VRouter partner configuration").
		WithOptionalFlag("b-end-partner-config", "JSON string with B-End VRouter partner configuration").
		WithExample("megaport-cli vxc update vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --interactive").
		WithExample("megaport-cli vxc update vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --name \"New VXC Name\" --rate-limit 2000 --cost-centre \"New Cost Centre\"").
		WithExample("megaport-cli vxc update vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --a-end-vlan 200 --b-end-vlan 300").
		WithExample("megaport-cli vxc update vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --b-end-partner-config '{\"connectType\":\"VROUTER\",\"interfaces\":[{\"vlan\":100,\"ipAddresses\":[\"192.168.1.1/30\"],\"bgpConnections\":[{\"peerAsn\":65000,\"localAsn\":64512,\"localIpAddress\":\"192.168.1.1\",\"peerIpAddress\":\"192.168.1.2\",\"password\":\"<your-bgp-password>\",\"shutdown\":false,\"bfdEnabled\":true}]}]}'").
		WithExample("megaport-cli vxc update vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --a-end-partner-config '{\"connectType\":\"VROUTER\",\"interfaces\":[{\"interfaceType\":\"ipSecTunnel\",\"ipSecTunnelOptions\":{\"sourceIpAddress\":\"192.0.2.1\",\"destinationIpAddress\":\"198.51.100.1\",\"preSharedKey\":\"<your-psk>\",\"phase1Lifetime\":28800,\"phase2Lifetime\":3600}}]}'").
		WithExample("megaport-cli vxc update vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --json '{\"name\":\"Updated VXC Name\",\"rateLimit\":2000,\"costCentre\":\"New Cost Centre\",\"aEndVlan\":200,\"bEndVlan\":300,\"term\":24,\"shutdown\":false}'").
		WithJSONExample(`{
  "name": "Updated VXC Name",
  "rateLimit": 2000,
  "costCentre": "New Cost Centre",
  "aEndVlan": 200,
  "bEndVlan": 300,
  "term": 24,
  "shutdown": false
}`).
		WithImportantNote("Only VRouter partner configurations can be updated after creation").
		WithImportantNote("CSP partner configurations (AWS, Azure, etc.) cannot be changed after creation").
		WithImportantNote("IPsec tunnels require interfaceType \"ipSecTunnel\" on the interface; treat the preSharedKey as a secret and avoid committing populated config files to source control").
		WithImportantNote("Changing the rate limit may result in additional charges").
		WithImportantNote("Updating VLANs will cause temporary disruption to the VXC connectivity").
		WithRootCmd(rootCmd).
		Build()

	// Create delete VXC command
	deleteVXCCmd := cmdbuilder.NewCommand("delete", "Delete an existing Virtual Cross Connect (VXC)").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(DeleteVXC).
		WithImmediateDeleteFlags().
		WithLongDesc("Delete an existing Virtual Cross Connect (VXC) through the Megaport API.\n\nThis command allows you to delete an existing VXC by providing its UID. Deletion is immediate.").
		WithExample("megaport-cli vxc delete vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx").
		WithExample("megaport-cli vxc delete vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --force").
		WithImportantNote("The VXC is disconnected and billing stops right away").
		WithImportantNote("Deletion is final and cannot be undone").
		WithRootCmd(rootCmd).
		WithAliases([]string{"rm"}).
		Build()

	// Add list-tags command
	listTagsCmd := cmdbuilder.NewCommand("list-tags", "List resource tags on a specific VXC").
		WithLongDesc("Lists all resource tags associated with a specific VXC").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(ListVXCResourceTags).
		WithExample("megaport-cli vxc list-tags vxc-abc123").
		Build()

	// Add update-tags command
	updateTagsCmd := cmdbuilder.NewCommand("update-tags", "Update resource tags on a specific VXC").
		WithLongDesc("Update resource tags associated with a specific VXC. Tags can be provided via interactive prompts, JSON string, or JSON file.").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(UpdateVXCResourceTags).
		WithStandardInputFlags().
		WithBoolFlag("force", false, "Skip the confirmation prompt").
		WithExample("megaport-cli vxc update-tags vxc-abc123 --interactive").
		WithExample("megaport-cli vxc update-tags vxc-abc123 --json '{\"env\":\"production\",\"team\":\"network\"}'").
		WithExample("megaport-cli vxc update-tags vxc-abc123 --json-file ./tags.json").
		WithJSONExample(`{
  "environment": "production",
  "owner": "network-team",
  "project": "cloud-migration"
}`).
		WithImportantNote("All existing tags will be replaced with the provided tags. To clear all tags, provide an empty tag set.").
		Build()

	// Create validate VXC command
	validateVXCCmd := cmdbuilder.NewCommand("validate", "Validate a VXC order without purchasing").
		WithColorAwareRunFunc(ValidateVXC).
		WithInteractiveFlag().
		WithVXCCreateFlags().
		WithJSONConfigFlags().
		WithLongDesc("Validates a VXC configuration against the Megaport API without creating the resource.\n\nUse this for dry-run validation before purchasing, or in CI pipelines to check configurations.").
		WithDocumentedRequiredFlag("name", "Name of the VXC").
		WithDocumentedRequiredFlag("rate-limit", "Bandwidth in Mbps").
		WithDocumentedRequiredFlag("term", fmt.Sprintf("Contract term in months (%s)", validation.FormatIntSlice(validation.ValidContractTerms))).
		WithDocumentedRequiredFlag("a-end-uid", "UID of the A-End product").
		WithExample(`megaport-cli vxc validate --name "My VXC" --rate-limit 1000 --term 12 --a-end-uid port-123 --b-end-uid port-456 --a-end-vlan 100 --b-end-vlan 200`).
		WithExample("megaport-cli vxc validate --json-file ./vxc-config.json").
		WithImportantNote("This command only validates the configuration — no resources are created and no charges are incurred").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("name", "rate-limit", "term", "a-end-uid").
		Build()

	// Add commands to their parents
	vxcCmd.AddCommand(
		listVXCsCmd,
		getVXCCmd,
		statusVXCCmd,
		buyVXCCmd,
		validateVXCCmd,
		updateVXCCmd,
		deleteVXCCmd,
		listTagsCmd,
		updateTagsCmd,
	)
	rootCmd.AddCommand(vxcCmd)
}
