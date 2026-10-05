package mcr

import (
	"context"
	"fmt"
	"time"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/config"
	"github.com/megaport/megaport-cli/internal/utils"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
)

func DeleteMCR(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	mcrUID := args[0]

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}
	if confirmed, err := utils.ConfirmDelete(utils.ResourceTypeMCR, mcrUID, force, noColor); !confirmed {
		return err
	}

	safeDelete, err := cmd.Flags().GetBool("safe-delete")
	if err != nil {
		return fmt.Errorf("failed to get safe-delete flag: %w", err)
	}

	// MCRs only support immediate deletion (CANCEL_NOW); the SDK rejects
	// DeleteNow=false with ErrMCRCancelLaterNotAllowed.
	deleteRequest := &megaport.DeleteMCRRequest{
		MCRID:      mcrUID,
		DeleteNow:  true,
		SafeDelete: safeDelete,
	}

	spinner := output.PrintResourceDeleting("MCR", mcrUID, noColor)

	var resp *megaport.DeleteMCRResponse
	err = utils.WithRetry(ctx, func(ctx context.Context) error {
		var e error
		resp, e = deleteMCRFunc(ctx, client, deleteRequest)
		return e
	})

	spinner.Stop()

	if err != nil {
		err = utils.WrapAPIError(err, "MCR", mcrUID)
		return fmt.Errorf("failed to delete MCR: %w", err)
	}

	if !resp.IsDeleting {
		output.PrintError("MCR deletion request was not successful for %s", noColor, mcrUID)
		return fmt.Errorf("MCR deletion request was not successful for %s", mcrUID)
	}

	output.PrintResourceDeleted("MCR", mcrUID, noColor)
	return nil
}

func LockMCR(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	mcrUID := args[0]

	output.PrintInfo("Locking MCR %s...", noColor, mcrUID)

	err = utils.WithRetry(ctx, func(ctx context.Context) error {
		_, e := lockMCRFunc(ctx, client, mcrUID)
		return e
	})
	if err != nil {
		return fmt.Errorf("failed to lock MCR: %w", err)
	}

	output.PrintSuccess("MCR %s locked successfully", noColor, mcrUID)
	return nil
}

func UnlockMCR(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	mcrUID := args[0]

	output.PrintInfo("Unlocking MCR %s...", noColor, mcrUID)

	err = utils.WithRetry(ctx, func(ctx context.Context) error {
		_, e := unlockMCRFunc(ctx, client, mcrUID)
		return e
	})
	if err != nil {
		return fmt.Errorf("failed to unlock MCR: %w", err)
	}

	output.PrintSuccess("MCR %s unlocked successfully", noColor, mcrUID)
	return nil
}

func ListMCRResourceTags(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	mcrUID := args[0]
	return utils.ListResourceTags("MCR", mcrUID, noColor, outputFormat, func(ctx context.Context, uid string) (map[string]string, error) {
		client, err := config.Login(ctx)
		if err != nil {
			return nil, err
		}
		return client.MCRService.ListMCRResourceTags(ctx, uid)
	})
}

func UpdateMCRResourceTags(cmd *cobra.Command, args []string, noColor bool) error {
	mcrUID := args[0]
	var client *megaport.Client
	login := func(ctx context.Context) error {
		var err error
		client, err = config.Login(ctx)
		return err
	}
	return utils.UpdateResourceTags(utils.UpdateTagsOptions{
		ResourceType: "MCR",
		UID:          mcrUID,
		NoColor:      noColor,
		Cmd:          cmd,
		ListFunc: func(ctx context.Context, uid string) (map[string]string, error) {
			if err := login(ctx); err != nil {
				return nil, err
			}
			return client.MCRService.ListMCRResourceTags(ctx, uid)
		},
		UpdateFunc: func(ctx context.Context, uid string, tags map[string]string) error {
			return client.MCRService.UpdateMCRResourceTags(ctx, uid, tags)
		},
	})
}

func GetMCRStatus(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	watch, _ := cmd.Flags().GetBool("watch")
	if watch {
		return watchMCRStatus(cmd, args, noColor, outputFormat)
	}

	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	mcrUID := args[0]

	spinner := output.PrintResourceGetting("MCR", mcrUID, noColor)

	mcr, err := client.MCRService.GetMCR(ctx, mcrUID)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to get MCR status: %v", noColor, err)
		return fmt.Errorf("failed to get MCR status: %w", err)
	}

	if mcr == nil {
		output.PrintError("No MCR found with UID: %s", noColor, mcrUID)
		return fmt.Errorf("no MCR found with UID: %s", mcrUID)
	}

	status := []MCRStatus{
		{
			UID:    mcr.UID,
			Name:   mcr.Name,
			Status: mcr.ProvisioningStatus,
			ASN:    mcr.Resources.VirtualRouter.ASN,
			Speed:  mcr.PortSpeed,
		},
	}

	return output.PrintOutput(status, outputFormat, noColor)
}

func watchMCRStatus(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	mcrUID := args[0]
	return utils.WatchResource(cmd, "MCR", mcrUID, noColor, outputFormat, config.Login,
		func(pollCtx context.Context, client *megaport.Client) (string, error) {
			mcr, err := client.MCRService.GetMCR(pollCtx, mcrUID)
			if err != nil {
				return "", err
			}
			if mcr == nil {
				return "", fmt.Errorf("no MCR found with UID: %s", mcrUID)
			}
			status := []MCRStatus{
				{
					UID:    mcr.UID,
					Name:   mcr.Name,
					Status: mcr.ProvisioningStatus,
					ASN:    mcr.Resources.VirtualRouter.ASN,
					Speed:  mcr.PortSpeed,
				},
			}
			err = output.PrintOutput(status, outputFormat, noColor)
			return mcr.ProvisioningStatus, err
		})
}
