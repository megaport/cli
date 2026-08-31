package ports

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/config"
	"github.com/megaport/megaport-cli/internal/testutil"
	"github.com/megaport/megaport-cli/internal/utils"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPortStatus(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name           string
		portUID        string
		setupMock      func(*MockPortService)
		expectedError  string
		expectedOutput string
		outputFormat   string
	}{
		{
			name:    "successful status retrieval - table format",
			portUID: "port-123abc",
			setupMock: func(m *MockPortService) {
				m.GetPortResult = &megaport.Port{
					UID:                "port-123abc",
					Name:               "Test Port",
					ProvisioningStatus: "CONFIGURED",
					PortSpeed:          10000,
					Type:               "MEGAPORT",
				}
			},
			expectedOutput: "port-123abc",
			outputFormat:   "table",
		},
		{
			name:    "successful status retrieval - json format",
			portUID: "port-123abc",
			setupMock: func(m *MockPortService) {
				m.GetPortResult = &megaport.Port{
					UID:                "port-123abc",
					Name:               "Test Port",
					ProvisioningStatus: "LIVE",
					PortSpeed:          1000,
					Type:               "MEGAPORT",
				}
			},
			expectedOutput: "port-123abc",
			outputFormat:   "json",
		},
		{
			name:    "Port not found",
			portUID: "port-notfound",
			setupMock: func(m *MockPortService) {
				m.GetPortErr = fmt.Errorf("Port not found")
			},
			expectedError: "failed to get Port status",
			outputFormat:  "table",
		},
		{
			name:    "API error",
			portUID: "port-error",
			setupMock: func(m *MockPortService) {
				m.GetPortErr = fmt.Errorf("API error")
			},
			expectedError: "API error",
			outputFormat:  "table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockPortService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.PortService = mockService
				return client, nil
			})

			cmd := &cobra.Command{
				Use: "status [portUID]",
			}

			var err error
			capturedOutput := output.CaptureOutput(func() {
				err = GetPortStatus(cmd, []string{tt.portUID}, true, tt.outputFormat)
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, capturedOutput, tt.expectedOutput)

				switch tt.outputFormat {
				case "json":
					assert.Contains(t, capturedOutput, "\"uid\":")
					assert.Contains(t, capturedOutput, "\"name\":")
					assert.Contains(t, capturedOutput, "\"status\":")
					assert.Contains(t, capturedOutput, "\"type\":")
				case "table":
					assert.Contains(t, capturedOutput, "UID")
					assert.Contains(t, capturedOutput, "NAME")
					assert.Contains(t, capturedOutput, "STATUS")
					assert.Contains(t, capturedOutput, "TYPE")
				}
			}
		})
	}
}

func TestGetPortCmd_WithMockClient(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name          string
		portID        string
		format        string
		setupMock     func(*MockPortService)
		expectedError string
		expectedOut   []string
	}{
		{
			name:   "get Port success table format",
			portID: "port-123",
			format: "table",
			setupMock: func(m *MockPortService) {
				m.GetPortResult = &megaport.Port{
					UID:                "port-123",
					Name:               "Test Port",
					LocationID:         123,
					PortSpeed:          10000,
					ProvisioningStatus: "LIVE",
				}
			},
			expectedOut: []string{"port-123", "Test Port", "10000", "LIVE"},
		},
		{
			name:   "get Port success json format",
			portID: "port-123",
			format: "json",
			setupMock: func(m *MockPortService) {
				m.GetPortResult = &megaport.Port{
					UID:                "port-123",
					Name:               "Test Port",
					LocationID:         123,
					PortSpeed:          10000,
					ProvisioningStatus: "LIVE",
				}
			},
			expectedOut: []string{`"uid": "port-123"`, `"name": "Test Port"`, `"port_speed": 10000`},
		},
		{
			name:   "get Port error",
			portID: "port-invalid",
			format: "table",
			setupMock: func(m *MockPortService) {
				m.GetPortErr = fmt.Errorf("Port not found")
			},
			expectedError: "Port not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPortService := &MockPortService{}
			tt.setupMock(mockPortService)

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.PortService = mockPortService
				return client, nil
			})

			cmd := testutil.NewCommand("get", testutil.OutputAdapter(GetPort))
			testutil.SetFlags(t, cmd, map[string]string{"output": tt.format})

			var err error
			output := output.CaptureOutput(func() {
				err = testutil.OutputAdapter(GetPort)(cmd, []string{tt.portID})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				for _, expected := range tt.expectedOut {
					assert.Contains(t, output, expected)
				}
			}
		})
	}
}

func TestGetPortStatus_NilPort(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	mockService := &MockPortService{ForceNilGetPort: true}
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.PortService = mockService
		return client, nil
	})

	cmd := &cobra.Command{Use: "status"}
	var err error
	output.CaptureOutput(func() {
		err = GetPortStatus(cmd, []string{"port-nil"}, true, "table")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no port found")
}

func TestBuyPort_EmptyUIDs(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	originalBuyPortFunc := buyPortFunc
	defer func() {
		buyPortFunc = originalBuyPortFunc
	}()
	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	mockService := &MockPortService{}
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.PortService = mockService
		return client, nil
	})
	buyPortFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error) {
		return &megaport.BuyPortResponse{
			TechnicalServiceUIDs: []string{},
		}, nil
	}

	cmd := &cobra.Command{Use: "buy"}
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	cmd.Flags().String("name", "test-port", "")
	cmd.Flags().Int("term", 12, "")
	cmd.Flags().Int("port-speed", 1000, "")
	cmd.Flags().Int("location-id", 1, "")
	cmd.Flags().Bool("marketplace-visibility", false, "")
	cmd.Flags().String("diversity-zone", "", "")
	cmd.Flags().Bool("cost-confirm", true, "")
	require.NoError(t, cmd.Flags().Set("name", "test-port"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("port-speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))
	require.NoError(t, cmd.Flags().Set("marketplace-visibility", "true"))

	var err error
	output.CaptureOutput(func() {
		err = BuyPort(cmd, nil, true)
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no UID returned")
}

func TestBuyLAGPort_EmptyUIDs(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	originalBuyPortFunc := buyPortFunc
	defer func() {
		buyPortFunc = originalBuyPortFunc
	}()
	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	mockService := &MockPortService{}
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.PortService = mockService
		return client, nil
	})
	buyPortFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error) {
		return &megaport.BuyPortResponse{
			TechnicalServiceUIDs: []string{},
		}, nil
	}

	cmd := &cobra.Command{Use: "buy-lag"}
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	cmd.Flags().String("name", "test-lag", "")
	cmd.Flags().Int("term", 12, "")
	cmd.Flags().Int("port-speed", 10000, "")
	cmd.Flags().Int("location-id", 1, "")
	cmd.Flags().Int("lag-count", 2, "")
	cmd.Flags().Bool("marketplace-visibility", false, "")
	cmd.Flags().String("diversity-zone", "", "")
	cmd.Flags().Bool("cost-confirm", true, "")
	require.NoError(t, cmd.Flags().Set("name", "test-lag"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("port-speed", "10000"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))
	require.NoError(t, cmd.Flags().Set("lag-count", "2"))
	require.NoError(t, cmd.Flags().Set("marketplace-visibility", "true"))

	var err error
	output.CaptureOutput(func() {
		err = BuyLAGPort(cmd, nil, true)
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no UID returned")
}

func TestBuyLAGPort_NilResponse(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	originalBuyPortFunc := buyPortFunc
	defer func() {
		buyPortFunc = originalBuyPortFunc
	}()
	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	mockService := &MockPortService{}
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.PortService = mockService
		return client, nil
	})
	buyPortFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error) {
		return nil, nil
	}

	cmd := &cobra.Command{Use: "buy-lag"}
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	cmd.Flags().String("name", "test-lag", "")
	cmd.Flags().Int("term", 12, "")
	cmd.Flags().Int("port-speed", 10000, "")
	cmd.Flags().Int("location-id", 1, "")
	cmd.Flags().Int("lag-count", 2, "")
	cmd.Flags().Bool("marketplace-visibility", false, "")
	cmd.Flags().String("diversity-zone", "", "")
	cmd.Flags().Bool("cost-confirm", true, "")
	require.NoError(t, cmd.Flags().Set("name", "test-lag"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("port-speed", "10000"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))
	require.NoError(t, cmd.Flags().Set("lag-count", "2"))
	require.NoError(t, cmd.Flags().Set("marketplace-visibility", "true"))

	var err error
	output.CaptureOutput(func() {
		err = BuyLAGPort(cmd, nil, true)
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty response from API")
}

func TestDeletePort_SafeDeleteFlag(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	mockService := &MockPortService{}
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.PortService = mockService
		return client, nil
	})

	cmd := &cobra.Command{
		Use: "delete",
		RunE: func(cmd *cobra.Command, args []string) error {
			return DeletePort(cmd, args, true)
		},
	}
	cmd.Flags().BoolP("force", "f", true, "")
	cmd.Flags().Bool("now", false, "")
	cmd.Flags().Bool("safe-delete", false, "")
	require.NoError(t, cmd.Flags().Set("safe-delete", "true"))

	var err error
	output.CaptureOutput(func() {
		err = cmd.RunE(cmd, []string{"port-uid-123"})
	})

	assert.NoError(t, err)
	assert.NotNil(t, mockService.CapturedDeletePortRequest)
	assert.True(t, mockService.CapturedDeletePortRequest.SafeDelete)
	assert.Equal(t, "port-uid-123", mockService.CapturedDeletePortRequest.PortID)
}

func TestListPortResourceTagsCmd(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name           string
		portUID        string
		setupMock      func(*MockPortService)
		expectedError  string
		expectedOutput string
	}{
		{
			name:    "successful list",
			portUID: "port-123",
			setupMock: func(m *MockPortService) {
				m.ListPortResourceTagsResult = map[string]string{
					"environment": "production",
					"team":        "networking",
				}
			},
			expectedOutput: "environment",
		},
		{
			name:    "empty tags",
			portUID: "port-empty",
			setupMock: func(m *MockPortService) {
				m.ListPortResourceTagsResult = map[string]string{}
			},
		},
		{
			name:    "API error",
			portUID: "port-error",
			setupMock: func(m *MockPortService) {
				m.ListPortResourceTagsErr = fmt.Errorf("API error: not found")
			},
			expectedError: "failed to get resource tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockPortService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.PortService = mockService
				return client, nil
			})

			cmd := &cobra.Command{
				Use: "list-tags [portUID]",
				RunE: func(cmd *cobra.Command, args []string) error {
					return ListPortResourceTags(cmd, args, false, "table")
				},
			}

			var err error
			capturedOutput := output.CaptureOutput(func() {
				err = cmd.RunE(cmd, []string{tt.portUID})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.expectedOutput != "" {
					assert.Contains(t, capturedOutput, tt.expectedOutput)
				}
			}
		})
	}
}

