package users

import (
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

func buildCreateUserRequest(cmd *cobra.Command, noColor bool) (*megaport.CreateUserRequest, error) {
	// Flag read errors are intentionally ignored — flags are registered by the command builder.
	interactive, _ := cmd.Flags().GetBool("interactive")
	jsonStr, _ := cmd.Flags().GetString("json")
	jsonFile, _ := cmd.Flags().GetString("json-file")

	flagsProvided := cmd.Flags().Changed("first-name") || cmd.Flags().Changed("last-name") ||
		cmd.Flags().Changed("email") || cmd.Flags().Changed("position")

	if err := utils.CheckInteractiveConflict(interactive, utils.HasConflictingInputFlags(cmd)); err != nil {
		output.PrintError("%v", noColor, err)
		return nil, err
	}

	if jsonStr != "" || jsonFile != "" {
		output.PrintInfo("Using JSON input", noColor)
		req, err := processJSONCreateUserInput(jsonStr, jsonFile)
		if err != nil {
			output.PrintError("Failed to process JSON input: %v", noColor, err)
			return nil, err
		}
		return req, nil
	} else if flagsProvided {
		output.PrintInfo("Using flag input", noColor)
		req, err := processFlagCreateUserInput(cmd)
		if err != nil {
			output.PrintError("Failed to process flag input: %v", noColor, err)
			return nil, err
		}
		return req, nil
	} else if interactive {
		output.PrintInfo("Starting interactive mode", noColor)
		req, err := promptForCreateUserDetails(noColor)
		if err != nil {
			output.PrintError("Interactive input failed: %v", noColor, err)
			return nil, err
		}
		return req, nil
	}
	output.PrintError("No input provided", noColor)
	return nil, fmt.Errorf("no input provided, use --interactive, --json, or flags to specify user details")
}

func buildUpdateUserRequest(cmd *cobra.Command, noColor bool) (*megaport.UpdateUserRequest, error) {
	interactive, _ := cmd.Flags().GetBool("interactive")
	jsonStr, _ := cmd.Flags().GetString("json")
	jsonFile, _ := cmd.Flags().GetString("json-file")

	flagsProvided := cmd.Flags().Changed("first-name") || cmd.Flags().Changed("last-name") ||
		cmd.Flags().Changed("email") || cmd.Flags().Changed("position") ||
		cmd.Flags().Changed("phone") || cmd.Flags().Changed("active") ||
		cmd.Flags().Changed("notification-enabled")

	if err := utils.CheckInteractiveConflict(interactive, utils.HasConflictingInputFlags(cmd)); err != nil {
		output.PrintError("%v", noColor, err)
		return nil, err
	}

	if jsonStr != "" || jsonFile != "" {
		output.PrintInfo("Using JSON input", noColor)
		req, err := processJSONUpdateUserInput(jsonStr, jsonFile)
		if err != nil {
			output.PrintError("Failed to process JSON input: %v", noColor, err)
			return nil, err
		}
		return req, nil
	} else if flagsProvided {
		output.PrintInfo("Using flag input", noColor)
		req, err := processFlagUpdateUserInput(cmd)
		if err != nil {
			output.PrintError("Failed to process flag input: %v", noColor, err)
			return nil, err
		}
		return req, nil
	} else if interactive {
		output.PrintInfo("Starting interactive mode", noColor)
		req, err := promptForUpdateUserDetails(noColor)
		if err != nil {
			output.PrintError("Interactive input failed: %v", noColor, err)
			return nil, err
		}
		return req, nil
	}
	output.PrintError("No input provided", noColor)
	return nil, fmt.Errorf("no input provided, use --interactive, --json, or flags to specify update details")
}

func ListUsers(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)
	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	spinner := output.PrintResourceListing("user", noColor)

	users, err := listCompanyUsersFunc(ctx, client)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to list users: %v", noColor, err)
		return fmt.Errorf("failed to list users: %w", err)
	}

	position, _ := cmd.Flags().GetString("position")
	activeOnly, _ := cmd.Flags().GetBool("active-only")
	inactiveOnly, _ := cmd.Flags().GetBool("inactive-only")

	filtered := filterUsers(users, position, activeOnly, inactiveOnly)

	return printUsers(filtered, outputFormat, noColor)
}

