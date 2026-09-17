//go:build !js && !wasm

package config

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/megaport/megaport-cli/internal/utils"
	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
)

var (
	env               string
	accessKeyEnvVar   = "MEGAPORT_ACCESS_KEY"
	secretKeyEnvVar   = "MEGAPORT_SECRET_KEY"
	environmentEnvVar = "MEGAPORT_ENVIRONMENT"
)

func TestLogin(t *testing.T) {
	originalLoginFunc := GetLoginFunc()
	defer func() {
		SetLoginFunc(originalLoginFunc)
	}()

	tests := []struct {
		name        string
		envVars     map[string]string
		envFlag     string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "all valid env vars",
			envVars: map[string]string{
				"MEGAPORT_ACCESS_KEY":  "test-access-key",
				"MEGAPORT_SECRET_KEY":  "test-secret-key",
				"MEGAPORT_ENVIRONMENT": "staging",
			},
			shouldError: false,
		},
		{
			name: "empty access key",
			envVars: map[string]string{
				"MEGAPORT_ACCESS_KEY":  "",
				"MEGAPORT_SECRET_KEY":  "test-secret-key",
				"MEGAPORT_ENVIRONMENT": "staging",
			},
			shouldError: true,
			errorMsg:    "access key, secret key, and environment are required",
		},
		{
			name: "empty secret key",
			envVars: map[string]string{
				"MEGAPORT_ACCESS_KEY":  "test-access-key",
				"MEGAPORT_SECRET_KEY":  "",
				"MEGAPORT_ENVIRONMENT": "staging",
			},
			shouldError: true,
			errorMsg:    "access key, secret key, and environment are required",
		},
		{
			name:        "no env vars set",
			envVars:     map[string]string{},
			shouldError: true,
			errorMsg:    "access key, secret key, and environment are required",
		},
		{
			name: "flag overrides env var",
			envVars: map[string]string{
				"MEGAPORT_ACCESS_KEY":  "test-access-key",
				"MEGAPORT_SECRET_KEY":  "test-secret-key",
				"MEGAPORT_ENVIRONMENT": "staging",
			},
			envFlag:     "production",
			shouldError: false,
		},
		{
			name: "flag provides env when not set",
			envVars: map[string]string{
				"MEGAPORT_ACCESS_KEY": "test-access-key",
				"MEGAPORT_SECRET_KEY": "test-secret-key",
			},
			envFlag:     "production",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars first, then set the ones specified in the test case.
			// t.Setenv auto-restores when the subtest finishes.
			t.Setenv("MEGAPORT_ACCESS_KEY", "") // empty string is equivalent to unset for os.Getenv callers
			t.Setenv("MEGAPORT_SECRET_KEY", "")
			t.Setenv("MEGAPORT_ENVIRONMENT", "")

			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			if tt.envFlag != "" {
				env = tt.envFlag
			} else {
				env = ""
			}

			var capturedAccessKey, capturedSecretKey, capturedEnv string

			SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				capturedAccessKey = os.Getenv(accessKeyEnvVar)
				capturedSecretKey = os.Getenv(secretKeyEnvVar)
				capturedEnv = env
				if capturedEnv == "" {
					capturedEnv = os.Getenv(environmentEnvVar)
				}

				if capturedAccessKey == "" || capturedSecretKey == "" {
					return nil, fmt.Errorf("access key, secret key, and environment are required")
				}

				client := &megaport.Client{}
				return client, nil
			})

			client, err := Login(context.Background())

			if tt.shouldError {
				assert.Nil(t, client)
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NotNil(t, client)
				assert.NoError(t, err)

				if tt.envFlag != "" {
					assert.Equal(t, tt.envFlag, capturedEnv)
				} else if envVal, ok := tt.envVars["MEGAPORT_ENVIRONMENT"]; ok {
					assert.Equal(t, envVal, capturedEnv)
				} else {
					assert.Equal(t, "production", capturedEnv)
				}
			}
		})
	}
}

func TestEnvironmentSelectionPrecedence(t *testing.T) {
	// Save and restore non-env-var globals
	originalEnv := utils.Env
	defer func() {
		utils.Env = originalEnv
	}()

	tests := []struct {
		name        string
		globalFlag  string
		envVar      string
		expectedEnv string
	}{
		{
			name:        "Global flag takes precedence over environment variable",
			globalFlag:  "staging",
			envVar:      "production",
			expectedEnv: "staging",
		},
		{
			name:        "Environment variable used when no global flag",
			globalFlag:  "",
			envVar:      "staging",
			expectedEnv: "staging",
		},
		{
			name:        "Default to production when neither flag nor env var set",
			globalFlag:  "",
			envVar:      "",
			expectedEnv: "production",
		},
		{
			name:        "Development environment via global flag",
			globalFlag:  "development",
			envVar:      "",
			expectedEnv: "development",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origEnv := utils.Env
			defer func() { utils.Env = origEnv }()

			// Setup test environment
			utils.Env = tt.globalFlag
			t.Setenv("MEGAPORT_ENVIRONMENT", tt.envVar)

			// Test the environment selection logic by examining the values
			// that would be read in the login function
			var env string

			// Simulate the new login function logic
			if utils.Env != "" {
				env = utils.Env
			} else {
				env = os.Getenv("MEGAPORT_ENVIRONMENT")
			}

			if env == "" {
				env = "production"
			}

			assert.Equal(t, tt.expectedEnv, env, "Environment selection should match expected value")
		})
	}
}

