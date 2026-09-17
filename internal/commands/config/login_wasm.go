//go:build js && wasm

package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall/js"
	"time"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/wasm"
	"github.com/megaport/megaport-cli/internal/wasm/wasmhttp"
	megaport "github.com/megaport/megaportgo"
)

// loginFuncMu guards loginFunc and newUnauthenticatedClientFunc.
var loginFuncMu sync.RWMutex

// GetLoginFunc returns the current login function in a thread-safe manner.
func GetLoginFunc() func(context.Context) (*megaport.Client, error) {
	loginFuncMu.RLock()
	defer loginFuncMu.RUnlock()
	return loginFunc
}

// SetLoginFunc replaces the login function in a thread-safe manner.
func SetLoginFunc(fn func(context.Context) (*megaport.Client, error)) {
	loginFuncMu.Lock()
	defer loginFuncMu.Unlock()
	loginFunc = fn
}

// GetLoginFuncWithOutput is not used in WASM but provided for API compatibility with testutil.
func GetLoginFuncWithOutput() func(context.Context, string) (*megaport.Client, error) {
	loginFuncMu.RLock()
	currentLoginFunc := loginFunc
	loginFuncMu.RUnlock()
	return func(ctx context.Context, _ string) (*megaport.Client, error) {
		return currentLoginFunc(ctx)
	}
}

// SetLoginFuncWithOutput is not used in WASM but provided for API compatibility with testutil.
func SetLoginFuncWithOutput(fn func(context.Context, string) (*megaport.Client, error)) {
	loginFuncMu.Lock()
	defer loginFuncMu.Unlock()
	loginFunc = func(ctx context.Context) (*megaport.Client, error) {
		return fn(ctx, "")
	}
}

// GetNewUnauthenticatedClientFunc returns the current unauthenticated client factory in a thread-safe manner.
func GetNewUnauthenticatedClientFunc() func() (*megaport.Client, error) {
	loginFuncMu.RLock()
	defer loginFuncMu.RUnlock()
	return newUnauthenticatedClientFunc
}

// SetNewUnauthenticatedClientFunc replaces the unauthenticated client factory in a thread-safe manner.
func SetNewUnauthenticatedClientFunc(fn func() (*megaport.Client, error)) {
	loginFuncMu.Lock()
	defer loginFuncMu.Unlock()
	newUnauthenticatedClientFunc = fn
}

func Login(ctx context.Context) (*megaport.Client, error) {
	return GetLoginFunc()(ctx)
}

func LoginWithOutput(ctx context.Context, outputFormat string) (*megaport.Client, error) {
	return GetLoginFuncWithOutput()(ctx, outputFormat)
}

