// These tests touch only buffers and JS globals that node's headless
// wasm_exec.js runtime already provides (no fetch, localStorage, or other
// browser-only bridges), so they run under the default `js,wasm` CI step.
// Tests that need a real browser host live in wasm_browser_test.go, gated
// behind the `browser` build tag.
//go:build js && wasm

package wasm

import (
	"bytes"
	"math"
	"os"
	"strings"
	"syscall/js"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetCapturedOutput_Priority verifies output priority order
func TestGetCapturedOutput_Priority(t *testing.T) {
	tests := []struct {
		name           string
		setupFn        func()
		expectedSource string
		expectedOutput string
	}{
		{
			name: "JSON has highest priority",
			setupFn: func() {
				ResetOutputBuffers()
				js.Global().Set("wasmJSONOutput", "json output")
				js.Global().Set("wasmCSVOutput", "csv output")
				js.Global().Set("wasmTableOutput", "table output")
				stdoutBuffer.WriteString("stdout output")
				_, _ = WasmOutputBuffer.Write([]byte("direct output"))
			},
			expectedSource: "JSON buffer",
			expectedOutput: "json output",
		},
		{
			name: "CSV has second priority",
			setupFn: func() {
				ResetOutputBuffers()
				js.Global().Set("wasmCSVOutput", "csv output")
				js.Global().Set("wasmTableOutput", "table output")
				stdoutBuffer.WriteString("stdout output")
				_, _ = WasmOutputBuffer.Write([]byte("direct output"))
			},
			expectedSource: "CSV buffer",
			expectedOutput: "csv output",
		},
		{
			name: "Table output is prefixed with direct status buffer",
			setupFn: func() {
				ResetOutputBuffers()
				js.Global().Set("wasmTableOutput", "table output")
				stdoutBuffer.WriteString("stdout output")
				_, _ = WasmOutputBuffer.Write([]byte("direct output"))
			},
			expectedSource: "table buffer",
			expectedOutput: "direct outputtable output",
		},
		{
			name: "Table without status returns table only",
			setupFn: func() {
				ResetOutputBuffers()
				js.Global().Set("wasmTableOutput", "table output")
			},
			expectedSource: "table buffer",
			expectedOutput: "table output",
		},
		{
			name: "Direct buffer has fourth priority",
			setupFn: func() {
				ResetOutputBuffers()
				stdoutBuffer.WriteString("stdout output")
				_, _ = WasmOutputBuffer.Write([]byte("direct output"))
			},
			expectedSource: "direct buffer",
			expectedOutput: "direct output",
		},
		{
			name: "Stdout/stderr is lowest priority",
			setupFn: func() {
				ResetOutputBuffers()
				stdoutBuffer.WriteString("stdout output")
				stderrBuffer.WriteString("stderr output")
			},
			expectedSource: "combined stdout/stderr",
			expectedOutput: "stdout outputstderr output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn()
			output := GetCapturedOutput()
			assert.Equal(t, tt.expectedOutput, output)
		})
	}
}

// TestDirectOutputBuffer_Write verifies thread-safe writes
func TestDirectOutputBuffer_Write(t *testing.T) {
	buffer := &DirectOutputBuffer{
		buffer: &bytes.Buffer{},
	}

	testData := []byte("test data")
	n, err := buffer.Write(testData)

	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)
	assert.Equal(t, "test data", buffer.String())
}

// TestDirectOutputBuffer_Concurrent verifies thread safety
func TestDirectOutputBuffer_Concurrent(t *testing.T) {
	buffer := &DirectOutputBuffer{
		buffer: &bytes.Buffer{},
	}

	done := make(chan bool)
	iterations := 100

	// Multiple goroutines writing concurrently
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				_, _ = buffer.Write([]byte("x"))
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have exactly 1000 'x' characters (10 goroutines * 100 iterations)
	result := buffer.String()
	assert.Equal(t, 1000, len(result))
	assert.Equal(t, strings.Repeat("x", 1000), result)
}

// TestDirectOutputBuffer_Reset verifies buffer reset
func TestDirectOutputBuffer_Reset(t *testing.T) {
	buffer := &DirectOutputBuffer{
		buffer: &bytes.Buffer{},
	}

	_, _ = buffer.Write([]byte("test data"))
	assert.NotEqual(t, "", buffer.String())

	buffer.Reset()
	assert.Equal(t, "", buffer.String())
}

// TestCustomWriter verifies custom writer implementation
func TestCustomWriter(t *testing.T) {
	var buf bytes.Buffer
	writer := &customWriter{writer: &buf}

	testData := []byte("test data")
	n, err := writer.Write(testData)

	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)
	assert.Equal(t, "test data", buf.String())
}

