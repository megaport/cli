package mve

import (
	"fmt"
	"testing"

	"github.com/megaport/megaport-cli/internal/utils"
	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockPromptSequence(responses []string) func(string, string, bool) (string, error) {
	idx := 0
	return func(_, _ string, _ bool) (string, error) {
		if idx >= len(responses) {
			return "", fmt.Errorf("unexpected prompt call #%d", idx)
		}
		val := responses[idx]
		idx++
		return val, nil
	}
}

// promptMVEBaseDetails tests

func TestPromptMVEBaseDetails_Success(t *testing.T) {
	original := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(original) }()

	// name, term, locationID, diversityZone, promoCode, costCentre, vendor, imageID, productSize, mveLabel
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"Test MVE", "12", "1", "blue", "PROMO", "IT-2024", "cisco", "42", "LARGE", "my-label",
	}))

	req, vendorStr, imageID, productSize, mveLabel, err := promptMVEBaseDetails(true)
	assert.NoError(t, err)
	assert.Equal(t, "Test MVE", req.Name)
	assert.Equal(t, 12, req.Term)
	assert.Equal(t, 1, req.LocationID)
	assert.Equal(t, "blue", req.DiversityZone)
	assert.Equal(t, "PROMO", req.PromoCode)
	assert.Equal(t, "IT-2024", req.CostCentre)
	assert.Equal(t, "cisco", vendorStr)
	assert.Equal(t, 42, imageID)
	assert.Equal(t, "LARGE", productSize)
	assert.Equal(t, "my-label", mveLabel)
}

func TestPromptMVEBaseDetails_NormalizesVendorAndProductSizeCase(t *testing.T) {
	original := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(original) }()

	// name, term, locationID, diversityZone, promoCode, costCentre, vendor, imageID, productSize, mveLabel
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"Test MVE", "12", "1", "", "", "", "Cisco", "42", "medium", "",
	}))

	_, vendorStr, _, productSize, _, err := promptMVEBaseDetails(true)
	require.NoError(t, err)
	assert.Equal(t, "cisco", vendorStr)
	assert.Equal(t, "MEDIUM", productSize)
}

func TestPromptForBuyMVEDetails_CapturesResourceTags(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	originalTags := utils.GetResourceTagsPrompt()
	defer func() {
		utils.SetResourcePrompt(originalPrompt)
		utils.SetResourceTagsPrompt(originalTags)
	}()

	// base details (10) + aruba vendor config (3) + one vnic then blank to finish (3)
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"Test MVE", "12", "1", "", "", "", "aruba", "1", "MEDIUM", "",
		"acct", "key", "systag",
		"eth0", "100", "",
	}))

	wantTags := map[string]string{"env": "prod", "owner": "netops"}
	utils.SetResourceTagsPrompt(func(bool) (map[string]string, error) { return wantTags, nil })

	req, err := promptForBuyMVEDetails(true)
	require.NoError(t, err)
	assert.Equal(t, wantTags, req.ResourceTags)
}

func TestPromptForBuyMVEDetails_ResourceTagsPromptError(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	originalTags := utils.GetResourceTagsPrompt()
	defer func() {
		utils.SetResourcePrompt(originalPrompt)
		utils.SetResourceTagsPrompt(originalTags)
	}()

	utils.SetResourcePrompt(mockPromptSequence([]string{
		"Test MVE", "12", "1", "", "", "", "aruba", "1", "MEDIUM", "",
		"acct", "key", "systag",
		"eth0", "100", "",
	}))
	utils.SetResourceTagsPrompt(func(bool) (map[string]string, error) {
		return nil, fmt.Errorf("tag prompt failed")
	})

	_, err := promptForBuyMVEDetails(true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tag prompt failed")
}

func TestPromptForBuyMVEDetails_MixedCaseVendorAndProductSizeAccepted(t *testing.T) {
	originalPrompt := utils.GetResourcePrompt()
	originalTags := utils.GetResourceTagsPrompt()
	defer func() {
		utils.SetResourcePrompt(originalPrompt)
		utils.SetResourceTagsPrompt(originalTags)
	}()

	// base details (10, vendor="Aruba", productSize="medium") + aruba vendor config (3) + one vnic then blank to finish (3)
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"Test MVE", "12", "1", "", "", "", "Aruba", "1", "medium", "",
		"acct", "key", "systag",
		"eth0", "100", "",
	}))
	utils.SetResourceTagsPrompt(func(bool) (map[string]string, error) { return nil, nil })

	req, err := promptForBuyMVEDetails(true)
	require.NoError(t, err)

	arubaCfg, ok := req.VendorConfig.(*megaport.ArubaConfig)
	require.True(t, ok, "expected an ArubaConfig, got %T", req.VendorConfig)
	assert.Equal(t, "aruba", arubaCfg.Vendor)
	assert.Equal(t, "MEDIUM", arubaCfg.ProductSize)
}

