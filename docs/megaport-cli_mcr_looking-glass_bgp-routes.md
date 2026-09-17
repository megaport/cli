# bgp-routes

List BGP routes with full BGP attributes

## Description

List BGP routes from the MCR Looking Glass.

This command retrieves routes learned via BGP with full BGP attributes including AS path, local preference, MED, communities, and origin.

### Important Notes
  - Shows BGP-specific attributes like AS path, local preference, MED, and communities

### Example Usage

```sh
  megaport-cli mcr looking-glass bgp-routes [mcrUID]
  megaport-cli mcr looking-glass bgp-routes [mcrUID] --ip 10.0.0.0/8
```

## Usage

```sh
megaport-cli mcr looking-glass bgp-routes [flags]
```


## Parent Command

* [megaport-cli mcr looking-glass](megaport-cli_mcr_looking-glass.md)
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--ip` |  |  | Filter by IP address or prefix (e.g., 10.0.0.0/8 or 192.168.1.1). The API returns the exact matching routes or the best matching route | false |

