package users

import (
	"github.com/megaport/megaport-cli/internal/base/cmdbuilder"
	"github.com/spf13/cobra"
)

// AddCommandsTo builds the users commands and adds them to the root command
func AddCommandsTo(rootCmd *cobra.Command) {
	usersCmd := cmdbuilder.NewCommand("users", "Manage users in the Megaport API").
		WithLongDesc("Manage users in the Megaport API.\n\nThis command groups operations related to user management. You can use the subcommands to list all company users, get details for a specific user, create a new user, update an existing user, delete a user, deactivate a user, and view user activity.").
		WithExample("megaport-cli users list").
		WithExample("megaport-cli users get 12345").
		WithExample("megaport-cli users create --interactive").
		WithExample("megaport-cli users update 12345 --first-name \"New Name\"").
		WithExample("megaport-cli users delete 12345 --force").
		WithExample("megaport-cli users deactivate 12345").
		WithExample("megaport-cli users activity").
		WithRootCmd(rootCmd).
		Build()

	listCmd := cmdbuilder.NewCommand("list", "List all company users").
		WithOutputFormatRunFunc(ListUsers).
		WithUserFilterFlags().
		WithLongDesc("List all users in your Megaport company.\n\nThis command fetches and displays a list of users with details such as employee ID, name, email, position, and active status.").
		WithOptionalFlag("position", "Filter users by position/role").
		WithOptionalFlag("active-only", "Show only active users").
		WithOptionalFlag("inactive-only", "Show only inactive users").
		WithExample("megaport-cli users list").
		WithExample("megaport-cli users list --active-only").
		WithExample("megaport-cli users list --position \"Technical Admin\"").
		WithRootCmd(rootCmd).
		WithAliases([]string{"ls"}).
		Build()

	getCmd := cmdbuilder.NewCommand("get", "Get details for a specific user").
		WithArgs(cobra.ExactArgs(1)).
		WithOutputFormatRunFunc(GetUser).
		WithLongDesc("Get details for a specific user by employee ID.\n\nThis command retrieves and displays detailed information about a specific user.").
		WithExample("megaport-cli users get 12345").
		WithRootCmd(rootCmd).
		WithAliases([]string{"show"}).
		Build()

	createCmd := cmdbuilder.NewCommand("create", "Create a new user").
		WithColorAwareRunFunc(CreateUser).
		WithInteractiveFlag().
		WithUserCreateFlags().
		WithJSONConfigFlags().
		WithLongDesc("Create a new user in your Megaport company.\n\nThis command allows you to create a new user by providing the necessary details.").
		WithDocumentedRequiredFlag("first-name", "First name of the user").
		WithDocumentedRequiredFlag("last-name", "Last name of the user").
		WithDocumentedRequiredFlag("email", "Email address of the user").
		WithDocumentedRequiredFlag("position", "Position/role (Company Admin, Technical Admin, Technical Contact, Finance, Financial Contact, Read Only)").
		WithOptionalFlag("phone", "Phone number in E.164 format: a leading +, digits only, no spaces, dashes or parentheses, and a real number (e.g., +61412345678)").
		WithExample("megaport-cli users create --interactive").
		WithExample(`megaport-cli users create --first-name "John" --last-name "Doe" --email "john@example.com" --position "Technical Admin"`).
		WithExample(`megaport-cli users create --json '{"firstName":"John","lastName":"Doe","email":"john@example.com","position":"Technical Admin"}'`).
		WithJSONExample(`{
  "firstName": "John",
  "lastName": "Doe",
  "email": "john@example.com",
  "position": "Technical Admin",
  "phone": "+61412345678"
}`).
		WithImportantNote("Valid positions: Company Admin, Technical Admin, Technical Contact, Finance, Financial Contact, Read Only").
		WithImportantNote("Required flags can be skipped when using --interactive, --json, or --json-file").
		WithRootCmd(rootCmd).
		WithConditionalRequirements("first-name", "last-name", "email", "position").
		Build()

	updateCmd := cmdbuilder.NewCommand("update", "Update an existing user").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(UpdateUser).
		WithInteractiveFlag().
		WithUserUpdateFlags().
		WithJSONConfigFlags().
		WithLongDesc("Update an existing user's details.\n\nThis command allows you to update specific properties of an existing user. Only provided fields will be changed.").
		WithOptionalFlag("first-name", "New first name").
		WithOptionalFlag("last-name", "New last name").
		WithOptionalFlag("email", "New email address").
		WithOptionalFlag("position", "New position/role").
		WithOptionalFlag("phone", "New phone number in E.164 format: a leading +, digits only, no spaces, dashes or parentheses, and a real number (e.g., +61412345678)").
		WithOptionalFlag("active", "Set user active status").
		WithOptionalFlag("notification-enabled", "Enable/disable notifications").
		WithExample("megaport-cli users update 12345 --interactive").
		WithExample(`megaport-cli users update 12345 --first-name "Jane" --last-name "Smith"`).
		WithExample(`megaport-cli users update 12345 --json '{"firstName":"Jane"}'`).
		WithJSONExample(`{
  "firstName": "Jane",
  "lastName": "Smith",
  "email": "jane.smith@example.com",
  "phone": "+61400000000",
  "position": "Network Engineer",
  "active": true,
  "notificationEnabled": true,
  "newsletter": false,
  "promotions": false
}`).
		WithImportantNote("Users with pending invitations cannot be updated").
		WithRootCmd(rootCmd).
		Build()

	deleteCmd := cmdbuilder.NewCommand("delete", "Delete a user").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(DeleteUser).
		WithLongDesc("Delete a user from your Megaport company.\n\nThis command deletes a user by their employee ID. Only users with pending invitations can be deleted. Users who have already logged in must be deactivated instead.").
		WithBoolFlag("force", false, "Skip the confirmation prompt").
		WithExample("megaport-cli users delete 12345").
		WithExample("megaport-cli users delete 12345 --force").
		WithImportantNote("Only users with pending invitations can be deleted").
		WithImportantNote("To remove access for active users, use the 'deactivate' command instead").
		WithRootCmd(rootCmd).
		WithAliases([]string{"rm"}).
		Build()

	deactivateCmd := cmdbuilder.NewCommand("deactivate", "Deactivate a user").
		WithArgs(cobra.ExactArgs(1)).
		WithColorAwareRunFunc(DeactivateUser).
		WithLongDesc("Deactivate a user in your Megaport company.\n\nThis command deactivates a user by setting their active status to false. The user will no longer be able to log in or perform actions.").
		WithBoolFlag("force", false, "Skip the confirmation prompt").
		WithExample("megaport-cli users deactivate 12345").
		WithExample("megaport-cli users deactivate 12345 --force").
		WithImportantNote("Deactivated users cannot log in or perform any actions").
		WithImportantNote("The user's email will be modified to prevent reuse").
		WithRootCmd(rootCmd).
		Build()

	activityCmd := cmdbuilder.NewCommand("activity", "View user activity logs").
		WithOutputFormatRunFunc(GetUserActivity).
		WithLongDesc("View activity logs for users in your Megaport company.\n\nThis command retrieves and displays user activity logs. Optionally filter by a specific user.").
		WithFlag("employee-id", "", "Filter activity by employee ID").
		WithExample("megaport-cli users activity").
		WithExample("megaport-cli users activity --employee-id 12345").
		WithRootCmd(rootCmd).
		Build()

	usersCmd.AddCommand(
		listCmd,
		getCmd,
		createCmd,
		updateCmd,
		deleteCmd,
		deactivateCmd,
		activityCmd,
	)
	rootCmd.AddCommand(usersCmd)
}
