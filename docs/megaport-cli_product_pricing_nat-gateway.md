# nat-gateway

Get the monthly price of a NAT Gateway

## Description

Get the monthly price of a NAT Gateway.

This command prints the product type, currency, monthly rate, and monthly rack rate for a NAT Gateway at the given location, speed, and session count.

### Required Fields
  - `location-id`: ID of the location where the NAT Gateway will be deployed
  - `session-count`: Number of NAT sessions, for example 250000
  - `speed`: NAT Gateway speed in Mbps

### Optional Fields
  - `company-uid`: UID of the company that will own the NAT Gateway (defaults to your company)
  - `currency`: ISO 4217 currency code for a new NAT Gateway (defaults to the billing currency for the location)
  - `product-uid`: UID of an existing NAT Gateway to price
  - `term`: Contract term in months (defaults to 1 for a new product)

### Important Notes
  - Required flags can be skipped when using --json or --json-file
  - --company-uid is a flag, not a JSON field, and works with --json or --json-file
  - The command sets productType, so leave it out of the JSON

### Example Usage

```sh
  megaport-cli product pricing nat-gateway --location-id 1 --speed 1000 --session-count 250000
  megaport-cli product pricing nat-gateway --location-id 1 --speed 1000 --session-count 250000 --term 12 --output json
  megaport-cli product pricing nat-gateway --json '{"locationId":1,"speed":1000,"sessionCount":250000}'
```
### JSON Format Example
```json
{
  "locationId": 1,
  "speed": 1000,
  "sessionCount": 250000,
  "term": 12
}

```

## Usage

```sh
megaport-cli product pricing nat-gateway [flags]
```


## Parent Command

* [megaport-cli product pricing](megaport-cli_product_pricing.md)
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--company-uid` |  |  | UID of the company that will own the NAT Gateway (defaults to your company) | false |
| `--currency` |  |  | ISO 4217 currency code for a new NAT Gateway (defaults to the billing currency for the location) | false |
| `--generate-skeleton` |  | `false` | Print a JSON skeleton template for --json or --json-file input and exit | false |
| `--json` |  |  | JSON string containing configuration | false |
| `--json-file` |  |  | Path to JSON file containing configuration | false |
| `--location-id` |  | `0` | ID of the location where the NAT Gateway will be deployed | true |
| `--product-uid` |  |  | UID of an existing NAT Gateway to price | false |
| `--session-count` |  | `0` | Number of NAT sessions, for example 250000 | true |
| `--speed` |  | `0` | NAT Gateway speed in Mbps | true |
| `--term` |  | `0` | Contract term in months (defaults to 1 for a new product) | false |

