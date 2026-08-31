package mcr

import (
	"context"
	"fmt"
	"testing"

	"github.com/megaport/megaport-cli/internal/utils"
	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockPromptSequence(responses []string) func(string, string, bool) (string, error) {
	idx := 0
	return func(resourceType, msg string, noColor bool) (string, error) {
		if idx >= len(responses) {
			return "", fmt.Errorf("unexpected prompt call #%d", idx)
		}
		val := responses[idx]
		idx++
		return val, nil
	}
}

func TestPromptForMCRDetails_Success(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	originalTagsPrompt := utils.GetResourceTagsPrompt()
	defer func() {
		utils.SetResourcePrompt(originalPrompt)
		utils.SetResourceTagsPrompt(originalTagsPrompt)
	}()

	// name, term, portSpeed, locationID, marketplaceVisibility, asn, diversityZone, costCentre, promoCode
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"Test MCR", "12", "5000", "1", "true", "65000", "blue", "IT-2024", "PROMO",
	}))
	utils.SetResourceTagsPrompt(func(noColor bool) (map[string]string, error) {
		return map[string]string{"env": "test"}, nil
	})

	req, err := promptForMCRDetails(true)
	assert.NoError(t, err)
	assert.Equal(t, "Test MCR", req.Name)
	assert.Equal(t, 12, req.Term)
	assert.Equal(t, 5000, req.PortSpeed)
	assert.Equal(t, 1, req.LocationID)
	require.NotNil(t, req.MarketplaceVisibility)
	assert.True(t, *req.MarketplaceVisibility)
	assert.Equal(t, 65000, req.MCRAsn)
	assert.Equal(t, "blue", req.DiversityZone)
	assert.Equal(t, "IT-2024", req.CostCentre)
	assert.Equal(t, "PROMO", req.PromoCode)
}

func TestPromptForMCRDetails_EmptyName(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{""}))

	_, err := promptForMCRDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestPromptForMCRDetails_InvalidTerm(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"MCR", "abc"}))

	_, err := promptForMCRDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid term")
}

func TestPromptForMCRDetails_InvalidPortSpeed(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"MCR", "12", "999"}))

	_, err := promptForMCRDetails(true)
	assert.Error(t, err)
}

func TestPromptForMCRDetails_InvalidLocationID(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"MCR", "12", "5000", "abc"}))

	_, err := promptForMCRDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid location ID")
}

func TestPromptForMCRDetails_MarketplaceVisibilityPromptError(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	// name, term, portSpeed, locationID succeed; marketplaceVisibility prompt (5th) fails.
	idx := 0
	responses := []string{"MCR", "12", "5000", "1"}
	utils.SetResourcePrompt(func(resourceType, msg string, noColor bool) (string, error) {
		if idx >= len(responses) {
			return "", fmt.Errorf("prompt failure on marketplace visibility")
		}
		val := responses[idx]
		idx++
		return val, nil
	})

	_, err := promptForMCRDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prompt failure on marketplace visibility")
}

func TestPromptForMCRDetails_InvalidPortSpeedNotNumeric(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"MCR", "12", "abc"}))

	_, err := promptForMCRDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid port speed")
	assert.NotContains(t, err.Error(), "strconv")
}

func TestPromptForMCRDetails_InvalidMarketplaceVisibility(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"MCR", "12", "5000", "1", "maybe"}))

	_, err := promptForMCRDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid marketplace visibility, must be true or false")
}

func TestPromptForMCRDetails_InvalidASN(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"MCR", "12", "5000", "1", "true", "abc"}))

	_, err := promptForMCRDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid ASN")
	assert.NotContains(t, err.Error(), "strconv")
}

