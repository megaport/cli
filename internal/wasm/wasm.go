//go:build js && wasm

package wasm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"syscall/js"
	"time"

	"github.com/spf13/cobra"
)

const apiURLProduction = "https://api.megaport.com/"
const apiURLNonProductionTemplate = "https://api-%s.megaport.com/"

// Character-class fragments shared between environment-name validation and
// the hostname patterns below. Keeping the env name in one place ensures the
// validator and the hostname extractor can't drift apart.
const (
	envNamePattern = `[a-z0-9-]+` // env: lowercase, digits, hyphens (e.g. "qa", "mpone-dev")
	appNamePattern = `[a-z0-9]+`  // app: lowercase, digits — no hyphens or dots
)

// validEnvironmentName restricts environment names to a safe character set.
// Names are interpolated into the API URL, so any character that could
// terminate the host portion (/, ., @, :, etc.) must be rejected to prevent
// hostname injection.
var validEnvironmentName = regexp.MustCompile(`^` + envNamePattern + `$`)

// environmentToAPIURL is the single source of truth for translating an
// environment name into a Megaport API base URL. "production" is the only
// special case; everything else becomes api-<env>.megaport.com so that new
// environments work without code changes.
//
// Callers must pass an already-validated environment name. The two upstream
// sources (the explicit override checked against validEnvironmentName, and
// the capture from appEnvHostnamePattern) both enforce envNamePattern, so the
// value is guaranteed safe to interpolate by the time it gets here.
func environmentToAPIURL(env string) string {
	if env == "production" {
		return apiURLProduction
	}
	return fmt.Sprintf(apiURLNonProductionTemplate, env)
}

// production hostname pattern includes an optional subdomain for an app name
// The app name is a single word (no hyphens)
// ie: portal.megaport.com, www.megaport.com and megaport.com are valid (last two being the website), but api-qa.megaport.com is not
var productionHostnamePattern = regexp.MustCompile(`^(?:` + appNamePattern + `\.)?megaport\.com$`)

// environment specific hostname must contain a single word app name and an environment name
// The environment name may contain hyphens, but the app name may not
// ie: portal-qa.megaport.com, api-mpone-dev.megaport.com are valid, but portal.megaport.com is not
// This pattern includes a capture group for the environment name so environmentFromHostname can extract it directly
var appEnvHostnamePattern = regexp.MustCompile(`^` + appNamePattern + `-(` + envNamePattern + `)\.megaport\.com$`)

// environmentFromHostname derives the environment name from a Megaport
// hostname. It accepts the two conventions Megaport uses for its apps:
//
//   - <app>.megaport.com         → production (e.g. portal.megaport.com)
//   - <app>-<env>.megaport.com   → <env>      (e.g. portal-qa.megaport.com,
//     dashboard-staging.megaport.com,
//     api-mpone-dev.megaport.com)
//
// Everything outside of megaport.com domains (localhost, private IPs, other non-Megaport domains)
// fails closed so callers must pass an explicit override
func environmentFromHostname(hostname string) (string, bool) {
	hostname = strings.ToLower(strings.TrimSpace(hostname))

	if productionHostnamePattern.MatchString(hostname) {
		return "production", true
	}
	if m := appEnvHostnamePattern.FindStringSubmatch(hostname); m != nil {
		return m[1], true
	}
	return "", false
}

// restrictEnvironmentNameStrict buckets an environment name into one of the
// three canonical values that downstream consumers of MEGAPORT_ENVIRONMENT
// understand ("production" / "staging" / "development"). "prod" is accepted
// as an alias for "production". Unlike restrictEnvironmentName, it reports
// whether the input matched a known bucket instead of silently defaulting,
// so callers that must reject unrecognized environments (e.g.
// setAuthCredentials) can fail closed.
func restrictEnvironmentNameStrict(env string) (string, bool) {
	switch env {
	case "production", "prod":
		return "production", true
	case "staging":
		return "staging", true
	case "development":
		return "development", true
	default:
		return "", false
	}
}

// restrictEnvironmentName collapses an environment name into one of the three
// canonical values that downstream consumers of MEGAPORT_ENVIRONMENT understand
// ("production" / "staging" / "development"). "prod" is accepted as an alias
// for "production"; anything else collapses to "development".
//
// Related: normalizeEnvironment in internal/commands/config/config_shared.go
// performs a similar canonicalisation for the native CLI path. The two
// functions agree on "production"/"prod" → "production" and "staging" →
// "staging", but their default cases differ: normalizeEnvironment defaults
// unknown values to "production" (so a malformed --env still hits the live
// API), while this function defaults to "development" (so the WASM portal
// fails into a safer bucket when the resolved env isn't canonical). The
// difference is intentional and the divergence is exercised by tests in both
// packages.
func restrictEnvironmentName(env string) string {
	if bucket, ok := restrictEnvironmentNameStrict(env); ok {
		return bucket
	}
	return "development"
}

var (
	// Use a single buffer pair with mutex protection
	stdoutBuffer bytes.Buffer
	stderrBuffer bytes.Buffer
	bufferMutex  sync.Mutex

	// Debug flag — accessed from multiple goroutines, so use atomic.Bool.
	debugMode atomic.Bool

	// terminalWidthCols holds the host-provided terminal width in columns, set
	// via the JS-exposed setTerminalWidth function and read by the table
	// renderer on a goroutine, hence atomic. Zero means "unset".
	terminalWidthCols atomic.Int64

	// outputStateReset is called by ResetOutputBuffers to clear --fields and
	// --query flag state between WASM invocations. Registered from main_wasm.go
	// to break the import cycle between this package and internal/base/output.
	outputStateReset func() = func() {}
)

