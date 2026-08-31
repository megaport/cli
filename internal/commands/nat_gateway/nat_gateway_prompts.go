package nat_gateway

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/megaport/megaport-cli/internal/validation"
	megaport "github.com/megaport/megaportgo"
)

func promptForCreateNATGatewayDetails(noColor bool) (*megaport.CreateNATGatewayRequest, error) {
	req := &megaport.CreateNATGatewayRequest{}

	name, err := utils.ResourcePrompt("nat-gateway", "NAT Gateway name: ", noColor)
	if err != nil {
		return nil, err
	}
	req.ProductName = strings.TrimSpace(name)

	locationIDStr, err := utils.ResourcePrompt("nat-gateway", "Location ID: ", noColor)
	if err != nil {
		return nil, err
	}
	locationID, err := strconv.Atoi(strings.TrimSpace(locationIDStr))
	if err != nil || locationID < 1 {
		return nil, fmt.Errorf("invalid location ID: %s", locationIDStr)
	}
	req.LocationID = locationID

	speedStr, err := utils.ResourcePrompt("nat-gateway", "Speed (Mbps): ", noColor)
	if err != nil {
		return nil, err
	}
	speed, err := strconv.Atoi(strings.TrimSpace(speedStr))
	if err != nil || speed < 1 {
		return nil, fmt.Errorf("invalid speed: %s", speedStr)
	}
	req.Speed = speed

	termStr, err := utils.ResourcePrompt("nat-gateway",
		fmt.Sprintf("Contract term (%s months): ", validation.FormatIntSlice(validation.ValidContractTerms)), noColor)
	if err != nil {
		return nil, err
	}
	term, err := strconv.Atoi(strings.TrimSpace(termStr))
	if err != nil {
		return nil, fmt.Errorf("invalid term: %s", termStr)
	}
	req.Term = term

	sessionCountStr, err := utils.ResourcePrompt("nat-gateway", "Session count (optional, leave empty for default): ", noColor)
	if err != nil {
		return nil, err
	}
	sessionCountStr = strings.TrimSpace(sessionCountStr)
	if sessionCountStr != "" {
		sc, err := strconv.Atoi(sessionCountStr)
		if err != nil {
			return nil, fmt.Errorf("invalid session count: %s", sessionCountStr)
		}
		req.Config.SessionCount = sc
	}

	asnStr, err := utils.ResourcePrompt("nat-gateway", "BGP ASN (optional, leave empty for default): ", noColor)
	if err != nil {
		return nil, err
	}
	asnStr = strings.TrimSpace(asnStr)
	if asnStr != "" {
		asn, err := strconv.Atoi(asnStr)
		if err != nil {
			return nil, fmt.Errorf("invalid ASN: %s", asnStr)
		}
		if err := validation.ValidateASN(asn); err != nil {
			return nil, err
		}
		req.Config.ASN = asn
	}

	diversityZone, err := utils.ResourcePrompt("nat-gateway", "Diversity zone (optional, leave empty to skip): ", noColor)
	if err != nil {
		return nil, err
	}
	req.Config.DiversityZone = strings.TrimSpace(diversityZone)

	autoRenewStr, err := utils.ResourcePrompt("nat-gateway", "Auto-renew (y/yes/n/no, leave empty for no): ", noColor)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(autoRenewStr)) {
	case "y", "yes":
		req.AutoRenewTerm = true
	}

	promoCode, err := utils.ResourcePrompt("nat-gateway", "Promo code (optional, leave empty to skip): ", noColor)
	if err != nil {
		return nil, err
	}
	req.PromoCode = strings.TrimSpace(promoCode)

	serviceLevelRef, err := utils.ResourcePrompt("nat-gateway", "Service level reference (optional, leave empty to skip): ", noColor)
	if err != nil {
		return nil, err
	}
	req.ServiceLevelReference = strings.TrimSpace(serviceLevelRef)

	tagsMap, err := utils.ResourceTagsPrompt(noColor)
	if err != nil {
		return nil, err
	}
	for k, v := range tagsMap {
		req.ResourceTags = append(req.ResourceTags, megaport.ResourceTag{Key: k, Value: v})
	}

	if err := validation.ValidateCreateNATGatewayRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

