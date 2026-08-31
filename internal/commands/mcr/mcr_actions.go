package mcr

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/config"
	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/megaport/megaport-cli/internal/validation"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
)

func exportMCRConfig(mcr *megaport.MCR) map[string]interface{} {
	m := map[string]interface{}{
		"name":       mcr.Name,
		"term":       mcr.ContractTermMonths,
		"portSpeed":  mcr.PortSpeed,
		"locationId": mcr.LocationID,
	}
	if mcr.Resources.VirtualRouter.ASN != 0 {
		m["mcrAsn"] = mcr.Resources.VirtualRouter.ASN
	}
	if mcr.DiversityZone != "" {
		m["diversityZone"] = mcr.DiversityZone
	}
	if mcr.CostCentre != "" {
		m["costCentre"] = mcr.CostCentre
	}
	return m
}

func buildMCRRequest(cmd *cobra.Command, noColor bool) (*megaport.BuyMCRRequest, error) {
	return utils.ResolveInput(utils.InputConfig[*megaport.BuyMCRRequest]{
		ResourceName: "MCR",
		Cmd:          cmd,
		NoColor:      noColor,
		FlagsProvided: func() bool {
			return cmd.Flags().Changed("name") || cmd.Flags().Changed("term") ||
				cmd.Flags().Changed("port-speed") || cmd.Flags().Changed("location-id") ||
				cmd.Flags().Changed("mcr-asn")
		},
		FromJSON:   processJSONMCRInput,
		FromFlags:  func() (*megaport.BuyMCRRequest, error) { return processFlagMCRInput(cmd) },
		FromPrompt: func() (*megaport.BuyMCRRequest, error) { return promptForMCRDetails(noColor) },
	})
}

func BuyMCR(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel := utils.ContextFromCmdWithDefault(cmd, utils.DefaultMutationTimeout)
	defer cancel()

	req, err := buildMCRRequest(cmd, noColor)
	if err != nil {
		return err
	}

	// If --ipsec-tunnel-count was explicitly set and the resolved request has no
	// add-ons yet (i.e. the interactive path was used, which doesn't process this
	// flag), apply it now. JSON and flag paths already populate req.AddOns, so the
	// len check prevents double-application.
	if cmd.Flags().Changed("ipsec-tunnel-count") && len(req.AddOns) == 0 {
		ipsecTunnelCount, _ := cmd.Flags().GetInt("ipsec-tunnel-count")
		if ipsecTunnelCount < 0 {
			return exitcodes.NewUsageError(fmt.Errorf("ipsec-tunnel-count must be 0 or a positive value (10, 20, or 30)"))
		}
		if ipsecTunnelCount > 0 {
			if err := validation.ValidateIPSecTunnelCount(ipsecTunnelCount, false); err != nil {
				return exitcodes.NewUsageError(err)
			}
		}
		req.AddOns = append(req.AddOns, &megaport.MCRAddOnIPsecConfig{
			AddOnType:   megaport.AddOnTypeIPsec,
			TunnelCount: ipsecTunnelCount,
		})
	}

	yes, err := utils.RequireYesForJSONBuy(cmd)
	if err != nil {
		return err
	}

	// Flag read errors are intentionally ignored — flags are registered by the command builder.
	noWait, _ := cmd.Flags().GetBool("no-wait")
	// Only the order submission is wrapped in WithOrderOnceRetry below, so the SDK
	// must not also poll for provisioning: a 429 raised during polling would
	// otherwise re-submit the order. Provisioning is awaited separately afterwards.
	req.WaitForProvision = false

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}

	spinner := output.PrintResourceValidating("MCR", noColor)
	err = client.MCRService.ValidateMCROrder(ctx, req)
	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to validate MCR order: %v", noColor, err)
		return err
	}

	if !yes {
		details := []utils.BuyConfirmDetail{
			{Key: "Name", Value: req.Name},
			{Key: "Term", Value: fmt.Sprintf("%d months", req.Term)},
			{Key: "Port Speed", Value: fmt.Sprintf("%d Mbps", req.PortSpeed)},
			{Key: "Location ID", Value: strconv.Itoa(req.LocationID)},
		}
		if req.MCRAsn != 0 {
			details = append(details, utils.BuyConfirmDetail{Key: "ASN", Value: strconv.Itoa(req.MCRAsn)})
		}
		if !utils.BuyConfirmPrompt("MCR", details, noColor) {
			output.PrintInfo("Purchase cancelled", noColor)
			return exitcodes.New(exitcodes.Cancelled, fmt.Errorf("cancelled by user"))
		}
	}

	var buySpinner *output.Spinner
	if noWait {
		buySpinner = output.PrintResourceCreating("MCR", req.Name, noColor)
	} else {
		buySpinner = output.PrintResourceProvisioning("MCR", req.Name, noColor)
	}
	var resp *megaport.BuyMCRResponse
	err = utils.WithOrderOnceRetry(ctx, func(ctx context.Context) error {
		var e error
		resp, e = buyMCRFunc(ctx, client, req)
		return e
	})
	if err != nil {
		buySpinner.Stop()
		output.PrintError("Failed to buy MCR: %v", noColor, err)
		return err
	}

	if resp == nil {
		buySpinner.Stop()
		output.PrintError("MCR buy returned an empty API response", noColor)
		return fmt.Errorf("empty response from API")
	}

	if resp.TechnicalServiceUID == "" {
		buySpinner.Stop()
		output.PrintError("MCR created but no UID returned", noColor)
		return fmt.Errorf("MCR created but no UID returned")
	}

	uid := resp.TechnicalServiceUID

	if !noWait {
		if err := waitForMCRProvision(ctx, client, req.Name, uid); err != nil {
			buySpinner.Stop()
			output.PrintError("MCR %s failed to provision: %v", noColor, uid, err)
			return err
		}
	}

	buySpinner.Stop()
	output.PrintResourceCreated("MCR", uid, noColor)
	return nil
}

