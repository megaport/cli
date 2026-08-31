package vxc

import (
	"context"
	"fmt"
	"testing"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/testutil"
	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/megaport/megaport-cli/internal/validation"
	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockPrompts(responses []string) func() {
	original := utils.GetResourcePrompt()
	idx := 0
	utils.SetResourcePrompt(func(_, _ string, _ bool) (string, error) {
		if idx < len(responses) {
			r := responses[idx]
			idx++
			return r, nil
		}
		return "", fmt.Errorf("unexpected prompt call at index %d", idx)
	})
	return func() { utils.SetResourcePrompt(original) }
}

func TestPromptAWSConfig(t *testing.T) {
	tests := []struct {
		name      string
		responses []string
		verify    func(t *testing.T, cfg *megaport.VXCPartnerConfigAWS)
	}{
		{
			name:      "valid AWS config",
			responses: []string{"AWS", "123456789", "my-conn", "65000", "", "", "", "", "", "private"},
			verify: func(t *testing.T, cfg *megaport.VXCPartnerConfigAWS) {
				assert.Equal(t, "AWS", cfg.ConnectType)
				assert.Equal(t, "123456789", cfg.OwnerAccount)
				assert.Equal(t, "my-conn", cfg.ConnectionName)
				assert.Equal(t, 65000, cfg.ASN)
				assert.Equal(t, 0, cfg.AmazonASN)
				assert.Equal(t, "", cfg.AuthKey)
				assert.Equal(t, "", cfg.Prefixes)
				assert.Equal(t, "", cfg.CustomerIPAddress)
				assert.Equal(t, "", cfg.AmazonIPAddress)
				assert.Equal(t, "private", cfg.Type)
			},
		},
		{
			name:      "valid AWSHC config",
			responses: []string{"AWSHC", "987654321", "hc-conn", "64512", "65000", "authkey123", "10.0.0.0/8", "1.2.3.4", "5.6.7.8"},
			verify: func(t *testing.T, cfg *megaport.VXCPartnerConfigAWS) {
				assert.Equal(t, "AWSHC", cfg.ConnectType)
				assert.Equal(t, "987654321", cfg.OwnerAccount)
				assert.Equal(t, "hc-conn", cfg.ConnectionName)
				assert.Equal(t, 64512, cfg.ASN)
				assert.Equal(t, 65000, cfg.AmazonASN)
				assert.Equal(t, "authkey123", cfg.AuthKey)
				assert.Equal(t, "10.0.0.0/8", cfg.Prefixes)
				assert.Equal(t, "1.2.3.4", cfg.CustomerIPAddress)
				assert.Equal(t, "5.6.7.8", cfg.AmazonIPAddress)
				assert.Equal(t, "", cfg.Type)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := mockPrompts(tc.responses)
			defer cleanup()

			cfg, err := promptAWSConfig(true)
			assert.NoError(t, err)
			assert.NotNil(t, cfg)
			tc.verify(t, cfg)
		})
	}
}

func TestPromptGoogleConfig(t *testing.T) {
	tests := []struct {
		name        string
		responses   []string
		mockUID     string
		wantKey     string
		wantUID     string
		wantConnect string
	}{
		{
			name:        "valid",
			responses:   []string{"pairing-key-123"},
			mockUID:     "partner-uid-1",
			wantKey:     "pairing-key-123",
			wantUID:     "partner-uid-1",
			wantConnect: "GOOGLE",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := mockPrompts(tc.responses)
			defer cleanup()

			originalGetPartnerPortUID := getPartnerPortUID
			getPartnerPortUID = func(_ context.Context, _ megaport.VXCService, _, _ string) (string, error) {
				return tc.mockUID, nil
			}
			defer func() { getPartnerPortUID = originalGetPartnerPortUID }()

			mockSvc := &MockVXCService{}
			ctx := context.Background()

			cfg, uid, err := promptGoogleConfig(ctx, mockSvc, true)
			assert.NoError(t, err)
			assert.NotNil(t, cfg)
			assert.Equal(t, tc.wantKey, cfg.PairingKey)
			assert.Equal(t, tc.wantConnect, cfg.ConnectType)
			assert.Equal(t, tc.wantUID, uid)
		})
	}
}

func TestPromptOracleConfig(t *testing.T) {
	tests := []struct {
		name      string
		responses []string
		mockUID   string
		wantVC    string
		wantUID   string
	}{
		{
			name:      "valid",
			responses: []string{"vc-123"},
			mockUID:   "oracle-uid-1",
			wantVC:    "vc-123",
			wantUID:   "oracle-uid-1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := mockPrompts(tc.responses)
			defer cleanup()

			originalGetPartnerPortUID := getPartnerPortUID
			getPartnerPortUID = func(_ context.Context, _ megaport.VXCService, _, _ string) (string, error) {
				return tc.mockUID, nil
			}
			defer func() { getPartnerPortUID = originalGetPartnerPortUID }()

			mockSvc := &MockVXCService{}
			ctx := context.Background()

			cfg, uid, err := promptOracleConfig(ctx, mockSvc, true)
			assert.NoError(t, err)
			assert.NotNil(t, cfg)
			assert.Equal(t, tc.wantVC, cfg.VirtualCircuitId)
			assert.Equal(t, "ORACLE", cfg.ConnectType)
			assert.Equal(t, tc.wantUID, uid)
		})
	}
}

