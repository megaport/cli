#!/bin/bash

set -e

echo "🧪 Megaport CLI WASM Test Runner"
echo "================================"
echo ""

# Build the test binary
echo "📦 Building WASM test binary..."
cd "$(dirname "$0")"
GOOS=js GOARCH=wasm go test -c -o wasm.test .

if [ ! -f "wasm.test" ]; then
    echo "❌ Failed to build test binary"
    exit 1
fi

echo "✅ Test binary built: wasm.test ($(du -h wasm.test | cut -f1))"
echo ""

# Copy wasm_exec.js on every run so it matches the toolchain that built wasm.test
GOROOT=$(go env GOROOT)
cp -f "$GOROOT/lib/wasm/wasm_exec.js" ../../web/wasm_exec.js
echo "✅ Copied wasm_exec.js from the Go installation"

echo "🌐 Starting test server on http://localhost:8765"
echo ""
echo "   Open http://localhost:8765/internal/wasm/test-runner.html in your browser"
echo ""
echo "Press Ctrl+C to stop the server"
echo ""

# Start a simple HTTP server from the project root
cd ../..
python3 -m http.server --bind 127.0.0.1 8765
