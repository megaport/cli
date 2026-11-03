package billing_market

import (
	"github.com/megaport/megaport-cli/internal/base/cmdbuilder"
	"github.com/spf13/cobra"
)

func AddCommandsTo(rootCmd *cobra.Command) {
	billingMarketCmd := cmdbuilder.NewCommand("billing-market", "Manage billing markets for the Megaport API").
		WithLongDesc("Manage billing markets for the Megaport API.\n\nThis command groups all operations related to billing markets. You can use its subcommands to get and set billing market configurations including currency and billing contact details.").
		WithExample("megaport-cli billing-market get").
		WithExample("megaport-cli billing-market set --currency USD --country US --first-party-id 1558 ...").
		WithRootCmd(rootCmd).
		Build()

	getBillingMarketCmd := cmdbuilder.NewCommand("get", "Get billing market configurations").
		WithLongDesc("Get billing market configurations for the current account.\n\nThis command retrieves and displays all billing markets and their contact details.").
		WithOutputFormatRunFunc(GetBillingMarkets).
		WithExample("megaport-cli billing-market get").
		WithExample("megaport-cli billing-market get -o json").
		WithRootCmd(rootCmd).
		WithAliases([]string{"show"}).
		Build()

	setBillingMarketCmd := cmdbuilder.NewCommand("set", "Set billing market configuration").
		WithLongDesc("Create or update a billing market configuration.\n\nThis command sets up billing market details including currency, billing contact information, and address details.\n\nRequired Fields:\n- `currency`: Billing currency code (e.g., USD, AUD, EUR)\n- `language`: Two-letter language code (e.g., en)\n- `billing-contact-name`: Name of the billing contact\n- `billing-contact-phone`: Phone number of the billing contact in E.164 format: a leading +, digits only, no spaces, dashes or parentheses, and a real number (e.g., +14155552671)\n- `billing-contact-email`: Email address of the billing contact\n- `address1`: Physical address line 1\n- `city`: City\n- `state`: State or region\n- `postcode`: Postal code\n- `country`: Country code (e.g., AU, US)\n- `first-party-id`: Numeric ID for the billing market region\n\nOptional Fields:\n- `address2`: Physical address line 2\n- `po-number`: Purchase order number for tracking\n- `tax-number`: Tax or VAT registration number").
		WithColorAwareRunFunc(SetBillingMarket).
		WithBillingMarketSetFlags().
		WithExample("megaport-cli billing-market set --currency USD --language en --billing-contact-name \"John Doe\" --billing-contact-phone \"+14155552671\" --billing-contact-email \"john@example.com\" --address1 \"123 Main St\" --city \"New York\" --state \"NY\" --postcode \"10001\" --country US --first-party-id 1558").
		WithExample("megaport-cli billing-market set --currency AUD --language en --billing-contact-name \"Jane Smith\" --billing-contact-phone \"+61400000000\" --billing-contact-email \"jane@example.com\" --address1 \"1 Market St\" --city \"Sydney\" --state \"NSW\" --postcode \"2000\" --country AU --first-party-id 808").
		WithImportantNote("Common first-party-id values: 1558 for US (USD), 808 for AU (AUD) — check with Megaport support if unsure of your region's ID").
		WithRootCmd(rootCmd).
		Build()

	billingMarketCmd.AddCommand(getBillingMarketCmd, setBillingMarketCmd)
	rootCmd.AddCommand(billingMarketCmd)
}