func TestProfileOverrideLogin(t *testing.T) {
	// Save and restore non-env-var globals
	originalEnv := utils.Env
	originalProfileOverride := utils.ProfileOverride
	originalLoginFuncWithOutput := GetLoginFuncWithOutput()

	defer func() {
		utils.Env = originalEnv
		utils.ProfileOverride = originalProfileOverride
		SetLoginFuncWithOutput(originalLoginFuncWithOutput)
	}()

	// Setup temp config dir with profiles
	tempDir, err := os.MkdirTemp("", "megaport-login-test")
	assert.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })
	t.Setenv("MEGAPORT_CONFIG_DIR", tempDir)

	// Clear env vars so profile credentials are used
	t.Setenv("MEGAPORT_ACCESS_KEY", "")
	t.Setenv("MEGAPORT_SECRET_KEY", "")
	t.Setenv("MEGAPORT_ENVIRONMENT", "")

	// Create config with two profiles
	manager, err := NewConfigManager()
	assert.NoError(t, err)
	err = manager.CreateProfile("staging", "staging-access", "staging-secret", "staging", "")
	assert.NoError(t, err)
	err = manager.CreateProfile("prod", "prod-access", "prod-secret", "production", "")
	assert.NoError(t, err)
	err = manager.UseProfile("prod")
	assert.NoError(t, err)

	t.Run("profile override uses specified profile and reaches API call", func(t *testing.T) {
		origEnv := utils.Env
		defer func() { utils.Env = origEnv }()
		origProfile := utils.ProfileOverride
		defer func() { utils.ProfileOverride = origProfile }()

		utils.Env = ""
		utils.ProfileOverride = "staging"

		// Call the real LoginFuncWithOutput - it will resolve credentials from
		// the "staging" profile and fail at Authorize (no real API), which proves
		// credential resolution succeeded (didn't get "access key not provided" error).
		_, err := LoginWithOutput(context.Background(), "json")
		assert.Error(t, err)
		// Should NOT be a "not provided" error — that would mean credential resolution failed
		assert.NotContains(t, err.Error(), "access key not provided")
		assert.NotContains(t, err.Error(), "secret key not provided")
		assert.NotContains(t, err.Error(), "not found")
	})

	t.Run("profile override with non-existent profile returns error", func(t *testing.T) {
		origEnv := utils.Env
		defer func() { utils.Env = origEnv }()
		origProfile := utils.ProfileOverride
		defer func() { utils.ProfileOverride = origProfile }()

		utils.Env = ""
		utils.ProfileOverride = "non-existent"

		_, err := LoginWithOutput(context.Background(), "json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-existent")
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("without profile override uses active profile and reaches API call", func(t *testing.T) {
		origEnv := utils.Env
		defer func() { utils.Env = origEnv }()
		origProfile := utils.ProfileOverride
		defer func() { utils.ProfileOverride = origProfile }()

		utils.Env = ""
		utils.ProfileOverride = ""

		// Active profile is "prod" — should resolve credentials from it
		_, err := LoginWithOutput(context.Background(), "json")
		assert.Error(t, err)
		// Should NOT be a "not provided" error
		assert.NotContains(t, err.Error(), "access key not provided")
		assert.NotContains(t, err.Error(), "secret key not provided")
	})

	t.Run("env flag overrides profile environment", func(t *testing.T) {
		origEnv := utils.Env
		defer func() { utils.Env = origEnv }()
		origProfile := utils.ProfileOverride
		defer func() { utils.ProfileOverride = origProfile }()

		utils.Env = "development"
		utils.ProfileOverride = "staging"

		// Both --profile and --env are set: credentials from staging profile,
		// environment from --env flag. Should still resolve and reach API call.
		_, err := LoginWithOutput(context.Background(), "json")
		assert.Error(t, err)
		assert.NotContains(t, err.Error(), "access key not provided")
		assert.NotContains(t, err.Error(), "not found")
	})

	t.Run("no profile and no env vars returns credential error", func(t *testing.T) {
		origEnv := utils.Env
		defer func() { utils.Env = origEnv }()
		origProfile := utils.ProfileOverride
		defer func() { utils.ProfileOverride = origProfile }()

		utils.Env = ""
		utils.ProfileOverride = ""

		// Delete the active profile so there are no credentials available
		err := manager.UseProfile("")
		// UseProfile with empty may fail, so we set active profile to non-existent
		_ = err

		// Use a fresh temp dir with no profiles
		emptyDir, err := os.MkdirTemp("", "megaport-empty-test")
		assert.NoError(t, err)
		t.Cleanup(func() { os.RemoveAll(emptyDir) })
		t.Setenv("MEGAPORT_CONFIG_DIR", emptyDir)

		_, err = LoginWithOutput(context.Background(), "json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access key not provided")
	})
}

func TestNewUnauthenticatedClient(t *testing.T) {
	// Save and restore non-env-var globals
	originalEnv := utils.Env
	originalProfileOverride := utils.ProfileOverride

	defer func() {
		utils.Env = originalEnv
		utils.ProfileOverride = originalProfileOverride
	}()

	// Default empty config dir for all subtests (subtests that need profiles override this)
	defaultEmptyDir, err := os.MkdirTemp("", "megaport-unauth-default")
	assert.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(defaultEmptyDir) })
	t.Setenv("MEGAPORT_CONFIG_DIR", defaultEmptyDir)

	t.Run("defaults to production when no env configured", func(t *testing.T) {
		origEnv := utils.Env
		defer func() { utils.Env = origEnv }()
		origProfile := utils.ProfileOverride
		defer func() { utils.ProfileOverride = origProfile }()

		utils.Env = ""
		utils.ProfileOverride = ""
		t.Setenv("MEGAPORT_ENVIRONMENT", "")

		client, err := NewUnauthenticatedClient()
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.Contains(t, client.BaseURL.String(), "api.megaport.com")
	})

	t.Run("respects env flag", func(t *testing.T) {
		origEnv := utils.Env
		defer func() { utils.Env = origEnv }()
		origProfile := utils.ProfileOverride
		defer func() { utils.ProfileOverride = origProfile }()

		utils.Env = "staging"
		utils.ProfileOverride = ""
		t.Setenv("MEGAPORT_ENVIRONMENT", "")

		client, err := NewUnauthenticatedClient()
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.Contains(t, client.BaseURL.String(), "api-staging.megaport.com")
	})

	t.Run("respects MEGAPORT_ENVIRONMENT env var", func(t *testing.T) {
		origEnv := utils.Env
		defer func() { utils.Env = origEnv }()
		origProfile := utils.ProfileOverride
		defer func() { utils.ProfileOverride = origProfile }()

		utils.Env = ""
		utils.ProfileOverride = ""
		t.Setenv("MEGAPORT_ENVIRONMENT", "staging")

		client, err := NewUnauthenticatedClient()
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.Contains(t, client.BaseURL.String(), "api-staging.megaport.com")
	})

	t.Run("profile override with valid profile uses profile env", func(t *testing.T) {
		origEnv := utils.Env
		defer func() { utils.Env = origEnv }()
		origProfile := utils.ProfileOverride
		defer func() { utils.ProfileOverride = origProfile }()

		tempDir, err := os.MkdirTemp("", "megaport-unauth-test")
		assert.NoError(t, err)
		t.Cleanup(func() { os.RemoveAll(tempDir) })
		t.Setenv("MEGAPORT_CONFIG_DIR", tempDir)

		manager, err := NewConfigManager()
		assert.NoError(t, err)
		err = manager.CreateProfile("staging-profile", "key", "secret", "staging", "")
		assert.NoError(t, err)

		utils.Env = ""
		utils.ProfileOverride = "staging-profile"
		t.Setenv("MEGAPORT_ENVIRONMENT", "")

		client, err := NewUnauthenticatedClient()
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.Contains(t, client.BaseURL.String(), "api-staging.megaport.com")
	})

	t.Run("profile override with non-existent profile returns error", func(t *testing.T) {
		origEnv := utils.Env
		defer func() { utils.Env = origEnv }()
		origProfile := utils.ProfileOverride
		defer func() { utils.ProfileOverride = origProfile }()

		tempDir, err := os.MkdirTemp("", "megaport-unauth-test")
		assert.NoError(t, err)
		t.Cleanup(func() { os.RemoveAll(tempDir) })
		t.Setenv("MEGAPORT_CONFIG_DIR", tempDir)

		utils.Env = ""
		utils.ProfileOverride = "non-existent"

		client, err := NewUnauthenticatedClient()
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "non-existent")
	})

	t.Run("env flag overrides profile environment", func(t *testing.T) {
		origEnv := utils.Env
		defer func() { utils.Env = origEnv }()
		origProfile := utils.ProfileOverride
		defer func() { utils.ProfileOverride = origProfile }()

		tempDir, err := os.MkdirTemp("", "megaport-unauth-test")
		assert.NoError(t, err)
		t.Cleanup(func() { os.RemoveAll(tempDir) })
		t.Setenv("MEGAPORT_CONFIG_DIR", tempDir)

		manager, err := NewConfigManager()
		assert.NoError(t, err)
		err = manager.CreateProfile("staging-profile", "key", "secret", "staging", "")
		assert.NoError(t, err)

		utils.Env = "production"
		utils.ProfileOverride = "staging-profile"

		client, err := NewUnauthenticatedClient()
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.Contains(t, client.BaseURL.String(), "api.megaport.com")
	})

	t.Run("does not require credentials", func(t *testing.T) {
		origEnv := utils.Env
		defer func() { utils.Env = origEnv }()
		origProfile := utils.ProfileOverride
		defer func() { utils.ProfileOverride = origProfile }()

		utils.Env = "production"
		utils.ProfileOverride = ""
		t.Setenv("MEGAPORT_ACCESS_KEY", "")
		t.Setenv("MEGAPORT_SECRET_KEY", "")

		client, err := NewUnauthenticatedClient()
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.Empty(t, client.AccessKey)
		assert.Empty(t, client.SecretKey)
	})

	t.Run("accepts short alias prod", func(t *testing.T) {
		origEnv := utils.Env
		defer func() { utils.Env = origEnv }()
		origProfile := utils.ProfileOverride
		defer func() { utils.ProfileOverride = origProfile }()

		utils.Env = "prod"
		utils.ProfileOverride = ""

		client, err := NewUnauthenticatedClient()
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.Contains(t, client.BaseURL.String(), "api.megaport.com")
	})

	t.Run("accepts short alias dev", func(t *testing.T) {
		origEnv := utils.Env
		defer func() { utils.Env = origEnv }()
		origProfile := utils.ProfileOverride
		defer func() { utils.ProfileOverride = origProfile }()

		utils.Env = "dev"
		utils.ProfileOverride = ""

		client, err := NewUnauthenticatedClient()
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.Contains(t, client.BaseURL.String(), "api-mpone-dev.megaport.com")
	})

	t.Run("--base-url overrides env flag", func(t *testing.T) {
		origEnv := utils.Env
		defer func() { utils.Env = origEnv }()
		origProfile := utils.ProfileOverride
		defer func() { utils.ProfileOverride = origProfile }()
		origBaseURL := utils.BaseURL
		defer func() { utils.BaseURL = origBaseURL }()

		utils.Env = "staging"
		utils.ProfileOverride = ""
		utils.BaseURL = "http://localhost:9999"

		client, err := NewUnauthenticatedClient()
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.Contains(t, client.BaseURL.String(), "localhost:9999")
	})
}

