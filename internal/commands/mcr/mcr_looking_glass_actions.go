package mcr

import (
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/config"
	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/megaport/megaport-cli/internal/validation"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
)

// mcrDiagnosticsPollTimeout mirrors the SDK's own poll timeout for the
// looking-glass operations so the CLI's context doesn't cut the wait short
// before the SDK gives up on its own.
const mcrDiagnosticsPollTimeout = 5 * time.Minute

// ListLookingGlassIPRoutes lists IP routes from the MCR Looking Glass
func ListLookingGlassIPRoutes(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	mcrUID := args[0]

	// --ip is a query parameter on the API. --protocol is not, so it is applied
	// to the returned slice below.
	protocol, _ := cmd.Flags().GetString("protocol")
	ipFilter, _ := cmd.Flags().GetString("ip")

	if protocol != "" && !validLookingGlassProtocols[strings.ToUpper(protocol)] {
		err := fmt.Errorf("invalid protocol %q: must be one of BGP, STATIC, CONNECTED, or LOCAL", protocol)
		output.PrintError("%v", noColor, err)
		return exitcodes.NewUsageError(err)
	}

	ctx, cancel := utils.ContextFromCmdWithDefault(cmd, mcrDiagnosticsPollTimeout)
	defer cancel()

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return fmt.Errorf("error logging in: %v", err)
	}

	spinner := output.PrintResourceListing("IP route", noColor)

	var routes []*megaport.LookingGlassIPRoute

	if ipFilter != "" {
		routes, err = listIPRoutesWithFilterFunc(ctx, client, &megaport.ListIPRoutesRequest{
			MCRID:    mcrUID,
			IPFilter: ipFilter,
		})
	} else {
		routes, err = listIPRoutesFunc(ctx, client, mcrUID)
	}

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to list IP routes: %v", noColor, err)
		return fmt.Errorf("error listing IP routes: %w", err)
	}

	routes = filterIPRoutesByProtocol(routes, protocol)

	if len(routes) == 0 {
		output.PrintWarning("No IP routes found", noColor)
	}

	return printIPRoutes(routes, outputFormat, noColor)
}

// ListLookingGlassBGPRoutes lists BGP routes from the MCR Looking Glass
func ListLookingGlassBGPRoutes(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	ctx, cancel := utils.ContextFromCmdWithDefault(cmd, mcrDiagnosticsPollTimeout)
	defer cancel()

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return fmt.Errorf("error logging in: %v", err)
	}

	mcrUID := args[0]

	// Get optional filter
	ipFilter, _ := cmd.Flags().GetString("ip")

	spinner := output.PrintResourceListing("BGP route", noColor)

	var routes []*megaport.LookingGlassBGPRoute

	if ipFilter != "" {
		req := &megaport.ListBGPRoutesRequest{
			MCRID:    mcrUID,
			IPFilter: ipFilter,
		}
		routes, err = listBGPRoutesWithFilterFunc(ctx, client, req)
	} else {
		routes, err = listBGPRoutesFunc(ctx, client, mcrUID)
	}

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to list BGP routes: %v", noColor, err)
		return fmt.Errorf("error listing BGP routes: %w", err)
	}

	if len(routes) == 0 {
		output.PrintWarning("No BGP routes found", noColor)
	}

	return printBGPRoutes(routes, outputFormat, noColor)
}

