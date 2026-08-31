package vxc

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/megaport/megaport-cli/internal/validation"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
)

var buildVXCRequestFromFlags = func(cmd *cobra.Command, ctx context.Context, svc megaport.VXCService) (*megaport.BuyVXCRequest, error) {
	// Flag read errors are intentionally ignored — flags are registered by the command builder.
	aEndUID, _ := cmd.Flags().GetString("a-end-uid")

	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		return nil, exitcodes.NewUsageError(validation.NewValidationError("VXC name", name, "cannot be empty"))
	}

	rateLimit, _ := cmd.Flags().GetInt("rate-limit")
	if err := validation.ValidateRateLimit(rateLimit); err != nil {
		return nil, err
	}

	term, _ := cmd.Flags().GetInt("term")
	if err := validation.ValidateContractTerm(term); err != nil {
		return nil, err
	}

	// Get optional fields
	aEndVLAN, _ := cmd.Flags().GetInt("a-end-vlan")
	bEndVLAN, _ := cmd.Flags().GetInt("b-end-vlan")
	aEndInnerVLAN, _ := cmd.Flags().GetInt("a-end-inner-vlan")
	bEndInnerVLAN, _ := cmd.Flags().GetInt("b-end-inner-vlan")
	aEndVNICIndex, _ := cmd.Flags().GetInt("a-end-vnic-index")
	bEndVNICIndex, _ := cmd.Flags().GetInt("b-end-vnic-index")
	promoCode, _ := cmd.Flags().GetString("promo-code")
	serviceKey, _ := cmd.Flags().GetString("service-key")
	costCentre, _ := cmd.Flags().GetString("cost-centre")

	resourceTagsStr, _ := cmd.Flags().GetString("resource-tags")
	resourceTagsFile, _ := cmd.Flags().GetString("resource-tags-file")
	resourceTags, err := utils.ParseResourceTagsFlagOrFile(resourceTagsStr, resourceTagsFile)
	if err != nil {
		return nil, exitcodes.NewUsageError(err)
	}

	// Create the base request
	req := &megaport.BuyVXCRequest{
		VXCName:      name,
		RateLimit:    rateLimit,
		Term:         term,
		PromoCode:    promoCode,
		ServiceKey:   serviceKey,
		CostCentre:   costCentre,
		ResourceTags: resourceTags,
	}

	// A-End configuration
	aEndConfig := megaport.VXCOrderEndpointConfiguration{
		VLAN: aEndVLAN,
	}

	// Set MVE config if needed. vNIC index 0 is valid, so gate on whether the
	// flag was set rather than on a non-zero value.
	if aEndInnerVLAN != 0 || cmd.Flags().Changed("a-end-vnic-index") {
		aEndConfig.VXCOrderMVEConfig = &megaport.VXCOrderMVEConfig{
			InnerVLAN:             aEndInnerVLAN,
			NetworkInterfaceIndex: aEndVNICIndex,
		}
	}

	// Parse A-End partner config if provided
	aEndPartnerConfigStr, _ := cmd.Flags().GetString("a-end-partner-config")
	if aEndPartnerConfigStr != "" {
		aEndPartnerConfig, err := parsePartnerConfigFromJSON(aEndPartnerConfigStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse a-end-partner-config: %w", err)
		}
		aEndConfig.PartnerConfig = aEndPartnerConfig
		if aEndUID == "" {
			uid, err := resolvePartnerPortUID(ctx, svc, aEndPartnerConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to look up A-End Partner Port: %w", err)
			}
			aEndUID = uid
		}
	}

	req.AEndConfiguration = aEndConfig

	if aEndUID == "" {
		return nil, fmt.Errorf("a-end-uid was neither specified nor could be looked up")
	}

	req.PortUID = aEndUID

	// B-End configuration
	bEndConfig := megaport.VXCOrderEndpointConfiguration{}

	// Parse B-End partner config if provided
	bEndPartnerConfigStr, _ := cmd.Flags().GetString("b-end-partner-config")
	if bEndPartnerConfigStr != "" {
		bEndPartnerConfig, err := parsePartnerConfigFromJSON(bEndPartnerConfigStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse b-end-partner-config: %w", err)
		}
		bEndConfig.PartnerConfig = bEndPartnerConfig
	}

	bEndUID, _ := cmd.Flags().GetString("b-end-uid")

	// Attempt to look up partner port UID if not provided
	if bEndUID == "" && bEndConfig.PartnerConfig != nil {
		uid, err := resolvePartnerPortUID(ctx, svc, bEndConfig.PartnerConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to look up B-End Partner Port: %w", err)
		}
		bEndUID = uid
	}

	if bEndUID == "" {
		return nil, fmt.Errorf("b-end-uid was neither provided nor could be looked up")
	}

	bEndConfig.ProductUID = bEndUID
	bEndConfig.VLAN = bEndVLAN

	// Set MVE config if needed. vNIC index 0 is valid, so gate on whether the
	// flag was set rather than on a non-zero value.
	if bEndInnerVLAN != 0 || cmd.Flags().Changed("b-end-vnic-index") {
		bEndConfig.VXCOrderMVEConfig = &megaport.VXCOrderMVEConfig{
			InnerVLAN:             bEndInnerVLAN,
			NetworkInterfaceIndex: bEndVNICIndex,
		}
	}

	req.BEndConfiguration = bEndConfig

	if err := validation.ValidateVXCRequest(req); err != nil {
		return nil, err
	}

	return req, nil
}

