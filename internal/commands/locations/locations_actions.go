package locations

import (
	"fmt"
	"strings"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/config"
	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/megaport/megaport-cli/internal/validation"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
)

func ListLocations(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	client, err := config.NewUnauthenticatedClient()
	if err != nil {
		output.PrintError("Failed to create API client: %v", noColor, err)
		return fmt.Errorf("failed to create API client: %w", err)
	}

	spinner := output.PrintResourceListing("Location", noColor)

	locations, err := listLocationsFunc(ctx, client)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to retrieve locations: %v", noColor, err)
		return fmt.Errorf("failed to list locations: %w", err)
	}

	filters := map[string]string{}
	if cmd.Flags().Changed("metro") {
		// Flag read errors are intentionally ignored — flags are registered by the command builder.
		metro, _ := cmd.Flags().GetString("metro")
		filters["metro"] = metro
		output.PrintInfo("Filtering by metro: %s", noColor, metro)
	}
	if cmd.Flags().Changed("country") {
		country, _ := cmd.Flags().GetString("country")
		filters["country"] = country
		output.PrintInfo("Filtering by country: %s", noColor, country)
	}
	if cmd.Flags().Changed("name") {
		name, _ := cmd.Flags().GetString("name")
		filters["name"] = name
		output.PrintInfo("Filtering by name: %s", noColor, name)
	}
	if cmd.Flags().Changed("market-code") {
		marketCode, _ := cmd.Flags().GetString("market-code")
		filters["market"] = marketCode
		output.PrintInfo("Filtering by market code: %s", noColor, marketCode)
	}
	if cmd.Flags().Changed("mcr-available") {
		mcrAvailable, _ := cmd.Flags().GetBool("mcr-available")
		if mcrAvailable {
			filters["mcrAvailable"] = "true"
			output.PrintInfo("Filtering by MCR availability", noColor)
		}
	}

	filteredLocations := filterLocations(locations, filters)

	limit, _ := cmd.Flags().GetInt("limit")
	return utils.ApplyLimitAndPrint(filteredLocations, limit, outputFormat, noColor,
		"No locations found matching your filters.", printLocations)
}

func ListCountries(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	client, err := config.NewUnauthenticatedClient()
	if err != nil {
		output.PrintError("Failed to create API client: %v", noColor, err)
		return fmt.Errorf("failed to create API client: %w", err)
	}

	spinner := output.PrintResourceListing("Country", noColor)

	countries, err := listCountriesFunc(ctx, client)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to retrieve countries: %v", noColor, err)
		return fmt.Errorf("failed to list countries: %w", err)
	}

	if len(countries) == 0 {
		output.PrintWarning("No countries found", noColor)
	} else {
		output.PrintInfo("Found %d countries", noColor, len(countries))
	}

	err = printCountries(countries, outputFormat, noColor)
	if err != nil {
		output.PrintError("Failed to print countries: %v", noColor, err)
		return fmt.Errorf("failed to print countries: %w", err)
	}
	return nil
}

func ListMarketCodes(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	client, err := config.NewUnauthenticatedClient()
	if err != nil {
		output.PrintError("Failed to create API client: %v", noColor, err)
		return fmt.Errorf("failed to create API client: %w", err)
	}

	spinner := output.PrintResourceListing("Market Code", noColor)

	marketCodes, err := listMarketCodesFunc(ctx, client)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to retrieve market codes: %v", noColor, err)
		return fmt.Errorf("failed to list market codes: %w", err)
	}

	// The API includes countries with no (or whitespace-only) market prefix;
	// drop them so the list doesn't render a blank entry.
	filtered := make([]string, 0, len(marketCodes))
	for _, mc := range marketCodes {
		if strings.TrimSpace(mc) != "" {
			filtered = append(filtered, mc)
		}
	}
	marketCodes = filtered

	if len(marketCodes) == 0 {
		output.PrintWarning("No market codes found", noColor)
	} else {
		output.PrintInfo("Found %d market codes", noColor, len(marketCodes))
	}

	err = printMarketCodes(marketCodes, outputFormat, noColor)
	if err != nil {
		output.PrintError("Failed to print market codes: %v", noColor, err)
		return fmt.Errorf("failed to print market codes: %w", err)
	}
	return nil
}

func SearchLocations(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	client, err := config.NewUnauthenticatedClient()
	if err != nil {
		output.PrintError("Failed to create API client: %v", noColor, err)
		return fmt.Errorf("failed to create API client: %w", err)
	}

	search := args[0]

	spinner := output.PrintResourceListing("Location", noColor)

	locations, err := searchLocationsFunc(ctx, client, search)

	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to search locations: %v", noColor, err)
		return fmt.Errorf("failed to search locations: %w", err)
	}

	if len(locations) == 0 {
		output.PrintWarning("No locations found matching '%s'", noColor, search)
	} else {
		output.PrintInfo("Found %d locations matching '%s'", noColor, len(locations), search)
	}

	err = printLocations(locations, outputFormat, noColor)
	if err != nil {
		output.PrintError("Failed to print locations: %v", noColor, err)
		return fmt.Errorf("failed to print locations: %w", err)
	}
	return nil
}