// loginFunc overrides the standard login for WASM environments.
// Note: WASM version uses session-based authentication managed by the browser UI.
// Config profiles are not supported in the WASM version.
var loginFunc = func(ctx context.Context) (*megaport.Client, error) {
	var accessKey, secretKey, env string

	// Add console logging for debugging
	js.Global().Get("console").Call("group", "🔐 Megaport Authentication Debug")
	js.Global().Get("console").Call("info", "ℹ️  WASM version uses session-based authentication")
	js.Global().Get("console").Call("info", "ℹ️  Config profiles are not supported - please use the login form in the UI")

	// PRIORITY 1: Check for external token from portal (bypasses OAuth flow)
	// Token is stored in env var (set by setAuthToken in wasm.go) rather than JS globals
	// to avoid exposing credentials in the browser's window object.
	megaportTokenGlobal := js.Global().Get("megaportToken")
	if !megaportTokenGlobal.IsUndefined() && !megaportTokenGlobal.IsNull() {
		token := os.Getenv("MEGAPORT_ACCESS_TOKEN")
		// Prefer the environment bucket stored by setAuthToken; only fall back to
		// the page-writable global if the env var is unset. Used for the fallback
		// host selection below and debug logging.
		tokenEnv := os.Getenv("MEGAPORT_ENVIRONMENT")
		if tokenEnv == "" {
			tokenEnv = megaportTokenGlobal.Get("environment").String()
		}
		// Read the API base URL from the env var set (and hostname-validated) by
		// setAuthToken, never from window.megaportToken.apiURL: that global is
		// page-writable, so trusting it would let another script redirect the
		// bearer token to an attacker-controlled host.
		apiURL := os.Getenv("MEGAPORT_API_URL")
		// Real expiry, if the host supplied one via setAuthToken's 4th argument
		// (stored as epoch-ms on this global). Zero remains the fallback when
		// the host didn't supply one; WithAccessToken treats zero as
		// non-expiring, matching the historical behavior.
		expiry := wasm.ParseExpiry(megaportTokenGlobal.Get("expiry"))

		if token != "" {
			js.Global().Get("console").Call("log", "✅ Using external token from portal (bypassing OAuth flow)")
			js.Global().Get("console").Call("log", "Environment: "+tokenEnv)
			js.Global().Get("console").Call("log", "API URL: "+apiURL)

			// Create WASM HTTP client
			httpClient := wasmhttp.NewWasmHTTPClient()
			httpClient.Timeout = 45 * time.Second

			// Build client options - prefer the validated API URL if available
			var clientOpts []megaport.ClientOpt
			clientOpts = append(clientOpts, megaport.WithAccessToken(token, expiry))

			if apiURL != "" {
				// Use the validated API URL - this auto-works for new environments
				js.Global().Get("console").Call("log", "🔗 Using validated API URL: "+apiURL)
				clientOpts = append(clientOpts, megaport.WithBaseURL(apiURL))
			} else {
				// Fallback to environment-based URL selection
				js.Global().Get("console").Call("log", "⚠️ No API URL provided, falling back to environment-based selection")
				var envOpt megaport.ClientOpt
				switch tokenEnv {
				case "production":
					envOpt = megaport.WithEnvironment(megaport.EnvironmentProduction)
				case "staging":
					envOpt = megaport.WithEnvironment(megaport.EnvironmentStaging)
				case "development":
					envOpt = megaport.WithEnvironment(megaport.EnvironmentDevelopment)
				default:
					envOpt = megaport.WithEnvironment(megaport.EnvironmentProduction)
				}
				clientOpts = append(clientOpts, envOpt)
			}

			// Create Megaport client with the external token (no OAuth flow needed!)
			clientOpts = append(clientOpts, megaport.WithCustomHeaders(cliHeaders))
			if uid := resolveManagedAccountUID(); uid != "" {
				js.Global().Get("console").Call("log", "Acting on behalf of managed account: "+uid)
				clientOpts = append(clientOpts, megaport.WithCallContext(uid))
			}
			megaportClient, err := megaport.New(httpClient, clientOpts...)
			if err != nil {
				js.Global().Get("console").Call("error", "Failed to create Megaport client: "+err.Error())
				js.Global().Get("console").Call("groupEnd")
				return nil, fmt.Errorf("failed to create Megaport client: %w", err)
			}

			js.Global().Get("console").Call("log", "✅ Client created with external token - no OAuth needed!")
			js.Global().Get("console").Call("groupEnd")
			return megaportClient, nil
		}
	}

	// PRIORITY 2: Check for API Key/Secret credentials (uses OAuth flow).
	// setAuthCredentials() stores credentials in Go env vars and exposes only
	// the non-secret environment name in window.megaportCredentials, so always
	// read accessKey/secretKey from env vars rather than the JS global.
	accessKey = os.Getenv("MEGAPORT_ACCESS_KEY")
	secretKey = os.Getenv("MEGAPORT_SECRET_KEY")
	env = os.Getenv("MEGAPORT_ENVIRONMENT")

	if accessKey != "" {
		js.Global().Get("console").Call("log", "Using access key from environment variable")
		js.Global().Get("console").Call("log", "Access Key: "+maskCredential(accessKey))
	}
	if secretKey != "" {
		js.Global().Get("console").Call("log", "Using secret key from environment variable")
	}

	// Allow the megaportCredentials JS global's environment field to override
	// the env var (the UI may set it before calling setAuthCredentials).
	megaportCredsGlobal := js.Global().Get("megaportCredentials")
	if !megaportCredsGlobal.IsUndefined() && !megaportCredsGlobal.IsNull() {
		envVal := megaportCredsGlobal.Get("environment")
		if envVal.Type() == js.TypeString {
			if s := envVal.String(); s != "" {
				env = s
			}
		}
	}

	if env != "" {
		js.Global().Get("console").Call("log", "Environment: "+env)
	}

	// Validate credentials
	if accessKey == "" {
		js.Global().Get("console").Call("error", "No access key or token provided")
		js.Global().Get("console").Call("error", "💡 WASM Tip: Use setAuthToken() for portal tokens or setAuthCredentials() for API keys")
		js.Global().Get("console").Call("groupEnd")
		return nil, fmt.Errorf("megaport API access key not provided. Please use setAuthToken() for portal tokens or setAuthCredentials() for API keys")
	}
	if secretKey == "" {
		js.Global().Get("console").Call("error", "No secret key provided")
		js.Global().Get("console").Call("error", "💡 WASM Tip: Use the login form in the browser UI or set MEGAPORT_SECRET_KEY environment variable")
		js.Global().Get("console").Call("groupEnd")
		return nil, fmt.Errorf("megaport API secret key not provided. Please use the login form in the browser UI or set MEGAPORT_SECRET_KEY environment variable")
	}

	// Default to production
	if env == "" {
		env = "production"
		js.Global().Get("console").Call("log", "No environment specified, defaulting to production")
	}

	// Set environment option
	var envOpt megaport.ClientOpt
	var apiEndpoint string

	switch env {
	case "production":
		apiEndpoint = "https://api.megaport.com"
		envOpt = megaport.WithEnvironment(megaport.EnvironmentProduction)
	case "staging":
		apiEndpoint = "https://api-staging.megaport.com"
		envOpt = megaport.WithEnvironment(megaport.EnvironmentStaging)
	case "development":
		apiEndpoint = "https://api-mpone-dev.megaport.com"
		envOpt = megaport.WithEnvironment(megaport.EnvironmentDevelopment)
	default:
		apiEndpoint = "https://api.megaport.com"
		envOpt = megaport.WithEnvironment(megaport.EnvironmentProduction)
	}

	js.Global().Get("console").Call("log", "Using API endpoint: "+apiEndpoint)

	// Create WASM HTTP client that uses browser fetch API
	httpClient := wasmhttp.NewWasmHTTPClient()
	httpClient.Timeout = 45 * time.Second // Increase from default timeout

	js.Global().Get("console").Call("log", "HTTP client timeout set to: "+httpClient.Timeout.String())
	js.Global().Get("console").Call("log", "✨ Using WASM HTTP transport with browser fetch")

	// Create Megaport client with credentials
	js.Global().Get("console").Call("log", "Creating Megaport client...")
	clientOpts := []megaport.ClientOpt{
		megaport.WithCredentials(accessKey, secretKey),
		envOpt,
		megaport.WithCustomHeaders(cliHeaders),
	}
	if uid := resolveManagedAccountUID(); uid != "" {
		js.Global().Get("console").Call("log", "Acting on behalf of managed account: "+uid)
		clientOpts = append(clientOpts, megaport.WithCallContext(uid))
	}
	megaportClient, err := megaport.New(httpClient, clientOpts...)
	if err != nil {
		js.Global().Get("console").Call("error", "Failed to create Megaport client: "+err.Error())
		js.Global().Get("console").Call("groupEnd")
		return nil, fmt.Errorf("failed to create Megaport client: %w", err)
	}

	js.Global().Get("console").Call("log", "Megaport client created successfully")

	// Show login indicator
	spinner := output.PrintLoggingIn(false)

	// Create a new context with longer timeout specifically for auth
	authCtx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()

	js.Global().Get("console").Call("log", "Checking CORS capabilities...")
	js.Global().Get("console").Call("log", "Origin:", js.Global().Get("location").Get("origin").String())
	js.Global().Get("console").Call("log", "Target auth endpoint:", "https://auth-m2m.megaport.com/oauth2/token")

	js.Global().Get("console").Call("log", "Starting authorization with 30 second timeout")

	// Attempt authentication with retry logic
	_, err = RetryWithBackoffAndConsoleLogging(authCtx, 3, megaportClient)

	if err != nil {
		spinner.Stop()
		js.Global().Get("console").Call("error", "Authentication failed: "+err.Error())

		// Check if this might be a CORS error
		if isCORSError(err) {
			js.Global().Get("console").Call("error", "CORS issue detected")
			js.Global().Get("console").Call("groupEnd")
			return nil, fmt.Errorf("authentication failed due to CORS policy: this may be due to browser security restrictions. %w", err)
		}

		js.Global().Get("console").Call("groupEnd")
		return nil, fmt.Errorf("authentication failed after multiple attempts: %w", err)
	} else {
		js.Global().Get("console").Call("log", "✅ Authentication successful!")
		js.Global().Get("console").Call("groupEnd")
		spinner.StopWithSuccess("Successfully logged in to Megaport")
	}

	return megaportClient, nil
}

