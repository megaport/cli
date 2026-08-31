package vxc

import (
	"context"
	"fmt"
	"strings"

	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/megaport/megaport-cli/internal/validation"
	megaport "github.com/megaport/megaportgo"
)

func promptPartnerConfig(end string, ctx context.Context, svc megaport.VXCService, noColor bool) (megaport.VXCPartnerConfiguration, string, error) {
	partner, err := utils.ResourcePrompt("vxc", fmt.Sprintf("Enter %s partner (AWS, Azure, Google, Oracle, IBM, VRouter, Transit) (optional): ", end), noColor)
	if err != nil {
		return nil, "", err
	}
	if partner == "" {
		return nil, "", nil
	}

	switch strings.ToLower(partner) {
	case "aws":
		awsPartner, err := promptAWSConfig(noColor)
		if err != nil {
			return nil, "", err
		}
		partnerPortUID, err := utils.ResourcePrompt("vxc", "Enter AWS Partner Port product UID (required): ", noColor)
		if err != nil {
			return nil, "", err
		}
		if partnerPortUID == "" {
			return nil, "", fmt.Errorf("AWS Partner Port product UID is required")
		}
		return awsPartner, partnerPortUID, nil
	case "azure":
		azurePartner, uid, err := promptAzureConfig(ctx, svc, noColor)
		if err != nil {
			return nil, "", err
		}
		return azurePartner, uid, nil
	case "google":
		googlePartner, uid, err := promptGoogleConfig(ctx, svc, noColor)
		if err != nil {
			return nil, "", err
		}
		return googlePartner, uid, nil
	case "oracle":
		oraclePartner, uid, err := promptOracleConfig(ctx, svc, noColor)
		if err != nil {
			return nil, "", err
		}
		return oraclePartner, uid, nil
	case "ibm":
		ibmPartner, err := promptIBMConfig(noColor)
		if err != nil {
			return nil, "", err
		}
		partnerPortUID, err := utils.ResourcePrompt("vxc", "Enter IBM Partner Port product UID (required): ", noColor)
		if err != nil {
			return nil, "", err
		}
		if partnerPortUID == "" {
			return nil, "", fmt.Errorf("IBM Partner Port product UID is required")
		}
		return ibmPartner, partnerPortUID, nil
	case "vrouter":
		vrouterPartner, err := promptVRouterConfig(noColor)
		if err != nil {
			return nil, "", err
		}
		return vrouterPartner, "", nil
	case "transit":
		return promptTransitConfig(), "", nil
	default:
		return nil, "", fmt.Errorf("unsupported partner: %s", partner)
	}
}

func promptTransitConfig() *megaport.VXCPartnerConfigTransit {
	return &megaport.VXCPartnerConfigTransit{
		ConnectType: "TRANSIT",
	}
}

func promptAWSConfig(noColor bool) (*megaport.VXCPartnerConfigAWS, error) {
	connectType, err := utils.ResourcePrompt("vxc", "Enter connect type (required - either AWS or AWSHC): ", noColor)
	if err != nil {
		return nil, err
	}

	if connectType != "AWS" && connectType != "AWSHC" {
		return nil, fmt.Errorf("connect type must be AWS or AWSHC")
	}

	ownerAccount, err := utils.ResourcePrompt("vxc", "Enter owner account ID (required): ", noColor)
	if err != nil {
		return nil, err
	}
	if ownerAccount == "" {
		return nil, fmt.Errorf("owner account ID is required")
	}

	connectionName, err := utils.ResourcePrompt("vxc", "Enter connection name (optional, defaults to MEGAPORT): ", noColor)
	if err != nil {
		return nil, err
	}

	asnStr, err := utils.ResourcePrompt("vxc", "Enter ASN (required): ", noColor)
	if err != nil {
		return nil, err
	}
	if asnStr == "" {
		return nil, fmt.Errorf("ASN is required")
	}
	asn, err := validation.ParseInt("ASN", asnStr)
	if err != nil {
		return nil, err
	}

	amazonASNStr, err := utils.ResourcePrompt("vxc", "Enter Amazon ASN (optional): ", noColor)
	if err != nil {
		return nil, err
	}
	var amazonASN int
	if amazonASNStr != "" {
		amazonASN, err = validation.ParseInt("Amazon ASN", amazonASNStr)
		if err != nil {
			return nil, err
		}
	}

	authKey, err := utils.ResourcePrompt("vxc", "Enter auth key (optional): ", noColor)
	if err != nil {
		return nil, err
	}

	prefixes, err := utils.ResourcePrompt("vxc", "Enter prefixes (optional): ", noColor)
	if err != nil {
		return nil, err
	}

	customerIPAddress, err := utils.ResourcePrompt("vxc", "Enter customer IP address (optional): ", noColor)
	if err != nil {
		return nil, err
	}

	amazonIPAddress, err := utils.ResourcePrompt("vxc", "Enter Amazon IP address (optional): ", noColor)
	if err != nil {
		return nil, err
	}

	partnerConfigAWS := &megaport.VXCPartnerConfigAWS{
		ConnectType:       connectType,
		OwnerAccount:      ownerAccount,
		ASN:               asn,
		AmazonASN:         amazonASN,
		AuthKey:           authKey,
		Prefixes:          prefixes,
		CustomerIPAddress: customerIPAddress,
		AmazonIPAddress:   amazonIPAddress,
		ConnectionName:    connectionName,
	}
	if connectType == "AWS" {
		vifType, err := utils.ResourcePrompt("vxc", "Enter VIF type (required - either private or public): ", noColor)
		if err != nil {
			return nil, err
		}
		if vifType != "private" && vifType != "public" {
			return nil, fmt.Errorf("VIF type must be private or public")
		}
		partnerConfigAWS.Type = vifType
	}
	return partnerConfigAWS, nil
}

