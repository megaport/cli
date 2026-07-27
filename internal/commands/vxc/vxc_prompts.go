package vxc

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/megaport/megaport-cli/internal/validation"
	megaport "github.com/megaport/megaportgo"
)

var buildVXCRequestFromPrompt = func(ctx context.Context, svc megaport.VXCService, noColor bool) (*megaport.BuyVXCRequest, error) {
	name, err := utils.ResourcePrompt("vxc", "Enter VXC name (required): ", noColor)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, validation.NewValidationError("VXC name", name, "cannot be empty")
	}

	rateLimitStr, err := utils.ResourcePrompt("vxc", "Enter rate limit in Mbps (required): ", noColor)
	if err != nil {
		return nil, err
	}
	rateLimit, err := strconv.Atoi(rateLimitStr)
	if err != nil {
		return nil, fmt.Errorf("rate limit must be a valid integer")
	}
	if err := validation.ValidateRateLimit(rateLimit); err != nil {
		return nil, err
	}

	termStr, err := utils.ResourcePrompt("vxc", fmt.Sprintf("Enter term in months (%s, required): ", validation.FormatIntSlice(validation.ValidContractTerms)), noColor)
	if err != nil {
		return nil, err
	}
	term, err := strconv.Atoi(termStr)
	if err != nil {
		return nil, fmt.Errorf("term must be a valid integer")
	}
	if err := validation.ValidateContractTerm(term); err != nil {
		return nil, err
	}

	aEndVLANStr, err := utils.ResourcePrompt("vxc", fmt.Sprintf("A-End VLAN (%s): ", validation.VLANHelpText()), noColor)
	if err != nil {
		return nil, err
	}
	var aEndVLAN int
	if aEndVLANStr != "" {
		aEndVLAN, err = strconv.Atoi(aEndVLANStr)
		if err != nil {
			return nil, fmt.Errorf("A-End VLAN must be a valid integer")
		}
		if err := validation.ValidateVXCEndVLAN(aEndVLAN); err != nil {
			return nil, err
		}
	}

	aEndInnerVLANStr, err := utils.ResourcePrompt("vxc", fmt.Sprintf("A-End Inner VLAN (optional, %s, press Enter to skip): ", validation.InnerVLANHelpText()), noColor)
	if err != nil {
		return nil, err
	}
	aEndInnerVLAN := 0
	if aEndInnerVLANStr != "" {
		aEndInnerVLAN, err = strconv.Atoi(aEndInnerVLANStr)
		if err != nil {
			return nil, fmt.Errorf("invalid A-End Inner VLAN")
		}
		if err := validation.ValidateVXCEndInnerVLAN(aEndInnerVLAN); err != nil {
			return nil, err
		}
	}

	aEndVNICIndexStr, err := utils.ResourcePrompt("vxc", "Enter A-End vNIC Index (optional): ", noColor)
	if err != nil {
		return nil, err
	}
	aEndVNICIndex := 0
	if aEndVNICIndexStr != "" {
		aEndVNICIndex, err = strconv.Atoi(aEndVNICIndexStr)
		if err != nil {
			return nil, fmt.Errorf("invalid A-End vNIC Index")
		}
	}

	hasAEndPartnerConfig, err := utils.ResourcePrompt("vxc", "Do you want to configure A-End partner? (yes/no): ", noColor)
	if err != nil {
		return nil, err
	}

	aEndConfig := megaport.VXCOrderEndpointConfiguration{
		VLAN: aEndVLAN,
	}

	if aEndInnerVLAN != 0 || aEndVNICIndexStr != "" {
		aEndConfig.VXCOrderMVEConfig = &megaport.VXCOrderMVEConfig{
			InnerVLAN:             aEndInnerVLAN,
			NetworkInterfaceIndex: aEndVNICIndex,
		}
	}

	req := &megaport.BuyVXCRequest{
		VXCName:   name,
		RateLimit: rateLimit,
		Term:      term,
	}

	if strings.ToLower(hasAEndPartnerConfig) == "yes" {
		aEndPartnerConfig, uid, err := promptPartnerConfig("A-End", ctx, svc, noColor)
		if err != nil {
			return nil, err
		}
		aEndConfig.PartnerConfig = aEndPartnerConfig
		if uid != "" {
			req.PortUID = uid
		}
	}

	if req.PortUID == "" {
		aEndUID, err := utils.ResourcePrompt("vxc", "Enter A-End product UID (required): ", noColor)
		if err != nil {
			return nil, err
		}
		if aEndUID == "" {
			return nil, fmt.Errorf("a-end product UID is required")
		}
		req.PortUID = aEndUID
	}
	req.AEndConfiguration = aEndConfig

	bEndConfig := megaport.VXCOrderEndpointConfiguration{}

	bEndVLANStr, err := utils.ResourcePrompt("vxc", fmt.Sprintf("B-End VLAN (%s): ", validation.VLANHelpText()), noColor)
	if err != nil {
		return nil, err
	}
	var bEndVLAN int
	if bEndVLANStr != "" {
		bEndVLAN, err = strconv.Atoi(bEndVLANStr)
		if err != nil {
			return nil, fmt.Errorf("B-End VLAN must be a valid integer")
		}
		if err := validation.ValidateVXCEndVLAN(bEndVLAN); err != nil {
			return nil, err
		}
		bEndConfig.VLAN = bEndVLAN
	}

	bEndInnerVLANStr, err := utils.ResourcePrompt("vxc", fmt.Sprintf("B-End Inner VLAN (optional, %s, press Enter to skip): ", validation.InnerVLANHelpText()), noColor)
	if err != nil {
		return nil, err
	}
	bEndInnerVLAN := 0
	if bEndInnerVLANStr != "" {
		bEndInnerVLAN, err = strconv.Atoi(bEndInnerVLANStr)
		if err != nil {
			return nil, fmt.Errorf("invalid B-End Inner VLAN")
		}
		if err := validation.ValidateVXCEndInnerVLAN(bEndInnerVLAN); err != nil {
			return nil, err
		}
	}

	bEndVNICIndexStr, err := utils.ResourcePrompt("vxc", "Enter B-End vNIC Index (optional): ", noColor)
	if err != nil {
		return nil, err
	}
	bEndVNICIndex := 0
	if bEndVNICIndexStr != "" {
		bEndVNICIndex, err = strconv.Atoi(bEndVNICIndexStr)
		if err != nil {
			return nil, fmt.Errorf("invalid B-End vNIC Index")
		}
	}

	if bEndInnerVLAN != 0 || bEndVNICIndexStr != "" {
		bEndConfig.VXCOrderMVEConfig = &megaport.VXCOrderMVEConfig{
			InnerVLAN:             bEndInnerVLAN,
			NetworkInterfaceIndex: bEndVNICIndex,
		}
	}

	hasBEndPartnerConfig, err := utils.ResourcePrompt("vxc", "Do you want to configure B-End partner? (yes/no): ", noColor)
	if err != nil {
		return nil, err
	}

	if strings.ToLower(hasBEndPartnerConfig) == "yes" {
		bEndPartnerConfig, uid, err := promptPartnerConfig("B-End", ctx, svc, noColor)
		if err != nil {
			return nil, err
		}
		if uid != "" {
			bEndConfig.ProductUID = uid
		}
		bEndConfig.PartnerConfig = bEndPartnerConfig
	}

	if bEndConfig.ProductUID == "" {
		bEndUID, err := utils.ResourcePrompt("vxc", "Enter B-End product UID (required): ", noColor)
		if err != nil {
			return nil, err
		}
		if bEndUID == "" {
			return nil, fmt.Errorf("B-End product UID is required")
		}
		bEndConfig.ProductUID = bEndUID
	}

	req.BEndConfiguration = bEndConfig

	promoCode, err := utils.ResourcePrompt("vxc", "Enter promo code (optional): ", noColor)
	if err != nil {
		return nil, err
	}
	req.PromoCode = promoCode

	serviceKey, err := utils.ResourcePrompt("vxc", "Enter service key (optional): ", noColor)
	if err != nil {
		return nil, err
	}
	req.ServiceKey = serviceKey

	costCentre, err := utils.ResourcePrompt("vxc", "Enter cost centre (optional): ", noColor)
	if err != nil {
		return nil, err
	}
	req.CostCentre = costCentre

	return req, nil
}

