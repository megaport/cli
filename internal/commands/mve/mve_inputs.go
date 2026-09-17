package mve

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/megaport/megaport-cli/internal/validation"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
)

func processJSONBuyMVEInput(jsonStr, jsonFilePath string) (*megaport.BuyMVERequest, error) {
	var jsonData map[string]interface{}

	rawBytes, err := utils.ReadJSONInput(jsonStr, jsonFilePath)
	if err != nil {
		return nil, exitcodes.NewUsageError(err)
	}
	if err := json.Unmarshal(rawBytes, &jsonData); err != nil {
		return nil, exitcodes.NewUsageError(fmt.Errorf("failed to parse JSON: %w", err))
	}

	req := &megaport.BuyMVERequest{}

	if name, present, err := utils.JSONString(jsonData, "name"); err != nil {
		return nil, exitcodes.NewUsageError(err)
	} else if present && name != "" {
		req.Name = name
	}

	if term, present, err := utils.JSONNumber(jsonData, "term"); err != nil {
		return nil, exitcodes.NewUsageError(err)
	} else if present && term > 0 {
		req.Term = int(term)
	}

	if locationID, present, err := utils.JSONNumber(jsonData, "locationId"); err != nil {
		return nil, exitcodes.NewUsageError(err)
	} else if present && locationID > 0 {
		req.LocationID = int(locationID)
	}

	if diversityZone, present, err := utils.JSONString(jsonData, "diversityZone"); err != nil {
		return nil, exitcodes.NewUsageError(err)
	} else if present {
		req.DiversityZone = diversityZone
	}

	if promoCode, present, err := utils.JSONString(jsonData, "promoCode"); err != nil {
		return nil, exitcodes.NewUsageError(err)
	} else if present {
		req.PromoCode = promoCode
	}

	if costCentre, present, err := utils.JSONString(jsonData, "costCentre"); err != nil {
		return nil, exitcodes.NewUsageError(err)
	} else if present {
		req.CostCentre = costCentre
	}

	if vendorConfigMap, present, err := utils.JSONObject(jsonData, "vendorConfig"); err != nil {
		return nil, exitcodes.NewUsageError(err)
	} else if present {
		vendorConfig, err := ParseVendorConfig(vendorConfigMap)
		if err != nil {
			return nil, exitcodes.NewUsageError(err)
		}
		req.VendorConfig = vendorConfig
	}

	if vnicsData, present, err := utils.JSONArray(jsonData, "vnics"); err != nil {
		return nil, exitcodes.NewUsageError(err)
	} else if present {
		vnics := make([]megaport.MVENetworkInterface, 0, len(vnicsData))
		for i, vnicData := range vnicsData {
			vnicMap, ok := vnicData.(map[string]interface{})
			if !ok {
				return nil, exitcodes.NewUsageError(fmt.Errorf("vnics[%d] must be an object", i))
			}
			vnic := megaport.MVENetworkInterface{}

			if description, present, err := utils.JSONString(vnicMap, "description"); err != nil {
				return nil, exitcodes.NewUsageError(fmt.Errorf("vnics[%d] %w", i, err))
			} else if present {
				vnic.Description = description
			}

			if vlan, present, err := utils.JSONNumber(vnicMap, "vlan"); err != nil {
				return nil, exitcodes.NewUsageError(fmt.Errorf("vnics[%d] %w", i, err))
			} else if present {
				if vlan != math.Trunc(vlan) || vlan < math.MinInt32 || vlan > math.MaxInt32 {
					return nil, fmt.Errorf("vnics[%d] vlan must be an integer", i)
				}
				if err := validation.ValidateVLAN(int(vlan)); err != nil {
					return nil, fmt.Errorf("vnics[%d] %w", i, err)
				}
				vnic.VLAN = int(vlan)
			}

			vnics = append(vnics, vnic)
		}
		req.Vnics = vnics
	}

	if resourceTags, present, err := utils.JSONObject(jsonData, "resourceTags"); err != nil {
		return nil, exitcodes.NewUsageError(err)
	} else if present {
		tags, err := utils.TagMapFromObject(resourceTags)
		if err != nil {
			return nil, exitcodes.NewUsageError(err)
		}
		req.ResourceTags = tags
	}

	if err := validation.ValidateBuyMVERequest(req); err != nil {
		return nil, err
	}

	return req, nil
}