func promptAzureConfig(ctx context.Context, svc megaport.VXCService, noColor bool) (*megaport.VXCPartnerConfigAzure, string, error) {
	serviceKey, err := utils.ResourcePrompt("vxc", "Enter service key (required): ", noColor)
	if err != nil {
		return nil, "", err
	}
	if serviceKey == "" {
		return nil, "", fmt.Errorf("service key is required")
	}

	portChoice, err := utils.ResourcePrompt("vxc", "Enter port choice (primary/secondary, optional, default value is primary): ", noColor)
	if err != nil {
		return nil, "", err
	}
	if portChoice != "" && portChoice != "primary" && portChoice != "secondary" {
		return nil, "", fmt.Errorf("port preference must be primary or secondary")
	}
	if portChoice == "" {
		portChoice = "primary"
	}

	var peers []megaport.PartnerOrderAzurePeeringConfig
	for {
		addPeer, err := utils.ResourcePrompt("vxc", "Add a peering config? (yes/no): ", noColor)
		if err != nil {
			return nil, "", err
		}
		if addPeer != "yes" {
			break
		}

		peerConfig, err := promptAzurePeeringConfig(noColor)
		if err != nil {
			return nil, "", err
		}
		peers = append(peers, peerConfig)
	}

	fmt.Println("Finding Azure partner port...")

	partnerPortRes, err := svc.ListPartnerPorts(ctx, &megaport.ListPartnerPortsRequest{
		Key:     serviceKey,
		Partner: "AZURE",
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to look up partner ports: %w", err)
	}
	var uid string
	for _, port := range partnerPortRes.Data.Megaports {
		p := &port
		if p.Type == portChoice {
			uid = p.ProductUID
		}
	}
	if uid == "" {
		return nil, "", fmt.Errorf("could not find azure port with type: %s", portChoice)
	}

	return &megaport.VXCPartnerConfigAzure{
		ConnectType: "AZURE",
		ServiceKey:  serviceKey,
		Peers:       peers,
	}, uid, nil
}

func promptAzurePeeringConfig(noColor bool) (megaport.PartnerOrderAzurePeeringConfig, error) {
	peeringType, err := utils.ResourcePrompt("vxc", "Enter peering type (required): ", noColor)
	if err != nil {
		return megaport.PartnerOrderAzurePeeringConfig{}, err
	}

	peerASN, err := utils.ResourcePrompt("vxc", "Enter Peer ASN (optional): ", noColor)
	if err != nil {
		return megaport.PartnerOrderAzurePeeringConfig{}, err
	}

	primarySubnet, err := utils.ResourcePrompt("vxc", "Enter Primary Subnet (optional): ", noColor)
	if err != nil {
		return megaport.PartnerOrderAzurePeeringConfig{}, err
	}

	secondarySubnet, err := utils.ResourcePrompt("vxc", "Enter Secondary Subnet (optional): ", noColor)
	if err != nil {
		return megaport.PartnerOrderAzurePeeringConfig{}, err
	}

	prefixes, err := utils.ResourcePrompt("vxc", "Enter Prefixes (optional): ", noColor)
	if err != nil {
		return megaport.PartnerOrderAzurePeeringConfig{}, err
	}

	sharedKey, err := utils.ResourcePrompt("vxc", "Enter Shared Key (optional): ", noColor)
	if err != nil {
		return megaport.PartnerOrderAzurePeeringConfig{}, err
	}

	vlanStr, err := utils.ResourcePrompt("vxc", "Enter VLAN (optional): ", noColor)
	if err != nil {
		return megaport.PartnerOrderAzurePeeringConfig{}, err
	}
	var vlan int
	if vlanStr != "" {
		vlan, err = validation.ParseInt("VLAN ID", vlanStr)
		if err != nil {
			return megaport.PartnerOrderAzurePeeringConfig{}, err
		}
		if err := validation.ValidateVLAN(vlan); err != nil {
			return megaport.PartnerOrderAzurePeeringConfig{}, err
		}
	}

	return megaport.PartnerOrderAzurePeeringConfig{
		Type:            peeringType,
		PeerASN:         peerASN,
		PrimarySubnet:   primarySubnet,
		SecondarySubnet: secondarySubnet,
		Prefixes:        prefixes,
		SharedKey:       sharedKey,
		VLAN:            vlan,
	}, nil
}

func promptGoogleConfig(ctx context.Context, svc megaport.VXCService, noColor bool) (*megaport.VXCPartnerConfigGoogle, string, error) {
	pairingKey, err := utils.ResourcePrompt("vxc", "Enter pairing key (required): ", noColor)
	if err != nil {
		return nil, "", err
	}
	if pairingKey == "" {
		return nil, "", fmt.Errorf("pairing key is required")
	}

	uid, err := getPartnerPortUID(ctx, svc, pairingKey, "GOOGLE")
	if err != nil {
		return nil, "", err
	}

	return &megaport.VXCPartnerConfigGoogle{
		ConnectType: "GOOGLE",
		PairingKey:  pairingKey,
	}, uid, nil
}

func promptOracleConfig(ctx context.Context, svc megaport.VXCService, noColor bool) (*megaport.VXCPartnerConfigOracle, string, error) {
	virtualCircuitId, err := utils.ResourcePrompt("vxc", "Enter virtual circuit ID (required): ", noColor)
	if err != nil {
		return nil, "", err
	}
	if virtualCircuitId == "" {
		return nil, "", fmt.Errorf("virtual circuit ID is required")
	}

	uid, err := getPartnerPortUID(ctx, svc, virtualCircuitId, "ORACLE")
	if err != nil {
		return nil, "", err
	}

	return &megaport.VXCPartnerConfigOracle{
		ConnectType:      "ORACLE",
		VirtualCircuitId: virtualCircuitId,
	}, uid, nil
}

func promptIBMConfig(noColor bool) (*megaport.VXCPartnerConfigIBM, error) {
	accountID, err := utils.ResourcePrompt("vxc", "Enter account ID (required): ", noColor)
	if err != nil {
		return nil, err
	}
	if accountID == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	name, err := utils.ResourcePrompt("vxc", "Enter name (optional, defaults to MEGAPORT): ", noColor)
	if err != nil {
		return nil, err
	}

	customerASNStr, err := utils.ResourcePrompt("vxc", "Enter customer ASN (required if opposite end is not an MCR): ", noColor)
	if err != nil {
		return nil, err
	}
	var customerASN int
	if customerASNStr != "" {
		customerASN, err = validation.ParseInt("customer ASN", customerASNStr)
		if err != nil {
			return nil, err
		}
	}

	customerIPAddress, err := utils.ResourcePrompt("vxc", "Enter customer IP address (optional): ", noColor)
	if err != nil {
		return nil, err
	}

	providerIPAddress, err := utils.ResourcePrompt("vxc", "Enter provider IP address (optional): ", noColor)
	if err != nil {
		return nil, err
	}

	partnerConfig := &megaport.VXCPartnerConfigIBM{
		ConnectType:       "IBM",
		AccountID:         accountID,
		CustomerIPAddress: customerIPAddress,
		ProviderIPAddress: providerIPAddress,
		Name:              name,
	}

	if customerASN != 0 {
		partnerConfig.CustomerASN = customerASN
	}

	return partnerConfig, nil
}
