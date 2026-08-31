package validation

import (
	"fmt"
	"strings"
	"unicode/utf8"

	megaport "github.com/megaport/megaportgo"
)

const (
	// MinASPathPrependCount is the minimum allowed AS path prepend count in BGP configurations.
	MinASPathPrependCount = 0
	// MaxASPathPrependCount is the maximum allowed AS path prepend count in BGP configurations.
	MaxASPathPrependCount = 10
	// MinBFDInterval is the minimum Bidirectional Forwarding Detection interval in milliseconds.
	MinBFDInterval = 300
	// MaxBFDInterval is the maximum Bidirectional Forwarding Detection interval in milliseconds.
	MaxBFDInterval = 30000
	// MinBFDMultiplier is the minimum Bidirectional Forwarding Detection multiplier value.
	MinBFDMultiplier = 3
	// MaxBFDMultiplier is the maximum Bidirectional Forwarding Detection multiplier value.
	MaxBFDMultiplier = 20
	// MinIPsecPhase1Lifetime is the minimum IKE phase 1 (ISAKMP SA) lifetime in seconds.
	MinIPsecPhase1Lifetime = 3600
	// MaxIPsecPhase1Lifetime is the maximum IKE phase 1 (ISAKMP SA) lifetime in seconds.
	MaxIPsecPhase1Lifetime = 604800
	// MinIPsecPhase2Lifetime is the minimum IKE phase 2 (IPsec SA) lifetime in seconds.
	MinIPsecPhase2Lifetime = 600
	// MaxIPsecPhase2Lifetime is the maximum IKE phase 2 (IPsec SA) lifetime in seconds.
	MaxIPsecPhase2Lifetime = 86400
	// MinMED is the minimum Multi-Exit Discriminator value for BGP routing.
	MinMED = 0
	// MaxMED is the maximum Multi-Exit Discriminator value for BGP routing.
	// Using int64 to avoid overflow on 32-bit platforms.
	MaxMED int64 = 4294967295
	// BGPPeerNonCloud identifies a non-cloud BGP peer type.
	BGPPeerNonCloud = "NON_CLOUD"
	// BGPPeerPrivCloud identifies a private cloud BGP peer type.
	BGPPeerPrivCloud = "PRIV_CLOUD"
	// BGPPeerPubCloud identifies a public cloud BGP peer type.
	BGPPeerPubCloud = "PUB_CLOUD"
	// BGPExportPolicyPermit defines the permit policy for BGP route exports.
	BGPExportPolicyPermit = "permit"
	// BGPExportPolicyDeny defines the deny policy for BGP route exports.
	BGPExportPolicyDeny = "deny"
	// MaxIBMNameLength is the maximum allowed length of an IBM connection name.
	MaxIBMNameLength = 100
	// IBMAccountIDLength is the required length of an IBM account ID.
	IBMAccountIDLength = 32
	// AWSConnectTypeAWS denotes a standard AWS connection type.
	AWSConnectTypeAWS = "AWS"
	// AWSConnectTypeAWSHC denotes a high-capacity AWS connection type.
	AWSConnectTypeAWSHC = "AWSHC"
	// AWSConnectTypeTransit denotes a transit AWS connection type.
	AWSConnectTypeTransit = "transit"
	// AWSConnectTypePrivate denotes a private AWS connection type.
	AWSConnectTypePrivate = "private"
	// AWSConnectTypePublic denotes a public AWS connection type.
	AWSConnectTypePublic = "public"
)

// ValidateVXCEndVLAN validates the VLAN ID for a VXC (Virtual Cross Connect) endpoint.
// This ensures the VLAN ID meets the Megaport requirements for VXC configurations.
//
// Parameters:
//   - vlan: The VLAN ID to validate (typically 0-4094)
//
// Validation checks:
//   - The VLAN must be one of the following:
//   - AutoAssignVLAN (0): System will auto-assign a VLAN
//   - UntaggedVLAN (-1): Packet will be untagged
//   - A value between MinAssignableVLAN (2) and MaxVLAN (4094) inclusive
//
// Returns:
//   - A ValidationError if the VLAN ID is not valid
//   - nil if the validation passes
func ValidateVXCEndVLAN(vlan int) error {
	return ValidateVLAN(vlan)
}

