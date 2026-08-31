# create

Create a new NAT Gateway

## Description

Create a new NAT Gateway through the Megaport API.

This command creates a NAT Gateway by providing the necessary details.

### Required Fields
  - `location-id`: The ID of the location where the NAT Gateway will be provisioned
  - `name`: The name of the NAT Gateway
  - `speed`: The speed of the NAT Gateway in Mbps
  - `term`: The contract term in months (1, 12, 24, 36, 48, or 60)

### Optional Fields
  - `asn`: The BGP ASN for the NAT Gateway
  - `auto-renew`: Whether to automatically renew the contract term
  - `diversity-zone`: The diversity zone for the NAT Gateway
  - `promo-code`: A promotional code for discounts
  - `resource-tags`: Resource tags as a JSON string (e.g. {"key1":"value1","key2":"value2"})
  - `resource-tags-file`: Path to JSON file containing resource tags
  - `service-level-reference`: A service level reference for the NAT Gateway
  - `session-count`: The number of NAT sessions

### Important Notes
  - Required flags can be skipped when using --interactive, --json, or --json-file

### Example Usage

```sh
  megaport-cli nat-gateway create --interactive
  megaport-cli nat-gateway create --name "My NAT GW" --term 12 --speed 1000 --location-id 123
  megaport-cli nat-gateway create --json '{"name":"My NAT GW","term":12,"speed":1000,"locationId":123}'
  megaport-cli nat-gateway create --json-file ./nat-gw-config.json
```
### JSON Format Example
```json
{
  "name": "My NAT Gateway",
  "term": 12,
  "speed": 1000,
  "locationId": 123,
  "sessionCount": 100,
  "diversityZone": "blue",
  "autoRenewTerm": false,
  "promoCode": "",
  "resourceTags": {
    "environment": "production",
    "team": "network"
  }
}

```

## Usage

```sh
megaport-cli nat-gateway create [flags]
```


## Parent Command

* [megaport-cli nat-gateway](megaport-cli_nat-gateway.md)
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--asn` |  | `0` | The BGP ASN for the NAT Gateway (optional; use a private ASN, e.g. 64512) | false |
| `--auto-renew` |  | `false` | Whether to automatically renew the contract term | false |
| `--diversity-zone` |  |  | The diversity zone for the NAT Gateway (optional) | false |
| `--generate-skeleton` |  | `false` | Print a JSON skeleton template for --json or --json-file input and exit | false |
| `--interactive` | `-i` | `false` | Use interactive mode with prompts | false |
| `--json` |  |  | JSON string containing configuration | false |
| `--json-file` |  |  | Path to JSON file containing configuration | false |
| `--location-id` |  | `0` | The ID of the location where the NAT Gateway will be provisioned | true |
| `--name` |  |  | The name of the NAT Gateway | true |
| `--promo-code` |  |  | A promotional code for discounts (optional) | false |
| `--resource-tags` |  |  | Resource tags as a JSON string (e.g. {"key1":"value1","key2":"value2"}) | false |
| `--resource-tags-file` |  |  | Path to JSON file containing resource tags | false |
| `--service-level-reference` |  |  | A service level reference for the NAT Gateway (optional) | false |
| `--session-count` |  | `0` | The number of NAT sessions (optional) | false |
| `--speed` |  | `0` | The speed of the NAT Gateway in Mbps | true |
| `--term` |  | `0` | The contract term in months (1, 12, 24, 36, 48, or 60) | true |
| `--yes` | `-y` | `false` | Skip the confirmation prompt for creating the NAT Gateway design (no charges are incurred until 'nat-gateway buy'); required when using --json or --json-file | false |

