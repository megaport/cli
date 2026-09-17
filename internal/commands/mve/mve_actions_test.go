package mve

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/config"
	"github.com/megaport/megaport-cli/internal/testutil"
	"github.com/megaport/megaport-cli/internal/utils"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

var noColor = true // Disable color for testing

// captureStderr captures what fn writes to os.Stderr, draining the read end
// concurrently so fn cannot block on a full pipe buffer.
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
	go func() {
		defer close(done)
		_, _ = io.Copy(&buf, r)
	}()

	defer func() {
		_ = w.Close()
		<-done
		_ = r.Close()
		result = buf.String()
	}()
	fn()
	return
}

var testMVEImages = []*megaport.MVEImage{
	{
		ID:                1,
		Version:           "1.0",
		Product:           "Product1",
		Vendor:            "Cisco",
		VendorDescription: "Cisco Description",
		ReleaseImage:      true,
		ProductCode:       "CISCO123",
	},
	{
		ID:                2,
		Version:           "2.0",
		Product:           "Product2",
		Vendor:            "Fortinet",
		VendorDescription: "Fortinet Description",
		ReleaseImage:      false,
		ProductCode:       "FORTINET456",
	},
}

var testMVESizes = []*megaport.MVESize{
	{
		Size:         "small",
		Label:        "Small",
		CPUCoreCount: 2,
		RamGB:        8,
	},
	{
		Size:         "large",
		Label:        "Large",
		CPUCoreCount: 8,
		RamGB:        32,
	},
}

func TestFilterMVEImages(t *testing.T) {
	tests := []struct {
		name          string
		vendor        string
		productCode   string
		id            int
		version       string
		releaseImage  bool
		expectedCount int
	}{
		{
			name:          "filter by vendor",
			vendor:        "Cisco",
			expectedCount: 1,
		},
		{
			name:          "filter by product code",
			productCode:   "FORTINET456",
			expectedCount: 1,
		},
		{
			name:          "filter by ID",
			id:            1,
			expectedCount: 1,
		},
		{
			name:          "filter by version",
			version:       "2.0",
			expectedCount: 1,
		},
		{
			name:          "filter by release image",
			releaseImage:  true,
			expectedCount: 1,
		},
		{
			name:          "no filters",
			expectedCount: 2,
		},
		{
			name:          "no matches",
			vendor:        "NonExistentVendor",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filteredImages := filterMVEImages(testMVEImages, tt.vendor, tt.productCode, tt.id, tt.version, tt.releaseImage)
			assert.Equal(t, tt.expectedCount, len(filteredImages))
		})
	}
}
func TestListMVEImages(t *testing.T) {
	mockService := &MockMVEService{
		ListMVEImagesResult: testMVEImages,
	}

	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.MVEService = mockService
	})
	defer cleanup()

	tests := []struct {
		name          string
		args          []string
		expectedCount int
		checkFor      string
	}{
		{
			name:          "no filters",
			args:          []string{},
			expectedCount: 2,
			checkFor:      "Product1",
		},
		{
			name:          "filter by vendor",
			args:          []string{"--vendor", "Cisco"},
			expectedCount: 1,
			checkFor:      "Product1",
		},
		{
			name:          "filter by product code",
			args:          []string{"--product-code", "FORTINET456"},
			expectedCount: 1,
			checkFor:      "Product2",
		},
		{
			name:          "filter by ID",
			args:          []string{"--id", "1"},
			expectedCount: 1,
			checkFor:      "Product1",
		},
		{
			name:          "filter by version",
			args:          []string{"--version", "2.0"},
			expectedCount: 1,
			checkFor:      "Product2",
		},
		{
			name:          "filter by release image",
			args:          []string{"--release-image"},
			expectedCount: 1,
			checkFor:      "Product1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := &cobra.Command{Use: "megaport"}
			mveCmd := &cobra.Command{Use: "mve"}

			listImagesCmd := &cobra.Command{
				Use: "list-images",
				RunE: func(cmd *cobra.Command, args []string) error {
					ctx := context.Background()
					client, err := config.Login(ctx)
					if err != nil {
						return err
					}

					vendor, _ := cmd.Flags().GetString("vendor")
					productCode, _ := cmd.Flags().GetString("product-code")
					id, _ := cmd.Flags().GetInt("id")
					version, _ := cmd.Flags().GetString("version")
					releaseImage, _ := cmd.Flags().GetBool("release-image")

					images, err := client.MVEService.ListMVEImages(ctx)
					if err != nil {
						return err
					}

					filteredImages := filterMVEImages(images, vendor, productCode, id, version, releaseImage)

					for _, img := range filteredImages {
						fmt.Printf("%d    %s       %s    %s     %s      %t           %s\n",
							img.ID, img.Version, img.Product, img.Vendor, img.VendorDescription, img.ReleaseImage, img.ProductCode)
					}

					return nil
				},
			}

			listImagesCmd.Flags().String("vendor", "", "Filter by vendor")
			listImagesCmd.Flags().String("product-code", "", "Filter by product code")
			listImagesCmd.Flags().Int("id", 0, "Filter by ID")
			listImagesCmd.Flags().String("version", "", "Filter by version")
			listImagesCmd.Flags().Bool("release-image", false, "Filter by release image")

			mveCmd.AddCommand(listImagesCmd)
			rootCmd.AddCommand(mveCmd)

			rootCmd.SetArgs(append([]string{"mve", "list-images"}, tt.args...))

			output, err := output.CaptureOutputErr(func() error {
				return rootCmd.Execute()
			})

			assert.NoError(t, err)
			assert.Contains(t, output, tt.checkFor)
			productCount := strings.Count(output, "Product")
			assert.Equal(t, tt.expectedCount, productCount, "Expected %d products, but got %d", tt.expectedCount, productCount)
		})
	}
}

func TestListAvailableMVESizes(t *testing.T) {
	mockService := &MockMVEService{
		ListAvailableMVESizesResult: testMVESizes,
	}

	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.MVEService = mockService
	})
	defer cleanup()

	rootCmd := &cobra.Command{Use: "megaport"}
	mveCmd := &cobra.Command{Use: "mve"}

	listSizesCmd := &cobra.Command{
		Use: "list-sizes",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := config.Login(ctx)
			if err != nil {
				return err
			}

			sizes, err := client.MVEService.ListAvailableMVESizes(ctx)
			if err != nil {
				return err
			}

			for _, size := range sizes {
				fmt.Printf("%s    %s    %d    %d\n", size.Size, size.Label, size.CPUCoreCount, size.RamGB)
			}

			return nil
		},
	}

	mveCmd.AddCommand(listSizesCmd)
	rootCmd.AddCommand(mveCmd)

	rootCmd.SetArgs([]string{"mve", "list-sizes"})

	output, err := output.CaptureOutputErr(func() error {
		return rootCmd.Execute()
	})

	assert.NoError(t, err)
	assert.Contains(t, output, "small")
	assert.Contains(t, output, "large")
}

