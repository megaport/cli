package ix

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

func TestListIXs(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	testIXs := []*megaport.IX{
		{
			ProductUID:         "ix-123",
			ProductName:        "ix-demo-01",
			NetworkServiceType: "Los Angeles IX",
			ASN:                65000,
			VLAN:               100,
			RateLimit:          1000,
			MACAddress:         "00:11:22:33:44:55",
			LocationID:         571,
			ProvisioningStatus: "LIVE",
		},
		{
			ProductUID:         "ix-456",
			ProductName:        "ix-demo-02",
			NetworkServiceType: "Sydney IX",
			ASN:                65001,
			VLAN:               200,
			RateLimit:          2000,
			MACAddress:         "AA:BB:CC:DD:EE:FF",
			LocationID:         558,
			ProvisioningStatus: "LIVE",
		},
		{
			ProductUID:         "ix-789",
			ProductName:        "production-ix",
			NetworkServiceType: "London IX",
			ASN:                65002,
			VLAN:               300,
			RateLimit:          5000,
			MACAddress:         "11:22:33:44:55:66",
			LocationID:         64,
			ProvisioningStatus: "LIVE",
		},
		{
			ProductUID:         "ix-decomm",
			ProductName:        "decommissioned-ix",
			NetworkServiceType: "Tokyo IX",
			ASN:                65003,
			VLAN:               400,
			RateLimit:          1000,
			MACAddress:         "FF:EE:DD:CC:BB:AA",
			LocationID:         571,
			ProvisioningStatus: "DECOMMISSIONED",
		},
		{
			ProductUID:         "ix-cancelled",
			ProductName:        "cancelled-ix",
			NetworkServiceType: "Paris IX",
			ASN:                65004,
			VLAN:               500,
			RateLimit:          1000,
			MACAddress:         "12:34:56:78:9A:BC",
			LocationID:         571,
			ProvisioningStatus: "CANCELLED",
		},
		{
			ProductUID:         "ix-decommissioning",
			ProductName:        "decommissioning-ix",
			NetworkServiceType: "Berlin IX",
			ASN:                65005,
			VLAN:               600,
			RateLimit:          1000,
			MACAddress:         "AB:CD:EF:01:23:45",
			LocationID:         571,
			ProvisioningStatus: "DECOMMISSIONING",
		},
	}

	tests := []struct {
		name          string
		flags         map[string]string
		setupMock     func(*MockIXService)
		expectedError string
		expectedIXs   []string
		unexpectedIXs []string
		outputFormat  string
	}{
		{
			name: "list all active IXs",
			setupMock: func(m *MockIXService) {
				m.listIXResponse = testIXs
			},
			expectedIXs:   []string{"ix-demo-01", "ix-demo-02", "production-ix"},
			unexpectedIXs: []string{"decommissioned-ix", "cancelled-ix", "decommissioning-ix"},
			outputFormat:  "table",
		},
		{
			name: "excludes DECOMMISSIONED status",
			setupMock: func(m *MockIXService) {
				m.listIXResponse = testIXs
			},
			expectedIXs:   []string{"ix-demo-01", "ix-demo-02", "production-ix"},
			unexpectedIXs: []string{"decommissioned-ix"},
			outputFormat:  "table",
		},
		{
			name: "excludes CANCELLED status",
			setupMock: func(m *MockIXService) {
				m.listIXResponse = testIXs
			},
			expectedIXs:   []string{"ix-demo-01", "ix-demo-02", "production-ix"},
			unexpectedIXs: []string{"cancelled-ix"},
			outputFormat:  "table",
		},
		{
			name: "excludes DECOMMISSIONING status",
			setupMock: func(m *MockIXService) {
				m.listIXResponse = testIXs
			},
			expectedIXs:   []string{"ix-demo-01", "ix-demo-02", "production-ix"},
			unexpectedIXs: []string{"decommissioning-ix"},
			outputFormat:  "table",
		},
		{
			name: "filter by name",
			flags: map[string]string{
				"name": "ix-demo",
			},
			setupMock: func(m *MockIXService) {
				m.listIXResponse = testIXs
			},
			expectedIXs:   []string{"ix-demo-01", "ix-demo-02"},
			unexpectedIXs: []string{"production-ix", "decommissioned-ix"},
			outputFormat:  "table",
		},
		{
			name: "filter by ASN",
			flags: map[string]string{
				"asn": "65000",
			},
			setupMock: func(m *MockIXService) {
				m.listIXResponse = testIXs
			},
			expectedIXs:   []string{"ix-demo-01"},
			unexpectedIXs: []string{"ix-demo-02", "production-ix"},
			outputFormat:  "table",
		},
		{
			name: "filter by VLAN",
			flags: map[string]string{
				"vlan": "200",
			},
			setupMock: func(m *MockIXService) {
				m.listIXResponse = testIXs
			},
			expectedIXs:   []string{"ix-demo-02"},
			unexpectedIXs: []string{"ix-demo-01", "production-ix"},
			outputFormat:  "table",
		},
		{
			name: "filter by network service type",
			flags: map[string]string{
				"network-service-type": "Los Angeles",
			},
			setupMock: func(m *MockIXService) {
				m.listIXResponse = testIXs
			},
			expectedIXs:   []string{"ix-demo-01"},
			unexpectedIXs: []string{"ix-demo-02", "production-ix"},
			outputFormat:  "table",
		},
		{
			name: "filter by location ID",
			flags: map[string]string{
				"location-id": "571",
			},
			setupMock: func(m *MockIXService) {
				m.listIXResponse = testIXs
			},
			expectedIXs:   []string{"ix-demo-01"},
			unexpectedIXs: []string{"ix-demo-02", "production-ix"},
			outputFormat:  "table",
		},
		{
			name: "filter by rate limit",
			flags: map[string]string{
				"rate-limit": "5000",
			},
			setupMock: func(m *MockIXService) {
				m.listIXResponse = testIXs
			},
			expectedIXs:   []string{"production-ix"},
			unexpectedIXs: []string{"ix-demo-01", "ix-demo-02"},
			outputFormat:  "table",
		},
		{
			name: "combined filters",
			flags: map[string]string{
				"name":        "demo",
				"location-id": "571",
			},
			setupMock: func(m *MockIXService) {
				m.listIXResponse = testIXs
			},
			expectedIXs:   []string{"ix-demo-01"},
			unexpectedIXs: []string{"ix-demo-02", "production-ix"},
			outputFormat:  "table",
		},
		{
			name: "include inactive",
			flags: map[string]string{
				"include-inactive": "true",
			},
			setupMock: func(m *MockIXService) {
				m.listIXResponse = testIXs
			},
			expectedIXs:   []string{"ix-demo-01", "ix-demo-02", "production-ix", "decommissioned-ix", "cancelled-ix", "decommissioning-ix"},
			unexpectedIXs: []string{},
			outputFormat:  "table",
		},
		{
			name: "include inactive with filter",
			flags: map[string]string{
				"include-inactive": "true",
				"location-id":      "571",
			},
			setupMock: func(m *MockIXService) {
				m.listIXResponse = testIXs
			},
			expectedIXs:   []string{"ix-demo-01", "decommissioned-ix", "cancelled-ix", "decommissioning-ix"},
			unexpectedIXs: []string{"ix-demo-02", "production-ix"},
			outputFormat:  "table",
		},
		{
			name: "empty API result",
			setupMock: func(m *MockIXService) {
				m.listIXResponse = []*megaport.IX{}
			},
			expectedIXs:  []string{},
			outputFormat: "table",
		},
		{
			name: "nil API result",
			setupMock: func(m *MockIXService) {
				m.listIXResponse = nil
			},
			expectedIXs:  []string{},
			outputFormat: "table",
		},
		{
			name: "JSON output format",
			flags: map[string]string{
				"name": "ix-demo-01",
			},
			setupMock: func(m *MockIXService) {
				m.listIXResponse = testIXs
			},
			expectedIXs:   []string{"ix-demo-01"},
			unexpectedIXs: []string{"ix-demo-02", "production-ix"},
			outputFormat:  "json",
		},
		{
			name: "CSV output format",
			flags: map[string]string{
				"name": "ix-demo-01",
			},
			setupMock: func(m *MockIXService) {
				m.listIXResponse = testIXs
			},
			expectedIXs:   []string{"ix-demo-01"},
			unexpectedIXs: []string{"ix-demo-02", "production-ix"},
			outputFormat:  "csv",
		},
		{
			name: "API error",
			setupMock: func(m *MockIXService) {
				m.listIXErr = fmt.Errorf("API error: service unavailable")
			},
			expectedError: "API error: service unavailable",
			outputFormat:  "table",
		},
		{
			name: "login error",
			setupMock: func(m *MockIXService) {
				// Will be handled by overriding LoginFunc to return error
			},
			expectedError: "login failed",
			outputFormat:  "table",
		},
		{
			name: "limit results",
			flags: map[string]string{
				"limit": "2",
			},
			setupMock: func(m *MockIXService) {
				m.listIXResponse = testIXs
			},
			expectedIXs:   []string{"ix-demo-01", "ix-demo-02"},
			unexpectedIXs: []string{"production-ix"},
			outputFormat:  "table",
		},
		{
			name: "negative limit returns error",
			flags: map[string]string{
				"limit": "-1",
			},
			setupMock: func(m *MockIXService) {
				m.listIXResponse = testIXs
			},
			expectedError: "--limit must be a non-negative integer",
			outputFormat:  "table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockIXService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			if tt.name == "login error" {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					return nil, fmt.Errorf("login failed")
				})
			} else {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.IXService = mockService
					return client, nil
				})
			}

			cmd := &cobra.Command{
				Use: "list",
				RunE: func(cmd *cobra.Command, args []string) error {
					return ListIXs(cmd, args, true, tt.outputFormat)
				},
			}

			cmd.Flags().String("name", "", "Filter IXs by name")
			cmd.Flags().Int("asn", 0, "Filter IXs by ASN")
			cmd.Flags().Int("vlan", 0, "Filter IXs by VLAN")
			cmd.Flags().String("network-service-type", "", "Filter IXs by network service type")
			cmd.Flags().Int("location-id", 0, "Filter IXs by location ID")
			cmd.Flags().Int("rate-limit", 0, "Filter IXs by rate limit")
			cmd.Flags().Bool("include-inactive", false, "Include inactive IXs")
			cmd.Flags().Int("limit", 0, "Maximum number of results to display")

			testutil.SetFlags(t, cmd, tt.flags)

			var err error
			var capturedStderr string
			capturedOutput := output.CaptureOutput(func() {
				capturedStderr = captureStderr(t, func() {
					err = cmd.RunE(cmd, []string{})
				})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)

				for _, expectedIX := range tt.expectedIXs {
					assert.Contains(t, capturedOutput, expectedIX,
						"Expected IX '%s' should be in output", expectedIX)
				}

				for _, unexpectedIX := range tt.unexpectedIXs {
					assert.NotContains(t, capturedOutput, unexpectedIX,
						"Unexpected IX '%s' should NOT be in output", unexpectedIX)
				}

				switch tt.outputFormat {
				case "json":
					if len(tt.expectedIXs) > 0 {
						assert.Contains(t, capturedOutput, "\"uid\":")
						assert.Contains(t, capturedOutput, "\"name\":")
					}
				case "table":
					if len(tt.expectedIXs) > 0 {
						assert.Contains(t, capturedOutput, "UID")
						assert.Contains(t, capturedOutput, "NAME")
					}
				}

				if len(tt.expectedIXs) == 0 && tt.expectedError == "" {
					assert.Contains(t, capturedStderr, "No IX connections found. Create one with 'megaport ix buy'.")
				}
			}

			if mockService.CapturedListIXsRequest != nil {
				if tt.flags["include-inactive"] == "true" {
					assert.True(t, mockService.CapturedListIXsRequest.IncludeInactive)
				} else {
					assert.False(t, mockService.CapturedListIXsRequest.IncludeInactive)
				}
			}
		})
	}
}

