package mcr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/config"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestListLookingGlassIPRoutes(t *testing.T) {
	// Store original functions and restore after test
	originalLoginFunc := config.GetLoginFunc()
	originalFunc := listIPRoutesFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		listIPRoutesFunc = originalFunc
	}()

	// Mock the login function
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})

	metric := 100
	localPref := 200
	age := 3600
	best := true

	// Mock the function
	listIPRoutesFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.LookingGlassIPRoute, error) {
		return []*megaport.LookingGlassIPRoute{
			{
				Prefix:    "10.0.0.0/24",
				NextHop:   "192.168.1.1",
				Protocol:  megaport.RouteProtocolBGP,
				Metric:    &metric,
				LocalPref: &localPref,
				ASPath:    []int{65001, 65002},
				Age:       &age,
				Interface: "eth0",
				VXCName:   "Test VXC",
				Best:      &best,
			},
		}, nil
	}

	// Create command
	cmd := &cobra.Command{}
	cmd.Flags().String("protocol", "", "")
	cmd.Flags().String("ip", "", "")

	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Execute
	err := ListLookingGlassIPRoutes(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestListLookingGlassIPRoutesWithFilter(t *testing.T) {
	// Store original functions and restore after test
	originalLoginFunc := config.GetLoginFunc()
	originalFunc := listIPRoutesWithFilterFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		listIPRoutesWithFilterFunc = originalFunc
	}()

	// Mock the login function
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})

	// Mock the function
	listIPRoutesWithFilterFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListIPRoutesRequest) ([]*megaport.LookingGlassIPRoute, error) {
		assert.Equal(t, "test-mcr-uid", req.MCRID)
		assert.Equal(t, megaport.RouteProtocolBGP, req.Protocol)
		return []*megaport.LookingGlassIPRoute{
			{
				Prefix:   "10.0.0.0/24",
				NextHop:  "192.168.1.1",
				Protocol: megaport.RouteProtocolBGP,
			},
		}, nil
	}

	// Create command with filter flags
	cmd := &cobra.Command{}
	cmd.Flags().String("protocol", "BGP", "")
	cmd.Flags().String("ip", "", "")

	// Execute
	err := ListLookingGlassIPRoutes(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestListLookingGlassBGPRoutes(t *testing.T) {
	// Store original functions and restore after test
	originalLoginFunc := config.GetLoginFunc()
	originalFunc := listBGPRoutesFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		listBGPRoutesFunc = originalFunc
	}()

	// Mock the login function
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})

	localPref := 100
	med := 50
	neighborASN := 65001
	age := 7200

	// Mock the function
	listBGPRoutesFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.LookingGlassBGPRoute, error) {
		return []*megaport.LookingGlassBGPRoute{
			{
				Prefix:      "10.0.0.0/24",
				NextHop:     "192.168.1.1",
				ASPath:      []int{65001, 65002, 65003},
				LocalPref:   &localPref,
				MED:         &med,
				Origin:      "IGP",
				Communities: []string{"65001:100", "65001:200"},
				Valid:       true,
				Best:        true,
				NeighborIP:  "192.168.1.2",
				NeighborASN: &neighborASN,
				Age:         &age,
				VXCName:     "Test VXC",
			},
		}, nil
	}

	// Create command
	cmd := &cobra.Command{}
	cmd.Flags().String("ip", "", "")

	// Execute
	err := ListLookingGlassBGPRoutes(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestListLookingGlassBGPSessions(t *testing.T) {
	// Store original functions and restore after test
	originalLoginFunc := config.GetLoginFunc()
	originalFunc := listBGPSessionsFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		listBGPSessionsFunc = originalFunc
	}()

	// Mock the login function
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})

	uptime := 86400
	prefixesIn := 100
	prefixesOut := 50

	// Mock the function
	listBGPSessionsFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.LookingGlassBGPSession, error) {
		return []*megaport.LookingGlassBGPSession{
			{
				SessionID:       "session-123",
				NeighborAddress: "192.168.1.2",
				NeighborASN:     65001,
				LocalASN:        65000,
				Status:          megaport.BGPSessionStatusUp,
				Uptime:          &uptime,
				PrefixesIn:      &prefixesIn,
				PrefixesOut:     &prefixesOut,
				VXCID:           12345,
				VXCName:         "Test VXC",
				Description:     "Test BGP Session",
			},
		}, nil
	}

	// Create command
	cmd := &cobra.Command{}

	// Execute
	err := ListLookingGlassBGPSessions(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestListLookingGlassBGPNeighborRoutes(t *testing.T) {
	// Store original functions and restore after test
	originalLoginFunc := config.GetLoginFunc()
	originalFunc := listBGPNeighborRoutesFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		listBGPNeighborRoutesFunc = originalFunc
	}()

	// Mock the login function
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})

	localPref := 100
	med := 50

	// Mock the function
	listBGPNeighborRoutesFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListBGPNeighborRoutesRequest) ([]*megaport.LookingGlassBGPNeighborRoute, error) {
		assert.Equal(t, "test-mcr-uid", req.MCRID)
		assert.Equal(t, "session-123", req.SessionID)
		assert.Equal(t, megaport.LookingGlassRouteDirectionReceived, req.Direction)
		return []*megaport.LookingGlassBGPNeighborRoute{
			{
				Prefix:      "10.0.0.0/24",
				NextHop:     "192.168.1.1",
				ASPath:      []int{65001, 65002},
				LocalPref:   &localPref,
				MED:         &med,
				Origin:      "IGP",
				Communities: []string{"65001:100"},
				Valid:       true,
				Best:        true,
			},
		}, nil
	}

	// Create command
	cmd := &cobra.Command{}
	cmd.Flags().String("ip", "", "")

	// Execute
	err := ListLookingGlassBGPNeighborRoutes(cmd, []string{"test-mcr-uid", "session-123", "received"}, true, "json")
	assert.NoError(t, err)
}

