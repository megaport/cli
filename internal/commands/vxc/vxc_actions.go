package vxc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/megaport/megaport-cli/internal/base/cmdbuilder"
	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/config"
	"github.com/megaport/megaport-cli/internal/utils"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
)

func exportVXCConfig(vxc *megaport.VXC) map[string]interface{} {
	m := map[string]interface{}{
		"portUid":   vxc.AEndConfiguration.UID,
		"vxcName":   vxc.Name,
		"rateLimit": vxc.RateLimit,
		"term":      vxc.ContractTermMonths,
	}
	if vxc.CostCentre != "" {
		m["costCentre"] = vxc.CostCentre
	}

	aEnd := map[string]interface{}{}
	if vxc.AEndConfiguration.VLAN != 0 {
		aEnd["vlan"] = vxc.AEndConfiguration.VLAN
	}
	if vxc.AEndConfiguration.InnerVLAN != 0 {
		aEnd["innerVlan"] = vxc.AEndConfiguration.InnerVLAN
	}
	if len(aEnd) > 0 {
		m["aEndConfiguration"] = aEnd
	}

	bEnd := map[string]interface{}{
		"productUID": vxc.BEndConfiguration.UID,
	}
	if vxc.BEndConfiguration.VLAN != 0 {
		bEnd["vlan"] = vxc.BEndConfiguration.VLAN
	}
	if vxc.BEndConfiguration.InnerVLAN != 0 {
		bEnd["innerVlan"] = vxc.BEndConfiguration.InnerVLAN
	}
	m["bEndConfiguration"] = bEnd

	return m
}

func ListVXCs(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	// Flag read errors are intentionally ignored — flags are registered by the command builder.
	name, _ := cmd.Flags().GetString("name")
	nameContains, _ := cmd.Flags().GetString("name-contains")
	rateLimit, _ := cmd.Flags().GetInt("rate-limit")
	aEndUID, _ := cmd.Flags().GetString("a-end-uid")
	bEndUID, _ := cmd.Flags().GetString("b-end-uid")
	includeInactive, _ := cmd.Flags().GetBool("include-inactive")
	statusStr, _ := cmd.Flags().GetString("status")

	// Determine server-side name filter: --name-contains takes precedence, else --name
	serverNameContains := nameContains
	if serverNameContains == "" {
		serverNameContains = name
	}

	// Parse comma-separated status filter
	var statusFilter []string
	if statusStr != "" {
		for _, s := range strings.Split(statusStr, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				statusFilter = append(statusFilter, s)
			}
		}
	}

	req := &megaport.ListVXCsRequest{
		IncludeInactive: includeInactive,
		NameContains:    serverNameContains,
		AEndProductUID:  aEndUID,
		BEndProductUID:  bEndUID,
		RateLimit:       rateLimit,
		Status:          statusFilter,
	}

	spinner := output.PrintResourceListing("VXC", noColor)

	vxcs, err := client.VXCService.ListVXCs(ctx, req)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to list VXCs: %v", noColor, err)
		return fmt.Errorf("failed to list VXCs: %w", err)
	}

	// Scrub nils and, unless --include-inactive, remove decommissioned/cancelled entries.
	// Name, a-end, b-end, rate-limit, and status are already filtered by ListVXCsRequest.
	var filteredVXCs []*megaport.VXC
	for _, vxc := range vxcs {
		if vxc == nil {
			continue
		}
		if !includeInactive &&
			(vxc.ProvisioningStatus == megaport.STATUS_DECOMMISSIONED ||
				vxc.ProvisioningStatus == megaport.STATUS_CANCELLED ||
				vxc.ProvisioningStatus == utils.StatusDecommissioning) {
			continue
		}
		filteredVXCs = append(filteredVXCs, vxc)
	}

	limit, _ := cmd.Flags().GetInt("limit") // applied client-side after fetch

	tagFilters, _ := cmd.Flags().GetStringArray("tag")
	if len(tagFilters) > 0 {
		tagSpinner := output.PrintCustomSpinner("Fetching tags for", "VXCs", noColor)
		var tagErrs map[string]error
		filteredVXCs, tagErrs = utils.ApplyTagFilter(ctx, filteredVXCs,
			func(v *megaport.VXC) string { return v.UID },
			func(ctx context.Context, uid string) (map[string]string, error) {
				return listVXCResourceTagsFunc(ctx, client, uid)
			},
			tagFilters, limit,
		)
		tagSpinner.Stop()
		if err := ctx.Err(); err != nil {
			return err
		}
		for uid, err := range tagErrs {
			output.PrintWarning("Failed to fetch tags for VXC %s, skipping: %v", noColor, uid, err)
		}
	}
	return utils.ApplyLimitAndPrint(filteredVXCs, limit, outputFormat, noColor,
		"No VXCs found. Create one with 'megaport vxc buy'.", printVXCs)
}

