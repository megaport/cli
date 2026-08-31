# megaport-cli

A CLI tool to interact with the Megaport API

## Description

Megaport CLI provides a command line interface to interact with the Megaport API.

The CLI allows you to manage Megaport resources such as ports, VXCs, MCRs, MVEs, service keys, and more.

### Optional Fields
  - `--env`: Environment to use (production, staging, development)
  - `--help`: Show help for any command
  - `--max-retries`: Maximum number of retries for transient API failures (default 3)
  - `--no-color`: Disable colored output
  - `--no-header`: Suppress table and CSV column headers (useful for scripting)
  - `--no-pager`: Disable pager for long table output
  - `--no-retry`: Disable automatic retry on transient API failures
  - `--output`: Output format (table, json, csv, xml, go-template)
  - `--quiet`: Suppress informational output, only show errors and data
  - `--template`: Go template string for --output go-template
  - `--verbose`: Show additional debug information

### Important Notes
  - Use the --help flag with any command to see specific usage information
  - Authentication is handled via the MEGAPORT_ACCESS_KEY and MEGAPORT_SECRET_KEY environment variables
  - By default, the CLI connects to the Megaport production environment
  - Set the MEGAPORT_ENVIRONMENT environment variable to connect to a different environment

### Example Usage

```sh
  megaport-cli ports list
  megaport-cli vxc buy --interactive
  megaport-cli mcr get [mcrUID]
  megaport-cli locations list
```

## Usage

```sh
megaport-cli [flags]
```


## Flags

| Name | Shorthand | Default | Description | Required |
|------|-----------|---------|-------------|----------|
| `--base-url` |  |  | Override the API base URL (e.g. http://localhost:8080); takes precedence over --env and any profile environment | false |
| `--env` |  |  | Environment to use (prod, dev, or staging) | false |
| `--fields` |  |  | Comma-separated list of fields to include in output (e.g., uid,name,status); use an unknown name to list available fields | false |
| `--log-http` |  | `false` | Log raw HTTP requests/responses to stderr for debugging (may include sensitive data such as auth tokens) | false |
| `--max-retries` |  | `3` | Maximum number of retries for transient API failures | false |
| `--no-color` |  | `false` | Disable colorful output | false |
| `--no-header` |  | `false` | Suppress table and CSV column headers (useful for scripting) | false |
| `--no-pager` |  | `false` | Disable pager for long table output | false |
| `--no-retry` |  | `false` | Disable automatic retry on transient API failures | false |
| `--on-behalf-of` |  |  | Act on behalf of a managed account: company UID sent as the X-Call-Context header on every authenticated request (falls back to MEGAPORT_MANAGED_ACCOUNT_UID) | false |
| `--output` | `-o` | `table` | Output format (table, json, csv, xml, go-template; requires --template when using go-template) | false |
| `--profile` |  |  | Use a specific config profile for this command | false |
| `--query` |  |  | JMESPath query to filter JSON output (requires --output json) | false |
| `--quiet` | `-q` | `false` | Suppress informational output, only show errors and data | false |
| `--template` |  |  | Go template string for --output go-template (e.g. '{{range .}}{{.Name}}{{"\n"}}{{end}}') | false |
| `--timeout` |  | `0s` | Timeout for the operation (e.g., 30s, 2m, 5m); must be positive. Omit to use each command's built-in default (see the command's own help) | false |
| `--token-url` |  |  | Override the OAuth token endpoint (typically used with --base-url when auth is served from a non-standard host) | false |
| `--verbose` | `-v` | `false` | Show additional debug information | false |

## Subcommands
* [apply](megaport-cli_apply.md)
* [auth](megaport-cli_auth.md)
* [billing-market](megaport-cli_billing-market.md)
* [completion](megaport-cli_completion.md)
* [config](megaport-cli_config.md)
* [generate-docs](megaport-cli_generate-docs.md)
* [ix](megaport-cli_ix.md)
* [locations](megaport-cli_locations.md)
* [managed-account](megaport-cli_managed-account.md)
* [mcr](megaport-cli_mcr.md)
* [mve](megaport-cli_mve.md)
* [nat-gateway](megaport-cli_nat-gateway.md)
* [partners](megaport-cli_partners.md)
* [ports](megaport-cli_ports.md)
* [product](megaport-cli_product.md)
* [servicekeys](megaport-cli_servicekeys.md)
* [status](megaport-cli_status.md)
* [topology](megaport-cli_topology.md)
* [users](megaport-cli_users.md)
* [version](megaport-cli_version.md)
* [vxc](megaport-cli_vxc.md)
* [whoami](megaport-cli_whoami.md)