// waitForMCRProvision polls the MCR's status until it is provisioned, bounding
// the wait by DefaultProvisionTimeout. It runs after the order is placed, so it
// must never be wrapped in an order-submission retry.
func waitForMCRProvision(ctx context.Context, client *megaport.Client, name, uid string) error {
	pollCtx, cancel := context.WithTimeout(ctx, utils.DefaultProvisionTimeout)
	defer cancel()
	return utils.WaitForProvision(pollCtx, "MCR", name, uid, func(ctx context.Context) (string, error) {
		m, err := getMCRFunc(ctx, client, uid)
		if err != nil {
			return "", err
		}
		if m == nil {
			return "", fmt.Errorf("MCR %s not found while waiting for provisioning", uid)
		}
		return m.ProvisioningStatus, nil
	})
}

func ValidateMCR(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	req, err := buildMCRRequest(cmd, noColor)
	if err != nil {
		return err
	}

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}

	spinner := output.PrintResourceValidating("MCR", noColor)
	err = client.MCRService.ValidateMCROrder(ctx, req)
	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to validate MCR order: %v", noColor, err)
		return err
	}

	output.PrintSuccess("MCR validation passed", noColor)
	return nil
}

func UpdateMCR(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel := utils.ContextFromCmdWithDefault(cmd, utils.DefaultMutationTimeout)
	defer cancel()

	mcrUID := args[0]

	interactive, _ := cmd.Flags().GetBool("interactive")
	jsonStr, _ := cmd.Flags().GetString("json")
	jsonFile, _ := cmd.Flags().GetString("json-file")

	flagsProvided := cmd.Flags().Changed("name") || cmd.Flags().Changed("cost-centre") ||
		cmd.Flags().Changed("marketplace-visibility") || cmd.Flags().Changed("term") ||
		cmd.Flags().Changed("mcr-asn")

	if err := utils.CheckInteractiveConflict(interactive, utils.HasConflictingInputFlags(cmd)); err != nil {
		output.PrintError("%v", noColor, err)
		return err
	}

	usingJSON := jsonStr != "" || jsonFile != ""
	if !usingJSON && !flagsProvided && !interactive {
		return fmt.Errorf("at least one field must be updated")
	}

	// Build and validate flag/JSON input before any network round-trip so
	// malformed input fails fast. Interactive prompts need the MCR's current
	// values, so they are built after login below.
	var req *megaport.ModifyMCRRequest
	var costCentreProvided bool
	var err error

	if usingJSON {
		output.PrintInfo("Using JSON input", noColor)
		req, costCentreProvided, err = processJSONUpdateMCRInput(jsonStr, jsonFile)
		if err != nil {
			output.PrintError("Failed to process JSON input: %v", noColor, err)
			return err
		}
		req.MCRID = mcrUID
	} else if flagsProvided {
		output.PrintInfo("Using flag input", noColor)
		req, costCentreProvided, err = processFlagUpdateMCRInput(cmd, mcrUID)
		if err != nil {
			output.PrintError("Failed to process flag input: %v", noColor, err)
			return err
		}
	}

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}

	// Fetch the current MCR so a name-only update can re-send the existing cost
	// centre (the SDK sends it without omitempty) and so changes can be diffed.
	originalMCR, err := getMCRFunc(ctx, client, mcrUID)
	if err != nil {
		output.PrintError("Failed to get original MCR: %v", noColor, err)
		return err
	}

	var currentCostCentre string
	if originalMCR != nil {
		currentCostCentre = originalMCR.CostCentre
	}

	if interactive {
		req, err = promptForUpdateMCRDetails(mcrUID, currentCostCentre, noColor)
		if err != nil {
			return err
		}
	} else if !costCentreProvided {
		req.CostCentre = currentCostCentre
	}

	req.WaitForUpdate = true
	req.WaitForTime = utils.DefaultProvisionTimeout

	updateSpinner := output.PrintResourceUpdating("MCR", mcrUID, noColor)
	var resp *megaport.ModifyMCRResponse
	err = utils.WithRetry(ctx, func(ctx context.Context) error {
		var e error
		resp, e = updateMCRFunc(ctx, client, req)
		return e
	})
	updateSpinner.Stop()

	if err != nil {
		output.PrintError("Failed to update MCR: %v", noColor, err)
		return err
	}

	if resp == nil {
		output.PrintError("MCR update returned an empty API response", noColor)
		return fmt.Errorf("empty response from API")
	}

	if !resp.IsUpdated {
		output.PrintError("MCR update request was not successful", noColor)
		return fmt.Errorf("MCR update request was not successful")
	}

	updatedMCR, err := getMCRFunc(ctx, client, mcrUID)
	if err != nil {
		output.PrintError("MCR was updated but failed to retrieve updated details: %v", noColor, err)
		output.PrintResourceUpdated("MCR", mcrUID, noColor)
		return nil
	}

	output.PrintResourceUpdated("MCR", mcrUID, noColor)

	displayMCRChanges(originalMCR, updatedMCR, noColor)

	return nil
}