// maskSensitiveValue masks a sensitive value for logging/display purposes
// Shows first and last few characters with "..." in between
func maskSensitiveValue(value string) string {
	if value == "" {
		return ""
	}
	if len(value) > 8 {
		return value[:4] + "..." + value[len(value)-4:]
	} else if len(value) > 4 {
		return value[:2] + "..." + value[len(value)-2:]
	}
	return "****"
}

// WasmOutputBuffer is used to directly capture output from Cobra commands
var WasmOutputBuffer = &DirectOutputBuffer{
	buffer: &bytes.Buffer{},
}

// DirectOutputBuffer provides a buffer that can be used directly by Cobra commands
type DirectOutputBuffer struct {
	buffer *bytes.Buffer
	mutex  sync.Mutex
}

func (d *DirectOutputBuffer) Write(p []byte) (n int, err error) {
	debug := debugMode.Load()
	stream := hasOutputHandler() && !outputHandlerFailed.Load()

	// Sanitize before buffering: the host writes this buffer's contents (both
	// the streamed chunk below and the full buffer read back later via
	// String()) straight to xterm without escaping, so a crafted resource
	// name or API response field carrying control bytes must be neutralized
	// here rather than at each read site. See SanitizeTerminalOutput. This
	// makes the []byte->string conversion unconditional, unlike before, since
	// the buffer itself must now hold sanitized text.
	chunk := SanitizeTerminalOutput(string(p))

	// Buffer the write under the lock in a closure so the mutex is released
	// (even on a panic in the debug-logging JS calls) before we push to the
	// live handler. Pushing outside the lock prevents a re-entrant write from
	// the JS callback deadlocking on d.mutex.
	func() {
		d.mutex.Lock()
		defer d.mutex.Unlock()

		if debug {
			js.Global().Get("console").Call("log", fmt.Sprintf("📝 BUFFER WRITE [%d bytes]:", len(p)))
			lines := strings.Split(chunk, "\n")
			for _, line := range lines {
				if line != "" {
					js.Global().Get("console").Call("log", fmt.Sprintf("  │ %s", line))
				}
			}
			js.Global().Get("console").Call("debug", chunk)
		}

		_, err = d.buffer.WriteString(chunk)
	}()

	// Stream the chunk live to a registered output handler, if any.
	if stream {
		pushOutputChunk(chunk)
	}
	return len(p), err
}

// TraceCommandExecution logs the command hierarchy and available subcommands when debugMode is on.
func TraceCommandExecution(cmd *cobra.Command, args []string) {
	if !debugMode.Load() {
		return
	}
	js.Global().Get("console").Call("group", "⚡ COMMAND TRAVERSAL")

	currentCmd := cmd
	var cmdPath []string
	for currentCmd != nil {
		cmdPath = append([]string{currentCmd.Name()}, cmdPath...)
		currentCmd = currentCmd.Parent()
	}

	js.Global().Get("console").Call("log", fmt.Sprintf("Command path: %s", strings.Join(cmdPath, " → ")))
	js.Global().Get("console").Call("log", fmt.Sprintf("Args: %v", args))

	if len(cmd.Commands()) > 0 {
		var subNames []string
		for _, sub := range cmd.Commands() {
			subNames = append(subNames, sub.Name())
		}
		js.Global().Get("console").Call("log", fmt.Sprintf("Available subcommands: %v", subNames))
	}

	js.Global().Get("console").Call("groupEnd")
}

// TraceCommand logs the command string, parsed args, and auth status when debugMode is on.
func TraceCommand(command string, args []string) {
	if !debugMode.Load() {
		return
	}
	js.Global().Get("console").Call("group", fmt.Sprintf("🔍 COMMAND: %s", command))
	js.Global().Get("console").Call("log", fmt.Sprintf("Full command: %s", command))
	js.Global().Get("console").Call("log", fmt.Sprintf("Parsed args: %v", args))

	accessKey := os.Getenv("MEGAPORT_ACCESS_KEY")
	secretKey := os.Getenv("MEGAPORT_SECRET_KEY")
	env := os.Getenv("MEGAPORT_ENVIRONMENT")

	js.Global().Get("console").Call("log", fmt.Sprintf("Environment: %s", env))
	js.Global().Get("console").Call("log", fmt.Sprintf("Auth configured: %v", accessKey != "" && secretKey != ""))
	js.Global().Get("console").Call("groupEnd")
}

func (d *DirectOutputBuffer) String() string {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.buffer.String()
}

func (d *DirectOutputBuffer) Reset() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.buffer.Reset()
}

// EnableDebugMode turns on additional debugging. This should be enabled only
// when the host page explicitly opts in via window.wasmDebugMode = true
// (see main_wasm.go) for local debugging or other controlled, non-public
// environments. When active, every buffer write is logged to the browser
// console — this may expose sensitive data (API keys, tokens, command
// output) to anyone with DevTools open, so it must not be enabled for
// public or hosted production deployments.
func EnableDebugMode() {
	debugMode.Store(true)
	js.Global().Set("wasmDebug", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return debugMode.Load()
	}))
}