// ListLookingGlassBGPNeighborRoutes lists routes advertised to or received from one BGP peer
func ListLookingGlassBGPNeighborRoutes(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	mcrUID := args[0]
	peerIP := args[1]
	direction, err := parseBGPRouteDirection(args[2])
	if err != nil {
		return exitcodes.NewUsageError(err)
	}
	if err := validation.ValidateIPAddress(peerIP, "peer IP"); err != nil {
		return exitcodes.NewUsageError(err)
	}

	// The neighbor endpoint has no ip_address query parameter, so --ip is
	// applied to the returned slice.
	ipFilter, _ := cmd.Flags().GetString("ip")
	var want netip.Prefix
	if ipFilter != "" {
		if want, err = parseIPFilter(ipFilter); err != nil {
			return exitcodes.NewUsageError(err)
		}
	}

	ctx, cancel := utils.ContextFromCmdWithDefault(cmd, mcrDiagnosticsPollTimeout)
	defer cancel()

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return fmt.Errorf("error logging in: %v", err)
	}

	spinner := output.PrintResourceListing("BGP neighbor route", noColor)

	routes, err := listBGPNeighborRoutesFunc(ctx, client, &megaport.ListBGPNeighborRoutesRequest{
		MCRID:         mcrUID,
		PeerIPAddress: peerIP,
		Direction:     direction,
	})

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to list BGP neighbor routes: %v", noColor, err)
		return fmt.Errorf("error listing BGP neighbor routes: %w", err)
	}

	routes = filterBGPRoutesByIP(routes, want)

	if len(routes) == 0 {
		output.PrintWarning("No BGP neighbor routes found", noColor)
	}

	return printBGPRoutes(routes, outputFormat, noColor)
}

// parseBGPRouteDirection maps the CLI's lower-case direction argument to the API enum.
func parseBGPRouteDirection(arg string) (string, error) {
	switch arg {
	case "advertised":
		return megaport.BGPRouteDirectionAdvertised, nil
	case "received":
		return megaport.BGPRouteDirectionReceived, nil
	}
	return "", fmt.Errorf("direction must be 'advertised' or 'received', got: %s", arg)
}

// validLookingGlassProtocols are the routing protocols megalith's
// ProtocolType enum serializes on a route (STATIC, CONNECTED, BGP, LOCAL).
var validLookingGlassProtocols = map[string]bool{
	"STATIC":    true,
	"CONNECTED": true,
	"BGP":       true,
	"LOCAL":     true,
}

// filterIPRoutesByProtocol keeps the routes whose protocol matches, ignoring
// case. An empty protocol keeps every route.
func filterIPRoutesByProtocol(routes []*megaport.LookingGlassIPRoute, protocol string) []*megaport.LookingGlassIPRoute {
	if protocol == "" {
		return routes
	}
	kept := make([]*megaport.LookingGlassIPRoute, 0, len(routes))
	for _, r := range routes {
		if r != nil && strings.EqualFold(r.Protocol, protocol) {
			kept = append(kept, r)
		}
	}
	return kept
}

// parseIPFilter accepts a single address or a prefix. A single address becomes
// a host prefix.
func parseIPFilter(filter string) (netip.Prefix, error) {
	if p, err := netip.ParsePrefix(filter); err == nil {
		return p.Masked(), nil
	}
	if a, err := netip.ParseAddr(filter); err == nil {
		return netip.PrefixFrom(a, a.BitLen()), nil
	}
	return netip.Prefix{}, fmt.Errorf("--ip must be an IP address or prefix, got: %s", filter)
}

// filterBGPRoutesByIP keeps the routes whose prefix contains the filter address,
// or falls inside the filter prefix. An invalid (zero) prefix keeps every route.
func filterBGPRoutesByIP(routes []*megaport.LookingGlassBGPRoute, want netip.Prefix) []*megaport.LookingGlassBGPRoute {
	if !want.IsValid() {
		return routes
	}
	kept := make([]*megaport.LookingGlassBGPRoute, 0, len(routes))
	for _, r := range routes {
		if r == nil {
			continue
		}
		have, err := netip.ParsePrefix(r.Prefix)
		if err != nil {
			continue
		}
		have = have.Masked()
		if have.Contains(want.Addr()) || want.Contains(have.Addr()) {
			kept = append(kept, r)
		}
	}
	return kept
}

