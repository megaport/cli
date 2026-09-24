package ix

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/utils"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

var noColor = true

var testIXs = []*megaport.IX{
	{
		ProductUID:         "ix-1",
		ProductName:        "MyIXOne",
		NetworkServiceType: "Los Angeles IX",
		ASN:                65000,
		RateLimit:          1000,
		VLAN:               100,
		MACAddress:         "00:11:22:33:44:55",
		ProvisioningStatus: "LIVE",
		LocationID:         123,
	},
	{
		ProductUID:         "ix-2",
		ProductName:        "AnotherIX",
		NetworkServiceType: "Sydney IX",
		ASN:                65001,
		RateLimit:          2000,
		VLAN:               200,
		MACAddress:         "AA:BB:CC:DD:EE:FF",
		ProvisioningStatus: "CONFIGURED",
		LocationID:         456,
	},
}

func TestPrintIXs_Table(t *testing.T) {
	out := output.CaptureOutput(func() {
		err := printIXs(testIXs, "table", noColor)
		assert.NoError(t, err)
	})

	assert.Contains(t, out, "UID")
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "NETWORK SERVICE TYPE")
	assert.Contains(t, out, "ASN")
	assert.Contains(t, out, "RATE LIMIT")
	assert.Contains(t, out, "VLAN")
	assert.Contains(t, out, "MAC ADDRESS")
	assert.Contains(t, out, "STATUS")

	assert.Contains(t, out, "ix-1")
	assert.Contains(t, out, "MyIXOne")
	assert.Contains(t, out, "Los Angeles IX")
	assert.Contains(t, out, "65000")
	assert.Contains(t, out, "1000")
	assert.Contains(t, out, "100")
	assert.Contains(t, out, "00:11:22:33:44:55")
	assert.Contains(t, out, "LIVE")

	assert.Contains(t, out, "ix-2")
	assert.Contains(t, out, "AnotherIX")
	assert.Contains(t, out, "Sydney IX")

	assert.Contains(t, out, "┌")
	assert.Contains(t, out, "┐")
	assert.Contains(t, out, "└")
	assert.Contains(t, out, "┘")
	assert.Contains(t, out, "├")
	assert.Contains(t, out, "┤")
	assert.Contains(t, out, "│")
	assert.Contains(t, out, "─")
}

func TestPrintIXs_JSON(t *testing.T) {
	out := output.CaptureOutput(func() {
		err := printIXs(testIXs, "json", noColor)
		assert.NoError(t, err)
	})

	expected := `[
  {
	"uid": "ix-1",
	"name": "MyIXOne",
	"network_service_type": "Los Angeles IX",
	"asn": 65000,
	"rate_limit": 1000,
	"vlan": 100,
	"mac_address": "00:11:22:33:44:55",
	"status": "LIVE"
  },
  {
	"uid": "ix-2",
	"name": "AnotherIX",
	"network_service_type": "Sydney IX",
	"asn": 65001,
	"rate_limit": 2000,
	"vlan": 200,
	"mac_address": "AA:BB:CC:DD:EE:FF",
	"status": "CONFIGURED"
  }
]`
	assert.JSONEq(t, expected, out)
}

func TestPrintIXs_CSV(t *testing.T) {
	out := output.CaptureOutput(func() {
		err := printIXs(testIXs, "csv", noColor)
		assert.NoError(t, err)
	})

	expected := `uid,name,network_service_type,asn,rate_limit,vlan,mac_address,status
ix-1,MyIXOne,Los Angeles IX,65000,1000,100,00:11:22:33:44:55,LIVE
ix-2,AnotherIX,Sydney IX,65001,2000,200,AA:BB:CC:DD:EE:FF,CONFIGURED
`
	assert.Equal(t, expected, out)
}

func TestPrintIXs_Invalid(t *testing.T) {
	var err error
	out := output.CaptureOutput(func() {
		err = printIXs(testIXs, "invalid", noColor)
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
	assert.Empty(t, out)
}

func TestPrintIXs_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		ixs    []*megaport.IX
		format string
	}{
		{
			name:   "empty slice table format",
			ixs:    []*megaport.IX{},
			format: "table",
		},
		{
			name:   "empty slice csv format",
			ixs:    []*megaport.IX{},
			format: "csv",
		},
		{
			name:   "empty slice json format",
			ixs:    []*megaport.IX{},
			format: "json",
		},
		{
			name:   "nil slice table format",
			ixs:    nil,
			format: "table",
		},
		{
			name:   "nil slice csv format",
			ixs:    nil,
			format: "csv",
		},
		{
			name:   "nil slice json format",
			ixs:    nil,
			format: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := output.CaptureOutput(func() {
				err := printIXs(tt.ixs, tt.format, noColor)
				assert.NoError(t, err)
			})

			switch tt.format {
			case "table":
				assert.Contains(t, out, "UID")
				assert.Contains(t, out, "NAME")
				assert.Contains(t, out, "NETWORK SERVICE TYPE")
				assert.Contains(t, out, "ASN")
				assert.Contains(t, out, "RATE LIMIT")
				assert.Contains(t, out, "VLAN")
				assert.Contains(t, out, "MAC ADDRESS")
				assert.Contains(t, out, "STATUS")
				assert.Contains(t, out, "┌")
				assert.Contains(t, out, "┐")
				assert.Contains(t, out, "└")
				assert.Contains(t, out, "┘")
				assert.Contains(t, out, "│")
				assert.Contains(t, out, "─")
			case "csv":
				expected := "uid,name,network_service_type,asn,rate_limit,vlan,mac_address,status\n"
				assert.Equal(t, expected, out)
			case "json":
				assert.Equal(t, "[]\n", out)
			}
		})
	}
}

