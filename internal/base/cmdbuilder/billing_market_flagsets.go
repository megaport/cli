package cmdbuilder

// WithBillingMarketSetFlags adds all flags needed for setting a billing market
func (b *CommandBuilder) WithBillingMarketSetFlags() *CommandBuilder {
	b.WithFlag("currency", "", "Billing currency (e.g., USD, AUD, EUR)")
	b.WithFlag("language", "", "Two-letter language code (e.g., en)")
	b.WithFlag("billing-contact-name", "", "Name of the billing contact")
	b.WithFlag("billing-contact-phone", "", "Phone number of the billing contact in E.164 format: a leading +, digits only, no spaces, dashes or parentheses, and a real number (e.g., +14155552671)")
	b.WithFlag("billing-contact-email", "", "Email address of the billing contact")
	b.WithFlag("address1", "", "Physical address line 1")
	b.WithFlag("address2", "", "Physical address line 2")
	b.WithFlag("city", "", "City")
	b.WithFlag("state", "", "State or region")
	b.WithFlag("postcode", "", "Postal code")
	b.WithFlag("country", "", "Country code (e.g., AU, US)")
	b.WithFlag("po-number", "", "Purchase order number for tracking")
	b.WithFlag("tax-number", "", "Tax or VAT registration number")
	b.WithIntFlag("first-party-id", 0, "First party ID for the billing market (e.g., 1558 for US, 808 for AU)")

	b.ReflagCmd("currency", "language", "billing-contact-name", "billing-contact-phone",
		"billing-contact-email", "address1", "city", "state", "postcode", "country", "first-party-id")
	return b
}
