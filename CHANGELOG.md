# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

<!-- Released sections below are generated from the release notes by the release
workflow (scripts/update-changelog.sh). Don't hand-edit them or add entries under
[Unreleased]; anything placed there is overwritten on the next release. -->

## [Unreleased]

## [v0.14.1] - 2026-10-09

### Other Changes
- Bump the Go toolchain to 1.26.9 and x/net to v0.60.0 (#583)

## [v1.0.0-beta.1] - 2026-07-01

### Added
- expose MCR IPsec tunnel config on vRouter interfaces (ESD-1538)
- register users, managed-account, billing-market modules
- register ix module in WASM browser build

### Fixed
- wire MCR tunnel_count to IPsec add-on and harden config parsing
- submit order under retry, poll provisioning separately (ESD-1520)
- publish stable wasm filenames and make CloudFront optional
- update wasm-publish workflow for content-hash build pipeline
- tighten JSON input value handling to match flags (ESD-1521)
- expose --asn flag on create; pass valid session-count and asn in lifecycle test
- stop spinner animating into non-TTY logs
- reject empty tag keys in buy/create JSON input (ESD-1527)
- address adversarial review findings
- align update non-interactive gate with registered flags (ESD-1550)
- treat empty partner-config flag as no update (ESD-1550)
- apply --timeout bounds each resource, not the whole run (ESD-1522)
- classify ValidationError as usage exit code (ESD-1523)
- derive changelog prev version from previous git tag
- extend nil-response guards and honor vNIC index 0 on interactive VXC (ESD-1524, ESD-1525)
- guard nil SDK responses and honor vNIC index 0 on VXC flag path (ESD-1524, ESD-1525)
- guard nil responses on service-key update and list paths (ESD-1524)
- print command errors to stderr and route status messages off stdout
- reject explicit non-positive --timeout instead of defaulting (ESD-1526)
- remove the non-functional mve restore command
- route NAT Gateway ASN through central ValidateASN

### Other Changes
- Add staging integration test for MCR IPsec tunnel VXC
- Register MCR cleanup before asserting captured UID in IPsec test
- address review findings: docs, lifetime comment, BGP password test
- model ipSecTunnelOptions as a single object per interface
- fix vxc buy dropping --resource-tags on the flags path
- clarify FinishPreRunError doc for the WASM caller
- emit JSON envelope for WASM root flag-validation errors too
- route PreRunE validation errors through finishWithError
- unit-test FinishPreRunError for codecov patch coverage
- unit-test invalid-output and max-retries PreRunE paths for patch coverage
- wire resource-tag flags into the buy flags builders
- fixup: address code review feedback on WASM CI / smoke harness
- fixup: address second-round code review feedback

## [v0.13.0] - 2026-06-23

### Added
- MCR ASN updates via `mcr update --mcr-asn`, with ASN range validation on buy and update paths
- One-command static web build for CDN hosting, with a content-hashed WASM filename for immutable caching
- WASM build now includes the `nat-gateway` module and the read-only `status`, `topology`, and `product` modules

### Changed
- `servicekeys update` now exits non-zero when the API reports the update was not applied (previously it warned and exited 0)
- Local web server hardened: binds to loopback by default, adds a documented `--bind` flag, and sets HTTP server timeouts
- Updated the Go toolchain to 1.26.4 and `golang.org/x/net` to v0.56.0
- Updated megaportgo to v1.13.1

### Removed
- `mve restore` command. It called the SDK's `RestoreProduct` (an un-cancel action), which requires a `CANCELLED` resource, but MVEs can only be deleted immediately (straight to `DECOMMISSIONED`) and the API rejects the terminate-later cancellation that would reach `CANCELLED`, so the command could never succeed.

### Fixed
- `servicekeys update` no longer resets `active` and `single-use` to false when those flags are omitted
- `servicekeys update`: passing both `--product-uid` and `--product-id` now errors instead of being sent to the SDK
- Client-side validation added for IX ASN, MAC address, VLAN, and rate limit
- Failed resource mutations now return a non-zero error instead of reporting success
- Usage errors (wrong argument count, `--json` parse failures, invalid integer arguments) now exit with code 2
- Buy and order calls are no longer retried on ambiguous errors, avoiding duplicate orders
- Password input masked in the WASM terminal prompt
- `apply` rollback now respects `--timeout` and uses a fresh context
- `list-market-codes` no longer emits blank market codes
- Credential keys redacted in `--log-http` output

### Removed
- Non-functional `--description` flag from `servicekeys update` (the update API does not support it)

## [v0.12.0] - 2026-06-11

### Added
- In-place vNIC description updates for MVE via `mve update --vnics`, with validation of vNIC entries

### Fixed
- `mve update` now honors the registered `--term` flag
- WASM: slice-valued flags (such as `--tag`) now reset correctly between invocations

## [v0.11.0] - 2026-06-05

### Added
- MCR Looking Glass commands for IP route, BGP route, and BGP session diagnostics
- `--base-url` flag to override the API base URL, with `--token-url` for authenticated login against it
- `--rollback-on-failure` flag on `apply`, with reporting of orphaned resources
- MVE `adminPassword` support for Cisco and Palo Alto vendor configs
- WASM `setAuthToken` now accepts an explicit environment override, with validation

### Changed
- Config profile commands now prompt securely for credentials (masked input, validated before prompting)
- Corrupt config files are preserved as `.corrupt-<timestamp>` instead of being overwritten, and backups are restricted to `0600`
- Updated megaportgo to v1.13.0

### Fixed
- `mcr update` now registers the `--term` flag
- MVE size labels such as "MVE 2/8" are normalized before validation
- `apply` rollback now covers provision-timeout failures
- Ports no longer experience delayed cancellation
- WASM secret prompts are routed through the JS callback in browser builds

## [v0.10.0] - 2026-05-04

### Added
- `nat-gateway buy` and `nat-gateway validate` subcommands
- `--yes` accepted as an alias for `--force` on `nat-gateway delete`

### Changed
- Output formatting unified through the output package for consistent spacing, with icon-free `PrintPlain` used for CSV section headers

### Fixed
- Closed a TOCTOU window in the tags-file size check and hardened file reads
- DNS temporary-error detection now uses a type assertion

## [v0.9.4] - 2026-04-30

### Fixed
- Homebrew tap update sets the head branch and PR base so the formula bump opens a pull request

## [v0.9.3] - 2026-04-29

### Changed
- Homebrew tap formula updates now open a pull request, authenticated with a minted app token

## [v0.9.2] - 2026-04-28

### Changed
- Updated megaportgo to v1.10.1

## [v0.9.1] - 2026-04-27

### Added
- SLSA build provenance attestation in the release workflow

## [v0.9.0] - 2026-04-24

### Added
- `--generate-skeleton` flag on buy and update commands to emit a JSON template
- `--output go-template` support for scriptable output
- `--tag` filter on `ports`, `vxc`, `mcr`, and `mve` list commands
- `--no-header` flag to suppress table and CSV column headers
- Structured JSON errors when `--output json` is active
- Pager integration and consistent "did you mean" command suggestions

### Changed
- Tag fetches parallelized with bounded concurrency

### Fixed
- VLAN help text and validation aligned across commands, with corrected inner-VLAN prompt text
- `--output` value normalized to lowercase and resolved from the executed command to handle local flag shadowing
- Output without a trailing newline preserved, and real TTY width preserved during pager buffering
- Context cancellation respected during tag fetches
- Redundant client-side name filter removed from VXC list in favor of SDK filtering

## [v0.8.0] - 2026-04-13

### Added
- `nat-gateway` command group with full CRUD, telemetry, and session listing
- `locations rtt` command for round-trip time queries
- `--log-http` global flag for API request/response debugging

### Fixed
- Spinner suppressed for CSV and XML output to prevent stream corruption
- `--log-http` redacts sensitive fields
- `x-app: cli` header sent on all API requests
- `locations rtt` defaults to the previous month, since data publishes after month end
- Auto-renew prompts accept `y`/`yes`/`n`/`no`

## [v0.7.1] - 2026-04-10

### Added
- `auth status` and `auth whoami` commands
- MCR IPSec add-on support

### Fixed
- IPSec tunnel count: consistent 0/omit semantics across `buy` and `add-ipsec-addon`, applied in interactive mode, with clearer help text

## [v0.7.0] - 2026-04-09

### Changed
- CSV and XML output now share deduplicated reflection logic

### Fixed
- `--watch`: `--interval` validated to prevent a ticker panic, with timeout handling added and duplicate error output on timeout removed
- CSV header fallback no longer includes JSON struct-tag options
- Login error messages no longer double-wrapped, and VXC input validation restored
- WASM proxy server applies security headers on all routes and sanitizes auth error messages

## [v0.6.0] - 2026-04-05

### Added
- `apply` command for bulk provisioning from a config file
- `topology` command showing a resource relationship tree
- `product` command group (`list`, `get-type`)
- `--query` flag with JMESPath support for filtering JSON output
- `--fields` flag for selecting output columns
- `--limit` flag on all list commands
- `--watch` flag for continuous status monitoring
- `--export` flag on get commands to emit recreatable JSON config
- Global `--timeout` flag for configurable request timeouts
- Shorthand command aliases (`ls`, `rm`, `show`, `st`)
- Version-update check when running `megaport version`
- Man page generation in the `generate-docs` command
- Automatic retry with exponential backoff for transient API failures
- Color-coded resource status values in table output
- Helpful messages when list commands return no results
- New `Cancelled` exit code returned when the user aborts an operation
- Elapsed time shown on the provisioning spinner
- Homebrew tap auto-update on release

### Changed
- Locations API commands no longer require authentication

### Fixed
- Race conditions, deadlocks, and panics resolved in the output package
- WASM proxy server hardened with a hostname allowlist, restricted CORS origins, and response bodies no longer logged
- `--limit` now rejects negative values
- Provisioning operations now use a 15-minute default timeout

## [v0.5.5] — 2026-04-02

### Added
- Docker image for the native CLI binary, published to GitHub Container Registry
- Aggregate status dashboard (`megaport-cli status`) showing all resources in one view
- Improved API error messages for 401, 403, 404, and 429 HTTP responses

### Fixed
- GoReleaser configuration updated for v1.26.2 compatibility (separate Docker build target, `builds` field syntax)

## [v0.5.4] — 2026-04-01

### Added
- `users` module: list, get, create, update, and delete users
- `--profile` global flag for per-command credential profile selection
- `validate` / dry-run subcommands for all resource types (port, lag, vxc, mcr, mve, ix)
- Confirmation prompt before buy operations
- `--no-wait` flag on all buy commands for non-blocking provisioning
- `lock` and `unlock` commands for MCR and MVE; `restore` for MVE
- VXC list command with `--status` and `--name-contains` filter flags
- Auto-disable colors and spinners when output is piped
- Test coverage reporting added to CI workflow

### Fixed
- VXC mock field names standardized to PascalCase for consistency

## [v0.5.3] — 2026-03-31

### Added
- `--quiet` / `-q` and `--verbose` / `-v` global flags

### Fixed
- Locations commands migrated to native LocationV3 structs
- Duplicate login and API calls removed from MCR prefix filter list updates
- Misleading AWS connect type validation error message corrected
- Duplicate flag definitions removed from partners list command
- Dead code and redundant branches removed

## [v0.5.2] — 2026-03-11

### Added
- XML output format via `--output xml`
- `list-countries`, `list-market-codes`, and `search` subcommands for `locations`
- Distinct exit codes: 2 = usage/validation, 3 = authentication, 4 = API error

### Fixed
- Date range validation for service key creation
- Output header tag casing standardized to Title Case across all resources
- `vxc update` now returns an error when called without flags or `--interactive`
- MCR delete now uses a confirm prompt for consistency

## [v0.5.1] — 2026-03-10

### Added
- Billing markets commands (`list-billing-markets`, `set-billing-market`)
- Token authentication support (`--auth-token`) for portal session tokens
- Full SDK flag parity: `--safe-delete`, `--is-approved`, `--vnic-index`, service key and location filters
- Thread-safe output format handling via atomic value

### Fixed
- Context timeout handling improvements across VXC, MCR, MVE, and port commands
- JSON output mode: spinner and progress messages routed to stderr for clean `jq` piping
- Extensive correctness fixes following code review

## [v0.5.0] — 2026-03-06

### Added
- `managed-account` commands: list, get, create, update, delete
- `ix` (Internet Exchange) commands: list, get, buy, update, delete with filtering
- VXC list command with client-side filtering

### Fixed
- VXC VLAN validation: correct upper bound (4094), support for untagged VLANs
- Azure partner port lookup in `getPartnerPortUID`
- Data race on output format global replaced with atomic value

## [v0.4.9] — 2026-02-26

### Added
- WebAssembly (WASM) browser version with session-based authentication
- Vue 3 + xterm.js frontend for the browser terminal
- Auto-environment detection for WASM builds based on hostname
- `setAuthToken` command for portal session authentication
- WASM support for MCR, MVE, and VXC commands

### Fixed
- Config file permissions hardened to 0600; recovery on file corruption
- Access keys masked in `config view` and `list-profiles` output
- Empty buy UID guard and nil status return handling

## [v0.4.8] — 2025-08-04

### Fixed
- Locations commands migrated from deprecated v2 API to v3
- Progress messages in JSON output mode now go to stderr (enables clean `jq` piping)
- GoReleaser `ldflags` updated to inject version into the binary at build time

## [v0.4.7] — 2025-07-31

### Fixed
- Environment variable credentials now take priority when `--env` is explicitly set

## [v0.4.6] — 2025-07-30

### Fixed
- MCR list command flag name inconsistency resolved

## [v0.4.5] — 2025-07-28

### Fixed
- GoReleaser `ldflags` version injection
- IBM name validation logic
- Updated megaportgo from v1.3.5 to v1.3.9

## [v0.4.4] — 2025-04-30

### Added
- `get-status` subcommand for port, MVE, MCR, and VXC resources

### Fixed
- Untagged VLAN now accepted in VXC flag validation
- JSON parsing in VXC input handling optimized

## [v0.4.3] — 2025-04-17

### Added
- Enhanced validation for VXC and MVE: VLAN ranges, IP/CIDR, partner config, untagged VLAN

### Fixed
- VXC type checking and validation corrections
- MCR and MVE validation consolidated into the `validation` package

## [v0.4.2] — 2025-04-15

### Added
- Resource tag support for VXC, port, and MCR (list and update subcommands)
- Support for higher-speed MCR configurations

## [v0.4.1] — 2025-04-14

### Added
- Colorized table output with Megaport branding for improved readability

## [v0.4.0] — 2025-04-11

### Changed
- Enhanced CLI prompting and feedback experience across interactive flows

## [v0.3.9] — 2025-04-10

### Added
- `mve list` command
- `mcr list` command
- `--show-inactive` flag for `ports list`

## [v0.3.8] — 2025-04-10

### Added
- Configuration management: named profiles, `config set`, `config get`, `list-profiles`, `import`, `export`

### Fixed
- golangci-lint CI configuration stabilized
- Table formatting updated for go-pretty library compatibility

## [v0.3.7] — 2025-04-09

### Added
- Table output now uses go-pretty with Megaport brand colors; `--no-color` renders plain ASCII tables

### Fixed
- Partners listing bug when filter flags were combined
- `golang.org/x/net` security vulnerability patched

## [v0.3.6] — 2025-04-09

### Added
- `docs` subcommand renders Markdown CLI reference in the terminal using Glamour

## [v0.3.1] – [v0.3.5] — 2025-04-05 to 2025-04-08

### Added
- Command builder pattern: all commands refactored to use `cmdbuilder.NewCommand()` fluent API
- Port, MCR, and MVE integration tests
- Improved output for update change display

### Fixed
- VXC flag sets for create and update
- MCR prefix filter list commands and buy interactive mode

## [v0.3.0] — 2025-04-04

### Added
- `docs generate` command for auto-generating CLI reference documentation
- Interactive mode for partner port selection in VXC buy flow
- VXC: enhanced buy and update with Azure, GCP, Oracle partner port lookup
- `--version` output based on git tag
- Shell `completion` command

### Changed
- CLI reorganized into modular command packages (`internal/commands/<resource>/`)

## [v0.2.0] – [v0.2.9] — 2025-03-05 to 2025-04-03

### Added
- Color output across all resource types with `--no-color` flag
- VXC full support: buy, update, list with interactive, flag, and JSON input modes
- Partner port selection for AWS, Azure, GCP, Oracle, and IBM VXC types
- MVE images and sizes listing with filtering
- MCR prefix filter list management commands
- `--env` flag for environment selection (`production`, `staging`, `development`)

### Fixed
- VXC partner port and VLAN prompt handling
- Documentation improvements across all command groups

## [v0.1.0] – [v0.1.9] — 2025-02-21 to 2025-03-04

### Added
- Initial release with ports, VXCs, MCRs, MVEs, partner ports, locations, and service keys
- Port lifecycle management: buy, update, delete, list, get, and LAG port support
- MCR lifecycle management: buy, update, delete, and prefix filter list creation
- Interactive, flag-based, and JSON input modes for buy and update commands
- CSV, JSON, and table output formats via `--output` flag
- Shell completion support
- Credential support via environment variables (`MEGAPORT_ACCESS_KEY`, `MEGAPORT_SECRET_KEY`, `MEGAPORT_ENVIRONMENT`)
- Automated CI with golangci-lint and Go test suite

[Unreleased]: https://github.com/megaport/megaport-cli/compare/v0.14.1...HEAD
[v0.14.1]: https://github.com/megaport/megaport-cli/compare/v0.14.0...v0.14.1
[v1.0.0-beta.1]: https://github.com/megaport/megaport-cli/compare/v0.13.0...v1.0.0-beta.1
[v0.13.0]: https://github.com/megaport/megaport-cli/compare/v0.12.0...v0.13.0
[v0.12.0]: https://github.com/megaport/megaport-cli/compare/v0.11.0...v0.12.0
[v0.11.0]: https://github.com/megaport/megaport-cli/compare/v0.10.0...v0.11.0
[v0.10.0]: https://github.com/megaport/megaport-cli/compare/v0.9.4...v0.10.0
[v0.9.4]: https://github.com/megaport/megaport-cli/compare/v0.9.3...v0.9.4
[v0.9.3]: https://github.com/megaport/megaport-cli/compare/v0.9.2...v0.9.3
[v0.9.2]: https://github.com/megaport/megaport-cli/compare/v0.9.1...v0.9.2
[v0.9.1]: https://github.com/megaport/megaport-cli/compare/v0.9.0...v0.9.1
[v0.9.0]: https://github.com/megaport/megaport-cli/compare/v0.8.0...v0.9.0
[v0.8.0]: https://github.com/megaport/megaport-cli/compare/v0.7.1...v0.8.0
[v0.7.1]: https://github.com/megaport/megaport-cli/compare/v0.7.0...v0.7.1
[v0.7.0]: https://github.com/megaport/megaport-cli/compare/v0.6.0...v0.7.0
[v0.6.0]: https://github.com/megaport/megaport-cli/compare/v0.5.5...v0.6.0
[v0.5.5]: https://github.com/megaport/megaport-cli/compare/v0.5.4...v0.5.5
[v0.5.4]: https://github.com/megaport/megaport-cli/compare/v0.5.3...v0.5.4
[v0.5.3]: https://github.com/megaport/megaport-cli/compare/v0.5.2...v0.5.3
[v0.5.2]: https://github.com/megaport/megaport-cli/compare/v0.5.1...v0.5.2
[v0.5.1]: https://github.com/megaport/megaport-cli/compare/v0.5.0...v0.5.1
[v0.5.0]: https://github.com/megaport/megaport-cli/compare/v0.4.9...v0.5.0
[v0.4.9]: https://github.com/megaport/megaport-cli/compare/v0.4.8...v0.4.9
[v0.4.8]: https://github.com/megaport/megaport-cli/compare/v0.4.7...v0.4.8
[v0.4.7]: https://github.com/megaport/megaport-cli/compare/v0.4.6...v0.4.7
[v0.4.6]: https://github.com/megaport/megaport-cli/compare/v0.4.5...v0.4.6
[v0.4.5]: https://github.com/megaport/megaport-cli/compare/v0.4.4...v0.4.5
[v0.4.4]: https://github.com/megaport/megaport-cli/compare/v0.4.3...v0.4.4
[v0.4.3]: https://github.com/megaport/megaport-cli/compare/v0.4.2...v0.4.3
[v0.4.2]: https://github.com/megaport/megaport-cli/compare/v0.4.1...v0.4.2
[v0.4.1]: https://github.com/megaport/megaport-cli/compare/v0.4.0...v0.4.1
[v0.4.0]: https://github.com/megaport/megaport-cli/compare/v0.3.9...v0.4.0
[v0.3.9]: https://github.com/megaport/megaport-cli/compare/v0.3.8...v0.3.9
[v0.3.8]: https://github.com/megaport/megaport-cli/compare/v0.3.7...v0.3.8
[v0.3.7]: https://github.com/megaport/megaport-cli/compare/v0.3.6...v0.3.7
[v0.3.6]: https://github.com/megaport/megaport-cli/compare/v0.3.5...v0.3.6
[v0.3.5]: https://github.com/megaport/megaport-cli/compare/v0.3.4...v0.3.5
[v0.3.4]: https://github.com/megaport/megaport-cli/compare/v0.3.3...v0.3.4
[v0.3.3]: https://github.com/megaport/megaport-cli/compare/v0.3.2...v0.3.3
[v0.3.2]: https://github.com/megaport/megaport-cli/compare/v0.3.1...v0.3.2
[v0.3.1]: https://github.com/megaport/megaport-cli/compare/v0.3.0...v0.3.1
[v0.3.0]: https://github.com/megaport/megaport-cli/compare/v0.2.9...v0.3.0
[v0.2.9]: https://github.com/megaport/megaport-cli/compare/v0.2.8...v0.2.9
[v0.2.8]: https://github.com/megaport/megaport-cli/compare/v0.2.7...v0.2.8
[v0.2.7]: https://github.com/megaport/megaport-cli/compare/v0.2.6...v0.2.7
[v0.2.6]: https://github.com/megaport/megaport-cli/compare/v0.2.5...v0.2.6
[v0.2.5]: https://github.com/megaport/megaport-cli/compare/v0.2.4...v0.2.5
[v0.2.4]: https://github.com/megaport/megaport-cli/compare/v0.2.3...v0.2.4
[v0.2.3]: https://github.com/megaport/megaport-cli/compare/v0.2.2...v0.2.3
[v0.2.2]: https://github.com/megaport/megaport-cli/compare/v0.2.1...v0.2.2
[v0.2.1]: https://github.com/megaport/megaport-cli/compare/v0.2.0...v0.2.1
[v0.2.0]: https://github.com/megaport/megaport-cli/compare/v0.1.9...v0.2.0
[v0.1.9]: https://github.com/megaport/megaport-cli/compare/v0.1.8...v0.1.9
[v0.1.8]: https://github.com/megaport/megaport-cli/compare/v0.1.7...v0.1.8
[v0.1.7]: https://github.com/megaport/megaport-cli/compare/v0.1.6...v0.1.7
[v0.1.6]: https://github.com/megaport/megaport-cli/compare/v0.1.5...v0.1.6
[v0.1.5]: https://github.com/megaport/megaport-cli/compare/v0.1.4...v0.1.5
[v0.1.4]: https://github.com/megaport/megaport-cli/compare/v0.1.3...v0.1.4
[v0.1.3]: https://github.com/megaport/megaport-cli/compare/v0.1.2...v0.1.3
[v0.1.2]: https://github.com/megaport/megaport-cli/compare/v0.1.1...v0.1.2
[v0.1.1]: https://github.com/megaport/megaport-cli/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/megaport/megaport-cli/releases/tag/v0.1.0
