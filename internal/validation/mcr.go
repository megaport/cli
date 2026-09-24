package validation

import (
	"fmt"
	"net"
	"strings"

	megaport "github.com/megaport/megaportgo"
)

// validIPSecTunnelCounts lists the allowed non-zero IPSec tunnel counts.
// Defined locally so the validation package does not depend on an SDK symbol
// that may not be present in all SDK branches (e.g. during workspace development).
var validIPSecTunnelCounts = []int{10, 20, 30}

// ValidateIPSecTunnelCount validates a tunnel count for an IPSec add-on.
// Valid non-zero values are always 10, 20, or 30.
// When allowZeroDisable is true (update mode), 0 is also accepted to disable IPSec.
// When allowZeroDisable is false (add mode), 0 is rejected; callers that wish to
// use the API default should skip calling this function when count is 0.
func ValidateIPSecTunnelCount(count int, allowZeroDisable bool) error {
	if count == 0 && allowZeroDisable {
		return nil
	}
	for _, valid := range validIPSecTunnelCounts {
		if count == valid {
			return nil
		}
	}
	counts := make([]string, len(validIPSecTunnelCounts))
	for i, v := range validIPSecTunnelCounts {
		counts[i] = fmt.Sprintf("%d", v)
	}
	validStr := strings.Join(counts, ", ")
	if allowZeroDisable {
		return fmt.Errorf("invalid IPSec tunnel count %d: must be %s, or 0 to disable", count, validStr)
	}
	return fmt.Errorf("invalid IPSec tunnel count %d: must be %s (0 uses the API default of 10)", count, validStr)
}

// ValidateMCRASN validates an explicit BGP ASN for an MCR. The argument is taken
// as int64 so a full 32-bit ASN is compared safely regardless of platform int width.
// MinASN/MaxASN (shared in common.go) bound it to the valid 32-bit range; the check
// only rejects out-of-range values, leaving assignment policy to the API.
func ValidateMCRASN(asn int64) error {
	if asn < MinASN || asn > MaxASN {
		return NewValidationError("MCR ASN", asn, fmt.Sprintf("must be between %d and %d", MinASN, MaxASN))
	}
	return nil
}

// ValidateMCRRequest validates a request to buy/provision a new MCR (Megaport Cloud Router).
// This function ensures all parameters meet the requirements for creating a new MCR.
//
// Parameters:
//   - req: The BuyMCRRequest object containing all MCR provisioning parameters
//
// Validation checks:
//   - Name cannot be empty
//   - Contract term must be valid (typically 1, 12, 24, 36, 48, or 60 months)
//   - Port speed must be one of the valid MCR port speeds
//   - Location ID must be a positive integer
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateMCRRequest(req *megaport.BuyMCRRequest) error {
	if req.Name == "" {
		return NewValidationError("MCR name", req.Name, "cannot be empty")
	}
	if err := ValidateContractTerm(req.Term); err != nil {
		return err
	}
	if err := ValidateMCRPortSpeed(req.PortSpeed); err != nil {
		return err
	}
	if req.LocationID <= 0 {
		return NewValidationError("location ID", req.LocationID, "must be a positive integer")
	}
	// ASN is optional on buy; 0 means "let the API assign a private ASN".
	if req.MCRAsn != 0 {
		if err := ValidateMCRASN(int64(req.MCRAsn)); err != nil {
			return err
		}
	}
	return nil
}

// ValidatePrefixFilterListRequest validates a request to create a new prefix filter list for an MCR.
// Prefix filter lists are used to control route advertisements in BGP sessions on MCRs.
//
// Parameters:
//   - req: The CreateMCRPrefixFilterListRequest object containing the prefix filter list definition
//
// Validation checks:
//   - Description cannot be empty
//   - Address family must be provided ("IPv4" or "IPv6")
//   - Address family must be a valid value ("IPv4" or "IPv6")
//   - At least one entry must be provided in the prefix filter list
//   - For each entry:
//   - Prefix cannot be empty
//   - Prefix must be a valid CIDR consistent with the list's address family
//   - Action must be "permit" or "deny"
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidatePrefixFilterListRequest(req *megaport.CreateMCRPrefixFilterListRequest) error {
	if req.PrefixFilterList.Description == "" {
		return NewValidationError("description", req.PrefixFilterList.Description, "cannot be empty")
	}
	if req.PrefixFilterList.AddressFamily == "" {
		return NewValidationError("address family", req.PrefixFilterList.AddressFamily, "cannot be empty")
	}
	if req.PrefixFilterList.AddressFamily != "IPv4" && req.PrefixFilterList.AddressFamily != "IPv6" {
		return NewValidationError("address family", req.PrefixFilterList.AddressFamily, "must be IPv4 or IPv6")
	}
	if len(req.PrefixFilterList.Entries) == 0 {
		return NewValidationError("entries", req.PrefixFilterList.Entries, "must contain at least one entry")
	}

	return validatePrefixFilterEntries(req.PrefixFilterList.Entries, req.PrefixFilterList.AddressFamily)
}