func GetVXC(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	watch, _ := cmd.Flags().GetBool("watch")
	if watch {
		return watchGetVXC(cmd, args, noColor, outputFormat)
	}

	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	vxcUID := args[0]

	spinner := output.PrintResourceGetting("VXC", vxcUID, noColor)

	vxc, err := client.VXCService.GetVXC(ctx, vxcUID)

	spinner.Stop()

	if err != nil {
		err = utils.WrapAPIError(err, "VXC", vxcUID)
		output.PrintError("Failed to get VXC: %v", noColor, err)
		return fmt.Errorf("failed to get VXC: %w", err)
	}

	if vxc == nil {
		output.PrintError("No VXC found with UID: %s", noColor, vxcUID)
		return fmt.Errorf("no VXC found with UID: %s", vxcUID)
	}

	export, _ := cmd.Flags().GetBool("export")
	if export {
		cfg := exportVXCConfig(vxc)
		jsonBytes, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal export config: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	err = printVXCs([]*megaport.VXC{vxc}, outputFormat, noColor)
	if err != nil {
		output.PrintError("Failed to print VXCs: %v", noColor, err)
		return fmt.Errorf("failed to print VXCs: %w", err)
	}
	return nil
}

func watchGetVXC(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	vxcUID := args[0]
	return utils.WatchResource(cmd, "VXC", vxcUID, noColor, outputFormat, config.Login,
		func(pollCtx context.Context, client *megaport.Client) (string, error) {
			vxc, err := client.VXCService.GetVXC(pollCtx, vxcUID)
			if err != nil {
				return "", err
			}
			if vxc == nil {
				return "", fmt.Errorf("no VXC found with UID: %s", vxcUID)
			}
			err = printVXCs([]*megaport.VXC{vxc}, outputFormat, noColor)
			return vxc.ProvisioningStatus, err
		})
}

// emptyMeansUnsetUpdateFlags are flags whose update builder skips an empty
// value, so an explicit empty string carries no change and must not pass the
// gate (buildUpdateVXCRequestFromFlags). Other string flags treat "" as a
// meaningful value (e.g. clearing the cost centre).
var emptyMeansUnsetUpdateFlags = map[string]bool{
	"a-end-partner-config": true,
	"b-end-partner-config": true,
}

var hasUpdateVXCNonInteractiveFlags = func(cmd *cobra.Command) bool {
	for _, name := range cmdbuilder.VXCUpdateFlagNames {
		if !cmd.Flags().Changed(name) {
			continue
		}
		if emptyMeansUnsetUpdateFlags[name] {
			if v, _ := cmd.Flags().GetString(name); v == "" {
				continue
			}
		}
		return true
	}
	return false
}

func buildVXCRequest(cmd *cobra.Command, ctx context.Context, client *megaport.Client, noColor bool) (*megaport.BuyVXCRequest, error) {
	return utils.ResolveInput(utils.InputConfig[*megaport.BuyVXCRequest]{
		ResourceName: "VXC",
		Cmd:          cmd,
		NoColor:      noColor,
		FlagsProvided: func() bool {
			return cmd.Flags().Changed("name") || cmd.Flags().Changed("rate-limit") ||
				cmd.Flags().Changed("term") || cmd.Flags().Changed("a-end-uid") ||
				cmd.Flags().Changed("a-end-vlan") || cmd.Flags().Changed("b-end-uid") ||
				cmd.Flags().Changed("b-end-vlan")
		},
		FromJSON: buildVXCRequestFromJSON,
		FromFlags: func() (*megaport.BuyVXCRequest, error) {
			return buildVXCRequestFromFlags(cmd, ctx, client.VXCService)
		},
		FromPrompt: func() (*megaport.BuyVXCRequest, error) {
			return buildVXCRequestFromPrompt(ctx, client.VXCService, noColor)
		},
	})
}

func BuyVXC(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel, client, err := utils.LoginClient(cmd, utils.DefaultMutationTimeout, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	req, err := buildVXCRequest(cmd, ctx, client, noColor)
	if err != nil {
		return err
	}

	yes, err := utils.RequireYesForJSONBuy(cmd)
	if err != nil {
		return err
	}

	noWait, _ := cmd.Flags().GetBool("no-wait")
	// Only the order submission is wrapped in WithOrderOnceRetry below, so the SDK
	// must not also poll for provisioning: a 429 raised during polling would
	// otherwise re-submit the order. Provisioning is awaited separately afterwards.
	req.WaitForProvision = false

	validateSpinner := output.PrintResourceValidating("VXC", noColor)

	err = client.VXCService.ValidateVXCOrder(ctx, req)
	validateSpinner.Stop()

	if err != nil {
		output.PrintError("Failed to validate VXC order: %v", noColor, err)
		return err
	}

	if !yes {
		details := []utils.BuyConfirmDetail{
			{Key: "Name", Value: req.VXCName},
			{Key: "Term", Value: fmt.Sprintf("%d months", req.Term)},
			{Key: "Rate Limit", Value: fmt.Sprintf("%d Mbps", req.RateLimit)},
			{Key: "A-End Port UID", Value: req.PortUID},
		}
		if !utils.BuyConfirmPrompt("VXC", details, noColor) {
			output.PrintInfo("Purchase cancelled", noColor)
			return exitcodes.New(exitcodes.Cancelled, fmt.Errorf("cancelled by user"))
		}
	}

	var spinner *output.Spinner
	if noWait {
		spinner = output.PrintResourceCreating("VXC", req.VXCName, noColor)
	} else {
		spinner = output.PrintResourceProvisioning("VXC", req.VXCName, noColor)
	}

	var resp *megaport.BuyVXCResponse
	err = utils.WithOrderOnceRetry(ctx, func(ctx context.Context) error {
		var e error
		resp, e = buyVXCFunc(ctx, client, req)
		return e
	})
	if err != nil {
		spinner.Stop()
		output.PrintError("Failed to buy VXC: %v", noColor, err)
		return err
	}

	if resp == nil {
		spinner.Stop()
		output.PrintError("VXC buy returned an empty API response", noColor)
		return fmt.Errorf("empty response from API")
	}

	if resp.TechnicalServiceUID == "" {
		spinner.Stop()
		output.PrintError("VXC created but no UID returned", noColor)
		return fmt.Errorf("VXC created but no UID returned")
	}

	uid := resp.TechnicalServiceUID

	if !noWait {
		if err := waitForVXCProvision(ctx, client, req.VXCName, uid); err != nil {
			spinner.Stop()
			output.PrintError("VXC %s failed to provision: %v", noColor, uid, err)
			return err
		}
	}

	spinner.Stop()
	output.PrintResourceCreated("VXC", uid, noColor)
	return nil
}

// waitForVXCProvision polls the VXC's status until it is provisioned, bounding
// the wait by DefaultProvisionTimeout. It runs after the order is placed, so it
// must never be wrapped in an order-submission retry.
func waitForVXCProvision(ctx context.Context, client *megaport.Client, name, uid string) error {
	pollCtx, cancel := context.WithTimeout(ctx, utils.DefaultProvisionTimeout)
	defer cancel()
	return utils.WaitForProvision(pollCtx, "VXC", name, uid, func(ctx context.Context) (string, error) {
		v, err := getVXCFunc(ctx, client, uid)
		if err != nil {
			return "", err
		}
		if v == nil {
			return "", fmt.Errorf("VXC %s not found while waiting for provisioning", uid)
		}
		return v.ProvisioningStatus, nil
	})
}

func ValidateVXC(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	req, err := buildVXCRequest(cmd, ctx, client, noColor)
	if err != nil {
		return err
	}

	spinner := output.PrintResourceValidating("VXC", noColor)
	err = client.VXCService.ValidateVXCOrder(ctx, req)
	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to validate VXC order: %v", noColor, err)
		return err
	}

	output.PrintSuccess("VXC validation passed", noColor)
	return nil
}

func UpdateVXC(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel := utils.ContextFromCmdWithDefault(cmd, utils.DefaultMutationTimeout)
	defer cancel()

	vxcUID := args[0]
	formattedUID := output.FormatUID(vxcUID, noColor)

	interactive, _ := cmd.Flags().GetBool("interactive")
	jsonStr, _ := cmd.Flags().GetString("json")
	jsonFilePath, _ := cmd.Flags().GetString("json-file")

	// Reject conflicting input modes before logging in, matching the other
	// update commands: a usage error must surface up front, not after a
	// login + GET round-trip.
	if err := utils.CheckInteractiveConflict(interactive, utils.HasConflictingInputFlags(cmd)); err != nil {
		output.PrintError("%v", noColor, err)
		return err
	}

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}

	getSpinner := output.PrintResourceGetting("VXC", vxcUID, noColor)

	originalVXC, err := client.VXCService.GetVXC(ctx, vxcUID)

	getSpinner.Stop()

	if err != nil {
		output.PrintError("Failed to retrieve original VXC details: %v", noColor, err)
		return fmt.Errorf("failed to retrieve original VXC details: %w", err)
	}

	var req *megaport.UpdateVXCRequest
	var buildErr error

	if jsonStr != "" || jsonFilePath != "" {
		output.PrintInfo("Using JSON input for VXC %s", noColor, formattedUID)
		req, buildErr = buildUpdateVXCRequestFromJSON(jsonStr, jsonFilePath)
	} else if hasUpdateVXCNonInteractiveFlags(cmd) {
		output.PrintInfo("Using flag input for VXC %s", noColor, formattedUID)
		req, buildErr = buildUpdateVXCRequestFromFlags(cmd)
	} else if interactive {
		output.PrintInfo("Starting interactive mode for VXC %s", noColor, formattedUID)
		req, buildErr = buildUpdateVXCRequestFromPrompt(ctx, client, vxcUID, noColor)
	} else {
		return fmt.Errorf("at least one field must be updated")
	}

	if buildErr != nil {
		output.PrintError("Failed to build update request: %v", noColor, buildErr)
		return buildErr
	}

	if req == nil {
		output.PrintError("No update parameters provided", noColor)
		return fmt.Errorf("no update parameters provided")
	}

	req.WaitForUpdate = true
	req.WaitForTime = utils.DefaultProvisionTimeout

	updateSpinner := output.PrintResourceUpdating("VXC", vxcUID, noColor)

	err = utils.WithRetry(ctx, func(ctx context.Context) error {
		return updateVXCFunc(ctx, client, vxcUID, req)
	})

	updateSpinner.Stop()

	if err != nil {
		output.PrintError("Failed to update VXC: %v", noColor, err)
		return fmt.Errorf("failed to update VXC: %w", err)
	}

	getUpdatedSpinner := output.PrintResourceGetting("VXC", vxcUID, noColor)

	updatedVXC, err := getVXCFunc(ctx, client, vxcUID)

	getUpdatedSpinner.Stop()

	if err != nil {
		output.PrintError("VXC was updated but failed to retrieve updated details: %v", noColor, err)
		output.PrintResourceUpdated("VXC", vxcUID, noColor)
		return nil
	}

	output.PrintResourceUpdated("VXC", vxcUID, noColor)

	displayVXCChanges(originalVXC, updatedVXC, noColor)

	return nil
}