// Helper function to mask credentials for logging
func maskCredential(cred string) string {
	if len(cred) < 4 {
		return "****"
	}
	if len(cred) <= 8 {
		return cred[:2] + "..." + cred[len(cred)-2:]
	}
	return cred[:4] + "..." + cred[len(cred)-4:]
}

// Enhanced retryWithBackoff that includes console logging and token caching
func RetryWithBackoffAndConsoleLogging(ctx context.Context, attempts int, client *megaport.Client) (*megaport.AuthInfo, error) {
	// Get environment from client
	environment := "production"
	if client != nil && client.BaseURL != nil {
		switch client.BaseURL.Host {
		case "api.megaport.com":
			// environment already defaults to "production"
		case "api-staging.megaport.com":
			environment = "staging"
		case "api-mpone-dev.megaport.com":
			environment = "development"
		default:
			// Unrecognised host (e.g. a non-standard environment set up via
			// setAuthToken's environment override): log the production
			// assumption explicitly rather than silently bucketing the token
			// cache under production, which would collide with a real
			// production-cached token for a different environment.
			js.Global().Get("console").Call("warn", fmt.Sprintf(
				"unrecognised API host %q for token-cache environment bucketing; assuming production", client.BaseURL.Host))
		}
	} else {
		return nil, errors.New("megaport client is nil or has no valid BaseURL")
	}

	// First try to get a cached token
	if cachedAuth := CheckCachedToken(environment); cachedAuth != nil {
		js.Global().Get("console").Call("log", "Using cached authentication token")
		return cachedAuth, nil
	}

	// If no valid cached token, proceed with authentication attempts
	var err error
	for i := 0; i < attempts; i++ {
		// Log attempt information
		if i == 0 {
			js.Global().Get("console").Call("log", "Initial authentication attempt using proxy...")
		} else {
			js.Global().Get("console").Call("log", fmt.Sprintf("Retry attempt %d of %d...", i+1, attempts))
		}

		// Create a longer timeout for each attempt
		authCtx, cancel := context.WithTimeout(ctx, 45*time.Second)

		// Attempt the authorization
		startTime := time.Now()

		// Use the SDK's built-in Authorize() method with our WASM HTTP transport
		authInfo, err := client.Authorize(authCtx)

		// Must cancel the context after use
		cancel()

		elapsed := time.Since(startTime)

		if err == nil {
			js.Global().Get("console").Call("log", fmt.Sprintf("Proxy authorization successful (took %v)", elapsed))

			// Verify we actually got a token
			if authInfo != nil && authInfo.AccessToken != "" {
				js.Global().Get("console").Call("log", "Valid access token received")
				return authInfo, nil
			} else {
				js.Global().Get("console").Call("error", "No valid token in auth info")
				err = errors.New("no valid token received")
			}
		}

		// Log error details
		js.Global().Get("console").Call("error", fmt.Sprintf("Authorization failed (took %v): %s", elapsed, err.Error()))

		// Check if we should abort early
		if strings.Contains(err.Error(), "invalid_client") ||
			strings.Contains(err.Error(), "unauthorized") {
			js.Global().Get("console").Call("warn", "Credential error detected - aborting retry attempts")
			return nil, err
		}

		// Don't wait after the last attempt
		if i < attempts-1 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			waitTime := time.Duration(500*(1<<i)) * time.Millisecond
			js.Global().Get("console").Call("log", fmt.Sprintf("Waiting %v before next attempt", waitTime))
			timer := time.NewTimer(waitTime)
			select {
			case <-timer.C:
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return nil, ctx.Err()
			}
		}
	}
	return nil, err
}