func TestPromptMVEBaseDetails_EmptyName(t *testing.T) {
	original := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(original) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{""}))

	_, _, _, _, _, err := promptMVEBaseDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestPromptMVEBaseDetails_InvalidTerm(t *testing.T) {
	original := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(original) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"MVE", "abc"}))

	_, _, _, _, _, err := promptMVEBaseDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid term")
}

func TestPromptMVEBaseDetails_InvalidLocationID(t *testing.T) {
	original := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(original) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"MVE", "12", "abc"}))

	_, _, _, _, _, err := promptMVEBaseDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid location ID")
}

func TestPromptMVEBaseDetails_InvalidImageID(t *testing.T) {
	original := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(original) }()

	// name, term, locationID, diversityZone, promoCode, costCentre, vendor, imageID (invalid)
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"MVE", "12", "1", "", "", "", "cisco", "abc",
	}))

	_, _, _, _, _, err := promptMVEBaseDetails(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid image ID")
}

// promptForUpdateMVEDetails tests

func TestPromptForUpdateMVEDetails_Success(t *testing.T) {
	original := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(original) }()

	// name, costCentre, contractTerm
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"NewName", "", "",
	}))

	req, err := promptForUpdateMVEDetails("mve-123", "", nil, true)
	assert.NoError(t, err)
	assert.Equal(t, "mve-123", req.MVEID)
	assert.Equal(t, "NewName", req.Name)
	assert.Equal(t, "", req.CostCentre)
	assert.Nil(t, req.ContractTermMonths)
}

func TestPromptForUpdateMVEDetails_NoChanges(t *testing.T) {
	original := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(original) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"", "", ""}))

	_, err := promptForUpdateMVEDetails("mve-123", "", nil, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one field must be provided for update")
}

func TestPromptForUpdateMVEDetails_InvalidContractTerm(t *testing.T) {
	original := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(original) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"", "", "abc"}))

	_, err := promptForUpdateMVEDetails("mve-123", "", nil, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid contract term")
}

func TestPromptForUpdateMVEDetails_UpdateVnicDescriptions(t *testing.T) {
	original := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(original) }()

	// name, costCentre, contractTerm, updateVnics?, vnic[0], vnic[1]
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"", "", "",
		"y",
		"New Data Plane", "", // second is left empty → keeps "Mgmt"
	}))

	currentVnics := []*megaport.MVENetworkInterface{
		{Description: "Data Plane"},
		{Description: "Mgmt"},
	}
	req, err := promptForUpdateMVEDetails("mve-123", "", currentVnics, true)
	require.NoError(t, err)
	require.Len(t, req.Vnics, 2)
	assert.Equal(t, "New Data Plane", req.Vnics[0].Description)
	assert.Equal(t, "Mgmt", req.Vnics[1].Description)
}

func TestPromptForUpdateMVEDetails_DeclineVnicUpdate(t *testing.T) {
	original := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(original) }()

	// name update, no cost-centre, no term, "n" to vnics
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"NewName", "", "",
		"n",
	}))

	currentVnics := []*megaport.MVENetworkInterface{{Description: "Data Plane"}}
	req, err := promptForUpdateMVEDetails("mve-123", "", currentVnics, true)
	require.NoError(t, err)
	assert.Equal(t, "NewName", req.Name)
	assert.Empty(t, req.Vnics)
}