func GetRoundTripTimes(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	client, err := config.NewUnauthenticatedClient()
	if err != nil {
		output.PrintError("Failed to create API client: %v", noColor, err)
		return fmt.Errorf("failed to create API client: %w", err)
	}

	srcLocation, _ := cmd.Flags().GetInt("src-location")
	if srcLocation <= 0 {
		output.PrintError("--src-location is required and must be a positive integer", noColor)
		return fmt.Errorf("--src-location is required and must be a positive integer")
	}

	year, _ := cmd.Flags().GetInt("year")
	month, _ := cmd.Flags().GetInt("month")

	if year == 0 || month == 0 {
		// Default to the previous month since RTT data is published after
		// month end — the current month will always return empty results.
		prev := timeNow().AddDate(0, -1, 0)
		if year == 0 {
			year = prev.Year()
		}
		if month == 0 {
			month = int(prev.Month())
		}
	}

	if year <= 0 {
		output.PrintError("--year must be a positive integer", noColor)
		return fmt.Errorf("--year must be a positive integer")
	}

	if month < 1 || month > 12 {
		output.PrintError("--month must be between 1 and 12", noColor)
		return fmt.Errorf("--month must be between 1 and 12")
	}

	// Validate dst-location if provided
	if cmd.Flags().Changed("dst-location") {
		dstLocation, _ := cmd.Flags().GetInt("dst-location")
		if dstLocation <= 0 {
			output.PrintError("--dst-location must be a positive integer", noColor)
			return fmt.Errorf("--dst-location must be a positive integer")
		}
	}

	spinner := output.PrintResourceListing("round-trip time", noColor)

	rtts, err := getRoundTripTimesFunc(ctx, client, srcLocation, year, month)
	spinner.Stop()

	if err != nil {
		output.PrintError("Failed to retrieve round-trip times: %v", noColor, err)
		return fmt.Errorf("failed to get round-trip times: %w", err)
	}

	// Strip nil entries so length checks and counts are accurate.
	cleaned := make([]*megaport.RoundTripTime, 0, len(rtts))
	for _, rtt := range rtts {
		if rtt != nil {
			cleaned = append(cleaned, rtt)
		}
	}
	rtts = cleaned

	// Filter by destination location if specified
	if cmd.Flags().Changed("dst-location") {
		dstLocation, _ := cmd.Flags().GetInt("dst-location")
		var filtered []*megaport.RoundTripTime
		for _, rtt := range rtts {
			if rtt.DstLocation == dstLocation {
				filtered = append(filtered, rtt)
			}
		}
		rtts = filtered
	}

	if len(rtts) == 0 {
		if outputFormat == utils.FormatTable {
			output.PrintInfo("No round-trip time data found.", noColor)
			return nil
		}
		return printRoundTripTimes(rtts, outputFormat, noColor)
	}

	if outputFormat == utils.FormatTable {
		output.PrintInfo("Found %d round-trip time entries", noColor, len(rtts))
	}
	return printRoundTripTimes(rtts, outputFormat, noColor)
}

func GetLocation(cmd *cobra.Command, args []string, noColor bool, outputFormat string) error {
	output.SetOutputFormat(outputFormat)

	ctx, cancel := utils.ContextFromCmd(cmd)
	defer cancel()

	client, err := config.NewUnauthenticatedClient()
	if err != nil {
		output.PrintError("Failed to create API client: %v", noColor, err)
		return fmt.Errorf("failed to create API client: %w", err)
	}

	locationID, err := validation.ParseInt("location ID", args[0])
	if err != nil {
		output.PrintError("%v", noColor, err)
		return err
	}

	spinner := output.PrintResourceGetting("Location", fmt.Sprintf("%d", locationID), noColor)

	locations, err := listLocationsFunc(ctx, client)

	if err != nil {
		spinner.Stop()
		output.PrintError("Failed to retrieve locations: %v", noColor, err)
		return fmt.Errorf("failed to list locations: %w", err)
	}

	var targetLocation *megaport.LocationV3
	for _, loc := range locations {
		if loc == nil {
			continue
		}
		if loc.ID == locationID {
			targetLocation = loc
			break
		}
	}

	if targetLocation == nil {
		spinner.Stop()
		output.PrintWarning("No location found with ID: %d", noColor, locationID)
		return fmt.Errorf("no location found with ID: %d", locationID)
	}

	spinner.StopWithSuccess(fmt.Sprintf("Found location with ID: %d", locationID))

	err = printLocations([]*megaport.LocationV3{targetLocation}, outputFormat, noColor)
	if err != nil {
		output.PrintError("Failed to print location details: %v", noColor, err)
		return err
	}
	return nil
}
