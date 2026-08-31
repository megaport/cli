package nat_gateway

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCreateFlagsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "create"}
	cmd.Flags().String("name", "", "")
	cmd.Flags().Int("term", 0, "")
	cmd.Flags().Int("speed", 0, "")
	cmd.Flags().Int("location-id", 0, "")
	cmd.Flags().Int("session-count", 0, "")
	cmd.Flags().String("diversity-zone", "", "")
	cmd.Flags().String("promo-code", "", "")
	cmd.Flags().String("service-level-reference", "", "")
	cmd.Flags().Bool("auto-renew", false, "")
	cmd.Flags().String("resource-tags", "", "")
	cmd.Flags().String("resource-tags-file", "", "")
	return cmd
}

func newUpdateFlagsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "update"}
	cmd.Flags().String("name", "", "")
	cmd.Flags().Int("term", 0, "")
	cmd.Flags().Int("speed", 0, "")
	cmd.Flags().Int("location-id", 0, "")
	cmd.Flags().Int("session-count", 0, "")
	cmd.Flags().String("diversity-zone", "", "")
	cmd.Flags().String("promo-code", "", "")
	cmd.Flags().String("service-level-reference", "", "")
	cmd.Flags().Bool("auto-renew", false, "")
	cmd.Flags().String("resource-tags", "", "")
	cmd.Flags().String("resource-tags-file", "", "")
	return cmd
}

// ---- processJSONCreateNATGatewayInput ----

