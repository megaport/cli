//go:build js && wasm

package main

import (
	"os"
	"sync"
	"syscall/js"
	"testing"
	"time"

	"github.com/megaport/megaport-cli/internal/base/cmdbuilder"
	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/wasm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain mirrors main()'s registration (minus the blocking channel wait):
// go test never calls a package main's main(), so without this the
// JS globals these tests depend on (executeMegaportCommand*,
// resetWasmOutput, etc.) are never registered and every test here fails.
func TestMain(m *testing.M) {
	wasm.RegisterOutputStateReset(func() {
		output.ResetState()
	})
	cmdbuilder.RegisterEmbeddedDocs(embeddedDocs)
	// Unlike main(), always enable debug mode here: TestWasmHelperFunctions
	// asserts every helper (including debug-only ones like debugAuthInfo and
	// wasmDebug) is registered.
	wasm.EnableDebugMode()
	wasm.RegisterJSFunctions()
	wasm.SetupIO()
	wasm.InitPromptSystem()
	js.Global().Set("executeMegaportCommand", js.FuncOf(executeMegaportCommand))
	js.Global().Set("executeMegaportCommandAsync", js.FuncOf(executeMegaportCommandAsync))

	// The bare-node WASM test host has no localStorage, so the config-file and
	// localStorage helpers, which call it directly, panic and take down the whole
	// test binary. Install a minimal in-memory shim so those tests run for real
	// under node; a browser host already provides localStorage, so only shim when
	// it is absent.
	if js.Global().Get("localStorage").IsUndefined() {
		js.Global().Call("eval", `(function(){var s={};globalThis.localStorage={getItem:function(k){return Object.prototype.hasOwnProperty.call(s,k)?s[k]:null;},setItem:function(k,v){s[k]=String(v);},removeItem:function(k){delete s[k];},clear:function(){s={};}};})();`)
	}

	os.Exit(m.Run())
}

// TestWasmInitialization verifies WASM module initializes correctly
func TestWasmInitialization(t *testing.T) {
	// Verify JS functions are registered
	executeMegaportCmd := js.Global().Get("executeMegaportCommand")
	assert.False(t, executeMegaportCmd.IsUndefined(), "executeMegaportCommand should be defined")
	assert.Equal(t, js.TypeFunction, executeMegaportCmd.Type())

	executeMegaportCmdAsync := js.Global().Get("executeMegaportCommandAsync")
	assert.False(t, executeMegaportCmdAsync.IsUndefined(), "executeMegaportCommandAsync should be defined")
	assert.Equal(t, js.TypeFunction, executeMegaportCmdAsync.Type())
}

// TestWasmHelperFunctions verifies helper functions are registered
func TestWasmHelperFunctions(t *testing.T) {
	expectedFunctions := []string{
		"readConfigFile",
		"writeConfigFile",
		"debugAuthInfo",
		"saveToLocalStorage",
		"loadFromLocalStorage",
		"resetWasmOutput",
		"getWasmOutput",
		"toggleWasmDebug",
		"dumpBuffers",
		"wasmDebug",
	}

	for _, funcName := range expectedFunctions {
		jsFunc := js.Global().Get(funcName)
		assert.False(t, jsFunc.IsUndefined(), "Function %s should be defined", funcName)
	}
}

// TestExecuteMegaportCommand_Deprecated verifies the legacy sync entrypoint no
// longer executes commands and always returns an immediate error pointing
// callers at executeMegaportCommandAsync, regardless of the command given.
func TestExecuteMegaportCommand_Deprecated(t *testing.T) {
	executeMegaportCmd := js.Global().Get("executeMegaportCommand")

	for _, cmd := range []string{"--help", "version", "nonexistent-command", "", "version --format json"} {
		result := executeMegaportCmd.Invoke(js.ValueOf(cmd))

		assert.Equal(t, js.TypeObject, result.Type())
		assert.True(t, result.Get("output").IsUndefined(), "sync entrypoint should not produce output for %q", cmd)

		errMsg := result.Get("error")
		assert.False(t, errMsg.IsUndefined())
		assert.Contains(t, errMsg.String(), "executeMegaportCommandAsync")
	}
}

// invokeAsyncAndWait calls executeMegaportCommandAsync and blocks until the
// callback fires, returning the result object. Fails the test instead of
// hanging forever if the callback never fires or fires without a result.
func invokeAsyncAndWait(t *testing.T, cmd string) js.Value {
	t.Helper()
	executeMegaportCmdAsync := js.Global().Get("executeMegaportCommandAsync")

	// errResult builds a well-formed result object so callers that immediately
	// do result.Get("output") on the return value fail the assertion cleanly
	// instead of panicking on an undefined value.
	errResult := func(msg string) js.Value {
		return js.ValueOf(map[string]interface{}{"error": msg})
	}

	resultCh := make(chan js.Value, 1)
	var callback js.Func
	callback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer callback.Release()
		if len(args) < 1 {
			t.Errorf("executeMegaportCommandAsync callback invoked with no arguments for command %q", cmd)
			resultCh <- errResult("test helper: callback invoked with no arguments")
			return nil
		}
		resultCh <- args[0]
		return nil
	})

	executeMegaportCmdAsync.Invoke(js.ValueOf(cmd), callback)

	select {
	case result := <-resultCh:
		return result
	case <-time.After(30 * time.Second):
		t.Fatalf("executeMegaportCommandAsync callback never fired for command %q", cmd)
		return errResult("test helper: callback never fired")
	}
}