func TestPromptIBMConfig(t *testing.T) {
	tests := []struct {
		name      string
		responses []string
		verify    func(t *testing.T, cfg *megaport.VXCPartnerConfigIBM)
	}{
		{
			name:      "valid with customer ASN",
			responses: []string{"acct-123", "my-ibm", "65000", "1.2.3.4", "5.6.7.8"},
			verify: func(t *testing.T, cfg *megaport.VXCPartnerConfigIBM) {
				assert.Equal(t, "IBM", cfg.ConnectType)
				assert.Equal(t, "acct-123", cfg.AccountID)
				assert.Equal(t, "my-ibm", cfg.Name)
				assert.Equal(t, 65000, cfg.CustomerASN)
				assert.Equal(t, "1.2.3.4", cfg.CustomerIPAddress)
				assert.Equal(t, "5.6.7.8", cfg.ProviderIPAddress)
			},
		},
		{
			name:      "opposite end is MCR",
			responses: []string{"acct-456", "ibm-mcr", "0", "2.3.4.5", "6.7.8.9"},
			verify: func(t *testing.T, cfg *megaport.VXCPartnerConfigIBM) {
				assert.Equal(t, "IBM", cfg.ConnectType)
				assert.Equal(t, "acct-456", cfg.AccountID)
				assert.Equal(t, "ibm-mcr", cfg.Name)
				assert.Equal(t, 0, cfg.CustomerASN)
				assert.Equal(t, "2.3.4.5", cfg.CustomerIPAddress)
				assert.Equal(t, "6.7.8.9", cfg.ProviderIPAddress)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := mockPrompts(tc.responses)
			defer cleanup()

			cfg, err := promptIBMConfig(true)
			assert.NoError(t, err)
			assert.NotNil(t, cfg)
			tc.verify(t, cfg)
		})
	}
}

func TestPromptBFDConfig(t *testing.T) {
	tests := []struct {
		name      string
		responses []string
		verify    func(t *testing.T, cfg megaport.BfdConfig)
	}{
		{
			name:      "defaults",
			responses: []string{"", "", ""},
			verify: func(t *testing.T, cfg megaport.BfdConfig) {
				assert.Equal(t, 300, cfg.TxInterval)
				assert.Equal(t, 300, cfg.RxInterval)
				assert.Equal(t, 3, cfg.Multiplier)
			},
		},
		{
			name:      "custom values",
			responses: []string{"500", "600", "5"},
			verify: func(t *testing.T, cfg megaport.BfdConfig) {
				assert.Equal(t, 500, cfg.TxInterval)
				assert.Equal(t, 600, cfg.RxInterval)
				assert.Equal(t, 5, cfg.Multiplier)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := mockPrompts(tc.responses)
			defer cleanup()

			cfg, err := promptBFDConfig(true)
			assert.NoError(t, err)
			tc.verify(t, cfg)
		})
	}
}

func TestPromptBGPConnections(t *testing.T) {
	tests := []struct {
		name            string
		responses       []string
		secretResponses []string
		verify          func(t *testing.T, conns []megaport.BgpConnectionConfig)
	}{
		{
			name:      "no connections",
			responses: []string{"no"},
			verify: func(t *testing.T, conns []megaport.BgpConnectionConfig) {
				assert.Empty(t, conns)
			},
		},
		{
			name: "one connection minimal",
			responses: []string{
				"yes",      // Add a BGP connection?
				"65000",    // peerAsn
				"10.0.0.1", // localIP
				"10.0.0.2", // peerIP
				"",         // localAsn (optional)
				"no",       // shutdown
				"",         // description (optional)
				"no",       // bfdEnabled
				"",         // asOverride (optional)
				"",         // exportPolicy (optional)
				"",         // peerType (optional)
				"",         // medIn (optional)
				"",         // medOut (optional)
				"",         // asPrepend (optional)
				"no",       // permitExportTo
				"no",       // denyExportTo
				"",         // importWhitelist
				"",         // importBlacklist
				"",         // exportWhitelist
				"",         // exportBlacklist
				"no",       // add another
			},
			secretResponses: []string{
				"", // password (read without echo)
			},
			verify: func(t *testing.T, conns []megaport.BgpConnectionConfig) {
				assert.Len(t, conns, 1)
				assert.Equal(t, 65000, conns[0].PeerAsn)
				assert.Equal(t, "10.0.0.1", conns[0].LocalIpAddress)
				assert.Equal(t, "10.0.0.2", conns[0].PeerIpAddress)
				assert.False(t, conns[0].Shutdown)
				assert.False(t, conns[0].BfdEnabled)
				assert.Nil(t, conns[0].AsOverride)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := mockPromptsAndSecrets(tc.responses, tc.secretResponses)
			defer cleanup()

			conns, err := promptBGPConnections(true)
			assert.NoError(t, err)
			tc.verify(t, conns)
		})
	}
}

func TestPromptIPRoutes(t *testing.T) {
	tests := []struct {
		name      string
		responses []string
		verify    func(t *testing.T, routes []megaport.IpRoute)
	}{
		{
			name:      "no routes",
			responses: []string{"no"},
			verify: func(t *testing.T, routes []megaport.IpRoute) {
				assert.Empty(t, routes)
			},
		},
		{
			name:      "one route",
			responses: []string{"yes", "10.0.0.0/24", "10.0.0.1", "my-route", "no"},
			verify: func(t *testing.T, routes []megaport.IpRoute) {
				assert.Len(t, routes, 1)
				assert.Equal(t, "10.0.0.0/24", routes[0].Prefix)
				assert.Equal(t, "10.0.0.1", routes[0].NextHop)
				assert.Equal(t, "my-route", routes[0].Description)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := mockPrompts(tc.responses)
			defer cleanup()

			routes, err := promptIPRoutes(true)
			assert.NoError(t, err)
			tc.verify(t, routes)
		})
	}
}