func TestUpdateMVE(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() {
		utils.SetResourcePrompt(originalPrompt)
	}()

	mockService := &MockMVEService{
		GetMVEResult: &megaport.MVE{
			Name: "Mock MVE",
		},
	}

	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.MVEService = mockService
	})
	defer cleanup()

	tests := []struct {
		name            string
		args            []string
		interactive     bool
		prompts         []string
		flags           map[string]string
		jsonInput       string
		mockSetup       func(*MockMVEService)
		expectedError   string
		expectedOutput  string
		validateRequest func(*testing.T, *megaport.ModifyMVERequest)
	}{
		{
			name:        "interactive mode success",
			args:        []string{"mve-123"},
			interactive: true,
			prompts: []string{
				"Updated MVE",
				"New Cost Centre",
				"24",
			},
			mockSetup: func(m *MockMVEService) {
				m.ModifyMVEResult = &megaport.ModifyMVEResponse{
					MVEUpdated: true,
				}
			},
			expectedOutput: "MVE updated mve-123",
			validateRequest: func(t *testing.T, req *megaport.ModifyMVERequest) {
				assert.Equal(t, "mve-123", req.MVEID)
				assert.Equal(t, "Updated MVE", req.Name)
				assert.Equal(t, "New Cost Centre", req.CostCentre)
				assert.NotNil(t, req.ContractTermMonths)
				assert.Equal(t, 24, *req.ContractTermMonths)
				assert.True(t, req.WaitForUpdate)
			},
		},
		{
			name: "flag mode success",
			args: []string{"mve-123"},
			flags: map[string]string{
				"name":        "Flag Updated MVE",
				"cost-centre": "Flag Cost Centre",
				"term":        "36",
			},
			mockSetup: func(m *MockMVEService) {
				m.ModifyMVEResult = &megaport.ModifyMVEResponse{
					MVEUpdated: true,
				}
			},
			expectedOutput: "MVE updated mve-123",
			validateRequest: func(t *testing.T, req *megaport.ModifyMVERequest) {
				assert.Equal(t, "mve-123", req.MVEID)
				assert.Equal(t, "Flag Updated MVE", req.Name)
				assert.Equal(t, "Flag Cost Centre", req.CostCentre)
				assert.NotNil(t, req.ContractTermMonths)
				assert.Equal(t, 36, *req.ContractTermMonths)
				assert.True(t, req.WaitForUpdate)
			},
		},
		{
			name: "json mode success",
			args: []string{"mve-123"},
			flags: map[string]string{
				"json": `{
					"name": "JSON Updated MVE",
					"costCentre": "JSON Cost Centre",
					"contractTermMonths": 12
				}`,
			},
			mockSetup: func(m *MockMVEService) {
				m.ModifyMVEResult = &megaport.ModifyMVEResponse{
					MVEUpdated: true,
				}
			},
			expectedOutput: "MVE updated mve-123",
			validateRequest: func(t *testing.T, req *megaport.ModifyMVERequest) {
				assert.Equal(t, "mve-123", req.MVEID)
				assert.Equal(t, "JSON Updated MVE", req.Name)
				assert.Equal(t, "JSON Cost Centre", req.CostCentre)
				assert.NotNil(t, req.ContractTermMonths)
				assert.Equal(t, 12, *req.ContractTermMonths)
				assert.True(t, req.WaitForUpdate)
			},
		},
		{
			name: "flag update without cost-centre preserves the existing one",
			args: []string{"mve-123"},
			flags: map[string]string{
				"name": "Renamed MVE",
			},
			mockSetup: func(m *MockMVEService) {
				m.ModifyMVEResult = &megaport.ModifyMVEResponse{MVEUpdated: true}
				m.GetMVEResult = &megaport.MVE{Name: "Mock MVE", CostCentre: "Existing CC"}
			},
			expectedOutput: "MVE updated mve-123",
			validateRequest: func(t *testing.T, req *megaport.ModifyMVERequest) {
				assert.Equal(t, "Renamed MVE", req.Name)
				assert.Equal(t, "Existing CC", req.CostCentre)
			},
		},
		{
			name: "flag update with empty cost-centre clears it",
			args: []string{"mve-123"},
			flags: map[string]string{
				"cost-centre": "",
			},
			mockSetup: func(m *MockMVEService) {
				m.ModifyMVEResult = &megaport.ModifyMVEResponse{MVEUpdated: true}
				m.GetMVEResult = &megaport.MVE{Name: "Mock MVE", CostCentre: "Existing CC"}
			},
			expectedOutput: "MVE updated mve-123",
			validateRequest: func(t *testing.T, req *megaport.ModifyMVERequest) {
				assert.Empty(t, req.CostCentre)
			},
		},
		{
			name: "json update without cost-centre preserves the existing one",
			args: []string{"mve-123"},
			flags: map[string]string{
				"json": `{"name":"JSON Renamed"}`,
			},
			mockSetup: func(m *MockMVEService) {
				m.ModifyMVEResult = &megaport.ModifyMVEResponse{MVEUpdated: true}
				m.GetMVEResult = &megaport.MVE{Name: "Mock MVE", CostCentre: "Existing CC"}
			},
			expectedOutput: "MVE updated mve-123",
			validateRequest: func(t *testing.T, req *megaport.ModifyMVERequest) {
				assert.Equal(t, "JSON Renamed", req.Name)
				assert.Equal(t, "Existing CC", req.CostCentre)
			},
		},
		{
			name: "json update with empty cost-centre clears it",
			args: []string{"mve-123"},
			flags: map[string]string{
				"json": `{"costCentre":""}`,
			},
			mockSetup: func(m *MockMVEService) {
				m.ModifyMVEResult = &megaport.ModifyMVEResponse{MVEUpdated: true}
				m.GetMVEResult = &megaport.MVE{Name: "Mock MVE", CostCentre: "Existing CC"}
			},
			expectedOutput: "MVE updated mve-123",
			validateRequest: func(t *testing.T, req *megaport.ModifyMVERequest) {
				assert.Empty(t, req.CostCentre)
			},
		},
		{
			name:        "interactive update preserves cost-centre when skipped",
			args:        []string{"mve-123"},
			interactive: true,
			prompts: []string{
				"Renamed MVE",
				"",
				"",
			},
			mockSetup: func(m *MockMVEService) {
				m.ModifyMVEResult = &megaport.ModifyMVEResponse{MVEUpdated: true}
				m.GetMVEResult = &megaport.MVE{Name: "Mock MVE", CostCentre: "Existing CC"}
			},
			expectedOutput: "MVE updated mve-123",
			validateRequest: func(t *testing.T, req *megaport.ModifyMVERequest) {
				assert.Equal(t, "Renamed MVE", req.Name)
				assert.Equal(t, "Existing CC", req.CostCentre)
			},
		},
		{
			name:          "no input provided",
			args:          []string{"mve-123"},
			expectedError: "no input provided",
		},
		{
			name: "update error",
			args: []string{"mve-123"},
			flags: map[string]string{
				"name": "Updated MVE",
			},
			mockSetup: func(m *MockMVEService) {
				m.ModifyMVEErr = fmt.Errorf("update failed")
			},
			expectedError: "update failed",
		},
		{
			name: "invalid contract term",
			args: []string{"mve-123"},
			flags: map[string]string{
				"term": "13",
			},
			expectedError: "Invalid contract term: 13 - must be one of: [1 12 24 36 48 60]",
		},
		{
			name: "update not successful",
			args: []string{"mve-123"},
			flags: map[string]string{
				"name": "Updated MVE",
			},
			mockSetup: func(m *MockMVEService) {
				m.ModifyMVEResult = &megaport.ModifyMVEResponse{
					MVEUpdated: false,
				}
				m.GetMVEResult = &megaport.MVE{
					Name: "Mock MVE",
				}
			},
			expectedError: "MVE update request was not successful",
		},
		{
			name: "nil response from API",
			args: []string{"mve-123"},
			flags: map[string]string{
				"name": "Updated MVE",
			},
			mockSetup: func(m *MockMVEService) {
				m.ModifyMVENilResp = true
			},
			expectedError: "empty response from API",
		},
		{
			name: "original MVE not found",
			args: []string{"mve-123"},
			flags: map[string]string{
				"name": "Updated MVE",
			},
			mockSetup: func(m *MockMVEService) {
				m.ForceNilGetMVE = true
			},
			expectedError: "MVE mve-123 not found",
		},
		{
			name: "vnic count mismatch",
			args: []string{"mve-123"},
			flags: map[string]string{
				"vnics": `[{"description":"Only One"}]`,
			},
			mockSetup: func(m *MockMVEService) {
				m.GetMVEResult = &megaport.MVE{
					Name: "Mock MVE",
					NetworkInterfaces: []*megaport.MVENetworkInterface{
						{Description: "Data Plane"},
						{Description: "Management"},
					},
				}
			},
			expectedError: "vnics length (1) must match the MVE's existing vNIC count (2)",
		},
		{
			name: "explicit empty vnics array rejected alongside other updates",
			args: []string{"mve-123"},
			flags: map[string]string{
				"name":  "Renamed",
				"vnics": `[]`,
			},
			mockSetup: func(m *MockMVEService) {
				m.GetMVEResult = &megaport.MVE{
					Name: "Mock MVE",
					NetworkInterfaces: []*megaport.MVENetworkInterface{
						{Description: "Data Plane"},
					},
				}
			},
			expectedError: "vnics must contain at least one object",
		},
		{
			name: "vnic count match success",
			args: []string{"mve-123"},
			flags: map[string]string{
				"vnics": `[{"description":"Data Plane Renamed"},{"description":"Mgmt"}]`,
			},
			mockSetup: func(m *MockMVEService) {
				m.ModifyMVEResult = &megaport.ModifyMVEResponse{MVEUpdated: true}
				m.GetMVEResult = &megaport.MVE{
					Name: "Mock MVE",
					NetworkInterfaces: []*megaport.MVENetworkInterface{
						{Description: "Data Plane"},
						{Description: "Management"},
					},
				}
			},
			expectedOutput: "MVE updated mve-123",
			validateRequest: func(t *testing.T, req *megaport.ModifyMVERequest) {
				assert.Len(t, req.Vnics, 2)
				assert.Equal(t, "Data Plane Renamed", req.Vnics[0].Description)
				assert.Equal(t, "Mgmt", req.Vnics[1].Description)
			},
		},
		{
			name:          "interactive combined with flags is a usage error",
			args:          []string{"mve-123"},
			interactive:   true,
			flags:         map[string]string{"name": "Updated MVE"},
			expectedError: "cannot be combined with",
		},
		{
			name:          "interactive combined with JSON is a usage error",
			args:          []string{"mve-123"},
			interactive:   true,
			flags:         map[string]string{"json": `{"name":"JSON MVE"}`},
			expectedError: "cannot be combined with",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.Reset()
			if tt.mockSetup != nil {
				tt.mockSetup(mockService)
			}

			promptIndex := 0
			utils.SetResourcePrompt(func(_, msg string, _ bool) (string, error) {
				if promptIndex < len(tt.prompts) {
					response := tt.prompts[promptIndex]
					promptIndex++
					return response, nil
				}
				return "", fmt.Errorf("unexpected prompt call")
			})

			cmd := &cobra.Command{Use: "update"}
			cmd.Flags().Bool("interactive", tt.interactive, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().String("cost-centre", "", "")
			cmd.Flags().Int("term", 0, "")
			cmd.Flags().String("vnics", "", "")

			testutil.SetFlags(t, cmd, tt.flags)

			var err error
			var capturedOutput string
			capturedStderr := captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = UpdateMVE(cmd, tt.args, noColor)
				})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, capturedOutput+capturedStderr, tt.expectedOutput)

				if tt.validateRequest != nil {
					tt.validateRequest(t, mockService.CapturedModifyMVERequest)
				}
			}
		})
	}
}

func TestDeleteMVE(t *testing.T) {
	originalConfirmPrompt := utils.GetConfirmPrompt()
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer func() {
		cleanup()
		utils.SetConfirmPrompt(originalConfirmPrompt)
	}()

	tests := []struct {
		name           string
		mockSetup      func(*MockMVEService)
		confirmDelete  bool
		forceFlag      bool
		safeDeleteFlag bool
		expectedError  string
		expectedOutput string
	}{
		{
			name: "successful MVE deletion",
			mockSetup: func(m *MockMVEService) {
				m.DeleteMVEErr = nil
			},
			confirmDelete:  true,
			expectedOutput: "MVE deleted mve-uid",
		},
		{
			name: "safe delete flag passed to request",
			mockSetup: func(m *MockMVEService) {
				m.DeleteMVEErr = nil
			},
			forceFlag:      true,
			safeDeleteFlag: true,
			expectedOutput: "MVE deleted mve-uid",
		},
		{
			name: "deletion error",
			mockSetup: func(m *MockMVEService) {
				m.DeleteMVEErr = fmt.Errorf("deletion failed")
				m.DeleteMVEResult = &megaport.DeleteMVEResponse{
					IsDeleted: false,
				}
			},
			confirmDelete: true,
			expectedError: "deletion failed",
		},
		{
			name: "deletion cancelled",
			mockSetup: func(m *MockMVEService) {
			},
			confirmDelete: false,
			expectedError: "cancelled by user",
		},
		{
			name: "force deletion",
			mockSetup: func(m *MockMVEService) {
				m.DeleteMVEErr = nil
			},
			forceFlag:      true,
			expectedOutput: "MVE deleted mve-uid",
		},
		{
			name: "nil response from API",
			mockSetup: func(m *MockMVEService) {
				m.DeleteMVENilResp = true
			},
			forceFlag:     true,
			expectedError: "empty response from API",
		},
		{
			name: "delete not successful",
			mockSetup: func(m *MockMVEService) {
				m.DeleteMVEResult = &megaport.DeleteMVEResponse{
					IsDeleted: false,
				}
			},
			forceFlag:     true,
			expectedError: "not successful for mve-uid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockMVEService{}
			tt.mockSetup(mockService)

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{
					MVEService: mockService,
				}
				return client, nil
			})

			utils.SetConfirmPrompt(func(question string, _ bool) bool {
				return tt.confirmDelete
			})

			cmd := &cobra.Command{
				Use: "delete",
				RunE: func(cmd *cobra.Command, args []string) error {
					return DeleteMVE(cmd, []string{"mve-uid"}, noColor)
				},
			}

			cmd.Flags().Bool("force", tt.forceFlag, "")
			cmd.Flags().Bool("safe-delete", false, "")
			if tt.safeDeleteFlag {
				_ = cmd.Flags().Set("safe-delete", "true")
			}

			var err error
			var capturedOutput string
			capturedStderr := captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = cmd.Execute()
				})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, capturedOutput+capturedStderr, tt.expectedOutput)
				if tt.safeDeleteFlag {
					assert.NotNil(t, mockService.CapturedDeleteMVERequest)
					assert.True(t, mockService.CapturedDeleteMVERequest.SafeDelete)
				}
			}
		})
	}
}

