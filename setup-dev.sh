#!/bin/bash
#
# Setup script for megaport-cli development environment
# Run this after cloning the repo to configure development tools

set -e

echo "🔧 Setting up megaport-cli development environment..."

# Configure Git to use .githooks directory
echo "📌 Configuring Git hooks..."
git config core.hooksPath .githooks
echo "✓ Git hooks configured"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.26 or later."
    exit 1
fi

echo "✓ Go is installed: $(go version)"

# Check that golangci-lint v2 is installed; .golangci.yml uses the v2 format
if ! golangci-lint version 2>/dev/null | grep -Eq 'version v?2\.'; then
    echo "⚠️  golangci-lint v2 is not installed. Installing..."
    go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
    echo "✓ golangci-lint installed"
else
    echo "✓ golangci-lint v2 is installed"
fi

# Build the project
echo "🏗️  Building megaport-cli..."
go build -v
echo "✓ Build successful"

# Run tests
echo "🧪 Running tests..."
go test -v ./... > /dev/null 2>&1 || true
echo "✓ Tests completed"

# Run linter
echo "🔍 Running linter..."
golangci-lint run > /dev/null 2>&1 || true
echo "✓ Linting completed"

echo ""
echo "✅ Development environment setup complete!"
echo ""
echo "Next steps:"
echo "  - See CONTRIBUTING.md for development guidelines"
echo "  - Run: go build -v (to build)"
echo "  - Run: go test -v ./... (to test)"
echo "  - Run: golangci-lint run (to lint)"
echo ""