// isCORSError detects common CORS-related error patterns
func isCORSError(err error) bool {
	// Network errors in browsers often don't have detailed information
	// but we can look for common patterns
	errStr := err.Error()
	return strings.Contains(errStr, "CORS") ||
		strings.Contains(errStr, "cross-origin") ||
		strings.Contains(errStr, "access control") ||
		strings.Contains(errStr, "blocked by") ||
		strings.Contains(errStr, "not allowed") ||
		strings.Contains(errStr, "No 'Access-Control-Allow-Origin'")
}

// checkCachedToken tries to get a cached token from the browser
func CheckCachedToken(environment string) *megaport.AuthInfo {
	js.Global().Get("console").Call("log", "Checking for cached token...")

	if js.Global().Get("tokenManager").IsUndefined() {
		js.Global().Get("console").Call("warn", "Token manager not available")
		return nil
	}

	// Get token from token manager. The host may return either a bare token
	// string (legacy contract, no TTL info) or an object {token, expiry}
	// where expiry is an epoch-ms number or RFC3339 string (see
	// wasm.ParseExpiry) — the object form lets the cache honor the token's
	// real TTL instead of guessing.
	result := js.Global().Get("tokenManager").Call("getToken", environment)

	if result.IsNull() || result.IsUndefined() {
		js.Global().Get("console").Call("log", "No valid cached token found")
		return nil
	}

	var tokenStr string
	var expiry time.Time
	switch result.Type() {
	case js.TypeString:
		tokenStr = result.String()
	case js.TypeObject:
		if tokenVal := result.Get("token"); tokenVal.Type() == js.TypeString {
			tokenStr = tokenVal.String()
		}
		expiry = wasm.ParseExpiry(result.Get("expiry"))
	}

	if tokenStr == "" {
		js.Global().Get("console").Call("log", "No valid cached token found")
		return nil
	}

	js.Global().Get("console").Call("log", "Found cached token")

	// Fall back to the historical 24h assumption only when the host didn't
	// supply a real TTL (legacy bare-string contract). The token manager
	// already checked the token is valid, so this is a display/renewal
	// timing heuristic, not a validity check.
	if expiry.IsZero() {
		expiry = time.Now().Add(24 * time.Hour)
	}

	return &megaport.AuthInfo{
		AccessToken: tokenStr,
		Expiration:  expiry,
	}
}

