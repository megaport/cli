# ix

Get the monthly price of an IX

## Description

Get the monthly price of an IX.

This command prints the product type, currency, monthly rate, and monthly rack rate for an IX at the given Port location, type, and speed.

### Required Fields
  - `ix-type`: IX type, for example Los Angeles IX
  - `port-location-id`: ID of the location of the Port the IX will attach to
  - `speed`: IX speed in Mbps

### Optional Fields
  - `company-uid`: UID of the company that will own the IX (defaults to your company)
  - `currency`: ISO 4217 currency code for a new IX (defaults to the billing currency for the location)
  - `product-uid`: UID of an existing IX to price
  - `term`: Contract term in months

### Important Notes
  - Required flags can be skipped when using --json or --json-file
  - --company-uid is a flag, not a JSON field, and works with --json or --json-file
  - The command sets productType, so leave it out of the JSON

### Example Usage

```sh
  megaport-cli product pricing ix --port-location-id 1 --ix-type 'Los Angeles IX' --speed 1000
  megaport-cli product pricing ix --port-location-id 1 --ix-type 'Los Angeles IX' --speed 1000 --term 12 --output json
  megaport-cli product pricing ix --json '{"portLocationId":1,"ixType":"Los Angeles IX","speed":1000}'
```
### JSON Format Example
```json
{
  "portLocationId": 1,
  "ixType": "Los Angeles IX",
  "speed": 1000,
  "term": 12
}

```

## Usage

```sh
megaport-cli product pricing ix [flags]
```


## Parent Command

* [megaport-cli product pricing](megaport-cli_product_pricing.md)
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--company-uid` |  |  | UID of the company that will own the IX (defaults to your company) | false |
| `--currency` |  |  | ISO 4217 currency code for a new IX (defaults to the billing currency for the location) | false |
| `--generate-skeleton` |  | `false` | Print a JSON skeleton template for --json or --json-file input and exit | false |
| `--ix-type` |  |  | IX type, for example Los Angeles IX | true |
| `--json` |  |  | JSON string containing configuration | false |
| `--json-file` |  |  | Path to JSON file containing configuration | false |
| `--port-location-id` |  | `0` | ID of the location of the Port the IX will attach to | true |
| `--product-uid` |  |  | UID of an existing IX to price | false |
| `--speed` |  | `0` | IX speed in Mbps | true |
| `--term` |  | `0` | Contract term in months | false |

