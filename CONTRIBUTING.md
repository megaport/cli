# Contributing to megaport-cli

Thank you for your interest in contributing to megaport-cli! This guide will help you get started.

## Development Setup

### Prerequisites

- Go 1.25 or later (matches `go.mod`)
- Git

### Initial Setup

1. Clone the repository:

```bash
git clone https://github.com/megaport/megaport-cli.git
cd megaport-cli
```

2. **Enable Git hooks for automatic documentation generation:**

```bash
git config core.hooksPath .githooks
```

This ensures that whenever you commit changes to command files, the documentation is automatically regenerated and included in your commit.

## Making Changes

### Building the Project

```bash
go build -v
```

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run tests for a specific package
go test -v ./internal/commands/ports

# Run a specific test
go test -v ./internal/commands/ports -run TestFilterPorts
```

### Linting

We use `golangci-lint` for code quality:

```bash
golangci-lint run
```

### Code Formatting

```bash
gofmt -w .
```

## Documentation

Documentation is auto-generated from command definitions. When you add or modify commands:

1. The pre-commit hook will automatically regenerate docs when you commit command changes
2. Or manually regenerate docs with:

```bash
./megaport-cli generate-docs ./docs
```

The generated markdown files include:

- Command descriptions and usage
- Flags and options
- **Command aliases** (e.g., `list → ls`, `get → show`, `delete → rm`)
- Examples

## Commit Messages

Every commit subject starts with a Jira key, then a colon and a short summary:

- `ENG-1234: add shorthand aliases for common commands`
- `ENG-1250: correct port deletion error handling`

GitHub enforces this on every branch and on `main`. If you're contributing from a fork you don't need a key: write a clear subject and a maintainer will squash-merge your pull request under one. Dependabot commits use the standing key `EIP-3148`.

## Pull Requests

1. Create a feature branch named after the Jira key: `git checkout -b feature/ENG-1234-your-feature` (`fix/`, `hotfix/`, and `release/` prefixes also pass the branch rule). This only applies to branches in this repository: contributing from a fork, name your branch however you like.
2. Make your changes
3. Ensure tests pass: `go test -v ./...`
4. Ensure linting passes: `golangci-lint run`
5. Commit your changes (docs will auto-regenerate if enabled)
6. Push and open a PR against `main`

## Command Aliases

We support convenient shorthand aliases for frequently-used commands:

| Full Command | Alias | Example                      |
| ------------ | ----- | ---------------------------- |
| list         | ls    | `megaport-cli ports ls`      |
| get          | show  | `megaport-cli vxc show <id>` |
| delete       | rm    | `megaport-cli mcr rm <id>`   |
| status       | st    | `megaport-cli st`            |

When adding new commands, consider adding aliases if appropriate.

## Architecture

See [CLAUDE.md](./CLAUDE.md) for detailed architecture documentation, including:

- Module registry pattern
- Command builder pattern
- Three input modes (interactive, CLI flags, JSON)
- Testing conventions

## Questions?

Feel free to open an issue or discussion on GitHub!