var buildUpdateVXCRequestFromPrompt = func(ctx context.Context, client *megaport.Client, vxcUID string, noColor bool) (*megaport.UpdateVXCRequest, error) {
	req := &megaport.UpdateVXCRequest{
		WaitForUpdate: true,
	}

	fmt.Fprintln(os.Stderr, "Fetching current VXC details...")
	vxc, err := client.VXCService.GetVXC(ctx, vxcUID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch VXC details: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Current name: %s\n", vxc.Name)
	updateName, err := utils.ResourcePrompt("vxc", "Update name? (yes/no): ", noColor)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(updateName) == "yes" {
		name, err := utils.ResourcePrompt("vxc", "Enter new name: ", noColor)
		if err != nil {
			return nil, err
		}
		req.Name = &name
	}

	fmt.Fprintf(os.Stderr, "Current rate limit: %d Mbps\n", vxc.RateLimit)
	updateRateLimit, err := utils.ResourcePrompt("vxc", "Update rate limit? (yes/no): ", noColor)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(updateRateLimit) == "yes" {
		rateLimitStr, err := utils.ResourcePrompt("vxc", "Enter new rate limit in Mbps: ", noColor)
		if err != nil {
			return nil, err
		}
		rateLimit, err := strconv.Atoi(rateLimitStr)
		if err != nil {
			return nil, fmt.Errorf("rate limit must be a valid integer")
		}
		if err := validation.ValidateRateLimit(rateLimit); err != nil {
			return nil, err
		}
		req.RateLimit = &rateLimit
	}

	fmt.Fprintf(os.Stderr, "Current term: %d months\n", vxc.ContractTermMonths)
	updateTerm, err := utils.ResourcePrompt("vxc", "Update term? (yes/no): ", noColor)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(updateTerm) == "yes" {
		termStr, err := utils.ResourcePrompt("vxc", fmt.Sprintf("Enter new term in months (%s): ", validation.FormatIntSlice(validation.ValidContractTerms)), noColor)
		if err != nil {
			return nil, err
		}
		term, err := strconv.Atoi(termStr)
		if err != nil {
			return nil, fmt.Errorf("term must be a valid integer")
		}
		if err := validation.ValidateContractTerm(term); err != nil {
			return nil, err
		}
		req.Term = &term
	}

	fmt.Fprintf(os.Stderr, "Current cost centre: %s\n", vxc.CostCentre)
	updateCostCentre, err := utils.ResourcePrompt("vxc", "Update cost centre? (yes/no): ", noColor)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(updateCostCentre) == "yes" {
		costCentre, err := utils.ResourcePrompt("vxc", "Enter new cost centre: ", noColor)
		if err != nil {
			return nil, err
		}
		req.CostCentre = &costCentre
	}

	shutdownStatus := "No"
	if vxc.Shutdown {
		shutdownStatus = "Yes"
	}
	fmt.Fprintf(os.Stderr, "Current shutdown status: %s\n", shutdownStatus)
	updateShutdown, err := utils.ResourcePrompt("vxc", "Update shutdown status? (yes/no): ", noColor)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(updateShutdown) == "yes" {
		shutdownStr, err := utils.ResourcePrompt("vxc", "Shut down the VXC? (yes/no): ", noColor)
		if err != nil {
			return nil, err
		}
		shutdown := strings.ToLower(shutdownStr) == "yes"
		req.Shutdown = &shutdown
	}

	fmt.Fprintf(os.Stderr, "Current A-End VLAN: %d\n", vxc.AEndConfiguration.VLAN)
	updateAEndVLAN, err := utils.ResourcePrompt("vxc", "Update A-End VLAN? (yes/no): ", noColor)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(updateAEndVLAN) == "yes" {
		aEndVLANStr, err := utils.ResourcePrompt("vxc", fmt.Sprintf("A-End VLAN (%s): ", validation.VLANHelpText()), noColor)
		if err != nil {
			return nil, err
		}
		aEndVLAN, err := strconv.Atoi(aEndVLANStr)
		if err != nil {
			return nil, fmt.Errorf("A-End VLAN must be a valid integer")
		}
		if err := validation.ValidateVXCEndVLAN(aEndVLAN); err != nil {
			return nil, err
		}
		req.AEndVLAN = &aEndVLAN
	}

	fmt.Fprintf(os.Stderr, "Current B-End VLAN: %d\n", vxc.BEndConfiguration.VLAN)
	updateBEndVLAN, err := utils.ResourcePrompt("vxc", "Update B-End VLAN? (yes/no): ", noColor)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(updateBEndVLAN) == "yes" {
		bEndVLANStr, err := utils.ResourcePrompt("vxc", fmt.Sprintf("B-End VLAN (%s): ", validation.VLANHelpText()), noColor)
		if err != nil {
			return nil, err
		}
		bEndVLAN, err := strconv.Atoi(bEndVLANStr)
		if err != nil {
			return nil, fmt.Errorf("B-End VLAN must be a valid integer")
		}
		if err := validation.ValidateVXCEndVLAN(bEndVLAN); err != nil {
			return nil, err
		}
		req.BEndVLAN = &bEndVLAN
	}

	innerVLANAEnd := 0
	if vxc.AEndConfiguration.InnerVLAN != 0 {
		innerVLANAEnd = vxc.AEndConfiguration.InnerVLAN
	}
	fmt.Fprintf(os.Stderr, "Current A-End Inner VLAN: %d\n", innerVLANAEnd)
	updateAEndInnerVLAN, err := utils.ResourcePrompt("vxc", "Update A-End Inner VLAN? (yes/no): ", noColor)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(updateAEndInnerVLAN) == "yes" {
		aEndInnerVLANStr, err := utils.ResourcePrompt("vxc", fmt.Sprintf("Enter new A-End Inner VLAN (%s): ", validation.InnerVLANHelpText()), noColor)
		if err != nil {
			return nil, err
		}
		aEndInnerVLAN, err := strconv.Atoi(aEndInnerVLANStr)
		if err != nil {
			return nil, fmt.Errorf("A-End Inner VLAN must be a valid integer")
		}

		if err := validation.ValidateVXCEndInnerVLAN(aEndInnerVLAN); err != nil {
			return nil, err
		}

		req.AEndInnerVLAN = &aEndInnerVLAN
	}

	innerVLANBEnd := 0
	if vxc.BEndConfiguration.InnerVLAN != 0 {
		innerVLANBEnd = vxc.BEndConfiguration.InnerVLAN
	}
	fmt.Fprintf(os.Stderr, "Current B-End Inner VLAN: %d\n", innerVLANBEnd)
	updateBEndInnerVLAN, err := utils.ResourcePrompt("vxc", "Update B-End Inner VLAN? (yes/no): ", noColor)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(updateBEndInnerVLAN) == "yes" {
		bEndInnerVLANStr, err := utils.ResourcePrompt("vxc", fmt.Sprintf("Enter new B-End Inner VLAN (%s): ", validation.InnerVLANHelpText()), noColor)
		if err != nil {
			return nil, err
		}
		bEndInnerVLAN, err := strconv.Atoi(bEndInnerVLANStr)
		if err != nil {
			return nil, fmt.Errorf("B-End Inner VLAN must be a valid integer")
		}

		if err := validation.ValidateVXCEndInnerVLAN(bEndInnerVLAN); err != nil {
			return nil, err
		}

		req.BEndInnerVLAN = &bEndInnerVLAN
	}

	fmt.Fprintf(os.Stderr, "Current A-End UID: %s\n", vxc.AEndConfiguration.UID)
	updateAEndUID, err := utils.ResourcePrompt("vxc", "Update A-End product UID? (yes/no): ", noColor)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(updateAEndUID) == "yes" {
		aEndUID, err := utils.ResourcePrompt("vxc", "Enter new A-End product UID: ", noColor)
		if err != nil {
			return nil, err
		}
		req.AEndProductUID = &aEndUID
	}

	fmt.Fprintf(os.Stderr, "Current B-End UID: %s\n", vxc.BEndConfiguration.UID)
	updateBEndUID, err := utils.ResourcePrompt("vxc", "Update B-End product UID? (yes/no): ", noColor)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(updateBEndUID) == "yes" {
		bEndUID, err := utils.ResourcePrompt("vxc", "Enter new B-End product UID: ", noColor)
		if err != nil {
			return nil, err
		}
		req.BEndProductUID = &bEndUID
	}

	wantsAEndPartnerConfig, err := utils.ResourcePrompt("vxc", "Do you want to configure an A-End VRouter partner configuration? (yes/no): ", noColor)
	if err != nil {
		return nil, err
	}

	if strings.ToLower(wantsAEndPartnerConfig) == "yes" {
		aEndPartnerConfig, err := promptVRouterConfig(noColor)
		if err != nil {
			return nil, err
		}
		req.AEndPartnerConfig = aEndPartnerConfig
	}

	wantsBEndPartnerConfig, err := utils.ResourcePrompt("vxc", "Do you want to configure a B-End VRouter partner configuration? (yes/no): ", noColor)
	if err != nil {
		return nil, err
	}

	if strings.ToLower(wantsBEndPartnerConfig) == "yes" {
		bEndPartnerConfig, err := promptVRouterConfig(noColor)
		if err != nil {
			return nil, err
		}
		req.BEndPartnerConfig = bEndPartnerConfig
	}

	return req, nil
}