func TestPromptForUpdateMCRDetails_Success(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	// name, costCentre, marketplaceVisibility(yes/no), visibilityValue, term, asn
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"Updated MCR", "IT-New", "yes", "true", "24", "65030",
	}))

	req, err := promptForUpdateMCRDetails("mcr-123", "", true)
	assert.NoError(t, err)
	assert.Equal(t, "Updated MCR", req.Name)
	assert.Equal(t, "IT-New", req.CostCentre)
	assert.NotNil(t, req.MarketplaceVisibility)
	assert.True(t, *req.MarketplaceVisibility)
	assert.NotNil(t, req.ContractTermMonths)
	assert.Equal(t, 24, *req.ContractTermMonths)
	require.NotNil(t, req.MCRAsn)
	assert.Equal(t, 65030, *req.MCRAsn)
}

func TestPromptForUpdateMCRDetails_MarketplaceVisibility_YAccepted(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	// name, costCentre, marketplaceVisibility(yes/no), visibilityValue(y)
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"", "", "yes", "y", "", "",
	}))

	req, err := promptForUpdateMCRDetails("mcr-123", "", true)
	assert.NoError(t, err)
	require.NotNil(t, req.MarketplaceVisibility)
	assert.True(t, *req.MarketplaceVisibility)
}

func TestPromptForUpdateMCRDetails_MarketplaceVisibility_NoAccepted(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	// name, costCentre, marketplaceVisibility(yes/no), visibilityValue(n)
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"", "", "yes", "n", "", "",
	}))

	req, err := promptForUpdateMCRDetails("mcr-123", "", true)
	assert.NoError(t, err)
	require.NotNil(t, req.MarketplaceVisibility)
	assert.False(t, *req.MarketplaceVisibility)
}

func TestPromptForUpdateMCRDetails_InvalidMarketplaceVisibilityValue(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	// name, costCentre, marketplaceVisibility(yes/no), visibilityValue(invalid)
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"", "", "yes", "ture",
	}))

	_, err := promptForUpdateMCRDetails("mcr-123", "", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marketplace visibility")
	assert.Contains(t, err.Error(), "not a recognized yes/no answer")
}

func TestPromptForUpdateMCRDetails_MarketplaceVisibilityGate_ShorthandY(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	// name, costCentre, marketplaceVisibility(y), visibilityValue(yes)
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"", "", "y", "yes", "", "",
	}))

	req, err := promptForUpdateMCRDetails("mcr-123", "", true)
	assert.NoError(t, err)
	require.NotNil(t, req.MarketplaceVisibility)
	assert.True(t, *req.MarketplaceVisibility)
}

func TestPromptForUpdateMCRDetails_InvalidMarketplaceVisibilityGateValue(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	// name, costCentre, marketplaceVisibility(invalid)
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"", "", "maybe",
	}))

	_, err := promptForUpdateMCRDetails("mcr-123", "", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update marketplace visibility")
	assert.Contains(t, err.Error(), "not a recognized yes/no answer")
}

func TestPromptForUpdateMCRDetails_MarketplaceVisibilityGate_NoSkipsUpdate(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	// name, costCentre, marketplaceVisibility(n), term, asn
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"Updated MCR", "", "n", "", "",
	}))

	req, err := promptForUpdateMCRDetails("mcr-123", "", true)
	assert.NoError(t, err)
	assert.Nil(t, req.MarketplaceVisibility)
	assert.Equal(t, "Updated MCR", req.Name)
}

func TestPromptForUpdateMCRDetails_CostCentrePreserved(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	// name set, cost centre left blank (skip) so the current value is kept.
	// name, costCentre, marketplaceVisibility(yes/no), term, asn
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"Updated MCR", "", "", "", "",
	}))

	req, err := promptForUpdateMCRDetails("mcr-123", "IT Dept", true)
	assert.NoError(t, err)
	assert.Equal(t, "IT Dept", req.CostCentre)
}

func TestPromptForUpdateMCRDetails_NoChanges(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"", "", "", "", ""}))

	_, err := promptForUpdateMCRDetails("mcr-123", "", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one field must be updated")
}

