package validation

import (
	"fmt"
	"net"
	"strings"
)

// IsValidationError checks if an error is an instance of ValidationError.
// Returns true if the error is a ValidationError, false otherwise.
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

// ValidateIntRange validates that an integer value falls within the specified range.
// Returns a ValidationError if the value is outside the range, nil otherwise.
func ValidateIntRange(value int, minValue int, maxValue int, fieldName string) error {
	if value < minValue || value > maxValue {
		return NewValidationError(fieldName, value,
			fmt.Sprintf("must be between %d-%d", minValue, maxValue))
	}
	return nil
}

// ValidateStringOneOf validates that a string value is one of the allowed values.
// Returns a ValidationError if the string is empty or not in the list of valid values.
func ValidateStringOneOf(value string, validValues []string, fieldName string) error {
	if value == "" {
		return NewValidationError(fieldName, value, "cannot be empty")
	}
	for _, validValue := range validValues {
		if value == validValue {
			return nil
		}
	}
	return NewValidationError(fieldName, value,
		fmt.Sprintf("must be one of: %v", validValues))
}

// ValidateIPv4 validates that a string is a valid IPv4 address.
// Returns a ValidationError if the string is empty or not a valid IPv4 address.
func ValidateIPv4(ip string, fieldName string) error {
	if ip == "" {
		return NewValidationError(fieldName, ip, "cannot be empty")
	}
	// net.ParseIP(...).To4() also succeeds for IPv4-mapped IPv6 forms
	// (e.g. "::ffff:1.2.3.4"); reject anything carrying a colon.
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil || parsedIP.To4() == nil || strings.Contains(ip, ":") {
		return NewValidationError(fieldName, ip, "must be a valid IPv4 address")
	}
	return nil
}

// ValidateIPAddress validates that a string is a valid IPv4 or IPv6 address.
// Returns a ValidationError if the string is empty or not a valid IP address.
func ValidateIPAddress(ip string, fieldName string) error {
	if ip == "" {
		return NewValidationError(fieldName, ip, "cannot be empty")
	}
	if net.ParseIP(ip) == nil {
		return NewValidationError(fieldName, ip, "must be a valid IPv4 or IPv6 address")
	}
	return nil
}

// ValidateCIDR validates that a string is in valid IPv4 CIDR notation.
// Returns a ValidationError if the string is empty or not a valid IPv4 CIDR.
func ValidateCIDR(cidr string, fieldName string) error {
	if cidr == "" {
		return NewValidationError(fieldName, cidr, "cannot be empty")
	}
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil || ip.To4() == nil || strings.Contains(cidr, ":") {
		return NewValidationError(fieldName, cidr, "must be a valid IPv4 CIDR notation")
	}
	return nil
}
