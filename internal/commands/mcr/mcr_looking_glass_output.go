package mcr

import (
	"fmt"
	"strings"

	"github.com/megaport/megaport-cli/internal/base/output"
	megaport "github.com/megaport/megaportgo"
)

// IPRouteOutput represents the output format for IP routes
type IPRouteOutput struct {
	output.Output `json:"-" header:"-"`
	Prefix        string `json:"prefix" header:"Prefix"`
	NextHop       string `json:"next_hop" header:"Next Hop"`
	Protocol      string `json:"protocol" header:"Protocol"`
	Distance      int    `json:"distance" header:"Distance"`
	Metric        int    `json:"metric" header:"Metric"`
	VXCName       string `json:"vxc_name,omitempty" header:"VXC Name"`
}

// ToIPRouteOutput converts a megaport.LookingGlassIPRoute to IPRouteOutput
func ToIPRouteOutput(route *megaport.LookingGlassIPRoute) (IPRouteOutput, error) {
	if route == nil {
		return IPRouteOutput{}, fmt.Errorf("invalid route: nil value")
	}

	return IPRouteOutput{
		Prefix:   route.Prefix,
		NextHop:  route.NextHop.IP,
		Protocol: route.Protocol,
		Distance: route.Distance,
		Metric:   route.Metric,
		VXCName:  route.NextHop.VXC.Name,
	}, nil
}

// BGPRouteOutput represents the output format for BGP routes, both the full
// table and the per-neighbor views.
type BGPRouteOutput struct {
	output.Output `json:"-" header:"-"`
	Prefix        string `json:"prefix" header:"Prefix"`
	NextHop       string `json:"next_hop" header:"Next Hop"`
	ASPath        string `json:"as_path,omitempty" header:"AS Path"`
	LocalPref     int    `json:"local_pref" header:"Local Pref"`
	MED           int    `json:"med" header:"MED"`
	Weight        int    `json:"weight" header:"Weight"`
	Origin        string `json:"origin,omitempty" header:"Origin"`
	Source        string `json:"source,omitempty" header:"Source"`
	Communities   string `json:"communities,omitempty" header:"Communities"`
	AdvertisedTo  string `json:"advertised_to,omitempty" header:"Advertised To"`
	Valid         string `json:"valid" header:"Valid"`
	Best          string `json:"best" header:"Best"`
	External      string `json:"external" header:"External"`
	VXCName       string `json:"vxc_name,omitempty" header:"VXC Name"`
	Since         string `json:"since,omitempty" header:"Since"`
}

// ToBGPRouteOutput converts a megaport.LookingGlassBGPRoute to BGPRouteOutput
func ToBGPRouteOutput(route *megaport.LookingGlassBGPRoute) (BGPRouteOutput, error) {
	if route == nil {
		return BGPRouteOutput{}, fmt.Errorf("invalid BGP route: nil value")
	}

	return BGPRouteOutput{
		Prefix:       route.Prefix,
		NextHop:      route.NextHop.IP,
		ASPath:       route.ASPath,
		LocalPref:    route.LocalPref,
		MED:          route.MED,
		Weight:       route.Weight,
		Origin:       route.Origin,
		Source:       route.Source,
		Communities:  strings.Join(route.Communities, ", "),
		AdvertisedTo: strings.Join(route.AdvertisedTo, ", "),
		Valid:        boolToYesNo(route.Valid),
		Best:         boolToYesNo(route.Best),
		External:     boolToYesNo(route.External),
		VXCName:      route.NextHop.VXC.Name,
		Since:        route.Since,
	}, nil
}

// PingResultOutput represents the output format for a ping result
type PingResultOutput struct {
	output.Output      `json:"-" header:"-"`
	PacketsTransmitted string `json:"packets_transmitted,omitempty" header:"Packets Transmitted"`
	PacketsReceived    string `json:"packets_received,omitempty" header:"Packets Received"`
	PacketLossPct      string `json:"packet_loss_pct,omitempty" header:"Packet Loss %"`
	RTTMinMs           string `json:"rtt_min_ms,omitempty" header:"RTT Min (ms)"`
	RTTAvgMs           string `json:"rtt_avg_ms,omitempty" header:"RTT Avg (ms)"`
	RTTMaxMs           string `json:"rtt_max_ms,omitempty" header:"RTT Max (ms)"`
	RTTMdevMs          string `json:"rtt_mdev_ms,omitempty" header:"RTT Mdev (ms)"`
	RawOutput          string `json:"raw_output,omitempty" header:"Raw Output"`
}

