//go:build js && wasm

package wasmhttp

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"syscall/js"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// installMockFetch replaces the global fetch for the duration of a test and
// restores the original on cleanup. behavior receives the request URL and the
// fetch options (including the AbortController's signal) and must return a
// JS Promise, mirroring what the real browser fetch returns.
func installMockFetch(t *testing.T, behavior func(url string, opts js.Value) js.Value) {
	t.Helper()
	installMockFetchByURL(t, map[string]func(url string, opts js.Value) js.Value{"": behavior})
}

// installMockFetchByURL is like installMockFetch but dispatches to a
// different behavior per request URL, so a test can drive multiple
// concurrent requests through the same mocked fetch. A single entry keyed by
// the empty string matches any URL.
func installMockFetchByURL(t *testing.T, behaviors map[string]func(url string, opts js.Value) js.Value) {
	t.Helper()

	original := js.Global().Get("fetch")
	fetchFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		url := args[0].String()
		behavior, ok := behaviors[url]
		if !ok {
			behavior, ok = behaviors[""]
		}
		if !ok {
			t.Errorf("no mock fetch behavior registered for URL %q", url)
			return nil
		}
		return behavior(url, args[1])
	})
	js.Global().Set("fetch", fetchFunc)

	t.Cleanup(func() {
		js.Global().Set("fetch", original)
		fetchFunc.Release()
	})
}

// newHangingFetch returns fetch behavior that never resolves on its own, only
// rejecting once the request's AbortSignal fires. aborted is set to 1 when
// that happens, so tests can assert the browser fetch was actually cancelled.
func newHangingFetch(aborted *int32) func(url string, opts js.Value) js.Value {
	return func(url string, opts js.Value) js.Value {
		signal := opts.Get("signal")

		var executor js.Func
		executor = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			defer executor.Release()
			reject := args[1]

			var onAbort js.Func
			onAbort = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				defer onAbort.Release()
				atomic.StoreInt32(aborted, 1)
				abortErr := js.Global().Get("Error").New("The operation was aborted")
				abortErr.Set("name", "AbortError")
				reject.Invoke(abortErr)
				return nil
			})

			if !signal.IsUndefined() {
				signal.Call("addEventListener", "abort", onAbort)
			}
			return nil
		})

		return js.Global().Get("Promise").New(executor)
	}
}

// newResolvingFetch returns fetch behavior that resolves immediately with a
// minimal Response-like object carrying status and a JSON body. aborted is
// set to 1 if the request's AbortSignal ever fires, so tests can assert a
// successful request was never cancelled. The abort listener is released via
// t.Cleanup rather than self-releasing, since a successful request never
// fires it.
func newResolvingFetch(t *testing.T, status int, body string, aborted *int32) func(url string, opts js.Value) js.Value {
	t.Helper()
	return func(url string, opts js.Value) js.Value {
		if signal := opts.Get("signal"); !signal.IsUndefined() {
			onAbort := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				atomic.StoreInt32(aborted, 1)
				return nil
			})
			t.Cleanup(onAbort.Release)
			signal.Call("addEventListener", "abort", onAbort)
		}

		var executor js.Func
		executor = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			defer executor.Release()
			resolve := args[0]
			resolve.Invoke(newFakeResponse(status, body))
			return nil
		})
		return js.Global().Get("Promise").New(executor)
	}
}

func newFakeResponse(status int, body string) js.Value {
	var forEachFunc js.Func
	forEachFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer forEachFunc.Release()
		return nil
	})

	headers := js.ValueOf(map[string]interface{}{})
	headers.Set("forEach", forEachFunc)

	var textFunc js.Func
	textFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer textFunc.Release()
		return js.Global().Get("Promise").Call("resolve", body)
	})

	resp := js.ValueOf(map[string]interface{}{
		"status": status,
	})
	resp.Set("headers", headers)
	resp.Set("text", textFunc)
	return resp
}

func newTestRequest(t *testing.T, ctx context.Context, url string) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)
	return req
}

func TestRoundTrip_ContextCancelledReturnsPromptly(t *testing.T) {
	var aborted int32
	installMockFetch(t, newHangingFetch(&aborted))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	transport := &WasmHTTPTransport{Timeout: 5 * time.Second}
	req := newTestRequest(t, ctx, "https://example.invalid/test")

	start := time.Now()
	resp, err := transport.RoundTrip(req)
	elapsed := time.Since(start)

	assert.Nil(t, resp)
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled), "expected context.Canceled, got %v", err)
	assert.Less(t, elapsed, 2*time.Second, "cancellation should not wait for the full transport timeout")
	assert.Equal(t, int32(1), atomic.LoadInt32(&aborted), "expected the browser fetch to be aborted")
}

func TestRoundTrip_AlreadyCancelledContextSkipsFetch(t *testing.T) {
	var fetchCalled int32
	installMockFetch(t, func(url string, opts js.Value) js.Value {
		atomic.StoreInt32(&fetchCalled, 1)
		return newHangingFetch(new(int32))(url, opts)
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	transport := &WasmHTTPTransport{Timeout: 5 * time.Second}
	req := newTestRequest(t, ctx, "https://example.invalid/test")

	resp, err := transport.RoundTrip(req)

	assert.Nil(t, resp)
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled), "expected context.Canceled, got %v", err)
	assert.Equal(t, int32(0), atomic.LoadInt32(&fetchCalled), "fetch should never be called for an already-cancelled context")
}