func processFlagBuyMVEInput(cmd *cobra.Command) (*megaport.BuyMVERequest, error) {
	// Flag read errors are intentionally ignored — flags are registered by the command builder.
	name, _ := cmd.Flags().GetString("name")
	term, _ := cmd.Flags().GetInt("term")
	locationID, _ := cmd.Flags().GetInt("location-id")
	diversityZone, _ := cmd.Flags().GetString("diversity-zone")
	promoCode, _ := cmd.Flags().GetString("promo-code")
	costCentre, _ := cmd.Flags().GetString("cost-centre")
	vendorConfigStr, _ := cmd.Flags().GetString("vendor-config")
	vnicsStr, _ := cmd.Flags().GetString("vnics")
	resourceTagsStr, _ := cmd.Flags().GetString("resource-tags")
	resourceTagsFile, _ := cmd.Flags().GetString("resource-tags-file")

	var vendorConfig megaport.VendorConfig
	if vendorConfigStr != "" {
		var vendorConfigMap map[string]interface{}
		err := json.Unmarshal([]byte(vendorConfigStr), &vendorConfigMap)
		if err != nil {
			return nil, fmt.Errorf("failed to parse vendor-config JSON string: %w", err)
		}
		vendorConfig, err = ParseVendorConfig(vendorConfigMap)
		if err != nil {
			return nil, err
		}
	}

	var vnics []megaport.MVENetworkInterface
	if vnicsStr != "" {
		var vnicsData []interface{}
		err := json.Unmarshal([]byte(vnicsStr), &vnicsData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse vnics JSON string: %w", err)
		}

		vnics = make([]megaport.MVENetworkInterface, 0, len(vnicsData))
		for i, vnicData := range vnicsData {
			vnicMap, ok := vnicData.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("vnics[%d] must be an object", i)
			}
			vnic := megaport.MVENetworkInterface{}

			if description, present, err := utils.JSONString(vnicMap, "description"); err != nil {
				return nil, fmt.Errorf("vnics[%d] %w", i, err)
			} else if present {
				vnic.Description = description
			}

			if vlan, present, err := utils.JSONNumber(vnicMap, "vlan"); err != nil {
				return nil, fmt.Errorf("vnics[%d] %w", i, err)
			} else if present {
				if vlan != math.Trunc(vlan) || vlan < math.MinInt32 || vlan > math.MaxInt32 {
					return nil, fmt.Errorf("vnics[%d] vlan must be an integer", i)
				}
				if err := validation.ValidateVLAN(int(vlan)); err != nil {
					return nil, fmt.Errorf("vnics[%d] %w", i, err)
				}
				vnic.VLAN = int(vlan)
			}

			vnics = append(vnics, vnic)
		}
	}

	resourceTags, err := utils.ParseResourceTagsFlagOrFile(resourceTagsStr, resourceTagsFile)
	if err != nil {
		return nil, exitcodes.NewUsageError(err)
	}

	req := &megaport.BuyMVERequest{
		Name:          name,
		Term:          term,
		LocationID:    locationID,
		DiversityZone: diversityZone,
		PromoCode:     promoCode,
		CostCentre:    costCentre,
		VendorConfig:  vendorConfig,
		Vnics:         vnics,
		ResourceTags:  resourceTags,
	}

	if err := validation.ValidateBuyMVERequest(req); err != nil {
		return nil, err
	}

	return req, nil
}

// ParseVendorConfig converts a vendor config map (e.g. decoded from YAML/JSON) into
// the appropriate typed VendorConfig based on the "vendor" key.
func ParseVendorConfig(vendorConfigMap map[string]interface{}) (megaport.VendorConfig, error) {
	vendor, ok := vendorConfigMap["vendor"].(string)
	if !ok {
		return nil, fmt.Errorf("vendor field is required in vendor config")
	}

	switch vendor {
	case "6wind":
		return parseSixwindConfig(vendorConfigMap)
	case "aruba":
		return parseArubaConfig(vendorConfigMap)
	case "aviatrix":
		return parseAviatrixConfig(vendorConfigMap)
	case "cisco":
		return parseCiscoConfig(vendorConfigMap)
	case "fortinet":
		return parseFortinetConfig(vendorConfigMap)
	case "palo_alto":
		return parsePaloAltoConfig(vendorConfigMap)
	case "prisma":
		return parsePrismaConfig(vendorConfigMap)
	case "versa":
		return parseVersaConfig(vendorConfigMap)
	case "vmware":
		return parseVmwareConfig(vendorConfigMap)
	case "meraki":
		return parseMerakiConfig(vendorConfigMap)
	default:
		return nil, fmt.Errorf("unsupported vendor: %s", vendor)
	}
}

