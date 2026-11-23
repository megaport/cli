package mcr

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

func CreateMCRPrefixFilterList(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	mcrUID := args[0]

	interactive, _ := cmd.Flags().GetBool("interactive")
	jsonStr, _ := cmd.Flags().GetString("json")
	jsonFile, _ := cmd.Flags().GetString("json-file")

	flagsProvided := cmd.Flags().Changed("description") || cmd.Flags().Changed("address-family") ||
		cmd.Flags().Changed("entries")

	if err := utils.CheckInteractiveConflict(interactive, utils.HasConflictingInputFlags(cmd)); err != nil {
		output.PrintError("%v", noColor, err)
		return err
	}

	var req *megaport.CreateMCRPrefixFilterListRequest
	var err error

	if jsonStr != "" || jsonFile != "" {
		output.PrintInfo("Using JSON input", noColor)
		req, err = processJSONPrefixFilterListInput(jsonStr, jsonFile, mcrUID)
		if err != nil {
			output.PrintError("Failed to process JSON input: %v", noColor, err)
			return err
		}
	} else if flagsProvided {
		output.PrintInfo("Using flag input", noColor)
		req, err = processFlagPrefixFilterListInput(cmd, mcrUID)
		if err != nil {
			output.PrintError("Failed to process flag input: %v", noColor, err)
			return err
		}
	} else if interactive {
		req, err = promptForPrefixFilterListDetails(mcrUID, noColor)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("no input provided, use --interactive, --json, or flags to specify prefix filter list details")
	}

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	spinner := output.PrintResourceCreating("Prefix Filter List", req.PrefixFilterList.Description, noColor)
	var resp *megaport.CreateMCRPrefixFilterListResponse
	err = utils.WithOrderOnceRetry(ctx, func(ctx context.Context) error {
		var e error
		resp, e = createMCRPrefixFilterListFunc(ctx, client, req)
		return e
	})
	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to create prefix filter list: %v", noColor, err)
		return err
	}

	output.PrintSuccess("Prefix filter list created successfully - ID: %d", noColor, resp.PrefixFilterListID)
	return nil
}

func UpdateMCRPrefixFilterList(cmd *cobra.Command, args []string, noColor bool) error {
	mcrUID := args[0]
	prefixFilterListID, err := validation.ParseInt("prefix filter list ID", args[1])
	if err != nil {
		return err
	}

	interactive, _ := cmd.Flags().GetBool("interactive")
	jsonStr, _ := cmd.Flags().GetString("json")
	jsonFile, _ := cmd.Flags().GetString("json-file")

	flagsProvided := cmd.Flags().Changed("description") || cmd.Flags().Changed("address-family") ||
		cmd.Flags().Changed("entries")

	if err := utils.CheckInteractiveConflict(interactive, utils.HasConflictingInputFlags(cmd)); err != nil {
		output.PrintError("%v", noColor, err)
		return err
	}

	// Validate input mode before logging in.
	if jsonStr == "" && jsonFile == "" && !flagsProvided && !interactive {
		return fmt.Errorf("at least one field must be updated")
	}

	// Login once — the client is reused for both prompts and the API mutation.
	_, loginCancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	loginCancel()

	var prefixFilterList *megaport.MCRPrefixFilterList
	var getErr error

	if jsonStr != "" || jsonFile != "" {
		output.PrintInfo("Using JSON input", noColor)
		prefixFilterList, getErr = processJSONUpdatePrefixFilterListInput(jsonStr, jsonFile, mcrUID, prefixFilterListID)
		if getErr != nil {
			output.PrintError("Failed to process JSON input: %v", noColor, getErr)
			return getErr
		}
	} else if flagsProvided {
		output.PrintInfo("Using flag input", noColor)
		prefixFilterList, getErr = processFlagUpdatePrefixFilterListInput(cmd, mcrUID, prefixFilterListID)
		if getErr != nil {
			output.PrintError("Failed to process flag input: %v", noColor, getErr)
			return getErr
		}
	} else if interactive {
		// Use cmd.Context() for API work during interactive input so it
		// remains cancellable. A fresh timed context is created below for
		// the mutation so user think-time doesn't consume the update timeout.
		prefixFilterList, getErr = promptForUpdatePrefixFilterListDetails(cmd.Context(), client, mcrUID, prefixFilterListID, noColor)
		if getErr != nil {
			return getErr
		}
	}

	// Fresh timed context for the API mutation (not consumed by prompt time).
	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	spinner := output.PrintResourceUpdating("Prefix Filter List", fmt.Sprintf("%d", prefixFilterListID), noColor)
	var resp *megaport.ModifyMCRPrefixFilterListResponse
	err = utils.WithRetry(ctx, func(ctx context.Context) error {
		var e error
		resp, e = modifyMCRPrefixFilterListFunc(ctx, client, mcrUID, prefixFilterListID, prefixFilterList)
		return e
	})
	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to update prefix filter list: %v", noColor, err)
		return err
	}

	if !resp.IsUpdated {
		output.PrintError("Prefix filter list update request was not successful for ID %d", noColor, prefixFilterListID)
		return fmt.Errorf("prefix filter list update request was not successful for ID %d", prefixFilterListID)
	}

	output.PrintSuccess("Prefix filter list updated successfully - ID: %d", noColor, prefixFilterListID)
	return nil
}