func DeleteVXC(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel := utils.ContextFromCmdWithDefault(cmd, 5*time.Minute)
	defer cancel()

	vxcUID := args[0]

	later, err := cmd.Flags().GetBool("later")
	if err != nil {
		output.PrintError("Failed to get later flag: %v", noColor, err)
		return err
	}
	deleteNow := !later

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		output.PrintError("Failed to get force flag: %v", noColor, err)
		return err
	}

	if confirmed, err := utils.ConfirmDelete(utils.ResourceTypeVXC, vxcUID, force, noColor); !confirmed {
		return err
	}

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}

	req := &megaport.DeleteVXCRequest{
		DeleteNow: deleteNow,
	}

	spinner := output.PrintResourceDeleting("VXC", vxcUID, noColor)

	err = utils.WithRetry(ctx, func(ctx context.Context) error {
		return deleteVXCFunc(ctx, client, vxcUID, req)
	})

	spinner.Stop()

	if err != nil {
		err = utils.WrapAPIError(err, "VXC", vxcUID)
		output.PrintError("Failed to delete VXC: %v", noColor, err)
		return err
	}

	output.PrintResourceDeleted("VXC", vxcUID, deleteNow, noColor)
	return nil
}

func ListVXCResourceTags(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	vxcUID := args[0]
	return utils.ListResourceTags("VXC", vxcUID, noColor, outputFormat, func(ctx context.Context, uid string) (map[string]string, error) {
		client, err := config.Login(ctx)
		if err != nil {
			return nil, err
		}
		return client.VXCService.ListVXCResourceTags(ctx, uid)
	})
}