// parseVXCEndpointConfig parses common endpoint configuration fields (productUID, vlan,
// diversityZone, partnerConfig, MVE config) from a raw JSON map into a VXCOrderEndpointConfiguration.
// The endLabel parameter (e.g. "A-End", "B-End") is used for error messages.
func parseVXCEndpointConfig(endConfigRaw map[string]interface{}, endLabel string) (megaport.VXCOrderEndpointConfiguration, error) {
	config := megaport.VXCOrderEndpointConfiguration{}

	if productUID, present, err := utils.JSONString(endConfigRaw, "productUID"); err != nil {
		return config, exitcodes.NewUsageError(fmt.Errorf("%s: %w", endLabel, err))
	} else if present {
		config.ProductUID = productUID
	}

	if vlan, present, err := utils.JSONNumber(endConfigRaw, "vlan"); err != nil {
		return config, exitcodes.NewUsageError(fmt.Errorf("%s: %w", endLabel, err))
	} else if present {
		config.VLAN = int(vlan)
	}

	if diversityZone, present, err := utils.JSONString(endConfigRaw, "diversityZone"); err != nil {
		return config, exitcodes.NewUsageError(fmt.Errorf("%s: %w", endLabel, err))
	} else if present {
		config.DiversityZone = diversityZone
	}

	// Handle partner config - directly use map data
	if partnerConfigRaw, present, err := utils.JSONObject(endConfigRaw, "partnerConfig"); err != nil {
		return config, exitcodes.NewUsageError(fmt.Errorf("%s: %w", endLabel, err))
	} else if present {
		partnerConfig, err := parsePartnerConfigFromMap(partnerConfigRaw)
		if err != nil {
			return config, exitcodes.NewUsageError(fmt.Errorf("failed to parse %s partner config: %w", endLabel, err))
		}

		config.PartnerConfig = partnerConfig
	}

	// Handle MVE config
	innerVLAN, hasInnerVLAN, err := utils.JSONNumber(endConfigRaw, "innerVlan")
	if err != nil {
		return config, exitcodes.NewUsageError(fmt.Errorf("%s: %w", endLabel, err))
	}
	vNicIndex, hasVNicIndex, err := utils.JSONNumber(endConfigRaw, "vNicIndex")
	if err != nil {
		return config, exitcodes.NewUsageError(fmt.Errorf("%s: %w", endLabel, err))
	}

	if hasInnerVLAN || hasVNicIndex {
		mveConfig := &megaport.VXCOrderMVEConfig{}

		if hasInnerVLAN {
			mveConfig.InnerVLAN = int(innerVLAN)
		}

		if hasVNicIndex {
			mveConfig.NetworkInterfaceIndex = int(vNicIndex)
		}

		config.VXCOrderMVEConfig = mveConfig
	}

	return config, nil
}

