package mcr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/netip"
	"testing"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/config"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// Test helpers for the route commands

func mockLogin(t *testing.T) {
	t.Helper()
	original := config.GetLoginFunc()
	t.Cleanup(func() { config.SetLoginFunc(original) })
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return &megaport.Client{}, nil
	})
}

func mockLoginError(t *testing.T) {
	t.Helper()
	original := config.GetLoginFunc()
	t.Cleanup(func() { config.SetLoginFunc(original) })
	config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
		return nil, errors.New("login failed")
	})
}

func routeFlagsCmd(protocol, ip string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("protocol", protocol, "")
	cmd.Flags().String("ip", ip, "")
	cmd.SetOut(&bytes.Buffer{})
	return cmd
}

func sampleIPRoutes() []*megaport.LookingGlassIPRoute {
	return []*megaport.LookingGlassIPRoute{
		{
			Prefix:   "10.0.0.0/24",
			Protocol: "BGP",
			Distance: 20,
			Metric:   100,
			NextHop:  megaport.LookingGlassRouteNextHop{IP: "192.168.1.1", VXC: megaport.LookingGlassRouteVXCRef{ID: "vxc-1", Name: "Test VXC"}},
		},
		{
			Prefix:   "172.16.0.0/16",
			Protocol: "STATIC",
			Distance: 1,
			NextHop:  megaport.LookingGlassRouteNextHop{IP: "192.168.1.2"},
		},
	}
}

func sampleBGPRoute() *megaport.LookingGlassBGPRoute {
	return &megaport.LookingGlassBGPRoute{
		Prefix:      "10.0.0.0/24",
		ASPath:      "65001 65002",
		Origin:      "IGP",
		Source:      "EBGP",
		LocalPref:   100,
		MED:         50,
		Weight:      0,
		Best:        true,
		External:    true,
		Valid:       true,
		Since:       "2026-09-01T00:00:00Z",
		Communities: []string{"65001:100", "65001:200"},
		NextHop:     megaport.LookingGlassRouteNextHop{IP: "192.168.1.1", VXC: megaport.LookingGlassRouteVXCRef{ID: "vxc-1", Name: "Test VXC"}},
	}
}

// IP route action tests

