# create

Create a new user

## Description

Create a new user in your Megaport company.

This command allows you to create a new user by providing the necessary details.

### Required Fields
  - `email`: Email address of the user
  - `first-name`: First name of the user
  - `last-name`: Last name of the user
  - `position`: Position/role (Company Admin, Technical Admin, Technical Contact, Finance, Financial Contact, Read Only)

### Optional Fields
  - `phone`: Phone number in E.164 format: a leading +, digits only, no spaces, dashes or parentheses, and a real number (e.g., +61412345678)

### Important Notes
  - Valid positions: Company Admin, Technical Admin, Technical Contact, Finance, Financial Contact, Read Only
  - Required flags can be skipped when using --interactive, --json, or --json-file

### Example Usage

```sh
  megaport-cli users create --interactive
  megaport-cli users create --first-name "John" --last-name "Doe" --email "john@example.com" --position "Technical Admin"
  megaport-cli users create --json '{"firstName":"John","lastName":"Doe","email":"john@example.com","position":"Technical Admin"}'
```
### JSON Format Example
```json
{
  "firstName": "John",
  "lastName": "Doe",
  "email": "john@example.com",
  "position": "Technical Admin",
  "phone": "+61412345678"
}

```

## Usage

```sh
megaport-cli users create [flags]
```


## Parent Command

* [megaport-cli users](megaport-cli_users.md)
## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--email` |  |  | Email address of the user | true |
| `--first-name` |  |  | First name of the user | true |
| `--generate-skeleton` |  | `false` | Print a JSON skeleton template for --json or --json-file input and exit | false |
| `--interactive` | `-i` | `false` | Use interactive mode with prompts | false |
| `--json` |  |  | JSON string containing configuration | false |
| `--json-file` |  |  | Path to JSON file containing configuration | false |
| `--last-name` |  |  | Last name of the user | true |
| `--phone` |  |  | Phone number in E.164 format (optional): a leading +, digits only, no spaces, dashes or parentheses, and a real number (e.g., +61412345678) | false |
| `--position` |  |  | Position/role (Company Admin, Technical Admin, Technical Contact, Finance, Financial Contact, Read Only) | true |

