# buy

Buy a port through the Megaport API

## Description

Buy a port through the Megaport API.

This command allows you to purchase a port by providing the necessary details.

### Required Fields
  - `location-id`: The ID of the location where the port will be provisioned
  - `marketplace-visibility`: Whether the port should be visible in the marketplace (true or false)
  - `name`: The name of the port (1-64 characters)
  - `port-speed`: The speed of the port (1000, 10000, or 100000 Mbps)
  - `term`: The term of the port (1, 12, 24, 36, 48, or 60 months)

### Optional Fields
  - `cost-centre`: The cost centre for the port
  - `diversity-zone`: The diversity zone for the port
  - `promo-code`: A promotional code for the port
  - `resource-tags`: Resource tags as a JSON string (e.g. {"key1":"value1","key2":"value2"})
  - `resource-tags-file`: Path to JSON file containing resource tags

### Important Notes
  - Resource tags allow you to categorize resources for organization, filtering, and billing purposes
  - Required flags (name, term, port-speed, location-id, marketplace-visibility) can be skipped when using --interactive, --json, or --json-file

### Example Usage

```sh
  megaport-cli ports buy --interactive
  megaport-cli ports buy --name "My Port" --term 12 --port-speed 10000 --location-id 123 --marketplace-visibility true
  megaport-cli ports buy --name "My Port" --term 12 --port-speed 10000 --location-id 123 --marketplace-visibility true --resource-tags '{"env":"prod","owner":"network-team"}'
  megaport-cli ports buy --json '{"name":"My Port","term":12,"portSpeed":10000,"locationId":123,"marketPlaceVisibility":true}'
  megaport-cli ports buy --json-file ./port-config.json
```
### JSON Format Example
```json
{
  "name": "My Port",
  "term": 12,
  "portSpeed": 10000,
  "locationId": 123,
  "marketPlaceVisibility": true,
  "diversityZone": "blue",
  "costCentre": "IT-2023",
  "resourceTags": {
    "environment": "production",
    "department": "networking",
    "project": "cloud-migration",
    "owner": "john.doe@example.com"
  }
}

```

## Usage

```sh
megaport-cli ports buy [flags]
```


## Parent Command

* [megaport-cli ports](megaport-cli_ports.md)
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--cost-centre` |  |  | Cost centre for billing | false |
| `--diversity-zone` |  |  | Diversity zone for the port | false |
| `--generate-skeleton` |  | `false` | Print a JSON skeleton template for --json or --json-file input and exit | false |
| `--interactive` | `-i` | `false` | Use interactive mode with prompts | false |
| `--json` |  |  | JSON string containing configuration | false |
| `--json-file` |  |  | Path to JSON file containing configuration | false |
| `--location-id` |  | `0` | The ID of the location where the port will be provisioned | true |
| `--marketplace-visibility` |  | `false` | Whether the port should be visible in the marketplace (true or false) | true |
| `--name` |  |  | The name of the port (1-64 characters) | true |
| `--no-wait` |  | `false` | Do not wait for provisioning to complete | false |
| `--port-speed` |  | `0` | The speed of the port (1000, 10000, or 100000 Mbps) | true |
| `--promo-code` |  |  | Promotional code for discounts | false |
| `--resource-tags` |  |  | Resource tags as a JSON string (e.g. {"key1":"value1","key2":"value2"}) | false |
| `--resource-tags-file` |  |  | Path to JSON file containing resource tags | false |
| `--term` |  | `0` | The term of the port (1, 12, 24, 36, 48, or 60 months) | true |
| `--yes` | `-y` | `false` | Skip confirmation prompt for purchase; required when using --json or --json-file | false |