func TestToIXOutput_EdgeCases(t *testing.T) {
	t.Run("nil IX", func(t *testing.T) {
		_, err := toIXOutput(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid IX: nil value")
	})

	t.Run("zero value IX", func(t *testing.T) {
		ix := &megaport.IX{}
		out, err := toIXOutput(ix)
		assert.NoError(t, err)
		assert.Equal(t, "", out.UID)
		assert.Equal(t, "", out.Name)
		assert.Equal(t, "", out.NetworkServiceType)
		assert.Equal(t, 0, out.ASN)
		assert.Equal(t, 0, out.RateLimit)
		assert.Equal(t, 0, out.VLAN)
		assert.Equal(t, "", out.MACAddress)
		assert.Equal(t, "", out.Status)
	})
}

func TestFilterIXs(t *testing.T) {
	activeIXs := []*megaport.IX{
		{
			ProductUID:         "ix-1",
			ProductName:        "TestIX-1",
			NetworkServiceType: "Los Angeles IX",
			ASN:                65000,
			VLAN:               100,
			LocationID:         123,
			RateLimit:          1000,
			ProvisioningStatus: "LIVE",
		},
		{
			ProductUID:         "ix-2",
			ProductName:        "TestIX-2",
			NetworkServiceType: "Sydney IX",
			ASN:                65001,
			VLAN:               200,
			LocationID:         456,
			RateLimit:          2000,
			ProvisioningStatus: "CONFIGURED",
		},
		{
			ProductUID:         "ix-3",
			ProductName:        "Production-IX",
			NetworkServiceType: "Los Angeles IX",
			ASN:                65002,
			VLAN:               300,
			LocationID:         123,
			RateLimit:          5000,
			ProvisioningStatus: "LIVE",
		},
		{
			ProductUID:         "ix-4",
			ProductName:        "Staging-IX",
			NetworkServiceType: "London IX",
			ASN:                65003,
			VLAN:               400,
			LocationID:         789,
			RateLimit:          1000,
			ProvisioningStatus: "LIVE",
		},
	}

	tests := []struct {
		name               string
		ixs                []*megaport.IX
		filterName         string
		networkServiceType string
		asn                int
		vlan               int
		locationID         int
		rateLimit          int
		expected           int
		expectedUIDs       []string
	}{
		{
			name:         "no filters",
			ixs:          activeIXs,
			expected:     4,
			expectedUIDs: []string{"ix-1", "ix-2", "ix-3", "ix-4"},
		},
		{
			name:         "filter by name (partial match)",
			ixs:          activeIXs,
			filterName:   "Test",
			expected:     2,
			expectedUIDs: []string{"ix-1", "ix-2"},
		},
		{
			name:         "filter by name (case insensitive)",
			ixs:          activeIXs,
			filterName:   "production",
			expected:     1,
			expectedUIDs: []string{"ix-3"},
		},
		{
			name:         "filter by ASN",
			ixs:          activeIXs,
			asn:          65000,
			expected:     1,
			expectedUIDs: []string{"ix-1"},
		},
		{
			name:         "filter by VLAN",
			ixs:          activeIXs,
			vlan:         200,
			expected:     1,
			expectedUIDs: []string{"ix-2"},
		},
		{
			name:               "filter by network service type",
			ixs:                activeIXs,
			networkServiceType: "Los Angeles",
			expected:           2,
			expectedUIDs:       []string{"ix-1", "ix-3"},
		},
		{
			name:         "filter by location ID",
			ixs:          activeIXs,
			locationID:   123,
			expected:     2,
			expectedUIDs: []string{"ix-1", "ix-3"},
		},
		{
			name:         "filter by rate limit",
			ixs:          activeIXs,
			rateLimit:    1000,
			expected:     2,
			expectedUIDs: []string{"ix-1", "ix-4"},
		},
		{
			name:               "all filters combined",
			ixs:                activeIXs,
			filterName:         "TestIX",
			networkServiceType: "Los Angeles",
			asn:                65000,
			vlan:               100,
			locationID:         123,
			rateLimit:          1000,
			expected:           1,
			expectedUIDs:       []string{"ix-1"},
		},
		{
			name:         "exact name match",
			ixs:          activeIXs,
			filterName:   "TestIX-1",
			expected:     1,
			expectedUIDs: []string{"ix-1"},
		},
		{
			name:         "non-matching filters",
			ixs:          activeIXs,
			filterName:   "nonexistent",
			expected:     0,
			expectedUIDs: []string{},
		},
		{
			name:         "nil slice",
			ixs:          nil,
			expected:     0,
			expectedUIDs: []string{},
		},
		{
			name:         "empty slice",
			ixs:          []*megaport.IX{},
			expected:     0,
			expectedUIDs: []string{},
		},
		{
			name:         "nil elements in slice",
			ixs:          []*megaport.IX{nil, activeIXs[0], nil, activeIXs[1]},
			expected:     2,
			expectedUIDs: []string{"ix-1", "ix-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterIXs(tt.ixs, tt.filterName, tt.networkServiceType, tt.asn, tt.vlan, tt.locationID, tt.rateLimit)

			assert.Equal(t, tt.expected, len(filtered), "Filtered IX count should match expected")

			if len(tt.expectedUIDs) > 0 {
				actualUIDs := make([]string, len(filtered))
				for i, ix := range filtered {
					actualUIDs[i] = ix.ProductUID
				}
				assert.ElementsMatch(t, tt.expectedUIDs, actualUIDs, "Filtered IX UIDs should match expected")
			}
		})
	}
}

func TestToIXOutput_FullIX(t *testing.T) {
	ix := &megaport.IX{
		ProductUID:         "ix-full-123",
		ProductName:        "Full Test IX",
		NetworkServiceType: "Los Angeles IX",
		ASN:                65000,
		RateLimit:          1000,
		VLAN:               100,
		MACAddress:         "00:11:22:33:44:55",
		ProvisioningStatus: "LIVE",
	}

	out, err := toIXOutput(ix)
	assert.NoError(t, err)
	assert.Equal(t, "ix-full-123", out.UID)
	assert.Equal(t, "Full Test IX", out.Name)
	assert.Equal(t, "Los Angeles IX", out.NetworkServiceType)
	assert.Equal(t, 65000, out.ASN)
	assert.Equal(t, 1000, out.RateLimit)
	assert.Equal(t, 100, out.VLAN)
	assert.Equal(t, "00:11:22:33:44:55", out.MACAddress)
	assert.Equal(t, "LIVE", out.Status)
}

func TestDisplayIXChanges(t *testing.T) {
	tests := []struct {
		name        string
		original    *megaport.IX
		updated     *megaport.IX
		expectedOut []string
	}{
		{
			name: "name changed",
			original: &megaport.IX{
				ProductName: "Old IX",
			},
			updated: &megaport.IX{
				ProductName: "New IX",
			},
			expectedOut: []string{"Name:", "Old IX", "New IX"},
		},
		{
			name: "rate limit changed",
			original: &megaport.IX{
				ProductName: "Test IX",
				RateLimit:   1000,
			},
			updated: &megaport.IX{
				ProductName: "Test IX",
				RateLimit:   2000,
			},
			expectedOut: []string{"Rate Limit:", "1000 Mbps", "2000 Mbps"},
		},
		{
			name: "VLAN changed",
			original: &megaport.IX{
				ProductName: "Test IX",
				VLAN:        100,
			},
			updated: &megaport.IX{
				ProductName: "Test IX",
				VLAN:        200,
			},
			expectedOut: []string{"VLAN:", "100", "200"},
		},
		{
			name: "MAC address changed",
			original: &megaport.IX{
				ProductName: "Test IX",
				MACAddress:  "00:11:22:33:44:55",
			},
			updated: &megaport.IX{
				ProductName: "Test IX",
				MACAddress:  "AA:BB:CC:DD:EE:FF",
			},
			expectedOut: []string{"MAC Address:", "00:11:22:33:44:55", "AA:BB:CC:DD:EE:FF"},
		},
		{
			name: "ASN changed",
			original: &megaport.IX{
				ProductName: "Test IX",
				ASN:         65000,
			},
			updated: &megaport.IX{
				ProductName: "Test IX",
				ASN:         65001,
			},
			expectedOut: []string{"ASN:", "65000", "65001"},
		},
		{
			name: "multiple fields changed",
			original: &megaport.IX{
				ProductName: "Old IX",
				RateLimit:   1000,
				VLAN:        100,
			},
			updated: &megaport.IX{
				ProductName: "New IX",
				RateLimit:   2000,
				VLAN:        200,
			},
			expectedOut: []string{"Name:", "Rate Limit:", "VLAN:"},
		},
		{
			name: "no changes",
			original: &megaport.IX{
				ProductName: "Same IX",
				RateLimit:   1000,
			},
			updated: &megaport.IX{
				ProductName: "Same IX",
				RateLimit:   1000,
			},
			expectedOut: []string{"No changes detected"},
		},
		{
			name:        "nil original",
			original:    nil,
			updated:     &megaport.IX{ProductName: "Test"},
			expectedOut: []string{},
		},
		{
			name:        "nil updated",
			original:    &megaport.IX{ProductName: "Test"},
			updated:     nil,
			expectedOut: []string{},
		},
		{
			name:        "both nil",
			original:    nil,
			updated:     nil,
			expectedOut: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capturedOutput := output.CaptureOutput(func() {
				displayIXChanges(tt.original, tt.updated, true)
			})

			for _, expected := range tt.expectedOut {
				assert.Contains(t, capturedOutput, expected)
			}
		})
	}
}

