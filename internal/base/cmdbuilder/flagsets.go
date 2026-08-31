package cmdbuilder

import "time"

// WithWatchFlags adds flags for continuous status monitoring
func (b *CommandBuilder) WithWatchFlags() *CommandBuilder {
	b.WithBoolFlagP("watch", "w", false, "Continuously poll and display resource status (Ctrl+C to stop)")
	b.WithDurationFlag("interval", 5*time.Second, "Polling interval for --watch mode (e.g. 5s, 1m)")
	return b
}

// AddStandardInputFlags adds interactive, json and json-file flags
func (b *CommandBuilder) WithStandardInputFlags() *CommandBuilder {
	b.WithBoolFlagP("interactive", "i", false, "Use interactive mode with prompts")
	b.WithFlag("json", "", "JSON string containing configuration")
	b.WithFlag("json-file", "", fileInputUsage("Path to JSON file containing configuration", "--json"))
	return b
}

// WithDateRangeFlags adds start and end date flags for time-bound resources
func (b *CommandBuilder) WithDateRangeFlags() *CommandBuilder {
	b.WithFlag("start-date", "", "Start date (YYYY-MM-DD)")
	b.WithFlag("end-date", "", "End date (YYYY-MM-DD)")
	return b
}

// WithJSONConfigFlags adds flags for JSON configuration input
func (b *CommandBuilder) WithJSONConfigFlags() *CommandBuilder {
	b.WithFlag("json", "", "JSON string containing configuration")
	b.WithFlag("json-file", "", fileInputUsage("Path to JSON file containing configuration", "--json"))
	return b
}

// WithResourceTagFlags adds flags for resource tagging
func (b *CommandBuilder) WithResourceTagFlags() *CommandBuilder {
	b.WithFlag("resource-tags", "", "Resource tags as a JSON string (e.g. {\"key1\":\"value1\",\"key2\":\"value2\"})")
	b.WithFlag("resource-tags-file", "", fileInputUsage("Path to JSON file containing resource tags", "--resource-tags"))
	b.WithOptionalFlag("resource-tags", "Resource tags as a JSON string (e.g. {\"key1\":\"value1\",\"key2\":\"value2\"})")
	b.WithOptionalFlag("resource-tags-file", fileInputUsage("Path to JSON file containing resource tags", "--resource-tags"))
	return b
}

// WithDeleteFlags adds flags for deletion
func (b *CommandBuilder) WithDeleteFlags() *CommandBuilder {
	b.WithBoolFlagP("force", "f", false, "Skip confirmation prompt")
	b.WithBoolFlag("now", false, "Delete resource immediately instead of at end of billing cycle")
	return b
}

// WithSafeDeleteFlags adds flags for deletion with safe-delete support (for resources that support it via the API)
func (b *CommandBuilder) WithSafeDeleteFlags() *CommandBuilder {
	b.WithDeleteFlags()
	b.WithBoolFlag("safe-delete", false, "Fail if the resource has attached VXCs or other active services")
	return b
}

// WithImmediateSafeDeleteFlags adds delete flags for resources that only support
// immediate deletion (no --now flag), with safe-delete support.
func (b *CommandBuilder) WithImmediateSafeDeleteFlags() *CommandBuilder {
	b.WithBoolFlagP("force", "f", false, "Skip confirmation prompt")
	b.WithBoolFlag("safe-delete", false, "Fail if the resource has attached VXCs or other active services")
	return b
}

// WithImmediateDeleteFlags adds delete flags for resources that only support
// immediate deletion (no --now flag), without safe-delete support.
func (b *CommandBuilder) WithImmediateDeleteFlags() *CommandBuilder {
	b.WithBoolFlagP("force", "f", false, "Skip confirmation prompt")
	return b
}

// WithDeferredDeleteFlags adds delete flags for resources where immediate
// deletion is the default and deferred end-of-term cancellation is opt-in
// via --later. Used by VXC and IX, which are the only product types whose
// API still accepts CANCEL (end-of-term).
func (b *CommandBuilder) WithDeferredDeleteFlags() *CommandBuilder {
	b.WithBoolFlagP("force", "f", false, "Skip confirmation prompt")
	b.WithBoolFlag("later", false, "Schedule deletion at the end of the current billing cycle (default: delete immediately)")
	return b
}

// WithBuyConfirmFlags adds the --yes/-y flag to skip buy confirmation prompts
func (b *CommandBuilder) WithBuyConfirmFlags() *CommandBuilder {
	b.WithBoolFlagP("yes", "y", false, "Skip confirmation prompt for purchase; required when using --json or --json-file")
	return b
}

// WithInteractiveFlag adds just the interactive flag
func (b *CommandBuilder) WithInteractiveFlag() *CommandBuilder {
	b.WithBoolFlagP("interactive", "i", false, "Use interactive mode with prompts")
	return b
}

// WithNoWaitFlag adds a flag to skip waiting for provisioning
func (b *CommandBuilder) WithNoWaitFlag() *CommandBuilder {
	b.WithBoolFlag("no-wait", false, "Do not wait for provisioning to complete")
	return b
}
