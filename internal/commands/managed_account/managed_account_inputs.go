package managed_account

import (
	"encoding/json"
	"fmt"

	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/megaport/megaport-cli/internal/utils"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
)

func parseManagedAccountRequestJSON(jsonStr, jsonFile string) (*megaport.ManagedAccountRequest, error) {
	jsonData, err := utils.ReadJSONInput(jsonStr, jsonFile)
	if err != nil {
		return nil, exitcodes.NewUsageError(err)
	}

	req := &megaport.ManagedAccountRequest{}
	if err := json.Unmarshal(jsonData, req); err != nil {
		return nil, exitcodes.NewUsageError(fmt.Errorf("failed to parse JSON: %w", err))
	}

	return req, nil
}

func buildManagedAccountRequestFromFlags(cmd *cobra.Command) (*megaport.ManagedAccountRequest, error) { //nolint:unparam
	// Flag read errors are intentionally ignored — flags are registered by the command builder.
	accountName, _ := cmd.Flags().GetString("account-name")
	accountRef, _ := cmd.Flags().GetString("account-ref")

	if accountName == "" || accountRef == "" {
		return nil, exitcodes.NewUsageError(fmt.Errorf("accountName and accountRef are required"))
	}

	req := &megaport.ManagedAccountRequest{
		AccountName: accountName,
		AccountRef:  accountRef,
	}

	return req, nil
}

func buildManagedAccountRequestFromJSON(jsonStr, jsonFile string) (*megaport.ManagedAccountRequest, error) {
	req, err := parseManagedAccountRequestJSON(jsonStr, jsonFile)
	if err != nil {
		return nil, err
	}

	if req.AccountName == "" || req.AccountRef == "" {
		return nil, exitcodes.NewUsageError(fmt.Errorf("accountName and accountRef are required"))
	}

	return req, nil
}

// buildUpdateManagedAccountRequestFromFlags seeds the request from the current
// account so a field the user didn't pass is preserved rather than blanked (the
// SDK request has no omitempty and PUTs every field).
func buildUpdateManagedAccountRequestFromFlags(cmd *cobra.Command, current *megaport.ManagedAccount) (*megaport.ManagedAccountRequest, error) { //nolint:unparam
	req := &megaport.ManagedAccountRequest{
		AccountName: current.AccountName,
		AccountRef:  current.AccountRef,
	}

	if cmd.Flags().Changed("account-name") {
		accountName, _ := cmd.Flags().GetString("account-name")
		req.AccountName = accountName
	}

	if cmd.Flags().Changed("account-ref") {
		accountRef, _ := cmd.Flags().GetString("account-ref")
		req.AccountRef = accountRef
	}

	return req, nil
}

// managedAccountUpdatePatch holds the fields present in an update JSON body.
// Pointer fields distinguish an omitted field from one explicitly set to "".
type managedAccountUpdatePatch struct {
	AccountName *string `json:"accountName"`
	AccountRef  *string `json:"accountRef"`
}

// parseManagedAccountUpdatePatchJSON reads and validates the update JSON body.
// It's called before the login/fetch round-trips so malformed JSON fails fast
// rather than being masked by a later "account not found" error.
func parseManagedAccountUpdatePatchJSON(jsonStr, jsonFile string) (*managedAccountUpdatePatch, error) {
	jsonData, err := utils.ReadJSONInput(jsonStr, jsonFile)
	if err != nil {
		return nil, err
	}

	patch := &managedAccountUpdatePatch{}
	if err := json.Unmarshal(jsonData, patch); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return patch, nil
}

// isEmpty reports whether the body carried no recognized fields, so the update
// can be rejected like the flag and interactive modes rather than sent as a
// no-op PUT.
func (p *managedAccountUpdatePatch) isEmpty() bool {
	return p.AccountName == nil && p.AccountRef == nil
}

// applyTo seeds the request from the current account and overrides only the
// fields present in the patch, so an omitted field keeps its current value
// instead of being sent as an empty string.
func (p *managedAccountUpdatePatch) applyTo(current *megaport.ManagedAccount) *megaport.ManagedAccountRequest {
	req := &megaport.ManagedAccountRequest{
		AccountName: current.AccountName,
		AccountRef:  current.AccountRef,
	}
	if p.AccountName != nil {
		req.AccountName = *p.AccountName
	}
	if p.AccountRef != nil {
		req.AccountRef = *p.AccountRef
	}
	return req
}
