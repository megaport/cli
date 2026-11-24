# ip-address

Get the monthly price of an IP address block

## Description

Get the monthly price of an IP address block.

This command prints the product type, currency, monthly rate, and monthly rack rate for an IP address block at the given location and block size.

### Required Fields
  - `ip-block`: IP address block size, for example /29
  - `location-id`: ID of the location where the IP address block will be allocated

### Optional Fields
  - `company-uid`: UID of the company that will own the IP address block (defaults to your company)
  - `currency`: ISO 4217 currency code (defaults to the billing currency for the location)

### Important Notes
  - Required flags can be skipped when using --json or --json-file
  - --company-uid is a flag, not a JSON field, and works with --json or --json-file
  - The command sets productType, so leave it out of the JSON

### Example Usage

```sh
  megaport-cli product pricing ip-address --location-id 1 --ip-block /29
  megaport-cli product pricing ip-address --location-id 1 --ip-block /24 --currency USD --output json
  megaport-cli product pricing ip-address --json '{"locationId":1,"ipBlock":"/29"}'
```
### JSON Format Example
```json
{
  "locationId": 1,
  "ipBlock": "/29",
  "currency": "USD"
}

```

## Usage

```sh
megaport-cli product pricing ip-address [flags]
```


## Parent Command

* [megaport-cli product pricing](megaport-cli_product_pricing.md)
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--company-uid` |  |  | UID of the company that will own the IP address block (defaults to your company) | false |
| `--currency` |  |  | ISO 4217 currency code (defaults to the billing currency for the location) | false |
| `--generate-skeleton` |  | `false` | Print a JSON skeleton template for --json or --json-file input and exit | false |
| `--ip-block` |  |  | IP address block size, for example /29 | true |
| `--json` |  |  | JSON string containing configuration | false |
| `--json-file` |  |  | Path to JSON file containing configuration | false |
| `--location-id` |  | `0` | ID of the location where the IP address block will be allocated | true |

