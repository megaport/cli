package product

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/config"
	"github.com/megaport/megaport-cli/internal/testutil"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProducts(t *testing.T) {
	output.SetTerminalWidthForTesting(200)
	defer output.SetTerminalWidthForTesting(0)
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name            string
		setupMock       func(*MockProductService)
		includeInactive bool
		limit           int
		outputFormat    string
		expectedError   string
		expectedOutput  string
	}{
		{
			name: "successful list - table format",
			setupMock: func(m *MockProductService) {
				m.ListProductsResult = []megaport.Product{
					&megaport.Port{
						UID:                "port-uid-1",
						Name:               "Test Port",
						Type:               "MEGAPORT",
						ProvisioningStatus: "LIVE",
						PortSpeed:          10000,
						LocationID:         1,
					},
					&megaport.MCR{
						UID:                "mcr-uid-1",
						Name:               "Test MCR",
						Type:               "MCR2",
						ProvisioningStatus: "CONFIGURED",
						PortSpeed:          5000,
						LocationID:         2,
					},
				}
			},
			outputFormat:   "table",
			expectedOutput: "port-uid-1",
		},
		{
			name: "successful list - json format",
			setupMock: func(m *MockProductService) {
				m.ListProductsResult = []megaport.Product{
					&megaport.Port{
						UID:                "port-uid-1",
						Name:               "Test Port",
						Type:               "MEGAPORT",
						ProvisioningStatus: "LIVE",
						PortSpeed:          10000,
						LocationID:         1,
					},
				}
			},
			outputFormat:   "json",
			expectedOutput: "port-uid-1",
		},
		{
			name: "filters cancelled products by default",
			setupMock: func(m *MockProductService) {
				m.ListProductsResult = []megaport.Product{
					&megaport.Port{
						UID:                "port-active",
						Name:               "Active Port",
						Type:               "MEGAPORT",
						ProvisioningStatus: "LIVE",
						PortSpeed:          10000,
						LocationID:         1,
					},
					&megaport.Port{
						UID:                "port-cancelled",
						Name:               "Cancelled Port",
						Type:               "MEGAPORT",
						ProvisioningStatus: "CANCELLED",
						PortSpeed:          10000,
						LocationID:         1,
					},
				}
			},
			outputFormat:   "table",
			expectedOutput: "port-active",
		},
		{
			name: "includes inactive when flag set",
			setupMock: func(m *MockProductService) {
				m.ListProductsResult = []megaport.Product{
					&megaport.Port{
						UID:                "port-active",
						Name:               "Active Port",
						Type:               "MEGAPORT",
						ProvisioningStatus: "LIVE",
						PortSpeed:          10000,
						LocationID:         1,
					},
					&megaport.Port{
						UID:                "port-cancelled",
						Name:               "Cancelled Port",
						Type:               "MEGAPORT",
						ProvisioningStatus: "CANCELLED",
						PortSpeed:          10000,
						LocationID:         1,
					},
				}
			},
			includeInactive: true,
			outputFormat:    "table",
			expectedOutput:  "port-cancelled",
		},
		{
			name: "API error",
			setupMock: func(m *MockProductService) {
				m.ListProductsErr = fmt.Errorf("API error")
			},
			outputFormat:  "table",
			expectedError: "failed to list products",
		},
		{
			name: "empty list",
			setupMock: func(m *MockProductService) {
				m.ListProductsResult = []megaport.Product{}
			},
			outputFormat: "table",
		},
		{
			name: "limit results",
			setupMock: func(m *MockProductService) {
				m.ListProductsResult = []megaport.Product{
					&megaport.Port{UID: "port-1", Name: "Port 1", Type: "MEGAPORT", ProvisioningStatus: "LIVE", PortSpeed: 10000, LocationID: 1},
					&megaport.Port{UID: "port-2", Name: "Port 2", Type: "MEGAPORT", ProvisioningStatus: "LIVE", PortSpeed: 10000, LocationID: 2},
					&megaport.Port{UID: "port-3", Name: "Port 3", Type: "MEGAPORT", ProvisioningStatus: "LIVE", PortSpeed: 10000, LocationID: 3},
				}
			},
			limit:          2,
			outputFormat:   "table",
			expectedOutput: "port-2",
		},
		{
			name:          "negative limit returns error",
			setupMock:     func(m *MockProductService) { m.ListProductsResult = []megaport.Product{} },
			limit:         -1,
			outputFormat:  "table",
			expectedError: "--limit must be a non-negative integer",
		},
		{
			name: "MVE product included",
			setupMock: func(m *MockProductService) {
				m.ListProductsResult = []megaport.Product{
					&megaport.MVE{
						UID:                "mve-uid-1",
						Name:               "Test MVE",
						Type:               "MVE",
						ProvisioningStatus: "LIVE",
						LocationID:         3,
					},
				}
			},
			outputFormat:   "table",
			expectedOutput: "mve-uid-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockProductService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.ProductService = mockService
				return client, nil
			})

			cmd := &cobra.Command{Use: "list"}
			cmd.Flags().Bool("include-inactive", false, "")
			cmd.Flags().Int("limit", 0, "")
			cmd.Flags().StringP("output", "o", "table", "")

			if tt.includeInactive {
				err := cmd.Flags().Set("include-inactive", "true")
				assert.NoError(t, err)
			}
			if tt.limit != 0 {
				err := cmd.Flags().Set("limit", fmt.Sprintf("%d", tt.limit))
				assert.NoError(t, err)
			}

			var err error
			capturedOutput := output.CaptureOutput(func() {
				err = ListProducts(cmd, []string{}, true, tt.outputFormat)
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

func TestGetProductType(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	tests := []struct {
		name           string
		productUID     string
		setupMock      func(*MockProductService)
		outputFormat   string
		expectedError  string
		expectedOutput string
	}{
		{
			name:       "successful get type - table format",
			productUID: "port-uid-1",
			setupMock: func(m *MockProductService) {
				m.GetProductTypeResult = "MEGAPORT"
			},
			outputFormat:   "table",
			expectedOutput: "MEGAPORT",
		},
		{
			name:       "successful get type - json format",
			productUID: "mcr-uid-1",
			setupMock: func(m *MockProductService) {
				m.GetProductTypeResult = "MCR2"
			},
			outputFormat:   "json",
			expectedOutput: "MCR2",
		},
		{
			name:       "product not found",
			productUID: "unknown-uid",
			setupMock: func(m *MockProductService) {
				m.GetProductTypeErr = fmt.Errorf("product not found")
			},
			outputFormat:  "table",
			expectedError: "failed to get product type",
		},
		{
			name:       "API error",
			productUID: "error-uid",
			setupMock: func(m *MockProductService) {
				m.GetProductTypeErr = fmt.Errorf("API error")
			},
			outputFormat:  "table",
			expectedError: "API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockProductService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				client := &megaport.Client{}
				client.ProductService = mockService
				return client, nil
			})

			cmd := &cobra.Command{Use: "get-type"}
			cmd.Flags().StringP("output", "o", "table", "")

			var err error
			capturedOutput := output.CaptureOutput(func() {
				err = GetProductType(cmd, []string{tt.productUID}, true, tt.outputFormat)
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

func TestAddCommandsTo(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test-cli"}
	AddCommandsTo(rootCmd)

	productCmd, _, err := rootCmd.Find([]string{"product"})
	assert.NoError(t, err)
	assert.NotNil(t, productCmd)
	assert.Equal(t, "product", productCmd.Use)

	listCmd, _, err := rootCmd.Find([]string{"product", "list"})
	assert.NoError(t, err)
	assert.NotNil(t, listCmd)
	assert.Equal(t, "list", listCmd.Use)

	getTypeCmd, _, err := rootCmd.Find([]string{"product", "get-type"})
	assert.NoError(t, err)
	assert.NotNil(t, getTypeCmd)
	assert.Equal(t, "get-type", getTypeCmd.Use)

	availabilityCmd, _, err := rootCmd.Find([]string{"product", "availability"})
	assert.NoError(t, err)
	assert.NotNil(t, availabilityCmd)
	assert.Equal(t, "availability", availabilityCmd.Use)
}

func TestProductModule(t *testing.T) {
	module := NewModule()
	assert.Equal(t, "product", module.Name())

	rootCmd := &cobra.Command{Use: "test-cli"}
	module.RegisterCommands(rootCmd)

	productCmd, _, err := rootCmd.Find([]string{"product"})
	assert.NoError(t, err)
	assert.NotNil(t, productCmd)
}

func TestListProductsLoginError(t *testing.T) {
	cleanup := testutil.SetupLoginError(fmt.Errorf("login failed"))
	defer cleanup()

	cmd := &cobra.Command{Use: "list"}
	cmd.Flags().Bool("include-inactive", false, "")
	cmd.Flags().Int("limit", 0, "")
	cmd.Flags().StringP("output", "o", "table", "")

	var err error
	output.CaptureOutput(func() {
		err = ListProducts(cmd, []string{}, true, "table")
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to log in")
}

func TestGetProductTypeLoginError(t *testing.T) {
	cleanup := testutil.SetupLoginError(fmt.Errorf("login failed"))
	defer cleanup()

	cmd := &cobra.Command{Use: "get-type"}
	cmd.Flags().StringP("output", "o", "table", "")

	var err error
	output.CaptureOutput(func() {
		err = GetProductType(cmd, []string{"some-uid"}, true, "table")
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to log in")
}

func TestListProductsEmptyListJsonFormat(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	defer cleanup()

	mockService := &MockProductService{
		ListProductsResult: []megaport.Product{},
	}
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		client := &megaport.Client{}
		client.ProductService = mockService
		return client, nil
	})

	cmd := &cobra.Command{Use: "list"}
	cmd.Flags().Bool("include-inactive", false, "")
	cmd.Flags().Int("limit", 0, "")
	cmd.Flags().StringP("output", "o", "json", "")

	var err error
	output.CaptureOutput(func() {
		err = ListProducts(cmd, []string{}, true, "json")
	})
	assert.NoError(t, err)
}

func TestFilterProductsEdgeCases(t *testing.T) {
	tests := []struct {
		name            string
		products        []megaport.Product
		includeInactive bool
		expectedCount   int
	}{
		{
			name:            "nil products in list",
			products:        []megaport.Product{nil, &megaport.Port{UID: "port-1", ProvisioningStatus: "LIVE"}},
			includeInactive: false,
			expectedCount:   1,
		},
		{
			name: "filters DECOMMISSIONED",
			products: []megaport.Product{
				&megaport.Port{UID: "port-1", ProvisioningStatus: "DECOMMISSIONED"},
				&megaport.Port{UID: "port-2", ProvisioningStatus: "LIVE"},
			},
			includeInactive: false,
			expectedCount:   1,
		},
		{
			name: "filters DECOMMISSIONING",
			products: []megaport.Product{
				&megaport.Port{UID: "port-1", ProvisioningStatus: "DECOMMISSIONING"},
				&megaport.Port{UID: "port-2", ProvisioningStatus: "LIVE"},
			},
			includeInactive: false,
			expectedCount:   1,
		},
		{
			name:            "nil input list",
			products:        nil,
			includeInactive: false,
			expectedCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterProducts(tt.products, tt.includeInactive)
			assert.Equal(t, tt.expectedCount, len(result))
		})
	}
}

func TestToProductOutput(t *testing.T) {
	t.Run("nil product", func(t *testing.T) {
		_, err := toProductOutput(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil value")
	})

	t.Run("MCR product", func(t *testing.T) {
		mcr := &megaport.MCR{
			UID:                "mcr-uid-1",
			Name:               "Test MCR",
			Type:               "MCR2",
			ProvisioningStatus: "LIVE",
			PortSpeed:          5000,
			LocationID:         2,
		}
		o, err := toProductOutput(mcr)
		assert.NoError(t, err)
		assert.Equal(t, "mcr-uid-1", o.UID)
		assert.Equal(t, "Test MCR", o.Name)
		assert.Equal(t, 5000, o.Speed)
		assert.Equal(t, 2, o.LocationID)
	})
}

func availabilityFixture() []*megaport.CompanyProductAvailability {
	return []*megaport.CompanyProductAvailability{
		{
			Category:    megaport.AvailableProductCategoryProduct,
			DisplayName: "Port",
			Variant:     "1G",
			Markets: []megaport.CompanyProductMarket{
				{Market: "MEGAPORT_AUSTRALIA", MarketName: "Australia", MarketOwner: "Megaport (Australia) Pty Ltd", CountryCode: "AU", Source: megaport.AvailabilitySourceMarket},
				{Market: "MEGAPORT_US", MarketName: "United States", MarketOwner: "Megaport (USA) Inc", CountryCode: "US", Source: megaport.AvailabilitySourceOverride},
			},
		},
		nil,
		{
			Category:    megaport.AvailableProductCategoryAddon,
			DisplayName: "IPsec",
			Variant:     "10G",
			Markets: []megaport.CompanyProductMarket{
				{Market: "MEGAPORT_AUSTRALIA", MarketName: "Australia", MarketOwner: "Megaport (Australia) Pty Ltd", CountryCode: "AU", Source: megaport.AvailabilitySourceMarket},
			},
		},
	}
}

func setupAvailabilityLogin(companyErr error, list []*megaport.CompanyProductAvailability, listErr error) (*MockProductAvailabilityService, func()) {
	availabilityMock := &MockProductAvailabilityService{
		ListCompanyProductAvailabilityResult: list,
		ListCompanyProductAvailabilityErr:    listErr,
	}
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.CompanyService = &MockCompanyService{
			GetCurrentCompanyResult: &megaport.CurrentCompany{CompanyUID: "company-uid-1"},
			GetCurrentCompanyErr:    companyErr,
		}
		c.ProductAvailabilityService = availabilityMock
	})
	return availabilityMock, cleanup
}

func TestListProductAvailability(t *testing.T) {
	output.SetTerminalWidthForTesting(200)
	defer output.SetTerminalWidthForTesting(0)

	tests := []struct {
		name         string
		list         []*megaport.CompanyProductAvailability
		outputFormat string
		check        func(t *testing.T, out string)
	}{
		{
			name:         "one row per product and market, skipping a nil product",
			list:         availabilityFixture(),
			outputFormat: "json",
			check: func(t *testing.T, out string) {
				var rows []map[string]any
				assert.NoError(t, json.Unmarshal([]byte(out), &rows))
				assert.Equal(t, []map[string]any{
					{"category": "PRODUCT", "display_name": "Port", "variant": "1G", "market": "MEGAPORT_AUSTRALIA", "market_name": "Australia", "market_owner": "Megaport (Australia) Pty Ltd", "country_code": "AU", "source": "MARKET"},
					{"category": "PRODUCT", "display_name": "Port", "variant": "1G", "market": "MEGAPORT_US", "market_name": "United States", "market_owner": "Megaport (USA) Inc", "country_code": "US", "source": "OVERRIDE"},
					{"category": "ADDON", "display_name": "IPsec", "variant": "10G", "market": "MEGAPORT_AUSTRALIA", "market_name": "Australia", "market_owner": "Megaport (Australia) Pty Ltd", "country_code": "AU", "source": "MARKET"},
				}, rows)
			},
		},
		{
			name:         "table headers",
			list:         availabilityFixture(),
			outputFormat: "table",
			check: func(t *testing.T, out string) {
				var header string
				for _, line := range strings.Split(out, "\n") {
					if strings.Contains(line, "CATEGORY") {
						header = strings.Join(strings.Fields(strings.ReplaceAll(line, "│", " ")), " ")
					}
				}
				assert.Equal(t, "CATEGORY PRODUCT VARIANT MARKET MARKET NAME MARKET OWNER COUNTRY SOURCE", header)
				assert.Contains(t, out, "United States")
			},
		},
		{
			name:         "no products prints an empty JSON list",
			list:         []*megaport.CompanyProductAvailability{},
			outputFormat: "json",
			check: func(t *testing.T, out string) {
				assert.Equal(t, "[]", strings.TrimSpace(out))
			},
		},
		{
			name:         "no products prints a message in table format",
			list:         []*megaport.CompanyProductAvailability{},
			outputFormat: "table",
			check: func(t *testing.T, out string) {
				assert.Contains(t, out, "No products available.")
			},
		},
		{
			name:         "products without markets print the empty message in table format",
			list:         []*megaport.CompanyProductAvailability{{DisplayName: "Port"}, nil},
			outputFormat: "table",
			check: func(t *testing.T, out string) {
				assert.Contains(t, out, "No products available.")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			availabilityMock, cleanup := setupAvailabilityLogin(nil, tt.list, nil)
			defer cleanup()

			cmd := &cobra.Command{Use: "availability"}
			cmd.Flags().StringP("output", "o", "table", "")

			// JSON comes from stdout alone; the table-mode empty message goes to stderr.
			capture := output.CaptureOutput
			if tt.outputFormat == "json" {
				capture = output.CaptureStdout
			}
			var err error
			out := capture(func() {
				err = ListProductAvailability(cmd, nil, true, tt.outputFormat)
			})

			assert.NoError(t, err)
			assert.Equal(t, 1, availabilityMock.ListCompanyProductAvailabilityCalls)
			assert.Equal(t, "company-uid-1", availabilityMock.ListCompanyProductAvailabilityCompany)
			tt.check(t, out)
		})
	}
}

func TestListProductAvailabilityErrors(t *testing.T) {
	tests := []struct {
		name          string
		companyErr    error
		listErr       error
		wantListCalls int
		wantErr       string
	}{
		{
			name:          "company error stops before the availability call",
			companyErr:    errors.New("boom"),
			wantListCalls: 0,
			wantErr:       "failed to get current company",
		},
		{
			name:          "availability error returns no rows",
			listErr:       errors.New("boom"),
			wantListCalls: 1,
			wantErr:       "failed to list product availability",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			availabilityMock, cleanup := setupAvailabilityLogin(tt.companyErr, availabilityFixture(), tt.listErr)
			defer cleanup()

			cmd := &cobra.Command{Use: "availability"}
			cmd.Flags().StringP("output", "o", "table", "")

			var err error
			out := output.CaptureStdout(func() {
				err = ListProductAvailability(cmd, nil, true, "table")
			})

			assert.ErrorContains(t, err, tt.wantErr)
			assert.Equal(t, tt.wantListCalls, availabilityMock.ListCompanyProductAvailabilityCalls)
			assert.Empty(t, strings.TrimSpace(out), "an error prints no rows")
		})
	}
}

func TestListProductAvailabilityExitCode(t *testing.T) {
	forbidden := &megaport.ErrorResponse{
		Response: &http.Response{StatusCode: 403, Header: http.Header{}, Request: &http.Request{}},
		Message:  "forbidden",
	}

	tests := []struct {
		name     string
		listErr  error
		wantCode int
	}{
		{name: "a 403 exits with the auth code", listErr: forbidden, wantCode: exitcodes.Authentication},
		{name: "any other error exits with the API code", listErr: errors.New("boom"), wantCode: exitcodes.API},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup := setupAvailabilityLogin(nil, nil, tt.listErr)
			defer cleanup()

			root := &cobra.Command{Use: "test-cli", SilenceUsage: true, SilenceErrors: true}
			root.PersistentFlags().StringP("output", "o", "table", "")
			AddCommandsTo(root)
			root.SetArgs([]string{"product", "availability"})

			var err error
			output.CaptureOutput(func() { err = root.Execute() })

			var cliErr *exitcodes.CLIError
			require.ErrorAs(t, err, &cliErr)
			assert.Equal(t, tt.wantCode, cliErr.Code)
		})
	}
}

func TestListProductAvailabilityLoginError(t *testing.T) {
	cleanup := testutil.SetupLoginError(fmt.Errorf("login failed"))
	defer cleanup()

	cmd := &cobra.Command{Use: "availability"}
	err := ListProductAvailability(cmd, nil, true, "table")
	assert.ErrorContains(t, err, "failed to log in")
}

func TestListProductAvailabilityRegisteredCommand(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.CompanyService = &MockCompanyService{GetCurrentCompanyResult: &megaport.CurrentCompany{CompanyUID: "company-uid-1"}}
		c.ProductAvailabilityService = &MockProductAvailabilityService{ListCompanyProductAvailabilityResult: availabilityFixture()}
	})
	defer cleanup()

	t.Run("runs with no arguments", func(t *testing.T) {
		root := &cobra.Command{Use: "test-cli"}
		root.PersistentFlags().StringP("output", "o", "table", "")
		AddCommandsTo(root)
		root.SetArgs([]string{"product", "availability", "-o", "json"})

		var err error
		out := output.CaptureStdout(func() { err = root.Execute() })
		assert.NoError(t, err)
		assert.Contains(t, out, `"display_name": "Port"`)
	})

	t.Run("rejects an argument", func(t *testing.T) {
		root := &cobra.Command{Use: "test-cli", SilenceUsage: true, SilenceErrors: true}
		root.PersistentFlags().StringP("output", "o", "table", "")
		AddCommandsTo(root)
		root.SetArgs([]string{"product", "availability", "extra"})
		assert.Error(t, root.Execute())
	})
}