func parseSixwindConfig(config map[string]interface{}) (*megaport.SixwindVSRConfig, error) {
	imageID, ok := getImageIDFromMap(config)
	if !ok {
		return nil, fmt.Errorf("imageId is required for 6WIND configuration")
	}

	productSize, ok := getStringFromMap(config, "productSize")
	if !ok {
		return nil, fmt.Errorf("productSize is required for 6WIND configuration")
	}
	productSize = validation.NormalizeMVEProductSize(strings.ToUpper(productSize))

	mveLabel, _ := getStringFromMap(config, "mveLabel")

	sshPublicKey, ok := getStringFromMap(config, "sshPublicKey")
	if !ok {
		return nil, fmt.Errorf("sshPublicKey is required for 6WIND configuration")
	}

	return &megaport.SixwindVSRConfig{
		Vendor:       "6wind",
		ImageID:      imageID,
		ProductSize:  productSize,
		MVELabel:     mveLabel,
		SSHPublicKey: sshPublicKey,
	}, nil
}

func parseArubaConfig(config map[string]interface{}) (*megaport.ArubaConfig, error) {
	imageID, ok := getImageIDFromMap(config)
	if !ok {
		return nil, fmt.Errorf("imageId is required for Aruba configuration")
	}

	productSize, ok := getStringFromMap(config, "productSize")
	if !ok {
		return nil, fmt.Errorf("productSize is required for Aruba configuration")
	}
	productSize = validation.NormalizeMVEProductSize(strings.ToUpper(productSize))

	mveLabel, _ := getStringFromMap(config, "mveLabel")

	accountName, ok := getStringFromMap(config, "accountName")
	if !ok {
		return nil, fmt.Errorf("accountName is required for Aruba configuration")
	}

	accountKey, ok := getStringFromMap(config, "accountKey")
	if !ok {
		return nil, fmt.Errorf("accountKey is required for Aruba configuration")
	}

	systemTag, _ := getStringFromMap(config, "systemTag")

	return &megaport.ArubaConfig{
		Vendor:      "aruba",
		ImageID:     imageID,
		ProductSize: productSize,
		MVELabel:    mveLabel,
		AccountName: accountName,
		AccountKey:  accountKey,
		SystemTag:   systemTag,
	}, nil
}

func parseAviatrixConfig(config map[string]interface{}) (*megaport.AviatrixConfig, error) {
	imageID, ok := getImageIDFromMap(config)
	if !ok {
		return nil, fmt.Errorf("imageId is required for Aviatrix configuration")
	}

	productSize, ok := getStringFromMap(config, "productSize")
	if !ok {
		return nil, fmt.Errorf("productSize is required for Aviatrix configuration")
	}
	productSize = validation.NormalizeMVEProductSize(strings.ToUpper(productSize))

	mveLabel, _ := getStringFromMap(config, "mveLabel")

	cloudInit, ok := getStringFromMap(config, "cloudInit")
	if !ok {
		return nil, fmt.Errorf("cloudInit is required for Aviatrix configuration")
	}

	return &megaport.AviatrixConfig{
		Vendor:      "aviatrix",
		ImageID:     imageID,
		ProductSize: productSize,
		MVELabel:    mveLabel,
		CloudInit:   cloudInit,
	}, nil
}