func TestPromptIPAddresses(t *testing.T) {
	tests := []struct {
		name      string
		responses []string
		verify    func(t *testing.T, addrs []string)
	}{
		{
			name:      "no addresses",
			responses: []string{"no"},
			verify: func(t *testing.T, addrs []string) {
				assert.Empty(t, addrs)
			},
		},
		{
			name:      "two addresses",
			responses: []string{"yes", "10.0.0.1", "yes", "10.0.0.2", "no"},
			verify: func(t *testing.T, addrs []string) {
				assert.Equal(t, []string{"10.0.0.1", "10.0.0.2"}, addrs)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := mockPrompts(tc.responses)
			defer cleanup()

			addrs, err := promptIPAddresses("IP address", true)
			assert.NoError(t, err)
			tc.verify(t, addrs)
		})
	}
}

func TestPromptNATIPAddresses(t *testing.T) {
	cleanup := mockPrompts([]string{"yes", "10.0.0.1", "yes", "10.0.0.2", "no"})
	defer cleanup()

	addrs, err := promptNATIPAddresses(true)
	assert.NoError(t, err)
	assert.Equal(t, []string{"10.0.0.1", "10.0.0.2"}, addrs)
}

func TestPromptTransitConfig(t *testing.T) {
	cfg := promptTransitConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, "TRANSIT", cfg.ConnectType)
}

func TestPromptAzureConfig(t *testing.T) {
	cleanup := mockPrompts([]string{
		"sk-azure-123", // service key
		"primary",      // port choice
		"yes",          // add peering config
		"private",      // peering type
		"65000",        // peer ASN
		"10.0.0.0/30",  // primary subnet
		"10.0.0.4/30",  // secondary subnet
		"10.1.0.0/16",  // prefixes
		"sharedkey",    // shared key
		"100",          // vlan
		"no",           // add another peering
	})
	defer cleanup()

	mockSvc := &MockVXCService{
		ListPartnerPortsResponse: &megaport.ListPartnerPortsResponse{
			Data: megaport.PartnerLookup{
				Megaports: []megaport.PartnerLookupItem{
					{ProductUID: "azure-uid-1", Type: "primary"},
					{ProductUID: "azure-uid-2", Type: "secondary"},
				},
			},
		},
	}

	ctx := context.Background()
	cfg, uid, err := promptAzureConfig(ctx, mockSvc, true)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "AZURE", cfg.ConnectType)
	assert.Equal(t, "sk-azure-123", cfg.ServiceKey)
	assert.Len(t, cfg.Peers, 1)
	assert.Equal(t, "private", cfg.Peers[0].Type)
	assert.Equal(t, "65000", cfg.Peers[0].PeerASN)
	assert.Equal(t, "10.0.0.0/30", cfg.Peers[0].PrimarySubnet)
	assert.Equal(t, "10.0.0.4/30", cfg.Peers[0].SecondarySubnet)
	assert.Equal(t, "10.1.0.0/16", cfg.Peers[0].Prefixes)
	assert.Equal(t, "sharedkey", cfg.Peers[0].SharedKey)
	assert.Equal(t, 100, cfg.Peers[0].VLAN)
	assert.Equal(t, "azure-uid-1", uid)
}

func TestPromptAzurePeeringConfig(t *testing.T) {
	cleanup := mockPrompts([]string{
		"Microsoft", // peering type
		"12076",     // peer ASN
		"10.0.0.0/30",
		"10.0.0.4/30",
		"10.2.0.0/16",
		"key123",
		"200",
	})
	defer cleanup()

	peer, err := promptAzurePeeringConfig(true)
	assert.NoError(t, err)
	assert.Equal(t, "Microsoft", peer.Type)
	assert.Equal(t, "12076", peer.PeerASN)
	assert.Equal(t, "10.0.0.0/30", peer.PrimarySubnet)
	assert.Equal(t, "10.0.0.4/30", peer.SecondarySubnet)
	assert.Equal(t, "10.2.0.0/16", peer.Prefixes)
	assert.Equal(t, "key123", peer.SharedKey)
	assert.Equal(t, 200, peer.VLAN)
}

func TestPromptPartnerConfig(t *testing.T) {
	tests := []struct {
		name      string
		responses []string
		setupMock func()
		cleanup   func()
		verify    func(t *testing.T, cfg megaport.VXCPartnerConfiguration, uid string)
	}{
		{
			name: "transit path",
			responses: []string{
				"transit", // partner type
			},
			verify: func(t *testing.T, cfg megaport.VXCPartnerConfiguration, uid string) {
				transitCfg, ok := cfg.(*megaport.VXCPartnerConfigTransit)
				assert.True(t, ok)
				assert.Equal(t, "TRANSIT", transitCfg.ConnectType)
				assert.Equal(t, "", uid)
			},
		},
		{
			name: "empty partner returns nil",
			responses: []string{
				"", // empty partner
			},
			verify: func(t *testing.T, cfg megaport.VXCPartnerConfiguration, uid string) {
				assert.Nil(t, cfg)
				assert.Equal(t, "", uid)
			},
		},
		{
			name: "google path",
			responses: []string{
				"google",         // partner type
				"pairing-key-99", // pairing key
			},
			setupMock: func() {
				getPartnerPortUID = func(_ context.Context, _ megaport.VXCService, _, _ string) (string, error) {
					return "google-uid-1", nil
				}
			},
			verify: func(t *testing.T, cfg megaport.VXCPartnerConfiguration, uid string) {
				googleCfg, ok := cfg.(*megaport.VXCPartnerConfigGoogle)
				assert.True(t, ok)
				assert.Equal(t, "pairing-key-99", googleCfg.PairingKey)
				assert.Equal(t, "google-uid-1", uid)
			},
		},
		{
			name: "ibm path",
			responses: []string{
				"ibm",       // partner type
				"ibm-acct",  // account ID
				"ibm-name",  // name
				"65000",     // customer ASN
				"1.2.3.4",   // customer IP
				"5.6.7.8",   // provider IP
				"ibm-uid-1", // partner port UID
			},
			verify: func(t *testing.T, cfg megaport.VXCPartnerConfiguration, uid string) {
				ibmCfg, ok := cfg.(*megaport.VXCPartnerConfigIBM)
				assert.True(t, ok)
				assert.Equal(t, "ibm-acct", ibmCfg.AccountID)
				assert.Equal(t, "ibm-uid-1", uid)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := mockPrompts(tc.responses)
			defer cleanup()

			originalGetPartnerPortUID := getPartnerPortUID
			if tc.setupMock != nil {
				tc.setupMock()
			}
			defer func() { getPartnerPortUID = originalGetPartnerPortUID }()

			mockSvc := &MockVXCService{}
			ctx := context.Background()

			cfg, uid, err := promptPartnerConfig("A-End", ctx, mockSvc, true)
			assert.NoError(t, err)
			tc.verify(t, cfg, uid)
		})
	}
}