// TestCaptureOutput verifies output capture functionality
func TestCaptureOutput(t *testing.T) {
	testOutput := "test output from function"

	captured := CaptureOutput(func() {
		_, _ = WasmOutputBuffer.Write([]byte(testOutput))
	})

	// Should contain the output (may have additional content from pipes)
	assert.Contains(t, captured, testOutput)
}

// TestSplitArgs verifies command argument parsing
func TestSplitArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple command",
			input:    "locations list",
			expected: []string{"locations", "list"},
		},
		{
			name:     "command with flags",
			input:    "ports list --format json",
			expected: []string{"ports", "list", "--format", "json"},
		},
		{
			name:     "command with double quotes",
			input:    `ports create --name "My Port"`,
			expected: []string{"ports", "create", "--name", "My Port"},
		},
		{
			name:     "command with single quotes",
			input:    `ports create --name 'My Port'`,
			expected: []string{"ports", "create", "--name", "My Port"},
		},
		{
			name:     "removes program name",
			input:    "megaport-cli locations list",
			expected: []string{"locations", "list"},
		},
		{
			name:     "removes program name with path",
			input:    "./megaport-cli locations list",
			expected: []string{"locations", "list"},
		},
		{
			name:     "removes leading program name literal megaport",
			input:    "megaport locations list",
			expected: []string{"locations", "list"},
		},
		{
			name:     "preserves megaport as a flag value",
			input:    "ports update abc123 --name megaport",
			expected: []string{"ports", "update", "abc123", "--name", "megaport"},
		},
		{
			name:     "preserves megaport as a positional argument",
			input:    "tag megaport ports",
			expected: []string{"tag", "megaport", "ports"},
		},
		{
			name:     "removes leading program name but preserves later megaport flag value",
			input:    "megaport-cli ports update abc123 --name megaport",
			expected: []string{"ports", "update", "abc123", "--name", "megaport"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: nil,
		},
		{
			name:     "complex with mixed quotes",
			input:    `ports create --name "Port 1" --description 'Test port' --speed 1000`,
			expected: []string{"ports", "create", "--name", "Port 1", "--description", "Test port", "--speed", "1000"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitArgs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSetTerminalWidth verifies the exported width setter/getter round-trip,
// that negative values reset to "unset" (0), and that absurdly large values
// are clamped rather than stored as-is.
func TestSetTerminalWidth(t *testing.T) {
	defer SetTerminalWidth(0)

	assert.Equal(t, 0, TerminalWidth(), "width should be unset by default")

	SetTerminalWidth(120)
	assert.Equal(t, 120, TerminalWidth())

	SetTerminalWidth(-5)
	assert.Equal(t, 0, TerminalWidth(), "negative width should reset to unset")

	SetTerminalWidth(1_000_000_000)
	assert.Equal(t, maxTerminalWidthCols, TerminalWidth(), "absurd width should clamp to the max")
}

// TestSetTerminalWidth_JSFunction verifies the JS-exposed setTerminalWidth
// function for present, absent (invalid), and absurd (tiny/huge) widths.
func TestSetTerminalWidth_JSFunction(t *testing.T) {
	RegisterJSFunctions()
	defer SetTerminalWidth(0)

	setFunc := js.Global().Get("setTerminalWidth")

	tests := []struct {
		name    string
		args    []interface{}
		wantOK  bool
		wantCol int
	}{
		{name: "normal width", args: []interface{}{80}, wantOK: true, wantCol: 80},
		{name: "tiny width", args: []interface{}{1}, wantOK: true, wantCol: 1},
		{name: "huge width clamps to max", args: []interface{}{100000}, wantOK: true, wantCol: maxTerminalWidthCols},
		{name: "out-of-int-range width clamps to max", args: []interface{}{1e300}, wantOK: true, wantCol: maxTerminalWidthCols},
		{name: "large negative width resets to unset", args: []interface{}{-1e300}, wantOK: true, wantCol: 0},
		{name: "zero width", args: []interface{}{0}, wantOK: true, wantCol: 0},
		{name: "missing argument", args: []interface{}{}, wantOK: false},
		{name: "non-numeric argument", args: []interface{}{"wide"}, wantOK: false},
		{name: "NaN argument", args: []interface{}{math.NaN()}, wantOK: false},
		{name: "positive infinity argument", args: []interface{}{math.Inf(1)}, wantOK: false},
		{name: "negative infinity argument", args: []interface{}{math.Inf(-1)}, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetTerminalWidth(0)
			result := setFunc.Invoke(tt.args...)

			assert.Equal(t, tt.wantOK, result.Get("success").Bool())
			if tt.wantOK {
				assert.Equal(t, tt.wantCol, TerminalWidth())
			} else {
				assert.Equal(t, 0, TerminalWidth())
				assert.False(t, result.Get("error").IsUndefined())
			}
		})
	}
}

// TestSplitArgs_EdgeCases verifies edge cases in argument parsing
func TestSplitArgs_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "unclosed double quote",
			input:    `command --flag "unclosed`,
			expected: []string{"command", "--flag", "unclosed"},
		},
		{
			name:     "unclosed single quote",
			input:    `command --flag 'unclosed`,
			expected: []string{"command", "--flag", "unclosed"},
		},
		{
			name:     "empty quotes",
			input:    `command --flag "" --other ''`,
			expected: []string{"command", "--flag", "", "--other", ""},
		},
		{
			name:     "multiple spaces",
			input:    "command    with    spaces",
			expected: []string{"command", "with", "spaces"},
		},
		{
			name:     "quotes with spaces inside",
			input:    `command "arg with   spaces"`,
			expected: []string{"command", "arg with   spaces"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitArgs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSetAuthToken verifies token-based authentication with hostname mapping
func TestSetAuthToken(t *testing.T) {
	EnableDebugMode()
	RegisterJSFunctions()

	tests := []struct {
		name           string
		token          string
		hostname       string
		explicitEnv    string
		expectError    bool
		expectedEnv    string // real env name returned in the JS result
		expectedURL    string
		expectedBucket string // MEGAPORT_ENVIRONMENT — always one of production/staging/development
	}{
		// --- hostname-derived cases (no override) ---
		{
			name:           "production portal host",
			token:          "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test",
			hostname:       "portal.megaport.com",
			expectedEnv:    "production",
			expectedURL:    "https://api.megaport.com/",
			expectedBucket: "production",
		},
		{
			name:           "staging portal host",
			token:          "valid-staging-token-12345",
			hostname:       "portal-staging.megaport.com",
			expectedEnv:    "staging",
			expectedURL:    "https://api-staging.megaport.com/",
			expectedBucket: "staging",
		},
		{
			name:           "portal-qa returns real qa env, bucketed to development",
			token:          "qa-token-12345",
			hostname:       "portal-qa.megaport.com",
			expectedEnv:    "qa",
			expectedURL:    "https://api-qa.megaport.com/",
			expectedBucket: "development",
		},
		{
			name:           "portal-uat returns real uat env, bucketed to development",
			token:          "uat-token-12345",
			hostname:       "portal-uat.megaport.com",
			expectedEnv:    "uat",
			expectedURL:    "https://api-uat.megaport.com/",
			expectedBucket: "development",
		},
		{
			name:           "another app hostname (dashboard) derives env via app-env convention",
			token:          "dashboard-token-12345",
			hostname:       "dashboard-qa.megaport.com",
			expectedEnv:    "qa",
			expectedURL:    "https://api-qa.megaport.com/",
			expectedBucket: "development",
		},
		{
			name:           "another app at the apex (dashboard.megaport.com) is production",
			token:          "dashboard-prod-token-12345",
			hostname:       "dashboard.megaport.com",
			expectedEnv:    "production",
			expectedURL:    "https://api.megaport.com/",
			expectedBucket: "production",
		},
		{
			name:           "multi-segment env (api-mpone-dev) preserved",
			token:          "mpone-dev-token-12345",
			hostname:       "api-mpone-dev.megaport.com",
			expectedEnv:    "mpone-dev",
			expectedURL:    "https://api-mpone-dev.megaport.com/",
			expectedBucket: "development",
		},

		// --- hostname-derived failure cases (no override) ---
		{
			name:        "localhost without override now errors (no silent staging fallback)",
			token:       "dev-token-12345",
			hostname:    "localhost",
			expectError: true,
		},
		{
			name:        "non-megaport hostname without override errors",
			token:       "random-token-12345",
			hostname:    "example.com",
			expectError: true,
		},
		{
			name:        "look-alike hostname (megaport.com.attacker.com) fails closed",
			token:       "lookalike-token-12345",
			hostname:    "megaport.com.attacker.com",
			expectError: true,
		},

		// --- explicit override cases ---
		{
			name:           "explicit qa override from production hostname (mismatch)",
			token:          "explicit-env-token-12345",
			hostname:       "portal.megaport.com",
			explicitEnv:    "qa",
			expectedEnv:    "qa",
			expectedURL:    "https://api-qa.megaport.com/",
			expectedBucket: "development",
		},
		{
			name:           "explicit production override from staging hostname",
			token:          "explicit-prod-token-12345",
			hostname:       "portal-staging.megaport.com",
			explicitEnv:    "production",
			expectedEnv:    "production",
			expectedURL:    "https://api.megaport.com/",
			expectedBucket: "production",
		},
		{
			name:           "explicit production override unblocks localhost",
			token:          "local-override-token-12345",
			hostname:       "localhost",
			explicitEnv:    "production",
			expectedEnv:    "production",
			expectedURL:    "https://api.megaport.com/",
			expectedBucket: "production",
		},
		{
			name:           "explicit qa override unblocks localhost",
			token:          "local-qa-token-12345",
			hostname:       "localhost",
			explicitEnv:    "qa",
			expectedEnv:    "qa",
			expectedURL:    "https://api-qa.megaport.com/",
			expectedBucket: "development",
		},
		{
			name:           "explicit override unblocks unrecognised host (non-megaport)",
			token:          "unknown-override-token-12345",
			hostname:       "example.com",
			explicitEnv:    "qa",
			expectedEnv:    "qa",
			expectedURL:    "https://api-qa.megaport.com/",
			expectedBucket: "development",
		},
		{
			name:           "explicit override matching derived (no mismatch)",
			token:          "match-token-12345",
			hostname:       "portal-staging.megaport.com",
			explicitEnv:    "staging",
			expectedEnv:    "staging",
			expectedURL:    "https://api-staging.megaport.com/",
			expectedBucket: "staging",
		},
		{
			name:           "explicit development pin",
			token:          "dev-pin-token-12345",
			hostname:       "localhost",
			explicitEnv:    "development",
			expectedEnv:    "development",
			expectedURL:    "https://api-development.megaport.com/",
			expectedBucket: "development",
		},
		{
			// "prod" passes the env-name regex and gets used as-is for the API URL
			// (yielding api-prod.megaport.com/), but restrictEnvironmentName recognises
			// it as an alias for production so MEGAPORT_ENVIRONMENT lands on
			// "production". This mirrors normalizeEnvironment in config_shared.go.
			name:           "'prod' alias buckets to production (matches normalizeEnvironment)",
			token:          "prod-alias-token-12345",
			hostname:       "portal.megaport.com",
			explicitEnv:    "prod",
			expectedEnv:    "prod",
			expectedURL:    "https://api-prod.megaport.com/",
			expectedBucket: "production",
		},

		// --- normalisation of override input ---
		{
			name:           "whitespace-only override is treated as no override",
			token:          "ws-token-12345",
			hostname:       "portal-staging.megaport.com",
			explicitEnv:    "   ",
			expectedEnv:    "staging",
			expectedURL:    "https://api-staging.megaport.com/",
			expectedBucket: "staging",
		},
		{
			name:           "uppercase override is lowercased",
			token:          "case-token-12345",
			hostname:       "portal.megaport.com",
			explicitEnv:    "PRODUCTION",
			expectedEnv:    "production",
			expectedURL:    "https://api.megaport.com/",
			expectedBucket: "production",
		},

		// --- injection rejection ---
		{
			name:        "override with host-injection chars is rejected",
			token:       "inject-token-12345",
			hostname:    "portal.megaport.com",
			explicitEnv: "foo.attacker.com/",
			expectError: true,
		},
		{
			name:        "override with slash is rejected",
			token:       "slash-token-12345",
			hostname:    "portal.megaport.com",
			explicitEnv: "qa/evil",
			expectError: true,
		},
		{
			name:        "override with @ is rejected",
			token:       "at-token-12345",
			hostname:    "portal.megaport.com",
			explicitEnv: "qa@evil.com",
			expectError: true,
		},

		// --- empty-input errors ---
		{
			name:        "empty token errors",
			token:       "",
			hostname:    "portal.megaport.com",
			expectError: true,
		},
		{
			name:        "empty hostname errors",
			token:       "valid-token-12345",
			hostname:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear any previous auth
			js.Global().Get("clearAuthCredentials").Invoke()

			// Call setAuthToken
			setAuthFunc := js.Global().Get("setAuthToken")
			assert.False(t, setAuthFunc.IsUndefined(), "setAuthToken should be registered")

			var result js.Value
			if tt.explicitEnv != "" {
				result = setAuthFunc.Invoke(tt.token, tt.hostname, tt.explicitEnv)
			} else {
				result = setAuthFunc.Invoke(tt.token, tt.hostname)
			}

			success := result.Get("success").Bool()

			if tt.expectError {
				assert.False(t, success, "should fail for invalid input")
				errMsg := result.Get("error").String()
				assert.NotEmpty(t, errMsg, "error message should be set on failure to guide the caller")
			} else {
				assert.True(t, success, "should succeed for valid input")

				// The JS return surface carries the real resolved env name (e.g. "qa"),
				// not the bucket — that's what the portal UI displays back to the user.
				returnedEnv := result.Get("environment").String()
				assert.Equal(t, tt.expectedEnv, returnedEnv, "returned environment should match expected")

				returnedURL := result.Get("apiURL").String()
				assert.Equal(t, tt.expectedURL, returnedURL, "returned API URL should match expected")

				// Verify auth info shows token is set
				authInfo := js.Global().Get("debugAuthInfo").Invoke()
				tokenSet := authInfo.Get("accessTokenSet").Bool()
				assert.True(t, tokenSet, "token should be marked as set")

				// MEGAPORT_ENVIRONMENT (surfaced via debugAuthInfo) holds the *bucket*
				// — one of production/staging/development — so downstream
				// normalizeEnvironment consumers don't silently coerce non-canonical
				// values to production.
				envVar := authInfo.Get("environment").String()
				assert.Equal(t, tt.expectedBucket, envVar, "MEGAPORT_ENVIRONMENT should be bucketed")

				authMethod := authInfo.Get("authMethod").String()
				assert.Equal(t, "token", authMethod, "authMethod should be 'token'")
			}
		})
	}
}

// TestSetAuthToken_NonStringThirdArg verifies that JS callers passing
// `undefined` or `null` (and other non-string values) as the third argument
// are treated as if no override was provided — rather than having
// Value.String() return "<undefined>"/"<null>" and triggering the regex
// rejection with a misleading error.
func TestSetAuthToken_NonStringThirdArg(t *testing.T) {
	RegisterJSFunctions()
	setAuthFunc := js.Global().Get("setAuthToken")

	tests := []struct {
		name     string
		thirdArg js.Value
	}{
		{"undefined third arg", js.Undefined()},
		{"null third arg", js.Null()},
		{"number third arg", js.ValueOf(0)},
		{"bool third arg", js.ValueOf(false)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			js.Global().Get("clearAuthCredentials").Invoke()

			// Recognised host so the call should succeed when the third arg is ignored.
			result := setAuthFunc.Invoke("tok", "portal-staging.megaport.com", tt.thirdArg)

			assert.True(t, result.Get("success").Bool(), "non-string third arg should be ignored, not trigger validation")
			assert.Equal(t, "staging", result.Get("environment").String(), "should fall through to hostname-derived env")
		})
	}

	t.Run("undefined third arg + unrecognised host fails closed (not regex error)", func(t *testing.T) {
		js.Global().Get("clearAuthCredentials").Invoke()
		result := setAuthFunc.Invoke("tok", "localhost", js.Undefined())
		assert.False(t, result.Get("success").Bool())
		errMsg := result.Get("error").String()
		assert.Contains(t, errMsg, "could not determine environment", "should hit fail-closed path, not regex validation")
		assert.NotContains(t, errMsg, "lowercase letters", "must not surface the regex rejection message")
	})
}

// TestSetAuthToken_TwoArgsSkipsValidation is a regression test for a bug
// where a flipped comparison in setAuthToken caused the env-name regex
// validation to fire on every no-override call, surfacing the misleading
// "environment must contain only lowercase letters..." error instead of the
// real fail-closed message. Two-argument calls must never reach validation.
func TestSetAuthToken_TwoArgsSkipsValidation(t *testing.T) {
	RegisterJSFunctions()
	js.Global().Get("clearAuthCredentials").Invoke()

	// Hostname that the derivation step rejects, so the call errors and we can
	// inspect which error path produced the message.
	result := js.Global().Get("setAuthToken").Invoke("tok", "localhost")
	assert.False(t, result.Get("success").Bool())
	errMsg := result.Get("error").String()
	assert.NotContains(t, errMsg, "lowercase letters", "two-arg call must not reach env-name validation")
	assert.Contains(t, errMsg, "could not determine environment", "two-arg call should hit the fail-closed path")
}

// TestParseExpiry covers the epoch-ms-number and RFC3339-string input shapes,
// plus absent/invalid values falling back to the zero Time.
func TestParseExpiry(t *testing.T) {
	ref := time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name string
		v    js.Value
		want time.Time
	}{
		{"epoch ms number", js.ValueOf(float64(ref.UnixMilli())), ref},
		{"RFC3339 string", js.ValueOf(ref.Format(time.RFC3339)), ref},
		{"undefined", js.Undefined(), time.Time{}},
		{"null", js.Null(), time.Time{}},
		{"zero number", js.ValueOf(0), time.Time{}},
		{"negative number", js.ValueOf(-1), time.Time{}},
		{"empty string", js.ValueOf(""), time.Time{}},
		{"whitespace string", js.ValueOf("   "), time.Time{}},
		{"unparseable string", js.ValueOf("not-a-date"), time.Time{}},
		{"bool", js.ValueOf(true), time.Time{}},
		{"NaN", js.ValueOf(math.NaN()), time.Time{}},
		{"positive infinity", js.ValueOf(math.Inf(1)), time.Time{}},
		{"negative infinity", js.ValueOf(math.Inf(-1)), time.Time{}},
		{"overflows int64", js.ValueOf(math.MaxFloat64), time.Time{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseExpiry(tt.v)
			assert.True(t, tt.want.Equal(got), "ParseExpiry(%v) = %v, want %v", tt.name, got, tt.want)
		})
	}
}