func TestListLookingGlassBGPNeighborRoutesInvalidDirection(t *testing.T) {
	// Store original function and restore after test
	originalLoginFunc := config.GetLoginFunc()
	defer func() { config.SetLoginFunc(originalLoginFunc) }()

	// Mock the login function
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})

	// Create command
	cmd := &cobra.Command{}
	cmd.Flags().String("ip", "", "")

	// Execute with invalid direction
	err := ListLookingGlassBGPNeighborRoutes(cmd, []string{"test-mcr-uid", "session-123", "invalid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "direction must be 'advertised' or 'received'")
}

// Output conversion tests

func TestToIPRouteOutput(t *testing.T) {
	metric := 100
	localPref := 200
	age := 3661 // 1 hour, 1 minute, 1 second
	best := true

	route := &megaport.LookingGlassIPRoute{
		Prefix:    "10.0.0.0/24",
		NextHop:   "192.168.1.1",
		Protocol:  megaport.RouteProtocolBGP,
		Metric:    &metric,
		LocalPref: &localPref,
		ASPath:    []int{65001, 65002},
		Age:       &age,
		Interface: "eth0",
		VXCName:   "Test VXC",
		Best:      &best,
	}

	output, err := ToIPRouteOutput(route)
	assert.NoError(t, err)
	assert.Equal(t, "10.0.0.0/24", output.Prefix)
	assert.Equal(t, "192.168.1.1", output.NextHop)
	assert.Equal(t, "BGP", output.Protocol)
	assert.Equal(t, "100", output.Metric)
	assert.Equal(t, "200", output.LocalPref)
	assert.Equal(t, "65001 65002", output.ASPath)
	assert.Equal(t, "1h1m", output.Age)
	assert.Equal(t, "eth0", output.Interface)
	assert.Equal(t, "Test VXC", output.VXCName)
	assert.Equal(t, "Yes", output.Best)
}

func TestToIPRouteOutputNil(t *testing.T) {
	_, err := ToIPRouteOutput(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid route: nil value")
}

func TestToBGPRouteOutput(t *testing.T) {
	localPref := 100
	med := 50
	neighborASN := 65001
	age := 90061 // 1 day, 1 hour, 1 minute, 1 second

	route := &megaport.LookingGlassBGPRoute{
		Prefix:      "10.0.0.0/24",
		NextHop:     "192.168.1.1",
		ASPath:      []int{65001, 65002, 65003},
		LocalPref:   &localPref,
		MED:         &med,
		Origin:      "IGP",
		Communities: []string{"65001:100", "65001:200"},
		Valid:       true,
		Best:        false,
		NeighborIP:  "192.168.1.2",
		NeighborASN: &neighborASN,
		Age:         &age,
		VXCName:     "Test VXC",
	}

	output, err := ToBGPRouteOutput(route)
	assert.NoError(t, err)
	assert.Equal(t, "10.0.0.0/24", output.Prefix)
	assert.Equal(t, "192.168.1.1", output.NextHop)
	assert.Equal(t, "65001 65002 65003", output.ASPath)
	assert.Equal(t, "100", output.LocalPref)
	assert.Equal(t, "50", output.MED)
	assert.Equal(t, "IGP", output.Origin)
	assert.Equal(t, "65001:100, 65001:200", output.Communities)
	assert.Equal(t, "Yes", output.Valid)
	assert.Equal(t, "No", output.Best)
	assert.Equal(t, "192.168.1.2", output.NeighborIP)
	assert.Equal(t, "65001", output.NeighborASN)
	assert.Equal(t, "1d1h", output.Age)
	assert.Equal(t, "Test VXC", output.VXCName)
}

func TestToBGPSessionOutput(t *testing.T) {
	uptime := 86400
	prefixesIn := 100
	prefixesOut := 50

	session := &megaport.LookingGlassBGPSession{
		SessionID:       "session-123",
		NeighborAddress: "192.168.1.2",
		NeighborASN:     65001,
		LocalASN:        65000,
		Status:          megaport.BGPSessionStatusUp,
		Uptime:          &uptime,
		PrefixesIn:      &prefixesIn,
		PrefixesOut:     &prefixesOut,
		VXCID:           12345,
		VXCName:         "Test VXC",
		Description:     "Test BGP Session",
	}

	output, err := ToBGPSessionOutput(session)
	assert.NoError(t, err)
	assert.Equal(t, "session-123", output.SessionID)
	assert.Equal(t, "192.168.1.2", output.NeighborAddress)
	assert.Equal(t, 65001, output.NeighborASN)
	assert.Equal(t, 65000, output.LocalASN)
	assert.Equal(t, "UP", output.Status)
	assert.Equal(t, "1d0h", output.Uptime)
	assert.Equal(t, "100", output.PrefixesIn)
	assert.Equal(t, "50", output.PrefixesOut)
	assert.Equal(t, "Test VXC", output.VXCName)
	assert.Equal(t, "Test BGP Session", output.Description)
}

func TestToBGPNeighborRouteOutput(t *testing.T) {
	localPref := 100
	med := 50

	route := &megaport.LookingGlassBGPNeighborRoute{
		Prefix:      "10.0.0.0/24",
		NextHop:     "192.168.1.1",
		ASPath:      []int{65001, 65002},
		LocalPref:   &localPref,
		MED:         &med,
		Origin:      "IGP",
		Communities: []string{"65001:100"},
		Valid:       true,
		Best:        true,
	}

	output, err := ToBGPNeighborRouteOutput(route)
	assert.NoError(t, err)
	assert.Equal(t, "10.0.0.0/24", output.Prefix)
	assert.Equal(t, "192.168.1.1", output.NextHop)
	assert.Equal(t, "65001 65002", output.ASPath)
	assert.Equal(t, "100", output.LocalPref)
	assert.Equal(t, "50", output.MED)
	assert.Equal(t, "IGP", output.Origin)
	assert.Equal(t, "65001:100", output.Communities)
	assert.Equal(t, "Yes", output.Valid)
	assert.Equal(t, "Yes", output.Best)
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  int
		expected string
	}{
		{30, "30s"},
		{61, "1m1s"},
		{3661, "1h1m"},
		{90061, "1d1h"},
		{180000, "2d2h"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.seconds)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBoolToYesNo(t *testing.T) {
	assert.Equal(t, "Yes", boolToYesNo(true))
	assert.Equal(t, "No", boolToYesNo(false))
}

// Error and edge-case tests for actions

func TestListLookingGlassIPRoutes_LoginError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	defer config.SetLoginFunc(originalLoginFunc)

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return nil, fmt.Errorf("login failed")
	})

	cmd := &cobra.Command{}
	cmd.Flags().String("protocol", "", "")
	cmd.Flags().String("ip", "", "")

	err := ListLookingGlassIPRoutes(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error logging in")
}

func TestListLookingGlassIPRoutes_APIError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalFunc := listIPRoutesFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		listIPRoutesFunc = originalFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})
	listIPRoutesFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.LookingGlassIPRoute, error) {
		return nil, fmt.Errorf("api error")
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("protocol", "", "")
	cmd.Flags().String("ip", "", "")

	err := ListLookingGlassIPRoutes(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error listing IP routes")
}

func TestListLookingGlassIPRoutes_Empty(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalFunc := listIPRoutesFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		listIPRoutesFunc = originalFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})
	listIPRoutesFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.LookingGlassIPRoute, error) {
		return []*megaport.LookingGlassIPRoute{}, nil
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("protocol", "", "")
	cmd.Flags().String("ip", "", "")

	err := ListLookingGlassIPRoutes(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestListLookingGlassIPRoutes_WithIPFilterOnly(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalFunc := listIPRoutesWithFilterFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		listIPRoutesWithFilterFunc = originalFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})
	listIPRoutesWithFilterFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListIPRoutesRequest) ([]*megaport.LookingGlassIPRoute, error) {
		assert.Equal(t, "10.0.0.0/24", req.IPFilter)
		assert.Equal(t, megaport.RouteProtocol(""), req.Protocol)
		return []*megaport.LookingGlassIPRoute{}, nil
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("protocol", "", "")
	cmd.Flags().String("ip", "10.0.0.0/24", "")

	err := ListLookingGlassIPRoutes(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestListLookingGlassIPRoutesWithFilter_APIError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalFunc := listIPRoutesWithFilterFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		listIPRoutesWithFilterFunc = originalFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})
	listIPRoutesWithFilterFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListIPRoutesRequest) ([]*megaport.LookingGlassIPRoute, error) {
		return nil, fmt.Errorf("api error")
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("protocol", "BGP", "")
	cmd.Flags().String("ip", "", "")

	err := ListLookingGlassIPRoutes(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error listing IP routes")
}

func TestListLookingGlassBGPRoutes_LoginError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	defer config.SetLoginFunc(originalLoginFunc)

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return nil, fmt.Errorf("login failed")
	})

	cmd := &cobra.Command{}
	cmd.Flags().String("ip", "", "")

	err := ListLookingGlassBGPRoutes(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error logging in")
}