func TestBuildIXRequestFromFlags(t *testing.T) {
	tests := []struct {
		name     string
		flags    map[string]string
		validate func(t *testing.T, req *megaport.BuyIXRequest)
	}{
		{
			name: "all flags provided",
			flags: map[string]string{
				"product-uid":          "port-123",
				"name":                 "Test IX",
				"network-service-type": "Los Angeles IX",
				"asn":                  "65000",
				"mac-address":          "00:11:22:33:44:55",
				"rate-limit":           "1000",
				"vlan":                 "100",
				"promo-code":           "PROMO",
			},
			validate: func(t *testing.T, req *megaport.BuyIXRequest) {
				assert.Equal(t, "port-123", req.ProductUID)
				assert.Equal(t, "Test IX", req.Name)
				assert.Equal(t, "Los Angeles IX", req.NetworkServiceType)
				assert.Equal(t, 65000, req.ASN)
				assert.Equal(t, "00:11:22:33:44:55", req.MACAddress)
				assert.Equal(t, 1000, req.RateLimit)
				assert.Equal(t, 100, req.VLAN)
				assert.Equal(t, "PROMO", req.PromoCode)
				assert.False(t, req.Shutdown)
			},
		},
		{
			name: "shutdown flag set",
			flags: map[string]string{
				"product-uid":          "port-123",
				"name":                 "Test IX",
				"network-service-type": "Los Angeles IX",
				"asn":                  "65000",
				"mac-address":          "00:11:22:33:44:55",
				"rate-limit":           "1000",
				"vlan":                 "100",
				"shutdown":             "true",
			},
			validate: func(t *testing.T, req *megaport.BuyIXRequest) {
				assert.True(t, req.Shutdown)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().String("product-uid", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().String("network-service-type", "", "")
			cmd.Flags().Int("asn", 0, "")
			cmd.Flags().String("mac-address", "", "")
			cmd.Flags().Int("rate-limit", 0, "")
			cmd.Flags().Int("vlan", 0, "")
			cmd.Flags().Bool("shutdown", false, "")
			cmd.Flags().String("promo-code", "", "")

			for k, v := range tt.flags {
				_ = cmd.Flags().Set(k, v)
			}

			req, err := buildIXRequestFromFlags(cmd)
			assert.NoError(t, err)
			assert.NotNil(t, req)
			tt.validate(t, req)
		})
	}
}

func TestBuildIXRequestFromFlags_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		flags       map[string]string
		errContains string
	}{
		{
			name: "invalid ASN zero",
			flags: map[string]string{
				"asn":         "0",
				"mac-address": "00:11:22:33:44:55",
				"rate-limit":  "1000",
			},
			errContains: "Invalid ASN",
		},
		{
			name: "invalid ASN negative",
			flags: map[string]string{
				"asn":         "-1",
				"mac-address": "00:11:22:33:44:55",
				"rate-limit":  "1000",
			},
			errContains: "Invalid ASN",
		},
		{
			name: "invalid MAC address",
			flags: map[string]string{
				"asn":         "65000",
				"mac-address": "not-a-mac",
				"rate-limit":  "1000",
			},
			errContains: "Invalid MAC address",
		},
		{
			name: "empty MAC address",
			flags: map[string]string{
				"asn":         "65000",
				"mac-address": "",
				"rate-limit":  "1000",
			},
			errContains: "Invalid MAC address",
		},
		{
			name: "invalid rate limit zero",
			flags: map[string]string{
				"asn":         "65000",
				"mac-address": "00:11:22:33:44:55",
				"rate-limit":  "0",
			},
			errContains: "Invalid rate limit",
		},
		{
			name: "invalid rate limit negative",
			flags: map[string]string{
				"asn":         "65000",
				"mac-address": "00:11:22:33:44:55",
				"rate-limit":  "-1",
			},
			errContains: "Invalid rate limit",
		},
		{
			name: "invalid VLAN reserved",
			flags: map[string]string{
				"asn":         "65000",
				"mac-address": "00:11:22:33:44:55",
				"rate-limit":  "1000",
				"vlan":        "1",
			},
			errContains: "Invalid VLAN ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().String("product-uid", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().String("network-service-type", "", "")
			cmd.Flags().Int("asn", 0, "")
			cmd.Flags().String("mac-address", "", "")
			cmd.Flags().Int("rate-limit", 0, "")
			cmd.Flags().Int("vlan", 0, "")
			cmd.Flags().Bool("shutdown", false, "")
			cmd.Flags().String("promo-code", "", "")

			for k, v := range tt.flags {
				_ = cmd.Flags().Set(k, v)
			}

			req, err := buildIXRequestFromFlags(cmd)
			assert.Error(t, err)
			assert.Nil(t, req)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestBuildIXRequestFromJSON(t *testing.T) {
	tests := []struct {
		name          string
		jsonStr       string
		jsonFile      string
		setupFile     func(t *testing.T) string // returns temp file path
		expectedError string
		validate      func(t *testing.T, req *megaport.BuyIXRequest)
	}{
		{
			name:    "valid JSON string with all fields",
			jsonStr: `{"productUid":"port-123","productName":"JSON IX","networkServiceType":"Sydney IX","asn":65100,"macAddress":"AA:BB:CC:DD:EE:FF","rateLimit":2000,"vlan":200,"shutdown":true,"promoCode":"PROMO"}`,
			validate: func(t *testing.T, req *megaport.BuyIXRequest) {
				assert.Equal(t, "port-123", req.ProductUID)
				assert.Equal(t, "JSON IX", req.Name)
				assert.Equal(t, "Sydney IX", req.NetworkServiceType)
				assert.Equal(t, 65100, req.ASN)
				assert.Equal(t, "AA:BB:CC:DD:EE:FF", req.MACAddress)
				assert.Equal(t, 2000, req.RateLimit)
				assert.Equal(t, 200, req.VLAN)
				assert.True(t, req.Shutdown)
				assert.Equal(t, "PROMO", req.PromoCode)
			},
		},
		{
			name:          "partial fields missing ASN",
			jsonStr:       `{"productUid":"port-123","productName":"Partial IX"}`,
			expectedError: "Invalid ASN",
		},
		{
			name:          "invalid JSON syntax",
			jsonStr:       `{invalid json}`,
			expectedError: "failed to parse JSON",
		},
		{
			name:          "empty JSON object",
			jsonStr:       `{}`,
			expectedError: "Invalid ASN",
		},
		{
			name:          "invalid MAC address in JSON",
			jsonStr:       `{"productUid":"port-123","productName":"IX","asn":65000,"macAddress":"bad-mac","rateLimit":1000,"vlan":100}`,
			expectedError: "Invalid MAC address",
		},
		{
			name:          "invalid rate limit in JSON",
			jsonStr:       `{"productUid":"port-123","productName":"IX","asn":65000,"macAddress":"00:11:22:33:44:55","rateLimit":0,"vlan":100}`,
			expectedError: "Invalid rate limit",
		},
		{
			name: "valid JSON file",
			setupFile: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "ix.json")
				err := os.WriteFile(path, []byte(`{"productUid":"file-port","productName":"File IX","asn":65000,"rateLimit":1000,"vlan":100,"macAddress":"00:11:22:33:44:55","networkServiceType":"London IX"}`), 0644)
				assert.NoError(t, err)
				return path
			},
			validate: func(t *testing.T, req *megaport.BuyIXRequest) {
				assert.Equal(t, "file-port", req.ProductUID)
				assert.Equal(t, "File IX", req.Name)
				assert.Equal(t, 65000, req.ASN)
			},
		},
		{
			name:          "JSON file not found",
			jsonFile:      "/nonexistent/path/ix.json",
			expectedError: "failed to read JSON file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonFile := tt.jsonFile
			if tt.setupFile != nil {
				jsonFile = tt.setupFile(t)
			}

			req, err := buildIXRequestFromJSON(tt.jsonStr, jsonFile)

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

func TestBuildUpdateIXRequestFromFlags(t *testing.T) {
	tests := []struct {
		name     string
		flags    map[string]string
		validate func(t *testing.T, req *megaport.UpdateIXRequest)
	}{
		{
			name: "name only",
			flags: map[string]string{
				"name": "Updated IX",
			},
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.NotNil(t, req.Name)
				assert.Equal(t, "Updated IX", *req.Name)
				assert.Nil(t, req.RateLimit)
				assert.Nil(t, req.VLAN)
			},
		},
		{
			name: "rate-limit only",
			flags: map[string]string{
				"rate-limit": "2000",
			},
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.NotNil(t, req.RateLimit)
				assert.Equal(t, 2000, *req.RateLimit)
				assert.Nil(t, req.Name)
			},
		},
		{
			name: "cost-centre only",
			flags: map[string]string{
				"cost-centre": "Finance",
			},
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.NotNil(t, req.CostCentre)
				assert.Equal(t, "Finance", *req.CostCentre)
			},
		},
		{
			name: "vlan only",
			flags: map[string]string{
				"vlan": "200",
			},
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.NotNil(t, req.VLAN)
				assert.Equal(t, 200, *req.VLAN)
			},
		},
		{
			name: "mac-address only",
			flags: map[string]string{
				"mac-address": "AA:BB:CC:DD:EE:FF",
			},
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.NotNil(t, req.MACAddress)
				assert.Equal(t, "AA:BB:CC:DD:EE:FF", *req.MACAddress)
			},
		},
		{
			name: "asn only",
			flags: map[string]string{
				"asn": "65001",
			},
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.NotNil(t, req.ASN)
				assert.Equal(t, 65001, *req.ASN)
			},
		},
		{
			name: "password only",
			flags: map[string]string{
				"password": "secret123",
			},
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.NotNil(t, req.Password)
				assert.Equal(t, "secret123", *req.Password)
			},
		},
		{
			name: "public-graph only",
			flags: map[string]string{
				"public-graph": "true",
			},
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.NotNil(t, req.PublicGraph)
				assert.True(t, *req.PublicGraph)
			},
		},
		{
			name: "reverse-dns only",
			flags: map[string]string{
				"reverse-dns": "host.example.com",
			},
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.NotNil(t, req.ReverseDns)
				assert.Equal(t, "host.example.com", *req.ReverseDns)
			},
		},
		{
			name: "a-end-product-uid only",
			flags: map[string]string{
				"a-end-product-uid": "port-new-uid",
			},
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.NotNil(t, req.AEndProductUid)
				assert.Equal(t, "port-new-uid", *req.AEndProductUid)
			},
		},
		{
			name: "shutdown only",
			flags: map[string]string{
				"shutdown": "true",
			},
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.NotNil(t, req.Shutdown)
				assert.True(t, *req.Shutdown)
			},
		},
		{
			name: "all flags",
			flags: map[string]string{
				"name":              "Updated IX",
				"rate-limit":        "2000",
				"cost-centre":       "Finance",
				"vlan":              "200",
				"mac-address":       "AA:BB:CC:DD:EE:FF",
				"asn":               "65001",
				"password":          "secret",
				"public-graph":      "true",
				"reverse-dns":       "host.example.com",
				"a-end-product-uid": "port-new",
				"shutdown":          "true",
			},
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.NotNil(t, req.Name)
				assert.NotNil(t, req.RateLimit)
				assert.NotNil(t, req.CostCentre)
				assert.NotNil(t, req.VLAN)
				assert.NotNil(t, req.MACAddress)
				assert.NotNil(t, req.ASN)
				assert.NotNil(t, req.Password)
				assert.NotNil(t, req.PublicGraph)
				assert.NotNil(t, req.ReverseDns)
				assert.NotNil(t, req.AEndProductUid)
				assert.NotNil(t, req.Shutdown)
			},
		},
		{
			name:  "no flags (all nil pointers)",
			flags: map[string]string{},
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.Nil(t, req.Name)
				assert.Nil(t, req.RateLimit)
				assert.Nil(t, req.CostCentre)
				assert.Nil(t, req.VLAN)
				assert.Nil(t, req.MACAddress)
				assert.Nil(t, req.ASN)
				assert.Nil(t, req.Password)
				assert.Nil(t, req.PublicGraph)
				assert.Nil(t, req.ReverseDns)
				assert.Nil(t, req.AEndProductUid)
				assert.Nil(t, req.Shutdown)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("rate-limit", 0, "")
			cmd.Flags().String("cost-centre", "", "")
			cmd.Flags().Int("vlan", 0, "")
			cmd.Flags().String("mac-address", "", "")
			cmd.Flags().Int("asn", 0, "")
			cmd.Flags().String("password", "", "")
			cmd.Flags().Bool("public-graph", false, "")
			cmd.Flags().String("reverse-dns", "", "")
			cmd.Flags().String("a-end-product-uid", "", "")
			cmd.Flags().Bool("shutdown", false, "")

			for k, v := range tt.flags {
				_ = cmd.Flags().Set(k, v)
			}

			req, err := buildUpdateIXRequestFromFlags(cmd)
			assert.NoError(t, err)
			assert.NotNil(t, req)
			tt.validate(t, req)
		})
	}
}