func ListMCRPrefixFilterLists(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	// Set output format for proper JSON mode handling
	output.SetOutputFormat(outputFormat)

	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	mcrUID := args[0]

	spinner := output.PrintResourceListing("Prefix filter list", noColor)

	prefixFilterLists, err := listMCRPrefixFilterListsFunc(ctx, client, mcrUID)

	spinner.Stop()

	if err != nil {
		return fmt.Errorf("failed to list prefix filter lists: %w", err)
	}

	err = output.PrintOutput(prefixFilterLists, outputFormat, noColor)
	if err != nil {
		return fmt.Errorf("failed to print prefix filter lists: %w", err)
	}
	return nil
}

func GetMCRPrefixFilterList(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	// Set output format for proper JSON mode handling
	output.SetOutputFormat(outputFormat)

	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	mcrUID := args[0]
	prefixFilterListID, err := validation.ParseInt("prefix filter list ID", args[1])
	if err != nil {
		return err
	}

	spinner := output.PrintResourceGetting("Prefix filter list", fmt.Sprintf("%d", prefixFilterListID), noColor)

	prefixFilterList, err := getMCRPrefixFilterListFunc(ctx, client, mcrUID, prefixFilterListID)

	spinner.Stop()

	if err != nil {
		return fmt.Errorf("failed to get prefix filter list: %w", err)
	}

	op, err := toPrefixFilterListOutput(prefixFilterList)
	if err != nil {
		return fmt.Errorf("failed to convert prefix filter list: %w", err)
	}

	err = output.PrintOutput([]prefixFilterListOutput{op}, outputFormat, noColor)
	if err != nil {
		return fmt.Errorf("failed to print prefix filter list: %w", err)
	}
	return nil
}

func DeleteMCRPrefixFilterList(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	mcrUID := args[0]
	prefixFilterListID, err := validation.ParseInt("prefix filter list ID", args[1])
	if err != nil {
		return err
	}

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}
	if confirmed, err := utils.ConfirmDelete(utils.ResourceTypePrefixFilterList, fmt.Sprintf("%d", prefixFilterListID), force, noColor); !confirmed {
		return err
	}

	spinner := output.PrintResourceDeleting("Prefix filter list", fmt.Sprintf("%d", prefixFilterListID), noColor)

	var resp *megaport.DeleteMCRPrefixFilterListResponse
	err = utils.WithRetry(ctx, func(ctx context.Context) error {
		var e error
		resp, e = deleteMCRPrefixFilterListFunc(ctx, client, mcrUID, prefixFilterListID)
		return e
	})

	spinner.Stop()

	if err != nil {
		return fmt.Errorf("failed to delete prefix filter list: %w", err)
	}

	if !resp.IsDeleted {
		output.PrintError("Prefix filter list deletion request was not successful for ID %d", noColor, prefixFilterListID)
		return fmt.Errorf("prefix filter list deletion request was not successful for ID %d", prefixFilterListID)
	}

	output.PrintSuccess("Prefix filter list deleted successfully - ID: %d", noColor, prefixFilterListID)
	return nil
}