func TestPromptForUpdateMVEDetails_VnicYesNoPromptError(t *testing.T) {
	original := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(original) }()

	// First three prompts (name/cost/term) succeed; fourth (y/N) returns an error.
	calls := 0
	utils.SetResourcePrompt(func(_, _ string, _ bool) (string, error) {
		calls++
		if calls == 4 {
			return "", fmt.Errorf("stdin closed")
		}
		return "", nil
	})

	currentVnics := []*megaport.MVENetworkInterface{{Description: "Data Plane"}}
	_, err := promptForUpdateMVEDetails("mve-123", "", currentVnics, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stdin closed")
}

func TestPromptForUpdateMVEDetails_VnicDescriptionPromptError(t *testing.T) {
	original := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(original) }()

	// name/cost/term empty, "y" to update vnics, then error on first description prompt.
	calls := 0
	utils.SetResourcePrompt(func(_, _ string, _ bool) (string, error) {
		calls++
		switch calls {
		case 1, 2, 3:
			return "", nil
		case 4:
			return "y", nil
		default:
			return "", fmt.Errorf("stdin closed")
		}
	})

	currentVnics := []*megaport.MVENetworkInterface{{Description: "Data Plane"}}
	_, err := promptForUpdateMVEDetails("mve-123", "", currentVnics, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stdin closed")
}

func TestPromptForUpdateMVEDetails_VnicNilEntryDefaultsToEmpty(t *testing.T) {
	original := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(original) }()

	// name, cost, term, accept vnic update, description for nil vnic
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"", "", "",
		"yes",
		"New Description",
	}))

	currentVnics := []*megaport.MVENetworkInterface{nil}
	req, err := promptForUpdateMVEDetails("mve-123", "", currentVnics, true)
	require.NoError(t, err)
	require.Len(t, req.Vnics, 1)
	assert.Equal(t, "New Description", req.Vnics[0].Description)
}

func TestPromptForUpdateMVEDetails_VnicEmptyCurrentEmptyInputErrors(t *testing.T) {
	original := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(original) }()

	// name, cost, term, accept vnic update, empty description for vnic with empty current desc
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"", "", "",
		"yes",
		"",
	}))

	currentVnics := []*megaport.MVENetworkInterface{{Description: ""}}
	_, err := promptForUpdateMVEDetails("mve-123", "", currentVnics, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vnics[0].description must not be empty")
}

// promptMVEVnics tests

func TestPromptMVEVnics_NoVnics(t *testing.T) {
	original := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(original) }()

	// empty description immediately to finish, then error because 0 VNICs
	utils.SetResourcePrompt(mockPromptSequence([]string{""}))

	_, err := promptMVEVnics(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one VNIC is required")
}

func TestPromptMVEVnics_OneVnic(t *testing.T) {
	original := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(original) }()

	// description, vlan, then empty description to finish
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"eth0", "100",
		"", // empty description to stop
	}))

	vnics, err := promptMVEVnics(true)
	assert.NoError(t, err)
	assert.Len(t, vnics, 1)
	assert.Equal(t, "eth0", vnics[0].Description)
	assert.Equal(t, 100, vnics[0].VLAN)
}

func TestPromptMVEVnics_MultipleVnics(t *testing.T) {
	original := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(original) }()

	// first VNIC: description, vlan; second VNIC: description, vlan; then empty to stop
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"eth0", "100",
		"eth1", "200",
		"", // empty description to stop
	}))

	vnics, err := promptMVEVnics(true)
	assert.NoError(t, err)
	assert.Len(t, vnics, 2)
	assert.Equal(t, "eth0", vnics[0].Description)
	assert.Equal(t, 100, vnics[0].VLAN)
	assert.Equal(t, "eth1", vnics[1].Description)
	assert.Equal(t, 200, vnics[1].VLAN)
}