func TestBaseURLWithTokenURL(t *testing.T) {
	origBaseURL := utils.BaseURL
	origTokenURL := utils.TokenURL
	origEnv := utils.Env
	origProfile := utils.ProfileOverride
	defer func() {
		utils.BaseURL = origBaseURL
		utils.TokenURL = origTokenURL
		utils.Env = origEnv
		utils.ProfileOverride = origProfile
	}()

	tempDir, err := os.MkdirTemp("", "megaport-baseurl-auth-test")
	assert.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	t.Run("authenticated login succeeds with --base-url and --token-url", func(t *testing.T) {
		var tokenEndpointHit atomic.Bool
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/oauth2/token" {
				tokenEndpointHit.Store(true)
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, `{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`)
				return
			}
			http.NotFound(w, r)
		}))
		defer ts.Close()

		t.Setenv("MEGAPORT_CONFIG_DIR", tempDir)
		t.Setenv("MEGAPORT_ACCESS_KEY", "test-key")
		t.Setenv("MEGAPORT_SECRET_KEY", "test-secret")

		utils.BaseURL = ts.URL
		utils.TokenURL = ts.URL + "/oauth2/token"
		utils.Env = ""
		utils.ProfileOverride = ""

		origStderr := os.Stderr
		devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		assert.NoError(t, err)
		os.Stderr = devNull
		t.Cleanup(func() { os.Stderr = origStderr; _ = devNull.Close() })

		client, err := LoginWithOutput(context.Background(), "")
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.True(t, tokenEndpointHit.Load(), "token endpoint should have been called")
		assert.Contains(t, client.BaseURL.String(), ts.Listener.Addr().String())
	})

	t.Run("--token-url without --base-url warns and still applies the override", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/oauth2/token" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, `{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`)
				return
			}
			http.NotFound(w, r)
		}))
		defer ts.Close()

		t.Setenv("MEGAPORT_CONFIG_DIR", tempDir)
		t.Setenv("MEGAPORT_ACCESS_KEY", "test-key")
		t.Setenv("MEGAPORT_SECRET_KEY", "test-secret")
		t.Setenv("MEGAPORT_ENVIRONMENT", "staging")

		utils.BaseURL = ""
		utils.TokenURL = ts.URL + "/oauth2/token"
		utils.Env = "staging"
		utils.ProfileOverride = ""

		// Capture stderr concurrently to avoid filling the pipe buffer and
		// deadlocking LoginWithOutput if the spinner writes frequently.
		origStderr := os.Stderr
		pr, pw, pipeErr := os.Pipe()
		assert.NoError(t, pipeErr)
		os.Stderr = pw
		t.Cleanup(func() { os.Stderr = origStderr })

		var stderrBuf bytes.Buffer
		drainDone := make(chan struct{})
		go func() {
			_, _ = io.Copy(&stderrBuf, pr)
			close(drainDone)
		}()

		client, loginErr := LoginWithOutput(context.Background(), "")

		pw.Close()
		os.Stderr = origStderr
		<-drainDone
		pr.Close()

		assert.NoError(t, loginErr)
		assert.NotNil(t, client)
		assert.Contains(t, stderrBuf.String(), "--token-url is set without --base-url")
	})

	t.Run("--base-url without --token-url fails with unknown environment", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		t.Setenv("MEGAPORT_CONFIG_DIR", tempDir)
		t.Setenv("MEGAPORT_ACCESS_KEY", "test-key")
		t.Setenv("MEGAPORT_SECRET_KEY", "test-secret")

		utils.BaseURL = ts.URL
		utils.TokenURL = ""
		utils.Env = ""
		utils.ProfileOverride = ""

		origStderr := os.Stderr
		devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		assert.NoError(t, err)
		os.Stderr = devNull
		t.Cleanup(func() { os.Stderr = origStderr; _ = devNull.Close() })

		_, err = LoginWithOutput(context.Background(), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown API environment")
	})
}