// TestSetAuthToken_Expiry verifies the optional 4th argument is parsed, stored
// on the megaportToken global (so login_wasm.go can read it back), and echoed
// in the success result; omitting it must not regress the zero-expiry fallback.
func TestSetAuthToken_Expiry(t *testing.T) {
	RegisterJSFunctions()
	setAuthFunc := js.Global().Get("setAuthToken")
	expiry := time.Date(2030, 6, 15, 12, 0, 0, 0, time.UTC)

	t.Run("epoch-ms expiry is stored and echoed", func(t *testing.T) {
		js.Global().Get("clearAuthCredentials").Invoke()

		result := setAuthFunc.Invoke("tok", "portal.megaport.com", "production", float64(expiry.UnixMilli()))
		require.True(t, result.Get("success").Bool())
		assert.Equal(t, expiry.Format(time.RFC3339), result.Get("expiry").String())

		stored := js.Global().Get("megaportToken").Get("expiry").Float()
		assert.Equal(t, float64(expiry.UnixMilli()), stored)
	})

	t.Run("RFC3339 string expiry is stored and echoed", func(t *testing.T) {
		js.Global().Get("clearAuthCredentials").Invoke()

		result := setAuthFunc.Invoke("tok", "portal.megaport.com", "production", expiry.Format(time.RFC3339))
		require.True(t, result.Get("success").Bool())
		assert.Equal(t, expiry.Format(time.RFC3339), result.Get("expiry").String())
	})

	t.Run("omitted expiry falls back to zero and is not echoed", func(t *testing.T) {
		js.Global().Get("clearAuthCredentials").Invoke()

		result := setAuthFunc.Invoke("tok", "portal.megaport.com", "production")
		require.True(t, result.Get("success").Bool())
		assert.True(t, result.Get("expiry").IsUndefined(), "expiry should be omitted from the result when not supplied")

		stored := js.Global().Get("megaportToken").Get("expiry").Float()
		assert.Zero(t, stored, "stored expiry should be 0 when not supplied")
	})

	t.Run("undefined expiry behaves like omitted", func(t *testing.T) {
		js.Global().Get("clearAuthCredentials").Invoke()

		result := setAuthFunc.Invoke("tok", "portal.megaport.com", "production", js.Undefined())
		require.True(t, result.Get("success").Bool())
		assert.True(t, result.Get("expiry").IsUndefined())
	})
}