func parseCiscoConfig(config map[string]interface{}) (*megaport.CiscoConfig, error) {
	imageID, ok := getImageIDFromMap(config)
	if !ok {
		return nil, fmt.Errorf("imageId is required for Cisco configuration")
	}

	productSize, ok := getStringFromMap(config, "productSize")
	if !ok {
		return nil, fmt.Errorf("productSize is required for Cisco configuration")
	}
	productSize = validation.NormalizeMVEProductSize(strings.ToUpper(productSize))

	adminSSHPublicKey, _ := getStringFromMap(config, "adminSshPublicKey")
	if adminSSHPublicKey == "" {
		return nil, fmt.Errorf("adminSshPublicKey is required for Cisco configuration")
	}

	sshPublicKey, _ := getStringFromMap(config, "sshPublicKey")
	if sshPublicKey == "" {
		return nil, fmt.Errorf("sshPublicKey is required for Cisco configuration")
	}

	// FMC fields are only required for FMC-managed (non-local) deployments,
	// mirroring ValidateCiscoConfig. manageLocally is optional and defaults to
	// false, but a present-but-non-boolean value is a clear error rather than a
	// silent false that would confusingly then demand the FMC fields.
	manageLocally, ok := getBoolFromMap(config, "manageLocally")
	if _, present := config["manageLocally"]; present && !ok {
		return nil, fmt.Errorf("manageLocally must be a boolean for Cisco configuration")
	}

	fmcIPAddress, _ := getStringFromMap(config, "fmcIpAddress")
	fmcRegistrationKey, _ := getStringFromMap(config, "fmcRegistrationKey")
	fmcNatID, _ := getStringFromMap(config, "fmcNatId")
	if !manageLocally {
		if fmcIPAddress == "" {
			return nil, fmt.Errorf("fmcIpAddress is required for Cisco configuration when not managing locally")
		}
		if fmcRegistrationKey == "" {
			return nil, fmt.Errorf("fmcRegistrationKey is required for Cisco configuration when not managing locally")
		}
		if fmcNatID == "" {
			return nil, fmt.Errorf("fmcNatId is required for Cisco configuration when not managing locally")
		}
	}

	mveLabel, _ := getStringFromMap(config, "mveLabel")
	cloudInit, _ := getStringFromMap(config, "cloudInit")
	adminPassword, _ := getStringFromMap(config, "adminPassword")

	return &megaport.CiscoConfig{
		Vendor:             "cisco",
		ImageID:            imageID,
		ProductSize:        productSize,
		MVELabel:           mveLabel,
		ManageLocally:      manageLocally,
		AdminSSHPublicKey:  adminSSHPublicKey,
		SSHPublicKey:       sshPublicKey,
		AdminPassword:      adminPassword,
		CloudInit:          cloudInit,
		FMCIPAddress:       fmcIPAddress,
		FMCRegistrationKey: fmcRegistrationKey,
		FMCNatID:           fmcNatID,
	}, nil
}

func parseFortinetConfig(config map[string]interface{}) (*megaport.FortinetConfig, error) {
	imageID, ok := getImageIDFromMap(config)
	if !ok {
		return nil, fmt.Errorf("imageId is required for Fortinet configuration")
	}

	productSize, ok := getStringFromMap(config, "productSize")
	if !ok {
		return nil, fmt.Errorf("productSize is required for Fortinet configuration")
	}
	productSize = validation.NormalizeMVEProductSize(strings.ToUpper(productSize))

	mveLabel, _ := getStringFromMap(config, "mveLabel")

	adminSSHPublicKey, ok := getStringFromMap(config, "adminSshPublicKey")
	if !ok {
		return nil, fmt.Errorf("adminSshPublicKey is required for Fortinet configuration")
	}

	sshPublicKey, ok := getStringFromMap(config, "sshPublicKey")
	if !ok {
		return nil, fmt.Errorf("sshPublicKey is required for Fortinet configuration")
	}

	licenseData, ok := getStringFromMap(config, "licenseData")
	if !ok {
		return nil, fmt.Errorf("licenseData is required for Fortinet configuration")
	}

	return &megaport.FortinetConfig{
		Vendor:            "fortinet",
		ImageID:           imageID,
		ProductSize:       productSize,
		MVELabel:          mveLabel,
		AdminSSHPublicKey: adminSSHPublicKey,
		SSHPublicKey:      sshPublicKey,
		LicenseData:       licenseData,
	}, nil
}