func TestBuyIX(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	originalBuyIXFunc := buyIXFunc
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer func() {
		utils.SetResourcePrompt(originalPrompt)
		cleanup()
		buyIXFunc = originalBuyIXFunc
	}()
	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	tests := []struct {
		name           string
		args           []string
		interactive    bool
		prompts        []string
		flags          map[string]string
		setupMock      func(*MockIXService)
		expectedError  string
		expectedOutput string
	}{
		{
			name: "flag mode success",
			flags: map[string]string{
				"product-uid":          "port-uid-123",
				"name":                 "Flag IX",
				"network-service-type": "Los Angeles IX",
				"asn":                  "65000",
				"mac-address":          "00:11:22:33:44:55",
				"rate-limit":           "1000",
				"vlan":                 "100",
			},
			setupMock: func(m *MockIXService) {
				m.buyIXResponse = &megaport.BuyIXResponse{
					TechnicalServiceUID: "ix-456-def",
				}
			},
			expectedOutput: "IX created",
		},
		{
			name:        "interactive mode success",
			interactive: true,
			prompts: []string{
				"port-uid-123",
				"Interactive IX",
				"Los Angeles IX",
				"65000",
				"00:11:22:33:44:55",
				"1000",
				"100",
				"",
			},
			setupMock: func(m *MockIXService) {
				m.buyIXResponse = &megaport.BuyIXResponse{
					TechnicalServiceUID: "ix-123-abc",
				}
			},
			expectedOutput: "IX created",
		},
		{
			name: "missing required fields in flag mode",
			flags: map[string]string{
				"name": "Test IX",
			},
			setupMock:     func(m *MockIXService) {},
			expectedError: "Invalid ASN",
		},
		{
			name: "API error",
			flags: map[string]string{
				"product-uid":          "port-uid-123",
				"name":                 "Test IX",
				"network-service-type": "Los Angeles IX",
				"asn":                  "65000",
				"mac-address":          "00:11:22:33:44:55",
				"rate-limit":           "1000",
				"vlan":                 "100",
			},
			setupMock: func(m *MockIXService) {
				m.buyIXError = fmt.Errorf("API error: service unavailable")
			},
			expectedError: "API error: service unavailable",
		},
		{
			name: "validation error",
			flags: map[string]string{
				"product-uid":          "port-uid-123",
				"name":                 "Test IX",
				"network-service-type": "Los Angeles IX",
				"asn":                  "65000",
				"mac-address":          "00:11:22:33:44:55",
				"rate-limit":           "1000",
				"vlan":                 "100",
			},
			setupMock: func(m *MockIXService) {
				m.validateIXOrderError = fmt.Errorf("validation error: invalid configuration")
			},
			expectedError: "validation error",
		},
		{
			name:          "no input provided",
			expectedError: "no input provided",
		},
		{
			name:        "interactive combined with JSON and flags is a usage error",
			interactive: true,
			flags: map[string]string{
				"json":                 `{"productUid":"port-uid-123","productName":"JSON IX","networkServiceType":"Los Angeles IX","asn":65000,"macAddress":"00:11:22:33:44:55","rateLimit":1000,"vlan":100}`,
				"product-uid":          "port-uid-123",
				"name":                 "JSON IX",
				"network-service-type": "Los Angeles IX",
			},
			setupMock:     func(m *MockIXService) {},
			expectedError: "cannot be combined with",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.prompts) > 0 {
				promptIndex := 0
				utils.SetResourcePrompt(func(_, msg string, _ bool) (string, error) {
					if promptIndex < len(tt.prompts) {
						response := tt.prompts[promptIndex]
						promptIndex++
						return response, nil
					}
					return "", fmt.Errorf("unexpected prompt call")
				})
			}

			mockService := &MockIXService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.IXService = mockService
				return client, nil
			})

			cmd := &cobra.Command{
				Use:  "buy",
				RunE: testutil.NoColorAdapter(BuyIX),
			}

			cmd.Flags().BoolP("interactive", "i", false, "Use interactive mode with prompts")
			cmd.Flags().String("product-uid", "", "Port UID")
			cmd.Flags().String("name", "", "IX name")
			cmd.Flags().String("network-service-type", "", "Network service type")
			cmd.Flags().Int("asn", 0, "ASN")
			cmd.Flags().String("mac-address", "", "MAC address")
			cmd.Flags().Int("rate-limit", 0, "Rate limit")
			cmd.Flags().Int("vlan", 0, "VLAN")
			cmd.Flags().Bool("shutdown", false, "Shutdown")
			cmd.Flags().String("promo-code", "", "Promo code")
			cmd.Flags().String("json", "", "JSON string containing IX configuration")
			cmd.Flags().String("json-file", "", "Path to JSON file containing IX configuration")

			if tt.interactive {
				if err := cmd.Flags().Set("interactive", "true"); err != nil {
					t.Fatalf("Failed to set interactive flag: %v", err)
				}
			}

			testutil.SetFlags(t, cmd, tt.flags)

			var err error
			var capturedStderr string
			capturedOutput := output.CaptureOutput(func() {
				capturedStderr = captureStderr(t, func() {
					err = cmd.RunE(cmd, tt.args)
				})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, capturedOutput+capturedStderr, tt.expectedOutput)

				if mockService.capturedBuyIXRequest != nil {
					req := mockService.capturedBuyIXRequest

					if tt.flags != nil {
						assert.Equal(t, tt.flags["product-uid"], req.ProductUID)
						assert.Equal(t, tt.flags["name"], req.Name)
						assert.Equal(t, tt.flags["network-service-type"], req.NetworkServiceType)
					}
				}
			}
		})
	}
}