func TestPromptBGPOptionalConfig_WithValues(t *testing.T) {
	cleanup := mockPromptsAndSecrets([]string{
		"64512",     // localAsn
		"yes",       // shutdown
		"my bgp",    // description
		"yes",       // bfdEnabled
		"yes",       // asOverride
		"permit",    // exportPolicy
		"NON_CLOUD", // peerType
		"100",       // medIn
		"200",       // medOut
		"3",         // asPathPrepend
	}, []string{
		"secret", // password (read without echo)
	})
	defer cleanup()

	bgp := &megaport.BgpConnectionConfig{}
	err := promptBGPOptionalConfig(bgp, true)
	assert.NoError(t, err)
	assert.NotNil(t, bgp.LocalAsn)
	assert.Equal(t, 64512, *bgp.LocalAsn)
	assert.Equal(t, "secret", bgp.Password)
	assert.True(t, bgp.Shutdown)
	assert.Equal(t, "my bgp", bgp.Description)
	assert.True(t, bgp.BfdEnabled)
	if assert.NotNil(t, bgp.AsOverride) {
		assert.True(t, *bgp.AsOverride)
	}
	assert.Equal(t, "permit", bgp.ExportPolicy)
	assert.Equal(t, "NON_CLOUD", bgp.PeerType)
	assert.Equal(t, 100, bgp.MedIn)
	assert.Equal(t, 200, bgp.MedOut)
	assert.Equal(t, 3, bgp.AsPathPrependCount)
}

func TestPromptBGPOptionalConfig_AsOverrideNo(t *testing.T) {
	cleanup := mockPromptsAndSecrets([]string{
		"",   // localAsn (optional)
		"no", // shutdown
		"",   // description
		"no", // bfdEnabled
		"no", // asOverride
		"",   // exportPolicy
		"",   // peerType
		"",   // medIn
		"",   // medOut
		"",   // asPathPrepend
	}, []string{
		"", // password
	})
	defer cleanup()

	bgp := &megaport.BgpConnectionConfig{}
	err := promptBGPOptionalConfig(bgp, true)
	assert.NoError(t, err)
	if assert.NotNil(t, bgp.AsOverride) {
		assert.False(t, *bgp.AsOverride)
	}
}

func TestPromptBGPOptionalConfig_AsOverrideInvalid(t *testing.T) {
	// The prompt returns before reaching later fields, so only the prompts up
	// to and including asOverride need feeding.
	cleanup := mockPromptsAndSecrets([]string{
		"",      // localAsn (optional)
		"no",    // shutdown
		"",      // description
		"no",    // bfdEnabled
		"maybe", // asOverride (invalid)
	}, []string{
		"", // password
	})
	defer cleanup()

	bgp := &megaport.BgpConnectionConfig{}
	err := promptBGPOptionalConfig(bgp, true)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "AS Override must be 'yes' or 'no'")
	}
	assert.Nil(t, bgp.AsOverride)
}

func TestPromptBGPOptionalConfig_AsOverridePromptError(t *testing.T) {
	// Exhaust the prompt queue exactly at the AS Override prompt so ResourcePrompt
	// errors, exercising the error-propagation branch.
	cleanup := mockPromptsAndSecrets([]string{
		"",   // localAsn
		"no", // shutdown
		"",   // description
		"no", // bfdEnabled
	}, []string{
		"", // password
	})
	defer cleanup()

	bgp := &megaport.BgpConnectionConfig{}
	err := promptBGPOptionalConfig(bgp, true)
	assert.Error(t, err)
	assert.Nil(t, bgp.AsOverride)
}

func TestPromptBGPExportAddresses_WithValues(t *testing.T) {
	cleanup := mockPrompts([]string{
		"yes",         // permit export to
		"10.0.0.1",    // permit address 1
		"10.0.0.2",    // permit address 2
		"",            // stop permit
		"yes",         // deny export to
		"192.168.0.1", // deny address 1
		"",            // stop deny
	})
	defer cleanup()

	bgp := &megaport.BgpConnectionConfig{}
	err := promptBGPExportAddresses(bgp, true)
	assert.NoError(t, err)
	assert.Equal(t, []string{"10.0.0.1", "10.0.0.2"}, bgp.PermitExportTo)
	assert.Equal(t, []string{"192.168.0.1"}, bgp.DenyExportTo)
}

