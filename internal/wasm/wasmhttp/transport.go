//go:build js && wasm

package wasmhttp

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"syscall/js"
	"time"
)

// WasmHTTPTransport implements http.RoundTripper using browser fetch API
// This allows the standard megaportgo SDK to work in WASM without modification
type WasmHTTPTransport struct {
	Timeout time.Duration // Request timeout (default: 60s)
}

// NewWasmHTTPClient returns an http.Client configured for WASM using browser fetch
func NewWasmHTTPClient() *http.Client {
	return &http.Client{
		Transport: &WasmHTTPTransport{
			Timeout: 60 * time.Second,
		},
	}
}

// RoundTrip executes a single HTTP transaction using browser fetch API
// This is the core method that bridges Go's http.Client with JavaScript fetch
func (t *WasmHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := req.Context().Err(); err != nil {
		return nil, err
	}

	console := js.Global().Get("console")

	// Determine timeout
	timeout := t.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	// Log the request
	console.Call("log", fmt.Sprintf("🌐 HTTP %s %s", req.Method, req.URL.String()))

	// Convert http.Request to fetch options
	fetchOpts, err := t.buildFetchOptions(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build fetch options: %w", err)
	}

	// Make the fetch call
	response, err := t.doFetch(req, fetchOpts, timeout)
	if err != nil {
		return nil, err
	}

	// Convert fetch response to http.Response
	httpResponse := t.buildHTTPResponse(response, req)

	console.Call("log", fmt.Sprintf("✅ HTTP %d %s (%d bytes)",
		httpResponse.StatusCode,
		req.URL.String(),
		len(response.Body)))

	return httpResponse, nil
}

// buildFetchOptions converts an http.Request to fetch API options
func (t *WasmHTTPTransport) buildFetchOptions(req *http.Request) (map[string]interface{}, error) {
	// Build headers map. Accept-Encoding is a forbidden fetch header the
	// browser silently strips, so drop it here too even if a caller set it
	// directly on the request rather than relying on us to add it.
	headers := make(map[string]interface{})
	for key, values := range req.Header {
		if strings.EqualFold(key, "Accept-Encoding") {
			continue
		}
		if len(values) > 0 {
			// fetch's Headers init dict accepts one string per key, so combine
			// multi-value headers the same way HTTP field values are combined
			// (RFC 7230 3.2.2). Cookie combines with "; " instead, but the SDK
			// never sets a multi-value Cookie header through this transport.
			headers[key] = strings.Join(values, ", ")
		}
	}

	fetchOpts := map[string]interface{}{
		"method":  req.Method,
		"headers": headers,
	}

	// Add body for POST, PUT, PATCH, DELETE requests
	if req.Body != nil && req.Method != http.MethodGet && req.Method != http.MethodHead {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}

		// Restore body for potential retries
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		if len(bodyBytes) > 0 {
			// Pass raw bytes as a Uint8Array rather than a Go string, which fetch
			// would otherwise treat as UTF-8 text and mangle a binary body.
			jsBody := js.Global().Get("Uint8Array").New(len(bodyBytes))
			js.CopyBytesToJS(jsBody, bodyBytes)
			fetchOpts["body"] = jsBody
		}
	}

	return fetchOpts, nil
}