// TestEnvironmentToAPIURL verifies the URL-builder helper. The helper assumes
// its input has already been validated upstream (see envNamePattern), so it
// is exercised only with values the upstream gates accept. Injection-vector
// rejection is covered at the boundary in TestSetAuthToken.
func TestEnvironmentToAPIURL(t *testing.T) {
	tests := []struct {
		env         string
		expectedURL string
	}{
		{"production", "https://api.megaport.com/"},
		{"staging", "https://api-staging.megaport.com/"},
		{"development", "https://api-development.megaport.com/"},
		{"qa", "https://api-qa.megaport.com/"},
		{"uat", "https://api-uat.megaport.com/"},
		{"mpone-dev", "https://api-mpone-dev.megaport.com/"},
		{"prod", "https://api-prod.megaport.com/"}, // typo "prod" still passes envNamePattern; reviewer wanted coverage
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			assert.Equal(t, tt.expectedURL, environmentToAPIURL(tt.env))
		})
	}
}

// TestEnvironmentFromHostname verifies the hostname extractor against the two
// conventions Megaport uses: <app>.megaport.com (production) and
// <app>-<env>.megaport.com.
func TestEnvironmentFromHostname(t *testing.T) {
	tests := []struct {
		hostname    string
		expectedEnv string
		expectedOK  bool
	}{
		// Production: apex, www, and any <app>.megaport.com.
		{"megaport.com", "production", true},
		{"www.megaport.com", "production", true},
		{"portal.megaport.com", "production", true},
		{"api.megaport.com", "production", true},
		{"dashboard.megaport.com", "production", true},
		{"tools.megaport.com", "production", true},

		// <app>-<env>.megaport.com — any app, env after the first hyphen.
		{"portal-staging.megaport.com", "staging", true},
		{"portal-qa.megaport.com", "qa", true},
		{"portal-uat.megaport.com", "uat", true},
		{"api-staging.megaport.com", "staging", true},
		{"api-qa.megaport.com", "qa", true},
		{"dashboard-staging.megaport.com", "staging", true},
		{"tools-uat.megaport.com", "uat", true},

		// Multi-segment env names (split on the FIRST hyphen).
		{"api-mpone-dev.megaport.com", "mpone-dev", true},
		{"portal-mpone-dev.megaport.com", "mpone-dev", true},

		// Case- and whitespace-insensitive.
		{"PORTAL-QA.MEGAPORT.COM", "qa", true},
		{"  portal-staging.megaport.com  ", "staging", true},

		// Fail closed for localhost and private IPs.
		{"localhost", "", false},
		{"127.0.0.1", "", false},
		{"192.168.1.100", "", false},
		{"10.0.0.1", "", false},

		// Fail closed for non-.megaport.com hostnames.
		{"example.com", "", false},
		{"attacker.com", "", false},
		// Sanity check the suffix guard: looks similar but isn't .megaport.com.
		{"megaport.com.attacker.com", "", false},
		{"xmegaport.com", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.hostname, func(t *testing.T) {
			env, ok := environmentFromHostname(tt.hostname)
			assert.Equal(t, tt.expectedOK, ok)
			assert.Equal(t, tt.expectedEnv, env)
		})
	}
}

