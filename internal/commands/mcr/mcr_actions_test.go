package mcr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/config"
	"github.com/megaport/megaport-cli/internal/testutil"
	"github.com/megaport/megaport-cli/internal/utils"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMCRCmd_WithMockClient(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name          string
		mcrID         string
		format        string
		setupMock     func(*MockMCRService)
		expectedError string
		expectedOut   []string
	}{
		{
			name:   "get MCR success table format",
			mcrID:  "mcr-123",
			format: "table",
			setupMock: func(m *MockMCRService) {
				m.GetMCRResult = &megaport.MCR{
					UID:                "mcr-123",
					Name:               "Test MCR",
					LocationID:         123,
					ProvisioningStatus: "LIVE",
					Resources: megaport.MCRResources{
						VirtualRouter: megaport.MCRVirtualRouter{
							ASN: 65000,
						},
					},
				}
			},
			expectedOut: []string{"mcr-123", "Test MCR", "LIVE"},
		},
		{
			name:   "get MCR success json format",
			mcrID:  "mcr-123",
			format: "json",
			setupMock: func(m *MockMCRService) {
				m.GetMCRResult = &megaport.MCR{
					UID:                "mcr-123",
					Name:               "Test MCR",
					LocationID:         123,
					ProvisioningStatus: "LIVE",
					Resources: megaport.MCRResources{
						VirtualRouter: megaport.MCRVirtualRouter{
							ASN: 65000,
						},
					},
				}
			},
			expectedOut: []string{`"uid": "mcr-123"`, `"name": "Test MCR"`, `"location_id": 123`},
		},
		{
			name:   "get MCR nil response",
			mcrID:  "mcr-nil",
			format: "table",
			setupMock: func(m *MockMCRService) {
				m.ForceNilGetMCR = true
			},
			expectedError: "no MCR found with UID: mcr-nil",
		},
		{
			name:   "get MCR error",
			mcrID:  "mcr-invalid",
			format: "table",
			setupMock: func(m *MockMCRService) {
				m.GetMCRErr = fmt.Errorf("MCR not found")
			},
			expectedError: "MCR not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMCRService := &MockMCRService{}
			tt.setupMock(mockMCRService)

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.MCRService = mockMCRService
				return client, nil
			})

			cmd := testutil.NewCommand("get-mcr [mcrID]", func(cmd *cobra.Command, args []string) error {
				return GetMCR(cmd, args, false, tt.format)
			})
			testutil.SetFlags(t, cmd, map[string]string{"output": tt.format})

			var err error

			output := output.CaptureOutput(func() {
				err = cmd.RunE(cmd, []string{tt.mcrID})
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

func TestDeleteMCRCmd_WithMockClient(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	originalPrompt := utils.GetResourcePrompt()
	originalConfirmPrompt := utils.GetConfirmPrompt()
	defer func() {
		utils.SetResourcePrompt(originalPrompt)
		utils.SetConfirmPrompt(originalConfirmPrompt)
	}()

	tests := []struct {
		name           string
		mcrID          string
		force          bool
		safeDelete     bool
		promptResponse string
		setupMock      func(*MockMCRService)
		expectedError  string
		expectedOutput string
		expectDeleted  bool
	}{
		{
			name:           "confirm deletion",
			mcrID:          "mcr-to-delete",
			force:          false,
			promptResponse: "y",
			setupMock: func(m *MockMCRService) {
				m.DeleteMCRResult = &megaport.DeleteMCRResponse{
					IsDeleting: true,
				}
			},
			expectedOutput: "MCR deleted",
			expectDeleted:  true,
		},
		{
			name:           "safe delete flag passed to request",
			mcrID:          "mcr-safe-delete",
			force:          true,
			safeDelete:     true,
			promptResponse: "",
			setupMock: func(m *MockMCRService) {
				m.DeleteMCRResult = &megaport.DeleteMCRResponse{
					IsDeleting: true,
				}
			},
			expectedOutput: "MCR deleted",
			expectDeleted:  true,
		},
		{
			name:  "force deletion",
			mcrID: "mcr-force-delete",
			force: true,
			setupMock: func(m *MockMCRService) {
				m.DeleteMCRResult = &megaport.DeleteMCRResponse{
					IsDeleting: true,
				}
			},
			expectedOutput: "MCR deleted",
			expectDeleted:  true,
		},
		{
			name:           "cancel deletion",
			mcrID:          "mcr-keep",
			force:          false,
			promptResponse: "n",
			setupMock:      func(m *MockMCRService) {},
			expectedError:  "cancelled by user",
			expectDeleted:  false,
		},
		{
			name:           "deletion error",
			mcrID:          "mcr-error",
			force:          false,
			promptResponse: "y",
			setupMock: func(m *MockMCRService) {
				m.DeleteMCRErr = fmt.Errorf("failed to delete MCR")
			},
			expectedError: "failed to delete MCR",
			expectDeleted: false,
		},
		{
			name:  "deletion not successful",
			mcrID: "mcr-not-deleting",
			force: true,
			setupMock: func(m *MockMCRService) {
				m.DeleteMCRResult = &megaport.DeleteMCRResponse{
					IsDeleting: false,
				}
			},
			expectedError: "not successful for mcr-not-deleting",
			expectDeleted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMCRService := &MockMCRService{}
			tt.setupMock(mockMCRService)

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.MCRService = mockMCRService
				return client, nil
			})

			utils.SetConfirmPrompt(func(message string, _ bool) bool {
				assert.Contains(t, message, fmt.Sprintf("Are you sure you want to delete MCR %s?", tt.mcrID))
				return tt.promptResponse == "y"
			})

			utils.SetResourcePrompt(func(_, msg string, _ bool) (string, error) {
				assert.Contains(t, msg, fmt.Sprintf("Are you sure you want to delete MCR %s?", tt.mcrID))
				return tt.promptResponse, nil
			})

			cmd := &cobra.Command{
				Use: "delete-mcr [mcrID]",
				RunE: func(cmd *cobra.Command, args []string) error {
					return DeleteMCR(cmd, args, false)
				},
			}
			cmd.Flags().BoolP("force", "f", false, "Force deletion without confirmation")
			cmd.Flags().Bool("safe-delete", false, "Fail if the resource has attached VXCs or other active services")
			flags := map[string]string{
				"force": fmt.Sprintf("%v", tt.force),
			}
			if tt.safeDelete {
				flags["safe-delete"] = "true"
			}
			testutil.SetFlags(t, cmd, flags)

			var err error
			output := captureStderr(t, func() {
				err = cmd.RunE(cmd, []string{tt.mcrID})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, output, tt.expectedOutput)

				if tt.expectDeleted {
					assert.NotNil(t, mockMCRService.CapturedDeleteMCRUID)
					assert.Equal(t, tt.mcrID, mockMCRService.CapturedDeleteMCRUID)
					assert.True(t, mockMCRService.CapturedDeleteMCRRequest.DeleteNow,
						"MCR delete must always send DeleteNow=true (SDK rejects deferred deletion)")
					if tt.safeDelete {
						assert.True(t, mockMCRService.CapturedDeleteMCRRequest.SafeDelete)
					}
				}
			}
		})
	}
}

// End-to-end proof that a name-only update re-sends the MCR's existing cost
// centre rather than wiping it (the SDK omits omitempty on costCentre).
func TestUpdateMCR_PreservesCostCentre(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	mockMCRService := &MockMCRService{
		GetMCRResult:    &megaport.MCR{UID: "mcr-cc-1", Name: "Original", CostCentre: "IT Dept", ProvisioningStatus: "LIVE"},
		ModifyMCRResult: &megaport.ModifyMCRResponse{IsUpdated: true},
	}
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.MCRService = mockMCRService
		return client, nil
	})

	cmd := &cobra.Command{Use: "update"}
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().String("cost-centre", "", "")
	cmd.Flags().Bool("marketplace-visibility", false, "")
	cmd.Flags().Int("term", 0, "")
	cmd.Flags().Int("mcr-asn", 0, "")
	require.NoError(t, cmd.Flags().Set("name", "Renamed MCR"))

	_ = captureStderr(t, func() {
		output.CaptureOutput(func() {
			require.NoError(t, UpdateMCR(cmd, []string{"mcr-cc-1"}, true))
		})
	})

	require.NotNil(t, mockMCRService.CapturedModifyMCRRequest)
	assert.Equal(t, "IT Dept", mockMCRService.CapturedModifyMCRRequest.CostCentre,
		"name-only update must preserve the existing cost centre")
}

func newUpdateMCRCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "update"}
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().String("cost-centre", "", "")
	cmd.Flags().Bool("marketplace-visibility", false, "")
	cmd.Flags().Int("term", 0, "")
	cmd.Flags().Int("mcr-asn", 0, "")
	return cmd
}

func TestUpdateMCR_CostCentreOverrideAndClear(t *testing.T) {
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

			mockMCRService := &MockMCRService{
				GetMCRResult:    &megaport.MCR{UID: "mcr-cc-1", Name: "Original", CostCentre: "IT Dept", ProvisioningStatus: "LIVE"},
				ModifyMCRResult: &megaport.ModifyMCRResponse{IsUpdated: true},
			}
			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.MCRService = mockMCRService
				return client, nil
			})

			cmd := newUpdateMCRCmd()
			tt.setFlags(cmd)

			_ = captureStderr(t, func() {
				output.CaptureOutput(func() {
					require.NoError(t, UpdateMCR(cmd, []string{"mcr-cc-1"}, true))
				})
			})

			require.NotNil(t, mockMCRService.CapturedModifyMCRRequest)
			assert.Equal(t, tt.expected, mockMCRService.CapturedModifyMCRRequest.CostCentre)
		})
	}
}

// Bad input must fail before any network round-trip, so login/fetch errors
// never mask the real validation error.
func TestUpdateMCR_ValidatesBeforeLogin(t *testing.T) {
	cleanup := testutil.SetupLoginError(fmt.Errorf("login must not be reached"))
	defer cleanup()

	cmd := newUpdateMCRCmd()
	require.NoError(t, cmd.Flags().Set("mcr-asn", "0"))

	var err error
	_ = captureStderr(t, func() {
		output.CaptureOutput(func() {
			err = UpdateMCR(cmd, []string{"mcr-1"}, true)
		})
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "MCR ASN", "input must be validated before login")
}

func TestRestoreMCRCmd_WithMockClient(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name          string
		mcrID         string
		setupMock     func(*MockMCRService)
		expectedError string
		expectedOut   string
	}{
		{
			name:  "restore MCR success",
			mcrID: "mcr-to-restore",
			setupMock: func(m *MockMCRService) {
				m.RestoreMCRResult = &megaport.RestoreMCRResponse{
					IsRestored: true,
				}
			},
			expectedOut: "MCR mcr-to-restore restored successfully",
		},
		{
			name:  "restore MCR error",
			mcrID: "mcr-error",
			setupMock: func(m *MockMCRService) {
				m.RestoreMCRErr = fmt.Errorf("failed to restore MCR")
			},
			expectedError: "failed to restore MCR",
		},
		{
			name:  "restore MCR unsuccessful",
			mcrID: "mcr-fail",
			setupMock: func(m *MockMCRService) {
				m.RestoreMCRResult = &megaport.RestoreMCRResponse{
					IsRestored: false,
				}
			},
			expectedError: "not successful for mcr-fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMCRService := &MockMCRService{}
			tt.setupMock(mockMCRService)

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.MCRService = mockMCRService
				return client, nil
			})

			restoreMCRCmd := &cobra.Command{
				Use: "restore-mcr [mcrID]",
				RunE: func(cmd *cobra.Command, args []string) error {
					return RestoreMCR(cmd, args, false)
				},
			}

			var err error
			output := captureStderr(t, func() {
				err = restoreMCRCmd.RunE(restoreMCRCmd, []string{tt.mcrID})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, output, tt.expectedOut)
			}
		})
	}
}

