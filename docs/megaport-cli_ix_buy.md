# buy

Buy an IX through the Megaport API

## Description

Buy an IX through the Megaport API.

This command allows you to purchase an IX by providing the necessary details.

### Required Fields
  - `asn`: ASN (Autonomous System Number) for BGP peering
  - `mac-address`: MAC address for the IX interface
  - `name`: The name of the IX
  - `network-service-type`: The IX type/network service to connect to
  - `product-uid`: The UID of the port to attach the IX to
  - `rate-limit`: Rate limit in Mbps
  - `vlan`: VLAN ID for the IX connection

### Optional Fields
  - `promo-code`: Optional promotion code for discounts
  - `shutdown`: Whether the IX is initially shut down

### Important Notes
  - The product-uid must correspond to a valid port in the Megaport API
  - Required flags (product-uid, name, network-service-type, asn, mac-address, rate-limit, vlan) can be skipped when using --interactive, --json, or --json-file

### Example Usage

```sh
  megaport-cli ix buy --interactive
  megaport-cli ix buy --product-uid port-uid --name "My IX" --network-service-type "Los Angeles IX" --asn 65000 --mac-address "00:11:22:33:44:55" --rate-limit 1000 --vlan 100
  megaport-cli ix buy --json '{"productUid":"port-uid","productName":"My IX","networkServiceType":"Los Angeles IX","asn":65000,"macAddress":"00:11:22:33:44:55","rateLimit":1000,"vlan":100}'
  megaport-cli ix buy --json-file ./ix-config.json
```
### JSON Format Example
```json
{
  "productUid": "port-uid-here",
  "productName": "My IX",
  "networkServiceType": "Los Angeles IX",
  "asn": 65000,
  "macAddress": "00:11:22:33:44:55",
  "rateLimit": 1000,
  "vlan": 100,
  "shutdown": false,
  "promoCode": "PROMO2025"
}

```

## Usage

```sh
megaport-cli ix buy [flags]
```


## Parent Command

* [megaport-cli ix](megaport-cli_ix.md)
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--asn` |  | `0` | ASN (Autonomous System Number) for BGP peering | true |
| `--generate-skeleton` |  | `false` | Print a JSON skeleton template for --json or --json-file input and exit | false |
| `--interactive` | `-i` | `false` | Use interactive mode with prompts | false |
| `--json` |  |  | JSON string containing configuration | false |
| `--json-file` |  |  | Path to JSON file containing configuration | false |
| `--mac-address` |  |  | MAC address for the IX interface | true |
| `--name` |  |  | The name of the IX | true |
| `--network-service-type` |  |  | The IX type/network service to connect to | true |
| `--no-wait` |  | `false` | Do not wait for provisioning to complete | false |
| `--product-uid` |  |  | The UID of the port to attach the IX to | true |
| `--promo-code` |  |  | Optional promotion code for discounts | false |
| `--rate-limit` |  | `0` | Rate limit in Mbps | true |
| `--shutdown` |  | `false` | Whether the IX is initially shut down | false |
| `--vlan` |  | `0` | VLAN ID for the IX connection | true |
| `--yes` | `-y` | `false` | Skip confirmation prompt for purchase; required when using --json or --json-file | false |