// promptForUpdateNATGatewayDetails prompts the user for update fields interactively.
// It returns the request, an updateExplicitFields value indicating which zero-valued
// fields were intentionally provided (so mergeUpdateDefaults can distinguish an
// explicit zero/false/empty from an omitted field), and any error.
func promptForUpdateNATGatewayDetails(uid string, noColor bool) (*megaport.UpdateNATGatewayRequest, updateExplicitFields, error) {
	req := &megaport.UpdateNATGatewayRequest{ProductUID: uid}
	var explicit updateExplicitFields

	// All fields are optional for updates — empty input keeps the current value.
	// The action layer merges unset fields from the original resource.
	name, err := utils.ResourcePrompt("nat-gateway", "NAT Gateway name (leave empty to keep current): ", noColor)
	if err != nil {
		return nil, explicit, err
	}
	req.ProductName = strings.TrimSpace(name)

	locationIDStr, err := utils.ResourcePrompt("nat-gateway", "Location ID (leave empty to keep current): ", noColor)
	if err != nil {
		return nil, explicit, err
	}
	locationIDStr = strings.TrimSpace(locationIDStr)
	if locationIDStr != "" {
		locationID, err := strconv.Atoi(locationIDStr)
		if err != nil || locationID < 1 {
			return nil, explicit, fmt.Errorf("invalid location ID: %s", locationIDStr)
		}
		req.LocationID = locationID
	}

	speedStr, err := utils.ResourcePrompt("nat-gateway", "Speed in Mbps (leave empty to keep current): ", noColor)
	if err != nil {
		return nil, explicit, err
	}
	speedStr = strings.TrimSpace(speedStr)
	if speedStr != "" {
		speed, err := strconv.Atoi(speedStr)
		if err != nil || speed < 1 {
			return nil, explicit, fmt.Errorf("invalid speed: %s", speedStr)
		}
		req.Speed = speed
	}

	termStr, err := utils.ResourcePrompt("nat-gateway",
		fmt.Sprintf("Contract term in months (%s, leave empty to keep current): ", validation.FormatIntSlice(validation.ValidContractTerms)), noColor)
	if err != nil {
		return nil, explicit, err
	}
	termStr = strings.TrimSpace(termStr)
	if termStr != "" {
		term, err := strconv.Atoi(termStr)
		if err != nil {
			return nil, explicit, fmt.Errorf("invalid term: %s", termStr)
		}
		req.Term = term
	}

	sessionCountStr, err := utils.ResourcePrompt("nat-gateway", "Session count (leave empty to keep current): ", noColor)
	if err != nil {
		return nil, explicit, err
	}
	sessionCountStr = strings.TrimSpace(sessionCountStr)
	if sessionCountStr != "" {
		sc, err := strconv.Atoi(sessionCountStr)
		if err != nil {
			return nil, explicit, fmt.Errorf("invalid session count: %s", sessionCountStr)
		}
		req.Config.SessionCount = sc
		explicit.SessionCount = true
	}

	diversityZone, err := utils.ResourcePrompt("nat-gateway", "Diversity zone (leave empty to keep current): ", noColor)
	if err != nil {
		return nil, explicit, err
	}
	dz := strings.TrimSpace(diversityZone)
	req.Config.DiversityZone = dz
	explicit.DiversityZone = dz != ""

	autoRenewStr, err := utils.ResourcePrompt("nat-gateway", "Auto-renew (y/yes/n/no, leave empty to keep current): ", noColor)
	if err != nil {
		return nil, explicit, err
	}
	switch strings.ToLower(strings.TrimSpace(autoRenewStr)) {
	case "y", "yes":
		req.AutoRenewTerm = true
		explicit.AutoRenewTerm = true
	case "n", "no":
		req.AutoRenewTerm = false
		explicit.AutoRenewTerm = true
	}

	promoCode, err := utils.ResourcePrompt("nat-gateway", "Promo code (leave empty to keep current): ", noColor)
	if err != nil {
		return nil, explicit, err
	}
	req.PromoCode = strings.TrimSpace(promoCode)
	explicit.PromoCode = req.PromoCode != ""

	serviceLevelRef, err := utils.ResourcePrompt("nat-gateway", "Service level reference (leave empty to keep current): ", noColor)
	if err != nil {
		return nil, explicit, err
	}
	req.ServiceLevelReference = strings.TrimSpace(serviceLevelRef)
	explicit.ServiceLevelReference = req.ServiceLevelReference != ""

	tagsMap, err := utils.ResourceTagsPrompt(noColor)
	if err != nil {
		return nil, explicit, err
	}
	for k, v := range tagsMap {
		req.ResourceTags = append(req.ResourceTags, megaport.ResourceTag{Key: k, Value: v})
	}

	return req, explicit, nil
}