// TestExecuteMegaportCommandAsync_UnknownCommand_RoutesToError is the ESD-1666
// bridge-level regression: an unknown command must populate result.error so the
// Portal renders the red Error: line and emits the failure telemetry event,
// rather than leaving the failure only in result.output (uncolored, uncounted).
func TestExecuteMegaportCommandAsync_UnknownCommand_RoutesToError(t *testing.T) {
	wasm.ResetOutputBuffers()

	result := invokeAsyncAndWait(t, "location list")

	errVal := result.Get("error")
	require.False(t, errVal.IsUndefined(), "unknown command should populate result.error")
	assert.Contains(t, errVal.String(), "unknown command")

	if out := result.Get("output"); !out.IsUndefined() {
		assert.NotContains(t, out.String(), "unknown command", "error text must not also be in result.output")
	}
}

// TestExecuteMegaportCommandAsync_ErrorIsSanitized guards the security fix that
// rides along with ESD-1666: a parser error echoes a user-typed flag name
// verbatim (pflag's "unknown flag: --%s"), and the host writes result.error
// straight to xterm. Routing that error to result.error must not bypass the
// control-byte sanitization WasmOutputBuffer applies to result.output, or a
// crafted flag name becomes a terminal-injection vector.
func TestExecuteMegaportCommandAsync_ErrorIsSanitized(t *testing.T) {
	wasm.ResetOutputBuffers()

	// An unknown flag whose name embeds an ESC-based clear-screen sequence.
	result := invokeAsyncAndWait(t, "locations list --\x1b[2Jevil")

	errVal := result.Get("error")
	require.False(t, errVal.IsUndefined(), "an unknown flag should populate result.error")
	assert.NotContains(t, errVal.String(), "\x1b", "control bytes must be stripped from result.error")
}

// TestExecuteMegaportCommandAsync_Callback verifies async command with callback
func TestExecuteMegaportCommandAsync_Callback(t *testing.T) {
	wasm.ResetOutputBuffers()

	// Create a callback function
	callback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Callback receives the result
		if len(args) > 0 {
			result := args[0]
			assert.Equal(t, js.TypeObject, result.Type())
		}
		return nil
	})
	defer callback.Release()

	// Call async version
	executeMegaportCmdAsync := js.Global().Get("executeMegaportCommandAsync")
	executeMegaportCmdAsync.Invoke(
		js.ValueOf("version"),
		callback,
	)

	// Give it a moment to execute (in real tests, you'd use proper async handling)
	// Note: In actual usage, the callback would be called asynchronously
	// For testing, we verify the function doesn't panic
	assert.NotPanics(t, func() {
		executeMegaportCmdAsync.Invoke(js.ValueOf("--help"), callback)
	})
}

// TestExecuteMegaportCommandAsync_NoCallback verifies error handling without callback
func TestExecuteMegaportCommandAsync_NoCallback(t *testing.T) {
	executeMegaportCmdAsync := js.Global().Get("executeMegaportCommandAsync")

	// Should not panic even with missing callback
	assert.NotPanics(t, func() {
		executeMegaportCmdAsync.Invoke(js.ValueOf("version"))
	})
}

// TestExecuteMegaportCommandAsync_InvalidCallback verifies invalid callback handling
func TestExecuteMegaportCommandAsync_InvalidCallback(t *testing.T) {
	executeMegaportCmdAsync := js.Global().Get("executeMegaportCommandAsync")

	// Should not panic with non-function callback
	assert.NotPanics(t, func() {
		executeMegaportCmdAsync.Invoke(
			js.ValueOf("version"),
			js.ValueOf("not a function"),
		)
	})
}

