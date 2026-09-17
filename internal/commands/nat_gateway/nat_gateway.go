package nat_gateway

import (
	"fmt"

	"github.com/megaport/megaport-cli/internal/base/cmdbuilder"
	"github.com/megaport/megaport-cli/internal/validation"
	"github.com/spf13/cobra"
)

// AddCommandsTo builds the nat-gateway commands and adds them to the root command.
func AddCommandsTo(rootCmd *cobra.Command) {
	natCmd := cmdbuilder.NewCommand("nat-gateway", "Manage NAT Gateways in the Megaport API").
		WithLongDesc("Manage NAT Gateways in the Megaport API.\n\nThis command groups all operations related to Megaport NAT Gateways. NAT Gateways provide network address translation services within the Megaport fabric.").
		WithExample("megaport-cli nat-gateway get [uid]").
		WithExample("megaport-cli nat-gateway list").
		WithExample("megaport-cli nat-gateway create --interactive").
		WithExample("megaport-cli nat-gateway update [uid]").
		WithExample("megaport-cli nat-gateway delete [uid]").
		WithExample("megaport-cli nat-gateway validate [uid]").
		WithExample("megaport-cli nat-gateway buy [uid]").
		WithExample("megaport-cli nat-gateway list-sessions").
		WithExample("megaport-cli nat-gateway telemetry [uid] --types BITS --days 7").
		WithRootCmd(rootCmd).
		Build()

	get, list, create, update, del, validate, buy, listSessions, telemetry := buildNATGatewayCommands(rootCmd)

	natCmd.AddCommand(get, list, create, update, del, validate, buy, listSessions, telemetry)
	rootCmd.AddCommand(natCmd)
}