func TestUpdatePortResourceTagsCmd(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name                 string
		portUID              string
		interactive          bool
		force                bool
		jsonInput            string
		tagsInput            string
		resourceTagsInput    string
		setupMock            func(*MockPortService)
		expectedError        string
		expectedOutput       string
		expectedCapturedTags map[string]string
	}{
		{
			name:      "successful update with json",
			portUID:   "port-456",
			jsonInput: `{"environment": "production", "team": "networking"}`,
			setupMock: func(m *MockPortService) {
				m.ListPortResourceTagsResult = map[string]string{}
			},
			expectedOutput: "Resource tags updated for Port port-456",
			expectedCapturedTags: map[string]string{
				"environment": "production",
				"team":        "networking",
			},
		},
		{
			name:      "successful update with tags flag",
			portUID:   "port-tags",
			tagsInput: `{"env": "dev"}`,
			setupMock: func(m *MockPortService) {
				m.ListPortResourceTagsResult = map[string]string{}
			},
			expectedOutput:       "Resource tags updated for Port port-tags",
			expectedCapturedTags: map[string]string{"env": "dev"},
		},
		{
			name:              "successful update with resource-tags flag",
			portUID:           "port-rt",
			resourceTagsInput: `{"env": "staging"}`,
			setupMock: func(m *MockPortService) {
				m.ListPortResourceTagsResult = map[string]string{}
			},
			expectedOutput:       "Resource tags updated for Port port-rt",
			expectedCapturedTags: map[string]string{"env": "staging"},
		},
		{
			name:      "error with invalid json",
			portUID:   "port-789",
			jsonInput: `{invalid json}`,
			setupMock: func(m *MockPortService) {
				m.ListPortResourceTagsResult = map[string]string{}
			},
			expectedError: "failed to parse JSON",
		},
		{
			name:      "error with API tag listing",
			portUID:   "port-list-error",
			jsonInput: `{"environment": "production"}`,
			setupMock: func(m *MockPortService) {
				m.ListPortResourceTagsErr = fmt.Errorf("API error: resource not found")
			},
			expectedError: "failed to log in or list existing resource tags",
		},
		{
			name:      "error with API update",
			portUID:   "port-update-error",
			jsonInput: `{"environment": "production"}`,
			setupMock: func(m *MockPortService) {
				m.ListPortResourceTagsResult = map[string]string{}
				m.UpdatePortResourceTagsErr = fmt.Errorf("API error: unauthorized")
			},
			expectedError: "failed to update resource tags",
		},
		{
			name:      "empty tags clear all existing tags",
			portUID:   "port-clear",
			force:     true,
			jsonInput: `{}`,
			setupMock: func(m *MockPortService) {
				m.ListPortResourceTagsResult = map[string]string{"env": "staging"}
			},
			expectedOutput:       "Resource tags updated for Port port-clear",
			expectedCapturedTags: map[string]string{},
		},
		{
			name:    "no input provided",
			portUID: "port-no-input",
			setupMock: func(m *MockPortService) {
				m.ListPortResourceTagsResult = map[string]string{}
			},
			expectedError: "no input provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockPortService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.PortService = mockService
				return client, nil
			})

			cmd := &cobra.Command{
				Use: "update-tags [portUID]",
				RunE: func(cmd *cobra.Command, args []string) error {
					return UpdatePortResourceTags(cmd, args, false)
				},
			}

			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().Bool("force", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("tags", "", "")
			cmd.Flags().String("tags-file", "", "")
			cmd.Flags().String("resource-tags", "", "")

			if tt.interactive {
				assert.NoError(t, cmd.Flags().Set("interactive", "true"))
			}
			if tt.force {
				assert.NoError(t, cmd.Flags().Set("force", "true"))
			}
			if tt.jsonInput != "" {
				assert.NoError(t, cmd.Flags().Set("json", tt.jsonInput))
			}
			if tt.tagsInput != "" {
				assert.NoError(t, cmd.Flags().Set("tags", tt.tagsInput))
			}
			if tt.resourceTagsInput != "" {
				assert.NoError(t, cmd.Flags().Set("resource-tags", tt.resourceTagsInput))
			}

			var err error
			var capturedOutput, stderrOutput string
			stderrOutput = captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = cmd.RunE(cmd, []string{tt.portUID})
				})
			})
			capturedOutput += stderrOutput

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.expectedOutput != "" {
					assert.Contains(t, capturedOutput, tt.expectedOutput)
				}
				if tt.expectedCapturedTags != nil {
					assert.Equal(t, tt.expectedCapturedTags, mockService.CapturedResourceTags)
				}
			}
		})
	}
}

func TestListPorts(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	originalListPortsFunc := listPortsFunc
	defer func() {
		listPortsFunc = originalListPortsFunc
	}()

	allPorts := []*megaport.Port{
		{UID: "port-1", Name: "Sydney Port", LocationID: 100, PortSpeed: 1000, ProvisioningStatus: "LIVE"},
		{UID: "port-2", Name: "Melbourne Port", LocationID: 200, PortSpeed: 10000, ProvisioningStatus: "LIVE"},
		{UID: "port-3", Name: "Sydney Fast", LocationID: 100, PortSpeed: 10000, ProvisioningStatus: "LIVE"},
		{UID: "port-4", Name: "Decommissioned Port", LocationID: 100, PortSpeed: 1000, ProvisioningStatus: megaport.STATUS_DECOMMISSIONED},
	}

	tests := []struct {
		name            string
		locationID      int
		portSpeed       int
		portName        string
		includeInactive bool
		limit           int
		ports           []*megaport.Port
		listErr         error
		loginErr        error
		expectedError   string
		expectedOutputs []string
		notExpected     []string
	}{
		{
			name:            "list all ports no filters",
			ports:           allPorts,
			expectedOutputs: []string{"port-1", "port-2", "port-3"},
			notExpected:     []string{"port-4"},
		},
		{
			name:            "filter by location-id",
			locationID:      100,
			ports:           allPorts,
			expectedOutputs: []string{"port-1", "port-3"},
			notExpected:     []string{"port-2"},
		},
		{
			name:            "filter by port-speed",
			portSpeed:       10000,
			ports:           allPorts,
			expectedOutputs: []string{"port-2", "port-3"},
			notExpected:     []string{"port-1"},
		},
		{
			name:            "filter by port-name",
			portName:        "Sydney",
			ports:           allPorts,
			expectedOutputs: []string{"port-1", "port-3"},
			notExpected:     []string{"port-2"},
		},
		{
			name:            "include-inactive shows decommissioned",
			includeInactive: true,
			ports:           allPorts,
			expectedOutputs: []string{"port-1", "port-2", "port-3", "port-4"},
		},
		{
			name:            "empty result",
			ports:           []*megaport.Port{},
			expectedOutputs: []string{"No ports found"},
		},
		{
			name:          "API error",
			listErr:       fmt.Errorf("service unavailable"),
			expectedError: "failed to list ports",
		},
		{
			name:          "login error",
			loginErr:      fmt.Errorf("invalid credentials"),
			expectedError: "failed to log in",
		},
		{
			name:            "limit results",
			limit:           2,
			ports:           allPorts,
			expectedOutputs: []string{"port-1", "port-2"},
			notExpected:     []string{"port-3"},
		},
		{
			name:          "negative limit returns error",
			limit:         -1,
			ports:         allPorts,
			expectedError: "--limit must be a non-negative integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockPortService{}

			if tt.loginErr != nil {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					return nil, tt.loginErr
				})
			} else {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.PortService = mockService
					return client, nil
				})
			}

			if tt.listErr != nil {
				listPortsFunc = func(ctx context.Context, client *megaport.Client) ([]*megaport.Port, error) {
					return nil, tt.listErr
				}
			} else {
				listPortsFunc = func(ctx context.Context, client *megaport.Client) ([]*megaport.Port, error) {
					return tt.ports, nil
				}
			}

			cmd := &cobra.Command{Use: "list"}
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().Int("port-speed", 0, "")
			cmd.Flags().String("port-name", "", "")
			cmd.Flags().Bool("include-inactive", false, "")
			cmd.Flags().Int("limit", 0, "")

			if tt.locationID > 0 {
				require.NoError(t, cmd.Flags().Set("location-id", fmt.Sprintf("%d", tt.locationID)))
			}
			if tt.portSpeed > 0 {
				require.NoError(t, cmd.Flags().Set("port-speed", fmt.Sprintf("%d", tt.portSpeed)))
			}
			if tt.portName != "" {
				require.NoError(t, cmd.Flags().Set("port-name", tt.portName))
			}
			if tt.includeInactive {
				require.NoError(t, cmd.Flags().Set("include-inactive", "true"))
			}
			if tt.limit != 0 {
				require.NoError(t, cmd.Flags().Set("limit", fmt.Sprintf("%d", tt.limit)))
			}

			var err error
			var capturedOutput, stderrOutput string
			stderrOutput = captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = ListPorts(cmd, nil, true, "table")
				})
			})
			capturedOutput += stderrOutput

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				for _, expected := range tt.expectedOutputs {
					assert.Contains(t, capturedOutput, expected)
				}
				for _, notExp := range tt.notExpected {
					assert.NotContains(t, capturedOutput, notExp)
				}
			}
		})
	}
}