func TestGetMCRFunc(t *testing.T) {
	mockMCRService := &MockMCRService{
		GetMCRResult: &megaport.MCR{
			UID:                "mcr-123",
			Name:               "Test MCR",
			LocationID:         123,
			ProvisioningStatus: "LIVE",
		},
	}

	client := &megaport.Client{
		MCRService: mockMCRService,
	}

	ctx := context.Background()
	mcr, err := getMCRFunc(ctx, client, "mcr-123")
	assert.NoError(t, err)
	assert.NotNil(t, mcr)
	assert.Equal(t, "mcr-123", mcr.UID)
	assert.Equal(t, "Test MCR", mcr.Name)
}

func TestBuyMCRFunc(t *testing.T) {
	mockMCRService := &MockMCRService{
		BuyMCRResult: &megaport.BuyMCRResponse{
			TechnicalServiceUID: "mcr-123-abc",
		},
	}

	client := &megaport.Client{
		MCRService: mockMCRService,
	}

	ctx := context.Background()
	req := &megaport.BuyMCRRequest{
		Name:             "Test MCR",
		Term:             12,
		PortSpeed:        1000,
		LocationID:       123,
		DiversityZone:    "red",
		CostCentre:       "cost-123",
		PromoCode:        "PROMO2025",
		WaitForProvision: true,
		WaitForTime:      10 * time.Minute,
	}
	resp, err := buyMCRFunc(ctx, client, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "mcr-123-abc", resp.TechnicalServiceUID)
}

func TestDeleteMCRFunc(t *testing.T) {
	mockMCRService := &MockMCRService{
		DeleteMCRResult: &megaport.DeleteMCRResponse{
			IsDeleting: true,
		},
	}

	client := &megaport.Client{
		MCRService: mockMCRService,
	}

	ctx := context.Background()
	req := &megaport.DeleteMCRRequest{
		MCRID:     "mcr-123",
		DeleteNow: true,
	}
	resp, err := deleteMCRFunc(ctx, client, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.IsDeleting)
}

func TestRestoreMCRFunc(t *testing.T) {
	mockMCRService := &MockMCRService{
		RestoreMCRResult: &megaport.RestoreMCRResponse{
			IsRestored: true,
		},
	}

	client := &megaport.Client{
		MCRService: mockMCRService,
	}

	ctx := context.Background()
	resp, err := restoreMCRFunc(ctx, client, "mcr-123")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.IsRestored)
}

func TestListMCRPrefixFilterListsCmd_WithMockClient(t *testing.T) {
	output.SetTerminalWidthForTesting(200)
	defer output.SetTerminalWidthForTesting(0)
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name           string
		mcrUID         string
		setupMock      func(*MockMCRService)
		expectedError  string
		expectedOutput string
	}{
		{
			name:   "successful list prefix filter lists",
			mcrUID: "mcr-123",
			setupMock: func(m *MockMCRService) {
				m.ListMCRPrefixFilterListsResult = []*megaport.PrefixFilterList{
					{
						Id:          1,
						Description: "Test Prefix Filter List 1",
					},
					{
						Id:          2,
						Description: "Test Prefix Filter List 2",
					},
				}
			},
			expectedOutput: "Test Prefix Filter List 1",
		},
		{
			name:   "API error",
			mcrUID: "mcr-123",
			setupMock: func(m *MockMCRService) {
				m.ListMCRPrefixFilterListsErr = fmt.Errorf("API error: service unavailable")
			},
			expectedError: "API error: service unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMCRService := &MockMCRService{}
			tt.setupMock(mockMCRService)

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.MCRService = mockMCRService
				return client, nil
			})

			listMCRPrefixFilterListsCmd := &cobra.Command{
				Use: "list-mcr-prefix-filter-lists [mcrUID]",
				RunE: func(cmd *cobra.Command, args []string) error {
					return ListMCRPrefixFilterLists(cmd, args, false, "table")
				},
			}

			cmd := listMCRPrefixFilterListsCmd
			var err error
			output := output.CaptureOutput(func() {
				err = cmd.RunE(cmd, []string{tt.mcrUID})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, output, tt.expectedOutput)
			}
		})
	}
}

func TestGetMCRPrefixFilterListCmd_WithMockClient(t *testing.T) {
	output.SetTerminalWidthForTesting(200)
	defer output.SetTerminalWidthForTesting(0)
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name            string
		mcrUID          string
		prefixListID    int
		rawPrefixListID string
		setupMock       func(*MockMCRService)
		expectedError   string
		expectedOutput  string
	}{
		{
			name:         "successful get prefix filter list",
			mcrUID:       "mcr-123",
			prefixListID: 1,
			setupMock: func(m *MockMCRService) {
				m.GetMCRPrefixFilterListResult = &megaport.MCRPrefixFilterList{
					ID:          1,
					Description: "Test Prefix Filter List",
				}
			},
			expectedOutput: "Test Prefix Filter List",
		},
		{
			name:         "API error",
			mcrUID:       "mcr-123",
			prefixListID: 1,
			setupMock: func(m *MockMCRService) {
				m.GetMCRPrefixFilterListErr = fmt.Errorf("API error: service unavailable")
			},
			expectedError: "API error: service unavailable",
		},
		{
			name:            "invalid prefix filter list ID",
			mcrUID:          "mcr-123",
			rawPrefixListID: "abc",
			setupMock:       func(m *MockMCRService) {},
			expectedError:   "Invalid prefix filter list ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMCRService := &MockMCRService{}
			tt.setupMock(mockMCRService)

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.MCRService = mockMCRService
				return client, nil
			})

			idStr := tt.rawPrefixListID
			if idStr == "" {
				idStr = fmt.Sprintf("%d", tt.prefixListID)
			}

			cmd := &cobra.Command{
				Use:  "get-mcr-prefix-filter-list [mcrUID] [prefixListID]",
				Args: cobra.ExactArgs(2),
				RunE: func(cmd *cobra.Command, args []string) error {
					return GetMCRPrefixFilterList(cmd, args, false, "table")
				},
			}
			var err error
			output := output.CaptureOutput(func() {
				err = cmd.RunE(cmd, []string{tt.mcrUID, idStr})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, output, tt.expectedOutput)
			}
		})
	}
}

func TestDeleteMCRPrefixFilterListCmd_WithMockClient(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	originalPrompt := utils.GetResourcePrompt()
	defer func() {
		utils.SetResourcePrompt(originalPrompt)
	}()

	tests := []struct {
		name            string
		mcrUID          string
		prefixListID    int
		rawPrefixListID string
		force           bool
		promptResponse  string
		setupMock       func(*MockMCRService)
		expectedError   string
		expectedOutput  string
	}{
		{
			name:           "successful delete prefix filter list",
			mcrUID:         "mcr-123",
			prefixListID:   1,
			force:          false,
			promptResponse: "y",
			setupMock: func(m *MockMCRService) {
				m.DeleteMCRPrefixFilterListResult = &megaport.DeleteMCRPrefixFilterListResponse{
					IsDeleted: true,
				}
			},
			expectedOutput: "Prefix filter list deleted successfully - ID: 1",
		},
		{
			name:           "API error",
			mcrUID:         "mcr-123",
			prefixListID:   1,
			force:          false,
			promptResponse: "y",
			setupMock: func(m *MockMCRService) {
				m.DeleteMCRPrefixFilterListErr = fmt.Errorf("API error: service unavailable")
			},
			expectedError: "API error: service unavailable",
		},
		{
			name:           "not successful",
			mcrUID:         "mcr-123",
			prefixListID:   1,
			force:          true,
			promptResponse: "y",
			setupMock: func(m *MockMCRService) {
				m.DeleteMCRPrefixFilterListResult = &megaport.DeleteMCRPrefixFilterListResponse{
					IsDeleted: false,
				}
			},
			expectedError: "not successful for ID 1",
		},
		{
			name:            "invalid prefix filter list ID",
			mcrUID:          "mcr-123",
			rawPrefixListID: "abc",
			force:           true,
			setupMock:       func(m *MockMCRService) {},
			expectedError:   "Invalid prefix filter list ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMCRService := &MockMCRService{}
			tt.setupMock(mockMCRService)

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.MCRService = mockMCRService
				return client, nil
			})

			utils.SetResourcePrompt(func(_, msg string, _ bool) (string, error) {
				return tt.promptResponse, nil
			})

			idStr := tt.rawPrefixListID
			if idStr == "" {
				idStr = fmt.Sprintf("%d", tt.prefixListID)
			}

			cmd := &cobra.Command{
				Use: "delete-mcr-prefix-filter-list [mcrUID] [prefixListID]",
				RunE: func(cmd *cobra.Command, args []string) error {
					return DeleteMCRPrefixFilterList(cmd, args, false)
				},
			}
			cmd.Flags().BoolP("force", "f", false, "Force deletion without confirmation")
			cmd.Flags().Bool("now", false, "Delete prefix filter list immediately instead of at end of billing cycle")
			testutil.SetFlags(t, cmd, map[string]string{
				"force": fmt.Sprintf("%v", tt.force),
				"now":   fmt.Sprintf("%v", false),
			})

			var err error
			output := output.CaptureOutput(func() {
				err = cmd.RunE(cmd, []string{tt.mcrUID, idStr})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, output, tt.expectedOutput)
			}
		})
	}
}

func TestBuyMCR_NoWaitFlag(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	originalBuyMCRFunc := buyMCRFunc
	originalGetMCRFunc := getMCRFunc
	defer func() {
		buyMCRFunc = originalBuyMCRFunc
		getMCRFunc = originalGetMCRFunc
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
			mockMCRService := &MockMCRService{}
			mockMCRService.BuyMCRResult = &megaport.BuyMCRResponse{
				TechnicalServiceUID: "mcr-uid-123",
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.MCRService = mockMCRService
				return client, nil
			})

			var capturedReq *megaport.BuyMCRRequest
			buyMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyMCRRequest) (*megaport.BuyMCRResponse, error) {
				capturedReq = req
				return &megaport.BuyMCRResponse{
					TechnicalServiceUID: "mcr-uid-123",
				}, nil
			}
			getMCRCalls := 0
			getMCRFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.MCR, error) {
				getMCRCalls++
				return &megaport.MCR{UID: mcrUID, ProvisioningStatus: "LIVE"}, nil
			}

			cmd := &cobra.Command{Use: "buy"}
			cmd.Flags().BoolP("interactive", "i", false, "")
			cmd.Flags().Bool("no-wait", false, "")
			cmd.Flags().BoolP("yes", "y", false, "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("term", 0, "")
			cmd.Flags().Int("port-speed", 0, "")
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().Int("mcr-asn", 0, "")
			cmd.Flags().String("diversity-zone", "", "")
			cmd.Flags().String("cost-centre", "", "")
			cmd.Flags().String("promo-code", "", "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")

			testutil.SetFlags(t, cmd, map[string]string{
				"name":        "Test MCR",
				"term":        "12",
				"port-speed":  "10000",
				"location-id": "123",
				"mcr-asn":     "65000",
			})
			if tt.noWait {
				assert.NoError(t, cmd.Flags().Set("no-wait", "true"))
			}

			var err error
			output.CaptureOutput(func() {
				err = BuyMCR(cmd, nil, true)
			})

			assert.NoError(t, err)
			assert.NotNil(t, capturedReq)
			// The SDK must never poll; provisioning is awaited outside the order retry.
			assert.False(t, capturedReq.WaitForProvision)
			if tt.expectPoll {
				assert.Positive(t, getMCRCalls, "expected a provisioning poll when not using --no-wait")
			} else {
				assert.Zero(t, getMCRCalls, "no provisioning poll expected with --no-wait")
			}
		})
	}
}