func buildNATGatewayCommands(rootCmd *cobra.Command) (get, list, create, update, del, validate, buy, listSessions, telemetry *cobra.Command) {
	get = cmdbuilder.NewCommand("get", "Get details for a single NAT Gateway").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(GetNATGateway).
		WithBoolFlag("export", false, "Output recreatable JSON config for use with create --json-file").
		WithWatchFlags().
		WithLongDesc("Get details for a single NAT Gateway.\n\nRetrieves and displays detailed information for a single NAT Gateway by its product UID.").
		WithExample("megaport-cli nat-gateway get a1b2c3d4-e5f6-7890-1234-567890abcdef").
		WithExample("megaport-cli nat-gateway get a1b2c3d4-e5f6-7890-1234-567890abcdef --export").
		WithExample("megaport-cli nat-gateway get a1b2c3d4-e5f6-7890-1234-567890abcdef --watch").
		WithAliases([]string{"show"}).
		WithRootCmd(rootCmd).
		Build()

	list = cmdbuilder.NewCommand("list", "List all NAT Gateways").
		WithOutputFormatRunFunc(ListNATGateways).
		WithNATGatewayFilterFlags().
		WithIntFlag("limit", 0, "Limit the number of results returned").
		WithOptionalFlag("location-id", "Filter NAT Gateways by location ID").
		WithOptionalFlag("name", "Filter NAT Gateways by name (substring match)").
		WithOptionalFlag("include-inactive", "Include inactive NAT Gateways in the results").
		WithOptionalFlag("limit", "Limit the number of results returned").
		WithLongDesc("List all NAT Gateways for your account.\n\nThis command retrieves and displays all NAT Gateways, with optional filtering.").
		WithExample("megaport-cli nat-gateway list").
		WithExample("megaport-cli nat-gateway list --location-id 67").
		WithExample("megaport-cli nat-gateway list --name \"my-gw\"").
		WithExample("megaport-cli nat-gateway list --include-inactive").
		WithAliases([]string{"ls"}).
		WithRootCmd(rootCmd).
		Build()

	create = cmdbuilder.NewCommand("create", "Create a new NAT Gateway").
		WithColorAwareRunFunc(CreateNATGateway).
		WithBoolFlagP("yes", "y", false, "Skip the confirmation prompt for creating the NAT Gateway design (no charges are incurred until 'nat-gateway buy'); required when using --json or --json-file").
		WithNATGatewayCreateFlags().
		WithStandardInputFlags().
		WithLongDesc("Create a new NAT Gateway through the Megaport API.\n\nThis command creates a NAT Gateway by providing the necessary details.").
		WithDocumentedRequiredFlag("name", "The name of the NAT Gateway").
		WithDocumentedRequiredFlag("term", fmt.Sprintf("The contract term in months (%s)", validation.FormatIntSlice(validation.ValidContractTerms))).
		WithDocumentedRequiredFlag("speed", "The speed of the NAT Gateway in Mbps").
		WithDocumentedRequiredFlag("location-id", "The ID of the location where the NAT Gateway will be provisioned").
		WithExample("megaport-cli nat-gateway create --interactive").
		WithExample("megaport-cli nat-gateway create --name \"My NAT GW\" --term 12 --speed 1000 --location-id 123").
		WithExample("megaport-cli nat-gateway create --json '{\"name\":\"My NAT GW\",\"term\":12,\"speed\":1000,\"locationId\":123}'").
		WithExample("megaport-cli nat-gateway create --json-file ./nat-gw-config.json").
		WithJSONExample(`{
  "name": "My NAT Gateway",
  "term": 12,
  "speed": 1000,
  "locationId": 123,
  "sessionCount": 100,
  "diversityZone": "blue",
  "autoRenewTerm": false,
  "promoCode": "",
  "resourceTags": {
    "environment": "production",
    "team": "network"
  }
}`).
		WithImportantNote("Required flags can be skipped when using --interactive, --json, or --json-file").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("name", "term", "speed", "location-id").
		Build()

	update = cmdbuilder.NewCommand("update", "Update an existing NAT Gateway").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(UpdateNATGateway).
		WithStandardInputFlags().
		WithNATGatewayUpdateFlags().
		WithLongDesc("Update an existing NAT Gateway.\n\nThis command allows you to update the details of an existing NAT Gateway.").
		WithExample("megaport-cli nat-gateway update [uid] --interactive").
		WithExample("megaport-cli nat-gateway update [uid] --name \"Updated GW\" --speed 2000").
		WithExample("megaport-cli nat-gateway update [uid] --json '{\"name\":\"Updated GW\",\"speed\":2000,\"locationId\":123,\"term\":12}'").
		WithExample("megaport-cli nat-gateway update [uid] --json-file ./update-config.json").
		WithJSONExample(`{
  "name": "Updated NAT Gateway",
  "term": 12,
  "speed": 2000,
  "locationId": 123,
  "sessionCount": 200
}`).
		WithRootCmd(rootCmd).
		Build()

	del = cmdbuilder.NewCommand("delete", "Delete a NAT Gateway").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(DeleteNATGateway).
		WithBoolFlag("force", false, "Skip the confirmation prompt").
		WithBoolFlag("yes", false, "Skip the confirmation prompt (alias of --force)").
		WithLongDesc("Delete a NAT Gateway.\n\nThis command deletes an existing NAT Gateway by its product UID.").
		WithExample("megaport-cli nat-gateway delete a1b2c3d4-e5f6-7890-1234-567890abcdef").
		WithExample("megaport-cli nat-gateway delete a1b2c3d4-e5f6-7890-1234-567890abcdef --force").
		WithImportantNote("This action is irreversible. The NAT Gateway will be deleted immediately.").
		WithAliases([]string{"rm"}).
		WithRootCmd(rootCmd).
		Build()

	validate = cmdbuilder.NewCommand("validate", "Validate a NAT Gateway design without purchasing").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(ValidateNATGateway).
		WithLongDesc("Validate a NAT Gateway design via /v3/networkdesign/validate.\n\nUse this after 'nat-gateway create' to preview pricing and confirm the design is valid before calling 'nat-gateway buy'. No resources are provisioned and no charges are incurred.").
		WithExample("megaport-cli nat-gateway validate a1b2c3d4-e5f6-7890-1234-567890abcdef").
		WithExample("megaport-cli nat-gateway validate a1b2c3d4-e5f6-7890-1234-567890abcdef --output json").
		WithImportantNote("The NAT Gateway must already exist in DESIGN state (create it first with 'nat-gateway create').").
		WithRootCmd(rootCmd).
		Build()

	buy = cmdbuilder.NewCommand("buy", "Purchase a NAT Gateway design to begin provisioning").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(BuyNATGateway).
		WithBoolFlag("yes", false, "Skip the confirmation prompt").
		WithLongDesc("Purchase a NAT Gateway design via /v3/networkdesign/buy.\n\nRun this after 'nat-gateway validate' to kick off provisioning of a NAT Gateway that currently exists in DESIGN state. Billing begins once the order is accepted.").
		WithExample("megaport-cli nat-gateway buy a1b2c3d4-e5f6-7890-1234-567890abcdef").
		WithExample("megaport-cli nat-gateway buy a1b2c3d4-e5f6-7890-1234-567890abcdef --yes").
		WithImportantNote("This action begins billing for the NAT Gateway and cannot be undone without deleting the resource.").
		WithRootCmd(rootCmd).
		Build()

	listSessions = cmdbuilder.NewCommand("list-sessions", "List available NAT Gateway speed/session-count combinations").
		WithOutputFormatRunFunc(ListNATGatewaySessions).
		WithLongDesc("List the available speed and session-count combinations for NAT Gateways.\n\nUse this command to discover valid speed/session-count pairs before creating a NAT Gateway.").
		WithExample("megaport-cli nat-gateway list-sessions").
		WithExample("megaport-cli nat-gateway list-sessions --output json").
		WithRootCmd(rootCmd).
		Build()

	telemetry = cmdbuilder.NewCommand("telemetry", "Get telemetry data for a NAT Gateway").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(GetNATGatewayTelemetry).
		WithNATGatewayTelemetryFlags().
		WithLongDesc("Get telemetry data for a NAT Gateway.\n\nRetrieves metric samples (bits, packets, speed, etc.) for a NAT Gateway over a specified time window.").
		WithDocumentedRequiredFlag("types", "Comma-separated telemetry types (e.g. BITS,PACKETS,SPEED)").
		WithExample("megaport-cli nat-gateway telemetry [uid] --types BITS --days 7").
		WithExample("megaport-cli nat-gateway telemetry [uid] --types BITS,PACKETS --from 2024-01-01T00:00:00Z --to 2024-01-07T00:00:00Z").
		WithExample("megaport-cli nat-gateway telemetry [uid] --types SPEED --days 30 --output json").
		WithImportantNote("Use --days for a rolling window, or --from/--to for an absolute range (they are mutually exclusive)").
		WithRootCmd(rootCmd).
		Build()

	// --types is always required; use MarkFlagRequired for a clear error
	// instead of WithConditionalRequirements which adds interactive/JSON
	// branches that this command doesn't support.
	_ = telemetry.MarkFlagRequired("types")
	telemetry.MarkFlagsMutuallyExclusive("days", "from")
	telemetry.MarkFlagsMutuallyExclusive("days", "to")

	return
}