func debugAuthInfo(this js.Value, args []js.Value) interface{} {
	accessKey := os.Getenv("MEGAPORT_ACCESS_KEY")
	secretKey := os.Getenv("MEGAPORT_SECRET_KEY")
	accessToken := os.Getenv("MEGAPORT_ACCESS_TOKEN")
	env := os.Getenv("MEGAPORT_ENVIRONMENT")
	apiURL := os.Getenv("MEGAPORT_API_URL")

	return map[string]interface{}{
		"accessKeySet":       accessKey != "",
		"accessKeyPreview":   maskSensitiveValue(accessKey),
		"secretKeySet":       secretKey != "",
		"secretKeyPreview":   maskSensitiveValue(secretKey),
		"accessTokenSet":     accessToken != "",
		"accessTokenPreview": maskSensitiveValue(accessToken),
		"environment":        env,
		"apiURL":             apiURL,
		"authMethod":         getAuthMethod(accessKey, secretKey, accessToken),
	}
}

// maxTerminalWidthCols bounds the stored width well above any real terminal
// (hundreds of columns at most), so a bogus or hostile caller can't push an
// astronomical value into the column-width math downstream.
const maxTerminalWidthCols = 2000

// SetTerminalWidth stores the host terminal width in columns for the table
// renderer to size columns against. Zero or negative resets to "unset" so
// callers fall back to a fixed layout; values above maxTerminalWidthCols are
// clamped rather than rejected.
func SetTerminalWidth(cols int) {
	switch {
	case cols < 0:
		cols = 0
	case cols > maxTerminalWidthCols:
		cols = maxTerminalWidthCols
	}
	terminalWidthCols.Store(int64(cols))
}

// TerminalWidth returns the last width set via SetTerminalWidth, or 0 if
// none has been set.
func TerminalWidth() int {
	return int(terminalWidthCols.Load())
}

// setTerminalWidth is the JS-exposed function the host page calls (on init
// and on resize, via the xterm fit addon) so table rendering can size
// columns to the real viewport instead of a fixed layout.
func setTerminalWidth(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 || args[0].Type() != js.TypeNumber {
		return map[string]interface{}{
			"success": false,
			"error":   "setTerminalWidth requires a numeric cols argument",
		}
	}
	cols := args[0].Float()
	if math.IsNaN(cols) || math.IsInf(cols, 0) {
		return map[string]interface{}{
			"success": false,
			"error":   "setTerminalWidth requires a finite cols argument",
		}
	}
	// Clamp in float space before converting: int(float64) is
	// implementation-defined when the value is out of int range, so a huge
	// finite input could otherwise slip past SetTerminalWidth's cap.
	switch {
	case cols < 0:
		cols = 0
	case cols > float64(maxTerminalWidthCols):
		cols = float64(maxTerminalWidthCols)
	}
	SetTerminalWidth(int(cols))
	return map[string]interface{}{"success": true}
}

// getAuthMethod returns the authentication method being used
func getAuthMethod(accessKey, secretKey, accessToken string) string {
	if accessToken != "" {
		return "token"
	}
	if accessKey != "" && secretKey != "" {
		return "apikey"
	}
	return "none"
}

// RegisterJSFunctions registers Go functions with JavaScript
func RegisterJSFunctions() {
	// Export file system operations
	js.Global().Set("readConfigFile", js.FuncOf(readConfigFile))
	js.Global().Set("writeConfigFile", js.FuncOf(writeConfigFile))

	// debugAuthInfo reveals which auth method and environment are configured.
	// Register it only when debug mode is enabled so that production deployments
	// do not expose an information-disclosure endpoint to arbitrary page scripts.
	if debugMode.Load() {
		js.Global().Set("debugAuthInfo", js.FuncOf(debugAuthInfo))
	}

	// Export storage operations
	js.Global().Set("saveToLocalStorage", js.FuncOf(saveToLocalStorage))
	js.Global().Set("loadFromLocalStorage", js.FuncOf(loadFromLocalStorage))

	// Export the host terminal width so table rendering can size columns to
	// the real viewport instead of a fixed layout.
	js.Global().Set("setTerminalWidth", js.FuncOf(setTerminalWidth))

	// Export auth operations (secure, in-memory only)
	js.Global().Set("setAuthCredentials", js.FuncOf(setAuthCredentials))
	js.Global().Set("setAuthToken", js.FuncOf(setAuthToken))
	js.Global().Set("clearAuthCredentials", js.FuncOf(clearAuthCredentials))

	// Debug functions
	js.Global().Set("resetWasmOutput", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ResetOutputBuffers()
		return true
	}))

	js.Global().Set("getWasmOutput", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return GetCapturedOutput()
	}))

	InstallCommandHooks()

	// Add a debug mode toggle function.
	// NOTE: The Load+Store is not a single atomic operation, but WASM is
	// single-threaded so concurrent toggles cannot interleave. The atomic
	// type is used to satisfy the Go race detector, not for true concurrency.
	js.Global().Set("toggleWasmDebug", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		newVal := !debugMode.Load()
		debugMode.Store(newVal)
		return newVal
	}))

	// Add a function to dump the full buffer contents for debugging
	js.Global().Set("dumpBuffers", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		stdoutContent := stdoutBuffer.String()
		stderrContent := stderrBuffer.String()
		directContent := WasmOutputBuffer.String()

		return map[string]interface{}{
			"stdout": stdoutContent,
			"stderr": stderrContent,
			"direct": directContent,
		}
	}))
}

