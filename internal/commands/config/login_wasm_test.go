//go:build js && wasm

package config

import (
	"context"
	"net/http"
	"os"
	"syscall/js"
	"testing"
	"time"

	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoginFunc_UnknownEnvironmentFailsClosed verifies that the credential
// login path rejects an unrecognized MEGAPORT_ENVIRONMENT instead of
// silently coercing it to production, since accessKey/secretKey callers
// only reach this switch via setAuthCredentials, which is expected to have
// already bucketed the value into production/staging/development.
func TestLoginFunc_UnknownEnvironmentFailsClosed(t *testing.T) {
	origAccessKey := os.Getenv("MEGAPORT_ACCESS_KEY")
	origSecretKey := os.Getenv("MEGAPORT_SECRET_KEY")
	origEnv := os.Getenv("MEGAPORT_ENVIRONMENT")
	defer func() {
		os.Setenv("MEGAPORT_ACCESS_KEY", origAccessKey)
		os.Setenv("MEGAPORT_SECRET_KEY", origSecretKey)
		os.Setenv("MEGAPORT_ENVIRONMENT", origEnv)
	}()

	js.Global().Delete("megaportToken")
	js.Global().Delete("megaportCredentials")

	os.Setenv("MEGAPORT_ACCESS_KEY", "test-access-key")
	os.Setenv("MEGAPORT_SECRET_KEY", "test-secret-key")
	os.Setenv("MEGAPORT_ENVIRONMENT", "not-a-real-environment")

	client, err := loginFunc(context.Background())

	assert.Nil(t, client, "no client should be created for an unrecognized environment")
	assert.ErrorContains(t, err, "unknown environment")
	assert.ErrorContains(t, err, "not-a-real-environment")
}

// TestLoginFunc_TokenPathUnknownEnvironmentFailsClosed verifies the bearer-token
// login path also fails closed on an unrecognized environment instead of silently
// routing the token to production, matching the credential path's behavior.
func TestLoginFunc_TokenPathUnknownEnvironmentFailsClosed(t *testing.T) {
	t.Setenv("MEGAPORT_ACCESS_TOKEN", "test-token-12345")
	t.Setenv("MEGAPORT_API_URL", "")
	t.Setenv("MEGAPORT_ENVIRONMENT", "not-a-real-environment")

	setTamperedTokenGlobal("not-a-real-environment", "")
	defer js.Global().Delete("megaportToken")

	client, err := loginFunc(context.Background())

	assert.Nil(t, client, "no client should be created for an unrecognized token environment")
	assert.ErrorContains(t, err, "unknown environment")
	assert.ErrorContains(t, err, "not-a-real-environment")
}

// TestLoginFunc_CredentialGlobalDoesNotOverrideValidatedEnv verifies the
// credential path ignores the page-writable megaportCredentials global when the
// trusted MEGAPORT_ENVIRONMENT bucket is already set, so a script that can write
// the global cannot redirect validated credentials to a different environment.
func TestLoginFunc_CredentialGlobalDoesNotOverrideValidatedEnv(t *testing.T) {
	js.Global().Delete("megaportToken")

	t.Setenv("MEGAPORT_ACCESS_TOKEN", "")
	t.Setenv("MEGAPORT_API_URL", "")
	t.Setenv("MEGAPORT_ACCESS_KEY", "test-access-key")
	t.Setenv("MEGAPORT_SECRET_KEY", "test-secret-key")
	t.Setenv("MEGAPORT_ENVIRONMENT", "staging")

	// Global claims production; the validated env-var bucket (staging) must win.
	credsObj := js.Global().Get("Object").New()
	credsObj.Set("environment", "production")
	js.Global().Set("megaportCredentials", credsObj)
	defer js.Global().Delete("megaportCredentials")

	// A cached token short-circuits the network Authorize() call.
	setTokenManager(t, "cached-token")

	// The credential success path logs location.origin, absent in the node
	// test host; stub it so loginFunc can run to completion.
	loc := js.Global().Get("Object").New()
	loc.Set("origin", "https://portal.megaport.com")
	js.Global().Set("location", loc)
	defer js.Global().Delete("location")

	client, err := loginFunc(context.Background())
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.BaseURL)

	assert.Equal(t, "api-staging.megaport.com", client.BaseURL.Host,
		"credential login must use the validated env-var bucket, not the page-writable global")
}

