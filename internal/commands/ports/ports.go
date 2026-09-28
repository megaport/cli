package ports

import (
	"fmt"

	"github.com/megaport/megaport-cli/internal/base/cmdbuilder"
	"github.com/megaport/megaport-cli/internal/validation"
	"github.com/spf13/cobra"
)

// AddCommandsTo builds the ports commands and adds them to the root command
func AddCommandsTo(rootCmd *cobra.Command) {
	// Create ports parent command
	portsCmd := cmdbuilder.NewCommand("ports", "Manage ports in the Megaport API").
		WithLongDesc("Manage ports in the Megaport API.\n\nThis command groups operations related to ports. You can use the subcommands to list all ports, get details for a specific port, buy a new port, buy a LAG port, update an existing port, delete a port, lock a port, unlock a port, and check VLAN availability on a port.").
		WithExample("megaport-cli ports list").
		WithExample("megaport-cli ports get [portUID]").
		WithExample("megaport-cli ports buy --interactive").
		WithExample("megaport-cli ports buy-lag --interactive").
		WithExample("megaport-cli ports update [portUID] --name \"Updated Port Name\"").
		WithExample("megaport-cli ports delete [portUID]").
		WithExample("megaport-cli ports lock [portUID]").
		WithExample("megaport-cli ports unlock [portUID]").
		WithExample("megaport-cli ports check-vlan [portUID] [vlan]").
		WithRootCmd(rootCmd).
		Build()

	buy, buyLag, update, validate, validateLag := buildPortBuyCommands(rootCmd)
	list, get, status, deleteCmd, lock, unlock, checkVLAN := buildPortManagementCommands(rootCmd)
	listTags, updateTags := buildPortTagCommands()

	portsCmd.AddCommand(
		buy, buyLag, update, validate, validateLag,
		list, get, status, deleteCmd, lock, unlock, checkVLAN,
		listTags, updateTags,
	)
	rootCmd.AddCommand(portsCmd)
}