// RegisterOutputStateReset registers the function that resets --fields and
// --query flag state between WASM invocations. Called from main_wasm.go to
// avoid an import cycle between this package and internal/base/output.
// Protected by bufferMutex to prevent a data race with ResetOutputBuffers.
func RegisterOutputStateReset(fn func()) {
	bufferMutex.Lock()
	defer bufferMutex.Unlock()
	outputStateReset = fn
}

// ResetOutputBuffers clears the output buffers and resets output package state.
// Must be called between WASM command invocations to prevent flag state bleed.
func ResetOutputBuffers() {
	// Reset Go-side buffers under the lock.
	bufferMutex.Lock()
	stdoutBuffer.Reset()
	stderrBuffer.Reset()
	WasmOutputBuffer.Reset()
	bufferMutex.Unlock()

	// A new command starts here: clear the streamed-since-reset marker and give
	// a previously-failed handler a fresh chance to stream.
	outputStreamed.Store(false)
	outputHandlerFailed.Store(false)

	// Reset output package flag state (--fields, --query, format, verbosity)
	// OUTSIDE bufferMutex: the callback acquires its own locks in the output
	// package, so holding bufferMutex here would be a lock-ordering violation
	// if any output-package path ever tried to acquire bufferMutex too.
	outputStateReset()

	// Clear all JS-side output globals. These are safe to touch without the
	// Go mutex because WASM runs on a single OS thread.
	js.Global().Delete("wasmJSONOutput")
	js.Global().Delete("wasmCSVOutput")
	js.Global().Delete("wasmTableOutput")
	js.Global().Delete("wasmXMLOutput")

	if debugMode.Load() {
		js.Global().Get("console").Call("log", "Output buffers reset (including all structured output globals)")
	}
}

// jsStringGlobal returns a JS global as a Go string, or "" if unset/null.
func jsStringGlobal(name string) string {
	if v := js.Global().Get(name); !v.IsUndefined() && !v.IsNull() {
		return v.String()
	}
	return ""
}

// structuredOutputs holds the four structured document buffers that WASM
// commands publish via JS globals. Unlike the direct/narrative buffer, these
// are complete documents delivered once at completion and are never streamed.
type structuredOutputs struct {
	json, csv, xml, table string
}

func readStructuredOutputs() structuredOutputs {
	return structuredOutputs{
		json:  jsStringGlobal("wasmJSONOutput"),
		csv:   jsStringGlobal("wasmCSVOutput"),
		xml:   jsStringGlobal("wasmXMLOutput"),
		table: jsStringGlobal("wasmTableOutput"),
	}
}

// pick returns the highest-priority structured document (JSON > CSV > XML >
// table) and a label for it, or ("", "") when none is set.
func (s structuredOutputs) pick() (string, string) {
	switch {
	case s.json != "":
		return s.json, "JSON buffer"
	case s.csv != "":
		return s.csv, "CSV buffer"
	case s.xml != "":
		return s.xml, "XML buffer"
	case s.table != "":
		return s.table, "table buffer"
	default:
		return "", ""
	}
}

// GetCapturedOutput returns all captured output. Go-side buffers are read under the
// lock; JS interop happens outside to avoid holding bufferMutex across JS callbacks.
func GetCapturedOutput() string {
	// Read Go-side buffers under the lock, then release before any JS calls.
	bufferMutex.Lock()
	out := stdoutBuffer.String()
	errStr := stderrBuffer.String()
	bufferMutex.Unlock()

	// WasmOutputBuffer has its own mutex; call outside bufferMutex to avoid lock ordering issues.
	direct := WasmOutputBuffer.String()

	// All JS interop happens outside the lock.
	structured := readStructuredOutputs()

	// Priority order: JSON > CSV > XML > table > direct > stdout/stderr combined.
	// JSON/CSV/XML are returned as a clean data-only stream (native routes these to
	// stdout with status messages on stderr). In table mode we prepend the direct
	// buffer (PrintInfo/PrintWarning/PrintSuccess/PrintPlain) so status lines aren't
	// shadowed by the table, matching native's separate stderr/stdout streams.
	finalOutput, outputSource := structured.pick()
	switch {
	case outputSource == "table buffer":
		finalOutput = direct + finalOutput
	case finalOutput == "" && direct != "":
		finalOutput, outputSource = direct, "direct buffer"
	case finalOutput == "":
		finalOutput, outputSource = out+errStr, "combined stdout/stderr"
	}

	if debugMode.Load() {
		js.Global().Get("console").Call("group", "📤 OUTPUT CAPTURE RESULTS")
		js.Global().Get("console").Call("log", fmt.Sprintf("stdout buffer: [%d bytes]", len(out)))
		js.Global().Get("console").Call("log", fmt.Sprintf("stderr buffer: [%d bytes]", len(errStr)))
		js.Global().Get("console").Call("log", fmt.Sprintf("direct buffer: [%d bytes]", len(direct)))
		js.Global().Get("console").Call("log", fmt.Sprintf("JSON buffer: [%d bytes]", len(structured.json)))
		js.Global().Get("console").Call("log", fmt.Sprintf("CSV buffer: [%d bytes]", len(structured.csv)))
		js.Global().Get("console").Call("log", fmt.Sprintf("XML buffer: [%d bytes]", len(structured.xml)))
		js.Global().Get("console").Call("log", fmt.Sprintf("table buffer: [%d bytes]", len(structured.table)))
		js.Global().Get("console").Call("log", fmt.Sprintf("Using %s for output (%d bytes)", outputSource, len(finalOutput)))
		js.Global().Get("console").Call("groupEnd")
	}

	return finalOutput
}