func TestBuyMVE(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() {
		utils.SetResourcePrompt(originalPrompt)
	}()
	originalSecretPrompt := utils.GetSecretResourcePrompt()
	defer func() { utils.SetSecretResourcePrompt(originalSecretPrompt) }()
	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	mockService := &MockMVEService{}

	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.MVEService = mockService
	})
	defer cleanup()

	tests := []struct {
		name            string
		args            []string
		interactive     bool
		prompts         []string
		flags           map[string]string
		jsonInput       string
		jsonFilePath    string
		mockSetup       func(*MockMVEService)
		expectedError   string
		expectedOutput  string
		validateRequest func(*testing.T, *megaport.BuyMVERequest)
	}{
		{
			name:        "interactive mode success",
			args:        []string{},
			interactive: true,
			prompts: []string{
				"Test MVE",
				"12",
				"123",
				"",
				"",
				"CC-123",
				"cisco",
				"1",
				"LARGE",
				"label-1",
				"true",
				"admin-ssh",
				"ssh-key",
				"cloud-init",
				"fmc-ip",
				"fmc-key",
				"fmc-nat",
				"admin-pass",
				"VNIC 1",
				"100",
				"",
				"",
			},
			mockSetup: func(m *MockMVEService) {
				m.ValidateMVEOrderErr = nil
				m.BuyMVEResult = &megaport.BuyMVEResponse{
					TechnicalServiceUID: "mock-mve-uid",
				}
			},
			expectedOutput: "MVE created mock-mve-uid",
			validateRequest: func(t *testing.T, req *megaport.BuyMVERequest) {
				assert.Equal(t, "Test MVE", req.Name)
				assert.Equal(t, 12, req.Term)
				assert.Equal(t, 123, req.LocationID)
				assert.Equal(t, "CC-123", req.CostCentre)

				ciscoConfig, ok := req.VendorConfig.(*megaport.CiscoConfig)
				assert.True(t, ok, "Expected a CiscoConfig")
				assert.Equal(t, 1, ciscoConfig.ImageID)
				assert.Equal(t, "LARGE", ciscoConfig.ProductSize)
				assert.Equal(t, "label-1", ciscoConfig.MVELabel)
				assert.True(t, ciscoConfig.ManageLocally)
				assert.Equal(t, "admin-ssh", ciscoConfig.AdminSSHPublicKey)
				assert.Equal(t, "ssh-key", ciscoConfig.SSHPublicKey)
				assert.Equal(t, "cloud-init", ciscoConfig.CloudInit)
				assert.Equal(t, "fmc-ip", ciscoConfig.FMCIPAddress)
				assert.Equal(t, "fmc-key", ciscoConfig.FMCRegistrationKey)
				assert.Equal(t, "fmc-nat", ciscoConfig.FMCNatID)
				assert.Equal(t, "admin-pass", ciscoConfig.AdminPassword)

				assert.Len(t, req.Vnics, 1)
				assert.Equal(t, "VNIC 1", req.Vnics[0].Description)
				assert.Equal(t, 100, req.Vnics[0].VLAN)
			},
		},
		{
			name: "flag mode success",
			args: []string{},
			flags: map[string]string{
				"name":          "Test MVE",
				"term":          "12",
				"location-id":   "123",
				"vendor-config": `{"vendor":"cisco","imageId":1,"productSize":"LARGE","mveLabel":"label-1","manageLocally":true,"adminSshPublicKey":"admin-ssh","sshPublicKey":"ssh-key","cloudInit":"cloud-init","fmcIpAddress":"fmc-ip","fmcRegistrationKey":"fmc-key","fmcNatId":"fmc-nat"}`,
				"vnics":         `[{"description":"VNIC 1","vlan":100}]`,
			},
			mockSetup: func(m *MockMVEService) {
				m.ValidateMVEOrderErr = nil
				m.BuyMVEResult = &megaport.BuyMVEResponse{
					TechnicalServiceUID: "mock-mve-uid",
				}
			},
			expectedOutput: "MVE created mock-mve-uid",
			validateRequest: func(t *testing.T, req *megaport.BuyMVERequest) {
				assert.Equal(t, "Test MVE", req.Name)
				assert.Equal(t, 12, req.Term)
				assert.Equal(t, 123, req.LocationID)

				ciscoConfig, ok := req.VendorConfig.(*megaport.CiscoConfig)
				assert.True(t, ok, "Expected a CiscoConfig")
				assert.Equal(t, 1, ciscoConfig.ImageID)
				assert.Equal(t, "LARGE", ciscoConfig.ProductSize)
				assert.Equal(t, "label-1", ciscoConfig.MVELabel)
				assert.True(t, ciscoConfig.ManageLocally)
				assert.Equal(t, "admin-ssh", ciscoConfig.AdminSSHPublicKey)
				assert.Equal(t, "ssh-key", ciscoConfig.SSHPublicKey)
				assert.Equal(t, "cloud-init", ciscoConfig.CloudInit)
				assert.Equal(t, "fmc-ip", ciscoConfig.FMCIPAddress)
				assert.Equal(t, "fmc-key", ciscoConfig.FMCRegistrationKey)
				assert.Equal(t, "fmc-nat", ciscoConfig.FMCNatID)

				assert.Len(t, req.Vnics, 1)
				assert.Equal(t, "VNIC 1", req.Vnics[0].Description)
				assert.Equal(t, 100, req.Vnics[0].VLAN)
			},
		},
		{
			name: "json mode success",
			args: []string{},
			flags: map[string]string{
				"yes": "true",
				"json": `{
					"name": "JSON MVE",
					"term": 12,
					"locationId": 123,
					"vendorConfig": {
						"vendor": "cisco",
						"imageId": 1,
						"productSize": "LARGE",
						"mveLabel": "json-label",
						"manageLocally": true,
						"adminSshPublicKey": "admin-ssh",
						"sshPublicKey": "ssh-key",
						"cloudInit": "cloud-init",
						"fmcIpAddress": "fmc-ip",
						"fmcRegistrationKey": "fmc-key",
						"fmcNatId": "fmc-nat"
					},
					"vnics": [
						{"description": "JSON VNIC", "vlan": 200}
					]
				}`,
			},
			mockSetup: func(m *MockMVEService) {
				m.ValidateMVEOrderErr = nil
				m.BuyMVEResult = &megaport.BuyMVEResponse{
					TechnicalServiceUID: "mock-mve-uid",
				}
			},
			expectedOutput: "MVE created mock-mve-uid",
			validateRequest: func(t *testing.T, req *megaport.BuyMVERequest) {
				assert.Equal(t, "JSON MVE", req.Name)
				assert.Equal(t, 12, req.Term)
				assert.Equal(t, 123, req.LocationID)

				ciscoConfig, ok := req.VendorConfig.(*megaport.CiscoConfig)
				assert.True(t, ok, "Expected a CiscoConfig")
				assert.Equal(t, 1, ciscoConfig.ImageID)
				assert.Equal(t, "LARGE", ciscoConfig.ProductSize)
				assert.Equal(t, "json-label", ciscoConfig.MVELabel)
				assert.True(t, ciscoConfig.ManageLocally)
				assert.Equal(t, "admin-ssh", ciscoConfig.AdminSSHPublicKey)
				assert.Equal(t, "ssh-key", ciscoConfig.SSHPublicKey)
				assert.Equal(t, "cloud-init", ciscoConfig.CloudInit)
				assert.Equal(t, "fmc-ip", ciscoConfig.FMCIPAddress)
				assert.Equal(t, "fmc-key", ciscoConfig.FMCRegistrationKey)
				assert.Equal(t, "fmc-nat", ciscoConfig.FMCNatID)

				assert.Len(t, req.Vnics, 1)
				assert.Equal(t, "JSON VNIC", req.Vnics[0].Description)
				assert.Equal(t, 200, req.Vnics[0].VLAN)
			},
		},
		{
			name:          "no input provided",
			args:          []string{},
			expectedError: "no input provided",
		}, {
			name: "validation error",
			flags: map[string]string{
				"name":          "Test MVE",
				"term":          "12",
				"location-id":   "123",
				"vendor-config": `{"vendor":"cisco","imageId":1,"productSize":"LARGE","mveLabel":"label-1","manageLocally":true,"adminSshPublicKey":"admin-ssh","sshPublicKey":"ssh-key","cloudInit":"cloud-init","fmcIpAddress":"fmc-ip","fmcRegistrationKey":"fmc-key","fmcNatId":"fmc-nat"}`,
				"vnics":         `[{"description":"VNIC 1","vlan":100}]`,
			},
			mockSetup: func(m *MockMVEService) {
				m.ValidateMVEOrderErr = fmt.Errorf("validation failed")
			},
			expectedError: "validation failed",
		},
		{
			name: "purchase error",
			flags: map[string]string{
				"name":          "Test MVE",
				"term":          "12",
				"location-id":   "123",
				"vendor-config": `{"vendor":"cisco","imageId":1,"productSize":"LARGE","mveLabel":"label-1","manageLocally":true,"adminSshPublicKey":"admin-ssh","sshPublicKey":"ssh-key","cloudInit":"cloud-init","fmcIpAddress":"fmc-ip","fmcRegistrationKey":"fmc-key","fmcNatId":"fmc-nat"}`,
				"vnics":         `[{"description":"VNIC 1","vlan":100}]`,
			},
			mockSetup: func(m *MockMVEService) {
				m.ValidateMVEOrderErr = nil
				m.BuyMVEErr = fmt.Errorf("purchase failed")
			},
			expectedError: "purchase failed",
		},
		{
			name: "nil response from API",
			flags: map[string]string{
				"name":          "Nil-MVE",
				"term":          "12",
				"location-id":   "1",
				"vendor-config": `{"vendor":"cisco","imageId":1,"productSize":"LARGE","mveLabel":"label-1","manageLocally":true,"adminSshPublicKey":"admin-ssh","sshPublicKey":"ssh-key","cloudInit":"cloud-init","fmcIpAddress":"fmc-ip","fmcRegistrationKey":"fmc-key","fmcNatId":"fmc-nat"}`,
				"vnics":         `[{"description":"VNIC 1","vlan":100}]`,
			},
			mockSetup: func(m *MockMVEService) {
				m.BuyMVENilResp = true
			},
			expectedError: "empty response from API",
		},
		{
			name: "invalid JSON returns error",
			flags: map[string]string{
				"yes":  "true",
				"json": `{bad json}`,
			},
			expectedError: "failed to parse JSON",
		},
		{
			name: "json without yes is a usage error",
			flags: map[string]string{
				"json": `{
					"name": "JSON MVE",
					"term": 12,
					"locationId": 123,
					"vendorConfig": {
						"vendor": "cisco",
						"imageId": 1,
						"productSize": "LARGE",
						"mveLabel": "json-label",
						"manageLocally": true,
						"adminSshPublicKey": "admin-ssh",
						"sshPublicKey": "ssh-key",
						"cloudInit": "cloud-init",
						"fmcIpAddress": "fmc-ip",
						"fmcRegistrationKey": "fmc-key",
						"fmcNatId": "fmc-nat"
					},
					"vnics": [{"description": "JSON VNIC", "vlan": 200}]
				}`,
			},
			expectedError: "--yes is required to confirm a purchase when using --json or --json-file",
		},
		{
			name:        "interactive combined with JSON is a usage error",
			interactive: true,
			flags: map[string]string{
				"json": `{
					"name": "JSON MVE",
					"term": 12,
					"locationId": 123,
					"vendorConfig": {
						"vendor": "cisco",
						"imageId": 1,
						"productSize": "LARGE",
						"mveLabel": "json-label",
						"manageLocally": true,
						"adminSshPublicKey": "admin-ssh",
						"sshPublicKey": "ssh-key",
						"cloudInit": "cloud-init",
						"fmcIpAddress": "fmc-ip",
						"fmcRegistrationKey": "fmc-key",
						"fmcNatId": "fmc-nat"
					},
					"vnics": [{"description": "JSON VNIC", "vlan": 200}]
				}`,
			},
			mockSetup:     func(m *MockMVEService) {},
			expectedError: "cannot be combined with",
		},
		{
			name:          "interactive combined with flags is a usage error",
			interactive:   true,
			flags:         map[string]string{"name": "Test MVE"},
			mockSetup:     func(m *MockMVEService) {},
			expectedError: "cannot be combined with",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.Reset()
			if tt.mockSetup != nil {
				tt.mockSetup(mockService)
			}

			promptIndex := 0
			next := func(_, msg string, _ bool) (string, error) {
				if promptIndex < len(tt.prompts) {
					response := tt.prompts[promptIndex]
					promptIndex++
					return response, nil
				}
				return "", fmt.Errorf("unexpected prompt call")
			}
			utils.SetResourcePrompt(next)
			utils.SetSecretResourcePrompt(next)

			cmd := &cobra.Command{Use: "buy"}
			cmd.Flags().Bool("interactive", tt.interactive, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("term", 0, "")
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().String("vendor-config", "", "")
			cmd.Flags().String("vnics", "", "")
			cmd.Flags().Bool("yes", false, "")

			testutil.SetFlags(t, cmd, tt.flags)

			var err error
			var capturedOutput string
			capturedStderr := captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = BuyMVE(cmd, tt.args, tt.interactive)
				})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, capturedOutput+capturedStderr, tt.expectedOutput)

				if tt.validateRequest != nil {
					tt.validateRequest(t, mockService.CapturedBuyMVERequest)
				}
			}
		})
	}
}

