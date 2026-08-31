package ix

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/megaport/megaport-cli/internal/validation"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildIXRequestFromJSON_BothEmpty(t *testing.T) {
	req, err := buildIXRequestFromJSON("", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON")
	assert.Nil(t, req)

	var cliErr *exitcodes.CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}

func TestBuildUpdateIXRequestFromJSON_BothEmpty(t *testing.T) {
	req, err := buildUpdateIXRequestFromJSON("", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON")
	assert.Nil(t, req)

	var cliErr *exitcodes.CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}

func TestBuildUpdateIXRequestFromJSON_FileNotFound(t *testing.T) {
	req, err := buildUpdateIXRequestFromJSON("", filepath.Join(t.TempDir(), "missing.json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read JSON file")
	assert.Nil(t, req)

	var cliErr *exitcodes.CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}

func TestBuildIXRequestFromJSON_AllFields(t *testing.T) {
	const validJSON = `{
		"productUid": "port-abc",
		"productName": "My IX",
		"networkServiceType": "Chicago IX",
		"asn": 64512,
		"macAddress": "DE:AD:BE:EF:00:01",
		"rateLimit": 500,
		"vlan": 42,
		"shutdown": true,
		"promoCode": "SAVE10"
	}`
	req, err := buildIXRequestFromJSON(validJSON, "")
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, "port-abc", req.ProductUID)
	assert.Equal(t, "My IX", req.Name)
	assert.Equal(t, "Chicago IX", req.NetworkServiceType)
	assert.Equal(t, 64512, req.ASN)
	assert.Equal(t, "DE:AD:BE:EF:00:01", req.MACAddress)
	assert.Equal(t, 500, req.RateLimit)
	assert.Equal(t, 42, req.VLAN)
	assert.True(t, req.Shutdown)
	assert.Equal(t, "SAVE10", req.PromoCode)
}

func TestBuildIXRequestFromJSON_TempFile(t *testing.T) {
	content := `{"productUid":"port-file","productName":"File IX","asn":65000,"rateLimit":1000,"vlan":100,"macAddress":"00:11:22:33:44:55","networkServiceType":"London IX"}`
	tmpFile, err := os.CreateTemp("", "ix-inputs-test-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	req, err := buildIXRequestFromJSON("", tmpFile.Name())
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, "port-file", req.ProductUID)
	assert.Equal(t, "File IX", req.Name)
}

func TestBuildIXRequestFromJSON_FileNotFound(t *testing.T) {
	_, err := buildIXRequestFromJSON("", filepath.Join(t.TempDir(), "missing.json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read JSON file")
}

func TestBuildUpdateIXRequestFromJSON_PointerFields(t *testing.T) {
	tests := []struct {
		name      string
		jsonStr   string
		checkFunc func(t *testing.T, req *megaport.UpdateIXRequest)
	}{
		{
			name:    "publicGraph true",
			jsonStr: `{"publicGraph":true}`,
			checkFunc: func(t *testing.T, req *megaport.UpdateIXRequest) {
				require.NotNil(t, req.PublicGraph)
				assert.True(t, *req.PublicGraph)
			},
		},
		{
			name:    "shutdown false explicit",
			jsonStr: `{"shutdown":false}`,
			checkFunc: func(t *testing.T, req *megaport.UpdateIXRequest) {
				require.NotNil(t, req.Shutdown)
				assert.False(t, *req.Shutdown)
			},
		},
		{
			name:    "aEndProductUid set",
			jsonStr: `{"aEndProductUid":"port-new"}`,
			checkFunc: func(t *testing.T, req *megaport.UpdateIXRequest) {
				require.NotNil(t, req.AEndProductUid)
				assert.Equal(t, "port-new", *req.AEndProductUid)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := buildUpdateIXRequestFromJSON(tt.jsonStr, "")
			require.NoError(t, err)
			tt.checkFunc(t, req)
		})
	}
}

func TestBuildUpdateIXRequestFromJSON_AtLeastOneFieldExitCode(t *testing.T) {
	req, err := buildUpdateIXRequestFromJSON(`{}`, "")
	assert.Nil(t, req)
	assert.Contains(t, err.Error(), "at least one field must be updated")

	var cliErr *exitcodes.CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, exitcodes.Usage, cliErr.Code)
}

func TestBuildUpdateIXRequestFromJSON_EmptyNameRejected(t *testing.T) {
	req, err := buildUpdateIXRequestFromJSON(`{"name":""}`, "")
	assert.Nil(t, req)

	var valErr *validation.ValidationError
	require.True(t, errors.As(err, &valErr))
	assert.Equal(t, "name", valErr.Field)
}

func TestBuildUpdateIXRequestFromFlags_EmptyNameRejected(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("name", "", "")
	require.NoError(t, cmd.Flags().Set("name", ""))
	cmd.Flags().Lookup("name").Changed = true

	req, err := buildUpdateIXRequestFromFlags(cmd)
	assert.Nil(t, req)

	var valErr *validation.ValidationError
	require.True(t, errors.As(err, &valErr))
	assert.Equal(t, "name", valErr.Field)
}

func TestBuildUpdateIXRequestFromFlags_NoopWhenUnchanged(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("name", "", "")
	cmd.Flags().Int("rate-limit", 0, "")
	cmd.Flags().String("cost-centre", "", "")
	cmd.Flags().Int("vlan", 0, "")
	cmd.Flags().String("mac-address", "", "")
	cmd.Flags().Int("asn", 0, "")
	cmd.Flags().String("password", "", "")
	cmd.Flags().Bool("public-graph", false, "")
	cmd.Flags().String("reverse-dns", "", "")
	cmd.Flags().String("a-end-product-uid", "", "")
	cmd.Flags().Bool("shutdown", false, "")

	req, err := buildUpdateIXRequestFromFlags(cmd)
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Nil(t, req.Name)
	assert.Nil(t, req.RateLimit)
	assert.Nil(t, req.CostCentre)
	assert.Nil(t, req.VLAN)
	assert.Nil(t, req.MACAddress)
	assert.Nil(t, req.ASN)
	assert.Nil(t, req.Password)
	assert.Nil(t, req.PublicGraph)
	assert.Nil(t, req.ReverseDns)
	assert.Nil(t, req.AEndProductUid)
	assert.Nil(t, req.Shutdown)
}