func TestPromptForUpdateMCRDetails_InvalidTerm(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"", "", "", "99"}))

	// Invalid term returns before the ASN prompt, so 4 responses suffice.
	_, err := promptForUpdateMCRDetails("mcr-123", "", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "contract term")
}

func TestPromptForUpdateMCRDetails_InvalidTermNotNumeric(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"", "", "", "abc"}))

	_, err := promptForUpdateMCRDetails("mcr-123", "", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid term")
	assert.NotContains(t, err.Error(), "strconv")
}

func TestPromptForUpdateMCRDetails_ASNOnly(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	// name, costCentre, marketplaceVisibility(yes/no), term, asn
	utils.SetResourcePrompt(mockPromptSequence([]string{"", "", "", "", "65020"}))

	req, err := promptForUpdateMCRDetails("mcr-123", "", true)
	assert.NoError(t, err)
	require.NotNil(t, req)
	require.NotNil(t, req.MCRAsn, "MCRAsn should be set when ASN prompt is answered")
	assert.Equal(t, 65020, *req.MCRAsn)
}

func TestPromptForUpdateMCRDetails_ASNWhitespaceTrimmed(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"", "", "", "", "  65020  "}))

	req, err := promptForUpdateMCRDetails("mcr-123", "", true)
	assert.NoError(t, err)
	require.NotNil(t, req)
	require.NotNil(t, req.MCRAsn)
	assert.Equal(t, 65020, *req.MCRAsn)
}

func TestPromptForUpdateMCRDetails_ASNWhitespaceOnlySkips(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"", "", "", "", "   "}))

	_, err := promptForUpdateMCRDetails("mcr-123", "", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one field must be updated")
}

func TestPromptForUpdateMCRDetails_InvalidASN(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"", "", "", "", "abc"}))

	_, err := promptForUpdateMCRDetails("mcr-123", "", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid ASN")
	assert.NotContains(t, err.Error(), "strconv")
}

func TestPromptForUpdateMCRDetails_ASNPromptError(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	// name, costCentre, mvFlag, term succeed; ASN prompt (5th) fails.
	idx := 0
	responses := []string{"", "", "", ""}
	utils.SetResourcePrompt(func(resourceType, msg string, noColor bool) (string, error) {
		if idx >= len(responses) {
			return "", fmt.Errorf("prompt failure on ASN")
		}
		val := responses[idx]
		idx++
		return val, nil
	})

	_, err := promptForUpdateMCRDetails("mcr-123", "", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prompt failure on ASN")
}

func TestPromptPrefixFilterEntry_Success(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{
		"192.168.0.0/24", "permit", "16", "24",
	}))

	entry, err := promptPrefixFilterEntry(true)
	assert.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, "192.168.0.0/24", entry.Prefix)
	assert.Equal(t, "permit", entry.Action)
	assert.Equal(t, 16, entry.Ge)
	assert.Equal(t, 24, entry.Le)
}

func TestPromptPrefixFilterEntry_EmptyPrefix(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{""}))

	entry, err := promptPrefixFilterEntry(true)
	assert.NoError(t, err)
	assert.Nil(t, entry)
}

func TestPromptPrefixFilterEntry_InvalidAction(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"10.0.0.0/8", "allow"}))

	_, err := promptPrefixFilterEntry(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid action")
}

func TestPromptPrefixFilterEntry_InvalidGE(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"10.0.0.0/8", "permit", "abc"}))

	_, err := promptPrefixFilterEntry(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid GE value")
}

func TestPromptAddNewPrefixEntries_Success(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	// First entry, then empty prefix to stop
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"10.0.0.0/8", "permit", "", "",
		"", // empty prefix to stop
	}))

	entries, err := promptAddNewPrefixEntries(true)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestPromptForPrefixFilterListDetails_Success(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	// description, addressFamily, then one entry + empty to stop
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"My PFL", "IPv4",
		"10.0.0.0/8", "permit", "", "",
		"", // stop adding entries
	}))

	req, err := promptForPrefixFilterListDetails("mcr-123", true)
	assert.NoError(t, err)
	assert.Equal(t, "My PFL", req.PrefixFilterList.Description)
	assert.Equal(t, "IPv4", req.PrefixFilterList.AddressFamily)
	assert.Len(t, req.PrefixFilterList.Entries, 1)
}

