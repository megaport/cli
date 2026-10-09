# WASM Test Runner

This directory contains tools for running Go WASM tests in a browser environment.

## Quick Start

```bash
./run-tests.sh
```

This will:

1. Build the WASM test binary (`wasm.test`)
2. Copy `wasm_exec.js` from your Go installation to `web/`
3. Start an HTTP server on port 8765
4. Open your browser to: http://localhost:8765/internal/wasm/test-runner.html

## Manual Steps

If you prefer to run steps manually:

### 1. Build the test binary

```bash
GOOS=js GOARCH=wasm go test -c -o wasm.test .
```

### 2. Copy the WASM loader

```bash
cp -f "$(go env GOROOT)/lib/wasm/wasm_exec.js" ../../web/wasm_exec.js
```

### 3. Start HTTP server

```bash
cd ../.. # Go to project root
python3 -m http.server --bind 127.0.0.1 8765
```

### 4. Open in browser

Navigate to: http://localhost:8765/internal/wasm/test-runner.html

## Test Output

The test runner will display:

- ✅ Pass/fail status for each test
- 📊 Test coverage information
- 🐛 Detailed error messages for failures
- ⏱️ Execution timing

## New Token Authentication Tests

The test suite includes comprehensive tests for the `setAuthToken` functionality:

- `TestSetAuthToken` - Token-based authentication with various environments
- `TestAuthMethodPriority` - Verifies token auth takes precedence over API key auth
- `TestSetAuthTokenMasking` - Validates token preview masking

## Troubleshooting

**Tests don't run:**

- Ensure `wasm_exec.js` exists in `../../web/` directory
- Check browser console for JavaScript errors
- Verify WASM is supported in your browser

**Build fails:**

- Ensure you have Go 1.26+ installed
- Verify GOOS=js GOARCH=wasm environment variables

**Server won't start:**

- Check if port 8765 is already in use
- Try a different port: `python3 -m http.server --bind 127.0.0.1 8766`
