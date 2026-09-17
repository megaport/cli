# buy

Purchase a new Megaport Virtual Edge (MVE) device

## Description

Purchase a new Megaport Virtual Edge (MVE) device through the Megaport API.

This command allows you to purchase an MVE by providing the necessary details.

### Required Fields
  - `location-id`: The ID of the location where the MVE will be provisioned
  - `name`: The name of the MVE
  - `term`: The term of the MVE (1, 12, 24, 36, 48, or 60 months)
  - `vendor-config`: JSON string with vendor-specific configuration (for flag mode)
  - `vnics`: JSON array of network interfaces (for flag mode)

### Optional Fields
  - `cost-centre`: Cost centre for billing
  - `diversity-zone`: The diversity zone for the MVE
  - `promo-code`: Promotional code for discounts
  - `resource-tags`: Resource tags as a JSON string (e.g. {"key1":"value1","key2":"value2"})
  - `resource-tags-file`: Path to JSON file containing resource tags

### Important Notes
  - For production deployments, you may want to use a JSON file to manage complex configurations
  - Treat fields like adminPassword, adminSshPublicKey, sshPublicKey, accountKey, and any other credentials as secrets — replace the placeholders in these examples and avoid committing populated config files to source control.
  - To list available images and their IDs, use: megaport-cli mve list-images
  - To list available sizes, use: megaport-cli mve list-sizes
  - Location IDs can be retrieved with: megaport-cli locations list
  - Resource tags are key-value pairs used for organizing and managing resources.

### Example Usage

```sh
  megaport-cli mve buy --interactive
  megaport-cli mve buy --name "My MVE" --term 12 --location-id 123 --vendor-config '{"vendor":"cisco","imageId":123,"productSize":"MEDIUM","manageLocally":true,"adminSshPublicKey":"ssh-rsa AAAA...","sshPublicKey":"ssh-rsa AAAA..."}' --vnics '[{"description":"Data Plane","vlan":100}]' --resource-tags '{"env":"prod","owner":"netops"}'
  megaport-cli mve buy --json '{"name":"My MVE","term":12,"locationId":123,"vendorConfig":{"vendor":"cisco","imageId":123,"productSize":"MEDIUM","manageLocally":true,"adminSshPublicKey":"ssh-rsa AAAA...","sshPublicKey":"ssh-rsa AAAA..."},"vnics":[{"description":"Data Plane","vlan":100}],"resourceTags":{"env":"prod","owner":"netops"}}'
  megaport-cli mve buy --json-file ./mve-config.json
```
### JSON Format Example
```json
{
  "name": "My MVE Display Name",
  "term": 12,
  "locationId": 123,
  "diversityZone": "blue",
  "promoCode": "PROMO2023",
  "costCentre": "Marketing Dept",
  "vendorConfig": {
    "vendor": "cisco",
    "imageId": 123,
    "productSize": "MEDIUM",
    "mveLabel": "custom-label",
    "manageLocally": true,
    "adminSshPublicKey": "ssh-rsa AAAA...",
    "sshPublicKey": "ssh-rsa AAAA...",
    "adminPassword": "<your-admin-password>",
    "cloudInit": "#cloud-config\npackages:\n - nginx\n"
  },
  "vnics": [
    {"description": "Data Plane", "vlan": 100},
    {"description": "Management", "vlan": 200}
  ],
  "resourceTags": {
    "environment": "production",
    "billing_code": "BC12345",
    "owner_team": "network-operations"
  }
}

```

## Usage

```sh
megaport-cli mve buy [flags]
```


## Parent Command

* [megaport-cli mve](megaport-cli_mve.md)
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--cost-centre` |  |  | Cost centre for billing | false |
| `--diversity-zone` |  |  | The diversity zone for the MVE | false |
| `--generate-skeleton` |  | `false` | Print a JSON skeleton template for --json or --json-file input and exit | false |
| `--interactive` | `-i` | `false` | Use interactive mode with prompts | false |
| `--json` |  |  | JSON string containing configuration | false |
| `--json-file` |  |  | Path to JSON file containing configuration | false |
| `--location-id` |  | `0` | The ID of the location where the MVE will be provisioned | true |
| `--name` |  |  | The name of the MVE | true |
| `--no-wait` |  | `false` | Do not wait for provisioning to complete | false |
| `--promo-code` |  |  | Promotional code for discounts | false |
| `--resource-tags` |  |  | Resource tags as a JSON string (e.g. {"key1":"value1","key2":"value2"}) | false |
| `--resource-tags-file` |  |  | Path to JSON file containing resource tags | false |
| `--term` |  | `0` | The term of the MVE (1, 12, 24, 36, 48, or 60 months) | true |
| `--vendor-config` |  |  | JSON string with vendor-specific configuration (for flag mode) | true |
| `--vnics` |  |  | JSON array of network interfaces (for flag mode) | true |
| `--yes` | `-y` | `false` | Skip confirmation prompt for purchase; required when using --json or --json-file | false |