func TestBuyPort(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	originalBuyPortFunc := buyPortFunc
	defer func() {
		buyPortFunc = originalBuyPortFunc
	}()
	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	tests := []struct {
		name             string
		flags            map[string]string
		jsonInput        string
		setupMock        func(*MockPortService)
		buyPortOverride  func(ctx context.Context, client *megaport.Client, req *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error)
		expectedError    string
		expectedContains string
	}{
		{
			name: "success with flags",
			flags: map[string]string{
				"name":                   "test-port",
				"term":                   "12",
				"port-speed":             "1000",
				"location-id":            "1",
				"marketplace-visibility": "true",
			},
			setupMock: func(m *MockPortService) {},
			buyPortOverride: func(ctx context.Context, client *megaport.Client, req *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error) {
				return &megaport.BuyPortResponse{
					TechnicalServiceUIDs: []string{"new-port-uid-123"},
				}, nil
			},
			expectedContains: "new-port-uid-123",
		},
		{
			name:      "success with JSON",
			jsonInput: `{"name":"json-port","term":12,"portSpeed":1000,"locationId":1,"marketPlaceVisibility":false}`,
			setupMock: func(m *MockPortService) {},
			buyPortOverride: func(ctx context.Context, client *megaport.Client, req *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error) {
				return &megaport.BuyPortResponse{
					TechnicalServiceUIDs: []string{"json-port-uid"},
				}, nil
			},
			expectedContains: "json-port-uid",
		},
		{
			name: "API error from buyPortFunc",
			flags: map[string]string{
				"name":                   "test-port",
				"term":                   "12",
				"port-speed":             "1000",
				"location-id":            "1",
				"marketplace-visibility": "true",
			},
			setupMock: func(m *MockPortService) {},
			buyPortOverride: func(ctx context.Context, client *megaport.Client, req *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error) {
				return nil, fmt.Errorf("buy port failed")
			},
			expectedError: "buy port failed",
		},
		{
			name: "nil response from buyPortFunc",
			flags: map[string]string{
				"name":                   "test-port",
				"term":                   "12",
				"port-speed":             "1000",
				"location-id":            "1",
				"marketplace-visibility": "true",
			},
			setupMock: func(m *MockPortService) {},
			buyPortOverride: func(ctx context.Context, client *megaport.Client, req *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error) {
				return nil, nil
			},
			expectedError: "empty response from API",
		},
		{
			name: "validation error",
			flags: map[string]string{
				"name":                   "test-port",
				"term":                   "12",
				"port-speed":             "1000",
				"location-id":            "1",
				"marketplace-visibility": "true",
			},
			setupMock: func(m *MockPortService) {
				m.ValidatePortOrderErr = fmt.Errorf("invalid port configuration")
			},
			expectedError: "invalid port configuration",
		},
		{
			name:          "invalid JSON returns error",
			jsonInput:     `{bad json}`,
			setupMock:     func(m *MockPortService) {},
			expectedError: "failed to parse JSON",
		},
		{
			name:      "interactive combined with JSON is a usage error",
			jsonInput: `{"name":"json-port","term":12,"portSpeed":1000,"locationId":1,"marketPlaceVisibility":false}`,
			flags: map[string]string{
				"interactive": "true",
			},
			setupMock:     func(m *MockPortService) {},
			expectedError: "cannot be combined with",
		},
		{
			name: "interactive combined with flags is a usage error",
			flags: map[string]string{
				"interactive": "true",
				"name":        "test-port",
			},
			setupMock:     func(m *MockPortService) {},
			expectedError: "cannot be combined with",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockPortService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.PortService = mockService
				return client, nil
			})

			if tt.buyPortOverride != nil {
				buyPortFunc = tt.buyPortOverride
			} else {
				buyPortFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error) {
					return client.PortService.BuyPort(ctx, req)
				}
			}

			cmd := &cobra.Command{Use: "buy"}
			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("term", 0, "")
			cmd.Flags().Int("port-speed", 0, "")
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().Bool("marketplace-visibility", false, "")
			cmd.Flags().String("diversity-zone", "", "")
			cmd.Flags().Bool("cost-confirm", true, "")
			cmd.Flags().Bool("yes", false, "")

			if tt.jsonInput != "" {
				require.NoError(t, cmd.Flags().Set("json", tt.jsonInput))
				require.NoError(t, cmd.Flags().Set("yes", "true"))
			}
			for k, v := range tt.flags {
				require.NoError(t, cmd.Flags().Set(k, v))
			}

			var err error
			var capturedOutput, stderrOutput string
			stderrOutput = captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = BuyPort(cmd, nil, true)
				})
			})
			capturedOutput += stderrOutput

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.expectedContains != "" {
					assert.Contains(t, capturedOutput, tt.expectedContains)
				}
			}
		})
	}
}

func TestBuyPort_NoWaitFlag(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	originalBuyPortFunc := buyPortFunc
	originalGetPortFunc := getPortFunc
	defer func() {
		cleanup()
		buyPortFunc = originalBuyPortFunc
		getPortFunc = originalGetPortFunc
	}()
	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	tests := []struct {
		name       string
		noWait     bool
		expectPoll bool
	}{
		{
			name:       "default waits for provisioning",
			noWait:     false,
			expectPoll: true,
		},
		{
			name:       "no-wait skips provisioning wait",
			noWait:     true,
			expectPoll: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockPortService{}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.PortService = mockService
				return client, nil
			})

			var capturedReq *megaport.BuyPortRequest
			buyPortFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error) {
				capturedReq = req
				return &megaport.BuyPortResponse{
					TechnicalServiceUIDs: []string{"port-uid-123"},
				}, nil
			}
			getPortCalls := 0
			getPortFunc = func(ctx context.Context, client *megaport.Client, portUID string) (*megaport.Port, error) {
				getPortCalls++
				return &megaport.Port{UID: portUID, ProvisioningStatus: "LIVE"}, nil
			}

			cmd := &cobra.Command{Use: "buy"}
			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().Bool("no-wait", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("term", 0, "")
			cmd.Flags().Int("port-speed", 0, "")
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().Bool("marketplace-visibility", false, "")
			cmd.Flags().String("diversity-zone", "", "")
			cmd.Flags().Bool("cost-confirm", true, "")

			require.NoError(t, cmd.Flags().Set("name", "test-port"))
			require.NoError(t, cmd.Flags().Set("term", "12"))
			require.NoError(t, cmd.Flags().Set("port-speed", "1000"))
			require.NoError(t, cmd.Flags().Set("location-id", "1"))
			require.NoError(t, cmd.Flags().Set("marketplace-visibility", "true"))
			if tt.noWait {
				require.NoError(t, cmd.Flags().Set("no-wait", "true"))
			}

			var err error
			output.CaptureOutput(func() {
				err = BuyPort(cmd, nil, true)
			})

			assert.NoError(t, err)
			require.NotNil(t, capturedReq)
			// The SDK must never poll; provisioning is awaited outside the order retry.
			assert.False(t, capturedReq.WaitForProvision)
			if tt.expectPoll {
				assert.Positive(t, getPortCalls, "expected a provisioning poll when not using --no-wait")
			} else {
				assert.Zero(t, getPortCalls, "no provisioning poll expected with --no-wait")
			}
		})
	}
}