func parsePaloAltoConfig(config map[string]interface{}) (*megaport.PaloAltoConfig, error) {
	imageID, ok := getImageIDFromMap(config)
	if !ok {
		return nil, fmt.Errorf("imageId is required for PaloAlto configuration")
	}

	productSize, ok := getStringFromMap(config, "productSize")
	if !ok {
		return nil, fmt.Errorf("productSize is required for PaloAlto configuration")
	}
	productSize = validation.NormalizeMVEProductSize(strings.ToUpper(productSize))

	mveLabel, _ := getStringFromMap(config, "mveLabel")

	sshPublicKey, ok := getStringFromMap(config, "sshPublicKey")
	if !ok {
		return nil, fmt.Errorf("sshPublicKey is required for PaloAlto configuration")
	}

	adminPasswordHash, _ := getStringFromMap(config, "adminPasswordHash")
	adminPassword, _ := getStringFromMap(config, "adminPassword")

	licenseData, ok := getStringFromMap(config, "licenseData")
	if !ok {
		return nil, fmt.Errorf("licenseData is required for PaloAlto configuration")
	}

	return &megaport.PaloAltoConfig{
		Vendor:            "palo_alto",
		ImageID:           imageID,
		ProductSize:       productSize,
		MVELabel:          mveLabel,
		SSHPublicKey:      sshPublicKey,
		AdminPasswordHash: adminPasswordHash,
		AdminPassword:     adminPassword,
		LicenseData:       licenseData,
	}, nil
}

func parsePrismaConfig(config map[string]interface{}) (*megaport.PrismaConfig, error) {
	imageID, ok := getImageIDFromMap(config)
	if !ok {
		return nil, fmt.Errorf("imageId is required for Prisma configuration")
	}

	productSize, ok := getStringFromMap(config, "productSize")
	if !ok {
		return nil, fmt.Errorf("productSize is required for Prisma configuration")
	}
	productSize = validation.NormalizeMVEProductSize(strings.ToUpper(productSize))

	mveLabel, _ := getStringFromMap(config, "mveLabel")

	ionKey, ok := getStringFromMap(config, "ionKey")
	if !ok {
		return nil, fmt.Errorf("ionKey is required for Prisma configuration")
	}

	secretKey, ok := getStringFromMap(config, "secretKey")
	if !ok {
		return nil, fmt.Errorf("secretKey is required for Prisma configuration")
	}

	return &megaport.PrismaConfig{
		Vendor:      "prisma",
		ImageID:     imageID,
		ProductSize: productSize,
		MVELabel:    mveLabel,
		IONKey:      ionKey,
		SecretKey:   secretKey,
	}, nil
}

func parseVersaConfig(config map[string]interface{}) (*megaport.VersaConfig, error) {
	imageID, ok := getImageIDFromMap(config)
	if !ok {
		return nil, fmt.Errorf("imageId is required for Versa configuration")
	}

	productSize, ok := getStringFromMap(config, "productSize")
	if !ok {
		return nil, fmt.Errorf("productSize is required for Versa configuration")
	}
	productSize = validation.NormalizeMVEProductSize(strings.ToUpper(productSize))

	mveLabel, _ := getStringFromMap(config, "mveLabel")

	directorAddress, ok := getStringFromMap(config, "directorAddress")
	if !ok {
		return nil, fmt.Errorf("directorAddress is required for Versa configuration")
	}

	controllerAddress, ok := getStringFromMap(config, "controllerAddress")
	if !ok {
		return nil, fmt.Errorf("controllerAddress is required for Versa configuration")
	}

	localAuth, ok := getStringFromMap(config, "localAuth")
	if !ok {
		return nil, fmt.Errorf("localAuth is required for Versa configuration")
	}

	remoteAuth, ok := getStringFromMap(config, "remoteAuth")
	if !ok {
		return nil, fmt.Errorf("remoteAuth is required for Versa configuration")
	}

	serialNumber, ok := getStringFromMap(config, "serialNumber")
	if !ok {
		return nil, fmt.Errorf("serialNumber is required for Versa configuration")
	}

	return &megaport.VersaConfig{
		Vendor:            "versa",
		ImageID:           imageID,
		ProductSize:       productSize,
		MVELabel:          mveLabel,
		DirectorAddress:   directorAddress,
		ControllerAddress: controllerAddress,
		LocalAuth:         localAuth,
		RemoteAuth:        remoteAuth,
		SerialNumber:      serialNumber,
	}, nil
}

