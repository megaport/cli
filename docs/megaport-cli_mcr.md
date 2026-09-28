# mcr

Manage MCRs in the Megaport API

## Description

Manage MCRs in the Megaport API.

This command groups all operations related to Megaport Cloud Routers (MCRs). MCRs are virtual routing appliances that run in the Megaport network, providing interconnection between your cloud environments and the Megaport fabric.

### Important Notes
  - With MCRs you can establish virtual cross-connects (VXCs) to cloud service providers
  - Create private network connections between different cloud regions
  - Implement hybrid cloud architectures with seamless connectivity
  - Peer with other networks using BGP routing

### Example Usage

```sh
  megaport-cli mcr get [mcrUID]
  megaport-cli mcr list --location-id 67
  megaport-cli mcr buy
  megaport-cli mcr update [mcrUID]
  megaport-cli mcr delete [mcrUID]
```

## Usage

```sh
megaport-cli mcr [flags]
```


## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|

## Subcommands
* [add-ipsec-addon](megaport-cli_mcr_add-ipsec-addon.md)
* [buy](megaport-cli_mcr_buy.md)
* [create-prefix-filter-list](megaport-cli_mcr_create-prefix-filter-list.md)
* [delete](megaport-cli_mcr_delete.md)
* [delete-prefix-filter-list](megaport-cli_mcr_delete-prefix-filter-list.md)
* [get](megaport-cli_mcr_get.md)
* [get-prefix-filter-list](megaport-cli_mcr_get-prefix-filter-list.md)
* [list](megaport-cli_mcr_list.md)
* [list-prefix-filter-lists](megaport-cli_mcr_list-prefix-filter-lists.md)
* [list-tags](megaport-cli_mcr_list-tags.md)
* [lock](megaport-cli_mcr_lock.md)
* [looking-glass](megaport-cli_mcr_looking-glass.md)
* [status](megaport-cli_mcr_status.md)
* [unlock](megaport-cli_mcr_unlock.md)
* [update](megaport-cli_mcr_update.md)
* [update-ipsec-addon](megaport-cli_mcr_update-ipsec-addon.md)
* [update-prefix-filter-list](megaport-cli_mcr_update-prefix-filter-list.md)
* [update-tags](megaport-cli_mcr_update-tags.md)
* [validate](megaport-cli_mcr_validate.md)

