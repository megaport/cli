package users

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/config"
	"github.com/megaport/megaport-cli/internal/testutil"
	"github.com/megaport/megaport-cli/internal/utils"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListUsers(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name          string
		setupMock     func(*MockUserManagementService)
		expectedError string
		expectedOut   string
	}{
		{
			name: "success",
			setupMock: func(m *MockUserManagementService) {
				m.ListUsersResult = []*megaport.User{
					{PartyId: 1, FirstName: "John", LastName: "Doe", Email: "john@example.com", Position: "Technical Admin", Active: true},
				}
			},
			expectedOut: "John",
		},
		{
			name: "API error",
			setupMock: func(m *MockUserManagementService) {
				m.ListUsersErr = fmt.Errorf("API failure")
			},
			expectedError: "API failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockUserManagementService{}
			tt.setupMock(mockService)

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.UserManagementService = mockService
				return client, nil
			})

			cmd := testutil.NewCommand("list", testutil.OutputAdapter(ListUsers))
			cmd.Flags().String("position", "", "")
			cmd.Flags().Bool("active-only", false, "")
			cmd.Flags().Bool("inactive-only", false, "")

			var err error
			capturedOutput := output.CaptureOutput(func() {
				err = testutil.OutputAdapter(ListUsers)(cmd, nil)
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.expectedOut != "" {
					assert.Contains(t, capturedOutput, tt.expectedOut)
				}
			}
		})
	}
}

func TestGetUser(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name          string
		args          []string
		setupMock     func(*MockUserManagementService)
		expectedError string
		expectedOut   string
	}{
		{
			name: "success",
			args: []string{"12345"},
			setupMock: func(m *MockUserManagementService) {
				m.GetUserResult = &megaport.User{
					PartyId: 12345, FirstName: "Jane", LastName: "Smith",
					Email: "jane@example.com", Position: "Company Admin", Active: true,
				}
			},
			expectedOut: "Jane",
		},
		{
			name: "API error",
			args: []string{"12345"},
			setupMock: func(m *MockUserManagementService) {
				m.GetUserErr = fmt.Errorf("user not found")
			},
			expectedError: "user not found",
		},
		{
			name:          "invalid employee ID",
			args:          []string{"abc"},
			setupMock:     func(m *MockUserManagementService) {},
			expectedError: "Invalid employee ID",
		},
		{
			name: "nil user",
			args: []string{"99999"},
			setupMock: func(m *MockUserManagementService) {
				m.ForceNilGetUser = true
			},
			expectedError: "no user found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockUserManagementService{}
			tt.setupMock(mockService)

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.UserManagementService = mockService
				return client, nil
			})

			cmd := testutil.NewCommand("get", testutil.OutputAdapter(GetUser))

			var err error
			capturedOutput := output.CaptureOutput(func() {
				err = testutil.OutputAdapter(GetUser)(cmd, tt.args)
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.expectedOut != "" {
					assert.Contains(t, capturedOutput, tt.expectedOut)
				}
			}
		})
	}
}

