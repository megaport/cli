package nat_gateway

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/testutil"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Command wiring ----

func TestAddCommandsTo(t *testing.T) {
	root := &cobra.Command{Use: "megaport-cli"}
	AddCommandsTo(root)

	var natCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "nat-gateway" {
			natCmd = cmd
			break
		}
	}
	require.NotNil(t, natCmd, "nat-gateway command should be registered")

	expectedSubs := []string{"get", "list", "create", "update", "delete", "list-sessions", "telemetry"}
	subs := make(map[string]bool)
	for _, sub := range natCmd.Commands() {
		subs[sub.Use] = true
	}
	for _, name := range expectedSubs {
		assert.True(t, subs[name], "nat-gateway should have subcommand %q", name)
	}
}

func TestModule(t *testing.T) {
	m := NewModule()
	assert.Equal(t, "nat-gateway", m.Name())

	root := &cobra.Command{Use: "megaport-cli"}
	m.RegisterCommands(root)

	found := false
	for _, cmd := range root.Commands() {
		if cmd.Use == "nat-gateway" {
			found = true
			break
		}
	}
	assert.True(t, found, "RegisterCommands should add nat-gateway command")
}

// ---- Login error paths ----

func TestGetNATGateway_LoginError(t *testing.T) {
	defer testutil.SetupLoginError(fmt.Errorf("auth failed"))()

	cmd := newTestCmd("get")
	err := GetNATGateway(cmd, []string{"uid-123"}, true, "table")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth failed")
}

func TestListNATGateways_LoginError(t *testing.T) {
	defer testutil.SetupLoginError(fmt.Errorf("auth failed"))()

	cmd := newTestCmd("list")
	err := ListNATGateways(cmd, nil, true, "table")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth failed")
}

