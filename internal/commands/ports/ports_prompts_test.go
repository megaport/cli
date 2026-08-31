package ports

import (
	"fmt"
	"testing"

	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/stretchr/testify/assert"
)

// mockPromptSequence returns a ResourcePrompt mock that returns values in order
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

func TestPromptForPortDetails_Success(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	originalTagsPrompt := utils.GetResourceTagsPrompt()
	defer func() {
		utils.SetResourcePrompt(originalPrompt)
		utils.SetResourceTagsPrompt(originalTagsPrompt)
	}()

	// Prompts: name, term, portSpeed, locationID, marketplaceVisibility, diversityZone, costCentre, promoCode
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"Test Port", // name
		"12",        // term
		"10000",     // portSpeed
		"1",         // locationID
		"true",      // marketplaceVisibility
		"blue",      // diversityZone
		"IT-2024",   // costCentre
		"PROMO123",  // promoCode
	}))
	utils.SetResourceTagsPrompt(func(noColor bool) (map[string]string, error) {
		return map[string]string{"env": "test"}, nil
	})

	req, err := promptForPortDetails(true)
	assert.NoError(t, err)
	assert.Equal(t, "Test Port", req.Name)
	assert.Equal(t, 12, req.Term)
	assert.Equal(t, 10000, req.PortSpeed)
	assert.Equal(t, 1, req.LocationId)
	assert.True(t, req.MarketPlaceVisibility)
	assert.Equal(t, "blue", req.DiversityZone)
	assert.Equal(t, "IT-2024", req.CostCentre)
	assert.Equal(t, "PROMO123", req.PromoCode)
	assert.Equal(t, map[string]string{"env": "test"}, req.ResourceTags)
}

func TestPromptForPortDetails_EmptyName(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{""}))

	_, err := promptForPortDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid port name")
}

func TestPromptForPortDetails_InvalidTerm(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"Port", "abc"}))

	_, err := promptForPortDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid term")
}

func TestPromptForPortDetails_InvalidPortSpeed(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"Port", "12", "999"}))

	_, err := promptForPortDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid port speed")
}

func TestPromptForPortDetails_InvalidLocationID(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"Port", "12", "10000", "abc"}))

	_, err := promptForPortDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid location ID")
}

func TestPromptForPortDetails_InvalidMarketplaceVisibility(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"Port", "12", "10000", "1", "maybe"}))

	_, err := promptForPortDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid marketplace visibility")
}

func TestPromptForLAGPortDetails_Success(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	originalTagsPrompt := utils.GetResourceTagsPrompt()
	defer func() {
		utils.SetResourcePrompt(originalPrompt)
		utils.SetResourceTagsPrompt(originalTagsPrompt)
	}()

	// Prompts: name, term, portSpeed, locationID, lagCount, marketplaceVisibility, diversityZone, costCentre, promoCode
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"LAG Port", // name
		"12",       // term
		"10000",    // portSpeed
		"1",        // locationID
		"2",        // lagCount
		"true",     // marketplaceVisibility
		"red",      // diversityZone
		"IT-LAG",   // costCentre
		"",         // promoCode
	}))
	utils.SetResourceTagsPrompt(func(noColor bool) (map[string]string, error) {
		return nil, nil
	})

	req, err := promptForLAGPortDetails(true)
	assert.NoError(t, err)
	assert.Equal(t, "LAG Port", req.Name)
	assert.Equal(t, 12, req.Term)
	assert.Equal(t, 10000, req.PortSpeed)
	assert.Equal(t, 1, req.LocationId)
	assert.Equal(t, 2, req.LagCount)
	assert.True(t, req.MarketPlaceVisibility)
}

func TestPromptForLAGPortDetails_InvalidLAGCount(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"LAG", "12", "10000", "1", "abc"}))

	_, err := promptForLAGPortDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid LAG count")
}

func TestPromptForLAGPortDetails_LAGCountOutOfRange(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"LAG", "12", "10000", "1", "9"}))

	_, err := promptForLAGPortDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "between 1 and 8")
}

func TestPromptForLAGPortDetails_InvalidPortSpeed(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"LAG", "12", "1000"}))

	_, err := promptForLAGPortDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid port speed")
}

func TestPromptForUpdatePortDetails_Success(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	// Prompts: name, marketplaceVisibility, costCentre, term
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"Updated Name", // name
		"true",         // marketplaceVisibility
		"IT-Updated",   // costCentre
		"24",           // term
	}))

	req, err := promptForUpdatePortDetails("port-123", "", true)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", req.Name)
	assert.Equal(t, "IT-Updated", req.CostCentre)
	assert.Equal(t, "port-123", req.PortID)
}

func TestPromptForUpdatePortDetails_CostCentrePreserved(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	// name set, cost centre left blank (skip) so the current value is kept.
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"Updated Name", // name
		"",             // marketplaceVisibility
		"",             // costCentre (skipped)
		"",             // term
	}))

	req, err := promptForUpdatePortDetails("port-123", "IT Dept", true)
	assert.NoError(t, err)
	assert.Equal(t, "IT Dept", req.CostCentre)
}

func TestPromptForUpdatePortDetails_NoChanges(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"", "", "", ""}))

	_, err := promptForUpdatePortDetails("port-123", "", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one field must be updated")
}

func TestPromptForUpdatePortDetails_InvalidTermNotNumeric(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"Updated Name", "true", "IT-Updated", "abc"}))

	_, err := promptForUpdatePortDetails("port-123", "", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid term")
	assert.NotContains(t, err.Error(), "strconv")
}

func TestPromptForLAGPortDetails_InvalidTerm(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"LAG", "abc"}))

	_, err := promptForLAGPortDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid term")
	assert.NotContains(t, err.Error(), "strconv")
}

func TestPromptForLAGPortDetails_InvalidLocationID(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(originalPrompt) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"LAG", "12", "10000", "abc"}))

	_, err := promptForLAGPortDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid location ID")
	assert.NotContains(t, err.Error(), "strconv")
}