// TestOutputBufferReset verifies output buffers reset between commands
func TestOutputBufferReset(t *testing.T) {
	result1 := invokeAsyncAndWait(t, "ports --help")
	assert.True(t, result1.Get("error").IsUndefined(), "ports --help command returned an error: %v", result1.Get("error"))
	output1 := result1.Get("output").String()

	result2 := invokeAsyncAndWait(t, "--help")
	assert.True(t, result2.Get("error").IsUndefined(), "--help command returned an error: %v", result2.Get("error"))
	output2 := result2.Get("output").String()

	// Outputs should be different (buffers were reset)
	assert.NotEqual(t, output1, output2)
	assert.NotEmpty(t, output1)
	assert.NotEmpty(t, output2)
}

// TestWasmOutputBufferIntegration verifies WasmOutputBuffer usage
func TestWasmOutputBufferIntegration(t *testing.T) {
	wasm.ResetOutputBuffers()

	// Write to the buffer
	testData := "test integration data"
	_, _ = wasm.WasmOutputBuffer.Write([]byte(testData))

	// Verify it can be retrieved
	output := wasm.WasmOutputBuffer.String()
	assert.Equal(t, testData, output)

	// Reset and verify
	wasm.WasmOutputBuffer.Reset()
	assert.Equal(t, "", wasm.WasmOutputBuffer.String())
}

// TestCommandArgumentParsing verifies complex argument parsing
func TestCommandArgumentParsing(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "simple command",
			command: "version",
		},
		{
			name:    "command with flags",
			command: "version --format json",
		},
		{
			name:    "command with quoted args",
			command: `config set-default --name "My Profile"`,
		},
		{
			name:    "complex command",
			command: "ports list --format table --no-color",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wasm.ResetOutputBuffers()

			assert.NotPanics(t, func() {
				result := invokeAsyncAndWait(t, tt.command)
				assert.Equal(t, js.TypeObject, result.Type())
			})
		})
	}
}

// TestGlobalVariableCleanup verifies globals are cleaned up
func TestGlobalVariableCleanup(t *testing.T) {
	// Set some globals
	js.Global().Set("wasmJSONOutput", "test json")
	js.Global().Set("wasmCSVOutput", "test csv")
	js.Global().Set("wasmTableOutput", "test table")

	// Reset should clear them
	wasm.ResetOutputBuffers()

	assert.True(t, js.Global().Get("wasmJSONOutput").IsUndefined())
	assert.True(t, js.Global().Get("wasmCSVOutput").IsUndefined())
	assert.True(t, js.Global().Get("wasmTableOutput").IsUndefined())
}

// TestConfigFileOperations verifies config file read/write
func TestConfigFileOperations(t *testing.T) {
	writeConfigFile := js.Global().Get("writeConfigFile")
	readConfigFile := js.Global().Get("readConfigFile")

	// Write a config file
	testContent := `{"test": "config"}`
	writeResult := writeConfigFile.Invoke(
		js.ValueOf("test-config.json"),
		js.ValueOf(testContent),
	)

	// Verify write succeeded
	assert.Equal(t, js.TypeObject, writeResult.Type())

	// Read it back
	readResult := readConfigFile.Invoke(js.ValueOf("test-config.json"))
	assert.Equal(t, js.TypeObject, readResult.Type())

	// Clean up
	js.Global().Get("localStorage").Call("removeItem", "megaport_fs_test-config.json")
}

// TestLocalStorageOperations verifies localStorage save/load
func TestLocalStorageOperations(t *testing.T) {
	saveToLocalStorage := js.Global().Get("saveToLocalStorage")
	loadFromLocalStorage := js.Global().Get("loadFromLocalStorage")

	// Save a value
	testKey := "test-key"
	testValue := "test-value"
	saveResult := saveToLocalStorage.Invoke(
		js.ValueOf(testKey),
		js.ValueOf(testValue),
	)
	assert.Equal(t, true, saveResult.Bool())

	// Load it back
	loadResult := loadFromLocalStorage.Invoke(js.ValueOf(testKey))
	assert.Equal(t, testValue, loadResult.String())

	// Clean up
	js.Global().Get("localStorage").Call("removeItem", testKey)
}

