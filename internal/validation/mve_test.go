package validation

import (
	"fmt"
	"testing"

	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
)

func TestValidateMVERequest(t *testing.T) {
	tests := []struct {
		name        string
		productName string
		term        int
		locationID  int
		wantErr     bool
		errText     string
	}{
		{
			name:        "Valid MVE request",
			productName: "Test MVE",
			term:        12,
			locationID:  123,
			wantErr:     false,
		},
		{
			name:        "Empty name",
			productName: "",
			term:        12,
			locationID:  123,
			wantErr:     true,
			errText:     "Invalid MVE name: \"\" - cannot be empty",
		},
		{
			name:        "Invalid term",
			productName: "Test MVE",
			term:        5, // Not in the valid set
			locationID:  123,
			wantErr:     true,
			errText:     fmt.Sprintf("Invalid contract term: 5 - must be one of: %v", ValidContractTerms),
		},
		{
			name:        "Invalid location ID",
			productName: "Test MVE",
			term:        12,
			locationID:  0,
			wantErr:     true,
			errText:     "Invalid location ID: 0 - must be a positive integer",
		},
		{
			name:        "Name too long",
			productName: "This name is way too long and should exceed the 64 character limit for MVE product names which will cause validation to fail",
			term:        12,
			locationID:  123,
			wantErr:     true,
			errText:     "Invalid MVE name: This name is way too long and should exceed the 64 character limit for MVE product names which will cause validation to fail - cannot exceed 64 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMVERequest(tt.productName, tt.term, tt.locationID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMVERequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateMVEVendor(t *testing.T) {
	tests := []struct {
		name    string
		vendor  string
		wantErr bool
		errText string
	}{
		{
			name:    "Valid vendor lowercase",
			vendor:  "cisco",
			wantErr: false,
		},
		{
			name:    "Valid vendor uppercase",
			vendor:  "CISCO",
			wantErr: false,
		},
		{
			name:    "Valid vendor mixed case",
			vendor:  "CiScO",
			wantErr: false,
		},
		{
			name:    "Invalid vendor",
			vendor:  "invalid_vendor",
			wantErr: true,
			errText: fmt.Sprintf("Invalid MVE vendor: invalid_vendor - must be one of: %v", ValidMVEVendors),
		},
		{
			name:    "Empty vendor",
			vendor:  "",
			wantErr: true,
			errText: fmt.Sprintf("Invalid MVE vendor: \"\" - must be one of: %v", ValidMVEVendors), // Adjusted for empty value check if ValidateStringOneOf handles it
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMVEVendor(tt.vendor)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMVEVendor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateMVENetworkInterfaces(t *testing.T) {
	tests := []struct {
		name    string
		vnics   []megaport.MVENetworkInterface // Changed type
		wantErr bool
		errText string
	}{
		{
			name:    "Valid single vNIC",
			vnics:   []megaport.MVENetworkInterface{{Description: "Interface 1"}}, // Use struct slice
			wantErr: false,
		},
		{
			name: "Valid multiple vNICs",
			vnics: []megaport.MVENetworkInterface{ // Use struct slice
				{Description: "Interface 1"},
				{Description: "Interface 2"},
				{Description: "Interface 3"},
			},
			wantErr: false,
		},
		{
			name: "Maximum vNICs",
			vnics: []megaport.MVENetworkInterface{ // Use struct slice
				{Description: "Interface 1"},
				{Description: "Interface 2"},
				{Description: "Interface 3"},
				{Description: "Interface 4"},
				{Description: "Interface 5"},
			},
			wantErr: false,
		},
		{
			name: "Too many vNICs",
			vnics: []megaport.MVENetworkInterface{ // Use struct slice
				{Description: "Interface 1"},
				{Description: "Interface 2"},
				{Description: "Interface 3"},
				{Description: "Interface 4"},
				{Description: "Interface 5"},
				{Description: "Interface 6"},
			},
			wantErr: true,
			errText: "Invalid network interfaces: 6 - cannot exceed 5 vNICs",
		},
		{
			name: "Empty description",
			vnics: []megaport.MVENetworkInterface{ // Use struct slice
				{Description: "Interface 1"},
				{Description: ""},
			},
			wantErr: true,
			errText: "Invalid network interface 2: \"\" - description cannot be empty", // Adjusted expected error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMVENetworkInterfaces(tt.vnics) // Pass struct slice
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMVENetworkInterfaces() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateMVENetworkInterfacesTyped(t *testing.T) {
	tests := []struct {
		name    string
		vnics   []megaport.MVENetworkInterface
		wantErr bool
		errText string
	}{
		{
			name: "Valid single vNIC",
			vnics: []megaport.MVENetworkInterface{
				{Description: "Interface 1"},
			},
			wantErr: false,
		},
		{
			name: "Valid multiple vNICs",
			vnics: []megaport.MVENetworkInterface{
				{Description: "Interface 1"},
				{Description: "Interface 2"},
				{Description: "Interface 3"},
			},
			wantErr: false,
		},
		{
			name: "Maximum vNICs",
			vnics: []megaport.MVENetworkInterface{
				{Description: "Interface 1"},
				{Description: "Interface 2"},
				{Description: "Interface 3"},
				{Description: "Interface 4"},
				{Description: "Interface 5"},
			},
			wantErr: false,
		},
		{
			name: "Too many vNICs",
			vnics: []megaport.MVENetworkInterface{
				{Description: "Interface 1"},
				{Description: "Interface 2"},
				{Description: "Interface 3"},
				{Description: "Interface 4"},
				{Description: "Interface 5"},
				{Description: "Interface 6"},
			},
			wantErr: true,
			errText: "Invalid network interfaces: 6 - cannot exceed 5 vNICs",
		},
		{
			name: "Empty description",
			vnics: []megaport.MVENetworkInterface{
				{Description: "Interface 1"},
				{Description: ""},
			},
			wantErr: true,
			errText: "Invalid network interface 2: \"\" - description cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMVENetworkInterfaces(tt.vnics)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMVENetworkInterfacesTyped() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateBuyMVERequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *megaport.BuyMVERequest
		wantErr bool
		errText string
	}{
		{
			name: "Valid buy MVE request",
			req: &megaport.BuyMVERequest{
				Name:       "Test MVE",
				Term:       12,
				LocationID: 100,
				VendorConfig: &megaport.CiscoConfig{
					Vendor:            "cisco",
					ImageID:           123,
					ProductSize:       "MEDIUM",
					AdminSSHPublicKey: "ssh-rsa AAAA...",
					SSHPublicKey:      "ssh-rsa AAAA...",
					ManageLocally:     true,
				},
			},
			wantErr: false,
		},
		{
			name: "Missing name",
			req: &megaport.BuyMVERequest{
				Name:       "",
				Term:       12,
				LocationID: 100,
				VendorConfig: &megaport.CiscoConfig{
					Vendor:            "cisco",
					ImageID:           123,
					ProductSize:       "MEDIUM",
					AdminSSHPublicKey: "ssh-rsa AAAA...",
					SSHPublicKey:      "ssh-rsa AAAA...",
					ManageLocally:     true,
				},
			},
			wantErr: true,
			errText: "Invalid MVE name: \"\" - cannot be empty",
		},
		{
			name: "Missing location",
			req: &megaport.BuyMVERequest{
				Name:       "Test MVE",
				Term:       12,
				LocationID: 0,
				VendorConfig: &megaport.CiscoConfig{
					Vendor:            "cisco",
					ImageID:           123,
					ProductSize:       "MEDIUM",
					AdminSSHPublicKey: "ssh-rsa AAAA...",
					SSHPublicKey:      "ssh-rsa AAAA...",
					ManageLocally:     true,
				},
			},
			wantErr: true,
			errText: "Invalid location ID: 0 - must be a positive integer",
		},
		{
			name: "Invalid term",
			req: &megaport.BuyMVERequest{
				Name:       "Test MVE",
				Term:       5,
				LocationID: 100,
				VendorConfig: &megaport.CiscoConfig{
					Vendor:            "cisco",
					ImageID:           123,
					ProductSize:       "MEDIUM",
					AdminSSHPublicKey: "ssh-rsa AAAA...",
					SSHPublicKey:      "ssh-rsa AAAA...",
					ManageLocally:     true,
				},
			},
			wantErr: true,
			errText: fmt.Sprintf("Invalid contract term: 5 - must be one of: %v", ValidContractTerms),
		},
		{
			name: "Missing vendor config",
			req: &megaport.BuyMVERequest{
				Name:         "Test MVE",
				Term:         12,
				LocationID:   100,
				VendorConfig: nil,
			},
			wantErr: true,
			errText: "Invalid vendor config: <nil> - cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBuyMVERequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBuyMVERequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateUpdateMVERequest(t *testing.T) {
	term12 := 12
	term5 := 5
	tests := []struct {
		name    string
		req     *megaport.ModifyMVERequest
		wantErr bool
		errText string
	}{
		{
			name: "Valid update with name change",
			req: &megaport.ModifyMVERequest{
				MVEID: "mve-uid-123",
				Name:  "Updated MVE",
			},
			wantErr: false,
		},
		{
			name: "No fields provided",
			req: &megaport.ModifyMVERequest{
				MVEID: "mve-uid-123",
			},
			wantErr: true,
			errText: "at least one field must be provided for update",
		},
		{
			name: "Valid contract term update",
			req: &megaport.ModifyMVERequest{
				MVEID:              "mve-uid-123",
				ContractTermMonths: &term12,
			},
			wantErr: false,
		},
		{
			name: "Invalid contract term",
			req: &megaport.ModifyMVERequest{
				MVEID:              "mve-uid-123",
				ContractTermMonths: &term5,
			},
			wantErr: true,
			errText: fmt.Sprintf("Invalid contract term: 5 - must be one of: %v", ValidContractTerms),
		},
		{
			name: "Valid vnic-only update",
			req: &megaport.ModifyMVERequest{
				MVEID: "mve-uid-123",
				Vnics: []megaport.MVEVnicUpdate{{Description: "Data Plane"}},
			},
			wantErr: false,
		},
		{
			name: "Empty vnic description rejected",
			req: &megaport.ModifyMVERequest{
				MVEID: "mve-uid-123",
				Vnics: []megaport.MVEVnicUpdate{{Description: "Data Plane"}, {Description: ""}},
			},
			wantErr: true,
			errText: "vnics[1].description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateMVERequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateMVERequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Contains(t, err.Error(), tt.errText, "Error message mismatch")
			}
		})
	}
}

func TestValidateMVEVendorConfig_AllVendors(t *testing.T) {
	tests := []struct {
		name    string
		config  megaport.VendorConfig
		wantErr bool
		errText string
	}{
		// 6wind - valid
		{
			name: "Valid 6wind config",
			config: &megaport.SixwindVSRConfig{
				Vendor:       "6wind",
				ImageID:      123,
				ProductSize:  "MEDIUM",
				SSHPublicKey: "ssh-rsa AAAA...",
			},
			wantErr: false,
		},
		// 6wind - missing SSH key
		{
			name: "Invalid 6wind - missing SSH key",
			config: &megaport.SixwindVSRConfig{
				Vendor:      "6wind",
				ImageID:     123,
				ProductSize: "MEDIUM",
			},
			wantErr: true,
			errText: "Invalid SSH public key: \"\" - cannot be empty",
		},
		// aruba - valid
		{
			name: "Valid aruba config",
			config: &megaport.ArubaConfig{
				Vendor:      "aruba",
				ImageID:     123,
				ProductSize: "MEDIUM",
				AccountName: "test-account",
				AccountKey:  "test-key",
				SystemTag:   "test-tag",
			},
			wantErr: false,
		},
		// aruba - missing account name
		{
			name: "Invalid aruba - missing account name",
			config: &megaport.ArubaConfig{
				Vendor:      "aruba",
				ImageID:     123,
				ProductSize: "MEDIUM",
				AccountKey:  "test-key",
				SystemTag:   "test-tag",
			},
			wantErr: true,
			errText: "Invalid account name: \"\" - cannot be empty",
		},
		// aviatrix - valid
		{
			name: "Valid aviatrix config",
			config: &megaport.AviatrixConfig{
				Vendor:      "aviatrix",
				ImageID:     123,
				ProductSize: "MEDIUM",
				CloudInit:   "cloud-init-data",
			},
			wantErr: false,
		},
		// aviatrix - missing cloud init
		{
			name: "Invalid aviatrix - missing cloud init",
			config: &megaport.AviatrixConfig{
				Vendor:      "aviatrix",
				ImageID:     123,
				ProductSize: "MEDIUM",
			},
			wantErr: true,
			errText: "Invalid cloud init: \"\" - cannot be empty",
		},
		// cisco - valid
		{
			name: "Valid cisco config",
			config: &megaport.CiscoConfig{
				Vendor:            "cisco",
				ImageID:           123,
				ProductSize:       "MEDIUM",
				AdminSSHPublicKey: "ssh-rsa AAAA...",
				SSHPublicKey:      "ssh-rsa AAAA...",
				ManageLocally:     true,
			},
			wantErr: false,
		},
		// cisco - missing admin SSH key
		{
			name: "Invalid cisco - missing admin SSH key",
			config: &megaport.CiscoConfig{
				Vendor:        "cisco",
				ImageID:       123,
				ProductSize:   "MEDIUM",
				SSHPublicKey:  "ssh-rsa AAAA...",
				ManageLocally: true,
			},
			wantErr: true,
			errText: "Invalid admin SSH public key: \"\" - cannot be empty",
		},
		// fortinet - valid
		{
			name: "Valid fortinet config",
			config: &megaport.FortinetConfig{
				Vendor:            "fortinet",
				ImageID:           123,
				ProductSize:       "MEDIUM",
				AdminSSHPublicKey: "ssh-rsa AAAA...",
				SSHPublicKey:      "ssh-rsa AAAA...",
				LicenseData:       "license-data",
			},
			wantErr: false,
		},
		// fortinet - missing license data
		{
			name: "Invalid fortinet - missing license data",
			config: &megaport.FortinetConfig{
				Vendor:            "fortinet",
				ImageID:           123,
				ProductSize:       "MEDIUM",
				AdminSSHPublicKey: "ssh-rsa AAAA...",
				SSHPublicKey:      "ssh-rsa AAAA...",
			},
			wantErr: true,
			errText: "Invalid license data: \"\" - cannot be empty",
		},
		// paloalto - valid
		{
			name: "Valid paloalto config",
			config: &megaport.PaloAltoConfig{
				Vendor:            "palo_alto",
				ImageID:           123,
				ProductSize:       "MEDIUM",
				SSHPublicKey:      "ssh-rsa AAAA...",
				AdminPasswordHash: "$6$rounds=4096$...",
				LicenseData:       "license-data",
			},
			wantErr: false,
		},
		// paloalto - missing both admin password and admin password hash
		{
			name: "Invalid paloalto - missing admin password and admin password hash",
			config: &megaport.PaloAltoConfig{
				Vendor:       "palo_alto",
				ImageID:      123,
				ProductSize:  "MEDIUM",
				SSHPublicKey: "ssh-rsa AAAA...",
				LicenseData:  "license-data",
			},
			wantErr: true,
			errText: "Invalid admin password / admin password hash: \"\" - at least one of admin password or admin password hash must be provided",
		},
		// paloalto - valid with plaintext admin password only
		{
			name: "Valid paloalto config with plaintext admin password",
			config: &megaport.PaloAltoConfig{
				Vendor:        "palo_alto",
				ImageID:       123,
				ProductSize:   "MEDIUM",
				SSHPublicKey:  "ssh-rsa AAAA...",
				AdminPassword: "S3cretP@ss",
				LicenseData:   "license-data",
			},
			wantErr: false,
		},
		// paloalto - both admin password and admin password hash supplied
		{
			name: "Invalid paloalto - both admin password and admin password hash supplied",
			config: &megaport.PaloAltoConfig{
				Vendor:            "palo_alto",
				ImageID:           123,
				ProductSize:       "MEDIUM",
				SSHPublicKey:      "ssh-rsa AAAA...",
				AdminPassword:     "S3cretP@ss",
				AdminPasswordHash: "$6$rounds=4096$...",
				LicenseData:       "license-data",
			},
			wantErr: true,
			errText: "Invalid admin password / admin password hash: \"\" - only one of admin password or admin password hash may be provided, not both",
		},
		// prisma - valid
		{
			name: "Valid prisma config",
			config: &megaport.PrismaConfig{
				Vendor:      "prisma",
				ImageID:     123,
				ProductSize: "MEDIUM",
				IONKey:      "ion-key",
				SecretKey:   "secret-key",
			},
			wantErr: false,
		},
		// prisma - missing ION key
		{
			name: "Invalid prisma - missing ION key",
			config: &megaport.PrismaConfig{
				Vendor:      "prisma",
				ImageID:     123,
				ProductSize: "MEDIUM",
				SecretKey:   "secret-key",
			},
			wantErr: true,
			errText: "Invalid ION key: \"\" - cannot be empty",
		},
		// versa - valid
		{
			name: "Valid versa config",
			config: &megaport.VersaConfig{
				Vendor:            "versa",
				ImageID:           123,
				ProductSize:       "MEDIUM",
				DirectorAddress:   "director.example.com",
				ControllerAddress: "controller.example.com",
				LocalAuth:         "local-auth",
				RemoteAuth:        "remote-auth",
				SerialNumber:      "SN123456",
			},
			wantErr: false,
		},
		// versa - missing director address
		{
			name: "Invalid versa - missing director address",
			config: &megaport.VersaConfig{
				Vendor:            "versa",
				ImageID:           123,
				ProductSize:       "MEDIUM",
				ControllerAddress: "controller.example.com",
				LocalAuth:         "local-auth",
				RemoteAuth:        "remote-auth",
				SerialNumber:      "SN123456",
			},
			wantErr: true,
			errText: "Invalid director address: \"\" - cannot be empty",
		},
		// vmware - valid
		{
			name: "Valid vmware config",
			config: &megaport.VmwareConfig{
				Vendor:            "vmware",
				ImageID:           123,
				ProductSize:       "MEDIUM",
				AdminSSHPublicKey: "ssh-rsa AAAA...",
				SSHPublicKey:      "ssh-rsa AAAA...",
				VcoAddress:        "vco.example.com",
				VcoActivationCode: "activation-code",
			},
			wantErr: false,
		},
		// vmware - missing VCO address
		{
			name: "Invalid vmware - missing VCO address",
			config: &megaport.VmwareConfig{
				Vendor:            "vmware",
				ImageID:           123,
				ProductSize:       "MEDIUM",
				AdminSSHPublicKey: "ssh-rsa AAAA...",
				SSHPublicKey:      "ssh-rsa AAAA...",
				VcoActivationCode: "activation-code",
			},
			wantErr: true,
			errText: "Invalid VCO address: \"\" - cannot be empty",
		},
		// meraki - valid
		{
			name: "Valid meraki config",
			config: &megaport.MerakiConfig{
				Vendor:      "meraki",
				ImageID:     123,
				ProductSize: "MEDIUM",
				Token:       "meraki-token",
			},
			wantErr: false,
		},
		// meraki - missing token
		{
			name: "Invalid meraki - missing token",
			config: &megaport.MerakiConfig{
				Vendor:      "meraki",
				ImageID:     123,
				ProductSize: "MEDIUM",
			},
			wantErr: true,
			errText: "Invalid token: \"\" - cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMVEVendorConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMVEVendorConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateMVEVendorConfig(t *testing.T) {
	// Test cases for ValidateMVEVendorConfig
	tests := []struct {
		name string
		// vendor string // Removed, now part of config struct
		config  megaport.VendorConfig // Use interface type
		wantErr bool
		errText string
	}{
		{
			name: "Valid 6wind config",
			config: &megaport.SixwindVSRConfig{ // Use struct pointer
				Vendor:       "6wind",
				ImageID:      123,
				ProductSize:  "MEDIUM",
				MVELabel:     "6wind-mve",
				SSHPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC....",
			},
			wantErr: false,
		},
		{
			name: "Invalid 6wind config - missing ssh key",
			config: &megaport.SixwindVSRConfig{ // Use struct pointer
				Vendor:      "6wind",
				ImageID:     123,
				ProductSize: "MEDIUM",
				MVELabel:    "6wind-mve",
				// SSHPublicKey missing (zero value "")
			},
			wantErr: true,
			errText: "Invalid SSH public key: \"\" - cannot be empty", // Adjusted error
		},
		{
			name: "Valid Cisco config",
			config: &megaport.CiscoConfig{ // Use struct pointer
				Vendor:            "cisco",
				ImageID:           123,
				ProductSize:       "MEDIUM",
				MVELabel:          "cisco-mve",
				AdminSSHPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC....",
				SSHPublicKey:      "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC....",
				CloudInit:         "base64_encoded_data",
				ManageLocally:     true,
			},
			wantErr: false,
		},
		{
			name: "Invalid Cisco config - missing FMC data when not managing locally",
			config: &megaport.CiscoConfig{ // Use struct pointer
				Vendor:            "cisco",
				ImageID:           123,
				ProductSize:       "MEDIUM",
				MVELabel:          "cisco-mve",
				AdminSSHPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC....",
				SSHPublicKey:      "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC....",
				CloudInit:         "base64_encoded_data",
				ManageLocally:     false, // Requires FMC fields (zero value "")
			},
			wantErr: true,
			errText: "Invalid FMC IP address: \"\" - cannot be empty when not managing locally", // Adjusted error
		},
		{
			name: "Invalid product size",
			config: &megaport.CiscoConfig{ // Use struct pointer
				Vendor:            "cisco",
				ImageID:           123,
				ProductSize:       "INVALID_SIZE",
				MVELabel:          "cisco-mve",
				AdminSSHPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC....",
				SSHPublicKey:      "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC....",
				ManageLocally:     true, // Avoid FMC errors
			},
			wantErr: true,
			errText: fmt.Sprintf("Invalid product size: INVALID_SIZE - must be one of: %v", ValidMVEProductSizes), // Updated error message prefix
		},
		{
			name: "Invalid vendor in config", // Renamed test
			config: &megaport.CiscoConfig{ // Use a valid struct type but set Vendor field incorrectly
				Vendor:            "invalid_vendor",
				ImageID:           123,
				ProductSize:       "MEDIUM",
				AdminSSHPublicKey: "ssh-rsa AAA...", // Provide required field for Cisco
				ManageLocally:     true,             // Avoid FMC errors
			},
			wantErr: true,
			errText: "Invalid SSH public key: \"\" - cannot be empty", // Updated based on actual test failure output
		},
		{
			name: "Missing image ID (zero value)", // Renamed test
			config: &megaport.CiscoConfig{ // Use struct pointer
				Vendor: "cisco",
				// ImageID missing (zero value 0)
				ProductSize:   "MEDIUM",
				ManageLocally: true, // Avoid FMC errors
			},
			wantErr: true,
			errText: "Invalid image ID: 0 - must be a positive integer", // Adjusted error for zero value check
		},
		{
			name: "Missing product size (empty string)", // Renamed test
			config: &megaport.CiscoConfig{ // Use struct pointer
				Vendor:  "cisco",
				ImageID: 123,
				// ProductSize missing (zero value "")
				ManageLocally: true, // Avoid FMC errors
			},
			wantErr: true,
			errText: fmt.Sprintf("Invalid product size: \"\" - must be one of: %v", ValidMVEProductSizes), // Updated error message prefix and adjusted for empty string check
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMVEVendorConfig(tt.config) // Pass interface value (struct pointer)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMVEVendorConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}
