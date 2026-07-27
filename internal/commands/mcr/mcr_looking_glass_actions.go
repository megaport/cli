package mcr

import (
	"fmt"
	"time"

	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/config"
	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/megaport/megaport-cli/internal/validation"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
)

// mcrDiagnosticsPollTimeout mirrors the SDK's own poll timeout for
// WaitForMCRPing/WaitForMCRTraceroute so the CLI's context doesn't cut the
// wait short before the SDK gives up on its own.
const mcrDiagnosticsPollTimeout = 5 * time.Minute

// ListLookingGlassIPRoutes lists IP routes from the MCR Looking Glass
func ListLookingGlassIPRoutes(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return fmt.Errorf("error logging in: %v", err)
	}

	mcrUID := args[0]

	// Get optional filters
	protocol, _ := cmd.Flags().GetString("protocol")
	ipFilter, _ := cmd.Flags().GetString("ip")

	spinner := output.PrintResourceListing("IP routes", noColor)

	var routes []*megaport.LookingGlassIPRoute

	if protocol != "" || ipFilter != "" {
		req := &megaport.ListIPRoutesRequest{
			MCRID:    mcrUID,
			IPFilter: ipFilter,
		}
		if protocol != "" {
			req.Protocol = megaport.RouteProtocol(protocol)
		}
		routes, err = listIPRoutesWithFilterFunc(ctx, client, req)
	} else {
		routes, err = listIPRoutesFunc(ctx, client, mcrUID)
	}

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to list IP routes: %v", noColor, err)
		return fmt.Errorf("error listing IP routes: %v", err)
	}

	if len(routes) == 0 {
		output.PrintWarning("No IP routes found", noColor)
	}

	return printIPRoutes(routes, outputFormat, noColor)
}

// ListLookingGlassBGPRoutes lists BGP routes from the MCR Looking Glass
func ListLookingGlassBGPRoutes(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return fmt.Errorf("error logging in: %v", err)
	}

	mcrUID := args[0]

	// Get optional filter
	ipFilter, _ := cmd.Flags().GetString("ip")

	spinner := output.PrintResourceListing("BGP routes", noColor)

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
		return fmt.Errorf("error listing BGP routes: %v", err)
	}

	if len(routes) == 0 {
		output.PrintWarning("No BGP routes found", noColor)
	}

	return printBGPRoutes(routes, outputFormat, noColor)
}

// ListLookingGlassBGPSessions lists BGP sessions from the MCR Looking Glass
func ListLookingGlassBGPSessions(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return fmt.Errorf("error logging in: %v", err)
	}

	mcrUID := args[0]

	spinner := output.PrintResourceListing("BGP sessions", noColor)

	sessions, err := listBGPSessionsFunc(ctx, client, mcrUID)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to list BGP sessions: %v", noColor, err)
		return fmt.Errorf("error listing BGP sessions: %v", err)
	}

	if len(sessions) == 0 {
		output.PrintWarning("No BGP sessions found", noColor)
	}

	return printBGPSessions(sessions, outputFormat, noColor)
}

// ListLookingGlassBGPNeighborRoutes lists routes advertised to or received from a specific BGP neighbor
func ListLookingGlassBGPNeighborRoutes(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return fmt.Errorf("error logging in: %v", err)
	}

	mcrUID := args[0]
	sessionID := args[1]
	direction := args[2]

	// Validate direction
	if direction != "advertised" && direction != "received" {
		return fmt.Errorf("direction must be 'advertised' or 'received', got: %s", direction)
	}

	// Get optional filter
	ipFilter, _ := cmd.Flags().GetString("ip")

	spinner := output.PrintResourceListing("BGP neighbor routes", noColor)

	req := &megaport.ListBGPNeighborRoutesRequest{
		MCRID:     mcrUID,
		SessionID: sessionID,
		Direction: megaport.LookingGlassRouteDirection(direction),
		IPFilter:  ipFilter,
	}

	routes, err := listBGPNeighborRoutesFunc(ctx, client, req)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to list BGP neighbor routes: %v", noColor, err)
		return fmt.Errorf("error listing BGP neighbor routes: %v", err)
	}

	if len(routes) == 0 {
		output.PrintWarning("No BGP neighbor routes found", noColor)
	}

	return printBGPNeighborRoutes(routes, outputFormat, noColor)
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
