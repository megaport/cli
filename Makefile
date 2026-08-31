.PHONY: build test test-cover test-cover-html test-integration test-integration-readonly e2e e2e-staging lint fmt vet check clean wasm wasm-build-guard wasm-vet wasm-smoke wasm-compress web-static

# Build the CLI binary
build:
	go build -v -o megaport-cli .

# Run all tests
test:
	go test -v ./...

# Run tests with coverage report
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@rm -f coverage.out

# Generate an HTML coverage report for local inspection (coverage.html kept on disk)
test-cover-html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@rm -f coverage.out
	@echo "Coverage report written to coverage.html"

# Run integration tests against staging API (requires credentials — see docs/INTEGRATION_TESTING.md)
# The `provisioning` tag also pulls in lifecycle tests that create and tear down
# real staging resources (e.g. the service key test provisions a port).
test-integration:
	go test -tags 'integration provisioning' -run '^TestIntegration_' -v -timeout 30m ./internal/commands/...

# Run only read-only integration tests — fast, no resources provisioned.
# The locations-style packages have only read-only tests; core packages also hold
# provisioning lifecycle tests, so they are scoped to the ReadOnly-suffixed names.
test-integration-readonly:
	go test -tags integration -run '^TestIntegration_' -v -timeout 5m \
		./internal/commands/billing_market/... \
		./internal/commands/locations/... \
		./internal/commands/managed_account/... \
		./internal/commands/partners/... \
		./internal/commands/product/... \
		./internal/commands/servicekeys/... \
		./internal/commands/status/... \
		./internal/commands/topology/... \
		./internal/commands/users/...
	go test -tags integration -run 'TestIntegration_.*ReadOnly$$' -v -timeout 5m \
		./internal/commands/ix/... \
		./internal/commands/mcr/... \
		./internal/commands/mve/... \
		./internal/commands/ports/... \
		./internal/commands/vxc/...

# Run the hermetic native-binary e2e tests (built behind the `e2e` build tag).
# Builds the CLI binary and drives it via argv; no credentials needed.
e2e:
	go test -tags e2e -run '^TestE2E_' -skip '^TestE2E_Staging_' -v ./e2e/...

# Run the live staging e2e tests (read-only, requires MEGAPORT_* credentials).
e2e-staging:
	go test -tags e2e -run '^TestE2E_Staging_' -v -timeout 15m ./e2e/...

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	gofmt -w .

# Run go vet
vet:
	go vet ./...

# Run all checks (lint + test) — use this before committing
check: lint test

# Build WASM binary
wasm:
	GOOS=js GOARCH=wasm go build -trimpath -tags js,wasm -ldflags="-s -w" -o web/megaport.wasm .

# Compile-only guard for the browser target. Fails fast if the WASM build breaks.
# Covers the whole module so wasm-tagged code outside the root package can't rot silently.
wasm-build-guard:
	GOOS=js GOARCH=wasm go build -tags js,wasm ./...

# Vet the whole js/wasm surface, including wasm-tagged test files. This compiles
# every wasm test package, so test files that stop building are caught in CI.
wasm-vet:
	GOOS=js GOARCH=wasm go vet -tags js,wasm ./...

# Smoke-test a read-only command end-to-end through the browser fetch transport.
# Builds the WASM binary, then runs it under Node against a live API (default: staging).
# wasm_exec.js is copied to /tmp to avoid dirtying the working tree.
wasm-smoke: wasm
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" /tmp/wasm_exec_smoke.js
	WASM_EXEC_JS=/tmp/wasm_exec_smoke.js node scripts/wasm-smoke.mjs

# Pre-compress the WASM artifact (brotli q11 + gzip -9) for CDN serving
wasm-compress: wasm
	go run ./cmd/wasmcompress web/megaport.wasm

# Build the WASM binary + wasm_exec.js loader into web/dist/ (for CDN hosting)
web-static:
	./scripts/build-web.sh

# Clean build artifacts
clean:
	rm -f megaport-cli cover*.out coverage*.out coverage.html web/megaport.wasm web/megaport.wasm.br web/megaport.wasm.gz web/wasm_exec.js
	rm -rf web/dist