func TestBuyMVE_NoWaitFlag(t *testing.T) {
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
			mockService := &MockMVEService{}
			mockService.ValidateMVEOrderErr = nil
			mockService.BuyMVEResult = &megaport.BuyMVEResponse{
				TechnicalServiceUID: "mve-uid-123",
			}

			cleanup := testutil.SetupLogin(func(c *megaport.Client) {
				c.MVEService = mockService
			})
			defer cleanup()

			cmd := &cobra.Command{Use: "buy"}
			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().Bool("no-wait", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("term", 0, "")
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().String("vendor-config", "", "")
			cmd.Flags().String("vnics", "", "")

			testutil.SetFlags(t, cmd, map[string]string{
				"name":          "Test MVE",
				"term":          "12",
				"location-id":   "123",
				"vendor-config": `{"vendor":"cisco","imageId":1,"productSize":"LARGE","mveLabel":"label-1","manageLocally":true,"adminSshPublicKey":"admin-ssh","sshPublicKey":"ssh-key","cloudInit":"cloud-init","fmcIpAddress":"fmc-ip","fmcRegistrationKey":"fmc-key","fmcNatId":"fmc-nat"}`,
				"vnics":         `[{"description":"VNIC 1","vlan":100}]`,
			})
			if tt.noWait {
				assert.NoError(t, cmd.Flags().Set("no-wait", "true"))
			}

			var err error
			output.CaptureOutput(func() {
				err = BuyMVE(cmd, nil, false)
			})

			assert.NoError(t, err)
			assert.NotNil(t, mockService.CapturedBuyMVERequest)
			// The SDK must never poll; provisioning is awaited outside the order retry.
			assert.False(t, mockService.CapturedBuyMVERequest.WaitForProvision)
			if tt.expectPoll {
				assert.Positive(t, mockService.GetMVECallCount, "expected a provisioning poll when not using --no-wait")
			} else {
				assert.Zero(t, mockService.GetMVECallCount, "no provisioning poll expected with --no-wait")
			}
		})
	}
}

// TestBuyMVE_NoResubmitOnPollRetryableError proves the order is submitted exactly
// once even when a retryable error (429) surfaces during the provisioning poll.
func TestBuyMVE_NoResubmitOnPollRetryableError(t *testing.T) {
	originalMaxRetries := utils.MaxRetries
	originalPollInterval := utils.ProvisionPollInterval
	originalNoRetry := utils.NoRetry
	defer func() {
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

	mockService := &MockMVEService{}
	mockService.BuyMVEResult = &megaport.BuyMVEResponse{TechnicalServiceUID: "mve-uid-123"}
	mockService.GetMVEErr = &megaport.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     http.Header{},
			Request:    &http.Request{URL: &url.URL{}},
		},
		Message: "too many requests",
	}

	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.MVEService = mockService
	})
	defer cleanup()

	cmd := &cobra.Command{Use: "buy"}
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().Bool("no-wait", false, "")
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().Int("term", 0, "")
	cmd.Flags().Int("location-id", 0, "")
	cmd.Flags().String("vendor-config", "", "")
	cmd.Flags().String("vnics", "", "")

	testutil.SetFlags(t, cmd, map[string]string{
		"name":          "Test MVE",
		"term":          "12",
		"location-id":   "123",
		"vendor-config": `{"vendor":"cisco","imageId":1,"productSize":"LARGE","mveLabel":"label-1","manageLocally":true,"adminSshPublicKey":"admin-ssh","sshPublicKey":"ssh-key","cloudInit":"cloud-init","fmcIpAddress":"fmc-ip","fmcRegistrationKey":"fmc-key","fmcNatId":"fmc-nat"}`,
		"vnics":         `[{"description":"VNIC 1","vlan":100}]`,
	})
	// no-wait stays false: the provisioning poll must run.

	var err error
	output.CaptureOutput(func() {
		err = BuyMVE(cmd, nil, true)
	})

	assert.Error(t, err, "a retryable error during the provisioning poll must surface as an error")
	assert.Equal(t, 1, mockService.BuyMVECallCount, "the order must be submitted exactly once; a poll-phase 429 must not re-submit it")
}