func TestListLookingGlassBGPRoutes_APIError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalFunc := listBGPRoutesFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		listBGPRoutesFunc = originalFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})
	listBGPRoutesFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.LookingGlassBGPRoute, error) {
		return nil, fmt.Errorf("api error")
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("ip", "", "")

	err := ListLookingGlassBGPRoutes(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error listing BGP routes")
}

func TestListLookingGlassBGPRoutes_Empty(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalFunc := listBGPRoutesFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		listBGPRoutesFunc = originalFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})
	listBGPRoutesFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.LookingGlassBGPRoute, error) {
		return []*megaport.LookingGlassBGPRoute{}, nil
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("ip", "", "")

	err := ListLookingGlassBGPRoutes(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestListLookingGlassBGPRoutes_WithFilter(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalFunc := listBGPRoutesWithFilterFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		listBGPRoutesWithFilterFunc = originalFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})
	listBGPRoutesWithFilterFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListBGPRoutesRequest) ([]*megaport.LookingGlassBGPRoute, error) {
		assert.Equal(t, "test-mcr-uid", req.MCRID)
		assert.Equal(t, "192.168.1.0/24", req.IPFilter)
		return []*megaport.LookingGlassBGPRoute{}, nil
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("ip", "192.168.1.0/24", "")

	err := ListLookingGlassBGPRoutes(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestListLookingGlassBGPRoutes_WithFilterError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalFunc := listBGPRoutesWithFilterFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		listBGPRoutesWithFilterFunc = originalFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})
	listBGPRoutesWithFilterFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListBGPRoutesRequest) ([]*megaport.LookingGlassBGPRoute, error) {
		return nil, fmt.Errorf("filter api error")
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("ip", "192.168.1.0/24", "")

	err := ListLookingGlassBGPRoutes(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error listing BGP routes")
}

func TestListLookingGlassBGPSessions_LoginError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	defer config.SetLoginFunc(originalLoginFunc)

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return nil, fmt.Errorf("login failed")
	})

	cmd := &cobra.Command{}
	err := ListLookingGlassBGPSessions(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error logging in")
}

func TestListLookingGlassBGPSessions_APIError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalFunc := listBGPSessionsFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		listBGPSessionsFunc = originalFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})
	listBGPSessionsFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.LookingGlassBGPSession, error) {
		return nil, fmt.Errorf("api error")
	}

	cmd := &cobra.Command{}
	err := ListLookingGlassBGPSessions(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error listing BGP sessions")
}

