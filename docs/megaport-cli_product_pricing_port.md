# port

Get the monthly price of a Port

## Description

Get the monthly price of a Port.

This command prints the product type, currency, monthly rate, and monthly rack rate for a Port at the given location and speed.

### Required Fields
  - `location-id`: ID of the location where the Port will be deployed
  - `speed`: Port speed in Mbps

### Optional Fields
  - `company-uid`: UID of the company that will own the Port (defaults to your company)
  - `currency`: ISO 4217 currency code (defaults to the company's preferred currency)
  - `product-uid`: UID of an existing Port to price
  - `term`: Contract term in months (defaults to 1 for a new product)

### Important Notes
  - Required flags can be skipped when using --json or --json-file
  - --company-uid is a flag, not a JSON field, and works with --json or --json-file
  - The command sets productType, so leave it out of the JSON

### Example Usage

```sh
  megaport-cli product pricing port --location-id 1 --speed 1000 --term 12
  megaport-cli product pricing port --location-id 1 --speed 10000 --currency USD --output json
  megaport-cli product pricing port --json '{"locationId":1,"speed":1000,"term":12}'
```
### JSON Format Example
```json
{
  "locationId": 1,
  "speed": 1000,
  "term": 12,
  "currency": "USD"
}

```

## Usage

```sh
megaport-cli product pricing port [flags]
```


## Parent Command

* [megaport-cli product pricing](megaport-cli_product_pricing.md)
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--company-uid` |  |  | UID of the company that will own the Port (defaults to your company) | false |
| `--currency` |  |  | ISO 4217 currency code (defaults to the company's preferred currency) | false |
| `--generate-skeleton` |  | `false` | Print a JSON skeleton template for --json or --json-file input and exit | false |
| `--json` |  |  | JSON string containing configuration | false |
| `--json-file` |  |  | Path to JSON file containing configuration | false |
| `--location-id` |  | `0` | ID of the location where the Port will be deployed | true |
| `--product-uid` |  |  | UID of an existing Port to price | false |
| `--speed` |  | `0` | Port speed in Mbps | true |
| `--term` |  | `0` | Contract term in months (defaults to 1 for a new product) | false |

