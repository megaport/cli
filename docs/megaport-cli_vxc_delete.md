# delete

Delete an existing Virtual Cross Connect (VXC)

## Description

Delete an existing Virtual Cross Connect (VXC) through the Megaport API.

This command allows you to delete an existing VXC by providing its UID. Deletion is immediate.

### Important Notes
  - The VXC is disconnected and billing stops right away
  - Deletion is final and cannot be undone

### Example Usage

```sh
  megaport-cli vxc delete vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
  megaport-cli vxc delete vxc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --force
```

## Usage

```sh
megaport-cli vxc delete [flags]
```


## Parent Command

* [megaport-cli vxc](megaport-cli_vxc.md)

## Aliases

* rm
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--force` | `-f` | `false` | Skip confirmation prompt | false |

