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

# Check if wasm_exec.js exists
if [ ! -f "../../web/wasm_exec.js" ]; then
    echo "⚠️  wasm_exec.js not found in ../../web/"
    echo "   Copying from Go installation..."
    GOROOT=$(go env GOROOT)
    cp "$GOROOT/lib/wasm/wasm_exec.js" ../../web/wasm_exec.js
    echo "✅ Copied wasm_exec.js"
fi

echo "🌐 Starting test server on http://localhost:8765"
echo ""
echo "   Open http://localhost:8765/internal/wasm/test-runner.html in your browser"
echo ""
echo "Press Ctrl+C to stop the server"
echo ""

# Start a simple HTTP server from the project root
cd ../..
python3 -m http.server 8765
