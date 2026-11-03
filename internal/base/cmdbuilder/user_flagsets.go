package cmdbuilder

// WithUserCreateFlags adds flags for user creation
func (b *CommandBuilder) WithUserCreateFlags() *CommandBuilder {
	b.WithFlag("first-name", "", "First name of the user").
		WithFlag("last-name", "", "Last name of the user").
		WithFlag("email", "", "Email address of the user").
		WithFlag("position", "", "Position/role of the user (Company Admin, Technical Admin, Technical Contact, Finance, Financial Contact, Read Only)").
		WithFlag("phone", "", "Phone number in E.164 format (optional): a leading +, digits only, no spaces, dashes or parentheses, and a real number (e.g., +61412345678)")
	return b
}

// WithUserUpdateFlags adds flags for user updates
func (b *CommandBuilder) WithUserUpdateFlags() *CommandBuilder {
	b.WithFlag("first-name", "", "New first name").
		WithFlag("last-name", "", "New last name").
		WithFlag("email", "", "New email address").
		WithFlag("position", "", "New position/role").
		WithFlag("phone", "", "New phone number in E.164 format: a leading +, digits only, no spaces, dashes or parentheses, and a real number (e.g., +61412345678)").
		WithBoolFlag("active", false, "Set user active status").
		WithBoolFlag("notification-enabled", false, "Enable/disable notifications")
	return b
}

// WithUserFilterFlags adds flags for filtering user lists
func (b *CommandBuilder) WithUserFilterFlags() *CommandBuilder {
	b.WithFlag("position", "", "Filter users by position/role").
		WithBoolFlag("active-only", false, "Show only active users").
		WithBoolFlag("inactive-only", false, "Show only inactive users")
	return b
}
