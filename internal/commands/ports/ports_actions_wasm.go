//go:build js && wasm

package ports

import (
	"context"
	"fmt"
	"strings"
	"syscall/js"

	"github.com/megaport/megaport-cli/internal/commands/config"
	megaport "github.com/megaport/megaportgo"
)

// Override the standard implementations with WASM-compatible versions
// These now use the SDK directly via the WASM HTTP transport!
func init() {
	listPortsFunc = listPortsWasmImpl
	getPortFunc = getPortWasmImpl
	updatePortFunc = updatePortWasmImpl
	deletePortFunc = deletePortWasmImpl
	restorePortFunc = restorePortWasmImpl
	lockPortFunc = lockPortWasmImpl
	unlockPortFunc = unlockPortWasmImpl
	checkPortVLANAvailabilityFunc = checkPortVLANAvailabilityWasmImpl
	buyPortFunc = buyPortWasmImpl
	listPortResourceTagsFunc = listPortResourceTagsWasmImpl
}

// isAuthError checks if an error is a 401 authentication error
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "Bad session token") ||
		strings.Contains(errStr, "Unauthorized") ||
		strings.Contains(errStr, "authentication failed")
}

// listPortsWasmImpl uses the SDK's PortService.ListPorts() method
// The WASM HTTP transport automatically handles the fetch API calls
func listPortsWasmImpl(ctx context.Context, client *megaport.Client) ([]*megaport.Port, error) {
	js.Global().Get("console").Call("log", "🚀 Using SDK PortService.ListPorts() with WASM HTTP transport")

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK PortService.ListPorts()...")
	ports, err := client.PortService.ListPorts(ctx)
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK ListPorts failed: %v", err))

		// Check if this is a 401 error and clear the cached token
		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to list ports: %w", err)
	}

	js.Global().Get("console").Call("log", fmt.Sprintf("✅ SDK returned %d ports successfully", len(ports)))
	return ports, nil
}

// getPortWasmImpl uses the SDK's PortService.GetPort() method
// The WASM HTTP transport automatically handles the fetch API calls
func getPortWasmImpl(ctx context.Context, client *megaport.Client, portID string) (*megaport.Port, error) {
	js.Global().Get("console").Call("log", fmt.Sprintf("🚀 Using SDK PortService.GetPort() for port %s", portID))

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", fmt.Sprintf("📡 Calling SDK PortService.GetPort(%s)...", portID))
	port, err := client.PortService.GetPort(ctx, portID)
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK GetPort failed: %v", err))

		// Check if this is a 401 error and clear the cached token
		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to get port: %w", err)
	}

	js.Global().Get("console").Call("log", fmt.Sprintf("✅ SDK returned port: %s", port.Name))
	return port, nil
}

// updatePortWasmImpl uses the SDK's PortService.ModifyPort() method
func updatePortWasmImpl(ctx context.Context, client *megaport.Client, req *megaport.ModifyPortRequest) (*megaport.ModifyPortResponse, error) {
	js.Global().Get("console").Call("log", fmt.Sprintf("🚀 Using SDK PortService.ModifyPort() for port %s", req.PortID))

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK PortService.ModifyPort()...")
	response, err := client.PortService.ModifyPort(ctx, req)
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK ModifyPort failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to modify port: %w", err)
	}

	js.Global().Get("console").Call("log", "✅ SDK ModifyPort successful")
	return response, nil
}

// deletePortWasmImpl uses the SDK's PortService.DeletePort() method
func deletePortWasmImpl(ctx context.Context, client *megaport.Client, req *megaport.DeletePortRequest) (*megaport.DeletePortResponse, error) {
	js.Global().Get("console").Call("log", fmt.Sprintf("🚀 Using SDK PortService.DeletePort() for port %s", req.PortID))

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK PortService.DeletePort()...")
	response, err := client.PortService.DeletePort(ctx, req)
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK DeletePort failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to delete port: %w", err)
	}

	js.Global().Get("console").Call("log", "✅ SDK DeletePort successful")
	return response, nil
}