func TestRoundTrip_TimeoutAbortsAndReturnsError(t *testing.T) {
	var aborted int32
	installMockFetch(t, newHangingFetch(&aborted))

	transport := &WasmHTTPTransport{Timeout: 50 * time.Millisecond}
	req := newTestRequest(t, context.Background(), "https://example.invalid/test")

	start := time.Now()
	resp, err := transport.RoundTrip(req)
	elapsed := time.Since(start)

	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.Less(t, elapsed, 2*time.Second, "timeout error should arrive close to the configured timeout")
	assert.Equal(t, int32(1), atomic.LoadInt32(&aborted), "expected the browser fetch to be aborted on timeout")
}

func TestRoundTrip_SuccessNormalResponseUnaffected(t *testing.T) {
	var aborted int32
	installMockFetch(t, newResolvingFetch(t, http.StatusOK, `{"ok":true}`, &aborted))

	transport := &WasmHTTPTransport{Timeout: 5 * time.Second}
	req := newTestRequest(t, context.Background(), "https://example.invalid/test")

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := make([]byte, 32)
	n, _ := resp.Body.Read(body)
	assert.Equal(t, `{"ok":true}`, string(body[:n]))
	assert.Equal(t, int32(0), atomic.LoadInt32(&aborted), "a successful request should never be aborted")
}

func TestRoundTrip_ConcurrentRequestsOnSharedTransport(t *testing.T) {
	const hangingURL = "https://example.invalid/hang"
	const okURL = "https://example.invalid/ok"

	var hangAborted, okAborted int32
	installMockFetchByURL(t, map[string]func(url string, opts js.Value) js.Value{
		hangingURL: newHangingFetch(&hangAborted),
		okURL:      newResolvingFetch(t, http.StatusOK, `{"ok":true}`, &okAborted),
	})

	transport := &WasmHTTPTransport{Timeout: 5 * time.Second}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	type result struct {
		resp *http.Response
		err  error
	}
	hangResultChan := make(chan result, 1)
	okResultChan := make(chan result, 1)

	go func() {
		resp, err := transport.RoundTrip(newTestRequest(t, ctx, hangingURL))
		hangResultChan <- result{resp, err}
	}()
	go func() {
		resp, err := transport.RoundTrip(newTestRequest(t, context.Background(), okURL))
		okResultChan <- result{resp, err}
	}()

	hangRes := <-hangResultChan
	okRes := <-okResultChan

	assert.Nil(t, hangRes.resp)
	require.Error(t, hangRes.err)
	assert.True(t, errors.Is(hangRes.err, context.Canceled), "expected context.Canceled, got %v", hangRes.err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&hangAborted), "the cancelled request's fetch should be aborted")

	require.NoError(t, okRes.err)
	require.NotNil(t, okRes.resp)
	defer okRes.resp.Body.Close()
	assert.Equal(t, http.StatusOK, okRes.resp.StatusCode)
	assert.Equal(t, int32(0), atomic.LoadInt32(&okAborted), "an unrelated request on the shared transport should not be aborted")
}

func TestBuildFetchOptions_NoForbiddenAcceptEncodingHeader(t *testing.T) {
	transport := &WasmHTTPTransport{}
	req, err := http.NewRequest(http.MethodGet, "https://example.invalid/test", nil)
	require.NoError(t, err)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")

	opts, err := transport.buildFetchOptions(req)
	require.NoError(t, err)

	headers, ok := opts["headers"].(map[string]interface{})
	require.True(t, ok)

	_, exists := headers["Accept-Encoding"]
	assert.False(t, exists, "Accept-Encoding is a forbidden fetch header and must not be set")
}

func TestBuildFetchOptions_MultiValueHeaderJoined(t *testing.T) {
	transport := &WasmHTTPTransport{}
	req, err := http.NewRequest(http.MethodGet, "https://example.invalid/test", nil)
	require.NoError(t, err)
	req.Header.Add("X-Custom", "first")
	req.Header.Add("X-Custom", "second")

	opts, err := transport.buildFetchOptions(req)
	require.NoError(t, err)

	headers, ok := opts["headers"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "first, second", headers["X-Custom"],
		"every value of a multi-value header should be forwarded, not just the first")
}

func TestBuildFetchOptions_BinaryBodyPreservedAsBytes(t *testing.T) {
	transport := &WasmHTTPTransport{}
	binary := []byte{0x00, 0xff, 0xfe, 'h', 'i', 0x80, 0x81}
	req, err := http.NewRequest(http.MethodPost, "https://example.invalid/test", bytes.NewReader(binary))
	require.NoError(t, err)

	opts, err := transport.buildFetchOptions(req)
	require.NoError(t, err)

	jsBody, ok := opts["body"].(js.Value)
	require.True(t, ok, "body should be passed as a js.Value (Uint8Array), not a Go string")
	assert.Equal(t, "Uint8Array", jsBody.Get("constructor").Get("name").String())

	length := jsBody.Get("length").Int()
	got := make([]byte, length)
	js.CopyBytesToGo(got, jsBody)
	assert.Equal(t, binary, got, "binary body bytes must round-trip without UTF-8 mangling")
}