func TestListMVEsCmd_WithMockClient(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	mves := []*megaport.MVE{
		{
			UID:                "mve-1",
			Name:               "TestMVE-1",
			LocationID:         123,
			ProvisioningStatus: "LIVE",
			Vendor:             "cisco",
		},
		{
			UID:                "mve-2",
			Name:               "TestMVE-2",
			LocationID:         456,
			ProvisioningStatus: "CONFIGURED",
			Vendor:             "fortinet",
		},
		{
			UID:                "mve-3",
			Name:               "MVE-Decommissioned",
			LocationID:         789,
			ProvisioningStatus: "DECOMMISSIONED",
			Vendor:             "versa",
		},
	}

	tests := []struct {
		name             string
		flags            map[string]string
		outputFormat     string
		setupMock        func(*MockMVEService)
		expectedError    string
		expectedOutput   []string
		unexpectedOutput []string
	}{
		{
			name:         "list all active mves",
			outputFormat: "table",
			setupMock: func(m *MockMVEService) {
				m.ListMVEsResult = mves
			},
			expectedOutput:   []string{"mve-1", "TestMVE-1", "mve-2", "TestMVE-2"},
			unexpectedOutput: []string{"mve-3", "MVE-Decommissioned", "DECOMMISSIONED"},
		},
		{
			name:         "list all mves including inactive",
			flags:        map[string]string{"include-inactive": "true"},
			outputFormat: "table",
			setupMock: func(m *MockMVEService) {
				m.ListMVEsResult = mves
			},
			expectedOutput: []string{"mve-1", "TestMVE-1", "mve-2", "TestMVE-2", "mve-3", "MVE-Decommissioned", "DECOMMISSIONED"},
		},
		{
			name:         "filter by location ID",
			flags:        map[string]string{"location-id": "123"},
			outputFormat: "table",
			setupMock: func(m *MockMVEService) {
				m.ListMVEsResult = mves
			},
			expectedOutput:   []string{"mve-1", "TestMVE-1"},
			unexpectedOutput: []string{"mve-2", "TestMVE-2", "mve-3"},
		},
		{
			name:         "filter by vendor",
			flags:        map[string]string{"vendor": "cisco"},
			outputFormat: "table",
			setupMock: func(m *MockMVEService) {
				m.ListMVEsResult = mves
			},
			expectedOutput:   []string{"mve-1", "TestMVE-1"},
			unexpectedOutput: []string{"mve-2", "fortinet", "mve-3"},
		},
		{
			name:         "filter by name",
			flags:        map[string]string{"name": "TestMVE"},
			outputFormat: "table",
			setupMock: func(m *MockMVEService) {
				m.ListMVEsResult = mves
			},
			expectedOutput:   []string{"mve-1", "TestMVE-1", "mve-2", "TestMVE-2"},
			unexpectedOutput: []string{"mve-3", "MVE-Decommissioned"},
		},
		{
			name: "combined filters",
			flags: map[string]string{
				"location-id": "123",
				"vendor":      "cisco",
				"name":        "TestMVE",
			},
			outputFormat: "table",
			setupMock: func(m *MockMVEService) {
				m.ListMVEsResult = mves
			},
			expectedOutput:   []string{"mve-1", "TestMVE-1"},
			unexpectedOutput: []string{"mve-2", "TestMVE-2", "mve-3"},
		},
		{
			name:         "json format",
			outputFormat: "json",
			setupMock: func(m *MockMVEService) {
				m.ListMVEsResult = mves
			},
			expectedOutput:   []string{`"uid": "mve-1"`, `"name": "TestMVE-1"`, `"uid": "mve-2"`, `"name": "TestMVE-2"`},
			unexpectedOutput: []string{`"uid": "mve-3"`},
		},
		{
			name:         "no matching mves",
			flags:        map[string]string{"location-id": "999"},
			outputFormat: "table",
			setupMock: func(m *MockMVEService) {
				m.ListMVEsResult = mves
			},
			expectedOutput:   []string{"No MVEs found. Create one with 'megaport mve buy'."},
			unexpectedOutput: []string{"mve-1", "mve-2", "mve-3"},
		},
		{
			name:         "API error",
			outputFormat: "table",
			setupMock: func(m *MockMVEService) {
				m.ListMVEsErr = fmt.Errorf("API error: service unavailable")
			},
			expectedError: "failed to list MVEs",
		},
		{
			name:         "empty result",
			outputFormat: "table",
			setupMock: func(m *MockMVEService) {
				m.ListMVEsResult = []*megaport.MVE{}
			},
			expectedOutput: []string{"No MVEs found. Create one with 'megaport mve buy'."},
		},
		{
			name:         "limit results",
			flags:        map[string]string{"limit": "1"},
			outputFormat: "table",
			setupMock: func(m *MockMVEService) {
				m.ListMVEsResult = mves
			},
			expectedOutput:   []string{"mve-1", "TestMVE-1"},
			unexpectedOutput: []string{"mve-2", "TestMVE-2"},
		},
		{
			name:         "negative limit returns error",
			flags:        map[string]string{"limit": "-1"},
			outputFormat: "table",
			setupMock: func(m *MockMVEService) {
				m.ListMVEsResult = mves
			},
			expectedError: "--limit must be a non-negative integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMVEService := &MockMVEService{}
			if tt.setupMock != nil {
				tt.setupMock(mockMVEService)
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				return &megaport.Client{
					MVEService: mockMVEService,
				}, nil
			})

			cmd := &cobra.Command{}
			cmd.Flags().Bool("include-inactive", false, "")
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().String("vendor", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("limit", 0, "")
			cmd.Flags().String("output", tt.outputFormat, "")

			testutil.SetFlags(t, cmd, tt.flags)
			err := cmd.Flags().Set("output", tt.outputFormat)
			assert.NoError(t, err)

			var out string
			capturedStderr := captureStderr(t, func() {
				out, err = output.CaptureOutputErr(func() error {
					return ListMVEs(cmd, []string{}, true, tt.outputFormat)
				})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)

				for _, expected := range tt.expectedOutput {
					assert.Contains(t, out+capturedStderr, expected)
				}

				for _, unexpected := range tt.unexpectedOutput {
					assert.NotContains(t, out, unexpected)
				}

				includeInactive, _ := cmd.Flags().GetBool("include-inactive")
				assert.Equal(t, includeInactive, mockMVEService.CapturedListMVEsRequest.IncludeInactive)
			}
		})
	}
}

func TestListMVEResourceTagsCmd_WithMockClient(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name          string
		mveUID        string
		setupMock     func(*MockMVEService)
		outputFormat  string
		expectedError string
		expectedOut   []string
	}{
		{
			name:         "successful list with multiple tags",
			mveUID:       "mve-123",
			outputFormat: "table",
			setupMock: func(m *MockMVEService) {
				m.ListMVEResourceTagsResult = map[string]string{
					"environment": "production",
					"team":        "networking",
					"project":     "sdwan-rollout",
				}
				m.ListMVEResourceTagsErr = nil
			},
			expectedOut: []string{"environment", "production", "team", "networking", "project", "sdwan-rollout"},
		},
		{
			name:         "successful list with no tags",
			mveUID:       "mve-456",
			outputFormat: "table",
			setupMock: func(m *MockMVEService) {
				m.ListMVEResourceTagsResult = map[string]string{}
				m.ListMVEResourceTagsErr = nil
			},
			expectedOut: []string{"KEY", "VALUE"},
		},
		{
			name:         "successful list with json format",
			mveUID:       "mve-789",
			outputFormat: "json",
			setupMock: func(m *MockMVEService) {
				m.ListMVEResourceTagsResult = map[string]string{
					"environment": "staging",
					"cost-center": "cc-123",
				}
				m.ListMVEResourceTagsErr = nil
			},
			expectedOut: []string{`"key": "environment"`, `"value": "staging"`, `"key": "cost-center"`, `"value": "cc-123"`},
		},
		{
			name:         "failed to fetch tags",
			mveUID:       "mve-error",
			outputFormat: "table",
			setupMock: func(m *MockMVEService) {
				m.ListMVEResourceTagsResult = make(map[string]string)
				m.ListMVEResourceTagsErr = fmt.Errorf("API error: not found")
			},
			expectedError: "failed to get resource tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMVEService := &MockMVEService{}
			tt.setupMock(mockMVEService)

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.MVEService = mockMVEService
				return client, nil
			})

			cmd := &cobra.Command{
				Use: "list-tags [mveUID]",
				RunE: func(cmd *cobra.Command, args []string) error {
					return ListMVEResourceTags(cmd, args, false, tt.outputFormat)
				},
			}

			cmd.Flags().StringP("output", "o", "table", "Output format (json, table)")
			if tt.outputFormat != "" {
				err := cmd.Flags().Set("output", tt.outputFormat)
				if err != nil {
					t.Fatalf("Failed to set output format: %v", err)
				}
			}

			var err error
			output := output.CaptureOutput(func() {
				err = cmd.RunE(cmd, []string{tt.mveUID})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				for _, expected := range tt.expectedOut {
					assert.Contains(t, output, expected)
				}

				if len(tt.expectedOut) == 0 && tt.expectedError == "" {
					assert.NotContains(t, output, "Error")
				}
			}
		})
	}
}

func TestUpdateMVEResourceTagsCmd_WithMockClient(t *testing.T) {
	originalResourcePrompt := utils.GetUpdateResourceTagsPrompt()
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer func() {
		cleanup()
		utils.SetUpdateResourceTagsPrompt(originalResourcePrompt)
	}()

	tests := []struct {
		name                 string
		mveUID               string
		interactive          bool
		force                bool
		promptResult         map[string]string
		promptError          error
		jsonInput            string
		jsonFile             string
		setupMock            func(*MockMVEService)
		expectedError        string
		expectedOutput       string
		expectedCapturedTags map[string]string
	}{
		{
			name:        "successful update with interactive mode",
			mveUID:      "mve-123",
			interactive: true,
			promptResult: map[string]string{
				"environment": "production",
				"team":        "networking",
			},
			setupMock: func(m *MockMVEService) {
				m.ListMVEResourceTagsResult = map[string]string{
					"environment": "staging",
				}
				m.ListMVEResourceTagsErr = nil
				m.UpdateMVEResourceTagsErr = nil
				m.CapturedUpdateMVEResourceTagsRequest = make(map[string]string)
			},
			expectedOutput: "Resource tags updated for MVE mve-123",
			expectedCapturedTags: map[string]string{
				"environment": "production",
				"team":        "networking",
			},
		},
		{
			name:   "successful update with json",
			mveUID: "mve-456",
			force:  true,
			jsonInput: `{
				"environment": "production",
				"project": "sdwan-rollout"
			}`,
			setupMock: func(m *MockMVEService) {
				m.ListMVEResourceTagsResult = map[string]string{
					"environment": "development",
					"owner":       "john.doe@example.com",
				}
				m.ListMVEResourceTagsErr = nil
				m.UpdateMVEResourceTagsErr = nil
				m.CapturedUpdateMVEResourceTagsRequest = make(map[string]string)
			},
			expectedOutput: "Resource tags updated for MVE mve-456",
			expectedCapturedTags: map[string]string{
				"environment": "production",
				"project":     "sdwan-rollout",
			},
		},
		{
			name:      "error with invalid json",
			mveUID:    "mve-789",
			jsonInput: `{invalid json}`,
			setupMock: func(m *MockMVEService) {
				m.ListMVEResourceTagsResult = map[string]string{}
				m.CapturedUpdateMVEResourceTagsRequest = make(map[string]string)
			},
			expectedError: "failed to parse JSON",
		},
		{
			name:        "error in interactive mode",
			mveUID:      "mve-prompt-error",
			interactive: true,
			promptError: fmt.Errorf("user cancelled input"),
			setupMock: func(m *MockMVEService) {
				m.ListMVEResourceTagsResult = map[string]string{
					"environment": "staging",
				}
				m.ListMVEResourceTagsErr = nil
				m.CapturedUpdateMVEResourceTagsRequest = make(map[string]string)
			},
			expectedError: "user cancelled input",
		},
		{
			name:   "error with API update",
			mveUID: "mve-update-error",
			jsonInput: `{
				"environment": "production"
			}`,
			setupMock: func(m *MockMVEService) {
				m.ListMVEResourceTagsResult = map[string]string{}
				m.ListMVEResourceTagsErr = nil
				m.UpdateMVEResourceTagsErr = fmt.Errorf("API error: unauthorized")
				m.CapturedUpdateMVEResourceTagsRequest = make(map[string]string)
			},
			expectedError: "failed to update resource tags",
		},
		{
			name:   "error with API tag listing",
			mveUID: "mve-list-error",
			jsonInput: `{
				"environment": "production"
			}`,
			setupMock: func(m *MockMVEService) {
				m.ListMVEResourceTagsResult = map[string]string{}
				m.ListMVEResourceTagsErr = fmt.Errorf("API error: resource not found")
				m.CapturedUpdateMVEResourceTagsRequest = make(map[string]string)
			},
			expectedError: "failed to log in or list existing resource tags",
		},
		{
			name:      "empty tags clear all existing tags",
			mveUID:    "mve-clear-tags",
			force:     true,
			jsonInput: `{}`,
			setupMock: func(m *MockMVEService) {
				m.ListMVEResourceTagsResult = map[string]string{
					"environment": "staging",
					"team":        "networking",
				}
				m.ListMVEResourceTagsErr = nil
				m.UpdateMVEResourceTagsErr = nil
				m.CapturedUpdateMVEResourceTagsRequest = make(map[string]string)
			},
			expectedOutput:       "No tags provided. The MVE will have all existing tags removed",
			expectedCapturedTags: map[string]string{},
		},
		{
			name:   "no input provided",
			mveUID: "mve-no-input",
			setupMock: func(m *MockMVEService) {
				m.ListMVEResourceTagsResult = map[string]string{}
				m.ListMVEResourceTagsErr = nil
				m.CapturedUpdateMVEResourceTagsRequest = make(map[string]string)
			},
			expectedError: "no input provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMVEService := &MockMVEService{}
			if tt.setupMock != nil {
				tt.setupMock(mockMVEService)
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.MVEService = mockMVEService
				return client, nil
			})

			utils.SetUpdateResourceTagsPrompt(func(existingTags map[string]string, noColor bool) (map[string]string, error) {
				return tt.promptResult, tt.promptError
			})

			cmd := &cobra.Command{
				Use: "update-tags [mveUID]",
				RunE: func(cmd *cobra.Command, args []string) error {
					return UpdateMVEResourceTags(cmd, args, false)
				},
			}

			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().Bool("force", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")

			if tt.interactive {
				err := cmd.Flags().Set("interactive", "true")
				if err != nil {
					t.Fatalf("Failed to set interactive flag: %v", err)
				}
			}

			if tt.force {
				err := cmd.Flags().Set("force", "true")
				if err != nil {
					t.Fatalf("Failed to set force flag: %v", err)
				}
			}

			if tt.jsonInput != "" {
				err := cmd.Flags().Set("json", tt.jsonInput)
				if err != nil {
					t.Fatalf("Failed to set json flag: %v", err)
				}
			}

			if tt.jsonFile != "" {
				err := cmd.Flags().Set("json-file", tt.jsonFile)
				if err != nil {
					t.Fatalf("Failed to set json-file flag: %v", err)
				}
			}

			var err error
			var capturedOutput string
			capturedStderr := captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = cmd.RunE(cmd, []string{tt.mveUID})
				})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, capturedOutput+capturedStderr, tt.expectedOutput)

				if tt.expectedCapturedTags != nil {
					assert.NotNil(t, mockMVEService.CapturedUpdateMVEResourceTagsRequest)
					assert.Equal(t, tt.expectedCapturedTags, mockMVEService.CapturedUpdateMVEResourceTagsRequest)
				}
			}
		})
	}
}