func GetMCR(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	watch, _ := cmd.Flags().GetBool("watch")
	if watch {
		return watchGetMCR(cmd, args, noColor, outputFormat)
	}

	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	mcrUID := args[0]

	spinner := output.PrintResourceGetting("MCR", mcrUID, noColor)

	mcr, err := getMCRFunc(ctx, client, mcrUID)

	spinner.Stop()

	if err != nil {
		err = utils.WrapAPIError(err, "MCR", mcrUID)
		output.PrintError("Failed to get MCR: %v", noColor, err)
		return fmt.Errorf("failed to get MCR: %w", err)
	}

	if mcr == nil {
		output.PrintError("No MCR found with UID: %s", noColor, mcrUID)
		return fmt.Errorf("no MCR found with UID: %s", mcrUID)
	}

	export, _ := cmd.Flags().GetBool("export")
	if export {
		cfg := exportMCRConfig(mcr)
		jsonBytes, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal export config: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	err = printMCRs([]*megaport.MCR{mcr}, outputFormat, noColor)
	if err != nil {
		return fmt.Errorf("failed to print MCRs: %w", err)
	}
	return nil
}

func watchGetMCR(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	mcrUID := args[0]
	return utils.WatchResource(cmd, "MCR", mcrUID, noColor, outputFormat, config.Login,
		func(pollCtx context.Context, client *megaport.Client) (string, error) {
			mcr, err := getMCRFunc(pollCtx, client, mcrUID)
			if err != nil {
				return "", err
			}
			if mcr == nil {
				return "", fmt.Errorf("no MCR found with UID: %s", mcrUID)
			}
			err = printMCRs([]*megaport.MCR{mcr}, outputFormat, noColor)
			return mcr.ProvisioningStatus, err
		})
}

func ListMCRs(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)
	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	locationID, _ := cmd.Flags().GetInt("location-id")
	portSpeed, _ := cmd.Flags().GetInt("port-speed")
	mcrName, _ := cmd.Flags().GetString("name")
	includeInactive, _ := cmd.Flags().GetBool("include-inactive")

	req := &megaport.ListMCRsRequest{
		IncludeInactive: includeInactive,
	}

	spinner := output.PrintResourceListing("MCR", noColor)

	mcrs, err := client.MCRService.ListMCRs(ctx, req)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to list MCRs: %v", noColor, err)
		return fmt.Errorf("failed to list MCRs: %w", err)
	}

	var activeMCRs []*megaport.MCR
	if !includeInactive {
		for _, mcr := range mcrs {
			if mcr != nil &&
				mcr.ProvisioningStatus != megaport.STATUS_DECOMMISSIONED &&
				mcr.ProvisioningStatus != megaport.STATUS_CANCELLED &&
				mcr.ProvisioningStatus != utils.StatusDecommissioning {
				activeMCRs = append(activeMCRs, mcr)
			}
		}
		mcrs = activeMCRs
	}

	// Name, locationID, and portSpeed filtering are client-side; SDK only supports IncludeInactive.
	filteredMCRs := filterMCRs(mcrs, locationID, portSpeed, mcrName)

	limit, _ := cmd.Flags().GetInt("limit") // applied client-side after fetch

	tagFilters, _ := cmd.Flags().GetStringArray("tag")
	if len(tagFilters) > 0 {
		tagSpinner := output.PrintCustomSpinner("Fetching tags for", "MCRs", noColor)
		var tagErrs map[string]error
		filteredMCRs, tagErrs = utils.ApplyTagFilter(ctx, filteredMCRs,
			func(m *megaport.MCR) string { return m.UID },
			func(ctx context.Context, uid string) (map[string]string, error) {
				return listMCRResourceTagsFunc(ctx, client, uid)
			},
			tagFilters, limit,
		)
		tagSpinner.Stop()
		if err := ctx.Err(); err != nil {
			return err
		}
		for uid, err := range tagErrs {
			output.PrintWarning("Failed to fetch tags for MCR %s, skipping: %v", noColor, uid, err)
		}
	}
	return utils.ApplyLimitAndPrint(filteredMCRs, limit, outputFormat, noColor,
		"No MCRs found. Create one with 'megaport mcr buy'.", printMCRs)
}
