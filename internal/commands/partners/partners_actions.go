package partners

import (
	"fmt"
	"strings"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/config"
	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/megaport/megaport-cli/internal/validation"
	"github.com/spf13/cobra"
)

func ListPartners(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)
	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return fmt.Errorf("failed to log in: %w", err)
	}

	spinner := output.PrintResourceListing("partner", noColor)
	partners, err := client.PartnerService.ListPartnerMegaports(ctx)
	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to list partner ports: %v", noColor, err)
		return fmt.Errorf("failed to list partners: %w", err)
	}

	// Flag read errors are intentionally ignored — flags are registered by the command builder.
	productName, _ := cmd.Flags().GetString("product-name")
	connectType, _ := cmd.Flags().GetString("connect-type")
	companyName, _ := cmd.Flags().GetString("company-name")
	locationID, _ := cmd.Flags().GetInt("location-id")
	diversityZone, _ := cmd.Flags().GetString("diversity-zone")

	filteredPartners := filterPartners(partners, productName, connectType, companyName, locationID, diversityZone)

	limit, _ := cmd.Flags().GetInt("limit")
	return utils.ApplyLimitAndPrint(filteredPartners, limit, outputFormat, noColor,
		"No partner ports found matching your filters.", printPartnersFunc)
}

func FindPartners(cmd *cobra.Command, args []string, noColor bool) error {
	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	output.PrintInfo("Searching for partner ports...", noColor)

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return fmt.Errorf("failed to log in: %w", err)
	}

	spinner := output.PrintResourceListing("partner", noColor)
	partners, err := client.PartnerService.ListPartnerMegaports(ctx)
	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to list partner ports: %v", noColor, err)
		return fmt.Errorf("failed to list partners: %w", err)
	}

	output.PrintInfo("Filter partner ports - press Enter to skip any filter", noColor)
	output.PrintInfo("----------------------------------------------------", noColor)

	productName, err := utils.Prompt("Product name: ", noColor)
	if err != nil {
		output.PrintError("Failed to get product name: %v", noColor, err)
		return err
	}

	connectType, err := utils.Prompt("Connect type (AWS, AWSHC, AZURE, GOOGLE, ORACLE, IBM, etc): ", noColor)
	if err != nil {
		output.PrintError("Failed to get connect type: %v", noColor, err)
		return err
	}

	companyName, err := utils.Prompt("Company name: ", noColor)
	if err != nil {
		output.PrintError("Failed to get company name: %v", noColor, err)
		return err
	}

	var locationID int
	locationIDStr, err := utils.Prompt("Location ID (numeric): ", noColor)
	if err != nil {
		output.PrintError("Failed to get location ID: %v", noColor, err)
		return err
	}
	if locationIDStr != "" {
		locationID, err = validation.ParseInt("location ID", locationIDStr)
		if err != nil {
			output.PrintError("%v", noColor, err)
			return err
		}
	}

	diversityZone, err := utils.Prompt("Diversity zone: ", noColor)
	if err != nil {
		output.PrintError("Failed to get diversity zone: %v", noColor, err)
		return err
	}

	format, err := utils.Prompt("Output format [table/json/csv/xml] (default: table): ", noColor)
	if err != nil {
		output.PrintError("Failed to get output format: %v", noColor, err)
		return err
	}

	normalizedFormat := strings.ToLower(strings.TrimSpace(format))
	selectedFormat := "table"
	switch normalizedFormat {
	case "json", "csv", "xml":
		selectedFormat = normalizedFormat
	}

	filteredPartners := filterPartners(partners, productName, connectType, companyName, locationID, diversityZone)

	output.PrintInfo("Found %d matching partner ports", noColor, len(filteredPartners))

	err = printPartnersFunc(filteredPartners, selectedFormat, noColor)
	if err != nil {
		output.PrintError("Failed to print partner ports: %v", noColor, err)
		return fmt.Errorf("failed to print partners: %w", err)
	}
	return nil
}