func TestBuildUpdateIXRequestFromFlags_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		flags       map[string]string
		errContains string
	}{
		{
			name:        "invalid ASN zero",
			flags:       map[string]string{"asn": "0"},
			errContains: "Invalid ASN",
		},
		{
			name:        "invalid MAC address",
			flags:       map[string]string{"mac-address": "bad-mac"},
			errContains: "Invalid MAC address",
		},
		{
			name:        "invalid rate limit zero",
			flags:       map[string]string{"rate-limit": "0"},
			errContains: "Invalid rate limit",
		},
		{
			name:        "invalid VLAN reserved",
			flags:       map[string]string{"vlan": "1"},
			errContains: "Invalid VLAN ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().String("name", "", "")
			cmd.Flags().Int("rate-limit", 0, "")
			cmd.Flags().String("cost-centre", "", "")
			cmd.Flags().Int("vlan", 0, "")
			cmd.Flags().String("mac-address", "", "")
			cmd.Flags().Int("asn", 0, "")
			cmd.Flags().String("password", "", "")
			cmd.Flags().Bool("public-graph", false, "")
			cmd.Flags().String("reverse-dns", "", "")
			cmd.Flags().String("a-end-product-uid", "", "")
			cmd.Flags().Bool("shutdown", false, "")

			for k, v := range tt.flags {
				_ = cmd.Flags().Set(k, v)
			}

			req, err := buildUpdateIXRequestFromFlags(cmd)
			assert.Error(t, err)
			assert.Nil(t, req)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestBuildUpdateIXRequestFromJSON_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		jsonStr     string
		errContains string
	}{
		{
			name:        "invalid ASN",
			jsonStr:     `{"asn": 0}`,
			errContains: "Invalid ASN",
		},
		{
			name:        "invalid MAC address",
			jsonStr:     `{"macAddress": "bad-mac"}`,
			errContains: "Invalid MAC address",
		},
		{
			name:        "invalid rate limit",
			jsonStr:     `{"rateLimit": -1}`,
			errContains: "Invalid rate limit",
		},
		{
			name:        "invalid VLAN",
			jsonStr:     `{"vlan": 1}`,
			errContains: "Invalid VLAN ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := buildUpdateIXRequestFromJSON(tt.jsonStr, "")
			assert.Error(t, err)
			assert.Nil(t, req)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestBuildUpdateIXRequestFromJSON(t *testing.T) {
	tests := []struct {
		name          string
		jsonStr       string
		jsonFile      string
		setupFile     func(t *testing.T) string
		expectedError string
		validate      func(t *testing.T, req *megaport.UpdateIXRequest)
	}{
		{
			name:    "valid JSON string with all update fields",
			jsonStr: `{"name":"Updated IX","rateLimit":2000,"costCentre":"Finance","vlan":200,"macAddress":"AA:BB:CC:DD:EE:FF","asn":65001,"password":"secret","publicGraph":true,"reverseDns":"host.example.com","aEndProductUid":"port-new","shutdown":true}`,
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.NotNil(t, req.Name)
				assert.Equal(t, "Updated IX", *req.Name)
				assert.NotNil(t, req.RateLimit)
				assert.Equal(t, 2000, *req.RateLimit)
				assert.NotNil(t, req.CostCentre)
				assert.Equal(t, "Finance", *req.CostCentre)
			},
		},
		{
			name:    "valid JSON string with partial fields",
			jsonStr: `{"name":"Updated IX"}`,
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.NotNil(t, req.Name)
				assert.Equal(t, "Updated IX", *req.Name)
				assert.Nil(t, req.RateLimit)
			},
		},
		{
			name:          "invalid JSON syntax",
			jsonStr:       `{invalid}`,
			expectedError: "failed to parse JSON",
		},
		{
			name:    "empty JSON object",
			jsonStr: `{}`,
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.Nil(t, req.Name)
				assert.Nil(t, req.RateLimit)
			},
		},
		{
			name: "valid JSON file",
			setupFile: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "update.json")
				err := os.WriteFile(path, []byte(`{"name":"File Updated IX","rateLimit":3000}`), 0644)
				assert.NoError(t, err)
				return path
			},
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.NotNil(t, req.Name)
				assert.Equal(t, "File Updated IX", *req.Name)
				assert.NotNil(t, req.RateLimit)
				assert.Equal(t, 3000, *req.RateLimit)
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

			req, err := buildUpdateIXRequestFromJSON(tt.jsonStr, jsonFile)

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

func TestBuildIXRequestFromPrompt(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() {
		utils.SetResourcePrompt(originalPrompt)
	}()

	tests := []struct {
		name          string
		prompts       []string
		expectedError string
		validate      func(t *testing.T, req *megaport.BuyIXRequest)
	}{
		{
			name: "all prompts answered successfully",
			prompts: []string{
				"port-uid-123",
				"Test IX",
				"Los Angeles IX",
				"65000",
				"00:11:22:33:44:55",
				"1000",
				"100",
				"PROMO",
			},
			validate: func(t *testing.T, req *megaport.BuyIXRequest) {
				assert.Equal(t, "port-uid-123", req.ProductUID)
				assert.Equal(t, "Test IX", req.Name)
				assert.Equal(t, "Los Angeles IX", req.NetworkServiceType)
				assert.Equal(t, 65000, req.ASN)
				assert.Equal(t, "00:11:22:33:44:55", req.MACAddress)
				assert.Equal(t, 1000, req.RateLimit)
				assert.Equal(t, 100, req.VLAN)
				assert.Equal(t, "PROMO", req.PromoCode)
			},
		},
		{
			name: "promo code skipped",
			prompts: []string{
				"port-uid-123",
				"Test IX",
				"Los Angeles IX",
				"65000",
				"00:11:22:33:44:55",
				"1000",
				"100",
				"",
			},
			validate: func(t *testing.T, req *megaport.BuyIXRequest) {
				assert.Equal(t, "", req.PromoCode)
			},
		},
		{
			name: "empty product UID",
			prompts: []string{
				"",
			},
			expectedError: "product UID is required",
		},
		{
			name: "empty name",
			prompts: []string{
				"port-uid-123",
				"",
			},
			expectedError: "name is required",
		},
		{
			name: "empty network service type",
			prompts: []string{
				"port-uid-123",
				"Test IX",
				"",
			},
			expectedError: "network service type is required",
		},
		{
			name: "invalid ASN (non-numeric)",
			prompts: []string{
				"port-uid-123",
				"Test IX",
				"Los Angeles IX",
				"notanumber",
			},
			expectedError: "Invalid ASN",
		},
		{
			name: "empty MAC address",
			prompts: []string{
				"port-uid-123",
				"Test IX",
				"Los Angeles IX",
				"65000",
				"",
			},
			expectedError: "Invalid MAC address",
		},
		{
			name: "invalid rate limit (non-numeric)",
			prompts: []string{
				"port-uid-123",
				"Test IX",
				"Los Angeles IX",
				"65000",
				"00:11:22:33:44:55",
				"notanumber",
			},
			expectedError: "Invalid rate limit",
		},
		{
			name: "invalid VLAN (non-numeric)",
			prompts: []string{
				"port-uid-123",
				"Test IX",
				"Los Angeles IX",
				"65000",
				"00:11:22:33:44:55",
				"1000",
				"notanumber",
			},
			expectedError: "Invalid VLAN",
		},
		{
			name: "prompt error on first prompt",
			prompts: []string{
				"ERROR",
			},
			expectedError: "prompt failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			promptIndex := 0
			utils.SetResourcePrompt(func(_, msg string, _ bool) (string, error) {
				if promptIndex < len(tt.prompts) {
					response := tt.prompts[promptIndex]
					promptIndex++
					if response == "ERROR" {
						return "", fmt.Errorf("prompt failed")
					}
					return response, nil
				}
				return "", fmt.Errorf("unexpected prompt call")
			})

			ctx := context.Background()
			req, err := buildIXRequestFromPrompt(ctx, true)

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

func TestBuildUpdateIXRequestFromPrompt(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	originalPasswordPrompt := utils.GetPasswordPrompt()
	defer func() {
		utils.SetResourcePrompt(originalPrompt)
		utils.SetPasswordPrompt(originalPasswordPrompt)
	}()

	tests := []struct {
		name          string
		prompts       []string
		expectedError string
		validate      func(t *testing.T, req *megaport.UpdateIXRequest)
	}{
		{
			name: "update name only",
			prompts: []string{
				"Updated IX", // name
				"",           // rate-limit
				"",           // cost-centre
				"",           // vlan
				"",           // mac-address
				"",           // asn
				"",           // password
				"",           // public-graph
				"",           // reverse-dns
				"",           // a-end-product-uid
				"",           // shutdown
			},
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.NotNil(t, req.Name)
				assert.Equal(t, "Updated IX", *req.Name)
				assert.Nil(t, req.RateLimit)
				assert.Nil(t, req.CostCentre)
				assert.Nil(t, req.VLAN)
				assert.Nil(t, req.MACAddress)
				assert.Nil(t, req.ASN)
				assert.Nil(t, req.Password)
				assert.Nil(t, req.PublicGraph)
				assert.Nil(t, req.ReverseDns)
				assert.Nil(t, req.AEndProductUid)
				assert.Nil(t, req.Shutdown)
			},
		},
		{
			name: "update multiple fields",
			prompts: []string{
				"Updated IX",        // name
				"2000",              // rate-limit
				"Finance",           // cost-centre
				"200",               // vlan
				"AA:BB:CC:DD:EE:FF", // mac-address
				"65001",             // asn
				"secret",            // password
				"yes",               // public-graph
				"host.example.com",  // reverse-dns
				"port-new-uid",      // a-end-product-uid
				"yes",               // shutdown
			},
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.NotNil(t, req.Name)
				assert.Equal(t, "Updated IX", *req.Name)
				assert.NotNil(t, req.RateLimit)
				assert.Equal(t, 2000, *req.RateLimit)
				assert.NotNil(t, req.CostCentre)
				assert.Equal(t, "Finance", *req.CostCentre)
				assert.NotNil(t, req.VLAN)
				assert.Equal(t, 200, *req.VLAN)
				assert.NotNil(t, req.MACAddress)
				assert.Equal(t, "AA:BB:CC:DD:EE:FF", *req.MACAddress)
				assert.NotNil(t, req.ASN)
				assert.Equal(t, 65001, *req.ASN)
				assert.NotNil(t, req.Password)
				assert.Equal(t, "secret", *req.Password)
				assert.NotNil(t, req.PublicGraph)
				assert.True(t, *req.PublicGraph)
				assert.NotNil(t, req.ReverseDns)
				assert.Equal(t, "host.example.com", *req.ReverseDns)
				assert.NotNil(t, req.AEndProductUid)
				assert.Equal(t, "port-new-uid", *req.AEndProductUid)
				assert.NotNil(t, req.Shutdown)
				assert.True(t, *req.Shutdown)
			},
		},
		{
			name: "public-graph and shutdown explicit no",
			prompts: []string{
				"",   // name
				"",   // rate-limit
				"",   // cost-centre
				"",   // vlan
				"",   // mac-address
				"",   // asn
				"",   // password
				"no", // public-graph
				"",   // reverse-dns
				"",   // a-end-product-uid
				"no", // shutdown
			},
			validate: func(t *testing.T, req *megaport.UpdateIXRequest) {
				assert.NotNil(t, req.PublicGraph)
				assert.False(t, *req.PublicGraph)
				assert.NotNil(t, req.Shutdown)
				assert.False(t, *req.Shutdown)
			},
		},
		{
			name: "no fields updated",
			prompts: []string{
				"", // name
				"", // rate-limit
				"", // cost-centre
				"", // vlan
				"", // mac-address
				"", // asn
				"", // password
				"", // public-graph
				"", // reverse-dns
				"", // a-end-product-uid
				"", // shutdown
			},
			expectedError: "at least one field must be updated",
		},
		{
			name: "invalid rate limit",
			prompts: []string{
				"",           // name
				"notanumber", // rate-limit
			},
			expectedError: "Invalid rate limit",
		},
		{
			name: "invalid VLAN",
			prompts: []string{
				"",           // name
				"",           // rate-limit
				"",           // cost-centre
				"notanumber", // vlan
			},
			expectedError: "Invalid VLAN",
		},
		{
			name: "invalid ASN",
			prompts: []string{
				"",           // name
				"",           // rate-limit
				"",           // cost-centre
				"",           // vlan
				"",           // mac-address
				"notanumber", // asn
			},
			expectedError: "Invalid ASN",
		},
		{
			name: "prompt error",
			prompts: []string{
				"ERROR",
			},
			expectedError: "prompt failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			promptIndex := 0
			nextPrompt := func() (string, error) {
				if promptIndex < len(tt.prompts) {
					response := tt.prompts[promptIndex]
					promptIndex++
					if response == "ERROR" {
						return "", fmt.Errorf("prompt failed")
					}
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

			req, err := buildUpdateIXRequestFromPrompt("ix-123", true)

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

func TestPrintIXs_NilIXInSlice(t *testing.T) {
	ixs := []*megaport.IX{
		{
			ProductUID:         "ix-1",
			ProductName:        "Test IX",
			ProvisioningStatus: "LIVE",
		},
		nil,
	}

	var err error
	output.CaptureOutput(func() {
		err = printIXs(ixs, "table", noColor)
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid IX: nil value")
}