func TestRestorePort(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	originalRestorePortFunc := restorePortFunc
	defer func() {
		restorePortFunc = originalRestorePortFunc
	}()

	tests := []struct {
		name          string
		portUID       string
		restoreResp   *megaport.RestorePortResponse
		restoreErr    error
		loginErr      error
		expectedError string
		expectedOut   string
	}{
		{
			name:        "success",
			portUID:     "port-restore-1",
			restoreResp: &megaport.RestorePortResponse{IsRestored: true},
			expectedOut: "restored successfully",
		},
		{
			name:          "API error",
			portUID:       "port-restore-err",
			restoreErr:    fmt.Errorf("restore service unavailable"),
			expectedError: "restore service unavailable",
		},
		{
			name:          "login error",
			portUID:       "port-restore-login",
			loginErr:      fmt.Errorf("authentication failed"),
			expectedError: "authentication failed",
		},
		{
			name:          "not successful",
			portUID:       "port-restore-fail",
			restoreResp:   &megaport.RestorePortResponse{IsRestored: false},
			expectedError: "not successful for port-restore-fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.loginErr != nil {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					return nil, tt.loginErr
				})
			} else {
				mockService := &MockPortService{}
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.PortService = mockService
					return client, nil
				})
			}

			if tt.restoreErr != nil {
				restorePortFunc = func(ctx context.Context, client *megaport.Client, portUID string) (*megaport.RestorePortResponse, error) {
					return nil, tt.restoreErr
				}
			} else if tt.restoreResp != nil {
				resp := tt.restoreResp
				restorePortFunc = func(ctx context.Context, client *megaport.Client, portUID string) (*megaport.RestorePortResponse, error) {
					return resp, nil
				}
			}

			cmd := &cobra.Command{Use: "restore"}

			var err error
			var capturedOutput, stderrOutput string
			stderrOutput = captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = RestorePort(cmd, []string{tt.portUID}, true)
				})
			})
			capturedOutput += stderrOutput

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

func TestLockPort(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	originalLockPortFunc := lockPortFunc
	defer func() {
		lockPortFunc = originalLockPortFunc
	}()

	tests := []struct {
		name          string
		portUID       string
		lockResp      *megaport.LockPortResponse
		lockErr       error
		loginErr      error
		expectedError string
		expectedOut   string
	}{
		{
			name:        "success",
			portUID:     "port-lock-1",
			lockResp:    &megaport.LockPortResponse{IsLocking: true},
			expectedOut: "locked successfully",
		},
		{
			name:          "API error",
			portUID:       "port-lock-err",
			lockErr:       fmt.Errorf("lock service unavailable"),
			expectedError: "lock service unavailable",
		},
		{
			name:          "login error",
			portUID:       "port-lock-login",
			loginErr:      fmt.Errorf("invalid token"),
			expectedError: "invalid token",
		},
		{
			name:          "not successful",
			portUID:       "port-lock-fail",
			lockResp:      &megaport.LockPortResponse{IsLocking: false},
			expectedError: "not successful for port-lock-fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.loginErr != nil {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					return nil, tt.loginErr
				})
			} else {
				mockService := &MockPortService{}
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.PortService = mockService
					return client, nil
				})
			}

			if tt.lockErr != nil {
				lockPortFunc = func(ctx context.Context, client *megaport.Client, portUID string) (*megaport.LockPortResponse, error) {
					return nil, tt.lockErr
				}
			} else if tt.lockResp != nil {
				resp := tt.lockResp
				lockPortFunc = func(ctx context.Context, client *megaport.Client, portUID string) (*megaport.LockPortResponse, error) {
					return resp, nil
				}
			}

			cmd := &cobra.Command{Use: "lock"}

			var err error
			var capturedOutput, stderrOutput string
			stderrOutput = captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = LockPort(cmd, []string{tt.portUID}, true)
				})
			})
			capturedOutput += stderrOutput

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

func TestUnlockPort(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	originalUnlockPortFunc := unlockPortFunc
	defer func() {
		unlockPortFunc = originalUnlockPortFunc
	}()

	tests := []struct {
		name          string
		portUID       string
		unlockResp    *megaport.UnlockPortResponse
		unlockErr     error
		loginErr      error
		expectedError string
		expectedOut   string
	}{
		{
			name:        "success",
			portUID:     "port-unlock-1",
			unlockResp:  &megaport.UnlockPortResponse{IsUnlocking: true},
			expectedOut: "unlocked successfully",
		},
		{
			name:          "API error",
			portUID:       "port-unlock-err",
			unlockErr:     fmt.Errorf("unlock service unavailable"),
			expectedError: "unlock service unavailable",
		},
		{
			name:          "login error",
			portUID:       "port-unlock-login",
			loginErr:      fmt.Errorf("session expired"),
			expectedError: "session expired",
		},
		{
			name:          "not successful",
			portUID:       "port-unlock-fail",
			unlockResp:    &megaport.UnlockPortResponse{IsUnlocking: false},
			expectedError: "not successful for port-unlock-fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.loginErr != nil {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					return nil, tt.loginErr
				})
			} else {
				mockService := &MockPortService{}
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.PortService = mockService
					return client, nil
				})
			}

			if tt.unlockErr != nil {
				unlockPortFunc = func(ctx context.Context, client *megaport.Client, portUID string) (*megaport.UnlockPortResponse, error) {
					return nil, tt.unlockErr
				}
			} else if tt.unlockResp != nil {
				resp := tt.unlockResp
				unlockPortFunc = func(ctx context.Context, client *megaport.Client, portUID string) (*megaport.UnlockPortResponse, error) {
					return resp, nil
				}
			}

			cmd := &cobra.Command{Use: "unlock"}

			var err error
			var capturedOutput, stderrOutput string
			stderrOutput = captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = UnlockPort(cmd, []string{tt.portUID}, true)
				})
			})
			capturedOutput += stderrOutput

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

func TestCheckPortVLANAvailability(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	originalCheckFunc := checkPortVLANAvailabilityFunc
	defer func() {
		checkPortVLANAvailabilityFunc = originalCheckFunc
	}()

	tests := []struct {
		name          string
		portUID       string
		vlanArg       string
		available     bool
		checkErr      error
		loginErr      error
		expectedError string
		expectedOut   string
	}{
		{
			name:        "available",
			portUID:     "port-vlan-1",
			vlanArg:     "100",
			available:   true,
			expectedOut: "is available",
		},
		{
			name:        "not available",
			portUID:     "port-vlan-2",
			vlanArg:     "200",
			available:   false,
			expectedOut: "is not available",
		},
		{
			name:          "invalid VLAN arg",
			portUID:       "port-vlan-3",
			vlanArg:       "abc",
			expectedError: "Invalid VLAN ID",
		},
		{
			name:          "VLAN out of assignable range",
			portUID:       "port-vlan-5",
			vlanArg:       "4094",
			expectedError: "must be between",
		},
		{
			name:          "API error",
			portUID:       "port-vlan-4",
			vlanArg:       "300",
			checkErr:      fmt.Errorf("VLAN check failed"),
			expectedError: "VLAN check failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.loginErr != nil {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					return nil, tt.loginErr
				})
			} else {
				mockService := &MockPortService{}
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.PortService = mockService
					return client, nil
				})
			}

			if tt.checkErr != nil {
				checkPortVLANAvailabilityFunc = func(ctx context.Context, client *megaport.Client, portUID string, vlan int) (bool, error) {
					return false, tt.checkErr
				}
			} else {
				avail := tt.available
				checkPortVLANAvailabilityFunc = func(ctx context.Context, client *megaport.Client, portUID string, vlan int) (bool, error) {
					return avail, nil
				}
			}

			cmd := &cobra.Command{Use: "check-vlan"}

			var err error
			var capturedOutput, stderrOutput string
			stderrOutput = captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = CheckPortVLANAvailability(cmd, []string{tt.portUID, tt.vlanArg}, true)
				})
			})
			capturedOutput += stderrOutput

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

func TestUpdatePort(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	originalGetPortFunc := getPortFunc
	originalUpdatePortFunc := updatePortFunc
	defer func() {
		getPortFunc = originalGetPortFunc
		updatePortFunc = originalUpdatePortFunc
	}()

	tests := []struct {
		name             string
		portUID          string
		flags            map[string]string
		jsonInput        string
		getPortResult    *megaport.Port
		getPortErr       error
		updateResult     *megaport.ModifyPortResponse
		updateErr        error
		expectedError    string
		expectedContains string
	}{
		{
			name:    "success with flags",
			portUID: "port-update-1",
			flags:   map[string]string{"name": "Updated Port Name"},
			getPortResult: &megaport.Port{
				UID:                "port-update-1",
				Name:               "Original Port",
				ProvisioningStatus: "LIVE",
			},
			updateResult:     &megaport.ModifyPortResponse{IsUpdated: true},
			expectedContains: "port-update-1",
		},
		{
			name:      "success with JSON",
			portUID:   "port-update-2",
			jsonInput: `{"name":"JSON Updated"}`,
			getPortResult: &megaport.Port{
				UID:                "port-update-2",
				Name:               "Original Port",
				ProvisioningStatus: "LIVE",
			},
			updateResult:     &megaport.ModifyPortResponse{IsUpdated: true},
			expectedContains: "port-update-2",
		},
		{
			name:          "get original error",
			portUID:       "port-update-3",
			flags:         map[string]string{"name": "New Name"},
			getPortErr:    fmt.Errorf("port not found"),
			expectedError: "port not found",
		},
		{
			name:    "update error",
			portUID: "port-update-4",
			flags:   map[string]string{"name": "New Name"},
			getPortResult: &megaport.Port{
				UID:                "port-update-4",
				Name:               "Original Port",
				ProvisioningStatus: "LIVE",
			},
			updateErr:     fmt.Errorf("update rejected"),
			expectedError: "update rejected",
		},
		{
			name:    "no fields provided",
			portUID: "port-update-5",
			getPortResult: &megaport.Port{
				UID:                "port-update-5",
				Name:               "Original Port",
				ProvisioningStatus: "LIVE",
			},
			expectedError: "at least one field must be updated",
		},
		{
			name:          "interactive combined with flags is a usage error",
			portUID:       "port-update-6",
			flags:         map[string]string{"interactive": "true", "name": "Updated Port Name"},
			expectedError: "cannot be combined with",
		},
		{
			name:          "interactive combined with JSON is a usage error",
			portUID:       "port-update-7",
			flags:         map[string]string{"interactive": "true"},
			jsonInput:     `{"name":"JSON Updated"}`,
			expectedError: "cannot be combined with",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockPortService{}
			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.PortService = mockService
				return client, nil
			})

			getCallCount := 0
			if tt.getPortErr != nil {
				getPortFunc = func(ctx context.Context, client *megaport.Client, portUID string) (*megaport.Port, error) {
					return nil, tt.getPortErr
				}
			} else if tt.getPortResult != nil {
				result := tt.getPortResult
				getPortFunc = func(ctx context.Context, client *megaport.Client, portUID string) (*megaport.Port, error) {
					getCallCount++
					if getCallCount == 1 {
						return result, nil
					}
					// Return updated port on second call
					updated := *result
					if tt.flags != nil {
						if n, ok := tt.flags["name"]; ok {
							updated.Name = n
						}
					}
					return &updated, nil
				}
			}

			if tt.updateErr != nil {
				updatePortFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ModifyPortRequest) (*megaport.ModifyPortResponse, error) {
					return nil, tt.updateErr
				}
			} else if tt.updateResult != nil {
				resp := tt.updateResult
				updatePortFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ModifyPortRequest) (*megaport.ModifyPortResponse, error) {
					return resp, nil
				}
			}

			cmd := &cobra.Command{Use: "update"}
			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().Bool("marketplace-visibility", false, "")
			cmd.Flags().String("cost-centre", "", "")
			cmd.Flags().Int("term", 0, "")

			if tt.jsonInput != "" {
				require.NoError(t, cmd.Flags().Set("json", tt.jsonInput))
			}
			for k, v := range tt.flags {
				require.NoError(t, cmd.Flags().Set(k, v))
			}

			var err error
			var capturedOutput, stderrOutput string
			stderrOutput = captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = UpdatePort(cmd, []string{tt.portUID}, true)
				})
			})
			capturedOutput += stderrOutput

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.expectedContains != "" {
					assert.Contains(t, capturedOutput, tt.expectedContains)
				}
			}
		})
	}
}

