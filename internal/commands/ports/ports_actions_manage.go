package ports

import (
	"context"
	"fmt"
	"time"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/config"
	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/megaport/megaport-cli/internal/validation"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
)

func UpdatePort(cmd *cobra.Command, args []string, noColor bool) error {
	portUID := args[0]

	interactive, _ := cmd.Flags().GetBool("interactive")
	jsonStr, _ := cmd.Flags().GetString("json")
	jsonFile, _ := cmd.Flags().GetString("json-file")
	flagsProvided := cmd.Flags().Changed("name") || cmd.Flags().Changed("marketplace-visibility") ||
		cmd.Flags().Changed("cost-centre") || cmd.Flags().Changed("term")

	// Reject conflicting input modes before logging in, matching the other
	// update commands: a usage error must surface up front, not after login.
	if err := utils.CheckInteractiveConflict(interactive, utils.HasConflictingInputFlags(cmd)); err != nil {
		output.PrintError("%v", noColor, err)
		return err
	}

	usingJSON := jsonStr != "" || jsonFile != ""
	if !usingJSON && !flagsProvided && !interactive {
		return fmt.Errorf("at least one field must be updated")
	}

	ctx, cancel := utils.ContextFromCmdWithDefault(cmd, utils.DefaultMutationTimeout)
	defer cancel()

	// Build and validate flag/JSON input before any network round-trip so
	// malformed input fails fast. Interactive prompts need the port's current
	// values, so they are built after login below.
	var req *megaport.ModifyPortRequest
	var costCentreProvided bool
	var err error

	if usingJSON {
		output.PrintInfo("Using JSON input", noColor)
		req, costCentreProvided, err = processJSONUpdatePortInput(jsonStr, jsonFile)
		if err != nil {
			return fmt.Errorf("failed to process input: %w", err)
		}
		req.PortID = portUID
	} else if flagsProvided {
		output.PrintInfo("Using flag input", noColor)
		req, costCentreProvided, err = processFlagUpdatePortInput(cmd, portUID)
		if err != nil {
			return fmt.Errorf("failed to process input: %w", err)
		}
	}

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}

	getSpinner := output.PrintResourceGetting("Port", portUID, noColor)

	originalPort, err := getPortFunc(ctx, client, portUID)

	getSpinner.Stop()

	if err != nil {
		output.PrintError("Failed to retrieve original port: %v", noColor, err)
		return err
	}

	var currentCostCentre string
	if originalPort != nil {
		currentCostCentre = originalPort.CostCentre
	}

	if interactive {
		output.PrintInfo("Starting interactive mode", noColor)
		req, err = promptForUpdatePortDetails(portUID, currentCostCentre, noColor)
		if err != nil {
			return fmt.Errorf("failed to process input: %w", err)
		}
	} else if !costCentreProvided {
		req.CostCentre = currentCostCentre
	}

	req.WaitForUpdate = true
	req.WaitForTime = utils.DefaultProvisionTimeout

	updateSpinner := output.PrintResourceUpdating("Port", portUID, noColor)

	var resp *megaport.ModifyPortResponse
	err = utils.WithRetry(ctx, func(ctx context.Context) error {
		var e error
		resp, e = updatePortFunc(ctx, client, req)
		return e
	})

	updateSpinner.Stop()

	if err != nil {
		output.PrintError("Failed to update port: %v", noColor, err)
		return err
	}

	if !resp.IsUpdated {
		output.PrintError("Port update request was not successful", noColor)
		return fmt.Errorf("port update request was not successful")
	}

	output.PrintResourceUpdated("Port", portUID, noColor)

	getUpdatedSpinner := output.PrintResourceGetting("Port", portUID, noColor)

	updatedPort, err := getPortFunc(ctx, client, portUID)

	getUpdatedSpinner.Stop()

	if err != nil {
		output.PrintError("Port was updated but failed to retrieve updated details: %v", noColor, err)
		return nil
	}

	displayPortChanges(originalPort, updatedPort, noColor)

	return nil
}

