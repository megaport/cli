# set

Set billing market configuration

## Description

Create or update a billing market configuration.

This command sets up billing market details including currency, billing contact information, and address details.

Required Fields:
- `currency`: Billing currency code (e.g., USD, AUD, EUR)
- `language`: Two-letter language code (e.g., en)
- `billing-contact-name`: Name of the billing contact
- `billing-contact-phone`: Phone number of the billing contact in E.164 format: a leading +, digits only, no spaces, dashes or parentheses, and a real number (e.g., +14155552671)
- `billing-contact-email`: Email address of the billing contact
- `address1`: Physical address line 1
- `city`: City
- `state`: State or region
- `postcode`: Postal code
- `country`: Country code (e.g., AU, US)
- `first-party-id`: Numeric ID for the billing market region

Optional Fields:
- `address2`: Physical address line 2
- `po-number`: Purchase order number for tracking
- `tax-number`: Tax or VAT registration number

### Important Notes
  - Common first-party-id values: 1558 for US (USD), 808 for AU (AUD) — check with Megaport support if unsure of your region's ID

### Example Usage

```sh
  megaport-cli billing-market set --currency USD --language en --billing-contact-name "John Doe" --billing-contact-phone "+14155552671" --billing-contact-email "john@example.com" --address1 "123 Main St" --city "New York" --state "NY" --postcode "10001" --country US --first-party-id 1558
  megaport-cli billing-market set --currency AUD --language en --billing-contact-name "Jane Smith" --billing-contact-phone "+61400000000" --billing-contact-email "jane@example.com" --address1 "1 Market St" --city "Sydney" --state "NSW" --postcode "2000" --country AU --first-party-id 808
```

## Usage

```sh
megaport-cli billing-market set [flags]
```


## Parent Command

* [megaport-cli billing-market](megaport-cli_billing-market.md)
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--address1` |  |  | Physical address line 1 | true |
| `--address2` |  |  | Physical address line 2 | false |
| `--billing-contact-email` |  |  | Email address of the billing contact | true |
| `--billing-contact-name` |  |  | Name of the billing contact | true |
| `--billing-contact-phone` |  |  | Phone number of the billing contact in E.164 format: a leading +, digits only, no spaces, dashes or parentheses, and a real number (e.g., +14155552671) | true |
| `--city` |  |  | City | true |
| `--country` |  |  | Country code (e.g., AU, US) | true |
| `--currency` |  |  | Billing currency (e.g., USD, AUD, EUR) | true |
| `--first-party-id` |  | `0` | First party ID for the billing market (e.g., 1558 for US, 808 for AU) | true |
| `--language` |  |  | Two-letter language code (e.g., en) | true |
| `--po-number` |  |  | Purchase order number for tracking | false |
| `--postcode` |  |  | Postal code | true |
| `--state` |  |  | State or region | true |
| `--tax-number` |  |  | Tax or VAT registration number | false |