// setTamperedTokenGlobal publishes a window.megaportToken global whose apiURL
// field has been overwritten to an attacker-controlled host, mimicking a script
// tampering with the page-writable global after a legitimate setAuthToken call.
func setTamperedTokenGlobal(env, apiURL string) {
	obj := js.Global().Get("Object").New()
	obj.Set("environment", env)
	obj.Set("apiURL", apiURL)
	js.Global().Set("megaportToken", obj)
}

// TestLoginReadsAPIURLFromEnvNotTamperedGlobal asserts the authenticated token
// login path routes to the host stored in MEGAPORT_API_URL by setAuthToken, even
// when window.megaportToken.apiURL has been overwritten with a hostile value.
func TestLoginReadsAPIURLFromEnvNotTamperedGlobal(t *testing.T) {
	const validHost = "api-staging.megaport.com"
	const tamperedURL = "https://evil.attacker.com"

	t.Setenv("MEGAPORT_ACCESS_TOKEN", "test-token-12345")
	t.Setenv("MEGAPORT_API_URL", "https://"+validHost)

	setTamperedTokenGlobal("staging", tamperedURL)
	defer js.Global().Delete("megaportToken")

	client, err := loginFunc(context.Background())
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.BaseURL)

	assert.Equal(t, validHost, client.BaseURL.Host,
		"login must route to the validated env-var host, not the tampered global")
}

// TestLoginFallbackEnvIgnoresTamperedGlobal asserts that when MEGAPORT_API_URL
// is unset and login falls back to environment-based host selection, it uses the
// bucket stored in MEGAPORT_ENVIRONMENT rather than the page-writable global.
func TestLoginFallbackEnvIgnoresTamperedGlobal(t *testing.T) {
	t.Setenv("MEGAPORT_ACCESS_TOKEN", "test-token-12345")
	t.Setenv("MEGAPORT_API_URL", "")
	t.Setenv("MEGAPORT_ENVIRONMENT", "staging")

	// Global claims production; the env-var bucket (staging) must win.
	setTamperedTokenGlobal("production", "https://evil.attacker.com")
	defer js.Global().Delete("megaportToken")

	client, err := loginFunc(context.Background())
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.BaseURL)

	assert.Equal(t, "api-staging.megaport.com", client.BaseURL.Host,
		"fallback host selection must use the env-var bucket, not the tampered global")
}

// TestUnauthenticatedClientReadsAPIURLFromEnvNotTamperedGlobal asserts the same
// for the unauthenticated client factory used by public endpoints.
func TestUnauthenticatedClientReadsAPIURLFromEnvNotTamperedGlobal(t *testing.T) {
	const validHost = "api-staging.megaport.com"
	const tamperedURL = "https://evil.attacker.com"

	t.Setenv("MEGAPORT_API_URL", "https://"+validHost)

	setTamperedTokenGlobal("staging", tamperedURL)
	defer js.Global().Delete("megaportToken")

	client, err := newUnauthenticatedClientFunc()
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.BaseURL)

	assert.Equal(t, validHost, client.BaseURL.Host,
		"unauthenticated client must route to the validated env-var host, not the tampered global")
}

// TestUnauthenticatedClientFallbackEnvIgnoresTamperedGlobal asserts that when
// MEGAPORT_API_URL is unset and the unauthenticated client falls back to
// environment-based host selection, it prefers the MEGAPORT_ENVIRONMENT bucket
// over the page-writable megaportCredentials global.
func TestUnauthenticatedClientFallbackEnvIgnoresTamperedGlobal(t *testing.T) {
	t.Setenv("MEGAPORT_API_URL", "")
	t.Setenv("MEGAPORT_ENVIRONMENT", "staging")

	// Global claims production; the env-var bucket (staging) must win.
	credsObj := js.Global().Get("Object").New()
	credsObj.Set("environment", "production")
	js.Global().Set("megaportCredentials", credsObj)
	defer js.Global().Delete("megaportCredentials")

	client, err := newUnauthenticatedClientFunc()
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.BaseURL)

	assert.Equal(t, "api-staging.megaport.com", client.BaseURL.Host,
		"fallback host selection must use the env-var bucket, not the tampered global")
}

// TestUnauthenticatedClientFallbackUsesGlobalWhenEnvUnset asserts the
// megaportCredentials global still drives host selection when neither
// MEGAPORT_API_URL nor MEGAPORT_ENVIRONMENT is set.
func TestUnauthenticatedClientFallbackUsesGlobalWhenEnvUnset(t *testing.T) {
	t.Setenv("MEGAPORT_API_URL", "")
	t.Setenv("MEGAPORT_ENVIRONMENT", "")

	credsObj := js.Global().Get("Object").New()
	credsObj.Set("environment", "staging")
	js.Global().Set("megaportCredentials", credsObj)
	defer js.Global().Delete("megaportCredentials")

	client, err := newUnauthenticatedClientFunc()
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.BaseURL)

	assert.Equal(t, "api-staging.megaport.com", client.BaseURL.Host)
}