func TestProcessJSONCreateNATGatewayInput_EmptyStrings(t *testing.T) {
	_, err := processJSONCreateNATGatewayInput("", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON")
}

func TestProcessJSONCreateNATGatewayInput_InvalidJSON(t *testing.T) {
	_, err := processJSONCreateNATGatewayInput(`{invalid}`, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON")

	var cliErr *exitcodes.CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}

func TestProcessJSONCreateNATGatewayInput_FileNotFound(t *testing.T) {
	_, err := processJSONCreateNATGatewayInput("", filepath.Join(t.TempDir(), "missing.json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read JSON file")
}

func TestProcessJSONCreateNATGatewayInput_ValidFile(t *testing.T) {
	content := `{"name":"File GW","term":12,"speed":1000,"locationId":1}`
	tmp, err := os.CreateTemp("", "ng-create-*.json")
	require.NoError(t, err)
	defer os.Remove(tmp.Name())
	_, err = tmp.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmp.Close())

	req, err := processJSONCreateNATGatewayInput("", tmp.Name())
	require.NoError(t, err)
	assert.Equal(t, "File GW", req.ProductName)
	assert.Equal(t, 12, req.Term)
	assert.Equal(t, 1000, req.Speed)
	assert.Equal(t, 1, req.LocationID)
}

func TestProcessJSONCreateNATGatewayInput_WithBooleanFields(t *testing.T) {
	req, err := processJSONCreateNATGatewayInput(
		`{"name":"GW","term":12,"speed":1000,"locationId":1,"bgpShutdownDefault":true,"autoRenewTerm":true}`, "")
	require.NoError(t, err)
	assert.True(t, req.Config.BGPShutdownDefault)
	assert.True(t, req.AutoRenewTerm)
}

func TestProcessJSONCreateNATGatewayInput_RejectsEmptyTagKey(t *testing.T) {
	_, err := processJSONCreateNATGatewayInput(
		`{"name":"GW","term":12,"speed":1000,"locationId":1,"resourceTags":{"":"x"}}`, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tag key must not be empty")

	var cliErr *exitcodes.CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}

func TestProcessJSONCreateNATGatewayInput_RejectsEmptyTagKeyFromFile(t *testing.T) {
	content := `{"name":"GW","term":12,"speed":1000,"locationId":1,"resourceTags":{"":"x"}}`
	tmp, err := os.CreateTemp("", "ng-create-emptytag-*.json")
	require.NoError(t, err)
	defer os.Remove(tmp.Name())
	_, err = tmp.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmp.Close())

	_, err = processJSONCreateNATGatewayInput("", tmp.Name())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tag key must not be empty")
}

func TestProcessJSONCreateNATGatewayInput_ValidResourceTags(t *testing.T) {
	req, err := processJSONCreateNATGatewayInput(
		`{"name":"GW","term":12,"speed":1000,"locationId":1,"resourceTags":{"env":"prod"}}`, "")
	require.NoError(t, err)
	require.Len(t, req.ResourceTags, 1)
	assert.Equal(t, "env", req.ResourceTags[0].Key)
	assert.Equal(t, "prod", req.ResourceTags[0].Value)
}

// ---- processFlagCreateNATGatewayInput ----

func TestProcessFlagCreateNATGatewayInput_Valid(t *testing.T) {
	cmd := newCreateFlagsCmd()
	require.NoError(t, cmd.Flags().Set("name", "Test GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))

	req, err := processFlagCreateNATGatewayInput(cmd)
	require.NoError(t, err)
	assert.Equal(t, "Test GW", req.ProductName)
	assert.Equal(t, 12, req.Term)
	assert.Equal(t, 1000, req.Speed)
	assert.Equal(t, 1, req.LocationID)
}

func TestProcessFlagCreateNATGatewayInput_WithOptionalFields(t *testing.T) {
	cmd := newCreateFlagsCmd()
	require.NoError(t, cmd.Flags().Set("name", "Test GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))
	require.NoError(t, cmd.Flags().Set("session-count", "500"))
	require.NoError(t, cmd.Flags().Set("diversity-zone", "blue"))
	require.NoError(t, cmd.Flags().Set("promo-code", "SAVE10"))
	require.NoError(t, cmd.Flags().Set("auto-renew", "true"))

	req, err := processFlagCreateNATGatewayInput(cmd)
	require.NoError(t, err)
	assert.Equal(t, 500, req.Config.SessionCount)
	assert.Equal(t, "blue", req.Config.DiversityZone)
	assert.Equal(t, "SAVE10", req.PromoCode)
	assert.True(t, req.AutoRenewTerm)
}

func TestProcessFlagCreateNATGatewayInput_MissingName(t *testing.T) {
	cmd := newCreateFlagsCmd()
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))

	_, err := processFlagCreateNATGatewayInput(cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestProcessFlagCreateNATGatewayInput_InvalidResourceTagsJSON(t *testing.T) {
	cmd := newCreateFlagsCmd()
	require.NoError(t, cmd.Flags().Set("name", "Test GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))
	require.NoError(t, cmd.Flags().Set("resource-tags", `{invalid}`))

	_, err := processFlagCreateNATGatewayInput(cmd)
	require.Error(t, err)

	var cliErr *exitcodes.CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}

func TestProcessFlagCreateNATGatewayInput_ValidResourceTags(t *testing.T) {
	cmd := newCreateFlagsCmd()
	require.NoError(t, cmd.Flags().Set("name", "Tag GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))
	require.NoError(t, cmd.Flags().Set("resource-tags", `{"env":"prod","team":"platform"}`))

	req, err := processFlagCreateNATGatewayInput(cmd)
	require.NoError(t, err)
	assert.Len(t, req.ResourceTags, 2)
}

func TestProcessFlagCreateNATGatewayInput_RejectsEmptyTagKey(t *testing.T) {
	cmd := newCreateFlagsCmd()
	require.NoError(t, cmd.Flags().Set("name", "Tag GW"))
	require.NoError(t, cmd.Flags().Set("term", "12"))
	require.NoError(t, cmd.Flags().Set("speed", "1000"))
	require.NoError(t, cmd.Flags().Set("location-id", "1"))
	require.NoError(t, cmd.Flags().Set("resource-tags", `{"":"x"}`))

	_, err := processFlagCreateNATGatewayInput(cmd)
	require.Error(t, err)

	var cliErr *exitcodes.CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}

// ---- processJSONUpdateNATGatewayInput ----

func TestProcessJSONUpdateNATGatewayInput_RejectsEmptyTagKey(t *testing.T) {
	_, _, err := processJSONUpdateNATGatewayInput(
		`{"name":"GW","resourceTags":{"":"x"}}`, "", "uid-empty-tag")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tag key must not be empty")

	var cliErr *exitcodes.CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}

func TestProcessJSONUpdateNATGatewayInput_ValidResourceTags(t *testing.T) {
	req, _, err := processJSONUpdateNATGatewayInput(
		`{"name":"GW","resourceTags":{"env":"prod"}}`, "", "uid-valid-tag")
	require.NoError(t, err)
	require.Len(t, req.ResourceTags, 1)
	assert.Equal(t, "env", req.ResourceTags[0].Key)
	assert.Equal(t, "prod", req.ResourceTags[0].Value)
}

func TestProcessJSONUpdateNATGatewayInput_Valid(t *testing.T) {
	req, explicit, err := processJSONUpdateNATGatewayInput(
		`{"name":"Updated GW","locationId":2,"speed":2000,"term":24}`, "", "uid-123")
	require.NoError(t, err)
	assert.Equal(t, "uid-123", req.ProductUID)
	assert.Equal(t, "Updated GW", req.ProductName)
	assert.Equal(t, 2, req.LocationID)
	assert.Equal(t, 2000, req.Speed)
	assert.Equal(t, 24, req.Term)
	assert.False(t, explicit.AutoRenewTerm)
}

func TestProcessJSONUpdateNATGatewayInput_EmptyStrings(t *testing.T) {
	_, _, err := processJSONUpdateNATGatewayInput("", "", "uid-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON")
}

func TestProcessJSONUpdateNATGatewayInput_InvalidJSON(t *testing.T) {
	_, _, err := processJSONUpdateNATGatewayInput(`{bad}`, "", "uid-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON")
}

func TestProcessJSONUpdateNATGatewayInput_FileNotFound(t *testing.T) {
	_, _, err := processJSONUpdateNATGatewayInput("", filepath.Join(t.TempDir(), "missing.json"), "uid-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read JSON file")
}

func TestProcessJSONUpdateNATGatewayInput_ValidFile(t *testing.T) {
	content := `{"name":"File Update GW","speed":3000}`
	tmp, err := os.CreateTemp("", "ng-update-*.json")
	require.NoError(t, err)
	defer os.Remove(tmp.Name())
	_, err = tmp.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmp.Close())

	req, _, err := processJSONUpdateNATGatewayInput("", tmp.Name(), "uid-file")
	require.NoError(t, err)
	assert.Equal(t, "uid-file", req.ProductUID)
	assert.Equal(t, "File Update GW", req.ProductName)
	assert.Equal(t, 3000, req.Speed)
}

func TestProcessJSONUpdateNATGatewayInput_ExplicitBoolFields(t *testing.T) {
	tests := []struct {
		name            string
		json            string
		wantAutoRenew   bool
		wantBGPShutdown bool
		autoRenewVal    bool
		bgpShutdownVal  bool
	}{
		{
			name:          "autoRenewTerm true",
			json:          `{"autoRenewTerm":true}`,
			wantAutoRenew: true,
			autoRenewVal:  true,
		},
		{
			name:            "bgpShutdownDefault true",
			json:            `{"bgpShutdownDefault":true}`,
			wantBGPShutdown: true,
			bgpShutdownVal:  true,
		},
		{
			name:          "autoRenewTerm false explicit",
			json:          `{"autoRenewTerm":false}`,
			wantAutoRenew: true,
			autoRenewVal:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, explicit, err := processJSONUpdateNATGatewayInput(tt.json, "", "uid-bool")
			require.NoError(t, err)
			assert.Equal(t, tt.wantAutoRenew, explicit.AutoRenewTerm)
			assert.Equal(t, tt.wantBGPShutdown, explicit.BGPShutdownDefault)
			if tt.wantAutoRenew {
				assert.Equal(t, tt.autoRenewVal, req.AutoRenewTerm)
			}
			if tt.wantBGPShutdown {
				assert.Equal(t, tt.bgpShutdownVal, req.Config.BGPShutdownDefault)
			}
		})
	}
}

func TestProcessJSONUpdateNATGatewayInput_ExplicitSessionCount(t *testing.T) {
	req, explicit, err := processJSONUpdateNATGatewayInput(
		`{"sessionCount":250}`, "", "uid-sc")
	require.NoError(t, err)
	assert.True(t, explicit.SessionCount)
	assert.Equal(t, 250, req.Config.SessionCount)
}

func TestProcessJSONUpdateNATGatewayInput_ExplicitDiversityZone(t *testing.T) {
	req, explicit, err := processJSONUpdateNATGatewayInput(
		`{"diversityZone":"red"}`, "", "uid-dz")
	require.NoError(t, err)
	assert.True(t, explicit.DiversityZone)
	assert.Equal(t, "red", req.Config.DiversityZone)
}

// ---- processFlagUpdateNATGatewayInput ----

func TestProcessFlagUpdateNATGatewayInput_Valid(t *testing.T) {
	cmd := newUpdateFlagsCmd()
	require.NoError(t, cmd.Flags().Set("name", "Updated GW"))
	require.NoError(t, cmd.Flags().Set("term", "24"))
	require.NoError(t, cmd.Flags().Set("speed", "2000"))
	require.NoError(t, cmd.Flags().Set("location-id", "2"))

	req, err := processFlagUpdateNATGatewayInput(cmd, "uid-upd")
	require.NoError(t, err)
	assert.Equal(t, "uid-upd", req.ProductUID)
	assert.Equal(t, "Updated GW", req.ProductName)
	assert.Equal(t, 24, req.Term)
}

func TestProcessFlagUpdateNATGatewayInput_EmptyUID(t *testing.T) {
	cmd := newUpdateFlagsCmd()
	_, err := processFlagUpdateNATGatewayInput(cmd, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "product UID is required")
}

func TestProcessFlagUpdateNATGatewayInput_WithResourceTags(t *testing.T) {
	cmd := newUpdateFlagsCmd()
	require.NoError(t, cmd.Flags().Set("resource-tags", `{"env":"staging"}`))

	req, err := processFlagUpdateNATGatewayInput(cmd, "uid-tags")
	require.NoError(t, err)
	assert.Equal(t, "uid-tags", req.ProductUID)
	assert.Len(t, req.ResourceTags, 1)
}

func TestProcessFlagUpdateNATGatewayInput_InvalidResourceTags(t *testing.T) {
	cmd := newUpdateFlagsCmd()
	require.NoError(t, cmd.Flags().Set("resource-tags", `{invalid}`))

	_, err := processFlagUpdateNATGatewayInput(cmd, "uid-bad")
	require.Error(t, err)

	var cliErr *exitcodes.CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}

func TestProcessFlagUpdateNATGatewayInput_RejectsEmptyTagKey(t *testing.T) {
	cmd := newUpdateFlagsCmd()
	require.NoError(t, cmd.Flags().Set("resource-tags", `{"":"x"}`))

	_, err := processFlagUpdateNATGatewayInput(cmd, "uid-empty-tag")
	require.Error(t, err)

	var cliErr *exitcodes.CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}