// LookingGlassPing runs an ICMP ping from the MCR Looking Glass to a destination
func LookingGlassPing(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	mcrUID := args[0]
	destination, _ := cmd.Flags().GetString("destination")
	if destination == "" {
		output.PrintError("--destination is required", noColor)
		return exitcodes.NewUsageError(fmt.Errorf("--destination is required"))
	}
	if err := validation.ValidateIPAddress(destination, "destination"); err != nil {
		return exitcodes.NewUsageError(err)
	}
	source, _ := cmd.Flags().GetString("source")
	if source != "" {
		if err := validation.ValidateIPAddress(source, "source"); err != nil {
			return exitcodes.NewUsageError(err)
		}
	}

	req := &megaport.MCRPingRequest{
		MCRID:              mcrUID,
		DestinationAddress: destination,
		SourceAddress:      source,
	}

	if cmd.Flags().Changed("packet-count") {
		packetCount, _ := cmd.Flags().GetInt("packet-count")
		if err := validation.ValidateIntRange(packetCount, 1, 60, "packet count"); err != nil {
			return exitcodes.NewUsageError(err)
		}
		count32 := int32(packetCount)
		req.PacketCount = &count32
	}
	if cmd.Flags().Changed("packet-size") {
		packetSize, _ := cmd.Flags().GetInt("packet-size")
		if err := validation.ValidateIntRange(packetSize, 1, 9186, "packet size"); err != nil {
			return exitcodes.NewUsageError(err)
		}
		size32 := int32(packetSize)
		req.PacketSize = &size32
	}

	ctx, cancel := utils.ContextFromCmdWithDefault(cmd, mcrDiagnosticsPollTimeout)
	defer cancel()

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return fmt.Errorf("error logging in: %v", err)
	}

	spinner := output.PrintCustomSpinner("Running ping on", mcrUID, noColor)

	operationID, err := pingMCRFunc(ctx, client, req)
	if err != nil {
		spinner.Stop()
		output.PrintError("Failed to start ping: %v", noColor, err)
		return fmt.Errorf("error starting ping: %w", err)
	}

	result, err := waitForMCRPingFunc(ctx, client, mcrUID, operationID)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to get ping result: %v", noColor, err)
		return fmt.Errorf("error waiting for ping result: %w", err)
	}

	return printPingResult(result, outputFormat, noColor)
}

// LookingGlassTraceroute runs a traceroute from the MCR Looking Glass to a destination
func LookingGlassTraceroute(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	mcrUID := args[0]
	destination, _ := cmd.Flags().GetString("destination")
	if destination == "" {
		output.PrintError("--destination is required", noColor)
		return exitcodes.NewUsageError(fmt.Errorf("--destination is required"))
	}
	if err := validation.ValidateIPAddress(destination, "destination"); err != nil {
		return exitcodes.NewUsageError(err)
	}
	source, _ := cmd.Flags().GetString("source")
	if source != "" {
		if err := validation.ValidateIPAddress(source, "source"); err != nil {
			return exitcodes.NewUsageError(err)
		}
	}

	req := &megaport.MCRTracerouteRequest{
		MCRID:              mcrUID,
		DestinationAddress: destination,
		SourceAddress:      source,
	}

	ctx, cancel := utils.ContextFromCmdWithDefault(cmd, mcrDiagnosticsPollTimeout)
	defer cancel()

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return fmt.Errorf("error logging in: %v", err)
	}

	spinner := output.PrintCustomSpinner("Running traceroute on", mcrUID, noColor)

	operationID, err := tracerouteMCRFunc(ctx, client, req)
	if err != nil {
		spinner.Stop()
		output.PrintError("Failed to start traceroute: %v", noColor, err)
		return fmt.Errorf("error starting traceroute: %w", err)
	}

	result, err := waitForMCRTracerouteFunc(ctx, client, mcrUID, operationID)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to get traceroute result: %v", noColor, err)
		return fmt.Errorf("error waiting for traceroute result: %w", err)
	}

	if result != nil && len(result.Hops) == 0 {
		output.PrintWarning("No traceroute hops found", noColor)
	}

	return printTracerouteResult(result, outputFormat, noColor)
}
