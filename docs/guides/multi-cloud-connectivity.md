# Multi-Cloud Connectivity with MCR

This guide walks through connecting AWS Direct Connect, Azure ExpressRoute, and Google Cloud Interconnect simultaneously using a Megaport Cloud Router (MCR) as a central hub. The MCR eliminates the need for a dedicated physical port — each cloud gets its own Virtual Cross Connect (VXC) from the same MCR.

```
        ┌─────────────────────────────┐
        │   Megaport Cloud Router     │
        │   (MCR-MultiCloud)          │
        └──────┬──────────┬───────────┘
               │          │           │
          VLAN 100    VLAN 200    VLAN 300
               │          │           │
         ┌─────┘    ┌─────┘    ┌──────┘
         ▼           ▼          ▼
       AWS          Azure       GCP
  Direct Connect  ExpressRoute  Interconnect
```

## Prerequisites

- Megaport account with API credentials ([Megaport Portal](https://portal.megaport.com))
- AWS account with Direct Connect permissions
- Azure account with an ExpressRoute circuit (and its Service Key)
- GCP project with a VLAN attachment Pairing Key
- `megaport-cli` installed ([installation instructions](../../README.md#installation))

## 1. Configure the CLI

Create a named profile with your Megaport API credentials:

```sh
megaport-cli config create-profile multicloud-guide \
  --access-key YOUR_ACCESS_KEY \
  --secret-key YOUR_SECRET_KEY \
  --environment production

megaport-cli config use-profile multicloud-guide
```

## 2. Find a Location

Search for a data centre that has partner ports for all three clouds:

```sh
megaport-cli locations search "Equinix"
```

Or filter by country:

```sh
megaport-cli locations list --country "US"
```

Note the **Location ID** from the output — you'll use it throughout this guide.

## 3. Create the MCR Hub

Create an MCR at your chosen location. The MCR acts as the A-End for all three VXCs:

```sh
megaport-cli mcr buy \
  --name "MCR-MultiCloud" \
  --port-speed 5000 \
  --location-id 15 \
  --term 12 \
  --marketplace-visibility false
```

> **Note:** Port speed must be one of: 1000, 2500, 5000, 10000, 25000, 50000, or 100000 Mbps. Choose a speed that comfortably accommodates the combined bandwidth of all three cloud connections.

The output includes your new **MCR UID** (e.g. `mcr-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`).

Check provisioning status:

```sh
megaport-cli mcr status mcr-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

## 4. Find Partner Ports

List partner ports for each cloud provider at your location. Use the Location ID from Step 2:

```sh
# AWS Direct Connect
megaport-cli partners list \
  --company-name "Amazon Web Services" \
  --location-id 15

# Azure ExpressRoute
megaport-cli partners list \
  --company-name "Microsoft" \
  --location-id 15

# Google Cloud Interconnect
megaport-cli partners list \
  --company-name "Google" \
  --location-id 15
```

Note the **Product UID** for each cloud:
- **AWS**: you will pass this UID as `--b-end-uid` in the `vxc buy` command.
- **Azure / GCP**: the CLI auto-discovers the B-End UID from the partner config (service key or pairing key); you do not need to pass `--b-end-uid`.

> **Tip:** Use `--output json` with `--query` to extract just the UIDs:
> ```sh
> megaport-cli partners list --company-name "Amazon Web Services" \
>   --location-id 15 --output json --query "[].{name:product_name,uid:uid}"
> ```

## 5. Create VXC to AWS Direct Connect

Connect the MCR to AWS using VLAN 100. Use the AWS partner port UID from Step 4 as `--b-end-uid`, and pass your AWS account details via `--b-end-partner-config`:

```sh
megaport-cli vxc buy \
  --name "MCR-to-AWS" \
  --rate-limit 1000 \
  --term 12 \
  --a-end-uid mcr-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --a-end-vlan 100 \
  --b-end-uid aws-partner-port-uid \
  --b-end-partner-config '{"connectType":"AWS","ownerAccount":"123456789012"}'
```

For a connection with full BGP configuration:

```sh
megaport-cli vxc buy \
  --name "MCR-to-AWS" \
  --rate-limit 1000 \
  --term 12 \
  --a-end-uid mcr-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --a-end-vlan 100 \
  --b-end-uid aws-partner-port-uid \
  --b-end-partner-config '{
    "connectType": "AWS",
    "ownerAccount": "123456789012",
    "asn": 65000,
    "amazonAsn": 64512,
    "authKey": "your-bgp-auth-key",
    "customerIPAddress": "169.254.10.1/30",
    "amazonIPAddress": "169.254.10.2/30"
  }'
```

> **Hosted Connection:** Use `"connectType": "AWSHC"` for an AWS Hosted Connection (lower-bandwidth, shared capacity).

### Partner config fields

| Field | Required | Description |
|---|---|---|
| `connectType` | Yes | `"AWS"` (dedicated) or `"AWSHC"` (hosted) |
| `ownerAccount` | Yes | Your 12-digit AWS account ID |
| `asn` | No | Your BGP ASN |
| `amazonAsn` | No | Amazon's BGP ASN |
| `authKey` | No | BGP authentication key |
| `customerIPAddress` | No | Your BGP peer IP (CIDR, e.g. `169.254.10.1/30`) |
| `amazonIPAddress` | No | Amazon's BGP peer IP (CIDR, e.g. `169.254.10.2/30`) |

Note the **VXC UID** for the AWS connection.

## 6. Create VXC to Azure ExpressRoute

Connect the MCR to Azure using VLAN 200. You need your ExpressRoute circuit's **Service Key** from the Azure Portal:

```sh
megaport-cli vxc buy \
  --name "MCR-to-Azure" \
  --rate-limit 1000 \
  --term 12 \
  --a-end-uid mcr-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --a-end-vlan 200 \
  --b-end-partner-config '{"connectType":"AZURE","serviceKey":"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"}'
```

For a connection with BGP peering configured:

```sh
megaport-cli vxc buy \
  --name "MCR-to-Azure" \
  --rate-limit 1000 \
  --term 12 \
  --a-end-uid mcr-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --a-end-vlan 200 \
  --b-end-partner-config '{
    "connectType": "AZURE",
    "serviceKey": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "peers": [
      {
        "type": "private",
        "peerASN": "65001",
        "primarySubnet": "192.168.10.0/30",
        "secondarySubnet": "192.168.10.4/30",
        "sharedKey": "your-shared-key"
      }
    ]
  }'
```

### Partner config fields

| Field | Required | Description |
|---|---|---|
| `connectType` | Yes | Must be `"AZURE"` |
| `serviceKey` | Yes | Azure ExpressRoute circuit Service Key (UUID) |
| `peers[].type` | No | Peering type: `"private"` or `"microsoft"` |
| `peers[].peerASN` | No | Your BGP ASN (as a string) |
| `peers[].primarySubnet` | No | Primary BGP subnet in CIDR format |
| `peers[].secondarySubnet` | No | Secondary BGP subnet in CIDR format |
| `peers[].sharedKey` | No | BGP shared key |
| `peers[].vlan` | No | VLAN ID for the peering |

Note the **VXC UID** for the Azure connection.

## 7. Create VXC to Google Cloud Interconnect

Connect the MCR to GCP using VLAN 300. You need the **Pairing Key** from your GCP VLAN attachment:

```sh
megaport-cli vxc buy \
  --name "MCR-to-GCP" \
  --rate-limit 1000 \
  --term 12 \
  --a-end-uid mcr-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --a-end-vlan 300 \
  --b-end-partner-config '{"connectType":"GOOGLE","pairingKey":"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/us-east1/1"}'
```

### Partner config fields

| Field | Required | Description |
|---|---|---|
| `connectType` | Yes | Must be `"GOOGLE"` |
| `pairingKey` | Yes | GCP VLAN attachment Pairing Key (from GCP Console) |

Note the **VXC UID** for the GCP connection.

## 8. Verify All Connections

Check the provisioning status of each VXC:

```sh
megaport-cli vxc status vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx   # AWS
megaport-cli vxc status vxc-yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy   # Azure
megaport-cli vxc status vxc-zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz  # GCP
```

Or list all VXCs attached to the MCR in one command:

```sh
megaport-cli vxc list --output json \
  --query "[?a_end_uid=='mcr-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx'].{name:name,status:status}"
```

Get full MCR details to confirm all connections are visible:

```sh
megaport-cli mcr get mcr-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

All three VXCs should show `status: LIVE` before completing cloud-side configuration.

## 9. Complete Cloud-Side Setup

Once all VXCs are LIVE, complete the setup in each cloud console:

**AWS:** In the AWS Console, go to **Direct Connect → Virtual Interfaces** and create a Private Virtual Interface (or Transit VIF) using the connection Megaport provisioned.

**Azure:** The ExpressRoute circuit status should change to **Provisioned** automatically. Configure BGP in **Azure Portal → ExpressRoute circuits → Peerings**.

**GCP:** In the GCP Console, go to **Network Connectivity → Cloud Interconnect → VLAN attachments** and confirm the attachment is active. BGP sessions start automatically once the pairing key is accepted.

## 10. Teardown

Delete VXCs before deleting the MCR:

```sh
# Delete VXCs first (deletion is immediate)
megaport-cli vxc delete vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx  # AWS
megaport-cli vxc delete vxc-yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy  # Azure
megaport-cli vxc delete vxc-zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz  # GCP

# Then delete the MCR
megaport-cli mcr delete mcr-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

## Related Commands

- [`megaport-cli locations`](../megaport-cli_locations.md) — explore available locations
- [`megaport-cli partners`](../megaport-cli_partners.md) — find cloud provider partner ports
- [`megaport-cli mcr`](../megaport-cli_mcr.md) — manage Megaport Cloud Routers
- [`megaport-cli vxc`](../megaport-cli_vxc.md) — manage Virtual Cross Connects
