package product

import (
	"github.com/megaport/megaport-cli/internal/base/cmdbuilder"
	"github.com/spf13/cobra"
)

// AddCommandsTo builds the product commands and adds them to the root command
func AddCommandsTo(rootCmd *cobra.Command) {
	productCmd := cmdbuilder.NewCommand("product", "Manage products in the Megaport API").
		WithLongDesc("Manage products in the Megaport API.\n\nThis command groups operations related to products. You can use the subcommands to list all products, get the type of a specific product, or list the products your company can order.").
		WithExample("megaport-cli product list").
		WithExample("megaport-cli product get-type [productUID]").
		WithExample("megaport-cli product availability").
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

	productCmd.AddCommand(list, getType, availability)
	rootCmd.AddCommand(productCmd)
}
