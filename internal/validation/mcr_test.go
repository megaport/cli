package validation

import (
	"fmt"
	"testing"

	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
)

func TestValidatePrefixFilterEntriesRejectsUnknownAddressFamily(t *testing.T) {
	entries := []*megaport.MCRPrefixListEntry{
		{Action: "permit", Prefix: "10.0.0.0/8"},
	}
	err := validatePrefixFilterEntries(entries, "IPv5")
	assert.IsType(t, &ValidationError{}, err)
	assert.Equal(t, "Invalid address family: IPv5 - must be IPv4 or IPv6", err.Error())
}

func TestValidateIPSecTunnelCount(t *testing.T) {
	tests := []struct {
		name             string
		count            int
		allowZeroDisable bool
		wantErr          bool
		errContains      string
	}{
		{"valid 10 add mode", 10, false, false, ""},
		{"valid 20 add mode", 20, false, false, ""},
		{"valid 30 add mode", 30, false, false, ""},
		{"valid 10 update mode", 10, true, false, ""},
		{"valid 20 update mode", 20, true, false, ""},
		{"valid 30 update mode", 30, true, false, ""},
		{"zero disable update mode", 0, true, false, ""},
		{"zero add mode is invalid", 0, false, true, "0 uses the API default of 10"},
		{"invalid count add mode", 5, false, true, "must be 10, 20, 30"},
		{"invalid count update mode", 5, true, true, "must be 10, 20, 30, or 0 to disable"},
		{"negative count", -1, false, true, "must be 10, 20, 30"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIPSecTunnelCount(tt.count, tt.allowZeroDisable)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateMCRRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *megaport.BuyMCRRequest
		wantErr bool
		errText string // Add expected error text for specific cases
	}{
		{
			name: "Valid MCR request",
			req: &megaport.BuyMCRRequest{
				Name:       "Test MCR",
				Term:       12,
				PortSpeed:  5000,
				LocationID: 100,
			},
			wantErr: false,
		},
		{
			name: "Empty MCR name",
			req: &megaport.BuyMCRRequest{
				Name:       "",
				Term:       12,
				PortSpeed:  5000,
				LocationID: 100,
			},
			wantErr: true,
			errText: "Invalid MCR name:  - cannot be empty", // Use ValidationError format
		},
		{
			name: "Invalid term",
			req: &megaport.BuyMCRRequest{
				Name:       "Test MCR",
				Term:       5,
				PortSpeed:  5000,
				LocationID: 100,
			},
			wantErr: true,
			errText: fmt.Sprintf("Invalid contract term: 5 - must be one of: %v", ValidContractTerms), // Expect ValidationError message
		},
		{
			name: "Invalid port speed",
			req: &megaport.BuyMCRRequest{
				Name:       "Test MCR",
				Term:       12,
				PortSpeed:  3000,
				LocationID: 100,
			},
			wantErr: true,
			errText: fmt.Sprintf("Invalid MCR port speed: 3000 - must be one of: %v", ValidMCRPortSpeeds), // Use ValidationError format
		},
		{
			name: "Invalid location ID",
			req: &megaport.BuyMCRRequest{
				Name:       "Test MCR",
				Term:       12,
				PortSpeed:  5000,
				LocationID: 0,
			},
			wantErr: true,
			errText: "Invalid location ID: 0 - must be a positive integer", // Use ValidationError format
		},
		{
			name: "Negative location ID",
			req: &megaport.BuyMCRRequest{
				Name:       "Test MCR",
				Term:       12,
				PortSpeed:  5000,
				LocationID: -1,
			},
			wantErr: true,
			errText: "Invalid location ID: -1 - must be a positive integer", // Use ValidationError format
		},
		{
			name: "Zero ASN is allowed (API auto-assigns)",
			req: &megaport.BuyMCRRequest{
				Name:       "Test MCR",
				Term:       12,
				PortSpeed:  5000,
				LocationID: 100,
				MCRAsn:     0,
			},
			wantErr: false,
		},
		{
			name: "Valid explicit ASN",
			req: &megaport.BuyMCRRequest{
				Name:       "Test MCR",
				Term:       12,
				PortSpeed:  5000,
				LocationID: 100,
				MCRAsn:     65000,
			},
			wantErr: false,
		},
		{
			name: "Negative ASN",
			req: &megaport.BuyMCRRequest{
				Name:       "Test MCR",
				Term:       12,
				PortSpeed:  5000,
				LocationID: 100,
				MCRAsn:     -1,
			},
			wantErr: true,
			errText: fmt.Sprintf("Invalid MCR ASN: -1 - must be between %d and %d", MinASN, MaxASN),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMCRRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMCRRequest() error = %v, wantErr %v", err, tt.wantErr)
				return // Avoid panic on nil error
			}
			// Check if the error message matches the expected text when an error is expected
			if err != nil && tt.wantErr {
				// Check if the error type is *ValidationError
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				// Check the error message
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateMCRASN(t *testing.T) {
	tests := []struct {
		name    string
		asn     int64
		wantErr bool
	}{
		{name: "minimum valid", asn: MinASN, wantErr: false},
		{name: "private ASN", asn: 65000, wantErr: false},
		{name: "maximum valid", asn: MaxASN, wantErr: false},
		{name: "zero rejected", asn: 0, wantErr: true},
		{name: "negative rejected", asn: -1, wantErr: true},
		{name: "above 32-bit max rejected", asn: MaxASN + 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMCRASN(tt.asn)
			if tt.wantErr {
				assert.Error(t, err)
				assert.IsType(t, &ValidationError{}, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePrefixFilterListRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *megaport.CreateMCRPrefixFilterListRequest
		wantErr bool
		errText string
	}{
		{
			name: "Valid prefix filter list request",
			req: &megaport.CreateMCRPrefixFilterListRequest{
				MCRID: "mcr-uid-123",
				PrefixFilterList: megaport.MCRPrefixFilterList{
					Description:   "Test filter list",
					AddressFamily: "IPv4",
					Entries: []*megaport.MCRPrefixListEntry{
						{Action: "permit", Prefix: "10.0.0.0/8"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Empty entry prefix",
			req: &megaport.CreateMCRPrefixFilterListRequest{
				MCRID: "mcr-uid-123",
				PrefixFilterList: megaport.MCRPrefixFilterList{
					Description:   "Test filter list",
					AddressFamily: "IPv4",
					Entries: []*megaport.MCRPrefixListEntry{
						{Action: "permit", Prefix: ""},
					},
				},
			},
			wantErr: true,
			errText: "Invalid entry prefix index 0:  - prefix cannot be empty",
		},
		{
			name: "Missing description",
			req: &megaport.CreateMCRPrefixFilterListRequest{
				MCRID: "mcr-uid-123",
				PrefixFilterList: megaport.MCRPrefixFilterList{
					Description:   "",
					AddressFamily: "IPv4",
					Entries: []*megaport.MCRPrefixListEntry{
						{Action: "permit", Prefix: "10.0.0.0/8"},
					},
				},
			},
			wantErr: true,
			errText: "Invalid description:  - cannot be empty",
		},
		{
			name: "Invalid address family",
			req: &megaport.CreateMCRPrefixFilterListRequest{
				MCRID: "mcr-uid-123",
				PrefixFilterList: megaport.MCRPrefixFilterList{
					Description:   "Test filter list",
					AddressFamily: "IPv5",
					Entries: []*megaport.MCRPrefixListEntry{
						{Action: "permit", Prefix: "10.0.0.0/8"},
					},
				},
			},
			wantErr: true,
			errText: "Invalid address family: IPv5 - must be IPv4 or IPv6",
		},
		{
			name: "Empty address family",
			req: &megaport.CreateMCRPrefixFilterListRequest{
				MCRID: "mcr-uid-123",
				PrefixFilterList: megaport.MCRPrefixFilterList{
					Description:   "Test filter list",
					AddressFamily: "",
					Entries: []*megaport.MCRPrefixListEntry{
						{Action: "permit", Prefix: "10.0.0.0/8"},
					},
				},
			},
			wantErr: true,
			errText: "Invalid address family:  - cannot be empty",
		},
		{
			name: "Empty entries",
			req: &megaport.CreateMCRPrefixFilterListRequest{
				MCRID: "mcr-uid-123",
				PrefixFilterList: megaport.MCRPrefixFilterList{
					Description:   "Test filter list",
					AddressFamily: "IPv4",
					Entries:       []*megaport.MCRPrefixListEntry{},
				},
			},
			wantErr: true,
			errText: "Invalid entries: [] - must contain at least one entry",
		},
		{
			name: "Nil request entries",
			req: &megaport.CreateMCRPrefixFilterListRequest{
				MCRID: "mcr-uid-123",
				PrefixFilterList: megaport.MCRPrefixFilterList{
					Description:   "Test filter list",
					AddressFamily: "IPv4",
					Entries:       nil,
				},
			},
			wantErr: true,
			errText: "Invalid entries: [] - must contain at least one entry",
		},
		{
			name: "Nil entry in entries",
			req: &megaport.CreateMCRPrefixFilterListRequest{
				MCRID: "mcr-uid-123",
				PrefixFilterList: megaport.MCRPrefixFilterList{
					Description:   "Test filter list",
					AddressFamily: "IPv4",
					Entries: []*megaport.MCRPrefixListEntry{
						{Action: "permit", Prefix: "10.0.0.0/8"},
						nil,
					},
				},
			},
			wantErr: true,
			errText: "Invalid entry index 1: <nil> - entry cannot be nil",
		},
		{
			name: "Valid GE/LE bounds",
			req: &megaport.CreateMCRPrefixFilterListRequest{
				MCRID: "mcr-uid-123",
				PrefixFilterList: megaport.MCRPrefixFilterList{
					Description:   "Test filter list",
					AddressFamily: "IPv4",
					Entries: []*megaport.MCRPrefixListEntry{
						{Action: "permit", Prefix: "10.0.0.0/8", Ge: 16, Le: 24},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "GE above IPv4 maximum",
			req: &megaport.CreateMCRPrefixFilterListRequest{
				MCRID: "mcr-uid-123",
				PrefixFilterList: megaport.MCRPrefixFilterList{
					Description:   "Test filter list",
					AddressFamily: "IPv4",
					Entries: []*megaport.MCRPrefixListEntry{
						{Action: "permit", Prefix: "10.0.0.0/8", Ge: 33},
					},
				},
			},
			wantErr: true,
			errText: "Invalid entry GE index 0: 33 - must be between 0 and 32 for IPv4",
		},
		{
			name: "Negative GE",
			req: &megaport.CreateMCRPrefixFilterListRequest{
				MCRID: "mcr-uid-123",
				PrefixFilterList: megaport.MCRPrefixFilterList{
					Description:   "Test filter list",
					AddressFamily: "IPv4",
					Entries: []*megaport.MCRPrefixListEntry{
						{Action: "permit", Prefix: "10.0.0.0/8", Ge: -1},
					},
				},
			},
			wantErr: true,
			errText: "Invalid entry GE index 0: -1 - must be between 0 and 32 for IPv4",
		},
		{
			name: "LE above IPv4 maximum",
			req: &megaport.CreateMCRPrefixFilterListRequest{
				MCRID: "mcr-uid-123",
				PrefixFilterList: megaport.MCRPrefixFilterList{
					Description:   "Test filter list",
					AddressFamily: "IPv4",
					Entries: []*megaport.MCRPrefixListEntry{
						{Action: "permit", Prefix: "10.0.0.0/8", Le: 33},
					},
				},
			},
			wantErr: true,
			errText: "Invalid entry LE index 0: 33 - must be between 0 and 32 for IPv4",
		},
		{
			name: "GE greater than LE",
			req: &megaport.CreateMCRPrefixFilterListRequest{
				MCRID: "mcr-uid-123",
				PrefixFilterList: megaport.MCRPrefixFilterList{
					Description:   "Test filter list",
					AddressFamily: "IPv4",
					Entries: []*megaport.MCRPrefixListEntry{
						{Action: "permit", Prefix: "10.0.0.0/8", Ge: 28, Le: 24},
					},
				},
			},
			wantErr: true,
			errText: "Invalid entry GE index 0: 28 - must not exceed the LE value",
		},
		{
			name: "IPv6 GE/LE beyond IPv4 range accepted",
			req: &megaport.CreateMCRPrefixFilterListRequest{
				MCRID: "mcr-uid-123",
				PrefixFilterList: megaport.MCRPrefixFilterList{
					Description:   "Test filter list",
					AddressFamily: "IPv6",
					Entries: []*megaport.MCRPrefixListEntry{
						{Action: "permit", Prefix: "2001:db8::/32", Ge: 48, Le: 64},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "IPv6 LE above maximum",
			req: &megaport.CreateMCRPrefixFilterListRequest{
				MCRID: "mcr-uid-123",
				PrefixFilterList: megaport.MCRPrefixFilterList{
					Description:   "Test filter list",
					AddressFamily: "IPv6",
					Entries: []*megaport.MCRPrefixListEntry{
						{Action: "permit", Prefix: "2001:db8::/32", Le: 129},
					},
				},
			},
			wantErr: true,
			errText: "Invalid entry LE index 0: 129 - must be between 0 and 128 for IPv6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePrefixFilterListRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePrefixFilterListRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateUpdatePrefixFilterList(t *testing.T) {
	tests := []struct {
		name    string
		req     *megaport.MCRPrefixFilterList
		wantErr bool
		errText string
	}{
		{
			name: "Valid update with entries",
			req: &megaport.MCRPrefixFilterList{
				Description:   "Updated filter list",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					{Action: "permit", Prefix: "10.0.0.0/8"},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid update with no entries",
			req: &megaport.MCRPrefixFilterList{
				Description:   "Updated filter list",
				AddressFamily: "IPv4",
				Entries:       []*megaport.MCRPrefixListEntry{},
			},
			wantErr: false,
		},
		{
			name: "Invalid entry action",
			req: &megaport.MCRPrefixFilterList{
				Description:   "Updated filter list",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					{Action: "allow", Prefix: "10.0.0.0/8"},
				},
			},
			wantErr: true,
			errText: "Invalid entry action: allow - must be permit or deny",
		},
		{
			name: "Invalid prefix CIDR",
			req: &megaport.MCRPrefixFilterList{
				Description:   "Updated filter list",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					{Action: "permit", Prefix: "not-a-cidr"},
				},
			},
			wantErr: true,
			errText: "Invalid entry prefix index 0: not-a-cidr - must be a valid IPv4 CIDR notation",
		},
		{
			name: "Prefix does not match declared IPv6 address family",
			req: &megaport.MCRPrefixFilterList{
				Description:   "Updated filter list",
				AddressFamily: "IPv6",
				Entries: []*megaport.MCRPrefixListEntry{
					{Action: "permit", Prefix: "10.0.0.0/8"},
				},
			},
			wantErr: true,
			errText: "Invalid entry prefix index 0: 10.0.0.0/8 - must be a valid IPv6 CIDR notation",
		},
		{
			name: "Valid IPv6 update with entries",
			req: &megaport.MCRPrefixFilterList{
				Description:   "Updated filter list",
				AddressFamily: "IPv6",
				Entries: []*megaport.MCRPrefixListEntry{
					{Action: "permit", Prefix: "2001:db8::/32"},
				},
			},
			wantErr: false,
		},
		{
			name: "GE greater than LE rejected on update",
			req: &megaport.MCRPrefixFilterList{
				Description:   "Updated filter list",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					{Action: "permit", Prefix: "10.0.0.0/8", Ge: 28, Le: 24},
				},
			},
			wantErr: true,
			errText: "Invalid entry GE index 0: 28 - must not exceed the LE value",
		},
		{
			name: "Empty address family with entries rejected",
			req: &megaport.MCRPrefixFilterList{
				Description:   "Updated filter list",
				AddressFamily: "",
				Entries: []*megaport.MCRPrefixListEntry{
					{Action: "permit", Prefix: "10.0.0.0/8"},
				},
			},
			wantErr: true,
			errText: "Invalid address family:  - cannot be empty",
		},
		{
			name: "Invalid address family with entries rejected",
			req: &megaport.MCRPrefixFilterList{
				Description:   "Updated filter list",
				AddressFamily: "IPv5",
				Entries: []*megaport.MCRPrefixListEntry{
					{Action: "permit", Prefix: "10.0.0.0/8"},
				},
			},
			wantErr: true,
			errText: "Invalid address family: IPv5 - must be IPv4 or IPv6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdatePrefixFilterList(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdatePrefixFilterList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}