// ValidateVXCEndInnerVLAN validates the inner VLAN ID (Q-in-Q) for a VXC endpoint.
// This function ensures the inner VLAN ID meets the requirements for QinQ configurations.
//
// Parameters:
//   - vlan: The inner VLAN ID to validate (typically 0-4094)
//
// Validation checks:
//   - Inner VLAN follows the same validation rules as outer VLANs
//   - Must be a valid VLAN ID (0, -1, or 2-4094)
//
// Returns:
//   - A ValidationError if the inner VLAN ID is not valid
//   - nil if the validation passes
func ValidateVXCEndInnerVLAN(vlan int) error {
	return ValidateVLAN(vlan)
}

// ValidateVNICIndex validates a vNIC index for a VXC endpoint.
// The index must be non-negative.
func ValidateVNICIndex(index int) error {
	if index < 0 {
		return NewValidationError("vNIC index", index, "must be non-negative")
	}
	return nil
}

// ValidateVXCRequest validates a VXC (Virtual Cross Connect) request.
// This function ensures all required parameters for creating a VXC are present and valid.
//
// Parameters:
//   - req: The BuyVXCRequest containing all VXC configuration parameters
//
// Validation checks include:
//   - VXC name cannot be empty
//   - Contract term must be valid (1, 12, 24, 36, 48, or 60 months)
//   - Rate limit must be a positive value
//   - Port UID (A-End) cannot be empty
//   - B-End UID cannot be empty when no partner configuration is provided
//   - Partner configurations (if present) must be valid for their respective types
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateVXCRequest(req *megaport.BuyVXCRequest) error {
	if req.VXCName == "" {
		return NewValidationError("VXC name", req.VXCName, "cannot be empty")
	}
	if err := ValidateContractTerm(req.Term); err != nil {
		return err
	}
	if err := ValidateRateLimit(req.RateLimit); err != nil {
		return err
	}
	if req.PortUID == "" {
		return NewValidationError("A-End UID (PortUID)", req.PortUID, "cannot be empty")
	}

	// Check if B-End has a partner config
	hasPartnerConfig := req.BEndConfiguration.PartnerConfig != nil

	// Check B-End UID (ProductUID in the BEndConfiguration)
	if req.BEndConfiguration.ProductUID == "" && !hasPartnerConfig {
		return NewValidationError("B-End UID", req.BEndConfiguration.ProductUID, "cannot be empty when no partner configuration is provided")
	}

	// Validate A-End partner configuration if present
	if req.AEndConfiguration.PartnerConfig != nil {
		if err := ValidateVXCPartnerConfig(req.AEndConfiguration.PartnerConfig); err != nil {
			return err
		}
	}

	// Validate B-End partner configuration if present
	if req.BEndConfiguration.PartnerConfig != nil {
		if err := ValidateVXCPartnerConfig(req.BEndConfiguration.PartnerConfig); err != nil {
			return err
		}
	}

	// Validate VLANs if specified
	if req.AEndConfiguration.VLAN != 0 {
		if err := ValidateVXCEndVLAN(req.AEndConfiguration.VLAN); err != nil {
			return NewValidationError("A-End VLAN", req.AEndConfiguration.VLAN, err.Error())
		}
	}

	if req.BEndConfiguration.VLAN != 0 {
		if err := ValidateVXCEndVLAN(req.BEndConfiguration.VLAN); err != nil {
			return NewValidationError("B-End VLAN", req.BEndConfiguration.VLAN, err.Error())
		}
	}

	// Validate inner VLANs if specified
	if req.AEndConfiguration.VXCOrderMVEConfig != nil && req.AEndConfiguration.InnerVLAN != 0 {
		if err := ValidateVXCEndInnerVLAN(req.AEndConfiguration.InnerVLAN); err != nil {
			return NewValidationError("A-End Inner VLAN", req.AEndConfiguration.InnerVLAN, err.Error())
		}
	}

	if req.BEndConfiguration.VXCOrderMVEConfig != nil && req.BEndConfiguration.InnerVLAN != 0 {
		if err := ValidateVXCEndInnerVLAN(req.BEndConfiguration.InnerVLAN); err != nil {
			return NewValidationError("B-End Inner VLAN", req.BEndConfiguration.InnerVLAN, err.Error())
		}
	}

	return nil
}