// TestBuyMCR_NoResubmitOnPollRetryableError proves the order is submitted exactly
// once even when a retryable error (429) surfaces during the provisioning poll.
func TestBuyMCR_NoResubmitOnPollRetryableError(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	originalBuyMCRFunc := buyMCRFunc
	originalGetMCRFunc := getMCRFunc
	originalMaxRetries := utils.MaxRetries
	originalPollInterval := utils.ProvisionPollInterval
	originalNoRetry := utils.NoRetry
	defer func() {
		cleanup()
		buyMCRFunc = originalBuyMCRFunc
		getMCRFunc = originalGetMCRFunc
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

	mockMCRService := &MockMCRService{}
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.MCRService = mockMCRService
		return client, nil
	})

	buyCalls := 0
	buyMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyMCRRequest) (*megaport.BuyMCRResponse, error) {
		buyCalls++
		return &megaport.BuyMCRResponse{TechnicalServiceUID: "mcr-uid-123"}, nil
	}
	getMCRFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.MCR, error) {
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
	cmd.Flags().BoolP("interactive", "i", false, "")
	cmd.Flags().Bool("no-wait", false, "")
	cmd.Flags().BoolP("yes", "y", false, "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().Int("term", 0, "")
	cmd.Flags().Int("port-speed", 0, "")
	cmd.Flags().Int("location-id", 0, "")
	cmd.Flags().Int("mcr-asn", 0, "")
	cmd.Flags().String("diversity-zone", "", "")
	cmd.Flags().String("cost-centre", "", "")
	cmd.Flags().String("promo-code", "", "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")

	testutil.SetFlags(t, cmd, map[string]string{
		"name":        "Test MCR",
		"term":        "12",
		"port-speed":  "10000",
		"location-id": "123",
		"mcr-asn":     "65000",
	})
	// no-wait stays false: the provisioning poll must run.

	var err error
	output.CaptureOutput(func() {
		err = BuyMCR(cmd, nil, true)
	})

	assert.Error(t, err, "a retryable error during the provisioning poll must surface as an error")
	assert.Equal(t, 1, buyCalls, "the order must be submitted exactly once; a poll-phase 429 must not re-submit it")
}

func TestBuyMCRCmd_WithMockClient(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	originalBuyMCRFunc := buyMCRFunc
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	defer func() {
		utils.SetResourcePrompt(originalPrompt)
		buyMCRFunc = originalBuyMCRFunc
	}()
	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	tests := []struct {
		name             string
		args             []string
		interactive      bool
		prompts          []string
		flags            map[string]string
		jsonInput        string
		jsonFilePath     string
		setupMock        func(*MockMCRService)
		expectedError    string
		expectedExitCode int
		expectedOutput   string
	}{
		{
			name:        "interactive mode success",
			interactive: true,
			prompts: []string{
				"Test MCR",
				"12",
				"10000",
				"123",
				"true",
				"65000",
				"red",
				"cost-123",
				"MCRPROMO2025",
			},
			setupMock: func(m *MockMCRService) {
				m.BuyMCRResult = &megaport.BuyMCRResponse{
					TechnicalServiceUID: "mcr-123-abc",
				}
			},
			expectedOutput: "MCR created",
		},
		{
			name: "flag mode success",
			flags: map[string]string{
				"name":                   "Flag MCR",
				"term":                   "12",
				"port-speed":             "10000",
				"location-id":            "123",
				"mcr-asn":                "65000",
				"diversity-zone":         "blue",
				"cost-centre":            "cost-456",
				"promo-code":             "FLAGPROMO",
				"marketplace-visibility": "true",
			},
			setupMock: func(m *MockMCRService) {
				m.BuyMCRResult = &megaport.BuyMCRResponse{
					TechnicalServiceUID: "mcr-456-def",
				}
			},
			expectedOutput: "MCR created",
		},
		{
			name: "JSON string mode success",
			flags: map[string]string{
				"json": `{"name":"JSON MCR","term":24,"portSpeed":10000,"locationId":123,"mcrAsn":65000,"diversityZone":"green","costCentre":"cost-789","promoCode":"JSONPROMO","marketplaceVisibility":true}`,
				"yes":  "true",
			},
			setupMock: func(m *MockMCRService) {
				m.BuyMCRResult = &megaport.BuyMCRResponse{
					TechnicalServiceUID: "mcr-789-ghi",
				}
			},
			expectedOutput: "MCR created",
		},
		{
			name: "missing required fields in flag mode",
			flags: map[string]string{
				"name": "Test MCR",
			},
			expectedError: "Invalid contract term: 0 - must be one of: [1 12 24 36 48 60]",
		},
		{
			name: "invalid term in flag mode",
			flags: map[string]string{
				"name":        "Test MCR",
				"term":        "13",
				"port-speed":  "10000",
				"location-id": "123",
				"mcr-asn":     "65000",
			},
			expectedError: "Invalid contract term: 13 - must be one of: [1 12 24 36 48 60]",
		},
		{
			name: "invalid JSON",
			flags: map[string]string{
				"json": `{"name":"Test MCR","term":"invalid"}`,
			},
			expectedError: "failed to parse JSON",
		},
		{
			name: "API error",
			flags: map[string]string{
				"name":        "Test MCR",
				"term":        "12",
				"port-speed":  "10000",
				"location-id": "123",
				"mcr-asn":     "65000",
			},
			setupMock: func(m *MockMCRService) {
				m.BuyMCRErr = fmt.Errorf("API error: service unavailable")
			},
			expectedError: "API error: service unavailable",
		},
		{
			name: "nil response from API",
			flags: map[string]string{
				"name":        "Test MCR",
				"term":        "12",
				"port-speed":  "10000",
				"location-id": "123",
				"mcr-asn":     "65000",
			},
			setupMock: func(m *MockMCRService) {
				// BuyMCRResult left nil → mock returns (nil, nil)
			},
			expectedError: "empty response from API",
		},
		{
			name:          "no input provided",
			expectedError: "no input provided",
		},
		{
			name:        "interactive combined with JSON is a usage error",
			interactive: true,
			flags: map[string]string{
				"json": `{"name":"JSON MCR","term":24,"portSpeed":10000,"locationId":123,"mcrAsn":65000,"diversityZone":"green","costCentre":"cost-789","promoCode":"JSONPROMO"}`,
			},
			setupMock:     func(m *MockMCRService) {},
			expectedError: "cannot be combined with",
		},
		{
			name:        "interactive combined with flags is a usage error",
			interactive: true,
			flags: map[string]string{
				"name": "Test MCR",
			},
			setupMock:     func(m *MockMCRService) {},
			expectedError: "cannot be combined with",
		},
		{
			name: "JSON mode with negative ipsec-tunnel-count flag is a usage error",
			flags: map[string]string{
				"json":               `{"name":"JSON MCR","term":24,"portSpeed":10000,"locationId":123,"mcrAsn":65000}`,
				"ipsec-tunnel-count": "-1",
			},
			setupMock:        func(m *MockMCRService) {},
			expectedError:    "ipsec-tunnel-count must be",
			expectedExitCode: exitcodes.Usage,
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

			mockMCRService := &MockMCRService{}
			if tt.setupMock != nil {
				tt.setupMock(mockMCRService)
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.MCRService = mockMCRService
				return client, nil
			})

			cmd := &cobra.Command{
				Use:  "buy",
				RunE: testutil.NoColorAdapter(BuyMCR),
			}

			cmd.Flags().BoolP("interactive", "i", false, "Use interactive mode with prompts")
			cmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
			cmd.Flags().String("name", "", "MCR name")
			cmd.Flags().Int("term", 0, "Contract term in months (1, 12, 24, or 36)")
			cmd.Flags().Int("port-speed", 0, "Port speed in Mbps")
			cmd.Flags().Int("location-id", 0, "Location ID where the MCR will be provisioned")
			cmd.Flags().Int("mcr-asn", 0, "ASN for the MCR")
			cmd.Flags().String("diversity-zone", "", "Diversity zone for the MCR")
			cmd.Flags().String("cost-centre", "", "Cost centre for billing")
			cmd.Flags().String("promo-code", "", "Promotional code for discounts")
			cmd.Flags().Bool("marketplace-visibility", false, "Whether the MCR is visible in the marketplace")
			cmd.Flags().String("json", "", "JSON string containing MCR configuration")
			cmd.Flags().String("json-file", "", "Path to JSON file containing MCR configuration")
			cmd.Flags().Int("ipsec-tunnel-count", 0, "Number of IPsec tunnels to provision")

			flags := make(map[string]string)
			if tt.interactive {
				flags["interactive"] = "true"
			}
			for k, v := range tt.flags {
				flags[k] = v
			}
			testutil.SetFlags(t, cmd, flags)

			var err error
			output := captureStderr(t, func() {
				err = cmd.RunE(cmd, tt.args)
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)

				if tt.expectedExitCode != 0 {
					var cliErr *exitcodes.CLIError
					require.True(t, errors.As(err, &cliErr))
					assert.Equal(t, tt.expectedExitCode, cliErr.Code)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, output, tt.expectedOutput)

				if tt.expectedOutput != "" && mockMCRService != nil && mockMCRService.CapturedBuyMCRRequest != nil {
					req := mockMCRService.CapturedBuyMCRRequest

					if tt.flags != nil && tt.flags["json"] != "" {
						assert.Equal(t, "JSON MCR", req.Name)
						assert.Equal(t, 24, req.Term)
						assert.Equal(t, 10000, req.PortSpeed)
						assert.Equal(t, 123, req.LocationID)
						assert.Equal(t, 65000, req.MCRAsn)
						assert.Equal(t, "green", req.DiversityZone)
						assert.Equal(t, "cost-789", req.CostCentre)
						assert.Equal(t, "JSONPROMO", req.PromoCode)
						require.NotNil(t, req.MarketplaceVisibility)
						assert.True(t, *req.MarketplaceVisibility)
					} else if tt.flags != nil {
						assert.Equal(t, "Flag MCR", req.Name)
						assert.Equal(t, 12, req.Term)
						assert.Equal(t, 10000, req.PortSpeed)
						assert.Equal(t, 123, req.LocationID)
						assert.Equal(t, 65000, req.MCRAsn)
						assert.Equal(t, "blue", req.DiversityZone)
						assert.Equal(t, "cost-456", req.CostCentre)
						assert.Equal(t, "FLAGPROMO", req.PromoCode)
						require.NotNil(t, req.MarketplaceVisibility)
						assert.True(t, *req.MarketplaceVisibility)
					} else if len(tt.prompts) > 0 {
						assert.Equal(t, "Test MCR", req.Name)
						assert.Equal(t, 12, req.Term)
						assert.Equal(t, 10000, req.PortSpeed)
						assert.Equal(t, 123, req.LocationID)
						assert.Equal(t, 65000, req.MCRAsn)
						assert.Equal(t, "red", req.DiversityZone)
						assert.Equal(t, "cost-123", req.CostCentre)
						assert.Equal(t, "MCRPROMO2025", req.PromoCode)
						require.NotNil(t, req.MarketplaceVisibility)
						assert.True(t, *req.MarketplaceVisibility)
					}
				}
			}
		})
	}
}