// GetCompletionOutput returns the output an async command should hand back once
// it finishes, honoring the live-streaming contract.
//
// Narrative is suppressed here only when a chunk actually streamed to the host
// (outputStreamed) and the handler has not since failed (outputHandlerFailed),
// since streamed chunks are already in the terminal and returning them again
// would double-render. In that case only the structured document
// (JSON/CSV/XML/table) is returned; it is "" when the command produced only
// streamed narrative.
//
// Otherwise (no handler, a handler that delivered nothing, or one that streamed
// some chunks and then threw) this falls back to GetCapturedOutput so the host
// receives the full output rather than silently losing the undelivered tail.
func GetCompletionOutput() string {
	if hasOutputHandler() && outputStreamed.Load() && !outputHandlerFailed.Load() {
		structured, _ := readStructuredOutputs().pick()
		return structured
	}
	return GetCapturedOutput()
}

// SetupIO redirects stdout/stderr to our buffers using WasmOutputBuffer
// In WASM, os.Pipe() is not implemented, so we don't need complex IO redirection
// The output is already captured through WasmOutputBuffer which Cobra commands use
func SetupIO() {
	// Note: In WASM, we don't need to redirect os.Stdout/Stderr because:
	// 1. Cobra commands are configured to write to WasmOutputBuffer directly
	// 2. Table output writes to WasmTableWriter and sets wasmTableOutput global
	// 3. Any fmt.Print() calls go to the console via wasm_exec.js

	// The output capture happens through:
	// - Direct writes to WasmOutputBuffer from Cobra commands
	// - Table output via WasmTableWriter -> wasmTableOutput global
	// - These are collected in GetCapturedOutput()

	if debugMode.Load() {
		js.Global().Get("console").Call("log", "✅ WASM IO setup complete (using WasmOutputBuffer)")
	}
}

// customWriter provides a custom io.Writer implementation that handles WASM quirks
type customWriter struct {
	writer io.Writer
}

func (cw *customWriter) Write(p []byte) (n int, err error) {
	bufferMutex.Lock()
	defer bufferMutex.Unlock()
	return cw.writer.Write(p)
}

// CaptureOutput runs a function and captures its stdout/stderr output.
// If os.Pipe() is unavailable (as it is in the js/wasm target), fn is executed
// normally and the direct buffer is used for output capture instead.
func CaptureOutput(fn func()) string {
	// Reset buffers before capture
	ResetOutputBuffers()

	// Create pipes for output capture. On js/wasm os.Pipe() is not implemented
	// and will return an error; in that case fall through to the direct buffer.
	rOut, wOut, errOut := os.Pipe()
	rErr, wErr, errErr := os.Pipe()
	if errOut != nil || errErr != nil {
		// Close any pipe endpoints that were successfully created before the
		// failure to avoid leaking file descriptors.
		if rOut != nil {
			_ = rOut.Close()
		}
		if wOut != nil {
			_ = wOut.Close()
		}
		if rErr != nil {
			_ = rErr.Close()
		}
		if wErr != nil {
			_ = wErr.Close()
		}
		fn()
		return WasmOutputBuffer.String()
	}

	// Save original stdout/stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Replace with pipes
	os.Stdout = wOut
	os.Stderr = wErr

	// Execute function in a controlled environment
	done := make(chan bool)
	go func() {
		fn()
		// Ensure all output is flushed
		wOut.Close()
		wErr.Close()
		done <- true
	}()

	// Read from pipes in parallel
	var stdoutStr, stderrStr string
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rOut)
		stdoutStr = buf.String()
		wg.Done()
	}()

	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rErr)
		stderrStr = buf.String()
		wg.Done()
	}()

	// Wait for the command to finish
	<-done

	// Wait for all reads to complete
	wg.Wait()

	// Restore original stdout/stderr
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Combine captured outputs
	combinedOutput := stdoutStr + stderrStr

	// Also check our direct buffer which may have captured output
	directOutput := WasmOutputBuffer.String()

	// Store in our main buffers for later access
	bufferMutex.Lock()
	stdoutBuffer.WriteString(stdoutStr)
	stderrBuffer.WriteString(stderrStr)
	bufferMutex.Unlock()

	// Return all output - prefer direct capture if available
	if directOutput != "" {
		return directOutput
	}
	return combinedOutput
}