func UpdateVXCResourceTags(cmd *cobra.Command, args []string, noColor bool) error {
	vxcUID := args[0]
	var client *megaport.Client
	login := func(ctx context.Context) error {
		var err error
		client, err = config.Login(ctx)
		return err
	}
	return utils.UpdateResourceTags(utils.UpdateTagsOptions{
		ResourceType: "VXC",
		UID:          vxcUID,
		NoColor:      noColor,
		Cmd:          cmd,
		ListFunc: func(ctx context.Context, uid string) (map[string]string, error) {
			if err := login(ctx); err != nil {
				return nil, err
			}
			return client.VXCService.ListVXCResourceTags(ctx, uid)
		},
		UpdateFunc: func(ctx context.Context, uid string, tags map[string]string) error {
			return client.VXCService.UpdateVXCResourceTags(ctx, uid, tags)
		},
	})
}

func GetVXCStatus(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	watch, _ := cmd.Flags().GetBool("watch")
	if watch {
		return watchVXCStatus(cmd, args, noColor, outputFormat)
	}

	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	vxcUID := args[0]

	spinner := output.PrintResourceGetting("VXC", vxcUID, noColor)

	vxc, err := client.VXCService.GetVXC(ctx, vxcUID)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to get VXC status: %v", noColor, err)
		return fmt.Errorf("failed to get VXC status: %w", err)
	}

	if vxc == nil {
		output.PrintError("No VXC found with UID: %s", noColor, vxcUID)
		return fmt.Errorf("no VXC found with UID: %s", vxcUID)
	}

	status := []VXCStatus{
		{
			UID:    vxc.UID,
			Name:   vxc.Name,
			Status: vxc.ProvisioningStatus,
			Type:   vxc.Type,
		},
	}

	return output.PrintOutput(status, outputFormat, noColor)
}

func watchVXCStatus(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	vxcUID := args[0]
	return utils.WatchResource(cmd, "VXC", vxcUID, noColor, outputFormat, config.Login,
		func(pollCtx context.Context, client *megaport.Client) (string, error) {
			vxc, err := client.VXCService.GetVXC(pollCtx, vxcUID)
			if err != nil {
				return "", err
			}
			if vxc == nil {
				return "", fmt.Errorf("no VXC found with UID: %s", vxcUID)
			}
			status := []VXCStatus{
				{
					UID:    vxc.UID,
					Name:   vxc.Name,
					Status: vxc.ProvisioningStatus,
					Type:   vxc.Type,
				},
			}
			err = output.PrintOutput(status, outputFormat, noColor)
			return vxc.ProvisioningStatus, err
		})
}
