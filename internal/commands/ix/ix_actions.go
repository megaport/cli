package ix

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
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
)

func exportIXConfig(ix *megaport.IX) map[string]interface{} {
	m := map[string]interface{}{
		"productName":        ix.ProductName,
		"networkServiceType": ix.NetworkServiceType,
		"rateLimit":          ix.RateLimit,
		"vlan":               ix.VLAN,
		"asn":                ix.ASN,
	}
	if ix.MACAddress != "" {
		m["macAddress"] = ix.MACAddress
	}
	// productUid (parent port UID) is not stored on the IX struct — must be added manually
	return m
}

func ListIXs(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)
	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	// Flag read errors are intentionally ignored — flags are registered by the command builder.
	name, _ := cmd.Flags().GetString("name")
	asn, _ := cmd.Flags().GetInt("asn")
	vlan, _ := cmd.Flags().GetInt("vlan")
	networkServiceType, _ := cmd.Flags().GetString("network-service-type")
	locationID, _ := cmd.Flags().GetInt("location-id")
	rateLimit, _ := cmd.Flags().GetInt("rate-limit")
	includeInactive, _ := cmd.Flags().GetBool("include-inactive")

	req := &megaport.ListIXsRequest{
		IncludeInactive: includeInactive,
	}

	spinner := output.PrintResourceListing("IX", noColor)

	ixs, err := client.IXService.ListIXs(ctx, req)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to list IXs: %v", noColor, err)
		return fmt.Errorf("failed to list IXs: %w", err)
	}

	if !includeInactive {
		var activeIXs []*megaport.IX
		for _, ix := range ixs {
			if ix != nil &&
				ix.ProvisioningStatus != megaport.STATUS_DECOMMISSIONED &&
				ix.ProvisioningStatus != megaport.STATUS_CANCELLED &&
				ix.ProvisioningStatus != utils.StatusDecommissioning {
				activeIXs = append(activeIXs, ix)
			}
		}
		ixs = activeIXs
	}

	filteredIXs := filterIXs(ixs, name, networkServiceType, asn, vlan, locationID, rateLimit)

	limit, _ := cmd.Flags().GetInt("limit")
	return utils.ApplyLimitAndPrint(filteredIXs, limit, outputFormat, noColor,
		"No IX connections found. Create one with 'megaport ix buy'.", printIXs)
}

func GetIX(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)
	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	ixUID := args[0]

	spinner := output.PrintResourceGetting("IX", ixUID, noColor)

	ix, err := getIXFunc(ctx, client, ixUID)

	spinner.Stop()

	if err != nil {
		err = utils.WrapAPIError(err, "IX", ixUID)
		output.PrintError("Failed to get IX: %v", noColor, err)
		return fmt.Errorf("failed to get IX: %w", err)
	}

	if ix == nil {
		output.PrintError("No IX found with UID: %s", noColor, ixUID)
		return fmt.Errorf("no IX found with UID: %s", ixUID)
	}

	export, _ := cmd.Flags().GetBool("export")
	if export {
		cfg := exportIXConfig(ix)
		jsonBytes, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal export config: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	err = printIXs([]*megaport.IX{ix}, outputFormat, noColor)
	if err != nil {
		return fmt.Errorf("failed to print IXs: %w", err)
	}
	return nil
}

func GetIXStatus(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)
	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	ixUID := args[0]

	spinner := output.PrintResourceGetting("IX", ixUID, noColor)

	ix, err := client.IXService.GetIX(ctx, ixUID)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to get IX status: %v", noColor, err)
		return fmt.Errorf("failed to get IX status: %w", err)
	}

	if ix == nil {
		output.PrintError("No IX found with UID: %s", noColor, ixUID)
		return fmt.Errorf("no IX found with UID: %s", ixUID)
	}

	status := []IXStatus{
		{
			UID:    ix.ProductUID,
			Name:   ix.ProductName,
			Status: ix.ProvisioningStatus,
			Type:   ix.NetworkServiceType,
		},
	}

	return output.PrintOutput(status, outputFormat, noColor)
}