func TestGetMVEStatus(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name           string
		mveUID         string
		setupMock      func(*MockMVEService)
		expectedError  string
		expectedOutput string
		outputFormat   string
	}{
		{
			name:   "successful status retrieval - table format",
			mveUID: "mve-123abc",
			setupMock: func(m *MockMVEService) {
				m.GetMVEResult = &megaport.MVE{
					UID:                "mve-123abc",
					Name:               "Test MVE",
					ProvisioningStatus: "CONFIGURED",
					Vendor:             "cisco",
					Size:               "MEDIUM",
				}
			},
			expectedOutput: "mve-123abc",
			outputFormat:   "table",
		},
		{
			name:   "successful status retrieval - json format",
			mveUID: "mve-123abc",
			setupMock: func(m *MockMVEService) {
				m.GetMVEResult = &megaport.MVE{
					UID:                "mve-123abc",
					Name:               "Test MVE",
					ProvisioningStatus: "LIVE",
					Vendor:             "fortinet",
					Size:               "LARGE",
				}
			},
			expectedOutput: "mve-123abc",
			outputFormat:   "json",
		},
		{
			name:   "MVE not found",
			mveUID: "mve-notfound",
			setupMock: func(m *MockMVEService) {
				m.GetMVEErr = fmt.Errorf("MVE not found")
			},
			expectedError: "failed to get MVE status",
			outputFormat:  "table",
		},
		{
			name:   "API error",
			mveUID: "mve-error",
			setupMock: func(m *MockMVEService) {
				m.GetMVEErr = fmt.Errorf("API error")
			},
			expectedError: "API error",
			outputFormat:  "table",
		},
		{
			name:   "nil MVE returned without error",
			mveUID: "mve-nil",
			setupMock: func(m *MockMVEService) {
				m.ForceNilGetMVE = true
			},
			expectedError: "no MVE found",
			outputFormat:  "table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockMVEService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.MVEService = mockService
				return client, nil
			})

			cmd := &cobra.Command{
				Use: "status [mveUID]",
			}

			var err error
			capturedOutput := output.CaptureOutput(func() {
				err = GetMVEStatus(cmd, []string{tt.mveUID}, true, tt.outputFormat)
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
					assert.Contains(t, capturedOutput, "\"vendor\":")
				case "table":
					assert.Contains(t, capturedOutput, "UID")
					assert.Contains(t, capturedOutput, "NAME")
					assert.Contains(t, capturedOutput, "STATUS")
					assert.Contains(t, capturedOutput, "VENDOR")
				}
			}
		})
	}
}

func TestGetMVE(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name           string
		mveUID         string
		setupMock      func(m *MockMVEService)
		outputFormat   string
		expectedError  string
		expectedOutput string
	}{
		{
			name:   "success table format",
			mveUID: "mve-uid-123",
			setupMock: func(m *MockMVEService) {
				m.GetMVEResult = &megaport.MVE{
					UID:                "mve-uid-123",
					Name:               "Test MVE",
					ProvisioningStatus: "LIVE",
					Vendor:             "cisco",
					Size:               "MEDIUM",
				}
			},
			outputFormat:   "table",
			expectedOutput: "mve-uid-123",
		},
		{
			name:   "success JSON format",
			mveUID: "mve-uid-456",
			setupMock: func(m *MockMVEService) {
				m.GetMVEResult = &megaport.MVE{
					UID:                "mve-uid-456",
					Name:               "JSON MVE",
					ProvisioningStatus: "LIVE",
					Vendor:             "fortinet",
					Size:               "LARGE",
				}
			},
			outputFormat:   "json",
			expectedOutput: "mve-uid-456",
		},
		{
			name:   "API error",
			mveUID: "mve-error",
			setupMock: func(m *MockMVEService) {
				m.GetMVEErr = fmt.Errorf("API failure")
			},
			outputFormat:  "table",
			expectedError: "failed to get MVE",
		},
		{
			name:   "nil MVE",
			mveUID: "mve-nil",
			setupMock: func(m *MockMVEService) {
				m.ForceNilGetMVE = true
			},
			outputFormat:  "table",
			expectedError: "no MVE found with UID: mve-nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockMVEService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.MVEService = mockService
				return client, nil
			})

			cmd := &cobra.Command{
				Use: "get [mveUID]",
			}

			defer output.SetOutputFormat("table")

			var err error
			capturedOutput := output.CaptureOutput(func() {
				err = GetMVE(cmd, []string{tt.mveUID}, noColor, tt.outputFormat)
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.outputFormat == "json" {
					var parsed []map[string]interface{}
					assert.NoError(t, json.Unmarshal([]byte(capturedOutput), &parsed), "JSON output should be valid JSON")
					if assert.NotEmpty(t, parsed) {
						assert.Equal(t, tt.mveUID, parsed[0]["uid"])
					}
				} else {
					assert.Contains(t, capturedOutput, tt.expectedOutput)
				}
			}
		})
	}
}

func TestLockMVECmd_WithMockClient(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name          string
		mveID         string
		lockErr       error
		expectedError string
		expectedOut   string
	}{
		{
			name:        "lock MVE success",
			mveID:       "mve-to-lock",
			expectedOut: "MVE mve-to-lock locked successfully",
		},
		{
			name:          "lock MVE error",
			mveID:         "mve-error",
			lockErr:       fmt.Errorf("failed to lock MVE"),
			expectedError: "failed to lock MVE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origFunc := lockMVEFunc
			defer func() { lockMVEFunc = origFunc }()

			lockMVEFunc = func(ctx context.Context, client *megaport.Client, mveUID string) (*megaport.ManageProductLockResponse, error) {
				if tt.lockErr != nil {
					return nil, tt.lockErr
				}
				return &megaport.ManageProductLockResponse{}, nil
			}

			lockMVECmd := &cobra.Command{
				Use: "lock [mveUID]",
				RunE: func(cmd *cobra.Command, args []string) error {
					return LockMVE(cmd, args, false)
				},
			}

			var err error
			var capturedOutput string
			capturedStderr := captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = lockMVECmd.RunE(lockMVECmd, []string{tt.mveID})
				})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, capturedOutput+capturedStderr, tt.expectedOut)
			}
		})
	}
}

func TestUnlockMVECmd_WithMockClient(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name          string
		mveID         string
		unlockErr     error
		expectedError string
		expectedOut   string
	}{
		{
			name:        "unlock MVE success",
			mveID:       "mve-to-unlock",
			expectedOut: "MVE mve-to-unlock unlocked successfully",
		},
		{
			name:          "unlock MVE error",
			mveID:         "mve-error",
			unlockErr:     fmt.Errorf("failed to unlock MVE"),
			expectedError: "failed to unlock MVE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origFunc := unlockMVEFunc
			defer func() { unlockMVEFunc = origFunc }()

			unlockMVEFunc = func(ctx context.Context, client *megaport.Client, mveUID string) (*megaport.ManageProductLockResponse, error) {
				if tt.unlockErr != nil {
					return nil, tt.unlockErr
				}
				return &megaport.ManageProductLockResponse{}, nil
			}

			unlockMVECmd := &cobra.Command{
				Use: "unlock [mveUID]",
				RunE: func(cmd *cobra.Command, args []string) error {
					return UnlockMVE(cmd, args, false)
				},
			}

			var err error
			var capturedOutput string
			capturedStderr := captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = unlockMVECmd.RunE(unlockMVECmd, []string{tt.mveID})
				})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, capturedOutput+capturedStderr, tt.expectedOut)
			}
		})
	}
}

func TestLockMVECmd_LoginError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	defer func() { config.SetLoginFunc(originalLoginFunc) }()
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return nil, fmt.Errorf("login failed")
	})

	cmd := &cobra.Command{}
	err := LockMVE(cmd, []string{"mve-123"}, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to log in")
}

func TestUnlockMVECmd_LoginError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	defer func() { config.SetLoginFunc(originalLoginFunc) }()
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return nil, fmt.Errorf("login failed")
	})

	cmd := &cobra.Command{}
	err := UnlockMVE(cmd, []string{"mve-123"}, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to log in")
}