// TestDebugMode verifies debug mode toggle
func TestDebugMode(t *testing.T) {
	toggleWasmDebug := js.Global().Get("toggleWasmDebug")
	wasmDebug := js.Global().Get("wasmDebug")

	// Get initial state
	initialState := wasmDebug.Invoke().Bool()

	// Toggle
	newState := toggleWasmDebug.Invoke().Bool()
	assert.NotEqual(t, initialState, newState)

	// Toggle back
	finalState := toggleWasmDebug.Invoke().Bool()
	assert.Equal(t, initialState, finalState)
}

// TestDumpBuffers verifies buffer dumping
func TestDumpBuffers(t *testing.T) {
	dumpBuffers := js.Global().Get("dumpBuffers")

	// Add some content
	wasm.ResetOutputBuffers()
	_, _ = wasm.WasmOutputBuffer.Write([]byte("test content"))

	// Dump buffers
	result := dumpBuffers.Invoke()
	assert.Equal(t, js.TypeObject, result.Type())

	// Verify structure
	stdout := result.Get("stdout")
	stderr := result.Get("stderr")
	direct := result.Get("direct")

	assert.False(t, stdout.IsUndefined())
	assert.False(t, stderr.IsUndefined())
	assert.False(t, direct.IsUndefined())
}

// TestErrorRecovery verifies panic recovery in the async execution path,
// which is now the only path that actually runs commands.
func TestErrorRecovery(t *testing.T) {
	// These should not cause the WASM module to crash
	assert.NotPanics(t, func() {
		invokeAsyncAndWait(t, "nonexistent")
	})

	assert.NotPanics(t, func() {
		invokeAsyncAndWait(t, "")
	})

	assert.NotPanics(t, func() {
		invokeAsyncAndWait(t, "invalid command with many args that don't make sense")
	})
}

// TestInvokeAsyncTimeoutCallback_ThrowingCallbackRecovered verifies a host
// completion callback that throws on the timeout path is recovered rather
// than crashing the runtime, and that once is still consumed so a
// simultaneous normal-completion callback.Invoke would not double-fire.
func TestInvokeAsyncTimeoutCallback_ThrowingCallbackRecovered(t *testing.T) {
	throwing := js.Global().Get("Function").New("throw new Error('timeout callback boom')")

	var once sync.Once
	assert.NotPanics(t, func() {
		invokeAsyncTimeoutCallback(throwing, &once)
	})

	fired := false
	once.Do(func() { fired = true })
	assert.False(t, fired, "once should already be consumed by the timeout invocation")
}

// TestInvokeAsyncTimeoutCallback_CleanInvocation verifies the normal
// (non-throwing) timeout path still delivers the timeout error to the
// callback.
func TestInvokeAsyncTimeoutCallback_CleanInvocation(t *testing.T) {
	var received js.Value
	fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			received = args[0]
		}
		return nil
	})
	defer fn.Release()

	var once sync.Once
	invokeAsyncTimeoutCallback(fn.Value, &once)

	assert.False(t, received.IsUndefined())
	assert.Equal(t, "command timed out", received.Get("error").String())
}

// TestConcurrentCommands verifies concurrent async commands don't interfere
// with each other's output (asyncCommandMu serializes execution).
func TestConcurrentCommands(t *testing.T) {
	executeMegaportCmdAsync := js.Global().Get("executeMegaportCommandAsync")

	type namedResult struct {
		cmd    string
		output string
	}
	resultCh := make(chan namedResult, 2)

	invoke := func(cmd string) {
		var callback js.Func
		callback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			defer callback.Release()
			if len(args) < 1 {
				t.Errorf("executeMegaportCommandAsync callback invoked with no arguments for command %q", cmd)
				resultCh <- namedResult{cmd: cmd}
				return nil
			}
			result := args[0]
			if err := result.Get("error"); !err.IsUndefined() {
				t.Errorf("executeMegaportCommandAsync returned an error for command %q: %v", cmd, err)
			}
			resultCh <- namedResult{cmd: cmd, output: result.Get("output").String()}
			return nil
		})
		executeMegaportCmdAsync.Invoke(js.ValueOf(cmd), callback)
	}

	invoke("ports --help")
	invoke("--help")

	results := make(map[string]string, 2)
	for i := 0; i < 2; i++ {
		select {
		case r := <-resultCh:
			results[r.cmd] = r.output
		case <-time.After(30 * time.Second):
			t.Fatalf("timed out waiting for concurrent command result %d/2", i+1)
		}
	}

	assert.NotEmpty(t, results["ports --help"])
	assert.NotEmpty(t, results["--help"])
	assert.NotEqual(t, results["ports --help"], results["--help"])
}