func SplitArgs(cmd string) []string {
	// Trim any leading/trailing whitespace
	cmd = strings.TrimSpace(cmd)

	if debugMode.Load() {
		js.Global().Get("console").Call("log", fmt.Sprintf("SplitArgs: parsing command: %q", cmd))
	}

	var args []string
	inQuote := false
	var currentArg strings.Builder
	quoteChar := rune(0) // Track which quote char opened the current quote
	wasQuoted := false   // Track if current arg was quoted (to preserve empty strings)

	for _, r := range cmd {
		isQuoteChar := r == '"' || r == '\''

		switch {
		case isQuoteChar && inQuote && r == quoteChar:
			// Closing quote that matches opening quote
			inQuote = false
			quoteChar = rune(0)
			wasQuoted = true // Mark that this arg was quoted
		case isQuoteChar && !inQuote:
			// Opening quote
			inQuote = true
			quoteChar = r
			wasQuoted = true // Mark that this arg was quoted
		case r == ' ' && !inQuote:
			// End of argument - include empty strings if they were quoted
			if currentArg.Len() > 0 || wasQuoted {
				args = append(args, currentArg.String())
				currentArg.Reset()
				wasQuoted = false
			}
		default:
			currentArg.WriteRune(r)
		}
	}

	// Don't forget the last argument - include empty strings if they were quoted
	if currentArg.Len() > 0 || wasQuoted {
		args = append(args, currentArg.String())
	}

	// Remove a leading program-name token if the user included it.
	// Only the first token is checked: a later argument, flag value, or
	// positional equal to "megaport" is a real value and must be kept.
	cleanedArgs := args
	if len(cleanedArgs) > 0 && (cleanedArgs[0] == "megaport-cli" || cleanedArgs[0] == "./megaport-cli" || cleanedArgs[0] == "megaport") {
		cleanedArgs = cleanedArgs[1:]
	}

	if debugMode.Load() {
		js.Global().Get("console").Call("log", fmt.Sprintf("SplitArgs: original=%v cleaned=%v", args, cleanedArgs))
	}

	return cleanedArgs
}

// isValidConfigFilename reports whether filename is safe to use as a localStorage key
// suffix. Only alphanumeric characters, dots, hyphens, and underscores are permitted,
// and the length is limited to prevent denial-of-service via enormous keys.
func isValidConfigFilename(filename string) bool {
	if filename == "" || len(filename) > 64 {
		return false
	}
	for _, c := range filename {
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		case c == '.' || c == '_' || c == '-':
		default:
			return false
		}
	}
	return true
}

func readConfigFile(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"error": "Filename required",
		}
	}

	filename := args[0].String()

	if !isValidConfigFilename(filename) {
		return map[string]interface{}{
			"error": "Invalid filename",
		}
	}

	// Get from localStorage
	content := js.Global().Get("localStorage").Call("getItem", "megaport_fs_"+filename)

	if content.IsNull() {
		// If config file doesn't exist, create an empty one with default structure
		if filename == "config.json" {
			defaultConfig := map[string]interface{}{
				"profiles":      map[string]interface{}{},
				"activeProfile": "",
				"defaults":      map[string]interface{}{},
			}
			defaultConfigBytes, _ := json.Marshal(defaultConfig)
			return map[string]interface{}{
				"content": string(defaultConfigBytes),
			}
		}
		return map[string]interface{}{
			"error": "File not found: " + filename,
		}
	}

	return map[string]interface{}{
		"content": content.String(),
	}
}

func writeConfigFile(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return map[string]interface{}{
			"error": "Filename and content required",
		}
	}

	filename := args[0].String()
	content := args[1].String()

	if !isValidConfigFilename(filename) {
		return map[string]interface{}{
			"error": "Invalid filename",
		}
	}

	// Save to localStorage
	js.Global().Get("localStorage").Call("setItem", "megaport_fs_"+filename, content)

	return map[string]interface{}{
		"success": true,
	}
}

// localStorage wrappers
func saveToLocalStorage(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return false
	}

	key := args[0].String()
	if !isValidConfigFilename(key) {
		return false
	}

	value := args[1].String()

	js.Global().Get("localStorage").Call("setItem", key, value)
	return true
}

func loadFromLocalStorage(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return ""
	}

	key := args[0].String()
	if !isValidConfigFilename(key) {
		return ""
	}
	return js.Global().Get("localStorage").Call("getItem", key)
}

// setAuthCredentials securely sets authentication credentials in-memory
// This is the recommended way to set credentials in WASM environment
// Credentials are stored in:
// 1. Go environment variables (for os.Getenv calls)
// 2. JavaScript global object (for direct access)
// This avoids localStorage which is vulnerable to XSS attacks
//
// The environment argument is normalized (lowercased, trimmed) and must
// resolve to one of "production"/"staging"/"development" (see
// restrictEnvironmentNameStrict); anything else is rejected rather than
// stored, since an unrecognized value previously fell through to
// login_wasm.go's default case and routed credentials to production.
func setAuthCredentials(this js.Value, args []js.Value) interface{} {
	if len(args) < 3 {
		return map[string]interface{}{
			"success": false,
			"error":   "accessKey, secretKey, and environment required",
		}
	}

	accessKey := args[0].String()
	secretKey := args[1].String()
	environment := strings.ToLower(strings.TrimSpace(args[2].String()))

	bucket, ok := restrictEnvironmentNameStrict(environment)
	if !ok {
		return map[string]interface{}{
			"success": false,
			"error":   `environment must be one of "production", "staging", or "development"`,
		}
	}

	// Store credentials only in Go-side environment variables.
	// Do NOT mirror them into JS globals — any script on the page can read those.
	os.Setenv("MEGAPORT_ACCESS_KEY", accessKey)
	os.Setenv("MEGAPORT_SECRET_KEY", secretKey)
	os.Setenv("MEGAPORT_ENVIRONMENT", bucket)

	// Clear any token left over from a prior setAuthToken call: loginFunc
	// checks the token path first, so a stale token would otherwise keep
	// silently overriding these credentials.
	os.Unsetenv("MEGAPORT_ACCESS_TOKEN")
	os.Unsetenv("MEGAPORT_API_URL")
	js.Global().Delete("megaportToken")

	// Expose only the non-secret environment name so the UI can reflect it.
	credentialsObj := js.Global().Get("Object").New()
	credentialsObj.Set("environment", bucket)
	js.Global().Set("megaportCredentials", credentialsObj)

	js.Global().Get("console").Call("log", "🔐 Credentials set (in-memory only)")

	return map[string]interface{}{
		"success":     true,
		"environment": bucket,
	}
}