func TestListLookingGlassBGPSessions_Empty(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalFunc := listBGPSessionsFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		listBGPSessionsFunc = originalFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})
	listBGPSessionsFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.LookingGlassBGPSession, error) {
		return []*megaport.LookingGlassBGPSession{}, nil
	}

	cmd := &cobra.Command{}
	err := ListLookingGlassBGPSessions(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestListLookingGlassBGPNeighborRoutes_LoginError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	defer config.SetLoginFunc(originalLoginFunc)

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return nil, fmt.Errorf("login failed")
	})

	cmd := &cobra.Command{}
	cmd.Flags().String("ip", "", "")

	err := ListLookingGlassBGPNeighborRoutes(cmd, []string{"test-mcr-uid", "session-123", "received"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error logging in")
}

func TestListLookingGlassBGPNeighborRoutes_APIError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalFunc := listBGPNeighborRoutesFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		listBGPNeighborRoutesFunc = originalFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})
	listBGPNeighborRoutesFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListBGPNeighborRoutesRequest) ([]*megaport.LookingGlassBGPNeighborRoute, error) {
		return nil, fmt.Errorf("api error")
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("ip", "", "")

	err := ListLookingGlassBGPNeighborRoutes(cmd, []string{"test-mcr-uid", "session-123", "advertised"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error listing BGP neighbor routes")
}

func TestListLookingGlassBGPNeighborRoutes_Empty(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalFunc := listBGPNeighborRoutesFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		listBGPNeighborRoutesFunc = originalFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})
	listBGPNeighborRoutesFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListBGPNeighborRoutesRequest) ([]*megaport.LookingGlassBGPNeighborRoute, error) {
		return []*megaport.LookingGlassBGPNeighborRoute{}, nil
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("ip", "", "")

	err := ListLookingGlassBGPNeighborRoutes(cmd, []string{"test-mcr-uid", "session-123", "advertised"}, true, "json")
	assert.NoError(t, err)
}

func TestListLookingGlassBGPNeighborRoutes_Advertised(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalFunc := listBGPNeighborRoutesFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		listBGPNeighborRoutesFunc = originalFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})
	listBGPNeighborRoutesFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListBGPNeighborRoutesRequest) ([]*megaport.LookingGlassBGPNeighborRoute, error) {
		assert.Equal(t, megaport.LookingGlassRouteDirectionAdvertised, req.Direction)
		return []*megaport.LookingGlassBGPNeighborRoute{}, nil
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("ip", "", "")

	err := ListLookingGlassBGPNeighborRoutes(cmd, []string{"test-mcr-uid", "session-123", "advertised"}, true, "json")
	assert.NoError(t, err)
}

func TestListLookingGlassBGPNeighborRoutes_WithIPFilter(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalFunc := listBGPNeighborRoutesFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		listBGPNeighborRoutesFunc = originalFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})
	listBGPNeighborRoutesFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListBGPNeighborRoutesRequest) ([]*megaport.LookingGlassBGPNeighborRoute, error) {
		assert.Equal(t, "10.0.0.0/24", req.IPFilter)
		return []*megaport.LookingGlassBGPNeighborRoute{}, nil
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("ip", "10.0.0.0/24", "")

	err := ListLookingGlassBGPNeighborRoutes(cmd, []string{"test-mcr-uid", "session-123", "received"}, true, "json")
	assert.NoError(t, err)
}

// Nil-input tests for output converters