func GetUser(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	employeeID, err := validation.ParseInt("employee ID", args[0])
	if err != nil {
		output.PrintError("%v", noColor, err)
		return err
	}

	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	spinner := output.PrintResourceGetting("User", args[0], noColor)

	user, err := getUserFunc(ctx, client, employeeID)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to get user: %v", noColor, err)
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		output.PrintError("No user found with employee ID: %s", noColor, args[0])
		return fmt.Errorf("no user found with employee ID: %s", args[0])
	}

	return printUsers([]*megaport.User{user}, outputFormat, noColor)
}

func CreateUser(cmd *cobra.Command, args []string, noColor bool) error {
	req, err := buildCreateUserRequest(cmd, noColor)
	if err != nil {
		return err
	}

	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	spinner := output.PrintResourceCreating("User", req.FirstName+" "+req.LastName, noColor)

	resp, err := createUserFunc(ctx, client, req)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to create user: %v", noColor, err)
		return err
	}

	if resp == nil {
		output.PrintError("User create returned an empty API response", noColor)
		return fmt.Errorf("empty response from API")
	}

	output.PrintResourceCreated("User", strconv.Itoa(resp.EmployeeID), noColor)
	return nil
}

func UpdateUser(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	employeeID, err := validation.ParseInt("employee ID", args[0])
	if err != nil {
		output.PrintError("%v", noColor, err)
		return err
	}

	req, err := buildUpdateUserRequest(cmd, noColor)
	if err != nil {
		return err
	}

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}

	originalUser, err := getUserFunc(ctx, client, employeeID)
	if err != nil {
		output.PrintError("Failed to get current user: %v", noColor, err)
		return fmt.Errorf("failed to get current user: %w", err)
	}

	spinner := output.PrintResourceUpdating("User", args[0], noColor)

	err = updateUserFunc(ctx, client, employeeID, req)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to update user: %v", noColor, err)
		return err
	}

	output.PrintResourceUpdated("User", args[0], noColor)

	updatedUser, getErr := getUserFunc(ctx, client, employeeID)
	if getErr == nil && updatedUser != nil {
		displayUserChanges(originalUser, updatedUser, noColor)
	}
	return nil
}

func DeleteUser(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	employeeID, err := validation.ParseInt("employee ID", args[0])
	if err != nil {
		output.PrintError("%v", noColor, err)
		return err
	}

	force, _ := cmd.Flags().GetBool("force")
	if !force {
		confirmMsg := fmt.Sprintf("Are you sure you want to delete user with employee ID %d? ", employeeID)
		if !utils.ConfirmPrompt(confirmMsg, noColor) {
			output.PrintInfo("Deletion cancelled", noColor)
			return exitcodes.New(exitcodes.Cancelled, fmt.Errorf("cancelled by user"))
		}
	}

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}

	spinner := output.PrintResourceDeleting("User", args[0], noColor)

	err = deleteUserFunc(ctx, client, employeeID)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to delete user: %v", noColor, err)
		return err
	}

	output.PrintResourceDeleted("User", args[0], noColor)
	return nil
}

func DeactivateUser(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	employeeID, err := validation.ParseInt("employee ID", args[0])
	if err != nil {
		output.PrintError("%v", noColor, err)
		return err
	}

	force, _ := cmd.Flags().GetBool("force")
	if !force {
		confirmMsg := fmt.Sprintf("Are you sure you want to deactivate user with employee ID %d? ", employeeID)
		if !utils.ConfirmPrompt(confirmMsg, noColor) {
			output.PrintInfo("Deactivation cancelled", noColor)
			return exitcodes.New(exitcodes.Cancelled, fmt.Errorf("cancelled by user"))
		}
	}

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}

	spinner := output.PrintResourceUpdating("User", args[0], noColor)

	err = deactivateUserFunc(ctx, client, employeeID)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to deactivate user: %v", noColor, err)
		return err
	}

	output.PrintSuccess("User deactivated successfully (employee ID: %s)", noColor, args[0])
	return nil
}

func GetUserActivity(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)
	ctx, cancel, client, err := utils.LoginClient(cmd, 90*time.Second, config.Login)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return err
	}
	defer cancel()

	req := &megaport.GetUserActivityRequest{}

	employeeID, _ := cmd.Flags().GetString("employee-id")
	if employeeID != "" {
		req.PersonIdOrUid = employeeID
	}

	spinner := output.PrintResourceListing("user activity", noColor)

	activities, err := getUserActivityFunc(ctx, client, req)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to get user activity: %v", noColor, err)
		return fmt.Errorf("failed to get user activity: %w", err)
	}

	return printUserActivities(activities, outputFormat, noColor)
}