// clearAuthCredentials removes authentication credentials from memory
func clearAuthCredentials(this js.Value, args []js.Value) interface{} {
	os.Unsetenv("MEGAPORT_ACCESS_KEY")
	os.Unsetenv("MEGAPORT_SECRET_KEY")
	os.Unsetenv("MEGAPORT_ENVIRONMENT")
	os.Unsetenv("MEGAPORT_ACCESS_TOKEN")
	os.Unsetenv("MEGAPORT_API_URL")

	js.Global().Delete("megaportCredentials")
	js.Global().Delete("megaportToken")

	js.Global().Get("console").Call("log", "🔓 Credentials cleared from memory")

	return map[string]interface{}{
		"success": true,
	}
}

// ParseExpiry converts a JS value into a time.Time, accepting either an
// epoch-millisecond number or an RFC3339 string. It returns the zero Time
// (no expiry known) when v is absent, null, undefined, zero, non-finite, or
// unparseable — callers should fall back to their own default in that case.
//
// The number form must be epoch milliseconds, not epoch seconds: a JWT `exp`
// claim (conventionally epoch seconds) passed through unconverted will parse
// as a timestamp in 1970 rather than fail loudly, since any positive number
// is otherwise accepted. Multiply a seconds-based value by 1000 first.
func ParseExpiry(v js.Value) time.Time {
	switch v.Type() {
	case js.TypeNumber:
		ms := v.Float()
		if ms <= 0 || ms > math.MaxInt64 || math.IsNaN(ms) || math.IsInf(ms, 0) {
			return time.Time{}
		}
		return time.UnixMilli(int64(ms)).UTC()
	case js.TypeString:
		s := strings.TrimSpace(v.String())
		if s == "" {
			return time.Time{}
		}
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}
		}
		return t.UTC()
	default:
		if v.Type() != js.TypeUndefined && v.Type() != js.TypeNull {
			js.Global().Get("console").Call("warn", fmt.Sprintf(
				"setAuthToken: expiry has unexpected type %q, ignoring (expected epoch-ms number or RFC3339 string)", v.Type().String(),
			))
		}
		return time.Time{}
	}
}

