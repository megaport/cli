package product

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/megaport/megaport-cli/internal/base/exitcodes"
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

var getProductPricingFunc = func(ctx context.Context, client *megaport.Client, req megaport.PriceBookRequest) (*megaport.PriceBookDTO, error) {
	return client.ProductService.GetProductPricing(ctx, req)
}

var getProductPricingForCompanyFunc = func(ctx context.Context, client *megaport.Client, req *megaport.GetProductPricingRequest) (*megaport.PriceBookDTO, error) {
	return client.ProductService.GetProductPricingForCompany(ctx, req)
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

func GetPortPricing(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	req := &megaport.MegaportPriceBookRequest{}
	return getProductPricing(cmd, noColor, outputFormat, "Port", req, func() {
		// Flag read errors are ignored: the command builder registers these flags.
		req.LocationID, _ = cmd.Flags().GetInt("location-id")
		req.Speed, _ = cmd.Flags().GetInt("speed")
		req.Term, _ = cmd.Flags().GetInt("term")
		req.Currency, _ = cmd.Flags().GetString("currency")
		req.ProductUID, _ = cmd.Flags().GetString("product-uid")
	})
}

func GetVXCPricing(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	req := &megaport.VXCPriceBookRequest{}
	return getProductPricing(cmd, noColor, outputFormat, "VXC", req, func() {
		req.ALocationID, _ = cmd.Flags().GetInt("a-location-id")
		req.BLocationID, _ = cmd.Flags().GetInt("b-location-id")
		req.Speed, _ = cmd.Flags().GetInt("speed")
		req.Term, _ = cmd.Flags().GetInt("term")
		req.AEndProductType, _ = cmd.Flags().GetString("a-end-product-type")
		req.ConnectType, _ = cmd.Flags().GetString("connect-type")
		req.Currency, _ = cmd.Flags().GetString("currency")
		req.ProductUID, _ = cmd.Flags().GetString("product-uid")
	})
}

func GetMCRPricing(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	req := &megaport.MCRPriceBookRequest{}
	return getProductPricing(cmd, noColor, outputFormat, "MCR", req, func() {
		req.LocationID, _ = cmd.Flags().GetInt("location-id")
		req.Speed, _ = cmd.Flags().GetInt("speed")
		req.Term, _ = cmd.Flags().GetInt("term")
		req.Currency, _ = cmd.Flags().GetString("currency")
		req.ProductUID, _ = cmd.Flags().GetString("product-uid")
	})
}

func GetMVEPricing(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	req := &megaport.MVEPriceBookRequest{}
	return getProductPricing(cmd, noColor, outputFormat, "MVE", req, func() {
		req.LocationID, _ = cmd.Flags().GetInt("location-id")
		req.Size, _ = cmd.Flags().GetString("size")
		req.Term, _ = cmd.Flags().GetInt("term")
		req.Currency, _ = cmd.Flags().GetString("currency")
		req.ProductUID, _ = cmd.Flags().GetString("product-uid")
	})
}

func GetIXPricing(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	req := &megaport.IXPriceBookRequest{}
	return getProductPricing(cmd, noColor, outputFormat, "IX", req, func() {
		req.PortLocationID, _ = cmd.Flags().GetInt("port-location-id")
		req.IXType, _ = cmd.Flags().GetString("ix-type")
		req.Speed, _ = cmd.Flags().GetInt("speed")
		req.Term, _ = cmd.Flags().GetInt("term")
		req.Currency, _ = cmd.Flags().GetString("currency")
		req.ProductUID, _ = cmd.Flags().GetString("product-uid")
	})
}

func GetNATGatewayPricing(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	req := &megaport.NATGatewayPriceBookRequest{}
	return getProductPricing(cmd, noColor, outputFormat, "NAT Gateway", req, func() {
		req.LocationID, _ = cmd.Flags().GetInt("location-id")
		req.Speed, _ = cmd.Flags().GetInt("speed")
		req.SessionCount, _ = cmd.Flags().GetInt("session-count")
		req.Term, _ = cmd.Flags().GetInt("term")
		req.Currency, _ = cmd.Flags().GetString("currency")
		req.ProductUID, _ = cmd.Flags().GetString("product-uid")
	})
}

func GetIPAddressPricing(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	req := &megaport.IPAddressPriceBookRequest{}
	return getProductPricing(cmd, noColor, outputFormat, "IP address", req, func() {
		req.LocationID, _ = cmd.Flags().GetInt("location-id")
		req.IPBlock, _ = cmd.Flags().GetString("ip-block")
		req.Currency, _ = cmd.Flags().GetString("currency")
	})
}

// pricingUsageErrors are the SDK errors for a missing or zero required field.
var pricingUsageErrors = []error{
	megaport.ErrPricingLocationRequired, megaport.ErrPricingSpeedRequired,
	megaport.ErrPricingVXCLocationRequired, megaport.ErrPricingVXCSpeedRequired,
	megaport.ErrPricingIXLocationRequired, megaport.ErrPricingIXTypeRequired,
	megaport.ErrPricingNATSessionRequired,
	megaport.ErrPricingIPLocationRequired, megaport.ErrPricingIPBlockRequired,
	megaport.ErrPricingMVELocationRequired,
}

// getProductPricing fills req from --json or --json-file when either is set,
// and from fromFlags otherwise.
func getProductPricing(cmd *cobra.Command, noColor bool, outputFormat, productName string, req megaport.PriceBookRequest, fromFlags func()) error {
	output.SetOutputFormat(outputFormat)
	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	jsonStr, _ := cmd.Flags().GetString("json")
	jsonFile, _ := cmd.Flags().GetString("json-file")
	if jsonStr != "" || jsonFile != "" {
		jsonData, err := utils.ReadJSONInput(jsonStr, jsonFile)
		if err != nil {
			output.PrintError("Failed to process JSON input: %v", noColor, err)
			return exitcodes.NewUsageError(err)
		}
		dec := json.NewDecoder(bytes.NewReader(jsonData))
		dec.DisallowUnknownFields()
		switch err = dec.Decode(req); {
		case errors.Is(err, io.EOF):
			err = errors.New("JSON input is empty")
		case err == nil:
			// Reject trailing data after the object, as json.Unmarshal does.
			if _, tokErr := dec.Token(); !errors.Is(tokErr, io.EOF) {
				err = errors.New("unexpected data after the JSON object")
			}
		}
		if err != nil {
			output.PrintError("Failed to process JSON input: %v", noColor, err)
			return exitcodes.NewUsageError(fmt.Errorf("failed to parse JSON: %w", err))
		}
	} else {
		fromFlags()
	}
	companyUID, _ := cmd.Flags().GetString("company-uid")

	client, err := config.Login(ctx)
	if err != nil {
		output.PrintError("Failed to log in: %v", noColor, err)
		return fmt.Errorf("failed to log in: %w", err)
	}

	spinner := output.PrintCustomSpinner("Getting pricing for", productName, noColor)

	var price *megaport.PriceBookDTO
	if companyUID != "" {
		price, err = getProductPricingForCompanyFunc(ctx, client, &megaport.GetProductPricingRequest{Req: req, CompanyUID: companyUID})
	} else {
		price, err = getProductPricingFunc(ctx, client, req)
	}

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to get %s pricing: %v", noColor, productName, err)
		err = fmt.Errorf("failed to get %s pricing: %w", productName, err)
		for _, sentinel := range pricingUsageErrors {
			if errors.Is(err, sentinel) {
				return exitcodes.NewUsageError(err)
			}
		}
		return err
	}

	if err := printProductPricing(price, outputFormat, noColor); err != nil {
		output.PrintError("Failed to print %s pricing: %v", noColor, productName, err)
		return fmt.Errorf("failed to print %s pricing: %w", productName, err)
	}
	return nil
}
