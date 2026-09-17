#!/bin/bash
# Build Megaport CLI for WASM

# Ensure we're in the project root
cd "$(dirname "$0")"

echo "Building Megaport CLI WASM version..."

# Create web directory if it doesn't exist
mkdir -p web

# Build the WASM binary
GOOS=js GOARCH=wasm go build -o web/megaport.wasm ./main_wasm.go

# Find and copy wasm_exec.js
GO_ROOT=$(go env GOROOT)

# Try multiple known locations (lib/wasm is the Go 1.21+ location)
POSSIBLE_PATHS=(
  "$GO_ROOT/lib/wasm/wasm_exec.js"
  "$GO_ROOT/misc/wasm/wasm_exec.js"
  "$GO_ROOT/js/wasm/wasm_exec.js"
  "$GO_ROOT/src/syscall/js/wasm_exec.js"
)

# Check if wasm_exec.js exists in known locations
FOUND=false
for path in "${POSSIBLE_PATHS[@]}"; do
  if [ -f "$path" ]; then
    cp "$path" ./web/
    echo "Copied wasm_exec.js from $path"
    FOUND=true
    break
  fi
done

# If not found in known locations, try a comprehensive search
if [ "$FOUND" = false ]; then
  echo "Searching for wasm_exec.js in Go installation..."
  FOUND_PATH=$(find "$GO_ROOT" -name "wasm_exec.js" -type f | head -n 1)
  
  if [ -n "$FOUND_PATH" ]; then
    cp "$FOUND_PATH" ./web/
    echo "Copied wasm_exec.js from $FOUND_PATH"
    FOUND=true
  fi
fi

# Download from GitHub as a last resort
if [ "$FOUND" = false ]; then
  echo "Could not find wasm_exec.js locally. Downloading from GitHub..."
  GO_VERSION=$(go version | cut -d " " -f 3 | sed 's/go//')
  MAJOR_MINOR=$(echo $GO_VERSION | cut -d "." -f 1-2)
  
  # URL for wasm_exec.js based on Go version
  WASM_EXEC_URL="https://raw.githubusercontent.com/golang/go/release-branch.go$MAJOR_MINOR/misc/wasm/wasm_exec.js"
  
  # Download the file
  if curl -s -o ./web/wasm_exec.js "$WASM_EXEC_URL"; then
    echo "Successfully downloaded wasm_exec.js for Go $GO_VERSION"
    FOUND=true
  else
    echo "Failed to download wasm_exec.js from primary location."
    
    # Try alternate location (master branch)
    ALT_URL="https://raw.githubusercontent.com/golang/go/master/misc/wasm/wasm_exec.js"
    if curl -s -o ./web/wasm_exec.js "$ALT_URL"; then
      echo "Successfully downloaded wasm_exec.js from master branch"
      FOUND=true
    else
      echo "ERROR: Could not obtain wasm_exec.js"
    fi
  fi
fi

if [ "$FOUND" = true ]; then
  echo "Build complete. Files in ./web/ directory."
else
  echo "Build incomplete. Missing wasm_exec.js"
  exit 1
fi