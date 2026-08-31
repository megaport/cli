package mcr

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/megaport/megaport-cli/internal/validation"
	megaport "github.com/megaport/megaportgo"
)

// promptForIPSecTunnelCount prompts the user to select a tunnel count for a new IPSec add-on.
// Valid values are 10, 20, or 30 (0 means default to 10).
func promptForIPSecTunnelCount(noColor bool) (int, error) {
	prompt := "Enter IPSec tunnel count (10, 20, or 30; leave empty for default of 10): "
	input, err := utils.ResourcePrompt("mcr", prompt, noColor)
	if err != nil {
		return 0, err
	}
	if input == "" {
		return 0, nil
	}
	count, err := validation.ParseInt("tunnel count", strings.TrimSpace(input))
	if err != nil {
		return 0, err
	}
	return count, nil
}

// promptForIPSecTunnelCountUpdate prompts the user to select a tunnel count for updating an IPSec add-on.
// Valid values are 10, 20, or 30. Enter 0 to disable IPSec.
func promptForIPSecTunnelCountUpdate(noColor bool) (int, error) {
	prompt := "Enter new IPSec tunnel count (10, 20, or 30; enter 0 to disable IPSec): "
	input, err := utils.ResourcePrompt("mcr", prompt, noColor)
	if err != nil {
		return 0, err
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, fmt.Errorf("tunnel count is required for update (use 0 to disable IPSec)")
	}
	count, err := validation.ParseInt("tunnel count", input)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func promptForUpdateMCRDetails(mcrUID, currentCostCentre string, noColor bool) (*megaport.ModifyMCRRequest, error) {
	req := &megaport.ModifyMCRRequest{
		MCRID: mcrUID,
	}

	fieldsUpdated := false

	namePrompt := "Enter new MCR name (leave empty to skip): "
	name, err := utils.ResourcePrompt("mcr", namePrompt, noColor)
	if err != nil {
		return nil, err
	}
	if name != "" {
		req.Name = name
		fieldsUpdated = true
	}

	costCentrePrompt := "Enter new cost centre (leave empty to skip): "
	costCentre, err := utils.ResourcePrompt("mcr", costCentrePrompt, noColor)
	if err != nil {
		return nil, err
	}
	if costCentre != "" {
		req.CostCentre = costCentre
		fieldsUpdated = true
	}

	marketplaceVisibilityPrompt := "Update marketplace visibility? (y/yes/true/n/no/false, leave empty to skip): "
	marketplaceVisibilityStr, err := utils.ResourcePrompt("mcr", marketplaceVisibilityPrompt, noColor)
	if err != nil {
		return nil, err
	}
	if marketplaceVisibilityStr != "" {
		wantsVisibilityUpdate, err := utils.ParseYesNo(marketplaceVisibilityStr)
		if err != nil {
			return nil, fmt.Errorf("update marketplace visibility: %w", err)
		}
		if wantsVisibilityUpdate {
			visibilityValuePrompt := "Enter marketplace visibility (y/yes/true/n/no/false) (required): "
			visibilityValue, err := utils.ResourcePrompt("mcr", visibilityValuePrompt, noColor)
			if err != nil {
				return nil, err
			}

			marketplaceVisibility, err := utils.ParseYesNo(visibilityValue)
			if err != nil {
				return nil, fmt.Errorf("marketplace visibility: %w", err)
			}
			req.MarketplaceVisibility = &marketplaceVisibility
			fieldsUpdated = true
		}
	}

	termPrompt := fmt.Sprintf("Enter new term (%s months, leave empty to skip): ", validation.FormatIntSlice(validation.ValidContractTerms))
	termStr, err := utils.ResourcePrompt("mcr", termPrompt, noColor)
	if err != nil {
		return nil, err
	}
	if termStr != "" {
		term, err := validation.ParseInt("term", termStr)
		if err != nil {
			return nil, err
		}

		if err := validation.ValidateContractTerm(term); err != nil {
			return nil, err
		}

		req.ContractTermMonths = &term
		fieldsUpdated = true
	}

	asnPrompt := "Enter new MCR ASN (leave empty to skip): "
	asnStr, err := utils.ResourcePrompt("mcr", asnPrompt, noColor)
	if err != nil {
		return nil, err
	}
	asnStr = strings.TrimSpace(asnStr)
	if asnStr != "" {
		asn, err := validation.ParseInt("ASN", asnStr)
		if err != nil {
			return nil, err
		}
		if err := validation.ValidateMCRASN(int64(asn)); err != nil {
			return nil, err
		}
		req.MCRAsn = &asn
		fieldsUpdated = true
	}

	if !fieldsUpdated {
		return nil, fmt.Errorf("at least one field must be updated")
	}

	// Re-send the current cost centre when the user skipped the prompt, so the
	// update doesn't wipe it.
	if req.CostCentre == "" {
		req.CostCentre = currentCostCentre
	}

	req.WaitForUpdate = true
	req.WaitForTime = 10 * time.Minute

	return req, nil
}

func promptForMCRDetails(noColor bool) (*megaport.BuyMCRRequest, error) {
	name, err := utils.ResourcePrompt("mcr", "Enter MCR name (required): ", noColor)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	termStr, err := utils.ResourcePrompt("mcr", fmt.Sprintf("Enter term (%s months) (required): ", validation.FormatIntSlice(validation.ValidContractTerms)), noColor)
	if err != nil {
		return nil, err
	}
	term, err := validation.ParseInt("term", termStr)
	if err != nil {
		return nil, err
	}

	portSpeedStr, err := utils.ResourcePrompt("mcr", fmt.Sprintf("Enter port speed - valid port speeds are %s Mbps (required): ", validation.FormatIntSlice(validation.ValidMCRPortSpeeds)), noColor)
	if err != nil {
		return nil, err
	}
	portSpeed, err := validation.ParseInt("port speed", portSpeedStr)
	if err != nil {
		return nil, err
	}
	if err := validation.ValidateMCRPortSpeed(portSpeed); err != nil {
		return nil, err
	}
	locationIDStr, err := utils.ResourcePrompt("mcr", "Enter location ID (required): ", noColor)
	if err != nil {
		return nil, err
	}
	locationID, err := validation.ParseInt("location ID", locationIDStr)
	if err != nil {
		return nil, err
	}

	marketplaceVisibilityStr, err := utils.ResourcePrompt("mcr", "Enter marketplace visibility (true/false) (required): ", noColor)
	if err != nil {
		return nil, err
	}
	marketplaceVisibility, err := strconv.ParseBool(marketplaceVisibilityStr)
	if err != nil {
		return nil, fmt.Errorf("invalid marketplace visibility, must be true or false")
	}

	asnStr, err := utils.ResourcePrompt("mcr", "Enter MCR ASN (optional): ", noColor)
	if err != nil {
		return nil, err
	}

	var asn int
	if asnStr != "" {
		asnValue, err := validation.ParseInt("ASN", asnStr)
		if err != nil {
			return nil, err
		}
		asn = asnValue
	}

	diversityZone, err := utils.ResourcePrompt("mcr", "Enter diversity zone (optional): ", noColor)
	if err != nil {
		return nil, err
	}

	costCentre, err := utils.ResourcePrompt("mcr", "Enter cost centre (optional): ", noColor)
	if err != nil {
		return nil, err
	}

	promoCode, err := utils.ResourcePrompt("mcr", "Enter promo code (optional): ", noColor)
	if err != nil {
		return nil, err
	}

	resourceTags, err := utils.ResourceTagsPrompt(noColor)
	if err != nil {
		return nil, err
	}

	req := &megaport.BuyMCRRequest{
		Name:                  name,
		Term:                  term,
		PortSpeed:             portSpeed,
		LocationID:            locationID,
		MCRAsn:                asn,
		DiversityZone:         diversityZone,
		CostCentre:            costCentre,
		PromoCode:             promoCode,
		ResourceTags:          resourceTags,
		MarketplaceVisibility: &marketplaceVisibility,
	}

	if err := validation.ValidateMCRRequest(req); err != nil {
		return nil, err
	}

	return req, nil
}

// promptPrefixFilterEntry collects a single prefix filter entry's fields (prefix, action, ge, le).
// Returns nil entry if prefix is empty, signaling "done".
func promptPrefixFilterEntry(noColor bool) (*megaport.MCRPrefixListEntry, error) {
	prefix, err := utils.ResourcePrompt("mcr", "Enter prefix (e.g., 192.168.0.0/24): ", noColor)
	if err != nil {
		return nil, err
	}
	if prefix == "" {
		return nil, nil
	}

	actionStr, err := utils.ResourcePrompt("mcr", "Enter action (permit or deny): ", noColor)
	if err != nil {
		return nil, err
	}
	if actionStr != "permit" && actionStr != "deny" {
		return nil, fmt.Errorf("invalid action, must be permit or deny")
	}

	geStr, err := utils.ResourcePrompt("mcr", "Enter GE value (optional): ", noColor)
	if err != nil {
		return nil, err
	}
	var ge int
	if geStr != "" {
		geVal, err := strconv.Atoi(geStr)
		if err != nil {
			return nil, fmt.Errorf("invalid GE value, must be a number")
		}
		ge = geVal
	}

	leStr, err := utils.ResourcePrompt("mcr", "Enter LE value (optional): ", noColor)
	if err != nil {
		return nil, err
	}
	var le int
	if leStr != "" {
		leVal, err := strconv.Atoi(leStr)
		if err != nil {
			return nil, fmt.Errorf("invalid LE value, must be a number")
		}
		le = leVal
	}

	return &megaport.MCRPrefixListEntry{
		Prefix: prefix,
		Action: actionStr,
		Ge:     ge,
		Le:     le,
	}, nil
}

// promptAddNewPrefixEntries prompts the user to add new prefix filter entries in a loop.
func promptAddNewPrefixEntries(noColor bool) ([]*megaport.MCRPrefixListEntry, error) {
	var entries []*megaport.MCRPrefixListEntry
	for {
		output.PrintPlain("Add a new prefix filter entry (leave prefix blank to finish):", noColor)

		entry, err := promptPrefixFilterEntry(noColor)
		if err != nil {
			return nil, err
		}
		if entry == nil {
			break
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// promptUpdateExistingEntries iterates existing entries and prompts keep/modify/delete for each.
func promptUpdateExistingEntries(currentEntries []*megaport.MCRPrefixListEntry, noColor bool) ([]*megaport.MCRPrefixListEntry, error) {
	var entries []*megaport.MCRPrefixListEntry
	for i, entry := range currentEntries {
		output.PrintPlain("Entry %d - Current: Action: %s, Prefix: %s, GE: %d, LE: %d", noColor,
			i+1, entry.Action, entry.Prefix, entry.Ge, entry.Le)

		keepEntryStr, err := utils.ResourcePrompt("mcr", fmt.Sprintf("Keep entry %d? (y/yes/true/n/no/false) (required): ", i+1), noColor)
		if err != nil {
			return nil, err
		}
		keepEntry, err := utils.ParseYesNo(keepEntryStr)
		if err != nil {
			return nil, fmt.Errorf("keep entry %d: %w", i+1, err)
		}

		if keepEntry {
			modifyEntryStr, err := utils.ResourcePrompt("mcr", fmt.Sprintf("Modify entry %d? (y/yes/true/n/no/false) (required): ", i+1), noColor)
			if err != nil {
				return nil, err
			}
			modifyEntry, err := utils.ParseYesNo(modifyEntryStr)
			if err != nil {
				return nil, fmt.Errorf("modify entry %d: %w", i+1, err)
			}

			if modifyEntry {
				prefix, err := utils.ResourcePrompt("mcr", fmt.Sprintf("Enter new prefix for entry %d (current: %s): ", i+1, entry.Prefix), noColor)
				if err != nil {
					return nil, err
				}
				if prefix == "" {
					prefix = entry.Prefix
				}

				actionStr, err := utils.ResourcePrompt("mcr", fmt.Sprintf("Enter new action for entry %d (permit or deny, current: %s): ", i+1, entry.Action), noColor)
				if err != nil {
					return nil, err
				}
				if actionStr == "" {
					actionStr = entry.Action
				} else if actionStr != "permit" && actionStr != "deny" {
					return nil, fmt.Errorf("invalid action, must be permit or deny")
				}

				geStr, err := utils.ResourcePrompt("mcr", fmt.Sprintf("Enter new GE value for entry %d (current: %d): ", i+1, entry.Ge), noColor)
				if err != nil {
					return nil, err
				}
				ge := entry.Ge
				if geStr != "" {
					geVal, err := strconv.Atoi(geStr)
					if err != nil {
						return nil, fmt.Errorf("invalid GE value, must be a number")
					}
					ge = geVal
				}

				leStr, err := utils.ResourcePrompt("mcr", fmt.Sprintf("Enter new LE value for entry %d (current: %d): ", i+1, entry.Le), noColor)
				if err != nil {
					return nil, err
				}
				le := entry.Le
				if leStr != "" {
					leVal, err := strconv.Atoi(leStr)
					if err != nil {
						return nil, fmt.Errorf("invalid LE value, must be a number")
					}
					le = leVal
				}

				entries = append(entries, &megaport.MCRPrefixListEntry{
					Prefix: prefix,
					Action: actionStr,
					Ge:     ge,
					Le:     le,
				})
			} else {
				entries = append(entries, entry)
			}
		}
	}
	return entries, nil
}

func promptForPrefixFilterListDetails(mcrUID string, noColor bool) (*megaport.CreateMCRPrefixFilterListRequest, error) {
	description, err := utils.ResourcePrompt("mcr", "Enter description (required): ", noColor)
	if err != nil {
		return nil, err
	}
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}

	addressFamily, err := utils.ResourcePrompt("mcr", "Enter address family (IPv4 or IPv6) (required): ", noColor)
	if err != nil {
		return nil, err
	}
	if addressFamily != "IPv4" && addressFamily != "IPv6" {
		return nil, fmt.Errorf("invalid address family, must be IPv4 or IPv6")
	}

	entries, err := promptAddNewPrefixEntries(noColor)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("at least one entry is required")
	}

	req := &megaport.CreateMCRPrefixFilterListRequest{
		MCRID: mcrUID,
		PrefixFilterList: megaport.MCRPrefixFilterList{
			Description:   description,
			AddressFamily: addressFamily,
			Entries:       entries,
		},
	}

	if err := validation.ValidatePrefixFilterListRequest(req); err != nil {
		return nil, err
	}

	return req, nil
}

func promptForUpdatePrefixFilterListDetails(ctx context.Context, client *megaport.Client, mcrUID string, prefixFilterListID int, noColor bool) (*megaport.MCRPrefixFilterList, error) {
	currentPrefixFilterList, err := getMCRPrefixFilterListFunc(ctx, client, mcrUID, prefixFilterListID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve current prefix filter list: %w", err)
	}

	output.PrintPlain("Current description: %s", noColor, currentPrefixFilterList.Description)
	description, err := utils.ResourcePrompt("mcr", "Enter new description (leave empty to keep current): ", noColor)
	if err != nil {
		return nil, err
	}
	if description == "" {
		description = currentPrefixFilterList.Description
	}

	output.PrintPlain("Address family: %s (cannot be changed after creation)", noColor, currentPrefixFilterList.AddressFamily)
	addressFamily := currentPrefixFilterList.AddressFamily

	entries := make([]*megaport.MCRPrefixListEntry, 0, len(currentPrefixFilterList.Entries))

	modifyExisting, err := utils.ResourcePrompt("mcr", "Do you want to modify existing entries? (yes/no): ", noColor)
	if err != nil {
		return nil, err
	}

	if strings.ToLower(modifyExisting) != "yes" {
		entries = append(entries, currentPrefixFilterList.Entries...)
	} else {
		existingEntries, err := promptUpdateExistingEntries(currentPrefixFilterList.Entries, noColor)
		if err != nil {
			return nil, err
		}
		entries = append(entries, existingEntries...)
	}

	addNew, err := utils.ResourcePrompt("mcr", "Do you want to add new entries? (yes/no): ", noColor)
	if err != nil {
		return nil, err
	}

	if strings.ToLower(addNew) == "yes" {
		newEntries, err := promptAddNewPrefixEntries(noColor)
		if err != nil {
			return nil, err
		}
		entries = append(entries, newEntries...)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("at least one entry is required")
	}

	prefixFilterList := &megaport.MCRPrefixFilterList{
		ID:            prefixFilterListID,
		Description:   description,
		AddressFamily: addressFamily,
		Entries:       entries,
	}

	if err := validation.ValidateUpdatePrefixFilterList(prefixFilterList); err != nil {
		return nil, err
	}

	return prefixFilterList, nil
}