func TestPromptMVEVnics_InvalidVLAN(t *testing.T) {
	original := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(original) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"eth0", "abc"}))

	_, err := promptMVEVnics(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid VLAN ID")
}

// promptMVEVendorConfig tests — cisco/palo_alto admin password handling

func TestPromptMVEVendorConfig_Cisco_WithAdminPassword(t *testing.T) {
	origResource := utils.GetResourcePrompt()
	origSecret := utils.GetSecretResourcePrompt()
	defer func() {
		utils.SetResourcePrompt(origResource)
		utils.SetSecretResourcePrompt(origSecret)
	}()

	// Order: manageLocally, adminSSHPublicKey, sshPublicKey, cloudInit,
	// fmcIPAddress, fmcRegistrationKey, fmcNatID, then SECRET adminPassword
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"true", "admin-ssh", "ssh-key", "cloud-init",
		"10.0.0.1", "fmc-reg", "fmc-nat",
	}))
	utils.SetSecretResourcePrompt(mockPromptSequence([]string{"s3cr3t!"}))

	cfg, err := promptMVEVendorConfig("cisco", 99, "LARGE", "lbl", true)
	assert.NoError(t, err)
	cisco, ok := cfg.(*megaport.CiscoConfig)
	assert.True(t, ok)
	assert.Equal(t, "cisco", cisco.Vendor)
	assert.Equal(t, 99, cisco.ImageID)
	assert.Equal(t, "LARGE", cisco.ProductSize)
	assert.Equal(t, "lbl", cisco.MVELabel)
	assert.True(t, cisco.ManageLocally)
	assert.Equal(t, "admin-ssh", cisco.AdminSSHPublicKey)
	assert.Equal(t, "ssh-key", cisco.SSHPublicKey)
	assert.Equal(t, "cloud-init", cisco.CloudInit)
	assert.Equal(t, "10.0.0.1", cisco.FMCIPAddress)
	assert.Equal(t, "fmc-reg", cisco.FMCRegistrationKey)
	assert.Equal(t, "fmc-nat", cisco.FMCNatID)
	assert.Equal(t, "s3cr3t!", cisco.AdminPassword)
}

func TestPromptMVEVendorConfig_Cisco_ManageLocally_ShorthandY(t *testing.T) {
	origResource := utils.GetResourcePrompt()
	origSecret := utils.GetSecretResourcePrompt()
	defer func() {
		utils.SetResourcePrompt(origResource)
		utils.SetSecretResourcePrompt(origSecret)
	}()

	// "y" must be interpreted as true, not silently treated as false.
	utils.SetResourcePrompt(mockPromptSequence([]string{
		"y", "admin-ssh", "ssh-key", "cloud-init",
		"10.0.0.1", "fmc-reg", "fmc-nat",
	}))
	utils.SetSecretResourcePrompt(mockPromptSequence([]string{""}))

	cfg, err := promptMVEVendorConfig("cisco", 99, "LARGE", "lbl", true)
	assert.NoError(t, err)
	cisco, ok := cfg.(*megaport.CiscoConfig)
	assert.True(t, ok)
	assert.True(t, cisco.ManageLocally)
}

func TestPromptMVEVendorConfig_Cisco_ManageLocally_ShorthandN(t *testing.T) {
	origResource := utils.GetResourcePrompt()
	origSecret := utils.GetSecretResourcePrompt()
	defer func() {
		utils.SetResourcePrompt(origResource)
		utils.SetSecretResourcePrompt(origSecret)
	}()

	utils.SetResourcePrompt(mockPromptSequence([]string{
		"n", "admin-ssh", "ssh-key", "cloud-init",
		"10.0.0.1", "fmc-reg", "fmc-nat",
	}))
	utils.SetSecretResourcePrompt(mockPromptSequence([]string{""}))

	cfg, err := promptMVEVendorConfig("cisco", 99, "LARGE", "lbl", true)
	assert.NoError(t, err)
	cisco, ok := cfg.(*megaport.CiscoConfig)
	assert.True(t, ok)
	assert.False(t, cisco.ManageLocally)
}