func TestPromptBGPPrefixLists_WithValues(t *testing.T) {
	cleanup := mockPrompts([]string{
		"101", // import whitelist
		"102", // import blacklist
		"201", // export whitelist
		"202", // export blacklist
	})
	defer cleanup()

	bgp := &megaport.BgpConnectionConfig{}
	err := promptBGPPrefixLists(bgp, true)
	assert.NoError(t, err)
	assert.Equal(t, 101, bgp.ImportWhitelist)
	assert.Equal(t, 102, bgp.ImportBlacklist)
	assert.Equal(t, 201, bgp.ExportWhitelist)
	assert.Equal(t, 202, bgp.ExportBlacklist)
}

func TestPromptVRouterConfig(t *testing.T) {
	tests := []struct {
		name            string
		responses       []string
		secretResponses []string
		verify          func(t *testing.T, cfg *megaport.VXCOrderVrouterPartnerConfig)
	}{
		{
			name: "single interface minimal",
			responses: []string{
				"1",   // num interfaces
				"100", // vlan
				"",    // interface type (default subInterface)
				"no",  // ip addresses
				"no",  // routes
				"no",  // NAT
				"no",  // BFD
				"no",  // BGP
			},
			verify: func(t *testing.T, cfg *megaport.VXCOrderVrouterPartnerConfig) {
				assert.Len(t, cfg.Interfaces, 1)
				assert.Equal(t, 100, cfg.Interfaces[0].VLAN)
			},
		},
		{
			name: "interface with routes NAT BFD and BGP",
			responses: []string{
				"1",             // num interfaces
				"200",           // vlan
				"",              // interface type (default subInterface)
				"yes",           // add IP address
				"10.0.0.1/30",   // IP address
				"no",            // more IPs
				"yes",           // add routes
				"yes",           // add route
				"10.1.0.0/24",   // prefix
				"10.0.0.2",      // next hop
				"route1",        // description
				"no",            // more routes
				"yes",           // NAT IPs
				"yes",           // add NAT IP
				"172.16.0.1/32", // NAT IP (CIDR notation)
				"no",            // more NAT IPs
				"yes",           // BFD
				"400",           // tx interval
				"400",           // rx interval
				"5",             // multiplier
				"yes",           // BGP
				"yes",           // add BGP connection
				"65000",         // peer ASN
				"10.0.0.1",      // local IP
				"10.0.0.2",      // peer IP
				"",              // local ASN (optional)
				"no",            // shutdown
				"",              // description
				"no",            // BFD enabled
				"",              // AS override
				"",              // export policy
				"",              // peer type
				"",              // MED in
				"",              // MED out
				"",              // AS path prepend
				"no",            // permit export to
				"no",            // deny export to
				"",              // import whitelist
				"",              // import blacklist
				"",              // export whitelist
				"",              // export blacklist
				"no",            // more BGP connections
			},
			secretResponses: []string{
				"", // BGP password (read without echo)
			},
			verify: func(t *testing.T, cfg *megaport.VXCOrderVrouterPartnerConfig) {
				assert.Len(t, cfg.Interfaces, 1)
				iface := cfg.Interfaces[0]
				assert.Equal(t, 200, iface.VLAN)
				assert.Equal(t, []string{"10.0.0.1/30"}, iface.IpAddresses)
				assert.Len(t, iface.IpRoutes, 1)
				assert.Equal(t, "10.1.0.0/24", iface.IpRoutes[0].Prefix)
				assert.Equal(t, "10.0.0.2", iface.IpRoutes[0].NextHop)
				assert.Equal(t, []string{"172.16.0.1/32"}, iface.NatIpAddresses)
				assert.Equal(t, 400, iface.Bfd.TxInterval)
				assert.Equal(t, 400, iface.Bfd.RxInterval)
				assert.Equal(t, 5, iface.Bfd.Multiplier)
				assert.Len(t, iface.BgpConnections, 1)
				assert.Equal(t, 65000, iface.BgpConnections[0].PeerAsn)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := mockPromptsAndSecrets(tc.responses, tc.secretResponses)
			defer cleanup()

			cfg, err := promptVRouterConfig(true)
			assert.NoError(t, err)
			assert.NotNil(t, cfg)
			tc.verify(t, cfg)
		})
	}
}

// mockPromptsAndSecrets feeds the regular prompt queue and the no-echo secret
// prompt queue separately, in call order. The PSK is collected via
// SecretResourcePrompt, so it pulls from secretResponses.
func mockPromptsAndSecrets(responses, secretResponses []string) func() {
	origPrompt := utils.GetResourcePrompt()
	origSecret := utils.GetSecretResourcePrompt()
	idx := 0
	utils.SetResourcePrompt(func(_, _ string, _ bool) (string, error) {
		if idx < len(responses) {
			r := responses[idx]
			idx++
			return r, nil
		}
		return "", fmt.Errorf("unexpected prompt call at index %d", idx)
	})
	sIdx := 0
	utils.SetSecretResourcePrompt(func(_, _ string, _ bool) (string, error) {
		if sIdx < len(secretResponses) {
			r := secretResponses[sIdx]
			sIdx++
			return r, nil
		}
		return "", fmt.Errorf("unexpected secret prompt call at index %d", sIdx)
	})
	return func() {
		utils.SetResourcePrompt(origPrompt)
		utils.SetSecretResourcePrompt(origSecret)
	}
}