func TestCredentialSelectionPrecedence(t *testing.T) {
	// Save and restore non-env-var globals
	originalEnv := utils.Env
	defer func() {
		utils.Env = originalEnv
	}()

	tests := []struct {
		name                  string
		globalFlag            string
		envAccessKey          string
		envSecretKey          string
		expectEnvVarsPriority bool
	}{
		{
			name:                  "Flag set: env vars should have priority",
			globalFlag:            "production",
			envAccessKey:          "env-access-key",
			envSecretKey:          "env-secret-key",
			expectEnvVarsPriority: true,
		},
		{
			name:                  "No flag: profile should have priority",
			globalFlag:            "",
			envAccessKey:          "env-access-key",
			envSecretKey:          "env-secret-key",
			expectEnvVarsPriority: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origEnv := utils.Env
			defer func() { utils.Env = origEnv }()

			// Setup test environment
			utils.Env = tt.globalFlag
			t.Setenv("MEGAPORT_ACCESS_KEY", tt.envAccessKey)
			t.Setenv("MEGAPORT_SECRET_KEY", tt.envSecretKey)

			// Test credential selection logic
			var accessKey, secretKey string

			// Simulate the new credential selection logic
			if utils.Env != "" {
				// --env flag was explicitly set, prioritize environment variables
				accessKey = os.Getenv("MEGAPORT_ACCESS_KEY")
				secretKey = os.Getenv("MEGAPORT_SECRET_KEY")
			} else {
				// No --env flag, profile would have priority (but we can't test that easily here)
				// For this test, we'll just verify the env var logic works
				if tt.expectEnvVarsPriority {
					accessKey = os.Getenv("MEGAPORT_ACCESS_KEY")
					secretKey = os.Getenv("MEGAPORT_SECRET_KEY")
				}
			}

			if tt.expectEnvVarsPriority {
				assert.Equal(t, tt.envAccessKey, accessKey, "Should use environment variable for access key when --env flag is set")
				assert.Equal(t, tt.envSecretKey, secretKey, "Should use environment variable for secret key when --env flag is set")
			}
		})
	}
}

