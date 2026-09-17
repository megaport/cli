# bgp-neighbor-routes

List routes advertised to or received from a BGP neighbor

## Description

List routes advertised to or received from a specific BGP neighbor.

The neighbor is identified by its BGP peer IP address, the peer IP configured on the MCR's BGP connection. Direction is 'received' for routes learned from the neighbor or 'advertised' for routes sent to it.

### Important Notes
  - Direction must be 'advertised' or 'received'
  - The peer IP is the BGP neighbor address on the MCR's BGP connection, for example 169.254.0.1
  - The API has no address filter on this endpoint, so --ip is applied locally after the routes are fetched

### Example Usage

```sh
  megaport-cli mcr looking-glass bgp-neighbor-routes [mcrUID] [peerIP] advertised
  megaport-cli mcr looking-glass bgp-neighbor-routes [mcrUID] [peerIP] received
  megaport-cli mcr looking-glass bgp-neighbor-routes [mcrUID] [peerIP] received --ip 10.0.0.0/8
```

## Usage

```sh
megaport-cli mcr looking-glass bgp-neighbor-routes [flags]
```


## Parent Command

* [megaport-cli mcr looking-glass](megaport-cli_mcr_looking-glass.md)
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--ip` |  |  | Filter by IP address or prefix (e.g., 10.0.0.0/8 or 192.168.1.1). Keeps routes that contain the address or fall inside the prefix | false |

