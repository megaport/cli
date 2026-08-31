package validation

import (
	"fmt"
	"strings"
	"unicode/utf8"

	megaport "github.com/megaport/megaportgo"
)

var (
	ValidMVEVendors = []string{
		"6wind",
		"aruba",
		"aviatrix",
		"cisco",
		"fortinet",
		"palo_alto",
		"prisma",
		"versa",
		"vmware",
		"meraki",
	}
)

// MVELabelToProductSize maps human-readable MVE size labels (as returned by the
// images API in the availableSizes field) to their canonical programmatic names.
var MVELabelToProductSize = map[string]string{
	"MVE 2/8":   "SMALL",
	"MVE 4/16":  "MEDIUM",
	"MVE 8/32":  "LARGE",
	"MVE 12/48": "X_LARGE_12",
}

// NormalizeMVEProductSize converts an MVE product size to its canonical form.
// It accepts both programmatic names ("SMALL") and human-readable labels
// ("MVE 2/8") as returned by the list-images API.
func NormalizeMVEProductSize(size string) string {
	if canonical, ok := MVELabelToProductSize[size]; ok {
		return canonical
	}
	return size
}

// NormalizeMVEVendor converts a vendor name to the canonical lowercase form
// used by ValidMVEVendors and ParseVendorConfig's vendor dispatch.
func NormalizeMVEVendor(vendor string) string {
	return strings.ToLower(vendor)
}

// ValidateMVEProductSize validates the product size for an MVE (Megaport Virtual Edge) instance.
// This function ensures the specified size is among the allowed values for MVE deployments.
//
// Parameters:
//   - size: The MVE product size to validate (e.g., "SMALL", "MEDIUM", "LARGE", "X_LARGE_12")
//
// Validation checks:
//   - Size must be one of the predefined valid values (ValidMVEProductSizes)
//   - Size cannot be empty
//
// Returns:
//   - A ValidationError if the size is not valid
//   - nil if the validation passes
func ValidateMVEProductSize(size string) error {
	for _, validSize := range ValidMVEProductSizes {
		if size == validSize {
			return nil
		}
	}
	return NewValidationError("product size", size,
		fmt.Sprintf("must be one of: %v", ValidMVEProductSizes))
}

// ValidateBuyMVERequest validates a request to buy/provision a new MVE (Megaport Virtual Edge) instance.
// This function ensures all required parameters are present and valid for creating a new MVE.
//
// Parameters:
//   - req: The BuyMVERequest object containing all MVE provisioning parameters
//
// Validation checks:
//   - Name must be provided and cannot exceed the maximum length (MaxMVENameLength)
//   - Contract term must be valid (typically 1, 12, 24, 36, 48, or 60 months)
//   - Location ID must be a positive integer
//   - Vendor configuration must be provided and valid
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateBuyMVERequest(req *megaport.BuyMVERequest) error {
	if req.Name == "" {
		return NewValidationError("MVE name", req.Name, "cannot be empty")
	}
	if utf8.RuneCountInString(req.Name) > MaxMVENameLength {
		return NewValidationError("MVE name", req.Name, fmt.Sprintf("cannot exceed %d characters", MaxMVENameLength))
	}
	if err := ValidateContractTerm(req.Term); err != nil {
		return err
	}
	if req.LocationID <= 0 {
		return NewValidationError("location ID", req.LocationID, "must be a positive integer")
	}

	// Vendor Config is required
	if req.VendorConfig == nil {
		return NewValidationError("vendor config", req.VendorConfig, "cannot be nil")
	}

	// Validate vendor config
	if err := ValidateMVEVendorConfig(req.VendorConfig); err != nil {
		return err
	}

	return nil
}