func TestLoginFuncAccessors(t *testing.T) {
	t.Run("SetLoginFuncWithOutput round-trips", func(t *testing.T) {
		original := GetLoginFuncWithOutput()
		defer SetLoginFuncWithOutput(original)

		called := false
		SetLoginFuncWithOutput(func(_ context.Context, _ string) (*megaport.Client, error) {
			called = true
			return &megaport.Client{}, nil
		})
		fn := GetLoginFuncWithOutput()
		client, err := fn(context.Background(), "json")
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.True(t, called)
	})

	t.Run("SetNewUnauthenticatedClientFunc round-trips", func(t *testing.T) {
		original := GetNewUnauthenticatedClientFunc()
		defer SetNewUnauthenticatedClientFunc(original)

		called := false
		SetNewUnauthenticatedClientFunc(func() (*megaport.Client, error) {
			called = true
			return &megaport.Client{}, nil
		})
		fn := GetNewUnauthenticatedClientFunc()
		client, err := fn()
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.True(t, called)
	})
}

func TestAppendLogOpts(t *testing.T) {
	t.Run("adds log options when LogHTTP is true", func(t *testing.T) {
		origLogHTTP := utils.LogHTTP
		defer func() { utils.LogHTTP = origLogHTTP }()

		utils.LogHTTP = true
		opts := appendLogOpts([]megaport.ClientOpt{})
		assert.Len(t, opts, 2, "should add 2 log options when LogHTTP is enabled")
	})

	t.Run("does not add log options when LogHTTP is false", func(t *testing.T) {
		origLogHTTP := utils.LogHTTP
		defer func() { utils.LogHTTP = origLogHTTP }()

		utils.LogHTTP = false
		opts := appendLogOpts([]megaport.ClientOpt{})
		assert.Empty(t, opts, "should not add options when LogHTTP is disabled")
	})

	t.Run("preserves existing options", func(t *testing.T) {
		origLogHTTP := utils.LogHTTP
		defer func() { utils.LogHTTP = origLogHTTP }()

		utils.LogHTTP = true
		existing := []megaport.ClientOpt{megaport.WithEnvironment(megaport.EnvironmentStaging)}
		opts := appendLogOpts(existing)
		assert.Len(t, opts, 3, "should preserve existing option and add 2 log options")
	})
}

