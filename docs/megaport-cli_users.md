# users

Manage users in the Megaport API

## Description

Manage users in the Megaport API.

This command groups operations related to user management. You can use the subcommands to list all company users, get details for a specific user, create a new user, update an existing user, delete a user, deactivate a user, and view your own activity log.

### Example Usage

```sh
  megaport-cli users list
  megaport-cli users get 12345
  megaport-cli users create --interactive
  megaport-cli users update 12345 --first-name "New Name"
  megaport-cli users delete 12345 --force
  megaport-cli users deactivate 12345
  megaport-cli users activity
```

## Usage

```sh
megaport-cli users [flags]
```


## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|

## Subcommands
* [activity](megaport-cli_users_activity.md)
* [create](megaport-cli_users_create.md)
* [deactivate](megaport-cli_users_deactivate.md)
* [delete](megaport-cli_users_delete.md)
* [get](megaport-cli_users_get.md)
* [list](megaport-cli_users_list.md)
* [update](megaport-cli_users_update.md)

