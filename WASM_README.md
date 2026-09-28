# Megaport CLI - WebAssembly (WASM) Browser Terminal

A browser-based terminal for the Megaport CLI powered by **WebAssembly (WASM)**. The
CLI is compiled to a `.wasm` binary that runs entirely in the browser, so customers can
use it without installing anything locally.

## What is This?

- **Runs in the browser** - No local installation required
- **Powered by WebAssembly** - Go code compiled to WASM runs directly in the browser
- **In-browser authentication** - Credentials stay in browser memory; the WASM calls the Megaport API directly, with no server-side component
- **XTerm.js Terminal** - Full-featured terminal emulator with ANSI support
- **Early Release** - Covers all registered resource modules (see Available Commands below). Auth, config, completion, `generate-docs`, and `version` are not applicable in the browser.

## How Customers Use It

The browser CLI is embedded in the Megaport Portal, which loads the WASM binary from the
CDN (`s3://media.megaport.com/portal/megaport-cli/`) and runs it in-page. There is no
server to deploy: the front end loads static assets and the WASM authenticates against
the Megaport API directly.

### Available Commands

The WASM build registers the following modules. Each module exposes the same subcommands it provides in the native CLI (so `partners` is still `list` / `find`, `locations` is still `list` / `get`, and so on), subject to the constraints of running in the browser:

- `locations` - Find Megaport locations and metros
- `ports` - Manage Megaport ports (including LAG ports)
- `mcr` - Manage Megaport Cloud Routers
- `mve` - Manage Megaport Virtual Edge devices
- `vxc` - Manage Virtual Cross Connects
- `nat-gateway` - Manage NAT gateways
- `partners` - Look up cloud partner ports
- `product` - Inspect provisioned products
- `servicekeys` - Manage service keys
- `status` - Check resource provisioning status
- `topology` - View network topology

The registered list is the source of truth in `cmd/megaport/modules_wasm.go`. Any module not registered there is unavailable in the WASM build, including `auth`, `config`, `completion`, `generate-docs`, `version`, `ix`, `users`, `apply`, `managed-account`, and `billing-market`. Some rely on the local filesystem (profile storage, completion scripts); others have simply not been wired into the browser build yet.

**Output Formats**: The WASM build supports the following output formats:

- `--output table` (default) - Formatted table with styled output
- `--output json` - JSON format for programmatic use
- `--output csv` - CSV format for data export
- `--output xml` - XML format

`--output go-template` is not supported in WASM/browser builds.

**Example Commands**:

```bash
megaport-cli locations list
megaport-cli locations list --output json
megaport-cli ports list
megaport-cli ports get <portUID>
megaport-cli vxc list --status LIVE
megaport-cli partners list --company-name "Amazon Web Services"
```

## Authentication

Authentication happens entirely in the browser. There is no login server, session
token, or API proxy.

- **Standalone demo**: a credential form calls the WASM's `setAuthCredentials`, which
  holds your access key and secret key in memory for the lifetime of the page. The WASM
  uses them to sign requests to the Megaport API directly.
- **Portal integration**: the host page passes an existing session token via the WASM's
  `setAuthToken`, so the user doesn't re-enter credentials.

Credentials and tokens live only in browser memory and are cleared on reload or when the
page calls `clearAuthCredentials`.

### Token injection contract (portal integration)

`setAuthToken(token, hostname, environment?, expiry?)`:

- `token`, `hostname` are required, as before.
- `environment` (optional) overrides the environment normally derived from `hostname`.
- `expiry` (optional, new) is the token's real expiry, as epoch milliseconds or an
  RFC3339 string. It is stored and echoed back in the response so the host can track
  it, but the CLI does not check it proactively (see below). Omit it, or pass
  `0`/an unparseable value, if the expiry is unknown.

There is no token refresh on this path: the CLI holds only the token the host gave it,
with no credentials to renew it with, and it never checks `expiry` on its own.
Instead, whenever the API rejects a request with 401/403, whether that's because the
token expired or because the host revoked it, the failing command's output contains
the substring `MEGAPORT_SESSION_EXPIRED`. The host is responsible for watching command
output for this marker and, on a match, prompting the user to re-authenticate and
calling `setAuthToken` again with a fresh token. The registration and doc comments
for `setAuthToken` live inline in `internal/wasm/wasm.go`.

Separately, the browser-cached OAuth token used by credentials-based logins
(`window.tokenManager.getToken(environment)`, checked to avoid re-authenticating on
every page load) may return either a bare token string (legacy, no TTL, treated as
valid for 24h) or `{ token, expiry }` to record a real TTL instead.

