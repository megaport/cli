#!/usr/bin/env bash
# Build the browser/WASM CLI's static assets (wasm + wasm_exec.js) into web/dist/.
# No Docker and no Go server. Production publishing goes through the wasm-publish
# workflow (brotli pre-compression + pinned headers), not a plain aws s3 sync; see
# the note printed at the end.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

# shellcheck source-path=SCRIPTDIR
# shellcheck source=lib/web-assets.sh
source "$repo_root/scripts/lib/web-assets.sh"

build_static_assets

echo ""
echo "Static assets ready: web/dist/"
echo "To publish, use the wasm-publish workflow (or the manual steps in WASM_README.md):"
echo "it brotli-compresses the wasm and pins Content-Type/Content-Encoding on upload."
echo "A plain 'aws s3 sync' would serve the wasm uncompressed and without those headers."
