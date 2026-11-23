# vxc

Get the monthly price of a VXC

## Description

Get the monthly price of a VXC.

This command prints the product type, currency, monthly rate, and monthly rack rate for a VXC between two locations at the given speed.

### Required Fields
  - `a-location-id`: ID of the A-End location
  - `b-location-id`: ID of the B-End location
  - `speed`: VXC speed in Mbps

### Optional Fields
  - `a-end-product-type`: Product type of the A-End service, for example MEGAPORT or MCR2
  - `company-uid`: UID of the company that will own the VXC (defaults to your company)
  - `connect-type`: Connect type of the VXC, for example AWS or AZURE
  - `currency`: ISO 4217 currency code (defaults to the company's preferred currency)
  - `product-uid`: UID of an existing VXC to price
  - `term`: Contract term in months (defaults to 1 for a new product)

### Important Notes
  - Required flags can be skipped when using --json or --json-file
  - --company-uid is a flag, not a JSON field, and works with --json or --json-file
  - The command sets productType, so leave it out of the JSON

### Example Usage

```sh
  megaport-cli product pricing vxc --a-location-id 1 --b-location-id 2 --speed 100
  megaport-cli product pricing vxc --a-location-id 1 --b-location-id 2 --speed 500 --term 12 --connect-type AWS
  megaport-cli product pricing vxc --json '{"aLocationId":1,"bLocationId":2,"speed":100}'
```
### JSON Format Example
```json
{
  "aLocationId": 1,
  "bLocationId": 2,
  "speed": 100,
  "term": 12,
  "aEndProductType": "MEGAPORT",
  "connectType": "AWS"
}

```

## Usage

```sh
megaport-cli product pricing vxc [flags]
```


## Parent Command

* [megaport-cli product pricing](megaport-cli_product_pricing.md)
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--a-end-product-type` |  |  | Product type of the A-End service, for example MEGAPORT or MCR2 | false |
| `--a-location-id` |  | `0` | ID of the A-End location | true |
| `--b-location-id` |  | `0` | ID of the B-End location | true |
| `--company-uid` |  |  | UID of the company that will own the VXC (defaults to your company) | false |
| `--connect-type` |  |  | Connect type of the VXC, for example AWS or AZURE | false |
| `--currency` |  |  | ISO 4217 currency code (defaults to the company's preferred currency) | false |
| `--generate-skeleton` |  | `false` | Print a JSON skeleton template for --json or --json-file input and exit | false |
| `--json` |  |  | JSON string containing configuration | false |
| `--json-file` |  |  | Path to JSON file containing configuration | false |
| `--product-uid` |  |  | UID of an existing VXC to price | false |
| `--speed` |  | `0` | VXC speed in Mbps | true |
| `--term` |  | `0` | Contract term in months (defaults to 1 for a new product) | false |

