# traceroute

Run a traceroute from the MCR to a destination

## Description

Run a traceroute from the MCR Looking Glass to a destination address.

This command starts a traceroute operation on the MCR, polls until it completes, then prints the hops.

### Required Fields
  - `destination`: Destination IP address to traceroute to

### Example Usage

```sh
  megaport-cli mcr looking-glass traceroute [mcrUID] --destination 8.8.8.8
  megaport-cli mcr looking-glass traceroute [mcrUID] --destination 8.8.8.8 --source 10.0.0.1
```

## Usage

```sh
megaport-cli mcr looking-glass traceroute [flags]
```


## Parent Command

* [megaport-cli mcr looking-glass](megaport-cli_mcr_looking-glass.md)
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--destination` |  |  | Destination IP address to traceroute to | true |
| `--source` |  |  | Source IP address to traceroute from | false |