func TestToBGPRouteOutputNil(t *testing.T) {
	_, err := ToBGPRouteOutput(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid BGP route: nil value")
}

func TestToBGPSessionOutputNil(t *testing.T) {
	_, err := ToBGPSessionOutput(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid BGP session: nil value")
}

func TestToBGPNeighborRouteOutputNil(t *testing.T) {
	_, err := ToBGPNeighborRouteOutput(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid BGP neighbor route: nil value")
}

// Minimal-field tests (no optional fields set) for branch coverage

func TestToIPRouteOutput_MinimalFields(t *testing.T) {
	route := &megaport.LookingGlassIPRoute{
		Prefix:  "10.0.0.0/24",
		NextHop: "192.168.1.1",
	}
	out, err := ToIPRouteOutput(route)
	assert.NoError(t, err)
	assert.Equal(t, "10.0.0.0/24", out.Prefix)
	assert.Empty(t, out.Metric)
	assert.Empty(t, out.LocalPref)
	assert.Empty(t, out.ASPath)
	assert.Empty(t, out.Age)
	assert.Empty(t, out.Best)
}

func TestToIPRouteOutput_BestFalse(t *testing.T) {
	best := false
	route := &megaport.LookingGlassIPRoute{
		Prefix:  "10.0.0.0/24",
		NextHop: "192.168.1.1",
		Best:    &best,
	}
	out, err := ToIPRouteOutput(route)
	assert.NoError(t, err)
	assert.Equal(t, "No", out.Best)
}

func TestToBGPRouteOutput_MinimalFields(t *testing.T) {
	route := &megaport.LookingGlassBGPRoute{
		Prefix:  "10.0.0.0/24",
		NextHop: "192.168.1.1",
	}
	out, err := ToBGPRouteOutput(route)
	assert.NoError(t, err)
	assert.Equal(t, "10.0.0.0/24", out.Prefix)
	assert.Empty(t, out.ASPath)
	assert.Empty(t, out.LocalPref)
	assert.Empty(t, out.MED)
	assert.Empty(t, out.Communities)
	assert.Empty(t, out.NeighborASN)
	assert.Empty(t, out.Age)
}

func TestToBGPSessionOutput_MinimalFields(t *testing.T) {
	session := &megaport.LookingGlassBGPSession{
		SessionID:       "session-1",
		NeighborAddress: "192.168.1.2",
		NeighborASN:     65001,
		LocalASN:        65000,
		Status:          megaport.BGPSessionStatusDown,
	}
	out, err := ToBGPSessionOutput(session)
	assert.NoError(t, err)
	assert.Equal(t, "DOWN", out.Status)
	assert.Empty(t, out.Uptime)
	assert.Empty(t, out.PrefixesIn)
	assert.Empty(t, out.PrefixesOut)
}

func TestToBGPNeighborRouteOutput_MinimalFields(t *testing.T) {
	route := &megaport.LookingGlassBGPNeighborRoute{
		Prefix:  "10.0.0.0/24",
		NextHop: "192.168.1.1",
	}
	out, err := ToBGPNeighborRouteOutput(route)
	assert.NoError(t, err)
	assert.Equal(t, "10.0.0.0/24", out.Prefix)
	assert.Empty(t, out.ASPath)
	assert.Empty(t, out.LocalPref)
	assert.Empty(t, out.MED)
	assert.Empty(t, out.Communities)
}

// Utils wrapper function tests — exercise the real var bodies via a mock client

func TestLookingGlassUtilsWrappers(t *testing.T) {
	mockSvc := &MockMCRLookingGlassService{}
	client := &megaport.Client{}
	client.MCRLookingGlassService = mockSvc
	ctx := context.Background()

	t.Run("listIPRoutesFunc", func(t *testing.T) {
		mockSvc.ListIPRoutesResult = []*megaport.LookingGlassIPRoute{{Prefix: "1.0.0.0/8"}}
		routes, err := listIPRoutesFunc(ctx, client, "mcr-1")
		assert.NoError(t, err)
		assert.Equal(t, "mcr-1", mockSvc.CapturedListIPRoutesMCRUID)
		assert.Len(t, routes, 1)
	})

	t.Run("listIPRoutesWithFilterFunc", func(t *testing.T) {
		req := &megaport.ListIPRoutesRequest{MCRID: "mcr-1", IPFilter: "1.0.0.0/8"}
		mockSvc.ListIPRoutesWithFilterResult = []*megaport.LookingGlassIPRoute{{Prefix: "1.0.0.0/8"}}
		routes, err := listIPRoutesWithFilterFunc(ctx, client, req)
		assert.NoError(t, err)
		assert.Equal(t, req, mockSvc.CapturedListIPRoutesWithFilter)
		assert.Len(t, routes, 1)
	})

	t.Run("listBGPRoutesFunc", func(t *testing.T) {
		mockSvc.ListBGPRoutesResult = []*megaport.LookingGlassBGPRoute{{Prefix: "1.0.0.0/8"}}
		routes, err := listBGPRoutesFunc(ctx, client, "mcr-1")
		assert.NoError(t, err)
		assert.Equal(t, "mcr-1", mockSvc.CapturedListBGPRoutesMCRUID)
		assert.Len(t, routes, 1)
	})

	t.Run("listBGPRoutesWithFilterFunc", func(t *testing.T) {
		req := &megaport.ListBGPRoutesRequest{MCRID: "mcr-1", IPFilter: "1.0.0.0/8"}
		mockSvc.ListBGPRoutesWithFilterResult = []*megaport.LookingGlassBGPRoute{{Prefix: "1.0.0.0/8"}}
		routes, err := listBGPRoutesWithFilterFunc(ctx, client, req)
		assert.NoError(t, err)
		assert.Equal(t, req, mockSvc.CapturedListBGPRoutesWithFilter)
		assert.Len(t, routes, 1)
	})

	t.Run("listBGPSessionsFunc", func(t *testing.T) {
		mockSvc.ListBGPSessionsResult = []*megaport.LookingGlassBGPSession{{SessionID: "s1"}}
		sessions, err := listBGPSessionsFunc(ctx, client, "mcr-1")
		assert.NoError(t, err)
		assert.Equal(t, "mcr-1", mockSvc.CapturedListBGPSessionsMCRUID)
		assert.Len(t, sessions, 1)
	})

	t.Run("listBGPNeighborRoutesFunc", func(t *testing.T) {
		req := &megaport.ListBGPNeighborRoutesRequest{MCRID: "mcr-1", SessionID: "s1"}
		mockSvc.ListBGPNeighborRoutesResult = []*megaport.LookingGlassBGPNeighborRoute{{Prefix: "1.0.0.0/8"}}
		routes, err := listBGPNeighborRoutesFunc(ctx, client, req)
		assert.NoError(t, err)
		assert.Equal(t, req, mockSvc.CapturedListBGPNeighborRoutes)
		assert.Len(t, routes, 1)
	})

	t.Run("pingMCRFunc", func(t *testing.T) {
		req := &megaport.MCRPingRequest{MCRID: "mcr-1", DestinationAddress: "8.8.8.8"}
		mockSvc.PingMCRResult = "op-1"
		opID, err := pingMCRFunc(ctx, client, req)
		assert.NoError(t, err)
		assert.Equal(t, "op-1", opID)
		assert.Equal(t, req, mockSvc.CapturedPingMCRRequest)
	})

	t.Run("tracerouteMCRFunc", func(t *testing.T) {
		req := &megaport.MCRTracerouteRequest{MCRID: "mcr-1", DestinationAddress: "8.8.8.8"}
		mockSvc.TracerouteMCRResult = "op-2"
		opID, err := tracerouteMCRFunc(ctx, client, req)
		assert.NoError(t, err)
		assert.Equal(t, "op-2", opID)
		assert.Equal(t, req, mockSvc.CapturedTracerouteMCRRequest)
	})

	t.Run("waitForMCRPingFunc", func(t *testing.T) {
		mockSvc.WaitForMCRPingResult = &megaport.LookingGlassPingResult{RawOutput: "ping output"}
		result, err := waitForMCRPingFunc(ctx, client, "mcr-1", "op-1")
		assert.NoError(t, err)
		assert.Equal(t, "mcr-1", mockSvc.CapturedWaitForMCRPingMCRUID)
		assert.Equal(t, "op-1", mockSvc.CapturedWaitForMCRPingOpID)
		assert.Equal(t, "ping output", result.RawOutput)
	})

	t.Run("waitForMCRTracerouteFunc", func(t *testing.T) {
		mockSvc.WaitForMCRTracerouteResult = &megaport.LookingGlassTracerouteResult{RawOutput: "traceroute output"}
		result, err := waitForMCRTracerouteFunc(ctx, client, "mcr-1", "op-2")
		assert.NoError(t, err)
		assert.Equal(t, "mcr-1", mockSvc.CapturedWaitForMCRTracerouteMCRUID)
		assert.Equal(t, "op-2", mockSvc.CapturedWaitForMCRTracerouteOpID)
		assert.Equal(t, "traceroute output", result.RawOutput)
	})
}

// Ping and traceroute action tests

func TestLookingGlassPing(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalPingFunc := pingMCRFunc
	originalWaitFunc := waitForMCRPingFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		pingMCRFunc = originalPingFunc
		waitForMCRPingFunc = originalWaitFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})

	pingMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.MCRPingRequest) (string, error) {
		assert.Equal(t, "test-mcr-uid", req.MCRID)
		assert.Equal(t, "8.8.8.8", req.DestinationAddress)
		return "op-123", nil
	}

	waitForMCRPingFunc = func(ctx context.Context, client *megaport.Client, mcrUID, operationID string) (*megaport.LookingGlassPingResult, error) {
		assert.Equal(t, "test-mcr-uid", mcrUID)
		assert.Equal(t, "op-123", operationID)
		return &megaport.LookingGlassPingResult{
			RawOutput: "PING 8.8.8.8",
			Statistics: &megaport.LookingGlassPingStatistics{
				PacketsTransmitted: 4,
				PacketsReceived:    4,
				PacketLossPct:      0,
				RTTMinMs:           10.1,
				RTTAvgMs:           12.2,
				RTTMaxMs:           14.3,
				RTTMdevMs:          1.5,
			},
		}, nil
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("destination", "8.8.8.8", "")
	cmd.Flags().String("source", "", "")
	cmd.Flags().Int("packet-count", 0, "")
	cmd.Flags().Int("packet-size", 0, "")

	err := LookingGlassPing(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestLookingGlassPing_WithOptionalFlags(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalPingFunc := pingMCRFunc
	originalWaitFunc := waitForMCRPingFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		pingMCRFunc = originalPingFunc
		waitForMCRPingFunc = originalWaitFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})

	pingMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.MCRPingRequest) (string, error) {
		assert.Equal(t, "10.0.0.1", req.SourceAddress)
		if assert.NotNil(t, req.PacketCount) {
			assert.Equal(t, int32(10), *req.PacketCount)
		}
		if assert.NotNil(t, req.PacketSize) {
			assert.Equal(t, int32(128), *req.PacketSize)
		}
		return "op-123", nil
	}

	waitForMCRPingFunc = func(ctx context.Context, client *megaport.Client, mcrUID, operationID string) (*megaport.LookingGlassPingResult, error) {
		return &megaport.LookingGlassPingResult{RawOutput: "PING"}, nil
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("destination", "8.8.8.8", "")
	cmd.Flags().String("source", "10.0.0.1", "")
	cmd.Flags().Int("packet-count", 0, "")
	cmd.Flags().Int("packet-size", 0, "")
	assert.NoError(t, cmd.Flags().Set("packet-count", "10"))
	assert.NoError(t, cmd.Flags().Set("packet-size", "128"))

	err := LookingGlassPing(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestLookingGlassPing_PacketCountOutOfRange(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	defer config.SetLoginFunc(originalLoginFunc)

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})

	cmd := &cobra.Command{}
	cmd.Flags().String("destination", "8.8.8.8", "")
	cmd.Flags().String("source", "", "")
	cmd.Flags().Int("packet-count", 0, "")
	cmd.Flags().Int("packet-size", 0, "")
	assert.NoError(t, cmd.Flags().Set("packet-count", "61"))

	err := LookingGlassPing(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "packet count")
}

func TestLookingGlassPing_PacketSizeOutOfRange(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	defer config.SetLoginFunc(originalLoginFunc)

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})

	cmd := &cobra.Command{}
	cmd.Flags().String("destination", "8.8.8.8", "")
	cmd.Flags().String("source", "", "")
	cmd.Flags().Int("packet-count", 0, "")
	cmd.Flags().Int("packet-size", 0, "")
	assert.NoError(t, cmd.Flags().Set("packet-size", "9187"))

	err := LookingGlassPing(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "packet size")
}

