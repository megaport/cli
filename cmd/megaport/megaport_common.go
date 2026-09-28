//go:build !wasm

package megaport

import (
	"github.com/megaport/megaport-cli/internal/base/registry"
	"github.com/megaport/megaport-cli/internal/commands/apply"
	"github.com/megaport/megaport-cli/internal/commands/auth"
	"github.com/megaport/megaport-cli/internal/commands/billing_market"
	"github.com/megaport/megaport-cli/internal/commands/completion"
	"github.com/megaport/megaport-cli/internal/commands/config"
	"github.com/megaport/megaport-cli/internal/commands/generate_docs"
	"github.com/megaport/megaport-cli/internal/commands/ix"
	"github.com/megaport/megaport-cli/internal/commands/locations"
	"github.com/megaport/megaport-cli/internal/commands/managed_account"
	"github.com/megaport/megaport-cli/internal/commands/mcr"
	"github.com/megaport/megaport-cli/internal/commands/mve"
	nat_gateway "github.com/megaport/megaport-cli/internal/commands/nat_gateway"
	"github.com/megaport/megaport-cli/internal/commands/partners"
	"github.com/megaport/megaport-cli/internal/commands/ports"
	"github.com/megaport/megaport-cli/internal/commands/product"
	"github.com/megaport/megaport-cli/internal/commands/servicekeys"
	"github.com/megaport/megaport-cli/internal/commands/status"
	"github.com/megaport/megaport-cli/internal/commands/topology"
	"github.com/megaport/megaport-cli/internal/commands/users"
	"github.com/megaport/megaport-cli/internal/commands/version"
	"github.com/megaport/megaport-cli/internal/commands/vxc"
	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/spf13/cobra"
)

// Common variables and declarations needed by both WASM and non-WASM builds
var (
	noColor      bool
	noHeader     bool
	noPager      bool
	outputFormat string
	quiet        bool
	verbose      bool

	// rootCmd is the root command for the CLI
	rootCmd = &cobra.Command{
		Use:           "megaport-cli",
		Short:         "A CLI tool to interact with the Megaport API",
		SilenceErrors: false, // Allow Cobra to print returned errors so failures are always visible
		// Long will be set by the help builder later
	}

	// moduleRegistry holds all command modules
	moduleRegistry *registry.Registry
)

// registerModules registers all command modules with the registry
func registerModules() {
	// Register all modules
	moduleRegistry.Register(auth.NewModule())
	moduleRegistry.Register(version.NewModule())
	moduleRegistry.Register(ports.NewModule())
	moduleRegistry.Register(vxc.NewModule())
	moduleRegistry.Register(mcr.NewModule())
	moduleRegistry.Register(mve.NewModule())
	moduleRegistry.Register(nat_gateway.NewModule())
	moduleRegistry.Register(locations.NewModule())
	moduleRegistry.Register(partners.NewModule())
	moduleRegistry.Register(product.NewModule())
	moduleRegistry.Register(servicekeys.NewModule())
	moduleRegistry.Register(generate_docs.NewModule())
	moduleRegistry.Register(completion.NewModule())
	moduleRegistry.Register(ix.NewModule())
	moduleRegistry.Register(managed_account.NewModule())
	moduleRegistry.Register(config.NewModule())
	moduleRegistry.Register(billing_market.NewModule())
	moduleRegistry.Register(users.NewModule())
	moduleRegistry.Register(status.NewModule())
	moduleRegistry.Register(topology.NewModule())
	moduleRegistry.Register(apply.NewModule())
}

// InitializeCommon performs initialization steps common to all platforms
func InitializeCommon() {
	// Initialize module registry
	moduleRegistry = registry.NewRegistry()

	// Register all modules
	registerModules()

	// Setup persistent flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", utils.FormatTable,
		"Output format (table, json, csv, xml, go-template; requires --template when using go-template)")
	rootCmd.PersistentFlags().String("template", "", "Go template string for --output go-template (e.g. '{{range .}}{{.Name}}{{\"\\n\"}}{{end}}')")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colorful output")
	rootCmd.PersistentFlags().StringVar(&utils.Env, "env", "", "Environment to use (prod, dev, or staging)")
	rootCmd.PersistentFlags().StringVar(&utils.ProfileOverride, "profile", "", "Use a specific config profile for this command")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress informational output, only show errors and data")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show additional debug information")
	rootCmd.PersistentFlags().Duration("timeout", 0, "Timeout for the operation (e.g., 30s, 2m, 5m); must be positive. Omit to use each command's built-in default (see the command's own help)")
	rootCmd.PersistentFlags().String("fields", "", "Comma-separated list of fields to include in output (e.g., uid,name,status); use an unknown name to list available fields")
	rootCmd.PersistentFlags().String("query", "", "JMESPath query to filter JSON output (requires --output json)")
	rootCmd.PersistentFlags().BoolVar(&utils.NoRetry, "no-retry", false, "Disable automatic retry on transient API failures")
	rootCmd.PersistentFlags().IntVar(&utils.MaxRetries, "max-retries", 3, "Maximum number of retries for transient API failures")
	rootCmd.PersistentFlags().BoolVar(&utils.LogHTTP, "log-http", false, "Log raw HTTP requests/responses to stderr for debugging (may include sensitive data such as auth tokens)")
	rootCmd.PersistentFlags().StringVar(&utils.BaseURL, "base-url", "", "Override the API base URL (e.g. http://localhost:8080); takes precedence over --env and any profile environment")
	rootCmd.PersistentFlags().StringVar(&utils.TokenURL, "token-url", "", "Override the OAuth token endpoint (typically used with --base-url when auth is served from a non-standard host)")
	rootCmd.PersistentFlags().StringVar(&utils.ManagedAccountUID, "on-behalf-of", "", "Act on behalf of a managed account: company UID sent as the X-Call-Context header on every authenticated request (falls back to MEGAPORT_MANAGED_ACCOUNT_UID)")
	rootCmd.PersistentFlags().BoolVar(&noHeader, "no-header", false, "Suppress table and CSV column headers (useful for scripting)")
	rootCmd.PersistentFlags().BoolVar(&noPager, "no-pager", false, "Disable pager for long table output")
	rootCmd.MarkFlagsMutuallyExclusive("quiet", "verbose")
	rootCmd.SuggestionsMinimumDistance = 2
}
