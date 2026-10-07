package mcr

import (
	"fmt"

	"github.com/megaport/megaport-cli/internal/base/output"
	megaport "github.com/megaport/megaportgo"
)

// mcrOutput represents the desired fields for JSON output of MCR details.
type mcrOutput struct {
	output.Output      `json:"-" header:"-"`
	UID                string           `json:"uid" header:"UID"`
	Name               string           `json:"name" header:"Name"`
	LocationID         int              `json:"location_id" header:"Location ID"`
	ProvisioningStatus string           `json:"provisioning_status" header:"Status"`
	ASN                int              `json:"asn" header:"ASN"`
	Speed              int              `json:"speed" header:"Speed"`
	AddOns             []mcrAddOnOutput `json:"add_ons,omitempty" header:"-" csv:"-"`
}

// mcrAddOnOutput represents an add-on attached to an MCR.
type mcrAddOnOutput struct {
	AddOnUID    string `json:"add_on_uid"`
	AddOnType   string `json:"add_on_type"`
	TunnelCount int    `json:"tunnel_count"`
}

// toMCROutput converts a *megaport.MCR to our mcrOutput struct.
func toMCROutput(mcr *megaport.MCR) (mcrOutput, error) {
	if mcr == nil {
		return mcrOutput{}, fmt.Errorf("invalid MCR: nil value")
	}

	o := mcrOutput{
		UID:                mcr.UID,
		Name:               mcr.Name,
		LocationID:         mcr.LocationID,
		ProvisioningStatus: mcr.ProvisioningStatus,
		Speed:              mcr.PortSpeed,
	}

	o.ASN = mcr.Resources.VirtualRouter.ASN

	for _, addon := range mcr.AddOns {
		if addon == nil {
			continue
		}
		o.AddOns = append(o.AddOns, mcrAddOnOutput{
			AddOnUID:    addon.AddOnUID,
			AddOnType:   addon.AddOnType,
			TunnelCount: addon.TunnelCount,
		})
	}

	return o, nil
}

// printMCRs prints a list of MCRs in the specified format.
func printMCRs(mcrs []*megaport.MCR, format string, noColor bool) error {
	outputs := make([]mcrOutput, 0, len(mcrs))
	for _, mcr := range mcrs {
		output, err := toMCROutput(mcr)
		if err != nil {
			return err
		}
		outputs = append(outputs, output)
	}
	return output.PrintOutput(outputs, format, noColor)
}

// displayMCRChanges compares the original and updated MCR and displays the differences
func displayMCRChanges(original, updated *megaport.MCR, noColor bool) {
	if original == nil || updated == nil {
		return
	}

	changes := []output.FieldChange{
		{Label: "Name", OldValue: original.Name, NewValue: updated.Name},
		{Label: "Cost Centre", OldValue: output.FormatOptionalString(original.CostCentre), NewValue: output.FormatOptionalString(updated.CostCentre)},
		{Label: "Contract Term", OldValue: fmt.Sprintf("%d months", original.ContractTermMonths), NewValue: fmt.Sprintf("%d months", updated.ContractTermMonths)},
		{Label: "Marketplace Visibility", OldValue: output.FormatBool(original.MarketplaceVisibility), NewValue: output.FormatBool(updated.MarketplaceVisibility)},
	}

	originalASN := original.Resources.VirtualRouter.ASN
	updatedASN := updated.Resources.VirtualRouter.ASN
	if originalASN != 0 || updatedASN != 0 {
		changes = append(changes, output.FieldChange{
			Label:    "ASN",
			OldValue: fmt.Sprintf("%d", originalASN),
			NewValue: fmt.Sprintf("%d", updatedASN),
		})
	}

	output.DisplayChanges(changes, noColor)
}

type prefixFilterListOutput struct {
	output.Output `json:"-" header:"-"`
	ID            int                           `json:"id"`
	Description   string                        `json:"description"`
	AddressFamily string                        `json:"address_family"`
	Entries       []prefixFilterListEntryOutput `json:"entries"`
}

type prefixFilterListEntryOutput struct {
	output.Output `json:"-" header:"-"`
	Action        string `json:"action"`
	Prefix        string `json:"prefix"`
	Ge            *int   `json:"ge,omitempty"`
	Le            *int   `json:"le,omitempty"`
}

func toPrefixFilterListOutput(prefixFilterList *megaport.MCRPrefixFilterList) (prefixFilterListOutput, error) {
	if prefixFilterList == nil {
		return prefixFilterListOutput{}, fmt.Errorf("invalid prefix filter list: nil value")
	}

	entries := make([]prefixFilterListEntryOutput, len(prefixFilterList.Entries))
	for i, entry := range prefixFilterList.Entries {
		entries[i] = prefixFilterListEntryOutput{
			Action: entry.Action,
			Prefix: entry.Prefix,
			Ge:     entry.Ge,
			Le:     entry.Le,
		}
	}

	return prefixFilterListOutput{
		ID:            prefixFilterList.ID,
		Description:   prefixFilterList.Description,
		AddressFamily: prefixFilterList.AddressFamily,
		Entries:       entries,
	}, nil
}

type MCRStatus struct {
	UID    string `json:"uid" header:"UID"`
	Name   string `json:"name" header:"Name"`
	Status string `json:"status" header:"Status"`
	ASN    int    `json:"asn" header:"ASN"`
	Speed  int    `json:"speed" header:"Speed"`
}
