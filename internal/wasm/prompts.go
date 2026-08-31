//go:build js && wasm

package wasm

import (
	"fmt"
	"sync"
	"sync/atomic"
	"syscall/js"
	"time"
)

// CommandTimeout is the maximum duration an async CLI command may run (see
// main_wasm.go's asyncCommandTimeout) and the ceiling PromptForInput waits
// for a single prompt response. The two must stay equal: a shorter prompt
// timeout would fail an interactive command before its own budget expires.
const CommandTimeout = 10 * time.Minute

var (
	// promptCallback is the JavaScript function to call when prompting for input.
	// Protected by promptCallbackMu.
	promptCallback   js.Value
	promptCallbackMu sync.RWMutex

	// pendingPrompts tracks prompts waiting for responses
	pendingPrompts = make(map[string]*PromptRequest)
	pendingMutex   sync.Mutex

	// promptCounter generates unique IDs for each prompt
	promptCounter atomic.Int64

	// promptTimeout is the effective per-prompt wait ceiling. It defaults to
	// CommandTimeout but is a var (rather than using CommandTimeout directly)
	// so tests can shrink it instead of waiting out the full command budget.
	promptTimeout = CommandTimeout
)

// PromptRequest represents a pending prompt waiting for user input
type PromptRequest struct {
	ID           string
	Message      string
	PromptType   string // "text", "confirm", "password", "resource"
	ResourceType string // set for "resource" prompts and secret-resource prompts sent as "password" (port, mcr, vxc, etc.); empty otherwise
	ResponseChan chan string
	ErrorChan    chan error
}

// RegisterPromptCallback allows JavaScript to register a callback function
// that will be invoked when the WASM code needs user input
func RegisterPromptCallback(callback js.Value) {
	if callback.Type() != js.TypeFunction {
		js.Global().Get("console").Call("error", "Prompt callback must be a function")
		return
	}

	promptCallbackMu.Lock()
	promptCallback = callback
	promptCallbackMu.Unlock()
	js.Global().Get("console").Call("log", "✅ Prompt callback registered")
}

// PromptForInput requests input from the user via JavaScript
// This function blocks until the JavaScript side provides a response
func PromptForInput(message string, promptType string, resourceType string) (string, error) {
	promptCallbackMu.RLock()
	cb := promptCallback
	promptCallbackMu.RUnlock()

	if cb.IsUndefined() {
		return "", fmt.Errorf("prompt callback not registered - interactive mode requires JavaScript integration")
	}

	// Create a unique ID for this prompt
	promptID := fmt.Sprintf("prompt_%d_%d", promptCounter.Add(1), time.Now().UnixNano())

	// Create channels for the response
	responseChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	// Register the pending prompt
	request := &PromptRequest{
		ID:           promptID,
		Message:      message,
		PromptType:   promptType,
		ResourceType: resourceType,
		ResponseChan: responseChan,
		ErrorChan:    errorChan,
	}

	pendingMutex.Lock()
	pendingPrompts[promptID] = request
	pendingMutex.Unlock()

	// Log the prompt request. Message/response payloads are omitted: prompt input can
	// carry secrets (access keys, passwords) and this ships in the public WASM console.
	js.Global().Get("console").Call("log", fmt.Sprintf("📝 Requesting input: ID=%s, Type=%s", promptID, promptType))

	// Call the JavaScript callback with the prompt details. A throwing host
	// callback is recovered so it cannot leak this pendingPrompts entry or
	// crash the command; the select below never runs in that case, so cleanup
	// happens here instead.
	if r := InvokeCallback(cb, map[string]interface{}{
		"id":           promptID,
		"message":      message,
		"type":         promptType,
		"resourceType": resourceType,
	}); r != nil {
		pendingMutex.Lock()
		delete(pendingPrompts, promptID)
		pendingMutex.Unlock()

		js.Global().Get("console").Call("error", fmt.Sprintf("❌ Prompt callback panicked for %s", promptID))
		return "", fmt.Errorf("prompt callback threw for prompt %s", promptID)
	}

	// Wait for response with timeout. A stoppable timer (rather than time.After)
	// is stopped as soon as a response/error arrives so it isn't left running in
	// the runtime's timer heap for up to promptTimeout on every answered prompt.
	timer := time.NewTimer(promptTimeout)
	defer timer.Stop()

	select {
	case response := <-responseChan:
		// Clean up
		pendingMutex.Lock()
		delete(pendingPrompts, promptID)
		pendingMutex.Unlock()

		js.Global().Get("console").Call("log", fmt.Sprintf("✅ Received response for %s", promptID))
		return response, nil

	case err := <-errorChan:
		// Clean up
		pendingMutex.Lock()
		delete(pendingPrompts, promptID)
		pendingMutex.Unlock()

		js.Global().Get("console").Call("error", fmt.Sprintf("❌ Error for %s: %v", promptID, err))
		return "", err

	case <-timer.C:
		// Timeout
		pendingMutex.Lock()
		delete(pendingPrompts, promptID)
		pendingMutex.Unlock()

		return "", fmt.Errorf("prompt timeout: no response received after %s; the host can cancel a pending prompt via cancelPrompt", promptTimeout)
	}
}