## JavaScript API

The WASM binary exposes command execution on `window`:

```javascript
window.executeMegaportCommandAsync(command, callback)
```

- `command` - full command string, e.g. `"port list --output json"`
- `callback` - called once with the result: `{ output?: string, error?: string }`

This is the only supported entrypoint for running commands. Execution has to be
asynchronous: a synchronous call would block the JS event loop while the CLI waits on the
browser's fetch-based transport, hanging the tab until the request times out. Internally,
`executeMegaportCommandAsync` runs the command on a goroutine and serializes access to the
shared output buffers so concurrent calls don't race each other.

`window.executeMegaportCommand` (no `Async` suffix) is a deprecated stub kept for one
release as a soft landing for hosts still detecting or calling it. It performs no work and
always returns `{ error: "synchronous execution is not supported; use executeMegaportCommandAsync" }`.
It will be removed in a future release; new integrations should not call it.

Full type definitions for the whole JS surface (auth, config file, prompts, telemetry)
are documented inline in `internal/wasm/`; there is no separate reference front end in
this repo. Front-end integrators (e.g. the Portal) own their own wrapper around the
`window.executeMegaportCommandAsync` API described above.

## Interactive Mode

Some commands prompt for input (interactive `buy`/`update` flows, confirmations, secrets).
In the browser there is no stdin, so the WASM asks the host page for each value through a
small set of JavaScript functions. Wire these up and run the command through
`executeMegaportCommandAsync` (see [JavaScript API](#javascript-api)), or interactive
commands will never receive a response.

### Host functions

The WASM registers these on `window` at startup:

| Function | Purpose |
|---|---|
| `registerPromptHandler(cb)` | Register a callback the WASM invokes with each prompt request. |
| `submitPromptResponse(id, response)` | Reply to the prompt `id` with the user's input (a string). |
| `cancelPrompt(id)` | Cancel the prompt `id`. A value prompt fails with a "prompt cancelled by user" error; a confirmation is treated as declined. |

### Prompt request shape

Your handler receives a single object:

```js
{
  id: "prompt_1_1700000000000000000", // unique id; echo it back in submit/cancel
  message: "Enter port name:",         // text to show the user
  type: "text",                        // "text" | "confirm" | "password" | "resource"
  resourceType: "port"                 // set for resource and secret-resource prompts (port, mcr, vxc, ...), else ""
}
```

Mask the input when `type === "password"`: render an `<input type="password">` or otherwise
hide the characters. Password prompts and secret-resource prompts (for example VXC/MVE
passwords and pre-shared keys) set this type. Note that some other secret-bearing inputs
(such as partner auth/service/shared keys and MVE registration keys) are currently sent as
`type === "resource"`, so don't rely on the password type alone if you want to mask every
possible secret.

### Lifecycle

```js
registerPromptHandler((request) => {
  const masked = request.type === 'password';
  showPrompt(request.message, { masked }).then((answer) => {
    if (answer === null) {
      cancelPrompt(request.id);        // user dismissed the prompt
    } else {
      submitPromptResponse(request.id, answer);
    }
  });
});

executeMegaportCommandAsync('vxc buy --interactive', (result) => {
  console.log(result.output || result.error);
});
```

A prompt left unanswered times out after 10 minutes and the command receives an error.

### Live output streaming

By default a command's output arrives once, in the async callback's result. To render
output as it is produced instead, register a handler before running the command:

```js
registerOutputHandler((chunk) => terminal.write(chunk)); // chunk is a string
```

The contract:

- Only narrative output (progress and status messages) streams. Structured document
  output (table/JSON/CSV/XML) is never streamed; it arrives once in the completion result.
- If your handler received at least one chunk without throwing, the completion result does
  **not** repeat the streamed narrative, so don't render both.
- If the handler throws or no chunk was delivered, streaming is disabled for the rest of
  that command and the completion result falls back to the full captured output
  (already-streamed chunks may then appear twice).

## Building

```bash
# WASM binary only (writes web/megaport.wasm)
make wasm

# WASM binary + wasm_exec.js loader, assembled into web/dist/
make web-static          # or: ./scripts/build-web.sh
```

`web-static` needs the Go toolchain on `PATH`. It produces a self-contained
**`web/dist/`** directory (`megaport.wasm` + `wasm_exec.js`) ready to publish to a CDN.
See [`web/README.md`](web/README.md) for the wasm pre-compression and cache-header
details.

## Enabling a Module for WASM

Every module follows the same recipe to go from native-only to browser-enabled:

1. **Register it.** Add the module to `registerModules()` in
   `cmd/megaport/modules_wasm.go` (import the package and call
   `moduleRegistry.Register(<module>.NewModule())`). This file is the source of
   truth for what ships in the browser build.

2. **Build the guard.** Run `make wasm-build-guard`
   (`GOOS=js GOARCH=wasm go build -tags js,wasm -o /dev/null .`). This fails if
   the module pulls in anything that does not compile under `js/wasm` (native
   filesystem, `os/exec`, `syscall`, etc.). CI runs the same step on every PR.

3. **Add a `_wasm` override only if needed.** An override is needed when the
   native action builds its own `http.Client` instead of going through
   `config.Login(ctx)` or `config.NewUnauthenticatedClient()` — the WASM
   versions of those functions already use the browser fetch transport, so any
   module that calls them directly needs no override. A module needs a
   `<module>_actions_wasm.go` (`//go:build js && wasm`) only when its native
   path does something the browser can't, such as reading the local filesystem
   or constructing its own `http.Client`. The pattern is to override the
   action's function variable in `init()`, using
   `config.NewUnauthenticatedClient()` for public endpoints or
   `config.Login(ctx)` for authenticated ones. If you need a reference,
   `internal/commands/ports/ports_actions_wasm.go` shows the authenticated
   pattern.

4. **Smoke-test it.** Run `make wasm-smoke` to round-trip a command through the
   browser fetch transport against a live API (defaults to `locations list`
   against staging, no credentials needed). Requires Node.js 22+ on `PATH`. To
   exercise a specific module:

   ```bash
   WASM_SMOKE_COMMAND='locations list --output json' make wasm-smoke
   ```

   The command must emit JSON (`--output json`) and return a non-empty JSON
   array (use a `list` subcommand, not a single-resource `get`). For anything
   that needs auth, set `MEGAPORT_ACCESS_KEY` / `MEGAPORT_SECRET_KEY` in the
   environment first (local use only; do not add credentials to the CI smoke
   job).

## Publishing to the Portal

The Portal loads the WASM binary from `s3://media.megaport.com/portal/megaport-cli/`. Publishing is currently manual.

### Prerequisites

- AWS CLI configured with SSO for the `ProductionDeveloper` role.

### Steps

```bash
# 1. Ensure AWS SSO auth is active (login if needed)
aws sso login
aws sts get-caller-identity
# 2. Build the WASM binary and copy the matching wasm_exec.js loader. The loader is
#    not checked in: copy it fresh from the toolchain so it stays paired with the Go
#    version that built the wasm (./wasm.sh does both steps if you prefer).
GOOS=js GOARCH=wasm go build -tags js,wasm -o web/megaport.wasm .
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/wasm_exec.js

# 3. Upload the WASM binary and the wasm_exec.js loader.
#    `--content-type application/wasm` is required so the file isn't served as
#    application/octet-stream: `WebAssembly.instantiateStreaming` rejects anything
#    else, which would break loading the WASM in the browser.
aws s3 cp web/megaport.wasm s3://media.megaport.com/portal/megaport-cli/megaport.wasm \
    --content-type application/wasm
aws s3 cp web/wasm_exec.js  s3://media.megaport.com/portal/megaport-cli/wasm_exec.js
```

For CDN hosting (S3 + CloudFront), publish via the `.github/workflows/wasm-publish.yaml`
workflow rather than a plain sync. It runs `make web-static`, brotli pre-compresses the
wasm (`cmd/wasmcompress`), then uploads `megaport.wasm` with `Content-Encoding: br` and
pins `Content-Type` on both the wasm and `wasm_exec.js`. It syncs only the remaining
static assets, so a bare `aws s3 sync web/dist/` would instead serve the wasm
uncompressed and let the CDN mis-infer its MIME type, which breaks
`WebAssembly.instantiateStreaming`.

## Troubleshooting

### Can't build WASM

```bash
# Make sure you have Go 1.25 or later (matches go.mod)
go version

# Check WASM support
GOOS=js GOARCH=wasm go version
```

### Can't connect to Megaport API

The browser calls the Megaport API directly, so debug from the browser:

- open the dev tools Network tab and inspect the failing request
- confirm the access key / secret key are correct for the selected environment

## License

See LICENSE file for details.
