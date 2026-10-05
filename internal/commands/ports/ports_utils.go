package ports

import (
	"context"
	"strings"

	"github.com/megaport/megaport-cli/internal/utils"
	megaport "github.com/megaport/megaportgo"
)

var getPortFunc = func(ctx context.Context, client *megaport.Client, portUID string) (*megaport.Port, error) {
	return client.PortService.GetPort(ctx, portUID)
}

var updatePortFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ModifyPortRequest) (*megaport.ModifyPortResponse, error) {
	return client.PortService.ModifyPort(ctx, req)
}

var deletePortFunc = func(ctx context.Context, client *megaport.Client, req *megaport.DeletePortRequest) (*megaport.DeletePortResponse, error) {
	return client.PortService.DeletePort(ctx, req)
}

var lockPortFunc = func(ctx context.Context, client *megaport.Client, portUID string) (*megaport.LockPortResponse, error) {
	return client.PortService.LockPort(ctx, portUID)
}

var unlockPortFunc = func(ctx context.Context, client *megaport.Client, portUID string) (*megaport.UnlockPortResponse, error) {
	return client.PortService.UnlockPort(ctx, portUID)
}

var checkPortVLANAvailabilityFunc = func(ctx context.Context, client *megaport.Client, portUID string, vlan int) (bool, error) {
	return client.PortService.CheckPortVLANAvailability(ctx, portUID, vlan)
}

var buyPortFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error) {
	return client.PortService.BuyPort(ctx, req)
}

var listPortResourceTagsFunc = func(ctx context.Context, client *megaport.Client, portUID string) (map[string]string, error) {
	return client.PortService.ListPortResourceTags(ctx, portUID)
}

func filterPorts(ports []*megaport.Port, locationID, portSpeed int, portName string, includeInactive bool) []*megaport.Port {
	return utils.Filter(ports, func(port *megaport.Port) bool {
		if port == nil {
			return false
		}
		if !includeInactive {
			if port.ProvisioningStatus == megaport.STATUS_CANCELLED ||
				port.ProvisioningStatus == megaport.STATUS_DECOMMISSIONED ||
				port.ProvisioningStatus == utils.StatusDecommissioning {
				return false
			}
		}
		if locationID > 0 && port.LocationID != locationID {
			return false
		}
		if portSpeed > 0 && port.PortSpeed != portSpeed {
			return false
		}
		if portName != "" && !strings.Contains(strings.ToLower(port.Name), strings.ToLower(portName)) {
			return false
		}
		return true
	})
}