func TestPromptVRouterConfigIPsec(t *testing.T) {
	responses := []string{
		"1",            // num interfaces
		"",             // vlan (none)
		"ipSecTunnel",  // interface type
		"192.0.2.1",    // source IP
		"198.51.100.1", // destination IP
		"yes",          // passive
		"local-x",      // local ID
		"remote-x",     // remote ID
		"28800",        // phase 1 lifetime
		"3600",         // phase 2 lifetime
	}
	secretResponses := []string{"topsecret"}

	cleanup := mockPromptsAndSecrets(responses, secretResponses)
	defer cleanup()

	cfg, err := promptVRouterConfig(true)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.Interfaces, 1)

	iface := cfg.Interfaces[0]
	assert.Equal(t, "ipSecTunnel", iface.InterfaceType)
	if !assert.NotNil(t, iface.IpSecTunnelOptions) {
		return
	}

	tunnel := iface.IpSecTunnelOptions
	assert.Equal(t, "192.0.2.1", tunnel.SourceIpAddress)
	assert.Equal(t, "198.51.100.1", tunnel.DestinationIpAddress)
	assert.Equal(t, "topsecret", tunnel.PreSharedKey)
	assert.Equal(t, "local-x", tunnel.LocalId)
	assert.Equal(t, "remote-x", tunnel.RemoteId)
	if assert.NotNil(t, tunnel.Passive) {
		assert.True(t, *tunnel.Passive)
	}
	if assert.NotNil(t, tunnel.Phase1Lifetime) {
		assert.Equal(t, 28800, *tunnel.Phase1Lifetime)
	}
	if assert.NotNil(t, tunnel.Phase2Lifetime) {
		assert.Equal(t, 3600, *tunnel.Phase2Lifetime)
	}
}

// Selecting the ipSecTunnel interface type but leaving the pre-shared key empty
// must be rejected by the validation the prompt path runs before returning.
func TestPromptVRouterConfigIPsecMissingPSK(t *testing.T) {
	responses := []string{
		"1",            // num interfaces
		"",             // vlan (none)
		"ipSecTunnel",  // interface type
		"192.0.2.1",    // source IP
		"198.51.100.1", // destination IP
		"",             // passive (API default)
		"",             // local ID
		"",             // remote ID
		"",             // phase 1 lifetime
		"",             // phase 2 lifetime
	}
	secretResponses := []string{""} // empty PSK

	cleanup := mockPromptsAndSecrets(responses, secretResponses)
	defer cleanup()

	_, err := promptVRouterConfig(true)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "pre-shared key")
	}
}