// TestRestrictEnvironmentName verifies the shim that collapses any
// environment name into the three canonical values that downstream
// MEGAPORT_ENVIRONMENT consumers understand.
func TestRestrictEnvironmentName(t *testing.T) {
	tests := []struct {
		env      string
		expected string
	}{
		{"production", "production"},
		{"staging", "staging"},
		{"development", "development"},
		{"qa", "development"},
		{"uat", "development"},
		{"mpone-dev", "development"},
		{"prod", "production"}, // alias accepted, matches normalizeEnvironment
		{"", "development"},    // shouldn't be called with empty, but defined behaviour
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			assert.Equal(t, tt.expected, restrictEnvironmentName(tt.env))
		})
	}
}

// TestRestrictEnvironmentNameStrict verifies the strict variant used by
// setAuthCredentials: known buckets (plus the "prod" alias) resolve, and
// anything else is reported as unrecognized rather than defaulted.
func TestRestrictEnvironmentNameStrict(t *testing.T) {
	tests := []struct {
		env      string
		expected string
		ok       bool
	}{
		{"production", "production", true},
		{"prod", "production", true},
		{"staging", "staging", true},
		{"development", "development", true},
		{"qa", "", false},
		{"uat", "", false},
		{"mpone-dev", "", false},
		{"Production", "", false}, // caller must normalize case before calling
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			bucket, ok := restrictEnvironmentNameStrict(tt.env)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.expected, bucket)
		})
	}
}