func TestBuyIX_NilResponse(t *testing.T) {
	originalBuyIXFunc := buyIXFunc
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer func() {
		cleanup()
		buyIXFunc = originalBuyIXFunc
	}()
	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.IXService = &MockIXService{}
		return client, nil
	})

	buyIXFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyIXRequest) (*megaport.BuyIXResponse, error) {
		return nil, nil
	}

	cmd := &cobra.Command{
		Use:  "buy",
		RunE: testutil.NoColorAdapter(BuyIX),
	}
	cmd.Flags().BoolP("interactive", "i", false, "")
	cmd.Flags().Bool("no-wait", false, "")
	cmd.Flags().String("product-uid", "", "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().String("network-service-type", "", "")
	cmd.Flags().Int("asn", 0, "")
	cmd.Flags().String("mac-address", "", "")
	cmd.Flags().Int("rate-limit", 0, "")
	cmd.Flags().Int("vlan", 0, "")
	cmd.Flags().Bool("shutdown", false, "")
	cmd.Flags().String("promo-code", "", "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")

	testutil.SetFlags(t, cmd, map[string]string{
		"product-uid":          "port-uid-123",
		"name":                 "Test IX",
		"network-service-type": "Los Angeles IX",
		"asn":                  "65000",
		"mac-address":          "00:11:22:33:44:55",
		"rate-limit":           "1000",
		"vlan":                 "100",
	})

	var err error
	require.NotPanics(t, func() {
		output.CaptureOutput(func() {
			err = cmd.RunE(cmd, nil)
		})
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty response from API")
}

func TestBuyIX_NoWaitFlag(t *testing.T) {
	originalBuyIXFunc := buyIXFunc
	originalGetIXFunc := getIXFunc
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer func() {
		cleanup()
		buyIXFunc = originalBuyIXFunc
		getIXFunc = originalGetIXFunc
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
			mockService := &MockIXService{}
			mockService.buyIXResponse = &megaport.BuyIXResponse{
				TechnicalServiceUID: "ix-uid-123",
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.IXService = mockService
				return client, nil
			})

			var capturedReq *megaport.BuyIXRequest
			buyIXFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyIXRequest) (*megaport.BuyIXResponse, error) {
				capturedReq = req
				return &megaport.BuyIXResponse{
					TechnicalServiceUID: "ix-uid-123",
				}, nil
			}
			getIXCalls := 0
			getIXFunc = func(ctx context.Context, client *megaport.Client, ixUID string) (*megaport.IX, error) {
				getIXCalls++
				return &megaport.IX{ProvisioningStatus: "LIVE"}, nil
			}

			cmd := &cobra.Command{
				Use:  "buy",
				RunE: testutil.NoColorAdapter(BuyIX),
			}
			cmd.Flags().BoolP("interactive", "i", false, "")
			cmd.Flags().Bool("no-wait", false, "")
			cmd.Flags().String("product-uid", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().String("network-service-type", "", "")
			cmd.Flags().Int("asn", 0, "")
			cmd.Flags().String("mac-address", "", "")
			cmd.Flags().Int("rate-limit", 0, "")
			cmd.Flags().Int("vlan", 0, "")
			cmd.Flags().Bool("shutdown", false, "")
			cmd.Flags().String("promo-code", "", "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")

			testutil.SetFlags(t, cmd, map[string]string{
				"product-uid":          "port-uid-123",
				"name":                 "Test IX",
				"network-service-type": "Los Angeles IX",
				"asn":                  "65000",
				"mac-address":          "00:11:22:33:44:55",
				"rate-limit":           "1000",
				"vlan":                 "100",
			})
			if tt.noWait {
				assert.NoError(t, cmd.Flags().Set("no-wait", "true"))
			}

			var err error
			output.CaptureOutput(func() {
				err = cmd.RunE(cmd, nil)
			})

			assert.NoError(t, err)
			assert.NotNil(t, capturedReq)
			// The SDK must never poll; provisioning is awaited outside the order retry.
			assert.False(t, capturedReq.WaitForProvision)
			if tt.expectPoll {
				assert.Positive(t, getIXCalls, "expected a provisioning poll when not using --no-wait")
			} else {
				assert.Zero(t, getIXCalls, "no provisioning poll expected with --no-wait")
			}
		})
	}
}

func TestGetIXStatus(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name           string
		ixUID          string
		setupMock      func(*MockIXService)
		expectedError  string
		expectedOutput string
		outputFormat   string
	}{
		{
			name:  "successful status retrieval - table format",
			ixUID: "ix-123abc",
			setupMock: func(m *MockIXService) {
				m.getIXResponse = &megaport.IX{
					ProductUID:         "ix-123abc",
					ProductName:        "Test IX",
					ProvisioningStatus: "CONFIGURED",
					NetworkServiceType: "Los Angeles IX",
				}
			},
			expectedOutput: "ix-123abc",
			outputFormat:   "table",
		},
		{
			name:  "successful status retrieval - json format",
			ixUID: "ix-123abc",
			setupMock: func(m *MockIXService) {
				m.getIXResponse = &megaport.IX{
					ProductUID:         "ix-123abc",
					ProductName:        "Test IX",
					ProvisioningStatus: "LIVE",
					NetworkServiceType: "Los Angeles IX",
				}
			},
			expectedOutput: "ix-123abc",
			outputFormat:   "json",
		},
		{
			name:  "API error",
			ixUID: "ix-error",
			setupMock: func(m *MockIXService) {
				m.getIXError = fmt.Errorf("API error")
			},
			expectedError: "API error",
			outputFormat:  "table",
		},
		{
			name:  "nil IX returned without error",
			ixUID: "ix-nil",
			setupMock: func(m *MockIXService) {
				m.forceNilGetIX = true
			},
			expectedError: "no IX found",
			outputFormat:  "table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockIXService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.IXService = mockService
				return client, nil
			})

			cmd := &cobra.Command{
				Use: "status [ixUID]",
			}

			var err error
			capturedOutput := output.CaptureOutput(func() {
				err = GetIXStatus(cmd, []string{tt.ixUID}, true, tt.outputFormat)
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

func TestDeleteIX(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer func() {
		cleanup()
		utils.SetResourcePrompt(originalPrompt)
	}()

	tests := []struct {
		name              string
		ixUID             string
		force             bool
		later             bool
		promptResponse    string
		setupMock         func(*MockIXService)
		expectedError     string
		expectedOutput    string
		expectDeleted     bool
		expectedDeleteNow *bool
	}{
		{
			name:              "force delete defaults to immediate",
			ixUID:             "ix-to-delete",
			force:             true,
			setupMock:         func(m *MockIXService) {},
			expectedOutput:    "IX deleted",
			expectDeleted:     true,
			expectedDeleteNow: boolPtr(true),
		},
		{
			name:              "force with --later defers cancellation",
			ixUID:             "ix-to-delete-later",
			force:             true,
			later:             true,
			setupMock:         func(m *MockIXService) {},
			expectedOutput:    "IX deleted",
			expectDeleted:     true,
			expectedDeleteNow: boolPtr(false),
		},
		{
			name:           "confirm deletion with prompt",
			ixUID:          "ix-to-delete",
			force:          false,
			promptResponse: "y",
			setupMock:      func(m *MockIXService) {},
			expectedOutput: "IX deleted",
			expectDeleted:  true,
		},
		{
			name:           "cancel deletion",
			ixUID:          "ix-keep",
			force:          false,
			promptResponse: "n",
			setupMock:      func(m *MockIXService) {},
			expectedError:  "cancelled by user",
			expectDeleted:  false,
		},
		{
			name:  "deletion error",
			ixUID: "ix-error",
			force: true,
			setupMock: func(m *MockIXService) {
				m.deleteIXError = fmt.Errorf("failed to delete IX")
			},
			expectedError: "failed to delete IX",
			expectDeleted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockIXService{}
			tt.setupMock(mockService)

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.IXService = mockService
				return client, nil
			})

			originalConfirmPrompt := utils.GetConfirmPrompt()
			defer func() { utils.SetConfirmPrompt(originalConfirmPrompt) }()
			utils.SetConfirmPrompt(func(_ string, _ bool) bool {
				return tt.promptResponse == "y"
			})

			cmd := &cobra.Command{
				Use: "delete [ixUID]",
				RunE: func(cmd *cobra.Command, args []string) error {
					return DeleteIX(cmd, args, false)
				},
			}
			cmd.Flags().BoolP("force", "f", false, "Force deletion without confirmation")
			cmd.Flags().Bool("later", false, "Defer deletion to end of billing cycle")
			err := cmd.Flags().Set("force", fmt.Sprintf("%v", tt.force))
			if err != nil {
				t.Fatalf("Failed to set force flag: %v", err)
			}
			err = cmd.Flags().Set("later", fmt.Sprintf("%v", tt.later))
			if err != nil {
				t.Fatalf("Failed to set later flag: %v", err)
			}

			var capturedStderr string
			capturedOutput := output.CaptureOutput(func() {
				capturedStderr = captureStderr(t, func() {
					err = cmd.RunE(cmd, []string{tt.ixUID})
				})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, capturedOutput+capturedStderr, tt.expectedOutput)

				if tt.expectDeleted {
					assert.Equal(t, tt.ixUID, mockService.capturedDeleteIXUID)
				}
				if tt.expectedDeleteNow != nil {
					if assert.NotNil(t, mockService.capturedDeleteIXReq, "expected DeleteIX to be called") {
						assert.Equal(t, *tt.expectedDeleteNow, mockService.capturedDeleteIXReq.DeleteNow,
							"DeleteNow value does not match expectation")
					}
				}
			}
		})
	}
}

