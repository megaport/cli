package validation

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateIntRange(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		min     int
		max     int
		wantErr bool
	}{
		{"at min", 0, 0, 10, false},
		{"at max", 10, 0, 10, false},
		{"mid range", 5, 0, 10, false},
		{"just below min", -1, 0, 10, true},
		{"just above max", 11, 0, 10, true},
		{"negative range valid", -5, -10, -1, false},
		{"single value range hit", 7, 7, 7, false},
		{"single value range miss", 8, 7, 7, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIntRange(tt.value, tt.min, tt.max, "field")
			if tt.wantErr {
				assert.Error(t, err)
				assert.IsType(t, &ValidationError{}, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateStringOneOf(t *testing.T) {
	valid := []string{"red", "green", "blue"}
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"first match", "red", false},
		{"last match", "blue", false},
		{"empty rejected", "", true},
		{"not in list", "purple", true},
		{"case sensitive miss", "RED", true},
		{"whitespace miss", " red", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStringOneOf(tt.value, valid, "color")
			if tt.wantErr {
				assert.Error(t, err)
				assert.IsType(t, &ValidationError{}, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateIPv4(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{"standard address", "192.168.1.1", false},
		{"min address", "0.0.0.0", false},
		{"max address", "255.255.255.255", false},
		{"empty rejected", "", true},
		{"octet over 255", "256.1.1.1", true},
		{"too many octets", "1.2.3.4.5", true},
		{"too few octets", "1.2.3", true},
		{"trailing dot", "1.2.3.4.", true},
		{"with CIDR suffix", "192.168.1.1/24", true},
		{"hostname", "example.com", true},
		// IPv6 forms must be rejected by an IPv4 check (regression: ESD-1386).
		{"ipv6 loopback rejected", "::1", true},
		{"ipv6 full rejected", "2001:db8::1", true},
		{"ipv4-mapped ipv6 rejected", "::ffff:192.168.0.1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIPv4(tt.ip, "ip")
			if tt.wantErr {
				assert.Error(t, err)
				assert.IsType(t, &ValidationError{}, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateIPAddress(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{"standard ipv4 address", "192.168.1.1", false},
		{"ipv6 loopback", "::1", false},
		{"ipv6 full", "2001:db8::1", false},
		{"ipv4-mapped ipv6", "::ffff:192.168.0.1", false},
		{"empty rejected", "", true},
		{"octet over 255", "256.1.1.1", true},
		{"hostname rejected", "example.com", true},
		{"with CIDR suffix rejected", "192.168.1.1/24", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIPAddress(tt.ip, "ip")
			if tt.wantErr {
				assert.Error(t, err)
				assert.IsType(t, &ValidationError{}, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCIDR(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		wantErr bool
	}{
		{"network address", "10.0.0.0/24", false},
		{"host bits set", "10.0.0.1/24", false},
		{"min prefix", "0.0.0.0/0", false},
		{"max prefix", "255.255.255.255/32", false},
		{"single host /32", "192.168.1.1/32", false},
		{"empty rejected", "", true},
		{"bare ip no prefix", "10.0.0.0", true},
		{"prefix too large", "10.0.0.0/33", true},
		{"negative prefix", "10.0.0.0/-1", true},
		{"non-numeric prefix", "10.0.0.0/ab", true},
		{"malformed", "not-a-cidr", true},
		{"octet over 255", "256.0.0.0/24", true},
		// IPv6 CIDR must be rejected by an IPv4 CIDR check (regression: ESD-1386).
		{"ipv6 cidr rejected", "2001:db8::/32", true},
		{"ipv4-mapped ipv6 cidr rejected", "::ffff:1.2.3.0/120", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCIDR(tt.cidr, "cidr")
			if tt.wantErr {
				assert.Error(t, err)
				assert.IsType(t, &ValidationError{}, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"validation error", NewValidationError("f", 1, "bad"), true},
		{"plain error", errors.New("boom"), false},
		{"nil error", nil, false},
		{"plain error containing validation message string", errors.New("wrap: " + NewValidationError("f", 1, "bad").Error()), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsValidationError(tt.err))
		})
	}
}