// ValidateUpdateMVERequest validates a request to update an existing MVE (Megaport Virtual Edge) instance.
// This function ensures that necessary fields are provided to modify an MVE.
//
// Parameters:
//   - req: The ModifyMVERequest object containing the fields to update
//
// Validation checks:
//   - At least one updateable field must be provided (name, cost center, contract term, or vNICs)
//   - If a name is provided, it cannot exceed the maximum length (MaxMVENameLength)
//   - If contract term is provided, it must be valid (typically 1, 12, 24, 36, 48, or 60 months)
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateUpdateMVERequest(req *megaport.ModifyMVERequest) error {
	// This field-count check is value-based, so it can't tell an explicitly
	// cleared cost centre ("") from an absent one. Callers that must allow a
	// clear-only update (see processFlagUpdateMVEInput) skip this validator for
	// that case; any new caller has to do the same.
	if req.Name == "" && req.CostCentre == "" && req.ContractTermMonths == nil && len(req.Vnics) == 0 {
		return NewValidationError("update request", req, "at least one field must be provided for update")
	}

	if req.Name != "" && utf8.RuneCountInString(req.Name) > MaxMVENameLength {
		return NewValidationError("MVE name", req.Name, fmt.Sprintf("cannot exceed %d characters", MaxMVENameLength))
	}

	// If contract term is provided, validate it
	if req.ContractTermMonths != nil {
		term := *req.ContractTermMonths
		err := ValidateContractTerm(term)
		if err != nil {
			return err
		}
	}

	for i, v := range req.Vnics {
		if v.Description == "" {
			return NewValidationError(fmt.Sprintf("vnics[%d].description", i), v.Description, "must not be empty")
		}
	}

	return nil
}

// ValidateMVERequest validates the core parameters for creating an MVE (Megaport Virtual Edge).
// This function is used by other validators to validate common MVE parameters.
//
// Parameters:
//   - name: The name to give the MVE instance
//   - term: The contract term in months
//   - locationID: The ID of the Megaport location where the MVE will be deployed
//
// Validation checks:
//   - Name cannot be empty
//   - Name cannot exceed the maximum length (MaxMVENameLength)
//   - Contract term must be valid (typically 1, 12, 24, 36, 48, or 60 months)
//   - Location ID must be a positive integer
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateMVERequest(name string, term int, locationID int) error {
	if name == "" {
		return NewValidationError("MVE name", name, "cannot be empty")
	}
	if utf8.RuneCountInString(name) > MaxMVENameLength {
		return NewValidationError("MVE name", name, fmt.Sprintf("cannot exceed %d characters", MaxMVENameLength))
	}
	if err := ValidateContractTerm(term); err != nil {
		return err
	}
	if locationID <= 0 {
		return NewValidationError("location ID", locationID, "must be a positive integer")
	}
	return nil
}

// ValidateMVEVendor validates that a vendor name is one of the supported MVE vendors.
// This ensures that only supported network virtualization platforms can be deployed.
//
// Parameters:
//   - vendor: The name of the vendor/platform (e.g., "cisco", "palo_alto", "vmware")
//
// Validation checks:
//   - Vendor name must be one of the predefined values in ValidMVEVendors
//   - Vendor name is case-insensitive (normalized to lowercase for comparison)
//
// Returns:
//   - A ValidationError if the vendor is not supported
//   - nil if the validation passes
func ValidateMVEVendor(vendor string) error {
	normalizedVendor := NormalizeMVEVendor(vendor)
	for _, validVendor := range ValidMVEVendors {
		if normalizedVendor == validVendor {
			return nil
		}
	}
	return NewValidationError("MVE vendor", vendor,
		fmt.Sprintf("must be one of: %v", ValidMVEVendors))
}

// ValidateMVENetworkInterfaces validates the network interfaces for an MVE instance.
// This function ensures the virtual network interfaces meet Megaport's requirements.
//
// Parameters:
//   - vnics: A slice of MVENetworkInterface objects representing the virtual NICs to configure
//
// Validation checks:
//   - Cannot have more than 5 vNICs per MVE instance
//   - Each vNIC must have a non-empty description
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateMVENetworkInterfaces(vnics []megaport.MVENetworkInterface) error {
	if len(vnics) > 5 {
		return NewValidationError("network interfaces", len(vnics), "cannot exceed 5 vNICs")
	}
	for i, vnic := range vnics {
		if vnic.Description == "" {
			return NewValidationError(fmt.Sprintf("network interface %d", i+1), vnic.Description, "description cannot be empty")
		}
	}
	return nil
}

// ValidateSixwindVSRConfig validates a 6WIND VSR configuration for an MVE deployment.
// This function ensures all required parameters for a 6WIND virtual router are provided.
//
// Parameters:
//   - config: The 6WIND VSR configuration to validate
//
// Validation checks:
//   - Image ID must be a positive integer
//   - Product size must be valid (calls ValidateMVEProductSize)
//   - SSH public key must be provided (for authentication)
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateSixwindVSRConfig(config *megaport.SixwindVSRConfig) error {
	if config.ImageID <= 0 {
		return NewValidationError("image ID", config.ImageID, "must be a positive integer")
	}
	if err := ValidateMVEProductSize(config.ProductSize); err != nil {
		return err
	}
	if config.SSHPublicKey == "" {
		return NewValidationError("SSH public key", config.SSHPublicKey, "cannot be empty")
	}
	return nil
}