// End-to-end proof that a name-only update re-sends the port's existing cost
// centre rather than wiping it (the SDK omits omitempty on costCentre).
func TestUpdatePort_PreservesCostCentre(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	originalGetPortFunc := getPortFunc
	originalUpdatePortFunc := updatePortFunc
	defer func() {
		getPortFunc = originalGetPortFunc
		updatePortFunc = originalUpdatePortFunc
	}()

	mockService := &MockPortService{}
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.PortService = mockService
		return client, nil
	})

	getPortFunc = func(ctx context.Context, client *megaport.Client, portUID string) (*megaport.Port, error) {
		return &megaport.Port{UID: portUID, Name: "Original", CostCentre: "IT Dept", ProvisioningStatus: "LIVE"}, nil
	}

	var captured *megaport.ModifyPortRequest
	updatePortFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ModifyPortRequest) (*megaport.ModifyPortResponse, error) {
		captured = req
		return &megaport.ModifyPortResponse{IsUpdated: true}, nil
	}

	cmd := &cobra.Command{Use: "update"}
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().Bool("marketplace-visibility", false, "")
	cmd.Flags().String("cost-centre", "", "")
	cmd.Flags().Int("term", 0, "")
	require.NoError(t, cmd.Flags().Set("name", "Renamed Port"))

	_ = captureStderr(t, func() {
		output.CaptureOutput(func() {
			require.NoError(t, UpdatePort(cmd, []string{"port-cc-1"}, true))
		})
	})

	require.NotNil(t, captured)
	assert.Equal(t, "IT Dept", captured.CostCentre, "name-only update must preserve the existing cost centre")
}

func newUpdatePortCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "update"}
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().Bool("marketplace-visibility", false, "")
	cmd.Flags().String("cost-centre", "", "")
	cmd.Flags().Int("term", 0, "")
	return cmd
}

func TestUpdatePort_CostCentreOverrideAndClear(t *testing.T) {
	originalGetPortFunc := getPortFunc
	originalUpdatePortFunc := updatePortFunc
	defer func() {
		getPortFunc = originalGetPortFunc
		updatePortFunc = originalUpdatePortFunc
	}()

	tests := []struct {
		name     string
		setFlags func(*cobra.Command)
		expected string
	}{
		{
			name:     "explicit cost centre overrides current",
			setFlags: func(c *cobra.Command) { require.NoError(t, c.Flags().Set("cost-centre", "Finance")) },
			expected: "Finance",
		},
		{
			name: "explicit empty cost centre clears it",
			setFlags: func(c *cobra.Command) {
				require.NoError(t, c.Flags().Set("name", "Renamed"))
				require.NoError(t, c.Flags().Set("cost-centre", ""))
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
			defer cleanup()

			mockService := &MockPortService{}
			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.PortService = mockService
				return client, nil
			})

			getPortFunc = func(ctx context.Context, client *megaport.Client, portUID string) (*megaport.Port, error) {
				return &megaport.Port{UID: portUID, Name: "Original", CostCentre: "IT Dept", ProvisioningStatus: "LIVE"}, nil
			}

			var captured *megaport.ModifyPortRequest
			updatePortFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ModifyPortRequest) (*megaport.ModifyPortResponse, error) {
				captured = req
				return &megaport.ModifyPortResponse{IsUpdated: true}, nil
			}

			cmd := newUpdatePortCmd()
			tt.setFlags(cmd)

			_ = captureStderr(t, func() {
				output.CaptureOutput(func() {
					require.NoError(t, UpdatePort(cmd, []string{"port-cc-1"}, true))
				})
			})

			require.NotNil(t, captured)
			assert.Equal(t, tt.expected, captured.CostCentre)
		})
	}
}

// Bad input must fail before any network round-trip, so login/fetch errors
// never mask the real validation error.
func TestUpdatePort_ValidatesBeforeLogin(t *testing.T) {
	cleanup := testutil.SetupLoginError(fmt.Errorf("login must not be reached"))
	defer cleanup()

	cmd := newUpdatePortCmd()
	require.NoError(t, cmd.Flags().Set("term", "99"))

	var err error
	_ = captureStderr(t, func() {
		output.CaptureOutput(func() {
			err = UpdatePort(cmd, []string{"port-1"}, true)
		})
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "contract term", "input must be validated before login")
}

func TestBuyPort_Confirmation(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	originalBuyPortFunc := buyPortFunc
	defer func() {
		buyPortFunc = originalBuyPortFunc
	}()
	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()

	tests := []struct {
		name                string
		flags               map[string]string
		jsonInput           string
		yesFlag             bool
		confirmReturn       bool
		expectPromptCalled  bool
		expectedError       string
		expectedContains    string
		expectedNotContains string
	}{
		{
			name: "confirmation accepted",
			flags: map[string]string{
				"name":                   "test-port",
				"term":                   "12",
				"port-speed":             "1000",
				"location-id":            "1",
				"marketplace-visibility": "true",
			},
			confirmReturn:      true,
			expectPromptCalled: true,
			expectedContains:   "new-port-uid-123",
		},
		{
			name: "confirmation denied",
			flags: map[string]string{
				"name":                   "test-port",
				"term":                   "12",
				"port-speed":             "1000",
				"location-id":            "1",
				"marketplace-visibility": "true",
			},
			confirmReturn:      false,
			expectPromptCalled: true,
			expectedError:      "cancelled by user",
		},
		{
			name: "yes flag skips confirmation",
			flags: map[string]string{
				"name":                   "test-port",
				"term":                   "12",
				"port-speed":             "1000",
				"location-id":            "1",
				"marketplace-visibility": "true",
			},
			yesFlag:            true,
			expectPromptCalled: false,
			expectedContains:   "new-port-uid-123",
		},
		{
			name:               "json input with yes skips confirmation",
			jsonInput:          `{"name":"json-port","term":12,"portSpeed":1000,"locationId":1,"marketPlaceVisibility":false}`,
			yesFlag:            true,
			expectPromptCalled: false,
			expectedContains:   "new-port-uid-123",
		},
		{
			name:          "json input without yes is a usage error",
			jsonInput:     `{"name":"json-port","term":12,"portSpeed":1000,"locationId":1,"marketPlaceVisibility":false}`,
			expectedError: "--yes is required to confirm a purchase when using --json or --json-file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockPortService{}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.PortService = mockService
				return client, nil
			})

			buyPortFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error) {
				return &megaport.BuyPortResponse{
					TechnicalServiceUIDs: []string{"new-port-uid-123"},
				}, nil
			}

			promptCalled := false
			utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool {
				promptCalled = true
				return tt.confirmReturn
			})

			cmd := &cobra.Command{Use: "buy"}
			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().Bool("no-wait", false, "")
			cmd.Flags().Bool("yes", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("term", 0, "")
			cmd.Flags().Int("port-speed", 0, "")
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().Bool("marketplace-visibility", false, "")
			cmd.Flags().String("diversity-zone", "", "")
			cmd.Flags().Bool("cost-confirm", true, "")

			if tt.jsonInput != "" {
				require.NoError(t, cmd.Flags().Set("json", tt.jsonInput))
			}
			for k, v := range tt.flags {
				require.NoError(t, cmd.Flags().Set(k, v))
			}
			if tt.yesFlag {
				require.NoError(t, cmd.Flags().Set("yes", "true"))
			}

			var err error
			var capturedOutput, stderrOutput string
			stderrOutput = captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = BuyPort(cmd, nil, true)
				})
			})
			capturedOutput += stderrOutput

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.expectedContains != "" {
					assert.Contains(t, capturedOutput, tt.expectedContains)
				}
				if tt.expectedNotContains != "" {
					assert.NotContains(t, capturedOutput, tt.expectedNotContains)
				}
			}
			assert.Equal(t, tt.expectPromptCalled, promptCalled, "BuyConfirmPrompt called expectation mismatch")
		})
	}
}