func TestBuyMVE_Confirmation(t *testing.T) {
	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()

	tests := []struct {
		name                 string
		flags                map[string]string
		confirmResult        bool
		expectBuyCalled      bool
		expectedOutput       string
		expectedError        string
		promptShouldBeCalled bool
	}{
		{
			name: "confirmation accepted",
			flags: map[string]string{
				"name":          "Test MVE",
				"term":          "12",
				"location-id":   "123",
				"vendor-config": `{"vendor":"cisco","imageId":1,"productSize":"LARGE","mveLabel":"label-1","manageLocally":true,"adminSshPublicKey":"admin-ssh","sshPublicKey":"ssh-key","cloudInit":"cloud-init","fmcIpAddress":"fmc-ip","fmcRegistrationKey":"fmc-key","fmcNatId":"fmc-nat"}`,
				"vnics":         `[{"description":"VNIC 1","vlan":100}]`,
			},
			confirmResult:        true,
			expectBuyCalled:      true,
			expectedOutput:       "MVE created",
			promptShouldBeCalled: true,
		},
		{
			name: "confirmation denied",
			flags: map[string]string{
				"name":          "Test MVE",
				"term":          "12",
				"location-id":   "123",
				"vendor-config": `{"vendor":"cisco","imageId":1,"productSize":"LARGE","mveLabel":"label-1","manageLocally":true,"adminSshPublicKey":"admin-ssh","sshPublicKey":"ssh-key","cloudInit":"cloud-init","fmcIpAddress":"fmc-ip","fmcRegistrationKey":"fmc-key","fmcNatId":"fmc-nat"}`,
				"vnics":         `[{"description":"VNIC 1","vlan":100}]`,
			},
			confirmResult:        false,
			expectBuyCalled:      false,
			expectedError:        "cancelled by user",
			promptShouldBeCalled: true,
		},
		{
			name: "yes flag skips confirmation",
			flags: map[string]string{
				"name":          "Test MVE",
				"term":          "12",
				"location-id":   "123",
				"vendor-config": `{"vendor":"cisco","imageId":1,"productSize":"LARGE","mveLabel":"label-1","manageLocally":true,"adminSshPublicKey":"admin-ssh","sshPublicKey":"ssh-key","cloudInit":"cloud-init","fmcIpAddress":"fmc-ip","fmcRegistrationKey":"fmc-key","fmcNatId":"fmc-nat"}`,
				"vnics":         `[{"description":"VNIC 1","vlan":100}]`,
				"yes":           "true",
			},
			confirmResult:        false,
			expectBuyCalled:      true,
			expectedOutput:       "MVE created",
			promptShouldBeCalled: false,
		},
		{
			name: "json input with yes skips confirmation",
			flags: map[string]string{
				"json": `{"name":"JSON MVE","term":12,"locationId":123,"vendorConfig":{"vendor":"cisco","imageId":1,"productSize":"LARGE","mveLabel":"label-1","manageLocally":true,"adminSshPublicKey":"admin-ssh","sshPublicKey":"ssh-key","cloudInit":"cloud-init","fmcIpAddress":"fmc-ip","fmcRegistrationKey":"fmc-key","fmcNatId":"fmc-nat"},"vnics":[{"description":"VNIC 1","vlan":100}]}`,
				"yes":  "true",
			},
			confirmResult:        false,
			expectBuyCalled:      true,
			expectedOutput:       "MVE created",
			promptShouldBeCalled: false,
		},
		{
			name: "json input without yes is a usage error",
			flags: map[string]string{
				"json": `{"name":"JSON MVE","term":12,"locationId":123,"vendorConfig":{"vendor":"cisco","imageId":1,"productSize":"LARGE","mveLabel":"label-1","manageLocally":true,"adminSshPublicKey":"admin-ssh","sshPublicKey":"ssh-key","cloudInit":"cloud-init","fmcIpAddress":"fmc-ip","fmcRegistrationKey":"fmc-key","fmcNatId":"fmc-nat"},"vnics":[{"description":"VNIC 1","vlan":100}]}`,
			},
			expectBuyCalled:      false,
			expectedError:        "--yes is required to confirm a purchase when using --json or --json-file",
			promptShouldBeCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockMVEService{
				BuyMVEResult: &megaport.BuyMVEResponse{
					TechnicalServiceUID: "mve-confirm-123",
				},
			}

			cleanup := testutil.SetupLogin(func(c *megaport.Client) {
				c.MVEService = mockService
			})
			defer cleanup()

			promptCalled := false
			utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool {
				promptCalled = true
				return tt.confirmResult
			})

			cmd := &cobra.Command{Use: "buy"}
			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().BoolP("yes", "y", false, "")
			cmd.Flags().Bool("no-wait", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("term", 0, "")
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().String("vendor-config", "", "")
			cmd.Flags().String("vnics", "", "")

			testutil.SetFlags(t, cmd, tt.flags)

			var err error
			var capturedOutput string
			capturedStderr := captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = BuyMVE(cmd, nil, noColor)
				})
			})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, capturedOutput+capturedStderr, tt.expectedOutput)
			}

			if tt.expectBuyCalled {
				assert.NotNil(t, mockService.CapturedBuyMVERequest, "expected BuyMVE to be called")
			} else {
				assert.Nil(t, mockService.CapturedBuyMVERequest, "expected BuyMVE not to be called")
			}
			assert.Equal(t, tt.promptShouldBeCalled, promptCalled, "BuyConfirmPrompt called expectation mismatch")
		})
	}
}