// ValidateArubaConfig validates an Aruba configuration for an MVE deployment.
// This function ensures all required parameters for an Aruba virtual appliance are provided.
//
// Parameters:
//   - config: The Aruba configuration to validate
//
// Validation checks:
//   - Image ID must be a positive integer
//   - Product size must be valid (calls ValidateMVEProductSize)
//   - Account name must be provided
//   - Account key must be provided
//   - System tag must be provided
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateArubaConfig(config *megaport.ArubaConfig) error {
	if config.ImageID <= 0 {
		return NewValidationError("image ID", config.ImageID, "must be a positive integer")
	}
	if err := ValidateMVEProductSize(config.ProductSize); err != nil {
		return err
	}
	if config.AccountName == "" {
		return NewValidationError("account name", config.AccountName, "cannot be empty")
	}
	if config.AccountKey == "" {
		return NewValidationError("account key", config.AccountKey, "cannot be empty")
	}
	if config.SystemTag == "" {
		return NewValidationError("system tag", config.SystemTag, "cannot be empty")
	}
	return nil
}

// ValidateAviatrixConfig validates an Aviatrix configuration for an MVE deployment.
// This function ensures all required parameters for an Aviatrix virtual appliance are provided.
//
// Parameters:
//   - config: The Aviatrix configuration to validate
//
// Validation checks:
//   - Image ID must be a positive integer
//   - Product size must be valid (calls ValidateMVEProductSize)
//   - Cloud init data must be provided (for initial configuration)
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateAviatrixConfig(config *megaport.AviatrixConfig) error {
	if config.ImageID <= 0 {
		return NewValidationError("image ID", config.ImageID, "must be a positive integer")
	}
	if err := ValidateMVEProductSize(config.ProductSize); err != nil {
		return err
	}
	if config.CloudInit == "" {
		return NewValidationError("cloud init", config.CloudInit, "cannot be empty")
	}
	return nil
}

// ValidateCiscoConfig validates a Cisco configuration for an MVE deployment.
// This function ensures all required parameters for a Cisco virtual appliance are provided.
//
// Parameters:
//   - config: The Cisco configuration to validate
//
// Validation checks:
//   - Image ID must be a positive integer
//   - Product size must be valid (calls ValidateMVEProductSize)
//   - Admin SSH public key must be provided
//   - SSH public key must be provided
//   - If not managing locally (FMC management):
//   - FMC IP address must be provided
//   - FMC registration key must be provided
//   - FMC NAT ID must be provided
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateCiscoConfig(config *megaport.CiscoConfig) error {
	if config.ImageID <= 0 {
		return NewValidationError("image ID", config.ImageID, "must be a positive integer")
	}
	if err := ValidateMVEProductSize(config.ProductSize); err != nil {
		return err
	}
	if config.AdminSSHPublicKey == "" {
		return NewValidationError("admin SSH public key", config.AdminSSHPublicKey, "cannot be empty")
	}
	if config.SSHPublicKey == "" {
		return NewValidationError("SSH public key", config.SSHPublicKey, "cannot be empty")
	}
	if !config.ManageLocally {
		if config.FMCIPAddress == "" {
			return NewValidationError("FMC IP address", config.FMCIPAddress, "cannot be empty when not managing locally")
		}
		if config.FMCRegistrationKey == "" {
			return NewValidationError("FMC registration key", config.FMCRegistrationKey, "cannot be empty when not managing locally")
		}
		if config.FMCNatID == "" {
			return NewValidationError("FMC NAT ID", config.FMCNatID, "cannot be empty when not managing locally")
		}
	}
	return nil
}

// ValidateVmwareConfig validates a VMware SD-WAN configuration for an MVE deployment.
// This function ensures all required parameters for a VMware SD-WAN virtual appliance are provided.
//
// Parameters:
//   - config: The VMware configuration to validate
//
// Validation checks:
//   - Image ID must be a positive integer
//   - Product size must be valid (calls ValidateMVEProductSize)
//   - Admin SSH public key must be provided
//   - SSH public key must be provided
//   - VCO address must be provided (VMware SD-WAN orchestrator address)
//   - VCO activation code must be provided
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateVmwareConfig(config *megaport.VmwareConfig) error {
	if config.ImageID <= 0 {
		return NewValidationError("image ID", config.ImageID, "must be a positive integer")
	}
	if err := ValidateMVEProductSize(config.ProductSize); err != nil {
		return err
	}
	if config.AdminSSHPublicKey == "" {
		return NewValidationError("admin SSH public key", config.AdminSSHPublicKey, "cannot be empty")
	}
	if config.SSHPublicKey == "" {
		return NewValidationError("SSH public key", config.SSHPublicKey, "cannot be empty")
	}
	if config.VcoAddress == "" {
		return NewValidationError("VCO address", config.VcoAddress, "cannot be empty")
	}
	if config.VcoActivationCode == "" {
		return NewValidationError("VCO activation code", config.VcoActivationCode, "cannot be empty")
	}
	return nil
}

