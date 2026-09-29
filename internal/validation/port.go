package validation

import (
	"fmt"
	"slices"
	"unicode/utf8"

	megaport "github.com/megaport/megaportgo"
)

// Define constants for LAG validation
var (
	ValidLAGPortSpeeds = []int{10000, 100000}
	MinLAGCount        = 1
	MaxLAGCount        = 8
)

// ValidatePortVLANAvailability validates if a VLAN ID is within the range
// typically available for user assignment on a Port (excluding special/reserved values).
//
// Parameters:
//   - vlan: The VLAN ID to validate (typically 2-4093 for user-assignable VLANs)
//
// Validation checks:
//   - VLAN must be within the assignable range (MinAssignableVLAN to MaxAssignableVLAN)
//   - Special values like AutoAssignVLAN, UntaggedVLAN, and ReservedVLAN are not valid for availability checks
//
// Returns:
//   - A ValidationError if the VLAN ID is not within the assignable range
//   - nil if the validation passes
func ValidatePortVLANAvailability(vlan int) error {
	// This function specifically checks the range for VLANs that can be assigned,
	// excluding special values like AutoAssign (-1) or Untagged (0), and potentially
	// reserved values like 1 or 4094/4095 depending on the platform.
	if vlan < MinAssignableVLAN || vlan > MaxAssignableVLAN {
		return NewValidationError("VLAN ID", vlan, fmt.Sprintf("must be between %d-%d for VLAN availability check", MinAssignableVLAN, MaxAssignableVLAN))
	}
	return nil
}

// ValidatePortName validates a port name for a Megaport port.
// This function ensures the port name meets Megaport's requirements.
//
// Parameters:
//   - name: The port name to validate
//
// Validation checks:
//   - Name cannot be empty
//   - Name cannot exceed the maximum length (MaxPortNameLength)
//
// Returns:
//   - A ValidationError if the port name is not valid
//   - nil if the validation passes
func ValidatePortName(name string) error {
	if name == "" {
		return NewValidationError("port name", name, "cannot be empty")
	}

	// The spec says names can be up to MaxPortNameLength characters (inclusive)
	if utf8.RuneCountInString(name) > MaxPortNameLength {
		return NewValidationError("port name", name, fmt.Sprintf("cannot exceed %d characters", MaxPortNameLength))
	}

	return nil
}

// ValidatePortRequest validates a standard port buy request.
// This function ensures all parameters meet Megaport's requirements for provisioning a new port.
//
// Parameters:
//   - req: The BuyPortRequest object containing all port provisioning parameters
//
// Validation checks:
//   - Port name cannot be empty
//   - Port name cannot exceed the maximum length (MaxPortNameLength)
//   - Location ID must be a positive integer
//   - Port speed must be one of the valid port speeds (typically 1000, 10000, or 100000 Mbps)
//   - Contract term must be valid (typically 1, 12, 24, 36, 48, or 60 months)
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidatePortRequest(req *megaport.BuyPortRequest) error {
	if req.Name == "" {
		return NewValidationError("port name", req.Name, "cannot be empty")
	}
	if utf8.RuneCountInString(req.Name) > MaxPortNameLength {
		return NewValidationError("port name", req.Name, fmt.Sprintf("cannot exceed %d characters", MaxPortNameLength))
	}
	if req.LocationId <= 0 {
		return NewValidationError("location ID", req.LocationId, "must be a positive integer")
	}
	if err := ValidatePortSpeed(req.PortSpeed); err != nil {
		return err
	}
	if err := ValidateContractTerm(req.Term); err != nil {
		return err
	}
	return nil
}

// ValidateLAGPortRequest validates a LAG (Link Aggregation Group) port buy request.
// This function ensures all parameters meet Megaport's requirements for provisioning a new LAG port.
//
// Parameters:
//   - req: The BuyPortRequest object containing LAG port provisioning parameters
//
// Validation checks:
//   - Port name cannot be empty
//   - Port name cannot exceed the maximum length (MaxPortNameLength)
//   - Location ID must be a positive integer
//   - Port speed must be one of the valid LAG port speeds (typically 10000 or 100000 Mbps)
//   - LAG count must be between the minimum and maximum allowed values (typically 1-8)
//   - Contract term must be valid (typically 1, 12, 24, 36, 48, or 60 months)
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateLAGPortRequest(req *megaport.BuyPortRequest) error {
	if req.Name == "" {
		return NewValidationError("port name", req.Name, "cannot be empty")
	}
	if utf8.RuneCountInString(req.Name) > MaxPortNameLength {
		return NewValidationError("port name", req.Name, fmt.Sprintf("cannot exceed %d characters", MaxPortNameLength))
	}
	if req.LocationId <= 0 {
		return NewValidationError("location ID", req.LocationId, "must be a positive integer")
	}
	// Use the defined constant for LAG port speeds with specific formatting
	if !slices.Contains(ValidLAGPortSpeeds, req.PortSpeed) {
		return NewValidationError("port speed", req.PortSpeed, fmt.Sprintf("must be one of: %v for LAG ports", ValidLAGPortSpeeds))
	}
	// Use the defined constants for LAG count
	if req.LagCount < MinLAGCount || req.LagCount > MaxLAGCount {
		return NewValidationError("LAG count", req.LagCount, fmt.Sprintf("must be between %d and %d", MinLAGCount, MaxLAGCount))
	}
	if err := ValidateContractTerm(req.Term); err != nil {
		return err
	}
	return nil
}