func TestBuyLAGPort_NoWaitFlag(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	originalBuyPortFunc := buyPortFunc
	originalGetPortFunc := getPortFunc
	defer func() {
		cleanup()
		buyPortFunc = originalBuyPortFunc
		getPortFunc = originalGetPortFunc
	}()
	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	tests := []struct {
		name       string
		noWait     bool
		expectPoll bool
	}{
		{
			name:       "default waits for provisioning",
			noWait:     false,
			expectPoll: true,
		},
		{
			name:       "no-wait skips provisioning wait",
			noWait:     true,
			expectPoll: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.PortService = &MockPortService{}
				return client, nil
			})

			var capturedReq *megaport.BuyPortRequest
			buyPortFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error) {
				capturedReq = req
				return &megaport.BuyPortResponse{
					TechnicalServiceUIDs: []string{"lag-uid-123"},
				}, nil
			}
			getPortCalls := 0
			getPortFunc = func(ctx context.Context, client *megaport.Client, portUID string) (*megaport.Port, error) {
				getPortCalls++
				return &megaport.Port{UID: portUID, ProvisioningStatus: "LIVE"}, nil
			}

			cmd := &cobra.Command{Use: "buy-lag"}
			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().Bool("no-wait", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("term", 0, "")
			cmd.Flags().Int("port-speed", 0, "")
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().Bool("marketplace-visibility", false, "")
			cmd.Flags().String("diversity-zone", "", "")
			cmd.Flags().Bool("cost-confirm", true, "")
			cmd.Flags().Int("lag-count", 0, "")

			require.NoError(t, cmd.Flags().Set("name", "test-lag"))
			require.NoError(t, cmd.Flags().Set("term", "12"))
			require.NoError(t, cmd.Flags().Set("port-speed", "10000"))
			require.NoError(t, cmd.Flags().Set("location-id", "1"))
			require.NoError(t, cmd.Flags().Set("lag-count", "2"))
			if tt.noWait {
				require.NoError(t, cmd.Flags().Set("no-wait", "true"))
			}

			var err error
			output.CaptureOutput(func() {
				err = BuyLAGPort(cmd, nil, true)
			})

			assert.NoError(t, err)
			require.NotNil(t, capturedReq)
			// The SDK must never poll; provisioning is awaited outside the order retry.
			assert.False(t, capturedReq.WaitForProvision)
			if tt.expectPoll {
				assert.Positive(t, getPortCalls, "expected a provisioning poll when not using --no-wait")
			} else {
				assert.Zero(t, getPortCalls, "no provisioning poll expected with --no-wait")
			}
		})
	}
}

func TestBuyLAGPort_ConfirmationDenied(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.PortService = &MockPortService{}
	})
	defer cleanup()

	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return false })

	cmd := &cobra.Command{Use: "buy-lag"}
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().Bool("no-wait", false, "")
	cmd.Flags().Bool("yes", false, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().Int("term", 0, "")
	cmd.Flags().Int("port-speed", 0, "")
	cmd.Flags().Int("location-id", 0, "")
	cmd.Flags().Bool("marketplace-visibility", false, "")
	cmd.Flags().String("diversity-zone", "", "")
	cmd.Flags().Bool("cost-confirm", true, "")
	cmd.Flags().Int("lag-count", 0, "")

	require.NoError(t, cmd.Flags().Set("name", "test-lag"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("port-speed", "10000"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))
	require.NoError(t, cmd.Flags().Set("lag-count", "2"))

	var err error
	var capturedOutput, stderrOutput string
	stderrOutput = captureStderr(t, func() {
		capturedOutput = output.CaptureOutput(func() {
			err = BuyLAGPort(cmd, nil, true)
		})
	})
	capturedOutput += stderrOutput
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled by user")
	assert.Contains(t, capturedOutput, "Purchase cancelled")
}

func TestBuyLAGPort_JSONWithoutYes(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.PortService = &MockPortService{}
	})
	defer cleanup()

	cmd := &cobra.Command{Use: "buy-lag"}
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().Bool("no-wait", false, "")
	cmd.Flags().Bool("yes", false, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().Int("term", 0, "")
	cmd.Flags().Int("port-speed", 0, "")
	cmd.Flags().Int("location-id", 0, "")
	cmd.Flags().Bool("marketplace-visibility", false, "")
	cmd.Flags().String("diversity-zone", "", "")
	cmd.Flags().Bool("cost-confirm", true, "")
	cmd.Flags().Int("lag-count", 0, "")

	require.NoError(t, cmd.Flags().Set("json", `{"name":"json-lag","term":12,"portSpeed":10000,"locationId":1,"lagCount":2,"marketPlaceVisibility":true}`))

	err := BuyLAGPort(cmd, nil, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--yes is required to confirm a purchase when using --json or --json-file")
}

func TestValidatePort(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name             string
		flags            map[string]string
		jsonInput        string
		jsonFileContent  string
		setupMock        func(*MockPortService)
		loginError       error
		expectedError    string
		expectedContains string
	}{
		{
			name: "success with flags",
			flags: map[string]string{
				"name":                   "test-port",
				"term":                   "12",
				"port-speed":             "1000",
				"location-id":            "1",
				"marketplace-visibility": "true",
			},
			setupMock:        func(m *MockPortService) {},
			expectedContains: "validation passed",
		},
		{
			name:             "success with JSON",
			jsonInput:        `{"name":"json-port","term":12,"portSpeed":1000,"locationId":1,"marketPlaceVisibility":false}`,
			setupMock:        func(m *MockPortService) {},
			expectedContains: "validation passed",
		},
		{
			name: "validation error",
			flags: map[string]string{
				"name":                   "test-port",
				"term":                   "12",
				"port-speed":             "1000",
				"location-id":            "1",
				"marketplace-visibility": "true",
			},
			setupMock: func(m *MockPortService) {
				m.ValidatePortOrderErr = fmt.Errorf("invalid port configuration")
			},
			expectedError: "invalid port configuration",
		},
		{
			name:          "no input provided",
			flags:         map[string]string{},
			setupMock:     func(m *MockPortService) {},
			expectedError: "no input provided",
		},
		{
			name: "login error",
			flags: map[string]string{
				"name":                   "test-port",
				"term":                   "12",
				"port-speed":             "1000",
				"location-id":            "1",
				"marketplace-visibility": "true",
			},
			setupMock:     func(m *MockPortService) {},
			loginError:    fmt.Errorf("authentication failed"),
			expectedError: "authentication failed",
		},
		{
			name:          "invalid JSON input",
			jsonInput:     `{invalid json}`,
			setupMock:     func(m *MockPortService) {},
			expectedError: "failed to parse JSON",
		},
		{
			name:             "success with JSON file",
			jsonFileContent:  `{"name":"file-port","term":12,"portSpeed":1000,"locationId":1,"marketPlaceVisibility":false}`,
			setupMock:        func(m *MockPortService) {},
			expectedContains: "validation passed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockPortService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			if tt.loginError != nil {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					return nil, tt.loginError
				})
			} else {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.PortService = mockService
					return client, nil
				})
			}

			cmd := &cobra.Command{Use: "validate"}
			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("term", 0, "")
			cmd.Flags().Int("port-speed", 0, "")
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().Bool("marketplace-visibility", false, "")
			cmd.Flags().String("diversity-zone", "", "")

			if tt.jsonInput != "" {
				require.NoError(t, cmd.Flags().Set("json", tt.jsonInput))
			}
			if tt.jsonFileContent != "" {
				tmpFile, tmpErr := os.CreateTemp("", "port-validate-*.json")
				require.NoError(t, tmpErr)
				defer os.Remove(tmpFile.Name())
				_, tmpErr = tmpFile.WriteString(tt.jsonFileContent)
				require.NoError(t, tmpErr)
				tmpFile.Close()
				require.NoError(t, cmd.Flags().Set("json-file", tmpFile.Name()))
			}
			for k, v := range tt.flags {
				require.NoError(t, cmd.Flags().Set(k, v))
			}

			var err error
			var capturedOutput, stderrOutput string
			stderrOutput = captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = ValidatePort(cmd, nil, true)
				})
			})
			capturedOutput += stderrOutput

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.expectedContains != "" {
					assert.Contains(t, capturedOutput, tt.expectedContains)
				}
			}
		})
	}
}