func TestGetMCRStatus(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name           string
		mcrUID         string
		setupMock      func(*MockMCRService)
		expectedError  string
		expectedOutput string
		outputFormat   string
	}{
		{
			name:   "successful status retrieval - table format",
			mcrUID: "mcr-123abc",
			setupMock: func(m *MockMCRService) {
				m.GetMCRResult = &megaport.MCR{
					UID:                "mcr-123abc",
					Name:               "Test MCR",
					ProvisioningStatus: "CONFIGURED",
					PortSpeed:          10000,
					Resources: megaport.MCRResources{
						VirtualRouter: megaport.MCRVirtualRouter{
							ASN: 64512,
						},
					},
				}
			},
			expectedOutput: "mcr-123abc",
			outputFormat:   "table",
		},
		{
			name:   "successful status retrieval - json format",
			mcrUID: "mcr-123abc",
			setupMock: func(m *MockMCRService) {
				m.GetMCRResult = &megaport.MCR{
					UID:                "mcr-123abc",
					Name:               "Test MCR",
					ProvisioningStatus: "LIVE",
					PortSpeed:          1000,
					Resources: megaport.MCRResources{
						VirtualRouter: megaport.MCRVirtualRouter{
							ASN: 64513,
						},
					},
				}
			},
			expectedOutput: "mcr-123abc",
			outputFormat:   "json",
		},
		{
			name:   "MCR not found",
			mcrUID: "mcr-notfound",
			setupMock: func(m *MockMCRService) {
				m.GetMCRErr = fmt.Errorf("MCR not found")
			},
			expectedError: "failed to get MCR status",
			outputFormat:  "table",
		},
		{
			name:   "API error",
			mcrUID: "mcr-error",
			setupMock: func(m *MockMCRService) {
				m.GetMCRErr = fmt.Errorf("API error")
			},
			expectedError: "API error",
			outputFormat:  "table",
		},
		{
			name:   "nil MCR returned without error",
			mcrUID: "mcr-nil",
			setupMock: func(m *MockMCRService) {
				m.ForceNilGetMCR = true
			},
			expectedError: "no MCR found",
			outputFormat:  "table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockMCRService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.MCRService = mockService
				return client, nil
			})

			cmd := &cobra.Command{
				Use: "status [mcrUID]",
			}

			var err error
			capturedOutput := output.CaptureOutput(func() {
				err = GetMCRStatus(cmd, []string{tt.mcrUID}, true, tt.outputFormat)
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
					assert.Contains(t, capturedOutput, "\"asn\":")
				case "table":
					assert.Contains(t, capturedOutput, "UID")
					assert.Contains(t, capturedOutput, "NAME")
					assert.Contains(t, capturedOutput, "STATUS")
					assert.Contains(t, capturedOutput, "ASN")
				}
			}
		})
	}
}

func TestListMCRsCmd_WithMockClient(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	// Sample MCRs for testing
	testMCRs := []*megaport.MCR{
		{
			UID:                "mcr-123",
			Name:               "mcr-demo-01",
			LocationID:         571,
			ProvisioningStatus: "LIVE",
			PortSpeed:          1000,
			Resources: megaport.MCRResources{
				VirtualRouter: megaport.MCRVirtualRouter{
					ASN: 133937,
				},
			},
		},
		{
			UID:                "mcr-456",
			Name:               "mcr-demo0-01",
			LocationID:         558,
			ProvisioningStatus: "LIVE",
			PortSpeed:          1000,
			Resources: megaport.MCRResources{
				VirtualRouter: megaport.MCRVirtualRouter{
					ASN: 133937,
				},
			},
		},
		{
			UID:                "mcr-789",
			Name:               "production-mcr",
			LocationID:         64,
			ProvisioningStatus: "LIVE",
			PortSpeed:          2500,
			Resources: megaport.MCRResources{
				VirtualRouter: megaport.MCRVirtualRouter{
					ASN: 133937,
				},
			},
		},
		{
			UID:                "mcr-abc",
			Name:               "test-mcr-sydney",
			LocationID:         571,
			ProvisioningStatus: "DECOMMISSIONED",
			PortSpeed:          5000,
			Resources: megaport.MCRResources{
				VirtualRouter: megaport.MCRVirtualRouter{
					ASN: 64512,
				},
			},
		},
	}

	tests := []struct {
		name           string
		flags          map[string]string
		setupMock      func(*MockMCRService)
		expectedError  string
		expectedMCRs   []string // Names of MCRs that should be in the output
		unexpectedMCRs []string // Names of MCRs that should NOT be in the output
		outputFormat   string
	}{
		{
			name: "list all active MCRs",
			setupMock: func(m *MockMCRService) {
				m.ListMCRsResult = testMCRs
			},
			expectedMCRs:   []string{"mcr-demo-01", "mcr-demo0-01", "production-mcr"},
			unexpectedMCRs: []string{"test-mcr-sydney"}, // Should be excluded due to DECOMMISSIONED status
			outputFormat:   "table",
		},
		{
			name: "filter by exact name match",
			flags: map[string]string{
				"name": "mcr-demo-01",
			},
			setupMock: func(m *MockMCRService) {
				m.ListMCRsResult = testMCRs
			},
			expectedMCRs:   []string{"mcr-demo-01"},
			unexpectedMCRs: []string{"mcr-demo0-01", "production-mcr", "test-mcr-sydney"},
			outputFormat:   "table",
		},
		{
			name: "filter by partial name match",
			flags: map[string]string{
				"name": "demo",
			},
			setupMock: func(m *MockMCRService) {
				m.ListMCRsResult = testMCRs
			},
			expectedMCRs:   []string{"mcr-demo-01", "mcr-demo0-01"},
			unexpectedMCRs: []string{"production-mcr", "test-mcr-sydney"},
			outputFormat:   "table",
		},
		{
			name: "filter by case insensitive name",
			flags: map[string]string{
				"name": "MCR-DEMO-01",
			},
			setupMock: func(m *MockMCRService) {
				m.ListMCRsResult = testMCRs
			},
			expectedMCRs:   []string{"mcr-demo-01"},
			unexpectedMCRs: []string{"mcr-demo0-01", "production-mcr", "test-mcr-sydney"},
			outputFormat:   "table",
		},
		{
			name: "filter by location ID",
			flags: map[string]string{
				"location-id": "571",
			},
			setupMock: func(m *MockMCRService) {
				m.ListMCRsResult = testMCRs
			},
			expectedMCRs:   []string{"mcr-demo-01"}, // Only active MCR at location 571
			unexpectedMCRs: []string{"mcr-demo0-01", "production-mcr", "test-mcr-sydney"},
			outputFormat:   "table",
		},
		{
			name: "filter by port speed",
			flags: map[string]string{
				"port-speed": "2500",
			},
			setupMock: func(m *MockMCRService) {
				m.ListMCRsResult = testMCRs
			},
			expectedMCRs:   []string{"production-mcr"},
			unexpectedMCRs: []string{"mcr-demo-01", "mcr-demo0-01", "test-mcr-sydney"},
			outputFormat:   "table",
		},
		{
			name: "filter by name and location combined",
			flags: map[string]string{
				"name":        "demo",
				"location-id": "571",
			},
			setupMock: func(m *MockMCRService) {
				m.ListMCRsResult = testMCRs
			},
			expectedMCRs:   []string{"mcr-demo-01"},
			unexpectedMCRs: []string{"mcr-demo0-01", "production-mcr", "test-mcr-sydney"},
			outputFormat:   "table",
		},
		{
			name: "include inactive MCRs",
			flags: map[string]string{
				"include-inactive": "true",
			},
			setupMock: func(m *MockMCRService) {
				m.ListMCRsResult = testMCRs
			},
			expectedMCRs:   []string{"mcr-demo-01", "mcr-demo0-01", "production-mcr", "test-mcr-sydney"},
			unexpectedMCRs: []string{},
			outputFormat:   "table",
		},
		{
			name: "filter with no matches",
			flags: map[string]string{
				"name": "nonexistent-mcr",
			},
			setupMock: func(m *MockMCRService) {
				m.ListMCRsResult = testMCRs
			},
			expectedMCRs:   []string{},
			unexpectedMCRs: []string{"mcr-demo-01", "mcr-demo0-01", "production-mcr", "test-mcr-sydney"},
			outputFormat:   "table",
		},
		{
			name: "JSON output format",
			flags: map[string]string{
				"name": "mcr-demo-01",
			},
			setupMock: func(m *MockMCRService) {
				m.ListMCRsResult = testMCRs
			},
			expectedMCRs:   []string{"mcr-demo-01"},
			unexpectedMCRs: []string{"mcr-demo0-01", "production-mcr"},
			outputFormat:   "json",
		},
		{
			name: "API error",
			setupMock: func(m *MockMCRService) {
				m.ListMCRsErr = fmt.Errorf("API error: service unavailable")
			},
			expectedError: "API error: service unavailable",
			outputFormat:  "table",
		},
		{
			name: "limit results",
			flags: map[string]string{
				"limit": "2",
			},
			setupMock: func(m *MockMCRService) {
				m.ListMCRsResult = testMCRs
			},
			expectedMCRs:   []string{"mcr-demo-01", "mcr-demo0-01"},
			unexpectedMCRs: []string{"production-mcr"},
			outputFormat:   "table",
		},
		{
			name: "negative limit returns error",
			flags: map[string]string{
				"limit": "-1",
			},
			setupMock: func(m *MockMCRService) {
				m.ListMCRsResult = testMCRs
			},
			expectedError: "--limit must be a non-negative integer",
			outputFormat:  "table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMCRService := &MockMCRService{}
			if tt.setupMock != nil {
				tt.setupMock(mockMCRService)
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.MCRService = mockMCRService
				return client, nil
			})

			cmd := &cobra.Command{
				Use: "list",
				RunE: func(cmd *cobra.Command, args []string) error {
					return ListMCRs(cmd, args, true, tt.outputFormat) // noColor = true for testing
				},
			}

			// Add flags
			cmd.Flags().String("name", "", "Filter MCRs by name")
			cmd.Flags().Int("location-id", 0, "Filter MCRs by location ID")
			cmd.Flags().Int("port-speed", 0, "Filter MCRs by port speed")
			cmd.Flags().Bool("include-inactive", false, "Include inactive MCRs")
			cmd.Flags().Int("limit", 0, "Maximum number of results to display")

			// Set flag values from test case
			testutil.SetFlags(t, cmd, tt.flags)

			var err error
			var capturedOutput string
			capturedStderr := captureStderr(t, func() {
				capturedOutput = output.CaptureStdout(func() {
					err = cmd.RunE(cmd, []string{})
				})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)

				// Check that expected MCRs are in the output
				for _, expectedMCR := range tt.expectedMCRs {
					assert.Contains(t, capturedOutput, expectedMCR,
						"Expected MCR '%s' should be in output", expectedMCR)
				}

				// Check that unexpected MCRs are NOT in the output
				for _, unexpectedMCR := range tt.unexpectedMCRs {
					assert.NotContains(t, capturedOutput, unexpectedMCR,
						"Unexpected MCR '%s' should NOT be in output", unexpectedMCR)
				}

				// Verify output format
				switch tt.outputFormat {
				case "json":
					if len(tt.expectedMCRs) > 0 {
						assert.Contains(t, capturedOutput, "\"uid\":")
						assert.Contains(t, capturedOutput, "\"name\":")
					}
				case "table":
					if len(tt.expectedMCRs) > 0 {
						assert.Contains(t, capturedOutput, "UID")
						assert.Contains(t, capturedOutput, "NAME")
					}
				}

				// Check warning message when no results found
				if len(tt.expectedMCRs) == 0 && tt.expectedError == "" {
					assert.Contains(t, capturedStderr, "No MCRs found. Create one with 'megaport mcr buy'.")
				}
			}

			// Verify that the correct request was passed to the mock
			if mockMCRService.CapturedListMCRsRequest != nil {
				if tt.flags["include-inactive"] == "true" {
					assert.True(t, mockMCRService.CapturedListMCRsRequest.IncludeInactive)
				} else {
					assert.False(t, mockMCRService.CapturedListMCRsRequest.IncludeInactive)
				}
			}
		})
	}
}