func DeletePort(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	portUID := args[0]

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		output.PrintError("Failed to get force flag: %v", noColor, err)
		return err
	}
	if confirmed, err := utils.ConfirmDelete(utils.ResourceTypePort, portUID, force, noColor); !confirmed {
		return err
	}

	safeDelete, err := cmd.Flags().GetBool("safe-delete")
	if err != nil {
		output.PrintError("Failed to get safe-delete flag: %v", noColor, err)
		return err
	}

	// Ports only support immediate deletion (CANCEL_NOW). The previous
	// "terminate later" option is no longer offered by the Megaport API,
	// and DeletePort requests with DeleteNow=false are rejected by the SDK
	// with ErrPortCancelLaterNotAllowed.
	deleteRequest := &megaport.DeletePortRequest{
		PortID:     portUID,
		DeleteNow:  true,
		SafeDelete: safeDelete,
	}

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}

	spinner := output.PrintResourceDeleting("Port", portUID, noColor)

	var resp *megaport.DeletePortResponse
	err = utils.WithRetry(ctx, func(ctx context.Context) error {
		var e error
		resp, e = deletePortFunc(ctx, client, deleteRequest)
		return e
	})

	spinner.Stop()

	if err != nil {
		err = utils.WrapAPIError(err, "Port", portUID)
		output.PrintError("Failed to delete port: %v", noColor, err)
		return err
	}

	if !resp.IsDeleting {
		output.PrintError("Port deletion request was not successful for %s", noColor, portUID)
		return fmt.Errorf("port deletion request was not successful for %s", portUID)
	}

	output.PrintResourceDeleted("Port", portUID, noColor)
	return nil
}

func LockPort(cmd *cobra.Command, args []string, noColor bool) error {
	portUID := args[0]
	formattedUID := output.FormatUID(portUID, noColor)

	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	spinner := output.PrintResourceUpdating("Port", portUID, noColor)

	var resp *megaport.LockPortResponse
	err = utils.WithRetry(ctx, func(ctx context.Context) error {
		var e error
		resp, e = lockPortFunc(ctx, client, portUID)
		return e
	})

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to lock port: %v", noColor, err)
		return err
	}

	if !resp.IsLocking {
		output.PrintError("Port lock request was not successful for %s", noColor, portUID)
		return fmt.Errorf("port lock request was not successful for %s", portUID)
	}

	output.PrintInfo("Port %s locked successfully", noColor, formattedUID)
	return nil
}

func UnlockPort(cmd *cobra.Command, args []string, noColor bool) error {
	portUID := args[0]
	formattedUID := output.FormatUID(portUID, noColor)

	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	spinner := output.PrintResourceUpdating("Port", portUID, noColor)

	var resp *megaport.UnlockPortResponse
	err = utils.WithRetry(ctx, func(ctx context.Context) error {
		var e error
		resp, e = unlockPortFunc(ctx, client, portUID)
		return e
	})

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to unlock port: %v", noColor, err)
		return err
	}

	if !resp.IsUnlocking {
		output.PrintError("Port unlock request was not successful for %s", noColor, portUID)
		return fmt.Errorf("port unlock request was not successful for %s", portUID)
	}

	output.PrintInfo("Port %s unlocked successfully", noColor, formattedUID)
	return nil
}

func CheckPortVLANAvailability(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	portUID := args[0]
	vlan, err := validation.ParseInt("VLAN ID", args[1])
	if err != nil {
		output.PrintError("%v", noColor, err)
		return err
	}
	if err := validation.ValidatePortVLANAvailability(vlan); err != nil {
		output.PrintError("Invalid VLAN ID: %v", noColor, err)
		return err
	}

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}

	spinner := output.PrintResourceGetting("Port", portUID, noColor)

	available, err := checkPortVLANAvailabilityFunc(ctx, client, portUID, vlan)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to check VLAN availability: %v", noColor, err)
		return err
	}

	if available {
		output.PrintInfo("VLAN %d is available on port %s", noColor, vlan, output.FormatUID(portUID, noColor))
	} else {
		output.PrintWarning("VLAN %d is not available on port %s", noColor, vlan, output.FormatUID(portUID, noColor))
	}
	return nil
}

func ListPortResourceTags(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	portUID := args[0]
	return utils.ListResourceTags("Port", portUID, noColor, outputFormat, func(ctx context.Context, uid string) (map[string]string, error) {
		client, err := config.Login(ctx)
		if err != nil {
			return nil, err
		}
		return listPortResourceTagsFunc(ctx, client, uid)
	})
}

func UpdatePortResourceTags(cmd *cobra.Command, args []string, noColor bool) error {
	portUID := args[0]
	var client *megaport.Client
	login := func(ctx context.Context) error {
		var err error
		client, err = config.Login(ctx)
		return err
	}
	return utils.UpdateResourceTags(utils.UpdateTagsOptions{
		ResourceType:  "Port",
		UID:           portUID,
		NoColor:       noColor,
		Cmd:           cmd,
		ExtraTagFlags: true,
		ListFunc: func(ctx context.Context, uid string) (map[string]string, error) {
			if err := login(ctx); err != nil {
				return nil, err
			}
			return client.PortService.ListPortResourceTags(ctx, uid)
		},
		UpdateFunc: func(ctx context.Context, uid string, tags map[string]string) error {
			return client.PortService.UpdatePortResourceTags(ctx, uid, tags)
		},
	})
}