// TestSetAuthCredentials verifies that setAuthCredentials rejects any
// environment outside production/staging/development (normalizing case and
// whitespace first) instead of storing it verbatim, since login_wasm.go's
// credential path used to fall through to production for anything unrecognized.
func TestSetAuthCredentials(t *testing.T) {
	RegisterJSFunctions()
	defer js.Global().Get("clearAuthCredentials").Invoke()

	tests := []struct {
		name           string
		environment    string
		expectError    bool
		expectedBucket string
	}{
		{name: "production", environment: "production", expectedBucket: "production"},
		{name: "staging", environment: "staging", expectedBucket: "staging"},
		{name: "development", environment: "development", expectedBucket: "development"},
		{name: "prod alias", environment: "prod", expectedBucket: "production"},
		{name: "uppercase is normalized", environment: "STAGING", expectedBucket: "staging"},
		{name: "surrounding whitespace is trimmed", environment: "  production  ", expectedBucket: "production"},
		{name: "unknown environment is rejected, not routed to production", environment: "dev", expectError: true},
		{name: "unrecognized bucket-like value is rejected", environment: "qa", expectError: true},
		{name: "value that only partially matches a known bucket is rejected", environment: "production2", expectError: true},
		{name: "empty environment is rejected", environment: "", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			js.Global().Get("clearAuthCredentials").Invoke()

			setFunc := js.Global().Get("setAuthCredentials")
			result := setFunc.Invoke("test-access-key", "test-secret-key", tt.environment)

			success := result.Get("success").Bool()
			if tt.expectError {
				assert.False(t, success, "should reject an unrecognized environment")
				assert.NotEmpty(t, result.Get("error").String(), "error message should be set on failure")
				assert.Empty(t, os.Getenv("MEGAPORT_ENVIRONMENT"), "MEGAPORT_ENVIRONMENT must not be set on failure")
				assert.Empty(t, os.Getenv("MEGAPORT_ACCESS_KEY"), "credentials must not be stored when the environment is rejected")
			} else {
				assert.True(t, success, "should accept a recognized environment")
				assert.Equal(t, tt.expectedBucket, result.Get("environment").String())
				assert.Equal(t, tt.expectedBucket, os.Getenv("MEGAPORT_ENVIRONMENT"))
				assert.Equal(t, "test-access-key", os.Getenv("MEGAPORT_ACCESS_KEY"))

				credsGlobal := js.Global().Get("megaportCredentials")
				assert.False(t, credsGlobal.IsUndefined())
				assert.Equal(t, tt.expectedBucket, credsGlobal.Get("environment").String())
			}
		})
	}
}