func TestUpdateNATGateway_LoginError(t *testing.T) {
	defer testutil.SetupLoginError(fmt.Errorf("auth failed"))()

	cmd := newTestCmd("update")
	require.NoError(t, cmd.Flags().Set("name", "GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))

	err := UpdateNATGateway(cmd, []string{"uid-upd"}, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth failed")
}

func TestDeleteNATGateway_LoginError(t *testing.T) {
	defer testutil.SetupLoginError(fmt.Errorf("auth failed"))()

	cmd := newTestCmd("delete")
	require.NoError(t, cmd.Flags().Set("force", "true"))

	err := DeleteNATGateway(cmd, []string{"uid-del"}, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth failed")
}

func TestListNATGatewaySessions_LoginError(t *testing.T) {
	defer testutil.SetupLoginError(fmt.Errorf("auth failed"))()

	cmd := newTestCmd("list-sessions")
	err := ListNATGatewaySessions(cmd, nil, true, "table")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth failed")
}

func TestGetNATGatewayTelemetry_LoginError(t *testing.T) {
	defer testutil.SetupLoginError(fmt.Errorf("auth failed"))()

	cmd := newTestCmd("telemetry")
	require.NoError(t, cmd.Flags().Set("types", "BITS"))
	require.NoError(t, cmd.Flags().Set("days", "7"))

	err := GetNATGatewayTelemetry(cmd, []string{"uid-tel"}, true, "table")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth failed")
}

// ---- Additional action coverage ----

func TestGetNATGateway_NilResult(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	// Override the getNATGatewayFunc to return nil explicitly
	origGet := getNATGatewayFunc
	getNATGatewayFunc = func(_ context.Context, _ *megaport.Client, _ string) (*megaport.NATGateway, error) {
		return nil, nil
	}
	defer func() { getNATGatewayFunc = origGet }()

	cmd := newTestCmd("get")
	err := GetNATGateway(cmd, []string{"uid-missing"}, true, "table")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no NAT Gateway found")
}

func TestListNATGatewaySessions_Empty(t *testing.T) {
	mock := &MockNATGatewayService{
		SessionsResult: []*megaport.NATGatewaySession{},
	}
	defer setupMockNATGateway(mock)()

	t.Run("table prints info message", func(t *testing.T) {
		cmd := newTestCmd("list-sessions")
		out := output.CaptureOutput(func() {
			assert.NoError(t, ListNATGatewaySessions(cmd, nil, true, "table"))
		})
		assert.Contains(t, out, "No NAT Gateway session options found")
	})

	t.Run("json emits empty array", func(t *testing.T) {
		defer output.SetOutputFormat("table")
		cmd := newTestCmd("list-sessions")
		out := output.CaptureOutput(func() {
			assert.NoError(t, ListNATGatewaySessions(cmd, nil, true, "json"))
		})
		assert.Equal(t, "[]", strings.TrimSpace(out))
	})
}

func TestListNATGateways_IncludeInactive(t *testing.T) {
	mock := &MockNATGatewayService{
		ListResult: []*megaport.NATGateway{
			{ProductUID: "uid-1", ProductName: "Live GW", ProvisioningStatus: "LIVE"},
			{ProductUID: "uid-2", ProductName: "Decom GW", ProvisioningStatus: "DECOMMISSIONED"},
			{ProductUID: "uid-3", ProductName: "Cancelled GW", ProvisioningStatus: "CANCELLED"},
		},
	}
	defer setupMockNATGateway(mock)()

	// Without include-inactive, decommissioned/cancelled are filtered out
	cmd := newTestCmd("list")
	err := ListNATGateways(cmd, nil, true, "json")
	assert.NoError(t, err)

	// With include-inactive, all are returned
	cmd2 := newTestCmd("list")
	require.NoError(t, cmd2.Flags().Set("include-inactive", "true"))
	err = ListNATGateways(cmd2, nil, true, "json")
	assert.NoError(t, err)
}

func TestListNATGateways_Empty(t *testing.T) {
	mock := &MockNATGatewayService{
		ListResult: []*megaport.NATGateway{},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("list")
	err := ListNATGateways(cmd, nil, true, "table")
	assert.NoError(t, err)
}

func TestListNATGateways_FilterByLocationID(t *testing.T) {
	mock := &MockNATGatewayService{
		ListResult: []*megaport.NATGateway{
			{ProductUID: "uid-1", ProductName: "GW 1", LocationID: 100, ProvisioningStatus: "LIVE"},
			{ProductUID: "uid-2", ProductName: "GW 2", LocationID: 200, ProvisioningStatus: "LIVE"},
		},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("list")
	require.NoError(t, cmd.Flags().Set("location-id", "100"))

	err := ListNATGateways(cmd, nil, true, "json")
	assert.NoError(t, err)
}

func TestListNATGateways_WithLimit(t *testing.T) {
	mock := &MockNATGatewayService{
		ListResult: []*megaport.NATGateway{
			{ProductUID: "uid-1", ProductName: "GW 1", ProvisioningStatus: "LIVE"},
			{ProductUID: "uid-2", ProductName: "GW 2", ProvisioningStatus: "LIVE"},
			{ProductUID: "uid-3", ProductName: "GW 3", ProvisioningStatus: "LIVE"},
		},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("list")
	require.NoError(t, cmd.Flags().Set("limit", "2"))

	err := ListNATGateways(cmd, nil, true, "json")
	assert.NoError(t, err)
}

func TestUpdateNATGateway_GetError(t *testing.T) {
	mock := &MockNATGatewayService{
		GetErr: fmt.Errorf("not found"),
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("update")
	require.NoError(t, cmd.Flags().Set("name", "GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))

	err := UpdateNATGateway(cmd, []string{"uid-err"}, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdateNATGateway_InvalidJSON(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("update")
	require.NoError(t, cmd.Flags().Set("json", "{invalid}"))

	err := UpdateNATGateway(cmd, []string{"uid-err"}, true)
	assert.Error(t, err)
}

func TestGetNATGatewayTelemetry_InvalidToTime(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("telemetry")
	require.NoError(t, cmd.Flags().Set("types", "BITS"))
	require.NoError(t, cmd.Flags().Set("from", "2024-01-01T00:00:00Z"))
	require.NoError(t, cmd.Flags().Set("to", "not-a-time"))

	err := GetNATGatewayTelemetry(cmd, []string{"uid-tel"}, true, "table")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --to time")
}

func TestGetNATGatewayTelemetry_NoTimeWindow(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("telemetry")
	require.NoError(t, cmd.Flags().Set("types", "BITS"))
	// Neither --days nor --from/--to is set

	err := GetNATGatewayTelemetry(cmd, []string{"uid-tel"}, true, "table")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "time window is required")
}

func TestGetNATGatewayTelemetry_MultipleTypes(t *testing.T) {
	mock := &MockNATGatewayService{
		TelemetryResult: &megaport.ServiceTelemetryResponse{
			ServiceUID: "uid-tel",
			Data: []*megaport.TelemetryMetricData{
				{
					Type:    "BITS",
					Subtype: "ingress",
					Unit:    megaport.TelemetryUnit{Name: "bps"},
					Samples: []megaport.TelemetrySample{{Timestamp: 1700000000000, Value: 100.5}},
				},
				{
					Type:    "PACKETS",
					Subtype: "egress",
					Unit:    megaport.TelemetryUnit{Name: "pps"},
					Samples: []megaport.TelemetrySample{{Timestamp: 1700000000000, Value: 50.0}},
				},
			},
		},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("telemetry")
	require.NoError(t, cmd.Flags().Set("types", "BITS,PACKETS"))
	require.NoError(t, cmd.Flags().Set("days", "30"))

	err := GetNATGatewayTelemetry(cmd, []string{"uid-tel"}, true, "table")
	assert.NoError(t, err)
	assert.Equal(t, []string{"BITS", "PACKETS"}, mock.CapturedTelemetryReq.Types)
}

// ---- Output helpers ----

func TestPrintNATGateways(t *testing.T) {
	gateways := []*megaport.NATGateway{
		{
			ProductUID: "uid-1", ProductName: "GW 1",
			LocationID: 100, Speed: 1000, Term: 12,
			ProvisioningStatus: "LIVE",
		},
	}
	err := printNATGateways(gateways, "json", true)
	assert.NoError(t, err)
}

func TestPrintNATGatewaySessions(t *testing.T) {
	sessions := []*megaport.NATGatewaySession{
		{SpeedMbps: 1000, SessionCount: []int{100, 200}},
		{SpeedMbps: 2000, SessionCount: []int{500}},
		nil, // nil entries should be skipped
	}
	err := printNATGatewaySessions(sessions, "json", true)
	assert.NoError(t, err)
}

func TestPrintNATGatewayTelemetry(t *testing.T) {
	resp := &megaport.ServiceTelemetryResponse{
		Data: []*megaport.TelemetryMetricData{
			{
				Type:    "BITS",
				Subtype: "ingress",
				Unit:    megaport.TelemetryUnit{Name: "bps"},
				Samples: []megaport.TelemetrySample{
					{Timestamp: 1700000000000, Value: 1234.5},
					{Timestamp: 1700000060000, Value: 5678.9},
				},
			},
			nil, // nil entries should be skipped
		},
	}
	err := printNATGatewayTelemetry(resp, "json", true)
	assert.NoError(t, err)
}

func TestPrintNATGatewayTelemetry_Empty(t *testing.T) {
	resp := &megaport.ServiceTelemetryResponse{}
	err := printNATGatewayTelemetry(resp, "table", true)
	assert.NoError(t, err)
}

func TestDisplayNATGatewayChanges(t *testing.T) {
	original := &megaport.NATGateway{
		ProductName: "Old Name", Speed: 1000, Term: 12,
		Config: megaport.NATGatewayNetworkConfig{SessionCount: 100, DiversityZone: "blue"},
	}
	updated := &megaport.NATGateway{
		ProductName: "New Name", Speed: 2000, Term: 24,
		Config: megaport.NATGatewayNetworkConfig{SessionCount: 200, DiversityZone: "red"},
	}
	// Should not panic
	displayNATGatewayChanges(original, updated, true)
}

func TestDisplayNATGatewayChanges_NilInputs(t *testing.T) {
	// Should not panic with nil inputs
	displayNATGatewayChanges(nil, nil, true)
	displayNATGatewayChanges(&megaport.NATGateway{}, nil, true)
	displayNATGatewayChanges(nil, &megaport.NATGateway{}, true)
}

func TestExportNATGatewayConfig_MinimalFields(t *testing.T) {
	gw := &megaport.NATGateway{
		ProductName: "Basic GW",
		Term:        1,
		Speed:       100,
		LocationID:  10,
	}
	cfg := exportNATGatewayConfig(gw)
	assert.Equal(t, "Basic GW", cfg["name"])
	assert.Equal(t, 1, cfg["term"])
	assert.Equal(t, 100, cfg["speed"])
	assert.Equal(t, 10, cfg["locationId"])
	// No sessionCount, diversityZone, or asn when zero/empty
	_, hasSessions := cfg["sessionCount"]
	assert.False(t, hasSessions)
	_, hasDZ := cfg["diversityZone"]
	assert.False(t, hasDZ)
	_, hasASN := cfg["asn"]
	assert.False(t, hasASN)
}

func TestExportNATGatewayConfig_WithASN(t *testing.T) {
	gw := &megaport.NATGateway{
		ProductName: "ASN GW",
		Term:        12,
		Speed:       1000,
		LocationID:  42,
		Config: megaport.NATGatewayNetworkConfig{
			ASN: 65000,
		},
	}
	cfg := exportNATGatewayConfig(gw)
	assert.Equal(t, 65000, cfg["asn"])
}

// ---- Input processing ----

func TestProcessJSONUpdateNATGatewayInput(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		uid     string
		wantErr bool
		check   func(*megaport.UpdateNATGatewayRequest)
	}{
		{
			name: "valid",
			json: `{"name":"Updated GW","term":12,"speed":1000,"locationId":1}`,
			uid:  "uid-upd",
			check: func(r *megaport.UpdateNATGatewayRequest) {
				assert.Equal(t, "uid-upd", r.ProductUID)
				assert.Equal(t, "Updated GW", r.ProductName)
				assert.Equal(t, 12, r.Term)
			},
		},
		{
			name: "partial update - name only",
			json: `{"name":"Just Name"}`,
			uid:  "uid-upd",
			check: func(r *megaport.UpdateNATGatewayRequest) {
				assert.Equal(t, "Just Name", r.ProductName)
				assert.Equal(t, 0, r.Speed, "unset fields should be zero before merge")
			},
		},
		{
			name: "partial update - speed only",
			json: `{"speed":2000}`,
			uid:  "uid-upd",
			check: func(r *megaport.UpdateNATGatewayRequest) {
				assert.Equal(t, 2000, r.Speed)
				assert.Equal(t, "", r.ProductName, "unset name should be empty before merge")
			},
		},
		{
			name:    "invalid JSON",
			json:    `{not valid}`,
			uid:     "uid-upd",
			wantErr: true,
		},
		{
			name: "with resource tags",
			json: `{"name":"GW","term":12,"speed":1000,"locationId":1,"resourceTags":{"env":"prod"}}`,
			uid:  "uid-upd",
			check: func(r *megaport.UpdateNATGatewayRequest) {
				assert.Len(t, r.ResourceTags, 1)
			},
		},
		{
			name: "with session count and diversity zone",
			json: `{"name":"GW","term":12,"speed":1000,"locationId":1,"sessionCount":500,"diversityZone":"blue"}`,
			uid:  "uid-upd",
			check: func(r *megaport.UpdateNATGatewayRequest) {
				assert.Equal(t, 500, r.Config.SessionCount)
				assert.Equal(t, "blue", r.Config.DiversityZone)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _, err := processJSONUpdateNATGatewayInput(tt.json, "", tt.uid)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.check != nil {
					tt.check(req)
				}
			}
		})
	}
}

func TestProcessJSONUpdateNATGatewayInputBoolPresence(t *testing.T) {
	t.Run("autoRenewTerm absent returns explicit=false", func(t *testing.T) {
		_, explicit, err := processJSONUpdateNATGatewayInput(`{"name":"GW"}`, "", "uid")
		autoRenewExplicit := explicit.AutoRenewTerm
		assert.NoError(t, err)
		assert.False(t, autoRenewExplicit)
	})
	t.Run("autoRenewTerm present true returns explicit=true and value=true", func(t *testing.T) {
		req, explicit, err := processJSONUpdateNATGatewayInput(`{"autoRenewTerm":true}`, "", "uid")
		autoRenewExplicit := explicit.AutoRenewTerm
		assert.NoError(t, err)
		assert.True(t, autoRenewExplicit)
		assert.True(t, req.AutoRenewTerm)
	})
	t.Run("autoRenewTerm present false returns explicit=true and value=false", func(t *testing.T) {
		req, explicit, err := processJSONUpdateNATGatewayInput(`{"autoRenewTerm":false}`, "", "uid")
		autoRenewExplicit := explicit.AutoRenewTerm
		assert.NoError(t, err)
		assert.True(t, autoRenewExplicit, "explicit false must be tracked so mergeUpdateDefaults does not override it")
		assert.False(t, req.AutoRenewTerm)
	})
	t.Run("bgpShutdownDefault absent returns explicit=false", func(t *testing.T) {
		_, explicit, err := processJSONUpdateNATGatewayInput(`{"name":"GW"}`, "", "uid")
		bgpExplicit := explicit.BGPShutdownDefault
		assert.NoError(t, err)
		assert.False(t, bgpExplicit)
	})
	t.Run("bgpShutdownDefault present false returns explicit=true and value=false", func(t *testing.T) {
		req, explicit, err := processJSONUpdateNATGatewayInput(`{"bgpShutdownDefault":false}`, "", "uid")
		bgpExplicit := explicit.BGPShutdownDefault
		assert.NoError(t, err)
		assert.True(t, bgpExplicit, "explicit false must be tracked so mergeUpdateDefaults does not override it")
		assert.False(t, req.Config.BGPShutdownDefault)
	})
	t.Run("sessionCount absent returns explicit=false", func(t *testing.T) {
		_, explicit, err := processJSONUpdateNATGatewayInput(`{"name":"GW"}`, "", "uid")
		assert.NoError(t, err)
		assert.False(t, explicit.SessionCount)
	})
	t.Run("sessionCount present zero returns explicit=true", func(t *testing.T) {
		req, explicit, err := processJSONUpdateNATGatewayInput(`{"sessionCount":0}`, "", "uid")
		assert.NoError(t, err)
		assert.True(t, explicit.SessionCount, "explicit 0 must be tracked so mergeUpdateDefaults does not override it")
		assert.Equal(t, 0, req.Config.SessionCount)
	})
	t.Run("diversityZone absent returns explicit=false", func(t *testing.T) {
		_, explicit, err := processJSONUpdateNATGatewayInput(`{"name":"GW"}`, "", "uid")
		assert.NoError(t, err)
		assert.False(t, explicit.DiversityZone)
	})
	t.Run("diversityZone present empty returns explicit=true", func(t *testing.T) {
		req, explicit, err := processJSONUpdateNATGatewayInput(`{"diversityZone":""}`, "", "uid")
		assert.NoError(t, err)
		assert.True(t, explicit.DiversityZone, "explicit empty string must be tracked so mergeUpdateDefaults does not override it")
		assert.Equal(t, "", req.Config.DiversityZone)
	})
}

func TestMergeUpdateDefaultsExplicitBools(t *testing.T) {
	original := &megaport.NATGateway{
		ProductName:   "Original",
		LocationID:    1,
		Speed:         1000,
		Term:          12,
		AutoRenewTerm: true,
		Config: megaport.NATGatewayNetworkConfig{
			BGPShutdownDefault: true,
			SessionCount:       50000,
			DiversityZone:      "blue",
		},
	}

	t.Run("explicit false for AutoRenewTerm preserved", func(t *testing.T) {
		req := &megaport.UpdateNATGatewayRequest{ProductUID: "uid", AutoRenewTerm: false}
		mergeUpdateDefaults(req, original, updateExplicitFields{AutoRenewTerm: true})
		assert.False(t, req.AutoRenewTerm, "explicit false should not be overridden by original true")
	})

	t.Run("explicit false for BGPShutdownDefault preserved", func(t *testing.T) {
		req := &megaport.UpdateNATGatewayRequest{ProductUID: "uid", Config: megaport.NATGatewayNetworkConfig{BGPShutdownDefault: false}}
		mergeUpdateDefaults(req, original, updateExplicitFields{BGPShutdownDefault: true})
		assert.False(t, req.Config.BGPShutdownDefault, "explicit false should not be overridden by original true")
	})

	t.Run("non-explicit false for AutoRenewTerm inherits from original", func(t *testing.T) {
		req := &megaport.UpdateNATGatewayRequest{ProductUID: "uid"}
		mergeUpdateDefaults(req, original, updateExplicitFields{})
		assert.True(t, req.AutoRenewTerm, "non-explicit false should inherit from original")
		assert.True(t, req.Config.BGPShutdownDefault, "non-explicit false should inherit from original")
	})

	t.Run("explicit zero for SessionCount preserved", func(t *testing.T) {
		req := &megaport.UpdateNATGatewayRequest{ProductUID: "uid", Config: megaport.NATGatewayNetworkConfig{SessionCount: 0}}
		mergeUpdateDefaults(req, original, updateExplicitFields{SessionCount: true})
		assert.Equal(t, 0, req.Config.SessionCount, "explicit 0 should not be overridden by original 50000")
	})

	t.Run("non-explicit zero for SessionCount inherits from original", func(t *testing.T) {
		req := &megaport.UpdateNATGatewayRequest{ProductUID: "uid"}
		mergeUpdateDefaults(req, original, updateExplicitFields{})
		assert.Equal(t, 50000, req.Config.SessionCount, "non-explicit 0 should inherit from original")
	})

	t.Run("explicit empty for DiversityZone preserved", func(t *testing.T) {
		req := &megaport.UpdateNATGatewayRequest{ProductUID: "uid", Config: megaport.NATGatewayNetworkConfig{DiversityZone: ""}}
		mergeUpdateDefaults(req, original, updateExplicitFields{DiversityZone: true})
		assert.Equal(t, "", req.Config.DiversityZone, "explicit empty should not be overridden by original 'blue'")
	})

	t.Run("non-explicit empty for DiversityZone inherits from original", func(t *testing.T) {
		req := &megaport.UpdateNATGatewayRequest{ProductUID: "uid"}
		mergeUpdateDefaults(req, original, updateExplicitFields{})
		assert.Equal(t, "blue", req.Config.DiversityZone, "non-explicit empty should inherit from original")
	})
}

func TestProcessFlagCreateNATGatewayInput(t *testing.T) {
	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("name", "Flag GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "123"))
	require.NoError(t, cmd.Flags().Set("session-count", "200"))
	require.NoError(t, cmd.Flags().Set("diversity-zone", "blue"))
	require.NoError(t, cmd.Flags().Set("promo-code", "PROMO1"))
	require.NoError(t, cmd.Flags().Set("auto-renew", "true"))

	req, err := processFlagCreateNATGatewayInput(cmd)
	assert.NoError(t, err)
	assert.Equal(t, "Flag GW", req.ProductName)
	assert.Equal(t, 12, req.Term)
	assert.Equal(t, 1000, req.Speed)
	assert.Equal(t, 123, req.LocationID)
	assert.Equal(t, 200, req.Config.SessionCount)
	assert.Equal(t, "blue", req.Config.DiversityZone)
	assert.Equal(t, "PROMO1", req.PromoCode)
	assert.True(t, req.AutoRenewTerm)
}

func TestProcessFlagCreateNATGatewayInput_WithResourceTags(t *testing.T) {
	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("name", "Tagged GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))
	require.NoError(t, cmd.Flags().Set("resource-tags", `{"env":"prod"}`))

	req, err := processFlagCreateNATGatewayInput(cmd)
	assert.NoError(t, err)
	assert.Len(t, req.ResourceTags, 1)
}

func TestProcessFlagCreateNATGatewayInput_InvalidResourceTags(t *testing.T) {
	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("name", "GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))
	require.NoError(t, cmd.Flags().Set("resource-tags", "not-json"))

	_, err := processFlagCreateNATGatewayInput(cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resource tags")
}

func TestProcessFlagCreateNATGatewayInput_ValidationError(t *testing.T) {
	cmd := newTestCmd("create")
	// Missing name
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))

	_, err := processFlagCreateNATGatewayInput(cmd)
	assert.Error(t, err)
}

func TestProcessFlagUpdateNATGatewayInput(t *testing.T) {
	cmd := newTestCmd("update")
	require.NoError(t, cmd.Flags().Set("name", "Updated GW"))
	require.NoError(t, cmd.Flags().Set("term", "24"))
	require.NoError(t, cmd.Flags().Set("speed", "2000"))
	require.NoError(t, cmd.Flags().Set("location-id", "50"))
	require.NoError(t, cmd.Flags().Set("session-count", "300"))
	require.NoError(t, cmd.Flags().Set("diversity-zone", "red"))

	req, err := processFlagUpdateNATGatewayInput(cmd, "uid-upd")
	assert.NoError(t, err)
	assert.Equal(t, "uid-upd", req.ProductUID)
	assert.Equal(t, "Updated GW", req.ProductName)
	assert.Equal(t, 24, req.Term)
	assert.Equal(t, 2000, req.Speed)
	assert.Equal(t, 50, req.LocationID)
	assert.Equal(t, 300, req.Config.SessionCount)
	assert.Equal(t, "red", req.Config.DiversityZone)
}

func TestProcessFlagUpdateNATGatewayInput_PartialUpdate(t *testing.T) {
	cmd := newTestCmd("update")
	// Only set name — other fields should be zero (filled by merge later)
	require.NoError(t, cmd.Flags().Set("name", "New Name"))

	req, err := processFlagUpdateNATGatewayInput(cmd, "uid-upd")
	assert.NoError(t, err)
	assert.Equal(t, "New Name", req.ProductName)
	assert.Equal(t, 0, req.Speed, "speed should be zero before merge")
	assert.Equal(t, 0, req.Term, "term should be zero before merge")
}

func TestMergeUpdateDefaults(t *testing.T) {
	t.Run("fills zero fields from original", func(t *testing.T) {
		req := &megaport.UpdateNATGatewayRequest{
			ProductUID:  "uid-1",
			ProductName: "New Name",
		}
		original := &megaport.NATGateway{
			ProductName: "Old Name",
			LocationID:  100,
			Speed:       2000,
			Term:        24,
		}
		mergeUpdateDefaults(req, original, updateExplicitFields{})
		assert.Equal(t, "New Name", req.ProductName, "explicitly set name should not be overwritten")
		assert.Equal(t, 100, req.LocationID, "should inherit from original")
		assert.Equal(t, 2000, req.Speed, "should inherit from original")
		assert.Equal(t, 24, req.Term, "should inherit from original")
	})

	t.Run("does not overwrite provided fields", func(t *testing.T) {
		req := &megaport.UpdateNATGatewayRequest{
			ProductUID:  "uid-1",
			ProductName: "New Name",
			LocationID:  200,
			Speed:       5000,
			Term:        12,
		}
		original := &megaport.NATGateway{
			ProductName: "Old Name",
			LocationID:  100,
			Speed:       2000,
			Term:        24,
		}
		mergeUpdateDefaults(req, original, updateExplicitFields{})
		assert.Equal(t, "New Name", req.ProductName)
		assert.Equal(t, 200, req.LocationID)
		assert.Equal(t, 5000, req.Speed)
		assert.Equal(t, 12, req.Term)
	})

	t.Run("fills zero Config fields from original", func(t *testing.T) {
		req := &megaport.UpdateNATGatewayRequest{
			ProductUID:  "uid-1",
			ProductName: "Name",
			LocationID:  100,
			Speed:       1000,
			Term:        12,
		}
		original := &megaport.NATGateway{
			ProductName: "Name",
			LocationID:  100,
			Speed:       1000,
			Term:        12,
			Config: megaport.NATGatewayNetworkConfig{
				SessionCount:  50000,
				DiversityZone: "blue",
				ASN:           133937,
			},
		}
		mergeUpdateDefaults(req, original, updateExplicitFields{})
		assert.Equal(t, 50000, req.Config.SessionCount, "should inherit SessionCount from original")
		assert.Equal(t, "blue", req.Config.DiversityZone, "should inherit DiversityZone from original")
		assert.Equal(t, 133937, req.Config.ASN, "should inherit ASN from original")
	})

	t.Run("does not overwrite provided Config fields", func(t *testing.T) {
		req := &megaport.UpdateNATGatewayRequest{
			ProductUID:  "uid-1",
			ProductName: "Name",
			LocationID:  100,
			Speed:       1000,
			Term:        12,
			Config: megaport.NATGatewayNetworkConfig{
				SessionCount:  100000,
				DiversityZone: "red",
				ASN:           65000,
			},
		}
		original := &megaport.NATGateway{
			ProductName: "Name",
			LocationID:  100,
			Speed:       1000,
			Term:        12,
			Config: megaport.NATGatewayNetworkConfig{
				SessionCount:  50000,
				DiversityZone: "blue",
				ASN:           133937,
			},
		}
		mergeUpdateDefaults(req, original, updateExplicitFields{})
		assert.Equal(t, 100000, req.Config.SessionCount, "explicitly set SessionCount should not be overwritten")
		assert.Equal(t, "red", req.Config.DiversityZone, "explicitly set DiversityZone should not be overwritten")
		assert.Equal(t, 65000, req.Config.ASN, "explicitly set ASN should not be overwritten")
	})

	t.Run("inherits bool fields from original when false", func(t *testing.T) {
		req := &megaport.UpdateNATGatewayRequest{
			ProductUID:  "uid-1",
			ProductName: "Name",
			LocationID:  100,
			Speed:       1000,
			Term:        12,
		}
		original := &megaport.NATGateway{
			ProductName:   "Name",
			LocationID:    100,
			Speed:         1000,
			Term:          12,
			AutoRenewTerm: true,
			Config: megaport.NATGatewayNetworkConfig{
				BGPShutdownDefault: true,
			},
		}
		mergeUpdateDefaults(req, original, updateExplicitFields{})
		assert.True(t, req.AutoRenewTerm, "should inherit AutoRenewTerm from original when not set")
		assert.True(t, req.Config.BGPShutdownDefault, "should inherit BGPShutdownDefault from original when not set")
	})

	t.Run("nil original is safe", func(t *testing.T) {
		req := &megaport.UpdateNATGatewayRequest{
			ProductUID:  "uid-1",
			ProductName: "Name",
		}
		mergeUpdateDefaults(req, nil, updateExplicitFields{})
		assert.Equal(t, "Name", req.ProductName)
	})
}

// ---- MockNATGatewayService.Reset ----

func TestMockNATGatewayService_Reset(t *testing.T) {
	mock := &MockNATGatewayService{
		CreateErr:         fmt.Errorf("err"),
		ListErr:           fmt.Errorf("err"),
		GetErr:            fmt.Errorf("err"),
		UpdateErr:         fmt.Errorf("err"),
		DeleteErr:         fmt.Errorf("err"),
		SessionsErr:       fmt.Errorf("err"),
		TelemetryErr:      fmt.Errorf("err"),
		CapturedDeleteUID: "uid",
		CapturedGetUID:    "uid",
	}
	mock.Reset()
	assert.Nil(t, mock.CreateErr)
	assert.Nil(t, mock.ListErr)
	assert.Nil(t, mock.GetErr)
	assert.Nil(t, mock.UpdateErr)
	assert.Nil(t, mock.DeleteErr)
	assert.Nil(t, mock.SessionsErr)
	assert.Nil(t, mock.TelemetryErr)
	assert.Empty(t, mock.CapturedDeleteUID)
	assert.Empty(t, mock.CapturedGetUID)
}

func TestListNATGatewaySessions_JSONOutput(t *testing.T) {
	mock := &MockNATGatewayService{
		SessionsResult: []*megaport.NATGatewaySession{
			{SpeedMbps: 1000, SessionCount: []int{100, 200, 500}},
		},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("list-sessions")
	err := ListNATGatewaySessions(cmd, nil, true, "json")
	assert.NoError(t, err)
}

func TestCreateNATGateway_JSONWithSessionCount(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("json", `{"name":"GW","term":12,"speed":1000,"locationId":1,"sessionCount":500,"diversityZone":"blue","autoRenewTerm":true}`))
	require.NoError(t, cmd.Flags().Set("yes", "true"))

	err := CreateNATGateway(cmd, nil, true)
	assert.NoError(t, err)
	require.NotNil(t, mock.CapturedCreateReq)
	assert.Equal(t, 500, mock.CapturedCreateReq.Config.SessionCount)
	assert.Equal(t, "blue", mock.CapturedCreateReq.Config.DiversityZone)
	assert.True(t, mock.CapturedCreateReq.AutoRenewTerm)
}

func TestCreateNATGateway_FlagsWithYes(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("name", "GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))
	require.NoError(t, cmd.Flags().Set("yes", "true"))

	err := CreateNATGateway(cmd, nil, true)
	assert.NoError(t, err)
}

func TestGetNATGateway_JSONOutput(t *testing.T) {
	mock := &MockNATGatewayService{
		GetResult: &megaport.NATGateway{
			ProductUID: "uid-123", ProductName: "Test GW",
			LocationID: 100, Speed: 1000, Term: 12,
			ProvisioningStatus: "LIVE",
		},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("get")
	err := GetNATGateway(cmd, []string{"uid-123"}, true, "table")
	assert.NoError(t, err)
}

func TestCreateNATGateway_JSONMissingSpeed(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("json", `{"name":"GW","term":12,"locationId":1}`))

	err := CreateNATGateway(cmd, nil, true)
	assert.Error(t, err)
}

// TestUpdateFlagExplicit_SessionCountZero verifies that --session-count 0 is treated
// as an explicit zero and not overwritten by mergeUpdateDefaults.
func TestUpdateFlagExplicit_SessionCountZero(t *testing.T) {
	original := &megaport.NATGateway{
		ProductName: "GW",
		LocationID:  1,
		Speed:       1000,
		Term:        12,
		Config:      megaport.NATGatewayNetworkConfig{SessionCount: 50000, DiversityZone: "blue"},
	}

	cmd := newTestCmd("update")
	require.NoError(t, cmd.Flags().Set("session-count", "0"))

	req, err := processFlagUpdateNATGatewayInput(cmd, "uid-1")
	require.NoError(t, err)

	explicit := updateExplicitFields{
		SessionCount: cmd.Flags().Changed("session-count"),
	}

	mergeUpdateDefaults(req, original, explicit)

	assert.Equal(t, 0, req.Config.SessionCount, "--session-count 0 should not be overwritten by original 50000")
	assert.Equal(t, "blue", req.Config.DiversityZone, "DiversityZone should inherit from original when not set")
}

// TestUpdateFlagExplicit_DiversityZoneEmpty verifies that --diversity-zone ""
// is treated as an explicit empty string and not overwritten by mergeUpdateDefaults.
func TestUpdateFlagExplicit_DiversityZoneEmpty(t *testing.T) {
	original := &megaport.NATGateway{
		ProductName: "GW",
		LocationID:  1,
		Speed:       1000,
		Term:        12,
		Config:      megaport.NATGatewayNetworkConfig{SessionCount: 50000, DiversityZone: "blue"},
	}

	cmd := newTestCmd("update")
	require.NoError(t, cmd.Flags().Set("diversity-zone", ""))

	req, err := processFlagUpdateNATGatewayInput(cmd, "uid-1")
	require.NoError(t, err)

	explicit := updateExplicitFields{
		DiversityZone: cmd.Flags().Changed("diversity-zone"),
	}

	mergeUpdateDefaults(req, original, explicit)

	assert.Equal(t, "", req.Config.DiversityZone, "--diversity-zone '' should not be overwritten by original 'blue'")
	assert.Equal(t, 50000, req.Config.SessionCount, "SessionCount should inherit from original when not set")
}