func TestProductPricing(t *testing.T) {
	output.SetTerminalWidthForTesting(200)
	defer output.SetTerminalWidthForTesting(0)
	jsonFile := filepath.Join(t.TempDir(), "port.json")
	err := os.WriteFile(jsonFile, []byte(`{"locationId":1,"speed":1000,"term":12,"currency":"USD","productUid":"port-uid"}`+"\n"), 0o600)
	assert.NoError(t, err)

	portReq := &megaport.MegaportPriceBookRequest{LocationID: 1, Speed: 1000, Term: 12, Currency: "USD", ProductUID: "port-uid"}
	vxcReq := &megaport.VXCPriceBookRequest{ALocationID: 1, BLocationID: 2, Speed: 100, Term: 12, AEndProductType: "MCR2", ConnectType: "AWS", Currency: "USD", ProductUID: "vxc-uid"}
	price := &megaport.PriceBookDTO{ProductType: "MEGAPORT", Currency: "USD", MonthlyRate: 450.5, MonthlyRackRate: 500}

	tests := []struct {
		name          string
		args          []string
		apiErr        error
		wantReq       megaport.PriceBookRequest
		wantCompany   string
		wantUsageErr  bool
		expectedError string
	}{
		{
			name:    "port from flags",
			args:    []string{"port", "--location-id", "1", "--speed", "1000", "--term", "12", "--currency", "USD", "--product-uid", "port-uid"},
			wantReq: portReq,
		},
		{
			name:    "port from json",
			args:    []string{"port", "--json", `{"locationId":1,"speed":1000,"term":12,"currency":"USD","productUid":"port-uid"}`},
			wantReq: portReq,
		},
		{
			name:    "port from json-file",
			args:    []string{"port", "--json-file", jsonFile},
			wantReq: portReq,
		},
		{
			name: "vxc from flags",
			args: []string{"vxc", "--a-location-id", "1", "--b-location-id", "2", "--speed", "100", "--term", "12",
				"--a-end-product-type", "MCR2", "--connect-type", "AWS", "--currency", "USD", "--product-uid", "vxc-uid"},
			wantReq: vxcReq,
		},
		{
			name:    "vxc from json",
			args:    []string{"vxc", "--json", `{"aLocationId":1,"bLocationId":2,"speed":100,"term":12,"aEndProductType":"MCR2","connectType":"AWS","currency":"USD","productUid":"vxc-uid"}`},
			wantReq: vxcReq,
		},
		{
			name:        "port with company-uid",
			args:        []string{"port", "--location-id", "1", "--speed", "1000", "--term", "12", "--currency", "USD", "--product-uid", "port-uid", "--company-uid", "company-uid-1"},
			wantReq:     portReq,
			wantCompany: "company-uid-1",
		},
		{
			name:        "vxc from json with company-uid",
			args:        []string{"vxc", "--company-uid", "company-uid-1", "--json", `{"aLocationId":1,"bLocationId":2,"speed":100,"term":12,"aEndProductType":"MCR2","connectType":"AWS","currency":"USD","productUid":"vxc-uid"}`},
			wantReq:     vxcReq,
			wantCompany: "company-uid-1",
		},
		{
			name:         "port missing location-id",
			args:         []string{"port", "--speed", "1000"},
			wantUsageErr: true,
		},
		{
			name:         "port missing speed",
			args:         []string{"port", "--location-id", "1"},
			wantUsageErr: true,
		},
		{
			name:         "vxc missing a-location-id",
			args:         []string{"vxc", "--b-location-id", "2", "--speed", "100"},
			wantUsageErr: true,
		},
		{
			name:         "vxc missing b-location-id",
			args:         []string{"vxc", "--a-location-id", "1", "--speed", "100"},
			wantUsageErr: true,
		},
		{
			name:         "vxc missing speed",
			args:         []string{"vxc", "--a-location-id", "1", "--b-location-id", "2"},
			wantUsageErr: true,
		},
		{
			name:         "invalid json",
			args:         []string{"port", "--json", `{"locationId":`},
			wantUsageErr: true,
		},
		{
			name:         "json with unknown key",
			args:         []string{"port", "--json", `{"locationId":1,"speed":1000,"companyUid":"company-uid-1"}`},
			wantUsageErr: true,
		},
		{
			name:         "vxc json with unknown key",
			args:         []string{"vxc", "--json", `{"aLocationId":1,"bLocationId":2,"speed":100,"locationId":1}`},
			wantUsageErr: true,
		},
		{
			name:         "json with trailing data",
			args:         []string{"port", "--json", `{"locationId":1,"speed":1000}{}`},
			wantUsageErr: true,
		},
		{
			name:         "json with trailing brace",
			args:         []string{"port", "--json", `{"locationId":1,"speed":1000}}`},
			wantUsageErr: true,
		},
		{
			name:          "empty json",
			args:          []string{"port", "--json", " "},
			wantUsageErr:  true,
			expectedError: "JSON input is empty",
		},
		{
			name:         "missing json-file",
			args:         []string{"port", "--json-file", filepath.Join(t.TempDir(), "missing.json")},
			wantUsageErr: true,
		},
		{
			name:         "json missing location",
			args:         []string{"port", "--json", `{"speed":1000}`},
			apiErr:       megaport.ErrPricingLocationRequired,
			wantUsageErr: true,
		},
		{
			name:         "json missing speed",
			args:         []string{"port", "--json", `{"locationId":1}`},
			apiErr:       megaport.ErrPricingSpeedRequired,
			wantUsageErr: true,
		},
		{
			name:         "vxc json missing location",
			args:         []string{"vxc", "--json", `{"aLocationId":1,"speed":100}`},
			apiErr:       megaport.ErrPricingVXCLocationRequired,
			wantUsageErr: true,
		},
		{
			name:         "vxc json missing speed",
			args:         []string{"vxc", "--json", `{"aLocationId":1,"bLocationId":2}`},
			apiErr:       megaport.ErrPricingVXCSpeedRequired,
			wantUsageErr: true,
		},
		{
			name:          "API error",
			args:          []string{"port", "--location-id", "1", "--speed", "1000"},
			apiErr:        fmt.Errorf("API error"),
			expectedError: "API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockProductService{
				GetProductPricingResult:           price,
				GetProductPricingErr:              tt.apiErr,
				GetProductPricingForCompanyResult: price,
			}
			cleanup := testutil.SetupLogin(func(c *megaport.Client) {
				c.ProductService = mockService
			})
			defer cleanup()

			rootCmd := &cobra.Command{Use: "megaport-cli"}
			rootCmd.PersistentFlags().StringP("output", "o", "table", "")
			AddCommandsTo(rootCmd)
			rootCmd.SetArgs(append([]string{"product", "pricing"}, tt.args...))

			capturedOutput, err := output.CaptureOutputErr(rootCmd.Execute)

			if tt.wantUsageErr {
				var cliErr *exitcodes.CLIError
				require.True(t, errors.As(err, &cliErr))
				assert.Equal(t, exitcodes.Usage, cliErr.Code)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
				if tt.apiErr == nil {
					assert.Nil(t, mockService.CapturedGetProductPricingRequest)
					assert.Nil(t, mockService.CapturedGetProductPricingForCompanyRequest)
				}
				return
			}
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				var cliErr *exitcodes.CLIError
				require.True(t, errors.As(err, &cliErr))
				assert.Equal(t, exitcodes.API, cliErr.Code)
				return
			}

			assert.NoError(t, err)
			if tt.wantCompany != "" {
				assert.Nil(t, mockService.CapturedGetProductPricingRequest)
				if assert.NotNil(t, mockService.CapturedGetProductPricingForCompanyRequest) {
					assert.Equal(t, tt.wantReq, mockService.CapturedGetProductPricingForCompanyRequest.Req)
					assert.Equal(t, tt.wantCompany, mockService.CapturedGetProductPricingForCompanyRequest.CompanyUID)
				}
			} else {
				assert.Nil(t, mockService.CapturedGetProductPricingForCompanyRequest)
				assert.Equal(t, tt.wantReq, mockService.CapturedGetProductPricingRequest)
			}
			assert.Contains(t, capturedOutput, "PRODUCT TYPE")
			assert.Contains(t, capturedOutput, "CURRENCY")
			assert.Contains(t, capturedOutput, "MONTHLY RATE")
			assert.Contains(t, capturedOutput, "MONTHLY RACK RATE")
			assert.Contains(t, capturedOutput, "USD")
			assert.Contains(t, capturedOutput, "450.5")
			assert.Contains(t, capturedOutput, "500")
		})
	}
}

func TestProductPricingJSONOutput(t *testing.T) {
	mockService := &MockProductService{
		GetProductPricingResult: &megaport.PriceBookDTO{ProductType: "VXC", Currency: "AUD", MonthlyRate: 120, MonthlyRackRate: 150},
	}
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.ProductService = mockService
	})
	defer cleanup()

	rootCmd := &cobra.Command{Use: "megaport-cli"}
	rootCmd.PersistentFlags().StringP("output", "o", "table", "")
	AddCommandsTo(rootCmd)
	rootCmd.SetArgs([]string{"product", "pricing", "vxc", "--a-location-id", "1", "--b-location-id", "2", "--speed", "100", "-o", "json"})

	var err error
	stdout := output.CaptureStdout(func() {
		err = rootCmd.Execute()
	})
	assert.NoError(t, err)

	var got []map[string]any
	assert.NoError(t, json.Unmarshal([]byte(stdout), &got))
	assert.Equal(t, []map[string]any{{"product_type": "VXC", "currency": "AUD", "monthly_rate": 120.0, "monthly_rack_rate": 150.0}}, got)
}