func buildIXRequest(cmd *cobra.Command, noColor bool) (*megaport.BuyIXRequest, error) {
	return utils.ResolveInput(utils.InputConfig[*megaport.BuyIXRequest]{
		ResourceName: "IX",
		Cmd:          cmd,
		NoColor:      noColor,
		FlagsProvided: func() bool {
			return cmd.Flags().Changed("name") || cmd.Flags().Changed("product-uid") ||
				cmd.Flags().Changed("network-service-type") || cmd.Flags().Changed("asn") ||
				cmd.Flags().Changed("mac-address") || cmd.Flags().Changed("rate-limit") ||
				cmd.Flags().Changed("vlan")
		},
		FromJSON:  buildIXRequestFromJSON,
		FromFlags: func() (*megaport.BuyIXRequest, error) { return buildIXRequestFromFlags(cmd) },
		FromPrompt: func() (*megaport.BuyIXRequest, error) {
			ctx, cancel := utils.ContextFromCmd(cmd)
			defer cancel()
			return buildIXRequestFromPrompt(ctx, noColor)
		},
	})
}

func BuyIX(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel := utils.ContextFromCmdWithDefault(cmd, utils.DefaultMutationTimeout)
	defer cancel()

	req, err := buildIXRequest(cmd, noColor)
	if err != nil {
		return err
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

	spinner := output.PrintResourceValidating("IX", noColor)
	err = client.IXService.ValidateIXOrder(ctx, req)
	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to validate IX order: %v", noColor, err)
		return err
	}

	if !yes {
		details := []utils.BuyConfirmDetail{
			{Key: "Name", Value: req.Name},
			{Key: "Network Service Type", Value: req.NetworkServiceType},
			{Key: "Rate Limit", Value: fmt.Sprintf("%d Mbps", req.RateLimit)},
			{Key: "ASN", Value: strconv.Itoa(req.ASN)},
		}
		if !utils.BuyConfirmPrompt("IX", details, noColor) {
			output.PrintInfo("Purchase cancelled", noColor)
			return exitcodes.New(exitcodes.Cancelled, fmt.Errorf("cancelled by user"))
		}
	}

	var buySpinner *output.Spinner
	if noWait {
		buySpinner = output.PrintResourceCreating("IX", req.Name, noColor)
	} else {
		buySpinner = output.PrintResourceProvisioning("IX", req.Name, noColor)
	}
	var resp *megaport.BuyIXResponse
	err = utils.WithOrderOnceRetry(ctx, func(ctx context.Context) error {
		var e error
		resp, e = buyIXFunc(ctx, client, req)
		return e
	})
	if err != nil {
		buySpinner.Stop()
		output.PrintError("Failed to buy IX: %v", noColor, err)
		return err
	}

	if resp == nil {
		buySpinner.Stop()
		output.PrintError("IX buy returned an empty API response", noColor)
		return fmt.Errorf("empty response from API")
	}

	if resp.TechnicalServiceUID == "" {
		buySpinner.Stop()
		output.PrintError("IX created but no UID returned", noColor)
		return fmt.Errorf("IX created but no UID returned")
	}

	uid := resp.TechnicalServiceUID

	if !noWait {
		if err := waitForIXProvision(ctx, client, req.Name, uid); err != nil {
			buySpinner.Stop()
			output.PrintError("IX %s failed to provision: %v", noColor, uid, err)
			return err
		}
	}

	buySpinner.Stop()
	output.PrintResourceCreated("IX", uid, noColor)
	return nil
}

// waitForIXProvision polls the IX's status until it is provisioned, bounding the
// wait by DefaultProvisionTimeout. It runs after the order is placed, so it must
// never be wrapped in an order-submission retry.
func waitForIXProvision(ctx context.Context, client *megaport.Client, name, uid string) error {
	pollCtx, cancel := context.WithTimeout(ctx, utils.DefaultProvisionTimeout)
	defer cancel()
	return utils.WaitForProvision(pollCtx, "IX", name, uid, func(ctx context.Context) (string, error) {
		ix, err := getIXFunc(ctx, client, uid)
		if err != nil {
			return "", err
		}
		if ix == nil {
			return "", fmt.Errorf("IX %s not found while waiting for provisioning", uid)
		}
		return ix.ProvisioningStatus, nil
	})
}

