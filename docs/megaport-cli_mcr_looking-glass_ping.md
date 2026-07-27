# ping

Run a ping from the MCR to a destination

## Description

Run an ICMP ping from the MCR Looking Glass to a destination address.

This command starts a ping operation on the MCR, polls until it completes, then prints the result including RTT statistics and packet loss.

### Required Fields
  - `destination`: Destination IP address to ping

### Example Usage

```sh
  megaport-cli mcr looking-glass ping [mcrUID] --destination 8.8.8.8
  megaport-cli mcr looking-glass ping [mcrUID] --destination 8.8.8.8 --source 10.0.0.1
  megaport-cli mcr looking-glass ping [mcrUID] --destination 8.8.8.8 --packet-count 10 --packet-size 128
```

## Usage

```sh
megaport-cli mcr looking-glass ping [flags]
```


## Parent Command

* [megaport-cli mcr looking-glass](megaport-cli_mcr_looking-glass.md)
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--destination` |  |  | Destination IP address to ping | true |
| `--packet-count` |  | `0` | Number of packets to send (1-60) | false |
| `--packet-size` |  | `0` | Packet size in bytes (1-9186) | false |
| `--source` |  |  | Source IP address to ping from | false |

