package vxc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/megaport/megaport-cli/internal/validation"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePartnerConfigFromJSON(t *testing.T) {
	tests := []struct {
		name          string
		jsonStr       string
		expectedError string
		validate      func(*testing.T, megaport.VXCPartnerConfiguration)
	}{
		{
			name:    "valid AWS config",
			jsonStr: `{"connectType":"AWS","ownerAccount":"123456"}`,
			validate: func(t *testing.T, cfg megaport.VXCPartnerConfiguration) {
				_, ok := cfg.(*megaport.VXCPartnerConfigAWS)
				assert.True(t, ok, "expected *megaport.VXCPartnerConfigAWS")
			},
		},
		{
			name:    "valid transit config",
			jsonStr: `{"connectType":"TRANSIT"}`,
			validate: func(t *testing.T, cfg megaport.VXCPartnerConfiguration) {
				_, ok := cfg.(*megaport.VXCPartnerConfigTransit)
				assert.True(t, ok, "expected *megaport.VXCPartnerConfigTransit")
			},
		},
		{
			name:    "case insensitive connectType",
			jsonStr: `{"connectType":"aws","ownerAccount":"123456"}`,
			validate: func(t *testing.T, cfg megaport.VXCPartnerConfiguration) {
				_, ok := cfg.(*megaport.VXCPartnerConfigAWS)
				assert.True(t, ok, "expected *megaport.VXCPartnerConfigAWS")
			},
		},
		{
			name:          "invalid JSON",
			jsonStr:       `{invalid}`,
			expectedError: "invalid",
		},
		{
			name:          "missing connectType",
			jsonStr:       `{"ownerAccount":"123456"}`,
			expectedError: "connectType",
		},
		{
			name:          "unsupported connectType",
			jsonStr:       `{"connectType":"UNKNOWN"}`,
			expectedError: "unsupported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsePartnerConfigFromJSON(tt.jsonStr)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestParseAWSConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]interface{}
		expectedError string
		validate      func(*testing.T, *megaport.VXCPartnerConfigAWS)
	}{
		{
			name: "required fields only",
			config: map[string]interface{}{
				"connectType":  "AWS",
				"ownerAccount": "123456",
			},
			validate: func(t *testing.T, cfg *megaport.VXCPartnerConfigAWS) {
				assert.Equal(t, "AWS", cfg.ConnectType)
				assert.Equal(t, "123456", cfg.OwnerAccount)
			},
		},
		{
			name: "all optional fields",
			config: map[string]interface{}{
				"connectType":       "AWS",
				"ownerAccount":      "123456",
				"asn":               65000.0,
				"amazonAsn":         64512.0,
				"authKey":           "key",
				"prefixes":          "10.0.0.0/8",
				"customerIPAddress": "1.2.3.4",
				"amazonIPAddress":   "5.6.7.8",
				"connectionName":    "conn1",
				"type":              "private",
			},
			validate: func(t *testing.T, cfg *megaport.VXCPartnerConfigAWS) {
				assert.Equal(t, "AWS", cfg.ConnectType)
				assert.Equal(t, "123456", cfg.OwnerAccount)
				assert.Equal(t, 65000, cfg.ASN)
				assert.Equal(t, 64512, cfg.AmazonASN)
				assert.Equal(t, "key", cfg.AuthKey)
				assert.Equal(t, "10.0.0.0/8", cfg.Prefixes)
				assert.Equal(t, "1.2.3.4", cfg.CustomerIPAddress)
				assert.Equal(t, "5.6.7.8", cfg.AmazonIPAddress)
				assert.Equal(t, "conn1", cfg.ConnectionName)
				assert.Equal(t, "private", cfg.Type)
			},
		},
		{
			name: "missing ownerAccount",
			config: map[string]interface{}{
				"connectType": "AWS",
			},
			expectedError: "ownerAccount",
		},
		{
			name: "empty ownerAccount",
			config: map[string]interface{}{
				"connectType":  "AWS",
				"ownerAccount": "",
			},
			expectedError: "cannot be empty",
		},
		{
			name: "asn wrong type",
			config: map[string]interface{}{
				"connectType":  "AWS",
				"ownerAccount": "123456",
				"asn":          "string",
			},
			expectedError: "must be a number",
		},
		{
			name: "negative asn",
			config: map[string]interface{}{
				"connectType":  "AWS",
				"ownerAccount": "123456",
				"asn":          -1.0,
			},
			expectedError: "cannot be negative",
		},
		{
			name: "AWSHC connectType",
			config: map[string]interface{}{
				"connectType":  "AWSHC",
				"ownerAccount": "123456",
			},
			validate: func(t *testing.T, cfg *megaport.VXCPartnerConfigAWS) {
				assert.Equal(t, "AWSHC", cfg.ConnectType)
				assert.Equal(t, "123456", cfg.OwnerAccount)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAWSConfig(tt.config)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestParseAzureConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]interface{}
		expectedError string
		validate      func(*testing.T, *megaport.VXCPartnerConfigAzure)
	}{
		{
			name: "required fields only",
			config: map[string]interface{}{
				"serviceKey": "sk-123",
			},
			validate: func(t *testing.T, cfg *megaport.VXCPartnerConfigAzure) {
				assert.Equal(t, "AZURE", cfg.ConnectType)
				assert.Equal(t, "sk-123", cfg.ServiceKey)
			},
		},
		{
			name: "with peers",
			config: map[string]interface{}{
				"serviceKey": "sk-123",
				"peers": []interface{}{
					map[string]interface{}{
						"type":            "private",
						"peerASN":         "65000",
						"primarySubnet":   "10.0.0.0/30",
						"secondarySubnet": "10.0.0.4/30",
						"prefixes":        "10.1.0.0/16",
						"sharedKey":       "key",
						"vlan":            100.0,
					},
				},
			},
			validate: func(t *testing.T, cfg *megaport.VXCPartnerConfigAzure) {
				assert.Equal(t, "AZURE", cfg.ConnectType)
				assert.Equal(t, "sk-123", cfg.ServiceKey)
				assert.Len(t, cfg.Peers, 1)
				peer := cfg.Peers[0]
				assert.Equal(t, "private", peer.Type)
				assert.Equal(t, "65000", peer.PeerASN)
				assert.Equal(t, "10.0.0.0/30", peer.PrimarySubnet)
				assert.Equal(t, "10.0.0.4/30", peer.SecondarySubnet)
				assert.Equal(t, "10.1.0.0/16", peer.Prefixes)
				assert.Equal(t, "key", peer.SharedKey)
				assert.Equal(t, 100, peer.VLAN)
			},
		},
		{
			name:          "missing serviceKey",
			config:        map[string]interface{}{},
			expectedError: "serviceKey",
		},
		{
			name: "empty serviceKey",
			config: map[string]interface{}{
				"serviceKey": "",
			},
			expectedError: "cannot be empty",
		},
		{
			name: "peers not array",
			config: map[string]interface{}{
				"serviceKey": "sk-123",
				"peers":      "string",
			},
			expectedError: "must be an array",
		},
		{
			name: "invalid vlan in peer",
			config: map[string]interface{}{
				"serviceKey": "sk-123",
				"peers": []interface{}{
					map[string]interface{}{
						"vlan": "string",
					},
				},
			},
			expectedError: "must be a number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAzureConfig(tt.config)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestParseGoogleConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]interface{}
		expectedError string
		validate      func(*testing.T, *megaport.VXCPartnerConfigGoogle)
	}{
		{
			name: "valid",
			config: map[string]interface{}{
				"pairingKey": "pk-123",
			},
			validate: func(t *testing.T, cfg *megaport.VXCPartnerConfigGoogle) {
				assert.Equal(t, "GOOGLE", cfg.ConnectType)
				assert.Equal(t, "pk-123", cfg.PairingKey)
			},
		},
		{
			name:          "missing pairingKey",
			config:        map[string]interface{}{},
			expectedError: "pairingKey",
		},
		{
			name: "empty pairingKey",
			config: map[string]interface{}{
				"pairingKey": "",
			},
			expectedError: "cannot be empty",
		},
		{
			name: "non-string pairingKey",
			config: map[string]interface{}{
				"pairingKey": 123.0,
			},
			expectedError: "must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseGoogleConfig(tt.config)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestParseOracleConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]interface{}
		expectedError string
		validate      func(*testing.T, *megaport.VXCPartnerConfigOracle)
	}{
		{
			name: "valid",
			config: map[string]interface{}{
				"virtualCircuitId": "vc-123",
			},
			validate: func(t *testing.T, cfg *megaport.VXCPartnerConfigOracle) {
				assert.Equal(t, "ORACLE", cfg.ConnectType)
				assert.Equal(t, "vc-123", cfg.VirtualCircuitId)
			},
		},
		{
			name:          "missing",
			config:        map[string]interface{}{},
			expectedError: "virtualCircuitId",
		},
		{
			name: "empty",
			config: map[string]interface{}{
				"virtualCircuitId": "",
			},
			expectedError: "cannot be empty",
		},
		{
			name: "non-string",
			config: map[string]interface{}{
				"virtualCircuitId": 123.0,
			},
			expectedError: "must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseOracleConfig(tt.config)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestParseIBMConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]interface{}
		expectedError string
		validate      func(*testing.T, *megaport.VXCPartnerConfigIBM)
	}{
		{
			name: "required fields only",
			config: map[string]interface{}{
				"accountID": "acct-123",
			},
			validate: func(t *testing.T, cfg *megaport.VXCPartnerConfigIBM) {
				assert.Equal(t, "IBM", cfg.ConnectType)
				assert.Equal(t, "acct-123", cfg.AccountID)
			},
		},
		{
			name: "all fields",
			config: map[string]interface{}{
				"accountID":         "acct-123",
				"customerASN":       65000.0,
				"customerIPAddress": "1.2.3.4",
				"providerIPAddress": "5.6.7.8",
				"name":              "my-ibm",
			},
			validate: func(t *testing.T, cfg *megaport.VXCPartnerConfigIBM) {
				assert.Equal(t, "IBM", cfg.ConnectType)
				assert.Equal(t, "acct-123", cfg.AccountID)
				assert.Equal(t, 65000, cfg.CustomerASN)
				assert.Equal(t, "1.2.3.4", cfg.CustomerIPAddress)
				assert.Equal(t, "5.6.7.8", cfg.ProviderIPAddress)
				assert.Equal(t, "my-ibm", cfg.Name)
			},
		},
		{
			name:          "missing accountID",
			config:        map[string]interface{}{},
			expectedError: "accountID",
		},
		{
			name: "empty accountID",
			config: map[string]interface{}{
				"accountID": "",
			},
			expectedError: "cannot be empty",
		},
		{
			name: "customerASN wrong type",
			config: map[string]interface{}{
				"accountID":   "acct-123",
				"customerASN": "string",
			},
			expectedError: "must be a number",
		},
		{
			name: "negative customerASN",
			config: map[string]interface{}{
				"accountID":   "acct-123",
				"customerASN": -1.0,
			},
			expectedError: "cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseIBMConfig(tt.config)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestParseVRouterConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]interface{}
		expectedError string
		validate      func(*testing.T, *megaport.VXCOrderVrouterPartnerConfig)
	}{
		{
			name: "no interfaces key",
			config: map[string]interface{}{
				"connectType": "VROUTER",
			},
			validate: func(t *testing.T, cfg *megaport.VXCOrderVrouterPartnerConfig) {
				assert.Empty(t, cfg.Interfaces)
			},
		},
		{
			name: "single interface",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"vlan":        100.0,
						"ipAddresses": []interface{}{"10.0.0.1/30"},
					},
				},
			},
			validate: func(t *testing.T, cfg *megaport.VXCOrderVrouterPartnerConfig) {
				assert.Len(t, cfg.Interfaces, 1)
				iface := cfg.Interfaces[0]
				assert.Equal(t, 100, iface.VLAN)
				assert.Equal(t, []string{"10.0.0.1/30"}, iface.IpAddresses)
			},
		},
		{
			name: "interface with routes and bfd",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"vlan":        100.0,
						"ipAddresses": []interface{}{"10.0.0.1/30"},
						"ipRoutes": []interface{}{
							map[string]interface{}{
								"prefix":      "10.1.0.0/24",
								"nextHop":     "10.0.0.2",
								"description": "route1",
							},
						},
						"bfd": map[string]interface{}{
							"txInterval": 300.0,
							"rxInterval": 300.0,
							"multiplier": 3.0,
						},
					},
				},
			},
			validate: func(t *testing.T, cfg *megaport.VXCOrderVrouterPartnerConfig) {
				assert.Len(t, cfg.Interfaces, 1)
				iface := cfg.Interfaces[0]
				assert.Equal(t, 100, iface.VLAN)
				assert.Equal(t, []string{"10.0.0.1/30"}, iface.IpAddresses)
				assert.Len(t, iface.IpRoutes, 1)
				route := iface.IpRoutes[0]
				assert.Equal(t, "10.1.0.0/24", route.Prefix)
				assert.Equal(t, "10.0.0.2", route.NextHop)
				assert.Equal(t, "route1", route.Description)
				assert.Equal(t, 300, iface.Bfd.TxInterval)
				assert.Equal(t, 300, iface.Bfd.RxInterval)
				assert.Equal(t, 3, iface.Bfd.Multiplier)
			},
		},
		{
			name: "interfaces not array",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces":  "string",
			},
			expectedError: "interfaces must be an array",
		},
		{
			name: "interface not object",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces":  []interface{}{123},
			},
			expectedError: "must be an object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseVRouterConfig(tt.config)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestParseVRouterConfigIPsec(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]interface{}
		expectedError string
		validate      func(*testing.T, *megaport.VXCOrderVrouterPartnerConfig)
	}{
		{
			name: "ipSecTunnel interface with a single tunnel object",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"interfaceType": "ipSecTunnel",
						"ipSecTunnelOptions": map[string]interface{}{
							"sourceIpAddress":      "192.0.2.1",
							"destinationIpAddress": "198.51.100.1",
							"preSharedKey":         "secret-one",
							"passive":              false,
							"localId":              "local-1",
							"remoteId":             "remote-1",
							"phase1Lifetime":       28800.0,
							"phase2Lifetime":       3600.0,
						},
					},
				},
			},
			validate: func(t *testing.T, cfg *megaport.VXCOrderVrouterPartnerConfig) {
				assert.Len(t, cfg.Interfaces, 1)
				iface := cfg.Interfaces[0]
				assert.Equal(t, "ipSecTunnel", iface.InterfaceType)
				if !assert.NotNil(t, iface.IpSecTunnelOptions) {
					return
				}

				tunnel := iface.IpSecTunnelOptions
				assert.Equal(t, "192.0.2.1", tunnel.SourceIpAddress)
				assert.Equal(t, "198.51.100.1", tunnel.DestinationIpAddress)
				assert.Equal(t, "secret-one", tunnel.PreSharedKey)
				assert.Equal(t, "local-1", tunnel.LocalId)
				assert.Equal(t, "remote-1", tunnel.RemoteId)
				if assert.NotNil(t, tunnel.Passive) {
					assert.False(t, *tunnel.Passive)
				}
				if assert.NotNil(t, tunnel.Phase1Lifetime) {
					assert.Equal(t, 28800, *tunnel.Phase1Lifetime)
				}
				if assert.NotNil(t, tunnel.Phase2Lifetime) {
					assert.Equal(t, 3600, *tunnel.Phase2Lifetime)
				}
			},
		},
		{
			name: "multiple ipSecTunnel interfaces each with one tunnel object",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"interfaceType": "ipSecTunnel",
						"ipSecTunnelOptions": map[string]interface{}{
							"sourceIpAddress":      "192.0.2.1",
							"destinationIpAddress": "198.51.100.1",
							"preSharedKey":         "secret-one",
						},
					},
					map[string]interface{}{
						"interfaceType": "ipSecTunnel",
						"ipSecTunnelOptions": map[string]interface{}{
							"sourceIpAddress":      "192.0.2.2",
							"destinationIpAddress": "198.51.100.2",
							"preSharedKey":         "secret-two",
						},
					},
				},
			},
			validate: func(t *testing.T, cfg *megaport.VXCOrderVrouterPartnerConfig) {
				assert.Len(t, cfg.Interfaces, 2)
				if assert.NotNil(t, cfg.Interfaces[0].IpSecTunnelOptions) {
					assert.Equal(t, "192.0.2.1", cfg.Interfaces[0].IpSecTunnelOptions.SourceIpAddress)
					assert.Equal(t, "secret-one", cfg.Interfaces[0].IpSecTunnelOptions.PreSharedKey)
				}
				if assert.NotNil(t, cfg.Interfaces[1].IpSecTunnelOptions) {
					assert.Equal(t, "192.0.2.2", cfg.Interfaces[1].IpSecTunnelOptions.SourceIpAddress)
					assert.Equal(t, "secret-two", cfg.Interfaces[1].IpSecTunnelOptions.PreSharedKey)
				}
			},
		},
		{
			name: "ipSecTunnelOptions not an object",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"interfaceType":      "ipSecTunnel",
						"ipSecTunnelOptions": "nope",
					},
				},
			},
			expectedError: "ipSecTunnelOptions must be an object",
		},
		{
			name: "ipSecTunnelOptions is an array",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"interfaceType":      "ipSecTunnel",
						"ipSecTunnelOptions": []interface{}{map[string]interface{}{"sourceIpAddress": "192.0.2.1"}},
					},
				},
			},
			expectedError: "ipSecTunnelOptions must be an object",
		},
		{
			name: "phase1Lifetime wrong type",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"interfaceType": "ipSecTunnel",
						"ipSecTunnelOptions": map[string]interface{}{
							"sourceIpAddress":      "192.0.2.1",
							"destinationIpAddress": "198.51.100.1",
							"preSharedKey":         "secret",
							"phase1Lifetime":       "lots",
						},
					},
				},
			},
			expectedError: "phase1Lifetime must be a number",
		},
		{
			name: "sourceIpAddress wrong type",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"interfaceType":      "ipSecTunnel",
						"ipSecTunnelOptions": map[string]interface{}{"sourceIpAddress": 123},
					},
				},
			},
			expectedError: "sourceIpAddress must be a string",
		},
		{
			name: "passive wrong type",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"interfaceType": "ipSecTunnel",
						"ipSecTunnelOptions": map[string]interface{}{
							"sourceIpAddress":      "192.0.2.1",
							"destinationIpAddress": "198.51.100.1",
							"preSharedKey":         "secret",
							"passive":              "yes",
						},
					},
				},
			},
			expectedError: "passive must be a boolean",
		},
		{
			name: "phase2Lifetime wrong type",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"interfaceType": "ipSecTunnel",
						"ipSecTunnelOptions": map[string]interface{}{
							"sourceIpAddress":      "192.0.2.1",
							"destinationIpAddress": "198.51.100.1",
							"preSharedKey":         "secret",
							"phase2Lifetime":       "forever",
						},
					},
				},
			},
			expectedError: "phase2Lifetime must be a number",
		},
		{
			name: "interfaceType wrong type",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{"interfaceType": 123},
				},
			},
			expectedError: "interfaceType must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseVRouterConfig(tt.config)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}

	// Field-type error coverage: one entry per IPsec tunnel field. Each entry
	// only sets the target field wrong; all earlier fields are absent.
	ipsecFieldTypeErrors := []struct {
		field string
		value interface{}
		want  string
	}{
		{"destinationIpAddress", 123, "destinationIpAddress must be a string"},
		{"preSharedKey", 123, "preSharedKey must be a string"},
		{"localId", 123, "localId must be a string"},
		{"remoteId", 123, "remoteId must be a string"},
	}
	for _, tc := range ipsecFieldTypeErrors {
		tc := tc
		t.Run("IPsec tunnel field "+tc.field+" wrong type", func(t *testing.T) {
			cfg := map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"interfaceType":      "ipSecTunnel",
						"ipSecTunnelOptions": map[string]interface{}{tc.field: tc.value},
					},
				},
			}
			_, err := parseVRouterConfig(cfg)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestParseVRouterConfigBGP(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]interface{}
		expectedError string
		validate      func(*testing.T, *megaport.VXCOrderVrouterPartnerConfig)
	}{
		{
			name: "interface with fully populated BGP connection",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"vlan": 100.0,
						"bgpConnections": []interface{}{
							map[string]interface{}{
								"peerAsn":            65000.0,
								"localAsn":           64512.0,
								"localIpAddress":     "192.168.1.1",
								"peerIpAddress":      "192.168.1.2",
								"password":           "bgppassword",
								"shutdown":           false,
								"description":        "primary",
								"medIn":              100.0,
								"medOut":             200.0,
								"bfdEnabled":         true,
								"exportPolicy":       "permit",
								"permitExportTo":     []interface{}{"10.0.0.0/8", "172.16.0.0/12"},
								"denyExportTo":       []interface{}{"192.0.2.0/24"},
								"importWhitelist":    101.0,
								"importBlacklist":    102.0,
								"exportWhitelist":    201.0,
								"exportBlacklist":    202.0,
								"asPathPrependCount": 3.0,
								"peerType":           "NON_CLOUD",
								"asOverride":         true,
							},
						},
					},
				},
			},
			validate: func(t *testing.T, cfg *megaport.VXCOrderVrouterPartnerConfig) {
				assert.Len(t, cfg.Interfaces, 1)
				// A regression to silently dropping bgpConnections fails here.
				assert.Len(t, cfg.Interfaces[0].BgpConnections, 1)
				conn := cfg.Interfaces[0].BgpConnections[0]
				assert.Equal(t, 65000, conn.PeerAsn)
				if assert.NotNil(t, conn.LocalAsn) {
					assert.Equal(t, 64512, *conn.LocalAsn)
				}
				assert.Equal(t, "192.168.1.1", conn.LocalIpAddress)
				assert.Equal(t, "192.168.1.2", conn.PeerIpAddress)
				assert.Equal(t, "bgppassword", conn.Password)
				assert.False(t, conn.Shutdown)
				assert.Equal(t, "primary", conn.Description)
				assert.Equal(t, 100, conn.MedIn)
				assert.Equal(t, 200, conn.MedOut)
				assert.True(t, conn.BfdEnabled)
				assert.Equal(t, "permit", conn.ExportPolicy)
				assert.Equal(t, []string{"10.0.0.0/8", "172.16.0.0/12"}, conn.PermitExportTo)
				assert.Equal(t, []string{"192.0.2.0/24"}, conn.DenyExportTo)
				assert.Equal(t, 101, conn.ImportWhitelist)
				assert.Equal(t, 102, conn.ImportBlacklist)
				assert.Equal(t, 201, conn.ExportWhitelist)
				assert.Equal(t, 202, conn.ExportBlacklist)
				assert.Equal(t, 3, conn.AsPathPrependCount)
				assert.Equal(t, "NON_CLOUD", conn.PeerType)
				if assert.NotNil(t, conn.AsOverride) {
					assert.True(t, *conn.AsOverride)
				}
			},
		},
		{
			name: "asOverride false sets pointer to false",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"bgpConnections": []interface{}{
							map[string]interface{}{
								"peerAsn":    65000.0,
								"asOverride": false,
							},
						},
					},
				},
			},
			validate: func(t *testing.T, cfg *megaport.VXCOrderVrouterPartnerConfig) {
				conn := cfg.Interfaces[0].BgpConnections[0]
				if assert.NotNil(t, conn.AsOverride) {
					assert.False(t, *conn.AsOverride)
				}
			},
		},
		{
			name: "minimal BGP connection leaves optional pointer nil",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"bgpConnections": []interface{}{
							map[string]interface{}{
								"peerAsn":        65000.0,
								"localIpAddress": "192.168.1.1",
								"peerIpAddress":  "192.168.1.2",
							},
						},
					},
				},
			},
			validate: func(t *testing.T, cfg *megaport.VXCOrderVrouterPartnerConfig) {
				conn := cfg.Interfaces[0].BgpConnections[0]
				assert.Equal(t, 65000, conn.PeerAsn)
				assert.Nil(t, conn.LocalAsn)
				assert.Nil(t, conn.PermitExportTo)
				assert.Nil(t, conn.DenyExportTo)
				assert.Nil(t, conn.AsOverride)
			},
		},
		{
			name: "bgpConnections not an array",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{"bgpConnections": "nope"},
				},
			},
			expectedError: "bgpConnections must be an array",
		},
		{
			name: "BGP connection not an object",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{"bgpConnections": []interface{}{123}},
				},
			},
			expectedError: "must be an object",
		},
		{
			name: "peerAsn wrong type",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"bgpConnections": []interface{}{
							map[string]interface{}{"peerAsn": "lots"},
						},
					},
				},
			},
			expectedError: "peerAsn must be a number",
		},
		{
			name: "permitExportTo not an array",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"bgpConnections": []interface{}{
							map[string]interface{}{"permitExportTo": "nope"},
						},
					},
				},
			},
			expectedError: "permitExportTo must be an array",
		},
		{
			name: "localIpAddress wrong type triggers strField error path",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"bgpConnections": []interface{}{
							map[string]interface{}{"localIpAddress": 123},
						},
					},
				},
			},
			expectedError: "localIpAddress must be a string",
		},
		{
			name: "shutdown wrong type triggers boolField error path",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"bgpConnections": []interface{}{
							map[string]interface{}{"shutdown": "yes"},
						},
					},
				},
			},
			expectedError: "shutdown must be a boolean",
		},
		{
			name: "permitExportTo element not a string",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"bgpConnections": []interface{}{
							map[string]interface{}{"permitExportTo": []interface{}{123}},
						},
					},
				},
			},
			expectedError: "permitExportTo[0] must be a string",
		},
		{
			name: "denyExportTo not an array",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"bgpConnections": []interface{}{
							map[string]interface{}{"denyExportTo": "nope"},
						},
					},
				},
			},
			expectedError: "denyExportTo must be an array",
		},
		{
			name: "importWhitelist wrong type",
			config: map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"bgpConnections": []interface{}{
							map[string]interface{}{"importWhitelist": "nope"},
						},
					},
				},
			},
			expectedError: "importWhitelist must be a number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseVRouterConfig(tt.config)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}

	// Field-type error coverage: one entry per BGP field that has its own call-site
	// error branch. Each entry only sets the target field (wrong type) so all
	// preceding fields are absent and do not trigger an earlier return.
	bgpFieldTypeErrors := []struct {
		field string
		value interface{}
		want  string
	}{
		{"localAsn", "wrong", "localAsn must be a number"},
		{"peerIpAddress", 123, "peerIpAddress must be a string"},
		{"password", 123, "password must be a string"},
		{"description", 123, "description must be a string"},
		{"medIn", "lots", "medIn must be a number"},
		{"medOut", "lots", "medOut must be a number"},
		{"bfdEnabled", "yes", "bfdEnabled must be a boolean"},
		{"exportPolicy", 123, "exportPolicy must be a string"},
		{"importBlacklist", "nope", "importBlacklist must be a number"},
		{"exportWhitelist", "nope", "exportWhitelist must be a number"},
		{"exportBlacklist", "nope", "exportBlacklist must be a number"},
		{"asPathPrependCount", "lots", "asPathPrependCount must be a number"},
		{"peerType", 123, "peerType must be a string"},
		{"asOverride", "yes", "asOverride must be a boolean"},
	}
	for _, tc := range bgpFieldTypeErrors {
		tc := tc
		t.Run("BGP field "+tc.field+" wrong type", func(t *testing.T) {
			cfg := map[string]interface{}{
				"connectType": "VROUTER",
				"interfaces": []interface{}{
					map[string]interface{}{
						"bgpConnections": []interface{}{
							map[string]interface{}{tc.field: tc.value},
						},
					},
				},
			}
			_, err := parseVRouterConfig(cfg)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestParseVXCEndpointConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]interface{}
		expectedError string
		validate      func(*testing.T, megaport.VXCOrderEndpointConfiguration)
	}{
		{
			name: "basic fields",
			config: map[string]interface{}{
				"productUID":    "uid-1",
				"vlan":          100.0,
				"diversityZone": "blue",
			},
			validate: func(t *testing.T, cfg megaport.VXCOrderEndpointConfiguration) {
				assert.Equal(t, "uid-1", cfg.ProductUID)
				assert.Equal(t, 100, cfg.VLAN)
				assert.Equal(t, "blue", cfg.DiversityZone)
			},
		},
		{
			name: "with partner config",
			config: map[string]interface{}{
				"productUID": "uid-1",
				"partnerConfig": map[string]interface{}{
					"connectType": "GOOGLE",
					"pairingKey":  "pk-1",
				},
			},
			validate: func(t *testing.T, cfg megaport.VXCOrderEndpointConfiguration) {
				assert.Equal(t, "uid-1", cfg.ProductUID)
				_, ok := cfg.PartnerConfig.(*megaport.VXCPartnerConfigGoogle)
				assert.True(t, ok, "expected *megaport.VXCPartnerConfigGoogle")
			},
		},
		{
			name: "with MVE config",
			config: map[string]interface{}{
				"innerVlan": 200.0,
				"vNicIndex": 1.0,
			},
			validate: func(t *testing.T, cfg megaport.VXCOrderEndpointConfiguration) {
				assert.NotNil(t, cfg.VXCOrderMVEConfig)
				assert.Equal(t, 200, cfg.InnerVLAN)
				assert.Equal(t, 1, cfg.NetworkInterfaceIndex)
			},
		},
		{
			name:   "empty map",
			config: map[string]interface{}{},
			validate: func(t *testing.T, cfg megaport.VXCOrderEndpointConfiguration) {
				assert.Equal(t, "", cfg.ProductUID)
				assert.Equal(t, 0, cfg.VLAN)
				assert.Nil(t, cfg.VXCOrderMVEConfig)
			},
		},
		{
			name: "invalid partner config",
			config: map[string]interface{}{
				"partnerConfig": map[string]interface{}{
					"ownerAccount": "123",
				},
			},
			expectedError: "partner config",
		},
		{
			name:          "vlan wrong type rejected",
			config:        map[string]interface{}{"vlan": "one hundred"},
			expectedError: "vlan must be a number",
		},
		{
			name:          "productUID wrong type rejected",
			config:        map[string]interface{}{"productUID": 123.0},
			expectedError: "productUID must be a string",
		},
		{
			name:          "partnerConfig wrong type rejected",
			config:        map[string]interface{}{"partnerConfig": "not-an-object"},
			expectedError: "partnerConfig must be an object",
		},
		{
			name:          "innerVlan wrong type rejected",
			config:        map[string]interface{}{"innerVlan": "x"},
			expectedError: "innerVlan must be a number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseVXCEndpointConfig(tt.config, "test")
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestBuildVXCRequestFromJSON(t *testing.T) {
	tests := []struct {
		name          string
		jsonStr       string
		jsonFilePath  string
		expectedError string
		validate      func(*testing.T, *megaport.BuyVXCRequest)
	}{
		{
			name:    "valid full JSON",
			jsonStr: `{"portUid":"port-1","vxcName":"Test VXC","rateLimit":1000,"term":12,"promoCode":"PROMO","costCentre":"CC","aEndConfiguration":{"vlan":100},"bEndConfiguration":{"productUID":"port-2","vlan":200}}`,
			validate: func(t *testing.T, req *megaport.BuyVXCRequest) {
				assert.Equal(t, "port-1", req.PortUID)
				assert.Equal(t, "Test VXC", req.VXCName)
				assert.Equal(t, 1000, req.RateLimit)
				assert.Equal(t, 12, req.Term)
				assert.Equal(t, "PROMO", req.PromoCode)
				assert.Equal(t, "CC", req.CostCentre)
				assert.Equal(t, 100, req.AEndConfiguration.VLAN)
				assert.Equal(t, "port-2", req.BEndConfiguration.ProductUID)
				assert.Equal(t, 200, req.BEndConfiguration.VLAN)
			},
		},
		{
			name:    "with resource tags",
			jsonStr: `{"portUid":"port-1","vxcName":"Test VXC","rateLimit":1000,"term":12,"resourceTags":{"env":"prod"},"bEndConfiguration":{"productUID":"port-2"}}`,
			validate: func(t *testing.T, req *megaport.BuyVXCRequest) {
				assert.NotNil(t, req.ResourceTags)
				assert.Equal(t, "prod", req.ResourceTags["env"])
			},
		},
		{
			name:          "missing portUid",
			jsonStr:       `{"vxcName":"Test VXC","rateLimit":1000,"term":12}`,
			expectedError: "portUid",
		},
		{
			name:          "invalid JSON",
			jsonStr:       `{bad}`,
			expectedError: "failed to parse JSON",
		},
		{
			name:          "empty inputs",
			jsonStr:       "",
			jsonFilePath:  "",
			expectedError: "either json or json-file must be provided",
		},
		{
			name:          "fractional rateLimit rejected",
			jsonStr:       `{"portUid":"port-1","rateLimit":10.9,"term":12}`,
			expectedError: "rateLimit must be a whole number",
		},
		{
			name:          "fractional term rejected",
			jsonStr:       `{"portUid":"port-1","rateLimit":1000,"term":10.9}`,
			expectedError: "term must be a whole number",
		},
		{
			name:          "rateLimit wrong type rejected",
			jsonStr:       `{"portUid":"port-1","rateLimit":"fast","term":12}`,
			expectedError: "rateLimit must be a number",
		},
		{
			name:          "term wrong type rejected",
			jsonStr:       `{"portUid":"port-1","rateLimit":1000,"term":"yearly"}`,
			expectedError: "term must be a number",
		},
		{
			name:          "vxcName wrong type rejected",
			jsonStr:       `{"portUid":"port-1","vxcName":123,"rateLimit":1000,"term":12}`,
			expectedError: "vxcName must be a string",
		},
		{
			name:          "costCentre wrong type rejected",
			jsonStr:       `{"portUid":"port-1","rateLimit":1000,"term":12,"costCentre":true}`,
			expectedError: "costCentre must be a string",
		},
		{
			name:          "shutdown wrong type rejected",
			jsonStr:       `{"portUid":"port-1","rateLimit":1000,"term":12,"shutdown":"yes"}`,
			expectedError: "shutdown must be a boolean",
		},
		{
			name:          "portUid wrong type rejected",
			jsonStr:       `{"portUid":123,"rateLimit":1000,"term":12}`,
			expectedError: "portUid must be a string",
		},
		{
			name:          "resourceTags value wrong type rejected",
			jsonStr:       `{"portUid":"port-1","rateLimit":1000,"term":12,"resourceTags":{"env":123}}`,
			expectedError: "resourceTags value for key \"env\" must be a string",
		},
		{
			name:          "aEndConfiguration wrong type rejected",
			jsonStr:       `{"portUid":"port-1","rateLimit":1000,"term":12,"aEndConfiguration":"not-an-object"}`,
			expectedError: "aEndConfiguration must be an object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildVXCRequestFromJSON(tt.jsonStr, tt.jsonFilePath)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)

				var cliErr *exitcodes.CLIError
				require.True(t, errors.As(err, &cliErr), "expected a *exitcodes.CLIError, got %T: %v", err, err)
				assert.Equal(t, exitcodes.Usage, cliErr.Code)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestBuildVXCRequestFromJSON_RejectsEmptyTagKey(t *testing.T) {
	const payload = `{"portUid":"port-1","vxcName":"Test VXC","rateLimit":1000,"term":12,"resourceTags":{"":"x"},"bEndConfiguration":{"productUID":"port-2"}}`

	t.Run("via json", func(t *testing.T) {
		_, err := buildVXCRequestFromJSON(payload, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tag key must not be empty")

		var cliErr *exitcodes.CLIError
		require.True(t, errors.As(err, &cliErr))
		assert.Equal(t, exitcodes.Usage, cliErr.Code)
	})

	t.Run("via json-file", func(t *testing.T) {
		tmp, err := os.CreateTemp("", "vxc-emptytag-*.json")
		require.NoError(t, err)
		defer os.Remove(tmp.Name())
		_, err = tmp.WriteString(payload)
		require.NoError(t, err)
		require.NoError(t, tmp.Close())

		_, err = buildVXCRequestFromJSON("", tmp.Name())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tag key must not be empty")

		var cliErr *exitcodes.CLIError
		require.True(t, errors.As(err, &cliErr))
		assert.Equal(t, exitcodes.Usage, cliErr.Code)
	})
}

func TestBuildVXCRequestFromJSON_EndpointConfigMalformedField(t *testing.T) {
	// Exercises parseVXCEndpointConfig's own error wrapping, distinct from the
	// top-level field checks in buildVXCRequestFromJSON.
	payload := `{"portUid":"port-1","rateLimit":1000,"term":12,"aEndConfiguration":{"vlan":"not-a-number"}}`

	_, err := buildVXCRequestFromJSON(payload, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "A-End")

	var cliErr *exitcodes.CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}

func TestBuildUpdateVXCRequestFromJSON_PartnerConfigs(t *testing.T) {
	tests := []struct {
		name          string
		json          string
		expectedError string
		validate      func(*testing.T, *megaport.UpdateVXCRequest)
	}{
		{
			name: "valid VRouter A-End partner config",
			json: `{"aEndPartnerConfig":{"connectType":"VROUTER","interfaces":[{"vlan":100,"ipAddresses":["10.0.0.1/30"]}]}}`,
			validate: func(t *testing.T, req *megaport.UpdateVXCRequest) {
				assert.NotNil(t, req.AEndPartnerConfig)
				vrouterCfg, ok := req.AEndPartnerConfig.(*megaport.VXCOrderVrouterPartnerConfig)
				assert.True(t, ok, "expected *megaport.VXCOrderVrouterPartnerConfig")
				assert.Len(t, vrouterCfg.Interfaces, 1)
			},
		},
		{
			name:          "non-VRouter A-End partner config",
			json:          `{"aEndPartnerConfig":{"connectType":"AWS","ownerAccount":"123"}}`,
			expectedError: "only VRouter",
		},
		{
			name:          "non-VRouter B-End partner config",
			json:          `{"bEndPartnerConfig":{"connectType":"IBM","accountID":"123"}}`,
			expectedError: "only VRouter",
		},
		{
			name: "valid VRouter A-End IPsec tunnel config",
			json: `{"aEndPartnerConfig":{"connectType":"VROUTER","interfaces":[{"interfaceType":"ipSecTunnel","ipSecTunnelOptions":{"sourceIpAddress":"192.0.2.1","destinationIpAddress":"198.51.100.1","preSharedKey":"topsecret"}}]}}`,
			validate: func(t *testing.T, req *megaport.UpdateVXCRequest) {
				vrouterCfg, ok := req.AEndPartnerConfig.(*megaport.VXCOrderVrouterPartnerConfig)
				assert.True(t, ok, "expected *megaport.VXCOrderVrouterPartnerConfig")
				assert.Len(t, vrouterCfg.Interfaces, 1)
				assert.Equal(t, megaport.InterfaceTypeIPSecTunnel, vrouterCfg.Interfaces[0].InterfaceType)
				assert.NotNil(t, vrouterCfg.Interfaces[0].IpSecTunnelOptions)
			},
		},
		{
			name:          "IPsec tunnel options without ipSecTunnel interface type rejected",
			json:          `{"aEndPartnerConfig":{"connectType":"VROUTER","interfaces":[{"ipSecTunnelOptions":{"sourceIpAddress":"192.0.2.1","destinationIpAddress":"198.51.100.1","preSharedKey":"topsecret"}}]}}`,
			expectedError: "requires interface type",
		},
		{
			name:          "IPsec tunnel missing pre-shared key rejected",
			json:          `{"aEndPartnerConfig":{"connectType":"VROUTER","interfaces":[{"interfaceType":"ipSecTunnel","ipSecTunnelOptions":{"sourceIpAddress":"192.0.2.1","destinationIpAddress":"198.51.100.1"}}]}}`,
			expectedError: "pre-shared key",
		},
		{
			name: "valid VRouter B-End partner config",
			json: `{"bEndPartnerConfig":{"connectType":"VROUTER","interfaces":[{"vlan":200,"ipAddresses":["10.0.1.1/30"]}]}}`,
			validate: func(t *testing.T, req *megaport.UpdateVXCRequest) {
				assert.Nil(t, req.AEndPartnerConfig)
				assert.NotNil(t, req.BEndPartnerConfig)
				vrouterCfg, ok := req.BEndPartnerConfig.(*megaport.VXCOrderVrouterPartnerConfig)
				assert.True(t, ok)
				assert.Len(t, vrouterCfg.Interfaces, 1)
			},
		},
		{
			name:          "B-End IPsec tunnel missing pre-shared key rejected",
			json:          `{"bEndPartnerConfig":{"connectType":"VROUTER","interfaces":[{"interfaceType":"ipSecTunnel","ipSecTunnelOptions":{"sourceIpAddress":"192.0.2.1","destinationIpAddress":"198.51.100.1"}}]}}`,
			expectedError: "pre-shared key",
		},
		{
			name:          "A-End VROUTER with unparseable interfaces rejected",
			json:          `{"aEndPartnerConfig":{"connectType":"VROUTER","interfaces":"not-an-array"}}`,
			expectedError: "failed to parse A-End",
		},
		{
			name:          "B-End VROUTER with unparseable interfaces rejected",
			json:          `{"bEndPartnerConfig":{"connectType":"VROUTER","interfaces":"not-an-array"}}`,
			expectedError: "failed to parse B-End",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildUpdateVXCRequestFromJSON(tt.json, "")
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestBuildUpdateVXCRequestFromJSON_FractionalRateLimit(t *testing.T) {
	_, err := buildUpdateVXCRequestFromJSON(`{"rateLimit":10.9}`, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rateLimit must be a whole number")
}

func TestBuildUpdateVXCRequestFromJSON_WholeRateLimit(t *testing.T) {
	req, err := buildUpdateVXCRequestFromJSON(`{"rateLimit":2000}`, "")
	require.NoError(t, err)
	require.NotNil(t, req.RateLimit)
	assert.Equal(t, 2000, *req.RateLimit)
}

func TestBuildUpdateVXCRequestFromJSON_TypoKeyIsRejected(t *testing.T) {
	// "rateLimt" is not a recognized field, so it must not be silently
	// dropped into an empty, successful update.
	_, err := buildUpdateVXCRequestFromJSON(`{"rateLimt": 500}`, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one field must be updated")
	assert.Contains(t, err.Error(), "rateLimt")
	var cliErr *exitcodes.CLIError
	require.ErrorAs(t, err, &cliErr)
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}

func TestBuildUpdateVXCRequestFromJSON_TypoKeyAlongsideValidFieldIsRejected(t *testing.T) {
	// A valid field must not let a sibling typo key slip through and get
	// silently dropped just because fieldSet is already true.
	_, err := buildUpdateVXCRequestFromJSON(`{"rateLimit":500,"rateLimt":999}`, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unrecognized keys")
	assert.Contains(t, err.Error(), "rateLimt")
	var cliErr *exitcodes.CLIError
	require.ErrorAs(t, err, &cliErr)
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}

func TestBuildUpdateVXCRequestFromJSON_NestedTypoKeyAlongsideValidFieldIsRejected(t *testing.T) {
	_, err := buildUpdateVXCRequestFromJSON(`{"name":"updated-vxc","aEndConfiguration":{"vlan":100,"vln":200}}`, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unrecognized keys")
	assert.Contains(t, err.Error(), "aEndConfiguration.vln")
	var cliErr *exitcodes.CLIError
	require.ErrorAs(t, err, &cliErr)
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}

func TestBuildUpdateVXCRequestFromJSON_WrongTypedValueIsRejected(t *testing.T) {
	_, err := buildUpdateVXCRequestFromJSON(`{"rateLimit": "500"}`, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rateLimit must be a number")
	var cliErr *exitcodes.CLIError
	require.ErrorAs(t, err, &cliErr)
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}

func TestBuildUpdateVXCRequestFromJSON_EmptyObjectIsRejected(t *testing.T) {
	_, err := buildUpdateVXCRequestFromJSON(`{}`, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one field must be updated")
	var cliErr *exitcodes.CLIError
	require.ErrorAs(t, err, &cliErr)
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}

func TestBuildUpdateVXCRequestFromJSON_ValidMultiFieldPayload(t *testing.T) {
	req, err := buildUpdateVXCRequestFromJSON(`{"name":"updated-vxc","rateLimit":1000,"costCentre":"eng","shutdown":true}`, "")
	require.NoError(t, err)
	require.NotNil(t, req.Name)
	assert.Equal(t, "updated-vxc", *req.Name)
	require.NotNil(t, req.RateLimit)
	assert.Equal(t, 1000, *req.RateLimit)
	require.NotNil(t, req.CostCentre)
	assert.Equal(t, "eng", *req.CostCentre)
	require.NotNil(t, req.Shutdown)
	assert.True(t, *req.Shutdown)
}

func TestBuildUpdateVXCRequestFromJSON_FlatVLANFallback(t *testing.T) {
	req, err := buildUpdateVXCRequestFromJSON(`{"aEndVlan":100,"bEndVlan":200}`, "")
	require.NoError(t, err)
	require.NotNil(t, req.AEndVLAN)
	assert.Equal(t, 100, *req.AEndVLAN)
	require.NotNil(t, req.BEndVLAN)
	assert.Equal(t, 200, *req.BEndVLAN)
}

func TestBuildUpdateVXCRequestFromJSON_WrongTypedPartnerConfigIsRejected(t *testing.T) {
	_, err := buildUpdateVXCRequestFromJSON(`{"aEndPartnerConfig":"not-an-object"}`, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aEndPartnerConfig must be an object")
	var cliErr *exitcodes.CLIError
	require.ErrorAs(t, err, &cliErr)
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}

func TestBuildUpdateVXCRequestFromJSON_NestedWrongTypedValueIsRejected(t *testing.T) {
	_, err := buildUpdateVXCRequestFromJSON(`{"aEndConfiguration":{"vlan":"not-a-number"}}`, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aEndConfiguration.vlan")
}

func TestBuildUpdateVXCRequestFromJSON_NestedTypoKeyIsRejected(t *testing.T) {
	// "vln" inside aEndConfiguration is not recognized, so the object is
	// present but contributes no field, and no flat aEndVlan fallback
	// applies once aEndConfiguration itself is present.
	_, err := buildUpdateVXCRequestFromJSON(`{"aEndConfiguration":{"vln":100}}`, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one field must be updated")
	assert.Contains(t, err.Error(), "aEndConfiguration.vln")
}

func TestBuildUpdateVXCRequestFromJSON_NestedBEndTypoKeyIsRejected(t *testing.T) {
	_, err := buildUpdateVXCRequestFromJSON(`{"bEndConfiguration":{"vln":100}}`, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one field must be updated")
	assert.Contains(t, err.Error(), "bEndConfiguration.vln")
}

func TestBuildUpdateVXCRequestFromJSON_NoInputProvided(t *testing.T) {
	_, err := buildUpdateVXCRequestFromJSON("", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "either json or json-file must be provided")
}

func TestBuildUpdateVXCRequestFromJSON_MissingJSONFile(t *testing.T) {
	_, err := buildUpdateVXCRequestFromJSON("", "/nonexistent/path/to/update.json")
	require.Error(t, err)
}

func TestBuildUpdateVXCRequestFromJSON_MalformedJSONIsRejected(t *testing.T) {
	_, err := buildUpdateVXCRequestFromJSON(`{"name":`, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON")
}

func TestBuildUpdateVXCRequestFromJSON_FieldCoverage(t *testing.T) {
	tests := []struct {
		name          string
		json          string
		expectedError string
		validate      func(*testing.T, *megaport.UpdateVXCRequest)
	}{
		{
			name:          "negative rate limit rejected",
			json:          `{"rateLimit":-5}`,
			expectedError: "rateLimit must be greater than or equal to 0",
		},
		{
			name:          "wrong-typed term rejected",
			json:          `{"term":"12"}`,
			expectedError: "term must be a number",
		},
		{
			name:          "fractional term rejected",
			json:          `{"term":12.5}`,
			expectedError: "term must be a whole number",
		},
		{
			name:          "invalid term rejected",
			json:          `{"term":5}`,
			expectedError: "contract term",
		},
		{
			name: "valid non-zero term",
			json: `{"term":12}`,
			validate: func(t *testing.T, req *megaport.UpdateVXCRequest) {
				require.NotNil(t, req.Term)
				assert.Equal(t, 12, *req.Term)
			},
		},
		{
			name:          "zero term rejected",
			json:          `{"term":0}`,
			expectedError: "contract term",
		},
		{
			name:          "wrong-typed cost centre rejected",
			json:          `{"costCentre":123}`,
			expectedError: "costCentre must be a string",
		},
		{
			name:          "wrong-typed shutdown rejected",
			json:          `{"shutdown":"yes"}`,
			expectedError: "shutdown must be a boolean",
		},
		{
			name:          "wrong-typed aEndConfiguration rejected",
			json:          `{"aEndConfiguration":"not-an-object"}`,
			expectedError: "aEndConfiguration must be an object",
		},
		{
			name:          "fractional aEndConfiguration.vlan rejected",
			json:          `{"aEndConfiguration":{"vlan":100.5}}`,
			expectedError: "aEndConfiguration.vlan must be a whole number",
		},
		{
			name:          "out-of-range aEndConfiguration.vlan rejected",
			json:          `{"aEndConfiguration":{"vlan":5000}}`,
			expectedError: "aEndConfiguration.vlan",
		},
		{
			name: "valid aEndConfiguration.vlan",
			json: `{"aEndConfiguration":{"vlan":100}}`,
			validate: func(t *testing.T, req *megaport.UpdateVXCRequest) {
				require.NotNil(t, req.AEndVLAN)
				assert.Equal(t, 100, *req.AEndVLAN)
			},
		},
		{
			name:          "wrong-typed bEndConfiguration.vlan rejected",
			json:          `{"bEndConfiguration":{"vlan":"x"}}`,
			expectedError: "bEndConfiguration.vlan: vlan must be a number",
		},
		{
			name:          "wrong-typed aEndPartnerConfig.connectType rejected",
			json:          `{"aEndPartnerConfig":{"connectType":123}}`,
			expectedError: "aEndPartnerConfig.connectType",
		},
		{
			name:          "missing aEndPartnerConfig.connectType rejected",
			json:          `{"aEndPartnerConfig":{"interfaces":[]}}`,
			expectedError: "aEndPartnerConfig.connectType is required",
		},
		{
			name:          "missing bEndPartnerConfig.connectType rejected",
			json:          `{"bEndPartnerConfig":{"interfaces":[]}}`,
			expectedError: "bEndPartnerConfig.connectType is required",
		},
		{
			name:          "wrong-typed flat aEndVlan rejected",
			json:          `{"aEndVlan":"x"}`,
			expectedError: "aEndVlan must be a number",
		},
		{
			name:          "fractional flat aEndVlan rejected",
			json:          `{"aEndVlan":100.5}`,
			expectedError: "aEndVlan must be a whole number",
		},
		{
			name:          "out-of-range flat aEndVlan rejected",
			json:          `{"aEndVlan":5000}`,
			expectedError: "aEndVlan",
		},
		{
			name:          "wrong-typed bEndConfiguration rejected",
			json:          `{"bEndConfiguration":"not-an-object"}`,
			expectedError: "bEndConfiguration must be an object",
		},
		{
			name:          "fractional bEndConfiguration.vlan rejected",
			json:          `{"bEndConfiguration":{"vlan":200.5}}`,
			expectedError: "bEndConfiguration.vlan must be a whole number",
		},
		{
			name:          "out-of-range bEndConfiguration.vlan rejected",
			json:          `{"bEndConfiguration":{"vlan":5000}}`,
			expectedError: "bEndConfiguration.vlan",
		},
		{
			name: "valid bEndConfiguration.vlan",
			json: `{"bEndConfiguration":{"vlan":200}}`,
			validate: func(t *testing.T, req *megaport.UpdateVXCRequest) {
				require.NotNil(t, req.BEndVLAN)
				assert.Equal(t, 200, *req.BEndVLAN)
			},
		},
		{
			name:          "wrong-typed flat bEndVlan rejected",
			json:          `{"bEndVlan":"x"}`,
			expectedError: "bEndVlan must be a number",
		},
		{
			name:          "fractional flat bEndVlan rejected",
			json:          `{"bEndVlan":200.5}`,
			expectedError: "bEndVlan must be a whole number",
		},
		{
			name:          "out-of-range flat bEndVlan rejected",
			json:          `{"bEndVlan":5000}`,
			expectedError: "bEndVlan",
		},
		{
			name:          "wrong-typed name rejected",
			json:          `{"name":123}`,
			expectedError: "name must be a string",
		},
		{
			name: "vxcName used when name absent",
			json: `{"vxcName":"foo"}`,
			validate: func(t *testing.T, req *megaport.UpdateVXCRequest) {
				require.NotNil(t, req.Name)
				assert.Equal(t, "foo", *req.Name)
			},
		},
		{
			name:          "wrong-typed vxcName rejected",
			json:          `{"vxcName":123}`,
			expectedError: "vxcName must be a string",
		},
		{
			name:          "wrong-typed aEndInnerVlan rejected",
			json:          `{"aEndInnerVlan":"x"}`,
			expectedError: "aEndInnerVlan must be a number",
		},
		{
			name:          "fractional aEndInnerVlan rejected",
			json:          `{"aEndInnerVlan":100.5}`,
			expectedError: "aEndInnerVlan must be a whole number",
		},
		{
			name:          "out-of-range aEndInnerVlan rejected",
			json:          `{"aEndInnerVlan":5000}`,
			expectedError: "invalid aEndInnerVlan",
		},
		{
			name: "valid aEndInnerVlan",
			json: `{"aEndInnerVlan":100}`,
			validate: func(t *testing.T, req *megaport.UpdateVXCRequest) {
				require.NotNil(t, req.AEndInnerVLAN)
				assert.Equal(t, 100, *req.AEndInnerVLAN)
			},
		},
		{
			name:          "wrong-typed bEndInnerVlan rejected",
			json:          `{"bEndInnerVlan":"x"}`,
			expectedError: "bEndInnerVlan must be a number",
		},
		{
			name:          "fractional bEndInnerVlan rejected",
			json:          `{"bEndInnerVlan":200.5}`,
			expectedError: "bEndInnerVlan must be a whole number",
		},
		{
			name:          "out-of-range bEndInnerVlan rejected",
			json:          `{"bEndInnerVlan":5000}`,
			expectedError: "invalid bEndInnerVlan",
		},
		{
			name: "valid bEndInnerVlan",
			json: `{"bEndInnerVlan":200}`,
			validate: func(t *testing.T, req *megaport.UpdateVXCRequest) {
				require.NotNil(t, req.BEndInnerVLAN)
				assert.Equal(t, 200, *req.BEndInnerVLAN)
			},
		},
		{
			name:          "wrong-typed aEndUid rejected",
			json:          `{"aEndUid":123}`,
			expectedError: "aEndUid must be a string",
		},
		{
			name: "valid aEndUid",
			json: `{"aEndUid":"a-end-uid"}`,
			validate: func(t *testing.T, req *megaport.UpdateVXCRequest) {
				require.NotNil(t, req.AEndProductUID)
				assert.Equal(t, "a-end-uid", *req.AEndProductUID)
			},
		},
		{
			name:          "wrong-typed bEndUid rejected",
			json:          `{"bEndUid":123}`,
			expectedError: "bEndUid must be a string",
		},
		{
			name: "valid bEndUid",
			json: `{"bEndUid":"b-end-uid"}`,
			validate: func(t *testing.T, req *megaport.UpdateVXCRequest) {
				require.NotNil(t, req.BEndProductUID)
				assert.Equal(t, "b-end-uid", *req.BEndProductUID)
			},
		},
		{
			name:          "wrong-typed bEndPartnerConfig rejected",
			json:          `{"bEndPartnerConfig":"not-an-object"}`,
			expectedError: "bEndPartnerConfig must be an object",
		},
		{
			name:          "wrong-typed bEndPartnerConfig.connectType rejected",
			json:          `{"bEndPartnerConfig":{"connectType":123}}`,
			expectedError: "bEndPartnerConfig.connectType",
		},
		{
			name:          "wrong-typed isApproved rejected",
			json:          `{"isApproved":"yes"}`,
			expectedError: "isApproved must be a boolean",
		},
		{
			name: "valid isApproved",
			json: `{"isApproved":true}`,
			validate: func(t *testing.T, req *megaport.UpdateVXCRequest) {
				require.NotNil(t, req.IsApproved)
				assert.True(t, *req.IsApproved)
			},
		},
		{
			name:          "wrong-typed aVnicIndex rejected",
			json:          `{"aVnicIndex":"x"}`,
			expectedError: "aVnicIndex must be a number",
		},
		{
			name:          "fractional aVnicIndex rejected",
			json:          `{"aVnicIndex":1.5}`,
			expectedError: "aVnicIndex must be a whole number",
		},
		{
			name:          "negative aVnicIndex rejected",
			json:          `{"aVnicIndex":-1}`,
			expectedError: "invalid aVnicIndex",
		},
		{
			name: "valid aVnicIndex",
			json: `{"aVnicIndex":1}`,
			validate: func(t *testing.T, req *megaport.UpdateVXCRequest) {
				require.NotNil(t, req.AVnicIndex)
				assert.Equal(t, 1, *req.AVnicIndex)
			},
		},
		{
			name:          "wrong-typed bVnicIndex rejected",
			json:          `{"bVnicIndex":"x"}`,
			expectedError: "bVnicIndex must be a number",
		},
		{
			name:          "fractional bVnicIndex rejected",
			json:          `{"bVnicIndex":1.5}`,
			expectedError: "bVnicIndex must be a whole number",
		},
		{
			name:          "negative bVnicIndex rejected",
			json:          `{"bVnicIndex":-1}`,
			expectedError: "invalid bVnicIndex",
		},
		{
			name: "valid bVnicIndex",
			json: `{"bVnicIndex":1}`,
			validate: func(t *testing.T, req *megaport.UpdateVXCRequest) {
				require.NotNil(t, req.BVnicIndex)
				assert.Equal(t, 1, *req.BVnicIndex)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildUpdateVXCRequestFromJSON(tt.json, "")
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				var cliErr *exitcodes.CLIError
				require.ErrorAs(t, err, &cliErr)
				assert.Equal(t, exitcodes.Usage, cliErr.Code)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestBuildUpdateVXCRequestFromJSON_Term(t *testing.T) {
	t.Run("term 0 is rejected", func(t *testing.T) {
		_, err := buildUpdateVXCRequestFromJSON(`{"term":0}`, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid contract term")
		var cliErr *exitcodes.CLIError
		require.ErrorAs(t, err, &cliErr)
		assert.Equal(t, exitcodes.Usage, cliErr.Code)
	})

	for _, term := range validation.ValidContractTerms {
		t.Run(fmt.Sprintf("term %d is accepted", term), func(t *testing.T) {
			req, err := buildUpdateVXCRequestFromJSON(fmt.Sprintf(`{"term":%d}`, term), "")
			require.NoError(t, err)
			require.NotNil(t, req)
			require.NotNil(t, req.Term)
			assert.Equal(t, term, *req.Term)
		})
	}
}

func TestResolvePartnerPortUID(t *testing.T) {
	origGetPartnerPortUID := getPartnerPortUID
	defer func() { getPartnerPortUID = origGetPartnerPortUID }()

	tests := []struct {
		name          string
		partnerConfig megaport.VXCPartnerConfiguration
		mockUID       string
		mockErr       error
		expectedUID   string
		expectedError string
	}{
		{
			name:          "Azure success",
			partnerConfig: &megaport.VXCPartnerConfigAzure{ServiceKey: "azure-key-123"},
			mockUID:       "azure-port-uid",
			expectedUID:   "azure-port-uid",
		},
		{
			name:          "Google success",
			partnerConfig: &megaport.VXCPartnerConfigGoogle{PairingKey: "google-key-456"},
			mockUID:       "google-port-uid",
			expectedUID:   "google-port-uid",
		},
		{
			name:          "Oracle success",
			partnerConfig: &megaport.VXCPartnerConfigOracle{VirtualCircuitId: "oracle-vc-789"},
			mockUID:       "oracle-port-uid",
			expectedUID:   "oracle-port-uid",
		},
		{
			name:          "Azure empty service key",
			partnerConfig: &megaport.VXCPartnerConfigAzure{ServiceKey: ""},
			expectedError: "serviceKey is required for Azure configuration",
		},
		{
			name:          "Google empty pairing key",
			partnerConfig: &megaport.VXCPartnerConfigGoogle{PairingKey: ""},
			expectedError: "pairingKey is required for Google configuration",
		},
		{
			name:          "Oracle empty virtual circuit ID",
			partnerConfig: &megaport.VXCPartnerConfigOracle{VirtualCircuitId: ""},
			expectedError: "virtualCircuitId is required for Oracle configuration",
		},
		{
			name:          "Azure lookup error wraps with partner name",
			partnerConfig: &megaport.VXCPartnerConfigAzure{ServiceKey: "bad-key"},
			mockErr:       fmt.Errorf("failed to look up partner port: API error"),
			expectedError: "azure partner port:",
		},
		{
			name:          "unsupported partner type returns empty",
			partnerConfig: &megaport.VXCPartnerConfigAWS{OwnerAccount: "123"},
			expectedUID:   "",
		},
		{
			name:          "transit partner type returns empty",
			partnerConfig: &megaport.VXCPartnerConfigTransit{ConnectType: "TRANSIT"},
			expectedUID:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getPartnerPortUID = func(_ context.Context, _ megaport.VXCService, _, _ string) (string, error) {
				return tt.mockUID, tt.mockErr
			}

			ctx := context.Background()
			mockSvc := &MockVXCService{}

			uid, err := resolvePartnerPortUID(ctx, mockSvc, tt.partnerConfig)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedUID, uid)
			}
		})
	}
}

func TestBuildVXCRequestFromFlags_VNICIndexZero(t *testing.T) {
	newBuyCmd := func() *cobra.Command {
		cmd := &cobra.Command{Use: "buy"}
		cmd.Flags().String("a-end-uid", "", "")
		cmd.Flags().String("b-end-uid", "", "")
		cmd.Flags().String("name", "", "")
		cmd.Flags().Int("rate-limit", 0, "")
		cmd.Flags().Int("term", 0, "")
		cmd.Flags().Int("a-end-vlan", 0, "")
		cmd.Flags().Int("b-end-vlan", 0, "")
		cmd.Flags().Int("a-end-inner-vlan", 0, "")
		cmd.Flags().Int("b-end-inner-vlan", 0, "")
		cmd.Flags().Int("a-end-vnic-index", 0, "")
		cmd.Flags().Int("b-end-vnic-index", 0, "")
		cmd.Flags().String("promo-code", "", "")
		cmd.Flags().String("service-key", "", "")
		cmd.Flags().String("cost-centre", "", "")
		cmd.Flags().String("a-end-partner-config", "", "")
		cmd.Flags().String("b-end-partner-config", "", "")
		return cmd
	}

	setRequired := func(t *testing.T, cmd *cobra.Command) {
		require.NoError(t, cmd.Flags().Set("name", "Test VXC"))
		require.NoError(t, cmd.Flags().Set("a-end-uid", "a-end-uid-123"))
		require.NoError(t, cmd.Flags().Set("b-end-uid", "b-end-uid-123"))
		require.NoError(t, cmd.Flags().Set("rate-limit", "100"))
		require.NoError(t, cmd.Flags().Set("term", "1"))
	}

	t.Run("A-End vNIC index 0 set via flag builds MVE config", func(t *testing.T) {
		cmd := newBuyCmd()
		setRequired(t, cmd)
		require.NoError(t, cmd.Flags().Set("a-end-vnic-index", "0"))

		req, err := buildVXCRequestFromFlags(cmd, context.Background(), &MockVXCService{})
		require.NoError(t, err)
		require.NotNil(t, req.AEndConfiguration.VXCOrderMVEConfig)
		assert.Equal(t, 0, req.AEndConfiguration.NetworkInterfaceIndex)
	})

	t.Run("B-End vNIC index 0 set via flag builds MVE config", func(t *testing.T) {
		cmd := newBuyCmd()
		setRequired(t, cmd)
		require.NoError(t, cmd.Flags().Set("b-end-vnic-index", "0"))

		req, err := buildVXCRequestFromFlags(cmd, context.Background(), &MockVXCService{})
		require.NoError(t, err)
		require.NotNil(t, req.BEndConfiguration.VXCOrderMVEConfig)
		assert.Equal(t, 0, req.BEndConfiguration.NetworkInterfaceIndex)
	})

	t.Run("vNIC index not set leaves MVE config nil", func(t *testing.T) {
		cmd := newBuyCmd()
		setRequired(t, cmd)

		req, err := buildVXCRequestFromFlags(cmd, context.Background(), &MockVXCService{})
		require.NoError(t, err)
		assert.Nil(t, req.AEndConfiguration.VXCOrderMVEConfig)
		assert.Nil(t, req.BEndConfiguration.VXCOrderMVEConfig)
	})
}

func TestBuildVXCRequestFromFlags_ResourceTags(t *testing.T) {
	newBuyCmd := func() *cobra.Command {
		cmd := &cobra.Command{Use: "buy"}
		cmd.Flags().String("a-end-uid", "", "")
		cmd.Flags().String("b-end-uid", "", "")
		cmd.Flags().String("name", "", "")
		cmd.Flags().Int("rate-limit", 0, "")
		cmd.Flags().Int("term", 0, "")
		cmd.Flags().Int("a-end-vlan", 0, "")
		cmd.Flags().Int("b-end-vlan", 0, "")
		cmd.Flags().Int("a-end-inner-vlan", 0, "")
		cmd.Flags().Int("b-end-inner-vlan", 0, "")
		cmd.Flags().Int("a-end-vnic-index", 0, "")
		cmd.Flags().Int("b-end-vnic-index", 0, "")
		cmd.Flags().String("promo-code", "", "")
		cmd.Flags().String("service-key", "", "")
		cmd.Flags().String("cost-centre", "", "")
		cmd.Flags().String("a-end-partner-config", "", "")
		cmd.Flags().String("b-end-partner-config", "", "")
		cmd.Flags().String("resource-tags", "", "")
		require.NoError(t, cmd.Flags().Set("name", "Test VXC"))
		require.NoError(t, cmd.Flags().Set("a-end-uid", "a-end-uid-123"))
		require.NoError(t, cmd.Flags().Set("b-end-uid", "b-end-uid-123"))
		require.NoError(t, cmd.Flags().Set("rate-limit", "100"))
		require.NoError(t, cmd.Flags().Set("term", "1"))
		return cmd
	}

	t.Run("tags round-trip through the flags path", func(t *testing.T) {
		cmd := newBuyCmd()
		require.NoError(t, cmd.Flags().Set("resource-tags", `{"env":"prod","team":"networking"}`))

		req, err := buildVXCRequestFromFlags(cmd, context.Background(), &MockVXCService{})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"env": "prod", "team": "networking"}, req.ResourceTags)
	})

	t.Run("no flag leaves tags nil", func(t *testing.T) {
		cmd := newBuyCmd()

		req, err := buildVXCRequestFromFlags(cmd, context.Background(), &MockVXCService{})
		require.NoError(t, err)
		assert.Nil(t, req.ResourceTags)
	})

	t.Run("malformed JSON is a usage error", func(t *testing.T) {
		cmd := newBuyCmd()
		require.NoError(t, cmd.Flags().Set("resource-tags", `{bad}`))

		_, err := buildVXCRequestFromFlags(cmd, context.Background(), &MockVXCService{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse resource tags JSON")

		var cliErr *exitcodes.CLIError
		require.True(t, errors.As(err, &cliErr))
		assert.Equal(t, exitcodes.Usage, cliErr.Code)
	})

	t.Run("empty tag key rejected", func(t *testing.T) {
		cmd := newBuyCmd()
		require.NoError(t, cmd.Flags().Set("resource-tags", `{"":"x"}`))

		_, err := buildVXCRequestFromFlags(cmd, context.Background(), &MockVXCService{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tag key must not be empty")

		var cliErr *exitcodes.CLIError
		require.True(t, errors.As(err, &cliErr))
		assert.Equal(t, exitcodes.Usage, cliErr.Code)
	})

	t.Run("null tag value rejected, matching the JSON path", func(t *testing.T) {
		cmd := newBuyCmd()
		require.NoError(t, cmd.Flags().Set("resource-tags", `{"env":null}`))

		_, err := buildVXCRequestFromFlags(cmd, context.Background(), &MockVXCService{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `resourceTags value for key "env" must be a string`)

		var cliErr *exitcodes.CLIError
		require.True(t, errors.As(err, &cliErr))
		assert.Equal(t, exitcodes.Usage, cliErr.Code)
	})
}

func TestBuildVXCRequestFromFlags_PartnerConfig(t *testing.T) {
	origResolve := resolvePartnerPortUID
	defer func() { resolvePartnerPortUID = origResolve }()

	newBuyCmd := func() *cobra.Command {
		cmd := &cobra.Command{Use: "buy"}
		cmd.Flags().String("a-end-uid", "", "")
		cmd.Flags().String("b-end-uid", "", "")
		cmd.Flags().String("name", "", "")
		cmd.Flags().Int("rate-limit", 0, "")
		cmd.Flags().Int("term", 0, "")
		cmd.Flags().Int("a-end-vlan", 0, "")
		cmd.Flags().Int("b-end-vlan", 0, "")
		cmd.Flags().Int("a-end-inner-vlan", 0, "")
		cmd.Flags().Int("b-end-inner-vlan", 0, "")
		cmd.Flags().Int("a-end-vnic-index", 0, "")
		cmd.Flags().Int("b-end-vnic-index", 0, "")
		cmd.Flags().String("promo-code", "", "")
		cmd.Flags().String("service-key", "", "")
		cmd.Flags().String("cost-centre", "", "")
		cmd.Flags().String("a-end-partner-config", "", "")
		cmd.Flags().String("b-end-partner-config", "", "")
		return cmd
	}

	tests := []struct {
		name            string
		flags           map[string]string
		flagsInt        map[string]int
		resolveUID      string
		resolveErr      error
		expectedError   string
		validateRequest func(*testing.T, *megaport.BuyVXCRequest)
	}{
		{
			name: "A-End Azure partner config resolves UID and is assigned",
			flags: map[string]string{
				"name":                 "Test VXC",
				"b-end-uid":            "b-end-uid-123",
				"a-end-partner-config": `{"connectType":"AZURE","serviceKey":"azure-svc-key"}`,
			},
			flagsInt:   map[string]int{"rate-limit": 100, "term": 1},
			resolveUID: "resolved-a-uid",
			validateRequest: func(t *testing.T, req *megaport.BuyVXCRequest) {
				assert.Equal(t, "resolved-a-uid", req.PortUID)
				assert.NotNil(t, req.AEndConfiguration.PartnerConfig)
				azCfg, ok := req.AEndConfiguration.PartnerConfig.(*megaport.VXCPartnerConfigAzure)
				require.True(t, ok)
				assert.Equal(t, "azure-svc-key", azCfg.ServiceKey)
			},
		},
		{
			name: "A-End Google partner config resolves UID and is assigned",
			flags: map[string]string{
				"name":                 "Test VXC",
				"b-end-uid":            "b-end-uid-123",
				"a-end-partner-config": `{"connectType":"GOOGLE","pairingKey":"google-pairing-key"}`,
			},
			flagsInt:   map[string]int{"rate-limit": 100, "term": 1},
			resolveUID: "resolved-a-uid",
			validateRequest: func(t *testing.T, req *megaport.BuyVXCRequest) {
				assert.Equal(t, "resolved-a-uid", req.PortUID)
				assert.NotNil(t, req.AEndConfiguration.PartnerConfig)
				_, ok := req.AEndConfiguration.PartnerConfig.(*megaport.VXCPartnerConfigGoogle)
				require.True(t, ok)
			},
		},
		{
			name: "B-End Azure partner config resolves UID",
			flags: map[string]string{
				"name":                 "Test VXC",
				"a-end-uid":            "a-end-uid-123",
				"b-end-partner-config": `{"connectType":"AZURE","serviceKey":"azure-svc-key"}`,
			},
			flagsInt:   map[string]int{"rate-limit": 100, "term": 1},
			resolveUID: "resolved-b-uid",
			validateRequest: func(t *testing.T, req *megaport.BuyVXCRequest) {
				assert.Equal(t, "resolved-b-uid", req.BEndConfiguration.ProductUID)
				assert.NotNil(t, req.BEndConfiguration.PartnerConfig)
			},
		},
		{
			name: "A-End partner resolve error",
			flags: map[string]string{
				"name":                 "Test VXC",
				"b-end-uid":            "b-end-uid-123",
				"a-end-partner-config": `{"connectType":"AZURE","serviceKey":"bad-key"}`,
			},
			flagsInt:      map[string]int{"rate-limit": 100, "term": 1},
			resolveErr:    fmt.Errorf("azure partner port: failed to look up partner port: API error"),
			expectedError: "failed to look up A-End Partner Port",
		},
		{
			name: "B-End partner resolve error",
			flags: map[string]string{
				"name":                 "Test VXC",
				"a-end-uid":            "a-end-uid-123",
				"b-end-partner-config": `{"connectType":"GOOGLE","pairingKey":"bad-key"}`,
			},
			flagsInt:      map[string]int{"rate-limit": 100, "term": 1},
			resolveErr:    fmt.Errorf("google partner port: failed to look up partner port: API error"),
			expectedError: "failed to look up B-End Partner Port",
		},
		{
			name: "A-End UID provided skips resolution but keeps partner config",
			flags: map[string]string{
				"name":                 "Test VXC",
				"a-end-uid":            "explicit-a-uid",
				"b-end-uid":            "b-end-uid-123",
				"a-end-partner-config": `{"connectType":"AZURE","serviceKey":"azure-svc-key"}`,
			},
			flagsInt: map[string]int{"rate-limit": 100, "term": 1},
			validateRequest: func(t *testing.T, req *megaport.BuyVXCRequest) {
				assert.Equal(t, "explicit-a-uid", req.PortUID)
				assert.NotNil(t, req.AEndConfiguration.PartnerConfig)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolvePartnerPortUID = func(_ context.Context, _ megaport.VXCService, _ megaport.VXCPartnerConfiguration) (string, error) {
				return tt.resolveUID, tt.resolveErr
			}

			cmd := newBuyCmd()
			for k, v := range tt.flags {
				require.NoError(t, cmd.Flags().Set(k, v))
			}
			for k, v := range tt.flagsInt {
				require.NoError(t, cmd.Flags().Set(k, fmt.Sprintf("%d", v)))
			}

			ctx := context.Background()
			mockSvc := &MockVXCService{}

			req, err := buildVXCRequestFromFlags(cmd, ctx, mockSvc)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				if tt.validateRequest != nil {
					tt.validateRequest(t, req)
				}
			}
		})
	}
}

func TestParseBGPConnections_PasswordNotEchoed(t *testing.T) {
	const password = "super-secret-bgp-password"
	cfg := map[string]interface{}{
		"connectType": "VROUTER",
		"interfaces": []interface{}{
			map[string]interface{}{
				"bgpConnections": []interface{}{
					map[string]interface{}{
						"password": password,
						// localAsn wrong type triggers a type error before password is used
						"localAsn": "not-a-number",
					},
				},
			},
		},
	}
	_, err := parseVRouterConfig(cfg)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), password, "BGP password must never appear in a parse error")
}
