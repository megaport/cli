//go:build js && wasm

package mcr

import (
	"context"
	"fmt"
	"syscall/js"

	"github.com/megaport/megaport-cli/internal/commands/config"
	megaport "github.com/megaport/megaportgo"
)

// Override the standard implementations with WASM-compatible versions
// These now use the SDK directly via the WASM HTTP transport!
func init() {
	getMCRFunc = getMCRWasmImpl
	buyMCRFunc = buyMCRWasmImpl
	updateMCRFunc = updateMCRWasmImpl
	deleteMCRFunc = deleteMCRWasmImpl
	restoreMCRFunc = restoreMCRWasmImpl
	lockMCRFunc = lockMCRWasmImpl
	unlockMCRFunc = unlockMCRWasmImpl
	createMCRPrefixFilterListFunc = createMCRPrefixFilterListWasmImpl
	listMCRPrefixFilterListsFunc = listMCRPrefixFilterListsWasmImpl
	getMCRPrefixFilterListFunc = getMCRPrefixFilterListWasmImpl
	modifyMCRPrefixFilterListFunc = modifyMCRPrefixFilterListWasmImpl
	deleteMCRPrefixFilterListFunc = deleteMCRPrefixFilterListWasmImpl
}

// isAuthError checks if the error is an authentication/authorization error
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return (errStr == "unauthorized" || errStr == "forbidden" ||
		errStr == "authentication required" || errStr == "token expired")
}

// getMCRWasmImpl uses the SDK's MCRService.GetMCR() method
func getMCRWasmImpl(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.MCR, error) {
	js.Global().Get("console").Call("log", fmt.Sprintf("🚀 Using SDK MCRService.GetMCR() for MCR %s", mcrUID))

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK MCRService.GetMCR()...")
	mcr, err := client.MCRService.GetMCR(ctx, mcrUID)
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK GetMCR failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to get MCR: %w", err)
	}

	js.Global().Get("console").Call("log", "✅ SDK GetMCR successful")
	return mcr, nil
}

// buyMCRWasmImpl uses the SDK's MCRService.BuyMCR() method
func buyMCRWasmImpl(ctx context.Context, client *megaport.Client, req *megaport.BuyMCRRequest) (*megaport.BuyMCRResponse, error) {
	js.Global().Get("console").Call("log", "🚀 Using SDK MCRService.BuyMCR()")

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK MCRService.BuyMCR()...")
	response, err := client.MCRService.BuyMCR(ctx, req)
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK BuyMCR failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to buy MCR: %w", err)
	}

	js.Global().Get("console").Call("log", "✅ SDK BuyMCR successful")
	return response, nil
}

// updateMCRWasmImpl uses the SDK's MCRService.ModifyMCR() method
func updateMCRWasmImpl(ctx context.Context, client *megaport.Client, req *megaport.ModifyMCRRequest) (*megaport.ModifyMCRResponse, error) {
	js.Global().Get("console").Call("log", "🚀 Using SDK MCRService.ModifyMCR()")

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK MCRService.ModifyMCR()...")
	response, err := client.MCRService.ModifyMCR(ctx, req)
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK ModifyMCR failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to modify MCR: %w", err)
	}

	js.Global().Get("console").Call("log", "✅ SDK ModifyMCR successful")
	return response, nil
}

// deleteMCRWasmImpl uses the SDK's MCRService.DeleteMCR() method
func deleteMCRWasmImpl(ctx context.Context, client *megaport.Client, req *megaport.DeleteMCRRequest) (*megaport.DeleteMCRResponse, error) {
	js.Global().Get("console").Call("log", "🚀 Using SDK MCRService.DeleteMCR()")

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK MCRService.DeleteMCR()...")
	response, err := client.MCRService.DeleteMCR(ctx, req)
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK DeleteMCR failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to delete MCR: %w", err)
	}

	js.Global().Get("console").Call("log", "✅ SDK DeleteMCR successful")
	return response, nil
}

// restoreMCRWasmImpl uses the SDK's MCRService.RestoreMCR() method
func restoreMCRWasmImpl(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.RestoreMCRResponse, error) {
	js.Global().Get("console").Call("log", fmt.Sprintf("🚀 Using SDK MCRService.RestoreMCR() for MCR %s", mcrUID))

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK MCRService.RestoreMCR()...")
	response, err := client.MCRService.RestoreMCR(ctx, mcrUID) //nolint:staticcheck // the SDK deprecated restore and always returns ErrRestoreNotAllowed
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK RestoreMCR failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to restore MCR: %w", err)
	}

	js.Global().Get("console").Call("log", "✅ SDK RestoreMCR successful")
	return response, nil
}

// createMCRPrefixFilterListWasmImpl uses the SDK's MCRService.CreateMCRPrefixFilterList() method
func createMCRPrefixFilterListWasmImpl(ctx context.Context, client *megaport.Client, req *megaport.CreateMCRPrefixFilterListRequest) (*megaport.CreateMCRPrefixFilterListResponse, error) {
	js.Global().Get("console").Call("log", "🚀 Using SDK MCRService.CreateMCRPrefixFilterList()")

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK MCRService.CreatePrefixFilterList()...")
	response, err := client.MCRService.CreatePrefixFilterList(ctx, req)
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK CreateMCRPrefixFilterList failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to create MCR prefix filter list: %w", err)
	}

	js.Global().Get("console").Call("log", "✅ SDK CreateMCRPrefixFilterList successful")
	return response, nil
}