func boolPtr(b bool) *bool { return &b }

func TestGetIX(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name          string
		ixUID         string
		format        string
		setupMock     func(*MockIXService)
		expectedError string
		expectedOut   []string
	}{
		{
			name:   "get IX success table format",
			ixUID:  "ix-123",
			format: "table",
			setupMock: func(m *MockIXService) {
				m.getIXResponse = &megaport.IX{
					ProductUID:         "ix-123",
					ProductName:        "Test IX",
					NetworkServiceType: "Los Angeles IX",
					ASN:                65000,
					RateLimit:          1000,
					VLAN:               100,
					MACAddress:         "00:11:22:33:44:55",
					ProvisioningStatus: "LIVE",
				}
			},
			expectedOut: []string{"ix-123", "Test IX", "Los Angeles IX", "LIVE"},
		},
		{
			name:   "get IX success json format",
			ixUID:  "ix-123",
			format: "json",
			setupMock: func(m *MockIXService) {
				m.getIXResponse = &megaport.IX{
					ProductUID:         "ix-123",
					ProductName:        "Test IX",
					NetworkServiceType: "Los Angeles IX",
					ASN:                65000,
					RateLimit:          1000,
					VLAN:               100,
					MACAddress:         "00:11:22:33:44:55",
					ProvisioningStatus: "LIVE",
				}
			},
			expectedOut: []string{`"uid": "ix-123"`, `"name": "Test IX"`, `"network_service_type": "Los Angeles IX"`},
		},
		{
			name:   "get IX success csv format",
			ixUID:  "ix-123",
			format: "csv",
			setupMock: func(m *MockIXService) {
				m.getIXResponse = &megaport.IX{
					ProductUID:         "ix-123",
					ProductName:        "Test IX",
					NetworkServiceType: "Los Angeles IX",
					ASN:                65000,
					RateLimit:          1000,
					VLAN:               100,
					MACAddress:         "00:11:22:33:44:55",
					ProvisioningStatus: "LIVE",
				}
			},
			expectedOut: []string{"ix-123", "Test IX", "Los Angeles IX"},
		},
		{
			name:   "get IX nil response",
			ixUID:  "ix-nil",
			format: "table",
			setupMock: func(m *MockIXService) {
				m.forceNilGetIX = true
			},
			expectedError: "no IX found with UID: ix-nil",
		},
		{
			name:   "get IX API error",
			ixUID:  "ix-invalid",
			format: "table",
			setupMock: func(m *MockIXService) {
				m.getIXError = fmt.Errorf("IX not found")
			},
			expectedError: "IX not found",
		},
		{
			name:   "get IX login error",
			ixUID:  "ix-123",
			format: "table",
			setupMock: func(m *MockIXService) {
				// login error handled separately
			},
			expectedError: "login failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockIXService{}
			tt.setupMock(mockService)

			if tt.name == "get IX login error" {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					return nil, fmt.Errorf("login failed")
				})
			} else {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.IXService = mockService
					return client, nil
				})
			}

			var err error
			cmd := &cobra.Command{
				Use: "get [ixUID]",
				RunE: func(cmd *cobra.Command, args []string) error {
					return GetIX(cmd, args, false, tt.format)
				},
			}

			cmd.Flags().StringP("output", "o", "table", "Output format")
			_ = cmd.Flags().Set("output", tt.format)

			capturedOutput := output.CaptureOutput(func() {
				err = cmd.RunE(cmd, []string{tt.ixUID})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				for _, expected := range tt.expectedOut {
					assert.Contains(t, capturedOutput, expected)
				}
			}
		})
	}
}