func TestPromptForPrefixFilterListDetails_EmptyDescription(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{""}))

	_, err := promptForPrefixFilterListDetails("mcr-123", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "description is required")
}

func TestPromptForPrefixFilterListDetails_InvalidAddressFamily(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"My PFL", "IPv5"}))

	_, err := promptForPrefixFilterListDetails("mcr-123", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid address family")
}

func TestPromptForPrefixFilterListDetails_NoEntries(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{
		"My PFL", "IPv4",
		"", // empty prefix immediately = no entries
	}))

	_, err := promptForPrefixFilterListDetails("mcr-123", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one entry is required")
}

func TestPromptForPrefixFilterListDetails_InvalidPrefix(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	// Prefix that isn't a CIDR passes the entry prompt but must fail
	// request-level validation before anything reaches the API.
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"My PFL", "IPv4",
		"banana", "permit", "", "",
		"", // stop adding entries
	}))

	_, err := promptForPrefixFilterListDetails("mcr-123", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a valid IPv4 CIDR notation")
}

func TestPromptForPrefixFilterListDetails_WrongFamilyPrefix(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{
		"My PFL", "IPv6",
		"10.0.0.0/8", "permit", "", "",
		"", // stop adding entries
	}))

	_, err := promptForPrefixFilterListDetails("mcr-123", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a valid IPv6 CIDR notation")
}

func TestPromptForIPSecTunnelCount(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    int
		wantErr     bool
		errContains string
	}{
		{"empty input uses default", "", 0, false, ""},
		{"valid count 10", "10", 10, false, ""},
		{"valid count 20", "20", 20, false, ""},
		{"valid count 30", "30", 30, false, ""},
		{"non-numeric input", "abc", 0, true, "invalid tunnel count"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalPrompt := utils.GetResourcePrompt()
			defer func() { utils.SetResourcePrompt(originalPrompt) }()

			utils.SetResourcePrompt(mockPromptSequence([]string{tt.input}))

			count, err := promptForIPSecTunnelCount(true)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, count)
			}
		})
	}
}

func TestPromptForIPSecTunnelCountUpdate(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    int
		wantErr     bool
		errContains string
	}{
		{"valid count 10", "10", 10, false, ""},
		{"valid count 20", "20", 20, false, ""},
		{"valid count 30", "30", 30, false, ""},
		{"zero disables IPSec", "0", 0, false, ""},
		{"empty input requires value", "", 0, true, "tunnel count is required"},
		{"non-numeric input", "abc", 0, true, "invalid tunnel count"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalPrompt := utils.GetResourcePrompt()
			defer func() { utils.SetResourcePrompt(originalPrompt) }()

			utils.SetResourcePrompt(mockPromptSequence([]string{tt.input}))

			count, err := promptForIPSecTunnelCountUpdate(true)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, count)
			}
		})
	}
}

func TestPromptUpdateExistingEntries_KeepUnmodified(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	current := []*megaport.MCRPrefixListEntry{
		{Prefix: "10.0.0.0/8", Action: "permit", Ge: 16, Le: 24},
	}

	// keep=yes, modify=no
	utils.SetResourcePrompt(mockPromptSequence([]string{"yes", "no"}))

	entries, err := promptUpdateExistingEntries(current, true)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "10.0.0.0/8", entries[0].Prefix)
}

