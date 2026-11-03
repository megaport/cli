# Megaport CLI WASM build artifacts

This directory holds the WebAssembly build output for the Megaport CLI. The portal
embeds the `.wasm` in its own front-end and runs it in-browser; this repo has no
reference front-end of its own. There is no server-side component: the assets are
served as static files by a CDN, and the WASM authenticates against the Megaport API
directly from the browser.

## Building

```bash
# WASM binary (writes web/megaport.wasm)
make wasm

# Assemble the WASM binary + wasm_exec.js loader into web/dist/
./scripts/build-web.sh
```

## Pre-compressed WASM artifacts

`megaport.wasm` is ~18.5 MiB raw, above CloudFront's 10 MB auto-compression cap, so the
build pre-compresses it at the origin. `cmd/wasmcompress` writes two sibling objects
next to the wasm:

- `megaport.wasm.br`: brotli, quality 11 (~2.95 MiB; preferred)
- `megaport.wasm.gz`: gzip -9 (~4.4 MiB; fallback)

`make wasm-compress` runs `cmd/wasmcompress` after building the wasm. The publish workflow
uploads only the brotli object, as `megaport.wasm` with `Content-Type: application/wasm`
and `Content-Encoding: br`. It does not upload the gzip or raw objects.

## Publishing to S3 (CDN)

`.github/workflows/wasm-publish.yaml` builds `web/dist/` and publishes it to a
CloudFront-fronted S3 bucket. It is manual-only (`workflow_dispatch`): a production
publish is always a deliberate action, never an automatic tag push.

Each run uploads `megaport.wasm`, `wasm_exec.js`, and `integrity.json` (their SHA-256 and
SRI hashes) under these prefixes:

- `<version>/`: immutable, long-lived cache. Treat a version label as write-once, so use
  a fresh tag or `version` input per release.
- `latest/`: short TTL, refreshed on every tag run unless the `publish_latest` input is
  false. A branch run never moves it.

These filenames are kept stable rather than content-hashed, so a consumer such as the
portal can load `megaport.wasm` and `wasm_exec.js` by static config URL. The version
prefix is the cache-buster instead. The `latest/` prefix reuses the same names and
refreshes within its short TTL.

Authentication uses GitHub OIDC (no long-lived keys). The workflow fails fast with a
clear message if its required repo configuration is missing; see the workflow file for
the expected secrets and variables.