func TestUpdateIX(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	originalPasswordPrompt := utils.GetPasswordPrompt()
	originalUpdateIXFunc := updateIXFunc
	originalGetIXFunc := getIXFunc
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer func() {
		utils.SetResourcePrompt(originalPrompt)
		utils.SetPasswordPrompt(originalPasswordPrompt)
		cleanup()
		updateIXFunc = originalUpdateIXFunc
		getIXFunc = originalGetIXFunc
	}()

	tests := []struct {
		name           string
		ixUID          string
		interactive    bool
		prompts        []string
		flags          map[string]string
		jsonInput      string
		setupMock      func(*MockIXService)
		expectedError  string
		expectedOutput string
	}{
		{
			name:  "flag mode - update name",
			ixUID: "ix-123",
			flags: map[string]string{
				"name": "Updated IX",
			},
			setupMock: func(m *MockIXService) {
				m.getIXResponse = &megaport.IX{
					ProductUID:         "ix-123",
					ProductName:        "Original IX",
					ProvisioningStatus: "LIVE",
				}
				m.updateIXResponse = &megaport.IX{
					ProductUID:         "ix-123",
					ProductName:        "Updated IX",
					ProvisioningStatus: "LIVE",
				}
			},
			expectedOutput: "IX updated",
		},
		{
			name:  "flag mode - update rate-limit",
			ixUID: "ix-123",
			flags: map[string]string{
				"rate-limit": "2000",
			},
			setupMock: func(m *MockIXService) {
				m.getIXResponse = &megaport.IX{
					ProductUID:         "ix-123",
					ProductName:        "Test IX",
					RateLimit:          1000,
					ProvisioningStatus: "LIVE",
				}
				m.updateIXResponse = &megaport.IX{
					ProductUID:         "ix-123",
					ProductName:        "Test IX",
					RateLimit:          2000,
					ProvisioningStatus: "LIVE",
				}
			},
			expectedOutput: "IX updated",
		},
		{
			name:  "flag mode - update multiple fields",
			ixUID: "ix-123",
			flags: map[string]string{
				"name":        "Updated IX",
				"rate-limit":  "2000",
				"vlan":        "200",
				"mac-address": "AA:BB:CC:DD:EE:FF",
				"asn":         "65001",
			},
			setupMock: func(m *MockIXService) {
				m.getIXResponse = &megaport.IX{
					ProductUID:         "ix-123",
					ProductName:        "Original IX",
					RateLimit:          1000,
					VLAN:               100,
					MACAddress:         "00:11:22:33:44:55",
					ASN:                65000,
					ProvisioningStatus: "LIVE",
				}
				m.updateIXResponse = &megaport.IX{
					ProductUID:         "ix-123",
					ProductName:        "Updated IX",
					RateLimit:          2000,
					VLAN:               200,
					MACAddress:         "AA:BB:CC:DD:EE:FF",
					ASN:                65001,
					ProvisioningStatus: "LIVE",
				}
			},
			expectedOutput: "IX updated",
		},
		{
			name:      "JSON string mode",
			ixUID:     "ix-123",
			jsonInput: `{"name":"JSON Updated IX","rateLimit":3000}`,
			setupMock: func(m *MockIXService) {
				m.getIXResponse = &megaport.IX{
					ProductUID:         "ix-123",
					ProductName:        "Original IX",
					RateLimit:          1000,
					ProvisioningStatus: "LIVE",
				}
				m.updateIXResponse = &megaport.IX{
					ProductUID:         "ix-123",
					ProductName:        "JSON Updated IX",
					RateLimit:          3000,
					ProvisioningStatus: "LIVE",
				}
			},
			expectedOutput: "IX updated",
		},
		{
			name:        "interactive mode",
			ixUID:       "ix-123",
			interactive: true,
			prompts: []string{
				"Updated IX", // name
				"",           // rate-limit (skip)
				"",           // cost-centre (skip)
				"",           // vlan (skip)
				"",           // mac-address (skip)
				"",           // asn (skip)
				"",           // password (skip)
				"",           // public-graph (skip)
				"",           // reverse-dns (skip)
				"",           // a-end-product-uid (skip)
				"",           // shutdown (skip)
			},
			setupMock: func(m *MockIXService) {
				m.getIXResponse = &megaport.IX{
					ProductUID:         "ix-123",
					ProductName:        "Original IX",
					ProvisioningStatus: "LIVE",
				}
				m.updateIXResponse = &megaport.IX{
					ProductUID:         "ix-123",
					ProductName:        "Updated IX",
					ProvisioningStatus: "LIVE",
				}
			},
			expectedOutput: "IX updated",
		},
		{
			name:          "no fields provided",
			ixUID:         "ix-123",
			setupMock:     func(m *MockIXService) {},
			expectedError: "at least one field must be updated",
		},
		{
			name:        "interactive combined with flags is rejected",
			ixUID:       "ix-123",
			interactive: true,
			flags: map[string]string{
				"name": "Updated IX",
			},
			setupMock:     func(m *MockIXService) {},
			expectedError: "cannot be combined with",
		},
		{
			name:          "interactive combined with JSON is rejected",
			ixUID:         "ix-123",
			interactive:   true,
			jsonInput:     `{"name":"JSON Updated IX"}`,
			setupMock:     func(m *MockIXService) {},
			expectedError: "cannot be combined with",
		},
		{
			name:  "API error during update",
			ixUID: "ix-123",
			flags: map[string]string{
				"name": "Updated IX",
			},
			setupMock: func(m *MockIXService) {
				m.getIXResponse = &megaport.IX{
					ProductUID:         "ix-123",
					ProductName:        "Original IX",
					ProvisioningStatus: "LIVE",
				}
				m.updateIXError = fmt.Errorf("API error: update failed")
			},
			expectedError: "API error: update failed",
		},
		{
			name:  "failed to get original IX",
			ixUID: "ix-123",
			flags: map[string]string{
				"name": "Updated IX",
			},
			setupMock: func(m *MockIXService) {
				m.getIXError = fmt.Errorf("IX not found")
			},
			expectedError: "IX not found",
		},
		{
			name:  "login error",
			ixUID: "ix-123",
			flags: map[string]string{
				"name": "Updated IX",
			},
			setupMock:     func(m *MockIXService) {},
			expectedError: "login failed",
		},
		{
			name:          "invalid JSON input",
			ixUID:         "ix-123",
			jsonInput:     `{invalid json`,
			setupMock:     func(m *MockIXService) {},
			expectedError: "failed to parse JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.prompts) > 0 {
				promptIndex := 0
				nextPrompt := func() (string, error) {
					if promptIndex < len(tt.prompts) {
						response := tt.prompts[promptIndex]
						promptIndex++
						return response, nil
					}
					return "", fmt.Errorf("unexpected prompt call")
				}
				utils.SetResourcePrompt(func(_, _ string, _ bool) (string, error) {
					return nextPrompt()
				})
				utils.SetPasswordPrompt(func(_ string, _ bool) (string, error) {
					return nextPrompt()
				})
			}

			mockService := &MockIXService{}
			tt.setupMock(mockService)

			if tt.name == "login error" {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					return nil, fmt.Errorf("login failed")
				})
			} else {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.IXService = mockService
					return client, nil
				})
			}

			cmd := &cobra.Command{
				Use:  "update",
				RunE: testutil.NoColorAdapter(UpdateIX),
			}

			cmd.Flags().BoolP("interactive", "i", false, "Use interactive mode")
			cmd.Flags().String("name", "", "IX name")
			cmd.Flags().Int("rate-limit", 0, "Rate limit")
			cmd.Flags().String("cost-centre", "", "Cost centre")
			cmd.Flags().Int("vlan", 0, "VLAN")
			cmd.Flags().String("mac-address", "", "MAC address")
			cmd.Flags().Int("asn", 0, "ASN")
			cmd.Flags().String("password", "", "Password")
			cmd.Flags().Bool("public-graph", false, "Public graph")
			cmd.Flags().String("reverse-dns", "", "Reverse DNS")
			cmd.Flags().String("a-end-product-uid", "", "A-End product UID")
			cmd.Flags().Bool("shutdown", false, "Shutdown")
			cmd.Flags().String("json", "", "JSON string")
			cmd.Flags().String("json-file", "", "JSON file")

			if tt.interactive {
				_ = cmd.Flags().Set("interactive", "true")
			}

			if tt.jsonInput != "" {
				_ = cmd.Flags().Set("json", tt.jsonInput)
			}

			testutil.SetFlags(t, cmd, tt.flags)

			var args []string
			if tt.ixUID != "" {
				args = []string{tt.ixUID}
			}

			var err error
			var capturedStderr string
			capturedOutput := output.CaptureOutput(func() {
				capturedStderr = captureStderr(t, func() {
					err = cmd.RunE(cmd, args)
				})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, capturedOutput+capturedStderr, tt.expectedOutput)

				// Verify the mock captured the update request
				if mockService.capturedUpdateIXUID != "" {
					assert.Equal(t, tt.ixUID, mockService.capturedUpdateIXUID)
				}
			}
		})
	}
}

func TestUpdateIXFunc(t *testing.T) {
	mockService := &MockIXService{
		updateIXResponse: &megaport.IX{
			ProductUID:         "ix-123",
			ProductName:        "Updated IX",
			ProvisioningStatus: "LIVE",
		},
	}

	client := &megaport.Client{
		IXService: mockService,
	}

	ctx := context.Background()
	name := "Updated IX"
	req := &megaport.UpdateIXRequest{
		Name: &name,
	}
	resp, err := updateIXFunc(ctx, client, "ix-123", req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Updated IX", resp.ProductName)
	assert.Equal(t, "ix-123", mockService.capturedUpdateIXUID)
	assert.Equal(t, req, mockService.capturedUpdateIXReq)
}

func TestUpdateIXFunc_Error(t *testing.T) {
	mockService := &MockIXService{
		updateIXError: fmt.Errorf("update failed"),
	}

	client := &megaport.Client{
		IXService: mockService,
	}

	ctx := context.Background()
	name := "Updated IX"
	req := &megaport.UpdateIXRequest{
		Name: &name,
	}
	resp, err := updateIXFunc(ctx, client, "ix-123", req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "update failed")
}

func TestBuyIX_JSONStringMode(t *testing.T) {
	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	mockService := &MockIXService{
		buyIXResponse: &megaport.BuyIXResponse{
			TechnicalServiceUID: "ix-json-abc",
		},
	}

	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.IXService = mockService
	})
	defer cleanup()

	cmd := &cobra.Command{
		Use:  "buy",
		RunE: testutil.NoColorAdapter(BuyIX),
	}

	cmd.Flags().BoolP("interactive", "i", false, "Use interactive mode")
	cmd.Flags().String("product-uid", "", "Port UID")
	cmd.Flags().String("name", "", "IX name")
	cmd.Flags().String("network-service-type", "", "Network service type")
	cmd.Flags().Int("asn", 0, "ASN")
	cmd.Flags().String("mac-address", "", "MAC address")
	cmd.Flags().Int("rate-limit", 0, "Rate limit")
	cmd.Flags().Int("vlan", 0, "VLAN")
	cmd.Flags().Bool("shutdown", false, "Shutdown")
	cmd.Flags().String("promo-code", "", "Promo code")
	cmd.Flags().String("json", "", "JSON string")
	cmd.Flags().String("json-file", "", "JSON file")
	cmd.Flags().Bool("yes", false, "Skip confirmation")

	jsonInput := `{"productUid":"port-uid-json","productName":"JSON IX","networkServiceType":"Sydney IX","asn":65100,"macAddress":"AA:BB:CC:DD:EE:FF","rateLimit":2000,"vlan":200}`
	_ = cmd.Flags().Set("json", jsonInput)
	_ = cmd.Flags().Set("yes", "true")

	var err error
	var capturedStderr string
	capturedOutput := output.CaptureOutput(func() {
		capturedStderr = captureStderr(t, func() {
			err = cmd.RunE(cmd, []string{})
		})
	})

	assert.NoError(t, err)
	assert.Contains(t, capturedOutput+capturedStderr, "IX created")
	assert.Contains(t, capturedOutput+capturedStderr, "ix-json-abc")

	// Verify captured request fields
	assert.NotNil(t, mockService.capturedBuyIXRequest)
	assert.Equal(t, "port-uid-json", mockService.capturedBuyIXRequest.ProductUID)
	assert.Equal(t, "JSON IX", mockService.capturedBuyIXRequest.Name)
	assert.Equal(t, "Sydney IX", mockService.capturedBuyIXRequest.NetworkServiceType)
	assert.Equal(t, 65100, mockService.capturedBuyIXRequest.ASN)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", mockService.capturedBuyIXRequest.MACAddress)
	assert.Equal(t, 2000, mockService.capturedBuyIXRequest.RateLimit)
	assert.Equal(t, 200, mockService.capturedBuyIXRequest.VLAN)
}