// ValidateMerakiConfig validates a Cisco Meraki configuration for an MVE deployment.
// This function ensures all required parameters for a Meraki virtual appliance are provided.
//
// Parameters:
//   - config: The Meraki configuration to validate
//
// Validation checks:
//   - Image ID must be a positive integer
//   - Product size must be valid (calls ValidateMVEProductSize)
//   - Authentication token must be provided
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateMerakiConfig(config *megaport.MerakiConfig) error {
	if config.ImageID <= 0 {
		return NewValidationError("image ID", config.ImageID, "must be a positive integer")
	}
	if err := ValidateMVEProductSize(config.ProductSize); err != nil {
		return err
	}
	if config.Token == "" {
		return NewValidationError("token", config.Token, "cannot be empty")
	}
	return nil
}

// ValidateFortinetConfig validates a Fortinet configuration for an MVE deployment.
// This function ensures all required parameters for a Fortinet virtual appliance are provided.
//
// Parameters:
//   - config: The Fortinet configuration to validate
//
// Validation checks:
//   - Image ID must be a positive integer
//   - Product size must be valid (calls ValidateMVEProductSize)
//   - Admin SSH public key must be provided (for administrator access)
//   - SSH public key must be provided (for regular user access)
//   - License data must be provided (for Fortinet licensing)
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateFortinetConfig(config *megaport.FortinetConfig) error {
	if config.ImageID <= 0 {
		return NewValidationError("image ID", config.ImageID, "must be a positive integer")
	}
	if err := ValidateMVEProductSize(config.ProductSize); err != nil {
		return err
	}
	if config.AdminSSHPublicKey == "" {
		return NewValidationError("admin SSH public key", config.AdminSSHPublicKey, "cannot be empty")
	}
	if config.SSHPublicKey == "" {
		return NewValidationError("SSH public key", config.SSHPublicKey, "cannot be empty")
	}
	if config.LicenseData == "" {
		return NewValidationError("license data", config.LicenseData, "cannot be empty")
	}
	return nil
}

// ValidatePaloAltoConfig validates a Palo Alto Networks configuration for an MVE deployment.
// This function ensures all required parameters for a Palo Alto Networks virtual appliance are provided.
//
// Parameters:
//   - config: The Palo Alto Networks configuration to validate
//
// Validation checks:
//   - Image ID must be a positive integer
//   - Product size must be valid (calls ValidateMVEProductSize)
//   - SSH public key must be provided (for user access)
//   - Exactly one of admin password or admin password hash must be provided
//     (for secure administrator authentication)
//   - License data must be provided (for Palo Alto Networks licensing)
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidatePaloAltoConfig(config *megaport.PaloAltoConfig) error {
	if config.ImageID <= 0 {
		return NewValidationError("image ID", config.ImageID, "must be a positive integer")
	}
	if err := ValidateMVEProductSize(config.ProductSize); err != nil {
		return err
	}
	if config.SSHPublicKey == "" {
		return NewValidationError("SSH public key", config.SSHPublicKey, "cannot be empty")
	}
	if config.AdminPassword == "" && config.AdminPasswordHash == "" {
		return NewValidationError("admin password / admin password hash", "", "at least one of admin password or admin password hash must be provided")
	}
	if config.AdminPassword != "" && config.AdminPasswordHash != "" {
		return NewValidationError("admin password / admin password hash", "", "only one of admin password or admin password hash may be provided, not both")
	}
	if config.LicenseData == "" {
		return NewValidationError("license data", config.LicenseData, "cannot be empty")
	}
	return nil
}