// TestSetAuthCredentials_MissingArgs verifies setAuthCredentials rejects a
// call missing accessKey, secretKey, or environment without touching any
// stored credentials.
func TestSetAuthCredentials_MissingArgs(t *testing.T) {
	RegisterJSFunctions()
	defer js.Global().Get("clearAuthCredentials").Invoke()
	js.Global().Get("clearAuthCredentials").Invoke()

	setFunc := js.Global().Get("setAuthCredentials")
	result := setFunc.Invoke("test-access-key", "test-secret-key")

	assert.False(t, result.Get("success").Bool())
	assert.NotEmpty(t, result.Get("error").String())
	assert.Empty(t, os.Getenv("MEGAPORT_ENVIRONMENT"))
	assert.Empty(t, os.Getenv("MEGAPORT_ACCESS_KEY"))
}

// TestAuthMethodPriority verifies that token auth takes precedence over API key auth
func TestAuthMethodPriority(t *testing.T) {
	EnableDebugMode()
	RegisterJSFunctions()

	// Clear any existing auth state first
	js.Global().Get("clearAuthCredentials").Invoke()

	// First set API key auth
	js.Global().Get("setAuthCredentials").Invoke("api-key", "api-secret", "staging")
	authInfo := js.Global().Get("debugAuthInfo").Invoke()
	assert.Equal(t, "apikey", authInfo.Get("authMethod").String())

	// Now set token auth - should override
	js.Global().Get("setAuthToken").Invoke("test-token-12345", "portal.megaport.com")
	authInfo = js.Global().Get("debugAuthInfo").Invoke()
	assert.Equal(t, "token", authInfo.Get("authMethod").String())
	assert.Equal(t, "production", authInfo.Get("environment").String())

	// Clear and verify
	js.Global().Get("clearAuthCredentials").Invoke()
	authInfo = js.Global().Get("debugAuthInfo").Invoke()
	assert.Equal(t, "none", authInfo.Get("authMethod").String())
}

