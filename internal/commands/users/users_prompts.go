package users

import (
	"fmt"
	"strings"

	"github.com/megaport/megaport-cli/internal/utils"
	megaport "github.com/megaport/megaportgo"
)

func promptForCreateUserDetails(noColor bool) (*megaport.CreateUserRequest, error) {
	firstName, err := utils.ResourcePrompt("user", "Enter first name (required): ", noColor)
	if err != nil {
		return nil, err
	}
	if firstName == "" {
		return nil, fmt.Errorf("first name is required")
	}

	lastName, err := utils.ResourcePrompt("user", "Enter last name (required): ", noColor)
	if err != nil {
		return nil, err
	}
	if lastName == "" {
		return nil, fmt.Errorf("last name is required")
	}

	email, err := utils.ResourcePrompt("user", "Enter email address (required): ", noColor)
	if err != nil {
		return nil, err
	}
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	position, err := utils.ResourcePrompt("user", "Enter position (Company Admin, Technical Admin, Technical Contact, Finance, Financial Contact, Read Only) (required): ", noColor)
	if err != nil {
		return nil, err
	}
	if position == "" {
		return nil, fmt.Errorf("position is required")
	}
	userPosition := megaport.UserPosition(position)
	if !userPosition.IsValid() {
		return nil, fmt.Errorf("invalid position: %s. Valid positions: %s", position, userPosition.ValidPositions())
	}

	phone, err := utils.ResourcePrompt("user", "Enter phone number (optional; E.164 format: a leading +, digits only, no spaces, dashes or parentheses, and a real number, e.g., +61412345678): ", noColor)
	if err != nil {
		return nil, err
	}

	return &megaport.CreateUserRequest{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		Position:  userPosition,
		Phone:     phone,
		Active:    true,
	}, nil
}

func promptForUpdateUserDetails(noColor bool) (*megaport.UpdateUserRequest, error) {
	req := &megaport.UpdateUserRequest{}
	fieldsUpdated := false

	firstName, err := utils.ResourcePrompt("user", "Enter new first name (leave empty to skip): ", noColor)
	if err != nil {
		return nil, err
	}
	if firstName != "" {
		req.FirstName = &firstName
		fieldsUpdated = true
	}

	lastName, err := utils.ResourcePrompt("user", "Enter new last name (leave empty to skip): ", noColor)
	if err != nil {
		return nil, err
	}
	if lastName != "" {
		req.LastName = &lastName
		fieldsUpdated = true
	}

	email, err := utils.ResourcePrompt("user", "Enter new email (leave empty to skip): ", noColor)
	if err != nil {
		return nil, err
	}
	if email != "" {
		req.Email = &email
		fieldsUpdated = true
	}

	position, err := utils.ResourcePrompt("user", "Enter new position (leave empty to skip): ", noColor)
	if err != nil {
		return nil, err
	}
	if position != "" {
		req.Position = &position
		fieldsUpdated = true
	}

	phone, err := utils.ResourcePrompt("user", "Enter new phone number (leave empty to skip; E.164 format: a leading +, digits only, no spaces, dashes or parentheses, and a real number, e.g., +61412345678): ", noColor)
	if err != nil {
		return nil, err
	}
	if phone != "" {
		req.Phone = &phone
		fieldsUpdated = true
	}

	activeStr, err := utils.ResourcePrompt("user", "Set user active? (y/yes/n/no, leave empty to skip): ", noColor)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(activeStr)) {
	case "":
	case "y", "yes":
		active := true
		req.Active = &active
		fieldsUpdated = true
	case "n", "no":
		active := false
		req.Active = &active
		fieldsUpdated = true
	default:
		return nil, fmt.Errorf("invalid response for active: %s (expected y/yes/n/no)", activeStr)
	}

	notificationEnabledStr, err := utils.ResourcePrompt("user", "Enable notifications? (y/yes/n/no, leave empty to skip): ", noColor)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(notificationEnabledStr)) {
	case "":
	case "y", "yes":
		notificationEnabled := true
		req.NotificationEnabled = &notificationEnabled
		fieldsUpdated = true
	case "n", "no":
		notificationEnabled := false
		req.NotificationEnabled = &notificationEnabled
		fieldsUpdated = true
	default:
		return nil, fmt.Errorf("invalid response for notification-enabled: %s (expected y/yes/n/no)", notificationEnabledStr)
	}

	if !fieldsUpdated {
		return nil, fmt.Errorf("at least one field must be updated")
	}

	return req, nil
}