func TestValidateLAGPort(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name             string
		flags            map[string]string
		jsonInput        string
		jsonFileContent  string
		setupMock        func(*MockPortService)
		loginError       error
		expectedError    string
		expectedContains string
	}{
		{
			name: "success with flags",
			flags: map[string]string{
				"name":                   "test-lag",
				"term":                   "12",
				"port-speed":             "10000",
				"location-id":            "1",
				"lag-count":              "2",
				"marketplace-visibility": "true",
			},
			setupMock:        func(m *MockPortService) {},
			expectedContains: "validation passed",
		},
		{
			name:             "success with JSON",
			jsonInput:        `{"name":"json-lag","term":12,"portSpeed":10000,"locationId":1,"lagCount":2,"marketPlaceVisibility":true}`,
			setupMock:        func(m *MockPortService) {},
			expectedContains: "validation passed",
		},
		{
			name: "validation error",
			flags: map[string]string{
				"name":                   "test-lag",
				"term":                   "12",
				"port-speed":             "10000",
				"location-id":            "1",
				"lag-count":              "2",
				"marketplace-visibility": "true",
			},
			setupMock: func(m *MockPortService) {
				m.ValidatePortOrderErr = fmt.Errorf("invalid LAG configuration")
			},
			expectedError: "invalid LAG configuration",
		},
		{
			name:          "no input provided",
			flags:         map[string]string{},
			setupMock:     func(m *MockPortService) {},
			expectedError: "no input provided",
		},
		{
			name: "login error",
			flags: map[string]string{
				"name":                   "test-lag",
				"term":                   "12",
				"port-speed":             "10000",
				"location-id":            "1",
				"lag-count":              "2",
				"marketplace-visibility": "true",
			},
			setupMock:     func(m *MockPortService) {},
			loginError:    fmt.Errorf("authentication failed"),
			expectedError: "authentication failed",
		},
		{
			name:          "invalid JSON input",
			jsonInput:     `{invalid json}`,
			setupMock:     func(m *MockPortService) {},
			expectedError: "failed to parse JSON",
		},
		{
			name:             "success with JSON file",
			jsonFileContent:  `{"name":"file-lag","term":12,"portSpeed":10000,"locationId":1,"lagCount":2,"marketPlaceVisibility":true}`,
			setupMock:        func(m *MockPortService) {},
			expectedContains: "validation passed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockPortService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			if tt.loginError != nil {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					return nil, tt.loginError
				})
			} else {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.PortService = mockService
					return client, nil
				})
			}

			cmd := &cobra.Command{Use: "validate-lag"}
			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("term", 0, "")
			cmd.Flags().Int("port-speed", 0, "")
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().Int("lag-count", 0, "")
			cmd.Flags().Bool("marketplace-visibility", false, "")
			cmd.Flags().String("diversity-zone", "", "")

			if tt.jsonInput != "" {
				require.NoError(t, cmd.Flags().Set("json", tt.jsonInput))
			}
			if tt.jsonFileContent != "" {
				tmpFile, tmpErr := os.CreateTemp("", "lag-validate-*.json")
				require.NoError(t, tmpErr)
				defer os.Remove(tmpFile.Name())
				_, tmpErr = tmpFile.WriteString(tt.jsonFileContent)
				require.NoError(t, tmpErr)
				tmpFile.Close()
				require.NoError(t, cmd.Flags().Set("json-file", tmpFile.Name()))
			}
			for k, v := range tt.flags {
				require.NoError(t, cmd.Flags().Set(k, v))
			}

			var err error
			var capturedOutput, stderrOutput string
			stderrOutput = captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = ValidateLAGPort(cmd, nil, true)
				})
			})
			capturedOutput += stderrOutput

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.expectedContains != "" {
					assert.Contains(t, capturedOutput, tt.expectedContains)
				}
			}
		})
	}
}

func TestDeletePort_Comprehensive(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	originalDeletePortFunc := deletePortFunc
	defer func() { deletePortFunc = originalDeletePortFunc }()
	originalConfirmPrompt := utils.GetConfirmPrompt()
	defer func() { utils.SetConfirmPrompt(originalConfirmPrompt) }()

	tests := []struct {
		name             string
		force            bool
		confirmResult    bool
		deleteErr        error
		isDeleting       bool
		expectedError    string
		expectedContains string
	}{
		{
			name:             "success with force flag",
			force:            true,
			isDeleting:       true,
			expectedContains: "port-123",
		},
		{
			name:          "user cancels confirmation",
			force:         false,
			confirmResult: false,
			expectedError: "cancelled by user",
		},
		{
			name:             "user confirms deletion",
			force:            false,
			confirmResult:    true,
			isDeleting:       true,
			expectedContains: "port-123",
		},
		{
			name:          "delete API error",
			force:         true,
			deleteErr:     fmt.Errorf("API failure"),
			expectedError: "API failure",
		},
		{
			name:          "delete returns not deleting",
			force:         true,
			isDeleting:    false,
			expectedError: "not successful for port-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockPortService{}
			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.PortService = mockService
				return client, nil
			})

			utils.SetConfirmPrompt(func(_ string, _ bool) bool {
				return tt.confirmResult
			})

			var capturedReq *megaport.DeletePortRequest
			deletePortFunc = func(ctx context.Context, client *megaport.Client, req *megaport.DeletePortRequest) (*megaport.DeletePortResponse, error) {
				capturedReq = req
				if tt.deleteErr != nil {
					return nil, tt.deleteErr
				}
				return &megaport.DeletePortResponse{IsDeleting: tt.isDeleting}, nil
			}

			cmd := &cobra.Command{Use: "delete"}
			cmd.Flags().Bool("force", false, "")
			cmd.Flags().Bool("safe-delete", false, "")

			if tt.force {
				require.NoError(t, cmd.Flags().Set("force", "true"))
			}

			var err error
			var capturedOutput, stderrOutput string
			stderrOutput = captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = DeletePort(cmd, []string{"port-123"}, true)
				})
			})
			capturedOutput += stderrOutput

			// Ports must always be deleted immediately — the previous
			// "terminate later" option is no longer supported.
			if capturedReq != nil {
				assert.True(t, capturedReq.DeleteNow, "DeletePort must always set DeleteNow=true")
			}

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.expectedContains != "" {
					assert.Contains(t, capturedOutput, tt.expectedContains)
				}
			}
		})
	}
}

