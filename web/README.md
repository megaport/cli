# Megaport CLI WASM build artifacts

This directory holds the WebAssembly build output for the Megaport CLI. The browser
front-end lives in [`vue-demo/`](./vue-demo); the portal and the public tutorial each
embed the `.wasm` in their own front-ends and run it in-browser. There is no server-side
component: the assets are served as static files by a CDN, and the WASM authenticates
against the Megaport API directly from the browser.

## Building

```bash
# WASM binary (writes web/megaport.wasm)
make wasm
# or ./wasm.sh, which also copies wasm_exec.js from the Go toolchain

# Assemble the static site into web/vue-demo/ (no Docker, no Go server)
./scripts/build-web.sh
```

## Pre-compressed WASM artifacts

`megaport.wasm` is ~18.5 MiB raw, above CloudFront's 10 MB auto-compression cap, so the
build pre-compresses it at the origin. `cmd/wasmcompress` writes two sibling objects
next to the wasm:

- `megaport.wasm.br`: brotli, quality 11 (~2.95 MiB; preferred)
- `megaport.wasm.gz`: gzip -9 (~4.4 MiB; fallback)

This runs in the **build**, not the CDN sync: `make wasm-compress` invokes
`cmd/wasmcompress` after building the wasm. The raw identity object is kept for clients
that accept neither encoding.

Whatever uploads these to the CDN origin must serve the compressed objects with
`Content-Type: application/wasm` and the matching `Content-Encoding` (`br` / `gzip`),
shipping `.br`/`.gz`/identity as separate objects selected on `Accept-Encoding`. The
upload itself and the `Accept-Encoding` routing are owned by the infra/CDN tooling, not
this repo.

## Caching

When the publishing flow content-hashes the wasm (`cmd/wasmhash`) into
`megaport.<hash>.wasm`, `index.html` carries that hashed URL via
`window.__MEGAPORT_WASM_URL__`. This only works if the CDN respects the right cache
lifetimes, so whatever serves these files (the CDN or origin) must apply them:

- `megaport.<hash>.wasm` (and its `.br`/`.gz` siblings): `Cache-Control: public,
  max-age=31536000, immutable`. The hash changes when the content does, so it never
  needs revalidating.
- `index.html`: `no-cache` (short TTL at most). It must be re-fetched so clients pick
  up a new hashed wasm URL.
- `wasm_exec.js`: `no-cache`. It is served from a fixed unhashed path, so it must
  revalidate to stay paired with the wasm's Go runtime after a toolchain upgrade.

A plain unhashed `megaport.wasm` must NOT be served immutable at a flat, unversioned
path. (The S3 publish flow below does serve an unhashed `megaport.wasm`, but only under
a per-version prefix where the path itself is unique, so immutable caching is safe
there.)

## Publishing to S3 (CDN)

`.github/workflows/wasm-publish.yaml` builds the static site and publishes it to a
CloudFront-fronted S3 bucket. It is manual-only (`workflow_dispatch`): a production
publish is always a deliberate action, never an automatic tag push.

Each run uploads the whole static site under two prefixes:

- `<version>/`: immutable, long-lived cache. Treat a version label as write-once, so use
  a fresh tag or `version` input per release.
- `latest/`: short TTL, refreshed on every run.

The portal loads `megaport.wasm` (brotli, served with `Content-Encoding: br`) and
`wasm_exec.js` by static config URL, so unlike the `cmd/server` flow above these
filenames are kept stable rather than content-hashed; the version/latest prefix is the
cache-buster instead.

Authentication uses GitHub OIDC (no long-lived keys). The workflow fails fast with a
clear message if its required repo configuration is missing; see the workflow file for
the expected secrets and variables.

## Local preview

The front end has its own Vite dev server in `frontend-integration/`.
[Local Development](../WASM_README.md#local-development) shows how to build the wasm into
that directory and start the server.
