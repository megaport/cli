package ports

import (
	"context"
	"fmt"

	megaport "github.com/megaport/megaportgo"
)

type MockPortService struct {
	GetPortErr                      error
	GetPortResult                   *megaport.Port
	ListPortsErr                    error
	ListPortsResult                 []*megaport.Port
	BuyPortErr                      error
	BuyPortResult                   *megaport.BuyPortResponse
	CapturedRequest                 *megaport.BuyPortRequest
	CheckPortVLANAvailabilityErr    error
	CheckPortVLANAvailabilityResult bool
	CapturedVLANRequest             struct {
		PortID string
		VLANID int
	}
	DeletePortErr              error
	DeletePortResult           *megaport.DeletePortResponse
	CapturedDeletePortUID      string
	CapturedDeletePortRequest  *megaport.DeletePortRequest
	ListPortResourceTagsErr    error
	ListPortResourceTagsResult map[string]string
	CapturedResourceTagPortUID string
	CapturedResourceTags       map[string]string
	ValidatePortOrderErr       error
	ModifyPortErr              error
	ModifyPortResult           *megaport.ModifyPortResponse
	CapturedModifyPortRequest  *megaport.ModifyPortRequest
	LockPortErr                error
	LockPortResult             *megaport.LockPortResponse
	CapturedLockPortUID        string
	UnlockPortErr              error
	UnlockPortResult           *megaport.UnlockPortResponse
	CapturedUnlockPortUID      string
	UpdatePortResourceTagsErr  error
	CapturedUpdateTagsRequest  struct {
		PortID string
		Tags   map[string]string
	}
	UpdatePortErr    error
	UpdatePortResult *megaport.ModifyPortResponse
	ForceNilGetPort  bool
}

func (m *MockPortService) GetPort(ctx context.Context, portID string) (*megaport.Port, error) {
	if m.GetPortErr != nil {
		return nil, m.GetPortErr
	}
	if m.ForceNilGetPort {
		return nil, nil
	}
	if m.GetPortResult != nil {
		return m.GetPortResult, nil
	}
	return &megaport.Port{
		UID:                portID,
		Name:               "Mock Port",
		ProvisioningStatus: "LIVE",
	}, nil
}

func (m *MockPortService) ListPorts(ctx context.Context) ([]*megaport.Port, error) {
	if m.ListPortsErr != nil {
		return nil, m.ListPortsErr
	}
	if m.ListPortsResult != nil {
		return m.ListPortsResult, nil
	}
	return []*megaport.Port{}, nil
}

func (m *MockPortService) BuyPort(ctx context.Context, req *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error) {
	m.CapturedRequest = req
	if m.BuyPortErr != nil {
		return nil, m.BuyPortErr
	}
	if m.BuyPortResult != nil {
		return m.BuyPortResult, nil
	}
	return &megaport.BuyPortResponse{
		TechnicalServiceUIDs: []string{"mock-port-uid"},
	}, nil
}

func (m *MockPortService) CheckPortVLANAvailability(ctx context.Context, portID string, vlanID int) (bool, error) {
	m.CapturedVLANRequest.PortID = portID
	m.CapturedVLANRequest.VLANID = vlanID
	if m.CheckPortVLANAvailabilityErr != nil {
		return false, m.CheckPortVLANAvailabilityErr
	}
	return m.CheckPortVLANAvailabilityResult, nil
}

func (m *MockPortService) DeletePort(ctx context.Context, req *megaport.DeletePortRequest) (*megaport.DeletePortResponse, error) {
	m.CapturedDeletePortUID = req.PortID
	m.CapturedDeletePortRequest = req
	if m.DeletePortErr != nil {
		return nil, m.DeletePortErr
	}
	if m.DeletePortResult != nil {
		return m.DeletePortResult, nil
	}
	return &megaport.DeletePortResponse{
		IsDeleting: true,
	}, nil
}