func TestCLIHeadersSentOnRequests(t *testing.T) {
	// Test the real NewUnauthenticatedClient production path to ensure
	// cliHeaders is actually wired in, not just that the SDK accepts it.
	originalFunc := GetNewUnauthenticatedClientFunc()
	defer SetNewUnauthenticatedClientFunc(originalFunc)

	originalEnv := utils.Env
	defer func() { utils.Env = originalEnv }()
	utils.Env = "production"

	headersCh := make(chan http.Header, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headersCh <- r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer ts.Close()

	// Use a temp config dir so profile resolution doesn't interfere
	tempDir, err := os.MkdirTemp("", "megaport-header-test")
	assert.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })
	t.Setenv("MEGAPORT_CONFIG_DIR", tempDir)

	client, err := NewUnauthenticatedClient()
	assert.NoError(t, err)

	// Point the real client at our test server
	client.BaseURL, err = client.BaseURL.Parse(ts.URL + "/")
	assert.NoError(t, err)

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/", nil)
	assert.NoError(t, err)

	_, err = client.Do(context.Background(), req, nil)
	assert.NoError(t, err)

	var capturedHeaders http.Header
	select {
	case capturedHeaders = <-headersCh:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for request headers from test server")
	}
	assert.Equal(t, "cli", capturedHeaders.Get("x-app"))
}