// buildPortBuyCommands creates the buy, buy-lag, and update port commands.
func buildPortBuyCommands(rootCmd *cobra.Command) (buy, buyLag, update, validate, validateLag *cobra.Command) {
	buy = cmdbuilder.NewCommand("buy", "Buy a port through the Megaport API").
		WithColorAwareRunFunc(BuyPort).
		WithInteractiveFlag().
		WithNoWaitFlag().
		WithBuyConfirmFlags().
		WithPortCreationFlags().
		WithJSONConfigFlags().
		WithLongDesc("Buy a port through the Megaport API.\n\nThis command allows you to purchase a port by providing the necessary details.").
		WithDocumentedRequiredFlag("name", "The name of the port (1-64 characters)").
		WithDocumentedRequiredFlag("term", fmt.Sprintf("The term of the port (%s months)", validation.FormatIntSlice(validation.ValidContractTerms))).
		WithDocumentedRequiredFlag("port-speed", "The speed of the port (1000, 10000, or 100000 Mbps)").
		WithDocumentedRequiredFlag("location-id", "The ID of the location where the port will be provisioned").
		WithDocumentedRequiredFlag("marketplace-visibility", "Whether the port should be visible in the marketplace (true or false)").
		WithOptionalFlag("diversity-zone", "The diversity zone for the port").
		WithOptionalFlag("cost-centre", "The cost centre for the port").
		WithOptionalFlag("promo-code", "A promotional code for the port").
		WithExample("megaport-cli ports buy --interactive").
		WithExample(`megaport-cli ports buy --name "My Port" --term 12 --port-speed 10000 --location-id 123 --marketplace-visibility true`).
		WithExample(`megaport-cli ports buy --name "My Port" --term 12 --port-speed 10000 --location-id 123 --marketplace-visibility true --resource-tags '{"env":"prod","owner":"network-team"}'`).
		WithExample("megaport-cli ports buy --json '{\"name\":\"My Port\",\"term\":12,\"portSpeed\":10000,\"locationId\":123,\"marketPlaceVisibility\":true}'").
		WithExample("megaport-cli ports buy --json-file ./port-config.json").
		WithJSONExample(`{
  "name": "My Port",
  "term": 12,
  "portSpeed": 10000,
  "locationId": 123,
  "marketPlaceVisibility": true,
  "diversityZone": "blue",
  "costCentre": "IT-2023",
  "resourceTags": {
    "environment": "production",
    "department": "networking",
    "project": "cloud-migration",
    "owner": "john.doe@example.com"
  }
}`).
		WithImportantNote("Resource tags allow you to categorize resources for organization, filtering, and billing purposes").
		WithImportantNote("Required flags (name, term, port-speed, location-id, marketplace-visibility) can be skipped when using --interactive, --json, or --json-file").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("name", "term", "port-speed", "location-id", "marketplace-visibility").
		Build()

	buyLag = cmdbuilder.NewCommand("buy-lag", "Buy a LAG port through the Megaport API").
		WithColorAwareRunFunc(BuyLAGPort).
		WithInteractiveFlag().
		WithNoWaitFlag().
		WithBuyConfirmFlags().
		WithPortLAGFlags().
		WithJSONConfigFlags().
		WithLongDesc("Buy a LAG port through the Megaport API.\n\nThis command allows you to purchase a LAG port by providing the necessary details.").
		WithDocumentedRequiredFlag("name", "The name of the port (1-64 characters)").
		WithDocumentedRequiredFlag("term", "The term of the port (1, 12, or 24 months)").
		WithDocumentedRequiredFlag("port-speed", "The speed of each LAG member port (10000 or 100000 Mbps)").
		WithDocumentedRequiredFlag("location-id", "The ID of the location where the port will be provisioned").
		WithDocumentedRequiredFlag("lag-count", "The number of LAG members (between 1 and 8)").
		WithDocumentedRequiredFlag("marketplace-visibility", "Whether the port should be visible in the marketplace (true or false)").
		WithOptionalFlag("diversity-zone", "The diversity zone for the LAG port").
		WithOptionalFlag("cost-centre", "The cost centre for the LAG port").
		WithOptionalFlag("promo-code", "A promotional code for the LAG port").
		WithExample("megaport-cli ports buy-lag --interactive").
		WithExample(`megaport-cli ports buy-lag --name "My LAG Port" --term 12 --port-speed 10000 --location-id 123 --lag-count 2 --marketplace-visibility true`).
		WithExample(`megaport-cli ports buy-lag --name "My LAG Port" --term 12 --port-speed 10000 --location-id 123 --lag-count 2 --marketplace-visibility true --resource-tags '{"env":"prod","owner":"network-team"}'`).
		WithExample("megaport-cli ports buy-lag --json '{\"name\":\"My LAG Port\",\"term\":12,\"portSpeed\":10000,\"locationId\":123,\"lagCount\":2,\"marketPlaceVisibility\":true}'").
		WithJSONExample(`{
  "name": "My LAG Port",
  "term": 12,
  "portSpeed": 10000,
  "locationId": 123,
  "lagCount": 2,
  "marketPlaceVisibility": true,
  "diversityZone": "blue",
  "costCentre": "IT-2023",
  "resourceTags": {
    "environment": "production",
    "department": "networking",
    "application": "database-cluster"
  }
}`).
		WithImportantNote("Resource tags allow you to categorize resources for organization, filtering, and billing purposes").
		WithImportantNote("Required flags can be skipped when using --interactive, --json, or --json-file").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("name", "term", "port-speed", "location-id", "lag-count", "marketplace-visibility").
		Build()

	update = cmdbuilder.NewCommand("update", "Update a port's details").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(UpdatePort).
		WithInteractiveFlag().
		WithPortUpdateFlags().
		WithJSONConfigFlags().
		WithLongDesc("Update a port's details in the Megaport API.\n\nThis command allows you to update the details of an existing port by providing the necessary fields.").
		WithOptionalFlag("name", "The new name of the port (1-64 characters)").
		WithOptionalFlag("marketplace-visibility", "Whether the port should be visible in the marketplace (true or false)").
		WithOptionalFlag("cost-centre", "The cost centre for billing purposes").
		WithOptionalFlag("term", fmt.Sprintf("The new contract term in months (%s)", validation.FormatIntSlice(validation.ValidContractTerms))).
		WithExample("megaport-cli ports update port-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --interactive").
		WithExample("megaport-cli ports update port-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --name \"Updated Port\" --marketplace-visibility true --cost-centre \"Finance\"").
		WithExample("megaport-cli ports update port-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --json '{\"name\":\"Updated Port\",\"marketPlaceVisibility\":true,\"costCentre\":\"Finance\"}'").
		WithExample("megaport-cli ports update port-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --json-file ./update-port-config.json").
		WithJSONExample(`{
  "name": "Updated Port",
  "marketPlaceVisibility": true,
  "costCentre": "IT-2024",
  "contractTermMonths": 24
}`).
		WithImportantNote("At least one update flag must be provided when not using --interactive, --json, or --json-file").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("at_least_one:name,marketplace-visibility,cost-centre,term").
		Build()

	validate = cmdbuilder.NewCommand("validate", "Validate a port order without purchasing").
		WithColorAwareRunFunc(ValidatePort).
		WithInteractiveFlag().
		WithPortCreationFlags().
		WithJSONConfigFlags().
		WithLongDesc("Validates a port configuration against the Megaport API without creating the resource.\n\nUse this for dry-run validation before purchasing, or in CI pipelines to check configurations.").
		WithDocumentedRequiredFlag("name", "The name of the port (1-64 characters)").
		WithDocumentedRequiredFlag("term", fmt.Sprintf("The term of the port (%s months)", validation.FormatIntSlice(validation.ValidContractTerms))).
		WithDocumentedRequiredFlag("port-speed", "The speed of the port (1000, 10000, or 100000 Mbps)").
		WithDocumentedRequiredFlag("location-id", "The ID of the location where the port will be provisioned").
		WithDocumentedRequiredFlag("marketplace-visibility", "Whether the port should be visible in the marketplace (true or false)").
		WithExample(`megaport-cli ports validate --name "My Port" --term 12 --port-speed 10000 --location-id 123 --marketplace-visibility true`).
		WithExample("megaport-cli ports validate --json-file ./port-config.json").
		WithImportantNote("This command only validates the configuration — no resources are created and no charges are incurred").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("name", "term", "port-speed", "location-id", "marketplace-visibility").
		Build()

	validateLag = cmdbuilder.NewCommand("validate-lag", "Validate a LAG port order without purchasing").
		WithColorAwareRunFunc(ValidateLAGPort).
		WithInteractiveFlag().
		WithPortLAGFlags().
		WithJSONConfigFlags().
		WithLongDesc("Validates a LAG port configuration against the Megaport API without creating the resource.\n\nUse this for dry-run validation before purchasing, or in CI pipelines to check configurations.").
		WithDocumentedRequiredFlag("name", "The name of the port (1-64 characters)").
		WithDocumentedRequiredFlag("term", "The term of the port (1, 12, or 24 months)").
		WithDocumentedRequiredFlag("port-speed", "The speed of each LAG member port (10000 or 100000 Mbps)").
		WithDocumentedRequiredFlag("location-id", "The ID of the location where the port will be provisioned").
		WithDocumentedRequiredFlag("lag-count", "The number of LAG members (between 1 and 8)").
		WithDocumentedRequiredFlag("marketplace-visibility", "Whether the port should be visible in the marketplace (true or false)").
		WithExample(`megaport-cli ports validate-lag --name "My LAG Port" --term 12 --port-speed 10000 --location-id 123 --lag-count 2 --marketplace-visibility true`).
		WithExample("megaport-cli ports validate-lag --json-file ./lag-config.json").
		WithImportantNote("This command only validates the configuration — no resources are created and no charges are incurred").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("name", "term", "port-speed", "location-id", "lag-count", "marketplace-visibility").
		Build()

	return buy, buyLag, update, validate, validateLag
}

