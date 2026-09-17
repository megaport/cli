package validation

import (
	"fmt"
	"math"
	"strings"
	"testing"

	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
)

func TestValidateVNICIndex(t *testing.T) {
	tests := []struct {
		name    string
		index   int
		wantErr bool
		errText string
	}{
		{"Valid index 0", 0, false, ""},
		{"Valid index 5", 5, false, ""},
		{"Valid index 100", 100, false, ""},
		{"Invalid negative index", -1, true, "Invalid vNIC index: -1 - must be non-negative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVNICIndex(tt.index)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVNICIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateBGPConnectionConfig(t *testing.T) {
	tests := []struct {
		name    string
		conn    megaport.BgpConnectionConfig
		wantErr bool
		errText string
	}{
		{
			name: "Valid BGP config",
			conn: megaport.BgpConnectionConfig{
				PeerAsn:        65000,
				LocalIpAddress: "192.168.1.1",
				PeerIpAddress:  "192.168.1.2",
			},
			wantErr: false,
		},
		{
			name: "Missing peer ASN",
			conn: megaport.BgpConnectionConfig{
				PeerAsn:        0,
				LocalIpAddress: "192.168.1.1",
				PeerIpAddress:  "192.168.1.2",
			},
			wantErr: true,
			errText: "Invalid vRouter interface [0] BGP connection [0] peer ASN: <nil> - is required",
		},
		// ASN boundary cases. Valid ASNs are 1-4294967295; the min, zero, and
		// negative are covered here. The 32-bit-max boundaries use values that
		// overflow int on 32-bit targets, so they live in a separate test.
		{
			name: "ASN 1 (min non-zero)",
			conn: megaport.BgpConnectionConfig{
				PeerAsn:        1,
				LocalIpAddress: "192.168.1.1",
				PeerIpAddress:  "192.168.1.2",
			},
			wantErr: false,
		},
		{
			name: "ASN 65535 (16-bit max)",
			conn: megaport.BgpConnectionConfig{
				PeerAsn:        65535,
				LocalIpAddress: "192.168.1.1",
				PeerIpAddress:  "192.168.1.2",
			},
			wantErr: false,
		},
		{
			name: "ASN 64512 (private range low)",
			conn: megaport.BgpConnectionConfig{
				PeerAsn:        64512,
				LocalIpAddress: "192.168.1.1",
				PeerIpAddress:  "192.168.1.2",
			},
			wantErr: false,
		},
		{
			name: "ASN 65534 (private range high)",
			conn: megaport.BgpConnectionConfig{
				PeerAsn:        65534,
				LocalIpAddress: "192.168.1.1",
				PeerIpAddress:  "192.168.1.2",
			},
			wantErr: false,
		},
		{
			name: "ASN negative rejected",
			conn: megaport.BgpConnectionConfig{
				PeerAsn:        -1,
				LocalIpAddress: "192.168.1.1",
				PeerIpAddress:  "192.168.1.2",
			},
			wantErr: true,
			errText: "Invalid vRouter interface [0] BGP connection [0] peer ASN: -1 - must be between 1-4294967295",
		},
		{
			name: "Invalid peer IP",
			conn: megaport.BgpConnectionConfig{
				PeerAsn:        65000,
				LocalIpAddress: "192.168.1.1",
				PeerIpAddress:  "not-an-ip",
			},
			wantErr: true,
			errText: "Invalid vRouter interface [0] BGP connection [0] peer IP address: not-an-ip - must be a valid IPv4 address",
		},
		{
			name: "Invalid local IP",
			conn: megaport.BgpConnectionConfig{
				PeerAsn:        65000,
				LocalIpAddress: "bad-ip",
				PeerIpAddress:  "192.168.1.2",
			},
			wantErr: true,
			errText: "Invalid vRouter interface [0] BGP connection [0] local IP address: bad-ip - must be a valid IPv4 address",
		},
		{
			name: "Invalid export policy",
			conn: megaport.BgpConnectionConfig{
				PeerAsn:        65000,
				LocalIpAddress: "192.168.1.1",
				PeerIpAddress:  "192.168.1.2",
				ExportPolicy:   "invalid",
			},
			wantErr: true,
			errText: "Invalid vRouter interface [0] BGP connection [0] export policy: invalid - must be 'permit' or 'deny'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBGPConnectionConfig(tt.conn, 0, 0)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBGPConnectionConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

// The 32-bit-max ASN boundaries (max valid and max+1) use values that overflow
// int on 32-bit targets like js/wasm, so they can't live as constants in the
// main table. Build them at runtime and skip where int is too narrow.
func TestValidateBGPConnectionConfig_HighASN(t *testing.T) {
	if int64(math.MaxInt) < int64(math.MaxUint32) {
		t.Skip("int is narrower than the 32-bit ASN range on this platform")
	}
	// Convert from a variable so these are runtime conversions, not constant
	// conversions that would overflow int at compile time on 32-bit targets.
	max64 := MaxASN
	maxValid := int(max64)     // 4294967295
	aboveMax := int(max64) + 1 // 4294967296
	tests := []struct {
		name    string
		peerAsn int
		wantErr bool
		errText string
	}{
		{"ASN 4294967295 (32-bit max)", maxValid, false, ""},
		{"ASN above 32-bit max rejected", aboveMax, true,
			fmt.Sprintf("Invalid vRouter interface [0] BGP connection [0] peer ASN: %d - must be between 1-4294967295", aboveMax)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := megaport.BgpConnectionConfig{
				PeerAsn:        tt.peerAsn,
				LocalIpAddress: "192.168.1.1",
				PeerIpAddress:  "192.168.1.2",
			}
			err := ValidateBGPConnectionConfig(conn, 0, 0)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBGPConnectionConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err)
				assert.Equal(t, tt.errText, err.Error())
			}
		})
	}
}

func TestValidateIPRouteConfig(t *testing.T) {
	tests := []struct {
		name    string
		route   megaport.IpRoute
		wantErr bool
		errText string
	}{
		{
			name: "Valid IP route",
			route: megaport.IpRoute{
				Prefix:  "10.0.0.0/24",
				NextHop: "192.168.1.1",
			},
			wantErr: false,
		},
		{
			name: "Invalid prefix - not CIDR",
			route: megaport.IpRoute{
				Prefix:  "10.0.0.0",
				NextHop: "192.168.1.1",
			},
			wantErr: true,
			errText: "Invalid vRouter interface [0] IP route [0] prefix: 10.0.0.0 - must be a valid IPv4 CIDR notation",
		},
		{
			name: "Invalid next hop",
			route: megaport.IpRoute{
				Prefix:  "10.0.0.0/24",
				NextHop: "not-an-ip",
			},
			wantErr: true,
			errText: "Invalid vRouter interface [0] IP route [0] next hop: not-an-ip - must be a valid IPv4 address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIPRouteConfig(tt.route, 0, 0)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIPRouteConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateBFDConfig(t *testing.T) {
	tests := []struct {
		name    string
		bfd     megaport.BfdConfig
		wantErr bool
		errText string
	}{
		{
			name: "Valid BFD config",
			bfd: megaport.BfdConfig{
				TxInterval: 500,
				RxInterval: 500,
				Multiplier: 5,
			},
			wantErr: false,
		},
		{
			name: "TX interval too low",
			bfd: megaport.BfdConfig{
				TxInterval: 100,
			},
			wantErr: true,
			errText: fmt.Sprintf("Invalid vRouter interface [0] BFD TX interval: 100 - must be between %d-%d milliseconds", MinBFDInterval, MaxBFDInterval),
		},
		{
			name: "RX interval too high",
			bfd: megaport.BfdConfig{
				RxInterval: 50000,
			},
			wantErr: true,
			errText: fmt.Sprintf("Invalid vRouter interface [0] BFD RX interval: 50000 - must be between %d-%d milliseconds", MinBFDInterval, MaxBFDInterval),
		},
		{
			name: "Multiplier out of range",
			bfd: megaport.BfdConfig{
				Multiplier: 1,
			},
			wantErr: true,
			errText: fmt.Sprintf("Invalid vRouter interface [0] BFD multiplier: 1 - must be between %d-%d", MinBFDMultiplier, MaxBFDMultiplier),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBFDConfig(tt.bfd, 0)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBFDConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func ptrInt(v int) *int { return &v }

func TestValidateIPsecTunnelConfig(t *testing.T) {
	tests := []struct {
		name    string
		tunnel  megaport.IPsecTunnelConfig
		wantErr bool
		errText string
	}{
		{
			name: "Valid minimal tunnel",
			tunnel: megaport.IPsecTunnelConfig{
				SourceIpAddress:      "192.0.2.1",
				DestinationIpAddress: "198.51.100.1",
				PreSharedKey:         "psk",
			},
			wantErr: false,
		},
		{
			name: "Valid tunnel with lifetimes",
			tunnel: megaport.IPsecTunnelConfig{
				SourceIpAddress:      "192.0.2.1",
				DestinationIpAddress: "198.51.100.1",
				PreSharedKey:         "psk",
				Phase1Lifetime:       ptrInt(28800),
				Phase2Lifetime:       ptrInt(3600),
			},
			wantErr: false,
		},
		{
			name: "Missing source IP",
			tunnel: megaport.IPsecTunnelConfig{
				DestinationIpAddress: "198.51.100.1",
				PreSharedKey:         "psk",
			},
			wantErr: true,
			errText: "Invalid vRouter interface [0] IPsec tunnel source IP address:  - cannot be empty",
		},
		{
			name: "Source IP not IPv4",
			tunnel: megaport.IPsecTunnelConfig{
				SourceIpAddress:      "not-an-ip",
				DestinationIpAddress: "198.51.100.1",
				PreSharedKey:         "psk",
			},
			wantErr: true,
			errText: "Invalid vRouter interface [0] IPsec tunnel source IP address: not-an-ip - must be a valid IPv4 address",
		},
		{
			name: "Missing destination IP",
			tunnel: megaport.IPsecTunnelConfig{
				SourceIpAddress: "192.0.2.1",
				PreSharedKey:    "psk",
			},
			wantErr: true,
			errText: "Invalid vRouter interface [0] IPsec tunnel destination IP address:  - cannot be empty",
		},
		{
			name: "Missing pre-shared key",
			tunnel: megaport.IPsecTunnelConfig{
				SourceIpAddress:      "192.0.2.1",
				DestinationIpAddress: "198.51.100.1",
			},
			wantErr: true,
			errText: "Invalid vRouter interface [0] IPsec tunnel pre-shared key: <nil> - cannot be empty",
		},
		{
			name: "Phase 1 lifetime too low",
			tunnel: megaport.IPsecTunnelConfig{
				SourceIpAddress:      "192.0.2.1",
				DestinationIpAddress: "198.51.100.1",
				PreSharedKey:         "psk",
				Phase1Lifetime:       ptrInt(60),
			},
			wantErr: true,
			errText: fmt.Sprintf("Invalid vRouter interface [0] IPsec tunnel phase 1 lifetime: 60 - must be between %d-%d seconds", MinIPsecPhase1Lifetime, MaxIPsecPhase1Lifetime),
		},
		{
			name: "Phase 2 lifetime too high",
			tunnel: megaport.IPsecTunnelConfig{
				SourceIpAddress:      "192.0.2.1",
				DestinationIpAddress: "198.51.100.1",
				PreSharedKey:         "psk",
				Phase2Lifetime:       ptrInt(100000),
			},
			wantErr: true,
			errText: fmt.Sprintf("Invalid vRouter interface [0] IPsec tunnel phase 2 lifetime: 100000 - must be between %d-%d seconds", MinIPsecPhase2Lifetime, MaxIPsecPhase2Lifetime),
		},
		{
			name: "Phase 2 not less than phase 1",
			tunnel: megaport.IPsecTunnelConfig{
				SourceIpAddress:      "192.0.2.1",
				DestinationIpAddress: "198.51.100.1",
				PreSharedKey:         "psk",
				Phase1Lifetime:       ptrInt(3600),
				Phase2Lifetime:       ptrInt(3600),
			},
			wantErr: true,
			errText: "Invalid vRouter interface [0] IPsec tunnel phase 2 lifetime: 3600 - must be less than phase 1 lifetime",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIPsecTunnelConfig(tt.tunnel, 0)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIPsecTunnelConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

// TestValidateIPsecTunnelConfig_PSKNotEchoed guards the rule that the
// pre-shared key value never appears in a validation error, even when the
// tunnel that fails validation has one set.
func TestValidateIPsecTunnelConfig_PSKNotEchoed(t *testing.T) {
	const psk = "super-secret-psk-value"
	tunnel := megaport.IPsecTunnelConfig{
		SourceIpAddress:      "192.0.2.1",
		DestinationIpAddress: "198.51.100.1",
		PreSharedKey:         psk,
		Phase1Lifetime:       ptrInt(3600),
		Phase2Lifetime:       ptrInt(3600), // not < phase1, so this fails validation
	}
	err := ValidateIPsecTunnelConfig(tunnel, 0)
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), psk, "PSK must never appear in a validation error")
}

func TestValidateVrouterPartnerConfig_IPsec(t *testing.T) {
	tests := []struct {
		name    string
		config  *megaport.VXCOrderVrouterPartnerConfig
		wantErr bool
		errText string
	}{
		{
			name: "tunnel without ipSecTunnel interface type",
			config: &megaport.VXCOrderVrouterPartnerConfig{
				Interfaces: []megaport.PartnerConfigInterface{
					{
						IpSecTunnelOptions: &megaport.IPsecTunnelConfig{
							SourceIpAddress: "192.0.2.1", DestinationIpAddress: "198.51.100.1", PreSharedKey: "psk",
						},
					},
				},
			},
			wantErr: true,
			errText: "requires interface type 'ipSecTunnel'",
		},
		{
			name: "unknown interface type",
			config: &megaport.VXCOrderVrouterPartnerConfig{
				Interfaces: []megaport.PartnerConfigInterface{
					{InterfaceType: "bogus"},
				},
			},
			wantErr: true,
			errText: "must be 'subInterface' or 'ipSecTunnel'",
		},
		{
			name: "valid ipSecTunnel interface with tunnel",
			config: &megaport.VXCOrderVrouterPartnerConfig{
				Interfaces: []megaport.PartnerConfigInterface{
					{
						InterfaceType: "ipSecTunnel",
						IpSecTunnelOptions: &megaport.IPsecTunnelConfig{
							SourceIpAddress: "192.0.2.1", DestinationIpAddress: "198.51.100.1", PreSharedKey: "psk",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple ipSecTunnel interfaces each with one tunnel",
			config: &megaport.VXCOrderVrouterPartnerConfig{
				Interfaces: []megaport.PartnerConfigInterface{
					{
						InterfaceType: "ipSecTunnel",
						IpSecTunnelOptions: &megaport.IPsecTunnelConfig{
							SourceIpAddress: "192.0.2.1", DestinationIpAddress: "198.51.100.1", PreSharedKey: "psk1",
						},
					},
					{
						InterfaceType: "ipSecTunnel",
						IpSecTunnelOptions: &megaport.IPsecTunnelConfig{
							SourceIpAddress: "192.0.2.2", DestinationIpAddress: "198.51.100.2", PreSharedKey: "psk2",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid tunnel surfaced through interface validation",
			config: &megaport.VXCOrderVrouterPartnerConfig{
				Interfaces: []megaport.PartnerConfigInterface{
					{
						InterfaceType: "ipSecTunnel",
						IpSecTunnelOptions: &megaport.IPsecTunnelConfig{
							DestinationIpAddress: "198.51.100.1", PreSharedKey: "psk", // missing source IP
						},
					},
				},
			},
			wantErr: true,
			errText: "source IP address",
		},
		{
			name: "ipSecTunnel interface with no tunnel",
			config: &megaport.VXCOrderVrouterPartnerConfig{
				Interfaces: []megaport.PartnerConfigInterface{
					{InterfaceType: "ipSecTunnel"},
				},
			},
			wantErr: true,
			errText: "a tunnel is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVrouterPartnerConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVrouterPartnerConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.Contains(t, err.Error(), tt.errText)
			}
		})
	}
}

func TestValidateVrouterPartnerConfig_VLAN(t *testing.T) {
	tests := []struct {
		name    string
		vlan    int
		wantErr bool
	}{
		{"auto-assign", AutoAssignVLAN, false},
		{"untagged", UntaggedVLAN, false},
		{"min assignable", MinAssignableVLAN, false},
		{"max assignable", MaxVLAN, false},
		{"reserved", ReservedVLAN, true},
		{"above max", MaxVLAN + 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &megaport.VXCOrderVrouterPartnerConfig{
				Interfaces: []megaport.PartnerConfigInterface{{VLAN: tt.vlan}},
			}
			err := ValidateVrouterPartnerConfig(config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVXCEndVLAN(t *testing.T) {
	tests := []struct {
		name    string
		vlan    int
		wantErr bool
	}{
		{"Valid VLAN 100", 100, false},
		{"Valid Min Assignable VLAN", MinAssignableVLAN, false},
		{"Valid Max VLAN", MaxVLAN, false},
		{"Valid Untagged", UntaggedVLAN, false},
		{"Valid Auto Assign", AutoAssignVLAN, false},
		{"Invalid VLAN 1", 1, true}, // Reserved VLAN
		{"Invalid VLAN Too Low", -2, true},
		{"Invalid VLAN Too High", MaxVLAN + 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVXCEndVLAN(tt.vlan)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVXCEndVLAN() error = %v, wantErr:%v", err, tt.wantErr)
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
			}
		})
	}
}

func TestValidateVXCEndInnerVLAN(t *testing.T) {
	// Inner VLAN validation uses the same logic as outer VLAN
	tests := []struct {
		name    string
		vlan    int
		wantErr bool
	}{
		{"Valid Inner VLAN 200", 200, false},
		{"Valid Inner Min Assignable VLAN", MinAssignableVLAN, false},
		{"Valid Inner Max VLAN", MaxVLAN, false},
		{"Valid Inner Untagged", UntaggedVLAN, false},
		{"Valid Inner Auto Assign", AutoAssignVLAN, false},
		{"Invalid Inner VLAN 1", 1, true}, // Reserved VLAN
		{"Invalid Inner VLAN Too Low", -2, true},
		{"Invalid Inner VLAN Too High", MaxVLAN + 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVXCEndInnerVLAN(tt.vlan)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVXCEndInnerVLAN() error = %v, wantErr:%v", err, tt.wantErr)
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
			}
		})
	}
}

func TestValidateVXCRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *megaport.BuyVXCRequest
		wantErr bool
		errText string
	}{
		{
			name: "Valid VXC request",
			req: &megaport.BuyVXCRequest{
				VXCName:   "Test VXC",
				Term:      12,
				RateLimit: 1000,
				PortUID:   "a-end-uid",
				BEndConfiguration: megaport.VXCOrderEndpointConfiguration{
					ProductUID: "b-end-uid",
				},
			},
			wantErr: false,
		},
		{
			name: "Valid VXC request with partner config",
			req: &megaport.BuyVXCRequest{
				VXCName:   "Test VXC Partner",
				Term:      1,
				RateLimit: 500,
				PortUID:   "a-end-uid",
				BEndConfiguration: megaport.VXCOrderEndpointConfiguration{
					PartnerConfig: &megaport.VXCPartnerConfigAWS{
						ConnectType:    "AWS",
						OwnerAccount:   "12345",
						ASN:            65000,
						ConnectionName: "MyAWSConnection",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			req: &megaport.BuyVXCRequest{
				VXCName:   "",
				Term:      12,
				RateLimit: 1000,
				PortUID:   "a-end-uid",
				BEndConfiguration: megaport.VXCOrderEndpointConfiguration{
					ProductUID: "b-end-uid",
				},
			},
			wantErr: true,
			errText: "Invalid VXC name:  - cannot be empty",
		},
		{
			name: "Invalid term",
			req: &megaport.BuyVXCRequest{
				VXCName:   "Test VXC",
				Term:      5, // Invalid term
				RateLimit: 1000,
				PortUID:   "a-end-uid",
				BEndConfiguration: megaport.VXCOrderEndpointConfiguration{
					ProductUID: "b-end-uid",
				},
			},
			wantErr: true,
			errText: fmt.Sprintf("Invalid contract term: 5 - must be one of: %v", ValidContractTerms),
		},
		{
			name: "Invalid rate limit",
			req: &megaport.BuyVXCRequest{
				VXCName:   "Test VXC",
				Term:      12,
				RateLimit: 0, // Invalid rate limit
				PortUID:   "a-end-uid",
				BEndConfiguration: megaport.VXCOrderEndpointConfiguration{
					ProductUID: "b-end-uid",
				},
			},
			wantErr: true,
			errText: "Invalid rate limit: 0 - must be a positive integer",
		},
		{
			name: "Empty A-End UID",
			req: &megaport.BuyVXCRequest{
				VXCName:   "Test VXC",
				Term:      12,
				RateLimit: 1000,
				PortUID:   "", // Empty A-End UID
				BEndConfiguration: megaport.VXCOrderEndpointConfiguration{
					ProductUID: "b-end-uid",
				},
			},
			wantErr: true,
			errText: "Invalid A-End UID (PortUID):  - cannot be empty",
		},
		{
			name: "Empty B-End UID without partner config",
			req: &megaport.BuyVXCRequest{
				VXCName:   "Test VXC",
				Term:      12,
				RateLimit: 1000,
				PortUID:   "a-end-uid",
				BEndConfiguration: megaport.VXCOrderEndpointConfiguration{
					ProductUID: "", // Empty B-End UID
				},
			},
			wantErr: true,
			errText: "Invalid B-End UID:  - cannot be empty when no partner configuration is provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVXCRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVXCRequest() error = %v, wantErr:%v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateAWSPartnerConfig(t *testing.T) {
	tests := []struct {
		name              string
		connectType       string
		ownerAccount      string
		asn               int
		amazonAsn         int
		authKey           string
		customerIPAddress string
		amazonIPAddress   string
		awsName           string
		awsType           string
		wantErr           bool
		errText           string
	}{
		{
			name:              "Valid AWS config",
			connectType:       "private",
			ownerAccount:      "123456789012",
			asn:               65000,
			amazonAsn:         64512,
			authKey:           "authkey123",
			customerIPAddress: "192.168.1.1/30",
			amazonIPAddress:   "192.168.1.2/30",
			awsName:           "MyAWSConnection",
			awsType:           "private",
			wantErr:           false,
		},
		{
			name:              "AWS ASN out of range",
			connectType:       "private",
			ownerAccount:      "123456789012",
			asn:               -1,
			amazonAsn:         64512,
			authKey:           "authkey123",
			customerIPAddress: "192.168.1.1/30",
			amazonIPAddress:   "192.168.1.2/30",
			awsName:           "MyAWSConnection",
			awsType:           "private",
			wantErr:           true,
			errText:           "Invalid ASN: -1 - must be between 1-4294967295",
		},
		{
			name:              "Valid AWSHC config",
			connectType:       "public",
			ownerAccount:      "123456789012",
			asn:               65001,
			amazonAsn:         64512,
			authKey:           "authkey123",
			customerIPAddress: "192.168.1.1/30",
			amazonIPAddress:   "192.168.1.2/30",
			awsName:           "MyAWSHCConnection",
			wantErr:           false,
		},
		{
			name:         "Empty connect type",
			connectType:  "",
			ownerAccount: "123456789012",
			asn:          65000,
			wantErr:      true,
			errText:      "Invalid AWS connect type:  - cannot be empty",
		},
		{
			connectType:  "INVALID",
			ownerAccount: "123456789012",
			asn:          65000,
			wantErr:      true,
			errText:      "Invalid AWS connect type: INVALID - must be 'AWS', 'AWSHC', 'private', or 'public'",
		},
		{
			name:         "Empty owner account",
			connectType:  "AWS",
			ownerAccount: "",
			asn:          65000,
			wantErr:      true,
			errText:      "Invalid AWS owner account:  - cannot be empty",
		},
		{
			name:              "Invalid customer IP CIDR",
			connectType:       "AWS",
			ownerAccount:      "123456789012",
			asn:               65000,
			customerIPAddress: "invalid-ip",
			awsName:           "MyAWSConnection",
			wantErr:           true,
			errText:           "Invalid AWS customer IP address: invalid-ip - must be a valid IPv4 CIDR notation", // Updated error message
		},
		{
			name:            "Invalid Amazon IP CIDR",
			connectType:     "AWS",
			ownerAccount:    "123456789012",
			asn:             65000,
			amazonIPAddress: "192.168.1.2/33", // Invalid mask
			awsName:         "MyAWSConnection",
			wantErr:         true,
			errText:         "Invalid AWS Amazon IP address: 192.168.1.2/33 - must be a valid IPv4 CIDR notation", // Updated error message
		},
		{
			name:         "AWS name too long",
			connectType:  "AWS",
			ownerAccount: "123456789012",
			asn:          65000,
			awsName:      string(make([]byte, 256)), // 256 chars
			wantErr:      true,
			errText:      "Invalid AWS connection name: ", // Error message includes the long name, truncated here
		},
		{
			name:         "AWS name at max length with multibyte characters accepted",
			connectType:  "AWS",
			ownerAccount: "123456789012",
			asn:          65000,
			awsName:      strings.Repeat("日", MaxAWSConnectionNameLength),
			wantErr:      false,
		},
		{
			name:         "AWS name over max length with multibyte characters rejected",
			connectType:  "AWS",
			ownerAccount: "123456789012",
			asn:          65000,
			awsName:      strings.Repeat("日", MaxAWSConnectionNameLength+1),
			wantErr:      true,
			errText:      "Invalid AWS connection name: ",
		},
		{
			name:         "Invalid AWS type for AWS connect type",
			connectType:  "AWS",
			ownerAccount: "123456789012",
			asn:          65000,
			awsType:      "invalid",
			awsName:      "MyAWSConnection",
			wantErr:      true,
			errText:      "Invalid AWS type: invalid - must be 'private' or 'public' for AWS connect type",
		},
		{
			name:         "Empty connection name is valid (API defaults to MEGAPORT)",
			connectType:  "AWS",
			ownerAccount: "123456789012",
			asn:          65000,
			awsName:      "",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			awsConfig := &megaport.VXCPartnerConfigAWS{
				ConnectType:       tt.connectType,
				OwnerAccount:      tt.ownerAccount,
				CustomerIPAddress: tt.customerIPAddress,
				AmazonIPAddress:   tt.amazonIPAddress,
				ConnectionName:    tt.awsName,
				Type:              tt.awsType,
				ASN:               tt.asn,
				AmazonASN:         tt.amazonAsn,
			}
			err := ValidateAWSPartnerConfig(awsConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAWSPartnerConfig() error = %v, wantErr:%v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				// Use Contains because the long name error message is hard to match exactly
				assert.Contains(t, err.Error(), tt.errText, "Error message mismatch")
			}
		})
	}
}

func TestValidateAWSPartnerConfig_NilConfig(t *testing.T) {
	err := ValidateAWSPartnerConfig(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid AWS partner config: <nil> - cannot be nil")
}

// TestValidateAWSPartnerConfig_HighASN mirrors TestValidateBGPConnectionConfig_HighASN:
// it confirms the AWS validator agrees with ValidateASN at the 32-bit max boundary.
func TestValidateAWSPartnerConfig_HighASN(t *testing.T) {
	if int64(math.MaxInt) < int64(math.MaxUint32) {
		t.Skip("int is narrower than the 32-bit ASN range on this platform")
	}
	max64 := MaxASN
	maxValid := int(max64)     // 4294967295
	aboveMax := int(max64) + 1 // 4294967296
	tests := []struct {
		name    string
		asn     int
		wantErr bool
		errText string
	}{
		{"ASN 4294967295 (32-bit max)", maxValid, false, ""},
		{"ASN above 32-bit max rejected", aboveMax, true,
			fmt.Sprintf("Invalid ASN: %d - must be between 1-4294967295", aboveMax)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			awsConfig := &megaport.VXCPartnerConfigAWS{
				ConnectType:       "private",
				OwnerAccount:      "123456789012",
				ASN:               tt.asn,
				AmazonASN:         64512,
				CustomerIPAddress: "192.168.1.1/30",
				AmazonIPAddress:   "192.168.1.2/30",
				ConnectionName:    "test",
				Type:              "private",
			}
			err := ValidateAWSPartnerConfig(awsConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAWSPartnerConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err)
				assert.Equal(t, tt.errText, err.Error())
			}
		})
	}
}

func TestValidateGooglePartnerConfig(t *testing.T) {
	tests := []struct {
		name       string
		pairingKey string
		wantErr    bool
		errText    string
	}{
		{"Valid Google config", "google-pairing-key", false, ""},
		{"Empty pairing key", "", true, "Invalid Google pairing key:  - cannot be empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			googleConfig := &megaport.VXCPartnerConfigGoogle{
				PairingKey: tt.pairingKey,
			}
			err := ValidateGooglePartnerConfig(googleConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGooglePartnerConfig() error = %v, wantErr:%v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateGooglePartnerConfig_NilConfig(t *testing.T) {
	err := ValidateGooglePartnerConfig(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid Google partner config: <nil> - cannot be nil")
}

func TestValidateOraclePartnerConfig(t *testing.T) {
	tests := []struct {
		name             string
		virtualCircuitID string
		wantErr          bool
		errText          string
	}{
		{"Valid Oracle config", "ocid1.virtualcircuit.oc1..example", false, ""},
		{"Empty virtual circuit ID", "", true, "Invalid Oracle virtual circuit ID:  - cannot be empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fix: Use the correct struct for ValidateOraclePartnerConfig
			oracleConfig := &megaport.VXCPartnerConfigOracle{
				VirtualCircuitId: tt.virtualCircuitID,
			}
			err := ValidateOraclePartnerConfig(oracleConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOraclePartnerConfig() error = %v, wantErr:%v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateOraclePartnerConfig_NilConfig(t *testing.T) {
	err := ValidateOraclePartnerConfig(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid Oracle partner config: <nil> - cannot be nil")
}

func TestValidateIBMPartnerConfig(t *testing.T) {
	validAccountID := "abcdef0123456789abcdef0123456789" // 32 hex chars
	tests := []struct {
		name              string
		accountID         string
		customerASN       int
		ibmName           string
		customerIPAddress string
		providerIPAddress string
		wantErr           bool
		errText           string
	}{
		{
			name:              "Valid IBM config",
			accountID:         validAccountID,
			customerASN:       65000,
			ibmName:           "MyIBMConnection",
			customerIPAddress: "10.1.1.1/30",
			providerIPAddress: "10.1.1.2/30",
			wantErr:           false,
		},
		{
			name:      "Empty account ID",
			accountID: "",
			wantErr:   true,
			errText:   "Invalid IBM account ID:  - cannot be empty",
		},
		{
			name:      "Account ID too short",
			accountID: "short",
			wantErr:   true,
			errText:   fmt.Sprintf("Invalid IBM account ID: short - must be exactly %d characters", IBMAccountIDLength),
		},
		{
			name:      "Account ID too long",
			accountID: validAccountID + "extra",
			wantErr:   true,
			errText:   fmt.Sprintf("Invalid IBM account ID: %sextra - must be exactly %d characters", validAccountID, IBMAccountIDLength),
		},
		{
			name:      "Account ID invalid characters",
			accountID: "abcdef0123456789abcdef012345678X", // X is invalid
			wantErr:   true,
			errText:   "Invalid IBM account ID: abcdef0123456789abcdef012345678X - must contain only hexadecimal characters (0-9, a-f, A-F)",
		},
		{
			name:      "Name too long",
			accountID: validAccountID,
			ibmName:   string(make([]byte, MaxIBMNameLength+1)),
			wantErr:   true,
			errText:   "Invalid IBM connection name: ", // Error message includes the long name, truncated here
		},
		{
			name:      "Name invalid characters",
			accountID: validAccountID,
			ibmName:   "My IBM Connection!", // ! is invalid
			wantErr:   true,
			errText:   "Invalid IBM connection name: My IBM Connection! - must only contain characters 0-9, a-z, A-Z, /, -, _, or ,",
		},
		{
			name:              "Invalid customer IP",
			accountID:         validAccountID,
			ibmName:           "MyIBMConnection",
			customerIPAddress: "invalid-ip",
			wantErr:           true,
			errText:           "Invalid IBM customer IP address: invalid-ip - must be a valid IPv4 CIDR notation",
		},
		{
			name:              "Invalid provider IP",
			accountID:         validAccountID,
			ibmName:           "MyIBMConnection",
			providerIPAddress: "10.1.1.2/33", // Invalid mask
			wantErr:           true,
			errText:           "Invalid IBM provider IP address: 10.1.1.2/33 - must be a valid IPv4 CIDR notation",
		},
		{
			name:      "Empty connection name is valid (API defaults to MEGAPORT)",
			accountID: validAccountID,
			ibmName:   "",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fix: Use the correct struct for ValidateIBMPartnerConfig
			ibmConfig := &megaport.VXCPartnerConfigIBM{
				AccountID:         tt.accountID,
				Name:              tt.ibmName,
				CustomerIPAddress: tt.customerIPAddress,
				ProviderIPAddress: tt.providerIPAddress,
			}
			err := ValidateIBMPartnerConfig(ibmConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIBMPartnerConfig() error = %v, wantErr:%v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Contains(t, err.Error(), tt.errText, "Error message mismatch")
			}
		})
	}
}

func TestValidateIBMPartnerConfig_NilConfig(t *testing.T) {
	err := ValidateIBMPartnerConfig(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid IBM partner config: <nil> - cannot be nil")
}

func TestValidateVXCPartnerConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  megaport.VXCPartnerConfiguration // Use interface type
		wantErr bool
		errText string
	}{
		{
			name: "Valid AWS partner config",
			config: &megaport.VXCPartnerConfigAWS{ // Use struct pointer
				ConnectType:    "AWS",
				OwnerAccount:   "123456789012",
				ASN:            65000,
				ConnectionName: "MyAWSConnection",
			},
			wantErr: false,
		},
		{
			name: "Valid Azure partner config",
			config: &megaport.VXCPartnerConfigAzure{ // Use struct pointer
				ConnectType: "AZURE", // Assuming ConnectType is needed
				ServiceKey:  "azure-key",
			},
			wantErr: false,
		},
		{
			name: "Valid Google partner config",
			config: &megaport.VXCPartnerConfigGoogle{ // Use struct pointer
				ConnectType: "GOOGLE", // Assuming ConnectType is needed
				PairingKey:  "google-key",
			},
			wantErr: false,
		},
		{
			name: "Valid Oracle partner config",
			config: &megaport.VXCPartnerConfigOracle{ // Use struct pointer
				ConnectType:      "ORACLE", // Assuming ConnectType is needed
				VirtualCircuitId: "oracle-vcid",
			},
			wantErr: false,
		},
		{
			name: "Valid IBM partner config",
			config: &megaport.VXCPartnerConfigIBM{ // Use struct pointer
				ConnectType: "IBM", // Assuming ConnectType is needed
				AccountID:   "abcdef0123456789abcdef0123456789",
				Name:        "MyIBMConnection",
			},
			wantErr: false,
		},
		{
			name: "Valid vRouter partner config",
			config: &megaport.VXCOrderVrouterPartnerConfig{ // Use struct pointer
				Interfaces: []megaport.PartnerConfigInterface{
					{
						VLAN: 100,
						IpAddresses: []string{
							"192.168.1.1/30",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid Transit partner config",
			config: &megaport.VXCPartnerConfigTransit{
				ConnectType: "TRANSIT",
			},
			wantErr: false,
		},
		{
			name:    "Missing partner type (nil config)", // Test case for nil config
			config:  nil,
			wantErr: true,
			errText: "Invalid Partner configuration type: <nil> - is not supported", // Adjusted error based on test output
		},
		{
			name: "Invalid partner type in AWS config",
			config: &megaport.VXCPartnerConfigAWS{
				ConnectType:  "invalid", // Invalid type set in a specific struct
				OwnerAccount: "123456789012",
				ASN:          65000,
			},
			wantErr: true,
			errText: "Invalid AWS connect type: invalid - must be 'AWS', 'AWSHC', 'private', or 'public'", // Adjusted error based on test output
		},
		{
			name:    "Missing config details (handled by specific validators)",
			config:  &megaport.VXCPartnerConfigAWS{}, // Empty AWS config
			wantErr: true,
			errText: "Invalid AWS connect type:  - cannot be empty", // Error from ValidateAWSPartnerConfig
		},
		{
			name: "Invalid AWS config details",
			config: &megaport.VXCPartnerConfigAWS{
				ConnectType: "", // Invalid connect type
				ASN:         65000,
			},
			wantErr: true,
			errText: "Invalid AWS connect type:  - cannot be empty",
		},
		{
			name: "Invalid Azure config details",
			config: &megaport.VXCPartnerConfigAzure{
				ServiceKey: "", // Invalid service key
			},
			wantErr: true,
			errText: "Invalid Azure service key:  - cannot be empty",
		},
		{
			name: "Invalid Transit config details",
			config: &megaport.VXCPartnerConfigTransit{
				ConnectType: "", // Invalid connect type
			},
			wantErr: true,
			errText: "Invalid Transit connect type:  - must be 'TRANSIT'",
		},
		{
			name: "Invalid vRouter config details",
			config: &megaport.VXCOrderVrouterPartnerConfig{
				Interfaces: []megaport.PartnerConfigInterface{
					{
						VLAN: 1, // Invalid VLAN
					},
				},
			},
			wantErr: true,
			errText: fmt.Sprintf("Invalid vRouter interface [0] VLAN: 1 - must be valid (%d for auto-assign, %d for untagged, or %d-%d except %d)", AutoAssignVLAN, UntaggedVLAN, MinAssignableVLAN, MaxVLAN, ReservedVLAN),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVXCPartnerConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVXCPartnerConfig() error = %v, wantErr:%v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestValidateTransitPartnerConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *megaport.VXCPartnerConfigTransit
		wantErr bool
		errText string
	}{
		{
			name: "Valid Transit config",
			config: &megaport.VXCPartnerConfigTransit{
				ConnectType: "TRANSIT",
			},
			wantErr: false,
		},
		{
			name:    "Nil config",
			config:  nil,
			wantErr: true,
			errText: "Invalid Transit partner config: <nil> - cannot be nil",
		},
		{
			name: "Empty connect type",
			config: &megaport.VXCPartnerConfigTransit{
				ConnectType: "",
			},
			wantErr: true,
			errText: "Invalid Transit connect type:  - must be 'TRANSIT'",
		},
		{
			name: "Wrong connect type",
			config: &megaport.VXCPartnerConfigTransit{
				ConnectType: "AWS",
			},
			wantErr: true,
			errText: "Invalid Transit connect type: AWS - must be 'TRANSIT'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransitPartnerConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTransitPartnerConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}

func TestIsValidIBMName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"alphanumeric", "abc123", true},
		{"uppercase", "ABC", true},
		{"with slash", "my/name", true},
		{"with dash", "my-name", true},
		{"with underscore", "my_name", true},
		{"with comma", "my,name", true},
		{"all special chars", "a/b-c_d,e", true},
		{"empty string", "", true},
		{"with space", "my name", false},
		{"with dot", "my.name", false},
		{"with at sign", "my@name", false},
		{"with exclamation", "hello!", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isValidIBMName(tt.input))
		})
	}
}

func TestValidateAzurePartnerConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *megaport.VXCPartnerConfigAzure
		wantErr bool
		errText string
	}{
		{
			name: "Valid Azure config without peers",
			config: &megaport.VXCPartnerConfigAzure{
				ServiceKey: "valid-service-key",
			},
			wantErr: false,
		},
		{
			name: "Valid Azure config with valid peer - primary subnet",
			config: &megaport.VXCPartnerConfigAzure{
				ServiceKey: "valid-service-key",
				Peers: []megaport.PartnerOrderAzurePeeringConfig{
					{
						PrimarySubnet: "10.0.0.0/24",
						VLAN:          100,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid Azure config with valid peer - secondary subnet",
			config: &megaport.VXCPartnerConfigAzure{
				ServiceKey: "valid-service-key",
				Peers: []megaport.PartnerOrderAzurePeeringConfig{
					{
						SecondarySubnet: "10.0.1.0/24",
						VLAN:            200, // Added valid VLAN
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid Azure config with valid peer - both subnets",
			config: &megaport.VXCPartnerConfigAzure{
				ServiceKey: "valid-service-key",
				Peers: []megaport.PartnerOrderAzurePeeringConfig{
					{
						PrimarySubnet:   "10.0.0.0/24",
						SecondarySubnet: "10.0.1.0/24",
						VLAN:            300, // Added valid VLAN
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid Azure config with auto-assign VLAN",
			config: &megaport.VXCPartnerConfigAzure{
				ServiceKey: "valid-service-key",
				Peers: []megaport.PartnerOrderAzurePeeringConfig{
					{
						PrimarySubnet: "10.0.0.0/24",
						VLAN:          0, // Auto-assign
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid Azure config with untagged VLAN",
			config: &megaport.VXCPartnerConfigAzure{
				ServiceKey: "valid-service-key",
				Peers: []megaport.PartnerOrderAzurePeeringConfig{
					{
						PrimarySubnet: "10.0.0.0/24",
						VLAN:          -1, // Untagged
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Empty service key",
			config: &megaport.VXCPartnerConfigAzure{
				ServiceKey: "",
			},
			wantErr: true,
			errText: "Invalid Azure service key:  - cannot be empty",
		},
		{
			name:    "Nil config",
			config:  nil,
			wantErr: true,
			errText: "Invalid Azure partner config: <nil> - cannot be nil",
		},
		{
			name: "Invalid peer - missing subnets",
			config: &megaport.VXCPartnerConfigAzure{
				ServiceKey: "valid-service-key",
				Peers: []megaport.PartnerOrderAzurePeeringConfig{
					{
						VLAN: 100, // Valid VLAN but missing subnets
					},
				},
			},
			wantErr: true,
			errText: "Invalid Azure peer [0] subnet: <nil> - at least one of primary_subnet or secondary_subnet must be provided",
		},
		{
			name: "Invalid peer - invalid VLAN (reserved value)",
			config: &megaport.VXCPartnerConfigAzure{
				ServiceKey: "valid-service-key",
				Peers: []megaport.PartnerOrderAzurePeeringConfig{
					{
						PrimarySubnet: "10.0.0.0/24",
						VLAN:          1, // Reserved VLAN
					},
				},
			},
			wantErr: true,
			errText: fmt.Sprintf("Invalid Azure peer [0] VLAN: 1 - must be valid (0 for auto-assign, -1 for untagged, or %d-%d except %d)",
				MinAssignableVLAN, MaxVLAN, ReservedVLAN),
		},
		{
			name: "Invalid peer - VLAN too high",
			config: &megaport.VXCPartnerConfigAzure{
				ServiceKey: "valid-service-key",
				Peers: []megaport.PartnerOrderAzurePeeringConfig{
					{
						PrimarySubnet: "10.0.0.0/24",
						VLAN:          4095, // Too high
					},
				},
			},
			wantErr: true,
			errText: fmt.Sprintf("Invalid Azure peer [0] VLAN: 4095 - must be valid (0 for auto-assign, -1 for untagged, or %d-%d except %d)",
				MinAssignableVLAN, MaxVLAN, ReservedVLAN),
		},
		{
			name: "Multiple peers with one invalid",
			config: &megaport.VXCPartnerConfigAzure{
				ServiceKey: "valid-service-key",
				Peers: []megaport.PartnerOrderAzurePeeringConfig{
					{
						PrimarySubnet: "10.0.0.0/24", // Valid
						VLAN:          100,           // Valid VLAN
					},
					{
						// Invalid - both subnets missing
						VLAN: 200, // Valid VLAN
					},
				},
			},
			wantErr: true,
			errText: "Invalid Azure peer [1] subnet: <nil> - at least one of primary_subnet or secondary_subnet must be provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAzurePartnerConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAzurePartnerConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				assert.IsType(t, &ValidationError{}, err, "Expected ValidationError type")
				assert.Equal(t, tt.errText, err.Error(), "Error message mismatch")
			}
		})
	}
}