// ValidatePrismaConfig validates a Prisma SD-WAN configuration for an MVE deployment.
// This function ensures all required parameters for a Prisma SD-WAN virtual appliance are provided.
//
// Parameters:
//   - config: The Prisma configuration to validate
//
// Validation checks:
//   - Image ID must be a positive integer
//   - Product size must be valid (calls ValidateMVEProductSize)
//   - ION key must be provided (required for Prisma SD-WAN activation)
//   - Secret key must be provided (required for Prisma SD-WAN authentication)
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidatePrismaConfig(config *megaport.PrismaConfig) error {
	if config.ImageID <= 0 {
		return NewValidationError("image ID", config.ImageID, "must be a positive integer")
	}
	if err := ValidateMVEProductSize(config.ProductSize); err != nil {
		return err
	}
	if config.IONKey == "" {
		return NewValidationError("ION key", config.IONKey, "cannot be empty")
	}
	if config.SecretKey == "" {
		return NewValidationError("secret key", config.SecretKey, "cannot be empty")
	}
	return nil
}

// ValidateVersaConfig validates a Versa Networks configuration for an MVE deployment.
// This function ensures all required parameters for a Versa Networks virtual appliance are provided.
//
// Parameters:
//   - config: The Versa Networks configuration to validate
//
// Validation checks:
//   - Image ID must be a positive integer
//   - Product size must be valid (calls ValidateMVEProductSize)
//   - Director address must be provided (Versa Director management plane)
//   - Controller address must be provided (Versa Controller address)
//   - Local auth credential must be provided
//   - Remote auth credential must be provided
//   - Serial number must be provided (for device registration)
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateVersaConfig(config *megaport.VersaConfig) error {
	if config.ImageID <= 0 {
		return NewValidationError("image ID", config.ImageID, "must be a positive integer")
	}
	if err := ValidateMVEProductSize(config.ProductSize); err != nil {
		return err
	}
	if config.DirectorAddress == "" {
		return NewValidationError("director address", config.DirectorAddress, "cannot be empty")
	}
	if config.ControllerAddress == "" {
		return NewValidationError("controller address", config.ControllerAddress, "cannot be empty")
	}
	if config.LocalAuth == "" {
		return NewValidationError("local auth", config.LocalAuth, "cannot be empty")
	}
	if config.RemoteAuth == "" {
		return NewValidationError("remote auth", config.RemoteAuth, "cannot be empty")
	}
	if config.SerialNumber == "" {
		return NewValidationError("serial number", config.SerialNumber, "cannot be empty")
	}
	return nil
}

// ValidateMVEVendorConfig validates vendor-specific configurations for an MVE deployment.
// This function acts as a dispatcher that routes validation to the appropriate vendor-specific
// validation function based on the concrete type of the configuration.
//
// Parameters:
//   - config: The vendor configuration to validate (an interface that can be any vendor-specific type)
//
// Validation checks:
//   - Configuration cannot be nil
//   - The concrete type must be one of the supported vendor configuration types
//   - Delegates to vendor-specific validation functions:
//   - ValidateSixwindVSRConfig
//   - ValidateArubaConfig
//   - ValidateAviatrixConfig
//   - ValidateCiscoConfig
//   - ValidateFortinetConfig
//   - ValidatePaloAltoConfig
//   - ValidatePrismaConfig
//   - ValidateVersaConfig
//   - ValidateVmwareConfig
//   - ValidateMerakiConfig
//
// Returns:
//   - A ValidationError if the configuration type is not supported or vendor-specific validation fails
//   - nil if all validation checks pass
func ValidateMVEVendorConfig(config megaport.VendorConfig) error {
	if config == nil {
		return NewValidationError("vendor config", nil, "cannot be nil")
	}
	switch cfg := config.(type) {
	case *megaport.SixwindVSRConfig:
		return ValidateSixwindVSRConfig(cfg)
	case *megaport.ArubaConfig:
		return ValidateArubaConfig(cfg)
	case *megaport.AviatrixConfig:
		return ValidateAviatrixConfig(cfg)
	case *megaport.CiscoConfig:
		return ValidateCiscoConfig(cfg)
	case *megaport.FortinetConfig:
		return ValidateFortinetConfig(cfg)
	case *megaport.PaloAltoConfig:
		return ValidatePaloAltoConfig(cfg)
	case *megaport.PrismaConfig:
		return ValidatePrismaConfig(cfg)
	case *megaport.VersaConfig:
		return ValidateVersaConfig(cfg)
	case *megaport.VmwareConfig:
		return ValidateVmwareConfig(cfg)
	case *megaport.MerakiConfig:
		return ValidateMerakiConfig(cfg)
	default:
		return NewValidationError("vendor config", config, fmt.Sprintf("unknown vendor type %T", config))
	}
}