// buildPortManagementCommands creates the list, get, status, delete, lock, unlock, and check-vlan commands.
func buildPortManagementCommands(rootCmd *cobra.Command) (list, get, status, deleteCmd, lock, unlock, checkVLAN *cobra.Command) {
	list = cmdbuilder.NewCommand("list", "List all ports with optional filters").
		WithOutputFormatRunFunc(ListPorts).
		WithPortFilterFlags().
		WithTagFilterFlags().
		WithLongDesc("List all ports available in the Megaport API.\n\nThis command fetches and displays a list of ports with details such as port ID, name, location, speed, and status. By default, only active ports are shown. You can also filter by resource tags.").
		WithOptionalFlag("location-id", "Filter ports by location ID").
		WithOptionalFlag("port-speed", "Filter ports by port speed").
		WithOptionalFlag("port-name", "Filter ports by port name").
		WithOptionalFlag("include-inactive", "Include ports in CANCELLED, DECOMMISSIONED, or DECOMMISSIONING states").
		WithExample("megaport-cli ports list").
		WithExample("megaport-cli ports list --location-id 1").
		WithExample("megaport-cli ports list --port-speed 10000").
		WithExample("megaport-cli ports list --port-name \"Data Center Primary\"").
		WithExample("megaport-cli ports list --include-inactive").
		WithExample("megaport-cli ports list --location-id 1 --port-speed 10000 --port-name \"Data Center Primary\"").
		WithExample("megaport-cli ports list --tag env=prod").
		WithExample("megaport-cli ports list --tag env=prod --tag team=network").
		WithIntFlag("limit", 0, "Maximum number of results to display (0 = unlimited)").
		WithRootCmd(rootCmd).
		WithAliases([]string{"ls"}).
		Build()

	get = cmdbuilder.NewCommand("get", "Get details for a single port").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(GetPort).
		WithBoolFlag("export", false, "Output recreatable JSON config for use with buy --json (excludes read-only fields)").
		WithWatchFlags().
		WithLongDesc("Get details for a single port from the Megaport API.\n\nThis command fetches and displays detailed information about a specific port. You need to provide the UID of the port as an argument.").
		WithExample("megaport-cli ports get port-abc123").
		WithExample("megaport-cli ports get 1a2b3c4d-5e6f-7g8h-9i0j-1k2l3m4n5o6p").
		WithExample("megaport-cli ports get port-abc123 --export").
		WithExample("megaport-cli ports get port-abc123 --watch").
		WithExample("megaport-cli ports get port-abc123 --watch --interval 10s").
		WithRootCmd(rootCmd).
		WithAliases([]string{"show"}).
		Build()

	status = cmdbuilder.NewCommand("status", "Check the provisioning status of a port").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(GetPortStatus).
		WithWatchFlags().
		WithLongDesc("Check the provisioning status of a port through the Megaport API.\n\nThis command retrieves only the essential status information for a port without all the details. It's useful for monitoring ongoing provisioning.").
		WithExample("megaport-cli ports status port-abc123").
		WithExample("megaport-cli ports status port-abc123 --watch").
		WithExample("megaport-cli ports status port-abc123 --watch --interval 10s").
		WithImportantNote("This is a lightweight command that only shows the port's status without retrieving all details.").
		WithRootCmd(rootCmd).
		WithAliases([]string{"st"}).
		Build()

	deleteCmd = cmdbuilder.NewCommand("delete", "Delete a port from your account").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(DeletePort).
		WithImmediateSafeDeleteFlags().
		WithLongDesc("Delete a port from your account in the Megaport API.\n\nThis command deletes an existing port by providing the UID of the port as an argument. Ports are deleted immediately; deferred cancellation at the end of the billing period is no longer supported.").
		WithOptionalFlag("force", "Skip the confirmation prompt and proceed with deletion").
		WithOptionalFlag("safe-delete", "Fail if the resource has attached VXCs or other active services").
		WithImportantNote("All VXCs associated with the port must be deleted before the port can be deleted").
		WithImportantNote("Ports are deleted immediately; the previous 'terminate later' option is no longer available").
		WithExample("megaport-cli ports delete 1a2b3c4d-5e6f-7g8h-9i0j-1k2l3m4n5o6p").
		WithExample("megaport-cli ports delete 1a2b3c4d-5e6f-7g8h-9i0j-1k2l3m4n5o6p --force").
		WithRootCmd(rootCmd).
		WithAliases([]string{"rm"}).
		Build()

	lock = cmdbuilder.NewCommand("lock", "Lock a port").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(LockPort).
		WithLongDesc("Lock a port in the Megaport API.\n\nThis command allows you to lock an existing port, preventing any changes or modifications to the port or its associated VXCs. Locking a port is useful for ensuring critical infrastructure remains stable and preventing accidental changes.").
		WithImportantNote("The port's configuration cannot be modified").
		WithImportantNote("New VXCs cannot be created on this port").
		WithImportantNote("Existing VXCs cannot be modified or deleted").
		WithImportantNote("The port itself cannot be deleted").
		WithImportantNote("To reverse this action, use the 'unlock' command").
		WithExample("megaport-cli ports lock 1a2b3c4d-5e6f-7g8h-9i0j-1k2l3m4n5o6p").
		WithRootCmd(rootCmd).
		Build()

	unlock = cmdbuilder.NewCommand("unlock", "Unlock a port").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(UnlockPort).
		WithLongDesc("Unlock a port in the Megaport API.\n\nThis command allows you to unlock a previously locked port, re-enabling the ability to make changes to the port and its associated VXCs.").
		WithImportantNote("The port's configuration can be modified").
		WithImportantNote("New VXCs can be created on this port").
		WithImportantNote("Existing VXCs can be modified or deleted").
		WithImportantNote("The port itself can be deleted if needed").
		WithExample("megaport-cli ports unlock 1a2b3c4d-5e6f-7g8h-9i0j-1k2l3m4n5o6p").
		WithRootCmd(rootCmd).
		Build()

	checkVLAN = cmdbuilder.NewCommand("check-vlan", "Check if a VLAN is available on a port").
		WithArgs(cobra.ExactArgs(2)).
		WithColorAwareRunFunc(CheckPortVLANAvailability).
		WithLongDesc(fmt.Sprintf("Check if a VLAN is available on a port in the Megaport API.\n\nThis command verifies whether a specific VLAN ID is available for use on a port. This is useful when planning new VXCs to ensure the VLAN ID you want to use is not already in use by another connection.\n\nVLAN ID must be between %d and %d (inclusive).", validation.MinAssignableVLAN, validation.MaxAssignableVLAN)).
		WithExample("megaport-cli ports check-vlan 1a2b3c4d-5e6f-7g8h-9i0j-1k2l3m4n5o6p 100").
		WithExample("megaport-cli ports check-vlan port-abc123 500").
		WithRootCmd(rootCmd).
		Build()

	return list, get, status, deleteCmd, lock, unlock, checkVLAN
}

// buildPortTagCommands creates the list-tags and update-tags commands.
func buildPortTagCommands() (listTags, updateTags *cobra.Command) {
	listTags = cmdbuilder.NewCommand("list-tags", "List resource tags on a specific port.").
		WithLongDesc("Lists all resource tags associated with a specific port").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(ListPortResourceTags).
		WithExample("megaport port list-tags port-abc123").
		Build()

	updateTags = cmdbuilder.NewCommand("update-tags", "Update resource tags on a specific port").
		WithLongDesc("Update resource tags associated with a specific port. Tags can be provided via interactive prompts, JSON string, or JSON file.").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(UpdatePortResourceTags).
		WithStandardInputFlags().
		WithBoolFlag("force", false, "Skip the confirmation prompt").
		WithExample("megaport-cli ports update-tags port-abc123 --interactive").
		WithExample("megaport-cli ports update-tags port-abc123 --json '{\"env\":\"production\",\"team\":\"network\"}'").
		WithExample("megaport-cli ports update-tags port-abc123 --json-file ./tags.json").
		WithImportantNote("All existing tags will be replaced with the provided tags. To clear all tags, provide an empty tag set.").
		Build()

	return listTags, updateTags
}