func TestCreateUser(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name             string
		flags            map[string]string
		jsonInput        string
		setupMock        func(*MockUserManagementService)
		loginError       error
		expectedError    string
		expectedContains string
	}{
		{
			name: "success with flags",
			flags: map[string]string{
				"first-name": "John",
				"last-name":  "Doe",
				"email":      "john@example.com",
				"position":   "Technical Admin",
			},
			setupMock: func(m *MockUserManagementService) {
				m.CreateUserResult = &megaport.CreateUserResponse{EmployeeID: 12345}
			},
			expectedContains: "12345",
		},
		{
			name:      "success with JSON",
			jsonInput: `{"firstName":"Jane","lastName":"Doe","email":"jane@example.com","position":"Finance","active":true}`,
			setupMock: func(m *MockUserManagementService) {
				m.CreateUserResult = &megaport.CreateUserResponse{EmployeeID: 67890}
			},
			expectedContains: "67890",
		},
		{
			name: "API error",
			flags: map[string]string{
				"first-name": "John",
				"last-name":  "Doe",
				"email":      "john@example.com",
				"position":   "Technical Admin",
			},
			setupMock: func(m *MockUserManagementService) {
				m.CreateUserErr = fmt.Errorf("creation failed")
			},
			expectedError: "creation failed",
		},
		{
			name:          "no input provided",
			flags:         map[string]string{},
			setupMock:     func(m *MockUserManagementService) {},
			expectedError: "no input provided",
		},
		{
			name: "login error",
			flags: map[string]string{
				"first-name": "John",
				"last-name":  "Doe",
				"email":      "john@example.com",
				"position":   "Technical Admin",
			},
			setupMock:     func(m *MockUserManagementService) {},
			loginError:    fmt.Errorf("auth failed"),
			expectedError: "auth failed",
		},
		{
			name:          "invalid JSON",
			jsonInput:     `{invalid}`,
			setupMock:     func(m *MockUserManagementService) {},
			expectedError: "failed to parse JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockUserManagementService{}
			tt.setupMock(mockService)

			if tt.loginError != nil {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					return nil, tt.loginError
				})
			} else {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.UserManagementService = mockService
					return client, nil
				})
			}

			cmd := &cobra.Command{Use: "create"}
			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("first-name", "", "")
			cmd.Flags().String("last-name", "", "")
			cmd.Flags().String("email", "", "")
			cmd.Flags().String("position", "", "")
			cmd.Flags().String("phone", "", "")

			if tt.jsonInput != "" {
				require.NoError(t, cmd.Flags().Set("json", tt.jsonInput))
			}
			for k, v := range tt.flags {
				require.NoError(t, cmd.Flags().Set(k, v))
			}

			var err error
			var capturedOutput string
			capturedErr := captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = CreateUser(cmd, nil, true)
				})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.NotContains(t, err.Error(), "strconv")
			} else {
				assert.NoError(t, err)
				if tt.expectedContains != "" {
					assert.Contains(t, capturedOutput+capturedErr, tt.expectedContains)
				}
			}
		})
	}
}

func TestCreateUser_NilResponse(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	mockService := &MockUserManagementService{ForceNilCreateUser: true}
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.UserManagementService = mockService
		return client, nil
	})

	cmd := &cobra.Command{Use: "create"}
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	cmd.Flags().String("first-name", "", "")
	cmd.Flags().String("last-name", "", "")
	cmd.Flags().String("email", "", "")
	cmd.Flags().String("position", "", "")
	cmd.Flags().String("phone", "", "")
	require.NoError(t, cmd.Flags().Set("first-name", "John"))
	require.NoError(t, cmd.Flags().Set("last-name", "Doe"))
	require.NoError(t, cmd.Flags().Set("email", "john@example.com"))
	require.NoError(t, cmd.Flags().Set("position", "Technical Admin"))

	var err error
	require.NotPanics(t, func() {
		output.CaptureOutput(func() {
			err = CreateUser(cmd, nil, true)
		})
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty response")
}

func TestUpdateUser(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name             string
		args             []string
		flags            map[string]string
		jsonInput        string
		setupMock        func(*MockUserManagementService)
		expectedError    string
		expectedContains string
	}{
		{
			name:  "success with flags",
			flags: map[string]string{"first-name": "Updated"},
			setupMock: func(m *MockUserManagementService) {
				m.GetUserResult = &megaport.User{
					PartyId: 12345, FirstName: "Original", LastName: "User",
					Email: "test@example.com", Active: true,
				}
			},
			expectedContains: "User updated",
		},
		{
			name:      "success with JSON",
			jsonInput: `{"firstName":"Updated"}`,
			setupMock: func(m *MockUserManagementService) {
				m.GetUserResult = &megaport.User{
					PartyId: 12345, FirstName: "Original", LastName: "User",
					Email: "test@example.com", Active: true,
				}
			},
			expectedContains: "User updated",
		},
		{
			name:  "API error",
			flags: map[string]string{"first-name": "Updated"},
			setupMock: func(m *MockUserManagementService) {
				m.UpdateUserErr = fmt.Errorf("update failed")
			},
			expectedError: "update failed",
		},
		{
			name:          "invalid employee ID",
			args:          []string{"abc"},
			setupMock:     func(m *MockUserManagementService) {},
			expectedError: "Invalid employee ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockUserManagementService{}
			tt.setupMock(mockService)

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.UserManagementService = mockService
				return client, nil
			})

			cmd := &cobra.Command{Use: "update"}
			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("first-name", "", "")
			cmd.Flags().String("last-name", "", "")
			cmd.Flags().String("email", "", "")
			cmd.Flags().String("position", "", "")
			cmd.Flags().String("phone", "", "")
			cmd.Flags().Bool("active", false, "")
			cmd.Flags().Bool("notification-enabled", false, "")

			if tt.jsonInput != "" {
				require.NoError(t, cmd.Flags().Set("json", tt.jsonInput))
			}
			for k, v := range tt.flags {
				require.NoError(t, cmd.Flags().Set(k, v))
			}

			args := tt.args
			if len(args) == 0 {
				args = []string{"12345"}
			}

			var err error
			var capturedOutput string
			capturedErr := captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = UpdateUser(cmd, args, true)
				})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.NotContains(t, err.Error(), "strconv")
			} else {
				assert.NoError(t, err)
				if tt.expectedContains != "" {
					assert.Contains(t, capturedOutput+capturedErr, tt.expectedContains)
				}
			}
		})
	}
}