func TestGetPort_NilPort(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	mockService := &MockPortService{ForceNilGetPort: true}
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.PortService = mockService
		return client, nil
	})

	var err error
	output.CaptureOutput(func() {
		err = testutil.OutputAdapter(GetPort)(
			testutil.NewCommand("get", testutil.OutputAdapter(GetPort)),
			[]string{"port-nil"},
		)
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no port found")
}

func TestGetPort_LoginError(t *testing.T) {
	cleanup := testutil.SetupLoginError(fmt.Errorf("auth failed"))
	defer cleanup()

	var err error
	output.CaptureOutput(func() {
		err = testutil.OutputAdapter(GetPort)(
			testutil.NewCommand("get", testutil.OutputAdapter(GetPort)),
			[]string{"port-123"},
		)
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth failed")
}

func TestListPorts_LoginError(t *testing.T) {
	cleanup := testutil.SetupLoginError(fmt.Errorf("auth failed"))
	defer cleanup()

	var err error
	output.CaptureOutput(func() {
		cmd := testutil.NewCommand("list", testutil.OutputAdapter(ListPorts))
		cmd.Flags().Int("location-id", 0, "")
		cmd.Flags().Int("port-speed", 0, "")
		cmd.Flags().String("port-name", "", "")
		cmd.Flags().Bool("include-inactive", false, "")
		err = testutil.OutputAdapter(ListPorts)(cmd, nil)
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth failed")
}

func TestExportPortConfig(t *testing.T) {
	port := &megaport.Port{
		Name:                  "My Port",
		ContractTermMonths:    12,
		PortSpeed:             10000,
		LocationID:            123,
		MarketplaceVisibility: true,
		DiversityZone:         "blue",
		CostCentre:            "IT Dept",
		UID:                   "port-should-not-appear",
		ProvisioningStatus:    "LIVE",
	}
	m := exportPortConfig(port)

	assert.Equal(t, "My Port", m["name"])
	assert.Equal(t, 12, m["term"])
	assert.Equal(t, 10000, m["portSpeed"])
	assert.Equal(t, 123, m["locationId"])
	assert.Equal(t, true, m["marketPlaceVisibility"])
	assert.Equal(t, "blue", m["diversityZone"])
	assert.Equal(t, "IT Dept", m["costCentre"])

	_, hasUID := m["productUid"]
	assert.False(t, hasUID, "export should not include productUid")
	_, hasStatus := m["provisioningStatus"]
	assert.False(t, hasStatus, "export should not include provisioningStatus")
}

func TestGetPort_Export(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	mockService := &MockPortService{
		GetPortResult: &megaport.Port{
			UID:                   "port-export-123",
			Name:                  "Export Port",
			ContractTermMonths:    12,
			PortSpeed:             10000,
			LocationID:            42,
			MarketplaceVisibility: true,
			ProvisioningStatus:    "LIVE",
		},
	}
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.PortService = mockService
		return client, nil
	})

	cmd := testutil.NewCommand("get", testutil.OutputAdapter(GetPort))
	cmd.Flags().Bool("export", false, "")
	require.NoError(t, cmd.Flags().Set("export", "true"))

	var err error
	capturedOutput := output.CaptureStdout(func() {
		err = testutil.OutputAdapter(GetPort)(cmd, []string{"port-export-123"})
	})

	assert.NoError(t, err)
	var parsed map[string]interface{}
	assert.NoError(t, json.Unmarshal([]byte(capturedOutput), &parsed), "export output must be valid JSON")
	assert.Equal(t, "Export Port", parsed["name"])
	assert.Equal(t, float64(12), parsed["term"])
	assert.Equal(t, float64(10000), parsed["portSpeed"])
	_, hasUID := parsed["productUid"]
	assert.False(t, hasUID, "export should not include productUid")
}

func TestMockPortServiceReset(t *testing.T) {
	m := &MockPortService{
		GetPortErr:      fmt.Errorf("test"),
		ListPortsErr:    fmt.Errorf("test"),
		BuyPortErr:      fmt.Errorf("test"),
		ForceNilGetPort: true,
	}
	m.Reset()
	assert.Nil(t, m.GetPortErr)
	assert.Nil(t, m.ListPortsErr)
	assert.Nil(t, m.BuyPortErr)
	assert.False(t, m.ForceNilGetPort)
}

func TestListPorts_TagFilter(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	origListPortsFunc := listPortsFunc
	origListTagsFunc := listPortResourceTagsFunc
	defer func() {
		listPortsFunc = origListPortsFunc
		listPortResourceTagsFunc = origListTagsFunc
	}()

	allPorts := []*megaport.Port{
		{UID: "port-1", Name: "PortAlpha", ProvisioningStatus: "LIVE"},
		{UID: "port-2", Name: "PortBeta", ProvisioningStatus: "LIVE"},
		{UID: "port-3", Name: "PortGamma", ProvisioningStatus: "LIVE"},
	}

	tagsByUID := map[string]map[string]string{
		"port-1": {"env": "prod", "team": "net"},
		"port-2": {"env": "staging"},
		"port-3": {},
	}

	listPortsFunc = func(ctx context.Context, client *megaport.Client) ([]*megaport.Port, error) {
		return allPorts, nil
	}

	tests := []struct {
		name            string
		tagFilters      []string
		tagFetchErrUID  string // UID that returns an error
		expectedOutputs []string
		notExpected     []string
	}{
		{
			name:            "no tag filter returns all",
			tagFilters:      nil,
			expectedOutputs: []string{"port-1", "port-2", "port-3"},
		},
		{
			name:            "exact match includes only matching resource",
			tagFilters:      []string{"env=prod"},
			expectedOutputs: []string{"port-1"},
			notExpected:     []string{"port-2", "port-3"},
		},
		{
			name:            "exact match excludes non-matching value",
			tagFilters:      []string{"env=staging"},
			expectedOutputs: []string{"port-2"},
			notExpected:     []string{"port-1", "port-3"},
		},
		{
			name:            "AND logic requires all filters to match",
			tagFilters:      []string{"env=prod", "team=net"},
			expectedOutputs: []string{"port-1"},
			notExpected:     []string{"port-2", "port-3"},
		},
		{
			name:            "key-exists match",
			tagFilters:      []string{"env"},
			expectedOutputs: []string{"port-1", "port-2"},
			notExpected:     []string{"port-3"},
		},
		{
			name:        "no match returns empty",
			tagFilters:  []string{"env=nonexistent"},
			notExpected: []string{"port-1", "port-2", "port-3"},
		},
		{
			name:            "tag fetch error excludes resource gracefully",
			tagFilters:      []string{"env=prod"},
			tagFetchErrUID:  "port-1",
			notExpected:     []string{"PortAlpha"}, // name absent from table; UID appears in warning
			expectedOutputs: []string{"Failed to fetch tags"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listPortResourceTagsFunc = func(ctx context.Context, client *megaport.Client, uid string) (map[string]string, error) {
				if tt.tagFetchErrUID != "" && uid == tt.tagFetchErrUID {
					return nil, fmt.Errorf("tag fetch error")
				}
				return tagsByUID[uid], nil
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				return &megaport.Client{PortService: &MockPortService{}}, nil
			})

			cmd := &cobra.Command{Use: "list"}
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().Int("port-speed", 0, "")
			cmd.Flags().String("port-name", "", "")
			cmd.Flags().Bool("include-inactive", false, "")
			cmd.Flags().Int("limit", 0, "")
			cmd.Flags().StringArray("tag", nil, "")

			for _, f := range tt.tagFilters {
				require.NoError(t, cmd.Flags().Set("tag", f))
			}

			var listErr error
			var capturedOutput, stderrOutput string
			stderrOutput = captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					listErr = ListPorts(cmd, nil, true, "table")
				})
			})
			capturedOutput += stderrOutput
			assert.NoError(t, listErr)

			for _, expected := range tt.expectedOutputs {
				assert.Contains(t, capturedOutput, expected)
			}
			for _, notExp := range tt.notExpected {
				assert.NotContains(t, capturedOutput, notExp)
			}
		})
	}
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

// TestBuyPort_NoResubmitOnPollRetryableError proves the order is submitted exactly
// once even when a retryable error (429) surfaces during the provisioning poll.
// The poll must run outside the order-submission retry, so a poll-phase 429 fails
// the command rather than re-placing the order and double-billing.
func TestBuyPort_NoResubmitOnPollRetryableError(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	originalBuyPortFunc := buyPortFunc
	originalGetPortFunc := getPortFunc
	originalMaxRetries := utils.MaxRetries
	originalPollInterval := utils.ProvisionPollInterval
	originalNoRetry := utils.NoRetry
	defer func() {
		cleanup()
		buyPortFunc = originalBuyPortFunc
		getPortFunc = originalGetPortFunc
		utils.MaxRetries = originalMaxRetries
		utils.ProvisionPollInterval = originalPollInterval
		utils.NoRetry = originalNoRetry
	}()
	// Retries are available; if the poll were wrapped in the order retry, the 429
	// would re-submit the order.
	utils.MaxRetries = 3
	utils.NoRetry = false
	utils.ProvisionPollInterval = time.Millisecond

	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	mockService := &MockPortService{}
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.PortService = mockService
		return client, nil
	})

	buyCalls := 0
	buyPortFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error) {
		buyCalls++
		return &megaport.BuyPortResponse{TechnicalServiceUIDs: []string{"port-uid-123"}}, nil
	}
	getPortFunc = func(ctx context.Context, client *megaport.Client, portUID string) (*megaport.Port, error) {
		return nil, &megaport.ErrorResponse{
			Response: &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Header:     http.Header{},
				Request:    &http.Request{URL: &url.URL{}},
			},
			Message: "too many requests",
		}
	}

	cmd := &cobra.Command{Use: "buy"}
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().Bool("no-wait", false, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().Int("term", 0, "")
	cmd.Flags().Int("port-speed", 0, "")
	cmd.Flags().Int("location-id", 0, "")
	cmd.Flags().Bool("marketplace-visibility", false, "")
	cmd.Flags().String("diversity-zone", "", "")
	cmd.Flags().Bool("cost-confirm", true, "")

	require.NoError(t, cmd.Flags().Set("name", "test-port"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("port-speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))
	require.NoError(t, cmd.Flags().Set("marketplace-visibility", "true"))
	// no-wait stays false: the provisioning poll must run.

	var err error
	output.CaptureOutput(func() {
		err = BuyPort(cmd, nil, true)
	})

	assert.Error(t, err, "a retryable error during the provisioning poll must surface as an error")
	assert.Equal(t, 1, buyCalls, "the order must be submitted exactly once; a poll-phase 429 must not re-submit it")
}

// TestBuyLAGPort_OrderAndProvisionFailures covers BuyLAGPort's order-submission
// error branch and the provisioning-poll branch where the port is not found.
func TestBuyLAGPort_OrderAndProvisionFailures(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	originalBuyPortFunc := buyPortFunc
	originalGetPortFunc := getPortFunc
	defer func() {
		cleanup()
		buyPortFunc = originalBuyPortFunc
		getPortFunc = originalGetPortFunc
	}()

	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	tests := []struct {
		name     string
		buyResp  *megaport.BuyPortResponse
		buyErr   error
		getNil   bool
		errMatch string
	}{
		{name: "order submission fails", buyErr: fmt.Errorf("order rejected"), errMatch: "order rejected"},
		{name: "LAG port not found during provisioning poll", buyResp: &megaport.BuyPortResponse{TechnicalServiceUIDs: []string{"lag-uid-123"}}, getNil: true, errMatch: "not found while waiting for provisioning"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.PortService = &MockPortService{}
				return client, nil
			})
			buyPortFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error) {
				return tt.buyResp, tt.buyErr
			}
			getPortFunc = func(ctx context.Context, client *megaport.Client, portUID string) (*megaport.Port, error) {
				if tt.getNil {
					return nil, nil
				}
				return &megaport.Port{UID: portUID, ProvisioningStatus: "LIVE"}, nil
			}

			cmd := &cobra.Command{Use: "buy-lag"}
			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().Bool("no-wait", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("term", 0, "")
			cmd.Flags().Int("port-speed", 0, "")
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().Bool("marketplace-visibility", false, "")
			cmd.Flags().String("diversity-zone", "", "")
			cmd.Flags().Bool("cost-confirm", true, "")
			cmd.Flags().Int("lag-count", 0, "")

			require.NoError(t, cmd.Flags().Set("name", "test-lag"))
			require.NoError(t, cmd.Flags().Set("term", "12"))
			require.NoError(t, cmd.Flags().Set("port-speed", "10000"))
			require.NoError(t, cmd.Flags().Set("location-id", "1"))
			require.NoError(t, cmd.Flags().Set("lag-count", "2"))

			var err error
			output.CaptureOutput(func() {
				err = BuyLAGPort(cmd, nil, true)
			})

			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tt.errMatch)
			}
		})
	}
}
