package managed_account

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newManagedAccountCmd(flags map[string]string) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("account-name", "", "")
	cmd.Flags().String("account-ref", "", "")
	for k, v := range flags {
		_ = cmd.Flags().Set(k, v)
	}
	return cmd
}

func TestParseManagedAccountRequestJSON(t *testing.T) {
	tests := []struct {
		name          string
		jsonStr       string
		setupFile     func(t *testing.T) string
		expectedError string
		validate      func(t *testing.T, req *megaport.ManagedAccountRequest)
	}{
		{
			name:    "valid JSON string",
			jsonStr: `{"accountName":"JSON Account","accountRef":"JSON-REF"}`,
			validate: func(t *testing.T, req *megaport.ManagedAccountRequest) {
				assert.Equal(t, "JSON Account", req.AccountName)
				assert.Equal(t, "JSON-REF", req.AccountRef)
			},
		},
		{
			name:          "invalid JSON syntax",
			jsonStr:       `{invalid json}`,
			expectedError: "failed to parse JSON",
		},
		{
			name:          "no input provided",
			expectedError: "failed to parse JSON",
		},
		{
			name: "valid JSON file",
			setupFile: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "account.json")
				assert.NoError(t, os.WriteFile(path, []byte(`{"accountName":"File Account","accountRef":"FILE-REF"}`), 0o644))
				return path
			},
			validate: func(t *testing.T, req *megaport.ManagedAccountRequest) {
				assert.Equal(t, "File Account", req.AccountName)
				assert.Equal(t, "FILE-REF", req.AccountRef)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var jsonFile string
			if tt.setupFile != nil {
				jsonFile = tt.setupFile(t)
			}

			req, err := parseManagedAccountRequestJSON(tt.jsonStr, jsonFile)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)

				var cliErr *exitcodes.CLIError
				require.True(t, errors.As(err, &cliErr))
				assert.Equal(t, exitcodes.Usage, cliErr.Code)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, req)
				tt.validate(t, req)
			}
		})
	}
}

func TestBuildManagedAccountRequestFromFlags(t *testing.T) {
	tests := []struct {
		name          string
		flags         map[string]string
		expectedError string
		validate      func(t *testing.T, req *megaport.ManagedAccountRequest)
	}{
		{
			name:  "both flags provided",
			flags: map[string]string{"account-name": "Test Account", "account-ref": "REF-001"},
			validate: func(t *testing.T, req *megaport.ManagedAccountRequest) {
				assert.Equal(t, "Test Account", req.AccountName)
				assert.Equal(t, "REF-001", req.AccountRef)
			},
		},
		{
			name:          "name only",
			flags:         map[string]string{"account-name": "Test Account"},
			expectedError: "accountName and accountRef are required",
		},
		{
			name:          "no flags (defaults)",
			flags:         map[string]string{},
			expectedError: "accountName and accountRef are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := buildManagedAccountRequestFromFlags(newManagedAccountCmd(tt.flags))

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)

				var cliErr *exitcodes.CLIError
				require.True(t, errors.As(err, &cliErr))
				assert.Equal(t, exitcodes.Usage, cliErr.Code)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, req)
				tt.validate(t, req)
			}
		})
	}
}

func TestBuildManagedAccountRequestFromJSON(t *testing.T) {
	tests := []struct {
		name          string
		jsonStr       string
		jsonFile      string
		setupFile     func(t *testing.T) string
		expectedError string
		validate      func(t *testing.T, req *megaport.ManagedAccountRequest)
	}{
		{
			name:    "valid JSON string",
			jsonStr: `{"accountName":"JSON Account","accountRef":"JSON-REF"}`,
			validate: func(t *testing.T, req *megaport.ManagedAccountRequest) {
				assert.Equal(t, "JSON Account", req.AccountName)
				assert.Equal(t, "JSON-REF", req.AccountRef)
			},
		},
		{
			name:          "JSON string missing account-ref",
			jsonStr:       `{"accountName":"Partial Account"}`,
			expectedError: "accountName and accountRef are required",
		},
		{
			name:          "invalid JSON syntax",
			jsonStr:       `{invalid json}`,
			expectedError: "failed to parse JSON",
		},
		{
			name:          "empty JSON object",
			jsonStr:       `{}`,
			expectedError: "accountName and accountRef are required",
		},
		{
			name: "valid JSON file",
			setupFile: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "account.json")
				assert.NoError(t, os.WriteFile(path, []byte(`{"accountName":"File Account","accountRef":"FILE-REF"}`), 0o644))
				return path
			},
			validate: func(t *testing.T, req *megaport.ManagedAccountRequest) {
				assert.Equal(t, "File Account", req.AccountName)
				assert.Equal(t, "FILE-REF", req.AccountRef)
			},
		},
		{
			name:          "JSON file not found",
			jsonFile:      "/nonexistent/path/account.json",
			expectedError: "failed to read JSON file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonFile := tt.jsonFile
			if tt.setupFile != nil {
				jsonFile = tt.setupFile(t)
			}

			req, err := buildManagedAccountRequestFromJSON(tt.jsonStr, jsonFile)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)

				var cliErr *exitcodes.CLIError
				require.True(t, errors.As(err, &cliErr))
				assert.Equal(t, exitcodes.Usage, cliErr.Code)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, req)
				tt.validate(t, req)
			}
		})
	}
}