// SubmitPromptResponse is called by JavaScript to provide the user's response
// Exported for testing
func SubmitPromptResponse(this js.Value, args []js.Value) interface{} {
	return submitPromptResponse(this, args)
}

// submitPromptResponse is the internal implementation
func submitPromptResponse(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		js.Global().Get("console").Call("error", "submitPromptResponse requires: id (string) and response (string)")
		return map[string]interface{}{
			"error": "Invalid arguments",
		}
	}

	promptID := args[0].String()
	response := args[1].String()

	js.Global().Get("console").Call("log", fmt.Sprintf("📨 Submitting response for %s", promptID))

	pendingMutex.Lock()
	request, exists := pendingPrompts[promptID]
	pendingMutex.Unlock()

	if !exists {
		js.Global().Get("console").Call("warn", fmt.Sprintf("No pending prompt found for ID: %s", promptID))
		return map[string]interface{}{
			"error": "Prompt not found",
		}
	}

	// Send the response to the waiting goroutine
	select {
	case request.ResponseChan <- response:
		js.Global().Get("console").Call("log", fmt.Sprintf("✅ Response sent for %s", promptID))
		return map[string]interface{}{
			"success": true,
		}
	default:
		js.Global().Get("console").Call("error", fmt.Sprintf("Failed to send response for %s", promptID))
		return map[string]interface{}{
			"error": "Failed to send response",
		}
	}
}

// CancelPrompt is called by JavaScript to cancel a pending prompt
func cancelPrompt(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		js.Global().Get("console").Call("error", "cancelPrompt requires: id (string)")
		return map[string]interface{}{
			"error": "Invalid arguments",
		}
	}

	promptID := args[0].String()

	pendingMutex.Lock()
	request, exists := pendingPrompts[promptID]
	pendingMutex.Unlock()

	if !exists {
		return map[string]interface{}{
			"error": "Prompt not found",
		}
	}

	// Send error to the waiting goroutine
	select {
	case request.ErrorChan <- fmt.Errorf("prompt cancelled by user"):
		js.Global().Get("console").Call("log", fmt.Sprintf("Cancelled prompt %s", promptID))
		return map[string]interface{}{
			"success": true,
		}
	default:
		return map[string]interface{}{
			"error": "Failed to cancel prompt",
		}
	}
}

// RegisterPromptHandler is called by JavaScript to register the prompt callback
func registerPromptHandler(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		js.Global().Get("console").Call("error", "registerPromptHandler requires: callback (function)")
		return false
	}

	callback := args[0]
	if callback.Type() != js.TypeFunction {
		js.Global().Get("console").Call("error", "Argument must be a function")
		return false
	}

	RegisterPromptCallback(callback)
	return true
}

// GetPendingPrompts returns info about pending prompts (for debugging)
func getPendingPrompts(this js.Value, args []js.Value) interface{} {
	pendingMutex.Lock()
	defer pendingMutex.Unlock()

	prompts := make([]map[string]interface{}, 0, len(pendingPrompts))
	for id, req := range pendingPrompts {
		prompts = append(prompts, map[string]interface{}{
			"id":           id,
			"message":      req.Message,
			"type":         req.PromptType,
			"resourceType": req.ResourceType,
		})
	}

	return prompts
}

// InitPromptSystem registers the JavaScript functions for the prompt system
func InitPromptSystem() {
	js.Global().Set("registerPromptHandler", js.FuncOf(registerPromptHandler))
	js.Global().Set("submitPromptResponse", js.FuncOf(submitPromptResponse))
	js.Global().Set("cancelPrompt", js.FuncOf(cancelPrompt))
	js.Global().Set("getPendingPrompts", js.FuncOf(getPendingPrompts))

	js.Global().Get("console").Call("log", "✅ WASM Prompt System initialized")
	js.Global().Get("console").Call("log", "Available functions:")
	js.Global().Get("console").Call("log", "  - registerPromptHandler(callback)")
	js.Global().Get("console").Call("log", "  - submitPromptResponse(id, response)")
	js.Global().Get("console").Call("log", "  - cancelPrompt(id)")
	js.Global().Get("console").Call("log", "  - getPendingPrompts()")
}