// doFetch performs the actual fetch call and handles the promise. The fetch
// is wired to an AbortController so the browser request actually stops when
// req.Context() is cancelled or the transport timeout fires, instead of
// continuing to run after Go has given up on it.
func (t *WasmHTTPTransport) doFetch(req *http.Request, options map[string]interface{}, timeout time.Duration) (*fetchResponse, error) {
	if err := req.Context().Err(); err != nil {
		return nil, err
	}

	console := js.Global().Get("console")
	startTime := time.Now()

	abortController := js.Global().Get("AbortController").New()
	options["signal"] = abortController.Get("signal")

	// Make the fetch call
	promise := js.Global().Call("fetch", req.URL.String(), options)

	// Channels for async response handling
	resultChan := make(chan *fetchResponse, 1)
	errorChan := make(chan error, 1)

	var thenFunc, catchFunc js.Func
	var responseStatus int
	var responseHeaders map[string]string

	// Success handler
	thenFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer thenFunc.Release()
		defer catchFunc.Release()

		if len(args) == 0 {
			errorChan <- fmt.Errorf("no response from fetch")
			return nil
		}

		response := args[0]
		responseStatus = response.Get("status").Int()
		responseHeaders = t.extractHeaders(response)

		// Log compression if present
		if encoding, ok := responseHeaders["content-encoding"]; ok && encoding != "" {
			console.Call("log", fmt.Sprintf("📥 Response status: %d (compressed: %s)", responseStatus, encoding))
		} else {
			console.Call("log", fmt.Sprintf("📥 Response status: %d", responseStatus))
		}

		// Create handlers for reading response body
		var textThen, textCatch js.Func

		textThen = js.FuncOf(func(this js.Value, textArgs []js.Value) interface{} {
			defer textThen.Release()
			defer textCatch.Release()

			if len(textArgs) == 0 {
				errorChan <- fmt.Errorf("no body in response")
				return nil
			}

			bodyText := textArgs[0].String()
			elapsed := time.Since(startTime).Milliseconds()

			console.Call("log", fmt.Sprintf("📦 Received %d bytes in %dms", len(bodyText), elapsed))

			resultChan <- &fetchResponse{
				StatusCode: responseStatus,
				Headers:    responseHeaders,
				Body:       []byte(bodyText),
			}

			return nil
		})

		textCatch = js.FuncOf(func(this js.Value, textArgs []js.Value) interface{} {
			defer textThen.Release()
			defer textCatch.Release()

			errMsg := "unknown error reading response body"
			if len(textArgs) > 0 && !textArgs[0].IsUndefined() {
				if msg := textArgs[0].Get("message"); !msg.IsUndefined() {
					errMsg = msg.String()
				}
			}

			console.Call("error", fmt.Sprintf("❌ Error reading body: %s", errMsg))
			errorChan <- fmt.Errorf("failed to read response body: %s", errMsg)
			return nil
		})

		// Read the response body as text
		textPromise := response.Call("text")
		textPromise.Call("then", textThen).Call("catch", textCatch)
		return nil
	})

	// Error handler
	catchFunc = js.FuncOf(func(this js.Value, catchArgs []js.Value) interface{} {
		defer thenFunc.Release()
		defer catchFunc.Release()

		errMsg := "unknown fetch error"
		isAbort := false
		if len(catchArgs) > 0 && !catchArgs[0].IsUndefined() {
			if name := catchArgs[0].Get("name"); !name.IsUndefined() && name.String() == "AbortError" {
				isAbort = true
			}
			if msg := catchArgs[0].Get("message"); !msg.IsUndefined() {
				errMsg = msg.String()
			}
		}

		// An abort is expected behavior (context cancellation or timeout), not
		// a genuine fetch failure, so it doesn't warrant an error-level log.
		if isAbort {
			console.Call("log", fmt.Sprintf("⏹️ Fetch aborted: %s", errMsg))
		} else {
			console.Call("error", fmt.Sprintf("❌ Fetch failed: %s", errMsg))
		}
		errorChan <- fmt.Errorf("fetch failed: %s", errMsg)
		return nil
	})

	promise.Call("then", thenFunc).Call("catch", catchFunc)

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	// Non-blocking check for an already-fired cancellation/timeout, used both
	// up front and right after a result/error arrives, so that outcome wins
	// deterministically instead of leaving it to Go's pseudo-random selection
	// among ready channels.
	checkPriority := func() error {
		select {
		case <-req.Context().Done():
			return req.Context().Err()
		case <-timer.C:
			return fmt.Errorf("fetch timeout after %v", timeout)
		default:
			return nil
		}
	}

	if err := checkPriority(); err != nil {
		abortController.Call("abort")
		return nil, err
	}

	// Wait for result, context cancellation, or timeout. Aborting the
	// controller stops the in-flight browser fetch; its eventual (unread)
	// rejection lands in the buffered errorChan and is discarded rather than
	// leaking a goroutine.
	select {
	case result := <-resultChan:
		if err := checkPriority(); err != nil {
			abortController.Call("abort")
			return nil, err
		}
		return result, nil
	case err := <-errorChan:
		if cancelErr := checkPriority(); cancelErr != nil {
			abortController.Call("abort")
			return nil, cancelErr
		}
		return nil, err
	case <-req.Context().Done():
		abortController.Call("abort")
		return nil, req.Context().Err()
	case <-timer.C:
		abortController.Call("abort")
		return nil, fmt.Errorf("fetch timeout after %v", timeout)
	}
}

// extractHeaders extracts headers from the fetch Response object
func (t *WasmHTTPTransport) extractHeaders(response js.Value) map[string]string {
	headers := make(map[string]string)

	if response.Get("headers").IsUndefined() {
		return headers
	}

	jsHeaders := response.Get("headers")

	// Try to use forEach if available
	if !jsHeaders.Get("forEach").IsUndefined() {
		forEachFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) >= 2 {
				value := args[0].String()
				key := args[1].String()
				headers[key] = value
			}
			return nil
		})
		defer forEachFunc.Release()

		jsHeaders.Call("forEach", forEachFunc)
	}

	return headers
}

// buildHTTPResponse converts a fetchResponse to http.Response
func (t *WasmHTTPTransport) buildHTTPResponse(fetchResp *fetchResponse, req *http.Request) *http.Response {
	// Build http.Header from map
	header := make(http.Header)
	for key, value := range fetchResp.Headers {
		header.Set(key, value)
	}

	return &http.Response{
		Status:        http.StatusText(fetchResp.StatusCode),
		StatusCode:    fetchResp.StatusCode,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(fetchResp.Body)),
		ContentLength: int64(len(fetchResp.Body)),
		Request:       req,
	}
}

// fetchResponse represents the result of a fetch call
type fetchResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}
