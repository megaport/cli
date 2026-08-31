package nat_gateway

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/megaport/megaport-cli/internal/testutil"
	"github.com/megaport/megaport-cli/internal/utils"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMockNATGateway wires a MockNATGatewayService into the login func.
func setupMockNATGateway(svc *MockNATGatewayService) func() {
	return testutil.SetupLogin(func(c *megaport.Client) {
		c.NATGatewayService = svc
	})
}

func newTestCmd(use string) *cobra.Command {
	cmd := &cobra.Command{Use: use}
	cmd.Flags().String("json", "", "")
	cmd.Flags().String("json-file", "", "")
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().Bool("yes", false, "")
	cmd.Flags().Bool("force", false, "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().Int("term", 0, "")
	cmd.Flags().Int("speed", 0, "")
	cmd.Flags().Int("location-id", 0, "")
	cmd.Flags().Int("session-count", 0, "")
	cmd.Flags().Int("asn", 0, "")
	cmd.Flags().String("diversity-zone", "", "")
	cmd.Flags().String("promo-code", "", "")
	cmd.Flags().String("service-level-reference", "", "")
	cmd.Flags().Bool("auto-renew", false, "")
	cmd.Flags().String("resource-tags", "", "")
	cmd.Flags().String("resource-tags-file", "", "")
	cmd.Flags().String("types", "", "")
	cmd.Flags().Int("days", 0, "")
	cmd.Flags().String("from", "", "")
	cmd.Flags().String("to", "", "")
	cmd.Flags().Bool("export", false, "")
	cmd.Flags().Bool("watch", false, "")
	cmd.Flags().Duration("interval", 0, "")
	cmd.Flags().Bool("include-inactive", false, "")
	cmd.Flags().Int("limit", 0, "")
	cmd.Flags().String("output", "table", "")
	cmd.Flags().Duration("timeout", 0, "")
	return cmd
}

// ---- Create ----

func TestCreateNATGateway_Flags(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("name", "My NAT GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "123"))
	require.NoError(t, cmd.Flags().Set("yes", "true"))

	err := CreateNATGateway(cmd, nil, true)
	assert.NoError(t, err)
	require.NotNil(t, mock.CapturedCreateReq)
	assert.Equal(t, "My NAT GW", mock.CapturedCreateReq.ProductName)
	assert.Equal(t, 12, mock.CapturedCreateReq.Term)
	assert.Equal(t, 1000, mock.CapturedCreateReq.Speed)
	assert.Equal(t, 123, mock.CapturedCreateReq.LocationID)
}

func TestCreateNATGateway_JSON(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("json", `{"name":"JSON GW","term":12,"speed":2000,"locationId":456}`))
	require.NoError(t, cmd.Flags().Set("yes", "true"))

	err := CreateNATGateway(cmd, nil, true)
	assert.NoError(t, err)
	require.NotNil(t, mock.CapturedCreateReq)
	assert.Equal(t, "JSON GW", mock.CapturedCreateReq.ProductName)
	assert.Equal(t, 456, mock.CapturedCreateReq.LocationID)
}

func TestCreateNATGateway_JSONWithoutYes(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("json", `{"name":"JSON GW","term":12,"speed":2000,"locationId":456}`))

	err := CreateNATGateway(cmd, nil, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--yes is required to confirm creating a NAT Gateway design when using --json or --json-file")
}

func TestCreateNATGateway_ConfirmationPrompt(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	originalDesignConfirmPrompt := utils.GetDesignConfirmPrompt()
	defer func() { utils.SetDesignConfirmPrompt(originalDesignConfirmPrompt) }()

	promptCalled := false
	utils.SetDesignConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool {
		promptCalled = true
		return true
	})

	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("name", "My NAT GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "123"))

	err := CreateNATGateway(cmd, nil, true)
	assert.NoError(t, err)
	assert.True(t, promptCalled)
	require.NotNil(t, mock.CapturedCreateReq)
	assert.Equal(t, "My NAT GW", mock.CapturedCreateReq.ProductName)
}

func TestCreateNATGateway_ConfirmationDenied(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	originalDesignConfirmPrompt := utils.GetDesignConfirmPrompt()
	defer func() { utils.SetDesignConfirmPrompt(originalDesignConfirmPrompt) }()
	utils.SetDesignConfirmPrompt(func(_ string, _ []utils.BuyConfirmDetail, _ bool) bool { return false })

	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("name", "My NAT GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "123"))

	err := CreateNATGateway(cmd, nil, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled by user")
	assert.Nil(t, mock.CapturedCreateReq)
}

func TestCreateNATGateway_NoInput(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("create")
	err := CreateNATGateway(cmd, nil, true)
	assert.Error(t, err)
}

func TestCreateNATGateway_ServiceError(t *testing.T) {
	mock := &MockNATGatewayService{CreateErr: fmt.Errorf("service unavailable")}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("name", "My NAT GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "123"))
	require.NoError(t, cmd.Flags().Set("yes", "true"))

	err := CreateNATGateway(cmd, nil, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service unavailable")
}

func TestCreateNATGateway_InvalidJSON(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("json", "{invalid json}"))

	err := CreateNATGateway(cmd, nil, true)
	assert.Error(t, err)
}

func TestCreateNATGateway_MissingName(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "123"))

	err := CreateNATGateway(cmd, nil, true)
	assert.Error(t, err)
}

func TestCreateNATGateway_LoginError(t *testing.T) {
	defer testutil.SetupLoginError(fmt.Errorf("auth failed"))()

	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("name", "GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))
	require.NoError(t, cmd.Flags().Set("yes", "true"))

	err := CreateNATGateway(cmd, nil, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth failed")
}

func TestCreateNATGateway_MatrixRejectsUnsupportedSpeed(t *testing.T) {
	mock := &MockNATGatewayService{
		SessionsResult: []*megaport.NATGatewaySession{
			{SpeedMbps: 1000, SessionCount: []int{1, 2, 4}},
		},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("name", "My NAT GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "5000"))
	require.NoError(t, cmd.Flags().Set("location-id", "123"))
	require.NoError(t, cmd.Flags().Set("yes", "true"))

	err := CreateNATGateway(cmd, nil, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "speed")
	assert.Contains(t, err.Error(), "1000")
	assert.Nil(t, mock.CapturedCreateReq)
}

func TestCreateNATGateway_MatrixRejectsUnsupportedSessionCount(t *testing.T) {
	mock := &MockNATGatewayService{
		SessionsResult: []*megaport.NATGatewaySession{
			{SpeedMbps: 1000, SessionCount: []int{1, 2, 4}},
		},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("name", "My NAT GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "123"))
	require.NoError(t, cmd.Flags().Set("session-count", "3"))
	require.NoError(t, cmd.Flags().Set("yes", "true"))

	err := CreateNATGateway(cmd, nil, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session count")
	assert.Contains(t, err.Error(), "[1 2 4]")
	assert.Nil(t, mock.CapturedCreateReq)
}

func TestCreateNATGateway_MatrixAllowsSupportedPair(t *testing.T) {
	mock := &MockNATGatewayService{
		SessionsResult: []*megaport.NATGatewaySession{
			{SpeedMbps: 1000, SessionCount: []int{1, 2, 4}},
		},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("name", "My NAT GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "123"))
	require.NoError(t, cmd.Flags().Set("session-count", "2"))
	require.NoError(t, cmd.Flags().Set("yes", "true"))

	err := CreateNATGateway(cmd, nil, true)
	assert.NoError(t, err)
	require.NotNil(t, mock.CapturedCreateReq)
}

func TestCreateNATGateway_MatrixFetchFailureFallsThrough(t *testing.T) {
	mock := &MockNATGatewayService{SessionsErr: fmt.Errorf("matrix unavailable")}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("name", "My NAT GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "5000"))
	require.NoError(t, cmd.Flags().Set("location-id", "123"))
	require.NoError(t, cmd.Flags().Set("yes", "true"))

	err := CreateNATGateway(cmd, nil, true)
	assert.NoError(t, err)
	require.NotNil(t, mock.CapturedCreateReq)
}

func TestCreateNATGateway_MatrixBypassesSessionCountZero(t *testing.T) {
	mock := &MockNATGatewayService{
		SessionsResult: []*megaport.NATGatewaySession{
			{SpeedMbps: 1000, SessionCount: []int{1, 2, 4}},
		},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("create")
	require.NoError(t, cmd.Flags().Set("name", "My NAT GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "123"))
	require.NoError(t, cmd.Flags().Set("yes", "true"))

	err := CreateNATGateway(cmd, nil, true)
	assert.NoError(t, err)
	require.NotNil(t, mock.CapturedCreateReq)
	assert.Equal(t, 0, mock.CapturedCreateReq.Config.SessionCount)
}

// ---- Get ----

func TestGetNATGateway(t *testing.T) {
	mock := &MockNATGatewayService{
		GetResult: &megaport.NATGateway{
			ProductUID: "uid-123", ProductName: "Test GW",
			LocationID: 100, Speed: 1000, Term: 12,
			ProvisioningStatus: "LIVE",
		},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("get")
	err := GetNATGateway(cmd, []string{"uid-123"}, true, "json")
	assert.NoError(t, err)
	assert.Equal(t, "uid-123", mock.CapturedGetUID)
}

func TestGetNATGateway_Export(t *testing.T) {
	mock := &MockNATGatewayService{
		GetResult: &megaport.NATGateway{
			ProductUID: "uid-export", ProductName: "Export GW",
			LocationID: 200, Speed: 2000, Term: 24,
		},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("get")
	require.NoError(t, cmd.Flags().Set("export", "true"))

	err := GetNATGateway(cmd, []string{"uid-export"}, true, "table")
	assert.NoError(t, err)
}

func TestGetNATGateway_ServiceError(t *testing.T) {
	mock := &MockNATGatewayService{GetErr: fmt.Errorf("not found")}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("get")
	err := GetNATGateway(cmd, []string{"uid-missing"}, true, "table")
	assert.Error(t, err)
}

// ---- List ----

func TestListNATGateways(t *testing.T) {
	mock := &MockNATGatewayService{
		ListResult: []*megaport.NATGateway{
			{ProductUID: "uid-1", ProductName: "GW 1", ProvisioningStatus: "LIVE"},
			{ProductUID: "uid-2", ProductName: "GW 2", ProvisioningStatus: "LIVE"},
		},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("list")
	err := ListNATGateways(cmd, nil, true, "json")
	assert.NoError(t, err)
}

func TestListNATGateways_FilterByName(t *testing.T) {
	mock := &MockNATGatewayService{
		ListResult: []*megaport.NATGateway{
			{ProductUID: "uid-1", ProductName: "Production GW", ProvisioningStatus: "LIVE"},
			{ProductUID: "uid-2", ProductName: "Staging GW", ProvisioningStatus: "LIVE"},
		},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("list")
	require.NoError(t, cmd.Flags().Set("name", "production"))

	err := ListNATGateways(cmd, nil, true, "json")
	assert.NoError(t, err)
}

func TestListNATGateways_ServiceError(t *testing.T) {
	mock := &MockNATGatewayService{ListErr: fmt.Errorf("API down")}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("list")
	err := ListNATGateways(cmd, nil, true, "table")
	assert.Error(t, err)
}

// ---- Update ----

func TestUpdateNATGateway_Flags(t *testing.T) {
	mock := &MockNATGatewayService{
		GetResult: &megaport.NATGateway{
			ProductUID: "uid-upd", ProductName: "Old Name",
			LocationID: 100, Speed: 1000, Term: 12,
		},
		UpdateResult: &megaport.NATGateway{
			ProductUID: "uid-upd", ProductName: "New Name",
			LocationID: 100, Speed: 2000, Term: 12,
		},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("update")
	require.NoError(t, cmd.Flags().Set("name", "New Name"))
	require.NoError(t, cmd.Flags().Set("speed", "2000"))
	require.NoError(t, cmd.Flags().Set("location-id", "100"))
	require.NoError(t, cmd.Flags().Set("term", "12"))

	err := UpdateNATGateway(cmd, []string{"uid-upd"}, true)
	assert.NoError(t, err)
	require.NotNil(t, mock.CapturedUpdateReq)
	assert.Equal(t, "New Name", mock.CapturedUpdateReq.ProductName)
	assert.Equal(t, 2000, mock.CapturedUpdateReq.Speed)
}

func TestUpdateNATGateway_JSON(t *testing.T) {
	mock := &MockNATGatewayService{
		GetResult: &megaport.NATGateway{
			ProductUID: "uid-json-upd", ProductName: "Old",
			LocationID: 50, Speed: 500, Term: 1,
		},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("update")
	require.NoError(t, cmd.Flags().Set("json", `{"name":"Updated","locationId":50,"speed":500,"term":1}`))

	err := UpdateNATGateway(cmd, []string{"uid-json-upd"}, true)
	assert.NoError(t, err)
	require.NotNil(t, mock.CapturedUpdateReq)
	assert.Equal(t, "Updated", mock.CapturedUpdateReq.ProductName)
}

func TestUpdateNATGateway_NoInput(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("update")
	err := UpdateNATGateway(cmd, []string{"uid-no-input"}, true)
	assert.Error(t, err)
}

func TestUpdateNATGateway_ServiceError(t *testing.T) {
	mock := &MockNATGatewayService{
		GetResult: &megaport.NATGateway{
			ProductUID: "uid-err", ProductName: "GW",
			LocationID: 1, Speed: 100, Term: 1,
		},
		UpdateErr: fmt.Errorf("update failed"),
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("update")
	require.NoError(t, cmd.Flags().Set("name", "GW"))
	require.NoError(t, cmd.Flags().Set("speed", "100"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))
	require.NoError(t, cmd.Flags().Set("term", "1"))

	err := UpdateNATGateway(cmd, []string{"uid-err"}, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update failed")
}

func TestUpdateNATGateway_EmptyUIDResponse(t *testing.T) {
	mock := &MockNATGatewayService{
		GetResult: &megaport.NATGateway{
			ProductUID: "uid-empty", ProductName: "GW",
			LocationID: 1, Speed: 100, Term: 1,
		},
		// Simulate a service response with no ProductUID (empty-UID guard).
		UpdateResult: &megaport.NATGateway{},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("update")
	require.NoError(t, cmd.Flags().Set("name", "GW"))
	require.NoError(t, cmd.Flags().Set("speed", "100"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))
	require.NoError(t, cmd.Flags().Set("term", "1"))

	err := UpdateNATGateway(cmd, []string{"uid-empty"}, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no NAT Gateway UID")
}

// ---- Delete ----

func TestDeleteNATGateway_Force(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("delete")
	require.NoError(t, cmd.Flags().Set("force", "true"))

	err := DeleteNATGateway(cmd, []string{"uid-del"}, true)
	assert.NoError(t, err)
	assert.Equal(t, "uid-del", mock.CapturedDeleteUID)
}

func TestDeleteNATGateway_ServiceError(t *testing.T) {
	mock := &MockNATGatewayService{DeleteErr: fmt.Errorf("delete failed")}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("delete")
	require.NoError(t, cmd.Flags().Set("force", "true"))

	err := DeleteNATGateway(cmd, []string{"uid-del-err"}, true)
	assert.Error(t, err)
}

// ---- Validate ----

func TestValidateNATGateway(t *testing.T) {
	mock := &MockNATGatewayService{
		ValidateResult: &megaport.NATGatewayValidateResult{
			ProductUID:  "uid-val",
			ProductType: "NATGW",
			Metro:       "Sydney",
			Price: megaport.NATGatewayOrderPrice{
				Currency:    "AUD",
				MonthlyRate: 123.45,
				HourlyRate:  0.17,
			},
		},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("validate")
	err := ValidateNATGateway(cmd, []string{"uid-val"}, true, "table")
	assert.NoError(t, err)
	assert.Equal(t, "uid-val", mock.CapturedValidateUID)
}

func TestValidateNATGateway_ServiceError(t *testing.T) {
	mock := &MockNATGatewayService{ValidateErr: fmt.Errorf("validate failed")}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("validate")
	err := ValidateNATGateway(cmd, []string{"uid-val-err"}, true, "table")
	assert.Error(t, err)
}

// ---- Buy ----

func TestBuyNATGateway(t *testing.T) {
	mock := &MockNATGatewayService{
		BuyResult: &megaport.NATGatewayBuyResult{
			ProductUID:         "uid-buy",
			ProductName:        "My NAT GW",
			ServiceName:        "NAT-SRV-01",
			ProductType:        "NATGW",
			ProvisioningStatus: "DEPLOYABLE",
			RateLimit:          1000,
			LocationID:         123,
			ContractTermMonths: 12,
		},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("buy")
	require.NoError(t, cmd.Flags().Set("yes", "true"))

	err := BuyNATGateway(cmd, []string{"uid-buy"}, true)
	assert.NoError(t, err)
	assert.Equal(t, "uid-buy", mock.CapturedBuyUID)
}

func TestBuyNATGateway_ServiceError(t *testing.T) {
	mock := &MockNATGatewayService{BuyErr: fmt.Errorf("buy failed")}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("buy")
	require.NoError(t, cmd.Flags().Set("yes", "true"))

	err := BuyNATGateway(cmd, []string{"uid-buy-err"}, true)
	assert.Error(t, err)
}

// ---- List Sessions ----

func TestListNATGatewaySessions(t *testing.T) {
	mock := &MockNATGatewayService{
		SessionsResult: []*megaport.NATGatewaySession{
			{SpeedMbps: 1000, SessionCount: []int{100, 200, 500}},
			{SpeedMbps: 10000, SessionCount: []int{1000, 5000}},
		},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("list-sessions")
	err := ListNATGatewaySessions(cmd, nil, true, "table")
	assert.NoError(t, err)
}

func TestListNATGatewaySessions_ServiceError(t *testing.T) {
	mock := &MockNATGatewayService{SessionsErr: fmt.Errorf("sessions unavailable")}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("list-sessions")
	err := ListNATGatewaySessions(cmd, nil, true, "table")
	assert.Error(t, err)
}

// ---- Telemetry ----

func TestGetNATGatewayTelemetry_Days(t *testing.T) {
	mock := &MockNATGatewayService{
		TelemetryResult: &megaport.ServiceTelemetryResponse{
			ServiceUID: "uid-tel",
			Data: []*megaport.TelemetryMetricData{
				{
					Type:    "BITS",
					Subtype: "ingress",
					Unit:    megaport.TelemetryUnit{Name: "bps"},
					Samples: []megaport.TelemetrySample{{Timestamp: 1700000000000, Value: 1234.5}},
				},
			},
		},
	}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("telemetry")
	require.NoError(t, cmd.Flags().Set("types", "BITS"))
	require.NoError(t, cmd.Flags().Set("days", "7"))

	err := GetNATGatewayTelemetry(cmd, []string{"uid-tel"}, true, "json")
	assert.NoError(t, err)
	require.NotNil(t, mock.CapturedTelemetryReq)
	assert.Equal(t, []string{"BITS"}, mock.CapturedTelemetryReq.Types)
	assert.NotNil(t, mock.CapturedTelemetryReq.Days)
	assert.Equal(t, int32(7), *mock.CapturedTelemetryReq.Days)
}

func TestGetNATGatewayTelemetry_FromTo(t *testing.T) {
	mock := &MockNATGatewayService{TelemetryResult: &megaport.ServiceTelemetryResponse{}}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("telemetry")
	require.NoError(t, cmd.Flags().Set("types", "BITS,PACKETS"))
	require.NoError(t, cmd.Flags().Set("from", "2024-01-01T00:00:00Z"))
	require.NoError(t, cmd.Flags().Set("to", "2024-01-07T00:00:00Z"))

	err := GetNATGatewayTelemetry(cmd, []string{"uid-tel"}, true, "json")
	assert.NoError(t, err)
	require.NotNil(t, mock.CapturedTelemetryReq)
	assert.Equal(t, []string{"BITS", "PACKETS"}, mock.CapturedTelemetryReq.Types)
	assert.NotNil(t, mock.CapturedTelemetryReq.From)
	assert.NotNil(t, mock.CapturedTelemetryReq.To)
}

func TestGetNATGatewayTelemetry_MissingTypes(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("telemetry")
	err := GetNATGatewayTelemetry(cmd, []string{"uid-tel"}, true, "table")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--types is required")
}

func TestGetNATGatewayTelemetry_MissingTo(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("telemetry")
	require.NoError(t, cmd.Flags().Set("types", "BITS"))
	require.NoError(t, cmd.Flags().Set("from", "2024-01-01T00:00:00Z"))

	err := GetNATGatewayTelemetry(cmd, []string{"uid-tel"}, true, "table")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--from and --to must both be provided")
}

func TestGetNATGatewayTelemetry_InvalidFromTime(t *testing.T) {
	mock := &MockNATGatewayService{}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("telemetry")
	require.NoError(t, cmd.Flags().Set("types", "BITS"))
	require.NoError(t, cmd.Flags().Set("from", "not-a-time"))
	require.NoError(t, cmd.Flags().Set("to", "2024-01-07T00:00:00Z"))

	err := GetNATGatewayTelemetry(cmd, []string{"uid-tel"}, true, "table")
	assert.Error(t, err)
}

func TestGetNATGatewayTelemetry_ServiceError(t *testing.T) {
	mock := &MockNATGatewayService{TelemetryErr: fmt.Errorf("telemetry unavailable")}
	defer setupMockNATGateway(mock)()

	cmd := newTestCmd("telemetry")
	require.NoError(t, cmd.Flags().Set("types", "BITS"))
	require.NoError(t, cmd.Flags().Set("days", "7"))

	err := GetNATGatewayTelemetry(cmd, []string{"uid-tel"}, true, "table")
	assert.Error(t, err)
}

// ---- Output ----

func TestToNATGatewayOutput(t *testing.T) {
	gw := &megaport.NATGateway{
		ProductUID:         "uid-out",
		ProductName:        "Output GW",
		LocationID:         99,
		Speed:              5000,
		Term:               36,
		ProvisioningStatus: "LIVE",
		Config: megaport.NATGatewayNetworkConfig{
			SessionCount:  250,
			DiversityZone: "red",
			ASN:           65000,
		},
	}
	o, err := toNATGatewayOutput(gw)
	assert.NoError(t, err)
	assert.Equal(t, "uid-out", o.UID)
	assert.Equal(t, "Output GW", o.Name)
	assert.Equal(t, 99, o.LocationID)
	assert.Equal(t, 5000, o.Speed)
	assert.Equal(t, 36, o.Term)
	assert.Equal(t, "LIVE", o.ProvisioningStatus)
	assert.Equal(t, 250, o.SessionCount)
	assert.Equal(t, "red", o.DiversityZone)
	assert.Equal(t, 65000, o.ASN)
}

func TestToNATGatewayOutput_Nil(t *testing.T) {
	_, err := toNATGatewayOutput(nil)
	assert.Error(t, err)
}

func TestExportNATGatewayConfig(t *testing.T) {
	gw := &megaport.NATGateway{
		ProductName: "Export GW",
		Term:        12,
		Speed:       1000,
		LocationID:  42,
		Config: megaport.NATGatewayNetworkConfig{
			SessionCount:  100,
			DiversityZone: "blue",
		},
	}
	cfg := exportNATGatewayConfig(gw)
	jsonBytes, err := json.Marshal(cfg)
	assert.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBytes, &m))
	assert.Equal(t, "Export GW", m["name"])
	assert.Equal(t, float64(42), m["locationId"])
	assert.Equal(t, float64(100), m["sessionCount"])
	assert.Equal(t, "blue", m["diversityZone"])
}

// ---- Input processing ----

func TestProcessJSONCreateNATGatewayInput(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
		check   func(*megaport.CreateNATGatewayRequest)
	}{
		{
			name: "valid",
			json: `{"name":"GW","term":12,"speed":1000,"locationId":1}`,
			check: func(r *megaport.CreateNATGatewayRequest) {
				assert.Equal(t, "GW", r.ProductName)
				assert.Equal(t, 12, r.Term)
				assert.Equal(t, 1000, r.Speed)
				assert.Equal(t, 1, r.LocationID)
			},
		},
		{
			name:    "missing name",
			json:    `{"term":12,"speed":1000,"locationId":1}`,
			wantErr: true,
		},
		{
			name:    "missing location",
			json:    `{"name":"GW","term":12,"speed":1000}`,
			wantErr: true,
		},
		{
			name:    "invalid term",
			json:    `{"name":"GW","term":5,"speed":1000,"locationId":1}`,
			wantErr: true,
		},
		{
			name: "with session count and diversity zone",
			json: `{"name":"GW","term":12,"speed":1000,"locationId":1,"sessionCount":500,"diversityZone":"blue"}`,
			check: func(r *megaport.CreateNATGatewayRequest) {
				assert.Equal(t, 500, r.Config.SessionCount)
				assert.Equal(t, "blue", r.Config.DiversityZone)
			},
		},
		{
			name: "with resource tags",
			json: `{"name":"GW","term":12,"speed":1000,"locationId":1,"resourceTags":{"env":"prod"}}`,
			check: func(r *megaport.CreateNATGatewayRequest) {
				assert.Len(t, r.ResourceTags, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := processJSONCreateNATGatewayInput(tt.json, "")
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

func TestParseTelemetryTypes(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"BITS", []string{"BITS"}},
		{"bits,packets", []string{"BITS", "PACKETS"}},
		{"BITS, PACKETS, SPEED", []string{"BITS", "PACKETS", "SPEED"}},
		{"", nil},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, parseTelemetryTypes(tt.input))
		})
	}
}

func TestFilterNATGateways(t *testing.T) {
	gateways := []*megaport.NATGateway{
		{ProductUID: "uid-1", ProductName: "Production GW", LocationID: 100},
		{ProductUID: "uid-2", ProductName: "Staging GW", LocationID: 200},
		{ProductUID: "uid-3", ProductName: "Dev GW", LocationID: 100},
	}

	t.Run("filter by location", func(t *testing.T) {
		result := filterNATGateways(gateways, 100, "")
		assert.Len(t, result, 2)
	})

	t.Run("filter by name", func(t *testing.T) {
		result := filterNATGateways(gateways, 0, "staging")
		assert.Len(t, result, 1)
		assert.Equal(t, "uid-2", result[0].ProductUID)
	})

	t.Run("filter by location and name", func(t *testing.T) {
		result := filterNATGateways(gateways, 100, "dev")
		assert.Len(t, result, 1)
		assert.Equal(t, "uid-3", result[0].ProductUID)
	})

	t.Run("no filter", func(t *testing.T) {
		result := filterNATGateways(gateways, 0, "")
		assert.Len(t, result, 3)
	})

	t.Run("nil entry skipped", func(t *testing.T) {
		result := filterNATGateways([]*megaport.NATGateway{nil, gateways[0]}, 0, "")
		assert.Len(t, result, 1)
	})
}