func TestBuyIX_InvalidJSON(t *testing.T) {
	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.IXService = &MockIXService{}
	})
	defer cleanup()

	cmd := &cobra.Command{
		Use:  "buy",
		RunE: testutil.NoColorAdapter(BuyIX),
	}

	cmd.Flags().BoolP("interactive", "i", false, "Use interactive mode")
	cmd.Flags().String("product-uid", "", "Port UID")
	cmd.Flags().String("name", "", "IX name")
	cmd.Flags().String("network-service-type", "", "Network service type")
	cmd.Flags().Int("asn", 0, "ASN")
	cmd.Flags().String("mac-address", "", "MAC address")
	cmd.Flags().Int("rate-limit", 0, "Rate limit")
	cmd.Flags().Int("vlan", 0, "VLAN")
	cmd.Flags().Bool("shutdown", false, "Shutdown")
	cmd.Flags().String("promo-code", "", "Promo code")
	cmd.Flags().String("json", "", "JSON string")
	cmd.Flags().String("json-file", "", "JSON file")

	_ = cmd.Flags().Set("json", `{invalid json}`)

	var err error
	output.CaptureOutput(func() {
		err = cmd.RunE(cmd, []string{})
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON")
}

func TestBuyIX_LoginError(t *testing.T) {
	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	cleanup := testutil.SetupLoginError(fmt.Errorf("login failed"))
	defer cleanup()

	cmd := &cobra.Command{
		Use:  "buy",
		RunE: testutil.NoColorAdapter(BuyIX),
	}

	cmd.Flags().BoolP("interactive", "i", false, "Use interactive mode")
	cmd.Flags().String("product-uid", "", "Port UID")
	cmd.Flags().String("name", "", "IX name")
	cmd.Flags().String("network-service-type", "", "Network service type")
	cmd.Flags().Int("asn", 0, "ASN")
	cmd.Flags().String("mac-address", "", "MAC address")
	cmd.Flags().Int("rate-limit", 0, "Rate limit")
	cmd.Flags().Int("vlan", 0, "VLAN")
	cmd.Flags().Bool("shutdown", false, "Shutdown")
	cmd.Flags().String("promo-code", "", "Promo code")
	cmd.Flags().String("json", "", "JSON string")
	cmd.Flags().String("json-file", "", "JSON file")

	_ = cmd.Flags().Set("product-uid", "port-uid-123")
	_ = cmd.Flags().Set("name", "Test IX")
	_ = cmd.Flags().Set("network-service-type", "Los Angeles IX")
	_ = cmd.Flags().Set("asn", "65000")
	_ = cmd.Flags().Set("mac-address", "00:11:22:33:44:55")
	_ = cmd.Flags().Set("rate-limit", "1000")
	_ = cmd.Flags().Set("vlan", "100")

	var err error
	output.CaptureOutput(func() {
		err = cmd.RunE(cmd, []string{})
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "login failed")
}

func TestBuyIX_Confirmation(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	originalBuyIXFunc := buyIXFunc
	defer func() { buyIXFunc = originalBuyIXFunc }()

	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()

	tests := []struct {
		name                 string
		flags                map[string]string
		jsonInput            string
		confirmResult        bool
		expectBuyCalled      bool
		expectedOutput       string
		expectedError        string
		promptShouldBeCalled bool
	}{
		{
			name: "confirmation accepted",
			flags: map[string]string{
				"product-uid":          "port-uid-123",
				"name":                 "Test IX",
				"network-service-type": "Los Angeles IX",
				"asn":                  "65000",
				"mac-address":          "00:11:22:33:44:55",
				"rate-limit":           "1000",
				"vlan":                 "100",
			},
			confirmResult:        true,
			expectBuyCalled:      true,
			expectedOutput:       "IX created",
			promptShouldBeCalled: true,
		},
		{
			name: "confirmation denied",
			flags: map[string]string{
				"product-uid":          "port-uid-123",
				"name":                 "Test IX",
				"network-service-type": "Los Angeles IX",
				"asn":                  "65000",
				"mac-address":          "00:11:22:33:44:55",
				"rate-limit":           "1000",
				"vlan":                 "100",
			},
			confirmResult:        false,
			expectBuyCalled:      false,
			expectedError:        "cancelled by user",
			promptShouldBeCalled: true,
		},
		{
			name: "yes flag skips confirmation",
			flags: map[string]string{
				"product-uid":          "port-uid-123",
				"name":                 "Test IX",
				"network-service-type": "Los Angeles IX",
				"asn":                  "65000",
				"mac-address":          "00:11:22:33:44:55",
				"rate-limit":           "1000",
				"vlan":                 "100",
				"yes":                  "true",
			},
			confirmResult:        false,
			expectBuyCalled:      true,
			expectedOutput:       "IX created",
			promptShouldBeCalled: false,
		},
		{
			name: "json input with yes skips confirmation",
			flags: map[string]string{
				"json": `{"productUid":"port-uid-123","productName":"Test IX","networkServiceType":"Los Angeles IX","asn":65000,"macAddress":"00:11:22:33:44:55","rateLimit":1000,"vlan":100}`,
				"yes":  "true",
			},
			confirmResult:        false,
			expectBuyCalled:      true,
			expectedOutput:       "IX created",
			promptShouldBeCalled: false,
		},
		{
			name: "json input without yes is a usage error",
			flags: map[string]string{
				"json": `{"productUid":"port-uid-123","productName":"Test IX","networkServiceType":"Los Angeles IX","asn":65000,"macAddress":"00:11:22:33:44:55","rateLimit":1000,"vlan":100}`,
			},
			expectBuyCalled:      false,
			expectedError:        "--yes is required to confirm a purchase when using --json or --json-file",
			promptShouldBeCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockIXService{
				buyIXResponse: &megaport.BuyIXResponse{
					TechnicalServiceUID: "ix-confirm-123",
				},
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.IXService = mockService
				return client, nil
			})

			buyCalled := false
			buyIXFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyIXRequest) (*megaport.BuyIXResponse, error) {
				buyCalled = true
				return &megaport.BuyIXResponse{
					TechnicalServiceUID: "ix-confirm-123",
				}, nil
			}

			promptCalled := false
			utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool {
				promptCalled = true
				return tt.confirmResult
			})

			cmd := &cobra.Command{
				Use:  "buy",
				RunE: testutil.NoColorAdapter(BuyIX),
			}

			cmd.Flags().BoolP("interactive", "i", false, "")
			cmd.Flags().BoolP("yes", "y", false, "")
			cmd.Flags().Bool("no-wait", false, "")
			cmd.Flags().String("product-uid", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().String("network-service-type", "", "")
			cmd.Flags().Int("asn", 0, "")
			cmd.Flags().String("mac-address", "", "")
			cmd.Flags().Int("rate-limit", 0, "")
			cmd.Flags().Int("vlan", 0, "")
			cmd.Flags().Bool("shutdown", false, "")
			cmd.Flags().String("promo-code", "", "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")

			testutil.SetFlags(t, cmd, tt.flags)

			var err error
			var capturedStderr string
			capturedOutput := output.CaptureOutput(func() {
				capturedStderr = captureStderr(t, func() {
					err = cmd.RunE(cmd, nil)
				})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, capturedOutput+capturedStderr, tt.expectedOutput)
			}
			assert.Equal(t, tt.expectBuyCalled, buyCalled, "buy function called mismatch")
			assert.Equal(t, tt.promptShouldBeCalled, promptCalled, "BuyConfirmPrompt called expectation mismatch")
		})
	}
}