func ValidateIX(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	req, err := buildIXRequest(cmd, noColor)
	if err != nil {
		return err
	}

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}

	spinner := output.PrintResourceValidating("IX", noColor)
	err = client.IXService.ValidateIXOrder(ctx, req)
	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to validate IX order: %v", noColor, err)
		return err
	}

	output.PrintSuccess("IX validation passed", noColor)
	return nil
}

func UpdateIX(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel := utils.ContextFromCmdWithDefault(cmd, utils.DefaultMutationTimeout)
	defer cancel()

	ixUID := args[0]

	interactive, _ := cmd.Flags().GetBool("interactive")
	jsonStr, _ := cmd.Flags().GetString("json")
	jsonFile, _ := cmd.Flags().GetString("json-file")

	flagsProvided := cmd.Flags().Changed("name") || cmd.Flags().Changed("rate-limit") ||
		cmd.Flags().Changed("cost-centre") || cmd.Flags().Changed("vlan") ||
		cmd.Flags().Changed("mac-address") || cmd.Flags().Changed("asn") ||
		cmd.Flags().Changed("password") || cmd.Flags().Changed("public-graph") ||
		cmd.Flags().Changed("reverse-dns") || cmd.Flags().Changed("a-end-product-uid") ||
		cmd.Flags().Changed("shutdown")

	if err := utils.CheckInteractiveConflict(interactive, utils.HasConflictingInputFlags(cmd)); err != nil {
		output.PrintError("%v", noColor, err)
		return err
	}

	var req *megaport.UpdateIXRequest
	var err error

	if jsonStr != "" || jsonFile != "" {
		output.PrintInfo("Using JSON input", noColor)
		req, err = buildUpdateIXRequestFromJSON(jsonStr, jsonFile)
		if err != nil {
			output.PrintError("Failed to process JSON input: %v", noColor, err)
			return err
		}
	} else if flagsProvided {
		output.PrintInfo("Using flag input", noColor)
		req, err = buildUpdateIXRequestFromFlags(cmd)
		if err != nil {
			output.PrintError("Failed to process flag input: %v", noColor, err)
			return err
		}
	} else if interactive {
		req, err = buildUpdateIXRequestFromPrompt(ixUID, noColor)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("at least one field must be updated")
	}

	req.WaitForUpdate = true
	req.WaitForTime = utils.DefaultProvisionTimeout

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}

	originalIX, err := getIXFunc(ctx, client, ixUID)
	if err != nil {
		output.PrintError("Failed to get original IX: %v", noColor, err)
		return err
	}

	updateSpinner := output.PrintResourceUpdating("IX", ixUID, noColor)
	var updatedIX *megaport.IX
	err = utils.WithRetry(ctx, func(ctx context.Context) error {
		var e error
		updatedIX, e = updateIXFunc(ctx, client, ixUID, req)
		return e
	})
	updateSpinner.Stop()

	if err != nil {
		output.PrintError("Failed to update IX: %v", noColor, err)
		return err
	}

	output.PrintResourceUpdated("IX", ixUID, noColor)

	displayIXChanges(originalIX, updatedIX, noColor)

	return nil
}

func DeleteIX(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}

	ixUID := args[0]

	later, err := cmd.Flags().GetBool("later")
	if err != nil {
		return err
	}
	deleteNow := !later

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}
	if confirmed, err := utils.ConfirmDelete(utils.ResourceTypeIX, ixUID, force, noColor); !confirmed {
		return err
	}

	deleteRequest := &megaport.DeleteIXRequest{
		DeleteNow: deleteNow,
	}

	spinner := output.PrintResourceDeleting("IX", ixUID, noColor)

	err = utils.WithRetry(ctx, func(ctx context.Context) error {
		return deleteIXFunc(ctx, client, ixUID, deleteRequest)
	})

	spinner.Stop()

	if err != nil {
		err = utils.WrapAPIError(err, "IX", ixUID)
		return fmt.Errorf("failed to delete IX: %w", err)
	}

	output.PrintResourceDeleted("IX", ixUID, deleteNow, noColor)

	return nil
}