func TestPromptMVEVendorConfig_Cisco_InvalidManageLocally(t *testing.T) {
	origResource := utils.GetResourcePrompt()
	defer func() { utils.SetResourcePrompt(origResource) }()

	utils.SetResourcePrompt(mockPromptSequence([]string{"ture"}))

	_, err := promptMVEVendorConfig("cisco", 99, "LARGE", "lbl", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "manage locally")
	assert.Contains(t, err.Error(), "not a recognized yes/no answer")
}

func TestPromptMVEVendorConfig_PaloAlto_PlaintextOnly(t *testing.T) {
	origResource := utils.GetResourcePrompt()
	origSecret := utils.GetSecretResourcePrompt()
	defer func() {
		utils.SetResourcePrompt(origResource)
		utils.SetSecretResourcePrompt(origSecret)
	}()

	// Order: sshPublicKey, SECRET adminPassword, licenseData.
	// The hash prompt is skipped when adminPassword is non-empty so the user
	// cannot accidentally provide both credentials in interactive mode.
	utils.SetResourcePrompt(mockPromptSequence([]string{"ssh-key", "license"}))
	utils.SetSecretResourcePrompt(mockPromptSequence([]string{"p4ssw0rd"}))

	cfg, err := promptMVEVendorConfig("palo_alto", 42, "MEDIUM", "lbl", true)
	assert.NoError(t, err)
	pa, ok := cfg.(*megaport.PaloAltoConfig)
	assert.True(t, ok)
	assert.Equal(t, "palo_alto", pa.Vendor)
	assert.Equal(t, "ssh-key", pa.SSHPublicKey)
	assert.Equal(t, "p4ssw0rd", pa.AdminPassword)
	assert.Equal(t, "", pa.AdminPasswordHash)
	assert.Equal(t, "license", pa.LicenseData)
}

func TestPromptMVEVendorConfig_PaloAlto_HashOnly(t *testing.T) {
	origResource := utils.GetResourcePrompt()
	origSecret := utils.GetSecretResourcePrompt()
	defer func() {
		utils.SetResourcePrompt(origResource)
		utils.SetSecretResourcePrompt(origSecret)
	}()

	utils.SetResourcePrompt(mockPromptSequence([]string{"ssh-key", "license"}))
	utils.SetSecretResourcePrompt(mockPromptSequence([]string{"", "hashed-value"}))

	cfg, err := promptMVEVendorConfig("palo_alto", 42, "MEDIUM", "lbl", true)
	assert.NoError(t, err)
	pa, ok := cfg.(*megaport.PaloAltoConfig)
	assert.True(t, ok)
	assert.Equal(t, "", pa.AdminPassword)
	assert.Equal(t, "hashed-value", pa.AdminPasswordHash)
}

// TestPromptMVEVendorConfig_PaloAlto_BothBlank verifies that the prompt
// itself does not fail when the user leaves both credentials blank — the
// "at least one required" rule is enforced by ValidatePaloAltoConfig in the
// shared validation layer, so the prompt only has to capture input.
func TestPromptMVEVendorConfig_PaloAlto_BothBlank(t *testing.T) {
	origResource := utils.GetResourcePrompt()
	origSecret := utils.GetSecretResourcePrompt()
	defer func() {
		utils.SetResourcePrompt(origResource)
		utils.SetSecretResourcePrompt(origSecret)
	}()

	utils.SetResourcePrompt(mockPromptSequence([]string{"ssh-key", "license"}))
	utils.SetSecretResourcePrompt(mockPromptSequence([]string{"", ""}))

	cfg, err := promptMVEVendorConfig("palo_alto", 42, "MEDIUM", "lbl", true)
	assert.NoError(t, err)
	pa, ok := cfg.(*megaport.PaloAltoConfig)
	assert.True(t, ok)
	assert.Equal(t, "", pa.AdminPassword)
	assert.Equal(t, "", pa.AdminPasswordHash)
}
