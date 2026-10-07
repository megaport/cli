package mcr

import (
	"context"
	"strings"

	"github.com/megaport/megaport-cli/internal/utils"
	megaport "github.com/megaport/megaportgo"
)

var getMCRFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.MCR, error) {
	return client.MCRService.GetMCR(ctx, mcrUID)
}

var buyMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.BuyMCRRequest) (*megaport.BuyMCRResponse, error) {
	return client.MCRService.BuyMCR(ctx, req)
}

var updateMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ModifyMCRRequest) (*megaport.ModifyMCRResponse, error) {
	return client.MCRService.ModifyMCR(ctx, req)
}

var createMCRPrefixFilterListFunc = func(ctx context.Context, client *megaport.Client, req *megaport.CreateMCRPrefixFilterListRequest) (*megaport.CreateMCRPrefixFilterListResponse, error) {
	return client.MCRService.CreatePrefixFilterList(ctx, req)
}

var listMCRPrefixFilterListsFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.PrefixFilterList, error) {
	return client.MCRService.ListMCRPrefixFilterLists(ctx, mcrUID)
}

var getMCRPrefixFilterListFunc = func(ctx context.Context, client *megaport.Client, mcrUID string, prefixFilterListID int) (*megaport.MCRPrefixFilterList, error) {
	return client.MCRService.GetMCRPrefixFilterList(ctx, mcrUID, prefixFilterListID)
}

var modifyMCRPrefixFilterListFunc = func(ctx context.Context, client *megaport.Client, mcrID string, prefixFilterListID int, prefixFilterList *megaport.MCRPrefixFilterList) (*megaport.ModifyMCRPrefixFilterListResponse, error) {
	return client.MCRService.ModifyMCRPrefixFilterList(ctx, mcrID, prefixFilterListID, prefixFilterList)
}

var deleteMCRPrefixFilterListFunc = func(ctx context.Context, client *megaport.Client, mcrID string, prefixFilterListID int) (*megaport.DeleteMCRPrefixFilterListResponse, error) {
	return client.MCRService.DeleteMCRPrefixFilterList(ctx, mcrID, prefixFilterListID)
}

var deleteMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.DeleteMCRRequest) (*megaport.DeleteMCRResponse, error) {
	return client.MCRService.DeleteMCR(ctx, req)
}

var restoreMCRFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.RestoreMCRResponse, error) {
	return client.MCRService.RestoreMCR(ctx, mcrUID) //nolint:staticcheck // the SDK deprecated restore and always returns ErrRestoreNotAllowed
}

var listMCRResourceTagsFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) (map[string]string, error) {
	return client.MCRService.ListMCRResourceTags(ctx, mcrUID)
}

var lockMCRFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.ManageProductLockResponse, error) {
	return client.ProductService.ManageProductLock(ctx, &megaport.ManageProductLockRequest{ProductID: mcrUID, ShouldLock: true})
}

var unlockMCRFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.ManageProductLockResponse, error) {
	return client.ProductService.ManageProductLock(ctx, &megaport.ManageProductLockRequest{ProductID: mcrUID, ShouldLock: false})
}

var updateMCRWithAddOnFunc = func(ctx context.Context, client *megaport.Client, mcrID string, req megaport.MCRAddOnRequest) error {
	return client.MCRService.UpdateMCRWithAddOn(ctx, mcrID, req)
}

var updateMCRIPsecAddOnFunc = func(ctx context.Context, client *megaport.Client, mcrID, addOnUID string, tunnelCount int) error {
	return client.MCRService.UpdateMCRIPsecAddOn(ctx, mcrID, addOnUID, tunnelCount)
}

func filterMCRs(mcrs []*megaport.MCR, locationID, portSpeed int, mcrName string) []*megaport.MCR {
	return utils.Filter(mcrs, func(mcr *megaport.MCR) bool {
		if mcr == nil {
			return false
		}
		if locationID > 0 && mcr.LocationID != locationID {
			return false
		}
		if portSpeed > 0 && mcr.PortSpeed != portSpeed {
			return false
		}
		if mcrName != "" && !strings.Contains(strings.ToLower(mcr.Name), strings.ToLower(mcrName)) {
			return false
		}
		return true
	})
}