func TestFilterMCRsFunction(t *testing.T) {
	testMCRs := []*megaport.MCR{
		{
			UID:        "mcr-123",
			Name:       "mcr-demo-01",
			LocationID: 571,
			PortSpeed:  1000,
		},
		{
			UID:        "mcr-456",
			Name:       "mcr-demo0-01",
			LocationID: 558,
			PortSpeed:  1000,
		},
		{
			UID:        "mcr-789",
			Name:       "Production-MCR",
			LocationID: 64,
			PortSpeed:  2500,
		},
		{
			UID:        "mcr-abc",
			Name:       "test-mcr-sydney",
			LocationID: 571,
			PortSpeed:  5000,
		},
	}

	tests := []struct {
		name         string
		locationID   int
		portSpeed    int
		mcrName      string
		expectedUIDs []string
	}{
		{
			name:         "no filters - return all",
			expectedUIDs: []string{"mcr-123", "mcr-456", "mcr-789", "mcr-abc"},
		},
		{
			name:         "filter by exact name",
			mcrName:      "mcr-demo-01",
			expectedUIDs: []string{"mcr-123"},
		},
		{
			name:         "filter by partial name",
			mcrName:      "demo",
			expectedUIDs: []string{"mcr-123", "mcr-456"},
		},
		{
			name:         "filter by case insensitive name",
			mcrName:      "PRODUCTION",
			expectedUIDs: []string{"mcr-789"},
		},
		{
			name:         "filter by location ID",
			locationID:   571,
			expectedUIDs: []string{"mcr-123", "mcr-abc"},
		},
		{
			name:         "filter by port speed",
			portSpeed:    1000,
			expectedUIDs: []string{"mcr-123", "mcr-456"},
		},
		{
			name:         "filter by name and location",
			mcrName:      "demo",
			locationID:   571,
			expectedUIDs: []string{"mcr-123"},
		},
		{
			name:         "filter by all parameters",
			mcrName:      "mcr-demo-01",
			locationID:   571,
			portSpeed:    1000,
			expectedUIDs: []string{"mcr-123"},
		},
		{
			name:         "no matches",
			mcrName:      "nonexistent",
			expectedUIDs: []string{},
		},
		{
			name:         "empty name filter - return all",
			mcrName:      "",
			expectedUIDs: []string{"mcr-123", "mcr-456", "mcr-789", "mcr-abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterMCRs(testMCRs, tt.locationID, tt.portSpeed, tt.mcrName)

			// Check that we got the expected number of results
			assert.Equal(t, len(tt.expectedUIDs), len(result),
				"Expected %d MCRs, got %d", len(tt.expectedUIDs), len(result))

			// Check that all expected UIDs are present
			resultUIDs := make([]string, len(result))
			for i, mcr := range result {
				resultUIDs[i] = mcr.UID
			}

			for _, expectedUID := range tt.expectedUIDs {
				assert.Contains(t, resultUIDs, expectedUID,
					"Expected UID '%s' should be in filtered results", expectedUID)
			}
		})
	}
}

func TestFilterMCRs_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		mcrs        []*megaport.MCR
		locationID  int
		portSpeed   int
		mcrName     string
		expectedLen int
	}{
		{
			name:        "nil slice",
			mcrs:        nil,
			expectedLen: 0,
		},
		{
			name:        "empty slice",
			mcrs:        []*megaport.MCR{},
			expectedLen: 0,
		},
		{
			name: "slice with nil elements",
			mcrs: []*megaport.MCR{
				{UID: "mcr-123", Name: "test-mcr"},
				nil,
				{UID: "mcr-456", Name: "another-mcr"},
			},
			expectedLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterMCRs(tt.mcrs, tt.locationID, tt.portSpeed, tt.mcrName)
			assert.Equal(t, tt.expectedLen, len(result))
		})
	}
}

