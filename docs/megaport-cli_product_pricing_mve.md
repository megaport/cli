# mve

Get the monthly price of an MVE

## Description

Get the monthly price of an MVE.

This command prints the product type, currency, monthly rate, and monthly rack rate for an MVE at the given location and size.

### Required Fields
  - `location-id`: ID of the location where the MVE will be deployed
  - `size`: MVE size, for example SMALL, MEDIUM, or LARGE

### Optional Fields
  - `company-uid`: UID of the company that will own the MVE (defaults to your company)
  - `currency`: ISO 4217 currency code for a new MVE (defaults to the billing currency for the location)
  - `product-uid`: UID of an existing MVE to price
  - `term`: Contract term in months (defaults to 1 for a new product)

### Important Notes
  - Required flags can be skipped when using --json or --json-file
  - --company-uid is a flag, not a JSON field, and works with --json or --json-file
  - The command sets productType, so leave it out of the JSON

### Example Usage

```sh
  megaport-cli product pricing mve --location-id 1 --size SMALL
  megaport-cli product pricing mve --location-id 1 --size MEDIUM --term 12 --output json
  megaport-cli product pricing mve --json '{"locationId":1,"size":"SMALL"}'
```
### JSON Format Example
```json
{
  "locationId": 1,
  "size": "SMALL",
  "term": 12,
  "currency": "USD"
}

```

## Usage

```sh
megaport-cli product pricing mve [flags]
```


## Parent Command

* [megaport-cli product pricing](megaport-cli_product_pricing.md)
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--company-uid` |  |  | UID of the company that will own the MVE (defaults to your company) | false |
| `--currency` |  |  | ISO 4217 currency code for a new MVE (defaults to the billing currency for the location) | false |
| `--generate-skeleton` |  | `false` | Print a JSON skeleton template for --json or --json-file input and exit | false |
| `--json` |  |  | JSON string containing configuration | false |
| `--json-file` |  |  | Path to JSON file containing configuration | false |
| `--location-id` |  | `0` | ID of the location where the MVE will be deployed | true |
| `--product-uid` |  |  | UID of an existing MVE to price | false |
| `--size` |  |  | MVE size, for example SMALL, MEDIUM, or LARGE | true |
| `--term` |  | `0` | Contract term in months (defaults to 1 for a new product) | false |

