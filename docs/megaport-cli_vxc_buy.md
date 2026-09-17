# buy

Purchase a new VXC

## Description

Purchase a new Megaport Virtual Cross Connect (VXC) through the Megaport API.

This command allows you to create a VXC by providing the necessary details.

### Required Fields
  - `a-end-uid`: UID of the A-End product
  - `a-end-vlan`: VLAN for A-End (0=auto-assign, -1=untagged, 2-4094 for specific VLAN (1 is reserved))
  - `b-end-uid`: UID of the B-End product (if not using partner configuration)
  - `b-end-vlan`: VLAN for B-End (0=auto-assign, -1=untagged, 2-4094 for specific VLAN (1 is reserved))
  - `name`: Name of the VXC
  - `rate-limit`: Bandwidth in Mbps
  - `term`: Contract term in months (1, 12, 24, 36, 48, or 60)

### Optional Fields
  - `resource-tags`: Resource tags as a JSON string (e.g. {"key1":"value1","key2":"value2"})
  - `resource-tags-file`: Path to JSON file containing resource tags

### Important Notes
  - To order an MCR IPsec tunnel, set interfaceType "ipSecTunnel" on the vRouter interface and provide ipSecTunnelOptions; treat the preSharedKey as a secret and avoid committing populated config files to source control

### Example Usage

```sh
  megaport-cli vxc buy --interactive
  megaport-cli vxc buy --name "My VXC" --rate-limit 1000 --term 12 --a-end-uid port-123 --b-end-uid port-456 --a-end-vlan 100 --b-end-vlan 200
  megaport-cli vxc buy --name "My VXC" --rate-limit 1000 --term 12 --a-end-uid port-123 --b-end-uid port-456 --a-end-vlan 100 --b-end-vlan 200 --resource-tags '{"environment":"production","team":"networking"}'
  megaport-cli vxc buy --json '{"vxcName":"My VXC","rateLimit":1000,"term":12,"portUid":"port-123","aEndConfiguration":{"vlan":100},"bEndConfiguration":{"productUID":"port-456","vlan":200},"resourceTags":{"environment":"production","owner":"network-team"}}'
  megaport-cli vxc buy --name "IPsec VXC" --rate-limit 1000 --term 12 --a-end-uid port-123 --b-end-uid port-456 --a-end-partner-config '{"connectType":"VROUTER","interfaces":[{"interfaceType":"ipSecTunnel","ipSecTunnelOptions":{"sourceIpAddress":"192.0.2.1","destinationIpAddress":"198.51.100.1","preSharedKey":"<your-psk>","phase1Lifetime":28800,"phase2Lifetime":3600}}]}'
  megaport-cli vxc buy --json-file ./vxc-config.json
```
### JSON Format Example
```json
{
  "vxcName": "My VXC",
  "rateLimit": 1000, 
  "term": 12,
  "portUid": "port-123",
  "aEndConfiguration": {
    "vlan": 100
  },
  "bEndConfiguration": {
    "productUID": "port-456",
    "vlan": 200
  },
  "costCentre": "IT Department",
  "resourceTags": {
    "environment": "production",
    "owner": "network-team",
    "project": "cloud-migration"
  }
}

```

## Usage

```sh
megaport-cli vxc buy [flags]
```


## Parent Command

* [megaport-cli vxc](megaport-cli_vxc.md)
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--a-end-inner-vlan` |  | `0` | Inner VLAN for A-End (0=none, -1=untagged, 2-4094 for specific VLAN (1 is reserved)) | false |
| `--a-end-partner-config` |  |  | JSON string with A-End partner configuration | false |
| `--a-end-uid` |  |  | UID of the A-End product | true |
| `--a-end-vlan` |  | `0` | VLAN for A-End (0=auto-assign, -1=untagged, 2-4094 for specific VLAN (1 is reserved)) | true |
| `--a-end-vnic-index` |  | `0` | vNIC index for A-End MVE | false |
| `--b-end-inner-vlan` |  | `0` | Inner VLAN for B-End (0=none, -1=untagged, 2-4094 for specific VLAN (1 is reserved)) | false |
| `--b-end-partner-config` |  |  | JSON string with B-End partner configuration | false |
| `--b-end-uid` |  |  | UID of the B-End product (if not using partner configuration) | true |
| `--b-end-vlan` |  | `0` | VLAN for B-End (0=auto-assign, -1=untagged, 2-4094 for specific VLAN (1 is reserved)) | true |
| `--b-end-vnic-index` |  | `0` | vNIC index for B-End MVE | false |
| `--cost-centre` |  |  | Cost centre for billing | false |
| `--generate-skeleton` |  | `false` | Print a JSON skeleton template for --json or --json-file input and exit | false |
| `--interactive` | `-i` | `false` | Use interactive mode with prompts | false |
| `--json` |  |  | JSON string containing configuration | false |
| `--json-file` |  |  | Path to JSON file containing configuration | false |
| `--name` |  |  | Name of the VXC | true |
| `--no-wait` |  | `false` | Do not wait for provisioning to complete | false |
| `--promo-code` |  |  | Promotional code | false |
| `--rate-limit` |  | `0` | Bandwidth in Mbps | true |
| `--resource-tags` |  |  | Resource tags as a JSON string (e.g. {"key1":"value1","key2":"value2"}) | false |
| `--resource-tags-file` |  |  | Path to JSON file containing resource tags | false |
| `--service-key` |  |  | Service key | false |
| `--term` |  | `0` | Contract term in months (1, 12, 24, 36, 48, or 60) | true |
| `--yes` | `-y` | `false` | Skip confirmation prompt for purchase; required when using --json or --json-file | false |