func TestListMCRResourceTagsCmd(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name           string
		mcrUID         string
		setupMock      func(*MockMCRService)
		expectedError  string
		expectedOutput string
	}{
		{
			name:   "successful list",
			mcrUID: "mcr-123",
			setupMock: func(m *MockMCRService) {
				m.ListMCRResourceTagsResult = map[string]string{
					"environment": "production",
					"team":        "networking",
				}
			},
			expectedOutput: "environment",
		},
		{
			name:   "empty tags",
			mcrUID: "mcr-empty",
			setupMock: func(m *MockMCRService) {
				m.ListMCRResourceTagsResult = map[string]string{}
			},
		},
		{
			name:   "API error",
			mcrUID: "mcr-error",
			setupMock: func(m *MockMCRService) {
				m.ListMCRResourceTagsErr = fmt.Errorf("API error: not found")
			},
			expectedError: "failed to get resource tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockMCRService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.MCRService = mockService
				return client, nil
			})

			cmd := &cobra.Command{
				Use: "list-tags [mcrUID]",
				RunE: func(cmd *cobra.Command, args []string) error {
					return ListMCRResourceTags(cmd, args, false, "table")
				},
			}

			var err error
			capturedOutput := output.CaptureOutput(func() {
				err = cmd.RunE(cmd, []string{tt.mcrUID})
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

func TestUpdateMCRResourceTagsCmd(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	originalResourcePrompt := utils.GetUpdateResourceTagsPrompt()
	defer func() {
		utils.SetUpdateResourceTagsPrompt(originalResourcePrompt)
	}()

	tests := []struct {
		name                 string
		mcrUID               string
		interactive          bool
		force                bool
		promptResult         map[string]string
		promptError          error
		jsonInput            string
		setupMock            func(*MockMCRService)
		expectedError        string
		expectedOutput       string
		expectedCapturedTags map[string]string
	}{
		{
			name:        "successful update with interactive mode",
			mcrUID:      "mcr-123",
			interactive: true,
			promptResult: map[string]string{
				"environment": "production",
			},
			setupMock: func(m *MockMCRService) {
				m.ListMCRResourceTagsResult = map[string]string{"environment": "staging"}
			},
			expectedOutput:       "Resource tags updated for MCR mcr-123",
			expectedCapturedTags: map[string]string{"environment": "production"},
		},
		{
			name:      "successful update with json",
			mcrUID:    "mcr-456",
			jsonInput: `{"environment": "production", "team": "networking"}`,
			setupMock: func(m *MockMCRService) {
				m.ListMCRResourceTagsResult = map[string]string{}
			},
			expectedOutput: "Resource tags updated for MCR mcr-456",
			expectedCapturedTags: map[string]string{
				"environment": "production",
				"team":        "networking",
			},
		},
		{
			name:      "error with invalid json",
			mcrUID:    "mcr-789",
			jsonInput: `{invalid json}`,
			setupMock: func(m *MockMCRService) {
				m.ListMCRResourceTagsResult = map[string]string{}
			},
			expectedError: "failed to parse JSON",
		},
		{
			name:      "error with API tag listing",
			mcrUID:    "mcr-list-error",
			jsonInput: `{"environment": "production"}`,
			setupMock: func(m *MockMCRService) {
				m.ListMCRResourceTagsErr = fmt.Errorf("API error: resource not found")
			},
			expectedError: "failed to log in or list existing resource tags",
		},
		{
			name:      "error with API update",
			mcrUID:    "mcr-update-error",
			jsonInput: `{"environment": "production"}`,
			setupMock: func(m *MockMCRService) {
				m.ListMCRResourceTagsResult = map[string]string{}
				m.UpdateMCRResourceTagsErr = fmt.Errorf("API error: unauthorized")
			},
			expectedError: "failed to update resource tags",
		},
		{
			name:      "empty tags clear all existing tags",
			mcrUID:    "mcr-clear",
			force:     true,
			jsonInput: `{}`,
			setupMock: func(m *MockMCRService) {
				m.ListMCRResourceTagsResult = map[string]string{"env": "staging"}
			},
			expectedOutput:       "Resource tags updated for MCR mcr-clear",
			expectedCapturedTags: map[string]string{},
		},
		{
			name:   "no input provided",
			mcrUID: "mcr-no-input",
			setupMock: func(m *MockMCRService) {
				m.ListMCRResourceTagsResult = map[string]string{}
			},
			expectedError: "no input provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockMCRService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.MCRService = mockService
				return client, nil
			})

			utils.SetUpdateResourceTagsPrompt(func(existingTags map[string]string, noColor bool) (map[string]string, error) {
				return tt.promptResult, tt.promptError
			})

			cmd := &cobra.Command{
				Use: "update-tags [mcrUID]",
				RunE: func(cmd *cobra.Command, args []string) error {
					return UpdateMCRResourceTags(cmd, args, false)
				},
			}

			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().Bool("force", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")

			if tt.interactive {
				err := cmd.Flags().Set("interactive", "true")
				assert.NoError(t, err)
			}
			if tt.force {
				err := cmd.Flags().Set("force", "true")
				assert.NoError(t, err)
			}
			if tt.jsonInput != "" {
				err := cmd.Flags().Set("json", tt.jsonInput)
				assert.NoError(t, err)
			}

			var err error
			capturedOutput := captureStderr(t, func() {
				err = cmd.RunE(cmd, []string{tt.mcrUID})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.expectedOutput != "" {
					assert.Contains(t, capturedOutput, tt.expectedOutput)
				}
				if tt.expectedCapturedTags != nil {
					assert.Equal(t, tt.expectedCapturedTags, mockService.CapturedUpdateMCRResourceTagsRequest)
				}
			}
		})
	}
}

func TestUpdateMCR(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	originalGetMCRFunc := getMCRFunc
	originalUpdateMCRFunc := updateMCRFunc
	defer func() {
		getMCRFunc = originalGetMCRFunc
		updateMCRFunc = originalUpdateMCRFunc
	}()

	tests := []struct {
		name           string
		args           []string
		flags          map[string]string
		setupLogin     func()
		setupGetMCR    func()
		setupUpdateMCR func()
		expectedError  string
		expectedOutput string
	}{
		{
			name: "success with flags",
			args: []string{"mcr-123"},
			flags: map[string]string{
				"name": "Updated MCR",
			},
			setupLogin: func() {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.MCRService = &MockMCRService{}
					return client, nil
				})
			},
			setupGetMCR: func() {
				getMCRFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.MCR, error) {
					return &megaport.MCR{
						UID:                mcrUID,
						Name:               "Original MCR",
						ProvisioningStatus: "LIVE",
					}, nil
				}
			},
			setupUpdateMCR: func() {
				updateMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ModifyMCRRequest) (*megaport.ModifyMCRResponse, error) {
					return &megaport.ModifyMCRResponse{IsUpdated: true}, nil
				}
			},
			expectedOutput: "MCR updated mcr-123",
		},
		{
			name: "success with term flag only",
			args: []string{"mcr-789"},
			flags: map[string]string{
				"term": "24",
			},
			setupLogin: func() {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.MCRService = &MockMCRService{}
					return client, nil
				})
			},
			setupGetMCR: func() {
				getMCRFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.MCR, error) {
					return &megaport.MCR{
						UID:                mcrUID,
						Name:               "Original MCR",
						ProvisioningStatus: "LIVE",
					}, nil
				}
			},
			setupUpdateMCR: func() {
				updateMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ModifyMCRRequest) (*megaport.ModifyMCRResponse, error) {
					if req.ContractTermMonths == nil {
						return nil, fmt.Errorf("expected ContractTermMonths to be set")
					}
					if *req.ContractTermMonths != 24 {
						return nil, fmt.Errorf("expected ContractTermMonths=24, got %d", *req.ContractTermMonths)
					}
					return &megaport.ModifyMCRResponse{IsUpdated: true}, nil
				}
			},
			expectedOutput: "MCR updated mcr-789",
		},
		{
			name: "success with mcr-asn flag only",
			args: []string{"mcr-789"},
			flags: map[string]string{
				"mcr-asn": "65020",
			},
			setupLogin: func() {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.MCRService = &MockMCRService{}
					return client, nil
				})
			},
			setupGetMCR: func() {
				getMCRFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.MCR, error) {
					return &megaport.MCR{
						UID:                mcrUID,
						Name:               "Original MCR",
						ProvisioningStatus: "LIVE",
					}, nil
				}
			},
			setupUpdateMCR: func() {
				updateMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ModifyMCRRequest) (*megaport.ModifyMCRResponse, error) {
					if req.MCRAsn == nil {
						return nil, fmt.Errorf("expected MCRAsn to be set")
					}
					if *req.MCRAsn != 65020 {
						return nil, fmt.Errorf("expected MCRAsn=65020, got %d", *req.MCRAsn)
					}
					return &megaport.ModifyMCRResponse{IsUpdated: true}, nil
				}
			},
			expectedOutput: "MCR updated mcr-789",
		},
		{
			name: "success with JSON",
			args: []string{"mcr-456"},
			flags: map[string]string{
				"json": `{"name":"JSON Updated MCR"}`,
			},
			setupLogin: func() {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.MCRService = &MockMCRService{}
					return client, nil
				})
			},
			setupGetMCR: func() {
				getMCRFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.MCR, error) {
					return &megaport.MCR{
						UID:                mcrUID,
						Name:               "JSON Updated MCR",
						ProvisioningStatus: "LIVE",
					}, nil
				}
			},
			setupUpdateMCR: func() {
				updateMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ModifyMCRRequest) (*megaport.ModifyMCRResponse, error) {
					return &megaport.ModifyMCRResponse{IsUpdated: true}, nil
				}
			},
			expectedOutput: "MCR updated mcr-456",
		},
		{
			name: "login error",
			args: []string{"mcr-123"},
			flags: map[string]string{
				"name": "Updated MCR",
			},
			setupLogin: func() {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					return nil, fmt.Errorf("authentication failed")
				})
			},
			expectedError: "authentication failed",
		},
		{
			name: "get original MCR error",
			args: []string{"mcr-123"},
			flags: map[string]string{
				"name": "Updated MCR",
			},
			setupLogin: func() {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.MCRService = &MockMCRService{}
					return client, nil
				})
			},
			setupGetMCR: func() {
				getMCRFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.MCR, error) {
					return nil, fmt.Errorf("MCR not found")
				}
			},
			expectedError: "MCR not found",
		},
		{
			name: "update API error",
			args: []string{"mcr-123"},
			flags: map[string]string{
				"name": "Updated MCR",
			},
			setupLogin: func() {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.MCRService = &MockMCRService{}
					return client, nil
				})
			},
			setupGetMCR: func() {
				getMCRFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.MCR, error) {
					return &megaport.MCR{
						UID:                mcrUID,
						Name:               "Original MCR",
						ProvisioningStatus: "LIVE",
					}, nil
				}
			},
			setupUpdateMCR: func() {
				updateMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ModifyMCRRequest) (*megaport.ModifyMCRResponse, error) {
					return nil, fmt.Errorf("API error: service unavailable")
				}
			},
			expectedError: "API error: service unavailable",
		},
		{
			name: "nil response from API",
			args: []string{"mcr-nil"},
			flags: map[string]string{
				"name": "Updated MCR",
			},
			setupLogin: func() {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.MCRService = &MockMCRService{}
					return client, nil
				})
			},
			setupGetMCR: func() {
				getMCRFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.MCR, error) {
					return &megaport.MCR{
						UID:                mcrUID,
						Name:               "Original MCR",
						ProvisioningStatus: "LIVE",
					}, nil
				}
			},
			setupUpdateMCR: func() {
				updateMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ModifyMCRRequest) (*megaport.ModifyMCRResponse, error) {
					return nil, nil
				}
			},
			expectedError: "empty response from API",
		},
		{
			name: "interactive combined with flags is a usage error",
			args: []string{"mcr-123"},
			flags: map[string]string{
				"interactive": "true",
				"name":        "Updated MCR",
			},
			expectedError: "cannot be combined with",
		},
		{
			name: "interactive combined with JSON is a usage error",
			args: []string{"mcr-123"},
			flags: map[string]string{
				"interactive": "true",
				"json":        `{"name":"JSON Updated MCR"}`,
			},
			expectedError: "cannot be combined with",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset to defaults
			getMCRFunc = originalGetMCRFunc
			updateMCRFunc = originalUpdateMCRFunc
			config.SetLoginFunc(originalLoginFunc)

			if tt.setupLogin != nil {
				tt.setupLogin()
			}
			if tt.setupGetMCR != nil {
				tt.setupGetMCR()
			}
			if tt.setupUpdateMCR != nil {
				tt.setupUpdateMCR()
			}

			cmd := &cobra.Command{
				Use:  "update",
				RunE: testutil.NoColorAdapter(UpdateMCR),
			}

			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().String("cost-centre", "", "")
			cmd.Flags().Bool("marketplace-visibility", false, "")
			cmd.Flags().Int("term", 0, "")
			cmd.Flags().Int("mcr-asn", 0, "")

			testutil.SetFlags(t, cmd, tt.flags)

			var err error
			capturedOutput := captureStderr(t, func() {
				err = cmd.RunE(cmd, tt.args)
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

func TestCreateMCRPrefixFilterList(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalCreateFunc := createMCRPrefixFilterListFunc
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	defer func() {
		createMCRPrefixFilterListFunc = originalCreateFunc
	}()

	tests := []struct {
		name           string
		args           []string
		flags          map[string]string
		setupLogin     func()
		setupCreate    func()
		expectedError  string
		expectedOutput string
	}{
		{
			name: "success with flags",
			args: []string{"mcr-123"},
			flags: map[string]string{
				"description":    "Test Prefix List",
				"address-family": "IPv4",
				"entries":        `[{"action":"permit","prefix":"10.0.0.0/8"}]`,
			},
			setupLogin: func() {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.MCRService = &MockMCRService{}
					return client, nil
				})
			},
			setupCreate: func() {
				createMCRPrefixFilterListFunc = func(ctx context.Context, client *megaport.Client, req *megaport.CreateMCRPrefixFilterListRequest) (*megaport.CreateMCRPrefixFilterListResponse, error) {
					return &megaport.CreateMCRPrefixFilterListResponse{PrefixFilterListID: 42}, nil
				}
			},
			expectedOutput: "Prefix filter list created successfully",
		},
		{
			name: "success with JSON",
			args: []string{"mcr-456"},
			flags: map[string]string{
				"json": `{"description":"JSON Prefix List","addressFamily":"IPv4","entries":[{"action":"deny","prefix":"192.168.0.0/16"}]}`,
			},
			setupLogin: func() {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.MCRService = &MockMCRService{}
					return client, nil
				})
			},
			setupCreate: func() {
				createMCRPrefixFilterListFunc = func(ctx context.Context, client *megaport.Client, req *megaport.CreateMCRPrefixFilterListRequest) (*megaport.CreateMCRPrefixFilterListResponse, error) {
					return &megaport.CreateMCRPrefixFilterListResponse{PrefixFilterListID: 99}, nil
				}
			},
			expectedOutput: "Prefix filter list created successfully",
		},
		{
			name: "login error",
			args: []string{"mcr-123"},
			flags: map[string]string{
				"description":    "Test Prefix List",
				"address-family": "IPv4",
				"entries":        `[{"action":"permit","prefix":"10.0.0.0/8"}]`,
			},
			setupLogin: func() {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					return nil, fmt.Errorf("authentication failed")
				})
			},
			expectedError: "authentication failed",
		},
		{
			name: "API error",
			args: []string{"mcr-123"},
			flags: map[string]string{
				"description":    "Test Prefix List",
				"address-family": "IPv4",
				"entries":        `[{"action":"permit","prefix":"10.0.0.0/8"}]`,
			},
			setupLogin: func() {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.MCRService = &MockMCRService{}
					return client, nil
				})
			},
			setupCreate: func() {
				createMCRPrefixFilterListFunc = func(ctx context.Context, client *megaport.Client, req *megaport.CreateMCRPrefixFilterListRequest) (*megaport.CreateMCRPrefixFilterListResponse, error) {
					return nil, fmt.Errorf("API error: prefix filter list creation failed")
				}
			},
			expectedError: "API error: prefix filter list creation failed",
		},
		{
			name: "interactive combined with flags is a usage error",
			args: []string{"mcr-123"},
			flags: map[string]string{
				"interactive": "true",
				"description": "Test Prefix List",
			},
			expectedError: "cannot be combined with",
		},
		{
			name: "interactive combined with JSON is a usage error",
			args: []string{"mcr-456"},
			flags: map[string]string{
				"interactive": "true",
				"json":        `{"description":"JSON Prefix List","addressFamily":"IPv4","entries":[{"action":"deny","prefix":"192.168.0.0/16"}]}`,
			},
			expectedError: "cannot be combined with",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset to defaults
			config.SetLoginFunc(originalLoginFunc)
			createMCRPrefixFilterListFunc = originalCreateFunc

			if tt.setupLogin != nil {
				tt.setupLogin()
			}
			if tt.setupCreate != nil {
				tt.setupCreate()
			}

			cmd := &cobra.Command{
				Use:  "create-prefix-filter-list",
				RunE: testutil.NoColorAdapter(CreateMCRPrefixFilterList),
			}

			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("description", "", "")
			cmd.Flags().String("address-family", "", "")
			cmd.Flags().String("entries", "", "")

			testutil.SetFlags(t, cmd, tt.flags)

			var err error
			capturedOutput := captureStderr(t, func() {
				err = cmd.RunE(cmd, tt.args)
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

func TestUpdateMCRPrefixFilterList(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalModifyFunc := modifyMCRPrefixFilterListFunc
	originalGetPrefixFunc := getMCRPrefixFilterListFunc
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()
	defer func() {
		modifyMCRPrefixFilterListFunc = originalModifyFunc
		getMCRPrefixFilterListFunc = originalGetPrefixFunc
	}()

	tests := []struct {
		name             string
		args             []string
		flags            map[string]string
		setupLogin       func()
		setupModify      func()
		setupGetPrefixFL func()
		expectedError    string
		expectedOutput   string
	}{
		{
			name: "success with flags",
			args: []string{"mcr-123", "456"},
			flags: map[string]string{
				"description": "Updated Prefix List",
			},
			setupLogin: func() {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.MCRService = &MockMCRService{}
					return client, nil
				})
			},
			setupGetPrefixFL: func() {
				getMCRPrefixFilterListFunc = func(ctx context.Context, client *megaport.Client, mcrUID string, prefixFilterListID int) (*megaport.MCRPrefixFilterList, error) {
					return &megaport.MCRPrefixFilterList{
						ID:            456,
						Description:   "Original Prefix List",
						AddressFamily: "IPv4",
						Entries: []*megaport.MCRPrefixListEntry{
							{Action: "permit", Prefix: "10.0.0.0/8"},
						},
					}, nil
				}
			},
			setupModify: func() {
				modifyMCRPrefixFilterListFunc = func(ctx context.Context, client *megaport.Client, mcrID string, prefixFilterListID int, prefixFilterList *megaport.MCRPrefixFilterList) (*megaport.ModifyMCRPrefixFilterListResponse, error) {
					return &megaport.ModifyMCRPrefixFilterListResponse{IsUpdated: true}, nil
				}
			},
			expectedOutput: "Prefix filter list updated successfully",
		},
		{
			name: "success with JSON",
			args: []string{"mcr-789", "123"},
			flags: map[string]string{
				"json": `{"description":"JSON Updated Prefix List","entries":[{"action":"deny","prefix":"172.16.0.0/12"}]}`,
			},
			setupLogin: func() {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.MCRService = &MockMCRService{}
					return client, nil
				})
			},
			setupGetPrefixFL: func() {
				getMCRPrefixFilterListFunc = func(ctx context.Context, client *megaport.Client, mcrUID string, prefixFilterListID int) (*megaport.MCRPrefixFilterList, error) {
					return &megaport.MCRPrefixFilterList{
						ID:            123,
						Description:   "Original Prefix List",
						AddressFamily: "IPv4",
						Entries: []*megaport.MCRPrefixListEntry{
							{Action: "permit", Prefix: "10.0.0.0/8"},
						},
					}, nil
				}
			},
			setupModify: func() {
				modifyMCRPrefixFilterListFunc = func(ctx context.Context, client *megaport.Client, mcrID string, prefixFilterListID int, prefixFilterList *megaport.MCRPrefixFilterList) (*megaport.ModifyMCRPrefixFilterListResponse, error) {
					return &megaport.ModifyMCRPrefixFilterListResponse{IsUpdated: true}, nil
				}
			},
			expectedOutput: "Prefix filter list updated successfully",
		},
		{
			name:          "invalid prefix filter list ID",
			args:          []string{"mcr-123", "abc"},
			expectedError: "Invalid prefix filter list ID",
		},
		{
			name: "API error",
			args: []string{"mcr-123", "456"},
			flags: map[string]string{
				"description": "Updated Prefix List",
			},
			setupLogin: func() {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.MCRService = &MockMCRService{}
					return client, nil
				})
			},
			setupGetPrefixFL: func() {
				getMCRPrefixFilterListFunc = func(ctx context.Context, client *megaport.Client, mcrUID string, prefixFilterListID int) (*megaport.MCRPrefixFilterList, error) {
					return &megaport.MCRPrefixFilterList{
						ID:            456,
						Description:   "Original Prefix List",
						AddressFamily: "IPv4",
						Entries: []*megaport.MCRPrefixListEntry{
							{Action: "permit", Prefix: "10.0.0.0/8"},
						},
					}, nil
				}
			},
			setupModify: func() {
				modifyMCRPrefixFilterListFunc = func(ctx context.Context, client *megaport.Client, mcrID string, prefixFilterListID int, prefixFilterList *megaport.MCRPrefixFilterList) (*megaport.ModifyMCRPrefixFilterListResponse, error) {
					return nil, fmt.Errorf("API error: update failed")
				}
			},
			expectedError: "API error: update failed",
		},
		{
			name: "not successful",
			args: []string{"mcr-123", "456"},
			flags: map[string]string{
				"description": "Updated Prefix List",
			},
			setupLogin: func() {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					client := &megaport.Client{}
					client.MCRService = &MockMCRService{}
					return client, nil
				})
			},
			setupGetPrefixFL: func() {
				getMCRPrefixFilterListFunc = func(ctx context.Context, client *megaport.Client, mcrUID string, prefixFilterListID int) (*megaport.MCRPrefixFilterList, error) {
					return &megaport.MCRPrefixFilterList{
						ID:            456,
						Description:   "Original Prefix List",
						AddressFamily: "IPv4",
						Entries: []*megaport.MCRPrefixListEntry{
							{Action: "permit", Prefix: "10.0.0.0/8"},
						},
					}, nil
				}
			},
			setupModify: func() {
				modifyMCRPrefixFilterListFunc = func(ctx context.Context, client *megaport.Client, mcrID string, prefixFilterListID int, prefixFilterList *megaport.MCRPrefixFilterList) (*megaport.ModifyMCRPrefixFilterListResponse, error) {
					return &megaport.ModifyMCRPrefixFilterListResponse{IsUpdated: false}, nil
				}
			},
			expectedError: "not successful for ID 456",
		},
		{
			name: "interactive combined with flags is a usage error",
			args: []string{"mcr-123", "456"},
			flags: map[string]string{
				"interactive": "true",
				"description": "Updated Prefix List",
			},
			expectedError: "cannot be combined with",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset to defaults
			config.SetLoginFunc(originalLoginFunc)
			modifyMCRPrefixFilterListFunc = originalModifyFunc
			getMCRPrefixFilterListFunc = originalGetPrefixFunc

			if tt.setupLogin != nil {
				tt.setupLogin()
			}
			if tt.setupGetPrefixFL != nil {
				tt.setupGetPrefixFL()
			}
			if tt.setupModify != nil {
				tt.setupModify()
			}

			cmd := &cobra.Command{
				Use:  "update-prefix-filter-list",
				RunE: testutil.NoColorAdapter(UpdateMCRPrefixFilterList),
			}

			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("description", "", "")
			cmd.Flags().String("address-family", "", "")
			cmd.Flags().String("entries", "", "")

			testutil.SetFlags(t, cmd, tt.flags)

			var err error
			capturedOutput := captureStderr(t, func() {
				err = cmd.RunE(cmd, tt.args)
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

func TestLockMCRCmd_WithMockClient(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name          string
		mcrID         string
		lockErr       error
		expectedError string
		expectedOut   string
	}{
		{
			name:        "lock MCR success",
			mcrID:       "mcr-to-lock",
			expectedOut: "MCR mcr-to-lock locked successfully",
		},
		{
			name:          "lock MCR error",
			mcrID:         "mcr-error",
			lockErr:       fmt.Errorf("failed to lock MCR"),
			expectedError: "failed to lock MCR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origFunc := lockMCRFunc
			defer func() { lockMCRFunc = origFunc }()

			lockMCRFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.ManageProductLockResponse, error) {
				if tt.lockErr != nil {
					return nil, tt.lockErr
				}
				return &megaport.ManageProductLockResponse{}, nil
			}

			lockMCRCmd := &cobra.Command{
				Use: "lock [mcrUID]",
				RunE: func(cmd *cobra.Command, args []string) error {
					return LockMCR(cmd, args, false)
				},
			}

			var err error
			capturedOutput := captureStderr(t, func() {
				err = lockMCRCmd.RunE(lockMCRCmd, []string{tt.mcrID})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, capturedOutput, tt.expectedOut)
			}
		})
	}
}

