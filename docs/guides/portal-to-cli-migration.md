# Migrating from the Megaport Portal to the CLI

This guide helps existing Megaport Portal users transition to `megaport-cli`. It explains why the CLI is useful, how to get started, and maps common Portal actions to their CLI equivalents.

## 1. Why Use the CLI?

The [Megaport Portal](https://portal.megaport.com) is great for exploration and one-off changes. The CLI unlocks workflows that are difficult or impossible in a browser:

- **Automation** — Script repetitive provisioning tasks; no clicking required.
- **CI/CD integration** — Provision and tear down resources from deployment pipelines.
- **Speed** — Create a resource in a single command instead of navigating a multi-step wizard.
- **Reproducibility** — Store JSON config files in version control and replay your topology exactly.
- **Bulk operations** — Loop over resource lists, apply filters, export and re-import configs.

## 2. Installation and Setup

Install `megaport-cli` by following the [installation instructions](../../README.md#installation).

Once installed, create a named profile with your API credentials. You can find your API keys in the Megaport Portal under **My Account → API Keys**:

```sh
megaport-cli config create-profile my-profile \
  --access-key YOUR_ACCESS_KEY \
  --secret-key YOUR_SECRET_KEY \
  --environment production

megaport-cli config use-profile my-profile
```

Alternatively, export credentials as environment variables (useful in CI/CD):

```sh
export MEGAPORT_ACCESS_KEY=YOUR_ACCESS_KEY
export MEGAPORT_SECRET_KEY=YOUR_SECRET_KEY
export MEGAPORT_ENVIRONMENT=production
```

Verify the connection:

```sh
megaport-cli locations list --limit 5
```

## 3. Portal vs CLI — Feature Comparison

| Portal Feature | CLI Equivalent |
|---|---|
| Dashboard / all services | `megaport-cli status` |
| Services → Ports | `megaport-cli ports list` |
| Services → VXCs | `megaport-cli vxc list` |
| Services → MCRs | `megaport-cli mcr list` |
| Services → MVEs | `megaport-cli mve list` |
| Order a port | `megaport-cli ports buy` |
| Order a VXC | `megaport-cli vxc buy` |
| Order an MCR | `megaport-cli mcr buy` |
| Order an MVE | `megaport-cli mve buy` |
| Service details page | `megaport-cli ports get <uid>` |
| Service status badge | `megaport-cli ports status <uid>` |
| Edit a service | `megaport-cli ports update <uid>` |
| Cancel a service | `megaport-cli ports delete <uid>` |
| Locations map | `megaport-cli locations search <name>` |
| Cloud partner ports | `megaport-cli partners list` |
| Network topology | `megaport-cli topology` |
| Billing contact | `megaport-cli billing-market get / set` |
| User management | `megaport-cli users list / create / update` |
| Service keys | `megaport-cli servicekeys list / create` |
| Internet Exchange | `megaport-cli ix list / buy` |

## 4. Common Portal Tasks → CLI Equivalents

### View all services

The Portal dashboard shows all services at once. In the CLI, list each resource type separately:

```sh
megaport-cli ports list
megaport-cli vxc list
megaport-cli mcr list
megaport-cli mve list
```

Filter by name or status:

```sh
megaport-cli ports list --port-name "SYD"
megaport-cli vxc list --status LIVE
megaport-cli vxc list --a-end-uid port-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

Or use `status` for a combined dashboard view:

```sh
megaport-cli status
```

### Create a port

In the Portal: **Order → Port** then fill in the wizard.

In the CLI:

```sh
megaport-cli ports buy \
  --name "My Port" \
  --term 12 \
  --port-speed 1000 \
  --location-id 15 \
  --marketplace-visibility false
```

> **Tip:** Don't know the location ID? Find it first:
> ```sh
> megaport-cli locations search "Sydney"
> ```

Validate the order without placing it (no charges):

```sh
megaport-cli ports validate \
  --name "My Port" \
  --term 12 \
  --port-speed 1000 \
  --location-id 15 \
  --marketplace-visibility false
```

### Create a VXC

```sh
megaport-cli vxc buy \
  --name "My VXC" \
  --rate-limit 1000 \
  --term 12 \
  --a-end-uid port-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --a-end-vlan 100 \
  --b-end-uid port-yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy \
  --b-end-vlan 200
```

For cloud connections (AWS Direct Connect, Azure ExpressRoute, GCP Interconnect), use `--b-end-partner-config` instead. See the [AWS Direct Connect guide](./aws-direct-connect.md) or the [Multi-Cloud Connectivity guide](./multi-cloud-connectivity.md) for details.

### Check service status

```sh
megaport-cli ports status port-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
megaport-cli vxc status vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
megaport-cli mcr status mcr-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

### View service details

```sh
megaport-cli ports get port-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
megaport-cli vxc get vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

### Update a service

```sh
megaport-cli ports update port-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --name "New Name"

megaport-cli vxc update vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --rate-limit 2000
```

### Delete a service

VXCs must be deleted before deleting their parent port or MCR:

```sh
# Delete VXCs first
megaport-cli vxc delete vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

# Then delete the port
megaport-cli ports delete port-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

> **Note:** All `delete` commands cancel resources immediately, and billing stops right away.

### Manage billing contact

```sh
# View current billing details
megaport-cli billing-market get

# Update billing contact
megaport-cli billing-market set \
  --currency USD \
  --language en \
  --billing-contact-name "Jane Smith" \
  --billing-contact-phone "+1-555-0100" \
  --billing-contact-email "jane@example.com" \
  --address1 "123 Main St" \
  --city "New York" \
  --state "NY" \
  --postcode "10001" \
  --country "US" \
  --first-party-id 1558
```

## 5. Automation Tips

### Use JSON input for repeatable deployments

All `buy` and `update` commands accept `--json` (inline) or `--json-file` (from a file):

```sh
megaport-cli ports buy --json-file ./my-port.json
megaport-cli vxc buy --json '{
  "name": "My VXC",
  "rateLimit": 1000,
  "term": 12,
  "aEndUid": "port-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "aEndVlan": 100,
  "bEndUid": "port-yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy",
  "bEndVlan": 200
}'
```

Export an existing resource to get a ready-to-reuse config file:

```sh
megaport-cli ports get port-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --export > my-port.json
```

### Skip confirmation prompts in scripts

Use `--force` to suppress interactive prompts (ports are always deleted immediately):

```sh
megaport-cli ports delete port-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --force
```

### Get machine-readable output

Use `--output json` for structured output that scripts can parse:

```sh
megaport-cli ports list --output json
megaport-cli vxc list --output json \
  --query "[?status=='LIVE'].{name:name,uid:uid,rate_limit:rate_limit}"
```

The `--query` flag accepts a [JMESPath](https://jmespath.org) expression and requires `--output json`.

### Suppress informational output

Use `--quiet` to print only data and errors — ideal for pipelines:

```sh
megaport-cli ports list --quiet --output json
```

### Handle errors with exit codes

`megaport-cli` returns structured exit codes so scripts can branch on failure:

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | General error |
| `2` | Usage error (invalid flags, missing arguments) |
| `3` | Authentication failure |
| `4` | Megaport API error |
| `5` | Cancelled by user |

## 6. Scripting Examples

### List all LIVE VXCs on a port

```sh
#!/usr/bin/env bash
PORT_UID="port-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

megaport-cli vxc list \
  --a-end-uid "$PORT_UID" \
  --status LIVE \
  --output json \
  --query "[].{name:name,uid:uid,rate_limit:rate_limit}"
```

### Create multiple ports from a config directory

```sh
#!/usr/bin/env bash
set -euo pipefail

for config in ./ports/*.json; do
  echo "Provisioning port from ${config}..."
  megaport-cli ports buy --json-file "$config" --yes --quiet
done
echo "All ports created."
```

### Wait for a port to become LIVE

```sh
#!/usr/bin/env bash
PORT_UID="$1"

echo "Waiting for ${PORT_UID} to become LIVE..."
until megaport-cli ports status "$PORT_UID" --output json \
    --query "[0].status" | grep -q '"LIVE"'; do
  sleep 10
done
echo "Port is LIVE."
```

### Tear down all VXCs on a port, then delete the port

```sh
#!/usr/bin/env bash
PORT_UID="$1"

# Collect VXC UIDs attached to this port
VXCS=$(megaport-cli vxc list --a-end-uid "$PORT_UID" --output json \
  --query "[].uid" | tr -d '[]"' | tr ',' '\n')

for uid in $VXCS; do
  echo "Deleting VXC ${uid}..."
  megaport-cli vxc delete "$uid" --force --quiet
done

echo "Deleting port ${PORT_UID}..."
megaport-cli ports delete "$PORT_UID" --force --quiet
echo "Done."
```

## Related Commands

- [`megaport-cli config`](../megaport-cli_config.md) — manage profiles and credentials
- [`megaport-cli ports`](../megaport-cli_ports.md) — manage physical ports
- [`megaport-cli vxc`](../megaport-cli_vxc.md) — manage Virtual Cross Connects
- [`megaport-cli mcr`](../megaport-cli_mcr.md) — manage Megaport Cloud Routers
- [`megaport-cli mve`](../megaport-cli_mve.md) — manage Megaport Virtual Edge devices
- [`megaport-cli locations`](../megaport-cli_locations.md) — find data centres
- [`megaport-cli status`](../megaport-cli_status.md) — dashboard view