func buildVXCRequestFromJSON(jsonStr string, jsonFilePath string, ctx context.Context, svc megaport.VXCService) (*megaport.BuyVXCRequest, error) {
	if jsonStr == "" && jsonFilePath == "" {
		return nil, exitcodes.NewUsageError(fmt.Errorf("either json or json-file must be provided"))
	}

	jsonData, err := utils.ReadJSONInput(jsonStr, jsonFilePath)
	if err != nil {
		return nil, exitcodes.NewUsageError(err)
	}

	// Parse raw JSON first to handle partner configs correctly
	var rawData map[string]interface{}
	if err := json.Unmarshal(jsonData, &rawData); err != nil {
		return nil, exitcodes.NewUsageError(fmt.Errorf("failed to parse JSON: %w", err))
	}

	// Parse A-End configuration early: its partner config, if present, can
	// resolve a missing top-level portUid the same way flags mode does.
	var aEndConfig megaport.VXCOrderEndpointConfiguration
	if aEndConfigRaw, present, err := utils.JSONObject(rawData, "aEndConfiguration"); err != nil {
		return nil, exitcodes.NewUsageError(err)
	} else if present {
		aEndConfig, err = parseVXCEndpointConfig(aEndConfigRaw, "A-End")
		if err != nil {
			return nil, err
		}
	}

	// Parse and format-check vxcName/rateLimit/term before any partner-port
	// lookup, so malformed JSON fails fast without an API round-trip.
	vxcName, _, err := utils.JSONString(rawData, "vxcName")
	if err != nil {
		return nil, exitcodes.NewUsageError(err)
	}

	rateLimit, _, err := utils.JSONNumber(rawData, "rateLimit")
	if err != nil {
		return nil, exitcodes.NewUsageError(err)
	}
	if rateLimit != math.Trunc(rateLimit) {
		return nil, exitcodes.NewUsageError(fmt.Errorf("rateLimit must be a whole number, got %v", rateLimit))
	}

	term, _, err := utils.JSONNumber(rawData, "term")
	if err != nil {
		return nil, exitcodes.NewUsageError(err)
	}
	if term != math.Trunc(term) {
		return nil, exitcodes.NewUsageError(fmt.Errorf("term must be a whole number, got %v", term))
	}

	// Required fields the buy request can't do without, checked before any
	// partner-port lookup so a partial JSON body fails fast rather than
	// spending an API round-trip on a request that will fail validation anyway.
	requireBuyFields := func() error {
		if vxcName == "" {
			return exitcodes.NewUsageError(validation.NewValidationError("VXC name", vxcName, "cannot be empty"))
		}
		if err := validation.ValidateRateLimit(int(rateLimit)); err != nil {
			return exitcodes.NewUsageError(err)
		}
		if err := validation.ValidateContractTerm(int(term)); err != nil {
			return exitcodes.NewUsageError(err)
		}
		return nil
	}

	portUID, present, err := utils.JSONString(rawData, "portUid")
	if err != nil {
		return nil, exitcodes.NewUsageError(err)
	}
	if !present {
		if aEndConfig.PartnerConfig == nil {
			return nil, exitcodes.NewUsageError(validation.NewValidationError("portUid", "", "Port UID is required"))
		}
		if err := requireBuyFields(); err != nil {
			return nil, err
		}
		uid, err := resolvePartnerPortUID(ctx, svc, aEndConfig.PartnerConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to look up A-End Partner Port: %w", err)
		}
		if uid == "" {
			return nil, exitcodes.NewUsageError(validation.NewValidationError("portUid", "", "Port UID is required"))
		}
		portUID = uid
	}

	// Create the base request
	req := &megaport.BuyVXCRequest{
		PortUID:           portUID,
		AEndConfiguration: aEndConfig,
		VXCName:           vxcName,
		RateLimit:         int(rateLimit),
		Term:              int(term),
	}

	if shutdown, present, err := utils.JSONBool(rawData, "shutdown"); err != nil {
		return nil, exitcodes.NewUsageError(err)
	} else if present {
		req.Shutdown = shutdown
	}

	if promoCode, present, err := utils.JSONString(rawData, "promoCode"); err != nil {
		return nil, exitcodes.NewUsageError(err)
	} else if present {
		req.PromoCode = promoCode
	}

	if serviceKey, present, err := utils.JSONString(rawData, "serviceKey"); err != nil {
		return nil, exitcodes.NewUsageError(err)
	} else if present {
		req.ServiceKey = serviceKey
	}

	if costCentre, present, err := utils.JSONString(rawData, "costCentre"); err != nil {
		return nil, exitcodes.NewUsageError(err)
	} else if present {
		req.CostCentre = costCentre
	}

	// Handle resource tags if they exist
	if resourceTags, present, err := utils.JSONObject(rawData, "resourceTags"); err != nil {
		return nil, exitcodes.NewUsageError(err)
	} else if present {
		tags, err := utils.TagMapFromObject(resourceTags)
		if err != nil {
			return nil, exitcodes.NewUsageError(err)
		}
		req.ResourceTags = tags
	}

	// Handle B-End configuration. Resolve a missing productUID from the
	// partner config the same way flags mode does, so the same logical
	// purchase succeeds through either input mode.
	if bEndConfigRaw, present, err := utils.JSONObject(rawData, "bEndConfiguration"); err != nil {
		return nil, exitcodes.NewUsageError(err)
	} else if present {
		bEndConfig, err := parseVXCEndpointConfig(bEndConfigRaw, "B-End")
		if err != nil {
			return nil, err
		}
		if bEndConfig.ProductUID == "" && bEndConfig.PartnerConfig != nil {
			if err := requireBuyFields(); err != nil {
				return nil, err
			}
			uid, err := resolvePartnerPortUID(ctx, svc, bEndConfig.PartnerConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to look up B-End Partner Port: %w", err)
			}
			if uid == "" {
				return nil, exitcodes.NewUsageError(fmt.Errorf("bEndConfiguration.productUID was neither provided nor could be looked up"))
			}
			bEndConfig.ProductUID = uid
		}
		req.BEndConfiguration = bEndConfig
	}

	if err := validation.ValidateVXCRequest(req); err != nil {
		return nil, err
	}

	return req, nil
}