func TestGetIXFunc(t *testing.T) {
	mockService := &MockIXService{
		getIXResponse: &megaport.IX{
			ProductUID:         "ix-123",
			ProductName:        "Test IX",
			ProvisioningStatus: "LIVE",
		},
	}

	client := &megaport.Client{
		IXService: mockService,
	}

	ctx := context.Background()
	ix, err := getIXFunc(ctx, client, "ix-123")
	assert.NoError(t, err)
	assert.NotNil(t, ix)
	assert.Equal(t, "ix-123", ix.ProductUID)
	assert.Equal(t, "Test IX", ix.ProductName)
}

func TestBuyIXFunc(t *testing.T) {
	mockService := &MockIXService{
		buyIXResponse: &megaport.BuyIXResponse{
			TechnicalServiceUID: "ix-123-abc",
		},
	}

	client := &megaport.Client{
		IXService: mockService,
	}

	ctx := context.Background()
	req := &megaport.BuyIXRequest{
		ProductUID:         "port-uid-123",
		Name:               "Test IX",
		NetworkServiceType: "Los Angeles IX",
		ASN:                65000,
		MACAddress:         "00:11:22:33:44:55",
		RateLimit:          1000,
		VLAN:               100,
	}
	resp, err := buyIXFunc(ctx, client, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "ix-123-abc", resp.TechnicalServiceUID)
}

func TestDeleteIXFunc(t *testing.T) {
	mockService := &MockIXService{}

	client := &megaport.Client{
		IXService: mockService,
	}

	ctx := context.Background()
	req := &megaport.DeleteIXRequest{
		DeleteNow: true,
	}
	err := deleteIXFunc(ctx, client, "ix-123", req)
	assert.NoError(t, err)
	assert.Equal(t, "ix-123", mockService.capturedDeleteIXUID)
}

func TestExportIXConfig(t *testing.T) {
	ix := &megaport.IX{
		ProductUID:         "ix-should-not-appear",
		ProductName:        "My IX",
		NetworkServiceType: "Los Angeles IX",
		RateLimit:          1000,
		VLAN:               100,
		ASN:                65000,
		MACAddress:         "00:11:22:33:44:55",
		ProvisioningStatus: "LIVE",
	}
	m := exportIXConfig(ix)

	assert.Equal(t, "My IX", m["productName"])
	assert.Equal(t, "Los Angeles IX", m["networkServiceType"])
	assert.Equal(t, 1000, m["rateLimit"])
	assert.Equal(t, 100, m["vlan"])
	assert.Equal(t, 65000, m["asn"])
	assert.Equal(t, "00:11:22:33:44:55", m["macAddress"])

	_, hasUID := m["productUid"]
	assert.False(t, hasUID, "export should not include productUid (parent port UID)")
	_, hasStatus := m["provisioningStatus"]
	assert.False(t, hasStatus, "export should not include provisioningStatus")
}

func TestGetIX_Export(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	mockService := &MockIXService{
		getIXResponse: &megaport.IX{
			ProductUID:         "ix-export-123",
			ProductName:        "Export IX",
			NetworkServiceType: "Sydney IX",
			RateLimit:          500,
			VLAN:               200,
			ASN:                65001,
			ProvisioningStatus: "LIVE",
		},
	}
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.IXService = mockService
		return client, nil
	})

	cmd := &cobra.Command{Use: "get"}
	cmd.Flags().Bool("export", false, "")
	assert.NoError(t, cmd.Flags().Set("export", "true"))

	var err error
	capturedOutput := output.CaptureStdout(func() {
		err = GetIX(cmd, []string{"ix-export-123"}, true, "table")
	})

	assert.NoError(t, err)
	var parsed map[string]interface{}
	assert.NoError(t, json.Unmarshal([]byte(capturedOutput), &parsed), "export output must be valid JSON")
	assert.Equal(t, "Export IX", parsed["productName"])
	assert.Equal(t, float64(200), parsed["vlan"])
	_, hasUID := parsed["productUid"]
	assert.False(t, hasUID, "export should not include productUid")
}

func TestValidateIX(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name             string
		flags            map[string]string
		jsonInput        string
		jsonFileContent  string
		setupMock        func(*MockIXService)
		loginError       error
		expectedError    string
		expectedContains string
	}{
		{
			name: "success with flags",
			flags: map[string]string{
				"name":                 "test-ix",
				"product-uid":          "port-123",
				"network-service-type": "Los Angeles IX",
				"asn":                  "65000",
				"mac-address":          "00:11:22:33:44:55",
				"rate-limit":           "1000",
				"vlan":                 "100",
			},
			setupMock:        func(m *MockIXService) {},
			expectedContains: "validation passed",
		},
		{
			name:             "success with JSON",
			jsonInput:        `{"productUid":"port-123","productName":"test-ix","networkServiceType":"Los Angeles IX","asn":65000,"macAddress":"00:11:22:33:44:55","rateLimit":1000,"vlan":100}`,
			setupMock:        func(m *MockIXService) {},
			expectedContains: "validation passed",
		},
		{
			name: "validation error",
			flags: map[string]string{
				"name":                 "test-ix",
				"product-uid":          "port-123",
				"network-service-type": "Los Angeles IX",
				"asn":                  "65000",
				"mac-address":          "00:11:22:33:44:55",
				"rate-limit":           "1000",
				"vlan":                 "100",
			},
			setupMock: func(m *MockIXService) {
				m.validateIXOrderError = fmt.Errorf("invalid IX configuration")
			},
			expectedError: "invalid IX configuration",
		},
		{
			name:          "no input provided",
			flags:         map[string]string{},
			setupMock:     func(m *MockIXService) {},
			expectedError: "no input provided",
		},
		{
			name: "login error",
			flags: map[string]string{
				"name":                 "test-ix",
				"product-uid":          "port-123",
				"network-service-type": "Los Angeles IX",
				"asn":                  "65000",
				"mac-address":          "00:11:22:33:44:55",
				"rate-limit":           "1000",
				"vlan":                 "100",
			},
			setupMock:     func(m *MockIXService) {},
			loginError:    fmt.Errorf("authentication failed"),
			expectedError: "authentication failed",
		},
		{
			name:          "invalid JSON input",
			jsonInput:     `{invalid json}`,
			setupMock:     func(m *MockIXService) {},
			expectedError: "failed to parse JSON",
		},
		{
			name:             "success with JSON file",
			jsonFileContent:  `{"productUid":"port-123","productName":"file-ix","networkServiceType":"Los Angeles IX","asn":65000,"macAddress":"00:11:22:33:44:55","rateLimit":1000,"vlan":100}`,
			setupMock:        func(m *MockIXService) {},
			expectedContains: "validation passed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockIXService{}
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
					client.IXService = mockService
					return client, nil
				})
			}

			cmd := &cobra.Command{Use: "validate"}
			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().String("product-uid", "", "")
			cmd.Flags().String("network-service-type", "", "")
			cmd.Flags().Int("asn", 0, "")
			cmd.Flags().String("mac-address", "", "")
			cmd.Flags().Int("rate-limit", 0, "")
			cmd.Flags().Int("vlan", 0, "")
			cmd.Flags().Bool("shutdown", false, "")
			cmd.Flags().String("promo-code", "", "")

			if tt.jsonInput != "" {
				assert.NoError(t, cmd.Flags().Set("json", tt.jsonInput))
			}
			if tt.jsonFileContent != "" {
				tmpFile, tmpErr := os.CreateTemp("", "ix-validate-*.json")
				assert.NoError(t, tmpErr)
				defer os.Remove(tmpFile.Name())
				_, tmpErr = tmpFile.WriteString(tt.jsonFileContent)
				assert.NoError(t, tmpErr)
				tmpFile.Close()
				assert.NoError(t, cmd.Flags().Set("json-file", tmpFile.Name()))
			}
			for k, v := range tt.flags {
				assert.NoError(t, cmd.Flags().Set(k, v))
			}

			var err error
			var capturedStderr string
			capturedOutput := output.CaptureOutput(func() {
				capturedStderr = captureStderr(t, func() {
					err = ValidateIX(cmd, nil, true)
				})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.expectedContains != "" {
					assert.Contains(t, capturedOutput+capturedStderr, tt.expectedContains)
				}
			}
		})
	}
}