func TestDeleteUser(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	originalConfirmPrompt := utils.GetConfirmPrompt()
	defer func() { utils.SetConfirmPrompt(originalConfirmPrompt) }()

	tests := []struct {
		name             string
		args             []string
		force            bool
		confirmResult    bool
		setupMock        func(*MockUserManagementService)
		expectedError    string
		expectedContains string
	}{
		{
			name:             "success with force",
			force:            true,
			setupMock:        func(m *MockUserManagementService) {},
			expectedContains: "User deleted",
		},
		{
			name:             "user confirms",
			force:            false,
			confirmResult:    true,
			setupMock:        func(m *MockUserManagementService) {},
			expectedContains: "User deleted",
		},
		{
			name:          "user cancels",
			force:         false,
			confirmResult: false,
			setupMock:     func(m *MockUserManagementService) {},
			expectedError: "cancelled by user",
		},
		{
			name:  "API error",
			force: true,
			setupMock: func(m *MockUserManagementService) {
				m.DeleteUserErr = fmt.Errorf("delete failed")
			},
			expectedError: "delete failed",
		},
		{
			name:          "invalid employee ID",
			args:          []string{"abc"},
			force:         true,
			setupMock:     func(m *MockUserManagementService) {},
			expectedError: "Invalid employee ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockUserManagementService{}
			tt.setupMock(mockService)

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.UserManagementService = mockService
				return client, nil
			})

			utils.SetConfirmPrompt(func(_ string, _ bool) bool {
				return tt.confirmResult
			})

			cmd := &cobra.Command{Use: "delete"}
			cmd.Flags().Bool("force", false, "")
			if tt.force {
				require.NoError(t, cmd.Flags().Set("force", "true"))
			}

			args := tt.args
			if len(args) == 0 {
				args = []string{"12345"}
			}

			var err error
			var capturedOutput string
			capturedErr := captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = DeleteUser(cmd, args, true)
				})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.NotContains(t, err.Error(), "strconv")
			} else {
				assert.NoError(t, err)
				if tt.expectedContains != "" {
					assert.Contains(t, capturedOutput+capturedErr, tt.expectedContains)
				}
			}
		})
	}
}