// listMCRPrefixFilterListsWasmImpl uses the SDK's MCRService.ListMCRPrefixFilterLists() method
func listMCRPrefixFilterListsWasmImpl(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.PrefixFilterList, error) {
	js.Global().Get("console").Call("log", fmt.Sprintf("🚀 Using SDK MCRService.ListMCRPrefixFilterLists() for MCR %s", mcrUID))

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK MCRService.ListMCRPrefixFilterLists()...")
	lists, err := client.MCRService.ListMCRPrefixFilterLists(ctx, mcrUID)
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK ListMCRPrefixFilterLists failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to list MCR prefix filter lists: %w", err)
	}

	js.Global().Get("console").Call("log", fmt.Sprintf("✅ SDK returned %d prefix filter lists successfully", len(lists)))
	return lists, nil
}

// getMCRPrefixFilterListWasmImpl uses the SDK's MCRService.GetMCRPrefixFilterList() method
func getMCRPrefixFilterListWasmImpl(ctx context.Context, client *megaport.Client, mcrUID string, prefixFilterListID int) (*megaport.MCRPrefixFilterList, error) {
	js.Global().Get("console").Call("log", fmt.Sprintf("🚀 Using SDK MCRService.GetMCRPrefixFilterList() for MCR %s, filter list %d", mcrUID, prefixFilterListID))

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK MCRService.GetMCRPrefixFilterList()...")
	list, err := client.MCRService.GetMCRPrefixFilterList(ctx, mcrUID, prefixFilterListID)
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK GetMCRPrefixFilterList failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to get MCR prefix filter list: %w", err)
	}

	js.Global().Get("console").Call("log", "✅ SDK GetMCRPrefixFilterList successful")
	return list, nil
}

// modifyMCRPrefixFilterListWasmImpl uses the SDK's MCRService.ModifyMCRPrefixFilterList() method
func modifyMCRPrefixFilterListWasmImpl(ctx context.Context, client *megaport.Client, mcrID string, prefixFilterListID int, prefixFilterList *megaport.MCRPrefixFilterList) (*megaport.ModifyMCRPrefixFilterListResponse, error) {
	js.Global().Get("console").Call("log", fmt.Sprintf("🚀 Using SDK MCRService.ModifyMCRPrefixFilterList() for MCR %s, filter list %d", mcrID, prefixFilterListID))

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK MCRService.ModifyMCRPrefixFilterList()...")
	response, err := client.MCRService.ModifyMCRPrefixFilterList(ctx, mcrID, prefixFilterListID, prefixFilterList)
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK ModifyMCRPrefixFilterList failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to modify MCR prefix filter list: %w", err)
	}

	js.Global().Get("console").Call("log", "✅ SDK ModifyMCRPrefixFilterList successful")
	return response, nil
}

// deleteMCRPrefixFilterListWasmImpl uses the SDK's MCRService.DeleteMCRPrefixFilterList() method
func deleteMCRPrefixFilterListWasmImpl(ctx context.Context, client *megaport.Client, mcrID string, prefixFilterListID int) (*megaport.DeleteMCRPrefixFilterListResponse, error) {
	js.Global().Get("console").Call("log", fmt.Sprintf("🚀 Using SDK MCRService.DeleteMCRPrefixFilterList() for MCR %s, filter list %d", mcrID, prefixFilterListID))

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK MCRService.DeleteMCRPrefixFilterList()...")
	response, err := client.MCRService.DeleteMCRPrefixFilterList(ctx, mcrID, prefixFilterListID)
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK DeleteMCRPrefixFilterList failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to delete MCR prefix filter list: %w", err)
	}

	js.Global().Get("console").Call("log", "✅ SDK DeleteMCRPrefixFilterList successful")
	return response, nil
}

// lockMCRWasmImpl uses the SDK's ProductService.ManageProductLock() method
func lockMCRWasmImpl(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.ManageProductLockResponse, error) {
	js.Global().Get("console").Call("log", fmt.Sprintf("🚀 Using SDK ProductService.ManageProductLock() to lock MCR %s", mcrUID))

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK ProductService.ManageProductLock()...")
	response, err := client.ProductService.ManageProductLock(ctx, &megaport.ManageProductLockRequest{ProductID: mcrUID, ShouldLock: true})
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK ManageProductLock (lock) failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to lock MCR: %w", err)
	}

	js.Global().Get("console").Call("log", "✅ SDK ManageProductLock (lock) successful")
	return response, nil
}

// unlockMCRWasmImpl uses the SDK's ProductService.ManageProductLock() method
func unlockMCRWasmImpl(ctx context.Context, client *megaport.Client, mcrUID string) (*megaport.ManageProductLockResponse, error) {
	js.Global().Get("console").Call("log", fmt.Sprintf("🚀 Using SDK ProductService.ManageProductLock() to unlock MCR %s", mcrUID))

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK ProductService.ManageProductLock()...")
	response, err := client.ProductService.ManageProductLock(ctx, &megaport.ManageProductLockRequest{ProductID: mcrUID, ShouldLock: false})
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK ManageProductLock (unlock) failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to unlock MCR: %w", err)
	}

	js.Global().Get("console").Call("log", "✅ SDK ManageProductLock (unlock) successful")
	return response, nil
}