func TestLookingGlassPing_MissingDestination(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	defer config.SetLoginFunc(originalLoginFunc)

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})

	cmd := &cobra.Command{}
	cmd.Flags().String("destination", "", "")
	cmd.Flags().String("source", "", "")
	cmd.Flags().Int("packet-count", 0, "")
	cmd.Flags().Int("packet-size", 0, "")

	err := LookingGlassPing(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "destination is required")
}

func TestLookingGlassPing_LoginError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	defer config.SetLoginFunc(originalLoginFunc)

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return nil, fmt.Errorf("login failed")
	})

	cmd := &cobra.Command{}
	cmd.Flags().String("destination", "8.8.8.8", "")
	cmd.Flags().String("source", "", "")
	cmd.Flags().Int("packet-count", 0, "")
	cmd.Flags().Int("packet-size", 0, "")

	err := LookingGlassPing(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error logging in")
}

func TestLookingGlassPing_StartError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalPingFunc := pingMCRFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		pingMCRFunc = originalPingFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})
	apiErr := fmt.Errorf("api error")
	pingMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.MCRPingRequest) (string, error) {
		return "", apiErr
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("destination", "8.8.8.8", "")
	cmd.Flags().String("source", "", "")
	cmd.Flags().Int("packet-count", 0, "")
	cmd.Flags().Int("packet-size", 0, "")

	err := LookingGlassPing(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error starting ping")
	assert.True(t, errors.Is(err, apiErr), "underlying error should be preserved via %%w")
}

func TestLookingGlassPing_WaitError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalPingFunc := pingMCRFunc
	originalWaitFunc := waitForMCRPingFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		pingMCRFunc = originalPingFunc
		waitForMCRPingFunc = originalWaitFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})
	pingMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.MCRPingRequest) (string, error) {
		return "op-123", nil
	}
	waitErr := fmt.Errorf("timed out")
	waitForMCRPingFunc = func(ctx context.Context, client *megaport.Client, mcrUID, operationID string) (*megaport.LookingGlassPingResult, error) {
		return nil, waitErr
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("destination", "8.8.8.8", "")
	cmd.Flags().String("source", "", "")
	cmd.Flags().Int("packet-count", 0, "")
	cmd.Flags().Int("packet-size", 0, "")

	err := LookingGlassPing(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error waiting for ping result")
	assert.True(t, errors.Is(err, waitErr), "underlying error should be preserved via %%w")
}

func TestLookingGlassPing_InvalidDestination(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	defer config.SetLoginFunc(originalLoginFunc)

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})

	cmd := &cobra.Command{}
	cmd.Flags().String("destination", "not-an-ip", "")
	cmd.Flags().String("source", "", "")
	cmd.Flags().Int("packet-count", 0, "")
	cmd.Flags().Int("packet-size", 0, "")

	err := LookingGlassPing(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "destination")
}