func TestBuildVXCRequestFromPrompt(t *testing.T) {
	tests := []struct {
		name      string
		responses []string
		verify    func(t *testing.T, req *megaport.BuyVXCRequest)
	}{
		{
			name: "minimal flow",
			responses: []string{
				"Test VXC",   // name
				"100",        // rate limit
				"12",         // term
				"100",        // A-End VLAN
				"",           // A-End inner VLAN
				"",           // A-End vNIC index
				"no",         // A-End partner config
				"port-a-123", // A-End product UID
				"200",        // B-End VLAN
				"",           // B-End inner VLAN
				"",           // B-End vNIC index
				"no",         // B-End partner config
				"port-b-456", // B-End product UID
				"",           // promo code
				"",           // service key
				"",           // cost centre
			},
			verify: func(t *testing.T, req *megaport.BuyVXCRequest) {
				assert.Equal(t, "Test VXC", req.VXCName)
				assert.Equal(t, 100, req.RateLimit)
				assert.Equal(t, 12, req.Term)
				assert.Equal(t, 100, req.AEndConfiguration.VLAN)
				assert.Equal(t, "port-a-123", req.PortUID)
				assert.Equal(t, 200, req.BEndConfiguration.VLAN)
				assert.Equal(t, "port-b-456", req.BEndConfiguration.ProductUID)
			},
		},
		{
			// Regression: the entered B-End VLAN was written to the request's
			// zero-value struct and then overwritten by the assembled bEndConfig,
			// so it never reached the submitted order.
			name: "entered B-End VLAN survives onto the request",
			responses: []string{
				"Test VXC",   // name
				"100",        // rate limit
				"12",         // term
				"100",        // A-End VLAN
				"",           // A-End inner VLAN
				"",           // A-End vNIC index
				"no",         // A-End partner config
				"port-a-123", // A-End product UID
				"3021",       // B-End VLAN
				"",           // B-End inner VLAN
				"",           // B-End vNIC index
				"no",         // B-End partner config
				"port-b-456", // B-End product UID
				"",           // promo code
				"",           // service key
				"",           // cost centre
			},
			verify: func(t *testing.T, req *megaport.BuyVXCRequest) {
				assert.Equal(t, 3021, req.BEndConfiguration.VLAN)
			},
		},
		{
			// Skipping the B-End VLAN prompt must leave the VLAN unset so the API
			// assigns one.
			name: "empty B-End VLAN leaves the VLAN unset",
			responses: []string{
				"Test VXC",   // name
				"100",        // rate limit
				"12",         // term
				"100",        // A-End VLAN
				"",           // A-End inner VLAN
				"",           // A-End vNIC index
				"no",         // A-End partner config
				"port-a-123", // A-End product UID
				"",           // B-End VLAN (skipped)
				"",           // B-End inner VLAN
				"",           // B-End vNIC index
				"no",         // B-End partner config
				"port-b-456", // B-End product UID
				"",           // promo code
				"",           // service key
				"",           // cost centre
			},
			verify: func(t *testing.T, req *megaport.BuyVXCRequest) {
				assert.Equal(t, 0, req.BEndConfiguration.VLAN)
			},
		},
		{
			// vNIC index 0 is a valid MVE interface. Entering "0" must build the
			// MVE config, not silently drop it (the old gate keyed on index > 0).
			name: "vNIC index 0 builds MVE config on both ends",
			responses: []string{
				"Test VXC",   // name
				"100",        // rate limit
				"12",         // term
				"100",        // A-End VLAN
				"",           // A-End inner VLAN
				"0",          // A-End vNIC index
				"no",         // A-End partner config
				"port-a-123", // A-End product UID
				"200",        // B-End VLAN
				"",           // B-End inner VLAN
				"0",          // B-End vNIC index
				"no",         // B-End partner config
				"port-b-456", // B-End product UID
				"",           // promo code
				"",           // service key
				"",           // cost centre
			},
			verify: func(t *testing.T, req *megaport.BuyVXCRequest) {
				if assert.NotNil(t, req.AEndConfiguration.VXCOrderMVEConfig) {
					assert.Equal(t, 0, req.AEndConfiguration.NetworkInterfaceIndex)
				}
				if assert.NotNil(t, req.BEndConfiguration.VXCOrderMVEConfig) {
					assert.Equal(t, 0, req.BEndConfiguration.NetworkInterfaceIndex)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := mockPrompts(tc.responses)
			defer cleanup()

			mockSvc := &MockVXCService{}
			cleanupLogin := testutil.SetupLogin(func(c *megaport.Client) {
				c.VXCService = mockSvc
			})
			defer cleanupLogin()

			ctx := context.Background()
			req, err := buildVXCRequestFromPrompt(ctx, mockSvc, true)
			assert.NoError(t, err)
			assert.NotNil(t, req)
			tc.verify(t, req)
		})
	}
}

func TestBuildUpdateVXCRequestFromPrompt(t *testing.T) {
	existingVXC := &megaport.VXC{
		UID:                "vxc-uid-123",
		Name:               "Old VXC",
		RateLimit:          100,
		ContractTermMonths: 12,
		CostCentre:         "CC-001",
		AdminLocked:        false,
		AEndConfiguration: megaport.VXCEndConfiguration{
			UID:  "a-end-uid",
			VLAN: 100,
		},
		BEndConfiguration: megaport.VXCEndConfiguration{
			UID:  "b-end-uid",
			VLAN: 200,
		},
	}

	tests := []struct {
		name      string
		responses []string
		verify    func(t *testing.T, req *megaport.UpdateVXCRequest)
	}{
		{
			name: "skip all",
			responses: []string{
				"no", // update name
				"no", // update rate limit
				"no", // update term
				"no", // update cost centre
				"no", // update shutdown
				"no", // update A-End VLAN
				"no", // update B-End VLAN
				"no", // update A-End inner VLAN
				"no", // update B-End inner VLAN
				"no", // update A-End UID
				"no", // update B-End UID
				"no", // A-End VRouter config
				"no", // B-End VRouter config
			},
			verify: func(t *testing.T, req *megaport.UpdateVXCRequest) {
				assert.Nil(t, req.Name)
				assert.Nil(t, req.RateLimit)
				assert.Nil(t, req.Term)
				assert.Nil(t, req.CostCentre)
				assert.Nil(t, req.Shutdown)
				assert.Nil(t, req.AEndVLAN)
				assert.Nil(t, req.BEndVLAN)
				assert.Nil(t, req.AEndInnerVLAN)
				assert.Nil(t, req.BEndInnerVLAN)
				assert.Nil(t, req.AEndProductUID)
				assert.Nil(t, req.BEndProductUID)
				assert.True(t, req.WaitForUpdate)
				// WaitForTime is set by the caller (UpdateVXC), not the prompt builder
			},
		},
		{
			name: "update name and rate limit",
			responses: []string{
				"yes",          // update name
				"New VXC Name", // new name
				"yes",          // update rate limit
				"500",          // new rate limit
				"no",           // update term
				"no",           // update cost centre
				"no",           // update shutdown
				"no",           // update A-End VLAN
				"no",           // update B-End VLAN
				"no",           // update A-End inner VLAN
				"no",           // update B-End inner VLAN
				"no",           // update A-End UID
				"no",           // update B-End UID
				"no",           // A-End VRouter config
				"no",           // B-End VRouter config
			},
			verify: func(t *testing.T, req *megaport.UpdateVXCRequest) {
				assert.NotNil(t, req.Name)
				assert.Equal(t, "New VXC Name", *req.Name)
				assert.NotNil(t, req.RateLimit)
				assert.Equal(t, 500, *req.RateLimit)
				assert.Nil(t, req.Term)
				assert.Nil(t, req.CostCentre)
				assert.Nil(t, req.Shutdown)
				assert.True(t, req.WaitForUpdate)
				// WaitForTime is set by the caller (UpdateVXC), not the prompt builder
			},
		},
		{
			// Regression: previously the A-End config was assigned to req.BEndPartnerConfig.
			name: "A-End VRouter config goes to AEndPartnerConfig not BEndPartnerConfig",
			responses: []string{
				"no",  // update name
				"no",  // update rate limit
				"no",  // update term
				"no",  // update cost centre
				"no",  // update shutdown
				"no",  // update A-End VLAN
				"no",  // update B-End VLAN
				"no",  // update A-End inner VLAN
				"no",  // update B-End inner VLAN
				"no",  // update A-End UID
				"no",  // update B-End UID
				"yes", // configure A-End VRouter partner config
				"1",   // number of interfaces
				"",    // VLAN (untagged)
				"",    // interface type (default subInterface)
				"no",  // add IP addresses
				"no",  // add IP routes
				"no",  // add NAT IPs
				"no",  // configure BFD
				"no",  // configure BGP
				"no",  // configure B-End VRouter partner config
			},
			verify: func(t *testing.T, req *megaport.UpdateVXCRequest) {
				assert.NotNil(t, req.AEndPartnerConfig, "A-End config must be set")
				assert.Nil(t, req.BEndPartnerConfig, "B-End config must be nil")
				vrouterCfg, ok := req.AEndPartnerConfig.(*megaport.VXCOrderVrouterPartnerConfig)
				assert.True(t, ok)
				assert.Len(t, vrouterCfg.Interfaces, 1)
				assert.Equal(t, -1, vrouterCfg.Interfaces[0].VLAN)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := mockPrompts(tc.responses)
			defer cleanup()

			mockSvc := &MockVXCService{
				GetVXCResponse: existingVXC,
			}
			mockClient := &megaport.Client{VXCService: mockSvc}

			req, err := buildUpdateVXCRequestFromPrompt(context.Background(), mockClient, "vxc-uid-123", true)
			assert.NoError(t, err)
			assert.NotNil(t, req)
			tc.verify(t, req)
		})
	}
}

func TestBuildUpdateVXCRequestFromPromptShutdownStatusDisplay(t *testing.T) {
	responses := []string{
		"no", // update name
		"no", // update rate limit
		"no", // update term
		"no", // update cost centre
		"no", // update shutdown
		"no", // update A-End VLAN
		"no", // update B-End VLAN
		"no", // update A-End inner VLAN
		"no", // update B-End inner VLAN
		"no", // update A-End UID
		"no", // update B-End UID
		"no", // A-End VRouter config
		"no", // B-End VRouter config
	}

	tests := []struct {
		name           string
		shutdown       bool
		adminLocked    bool
		expectedStatus string
	}{
		{
			name:           "shutdown true but not admin locked shows shutdown status",
			shutdown:       true,
			adminLocked:    false,
			expectedStatus: "Current shutdown status: Yes",
		},
		{
			name:           "shutdown false but admin locked shows enabled status",
			shutdown:       false,
			adminLocked:    true,
			expectedStatus: "Current shutdown status: No",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := mockPrompts(responses)
			defer cleanup()

			existingVXC := &megaport.VXC{
				UID:                "vxc-uid-123",
				Name:               "Old VXC",
				RateLimit:          100,
				ContractTermMonths: 12,
				CostCentre:         "CC-001",
				Shutdown:           tc.shutdown,
				AdminLocked:        tc.adminLocked,
				AEndConfiguration: megaport.VXCEndConfiguration{
					UID:  "a-end-uid",
					VLAN: 100,
				},
				BEndConfiguration: megaport.VXCEndConfiguration{
					UID:  "b-end-uid",
					VLAN: 200,
				},
			}

			mockSvc := &MockVXCService{
				GetVXCResponse: existingVXC,
			}
			mockClient := &megaport.Client{VXCService: mockSvc}

			var req *megaport.UpdateVXCRequest
			var err error
			capturedOutput := output.CaptureOutput(func() {
				req, err = buildUpdateVXCRequestFromPrompt(context.Background(), mockClient, "vxc-uid-123", true)
			})

			assert.NoError(t, err)
			assert.NotNil(t, req)
			assert.Contains(t, capturedOutput, tc.expectedStatus)
		})
	}
}