func TestListLookingGlassIPRoutes(t *testing.T) {
	mockLogin(t)
	originalFunc := listIPRoutesFunc
	defer func() { listIPRoutesFunc = originalFunc }()

	listIPRoutesFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.LookingGlassIPRoute, error) {
		assert.Equal(t, "test-mcr-uid", mcrUID)
		return sampleIPRoutes(), nil
	}

	err := ListLookingGlassIPRoutes(routeFlagsCmd("", ""), []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestListLookingGlassIPRoutes_ProtocolFilterIsLocal(t *testing.T) {
	mockLogin(t)
	originalFunc := listIPRoutesFunc
	originalFilterFunc := listIPRoutesWithFilterFunc
	defer func() {
		listIPRoutesFunc = originalFunc
		listIPRoutesWithFilterFunc = originalFilterFunc
	}()

	// --protocol on its own must call the unfiltered endpoint: the API has no protocol parameter.
	listIPRoutesFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.LookingGlassIPRoute, error) {
		return sampleIPRoutes(), nil
	}
	listIPRoutesWithFilterFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListIPRoutesRequest) ([]*megaport.LookingGlassIPRoute, error) {
		t.Fatal("ListIPRoutesWithFilter must not be called when only --protocol is set")
		return nil, nil
	}

	err := ListLookingGlassIPRoutes(routeFlagsCmd("bgp", ""), []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestListLookingGlassIPRoutes_WithIPFilter(t *testing.T) {
	mockLogin(t)
	originalFunc := listIPRoutesWithFilterFunc
	defer func() { listIPRoutesWithFilterFunc = originalFunc }()

	listIPRoutesWithFilterFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListIPRoutesRequest) ([]*megaport.LookingGlassIPRoute, error) {
		assert.Equal(t, "test-mcr-uid", req.MCRID)
		assert.Equal(t, "10.0.0.0/8", req.IPFilter)
		return sampleIPRoutes()[:1], nil
	}

	err := ListLookingGlassIPRoutes(routeFlagsCmd("", "10.0.0.0/8"), []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestListLookingGlassIPRoutes_WithIPFilter_APIError(t *testing.T) {
	mockLogin(t)
	originalFunc := listIPRoutesWithFilterFunc
	defer func() { listIPRoutesWithFilterFunc = originalFunc }()

	listIPRoutesWithFilterFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListIPRoutesRequest) ([]*megaport.LookingGlassIPRoute, error) {
		return nil, errors.New("filter API error")
	}

	err := ListLookingGlassIPRoutes(routeFlagsCmd("", "10.0.0.0/8"), []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error listing IP routes")
}

func TestListLookingGlassIPRoutes_LoginError(t *testing.T) {
	mockLoginError(t)

	err := ListLookingGlassIPRoutes(routeFlagsCmd("", ""), []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error logging in")
}

func TestListLookingGlassIPRoutes_APIError(t *testing.T) {
	mockLogin(t)
	originalFunc := listIPRoutesFunc
	defer func() { listIPRoutesFunc = originalFunc }()

	listIPRoutesFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.LookingGlassIPRoute, error) {
		return nil, errors.New("API error")
	}

	err := ListLookingGlassIPRoutes(routeFlagsCmd("", ""), []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error listing IP routes")
}

func TestListLookingGlassIPRoutes_Empty(t *testing.T) {
	mockLogin(t)
	originalFunc := listIPRoutesFunc
	defer func() { listIPRoutesFunc = originalFunc }()

	listIPRoutesFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.LookingGlassIPRoute, error) {
		return []*megaport.LookingGlassIPRoute{}, nil
	}

	err := ListLookingGlassIPRoutes(routeFlagsCmd("", ""), []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestFilterIPRoutesByProtocol(t *testing.T) {
	routes := sampleIPRoutes()

	assert.Len(t, filterIPRoutesByProtocol(routes, ""), 2)

	bgp := filterIPRoutesByProtocol(routes, "bgp")
	assert.Len(t, bgp, 1)
	assert.Equal(t, "10.0.0.0/24", bgp[0].Prefix)

	static := filterIPRoutesByProtocol(routes, "STATIC")
	assert.Len(t, static, 1)
	assert.Equal(t, "172.16.0.0/16", static[0].Prefix)

	assert.Empty(t, filterIPRoutesByProtocol(routes, "OSPF"))
	assert.Empty(t, filterIPRoutesByProtocol([]*megaport.LookingGlassIPRoute{nil}, "BGP"))
}

// BGP route action tests

func TestListLookingGlassBGPRoutes(t *testing.T) {
	mockLogin(t)
	originalFunc := listBGPRoutesFunc
	defer func() { listBGPRoutesFunc = originalFunc }()

	listBGPRoutesFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.LookingGlassBGPRoute, error) {
		assert.Equal(t, "test-mcr-uid", mcrUID)
		return []*megaport.LookingGlassBGPRoute{sampleBGPRoute()}, nil
	}

	err := ListLookingGlassBGPRoutes(routeFlagsCmd("", ""), []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestListLookingGlassBGPRoutes_LoginError(t *testing.T) {
	mockLoginError(t)

	err := ListLookingGlassBGPRoutes(routeFlagsCmd("", ""), []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error logging in")
}

func TestListLookingGlassBGPRoutes_APIError(t *testing.T) {
	mockLogin(t)
	originalFunc := listBGPRoutesFunc
	defer func() { listBGPRoutesFunc = originalFunc }()

	listBGPRoutesFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.LookingGlassBGPRoute, error) {
		return nil, errors.New("BGP API error")
	}

	err := ListLookingGlassBGPRoutes(routeFlagsCmd("", ""), []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error listing BGP routes")
}

func TestListLookingGlassBGPRoutes_Empty(t *testing.T) {
	mockLogin(t)
	originalFunc := listBGPRoutesFunc
	defer func() { listBGPRoutesFunc = originalFunc }()

	listBGPRoutesFunc = func(ctx context.Context, client *megaport.Client, mcrUID string) ([]*megaport.LookingGlassBGPRoute, error) {
		return []*megaport.LookingGlassBGPRoute{}, nil
	}

	err := ListLookingGlassBGPRoutes(routeFlagsCmd("", ""), []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestListLookingGlassBGPRoutes_WithFilter(t *testing.T) {
	mockLogin(t)
	originalFunc := listBGPRoutesWithFilterFunc
	defer func() { listBGPRoutesWithFilterFunc = originalFunc }()

	listBGPRoutesWithFilterFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListBGPRoutesRequest) ([]*megaport.LookingGlassBGPRoute, error) {
		assert.Equal(t, "test-mcr-uid", req.MCRID)
		assert.Equal(t, "10.0.0.0/8", req.IPFilter)
		return []*megaport.LookingGlassBGPRoute{sampleBGPRoute()}, nil
	}

	err := ListLookingGlassBGPRoutes(routeFlagsCmd("", "10.0.0.0/8"), []string{"test-mcr-uid"}, true, "json")
	assert.NoError(t, err)
}

func TestListLookingGlassBGPRoutes_WithFilterError(t *testing.T) {
	mockLogin(t)
	originalFunc := listBGPRoutesWithFilterFunc
	defer func() { listBGPRoutesWithFilterFunc = originalFunc }()

	listBGPRoutesWithFilterFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListBGPRoutesRequest) ([]*megaport.LookingGlassBGPRoute, error) {
		return nil, errors.New("filter error")
	}

	err := ListLookingGlassBGPRoutes(routeFlagsCmd("", "10.0.0.0/8"), []string{"test-mcr-uid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error listing BGP routes")
}

// BGP neighbor route action tests

func TestListLookingGlassBGPNeighborRoutes_SendsPeerIPAndDirection(t *testing.T) {
	mockLogin(t)
	originalFunc := listBGPNeighborRoutesFunc
	defer func() { listBGPNeighborRoutesFunc = originalFunc }()

	var captured *megaport.ListBGPNeighborRoutesRequest
	listBGPNeighborRoutesFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListBGPNeighborRoutesRequest) ([]*megaport.LookingGlassBGPRoute, error) {
		captured = req
		return []*megaport.LookingGlassBGPRoute{sampleBGPRoute()}, nil
	}

	err := ListLookingGlassBGPNeighborRoutes(routeFlagsCmd("", ""), []string{"test-mcr-uid", "169.254.0.1", "received"}, true, "json")
	assert.NoError(t, err)
	assert.Equal(t, &megaport.ListBGPNeighborRoutesRequest{
		MCRID:         "test-mcr-uid",
		PeerIPAddress: "169.254.0.1",
		Direction:     megaport.BGPRouteDirectionReceived,
	}, captured)
	assert.Equal(t, "RECEIVED", captured.Direction)
}

func TestListLookingGlassBGPNeighborRoutes_Advertised(t *testing.T) {
	mockLogin(t)
	originalFunc := listBGPNeighborRoutesFunc
	defer func() { listBGPNeighborRoutesFunc = originalFunc }()

	listBGPNeighborRoutesFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListBGPNeighborRoutesRequest) ([]*megaport.LookingGlassBGPRoute, error) {
		assert.Equal(t, "ADVERTISED", req.Direction)
		assert.Equal(t, "2001:db8::1", req.PeerIPAddress)
		return []*megaport.LookingGlassBGPRoute{sampleBGPRoute()}, nil
	}

	err := ListLookingGlassBGPNeighborRoutes(routeFlagsCmd("", ""), []string{"test-mcr-uid", "2001:db8::1", "advertised"}, true, "json")
	assert.NoError(t, err)
}

func TestListLookingGlassBGPNeighborRoutes_InvalidDirection(t *testing.T) {
	mockLogin(t)
	originalFunc := listBGPNeighborRoutesFunc
	defer func() { listBGPNeighborRoutesFunc = originalFunc }()

	listBGPNeighborRoutesFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListBGPNeighborRoutesRequest) ([]*megaport.LookingGlassBGPRoute, error) {
		t.Fatal("API must not be called with an invalid direction")
		return nil, nil
	}

	err := ListLookingGlassBGPNeighborRoutes(routeFlagsCmd("", ""), []string{"test-mcr-uid", "169.254.0.1", "invalid"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "direction must be 'advertised' or 'received'")
}

func TestListLookingGlassBGPNeighborRoutes_InvalidPeerIP(t *testing.T) {
	mockLogin(t)
	originalFunc := listBGPNeighborRoutesFunc
	defer func() { listBGPNeighborRoutesFunc = originalFunc }()

	listBGPNeighborRoutesFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListBGPNeighborRoutesRequest) ([]*megaport.LookingGlassBGPRoute, error) {
		t.Fatal("API must not be called with an invalid peer IP")
		return nil, nil
	}

	err := ListLookingGlassBGPNeighborRoutes(routeFlagsCmd("", ""), []string{"test-mcr-uid", "session-123", "received"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "peer IP")
}

func TestListLookingGlassBGPNeighborRoutes_InvalidIPFilter(t *testing.T) {
	mockLogin(t)

	err := ListLookingGlassBGPNeighborRoutes(routeFlagsCmd("", "not-an-ip"), []string{"test-mcr-uid", "169.254.0.1", "received"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--ip must be an IP address or prefix")
}

func TestListLookingGlassBGPNeighborRoutes_IPFilterIsLocal(t *testing.T) {
	mockLogin(t)
	originalFunc := listBGPNeighborRoutesFunc
	defer func() { listBGPNeighborRoutesFunc = originalFunc }()

	inside := sampleBGPRoute()
	outside := sampleBGPRoute()
	outside.Prefix = "192.0.2.0/24"

	var captured *megaport.ListBGPNeighborRoutesRequest
	listBGPNeighborRoutesFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListBGPNeighborRoutesRequest) ([]*megaport.LookingGlassBGPRoute, error) {
		captured = req
		return []*megaport.LookingGlassBGPRoute{inside, outside}, nil
	}

	err := ListLookingGlassBGPNeighborRoutes(routeFlagsCmd("", "10.0.0.0/8"), []string{"test-mcr-uid", "169.254.0.1", "received"}, true, "json")
	assert.NoError(t, err)
	// The request carries no filter: the API has no ip_address parameter on this endpoint.
	assert.Equal(t, &megaport.ListBGPNeighborRoutesRequest{
		MCRID:         "test-mcr-uid",
		PeerIPAddress: "169.254.0.1",
		Direction:     megaport.BGPRouteDirectionReceived,
	}, captured)
}

func TestListLookingGlassBGPNeighborRoutes_LoginError(t *testing.T) {
	mockLoginError(t)

	err := ListLookingGlassBGPNeighborRoutes(routeFlagsCmd("", ""), []string{"test-mcr-uid", "169.254.0.1", "received"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error logging in")
}

func TestListLookingGlassBGPNeighborRoutes_APIError(t *testing.T) {
	mockLogin(t)
	originalFunc := listBGPNeighborRoutesFunc
	defer func() { listBGPNeighborRoutesFunc = originalFunc }()

	listBGPNeighborRoutesFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListBGPNeighborRoutesRequest) ([]*megaport.LookingGlassBGPRoute, error) {
		return nil, errors.New("neighbor API error")
	}

	err := ListLookingGlassBGPNeighborRoutes(routeFlagsCmd("", ""), []string{"test-mcr-uid", "169.254.0.1", "received"}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error listing BGP neighbor routes")
}

func TestListLookingGlassBGPNeighborRoutes_Empty(t *testing.T) {
	mockLogin(t)
	originalFunc := listBGPNeighborRoutesFunc
	defer func() { listBGPNeighborRoutesFunc = originalFunc }()

	listBGPNeighborRoutesFunc = func(ctx context.Context, client *megaport.Client, req *megaport.ListBGPNeighborRoutesRequest) ([]*megaport.LookingGlassBGPRoute, error) {
		return []*megaport.LookingGlassBGPRoute{}, nil
	}

	err := ListLookingGlassBGPNeighborRoutes(routeFlagsCmd("", ""), []string{"test-mcr-uid", "169.254.0.1", "received"}, true, "json")
	assert.NoError(t, err)
}

func TestParseBGPRouteDirection(t *testing.T) {
	d, err := parseBGPRouteDirection("received")
	assert.NoError(t, err)
	assert.Equal(t, "RECEIVED", d)

	d, err = parseBGPRouteDirection("advertised")
	assert.NoError(t, err)
	assert.Equal(t, "ADVERTISED", d)

	_, err = parseBGPRouteDirection("RECEIVED")
	assert.Error(t, err)
}

func TestFilterBGPRoutesByIP(t *testing.T) {
	r1 := sampleBGPRoute() // 10.0.0.0/24
	r2 := sampleBGPRoute()
	r2.Prefix = "10.1.0.0/16"
	r3 := sampleBGPRoute()
	r3.Prefix = "192.0.2.0/24"
	r4 := sampleBGPRoute()
	r4.Prefix = "garbage"
	routes := []*megaport.LookingGlassBGPRoute{r1, r2, r3, r4, nil}

	assert.Len(t, filterBGPRoutesByIP(routes, netip.Prefix{}), 5)

	// Prefix filter keeps the routes inside it.
	kept := filterBGPRoutesByIP(routes, netip.MustParsePrefix("10.0.0.0/8"))
	assert.Len(t, kept, 2)
	assert.Equal(t, "10.0.0.0/24", kept[0].Prefix)
	assert.Equal(t, "10.1.0.0/16", kept[1].Prefix)

	// Address filter keeps the routes that contain it.
	kept = filterBGPRoutesByIP(routes, netip.MustParsePrefix("192.0.2.77/32"))
	assert.Len(t, kept, 1)
	assert.Equal(t, "192.0.2.0/24", kept[0].Prefix)

	// A prefix filter more specific than a route keeps the route that covers it.
	kept = filterBGPRoutesByIP(routes, netip.MustParsePrefix("10.1.2.0/24"))
	assert.Len(t, kept, 1)
	assert.Equal(t, "10.1.0.0/16", kept[0].Prefix)

	assert.Empty(t, filterBGPRoutesByIP(routes, netip.MustParsePrefix("203.0.113.0/24")))
}

func TestParseIPFilter(t *testing.T) {
	p, err := parseIPFilter("10.0.0.0/8")
	assert.NoError(t, err)
	assert.Equal(t, "10.0.0.0/8", p.String())

	p, err = parseIPFilter("10.1.2.3/8")
	assert.NoError(t, err)
	assert.Equal(t, "10.0.0.0/8", p.String())

	p, err = parseIPFilter("192.168.1.1")
	assert.NoError(t, err)
	assert.Equal(t, "192.168.1.1/32", p.String())

	p, err = parseIPFilter("2001:db8::1")
	assert.NoError(t, err)
	assert.Equal(t, "2001:db8::1/128", p.String())

	_, err = parseIPFilter("not-an-ip")
	assert.Error(t, err)
}

// Output conversion tests

func TestToIPRouteOutput(t *testing.T) {
	out, err := ToIPRouteOutput(sampleIPRoutes()[0])
	assert.NoError(t, err)
	assert.Equal(t, IPRouteOutput{
		Prefix:   "10.0.0.0/24",
		NextHop:  "192.168.1.1",
		Protocol: "BGP",
		Distance: 20,
		Metric:   100,
		VXCName:  "Test VXC",
	}, out)
}

func TestToIPRouteOutput_MinimalFields(t *testing.T) {
	out, err := ToIPRouteOutput(&megaport.LookingGlassIPRoute{Prefix: "10.0.0.0/24", Protocol: "STATIC"})
	assert.NoError(t, err)
	assert.Equal(t, "10.0.0.0/24", out.Prefix)
	assert.Equal(t, "STATIC", out.Protocol)
	assert.Empty(t, out.NextHop)
	assert.Empty(t, out.VXCName)
	assert.Zero(t, out.Distance)
	assert.Zero(t, out.Metric)
}

func TestToIPRouteOutputNil(t *testing.T) {
	_, err := ToIPRouteOutput(nil)
	assert.Error(t, err)
}

func TestToBGPRouteOutput(t *testing.T) {
	out, err := ToBGPRouteOutput(sampleBGPRoute())
	assert.NoError(t, err)
	assert.Equal(t, BGPRouteOutput{
		Prefix:      "10.0.0.0/24",
		NextHop:     "192.168.1.1",
		ASPath:      "65001 65002",
		LocalPref:   100,
		MED:         50,
		Weight:      0,
		Origin:      "IGP",
		Source:      "EBGP",
		Communities: "65001:100, 65001:200",
		Valid:       "Yes",
		Best:        "Yes",
		External:    "Yes",
		VXCName:     "Test VXC",
		Since:       "2026-09-01T00:00:00Z",
	}, out)
}

func TestToBGPRouteOutput_MinimalFields(t *testing.T) {
	out, err := ToBGPRouteOutput(&megaport.LookingGlassBGPRoute{Prefix: "10.0.0.0/24"})
	assert.NoError(t, err)
	assert.Equal(t, "10.0.0.0/24", out.Prefix)
	assert.Empty(t, out.ASPath)
	assert.Empty(t, out.Communities)
	assert.Equal(t, "No", out.Valid)
	assert.Equal(t, "No", out.Best)
	assert.Equal(t, "No", out.External)
	assert.Empty(t, out.Since)
}

func TestToBGPRouteOutputNil(t *testing.T) {
	_, err := ToBGPRouteOutput(nil)
	assert.Error(t, err)
}

func TestBoolToYesNo(t *testing.T) {
	assert.Equal(t, "Yes", boolToYesNo(true))
	assert.Equal(t, "No", boolToYesNo(false))
}

func TestPrintRoutes_TableFormat(t *testing.T) {
	assert.NoError(t, printIPRoutes(sampleIPRoutes(), "table", true))
	assert.NoError(t, printBGPRoutes([]*megaport.LookingGlassBGPRoute{sampleBGPRoute()}, "table", true))
}

func TestPrintRoutes_NilEntry(t *testing.T) {
	assert.Error(t, printIPRoutes([]*megaport.LookingGlassIPRoute{nil}, "json", true))
	assert.Error(t, printBGPRoutes([]*megaport.LookingGlassBGPRoute{nil}, "json", true))
}

// Wrapper function tests

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

	t.Run("listBGPNeighborRoutesFunc", func(t *testing.T) {
		req := &megaport.ListBGPNeighborRoutesRequest{MCRID: "mcr-1", PeerIPAddress: "169.254.0.1", Direction: megaport.BGPRouteDirectionReceived}
		mockSvc.ListBGPNeighborRoutesResult = []*megaport.LookingGlassBGPRoute{{Prefix: "1.0.0.0/8"}}
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
