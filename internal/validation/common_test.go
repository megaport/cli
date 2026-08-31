package validation

import (
	"fmt"
	"sort"
	"testing"

	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
)

func TestValidateContractTerm(t *testing.T) {
	tests := []struct {
		name    string
		term    int
		wantErr bool
		errText string
	}{
		{"Valid term 1", 1, false, ""},
		{"Valid term 12", 12, false, ""},
		{"Valid term 24", 24, false, ""},
		{"Valid term 36", 36, false, ""},
		{"Valid term 48", 48, false, ""},
		{"Valid term 60", 60, false, ""},
		{"Invalid term 0", 0, true, fmt.Sprintf("Invalid contract term: 0 - must be one of: %v", ValidContractTerms)},
		{"Invalid term -1", -1, true, fmt.Sprintf("Invalid contract term: -1 - must be one of: %v", ValidContractTerms)},
		{"Invalid term 6", 6, true, fmt.Sprintf("Invalid contract term: 6 - must be one of: %v", ValidContractTerms)},
		{"Invalid term 72", 72, true, fmt.Sprintf("Invalid contract term: 72 - must be one of: %v", ValidContractTerms)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContractTerm(tt.term)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContractTerm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateMCRPortSpeed(t *testing.T) {
	tests := []struct {
		name    string
		speed   int
		wantErr bool
		errText string
	}{
		{"Valid speed 1000", 1000, false, ""},
		{"Valid speed 2500", 2500, false, ""},
		{"Valid speed 5000", 5000, false, ""},
		{"Valid speed 10000", 10000, false, ""},
		{"Valid speed 25000", 25000, false, ""},
		{"Valid speed 50000", 50000, false, ""},
		{"Valid speed 100000", 100000, false, ""},
		{"Invalid speed 0", 0, true, fmt.Sprintf("Invalid MCR port speed: 0 - must be one of: %v", ValidMCRPortSpeeds)},
		{"Invalid speed -1", -1, true, fmt.Sprintf("Invalid MCR port speed: -1 - must be one of: %v", ValidMCRPortSpeeds)},
		{"Invalid speed 500", 500, true, fmt.Sprintf("Invalid MCR port speed: 500 - must be one of: %v", ValidMCRPortSpeeds)},
		{"Invalid speed 150000", 150000, true, fmt.Sprintf("Invalid MCR port speed: 150000 - must be one of: %v", ValidMCRPortSpeeds)},
		{"Invalid speed 400000 (400G not orderable for MCR)", 400000, true, fmt.Sprintf("Invalid MCR port speed: 400000 - must be one of: %v", ValidMCRPortSpeeds)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMCRPortSpeed(tt.speed)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMCRPortSpeed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidatePortSpeed(t *testing.T) {
	tests := []struct {
		name    string
		speed   int
		wantErr bool
		errText string
	}{
		{"Valid speed 1000", 1000, false, ""},
		{"Valid speed 10000", 10000, false, ""},
		{"Valid speed 100000", 100000, false, ""},
		{"Invalid speed 0", 0, true, fmt.Sprintf("Invalid port speed: 0 - must be one of: %v", ValidPortSpeeds)},
		{"Invalid speed 5000", 5000, true, fmt.Sprintf("Invalid port speed: 5000 - must be one of: %v", ValidPortSpeeds)},
		{"Invalid speed -1000", -1000, true, fmt.Sprintf("Invalid port speed: -1000 - must be one of: %v", ValidPortSpeeds)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePortSpeed(tt.speed)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePortSpeed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateVLAN(t *testing.T) {
	baseErrMsg := fmt.Sprintf("must be %d, %d, or between %d-%d", AutoAssignVLAN, UntaggedVLAN, MinAssignableVLAN, MaxVLAN)
	tests := []struct {
		name    string
		vlan    int
		wantErr bool
		errText string
	}{
		{"Valid Auto Assign", AutoAssignVLAN, false, ""},
		{"Valid Untagged", UntaggedVLAN, false, ""},
		{"Valid Min Assignable", MinAssignableVLAN, false, ""},
		{"Valid Max Assignable", MaxAssignableVLAN, false, ""},
		{"Valid Max VLAN", MaxVLAN, false, ""},
		{"Valid Mid Range", 1000, false, ""},
		{"Invalid Too Low", -2, true, fmt.Sprintf("Invalid VLAN ID: %d - %s", -2, baseErrMsg)},
		{"Invalid Reserved 1", 1, true, fmt.Sprintf("Invalid VLAN ID: %d - %s", 1, baseErrMsg)},
		{"Invalid Too High", MaxVLAN + 1, true, fmt.Sprintf("Invalid VLAN ID: %d - %s", MaxVLAN+1, baseErrMsg)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVLAN(tt.vlan)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVLAN() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateRateLimit(t *testing.T) {
	tests := []struct {
		name      string
		rateLimit int
		wantErr   bool
		errText   string
	}{
		{"Valid rate limit 1", 1, false, ""},
		{"Valid rate limit 100", 100, false, ""},
		{"Valid rate limit 1000", 1000, false, ""},
		{"Invalid rate limit 0", 0, true, "Invalid rate limit: 0 - must be a positive integer"},
		{"Invalid rate limit -1", -1, true, "Invalid rate limit: -1 - must be a positive integer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRateLimit(tt.rateLimit)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRateLimit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateASN(t *testing.T) {
	maxAllowed := maxSupportedASNForInt()

	tests := []struct {
		name    string
		asn     int
		wantErr bool
		errText string
	}{
		{"Valid min ASN", int(MinASN), false, ""},
		{"Valid typical ASN", 65000, false, ""},
		{"Valid 4-byte ASN", 400000, false, ""},
		{"Valid max ASN", int(maxAllowed), false, ""},
		{"Invalid zero", 0, true, fmt.Sprintf("Invalid ASN: 0 - must be between %d and %d", MinASN, maxAllowed)},
		{"Invalid negative", -1, true, fmt.Sprintf("Invalid ASN: -1 - must be between %d and %d", MinASN, maxAllowed)},
	}

	// Only test "max+1" when the next int value is representable on this target.
	if maxAllowed < int64(^uint(0)>>1) {
		overflow := int(maxAllowed) + 1
		tests = append(tests, struct {
			name    string
			asn     int
			wantErr bool
			errText string
		}{
			"Invalid max ASN plus one",
			overflow,
			true,
			fmt.Sprintf("Invalid ASN: %d - must be between %d and %d", overflow, MinASN, maxAllowed),
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateASN(tt.asn)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateASN() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateMACAddress(t *testing.T) {
	tests := []struct {
		name    string
		mac     string
		wantErr bool
		errText string
	}{
		{"Valid colon-separated lowercase", "00:11:22:33:44:55", false, ""},
		{"Valid colon-separated uppercase", "AA:BB:CC:DD:EE:FF", false, ""},
		{"Valid colon-separated mixed case", "aA:bB:cC:dD:eE:fF", false, ""},
		{"Valid hyphen-separated", "00-11-22-33-44-55", false, ""},
		{"Invalid empty", "", true, "Invalid MAC address: \"\" - cannot be empty"},
		{"Invalid too short", "00:11:22:33:44", true, "Invalid MAC address: 00:11:22:33:44 - must be a valid MAC address (e.g. 00:11:22:33:44:55)"},
		{"Invalid too long", "00:11:22:33:44:55:66", true, "Invalid MAC address: 00:11:22:33:44:55:66 - must be a valid MAC address (e.g. 00:11:22:33:44:55)"},
		{"Invalid EUI-64", "00:11:22:33:44:55:66:77", true, "Invalid MAC address: 00:11:22:33:44:55:66:77 - must be a 6-byte (EUI-48) MAC address"},
		{"Invalid hex chars", "GG:HH:II:JJ:KK:LL", true, "Invalid MAC address: GG:HH:II:JJ:KK:LL - must be a valid MAC address (e.g. 00:11:22:33:44:55)"},
		{"Invalid format", "not-a-mac", true, "Invalid MAC address: not-a-mac - must be a valid MAC address (e.g. 00:11:22:33:44:55)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMACAddress(tt.mac)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMACAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateMVEProductSize(t *testing.T) {
	tests := []struct {
		name    string
		size    string
		wantErr bool
		errText string
	}{
		{"Valid SMALL", "SMALL", false, ""},
		{"Valid MEDIUM", "MEDIUM", false, ""},
		{"Valid LARGE", "LARGE", false, ""},
		{"Invalid lowercase", "small", true, fmt.Sprintf("Invalid product size: small - must be one of: %v", ValidMVEProductSizes)},
		{"Invalid value", "XLARGE", true, fmt.Sprintf("Invalid product size: XLARGE - must be one of: %v", ValidMVEProductSizes)},
		{"Empty value", "", true, fmt.Sprintf("Invalid product size: \"\" - must be one of: %v", ValidMVEProductSizes)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMVEProductSize(tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMVEProductSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateDateRange(t *testing.T) {
	tests := []struct {
		name      string
		startDate string
		endDate   string
		wantErr   bool
		errText   string
	}{
		{"Both empty", "", "", false, ""},
		{"Both valid end after start", "2026-01-01", "2026-06-01", false, ""},
		{"Only start provided", "2026-01-01", "", true, "Invalid date range: end-date - both --start-date and --end-date must be provided together"},
		{"Only end provided", "", "2026-06-01", true, "Invalid date range: start-date - both --start-date and --end-date must be provided together"},
		{"Invalid start format", "01-01-2026", "2026-06-01", true, "Invalid start-date: 01-01-2026 - must be in YYYY-MM-DD format"},
		{"Invalid end format", "2026-01-01", "01-06-2026", true, "Invalid end-date: 01-06-2026 - must be in YYYY-MM-DD format"},
		{"End before start", "2026-06-01", "2026-01-01", true, "Invalid date range: 2026-06-01 to 2026-01-01 - end date must be after start date"},
		{"Same date", "2026-06-01", "2026-06-01", true, "Invalid date range: 2026-06-01 to 2026-06-01 - end date must be after start date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDateRange(tt.startDate, tt.endDate)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDateRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("test field", 123, "test reason")
	expected := "Invalid test field: 123 - test reason"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %v, want %v", err.Error(), expected)
	}
	assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
}

func TestFormatIntSlice(t *testing.T) {
	tests := []struct {
		name string
		vals []int
		want string
	}{
		{"empty", nil, ""},
		{"single", []int{1}, "1"},
		{"two", []int{1, 12}, "1 or 12"},
		{"three", []int{1, 12, 24}, "1, 12, or 24"},
		{"four", []int{1, 12, 24, 36}, "1, 12, 24, or 36"},
		{"port speeds", []int{1000, 10000, 100000}, "1000, 10000, or 100000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FormatIntSlice(tt.vals))
		})
	}
}

// TestValidContractTermsMatchSDK guards against the CLI's contract terms silently
// drifting from the SDK's. If this fails after an SDK bump, get product confirmation
// that the new term is actually orderable before updating ValidContractTerms.
func TestValidContractTermsMatchSDK(t *testing.T) {
	assert.ElementsMatch(t, megaport.VALID_CONTRACT_TERMS, ValidContractTerms)
	assert.True(t, sort.IntsAreSorted(ValidContractTerms), "ValidContractTerms must stay sorted; it feeds FormatIntSlice's user-facing help text")
}

// TestValidMCRPortSpeedsSubsetOfSDK ensures the CLI never accepts an MCR speed the
// SDK rejects, while allowing the CLI to stay deliberately narrower (400G MCR is not
// orderable today, see ValidMCRPortSpeeds).
func TestValidMCRPortSpeedsSubsetOfSDK(t *testing.T) {
	for _, speed := range ValidMCRPortSpeeds {
		assert.Contains(t, megaport.VALID_MCR_PORT_SPEEDS, speed)
	}
	assert.NotContains(t, ValidMCRPortSpeeds, 400000, "400G MCR is not orderable yet; only physical ports support 400G")
}

func TestVLANHelpText(t *testing.T) {
	assert.Equal(t, "0=auto-assign, -1=untagged, 2-4094 for specific VLAN (1 is reserved)", VLANHelpText())
}

func TestInnerVLANHelpText(t *testing.T) {
	assert.Equal(t, "0=none, -1=untagged, 2-4094 for specific VLAN (1 is reserved)", InnerVLANHelpText())
}