func TestDeactivateUser(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	originalConfirmPrompt := utils.GetConfirmPrompt()
	defer func() { utils.SetConfirmPrompt(originalConfirmPrompt) }()

	tests := []struct {
		name             string
		args             []string
		force            bool
		confirmResult    bool
		setupMock        func(*MockUserManagementService)
		expectedError    string
		expectedContains string
	}{
		{
			name:             "success with force",
			force:            true,
			setupMock:        func(m *MockUserManagementService) {},
			expectedContains: "deactivated successfully",
		},
		{
			name:          "user cancels",
			force:         false,
			confirmResult: false,
			setupMock:     func(m *MockUserManagementService) {},
			expectedError: "cancelled by user",
		},
		{
			name:  "API error",
			force: true,
			setupMock: func(m *MockUserManagementService) {
				m.DeactivateErr = fmt.Errorf("deactivation failed")
			},
			expectedError: "deactivation failed",
		},
		{
			name:          "invalid employee ID",
			args:          []string{"abc"},
			force:         true,
			setupMock:     func(m *MockUserManagementService) {},
			expectedError: "Invalid employee ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockUserManagementService{}
			tt.setupMock(mockService)

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.UserManagementService = mockService
				return client, nil
			})

			utils.SetConfirmPrompt(func(_ string, _ bool) bool {
				return tt.confirmResult
			})

			cmd := &cobra.Command{Use: "deactivate"}
			cmd.Flags().Bool("force", false, "")
			if tt.force {
				require.NoError(t, cmd.Flags().Set("force", "true"))
			}

			args := tt.args
			if len(args) == 0 {
				args = []string{"12345"}
			}

			var err error
			var capturedOutput string
			capturedErr := captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = DeactivateUser(cmd, args, true)
				})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.NotContains(t, err.Error(), "strconv")
			} else {
				assert.NoError(t, err)
				if tt.expectedContains != "" {
					assert.Contains(t, capturedOutput+capturedErr, tt.expectedContains)
				}
			}
		})
	}
}

func TestGetUserActivity(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name          string
		setupMock     func(*MockUserManagementService)
		expectedError string
		expectedOut   string
	}{
		{
			name: "success",
			setupMock: func(m *MockUserManagementService) {
				m.ActivityResult = []*megaport.UserActivity{
					{LoginName: "john@example.com", Name: "Login", Description: "User logged in"},
				}
			},
			expectedOut: "Login",
		},
		{
			name: "API error",
			setupMock: func(m *MockUserManagementService) {
				m.ActivityErr = fmt.Errorf("activity fetch failed")
			},
			expectedError: "activity fetch failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockUserManagementService{}
			tt.setupMock(mockService)

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.UserManagementService = mockService
				return client, nil
			})

			cmd := testutil.NewCommand("activity", testutil.OutputAdapter(GetUserActivity))
			cmd.Flags().String("employee-id", "", "")

			var err error
			capturedOutput := output.CaptureOutput(func() {
				err = testutil.OutputAdapter(GetUserActivity)(cmd, nil)
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.expectedOut != "" {
					assert.Contains(t, capturedOutput, tt.expectedOut)
				}
			}
		})
	}
}

func TestFilterUsers(t *testing.T) {
	users := []*megaport.User{
		{FirstName: "Active Admin", Position: "Company Admin", Active: true},
		{FirstName: "Active Tech", Position: "Technical Admin", Active: true},
		{FirstName: "Inactive", Position: "Company Admin", Active: false},
		nil,
	}

	assert.Len(t, filterUsers(users, "", false, false), 3)
	assert.Len(t, filterUsers(users, "", true, false), 2)
	assert.Len(t, filterUsers(users, "", false, true), 1)
	assert.Len(t, filterUsers(users, "Company Admin", false, false), 2)
	assert.Len(t, filterUsers(users, "Company Admin", true, false), 1)
	assert.Len(t, filterUsers(nil, "", false, false), 0)
}

func captureStderr(t *testing.T, fn func()) (result string) {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() { os.Stderr = old }()
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() { defer close(done); _, _ = io.Copy(&buf, r) }()
	defer func() { _ = w.Close(); <-done; _ = r.Close(); result = buf.String() }()
	fn()
	return
}
