# Integration Testing

Integration tests run against a Megaport API environment (staging by default) and verify that CLI commands work correctly end-to-end. They are separate from unit tests, which use mocks and run on every PR.

## Prerequisites

Set the following environment variables before running integration tests:

```bash
export MEGAPORT_ACCESS_KEY=<your-access-key>
export MEGAPORT_SECRET_KEY=<your-secret-key>
export MEGAPORT_ENVIRONMENT=staging  # staging (default), production, or development
```

Credentials can be obtained from the relevant Megaport portal. `MEGAPORT_ENVIRONMENT` selects the target API: `staging` (default), `production`, or `development`. The test helper (`testutil.IntegrationEnvironment`) defaults to staging when the variable is empty or unrecognized, so a typo can never silently point the suite at production. The credentials must match the chosen environment.

The read-only smoke tests discover resources dynamically and run in any environment. The provisioning lifecycle tests use hardcoded staging location IDs and are staging-only: they skip automatically (via `testutil.RequireStagingForProvisioning`) whenever `MEGAPORT_ENVIRONMENT` is anything other than staging, so they can never create real resources in production or development.

## Running tests

```bash
# Read-only tests only — fast (< 5 min), no resources provisioned
make test-integration-readonly

# Full suite including provisioning lifecycle tests (~20-30 min against staging)
# Provisioning tests create and tear down real staging resources;
# more provisioning coverage is added incrementally.
make test-integration

# A single package
go test -tags integration -run '^TestIntegration_' -v -timeout 30m ./internal/commands/ports/...
```

## What gets created on staging

The provisioning lifecycle tests create real resources on the staging account: ports, VXC, MCR, MVE, IX, and NAT Gateway, plus a service key (against a port the test buys) and a pending-invite user. More resources are added incrementally. They run whenever the full suite runs, locally via `make test-integration` and in the manual provisioning CI job, never in the nightly read-only job. All test resources are named with the prefix `CLI-Test-` for easy identification.

Resources are cleaned up automatically via `t.Cleanup()` at the end of each test, even when the test fails. However, if a test run is interrupted (e.g. `Ctrl+C`), cleanup may not run. In that case, log in to the staging portal and delete any leftover resources prefixed with `CLI-Test-`.

The read-only integration tests cover `billing_market`, `locations`, `managed_account`, `partners`, `product`, `servicekeys`, `status`, `topology`, and `users` outright, plus read-only smoke tests for ports, VXC, MCR, MVE, IX, and NAT Gateway. They provision nothing: they list/get/status existing resources and skip cleanly when the account has none.

### MCR routing coverage

The MCR lifecycle tests cover prefix filter lists for both IPv4 and IPv6 (create/get/update/delete). The CLI exposes no standalone command to configure static IP routes or BGP peering on an MCR, so there is no MCR route/BGP lifecycle test: that configuration is set through a VXC's A-end MCR partner config. Static IP routes are exercised by the VXC MCR-vrouter integration test; BGP peering is configured the same way but is not yet covered by an integration test. The MCR looking-glass `ip-routes`/`bgp-routes`/`bgp-neighbor-routes` commands are read-only diagnostics, not routing config.

## Build tag

All integration test files carry the `//go:build integration` build tag; some lifecycle files also add `&& provisioning` (see below):

```go
//go:build integration

package ports
```

Running `go test ./...` (without `-tags integration`) excludes these files entirely. They only compile and run when `-tags integration` is passed explicitly.

### The `provisioning` sub-tag

A lifecycle test that lives in a package the nightly read-only job also runs carries an extra `provisioning` tag alongside `integration`:

```go
//go:build integration && provisioning

package servicekeys
```

The nightly read-only job builds only `-tags integration`, so adding `provisioning` excludes a test from that build entirely. It is load-bearing for `servicekeys` and `users`: their nightly step runs `-run '^TestIntegration_'` with no `ReadOnly` name filter, so without the tag their lifecycle tests (the service key test buys a port to use as its product) would run nightly. The core-resource packages (ports, VXC, MCR, MVE, IX, and NAT Gateway) are instead run nightly under `-run 'TestIntegration_.*ReadOnly$'`, so their lifecycle tests are already kept out by the name filter. The ports, MCR, and NAT Gateway lifecycle files (`ports_integration_test.go`, `mcr_integration_test.go`, `nat_gateway_lifecycle_integration_test.go`) additionally carry `provisioning` as a second, build-level guarantee that their tests can never run nightly; VXC, MVE, and IX rely on the name filter alone and stay `//go:build integration`. IX is the weakest case: its lifecycle and read-only tests share one file (`ix_integration_test.go`), so only the `-run` regex keeps the IX lifecycle test out of nightly.

## CI

Integration tests run in CI via `.github/workflows/integration-test.yml`:

- **Read-only job**: runs nightly on `main` and on manual trigger. Tests `billing_market`, `locations`, `managed_account`, `partners`, `product`, `servicekeys`, `status`, `topology`, and `users` outright, plus read-only smoke tests (`list`/`get`/`status`) for ports, VXC, MCR, MVE, IX, and NAT Gateway selected via `-run 'TestIntegration_.*ReadOnly$'` so the provisioning lifecycle tests in those packages never run nightly. Fast, no resource cost.
- **Provisioning job**: manual trigger only (`workflow_dispatch`), built with `-tags 'integration provisioning'`. Runs lifecycle tests for ports, VXC, MCR, MVE, IX, and NAT Gateway, plus the service key and user lifecycles, and additional resources as they are added.

## Adding a new integration test

1. Create `internal/commands/<resource>/<resource>_integration_test.go`
2. Set `//go:build integration` at the top and use `package <resource>` (not `package <resource>_test`)
3. Authenticate using one of the helpers below (see "Authentication helpers")
4. Call action functions directly. For parallel tests, read state via `testutil.SharedIntegrationClient(t)` rather than `output.CaptureOutput` (see "Output capture and parallelism")
5. Use `t.Cleanup()` for resource deletion (e.g. deleting staging ports/VXCs) so cleanup runs even on test failure; `defer` is fine for non-resource cleanup like restoring login state
6. Keep mutating tests out of the nightly read-only job. The nightly job runs `servicekeys` and `users` with no `ReadOnly` name filter, so a lifecycle test there MUST carry `&& provisioning` (in its own file) to stay out of the `-tags integration`-only build. The core-resource packages (ports, VXC, MCR, MVE, IX, and NAT Gateway) run nightly under `-run 'TestIntegration_.*ReadOnly$'`, so naming a lifecycle test something other than `...ReadOnly` already excludes it; ports, MCR, and NAT Gateway additionally carry `&& provisioning` as an extra guard, while VXC, MVE, and IX rely on the name filter alone. A package the read-only job never runs (e.g. `apply`) needs only `//go:build integration`
7. Add the package to the provisioning job in `.github/workflows/integration-test.yml`

See `internal/commands/locations/locations_integration_test.go` for a serial read-only example and `internal/commands/ports/ports_integration_test.go` for a parallel provisioning-lifecycle example.

### Authentication helpers

Two helpers in `internal/testutil` handle authentication against the configured environment. Pick based on whether your tests use `t.Parallel()`.

**Serial tests** (no `t.Parallel()`): use `testutil.SetupIntegrationClient` plus `testutil.LoginWithClient`. The login override is saved on entry and restored on cleanup:

```go
func TestIntegration_Foo(t *testing.T) {
    client := testutil.SetupIntegrationClient(t)
    defer testutil.LoginWithClient(t, client)()
    // ...
}
```

**Parallel tests** (`t.Parallel()`): use `testutil.RequireSharedIntegrationClient`. It authorises once per process via `sync.Once` and installs the login override exactly once; it never restores. All callers share a single authorised `*megaport.Client`, which is safe because they all target the same environment.

```go
func TestIntegration_Foo(t *testing.T) {
    t.Parallel()
    testutil.RequireSharedIntegrationClient(t)
    // ...
}
```

Do not combine `LoginWithClient` with `t.Parallel()`: its save/restore pattern races when concurrent tests each capture and restore `config.LoginFunc`, leaving the global pointing at a stale closure.

### Output capture and parallelism

`output.CaptureOutput` swaps the process-wide `os.Stdout` for a tmpfile and holds a mutex for the duration of the wrapped function. Combined with `t.Parallel()`, that has two failure modes:

1. **Serialisation**: the lock serialises every captured call, so parallel goroutines block whenever any one of them is inside `CaptureOutput`. Long blocking actions (e.g. `BuyPort` with `WaitForProvision=true`) erase the wall-time benefit of `t.Parallel()`.
2. **Stdout races**: action code uses spinner goroutines that write to `os.Stdout` asynchronously. While goroutine A holds the capture mutex with `os.Stdout = tmpA`, goroutine B's spinner can write into `tmpA` or into a just-closed file descriptor after A's defer runs. Captures come back polluted or empty.

For parallel lifecycle tests, prefer this pattern:

- Drive the side-effecting action (`BuyPort`, `UpdatePort`, `DeletePort`, …) without wrapping it in `CaptureOutput`. Its real stdout output interleaves with other tests' output on the terminal, which is harmless.
- If the action returns data via the SDK response (e.g. `BuyPortResponse.TechnicalServiceUIDs`), hook the underlying package-level function variable (`buyPortFunc` etc.) in an `init()` and store the response in a `sync.Map` keyed by something the test controls (request name, UID). See `internal/commands/ports/ports_integration_test.go` for an example.
- Read state for assertions via `testutil.SharedIntegrationClient(t)` rather than scraping `GetPort`/`ListPorts` stdout. This bypasses `CaptureOutput` entirely and is parallel-safe.

Serial read-only tests (e.g. `locations`) can continue to use `output.CaptureOutput` directly because there is no concurrent goroutine to race with.
