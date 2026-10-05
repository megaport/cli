package registry

import (
	"github.com/spf13/cobra"
)

// Module is the interface that all command modules must implement
type Module interface {
	Name() string
	RegisterCommands(rootCmd *cobra.Command)
}

// Registry keeps track of all registered modules
type Registry struct {
	modules []Module
}

// NewRegistry creates a new module registry
func NewRegistry() *Registry {
	return &Registry{
		modules: make([]Module, 0),
	}
}

// Register adds a module to the registry
func (r *Registry) Register(module Module) {
	r.modules = append(r.modules, module)
}

// RegisterAll registers all modules with the root command
func (r *Registry) RegisterAll(rootCmd *cobra.Command) {
	for _, module := range r.modules {
		module.RegisterCommands(rootCmd)
	}
	for _, cmd := range rootCmd.Commands() {
		rejectUnknownSubcommands(cmd)
	}
}

// rejectUnknownSubcommands makes each group below the root return cobra's
// "unknown command" error for an unmatched argument. Cobra checks this only on
// the root, and a group with no Run prints its help and exits 0 instead.
func rejectUnknownSubcommands(cmd *cobra.Command) {
	if cmd.HasSubCommands() && !cmd.Runnable() {
		cmd.Args = cobra.NoArgs
		cmd.RunE = func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		}
	}
	for _, sub := range cmd.Commands() {
		rejectUnknownSubcommands(sub)
	}
}