func parseVmwareConfig(config map[string]interface{}) (*megaport.VmwareConfig, error) {
	imageID, ok := getImageIDFromMap(config)
	if !ok {
		return nil, fmt.Errorf("imageId is required for VMware configuration")
	}

	productSize, ok := getStringFromMap(config, "productSize")
	if !ok {
		return nil, fmt.Errorf("productSize is required for VMware configuration")
	}
	productSize = validation.NormalizeMVEProductSize(strings.ToUpper(productSize))

	mveLabel, _ := getStringFromMap(config, "mveLabel")

	adminSSHPublicKey, ok := getStringFromMap(config, "adminSshPublicKey")
	if !ok {
		return nil, fmt.Errorf("adminSshPublicKey is required for VMware configuration")
	}

	sshPublicKey, ok := getStringFromMap(config, "sshPublicKey")
	if !ok {
		return nil, fmt.Errorf("sshPublicKey is required for VMware configuration")
	}

	vcoAddress, ok := getStringFromMap(config, "vcoAddress")
	if !ok {
		return nil, fmt.Errorf("vcoAddress is required for VMware configuration")
	}

	vcoActivationCode, ok := getStringFromMap(config, "vcoActivationCode")
	if !ok {
		return nil, fmt.Errorf("vcoActivationCode is required for VMware configuration")
	}

	return &megaport.VmwareConfig{
		Vendor:            "vmware",
		ImageID:           imageID,
		ProductSize:       productSize,
		MVELabel:          mveLabel,
		AdminSSHPublicKey: adminSSHPublicKey,
		SSHPublicKey:      sshPublicKey,
		VcoAddress:        vcoAddress,
		VcoActivationCode: vcoActivationCode,
	}, nil
}

func parseMerakiConfig(config map[string]interface{}) (*megaport.MerakiConfig, error) {
	imageID, ok := getImageIDFromMap(config)
	if !ok {
		return nil, fmt.Errorf("imageId is required for Meraki configuration")
	}

	productSize, ok := getStringFromMap(config, "productSize")
	if !ok {
		return nil, fmt.Errorf("productSize is required for Meraki configuration")
	}

	productSize = validation.NormalizeMVEProductSize(strings.ToUpper(productSize))

	mveLabel, _ := getStringFromMap(config, "mveLabel")

	token, ok := getStringFromMap(config, "token")
	if !ok {
		return nil, fmt.Errorf("token is required for Meraki configuration")
	}

	return &megaport.MerakiConfig{
		Vendor:      "meraki",
		ImageID:     imageID,
		ProductSize: productSize,
		MVELabel:    mveLabel,
		Token:       token,
	}, nil
}

