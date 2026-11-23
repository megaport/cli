package product

import (
	"github.com/megaport/megaport-cli/internal/base/cmdbuilder"
	"github.com/spf13/cobra"
)

// AddCommandsTo builds the product commands and adds them to the root command
func AddCommandsTo(rootCmd *cobra.Command) {
	productCmd := cmdbuilder.NewCommand("product", "Manage products in the Megaport API").
		WithLongDesc("Manage products in the Megaport API.\n\nThis command groups operations related to products. You can use the subcommands to list all products, get the type of a specific product, list the products your company can order, or get the monthly price of a product.").
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
		WithLongDesc("Get the monthly price of a new or existing product.\n\nUse the subcommands to price a Port, VXC, MCR, MVE, IX, NAT Gateway, or IP address.").
		WithExample("megaport-cli product pricing port --location-id 1 --speed 1000 --term 12").
		WithExample("megaport-cli product pricing vxc --a-location-id 1 --b-location-id 2 --speed 100").
		WithRootCmd(rootCmd).
		Build()

	pricingPort := cmdbuilder.NewCommand("port", "Get the monthly price of a Port").
		WithOutputFormatRunFunc(GetPortPricing).
		WithIntFlag("location-id", 0, "ID of the location where the Port will be deployed").
		WithIntFlag("speed", 0, "Port speed in Mbps").
		WithIntFlag("term", 0, "Contract term in months (defaults to 1 for a new product)").
		WithFlag("currency", "", "ISO 4217 currency code for a new Port (defaults to the billing currency for the location)").
		WithFlag("product-uid", "", "UID of an existing Port to price").
		WithFlag("company-uid", "", "UID of the company that will own the Port (defaults to your company)").
		WithJSONConfigFlags().
		WithLongDesc("Get the monthly price of a Port.\n\nThis command prints the product type, currency, monthly rate, and monthly rack rate for a Port at the given location and speed.").
		WithDocumentedRequiredFlag("location-id", "ID of the location where the Port will be deployed").
		WithDocumentedRequiredFlag("speed", "Port speed in Mbps").
		WithOptionalFlag("term", "Contract term in months (defaults to 1 for a new product)").
		WithOptionalFlag("currency", "ISO 4217 currency code for a new Port (defaults to the billing currency for the location)").
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
		WithFlag("currency", "", "ISO 4217 currency code for a new VXC (defaults to the billing currency for the A-End location)").
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
		WithOptionalFlag("currency", "ISO 4217 currency code for a new VXC (defaults to the billing currency for the A-End location)").
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

	pricingMCR := cmdbuilder.NewCommand("mcr", "Get the monthly price of an MCR").
		WithOutputFormatRunFunc(GetMCRPricing).
		WithIntFlag("location-id", 0, "ID of the location where the MCR will be deployed").
		WithIntFlag("speed", 0, "MCR speed in Mbps").
		WithIntFlag("term", 0, "Contract term in months (defaults to 1 for a new product)").
		WithFlag("currency", "", "ISO 4217 currency code for a new MCR (defaults to the billing currency for the location)").
		WithFlag("product-uid", "", "UID of an existing MCR to price").
		WithFlag("company-uid", "", "UID of the company that will own the MCR (defaults to your company)").
		WithJSONConfigFlags().
		WithLongDesc("Get the monthly price of an MCR.\n\nThis command prints the product type, currency, monthly rate, and monthly rack rate for an MCR at the given location and speed.").
		WithDocumentedRequiredFlag("location-id", "ID of the location where the MCR will be deployed").
		WithDocumentedRequiredFlag("speed", "MCR speed in Mbps").
		WithOptionalFlag("term", "Contract term in months (defaults to 1 for a new product)").
		WithOptionalFlag("currency", "ISO 4217 currency code for a new MCR (defaults to the billing currency for the location)").
		WithOptionalFlag("product-uid", "UID of an existing MCR to price").
		WithOptionalFlag("company-uid", "UID of the company that will own the MCR (defaults to your company)").
		WithExample("megaport-cli product pricing mcr --location-id 1 --speed 1000 --term 12").
		WithExample("megaport-cli product pricing mcr --location-id 1 --speed 5000 --currency USD --output json").
		WithExample(`megaport-cli product pricing mcr --json '{"locationId":1,"speed":1000,"term":12}'`).
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

	pricingMVE := cmdbuilder.NewCommand("mve", "Get the monthly price of an MVE").
		WithOutputFormatRunFunc(GetMVEPricing).
		WithIntFlag("location-id", 0, "ID of the location where the MVE will be deployed").
		WithFlag("size", "", "MVE size, for example SMALL, MEDIUM, or LARGE").
		WithIntFlag("term", 0, "Contract term in months (defaults to 1 for a new product)").
		WithFlag("currency", "", "ISO 4217 currency code for a new MVE (defaults to the billing currency for the location)").
		WithFlag("product-uid", "", "UID of an existing MVE to price").
		WithFlag("company-uid", "", "UID of the company that will own the MVE (defaults to your company)").
		WithJSONConfigFlags().
		WithLongDesc("Get the monthly price of an MVE.\n\nThis command prints the product type, currency, monthly rate, and monthly rack rate for an MVE at the given location and size.").
		WithDocumentedRequiredFlag("location-id", "ID of the location where the MVE will be deployed").
		WithDocumentedRequiredFlag("size", "MVE size, for example SMALL, MEDIUM, or LARGE").
		WithOptionalFlag("term", "Contract term in months (defaults to 1 for a new product)").
		WithOptionalFlag("currency", "ISO 4217 currency code for a new MVE (defaults to the billing currency for the location)").
		WithOptionalFlag("product-uid", "UID of an existing MVE to price").
		WithOptionalFlag("company-uid", "UID of the company that will own the MVE (defaults to your company)").
		WithExample("megaport-cli product pricing mve --location-id 1 --size SMALL").
		WithExample("megaport-cli product pricing mve --location-id 1 --size MEDIUM --term 12 --output json").
		WithExample(`megaport-cli product pricing mve --json '{"locationId":1,"size":"SMALL"}'`).
		WithJSONExample(`{
  "locationId": 1,
  "size": "SMALL",
  "term": 12,
  "currency": "USD"
}`).
		WithImportantNote("Required flags can be skipped when using --json or --json-file").
		WithImportantNote("--company-uid is a flag, not a JSON field, and works with --json or --json-file").
		WithImportantNote("The command sets productType, so leave it out of the JSON").
		WithImportantNote("The API prices an MVE by size and ignores mveLabel, so set size in the JSON").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("location-id", "size").
		Build()

	pricingIX := cmdbuilder.NewCommand("ix", "Get the monthly price of an IX").
		WithOutputFormatRunFunc(GetIXPricing).
		WithIntFlag("port-location-id", 0, "ID of the location of the Port the IX will attach to").
		WithFlag("ix-type", "", "IX type, for example Los Angeles IX").
		WithIntFlag("speed", 0, "IX speed in Mbps").
		WithIntFlag("term", 0, "Contract term in months").
		WithFlag("currency", "", "ISO 4217 currency code for a new IX (defaults to the billing currency for the location)").
		WithFlag("product-uid", "", "UID of an existing IX to price").
		WithFlag("company-uid", "", "UID of the company that will own the IX (defaults to your company)").
		WithJSONConfigFlags().
		WithLongDesc("Get the monthly price of an IX.\n\nThis command prints the product type, currency, monthly rate, and monthly rack rate for an IX at the given Port location, type, and speed.").
		WithDocumentedRequiredFlag("port-location-id", "ID of the location of the Port the IX will attach to").
		WithDocumentedRequiredFlag("ix-type", "IX type, for example Los Angeles IX").
		WithDocumentedRequiredFlag("speed", "IX speed in Mbps").
		WithOptionalFlag("term", "Contract term in months").
		WithOptionalFlag("currency", "ISO 4217 currency code for a new IX (defaults to the billing currency for the location)").
		WithOptionalFlag("product-uid", "UID of an existing IX to price").
		WithOptionalFlag("company-uid", "UID of the company that will own the IX (defaults to your company)").
		WithExample("megaport-cli product pricing ix --port-location-id 1 --ix-type 'Los Angeles IX' --speed 1000").
		WithExample("megaport-cli product pricing ix --port-location-id 1 --ix-type 'Los Angeles IX' --speed 1000 --term 12 --output json").
		WithExample(`megaport-cli product pricing ix --json '{"portLocationId":1,"ixType":"Los Angeles IX","speed":1000}'`).
		WithJSONExample(`{
  "portLocationId": 1,
  "ixType": "Los Angeles IX",
  "speed": 1000,
  "term": 12
}`).
		WithImportantNote("Required flags can be skipped when using --json or --json-file").
		WithImportantNote("--company-uid is a flag, not a JSON field, and works with --json or --json-file").
		WithImportantNote("The command sets productType, so leave it out of the JSON").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("port-location-id", "ix-type", "speed").
		Build()

	pricingNATGateway := cmdbuilder.NewCommand("nat-gateway", "Get the monthly price of a NAT Gateway").
		WithOutputFormatRunFunc(GetNATGatewayPricing).
		WithIntFlag("location-id", 0, "ID of the location where the NAT Gateway will be deployed").
		WithIntFlag("speed", 0, "NAT Gateway speed in Mbps").
		WithIntFlag("session-count", 0, "Number of NAT sessions, for example 250000").
		WithIntFlag("term", 0, "Contract term in months (defaults to 1 for a new product)").
		WithFlag("currency", "", "ISO 4217 currency code for a new NAT Gateway (defaults to the billing currency for the location)").
		WithFlag("product-uid", "", "UID of an existing NAT Gateway to price").
		WithFlag("company-uid", "", "UID of the company that will own the NAT Gateway (defaults to your company)").
		WithJSONConfigFlags().
		WithLongDesc("Get the monthly price of a NAT Gateway.\n\nThis command prints the product type, currency, monthly rate, and monthly rack rate for a NAT Gateway at the given location, speed, and session count.").
		WithDocumentedRequiredFlag("location-id", "ID of the location where the NAT Gateway will be deployed").
		WithDocumentedRequiredFlag("speed", "NAT Gateway speed in Mbps").
		WithDocumentedRequiredFlag("session-count", "Number of NAT sessions, for example 250000").
		WithOptionalFlag("term", "Contract term in months (defaults to 1 for a new product)").
		WithOptionalFlag("currency", "ISO 4217 currency code for a new NAT Gateway (defaults to the billing currency for the location)").
		WithOptionalFlag("product-uid", "UID of an existing NAT Gateway to price").
		WithOptionalFlag("company-uid", "UID of the company that will own the NAT Gateway (defaults to your company)").
		WithExample("megaport-cli product pricing nat-gateway --location-id 1 --speed 1000 --session-count 250000").
		WithExample("megaport-cli product pricing nat-gateway --location-id 1 --speed 1000 --session-count 250000 --term 12 --output json").
		WithExample(`megaport-cli product pricing nat-gateway --json '{"locationId":1,"speed":1000,"sessionCount":250000}'`).
		WithJSONExample(`{
  "locationId": 1,
  "speed": 1000,
  "sessionCount": 250000,
  "term": 12
}`).
		WithImportantNote("Required flags can be skipped when using --json or --json-file").
		WithImportantNote("--company-uid is a flag, not a JSON field, and works with --json or --json-file").
		WithImportantNote("The command sets productType, so leave it out of the JSON").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("location-id", "speed", "session-count").
		Build()

	pricingIPAddress := cmdbuilder.NewCommand("ip-address", "Get the monthly price of an IP address block").
		WithOutputFormatRunFunc(GetIPAddressPricing).
		WithIntFlag("location-id", 0, "ID of the location where the IP address block will be allocated").
		WithFlag("ip-block", "", "IP address block size, for example /29").
		WithFlag("currency", "", "ISO 4217 currency code (defaults to the billing currency for the location)").
		WithFlag("company-uid", "", "UID of the company that will own the IP address block (defaults to your company)").
		WithJSONConfigFlags().
		WithLongDesc("Get the monthly price of an IP address block.\n\nThis command prints the product type, currency, monthly rate, and monthly rack rate for an IP address block at the given location and block size.").
		WithDocumentedRequiredFlag("location-id", "ID of the location where the IP address block will be allocated").
		WithDocumentedRequiredFlag("ip-block", "IP address block size, for example /29").
		WithOptionalFlag("currency", "ISO 4217 currency code (defaults to the billing currency for the location)").
		WithOptionalFlag("company-uid", "UID of the company that will own the IP address block (defaults to your company)").
		WithExample("megaport-cli product pricing ip-address --location-id 1 --ip-block /29").
		WithExample("megaport-cli product pricing ip-address --location-id 1 --ip-block /24 --currency USD --output json").
		WithExample(`megaport-cli product pricing ip-address --json '{"locationId":1,"ipBlock":"/29"}'`).
		WithJSONExample(`{
  "locationId": 1,
  "ipBlock": "/29",
  "currency": "USD"
}`).
		WithImportantNote("Required flags can be skipped when using --json or --json-file").
		WithImportantNote("--company-uid is a flag, not a JSON field, and works with --json or --json-file").
		WithImportantNote("The command sets productType, so leave it out of the JSON").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("location-id", "ip-block").
		Build()

	pricing.AddCommand(pricingPort, pricingVXC, pricingMCR, pricingMVE, pricingIX, pricingNATGateway, pricingIPAddress)
	productCmd.AddCommand(list, getType, availability, pricing)
	rootCmd.AddCommand(productCmd)
}