func TestPromptUpdateExistingEntries_ModifyEntry(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	current := []*megaport.MCRPrefixListEntry{
		{Prefix: "10.0.0.0/8", Action: "permit", Ge: 16, Le: 24},
	}

	// keep=yes, modify=yes, new prefix, new action, new ge, new le
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"yes", "yes", "192.168.0.0/16", "deny", "20", "28",
	}))

	entries, err := promptUpdateExistingEntries(current, true)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "192.168.0.0/16", entries[0].Prefix)
	assert.Equal(t, "deny", entries[0].Action)
	assert.Equal(t, 20, entries[0].Ge)
	assert.Equal(t, 28, entries[0].Le)
}

func TestPromptUpdateExistingEntries_DeleteEntry(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	current := []*megaport.MCRPrefixListEntry{
		{Prefix: "10.0.0.0/8", Action: "permit"},
	}

	// keep=no (delete)
	utils.SetResourcePrompt(mockPromptSequence([]string{"no"}))

	entries, err := promptUpdateExistingEntries(current, true)
	assert.NoError(t, err)
	assert.Len(t, entries, 0)
}

func TestPromptUpdateExistingEntries_KeepUnmodified_ShorthandYN(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	current := []*megaport.MCRPrefixListEntry{
		{Prefix: "10.0.0.0/8", Action: "permit", Ge: 16, Le: 24},
	}

	// keep=y, modify=n -- a "y" answer must keep the entry, not drop it.
	utils.SetResourcePrompt(mockPromptSequence([]string{"y", "n"}))

	entries, err := promptUpdateExistingEntries(current, true)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "10.0.0.0/8", entries[0].Prefix)
}

func TestPromptUpdateExistingEntries_DeleteEntry_ShorthandN(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	current := []*megaport.MCRPrefixListEntry{
		{Prefix: "10.0.0.0/8", Action: "permit"},
	}

	// keep=n (delete)
	utils.SetResourcePrompt(mockPromptSequence([]string{"n"}))

	entries, err := promptUpdateExistingEntries(current, true)
	assert.NoError(t, err)
	assert.Len(t, entries, 0)
}

func TestPromptUpdateExistingEntries_InvalidKeepResponse(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	current := []*megaport.MCRPrefixListEntry{
		{Prefix: "10.0.0.0/8", Action: "permit"},
	}

	utils.SetResourcePrompt(mockPromptSequence([]string{"maybe"}))

	_, err := promptUpdateExistingEntries(current, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "keep entry 1")
	assert.Contains(t, err.Error(), "not a recognized yes/no answer")
}

func TestPromptUpdateExistingEntries_InvalidModifyResponse(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	current := []*megaport.MCRPrefixListEntry{
		{Prefix: "10.0.0.0/8", Action: "permit"},
	}

	utils.SetResourcePrompt(mockPromptSequence([]string{"yes", "maybe"}))

	_, err := promptUpdateExistingEntries(current, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "modify entry 1")
	assert.Contains(t, err.Error(), "not a recognized yes/no answer")
}

func TestPromptForUpdatePrefixFilterListDetails_Success(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	originalGetPFL := getMCRPrefixFilterListFunc
	defer func() {
		utils.SetResourcePrompt(originalPrompt)
		getMCRPrefixFilterListFunc = originalGetPFL
	}()

	getMCRPrefixFilterListFunc = func(_ context.Context, _ *megaport.Client, _ string, _ int) (*megaport.MCRPrefixFilterList, error) {
		return &megaport.MCRPrefixFilterList{
			ID:            1,
			Description:   "old desc",
			AddressFamily: "IPv4",
			Entries: []*megaport.MCRPrefixListEntry{
				{Prefix: "10.0.0.0/8", Action: "permit"},
			},
		}, nil
	}

	// description (empty = keep current), modifyExisting=no, addNew=no
	utils.SetResourcePrompt(mockPromptSequence([]string{"", "no", "no"}))

	result, err := promptForUpdatePrefixFilterListDetails(context.Background(), nil, "mcr-uid", 1, true)
	assert.NoError(t, err)
	assert.Equal(t, "old desc", result.Description)
	assert.Equal(t, "IPv4", result.AddressFamily)
	assert.Len(t, result.Entries, 1)
}
