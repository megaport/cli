# shellcheck shell=bash
# Shared build step for the browser/WASM CLI. Source this and call
# build_static_assets from the repo root; build-web.sh wraps it to produce the
# CDN-publishable output.

build_static_assets() {
  local publish_dir="web/dist"

  if ! command -v go >/dev/null 2>&1; then
    echo "error: 'go' is required but was not found on PATH" >&2
    return 1
  fi

  # Wipe any stale output (e.g. .br/.gz from a prior run) so the published
  # directory only ever contains this build's artifacts.
  rm -rf "$publish_dir" || return 1
  mkdir -p "$publish_dir" || return 1

  echo "==> Building WASM binary ($publish_dir/megaport.wasm)"
  # GOWORK=off so the build uses the module's pinned deps, not the go.work workspace.
  # -trimpath and -ldflags="-s -w" match the Makefile wasm target: strip debug info
  # and build-machine paths from the published binary.
  GOWORK=off GOOS=js GOARCH=wasm go build -trimpath -tags js,wasm -ldflags="-s -w" -o "$publish_dir/megaport.wasm" . || return 1

  echo "==> Copying wasm_exec.js"
  local goroot
  if ! goroot="$(go env GOROOT)" || [ -z "$goroot" ]; then
    echo "error: could not determine GOROOT via 'go env GOROOT'" >&2
    return 1
  fi
  if [ -f "$goroot/lib/wasm/wasm_exec.js" ]; then
    cp "$goroot/lib/wasm/wasm_exec.js" "$publish_dir/" || return 1
  elif [ -f "$goroot/misc/wasm/wasm_exec.js" ]; then
    cp "$goroot/misc/wasm/wasm_exec.js" "$publish_dir/" || return 1
  else
    echo "error: wasm_exec.js not found under $goroot" >&2
    return 1
  fi
}
