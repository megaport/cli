# Copilot Instructions for megaport-cli

## Build, Test, and Lint

```bash
go build -v                                              # Build
go test -v ./...                                         # All tests
go test -v ./internal/commands/ports                     # Single package
go test -v ./internal/commands/ports -run TestFilterPorts # Single test
golangci-lint run                                        # Lint (.golangci.yml)
gofmt -w .                                               # Format
GOOS=js GOARCH=wasm go build -tags js,wasm -o web/megaport.wasm .  # WASM build
```

`make check` runs lint + test together. Use before committing.

## Architecture

Go CLI for managing Megaport network infrastructure (ports, VXCs, MCRs, MVEs). Dual build targets: native CLI and WebAssembly (browser).

### Entry Points

- `main.go` → calls `megaport.Execute()` (native CLI)
- `main_wasm.go` → browser WASM entry point
- `cmd/megaport/megaport_common.go` → root command setup, `registerModules()` connects all command groups

### Module Registry Pattern

All command groups implement `registry.Module` (`Name()` + `RegisterCommands()`). Modules are registered in `cmd/megaport/megaport_common.go:registerModules()` and connected to the root cobra command via `moduleRegistry.RegisterAll()`.

### Command Builder Pattern

Commands use a fluent builder API from `internal/base/cmdbuilder/builder.go`:

```go
cmdbuilder.NewCommand("buy", "Buy a port").
    WithColorAwareRunFunc(BuyPort).
    WithInteractiveFlag().
    WithPortCreationFlags().
    WithJSONConfigFlags().
    Build()
```

Flag sets for each resource type live in separate files (e.g., `port_flagsets.go`, `vxc_flagsets.go`).

### Three Input Modes

All "buy" and "update" commands support:
1. Interactive prompts (`--interactive`)
2. CLI flags (`--name "..." --term 12`)
3. JSON input (`--json '{...}'` or `--json-file ./config.json`)

Use `WithConditionalRequirements()` to enforce required flags only when not using interactive/JSON mode.

### Per-Resource Package Structure

Each resource in `internal/commands/<resource>/` follows this file layout:
- `<resource>.go` — Command definitions (builder calls)
- `<resource>_actions.go` — Business logic (API calls, input processing)
- `<resource>_module.go` — Module interface implementation
- `<resource>_mock.go` — Mock service structs for testing
- `<resource>_test.go` — Tests
- `<resource>_prompts.go` — Interactive prompt logic
- `<resource>_inputs.go` — Input parsing/validation
- `<resource>_output.go` — Output formatting

### Build Tags

Platform-specific code uses build tags:
- Native: `//go:build !js || !wasm`
- WASM: `//go:build js && wasm`

Files with `_wasm` suffix contain browser-specific implementations. When adding platform-specific behavior, create paired files with appropriate build tags.

## Key Conventions

### Testing

- `testify/assert` for assertions; table-driven tests are standard
- Mock services are hand-written structs (not generated) — each package defines its own (e.g., `MockPortService` in `ports_mock.go`)
- Tests override `config.LoginFunc` to inject mock clients, restoring the original in `defer`

### Authentication

API auth goes through a package-level `config.LoginFunc` variable. Tests replace this to inject mocks.

### Output

All commands displaying data use the `output` package. Support `--output` flag with table (default), json, csv, xml formats. Use `output.PrintInfo`, `output.PrintError`, `output.PrintSuccess` for user messages.

### Documentation

Docs are auto-generated from command definitions. Regenerate with `./megaport-cli generate-docs ./docs`. A pre-commit hook handles this automatically if you run `git config core.hooksPath .githooks`.

### Commit Messages

Every commit subject starts with a Jira key: `ENG-1234: add shorthand aliases for common commands`. See CONTRIBUTING.md.

### Command Aliases

Standard aliases: `list→ls`, `get→show`, `delete→rm`, `status→st`. Add aliases for new commands when appropriate.