func TestBuildUpdateVXCRequestFromPrompt_TermValidation(t *testing.T) {
	existingVXC := &megaport.VXC{
		UID:                "vxc-uid-123",
		Name:               "Old VXC",
		RateLimit:          100,
		ContractTermMonths: 12,
		CostCentre:         "CC-001",
		AdminLocked:        false,
		AEndConfiguration: megaport.VXCEndConfiguration{
			UID:  "a-end-uid",
			VLAN: 100,
		},
		BEndConfiguration: megaport.VXCEndConfiguration{
			UID:  "b-end-uid",
			VLAN: 200,
		},
	}

	baseResponses := func(termResponses ...string) []string {
		responses := []string{
			"no", // update name
			"no", // update rate limit
			"yes",
		}
		responses = append(responses, termResponses...)
		responses = append(responses,
			"no", // update cost centre
			"no", // update shutdown
			"no", // update A-End VLAN
			"no", // update B-End VLAN
			"no", // update A-End inner VLAN
			"no", // update B-End inner VLAN
			"no", // update A-End UID
			"no", // update B-End UID
			"no", // A-End VRouter config
			"no", // B-End VRouter config
		)
		return responses
	}

	t.Run("term 0 is rejected", func(t *testing.T) {
		cleanup := mockPrompts(baseResponses("0"))
		defer cleanup()

		mockSvc := &MockVXCService{GetVXCResponse: existingVXC}
		mockClient := &megaport.Client{VXCService: mockSvc}

		_, err := buildUpdateVXCRequestFromPrompt(context.Background(), mockClient, "vxc-uid-123", true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid contract term")
	})

	for _, term := range validation.ValidContractTerms {
		t.Run(fmt.Sprintf("term %d is accepted", term), func(t *testing.T) {
			cleanup := mockPrompts(baseResponses(fmt.Sprintf("%d", term)))
			defer cleanup()

			mockSvc := &MockVXCService{GetVXCResponse: existingVXC}
			mockClient := &megaport.Client{VXCService: mockSvc}

			req, err := buildUpdateVXCRequestFromPrompt(context.Background(), mockClient, "vxc-uid-123", true)
			require.NoError(t, err)
			require.NotNil(t, req)
			require.NotNil(t, req.Term)
			assert.Equal(t, term, *req.Term)
		})
	}
}
