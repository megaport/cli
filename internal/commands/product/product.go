package product

import (
	"github.com/megaport/megaport-cli/internal/base/cmdbuilder"
	"github.com/spf13/cobra"
)

// AddCommandsTo builds the product commands and adds them to the root command
func AddCommandsTo(rootCmd *cobra.Command) {
	productCmd := cmdbuilder.NewCommand("product", "Manage products in the Megaport API").
		WithLongDesc("Manage products in the Megaport API.\n\nThis command groups operations related to products. You can use the subcommands to list all products, get the type of a specific product, list the products your company can order, or get the monthly price of a Port or VXC.").
		WithExample("megaport-cli product list").
		WithExample("megaport-cli product get-type [productUID]").
		WithExample("megaport-cli product availability").
		WithExample("megaport-cli product pricing port --location-id 1 --speed 1000").
		WithRootCmd(rootCmd).
		Build()

	list := cmdbuilder.NewCommand("list", "List all products with optional filters").
		WithOutputFormatRunFunc(ListProducts).
		WithLongDesc("List all products available in the Megaport API.\n\nThis command fetches and displays a list of products with details such as UID, name, type, location, speed, and status. By default, only active products are shown.").
		WithBoolFlag("include-inactive", false, "Include products in CANCELLED, DECOMMISSIONED, or DECOMMISSIONING states").
		WithOptionalFlag("include-inactive", "Include products in CANCELLED, DECOMMISSIONED, or DECOMMISSIONING states").
		WithIntFlag("limit", 0, "Maximum number of results to display (0 = unlimited)").
		WithExample("megaport-cli product list").
		WithExample("megaport-cli product list --include-inactive").
		WithExample("megaport-cli product list --limit 10").
		WithRootCmd(rootCmd).
		Build()

	getType := cmdbuilder.NewCommand("get-type", "Get the type of a product by UID").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(GetProductType).
		WithLongDesc("Get the type of a product based on a product UID.\n\nThis command retrieves and displays the product type for the specified product UID.").
		WithExample("megaport-cli product get-type [productUID]").
		WithRootCmd(rootCmd).
		Build()

	availability := cmdbuilder.NewCommand("availability", "List the products your company can order and the markets for each").
		WithArgs(cobra.NoArgs).
		WithOutputFormatRunFunc(ListProductAvailability).
		WithLongDesc("List the products your company can order and the markets for each.\n\nThis command looks up the company your API key belongs to. It prints one row for each product and market.").
		WithExample("megaport-cli product availability").
		WithExample("megaport-cli product availability --output json").
		WithRootCmd(rootCmd).
		Build()

	pricing := cmdbuilder.NewCommand("pricing", "Get the monthly price of a product").
		WithLongDesc("Get the monthly price of a new or existing product.\n\nUse the subcommands to price a Port or a VXC.").
		WithExample("megaport-cli product pricing port --location-id 1 --speed 1000 --term 12").
		WithExample("megaport-cli product pricing vxc --a-location-id 1 --b-location-id 2 --speed 100").
		WithRootCmd(rootCmd).
		Build()

	pricingPort := cmdbuilder.NewCommand("port", "Get the monthly price of a Port").
		WithOutputFormatRunFunc(GetPortPricing).
		WithIntFlag("location-id", 0, "ID of the location where the Port will be deployed").
		WithIntFlag("speed", 0, "Port speed in Mbps").
		WithIntFlag("term", 0, "Contract term in months (defaults to 1 for a new product)").
		WithFlag("currency", "", "ISO 4217 currency code for a new Port (defaults to your billing currency for the location)").
		WithFlag("product-uid", "", "UID of an existing Port to price").
		WithFlag("company-uid", "", "UID of the company that will own the Port (defaults to your company)").
		WithJSONConfigFlags().
		WithLongDesc("Get the monthly price of a Port.\n\nThis command prints the product type, currency, monthly rate, and monthly rack rate for a Port at the given location and speed.").
		WithDocumentedRequiredFlag("location-id", "ID of the location where the Port will be deployed").
		WithDocumentedRequiredFlag("speed", "Port speed in Mbps").
		WithOptionalFlag("term", "Contract term in months (defaults to 1 for a new product)").
		WithOptionalFlag("currency", "ISO 4217 currency code for a new Port (defaults to your billing currency for the location)").
		WithOptionalFlag("product-uid", "UID of an existing Port to price").
		WithOptionalFlag("company-uid", "UID of the company that will own the Port (defaults to your company)").
		WithExample("megaport-cli product pricing port --location-id 1 --speed 1000 --term 12").
		WithExample("megaport-cli product pricing port --location-id 1 --speed 10000 --currency USD --output json").
		WithExample(`megaport-cli product pricing port --json '{"locationId":1,"speed":1000,"term":12}'`).
		WithJSONExample(`{
  "locationId": 1,
  "speed": 1000,
  "term": 12,
  "currency": "USD"
}`).
		WithImportantNote("Required flags can be skipped when using --json or --json-file").
		WithImportantNote("--company-uid is a flag, not a JSON field, and works with --json or --json-file").
		WithImportantNote("The command sets productType, so leave it out of the JSON").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("location-id", "speed").
		Build()

	pricingVXC := cmdbuilder.NewCommand("vxc", "Get the monthly price of a VXC").
		WithOutputFormatRunFunc(GetVXCPricing).
		WithIntFlag("a-location-id", 0, "ID of the A-End location").
		WithIntFlag("b-location-id", 0, "ID of the B-End location").
		WithIntFlag("speed", 0, "VXC speed in Mbps").
		WithIntFlag("term", 0, "Contract term in months (defaults to 1 for a new product)").
		WithFlag("a-end-product-type", "", "Product type of the A-End service, for example MEGAPORT or MCR2").
		WithFlag("connect-type", "", "Connect type of the VXC, for example AWS or AZURE").
		WithFlag("currency", "", "ISO 4217 currency code for a new VXC (defaults to your billing currency)").
		WithFlag("product-uid", "", "UID of an existing VXC to price").
		WithFlag("company-uid", "", "UID of the company that will own the VXC (defaults to your company)").
		WithJSONConfigFlags().
		WithLongDesc("Get the monthly price of a VXC.\n\nThis command prints the product type, currency, monthly rate, and monthly rack rate for a VXC between two locations at the given speed.").
		WithDocumentedRequiredFlag("a-location-id", "ID of the A-End location").
		WithDocumentedRequiredFlag("b-location-id", "ID of the B-End location").
		WithDocumentedRequiredFlag("speed", "VXC speed in Mbps").
		WithOptionalFlag("term", "Contract term in months (defaults to 1 for a new product)").
		WithOptionalFlag("a-end-product-type", "Product type of the A-End service, for example MEGAPORT or MCR2").
		WithOptionalFlag("connect-type", "Connect type of the VXC, for example AWS or AZURE").
		WithOptionalFlag("currency", "ISO 4217 currency code for a new VXC (defaults to your billing currency)").
		WithOptionalFlag("product-uid", "UID of an existing VXC to price").
		WithOptionalFlag("company-uid", "UID of the company that will own the VXC (defaults to your company)").
		WithExample("megaport-cli product pricing vxc --a-location-id 1 --b-location-id 2 --speed 100").
		WithExample("megaport-cli product pricing vxc --a-location-id 1 --b-location-id 2 --speed 500 --term 12 --connect-type AWS").
		WithExample(`megaport-cli product pricing vxc --json '{"aLocationId":1,"bLocationId":2,"speed":100}'`).
		WithJSONExample(`{
  "aLocationId": 1,
  "bLocationId": 2,
  "speed": 100,
  "term": 12,
  "aEndProductType": "MEGAPORT",
  "connectType": "AWS"
}`).
		WithImportantNote("Required flags can be skipped when using --json or --json-file").
		WithImportantNote("--company-uid is a flag, not a JSON field, and works with --json or --json-file").
		WithImportantNote("The command sets productType, so leave it out of the JSON").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("a-location-id", "b-location-id", "speed").
		Build()

	pricing.AddCommand(pricingPort, pricingVXC)
	productCmd.AddCommand(list, getType, availability, pricing)
	rootCmd.AddCommand(productCmd)
}