// TestSetAuthCredentials_ClearsStaleToken is a regression test: switching
// from token auth to API key auth via setAuthCredentials must actually take
// effect, not silently keep using a token injected by an earlier
// setAuthToken call (loginFunc checks the token path first).
func TestSetAuthCredentials_ClearsStaleToken(t *testing.T) {
	EnableDebugMode()
	RegisterJSFunctions()
	js.Global().Get("clearAuthCredentials").Invoke()

	js.Global().Get("setAuthToken").Invoke("test-token-12345", "portal.megaport.com")
	authInfo := js.Global().Get("debugAuthInfo").Invoke()
	assert.Equal(t, "token", authInfo.Get("authMethod").String())

	js.Global().Get("setAuthCredentials").Invoke("api-key", "api-secret", "staging")
	authInfo = js.Global().Get("debugAuthInfo").Invoke()
	assert.Equal(t, "apikey", authInfo.Get("authMethod").String())
	assert.True(t, js.Global().Get("megaportToken").IsUndefined(), "megaportToken global should be cleared")

	js.Global().Get("clearAuthCredentials").Invoke()
}

// TestSetAuthTokenMasking verifies token preview masking
func TestSetAuthTokenMasking(t *testing.T) {
	RegisterJSFunctions()

	testToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.test"
	js.Global().Get("setAuthToken").Invoke(testToken, "portal.megaport.com")

	authInfo := js.Global().Get("debugAuthInfo").Invoke()
	preview := authInfo.Get("accessTokenPreview").String()

	// Verify preview is masked
	assert.Contains(t, preview, "...")
	assert.NotEqual(t, testToken, preview, "full token should not be in preview")
	assert.True(t, len(preview) < len(testToken), "preview should be shorter than full token")
}

// TestBufferThreadSafety verifies all buffers are thread-safe
func TestBufferThreadSafety(t *testing.T) {
	ResetOutputBuffers()

	done := make(chan bool)
	iterations := 100

	// Test stdout buffer
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				bufferMutex.Lock()
				stdoutBuffer.WriteString("x")
				bufferMutex.Unlock()
			}
			done <- true
		}()
	}

	// Wait for completion
	for i := 0; i < 5; i++ {
		<-done
	}

	// Should have exactly 500 'x' characters
	result := stdoutBuffer.String()
	assert.Equal(t, 500, len(result))
}