func TestLookingGlassPing_InvalidSource(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	defer config.SetLoginFunc(originalLoginFunc)

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})

	cmd := &cobra.Command{}
	cmd.Flags().String("destination", "8.8.8.8", "")
	cmd.Flags().String("source", "not-an-ip", "")
	cmd.Flags().Int("packet-count", 0, "")
	cmd.Flags().Int("packet-size", 0, "")

	err := LookingGlassPing(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source")
}

func TestLookingGlassTraceroute(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalTracerouteFunc := tracerouteMCRFunc
	originalWaitFunc := waitForMCRTracerouteFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		tracerouteMCRFunc = originalTracerouteFunc
		waitForMCRTracerouteFunc = originalWaitFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})

	tracerouteMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.MCRTracerouteRequest) (string, error) {
		assert.Equal(t, "test-mcr-uid", req.MCRID)
		assert.Equal(t, "8.8.8.8", req.DestinationAddress)
		return "op-456", nil
	}

	waitForMCRTracerouteFunc = func(ctx context.Context, client *megaport.Client, mcrUID, operationID string) (*megaport.LookingGlassTracerouteResult, error) {
		assert.Equal(t, "test-mcr-uid", mcrUID)
		assert.Equal(t, "op-456", operationID)
		return &megaport.LookingGlassTracerouteResult{
			RawOutput: "traceroute to 8.8.8.8",
			Hops: []*megaport.LookingGlassTracerouteHop{
				{
					Hop: "1",
					Probes: []*megaport.LookingGlassTracerouteProbe{
						{Host: "10.0.0.1", RTTMs: 1.2},
					},
				},
			},
		}, nil
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("destination", "8.8.8.8", "")
	cmd.Flags().String("source", "", "")

	err := LookingGlassTraceroute(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestLookingGlassTraceroute_NoHops(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalTracerouteFunc := tracerouteMCRFunc
	originalWaitFunc := waitForMCRTracerouteFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		tracerouteMCRFunc = originalTracerouteFunc
		waitForMCRTracerouteFunc = originalWaitFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})

	tracerouteMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.MCRTracerouteRequest) (string, error) {
		return "op-456", nil
	}

	waitForMCRTracerouteFunc = func(ctx context.Context, client *megaport.Client, mcrUID, operationID string) (*megaport.LookingGlassTracerouteResult, error) {
		return &megaport.LookingGlassTracerouteResult{RawOutput: "traceroute to 8.8.8.8"}, nil
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("destination", "8.8.8.8", "")
	cmd.Flags().String("source", "", "")

	err := LookingGlassTraceroute(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestLookingGlassTraceroute_MissingDestination(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	defer config.SetLoginFunc(originalLoginFunc)

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})

	cmd := &cobra.Command{}
	cmd.Flags().String("destination", "", "")
	cmd.Flags().String("source", "", "")

	err := LookingGlassTraceroute(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "destination is required")
}

func TestLookingGlassTraceroute_LoginError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	defer config.SetLoginFunc(originalLoginFunc)

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return nil, fmt.Errorf("login failed")
	})

	cmd := &cobra.Command{}
	cmd.Flags().String("destination", "8.8.8.8", "")
	cmd.Flags().String("source", "", "")

	err := LookingGlassTraceroute(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error logging in")
}