func (m *MockPortService) ListPortResourceTags(ctx context.Context, portID string) (map[string]string, error) {
	m.CapturedResourceTagPortUID = portID
	if m.ListPortResourceTagsErr != nil {
		return nil, m.ListPortResourceTagsErr
	}
	if m.ListPortResourceTagsResult != nil {
		return m.ListPortResourceTagsResult, nil
	}
	return map[string]string{
		"environment": "test",
		"owner":       "automation",
	}, nil
}

func (m *MockPortService) ValidatePortOrder(ctx context.Context, req *megaport.BuyPortRequest) error {
	if m.ValidatePortOrderErr != nil {
		return m.ValidatePortOrderErr
	}
	return nil
}

func (m *MockPortService) LockPort(ctx context.Context, portId string) (*megaport.LockPortResponse, error) {
	m.CapturedLockPortUID = portId
	if m.LockPortErr != nil {
		return nil, m.LockPortErr
	}
	if m.LockPortResult != nil {
		return m.LockPortResult, nil
	}
	return &megaport.LockPortResponse{
		IsLocking: true,
	}, nil
}

func (m *MockPortService) ModifyPort(ctx context.Context, req *megaport.ModifyPortRequest) (*megaport.ModifyPortResponse, error) {
	m.CapturedModifyPortRequest = req
	if m.ModifyPortErr != nil {
		return nil, m.ModifyPortErr
	}
	if m.ModifyPortResult != nil {
		return m.ModifyPortResult, nil
	}
	return &megaport.ModifyPortResponse{
		IsUpdated: true,
	}, nil
}

func (m *MockPortService) RestorePort(ctx context.Context, portId string) (*megaport.RestorePortResponse, error) {
	return nil, fmt.Errorf("mock: RestorePort not configured")
}

func (m *MockPortService) UnlockPort(ctx context.Context, portId string) (*megaport.UnlockPortResponse, error) {
	m.CapturedUnlockPortUID = portId
	if m.UnlockPortErr != nil {
		return nil, m.UnlockPortErr
	}
	if m.UnlockPortResult != nil {
		return m.UnlockPortResult, nil
	}
	return &megaport.UnlockPortResponse{
		IsUnlocking: true,
	}, nil
}

func (m *MockPortService) UpdatePortResourceTags(ctx context.Context, portID string, tags map[string]string) error {
	if m.CapturedResourceTags == nil {
		m.CapturedResourceTags = make(map[string]string)
	}
	for k, v := range tags {
		m.CapturedResourceTags[k] = v
	}
	return m.UpdatePortResourceTagsErr
}

// Reset clears all configured results, errors, and captured requests.
func (m *MockPortService) Reset() {
	m.GetPortErr = nil
	m.GetPortResult = nil
	m.ListPortsErr = nil
	m.ListPortsResult = nil
	m.BuyPortErr = nil
	m.BuyPortResult = nil
	m.CapturedRequest = nil
	m.CheckPortVLANAvailabilityErr = nil
	m.CheckPortVLANAvailabilityResult = false
	m.CapturedVLANRequest = struct {
		PortID string
		VLANID int
	}{}
	m.DeletePortErr = nil
	m.DeletePortResult = nil
	m.CapturedDeletePortUID = ""
	m.CapturedDeletePortRequest = nil
	m.ListPortResourceTagsErr = nil
	m.ListPortResourceTagsResult = nil
	m.CapturedResourceTagPortUID = ""
	m.CapturedResourceTags = nil
	m.ValidatePortOrderErr = nil
	m.ModifyPortErr = nil
	m.ModifyPortResult = nil
	m.CapturedModifyPortRequest = nil
	m.LockPortErr = nil
	m.LockPortResult = nil
	m.CapturedLockPortUID = ""
	m.UnlockPortErr = nil
	m.UnlockPortResult = nil
	m.CapturedUnlockPortUID = ""
	m.UpdatePortResourceTagsErr = nil
	m.CapturedUpdateTagsRequest = struct {
		PortID string
		Tags   map[string]string
	}{}
	m.UpdatePortErr = nil
	m.UpdatePortResult = nil
	m.ForceNilGetPort = false
}