// ToPingResultOutput converts a megaport.LookingGlassPingResult to PingResultOutput
func ToPingResultOutput(result *megaport.LookingGlassPingResult) (PingResultOutput, error) {
	if result == nil {
		return PingResultOutput{}, fmt.Errorf("invalid ping result: nil value")
	}

	out := PingResultOutput{
		RawOutput: result.RawOutput,
	}

	if result.Statistics != nil {
		stats := result.Statistics
		out.PacketsTransmitted = fmt.Sprintf("%d", stats.PacketsTransmitted)
		out.PacketsReceived = fmt.Sprintf("%d", stats.PacketsReceived)
		out.PacketLossPct = fmt.Sprintf("%.1f", stats.PacketLossPct)
		out.RTTMinMs = fmt.Sprintf("%.3f", stats.RTTMinMs)
		out.RTTAvgMs = fmt.Sprintf("%.3f", stats.RTTAvgMs)
		out.RTTMaxMs = fmt.Sprintf("%.3f", stats.RTTMaxMs)
		out.RTTMdevMs = fmt.Sprintf("%.3f", stats.RTTMdevMs)
	}

	return out, nil
}

// TracerouteHopOutput represents the output format for a single traceroute hop
type TracerouteHopOutput struct {
	output.Output `json:"-" header:"-"`
	Hop           string `json:"hop" header:"Hop"`
	Probes        string `json:"probes,omitempty" header:"Probes"`
}

// ToTracerouteHopOutput converts a megaport.LookingGlassTracerouteHop to TracerouteHopOutput
func ToTracerouteHopOutput(hop *megaport.LookingGlassTracerouteHop) (TracerouteHopOutput, error) {
	if hop == nil {
		return TracerouteHopOutput{}, fmt.Errorf("invalid traceroute hop: nil value")
	}

	out := TracerouteHopOutput{
		Hop: hop.Hop,
	}

	if len(hop.Probes) > 0 {
		probeStrs := make([]string, len(hop.Probes))
		for i, probe := range hop.Probes {
			if probe == nil {
				probeStrs[i] = "*"
				continue
			}
			host := probe.Host
			if host == "" {
				probeStrs[i] = "*"
				continue
			}
			probeStrs[i] = fmt.Sprintf("%s (%.3fms)", host, probe.RTTMs)
		}
		out.Probes = strings.Join(probeStrs, ", ")
	}

	return out, nil
}

// Helper functions

func boolToYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

// Print functions for each output type

func printIPRoutes(routes []*megaport.LookingGlassIPRoute, format string, noColor bool) error {
	outputs := make([]IPRouteOutput, 0, len(routes))
	for _, route := range routes {
		out, err := ToIPRouteOutput(route)
		if err != nil {
			return err
		}
		outputs = append(outputs, out)
	}
	return output.PrintOutput(outputs, format, noColor)
}

func printBGPRoutes(routes []*megaport.LookingGlassBGPRoute, format string, noColor bool) error {
	outputs := make([]BGPRouteOutput, 0, len(routes))
	for _, route := range routes {
		out, err := ToBGPRouteOutput(route)
		if err != nil {
			return err
		}
		outputs = append(outputs, out)
	}
	return output.PrintOutput(outputs, format, noColor)
}

func printPingResult(result *megaport.LookingGlassPingResult, format string, noColor bool) error {
	out, err := ToPingResultOutput(result)
	if err != nil {
		return err
	}
	return output.PrintOutput([]PingResultOutput{out}, format, noColor)
}

func printTracerouteResult(result *megaport.LookingGlassTracerouteResult, format string, noColor bool) error {
	if result == nil {
		return fmt.Errorf("invalid traceroute result: nil value")
	}
	outputs := make([]TracerouteHopOutput, 0, len(result.Hops))
	for _, hop := range result.Hops {
		if hop == nil {
			outputs = append(outputs, TracerouteHopOutput{Hop: "*"})
			continue
		}
		out, err := ToTracerouteHopOutput(hop)
		if err != nil {
			return err
		}
		outputs = append(outputs, out)
	}
	return output.PrintOutput(outputs, format, noColor)
}