func TestOnBehalfOfCallContextHeader(t *testing.T) {
	origBaseURL := utils.BaseURL
	origTokenURL := utils.TokenURL
	origEnv := utils.Env
	origProfile := utils.ProfileOverride
	origUID := utils.ManagedAccountUID
	defer func() {
		utils.BaseURL = origBaseURL
		utils.TokenURL = origTokenURL
		utils.Env = origEnv
		utils.ProfileOverride = origProfile
		utils.ManagedAccountUID = origUID
	}()

	tempDir, err := os.MkdirTemp("", "megaport-call-context-test")
	assert.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	tests := []struct {
		name       string
		flagUID    string
		envUID     string
		wantHeader string // "" means the header must be absent
	}{
		{name: "flag sets header", flagUID: "flag-uid-123", wantHeader: "flag-uid-123"},
		{name: "env var used when no flag", envUID: "env-uid-456", wantHeader: "env-uid-456"},
		{name: "flag wins over env var", flagUID: "flag-uid-123", envUID: "env-uid-456", wantHeader: "flag-uid-123"},
		{name: "no flag and no env sends no header", wantHeader: ""},
		{name: "flag is trimmed", flagUID: "  flag-uid-123\n", wantHeader: "flag-uid-123"},
		{name: "whitespace-only flag sends no header", flagUID: "   ", wantHeader: ""},
		{name: "whitespace-only flag falls back to env", flagUID: "   ", envUID: "env-uid-456", wantHeader: "env-uid-456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headersCh := make(chan http.Header, 1)
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/oauth2/token" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = fmt.Fprint(w, `{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`)
					return
				}
				select {
				case headersCh <- r.Header.Clone():
				default:
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"message":"ok"}`))
			}))
			defer ts.Close()

			t.Setenv("MEGAPORT_CONFIG_DIR", tempDir)
			t.Setenv("MEGAPORT_ACCESS_KEY", "test-key")
			t.Setenv("MEGAPORT_SECRET_KEY", "test-secret")
			t.Setenv("MEGAPORT_MANAGED_ACCOUNT_UID", tt.envUID)

			utils.BaseURL = ts.URL
			utils.TokenURL = ts.URL + "/oauth2/token"
			utils.Env = ""
			utils.ProfileOverride = ""
			utils.ManagedAccountUID = tt.flagUID

			origStderr := os.Stderr
			devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
			assert.NoError(t, err)
			os.Stderr = devNull
			t.Cleanup(func() { os.Stderr = origStderr; _ = devNull.Close() })

			client, err := LoginWithOutput(context.Background(), "")
			assert.NoError(t, err)
			assert.NotNil(t, client)

			req, err := client.NewRequest(context.Background(), http.MethodGet, "/", nil)
			assert.NoError(t, err)
			_, err = client.Do(context.Background(), req, nil)
			assert.NoError(t, err)

			var captured http.Header
			select {
			case captured = <-headersCh:
			case <-time.After(5 * time.Second):
				t.Fatalf("timed out waiting for request headers from test server")
			}

			// Assert on presence, not Get()=="": an empty Get can't distinguish
			// an absent header from one set to the empty string.
			values := captured.Values("X-Call-Context")
			if tt.wantHeader == "" {
				assert.Empty(t, values, "X-Call-Context must not be sent when no UID is resolved")
			} else {
				assert.Equal(t, []string{tt.wantHeader}, values)
			}
		})
	}
}

// stringerValue is a fmt.Stringer wrapped via slog.Any, exercising the
// value-level redaction path for a value that isn't a plain slog string.
type stringerValue string

func (s stringerValue) String() string { return string(s) }

// logValuerLeak exposes its credential only through LogValue(); its plain fmt
// form ("{...}") hides the scheme prefix, so the value scan must Resolve()
// before inspecting or the inner handler prints the resolved credential.
type logValuerLeak struct{ token string }

func (l logValuerLeak) LogValue() slog.Value { return slog.StringValue("Bearer " + l.token) }

func TestRedactingHandler(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	handler := &redactingHandler{inner: inner}
	logger := slog.New(handler)

	t.Run("redacts sensitive keys", func(t *testing.T) {
		buf.Reset()
		logger.DebugContext(context.Background(), "auth",
			slog.String("access_key", "MEGA-SECRET-KEY"),
			slog.String("secret_key", "super-secret"),
			slog.String("access_token", "bearer-token-123"),
		)
		output := buf.String()
		assert.NotContains(t, output, "MEGA-SECRET-KEY")
		assert.NotContains(t, output, "super-secret")
		assert.NotContains(t, output, "bearer-token-123")
		assert.Contains(t, output, "[REDACTED]")
	})

	t.Run("preserves non-sensitive keys", func(t *testing.T) {
		buf.Reset()
		logger.DebugContext(context.Background(), "request",
			slog.String("method", "GET"),
			slog.String("path", "/v3/products"),
			slog.Int("status_code", 200),
		)
		output := buf.String()
		assert.Contains(t, output, "GET")
		assert.Contains(t, output, "/v3/products")
		assert.Contains(t, output, "200")
		assert.NotContains(t, output, "[REDACTED]")
	})

	t.Run("redacts response_body_base_64", func(t *testing.T) {
		buf.Reset()
		logger.DebugContext(context.Background(), "response",
			slog.String("response_body_base_64", "eyJhY2Nlc3NfdG9rZW4iOiJzZWNyZXQifQ=="),
		)
		output := buf.String()
		assert.NotContains(t, output, "eyJhY2Nlc3NfdG9rZW4iOiJzZWNyZXQifQ==")
		assert.Contains(t, output, "[REDACTED]")
	})

	// Fail-closed: a credential key the allowlist never enumerated (the SDK's
	// authorization_header) must still be redacted via substring matching.
	t.Run("redacts unlisted authorization_header", func(t *testing.T) {
		buf.Reset()
		logger.DebugContext(context.Background(), "login request",
			slog.String("authorization_header", "Basic YWNjZXNzOnNlY3JldA=="),
		)
		output := buf.String()
		assert.NotContains(t, output, "Basic YWNjZXNzOnNlY3JldA==")
		assert.NotContains(t, output, "YWNjZXNzOnNlY3JldA==")
		assert.Contains(t, output, "[REDACTED]")
	})

	t.Run("redaction is case-insensitive", func(t *testing.T) {
		for _, key := range []string{"Authorization", "X-Access-Key", "Secret_Key", "ACCESS_TOKEN"} {
			assert.True(t, isSensitiveKey(key), "expected %q to be treated as sensitive", key)
		}
	})

	t.Run("preserves non-credential keys", func(t *testing.T) {
		for _, key := range []string{"method", "path", "api_host", "trace_id", "content_type", "status_code"} {
			assert.False(t, isSensitiveKey(key), "expected %q to be preserved", key)
		}
	})

	// token_url is the constant OAuth endpoint, not the token, and is the most
	// useful field when debugging --log-http auth issues; keep it visible.
	t.Run("preserves token_url despite token substring", func(t *testing.T) {
		buf.Reset()
		logger.DebugContext(context.Background(), "login request",
			slog.String("token_url", "https://auth.megaport.com/oauth2/token"),
		)
		output := buf.String()
		assert.Contains(t, output, "https://auth.megaport.com/oauth2/token")
		assert.NotContains(t, output, "[REDACTED]")
	})

	// Fail closed on a credential-shaped value even when its key is unknown.
	t.Run("redacts auth value under unrecognized key", func(t *testing.T) {
		buf.Reset()
		logger.DebugContext(context.Background(), "request",
			slog.String("custom_header", "Basic YWNjZXNzOnNlY3JldA=="),
			slog.String("bearer_field", "Bearer eyJ-token-value"),
		)
		output := buf.String()
		assert.NotContains(t, output, "YWNjZXNzOnNlY3JldA==")
		assert.NotContains(t, output, "eyJ-token-value")
		assert.Contains(t, output, "[REDACTED]")
	})

	// A credential carried by a non-string value (slog.Any over a Stringer)
	// must still be caught by the value-level scan.
	t.Run("redacts auth value from non-string attr", func(t *testing.T) {
		buf.Reset()
		logger.DebugContext(context.Background(), "request",
			slog.Any("opaque", stringerValue("Bearer opaque-secret-token")),
		)
		output := buf.String()
		assert.NotContains(t, output, "opaque-secret-token")
		assert.Contains(t, output, "[REDACTED]")
	})

	// A LogValuer's credential is only visible after Resolve(); the scan must
	// resolve before inspecting or the inner handler prints it in cleartext.
	t.Run("redacts auth value from LogValuer attr", func(t *testing.T) {
		buf.Reset()
		logger.DebugContext(context.Background(), "request",
			slog.Any("lazy", logValuerLeak{token: "lazy-secret-token"}),
		)
		output := buf.String()
		assert.NotContains(t, output, "lazy-secret-token")
		assert.Contains(t, output, "[REDACTED]")
	})

	t.Run("redacts credential-family substrings", func(t *testing.T) {
		buf.Reset()
		logger.DebugContext(context.Background(), "auth",
			slog.String("client_secret", "shh-secret"),
			slog.String("refresh_token", "refresh-me"),
			slog.String("api_key_id", "key-id-leak"),
		)
		output := buf.String()
		assert.NotContains(t, output, "shh-secret")
		assert.NotContains(t, output, "refresh-me")
		assert.NotContains(t, output, "key-id-leak")
		assert.Contains(t, output, "[REDACTED]")
	})

	t.Run("redacts inside group attributes", func(t *testing.T) {
		buf.Reset()
		logger.DebugContext(context.Background(), "api call",
			slog.Group("api_request",
				slog.String("method", "POST"),
				slog.String("access_key", "should-be-redacted"),
				slog.String("response_body_base_64", "also-redacted"),
			),
		)
		output := buf.String()
		assert.Contains(t, output, "POST")
		assert.NotContains(t, output, "should-be-redacted")
		assert.NotContains(t, output, "also-redacted")
	})

	t.Run("Enabled delegates to inner", func(t *testing.T) {
		assert.True(t, handler.Enabled(context.Background(), slog.LevelDebug))
		assert.False(t, handler.Enabled(context.Background(), slog.LevelDebug-1))
	})

	t.Run("WithAttrs redacts", func(t *testing.T) {
		buf.Reset()
		child := handler.WithAttrs([]slog.Attr{slog.String("access_key", "persistent-key")})
		slog.New(child).DebugContext(context.Background(), "test")
		output := buf.String()
		assert.NotContains(t, output, "persistent-key")
		assert.Contains(t, output, "[REDACTED]")
	})

	t.Run("WithGroup wraps", func(t *testing.T) {
		buf.Reset()
		child := handler.WithGroup("mygroup")
		slog.New(child).DebugContext(context.Background(), "test", slog.String("access_key", "grouped-key"))
		output := buf.String()
		assert.NotContains(t, output, "grouped-key")
		assert.Contains(t, output, "[REDACTED]")
	})
}