func TestUnlockMCRCmd_WithMockClient(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name          string
		mcrID         string
		unlockErr     error
		expectedError string
		expectedOut   string
	}{
		{
			name:        "unlock MCR success",
			mcrID:       "mcr-to-unlock",
			expectedOut: "MCR mcr-to-unlock unlocked successfully",
		},
		{
			name:          "unlock MCR error",
			mcrID:         "mcr-error",
			unlockErr:     fmt.Errorf("failed to unlock MCR"),
			expectedError: "failed to unlock MCR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origFunc := unlockMCRFunc
			defer func() { unlockMCRFunc = origFunc }()

			unlockMCRFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.ManageProductLockResponse, error) {
				if tt.unlockErr != nil {
					return nil, tt.unlockErr
				}
				return &megaport.ManageProductLockResponse{}, nil
			}

			unlockMCRCmd := &cobra.Command{
				Use: "unlock [mcrUID]",
				RunE: func(cmd *cobra.Command, args []string) error {
					return UnlockMCR(cmd, args, false)
				},
			}

			var err error
			capturedOutput := captureStderr(t, func() {
				err = unlockMCRCmd.RunE(unlockMCRCmd, []string{tt.mcrID})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, capturedOutput, tt.expectedOut)
			}
		})
	}
}

func TestLockMCRCmd_LoginError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	defer func() { config.SetLoginFunc(originalLoginFunc) }()
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return nil, fmt.Errorf("login failed")
	})

	cmd := &cobra.Command{}
	err := LockMCR(cmd, []string{"mcr-123"}, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to log in")
}

func TestUnlockMCRCmd_LoginError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	defer func() { config.SetLoginFunc(originalLoginFunc) }()
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return nil, fmt.Errorf("login failed")
	})

	cmd := &cobra.Command{}
	err := UnlockMCR(cmd, []string{"mcr-123"}, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to log in")
}

func TestBuyMCR_Confirmation(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	originalBuyMCRFunc := buyMCRFunc
	defer func() { buyMCRFunc = originalBuyMCRFunc }()

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
				"name":        "Test MCR",
				"term":        "12",
				"port-speed":  "10000",
				"location-id": "123",
				"mcr-asn":     "65000",
			},
			confirmResult:        true,
			expectBuyCalled:      true,
			expectedOutput:       "MCR created",
			promptShouldBeCalled: true,
		},
		{
			name: "confirmation denied",
			flags: map[string]string{
				"name":        "Test MCR",
				"term":        "12",
				"port-speed":  "10000",
				"location-id": "123",
				"mcr-asn":     "65000",
			},
			confirmResult:        false,
			expectBuyCalled:      false,
			expectedError:        "cancelled by user",
			promptShouldBeCalled: true,
		},
		{
			name: "yes flag skips confirmation",
			flags: map[string]string{
				"name":        "Test MCR",
				"term":        "12",
				"port-speed":  "10000",
				"location-id": "123",
				"mcr-asn":     "65000",
				"yes":         "true",
			},
			confirmResult:        false,
			expectBuyCalled:      true,
			expectedOutput:       "MCR created",
			promptShouldBeCalled: false,
		},
		{
			name: "json input with yes skips confirmation",
			flags: map[string]string{
				"json": `{"name":"JSON MCR","term":12,"portSpeed":10000,"locationId":123,"mcrAsn":65000}`,
				"yes":  "true",
			},
			confirmResult:        false,
			expectBuyCalled:      true,
			expectedOutput:       "MCR created",
			promptShouldBeCalled: false,
		},
		{
			name: "json input without yes is a usage error",
			flags: map[string]string{
				"json": `{"name":"JSON MCR","term":12,"portSpeed":10000,"locationId":123,"mcrAsn":65000}`,
			},
			expectBuyCalled:      false,
			expectedError:        "--yes is required to confirm a purchase when using --json or --json-file",
			promptShouldBeCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMCRService := &MockMCRService{
				BuyMCRResult: &megaport.BuyMCRResponse{
					TechnicalServiceUID: "mcr-confirm-123",
				},
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.MCRService = mockMCRService
				return client, nil
			})

			buyCalled := false
			buyMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyMCRRequest) (*megaport.BuyMCRResponse, error) {
				buyCalled = true
				return &megaport.BuyMCRResponse{
					TechnicalServiceUID: "mcr-confirm-123",
				}, nil
			}

			promptCalled := false
			utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool {
				promptCalled = true
				return tt.confirmResult
			})

			cmd := &cobra.Command{
				Use:  "buy",
				RunE: testutil.NoColorAdapter(BuyMCR),
			}

			cmd.Flags().BoolP("interactive", "i", false, "")
			cmd.Flags().BoolP("yes", "y", false, "")
			cmd.Flags().Bool("no-wait", false, "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("term", 0, "")
			cmd.Flags().Int("port-speed", 0, "")
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().Int("mcr-asn", 0, "")
			cmd.Flags().String("diversity-zone", "", "")
			cmd.Flags().String("cost-centre", "", "")
			cmd.Flags().String("promo-code", "", "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")

			testutil.SetFlags(t, cmd, tt.flags)

			var err error
			capturedOutput := captureStderr(t, func() {
				err = cmd.RunE(cmd, nil)
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, capturedOutput, tt.expectedOutput)
			}

			assert.Equal(t, tt.expectBuyCalled, buyCalled, "buy function called mismatch")
			assert.Equal(t, tt.promptShouldBeCalled, promptCalled, "confirmation prompt called mismatch")
		})
	}
}

