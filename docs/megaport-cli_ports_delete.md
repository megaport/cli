# delete

Delete a port from your account

## Description

Delete a port from your account in the Megaport API.

This command deletes an existing port by providing the UID of the port as an argument. Ports are deleted immediately; deferred cancellation at the end of the billing period is no longer supported.

### Optional Fields
  - `force`: Skip the confirmation prompt and proceed with deletion
  - `safe-delete`: Fail if the resource has attached VXCs or other active services

### Important Notes
  - All VXCs associated with the port must be deleted before the port can be deleted
  - Ports are deleted immediately; the previous 'terminate later' option is no longer available

### Example Usage

```sh
  megaport-cli ports delete 1a2b3c4d-5e6f-7g8h-9i0j-1k2l3m4n5o6p
  megaport-cli ports delete 1a2b3c4d-5e6f-7g8h-9i0j-1k2l3m4n5o6p --force
```

## Usage

```sh
megaport-cli ports delete [flags]
```


## Parent Command

* [megaport-cli ports](megaport-cli_ports.md)

## Aliases

* rm
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--force` | `-f` | `false` | Skip confirmation prompt | false |
| `--safe-delete` |  | `false` | Fail if the resource has attached VXCs or other active services | false |