// ValidateAWSPartnerConfig validates an AWS partner configuration for a VXC connection.
// This function ensures the AWS-specific connection parameters meet all requirements.
//
// Parameters:
//   - config: The AWS partner configuration to validate
//
// Validation checks include:
//   - Configuration cannot be nil
//   - Connect type must be provided and be one of the valid types ('AWS', 'AWSHC', 'private', 'public')
//   - Owner account must be provided (AWS account ID)
//   - ASN must be provided and within the valid range (1-4294967295)
//   - If customer IP address is provided, it must be in valid IPv4 CIDR notation
//   - If Amazon IP address is provided, it must be in valid IPv4 CIDR notation
//   - Connection name is optional (the API defaults it to "MEGAPORT"); when
//     provided it must not exceed 255 characters
//   - For 'AWS' connect type with a specified connection type, it must be 'private' or 'public'
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateAWSPartnerConfig(config *megaport.VXCPartnerConfigAWS) error {
	if config == nil {
		return NewValidationError("AWS partner config", nil, "cannot be nil")
	}
	if config.ConnectType == "" {
		return NewValidationError("AWS connect type", config.ConnectType, "cannot be empty")
	}
	validTypes := []string{"AWS", "AWSHC", "private", "public"}
	isValidType := false
	for _, t := range validTypes {
		if config.ConnectType == t {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return NewValidationError("AWS connect type", config.ConnectType, "must be 'AWS', 'AWSHC', 'private', or 'public'")
	}

	if config.OwnerAccount == "" {
		return NewValidationError("AWS owner account", config.OwnerAccount, "cannot be empty")
	}

	if config.ConnectionName != "" && utf8.RuneCountInString(config.ConnectionName) > MaxAWSConnectionNameLength {
		return NewValidationError("AWS connection name", config.ConnectionName, fmt.Sprintf("cannot exceed %d characters", MaxAWSConnectionNameLength))
	}
	if config.CustomerIPAddress != "" {
		if err := ValidateCIDR(config.CustomerIPAddress, "AWS customer IP address"); err != nil {
			return err
		}
	}
	if config.AmazonIPAddress != "" {
		if err := ValidateCIDR(config.AmazonIPAddress, "AWS Amazon IP address"); err != nil {
			return err
		}
	}

	if config.ConnectType == "AWS" && config.Type != "" && config.Type != "private" && config.Type != "public" {
		return NewValidationError("AWS type", config.Type, "must be 'private' or 'public' for AWS connect type")
	}
	if config.ASN == 0 {
		return NewValidationError("ASN", config.ASN, "cannot be empty")
	}
	// Delegate to ValidateASN for the platform-aware bound; re-wrap to keep field name and message format.
	if err := ValidateASN(config.ASN); err != nil {
		return NewValidationError("ASN", config.ASN, fmt.Sprintf("must be between %d-%d", MinASN, maxSupportedASNForInt()))
	}
	return nil
}

// ValidateAzurePartnerConfig validates an Azure partner configuration for a VXC connection.
// This function ensures the Azure-specific connection parameters meet all requirements.
//
// Parameters:
//   - config: The Azure partner configuration to validate
//
// Validation checks include:
//   - Configuration cannot be nil
//   - Service key must be provided (required for Azure connections)
//   - For each peer, at least one of primary_subnet or secondary_subnet must be provided
//   - For each peer, VLAN must be provided and valid
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateAzurePartnerConfig(config *megaport.VXCPartnerConfigAzure) error {
	if config == nil {
		return NewValidationError("Azure partner config", nil, "cannot be nil")
	}
	if config.ServiceKey == "" {
		return NewValidationError("Azure service key", config.ServiceKey, "cannot be empty")
	}

	// Validate each peer configuration
	if len(config.Peers) > 0 {
		for i, peer := range config.Peers {
			if peer.PrimarySubnet == "" && peer.SecondarySubnet == "" {
				return NewValidationError(fmt.Sprintf("Azure peer [%d] subnet", i), nil,
					"at least one of primary_subnet or secondary_subnet must be provided")
			}

			// Validate VLAN
			if err := ValidateVLAN(peer.VLAN); err != nil {
				return NewValidationError(fmt.Sprintf("Azure peer [%d] VLAN", i), peer.VLAN,
					fmt.Sprintf("must be valid (0 for auto-assign, -1 for untagged, or %d-%d except %d)",
						MinAssignableVLAN, MaxVLAN, ReservedVLAN))
			}
		}
	}

	return nil
}

// ValidateGooglePartnerConfig validates a Google Cloud partner configuration for a VXC connection.
// This function ensures the Google-specific connection parameters meet all requirements.
//
// Parameters:
//   - config: The Google partner configuration to validate
//
// Validation checks include:
//   - Configuration cannot be nil
//   - Pairing key must be provided (required for Google Cloud connections)
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateGooglePartnerConfig(config *megaport.VXCPartnerConfigGoogle) error {
	if config == nil {
		return NewValidationError("Google partner config", nil, "cannot be nil")
	}
	if config.PairingKey == "" {
		return NewValidationError("Google pairing key", config.PairingKey, "cannot be empty")
	}
	return nil
}

// ValidateOraclePartnerConfig validates an Oracle Cloud partner configuration for a VXC connection.
// This function ensures the Oracle-specific connection parameters meet all requirements.
//
// Parameters:
//   - config: The Oracle partner configuration to validate
//
// Validation checks include:
//   - Configuration cannot be nil
//   - Virtual Circuit ID must be provided (required for Oracle Cloud connections)
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateOraclePartnerConfig(config *megaport.VXCPartnerConfigOracle) error {
	if config == nil {
		return NewValidationError("Oracle partner config", nil, "cannot be nil")
	}
	if config.VirtualCircuitId == "" {
		return NewValidationError("Oracle virtual circuit ID", config.VirtualCircuitId, "cannot be empty")
	}
	return nil
}

// ValidateIBMPartnerConfig validates an IBM Cloud partner configuration for a VXC connection.
// This function ensures the IBM-specific connection parameters meet all requirements.
//
// Parameters:
//   - config: The IBM partner configuration to validate
//
// Validation checks include:
//   - Configuration cannot be nil
//   - Account ID must be provided
//   - Account ID must be exactly 32 characters (IBMAccountIDLength)
//   - Account ID must contain only hexadecimal characters (0-9, a-f, A-F)
//   - Connection name is optional (the API defaults it to "MEGAPORT"); when
//     provided it must not exceed the maximum length (MaxIBMNameLength) and
//     must contain only allowed characters (0-9, a-z, A-Z, /, -, _, ,)
//   - If customer IP address is provided, it must be in valid IPv4 CIDR notation
//   - If provider IP address is provided, it must be in valid IPv4 CIDR notation
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateIBMPartnerConfig(config *megaport.VXCPartnerConfigIBM) error {
	if config == nil {
		return NewValidationError("IBM partner config", nil, "cannot be nil")
	}
	if config.AccountID == "" {
		return NewValidationError("IBM account ID", config.AccountID, "cannot be empty")
	}
	if len(config.AccountID) != IBMAccountIDLength {
		return NewValidationError("IBM account ID", config.AccountID, fmt.Sprintf("must be exactly %d characters", IBMAccountIDLength))
	}
	for _, c := range config.AccountID {
		if c < '0' || (c > '9' && c < 'A') || (c > 'F' && c < 'a') || c > 'f' {
			return NewValidationError("IBM account ID", config.AccountID, "must contain only hexadecimal characters (0-9, a-f, A-F)")
		}
	}
	if config.Name != "" {
		if utf8.RuneCountInString(config.Name) > MaxIBMNameLength {
			return NewValidationError("IBM connection name", config.Name, fmt.Sprintf("cannot exceed %d characters", MaxIBMNameLength))
		}
		if !isValidIBMName(config.Name) {
			return NewValidationError("IBM connection name", config.Name, "must only contain characters 0-9, a-z, A-Z, /, -, _, or ,")
		}
	}
	if config.CustomerIPAddress != "" {
		if err := ValidateCIDR(config.CustomerIPAddress, "IBM customer IP address"); err != nil {
			return err
		}
	}
	if config.ProviderIPAddress != "" {
		if err := ValidateCIDR(config.ProviderIPAddress, "IBM provider IP address"); err != nil {
			return err
		}
	}
	return nil
}

func isValidIBMName(name string) bool {
	for _, c := range name {
		if (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '/' || c == '-' || c == '_' || c == ',' {
			continue
		}
		return false
	}
	return true
}

// ValidateTransitPartnerConfig validates a Transit partner configuration for a
// VXC connection. Transit carries no partner-specific fields beyond the
// connect type, so this only guards against a nil config or a connect type
// that doesn't match "TRANSIT".
func ValidateTransitPartnerConfig(config *megaport.VXCPartnerConfigTransit) error {
	if config == nil {
		return NewValidationError("Transit partner config", nil, "cannot be nil")
	}
	if config.ConnectType != "TRANSIT" {
		return NewValidationError("Transit connect type", config.ConnectType, "must be 'TRANSIT'")
	}
	return nil
}

// ValidateVrouterPartnerConfig validates a vRouter partner configuration: it
// requires at least one interface and validates each interface's VLAN, IP
// addresses, NAT IPs, routes, BFD, BGP connections, interface type, and IPsec
// tunnel. An ipSecTunnel interface carries exactly one tunnel, and a tunnel may
// only appear on an ipSecTunnel interface.
func ValidateVrouterPartnerConfig(config *megaport.VXCOrderVrouterPartnerConfig) error {
	if config == nil {
		return NewValidationError("vRouter partner config", nil, "cannot be nil")
	}
	if len(config.Interfaces) == 0 {
		return NewValidationError("vRouter interfaces", nil, "at least one interface must be provided")
	}
	for i, iface := range config.Interfaces {
		if err := ValidateVLAN(iface.VLAN); err != nil {
			return NewValidationError(fmt.Sprintf("vRouter interface [%d] VLAN", i), iface.VLAN,
				fmt.Sprintf("must be valid (%d for auto-assign, %d for untagged, or %d-%d except %d)",
					AutoAssignVLAN, UntaggedVLAN, MinAssignableVLAN, MaxVLAN, ReservedVLAN))
		}
		if len(iface.IpAddresses) > 0 {
			for j, ip := range iface.IpAddresses {
				if err := ValidateCIDR(ip, fmt.Sprintf("vRouter interface [%d] IP address [%d]", i, j)); err != nil {
					return err
				}
			}
		}
		if len(iface.NatIpAddresses) > 0 {
			for j, ip := range iface.NatIpAddresses {
				if err := ValidateCIDR(ip, fmt.Sprintf("vRouter interface [%d] NAT IP address [%d]", i, j)); err != nil {
					return err
				}
			}
		}
		if len(iface.IpRoutes) > 0 {
			for j, route := range iface.IpRoutes {
				if err := ValidateIPRouteConfig(route, i, j); err != nil {
					return err
				}
			}
		}
		if err := ValidateBFDConfig(iface.Bfd, i); err != nil {
			return err
		}
		if len(iface.BgpConnections) > 0 {
			for j, conn := range iface.BgpConnections {
				if err := ValidateBGPConnectionConfig(conn, i, j); err != nil {
					return err
				}
			}
		}
		if iface.InterfaceType != "" && iface.InterfaceType != megaport.InterfaceTypeSubInterface && iface.InterfaceType != megaport.InterfaceTypeIPSecTunnel {
			return NewValidationError(fmt.Sprintf("vRouter interface [%d] interface type", i), iface.InterfaceType, fmt.Sprintf("must be '%s' or '%s'", megaport.InterfaceTypeSubInterface, megaport.InterfaceTypeIPSecTunnel))
		}
		if iface.InterfaceType == megaport.InterfaceTypeIPSecTunnel && iface.IpSecTunnelOptions == nil {
			return NewValidationError(fmt.Sprintf("vRouter interface [%d] ipSecTunnelOptions", i), nil, fmt.Sprintf("a tunnel is required when interface type is '%s'", megaport.InterfaceTypeIPSecTunnel))
		}
		if iface.IpSecTunnelOptions != nil {
			if iface.InterfaceType != megaport.InterfaceTypeIPSecTunnel {
				return NewValidationError(fmt.Sprintf("vRouter interface [%d] ipSecTunnelOptions", i), iface.InterfaceType, fmt.Sprintf("requires interface type '%s'", megaport.InterfaceTypeIPSecTunnel))
			}
			if err := ValidateIPsecTunnelConfig(*iface.IpSecTunnelOptions, i); err != nil {
				return err
			}
		}
	}
	return nil
}

// ValidateVXCPartnerConfig validates a partner configuration for a VXC connection.
// This function uses type switching to determine the specific type of partner configuration
// and delegates to the appropriate type-specific validation function.
//
// Parameters:
//   - config: The partner configuration to validate (can be any of the supported partner types)
//
// Validation checks include:
//   - Type-specific validations delegated to:
//   - ValidateAWSPartnerConfig
//   - ValidateAzurePartnerConfig
//   - ValidateGooglePartnerConfig
//   - ValidateOraclePartnerConfig
//   - ValidateIBMPartnerConfig
//   - ValidateVrouterPartnerConfig
//   - ValidateTransitPartnerConfig
//   - Configuration type must be one of the supported types
//
// Returns:
//   - A ValidationError if the type is not supported or if type-specific validation fails
//   - nil if all validation checks pass
func ValidateVXCPartnerConfig(config megaport.VXCPartnerConfiguration) error {
	switch v := config.(type) {
	case *megaport.VXCPartnerConfigAWS:
		return ValidateAWSPartnerConfig(v)
	case *megaport.VXCPartnerConfigAzure:
		return ValidateAzurePartnerConfig(v)
	case *megaport.VXCPartnerConfigGoogle:
		return ValidateGooglePartnerConfig(v)
	case *megaport.VXCPartnerConfigOracle:
		return ValidateOraclePartnerConfig(v)
	case *megaport.VXCPartnerConfigIBM:
		return ValidateIBMPartnerConfig(v)
	case *megaport.VXCPartnerConfigTransit:
		return ValidateTransitPartnerConfig(v)
	case *megaport.VXCOrderVrouterPartnerConfig:
		return ValidateVrouterPartnerConfig(v)
	default:
		return NewValidationError("Partner configuration type", fmt.Sprintf("%T", v), "is not supported")
	}
}

// ValidateBGPConnectionConfig validates the configuration for a BGP (Border Gateway Protocol) connection.
// This function performs comprehensive validation of all BGP connection parameters to ensure they meet
// the requirements for establishing BGP peering sessions in a vRouter interface.
//
// Parameters:
//   - conn: The BGP connection configuration to validate
//   - ifaceIndex: The index of the interface this BGP connection belongs to (used for error messages)
//   - connIndex: The index of this BGP connection within the interface (used for error messages)
//
// Validation checks include:
//   - Peer ASN must be provided and within the valid range (1-4294967295)
//   - Local IP address must be provided and be a valid IPv4 address or IPv4 CIDR
//   - Peer IP address must be provided and be a valid IPv4 address or IPv4 CIDR
//   - If Peer Type is provided, it must be one of the predefined values (NON_CLOUD, PRIV_CLOUD, PUB_CLOUD)
//   - If MED values (Multi-Exit Discriminator) are provided, they must be within allowed range (0-4294967295)
//   - If AS path prepend count is provided, it must be within allowed range (0-10)
//   - If Export Policy is provided, it must be either "permit" or "deny"
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateBGPConnectionConfig(conn megaport.BgpConnectionConfig, ifaceIndex, connIndex int) error {
	fieldPrefix := fmt.Sprintf("vRouter interface [%d] BGP connection [%d]", ifaceIndex, connIndex)
	if conn.PeerAsn == 0 {
		return NewValidationError(fmt.Sprintf("%s peer ASN", fieldPrefix), nil, "is required")
	}
	// Delegate to ValidateASN for the platform-aware bound; re-wrap to keep field name and message format.
	if err := ValidateASN(conn.PeerAsn); err != nil {
		return NewValidationError(fmt.Sprintf("%s peer ASN", fieldPrefix), conn.PeerAsn, fmt.Sprintf("must be between %d-%d", MinASN, maxSupportedASNForInt()))
	}
	if conn.LocalIpAddress == "" {
		return NewValidationError(fmt.Sprintf("%s local IP address", fieldPrefix), conn.LocalIpAddress, "cannot be empty")
	}
	if strings.Contains(conn.LocalIpAddress, "/") {
		if err := ValidateCIDR(conn.LocalIpAddress, fmt.Sprintf("%s local IP address", fieldPrefix)); err != nil {
			return err
		}
	} else {
		if err := ValidateIPv4(conn.LocalIpAddress, fmt.Sprintf("%s local IP address", fieldPrefix)); err != nil {
			return err
		}
	}
	if conn.PeerIpAddress == "" {
		return NewValidationError(fmt.Sprintf("%s peer IP address", fieldPrefix), conn.PeerIpAddress, "cannot be empty")
	}
	if strings.Contains(conn.PeerIpAddress, "/") {
		if err := ValidateCIDR(conn.PeerIpAddress, fmt.Sprintf("%s peer IP address", fieldPrefix)); err != nil {
			return err
		}
	} else {
		if err := ValidateIPv4(conn.PeerIpAddress, fmt.Sprintf("%s peer IP address", fieldPrefix)); err != nil {
			return err
		}
	}
	if conn.PeerType != "" {
		validTypes := []string{BGPPeerNonCloud, BGPPeerPrivCloud, BGPPeerPubCloud}
		isValid := false
		for _, vt := range validTypes {
			if conn.PeerType == vt {
				isValid = true
				break
			}
		}
		if !isValid {
			return NewValidationError(fmt.Sprintf("%s peer type", fieldPrefix), conn.PeerType, fmt.Sprintf("must be one of '%s', '%s', or '%s'", BGPPeerNonCloud, BGPPeerPrivCloud, BGPPeerPubCloud))
		}
	}
	if conn.MedIn != 0 {
		// Convert to int64 for comparison with MaxMED
		medIn := int64(conn.MedIn)
		if medIn < MinMED || medIn > MaxMED {
			return NewValidationError(fmt.Sprintf("%s MED in", fieldPrefix), conn.MedIn, fmt.Sprintf("must be between %d-%s", MinMED, "4294967295"))
		}
	}
	if conn.MedOut != 0 {
		// Convert to int64 for comparison with MaxMED
		medOut := int64(conn.MedOut)
		if medOut < MinMED || medOut > MaxMED {
			return NewValidationError(fmt.Sprintf("%s MED out", fieldPrefix), conn.MedOut, fmt.Sprintf("must be between %d-%s", MinMED, "4294967295"))
		}
	}
	if conn.AsPathPrependCount != 0 {
		if conn.AsPathPrependCount < MinASPathPrependCount || conn.AsPathPrependCount > MaxASPathPrependCount {
			return NewValidationError(fmt.Sprintf("%s AS path prepend count", fieldPrefix), conn.AsPathPrependCount, fmt.Sprintf("must be between %d-%d", MinASPathPrependCount, MaxASPathPrependCount))
		}
	}
	if conn.ExportPolicy != "" {
		if conn.ExportPolicy != BGPExportPolicyPermit && conn.ExportPolicy != BGPExportPolicyDeny {
			return NewValidationError(fmt.Sprintf("%s export policy", fieldPrefix), conn.ExportPolicy, "must be 'permit' or 'deny'")
		}
	}
	return nil
}

// ValidateIPRouteConfig validates an IP route configuration for a vRouter interface.
// This function ensures that IP routes are correctly formatted according to networking requirements.
//
// Parameters:
//   - route: The IP route configuration to validate (contains prefix and next hop)
//   - ifaceIndex: The index of the interface this route belongs to (used for error messages)
//   - routeIndex: The index of this route within the interface (used for error messages)
//
// Validation checks include:
//   - Prefix must be provided and be a valid IPv4 CIDR notation (e.g., "10.0.0.0/24")
//   - Next hop must be provided and be a valid IPv4 address (not a CIDR)
//   - Next hop must be in the standard IPv4 format (e.g., "192.168.1.1")
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateIPRouteConfig(route megaport.IpRoute, ifaceIndex, routeIndex int) error {
	if route.Prefix == "" {
		return NewValidationError(fmt.Sprintf("vRouter interface [%d] IP route [%d] prefix", ifaceIndex, routeIndex), route.Prefix, "cannot be empty")
	}
	if err := ValidateCIDR(route.Prefix, fmt.Sprintf("vRouter interface [%d] IP route [%d] prefix", ifaceIndex, routeIndex)); err != nil {
		return err
	}
	if route.NextHop == "" {
		return NewValidationError(fmt.Sprintf("vRouter interface [%d] IP route [%d] next hop", ifaceIndex, routeIndex), route.NextHop, "cannot be empty")
	}
	if strings.Contains(route.NextHop, "/") {
		return NewValidationError(fmt.Sprintf("vRouter interface [%d] IP route [%d] next hop", ifaceIndex, routeIndex), route.NextHop, "must be a valid IPv4 address (not CIDR)")
	}
	if err := ValidateIPv4(route.NextHop, fmt.Sprintf("vRouter interface [%d] IP route [%d] next hop", ifaceIndex, routeIndex)); err != nil {
		return err
	}
	return nil
}

// ValidateBFDConfig validates a Bidirectional Forwarding Detection (BFD) configuration.
// BFD is a network protocol used to detect link failures between adjacent forwarding engines.
// This function ensures that BFD parameters are within acceptable ranges for stable operation.
//
// Parameters:
//   - bfd: The BFD configuration to validate, containing interval and multiplier settings
//   - ifaceIndex: The index of the interface this BFD configuration belongs to (used for error messages)
//
// Validation checks include:
//   - TX interval (transmission interval) must be within allowed range (300-30000 milliseconds)
//   - RX interval (receive interval) must be within allowed range (300-30000 milliseconds)
//   - Multiplier must be within allowed range (3-20)
//   - Zero values are allowed and considered as "not specified"
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateBFDConfig(bfd megaport.BfdConfig, ifaceIndex int) error {
	if bfd.TxInterval != 0 {
		if bfd.TxInterval < MinBFDInterval || bfd.TxInterval > MaxBFDInterval {
			return NewValidationError(fmt.Sprintf("vRouter interface [%d] BFD TX interval", ifaceIndex), bfd.TxInterval, fmt.Sprintf("must be between %d-%d milliseconds", MinBFDInterval, MaxBFDInterval))
		}
	}
	if bfd.RxInterval != 0 {
		if bfd.RxInterval < MinBFDInterval || bfd.RxInterval > MaxBFDInterval {
			return NewValidationError(fmt.Sprintf("vRouter interface [%d] BFD RX interval", ifaceIndex), bfd.RxInterval, fmt.Sprintf("must be between %d-%d milliseconds", MinBFDInterval, MaxBFDInterval))
		}
	}
	if bfd.Multiplier != 0 {
		if bfd.Multiplier < MinBFDMultiplier || bfd.Multiplier > MaxBFDMultiplier {
			return NewValidationError(fmt.Sprintf("vRouter interface [%d] BFD multiplier", ifaceIndex), bfd.Multiplier, fmt.Sprintf("must be between %d-%d", MinBFDMultiplier, MaxBFDMultiplier))
		}
	}
	return nil
}

// ValidateIPsecTunnelConfig validates a single IPsec tunnel: source IP,
// destination IP, and PSK are required; lifetimes are optional but range-checked,
// and phase 2 must be shorter than phase 1. The PSK is never echoed in an error.
func ValidateIPsecTunnelConfig(tunnel megaport.IPsecTunnelConfig, ifaceIndex int) error {
	fieldPrefix := fmt.Sprintf("vRouter interface [%d] IPsec tunnel", ifaceIndex)

	if tunnel.SourceIpAddress == "" {
		return NewValidationError(fmt.Sprintf("%s source IP address", fieldPrefix), tunnel.SourceIpAddress, "cannot be empty")
	}
	if err := ValidateIPv4(tunnel.SourceIpAddress, fmt.Sprintf("%s source IP address", fieldPrefix)); err != nil {
		return err
	}
	if tunnel.DestinationIpAddress == "" {
		return NewValidationError(fmt.Sprintf("%s destination IP address", fieldPrefix), tunnel.DestinationIpAddress, "cannot be empty")
	}
	if err := ValidateIPv4(tunnel.DestinationIpAddress, fmt.Sprintf("%s destination IP address", fieldPrefix)); err != nil {
		return err
	}
	if tunnel.PreSharedKey == "" {
		// Never echo the PSK: pass nil as the offending value.
		return NewValidationError(fmt.Sprintf("%s pre-shared key", fieldPrefix), nil, "cannot be empty")
	}
	if tunnel.Phase1Lifetime != nil {
		if *tunnel.Phase1Lifetime < MinIPsecPhase1Lifetime || *tunnel.Phase1Lifetime > MaxIPsecPhase1Lifetime {
			return NewValidationError(fmt.Sprintf("%s phase 1 lifetime", fieldPrefix), *tunnel.Phase1Lifetime, fmt.Sprintf("must be between %d-%d seconds", MinIPsecPhase1Lifetime, MaxIPsecPhase1Lifetime))
		}
	}
	if tunnel.Phase2Lifetime != nil {
		if *tunnel.Phase2Lifetime < MinIPsecPhase2Lifetime || *tunnel.Phase2Lifetime > MaxIPsecPhase2Lifetime {
			return NewValidationError(fmt.Sprintf("%s phase 2 lifetime", fieldPrefix), *tunnel.Phase2Lifetime, fmt.Sprintf("must be between %d-%d seconds", MinIPsecPhase2Lifetime, MaxIPsecPhase2Lifetime))
		}
	}
	// Only cross-check when both are explicitly set. When one is omitted the API
	// applies its default (phase1=28800s, phase2=3600s). A value like phase2=40000
	// with no phase1 passes here but violates the constraint at the API level —
	// this is a known gap that requires knowledge of API defaults to close properly.
	if tunnel.Phase1Lifetime != nil && tunnel.Phase2Lifetime != nil && *tunnel.Phase2Lifetime >= *tunnel.Phase1Lifetime {
		return NewValidationError(fmt.Sprintf("%s phase 2 lifetime", fieldPrefix), *tunnel.Phase2Lifetime, "must be less than phase 1 lifetime")
	}
	return nil
}