// setTokenManager installs a mock window.tokenManager whose getToken always
// returns result, restoring the previous global (if any) after the test.
func setTokenManager(t *testing.T, result interface{}) {
	t.Helper()
	prev := js.Global().Get("tokenManager")
	tm := js.Global().Get("Object").New()
	tm.Set("getToken", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return result
	}))
	js.Global().Set("tokenManager", tm)
	t.Cleanup(func() {
		if prev.IsUndefined() {
			js.Global().Delete("tokenManager")
		} else {
			js.Global().Set("tokenManager", prev)
		}
	})
}

func TestCheckCachedToken_NoTokenManager(t *testing.T) {
	prev := js.Global().Get("tokenManager")
	js.Global().Delete("tokenManager")
	defer func() {
		if !prev.IsUndefined() {
			js.Global().Set("tokenManager", prev)
		}
	}()

	assert.Nil(t, CheckCachedToken("production"))
}

func TestCheckCachedToken_NoCachedToken(t *testing.T) {
	setTokenManager(t, js.Null())
	assert.Nil(t, CheckCachedToken("production"))
}

func TestCheckCachedToken_LegacyStringFallsBackTo24h(t *testing.T) {
	setTokenManager(t, "legacy-bare-token")

	before := time.Now()
	auth := CheckCachedToken("production")
	require.NotNil(t, auth)
	assert.Equal(t, "legacy-bare-token", auth.AccessToken)

	// Historical fallback: ~24h from now, not zero and not the token's own TTL
	// (there isn't one on the legacy contract).
	assert.WithinDuration(t, before.Add(24*time.Hour), auth.Expiration, 5*time.Second)
}

func TestCheckCachedToken_ObjectWithExpiryHonorsRealTTL(t *testing.T) {
	want := time.Date(2031, 3, 4, 5, 6, 7, 0, time.UTC)

	tokenObj := js.Global().Get("Object").New()
	tokenObj.Set("token", "object-form-token")
	tokenObj.Set("expiry", float64(want.UnixMilli()))
	setTokenManager(t, tokenObj)

	auth := CheckCachedToken("production")
	require.NotNil(t, auth)
	assert.Equal(t, "object-form-token", auth.AccessToken)
	assert.True(t, want.Equal(auth.Expiration), "expiration = %v, want %v", auth.Expiration, want)
}

func TestCheckCachedToken_ObjectWithoutExpiryFallsBackTo24h(t *testing.T) {
	tokenObj := js.Global().Get("Object").New()
	tokenObj.Set("token", "object-form-token-no-expiry")
	setTokenManager(t, tokenObj)

	before := time.Now()
	auth := CheckCachedToken("production")
	require.NotNil(t, auth)
	assert.WithinDuration(t, before.Add(24*time.Hour), auth.Expiration, 5*time.Second)
}

func TestCheckCachedToken_ObjectWithEmptyTokenIsTreatedAsNoToken(t *testing.T) {
	tokenObj := js.Global().Get("Object").New()
	setTokenManager(t, tokenObj)

	assert.Nil(t, CheckCachedToken("production"))
}

func TestRetryWithBackoffAndConsoleLogging_NilClient(t *testing.T) {
	_, err := RetryWithBackoffAndConsoleLogging(context.Background(), 1, nil)
	require.Error(t, err)
}

// TestRetryWithBackoffAndConsoleLogging_UnrecognisedHostUsesCache verifies that
// a non-standard API host still buckets to production for the token cache
// (rather than erroring) and, critically, that the cached-token short-circuit
// still fires before any network Authorize() call would happen.
func TestRetryWithBackoffAndConsoleLogging_UnrecognisedHostUsesCache(t *testing.T) {
	setTokenManager(t, "cached-for-unrecognised-host")

	client, err := megaport.New(http.DefaultClient, megaport.WithBaseURL("https://api-custom-env.example.com"))
	require.NoError(t, err)

	auth, err := RetryWithBackoffAndConsoleLogging(context.Background(), 1, client)
	require.NoError(t, err)
	require.NotNil(t, auth)
	assert.Equal(t, "cached-for-unrecognised-host", auth.AccessToken)
}