// validatePrefixFilterEntries validates each prefix filter entry's prefix as a
// CIDR consistent with the list's declared address family, that the action is
// permit or deny, and that any GE/LE bounds fit the family's prefix length
// (GE/LE are optional; 0 means unset, matching their omitempty JSON encoding).
func validatePrefixFilterEntries(entries []*megaport.MCRPrefixListEntry, addressFamily string) error {
	var maxPrefixLen int
	switch addressFamily {
	case "IPv4":
		maxPrefixLen = 32
	case "IPv6":
		maxPrefixLen = 128
	default:
		return NewValidationError("address family", addressFamily, "must be IPv4 or IPv6")
	}
	for i, entry := range entries {
		if entry == nil {
			return NewValidationError(fmt.Sprintf("entry index %d", i), nil, "entry cannot be nil")
		}
		if entry.Prefix == "" {
			return NewValidationError(fmt.Sprintf("entry prefix index %d", i), entry.Prefix, "prefix cannot be empty")
		}
		if addressFamily == "IPv4" {
			if err := ValidateCIDR(entry.Prefix, fmt.Sprintf("entry prefix index %d", i)); err != nil {
				return err
			}
		} else {
			ip, _, err := net.ParseCIDR(entry.Prefix)
			if err != nil || ip.To4() != nil {
				return NewValidationError(fmt.Sprintf("entry prefix index %d", i), entry.Prefix, "must be a valid IPv6 CIDR notation")
			}
		}
		if entry.Action != "permit" && entry.Action != "deny" {
			return NewValidationError("entry action", entry.Action, "must be permit or deny")
		}
		if entry.Ge != 0 && (entry.Ge < 0 || entry.Ge > maxPrefixLen) {
			return NewValidationError(fmt.Sprintf("entry GE index %d", i), entry.Ge, fmt.Sprintf("must be between 0 and %d for %s", maxPrefixLen, addressFamily))
		}
		if entry.Le != 0 && (entry.Le < 0 || entry.Le > maxPrefixLen) {
			return NewValidationError(fmt.Sprintf("entry LE index %d", i), entry.Le, fmt.Sprintf("must be between 0 and %d for %s", maxPrefixLen, addressFamily))
		}
		if entry.Ge != 0 && entry.Le != 0 && entry.Ge > entry.Le {
			return NewValidationError(fmt.Sprintf("entry GE index %d", i), entry.Ge, "must not exceed the LE value")
		}
	}
	return nil
}

// ValidateUpdatePrefixFilterList validates a request to update an existing prefix filter list for an MCR.
// This function ensures that the updated prefix filter list entries meet the requirements.
//
// Parameters:
//   - prefixFilterList: The MCRPrefixFilterList object containing the updated prefix filter list
//
// Validation checks:
//   - If entries are provided:
//   - Address family must be provided and a valid value ("IPv4" or "IPv6")
//   - For each entry:
//   - Prefix cannot be empty
//   - Prefix must be a valid CIDR consistent with the list's address family
//   - Action must be "permit" or "deny"
//
// Returns:
//   - A ValidationError if any validation check fails
//   - nil if all validation checks pass
func ValidateUpdatePrefixFilterList(prefixFilterList *megaport.MCRPrefixFilterList) error {
	if len(prefixFilterList.Entries) == 0 {
		return nil
	}
	if prefixFilterList.AddressFamily == "" {
		return NewValidationError("address family", prefixFilterList.AddressFamily, "cannot be empty")
	}
	if prefixFilterList.AddressFamily != "IPv4" && prefixFilterList.AddressFamily != "IPv6" {
		return NewValidationError("address family", prefixFilterList.AddressFamily, "must be IPv4 or IPv6")
	}
	return validatePrefixFilterEntries(prefixFilterList.Entries, prefixFilterList.AddressFamily)
}
