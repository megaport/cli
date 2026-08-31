package vxc

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/megaport/megaport-cli/internal/validation"
	megaport "github.com/megaport/megaportgo"
)

func promptVRouterConfig(noColor bool) (*megaport.VXCOrderVrouterPartnerConfig, error) {
	config := &megaport.VXCOrderVrouterPartnerConfig{
		Interfaces: []megaport.PartnerConfigInterface{},
	}

	interfaceCountStr, err := utils.ResourcePrompt("vxc", "Number of interfaces to configure: ", noColor)
	if err != nil {
		return nil, err
	}
	interfaceCount, err := strconv.Atoi(interfaceCountStr)
	if err != nil || interfaceCount < 1 {
		return nil, fmt.Errorf("number of interfaces must be a positive integer")
	}

	for i := 0; i < interfaceCount; i++ {
		iface := megaport.PartnerConfigInterface{}

		vlanStr, err := utils.ResourcePrompt("vxc", fmt.Sprintf("VLAN (%d=auto-assign, %d=untagged, %d-%d; press Enter for untagged): ", validation.AutoAssignVLAN, validation.UntaggedVLAN, validation.MinAssignableVLAN, validation.MaxVLAN), noColor)
		if err != nil {
			return nil, err
		}

		if vlanStr != "" {
			vlan, err := strconv.Atoi(vlanStr)
			if err != nil {
				return nil, fmt.Errorf("VLAN must be a valid integer")
			}
			if err := validation.ValidateVLAN(vlan); err != nil {
				return nil, validation.NewValidationError("VRouter interface VLAN", vlan,
					fmt.Sprintf("must be %d for auto-assign, %d for untagged, or %d-%d (%d is reserved)",
						validation.AutoAssignVLAN, validation.UntaggedVLAN, validation.MinAssignableVLAN, validation.MaxVLAN, validation.ReservedVLAN))
			}
			iface.VLAN = vlan
		} else {
			iface.VLAN = validation.UntaggedVLAN
		}

		interfaceType, err := utils.ResourcePrompt("vxc", fmt.Sprintf("Interface type (%s/%s, default %s): ", megaport.InterfaceTypeSubInterface, megaport.InterfaceTypeIPSecTunnel, megaport.InterfaceTypeSubInterface), noColor)
		if err != nil {
			return nil, err
		}
		if interfaceType != "" && interfaceType != megaport.InterfaceTypeSubInterface && interfaceType != megaport.InterfaceTypeIPSecTunnel {
			return nil, fmt.Errorf("interface type must be %s or %s", megaport.InterfaceTypeSubInterface, megaport.InterfaceTypeIPSecTunnel)
		}
		iface.InterfaceType = interfaceType

		// An ipSecTunnel interface carries only its tunnel; the source IP lives on
		// a separate subInterface, so the IP/route/NAT/BFD/BGP prompts are skipped.
		if interfaceType == megaport.InterfaceTypeIPSecTunnel {
			tunnel, err := promptIPsecTunnel(noColor)
			if err != nil {
				return nil, err
			}
			iface.IpSecTunnelOptions = tunnel
		} else if err := promptVRouterSubInterface(&iface, noColor); err != nil {
			return nil, err
		}

		config.Interfaces = append(config.Interfaces, iface)
	}

	// Validate before returning so the interactive path rejects bad input with a
	// clear message before any API call, matching the flag and JSON paths.
	if err := validation.ValidateVrouterPartnerConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// promptVRouterSubInterface collects the IP addresses, routes, NAT IPs, BFD, and
// BGP connections that apply to a subInterface but not to an ipSecTunnel interface.
func promptVRouterSubInterface(iface *megaport.PartnerConfigInterface, noColor bool) error {
	ipAddrs, err := promptIPAddresses("IP Addresses (CIDR notation, e.g., 192.168.1.1/30)", noColor)
	if err != nil {
		return err
	}
	iface.IpAddresses = ipAddrs

	hasRoutes, err := utils.ResourcePrompt("vxc", "Do you want to add IP routes? (yes/no): ", noColor)
	if err != nil {
		return err
	}
	if strings.ToLower(hasRoutes) == "yes" {
		routes, err := promptIPRoutes(noColor)
		if err != nil {
			return err
		}
		iface.IpRoutes = routes
	}

	hasNatIPs, err := utils.ResourcePrompt("vxc", "Do you want to add NAT IP addresses? (yes/no): ", noColor)
	if err != nil {
		return err
	}
	if strings.ToLower(hasNatIPs) == "yes" {
		natIPs, err := promptNATIPAddresses(noColor)
		if err != nil {
			return err
		}
		iface.NatIpAddresses = natIPs
	}

	hasBFD, err := utils.ResourcePrompt("vxc", "Do you want to configure BFD? (yes/no): ", noColor)
	if err != nil {
		return err
	}
	if strings.ToLower(hasBFD) == "yes" {
		bfd, err := promptBFDConfig(noColor)
		if err != nil {
			return err
		}
		iface.Bfd = bfd
	}

	hasBGP, err := utils.ResourcePrompt("vxc", "Do you want to configure BGP connections? (yes/no): ", noColor)
	if err != nil {
		return err
	}
	if strings.ToLower(hasBGP) == "yes" {
		bgpConns, err := promptBGPConnections(noColor)
		if err != nil {
			return err
		}
		iface.BgpConnections = bgpConns
	}

	return nil
}

func promptIPRoutes(noColor bool) ([]megaport.IpRoute, error) {
	var routes []megaport.IpRoute

	for {
		addRoute, err := utils.ResourcePrompt("vxc", "Add an IP route? (yes/no): ", noColor)
		if err != nil {
			return nil, err
		}
		if strings.ToLower(addRoute) != "yes" {
			break
		}

		prefix, err := utils.ResourcePrompt("vxc", "Enter prefix (e.g., 192.168.0.0/24): ", noColor)
		if err != nil {
			return nil, err
		}

		nextHop, err := utils.ResourcePrompt("vxc", "Enter next hop IP: ", noColor)
		if err != nil {
			return nil, err
		}

		description, err := utils.ResourcePrompt("vxc", "Enter description (optional): ", noColor)
		if err != nil {
			return nil, err
		}

		route := megaport.IpRoute{
			Prefix:      prefix,
			NextHop:     nextHop,
			Description: description,
		}

		routes = append(routes, route)
	}

	return routes, nil
}

func promptIPAddresses(message string, noColor bool) ([]string, error) {
	var addresses []string

	for {
		addIP, err := utils.ResourcePrompt("vxc", fmt.Sprintf("Add %s? (yes/no): ", message), noColor)
		if err != nil {
			return nil, err
		}
		if strings.ToLower(addIP) != "yes" {
			break
		}

		ip, err := utils.ResourcePrompt("vxc", "Enter IP address: ", noColor)
		if err != nil {
			return nil, err
		}

		addresses = append(addresses, ip)
	}

	return addresses, nil
}

func promptNATIPAddresses(noColor bool) ([]string, error) {
	return promptIPAddresses("a NAT IP address", noColor)
}

func promptBFDConfig(noColor bool) (megaport.BfdConfig, error) {
	bfd := megaport.BfdConfig{}

	txIntervalStr, err := utils.ResourcePrompt("vxc", "Enter transmit interval in ms (default 300): ", noColor)
	if err != nil {
		return bfd, err
	}
	if txIntervalStr != "" {
		txInterval, err := strconv.Atoi(txIntervalStr)
		if err != nil {
			return bfd, fmt.Errorf("transmit interval must be an integer")
		}
		bfd.TxInterval = txInterval
	} else {
		bfd.TxInterval = 300
	}

	rxIntervalStr, err := utils.ResourcePrompt("vxc", "Enter receive interval in ms (default 300): ", noColor)
	if err != nil {
		return bfd, err
	}
	if rxIntervalStr != "" {
		rxInterval, err := strconv.Atoi(rxIntervalStr)
		if err != nil {
			return bfd, fmt.Errorf("receive interval must be an integer")
		}
		bfd.RxInterval = rxInterval
	} else {
		bfd.RxInterval = 300
	}

	multiplierStr, err := utils.ResourcePrompt("vxc", "Enter multiplier (default 3): ", noColor)
	if err != nil {
		return bfd, err
	}
	if multiplierStr != "" {
		multiplier, err := strconv.Atoi(multiplierStr)
		if err != nil {
			return bfd, fmt.Errorf("multiplier must be an integer")
		}
		bfd.Multiplier = multiplier
	} else {
		bfd.Multiplier = 3
	}

	return bfd, nil
}

func promptBGPConnections(noColor bool) ([]megaport.BgpConnectionConfig, error) {
	var bgpConnections []megaport.BgpConnectionConfig

	for {
		addBGP, err := utils.ResourcePrompt("vxc", "Add a BGP connection? (yes/no): ", noColor)
		if err != nil {
			return nil, err
		}
		if strings.ToLower(addBGP) != "yes" {
			break
		}

		bgp := megaport.BgpConnectionConfig{}

		peerAsn, localIP, peerIP, err := promptBGPRequiredFields(noColor)
		if err != nil {
			return nil, err
		}
		bgp.PeerAsn = peerAsn
		bgp.LocalIpAddress = localIP
		bgp.PeerIpAddress = peerIP

		if err := promptBGPOptionalConfig(&bgp, noColor); err != nil {
			return nil, err
		}

		if err := promptBGPExportAddresses(&bgp, noColor); err != nil {
			return nil, err
		}

		if err := promptBGPPrefixLists(&bgp, noColor); err != nil {
			return nil, err
		}

		bgpConnections = append(bgpConnections, bgp)
	}

	return bgpConnections, nil
}

func promptBGPRequiredFields(noColor bool) (peerAsn int, localIP string, peerIP string, err error) {
	peerAsnStr, err := utils.ResourcePrompt("vxc", "Enter peer ASN (required): ", noColor)
	if err != nil {
		return 0, "", "", err
	}
	peerAsn, err = strconv.Atoi(peerAsnStr)
	if err != nil || peerAsn <= 0 {
		return 0, "", "", fmt.Errorf("peer ASN must be a positive integer")
	}

	localIP, err = utils.ResourcePrompt("vxc", "Enter local IP address (required): ", noColor)
	if err != nil {
		return 0, "", "", err
	}
	if localIP == "" {
		return 0, "", "", fmt.Errorf("local IP address is required")
	}

	peerIP, err = utils.ResourcePrompt("vxc", "Enter peer IP address (required): ", noColor)
	if err != nil {
		return 0, "", "", err
	}
	if peerIP == "" {
		return 0, "", "", fmt.Errorf("peer IP address is required")
	}

	return peerAsn, localIP, peerIP, nil
}

func promptBGPOptionalConfig(bgp *megaport.BgpConnectionConfig, noColor bool) error {
	localAsnStr, err := utils.ResourcePrompt("vxc", "Enter local ASN (optional): ", noColor)
	if err != nil {
		return err
	}
	if localAsnStr != "" {
		localAsn, err := strconv.Atoi(localAsnStr)
		if err != nil || localAsn <= 0 {
			return fmt.Errorf("local ASN must be a positive integer")
		}
		bgp.LocalAsn = &localAsn
	}

	password, err := utils.SecretResourcePrompt("vxc", "Enter password (optional): ", noColor)
	if err != nil {
		return err
	}
	bgp.Password = password

	shutdownStr, err := utils.ResourcePrompt("vxc", "Shutdown connection? (yes/no, default: no): ", noColor)
	if err != nil {
		return err
	}
	bgp.Shutdown = strings.ToLower(shutdownStr) == "yes"

	description, err := utils.ResourcePrompt("vxc", "Enter description (optional): ", noColor)
	if err != nil {
		return err
	}
	bgp.Description = description

	bfdEnabledStr, err := utils.ResourcePrompt("vxc", "Enable BFD? (yes/no, default: no): ", noColor)
	if err != nil {
		return err
	}
	bgp.BfdEnabled = strings.ToLower(bfdEnabledStr) == "yes"

	asOverrideStr, err := utils.ResourcePrompt("vxc", "Enable AS Override? (yes/no, optional - press Enter for API default): ", noColor)
	if err != nil {
		return err
	}
	if asOverrideStr != "" {
		switch strings.ToLower(asOverrideStr) {
		case "yes":
			v := true
			bgp.AsOverride = &v
		case "no":
			v := false
			bgp.AsOverride = &v
		default:
			return fmt.Errorf("AS Override must be 'yes' or 'no'")
		}
	}

	exportPolicy, err := utils.ResourcePrompt("vxc", "Enter export policy (permit/deny, optional): ", noColor)
	if err != nil {
		return err
	}
	if exportPolicy != "" && exportPolicy != "permit" && exportPolicy != "deny" {
		return fmt.Errorf("export policy must be 'permit' or 'deny'")
	}
	bgp.ExportPolicy = exportPolicy

	peerType, err := utils.ResourcePrompt("vxc", "Enter peer type (NON_CLOUD/PRIV_CLOUD/PUB_CLOUD, optional): ", noColor)
	if err != nil {
		return err
	}
	if peerType != "" && peerType != "NON_CLOUD" && peerType != "PRIV_CLOUD" && peerType != "PUB_CLOUD" {
		return fmt.Errorf("peer type must be NON_CLOUD, PRIV_CLOUD, or PUB_CLOUD")
	}
	bgp.PeerType = peerType

	medInStr, err := utils.ResourcePrompt("vxc", "Enter MED in (optional): ", noColor)
	if err != nil {
		return err
	}
	if medInStr != "" {
		medIn, err := strconv.Atoi(medInStr)
		if err != nil {
			return fmt.Errorf("MED in must be an integer")
		}
		bgp.MedIn = medIn
	}

	medOutStr, err := utils.ResourcePrompt("vxc", "Enter MED out (optional): ", noColor)
	if err != nil {
		return err
	}
	if medOutStr != "" {
		medOut, err := strconv.Atoi(medOutStr)
		if err != nil {
			return fmt.Errorf("MED out must be an integer")
		}
		bgp.MedOut = medOut
	}

	asPathPrependStr, err := utils.ResourcePrompt("vxc", "Enter AS path prepend count (0-10, optional): ", noColor)
	if err != nil {
		return err
	}
	if asPathPrependStr != "" {
		asPathPrepend, err := strconv.Atoi(asPathPrependStr)
		if err != nil || asPathPrepend < 0 || asPathPrepend > 10 {
			return fmt.Errorf("AS path prepend count must be between 0 and 10")
		}
		bgp.AsPathPrependCount = asPathPrepend
	}

	return nil
}

func promptBGPExportAddresses(bgp *megaport.BgpConnectionConfig, noColor bool) error {
	hasPermitExportTo, err := utils.ResourcePrompt("vxc", "Add permit export to addresses? (yes/no): ", noColor)
	if err != nil {
		return err
	}
	if strings.ToLower(hasPermitExportTo) == "yes" {
		for i := 0; i < 17; i++ {
			ipAddress, err := utils.ResourcePrompt("vxc", fmt.Sprintf("Enter IP address to permit export to (or empty to finish) [%d/17]: ", i+1), noColor)
			if err != nil {
				return err
			}
			if ipAddress == "" {
				break
			}
			bgp.PermitExportTo = append(bgp.PermitExportTo, ipAddress)
		}
	}

	hasDenyExportTo, err := utils.ResourcePrompt("vxc", "Add deny export to addresses? (yes/no): ", noColor)
	if err != nil {
		return err
	}
	if strings.ToLower(hasDenyExportTo) == "yes" {
		for i := 0; i < 17; i++ {
			ipAddress, err := utils.ResourcePrompt("vxc", fmt.Sprintf("Enter IP address to deny export to (or empty to finish) [%d/17]: ", i+1), noColor)
			if err != nil {
				return err
			}
			if ipAddress == "" {
				break
			}
			bgp.DenyExportTo = append(bgp.DenyExportTo, ipAddress)
		}
	}

	return nil
}

func promptBGPPrefixLists(bgp *megaport.BgpConnectionConfig, noColor bool) error {
	importWhitelistStr, err := utils.ResourcePrompt("vxc", "Enter import whitelist prefix list ID (optional): ", noColor)
	if err != nil {
		return err
	}
	if importWhitelistStr != "" {
		importWhitelist, err := strconv.Atoi(importWhitelistStr)
		if err != nil {
			return fmt.Errorf("import whitelist must be an integer")
		}
		bgp.ImportWhitelist = importWhitelist
	}

	importBlacklistStr, err := utils.ResourcePrompt("vxc", "Enter import blacklist prefix list ID (optional): ", noColor)
	if err != nil {
		return err
	}
	if importBlacklistStr != "" {
		importBlacklist, err := strconv.Atoi(importBlacklistStr)
		if err != nil {
			return fmt.Errorf("import blacklist must be an integer")
		}
		bgp.ImportBlacklist = importBlacklist
	}

	exportWhitelistStr, err := utils.ResourcePrompt("vxc", "Enter export whitelist prefix list ID (optional): ", noColor)
	if err != nil {
		return err
	}
	if exportWhitelistStr != "" {
		exportWhitelist, err := strconv.Atoi(exportWhitelistStr)
		if err != nil {
			return fmt.Errorf("export whitelist must be an integer")
		}
		bgp.ExportWhitelist = exportWhitelist
	}

	exportBlacklistStr, err := utils.ResourcePrompt("vxc", "Enter export blacklist prefix list ID (optional): ", noColor)
	if err != nil {
		return err
	}
	if exportBlacklistStr != "" {
		exportBlacklist, err := strconv.Atoi(exportBlacklistStr)
		if err != nil {
			return fmt.Errorf("export blacklist must be an integer")
		}
		bgp.ExportBlacklist = exportBlacklist
	}

	return nil
}

// promptIPsecTunnel collects the single IPsec tunnel carried by one ipSecTunnel
// interface. To configure multiple tunnels, add multiple ipSecTunnel interfaces.
func promptIPsecTunnel(noColor bool) (*megaport.IPsecTunnelConfig, error) {
	tunnel := &megaport.IPsecTunnelConfig{}

	sourceIP, err := utils.ResourcePrompt("vxc", "Enter source IP address (required): ", noColor)
	if err != nil {
		return nil, err
	}
	tunnel.SourceIpAddress = sourceIP

	destIP, err := utils.ResourcePrompt("vxc", "Enter destination IP address (required): ", noColor)
	if err != nil {
		return nil, err
	}
	tunnel.DestinationIpAddress = destIP

	psk, err := utils.SecretResourcePrompt("vxc", "Enter pre-shared key (required): ", noColor)
	if err != nil {
		return nil, err
	}
	tunnel.PreSharedKey = psk

	if err := promptIPsecTunnelOptional(tunnel, noColor); err != nil {
		return nil, err
	}

	return tunnel, nil
}

func promptIPsecTunnelOptional(tunnel *megaport.IPsecTunnelConfig, noColor bool) error {
	passiveStr, err := utils.ResourcePrompt("vxc", "Passive mode? (yes/no, optional - press Enter for API default): ", noColor)
	if err != nil {
		return err
	}
	if passiveStr != "" {
		switch strings.ToLower(passiveStr) {
		case "yes":
			v := true
			tunnel.Passive = &v
		case "no":
			v := false
			tunnel.Passive = &v
		default:
			return fmt.Errorf("passive mode must be 'yes' or 'no'")
		}
	}

	localID, err := utils.ResourcePrompt("vxc", "Enter local ID (optional): ", noColor)
	if err != nil {
		return err
	}
	tunnel.LocalId = localID

	remoteID, err := utils.ResourcePrompt("vxc", "Enter remote ID (optional): ", noColor)
	if err != nil {
		return err
	}
	tunnel.RemoteId = remoteID

	phase1Str, err := utils.ResourcePrompt("vxc", "Enter phase 1 lifetime in seconds (3600-604800, optional): ", noColor)
	if err != nil {
		return err
	}
	if phase1Str != "" {
		phase1, err := strconv.Atoi(phase1Str)
		if err != nil {
			return fmt.Errorf("phase 1 lifetime must be an integer")
		}
		tunnel.Phase1Lifetime = &phase1
	}

	phase2Str, err := utils.ResourcePrompt("vxc", "Enter phase 2 lifetime in seconds (600-86400, optional): ", noColor)
	if err != nil {
		return err
	}
	if phase2Str != "" {
		phase2, err := strconv.Atoi(phase2Str)
		if err != nil {
			return fmt.Errorf("phase 2 lifetime must be an integer")
		}
		tunnel.Phase2Lifetime = &phase2
	}

	return nil
}