// setAuthToken sets an external access token for authentication.
// This bypasses the OAuth flow and uses the token directly from the portal session.
// Use this when the portal already has a valid login token stored in the browser.
//
// The environment is resolved with the following precedence:
//  1. The explicit override passed as the 3rd argument (must match [a-z0-9-]+).
//  2. The environment derived from the hostname via environmentFromHostname.
//     setAuthToken(token, hostname)                       // environment derived from hostname
//     setAuthToken(token, hostname, environment)          // explicit environment override
//     setAuthToken(token, hostname, environment, expiry)  // also pass the token's expiry
//
// expiry is optional and accepts either an epoch-millisecond number or an
// RFC3339 string (see ParseExpiry). Pass null/undefined for environment to
// skip it while still supplying expiry. When omitted, the token is treated as
// non-expiring (the SDK's zero-expiry fallback) — the host must call
// setAuthToken again on a session-expired signal regardless.
func setAuthToken(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return map[string]interface{}{
			"success": false,
			"error":   "token and hostname required",
		}
	}

	token := args[0].String()
	hostname := args[1].String()

	if token == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "token cannot be empty",
		}
	}

	if hostname == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "hostname cannot be empty",
		}
	}

	// Optional explicit environment override. Restricted to [a-z0-9-] to prevent
	// hostname injection when interpolated into the API URL — values containing
	// '/', '.', '@', or ':' could redirect the access token to a third-party host.
	// Only treat the third argument as an override when it's an actual string.
	// JS callers commonly pass `undefined` or `null` when they don't want an
	// override; Value.String() on those returns "<undefined>"/"<null>" which
	// would then fail the regex below and surface a misleading error.
	var explicitEnv string
	if len(args) >= 3 && args[2].Type() == js.TypeString {
		explicitEnv = strings.ToLower(strings.TrimSpace(args[2].String()))
	}
	if explicitEnv != "" && !validEnvironmentName.MatchString(explicitEnv) {
		return map[string]interface{}{
			"success": false,
			"error":   "environment must contain only lowercase letters, digits, and hyphens",
		}
	}

	// Optional 4th argument: the token's real expiry, as an epoch-ms number or
	// an RFC3339 string. Absent/invalid resolves to the zero Time, which the
	// caller falls back to a non-expiring token for (see WithAccessToken).
	var expiry time.Time
	if len(args) >= 4 {
		expiry = ParseExpiry(args[3])
	}

	// Resolve the environment. The override wins; otherwise we derive it from
	// the hostname. If neither yields a value we fail closed and tell the caller
	// to pass an explicit environment as the third argument — never default to
	// production silently.
	derivedEnv, derivedOK := environmentFromHostname(hostname)
	var environment string
	if explicitEnv != "" {
		environment = explicitEnv
	} else if derivedOK {
		environment = derivedEnv
	} else {
		return map[string]interface{}{
			"success": false,
			"error": fmt.Sprintf(
				"could not determine environment from hostname %q. Pass an explicit environment as the third argument, e.g. setAuthToken(token, hostname, \"production\")",
				hostname,
			),
		}
	}

	// Build the API URL from the resolved environment via the single helper, so
	// the override path and the derived path can't drift apart. The env is
	// already known-valid here (override passed validEnvironmentName above; the
	// derived value came from appEnvHostnamePattern's env capture).
	apiURL := environmentToAPIURL(environment)
	// A mismatch is detected when the caller provides an explicit environment that disagrees with the hostname-derived environment.
	// This is a strong signal that something is misconfigured: either the caller is passing the wrong override, or the hostname doesn't match the expected pattern for that environment.
	// In either case we want to flag this clearly in the console logs so it's obvious to the user and can be easily debugged.
	mismatch := explicitEnv != "" && derivedOK && explicitEnv != derivedEnv

	// Build a console-friendly description of the resolved environment, annotating
	// where it came from and flagging override-vs-hostname mismatches.
	var environmentLog string
	if explicitEnv == "" {
		environmentLog = fmt.Sprintf("%s (derived from hostname)", environment)
	} else if mismatch {
		environmentLog = fmt.Sprintf("%s (mismatch with hostname-derived %q)", environment, derivedEnv)
	} else if derivedOK {
		environmentLog = environment + " (override matches hostname)"
	} else {
		environmentLog = environment
	}

	// At-a-glance status marker for the browser console:
	// 🔴 override disagrees with hostname, 🟠 talking to production, 🟢 otherwise.
	var environmentRiskStatus string
	if mismatch {
		environmentRiskStatus = "🔴"
	} else if environment == "production" {
		environmentRiskStatus = "🟠"
	} else {
		environmentRiskStatus = "🟢"
	}

	js.Global().Get("console").Call("log", fmt.Sprintf("🌐 Hostname: %s", hostname))
	js.Global().Get("console").Call("log", fmt.Sprintf("%s Environment: %s", environmentRiskStatus, environmentLog))
	js.Global().Get("console").Call("log", fmt.Sprintf("🔗 API URL: %s", apiURL))
	if !expiry.IsZero() {
		js.Global().Get("console").Call("log", fmt.Sprintf("⏰ Token expiry: %s", expiry.Format(time.RFC3339)))
	} else {
		js.Global().Get("console").Call("log", "⏰ Token expiry: none provided (treated as non-expiring)")
	}

	// Store token, environment bucket, and API URL for downstream Go code.
	// MEGAPORT_ENVIRONMENT is bucketed to one of "production"/"staging"/"development"
	// because the downstream restrictEnvironmentName consumers (login.go, login_wasm.go,
	// auth_actions.go) only understand those three values and would silently coerce
	// anything else to "production". MEGAPORT_API_URL still carries the real per-env
	// URL so the access token routes to the correct backend.
	os.Setenv("MEGAPORT_ACCESS_TOKEN", token)
	os.Setenv("MEGAPORT_ENVIRONMENT", restrictEnvironmentName(environment))
	os.Setenv("MEGAPORT_API_URL", apiURL)

	// Clear any existing API key credentials to avoid confusion
	os.Setenv("MEGAPORT_ACCESS_KEY", "")
	os.Setenv("MEGAPORT_SECRET_KEY", "")

	// Expose only metadata (not the raw token) in the JS global so the UI can
	// reflect the current session state without re-exposing the bearer token.
	// expiry is stored as epoch-ms (0 when unknown) so it round-trips through
	// ParseExpiry the same way a fresh call's 4th argument would.
	tokenObj := js.Global().Get("Object").New()
	tokenObj.Set("environment", environment)
	tokenObj.Set("hostname", hostname)
	tokenObj.Set("apiURL", apiURL)
	if !expiry.IsZero() {
		tokenObj.Set("expiry", float64(expiry.UnixMilli()))
	} else {
		tokenObj.Set("expiry", 0)
	}
	js.Global().Set("megaportToken", tokenObj)

	// Clear any existing credentials object
	js.Global().Delete("megaportCredentials")

	js.Global().Get("console").Call("log", "🔐 External token set (in-memory only, bypassing OAuth flow)")

	result := map[string]interface{}{
		"success":     true,
		"environment": environment,
		"hostname":    hostname,
		"apiURL":      apiURL,
	}
	if !expiry.IsZero() {
		result["expiry"] = expiry.Format(time.RFC3339)
	}
	return result
}

// InstallCommandHooks registers JavaScript helper functions for command debugging.
func InstallCommandHooks() {
	// Create a global JavaScript function to log command-specific details
	js.Global().Set("logLocationCommand", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		message := "No message provided"
		if len(args) > 0 {
			message = args[0].String()
		}

		js.Global().Get("console").Call("group", "🌎 LOCATIONS COMMAND DEBUG")
		js.Global().Get("console").Call("log", message)

		// Log environment status
		accessKey := os.Getenv("MEGAPORT_ACCESS_KEY")
		secretKey := os.Getenv("MEGAPORT_SECRET_KEY")
		env := os.Getenv("MEGAPORT_ENVIRONMENT")

		js.Global().Get("console").Call("log", fmt.Sprintf("Environment: %s", env))
		js.Global().Get("console").Call("log", fmt.Sprintf("Access Key: %v", accessKey != ""))
		js.Global().Get("console").Call("log", fmt.Sprintf("Secret Key: %v", secretKey != ""))
		js.Global().Get("console").Call("groupEnd")

		return nil
	}))
}