func TestBuildUpdateManagedAccountRequestFromFlags(t *testing.T) {
	// A field whose flag isn't passed must keep the current account's value.
	current := &megaport.ManagedAccount{AccountName: "Current Name", AccountRef: "CURRENT-REF"}

	tests := []struct {
		name     string
		flags    map[string]string
		validate func(t *testing.T, req *megaport.ManagedAccountRequest)
	}{
		{
			name:  "account-name only preserves current ref",
			flags: map[string]string{"account-name": "Updated Account"},
			validate: func(t *testing.T, req *megaport.ManagedAccountRequest) {
				assert.Equal(t, "Updated Account", req.AccountName)
				assert.Equal(t, "CURRENT-REF", req.AccountRef)
			},
		},
		{
			name:  "account-ref only preserves current name",
			flags: map[string]string{"account-ref": "NEW-REF"},
			validate: func(t *testing.T, req *megaport.ManagedAccountRequest) {
				assert.Equal(t, "Current Name", req.AccountName)
				assert.Equal(t, "NEW-REF", req.AccountRef)
			},
		},
		{
			name:  "both flags override current",
			flags: map[string]string{"account-name": "Updated Account", "account-ref": "NEW-REF"},
			validate: func(t *testing.T, req *megaport.ManagedAccountRequest) {
				assert.Equal(t, "Updated Account", req.AccountName)
				assert.Equal(t, "NEW-REF", req.AccountRef)
			},
		},
		{
			name:  "no flags keeps both current values",
			flags: map[string]string{},
			validate: func(t *testing.T, req *megaport.ManagedAccountRequest) {
				assert.Equal(t, "Current Name", req.AccountName)
				assert.Equal(t, "CURRENT-REF", req.AccountRef)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := buildUpdateManagedAccountRequestFromFlags(newManagedAccountCmd(tt.flags), current)
			assert.NoError(t, err)
			assert.NotNil(t, req)
			tt.validate(t, req)
		})
	}
}

// Only fields whose flag was explicitly changed override the current value;
// an untouched flag leaves the current account's value in place.
func TestBuildUpdateManagedAccountRequestFromFlags_ExplicitFieldTracking(t *testing.T) {
	current := &megaport.ManagedAccount{AccountName: "Current Name", AccountRef: "CURRENT-REF"}
	cmd := newManagedAccountCmd(nil)
	// account-ref is registered but never set, so it keeps the current value.
	_ = cmd.Flags().Set("account-name", "Only Name")

	req, err := buildUpdateManagedAccountRequestFromFlags(cmd, current)
	assert.NoError(t, err)
	assert.Equal(t, "Only Name", req.AccountName)
	assert.Equal(t, "CURRENT-REF", req.AccountRef)
	assert.False(t, cmd.Flags().Changed("account-ref"))
}

func TestBuildUpdateManagedAccountRequestFromJSON(t *testing.T) {
	// A field omitted from the JSON body must keep the current account's value.
	current := &megaport.ManagedAccount{AccountName: "Current Name", AccountRef: "CURRENT-REF"}

	tests := []struct {
		name          string
		jsonStr       string
		jsonFile      string
		setupFile     func(t *testing.T) string
		expectedError string
		validate      func(t *testing.T, req *megaport.ManagedAccountRequest)
	}{
		{
			name:    "valid JSON string overrides both",
			jsonStr: `{"accountName":"Updated Account","accountRef":"NEW-REF"}`,
			validate: func(t *testing.T, req *megaport.ManagedAccountRequest) {
				assert.Equal(t, "Updated Account", req.AccountName)
				assert.Equal(t, "NEW-REF", req.AccountRef)
			},
		},
		{
			name:    "partial JSON preserves omitted ref",
			jsonStr: `{"accountName":"Updated Account"}`,
			validate: func(t *testing.T, req *megaport.ManagedAccountRequest) {
				assert.Equal(t, "Updated Account", req.AccountName)
				assert.Equal(t, "CURRENT-REF", req.AccountRef)
			},
		},
		{
			name:    "empty JSON object keeps both current values",
			jsonStr: `{}`,
			validate: func(t *testing.T, req *megaport.ManagedAccountRequest) {
				assert.Equal(t, "Current Name", req.AccountName)
				assert.Equal(t, "CURRENT-REF", req.AccountRef)
			},
		},
		{
			name:    "explicit empty ref clears it",
			jsonStr: `{"accountRef":""}`,
			validate: func(t *testing.T, req *megaport.ManagedAccountRequest) {
				assert.Equal(t, "Current Name", req.AccountName)
				assert.Equal(t, "", req.AccountRef)
			},
		},
		{
			name:          "invalid JSON syntax",
			jsonStr:       `{invalid}`,
			expectedError: "failed to parse JSON",
		},
		{
			name: "valid JSON file",
			setupFile: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "update.json")
				assert.NoError(t, os.WriteFile(path, []byte(`{"accountName":"File Updated","accountRef":"FILE-REF"}`), 0o644))
				return path
			},
			validate: func(t *testing.T, req *megaport.ManagedAccountRequest) {
				assert.Equal(t, "File Updated", req.AccountName)
				assert.Equal(t, "FILE-REF", req.AccountRef)
			},
		},
		{
			name:          "JSON file not found",
			jsonFile:      "/nonexistent/path/update.json",
			expectedError: "failed to read JSON file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonFile := tt.jsonFile
			if tt.setupFile != nil {
				jsonFile = tt.setupFile(t)
			}

			patch, err := parseManagedAccountUpdatePatchJSON(tt.jsonStr, jsonFile)
			var req *megaport.ManagedAccountRequest
			if err == nil {
				req = patch.applyTo(current)
			}

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, req)
				tt.validate(t, req)
			}
		})
	}
}
