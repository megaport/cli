# looking-glass

MCR Looking Glass diagnostic commands

## Description

MCR Looking Glass diagnostic commands.

The Looking Glass provides visibility into traffic routing on your MCR, helping you troubleshoot connections by showing the status of protocols and routing tables.

### Important Notes
  - Looking Glass commands are read-only diagnostic tools
  - Use --ip flag to filter routes by IP address or prefix

### Example Usage

```sh
  megaport-cli mcr looking-glass ip-routes [mcrUID]
  megaport-cli mcr looking-glass bgp-routes [mcrUID]
  megaport-cli mcr looking-glass bgp-sessions [mcrUID]
```

## Usage

```sh
megaport-cli mcr looking-glass [flags]
```


## Parent Command

* [megaport-cli mcr](megaport-cli_mcr.md)
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|

## Subcommands
* [bgp-neighbor-routes](megaport-cli_mcr_looking-glass_bgp-neighbor-routes.md)
* [bgp-routes](megaport-cli_mcr_looking-glass_bgp-routes.md)
* [bgp-sessions](megaport-cli_mcr_looking-glass_bgp-sessions.md)
* [ip-routes](megaport-cli_mcr_looking-glass_ip-routes.md)
* [ping](megaport-cli_mcr_looking-glass_ping.md)
* [traceroute](megaport-cli_mcr_looking-glass_traceroute.md)

