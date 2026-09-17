package mve

import (
	"os"
	"testing"

	"github.com/megaport/megaport-cli/internal/validation"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVendorConfig_AdminPasswordPropagation(t *testing.T) {
	t.Run("cisco AdminPassword set on returned config", func(t *testing.T) {
		cfg, err := ParseVendorConfig(map[string]interface{}{
			"vendor": "cisco", "imageId": float64(1), "productSize": "MEDIUM",
			"mveLabel": "label", "manageLocally": true,
			"adminSshPublicKey": "ssh-rsa", "sshPublicKey": "ssh-rsa",
			"adminPassword": "s3cret", "cloudInit": "#cloud",
			"fmcIpAddress": "10.0.0.1", "fmcRegistrationKey": "key",
			"fmcNatId": "nat",
		})
		require.NoError(t, err)
		cisco, ok := cfg.(*megaport.CiscoConfig)
		require.True(t, ok)
		assert.Equal(t, "s3cret", cisco.AdminPassword)
	})

	t.Run("palo_alto AdminPassword set on returned config", func(t *testing.T) {
		cfg, err := ParseVendorConfig(map[string]interface{}{
			"vendor": "palo_alto", "imageId": float64(1), "productSize": "MEDIUM",
			"sshPublicKey": "ssh-rsa", "adminPassword": "s3cret",
			"licenseData": "license",
		})
		require.NoError(t, err)
		pa, ok := cfg.(*megaport.PaloAltoConfig)
		require.True(t, ok)
		assert.Equal(t, "s3cret", pa.AdminPassword)
		assert.Empty(t, pa.AdminPasswordHash)
	})

	t.Run("cisco AdminPassword empty when key absent", func(t *testing.T) {
		cfg, err := ParseVendorConfig(map[string]interface{}{
			"vendor": "cisco", "imageId": float64(1), "productSize": "MEDIUM",
			"mveLabel": "label", "manageLocally": true,
			"adminSshPublicKey": "ssh-rsa", "sshPublicKey": "ssh-rsa",
			"cloudInit": "#cloud", "fmcIpAddress": "10.0.0.1",
			"fmcRegistrationKey": "key", "fmcNatId": "nat",
		})
		require.NoError(t, err)
		cisco, ok := cfg.(*megaport.CiscoConfig)
		require.True(t, ok)
		assert.Empty(t, cisco.AdminPassword)
	})

	t.Run("palo_alto AdminPasswordHash only, AdminPassword empty", func(t *testing.T) {
		cfg, err := ParseVendorConfig(map[string]interface{}{
			"vendor": "palo_alto", "imageId": float64(1), "productSize": "MEDIUM",
			"sshPublicKey": "ssh-rsa", "adminPasswordHash": "hash",
			"licenseData": "license",
		})
		require.NoError(t, err)
		pa, ok := cfg.(*megaport.PaloAltoConfig)
		require.True(t, ok)
		assert.Equal(t, "hash", pa.AdminPasswordHash)
		assert.Empty(t, pa.AdminPassword)
	})

	t.Run("palo_alto both AdminPassword and AdminPasswordHash accepted", func(t *testing.T) {
		cfg, err := ParseVendorConfig(map[string]interface{}{
			"vendor": "palo_alto", "imageId": float64(1), "productSize": "MEDIUM",
			"sshPublicKey": "ssh-rsa", "adminPassword": "s3cret",
			"adminPasswordHash": "hash", "licenseData": "license",
		})
		require.NoError(t, err)
		pa, ok := cfg.(*megaport.PaloAltoConfig)
		require.True(t, ok)
		assert.Equal(t, "s3cret", pa.AdminPassword)
		assert.Equal(t, "hash", pa.AdminPasswordHash)
	})
}

func TestParseVendorConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]interface{}
		expectedError string
	}{
		{
			name:          "missing vendor field",
			config:        map[string]interface{}{"imageId": float64(1)},
			expectedError: "vendor field is required",
		},
		{
			name:          "unsupported vendor",
			config:        map[string]interface{}{"vendor": "unknown"},
			expectedError: "unsupported vendor",
		},
		{
			name: "6wind success",
			config: map[string]interface{}{
				"vendor": "6wind", "imageId": float64(1), "productSize": "MEDIUM",
				"sshPublicKey": "ssh-rsa AAAA",
			},
		},
		{
			name: "aruba success",
			config: map[string]interface{}{
				"vendor": "aruba", "imageId": float64(1), "productSize": "MEDIUM",
				"accountName": "acct", "accountKey": "key",
			},
		},
		{
			name: "aviatrix success",
			config: map[string]interface{}{
				"vendor": "aviatrix", "imageId": float64(1), "productSize": "MEDIUM",
				"cloudInit": "#cloud-config",
			},
		},
		{
			name: "cisco success",
			config: map[string]interface{}{
				"vendor": "cisco", "imageId": float64(1), "productSize": "MEDIUM",
				"mveLabel": "label", "manageLocally": true,
				"adminSshPublicKey": "ssh-rsa", "sshPublicKey": "ssh-rsa",
				"cloudInit": "#cloud", "fmcIpAddress": "10.0.0.1",
				"fmcRegistrationKey": "key", "fmcNatId": "nat",
			},
		},
		{
			name: "fortinet success",
			config: map[string]interface{}{
				"vendor": "fortinet", "imageId": float64(1), "productSize": "MEDIUM",
				"adminSshPublicKey": "ssh-rsa", "sshPublicKey": "ssh-rsa",
				"licenseData": "license",
			},
		},
		{
			name: "palo_alto success",
			config: map[string]interface{}{
				"vendor": "palo_alto", "imageId": float64(1), "productSize": "MEDIUM",
				"sshPublicKey": "ssh-rsa", "adminPasswordHash": "hash",
				"licenseData": "license",
			},
		},
		{
			name: "prisma success",
			config: map[string]interface{}{
				"vendor": "prisma", "imageId": float64(1), "productSize": "MEDIUM",
				"ionKey": "ion", "secretKey": "secret",
			},
		},
		{
			name: "versa success",
			config: map[string]interface{}{
				"vendor": "versa", "imageId": float64(1), "productSize": "MEDIUM",
				"directorAddress": "dir", "controllerAddress": "ctrl",
				"localAuth": "local", "remoteAuth": "remote", "serialNumber": "SN123",
			},
		},
		{
			name: "vmware success",
			config: map[string]interface{}{
				"vendor": "vmware", "imageId": float64(1), "productSize": "MEDIUM",
				"adminSshPublicKey": "ssh-rsa", "sshPublicKey": "ssh-rsa",
				"vcoAddress": "vco.example.com", "vcoActivationCode": "code",
			},
		},
		{
			name: "meraki success",
			config: map[string]interface{}{
				"vendor": "meraki", "imageId": float64(1), "productSize": "MEDIUM",
				"token": "token123",
			},
		},
		// Missing required field tests
		{
			name:          "6wind missing sshPublicKey",
			config:        map[string]interface{}{"vendor": "6wind", "imageId": float64(1), "productSize": "MEDIUM"},
			expectedError: "sshPublicKey is required",
		},
		{
			name:          "aruba missing accountName",
			config:        map[string]interface{}{"vendor": "aruba", "imageId": float64(1), "productSize": "MEDIUM"},
			expectedError: "accountName is required",
		},
		{
			name:          "aviatrix missing cloudInit",
			config:        map[string]interface{}{"vendor": "aviatrix", "imageId": float64(1), "productSize": "MEDIUM"},
			expectedError: "cloudInit is required",
		},
		{
			name:          "fortinet missing adminSshPublicKey",
			config:        map[string]interface{}{"vendor": "fortinet", "imageId": float64(1), "productSize": "MEDIUM"},
			expectedError: "adminSshPublicKey is required",
		},
		{
			name:          "palo_alto missing sshPublicKey",
			config:        map[string]interface{}{"vendor": "palo_alto", "imageId": float64(1), "productSize": "MEDIUM"},
			expectedError: "sshPublicKey is required",
		},
		{
			// adminPassword / adminPasswordHash combination rules are enforced by
			// ValidatePaloAltoConfig — parsing accepts either, both, or neither.
			name: "palo_alto neither adminPassword nor adminPasswordHash parses without error",
			config: map[string]interface{}{
				"vendor": "palo_alto", "imageId": float64(1), "productSize": "MEDIUM",
				"sshPublicKey": "ssh-rsa", "licenseData": "license",
			},
		},
		{
			name: "palo_alto success with plaintext adminPassword",
			config: map[string]interface{}{
				"vendor": "palo_alto", "imageId": float64(1), "productSize": "MEDIUM",
				"sshPublicKey": "ssh-rsa", "adminPassword": "s3cret",
				"licenseData": "license",
			},
		},
		{
			name: "cisco success with adminPassword",
			config: map[string]interface{}{
				"vendor": "cisco", "imageId": float64(1), "productSize": "MEDIUM",
				"mveLabel": "label", "manageLocally": true,
				"adminSshPublicKey": "ssh-rsa", "sshPublicKey": "ssh-rsa",
				"adminPassword": "s3cret", "cloudInit": "#cloud",
				"fmcIpAddress": "10.0.0.1", "fmcRegistrationKey": "key",
				"fmcNatId": "nat",
			},
		},
		{
			name:          "prisma missing ionKey",
			config:        map[string]interface{}{"vendor": "prisma", "imageId": float64(1), "productSize": "MEDIUM"},
			expectedError: "ionKey is required",
		},
		{
			name:          "versa missing directorAddress",
			config:        map[string]interface{}{"vendor": "versa", "imageId": float64(1), "productSize": "MEDIUM"},
			expectedError: "directorAddress is required",
		},
		{
			name:          "vmware missing adminSshPublicKey",
			config:        map[string]interface{}{"vendor": "vmware", "imageId": float64(1), "productSize": "MEDIUM"},
			expectedError: "adminSshPublicKey is required",
		},
		{
			name:          "meraki missing token",
			config:        map[string]interface{}{"vendor": "meraki", "imageId": float64(1), "productSize": "MEDIUM"},
			expectedError: "token is required",
		},
		{
			name:          "missing imageId",
			config:        map[string]interface{}{"vendor": "cisco", "productSize": "MEDIUM"},
			expectedError: "imageId is required",
		},
		{
			name:          "missing productSize",
			config:        map[string]interface{}{"vendor": "cisco", "imageId": float64(1)},
			expectedError: "productSize is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseVendorConfig(tt.config)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestProcessJSONUpdateMVEInput(t *testing.T) {
	tests := []struct {
		name          string
		jsonStr       string
		jsonFile      string
		writeFile     string
		expectedError string
	}{
		{
			name:    "valid name update",
			jsonStr: `{"name":"updated-mve"}`,
		},
		{
			name:    "valid cost centre update",
			jsonStr: `{"costCentre":"IT-2024"}`,
		},
		{
			name:    "valid term update",
			jsonStr: `{"contractTermMonths":24}`,
		},
		{
			name:          "name wrong type",
			jsonStr:       `{"name":123}`,
			expectedError: "name must be a string",
		},
		{
			name:          "cost centre wrong type",
			jsonStr:       `{"costCentre":true}`,
			expectedError: "costCentre must be a string",
		},
		{
			name:          "contract term wrong type",
			jsonStr:       `{"contractTermMonths":"two years"}`,
			expectedError: "contractTermMonths must be a number",
		},
		{
			name:          "invalid JSON",
			jsonStr:       `{invalid}`,
			expectedError: "failed to parse JSON",
		},
		{
			name:          "file not found",
			jsonFile:      "/nonexistent/path.json",
			expectedError: "failed to read JSON file",
		},
		{
			name:      "valid JSON file",
			writeFile: `{"name":"file-update"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonFile := tt.jsonFile
			if tt.writeFile != "" {
				tmpFile, err := os.CreateTemp("", "mve-update-test-*.json")
				require.NoError(t, err)
				defer os.Remove(tmpFile.Name())
				_, err = tmpFile.WriteString(tt.writeFile)
				require.NoError(t, err)
				tmpFile.Close()
				jsonFile = tmpFile.Name()
			}

			req, _, err := processJSONUpdateMVEInput(tt.jsonStr, jsonFile, "mve-123")
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, req)
				assert.Equal(t, "mve-123", req.MVEID)
			}
		})
	}
}

func TestProcessJSONBuyMVEInput(t *testing.T) {
	validVendorConfig := `"vendorConfig":{"vendor":"6wind","imageId":1,"productSize":"MEDIUM","sshPublicKey":"ssh-rsa AAAA"}`

	tests := []struct {
		name          string
		jsonStr       string
		expectedError string
	}{
		{
			name:    "valid buy request",
			jsonStr: `{"name":"test-mve","term":12,"locationId":5,` + validVendorConfig + `}`,
		},
		{
			name:    "valid buy request with vnics",
			jsonStr: `{"name":"test-mve","term":12,"locationId":5,` + validVendorConfig + `,"vnics":[{"description":"data","vlan":100}]}`,
		},
		{
			name:    "valid buy request with empty vnics array",
			jsonStr: `{"name":"test-mve","term":12,"locationId":5,` + validVendorConfig + `,"vnics":[]}`,
		},
		{
			name:          "name wrong type",
			jsonStr:       `{"name":123}`,
			expectedError: "name must be a string",
		},
		{
			name:          "term wrong type",
			jsonStr:       `{"term":"yearly"}`,
			expectedError: "term must be a number",
		},
		{
			name:          "locationId wrong type",
			jsonStr:       `{"locationId":"five"}`,
			expectedError: "locationId must be a number",
		},
		{
			name:          "diversityZone wrong type",
			jsonStr:       `{"diversityZone":5}`,
			expectedError: "diversityZone must be a string",
		},
		{
			name:          "promoCode wrong type",
			jsonStr:       `{"promoCode":5}`,
			expectedError: "promoCode must be a string",
		},
		{
			name:          "costCentre wrong type",
			jsonStr:       `{"costCentre":true}`,
			expectedError: "costCentre must be a string",
		},
		{
			name:          "vendorConfig wrong type",
			jsonStr:       `{"vendorConfig":"6wind"}`,
			expectedError: "vendorConfig must be an object",
		},
		{
			name:          "vnics wrong type",
			jsonStr:       `{"vnics":"data"}`,
			expectedError: "vnics must be an array",
		},
		{
			name:          "vnics entry wrong type",
			jsonStr:       `{"vnics":["data"]}`,
			expectedError: "vnics[0] must be an object",
		},
		{
			name:          "vnics description wrong type",
			jsonStr:       `{"vnics":[{"description":123}]}`,
			expectedError: "description must be a string",
		},
		{
			name:          "vnics vlan wrong type",
			jsonStr:       `{"vnics":[{"vlan":"hundred"}]}`,
			expectedError: "vlan must be a number",
		},
		{
			name:          "vnics vlan not an integer",
			jsonStr:       `{"vnics":[{"vlan":100.5}]}`,
			expectedError: "vlan must be an integer",
		},
		{
			name:          "vnics vlan reserved",
			jsonStr:       `{"vnics":[{"vlan":1}]}`,
			expectedError: "VLAN ID",
		},
		{
			name:          "vnics vlan out of range",
			jsonStr:       `{"vnics":[{"vlan":4095}]}`,
			expectedError: "VLAN ID",
		},
		{
			name:          "vnics vlan overflow",
			jsonStr:       `{"vnics":[{"vlan":1e18}]}`,
			expectedError: "vlan must be an integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := processJSONBuyMVEInput(tt.jsonStr, "")
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, req)
			}
		})
	}
}

func TestProcessFlagBuyMVEInput_Vnics(t *testing.T) {
	validVendorConfig := `{"vendor":"6wind","imageId":1,"productSize":"MEDIUM","sshPublicKey":"ssh-rsa AAAA"}`

	newBuyCmd := func(t *testing.T, vnics string) *cobra.Command {
		cmd := &cobra.Command{Use: "buy"}
		cmd.Flags().String("name", "", "")
		cmd.Flags().Int("term", 0, "")
		cmd.Flags().Int("location-id", 0, "")
		cmd.Flags().String("diversity-zone", "", "")
		cmd.Flags().String("promo-code", "", "")
		cmd.Flags().String("cost-centre", "", "")
		cmd.Flags().String("vendor-config", "", "")
		cmd.Flags().String("vnics", "", "")
		cmd.Flags().String("resource-tags", "", "")
		cmd.Flags().String("resource-tags-file", "", "")
		require.NoError(t, cmd.Flags().Set("name", "test-mve"))
		require.NoError(t, cmd.Flags().Set("term", "12"))
		require.NoError(t, cmd.Flags().Set("location-id", "5"))
		require.NoError(t, cmd.Flags().Set("vendor-config", validVendorConfig))
		if vnics != "" {
			require.NoError(t, cmd.Flags().Set("vnics", vnics))
		}
		return cmd
	}

	tests := []struct {
		name          string
		vnics         string
		expectedError string
	}{
		{
			name:  "valid vnics",
			vnics: `[{"description":"data","vlan":100}]`,
		},
		{
			name:  "empty vnics array",
			vnics: `[]`,
		},
		{
			name:          "malformed vnics JSON",
			vnics:         `[{`,
			expectedError: "failed to parse vnics JSON string",
		},
		{
			name:          "vnics entry wrong type",
			vnics:         `["data"]`,
			expectedError: "vnics[0] must be an object",
		},
		{
			name:          "vnics description wrong type",
			vnics:         `[{"description":123}]`,
			expectedError: "description must be a string",
		},
		{
			name:          "vnics vlan wrong type",
			vnics:         `[{"vlan":"hundred"}]`,
			expectedError: "vlan must be a number",
		},
		{
			// Regression: a string VLAN used to be silently dropped, yielding VLAN 0.
			// Since VLANs are immutable after provisioning, that must error instead.
			name:          "vnics vlan as numeric string is rejected instead of silently zeroed",
			vnics:         `[{"description":"data","vlan":"100"}]`,
			expectedError: "vlan must be a number",
		},
		{
			// Regression: a fractional VLAN used to be silently truncated (100.9 -> 100).
			name:          "vnics vlan not an integer",
			vnics:         `[{"vlan":100.5}]`,
			expectedError: "vlan must be an integer",
		},
		{
			name:          "vnics vlan reserved",
			vnics:         `[{"vlan":1}]`,
			expectedError: "VLAN ID",
		},
		{
			name:          "vnics vlan out of range",
			vnics:         `[{"vlan":4095}]`,
			expectedError: "VLAN ID",
		},
		{
			name:          "vnics vlan overflow",
			vnics:         `[{"vlan":1e18}]`,
			expectedError: "vlan must be an integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := processFlagBuyMVEInput(newBuyCmd(t, tt.vnics))
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, req)
		})
	}

	t.Run("vnic fields round-trip", func(t *testing.T) {
		req, err := processFlagBuyMVEInput(newBuyCmd(t, `[{"description":"data","vlan":100}]`))
		require.NoError(t, err)
		require.Len(t, req.Vnics, 1)
		assert.Equal(t, "data", req.Vnics[0].Description)
		assert.Equal(t, 100, req.Vnics[0].VLAN)
	})
}

func TestProcessJSONBuyMVEInput_OptionalStringsRoundTrip(t *testing.T) {
	jsonStr := `{"name":"test-mve","term":12,"locationId":5,"diversityZone":"red","promoCode":"PROMO","costCentre":"CC-1","vendorConfig":{"vendor":"6wind","imageId":1,"productSize":"MEDIUM","sshPublicKey":"ssh-rsa AAAA"}}`

	req, err := processJSONBuyMVEInput(jsonStr, "")
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, "red", req.DiversityZone)
	assert.Equal(t, "PROMO", req.PromoCode)
	assert.Equal(t, "CC-1", req.CostCentre)
}

func TestProcessFlagUpdateMVEInput(t *testing.T) {
	tests := []struct {
		name          string
		nameVal       string
		costCentre    string
		contractTerm  int
		expectedError string
	}{
		{
			name:    "name update",
			nameVal: "new-mve",
		},
		{
			name:       "cost centre update",
			costCentre: "IT-2024",
		},
		{
			name:         "contract term update",
			contractTerm: 24,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := createTestCmd()
			if tt.nameVal != "" {
				require.NoError(t, cmd.Flags().Set("name", tt.nameVal))
			}
			if tt.costCentre != "" {
				require.NoError(t, cmd.Flags().Set("cost-centre", tt.costCentre))
			}
			if tt.contractTerm > 0 {
				require.NoError(t, cmd.Flags().Set("term", "24"))
			}

			req, _, err := processFlagUpdateMVEInput(cmd, "mve-123")
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, req)
			}
		})
	}
}

func TestProcessFlagUpdateMVEInput_ExplicitEmptyNameRejected(t *testing.T) {
	cmd := createTestCmd()
	require.NoError(t, cmd.Flags().Set("name", ""))
	_, _, err := processFlagUpdateMVEInput(cmd, "mve-123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestProcessJSONUpdateMVEInput_ExplicitEmptyNameRejected(t *testing.T) {
	_, _, err := processJSONUpdateMVEInput(`{"name":""}`, "", "mve-123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestProcessFlagUpdateMVEInput_CostCentreProvided(t *testing.T) {
	t.Run("flag not set reports not provided", func(t *testing.T) {
		cmd := createTestCmd()
		require.NoError(t, cmd.Flags().Set("name", "renamed"))
		req, provided, err := processFlagUpdateMVEInput(cmd, "mve-123")
		require.NoError(t, err)
		assert.False(t, provided)
		assert.Empty(t, req.CostCentre)
	})

	t.Run("flag set to a value reports provided", func(t *testing.T) {
		cmd := createTestCmd()
		require.NoError(t, cmd.Flags().Set("cost-centre", "CC-1"))
		req, provided, err := processFlagUpdateMVEInput(cmd, "mve-123")
		require.NoError(t, err)
		assert.True(t, provided)
		assert.Equal(t, "CC-1", req.CostCentre)
	})

	t.Run("flag set to empty reports provided and empty", func(t *testing.T) {
		cmd := createTestCmd()
		require.NoError(t, cmd.Flags().Set("cost-centre", ""))
		req, provided, err := processFlagUpdateMVEInput(cmd, "mve-123")
		require.NoError(t, err)
		assert.True(t, provided)
		assert.Empty(t, req.CostCentre)
	})
}

func TestProcessJSONUpdateMVEInput_CostCentreProvided(t *testing.T) {
	t.Run("key absent reports not provided", func(t *testing.T) {
		req, provided, err := processJSONUpdateMVEInput(`{"name":"renamed"}`, "", "mve-123")
		require.NoError(t, err)
		assert.False(t, provided)
		assert.Empty(t, req.CostCentre)
	})

	t.Run("key present with value reports provided", func(t *testing.T) {
		req, provided, err := processJSONUpdateMVEInput(`{"costCentre":"CC-1"}`, "", "mve-123")
		require.NoError(t, err)
		assert.True(t, provided)
		assert.Equal(t, "CC-1", req.CostCentre)
	})

	t.Run("key present but empty reports provided and empty", func(t *testing.T) {
		req, provided, err := processJSONUpdateMVEInput(`{"costCentre":""}`, "", "mve-123")
		require.NoError(t, err)
		assert.True(t, provided)
		assert.Empty(t, req.CostCentre)
	})
}

func createTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("name", "", "")
	cmd.Flags().String("cost-centre", "", "")
	cmd.Flags().Int("term", 0, "")
	cmd.Flags().String("vnics", "", "")
	return cmd
}

func TestProcessJSONUpdateMVEInput_Vnics(t *testing.T) {
	req, _, err := processJSONUpdateMVEInput(`{"vnics":[{"description":"Data Plane"},{"description":"Management"}]}`, "", "mve-123")
	require.NoError(t, err)
	require.NotNil(t, req)
	require.Len(t, req.Vnics, 2)
	assert.Equal(t, "Data Plane", req.Vnics[0].Description)
	assert.Equal(t, "Management", req.Vnics[1].Description)
}

func TestProcessJSONUpdateMVEInput_VnicMissingDescription(t *testing.T) {
	_, _, err := processJSONUpdateMVEInput(`{"vnics":[{}]}`, "", "mve-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vnics[0].description")
}

func TestProcessFlagUpdateMVEInput_Vnics(t *testing.T) {
	cmd := createTestCmd()
	require.NoError(t, cmd.Flags().Set("vnics", `[{"description":"Data Plane"}]`))

	req, _, err := processFlagUpdateMVEInput(cmd, "mve-123")
	require.NoError(t, err)
	require.NotNil(t, req)
	require.Len(t, req.Vnics, 1)
	assert.Equal(t, "Data Plane", req.Vnics[0].Description)
}

func TestProcessFlagUpdateMVEInput_VnicsInvalidJSON(t *testing.T) {
	cmd := createTestCmd()
	require.NoError(t, cmd.Flags().Set("vnics", `[{`))

	_, _, err := processFlagUpdateMVEInput(cmd, "mve-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse vnics JSON")
}

func TestProcessJSONUpdateMVEInput_VnicEntryNotObject(t *testing.T) {
	_, _, err := processJSONUpdateMVEInput(`{"vnics":["not-an-object"]}`, "", "mve-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vnics[0] must be an object")
}

func TestProcessFlagUpdateMVEInput_VnicEntryNotObject(t *testing.T) {
	cmd := createTestCmd()
	require.NoError(t, cmd.Flags().Set("vnics", `["not-an-object"]`))

	_, _, err := processFlagUpdateMVEInput(cmd, "mve-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vnics[0] must be an object")
}

func TestProcessJSONUpdateMVEInput_VnicsNotArray(t *testing.T) {
	cases := []string{
		`{"vnics":{"description":"Data Plane"}}`,
		`{"vnics":"Data Plane"}`,
		`{"vnics":null}`,
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			_, _, err := processJSONUpdateMVEInput(in, "", "mve-123")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "vnics must be an array")
		})
	}
}

func TestProcessJSONUpdateMVEInput_VnicUnsupportedKey(t *testing.T) {
	_, _, err := processJSONUpdateMVEInput(`{"vnics":[{"description":"Data Plane","vlan":100}]}`, "", "mve-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vnics[0].vlan is not supported")
}

func TestProcessJSONUpdateMVEInput_VnicEmptyDescription(t *testing.T) {
	cases := []string{
		`{"vnics":[{"description":""}]}`,
		`{"vnics":[{"description":"   "}]}`,
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			_, _, err := processJSONUpdateMVEInput(in, "", "mve-123")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "vnics[0].description must not be empty")
		})
	}
}

func TestProcessJSONUpdateMVEInput_VnicTrimsDescription(t *testing.T) {
	req, _, err := processJSONUpdateMVEInput(`{"vnics":[{"description":"  Data Plane  "}]}`, "", "mve-123")
	require.NoError(t, err)
	require.NotNil(t, req)
	require.Len(t, req.Vnics, 1)
	assert.Equal(t, "Data Plane", req.Vnics[0].Description)
}

func TestProcessJSONUpdateMVEInput_VnicsEmptyArray(t *testing.T) {
	_, _, err := processJSONUpdateMVEInput(`{"vnics":[]}`, "", "mve-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vnics must contain at least one object")
}

func TestProcessFlagUpdateMVEInput_VnicsEmptyArray(t *testing.T) {
	cmd := createTestCmd()
	require.NoError(t, cmd.Flags().Set("vnics", `[]`))

	_, _, err := processFlagUpdateMVEInput(cmd, "mve-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vnics must contain at least one object")
}

func TestProcessFlagUpdateMVEInput_TermZeroIsValidationError(t *testing.T) {
	cmd := createTestCmd()
	require.NoError(t, cmd.Flags().Set("term", "0"))

	_, _, err := processFlagUpdateMVEInput(cmd, "mve-123")
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "at least one field must be provided")
}

func TestProcessJSONUpdateMVEInput_TermZeroIsValidationError(t *testing.T) {
	_, _, err := processJSONUpdateMVEInput(`{"contractTermMonths":0}`, "", "mve-123")
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "at least one field must be provided")
}

func TestProcessFlagUpdateMVEInput_VnicsEmptyString(t *testing.T) {
	cmd := createTestCmd()
	require.NoError(t, cmd.Flags().Set("vnics", ""))

	_, _, err := processFlagUpdateMVEInput(cmd, "mve-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vnics must be a non-empty JSON array")
}

func TestProcessFlagUpdateMVEInput_VnicsWhitespaceString(t *testing.T) {
	cmd := createTestCmd()
	require.NoError(t, cmd.Flags().Set("vnics", "   "))

	_, _, err := processFlagUpdateMVEInput(cmd, "mve-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vnics must be a non-empty JSON array")
}

// parseCiscoConfig must mirror ValidateCiscoConfig: FMC fields are only
// required when not managing locally, and mveLabel/cloudInit are optional.
func TestParseCiscoConfig_ManageLocallyParity(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]interface{}
		expectedError string
	}{
		{
			name: "locally managed accepted without FMC, mveLabel, or cloudInit",
			config: map[string]interface{}{
				"vendor": "cisco", "imageId": float64(1), "productSize": "MEDIUM",
				"manageLocally":     true,
				"adminSshPublicKey": "ssh-rsa AAAA", "sshPublicKey": "ssh-rsa AAAA",
			},
		},
		{
			name: "FMC managed accepted when manageLocally absent and FMC present",
			config: map[string]interface{}{
				"vendor": "cisco", "imageId": float64(1), "productSize": "MEDIUM",
				"adminSshPublicKey": "ssh-rsa AAAA", "sshPublicKey": "ssh-rsa AAAA",
				"fmcIpAddress": "10.0.0.1", "fmcRegistrationKey": "key", "fmcNatId": "nat",
			},
		},
		{
			name: "FMC managed rejected when FMC fields missing",
			config: map[string]interface{}{
				"vendor": "cisco", "imageId": float64(1), "productSize": "MEDIUM",
				"manageLocally":     false,
				"adminSshPublicKey": "ssh-rsa AAAA", "sshPublicKey": "ssh-rsa AAAA",
			},
			expectedError: "fmcIpAddress is required for Cisco configuration when not managing locally",
		},
		{
			name: "locally managed still requires adminSshPublicKey",
			config: map[string]interface{}{
				"vendor": "cisco", "imageId": float64(1), "productSize": "MEDIUM",
				"manageLocally": true, "sshPublicKey": "ssh-rsa AAAA",
			},
			expectedError: "adminSshPublicKey is required",
		},
		{
			name: "locally managed still requires sshPublicKey",
			config: map[string]interface{}{
				"vendor": "cisco", "imageId": float64(1), "productSize": "MEDIUM",
				"manageLocally": true, "adminSshPublicKey": "ssh-rsa AAAA",
			},
			expectedError: "sshPublicKey is required",
		},
		{
			name: "non-boolean manageLocally is a clear type error",
			config: map[string]interface{}{
				"vendor": "cisco", "imageId": float64(1), "productSize": "MEDIUM",
				"manageLocally":     "true",
				"adminSshPublicKey": "ssh-rsa AAAA", "sshPublicKey": "ssh-rsa AAAA",
			},
			expectedError: "manageLocally must be a boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseVendorConfig(tt.config)
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}
			require.NoError(t, err)
			require.NoError(t, validation.ValidateMVEVendorConfig(cfg),
				"a config accepted by the parser must also pass the validator")
		})
	}
}

// The documented `mve buy` Cisco examples must parse and validate on both the
// flags path and the JSON path.
func TestBuyMVE_CiscoConfigPaths(t *testing.T) {
	cases := []struct {
		name         string
		vendorConfig string
	}{
		{
			name:         "locally managed (documented example)",
			vendorConfig: `{"vendor":"cisco","imageId":123,"productSize":"MEDIUM","manageLocally":true,"adminSshPublicKey":"ssh-rsa AAAA...","sshPublicKey":"ssh-rsa AAAA..."}`,
		},
		{
			name:         "FMC managed",
			vendorConfig: `{"vendor":"cisco","imageId":123,"productSize":"MEDIUM","adminSshPublicKey":"ssh-rsa AAAA...","sshPublicKey":"ssh-rsa AAAA...","fmcIpAddress":"10.0.0.1","fmcRegistrationKey":"key","fmcNatId":"nat"}`,
		},
	}

	newCiscoBuyCmd := func(t *testing.T, vendorConfig string) *cobra.Command {
		cmd := &cobra.Command{Use: "buy"}
		cmd.Flags().String("name", "", "")
		cmd.Flags().Int("term", 0, "")
		cmd.Flags().Int("location-id", 0, "")
		cmd.Flags().String("diversity-zone", "", "")
		cmd.Flags().String("promo-code", "", "")
		cmd.Flags().String("cost-centre", "", "")
		cmd.Flags().String("vendor-config", "", "")
		cmd.Flags().String("vnics", "", "")
		cmd.Flags().String("resource-tags", "", "")
		cmd.Flags().String("resource-tags-file", "", "")
		require.NoError(t, cmd.Flags().Set("name", "My MVE"))
		require.NoError(t, cmd.Flags().Set("term", "12"))
		require.NoError(t, cmd.Flags().Set("location-id", "123"))
		require.NoError(t, cmd.Flags().Set("vendor-config", vendorConfig))
		require.NoError(t, cmd.Flags().Set("vnics", `[{"description":"Data Plane","vlan":100}]`))
		return cmd
	}

	for _, tc := range cases {
		t.Run(tc.name+" via flags", func(t *testing.T) {
			req, err := processFlagBuyMVEInput(newCiscoBuyCmd(t, tc.vendorConfig))
			require.NoError(t, err)
			require.NoError(t, validation.ValidateMVEVendorConfig(req.VendorConfig))
		})

		t.Run(tc.name+" via JSON", func(t *testing.T) {
			jsonStr := `{"name":"My MVE","term":12,"locationId":123,"vendorConfig":` + tc.vendorConfig + `,"vnics":[{"description":"Data Plane","vlan":100}]}`
			req, err := processJSONBuyMVEInput(jsonStr, "")
			require.NoError(t, err)
			require.NoError(t, validation.ValidateMVEVendorConfig(req.VendorConfig))
		})
	}
}