func TestMockIXServiceReset(t *testing.T) {
	m := &MockIXService{
		buyIXError:    fmt.Errorf("test"),
		getIXError:    fmt.Errorf("test"),
		deleteIXError: fmt.Errorf("test"),
		forceNilGetIX: true,
	}
	m.Reset()
	assert.Nil(t, m.buyIXError)
	assert.Nil(t, m.getIXError)
	assert.Nil(t, m.deleteIXError)
	assert.False(t, m.forceNilGetIX)
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

// TestBuyIX_NoRetryOnAmbiguousError ensures a buy is never re-submitted after an
// ambiguous 502, which could double-submit the order and duplicate billing.
func TestBuyIX_NoRetryOnAmbiguousError(t *testing.T) {
	originalBuyIXFunc := buyIXFunc
	originalMaxRetries := utils.MaxRetries
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer func() {
		cleanup()
		buyIXFunc = originalBuyIXFunc
		utils.MaxRetries = originalMaxRetries
	}()
	// Retries are available; the order-safe policy must still refuse to use them on a 502.
	utils.MaxRetries = 3
	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.IXService = &MockIXService{}
		return client, nil
	})

	calls := 0
	buyIXFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyIXRequest) (*megaport.BuyIXResponse, error) {
		calls++
		return nil, &megaport.ErrorResponse{
			Response: &http.Response{
				StatusCode: http.StatusBadGateway,
				Header:     http.Header{},
				Request:    &http.Request{URL: &url.URL{}},
			},
			Message: "bad gateway",
		}
	}

	cmd := &cobra.Command{Use: "buy", RunE: testutil.NoColorAdapter(BuyIX)}
	cmd.Flags().BoolP("interactive", "i", false, "")
	cmd.Flags().Bool("no-wait", false, "")
	cmd.Flags().String("product-uid", "", "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().String("network-service-type", "", "")
	cmd.Flags().Int("asn", 0, "")
	cmd.Flags().String("mac-address", "", "")
	cmd.Flags().Int("rate-limit", 0, "")
	cmd.Flags().Int("vlan", 0, "")
	cmd.Flags().Bool("shutdown", false, "")
	cmd.Flags().String("promo-code", "", "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")

	testutil.SetFlags(t, cmd, map[string]string{
		"product-uid":          "port-uid-123",
		"name":                 "Test IX",
		"network-service-type": "Los Angeles IX",
		"asn":                  "65000",
		"mac-address":          "00:11:22:33:44:55",
		"rate-limit":           "1000",
		"vlan":                 "100",
	})
	assert.NoError(t, cmd.Flags().Set("no-wait", "true"))

	var err error
	output.CaptureOutput(func() {
		err = cmd.RunE(cmd, nil)
	})

	assert.Error(t, err)
	assert.Equal(t, 1, calls, "buy must not retry on an ambiguous 502")
}

// TestBuyIX_NoResubmitOnPollRetryableError proves the order is submitted exactly
// once even when a retryable error (429) surfaces during the provisioning poll.
func TestBuyIX_NoResubmitOnPollRetryableError(t *testing.T) {
	originalBuyIXFunc := buyIXFunc
	originalGetIXFunc := getIXFunc
	originalMaxRetries := utils.MaxRetries
	originalPollInterval := utils.ProvisionPollInterval
	originalNoRetry := utils.NoRetry
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer func() {
		cleanup()
		buyIXFunc = originalBuyIXFunc
		getIXFunc = originalGetIXFunc
		utils.MaxRetries = originalMaxRetries
		utils.ProvisionPollInterval = originalPollInterval
		utils.NoRetry = originalNoRetry
	}()
	utils.MaxRetries = 3
	utils.NoRetry = false
	utils.ProvisionPollInterval = time.Millisecond

	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.IXService = &MockIXService{}
		return client, nil
	})

	buyCalls := 0
	buyIXFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyIXRequest) (*megaport.BuyIXResponse, error) {
		buyCalls++
		return &megaport.BuyIXResponse{TechnicalServiceUID: "ix-uid-123"}, nil
	}
	getIXFunc = func(ctx context.Context, client *megaport.Client, ixUID string) (*megaport.IX, error) {
		return nil, &megaport.ErrorResponse{
			Response: &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Header:     http.Header{},
				Request:    &http.Request{URL: &url.URL{}},
			},
			Message: "too many requests",
		}
	}

	cmd := &cobra.Command{Use: "buy", RunE: testutil.NoColorAdapter(BuyIX)}
	cmd.Flags().BoolP("interactive", "i", false, "")
	cmd.Flags().Bool("no-wait", false, "")
	cmd.Flags().String("product-uid", "", "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().String("network-service-type", "", "")
	cmd.Flags().Int("asn", 0, "")
	cmd.Flags().String("mac-address", "", "")
	cmd.Flags().Int("rate-limit", 0, "")
	cmd.Flags().Int("vlan", 0, "")
	cmd.Flags().Bool("shutdown", false, "")
	cmd.Flags().String("promo-code", "", "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")

	testutil.SetFlags(t, cmd, map[string]string{
		"product-uid":          "port-uid-123",
		"name":                 "Test IX",
		"network-service-type": "Los Angeles IX",
		"asn":                  "65000",
		"mac-address":          "00:11:22:33:44:55",
		"rate-limit":           "1000",
		"vlan":                 "100",
	})
	// no-wait stays false: the provisioning poll must run.

	var err error
	output.CaptureOutput(func() {
		err = cmd.RunE(cmd, nil)
	})

	assert.Error(t, err, "a retryable error during the provisioning poll must surface as an error")
	assert.Equal(t, 1, buyCalls, "the order must be submitted exactly once; a poll-phase 429 must not re-submit it")
}

// TestBuyIX_OrderAndProvisionFailures covers the order-response and
// provisioning-poll failure branches: an empty API response, a missing UID, and
// the IX vanishing during the provisioning poll.
func TestBuyIX_OrderAndProvisionFailures(t *testing.T) {
	originalBuyIXFunc := buyIXFunc
	originalGetIXFunc := getIXFunc
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer func() {
		cleanup()
		buyIXFunc = originalBuyIXFunc
		getIXFunc = originalGetIXFunc
	}()

	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.IXService = &MockIXService{}
		return client, nil
	})

	tests := []struct {
		name     string
		buyResp  *megaport.BuyIXResponse
		getNil   bool
		errMatch string
	}{
		{name: "empty API response", buyResp: nil, errMatch: "empty response from API"},
		{name: "no UID returned", buyResp: &megaport.BuyIXResponse{}, errMatch: "no UID returned"},
		{name: "IX not found during provisioning poll", buyResp: &megaport.BuyIXResponse{TechnicalServiceUID: "ix-uid-123"}, getNil: true, errMatch: "not found while waiting for provisioning"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buyIXFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyIXRequest) (*megaport.BuyIXResponse, error) {
				return tt.buyResp, nil
			}
			getIXFunc = func(ctx context.Context, client *megaport.Client, ixUID string) (*megaport.IX, error) {
				if tt.getNil {
					return nil, nil
				}
				return &megaport.IX{ProvisioningStatus: "LIVE"}, nil
			}

			cmd := &cobra.Command{Use: "buy"}
			cmd.Flags().BoolP("interactive", "i", false, "")
			cmd.Flags().Bool("no-wait", false, "")
			cmd.Flags().String("product-uid", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().String("network-service-type", "", "")
			cmd.Flags().Int("asn", 0, "")
			cmd.Flags().String("mac-address", "", "")
			cmd.Flags().Int("rate-limit", 0, "")
			cmd.Flags().Int("vlan", 0, "")
			cmd.Flags().Bool("shutdown", false, "")
			cmd.Flags().String("promo-code", "", "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")

			testutil.SetFlags(t, cmd, map[string]string{
				"product-uid":          "port-uid-123",
				"name":                 "Test IX",
				"network-service-type": "Los Angeles IX",
				"asn":                  "65000",
				"mac-address":          "00:11:22:33:44:55",
				"rate-limit":           "1000",
				"vlan":                 "100",
			})

			var err error
			output.CaptureOutput(func() {
				err = BuyIX(cmd, nil, true)
			})

			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tt.errMatch)
			}
		})
	}
}