func TestValidateMVE(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name             string
		flags            map[string]string
		jsonInput        string
		jsonFileContent  string
		setupMock        func(*MockMVEService)
		loginError       error
		expectedError    string
		expectedContains string
	}{
		{
			name: "success with flags",
			flags: map[string]string{
				"name":          "test-mve",
				"term":          "12",
				"location-id":   "1",
				"vendor-config": `{"vendor":"cisco","imageId":1,"productSize":"MEDIUM","mveLabel":"test-label","manageLocally":true,"adminSshPublicKey":"ssh-rsa AAAA","sshPublicKey":"ssh-rsa AAAA","cloudInit":"#cloud-config","fmcIpAddress":"10.0.0.1","fmcRegistrationKey":"reg-key","fmcNatId":"nat-id"}`,
				"vnics":         `[{"description":"Data Plane","vlan":100}]`,
			},
			setupMock:        func(m *MockMVEService) {},
			expectedContains: "validation passed",
		},
		{
			name:             "success with JSON",
			jsonInput:        `{"name":"test-mve","term":12,"locationId":1,"vendorConfig":{"vendor":"cisco","imageId":1,"productSize":"MEDIUM","mveLabel":"test-label","manageLocally":true,"adminSshPublicKey":"ssh-rsa AAAA","sshPublicKey":"ssh-rsa AAAA","cloudInit":"#cloud-config","fmcIpAddress":"10.0.0.1","fmcRegistrationKey":"reg-key","fmcNatId":"nat-id"},"vnics":[{"description":"Data Plane","vlan":100}]}`,
			setupMock:        func(m *MockMVEService) {},
			expectedContains: "validation passed",
		},
		{
			name:      "validation error",
			jsonInput: `{"name":"test-mve","term":12,"locationId":1,"vendorConfig":{"vendor":"cisco","imageId":1,"productSize":"MEDIUM","mveLabel":"test-label","manageLocally":true,"adminSshPublicKey":"ssh-rsa AAAA","sshPublicKey":"ssh-rsa AAAA","cloudInit":"#cloud-config","fmcIpAddress":"10.0.0.1","fmcRegistrationKey":"reg-key","fmcNatId":"nat-id"},"vnics":[{"description":"Data Plane","vlan":100}]}`,
			setupMock: func(m *MockMVEService) {
				m.ValidateMVEOrderErr = fmt.Errorf("invalid MVE configuration")
			},
			expectedError: "invalid MVE configuration",
		},
		{
			name:          "no input provided",
			setupMock:     func(m *MockMVEService) {},
			expectedError: "no input provided",
		},
		{
			name:          "login error",
			jsonInput:     `{"name":"test-mve","term":12,"locationId":1,"vendorConfig":{"vendor":"cisco","imageId":1,"productSize":"MEDIUM","mveLabel":"test-label","manageLocally":true,"adminSshPublicKey":"ssh-rsa AAAA","sshPublicKey":"ssh-rsa AAAA","cloudInit":"#cloud-config","fmcIpAddress":"10.0.0.1","fmcRegistrationKey":"reg-key","fmcNatId":"nat-id"},"vnics":[{"description":"Data Plane","vlan":100}]}`,
			setupMock:     func(m *MockMVEService) {},
			loginError:    fmt.Errorf("authentication failed"),
			expectedError: "authentication failed",
		},
		{
			name:          "invalid JSON input",
			jsonInput:     `{invalid json}`,
			setupMock:     func(m *MockMVEService) {},
			expectedError: "failed to parse JSON",
		},
		{
			name:          "vendor config validation failure",
			jsonInput:     `{"name":"test-mve","term":12,"locationId":1,"vendorConfig":{"vendor":"unknown_vendor","imageId":1,"productSize":"MEDIUM"},"vnics":[{"description":"Data Plane","vlan":100}]}`,
			setupMock:     func(m *MockMVEService) {},
			expectedError: "unsupported vendor",
		},
		{
			name:             "success with JSON file",
			jsonFileContent:  `{"name":"file-mve","term":12,"locationId":1,"vendorConfig":{"vendor":"cisco","imageId":1,"productSize":"MEDIUM","mveLabel":"test-label","manageLocally":true,"adminSshPublicKey":"ssh-rsa AAAA","sshPublicKey":"ssh-rsa AAAA","cloudInit":"#cloud-config","fmcIpAddress":"10.0.0.1","fmcRegistrationKey":"reg-key","fmcNatId":"nat-id"},"vnics":[{"description":"Data Plane","vlan":100}]}`,
			setupMock:        func(m *MockMVEService) {},
			expectedContains: "validation passed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockMVEService{}
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
					client.MVEService = mockService
					return client, nil
				})
			}

			cmd := &cobra.Command{Use: "validate"}
			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("term", 0, "")
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().String("vendor-config", "", "")
			cmd.Flags().String("vnics", "", "")
			cmd.Flags().String("diversity-zone", "", "")
			cmd.Flags().String("promo-code", "", "")
			cmd.Flags().String("cost-centre", "", "")
			cmd.Flags().String("resource-tags", "", "")

			if tt.jsonInput != "" {
				assert.NoError(t, cmd.Flags().Set("json", tt.jsonInput))
			}
			if tt.jsonFileContent != "" {
				tmpFile, tmpErr := os.CreateTemp("", "mve-validate-*.json")
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
			var capturedOutput string
			capturedStderr := captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					err = ValidateMVE(cmd, nil, true)
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

func TestListMVEImages_NilResult(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	mockService := &MockMVEService{}
	// Force nil return by not setting result and not setting error
	// The mock returns empty slice by default, so we need to override
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.MVEService = mockService
		return client, nil
	})

	mockService.ListMVEImagesErr = fmt.Errorf("API failure")

	cmd := testutil.NewCommand("list-images", testutil.OutputAdapter(ListMVEImages))
	cmd.Flags().String("vendor", "", "")
	cmd.Flags().String("product-code", "", "")
	cmd.Flags().Int("id", 0, "")
	cmd.Flags().String("version", "", "")
	cmd.Flags().Bool("release-image", false, "")

	var err error
	output.CaptureOutput(func() {
		err = testutil.OutputAdapter(ListMVEImages)(cmd, nil)
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API failure")
}

func TestListMVEImages_LoginError(t *testing.T) {
	cleanup := testutil.SetupLoginError(fmt.Errorf("auth failed"))
	defer cleanup()

	cmd := testutil.NewCommand("list-images", testutil.OutputAdapter(ListMVEImages))
	cmd.Flags().String("vendor", "", "")
	cmd.Flags().String("product-code", "", "")
	cmd.Flags().Int("id", 0, "")
	cmd.Flags().String("version", "", "")
	cmd.Flags().Bool("release-image", false, "")

	var err error
	output.CaptureOutput(func() {
		err = testutil.OutputAdapter(ListMVEImages)(cmd, nil)
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth failed")
}

func TestListAvailableMVESizes_Error(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	mockService := &MockMVEService{}
	mockService.ListAvailableMVESizesErr = fmt.Errorf("API failure")

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.MVEService = mockService
		return client, nil
	})

	cmd := testutil.NewCommand("list-sizes", testutil.OutputAdapter(ListAvailableMVESizes))

	var err error
	output.CaptureOutput(func() {
		err = testutil.OutputAdapter(ListAvailableMVESizes)(cmd, nil)
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API failure")
}

func TestExportMVEConfig(t *testing.T) {
	mve := &megaport.MVE{
		UID:                "mve-should-not-appear",
		Name:               "My MVE",
		ContractTermMonths: 12,
		LocationID:         55,
		DiversityZone:      "green",
		CostCentre:         "EdgeOps",
		ProvisioningStatus: "LIVE",
		NetworkInterfaces: []*megaport.MVENetworkInterface{
			{Description: "eth0", VLAN: 100},
			{Description: "eth1"},
		},
	}
	m := exportMVEConfig(mve)

	assert.Equal(t, "My MVE", m["name"])
	assert.Equal(t, 12, m["term"])
	assert.Equal(t, 55, m["locationId"])
	assert.Equal(t, "green", m["diversityZone"])
	assert.Equal(t, "EdgeOps", m["costCentre"])

	vnics, ok := m["vnics"].([]map[string]interface{})
	assert.True(t, ok)
	assert.Len(t, vnics, 2)
	assert.Equal(t, "eth0", vnics[0]["description"])
	assert.Equal(t, 100, vnics[0]["vlan"])
	assert.Equal(t, "eth1", vnics[1]["description"])
	_, hasVLAN := vnics[1]["vlan"]
	assert.False(t, hasVLAN, "zero VLAN should be omitted")

	_, hasUID := m["productUid"]
	assert.False(t, hasUID, "export should not include productUid")
	_, hasVendor := m["vendorConfig"]
	assert.False(t, hasVendor, "vendorConfig is not available from API")
}

func TestGetMVE_Export(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	mockService := &MockMVEService{
		GetMVEResult: &megaport.MVE{
			UID:                "mve-export-123",
			Name:               "Export MVE",
			ContractTermMonths: 12,
			LocationID:         55,
			ProvisioningStatus: "LIVE",
			NetworkInterfaces: []*megaport.MVENetworkInterface{
				{Description: "Data"},
			},
		},
	}
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.MVEService = mockService
		return client, nil
	})

	cmd := &cobra.Command{Use: "get"}
	cmd.Flags().Bool("export", false, "")
	assert.NoError(t, cmd.Flags().Set("export", "true"))

	var err error
	capturedOutput := output.CaptureStdout(func() {
		err = GetMVE(cmd, []string{"mve-export-123"}, true, "table")
	})

	assert.NoError(t, err)
	var parsed map[string]interface{}
	assert.NoError(t, json.Unmarshal([]byte(capturedOutput), &parsed), "export output must be valid JSON")
	assert.Equal(t, "Export MVE", parsed["name"])
	assert.Equal(t, float64(55), parsed["locationId"])
	_, hasUID := parsed["productUid"]
	assert.False(t, hasUID, "export should not include productUid")
}

func TestListAvailableMVESizes_LoginError(t *testing.T) {
	cleanup := testutil.SetupLoginError(fmt.Errorf("auth failed"))
	defer cleanup()

	cmd := testutil.NewCommand("list-sizes", testutil.OutputAdapter(ListAvailableMVESizes))

	var err error
	output.CaptureOutput(func() {
		err = testutil.OutputAdapter(ListAvailableMVESizes)(cmd, nil)
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth failed")
}

func TestListMVEs_TagFilter(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	origListTagsFunc := listMVEResourceTagsFunc
	defer func() { listMVEResourceTagsFunc = origListTagsFunc }()

	allMVEs := []*megaport.MVE{
		{UID: "mve-1", Name: "MVE-Alpha", ProvisioningStatus: "LIVE"},
		{UID: "mve-2", Name: "MVE-Beta", ProvisioningStatus: "LIVE"},
		{UID: "mve-3", Name: "MVE-Gamma", ProvisioningStatus: "LIVE"},
	}

	tagsByUID := map[string]map[string]string{
		"mve-1": {"env": "prod", "team": "net"},
		"mve-2": {"env": "staging"},
		"mve-3": {},
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
			expectedOutputs: []string{"mve-1", "mve-2", "mve-3"},
		},
		{
			name:            "exact match includes only matching resource",
			tagFilters:      []string{"env=prod"},
			expectedOutputs: []string{"mve-1"},
			notExpected:     []string{"mve-2", "mve-3"},
		},
		{
			name:            "AND logic requires all filters to match",
			tagFilters:      []string{"env=prod", "team=net"},
			expectedOutputs: []string{"mve-1"},
			notExpected:     []string{"mve-2", "mve-3"},
		},
		{
			name:            "key-exists match",
			tagFilters:      []string{"env"},
			expectedOutputs: []string{"mve-1", "mve-2"},
			notExpected:     []string{"mve-3"},
		},
		{
			name:        "no match returns empty",
			tagFilters:  []string{"env=nonexistent"},
			notExpected: []string{"mve-1", "mve-2", "mve-3"},
		},
		{
			name:            "tag fetch error excludes resource gracefully",
			tagFilters:      []string{"env=prod"},
			tagFetchErrUID:  "mve-1",
			notExpected:     []string{"MVE-Alpha"}, // name absent from table; UID appears in warning
			expectedOutputs: []string{"Failed to fetch tags"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listMVEResourceTagsFunc = func(ctx context.Context, client *megaport.Client, uid string) (map[string]string, error) {
				if tt.tagFetchErrUID != "" && uid == tt.tagFetchErrUID {
					return nil, fmt.Errorf("tag fetch error")
				}
				return tagsByUID[uid], nil
			}

			mockSvc := &MockMVEService{ListMVEsResult: allMVEs}
			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				return &megaport.Client{MVEService: mockSvc}, nil
			})

			cmd := &cobra.Command{Use: "list"}
			cmd.Flags().Bool("include-inactive", false, "")
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().String("vendor", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("limit", 0, "")
			cmd.Flags().StringArray("tag", nil, "")

			for _, f := range tt.tagFilters {
				assert.NoError(t, cmd.Flags().Set("tag", f))
			}

			var listErr error
			var capturedOutput string
			capturedStderr := captureStderr(t, func() {
				capturedOutput = output.CaptureOutput(func() {
					listErr = ListMVEs(cmd, nil, true, "table")
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

// TestBuyMVE_OrderAndProvisionFailures covers the order-response and
// provisioning-poll failure branches: an empty API response, a missing UID, and
// the MVE vanishing during the provisioning poll.
func TestBuyMVE_OrderAndProvisionFailures(t *testing.T) {
	originalBuyConfirmPrompt := utils.GetBuyConfirmPrompt()
	defer func() { utils.SetBuyConfirmPrompt(originalBuyConfirmPrompt) }()
	utils.SetBuyConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return true })

	tests := []struct {
		name     string
		nilResp  bool
		buyResp  *megaport.BuyMVEResponse
		getNil   bool
		errMatch string
	}{
		{name: "empty API response", nilResp: true, errMatch: "empty response from API"},
		{name: "no UID returned", buyResp: &megaport.BuyMVEResponse{}, errMatch: "no UID returned"},
		{name: "MVE not found during provisioning poll", buyResp: &megaport.BuyMVEResponse{TechnicalServiceUID: "mve-uid-123"}, getNil: true, errMatch: "not found while waiting for provisioning"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockMVEService{
				BuyMVENilResp:  tt.nilResp,
				BuyMVEResult:   tt.buyResp,
				ForceNilGetMVE: tt.getNil,
			}
			cleanup := testutil.SetupLogin(func(c *megaport.Client) {
				c.MVEService = mockService
			})
			defer cleanup()

			cmd := &cobra.Command{Use: "buy"}
			cmd.Flags().Bool("interactive", false, "")
			cmd.Flags().Bool("no-wait", false, "")
			cmd.Flags().String("json", "", "")
			cmd.Flags().String("json-file", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("term", 0, "")
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().String("vendor-config", "", "")
			cmd.Flags().String("vnics", "", "")

			testutil.SetFlags(t, cmd, map[string]string{
				"name":          "Test MVE",
				"term":          "12",
				"location-id":   "123",
				"vendor-config": `{"vendor":"cisco","imageId":1,"productSize":"LARGE","mveLabel":"label-1","manageLocally":true,"adminSshPublicKey":"admin-ssh","sshPublicKey":"ssh-key","cloudInit":"cloud-init","fmcIpAddress":"fmc-ip","fmcRegistrationKey":"fmc-key","fmcNatId":"fmc-nat"}`,
				"vnics":         `[{"description":"VNIC 1","vlan":100}]`,
			})

			var err error
			output.CaptureOutput(func() {
				err = BuyMVE(cmd, nil, true)
			})

			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tt.errMatch)
			}
		})
	}
}