// clearCachedToken removes any stored token
func ClearCachedToken() {
	if !js.Global().Get("tokenManager").IsUndefined() {
		js.Global().Get("tokenManager").Call("clearToken")
	}
}

func Logout() {
	ClearCachedToken()
	js.Global().Get("console").Call("log", "User logged out and tokens cleared")
}

// newUnauthenticatedClientFunc creates a Megaport API client without authentication.
// Used for public API endpoints (e.g., locations) that don't require credentials.
var newUnauthenticatedClientFunc = func() (*megaport.Client, error) {
	var clientOpts []megaport.ClientOpt

	// Prefer the API base URL validated and stored by setAuthToken. Read it from
	// the env var, never from the page-writable window.megaportToken.apiURL
	// global, which another script could overwrite to redirect API traffic.
	apiURL := os.Getenv("MEGAPORT_API_URL")

	if apiURL != "" {
		clientOpts = append(clientOpts, megaport.WithBaseURL(apiURL))
	} else {
		// Fall back to environment-based URL selection. Prefer the bucket stored
		// by setAuthToken/setAuthCredentials; only consult the page-writable
		// megaportCredentials global if the env var is unset.
		env := os.Getenv("MEGAPORT_ENVIRONMENT")
		if env == "" {
			megaportCredsGlobal := js.Global().Get("megaportCredentials")
			if !megaportCredsGlobal.IsUndefined() && !megaportCredsGlobal.IsNull() {
				envVal := megaportCredsGlobal.Get("environment")
				if envVal.Type() == js.TypeString {
					env = envVal.String()
				}
			}
		}
		clientOpts = append(clientOpts, environmentOption(normalizeEnvironment(env)))
	}

	httpClient := wasmhttp.NewWasmHTTPClient()
	httpClient.Timeout = 45 * time.Second

	clientOpts = append(clientOpts, megaport.WithCustomHeaders(cliHeaders))
	return megaport.New(httpClient, clientOpts...)
}

// NewUnauthenticatedClient creates an unauthenticated Megaport API client.
func NewUnauthenticatedClient() (*megaport.Client, error) {
	return GetNewUnauthenticatedClientFunc()()
}