func TestExportMCRConfig(t *testing.T) {
	mcr := &megaport.MCR{
		UID:                "mcr-should-not-appear",
		Name:               "My MCR",
		ContractTermMonths: 12,
		PortSpeed:          1000,
		LocationID:         99,
		DiversityZone:      "red",
		CostCentre:         "NetOps",
		ProvisioningStatus: "LIVE",
		Resources: megaport.MCRResources{
			VirtualRouter: megaport.MCRVirtualRouter{
				ASN: 65000,
			},
		},
	}
	m := exportMCRConfig(mcr)

	assert.Equal(t, "My MCR", m["name"])
	assert.Equal(t, 12, m["term"])
	assert.Equal(t, 1000, m["portSpeed"])
	assert.Equal(t, 99, m["locationId"])
	assert.Equal(t, "red", m["diversityZone"])
	assert.Equal(t, "NetOps", m["costCentre"])
	assert.Equal(t, 65000, m["mcrAsn"])

	_, hasUID := m["productUid"]
	assert.False(t, hasUID, "export should not include productUid")
	_, hasStatus := m["provisioningStatus"]
	assert.False(t, hasStatus, "export should not include provisioningStatus")
}

func TestGetMCR_Export(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	mockService := &MockMCRService{
		GetMCRResult: &megaport.MCR{
			UID:                "mcr-export-123",
			Name:               "Export MCR",
			ContractTermMonths: 12,
			PortSpeed:          1000,
			LocationID:         42,
			ProvisioningStatus: "LIVE",
		},
	}
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.MCRService = mockService
		return client, nil
	})

	cmd := &cobra.Command{Use: "get"}
	cmd.Flags().Bool("export", false, "")
	assert.NoError(t, cmd.Flags().Set("export", "true"))

	var err error
	capturedOutput := output.CaptureStdout(func() {
		err = GetMCR(cmd, []string{"mcr-export-123"}, true, "table")
	})

	assert.NoError(t, err)
	var parsed map[string]interface{}
	assert.NoError(t, json.Unmarshal([]byte(capturedOutput), &parsed), "export output must be valid JSON")
	assert.Equal(t, "Export MCR", parsed["name"])
	assert.Equal(t, float64(42), parsed["locationId"])
	_, hasUID := parsed["productUid"]
	assert.False(t, hasUID, "export should not include productUid")
}

func TestValidateMCR(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name             string
		flags            map[string]string
		jsonInput        string
		jsonFileContent  string
		setupMock        func(*MockMCRService)
		loginError       error
		expectedError    string
		expectedContains string
	}{
		{
			name: "success with flags",
			flags: map[string]string{
				"name":                   "test-mcr",
				"term":                   "12",
				"port-speed":             "5000",
				"location-id":            "1",
				"marketplace-visibility": "true",
			},
			setupMock:        func(m *MockMCRService) {},
			expectedContains: "validation passed",
		},
		{
			name:             "success with JSON",
			jsonInput:        `{"name":"json-mcr","term":12,"portSpeed":5000,"locationId":1,"marketplaceVisibility":true}`,
			setupMock:        func(m *MockMCRService) {},
			expectedContains: "validation passed",
		},
		{
			name: "validation error",
			flags: map[string]string{
				"name":                   "test-mcr",
				"term":                   "12",
				"port-speed":             "5000",
				"location-id":            "1",
				"marketplace-visibility": "true",
			},
			setupMock: func(m *MockMCRService) {
				m.ValidateMCROrderErr = fmt.Errorf("invalid MCR configuration")
			},
			expectedError: "invalid MCR configuration",
		},
		{
			name:          "no input provided",
			flags:         map[string]string{},
			setupMock:     func(m *MockMCRService) {},
			expectedError: "no input provided",
		},
		{
			name: "login error",
			flags: map[string]string{
				"name":                   "test-mcr",
				"term":                   "12",
				"port-speed":             "5000",
				"location-id":            "1",
				"marketplace-visibility": "true",
			},
			setupMock:     func(m *MockMCRService) {},
			loginError:    fmt.Errorf("authentication failed"),
			expectedError: "authentication failed",
		},
		{
			name:          "invalid JSON input",
			jsonInput:     `{invalid json}`,
			setupMock:     func(m *MockMCRService) {},
			expectedError: "failed to parse JSON",
		},
		{
			name:             "success with JSON file",
			jsonFileContent:  `{"name":"file-mcr","term":12,"portSpeed":5000,"locationId":1,"marketplaceVisibility":true}`,
			setupMock:        func(m *MockMCRService) {},
			expectedContains: "validation passed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockMCRService{}
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
					client.MCRService = mockService
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
			cmd.Flags().Int("mcr-asn", 0, "")
			cmd.Flags().String("diversity-zone", "", "")
			cmd.Flags().String("cost-centre", "", "")
			cmd.Flags().String("promo-code", "", "")
			cmd.Flags().String("resource-tags", "", "")

			if tt.jsonInput != "" {
				assert.NoError(t, cmd.Flags().Set("json", tt.jsonInput))
			}
			if tt.jsonFileContent != "" {
				tmpFile, tmpErr := os.CreateTemp("", "mcr-validate-*.json")
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
			capturedOutput := captureStderr(t, func() {
				err = ValidateMCR(cmd, nil, true)
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.expectedContains != "" {
					assert.Contains(t, capturedOutput, tt.expectedContains)
				}
				if tt.expectedContains == "validation passed" {
					require.NotNil(t, mockService.CapturedValidateMCROrderRequest)
					require.NotNil(t, mockService.CapturedValidateMCROrderRequest.MarketplaceVisibility)
					assert.True(t, *mockService.CapturedValidateMCROrderRequest.MarketplaceVisibility)
				}
			}
		})
	}
}

func TestListMCRs_TagFilter(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	origListTagsFunc := listMCRResourceTagsFunc
	defer func() { listMCRResourceTagsFunc = origListTagsFunc }()

	allMCRs := []*megaport.MCR{
		{UID: "mcr-1", Name: "MCR-Alpha", ProvisioningStatus: "LIVE"},
		{UID: "mcr-2", Name: "MCR-Beta", ProvisioningStatus: "LIVE"},
		{UID: "mcr-3", Name: "MCR-Gamma", ProvisioningStatus: "LIVE"},
	}

	tagsByUID := map[string]map[string]string{
		"mcr-1": {"env": "prod", "team": "net"},
		"mcr-2": {"env": "staging"},
		"mcr-3": {},
	}

	tests := []struct {
		name            string
		tagFilters      []string
		tagFetchErrUID  string
		expectedOutputs []string
		notExpected     []string
	}{
		{
			name:            "no tag filter returns all",
			tagFilters:      nil,
			expectedOutputs: []string{"mcr-1", "mcr-2", "mcr-3"},
		},
		{
			name:            "exact match includes only matching resource",
			tagFilters:      []string{"env=prod"},
			expectedOutputs: []string{"mcr-1"},
			notExpected:     []string{"mcr-2", "mcr-3"},
		},
		{
			name:            "AND logic requires all filters to match",
			tagFilters:      []string{"env=prod", "team=net"},
			expectedOutputs: []string{"mcr-1"},
			notExpected:     []string{"mcr-2", "mcr-3"},
		},
		{
			name:            "key-exists match",
			tagFilters:      []string{"env"},
			expectedOutputs: []string{"mcr-1", "mcr-2"},
			notExpected:     []string{"mcr-3"},
		},
		{
			name:        "no match returns empty",
			tagFilters:  []string{"env=nonexistent"},
			notExpected: []string{"mcr-1", "mcr-2", "mcr-3"},
		},
		{
			name:            "tag fetch error excludes resource gracefully",
			tagFilters:      []string{"env=prod"},
			tagFetchErrUID:  "mcr-1",
			notExpected:     []string{"MCR-Alpha"}, // name absent from table; UID appears in warning
			expectedOutputs: []string{"Failed to fetch tags"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listMCRResourceTagsFunc = func(ctx context.Context, client *megaport.Client, uid string) (map[string]string, error) {
				if tt.tagFetchErrUID != "" && uid == tt.tagFetchErrUID {
					return nil, fmt.Errorf("tag fetch error")
				}
				return tagsByUID[uid], nil
			}

			mockSvc := &MockMCRService{ListMCRsResult: allMCRs}
			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				return &megaport.Client{MCRService: mockSvc}, nil
			})

			cmd := &cobra.Command{Use: "list"}
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().Int("port-speed", 0, "")
			cmd.Flags().Bool("include-inactive", false, "")
			cmd.Flags().Int("limit", 0, "")
			cmd.Flags().StringArray("tag", nil, "")

			for _, f := range tt.tagFilters {
				assert.NoError(t, cmd.Flags().Set("tag", f))
			}

			var listErr error
			var capturedOutput string
			capturedStderr := captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					listErr = ListMCRs(cmd, nil, true, "table")
				})
			})
			assert.NoError(t, listErr)

			for _, expected := range tt.expectedOutputs {
				assert.Contains(t, capturedOutput+capturedStderr, expected)
			}
			for _, notExp := range tt.notExpected {
				assert.NotContains(t, capturedOutput, notExp)
			}
		})
	}
}

// TestBuyMCR_OrderAndProvisionFailures covers the order-response and
// provisioning-poll failure branches: an empty API response, a missing UID, and
// the MCR vanishing during the provisioning poll.
func TestBuyMCR_OrderAndProvisionFailures(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	originalBuyMCRFunc := buyMCRFunc
	originalGetMCRFunc := getMCRFunc
	defer func() {
		cleanup()
		buyMCRFunc = originalBuyMCRFunc
		getMCRFunc = originalGetMCRFunc
	}()

	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.MCRService = &MockMCRService{}
		return client, nil
	})

	tests := []struct {
		name     string
		buyResp  *megaport.BuyMCRResponse
		getNil   bool
		errMatch string
	}{
		{name: "empty API response", buyResp: nil, errMatch: "empty response from API"},
		{name: "no UID returned", buyResp: &megaport.BuyMCRResponse{}, errMatch: "no UID returned"},
		{name: "MCR not found during provisioning poll", buyResp: &megaport.BuyMCRResponse{TechnicalServiceUID: "mcr-uid-123"}, getNil: true, errMatch: "not found while waiting for provisioning"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buyMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyMCRRequest) (*megaport.BuyMCRResponse, error) {
				return tt.buyResp, nil
			}
			getMCRFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.MCR, error) {
				if tt.getNil {
					return nil, nil
				}
				return &megaport.MCR{UID: mcrUID, ProvisioningStatus: "LIVE"}, nil
			}

			cmd := &cobra.Command{Use: "buy"}
			cmd.Flags().BoolP("interactive", "i", false, "")
			cmd.Flags().Bool("no-wait", false, "")
			cmd.Flags().BoolP("yes", "y", false, "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("term", 0, "")
			cmd.Flags().Int("port-speed", 0, "")
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().Int("mcr-asn", 0, "")
			cmd.Flags().String("diversity-zone", "", "")
			cmd.Flags().String("cost-centre", "", "")
			cmd.Flags().String("promo-code", "", "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")

			testutil.SetFlags(t, cmd, map[string]string{
				"name":        "Test MCR",
				"term":        "12",
				"port-speed":  "10000",
				"location-id": "123",
				"mcr-asn":     "65000",
			})

			var err error
			output.CaptureOutput(func() {
				err = BuyMCR(cmd, nil, true)
			})

			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tt.errMatch)
			}
		})
	}
}