// restorePortWasmImpl uses the SDK's PortService.RestorePort() method
func restorePortWasmImpl(ctx context.Context, client *megaport.Client, portUID string) (*megaport.RestorePortResponse, error) {
	js.Global().Get("console").Call("log", fmt.Sprintf("🚀 Using SDK PortService.RestorePort() for port %s", portUID))

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK PortService.RestorePort()...")
	response, err := client.PortService.RestorePort(ctx, portUID) //nolint:staticcheck // the SDK deprecated restore and always returns ErrRestoreNotAllowed
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK RestorePort failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to restore port: %w", err)
	}

	js.Global().Get("console").Call("log", "✅ SDK RestorePort successful")
	return response, nil
}

// lockPortWasmImpl uses the SDK's PortService.LockPort() method
func lockPortWasmImpl(ctx context.Context, client *megaport.Client, portUID string) (*megaport.LockPortResponse, error) {
	js.Global().Get("console").Call("log", fmt.Sprintf("🚀 Using SDK PortService.LockPort() for port %s", portUID))

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK PortService.LockPort()...")
	response, err := client.PortService.LockPort(ctx, portUID)
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK LockPort failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to lock port: %w", err)
	}

	js.Global().Get("console").Call("log", "✅ SDK LockPort successful")
	return response, nil
}

// unlockPortWasmImpl uses the SDK's PortService.UnlockPort() method
func unlockPortWasmImpl(ctx context.Context, client *megaport.Client, portUID string) (*megaport.UnlockPortResponse, error) {
	js.Global().Get("console").Call("log", fmt.Sprintf("🚀 Using SDK PortService.UnlockPort() for port %s", portUID))

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK PortService.UnlockPort()...")
	response, err := client.PortService.UnlockPort(ctx, portUID)
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK UnlockPort failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to unlock port: %w", err)
	}

	js.Global().Get("console").Call("log", "✅ SDK UnlockPort successful")
	return response, nil
}

// checkPortVLANAvailabilityWasmImpl uses the SDK's PortService.CheckPortVLANAvailability() method
func checkPortVLANAvailabilityWasmImpl(ctx context.Context, client *megaport.Client, portUID string, vlan int) (bool, error) {
	js.Global().Get("console").Call("log", fmt.Sprintf("🚀 Using SDK PortService.CheckPortVLANAvailability() for port %s, VLAN %d", portUID, vlan))

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return false, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK PortService.CheckPortVLANAvailability()...")
	available, err := client.PortService.CheckPortVLANAvailability(ctx, portUID, vlan)
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK CheckPortVLANAvailability failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return false, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return false, fmt.Errorf("failed to check VLAN availability: %w", err)
	}

	js.Global().Get("console").Call("log", fmt.Sprintf("✅ SDK CheckPortVLANAvailability result: %v", available))
	return available, nil
}

// buyPortWasmImpl uses the SDK's PortService.BuyPort() method
func buyPortWasmImpl(ctx context.Context, client *megaport.Client, req *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error) {
	js.Global().Get("console").Call("log", "🚀 Using SDK PortService.BuyPort() with WASM HTTP transport")

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK PortService.BuyPort()...")
	response, err := client.PortService.BuyPort(ctx, req)
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK BuyPort failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to buy port: %w", err)
	}

	js.Global().Get("console").Call("log", "✅ SDK BuyPort successful")
	return response, nil
}

// listPortResourceTagsWasmImpl uses the SDK's PortService.ListPortResourceTags() method
func listPortResourceTagsWasmImpl(ctx context.Context, client *megaport.Client, portUID string) (map[string]string, error) {
	js.Global().Get("console").Call("log", fmt.Sprintf("🚀 Using SDK PortService.ListPortResourceTags() for port %s", portUID))

	if client == nil {
		var err error
		client, err = config.Login(ctx)
		if err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("❌ Login failed: %v", err))
			return nil, fmt.Errorf("failed to log in: %w", err)
		}
	}

	js.Global().Get("console").Call("log", "📡 Calling SDK PortService.ListPortResourceTags()...")
	tags, err := client.PortService.ListPortResourceTags(ctx, portUID)
	if err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("❌ SDK ListPortResourceTags failed: %v", err))

		if isAuthError(err) {
			js.Global().Get("console").Call("warn", "🔓 Authentication token expired or invalid, clearing cache")
			config.ClearCachedToken()
			return nil, fmt.Errorf("authentication token expired. Please run the command again to re-authenticate")
		}

		return nil, fmt.Errorf("failed to list port resource tags: %w", err)
	}

	js.Global().Get("console").Call("log", fmt.Sprintf("✅ SDK returned %d resource tags successfully", len(tags)))
	return tags, nil
}
