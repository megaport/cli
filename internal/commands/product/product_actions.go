package product

import (
	"context"
	"fmt"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/config"
	"github.com/megaport/megaport-cli/internal/utils"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
)

var listProductsFunc = func(ctx context.Context, client *megaport.Client) ([]megaport.Product, error) {
	return client.ProductService.ListProducts(ctx)
}

var getProductTypeFunc = func(ctx context.Context, client *megaport.Client, productUID string) (string, error) {
	return client.ProductService.GetProductType(ctx, productUID)
}

func ListProducts(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)
	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return fmt.Errorf("failed to log in: %w", err)
	}

	spinner := output.PrintResourceListing("Product", noColor)

	products, err := listProductsFunc(ctx, client)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to list products: %v", noColor, err)
		return fmt.Errorf("failed to list products: %w", err)
	}

	// Flag read errors are intentionally ignored — flags are registered by the command builder.
	includeInactive, _ := cmd.Flags().GetBool("include-inactive")
	filtered := filterProducts(products, includeInactive)

	limit, _ := cmd.Flags().GetInt("limit")
	return utils.ApplyLimitAndPrint(filtered, limit, outputFormat, noColor,
		"No products found.", printProducts)
}

func GetProductType(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)
	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return fmt.Errorf("failed to log in: %w", err)
	}

	productUID := args[0]

	spinner := output.PrintResourceGetting("Product Type", productUID, noColor)

	productType, err := getProductTypeFunc(ctx, client, productUID)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to get product type: %v", noColor, err)
		return fmt.Errorf("failed to get product type: %w", err)
	}

	outputs := []productTypeOutput{
		{
			UID:  productUID,
			Type: productType,
		},
	}

	err = output.PrintOutput(outputs, outputFormat, noColor)
	if err != nil {
		output.PrintError("Failed to print product type: %v", noColor, err)
		return fmt.Errorf("failed to print product type: %w", err)
	}
	return nil
}

func ListProductAvailability(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)
	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return fmt.Errorf("failed to log in: %w", err)
	}

	spinner := output.PrintResourceListing("Available Product", noColor)

	company, err := client.CompanyService.GetCurrentCompany(ctx)
	if err != nil {
		spinner.Stop()
		output.PrintError("Failed to get current company: %v", noColor, err)
		return fmt.Errorf("failed to get current company: %w", err)
	}

	availability, err := client.ProductAvailabilityService.ListCompanyProductAvailability(ctx, company.CompanyUID)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to list product availability: %v", noColor, err)
		return fmt.Errorf("failed to list product availability: %w", err)
	}

	return utils.ApplyLimitAndPrint(toProductAvailabilityOutputs(availability), 0, outputFormat, noColor,
		"No products available.", output.PrintOutput[productAvailabilityOutput])
}
