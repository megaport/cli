//go:build js && wasm

package main

import (
	"embed"
	"fmt"
	"sync"
	"syscall/js"
	"time"

	"github.com/megaport/megaport-cli/cmd/megaport"
	"github.com/megaport/megaport-cli/internal/base/cmdbuilder"
	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/wasm"
)

//go:embed docs/*.md
var embeddedDocs embed.FS

// asyncCommandMu prevents concurrent async command executions from corrupting
// the shared global output buffers.
var asyncCommandMu sync.Mutex

// executeMegaportCommand is retained so host pages that still detect or call
// it get an immediate, well-formed response instead of a broken function.
// It no longer executes commands: running a command synchronously blocks the
// JS event loop while Cobra waits on the async fetch/prompt transport, which
// hangs the tab until the transport times out, and it bypasses asyncCommandMu,
// letting a sync call race an in-flight async command over the shared output
// buffers. Use executeMegaportCommandAsync instead.
func executeMegaportCommand(this js.Value, args []js.Value) interface{} {
	return map[string]interface{}{
		"error": "synchronous execution is not supported; use executeMegaportCommandAsync",
	}
}

// invokeAsyncTimeoutCallback fires the completion callback with a timeout
// error, guarded by once so it never double-fires with the normal completion
// path. It is factored out of executeMegaportCommandAsync so the recover
// behavior can be exercised directly in tests without waiting out the real
// asyncCommandTimeout.
func invokeAsyncTimeoutCallback(callback js.Value, once *sync.Once) {
	once.Do(func() {
		if r := wasm.InvokeCallback(callback, map[string]interface{}{
			"error": "command timed out",
		}); r != nil {
			js.Global().Get("console").Call("error", "Timeout callback panicked")
		}
	})
}

// executeMegaportCommandAsync runs CLI commands asynchronously with a callback
// This is the CORRECT way to handle commands that involve async operations (like auth)
func executeMegaportCommandAsync(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		js.Global().Get("console").Call("error", "executeMegaportCommandAsync requires: command (string) and callback (function)")
		return nil
	}

	// Get command string and callback from JavaScript
	cmdString := args[0].String()
	callback := args[1]

	if callback.Type() != js.TypeFunction {
		js.Global().Get("console").Call("error", "Second argument must be a callback function")
		return nil
	}

	// asyncCommandTimeout is the maximum time an async command may run before
	// the callback is invoked with a timeout error. It is wasm.CommandTimeout,
	// the same budget PromptForInput waits for a single prompt response, so an
	// interactive command's pending prompt cannot time out before the command
	// itself does. The inner goroutine may still be running after the timeout
	// (Go goroutines cannot be forcibly cancelled), but the JS caller will not
	// be left waiting indefinitely.
	const asyncCommandTimeout = wasm.CommandTimeout

	// once ensures the callback is invoked exactly once even if both the timeout
	// and the normal completion path race.
	var once sync.Once

	// asyncCommandMu serializes async command executions so that concurrent calls
	// do not corrupt the shared global output buffers (ResetOutputBuffers /
	// WasmOutputBuffer / JS globals), which are not safe for concurrent use.
	//
	// commandDone is closed when the goroutine exits (normally or via panic) so
	// that the timeout timer can be stopped promptly rather than expiring after
	// up to asyncCommandTimeout even when the command finishes quickly.
	commandDone := make(chan struct{})

	go func() {
		defer close(commandDone)
		defer func() {
			if r := recover(); r != nil {
				once.Do(func() {
					callback.Invoke(map[string]interface{}{
						"error": wasm.SanitizeTerminalText(fmt.Sprintf("Command panicked: %v", r)),
					})
				})
			}
		}()

		asyncCommandMu.Lock()
		defer asyncCommandMu.Unlock()

		// Reset all output buffers
		wasm.ResetOutputBuffers()

		// Split the command string into arguments
		cmdArgs := wasm.SplitArgs(cmdString)

		// Create a new slice with the program name
		originalArgs := append([]string{"megaport-cli"}, cmdArgs...)

		// Use our tracing function (no-op when debug mode is off)
		wasm.TraceCommand(cmdString, originalArgs)

		// Ensure Cobra gets all our commands
		megaport.EnsureRootCommandOutput(wasm.WasmOutputBuffer)

		execErr := megaport.ExecuteWithArgs(originalArgs)

		// When a live-output handler is registered the narrative has already
		// streamed to the host; GetCompletionOutput returns only the structured
		// document (or "") so the host does not double-render it.
		result := wasm.GetCompletionOutput()
		once.Do(func() {
			resultObj := map[string]interface{}{
				"output": result,
			}
			// Route failures to result.error so the host colors them and fires
			// failure telemetry (ExecuteWithArgs returns nil when the error already
			// reached the terminal, so no double-render). SanitizeTerminalText strips
			// every control byte: the message can echo a user-typed flag verbatim
			// and the host writes result.error straight to xterm under its own color.
			if execErr != nil {
				resultObj["error"] = wasm.SanitizeTerminalText(execErr.Error())
			}
			callback.Invoke(resultObj)
		})
	}()

	// Fire a timeout so the callback is always invoked within asyncCommandTimeout.
	// This runs on the timer goroutine, not the goroutine above, so it needs its
	// own recover: a throwing callback here would otherwise terminate the whole
	// WASM runtime rather than just this command.
	t := time.AfterFunc(asyncCommandTimeout, func() {
		invokeAsyncTimeoutCallback(callback, &once)
	})

	// Stop the timer once the goroutine finishes so it is not kept alive for the
	// full asyncCommandTimeout duration when the command completes quickly.
	go func() {
		<-commandDone
		t.Stop()
	}()

	return nil
}

func main() {
	// Wire output state reset into the wasm package. Done here (rather than in
	// internal/wasm) to break the import cycle between wasm and output packages.
	wasm.RegisterOutputStateReset(func() {
		output.ResetState()
	})

	// Register the embedded documentation with the cmdbuilder package
	cmdbuilder.RegisterEmbeddedDocs(embeddedDocs)

	// Enable debug mode only when explicitly requested by the host page.
	// Set window.wasmDebugMode = true before the WASM module loads to opt in.
	if js.Global().Get("wasmDebugMode").Truthy() {
		wasm.EnableDebugMode()
	}

	// Register JavaScript functions
	wasm.RegisterJSFunctions()

	// Setup output redirection
	wasm.SetupIO()

	// Initialize the prompt system for interactive mode
	wasm.InitPromptSystem()

	// Initialize the live-output streaming system
	wasm.InitOutputSystem()

	// executeMegaportCommand is a deprecated stub kept for one release as a soft
	// landing for hosts still detecting/calling it; executeMegaportCommandAsync
	// is the only supported entrypoint.
	js.Global().Set("executeMegaportCommand", js.FuncOf(executeMegaportCommand))
	js.Global().Set("executeMegaportCommandAsync", js.FuncOf(executeMegaportCommandAsync))

	// Prevent Go WASM from exiting after main finishes
	<-make(chan bool)
}