func TestLookingGlassTraceroute_StartError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalTracerouteFunc := tracerouteMCRFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		tracerouteMCRFunc = originalTracerouteFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})
	apiErr := fmt.Errorf("api error")
	tracerouteMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.MCRTracerouteRequest) (string, error) {
		return "", apiErr
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("destination", "8.8.8.8", "")
	cmd.Flags().String("source", "", "")

	err := LookingGlassTraceroute(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error starting traceroute")
	assert.True(t, errors.Is(err, apiErr), "underlying error should be preserved via %%w")
}

func TestLookingGlassTraceroute_WaitError(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	originalTracerouteFunc := tracerouteMCRFunc
	originalWaitFunc := waitForMCRTracerouteFunc
	defer func() {
		config.SetLoginFunc(originalLoginFunc)
		tracerouteMCRFunc = originalTracerouteFunc
		waitForMCRTracerouteFunc = originalWaitFunc
	}()

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})
	tracerouteMCRFunc = func(ctx context.Context, client *megaport.Client, req *megaport.MCRTracerouteRequest) (string, error) {
		return "op-456", nil
	}
	waitErr := fmt.Errorf("timed out")
	waitForMCRTracerouteFunc = func(ctx context.Context, client *megaport.Client, mcrUID, operationID string) (*megaport.LookingGlassTracerouteResult, error) {
		return nil, waitErr
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("destination", "8.8.8.8", "")
	cmd.Flags().String("source", "", "")

	err := LookingGlassTraceroute(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error waiting for traceroute result")
	assert.True(t, errors.Is(err, waitErr), "underlying error should be preserved via %%w")
}

func TestLookingGlassTraceroute_InvalidDestination(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	defer config.SetLoginFunc(originalLoginFunc)

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})

	cmd := &cobra.Command{}
	cmd.Flags().String("destination", "not-an-ip", "")
	cmd.Flags().String("source", "", "")

	err := LookingGlassTraceroute(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "destination")
}

func TestLookingGlassTraceroute_InvalidSource(t *testing.T) {
	originalLoginFunc := config.GetLoginFunc()
	defer config.SetLoginFunc(originalLoginFunc)

	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})

	cmd := &cobra.Command{}
	cmd.Flags().String("destination", "8.8.8.8", "")
	cmd.Flags().String("source", "not-an-ip", "")

	err := LookingGlassTraceroute(cmd, []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source")
}

// Output conversion tests for ping/traceroute

func TestToPingResultOutput(t *testing.T) {
	result := &megaport.LookingGlassPingResult{
		RawOutput: "PING 8.8.8.8",
		Statistics: &megaport.LookingGlassPingStatistics{
			PacketsTransmitted: 4,
			PacketsReceived:    4,
			PacketLossPct:      0,
			RTTMinMs:           10.123,
			RTTAvgMs:           12.234,
			RTTMaxMs:           14.345,
			RTTMdevMs:          1.512,
		},
	}

	out, err := ToPingResultOutput(result)
	assert.NoError(t, err)
	assert.Equal(t, "4", out.PacketsTransmitted)
	assert.Equal(t, "4", out.PacketsReceived)
	assert.Equal(t, "0.0", out.PacketLossPct)
	assert.Equal(t, "10.123", out.RTTMinMs)
	assert.Equal(t, "12.234", out.RTTAvgMs)
	assert.Equal(t, "14.345", out.RTTMaxMs)
	assert.Equal(t, "1.512", out.RTTMdevMs)
	assert.Equal(t, "PING 8.8.8.8", out.RawOutput)
}

func TestToPingResultOutput_NoStatistics(t *testing.T) {
	result := &megaport.LookingGlassPingResult{RawOutput: "PING 8.8.8.8"}
	out, err := ToPingResultOutput(result)
	assert.NoError(t, err)
	assert.Equal(t, "PING 8.8.8.8", out.RawOutput)
	assert.Empty(t, out.PacketsTransmitted)
}

func TestToPingResultOutputNil(t *testing.T) {
	_, err := ToPingResultOutput(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid ping result: nil value")
}

func TestToTracerouteHopOutput(t *testing.T) {
	hop := &megaport.LookingGlassTracerouteHop{
		Hop: "1",
		Probes: []*megaport.LookingGlassTracerouteProbe{
			{Host: "10.0.0.1", RTTMs: 1.234},
			{Host: "10.0.0.2", RTTMs: 1.456},
		},
	}

	out, err := ToTracerouteHopOutput(hop)
	assert.NoError(t, err)
	assert.Equal(t, "1", out.Hop)
	assert.Equal(t, "10.0.0.1 (1.234ms), 10.0.0.2 (1.456ms)", out.Probes)
}

func TestToTracerouteHopOutput_TimeoutProbe(t *testing.T) {
	hop := &megaport.LookingGlassTracerouteHop{
		Hop: "2",
		Probes: []*megaport.LookingGlassTracerouteProbe{
			{Host: "", RTTMs: 0},
		},
	}

	out, err := ToTracerouteHopOutput(hop)
	assert.NoError(t, err)
	assert.Equal(t, "*", out.Probes)
}

func TestToTracerouteHopOutputNil(t *testing.T) {
	_, err := ToTracerouteHopOutput(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid traceroute hop: nil value")
}

func TestToTracerouteHopOutput_NilProbe(t *testing.T) {
	hop := &megaport.LookingGlassTracerouteHop{
		Hop:    "3",
		Probes: []*megaport.LookingGlassTracerouteProbe{nil},
	}

	out, err := ToTracerouteHopOutput(hop)
	assert.NoError(t, err)
	assert.Equal(t, "*", out.Probes)
}

func TestPrintTracerouteResultNil(t *testing.T) {
	err := printTracerouteResult(nil, "json", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid traceroute result: nil value")
}

func TestPrintTracerouteResult_NilHop(t *testing.T) {
	result := &megaport.LookingGlassTracerouteResult{
		Hops: []*megaport.LookingGlassTracerouteHop{nil},
	}
	out := output.CaptureOutput(func() {
		err := printTracerouteResult(result, "json", true)
		assert.NoError(t, err)
	})
	assert.Contains(t, out, `"hop": "*"`)
}

func TestPrintPingResultNil(t *testing.T) {
	err := printPingResult(nil, "json", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid ping result: nil value")
}