func getStringFromMap(m map[string]interface{}, key string) (string, bool) {
	val, ok := m[key]
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

func getImageIDFromMap(m map[string]interface{}) (int, bool) {
	val, ok := m["imageId"].(float64)
	if !ok {
		return 0, false
	}
	return int(val), true
}

func getBoolFromMap(m map[string]interface{}, key string) (bool, bool) {
	val, ok := m[key]
	if !ok {
		return false, false
	}
	b, ok := val.(bool)
	return b, ok
}

func processJSONUpdateMVEInput(jsonStr, jsonFilePath, mveUID string) (*megaport.ModifyMVERequest, bool, error) {
	var jsonData map[string]interface{}

	rawBytes, err := utils.ReadJSONInput(jsonStr, jsonFilePath)
	if err != nil {
		return nil, false, err
	}
	if err := json.Unmarshal(rawBytes, &jsonData); err != nil {
		return nil, false, fmt.Errorf("failed to parse JSON: %w", err)
	}

	req := &megaport.ModifyMVERequest{
		MVEID: mveUID,
	}

	// Gate on presence, not a non-empty value, so an explicit "name": "" is
	// rejected deterministically instead of being silently treated the same
	// as the key being absent, matching the flag path.
	if name, present, err := utils.JSONString(jsonData, "name"); err != nil {
		return nil, false, err
	} else if present {
		if name == "" {
			return nil, false, validation.NewValidationError("name", name, "cannot be empty")
		}
		req.Name = name
	}

	// A present key (even empty) is applied as given, so an explicit "" clears
	// it. When absent, the caller re-applies the current cost centre.
	costCentre, costCentreProvided, err := utils.JSONString(jsonData, "costCentre")
	if err != nil {
		return nil, false, err
	}
	if costCentreProvided {
		req.CostCentre = costCentre
	}

	if contractTermMonths, present, err := utils.JSONNumber(jsonData, "contractTermMonths"); err != nil {
		return nil, false, err
	} else if present {
		termMonths := int(contractTermMonths)
		req.ContractTermMonths = &termMonths
	}

	if rawVnics, exists := jsonData["vnics"]; exists {
		vnicsData, ok := rawVnics.([]interface{})
		if !ok {
			return nil, false, fmt.Errorf("vnics must be an array of objects with a description field")
		}
		vnics, err := parseVnicUpdates(vnicsData)
		if err != nil {
			return nil, false, err
		}
		req.Vnics = vnics
	}

	if needsUpdateMVEValidation(req, costCentreProvided) {
		if err := validation.ValidateUpdateMVERequest(req); err != nil {
			return nil, false, err
		}
	}

	return req, costCentreProvided, nil
}

// needsUpdateMVEValidation reports whether the request has fields the
// value-based ValidateUpdateMVERequest can meaningfully check. A pure
// cost-centre change (set or explicit clear) has none, and the validator can't
// tell a cleared cost centre from an absent one, so it's skipped in that case
// to avoid rejecting a legitimate clear-only update as "no fields".
func needsUpdateMVEValidation(req *megaport.ModifyMVERequest, costCentreProvided bool) bool {
	return req.Name != "" || req.ContractTermMonths != nil || len(req.Vnics) != 0 || !costCentreProvided
}

func processFlagUpdateMVEInput(cmd *cobra.Command, mveUID string) (*megaport.ModifyMVERequest, bool, error) {
	name, _ := cmd.Flags().GetString("name")
	costCentre, _ := cmd.Flags().GetString("cost-centre")
	contractTerm, _ := cmd.Flags().GetInt("term")
	vnicsStr, _ := cmd.Flags().GetString("vnics")

	req := &megaport.ModifyMVERequest{
		MVEID: mveUID,
	}

	// Gate on Changed, not a non-empty value, so an explicit --name "" is
	// rejected deterministically instead of being silently treated the same
	// as the flag not being passed at all.
	if cmd.Flags().Changed("name") {
		if name == "" {
			return nil, false, validation.NewValidationError("name", name, "cannot be empty")
		}
		req.Name = name
	}

	// Apply whatever the user passed, including an explicit empty value to
	// clear it. When the flag wasn't set, the caller re-applies the current
	// cost centre so the update doesn't wipe it.
	costCentreProvided := cmd.Flags().Changed("cost-centre")
	if costCentreProvided {
		req.CostCentre = costCentre
	}

	if cmd.Flags().Changed("term") {
		req.ContractTermMonths = &contractTerm
	}

	if cmd.Flags().Changed("vnics") {
		if strings.TrimSpace(vnicsStr) == "" {
			return nil, false, fmt.Errorf("vnics must be a non-empty JSON array of objects with a description field")
		}
		var vnicsData []interface{}
		if err := json.Unmarshal([]byte(vnicsStr), &vnicsData); err != nil {
			return nil, false, fmt.Errorf("failed to parse vnics JSON string: %w", err)
		}
		vnics, err := parseVnicUpdates(vnicsData)
		if err != nil {
			return nil, false, err
		}
		req.Vnics = vnics
	}

	if needsUpdateMVEValidation(req, costCentreProvided) {
		if err := validation.ValidateUpdateMVERequest(req); err != nil {
			return nil, false, err
		}
	}

	return req, costCentreProvided, nil
}

// parseVnicUpdates decodes a slice of {description: string} maps into
// []megaport.MVEVnicUpdate. Order is preserved — the API applies updates
// positionally to the existing vNICs. Only description can be updated;
// unknown keys are rejected so callers don't silently lose input. An
// empty array is rejected so callers don't get a misleading
// "at least one field must be provided" error.
func parseVnicUpdates(vnicsData []interface{}) ([]megaport.MVEVnicUpdate, error) {
	if len(vnicsData) == 0 {
		return nil, fmt.Errorf("vnics must contain at least one object with a description field")
	}
	vnics := make([]megaport.MVEVnicUpdate, 0, len(vnicsData))
	for i, vnicData := range vnicsData {
		vnicMap, ok := vnicData.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("vnics[%d] must be an object with a description field", i)
		}
		for k := range vnicMap {
			if k != "description" {
				return nil, fmt.Errorf("vnics[%d].%s is not supported; only description can be updated", i, k)
			}
		}
		description, ok := vnicMap["description"].(string)
		if !ok {
			return nil, fmt.Errorf("vnics[%d].description is required and must be a string", i)
		}
		description = strings.TrimSpace(description)
		if description == "" {
			return nil, fmt.Errorf("vnics[%d].description must not be empty", i)
		}
		vnics = append(vnics, megaport.MVEVnicUpdate{Description: description})
	}
	return vnics, nil
}
