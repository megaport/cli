package mcr

import (
	"context"
	"testing"

	"github.com/megaport/megaport-cli/internal/base/output"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var noColor = true

var testMCRs = []*megaport.MCR{
	{
		UID:                "mcr-1",
		Name:               "MyMCROne",
		LocationID:         1,
		ProvisioningStatus: "ACTIVE",
		PortSpeed:          1000,
		Resources: megaport.MCRResources{
			VirtualRouter: megaport.MCRVirtualRouter{
				ASN: 64512,
			},
		},
	},
	{
		UID:                "mcr-2",
		Name:               "AnotherMCR",
		LocationID:         2,
		ProvisioningStatus: "INACTIVE",
		PortSpeed:          5000,
		Resources: megaport.MCRResources{
			VirtualRouter: megaport.MCRVirtualRouter{
				ASN: 64513,
			},
		},
	},
}

func TestPrintMCRs_Table(t *testing.T) {
	output := output.CaptureOutput(func() {
		err := printMCRs(testMCRs, "table", noColor)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "UID")
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "LOCATION ID")
	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "ASN")
	assert.Contains(t, output, "SPEED")

	assert.Contains(t, output, "mcr-1")
	assert.Contains(t, output, "MyMCROne")
	assert.Contains(t, output, "ACTIVE")
	assert.Contains(t, output, "64512")

	assert.Contains(t, output, "mcr-2")
	assert.Contains(t, output, "AnotherMCR")
	assert.Contains(t, output, "INACTIVE")
	assert.Contains(t, output, "64513")

	assert.Contains(t, output, "┌")
	assert.Contains(t, output, "┐")
	assert.Contains(t, output, "└")
	assert.Contains(t, output, "┘")
	assert.Contains(t, output, "├")
	assert.Contains(t, output, "┤")
	assert.Contains(t, output, "│")
	assert.Contains(t, output, "─")
}

func TestPrintMCRs_JSON(t *testing.T) {
	output := output.CaptureOutput(func() {
		err := printMCRs(testMCRs, "json", noColor)
		assert.NoError(t, err)
	})

	expected := `[
  {
	"uid": "mcr-1",
	"name": "MyMCROne",
	"location_id": 1,
	"provisioning_status": "ACTIVE",
	"asn": 64512,
	"speed": 1000
  },
  {
	"uid": "mcr-2",
	"name": "AnotherMCR",
	"location_id": 2,
	"provisioning_status": "INACTIVE",
	"asn": 64513,
	"speed": 5000
  }
]`
	assert.JSONEq(t, expected, output)
}

func TestPrintMCRs_CSV(t *testing.T) {
	output := output.CaptureOutput(func() {
		err := printMCRs(testMCRs, "csv", noColor)
		assert.NoError(t, err)
	})

	expected := `uid,name,location_id,provisioning_status,asn,speed
mcr-1,MyMCROne,1,ACTIVE,64512,1000
mcr-2,AnotherMCR,2,INACTIVE,64513,5000
`
	assert.Equal(t, expected, output)
}

func TestPrintMCRs_Invalid(t *testing.T) {
	var err error
	output := output.CaptureOutput(func() {
		err = printMCRs(testMCRs, "invalid", noColor)
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
	assert.Empty(t, output)
}

func TestPrintMCRs_EmptyAndNilSlice(t *testing.T) {
	tests := []struct {
		name   string
		mcrs   []*megaport.MCR
		format string
	}{
		{
			name:   "empty slice table format",
			mcrs:   []*megaport.MCR{},
			format: "table",
		},
		{
			name:   "empty slice csv format",
			mcrs:   []*megaport.MCR{},
			format: "csv",
		},
		{
			name:   "empty slice json format",
			mcrs:   []*megaport.MCR{},
			format: "json",
		},
		{
			name:   "nil slice table format",
			mcrs:   nil,
			format: "table",
		},
		{
			name:   "nil slice csv format",
			mcrs:   nil,
			format: "csv",
		},
		{
			name:   "nil slice json format",
			mcrs:   nil,
			format: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := output.CaptureOutput(func() {
				err := printMCRs(tt.mcrs, tt.format, noColor)
				assert.NoError(t, err)
			})

			switch tt.format {
			case "table":
				assert.Contains(t, output, "UID")
				assert.Contains(t, output, "NAME")
				assert.Contains(t, output, "LOCATION ID")
				assert.Contains(t, output, "STATUS")
				assert.Contains(t, output, "ASN")
				assert.Contains(t, output, "SPEED")
				assert.Contains(t, output, "┌")
				assert.Contains(t, output, "┐")
				assert.Contains(t, output, "└")
				assert.Contains(t, output, "┘")
				assert.Contains(t, output, "│")
				assert.Contains(t, output, "─")
			case "csv":
				expected := "uid,name,location_id,provisioning_status,asn,speed\n"
				assert.Equal(t, expected, output)
			case "json":
				assert.Equal(t, "[]\n", output)
			}
		})
	}
}

func TestFilterMCRs(t *testing.T) {
	activeMCRs := []*megaport.MCR{
		{
			UID:                "mcr-1",
			Name:               "TestMCR-1",
			LocationID:         123,
			PortSpeed:          1000,
			ProvisioningStatus: "LIVE",
			Resources: megaport.MCRResources{
				VirtualRouter: megaport.MCRVirtualRouter{
					ASN: 64512,
				},
			},
		},
		{
			UID:                "mcr-2",
			Name:               "TestMCR-2",
			LocationID:         456,
			PortSpeed:          10000,
			ProvisioningStatus: "CONFIGURED",
			Resources: megaport.MCRResources{
				VirtualRouter: megaport.MCRVirtualRouter{
					ASN: 64513,
				},
			},
		},
		{
			UID:                "mcr-3",
			Name:               "Production-MCR",
			LocationID:         123,
			PortSpeed:          10000,
			ProvisioningStatus: "LIVE",
			Resources: megaport.MCRResources{
				VirtualRouter: megaport.MCRVirtualRouter{
					ASN: 64514,
				},
			},
		},
		{
			UID:                "mcr-4",
			Name:               "Staging-MCR",
			LocationID:         789,
			PortSpeed:          5000,
			ProvisioningStatus: "LIVE",
			Resources: megaport.MCRResources{
				VirtualRouter: megaport.MCRVirtualRouter{
					ASN: 64515,
				},
			},
		},
	}

	tests := []struct {
		name         string
		mcrs         []*megaport.MCR
		locationID   int
		portSpeed    int
		mcrName      string
		expected     int
		expectedUIDs []string
	}{
		{
			name:         "no filters",
			mcrs:         activeMCRs,
			locationID:   0,
			portSpeed:    0,
			mcrName:      "",
			expected:     4,
			expectedUIDs: []string{"mcr-1", "mcr-2", "mcr-3", "mcr-4"},
		},
		{
			name:         "filter by location ID",
			mcrs:         activeMCRs,
			locationID:   123,
			portSpeed:    0,
			mcrName:      "",
			expected:     2,
			expectedUIDs: []string{"mcr-1", "mcr-3"},
		},
		{
			name:         "filter by port speed",
			mcrs:         activeMCRs,
			locationID:   0,
			portSpeed:    10000,
			mcrName:      "",
			expected:     2,
			expectedUIDs: []string{"mcr-2", "mcr-3"},
		},
		{
			name:         "filter by name (case insensitive)",
			mcrs:         activeMCRs,
			locationID:   0,
			portSpeed:    0,
			mcrName:      "test",
			expected:     2,
			expectedUIDs: []string{"mcr-1", "mcr-2"},
		},
		{
			name:         "filter by name (partial match)",
			mcrs:         activeMCRs,
			locationID:   0,
			portSpeed:    0,
			mcrName:      "Production",
			expected:     1,
			expectedUIDs: []string{"mcr-3"},
		},
		{
			name:         "multiple filters (location and port speed)",
			mcrs:         activeMCRs,
			locationID:   123,
			portSpeed:    10000,
			mcrName:      "",
			expected:     1,
			expectedUIDs: []string{"mcr-3"},
		},
		{
			name:         "multiple filters (location, port speed, and name)",
			mcrs:         activeMCRs,
			locationID:   123,
			portSpeed:    10000,
			mcrName:      "Production",
			expected:     1,
			expectedUIDs: []string{"mcr-3"},
		},
		{
			name:         "no matching mcrs",
			mcrs:         activeMCRs,
			locationID:   999,
			portSpeed:    0,
			mcrName:      "",
			expected:     0,
			expectedUIDs: []string{},
		},
		{
			name:         "nil slice",
			mcrs:         nil,
			locationID:   0,
			portSpeed:    0,
			mcrName:      "",
			expected:     0,
			expectedUIDs: []string{},
		},
		{
			name:         "empty slice",
			mcrs:         []*megaport.MCR{},
			locationID:   0,
			portSpeed:    0,
			mcrName:      "",
			expected:     0,
			expectedUIDs: []string{},
		},
		{
			name:         "slice with nil mcr",
			mcrs:         []*megaport.MCR{nil, activeMCRs[0], nil, activeMCRs[1]},
			locationID:   0,
			portSpeed:    0,
			mcrName:      "",
			expected:     2,
			expectedUIDs: []string{"mcr-1", "mcr-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterMCRs(tt.mcrs, tt.locationID, tt.portSpeed, tt.mcrName)

			assert.Equal(t, tt.expected, len(filtered), "Filtered MCR count should match expected")

			if len(tt.expectedUIDs) > 0 {
				actualUIDs := make([]string, len(filtered))
				for i, mcr := range filtered {
					actualUIDs[i] = mcr.UID
				}
				assert.ElementsMatch(t, tt.expectedUIDs, actualUIDs, "Filtered MCR UIDs should match expected")
			}
		})
	}
}

func TestMCRUpdateTagsHasGenerateSkeleton(t *testing.T) {
	_, updateTags := buildMCRTagCommands()
	assert.NotNil(t, updateTags.Flags().Lookup("generate-skeleton"))
}

func TestMCRListHasTagFlag(t *testing.T) {
	_, _, _, _, _, _, list, _, _ := buildMCRCommands(nil)
	require.NotNil(t, list.Flags().Lookup("tag"), "list command should have --tag flag")
}

func TestMCRBuyAndValidateRejectPositionalArgs(t *testing.T) {
	_, buy, _, _, _, _, _, _, validate := buildMCRCommands(nil)

	require.NotNil(t, buy.Args, "buy command should restrict positional args")
	assert.Error(t, buy.Args(buy, []string{"false"}),
		"buy should reject a stray positional arg, e.g. from --marketplace-visibility false being parsed as --marketplace-visibility=true plus a bare \"false\" token")

	require.NotNil(t, validate.Args, "validate command should restrict positional args")
	assert.Error(t, validate.Args(validate, []string{"false"}),
		"validate should reject a stray positional arg, e.g. from --marketplace-visibility false being parsed as --marketplace-visibility=true plus a bare \"false\" token")
}

func TestMCRModule(t *testing.T) {
	m := NewModule()
	assert.Equal(t, "mcr", m.Name())

	root := &cobra.Command{Use: "megaport-cli"}
	m.RegisterCommands(root)

	found := false
	for _, cmd := range root.Commands() {
		if cmd.Use == "mcr" {
			found = true
			break
		}
	}
	assert.True(t, found, "RegisterCommands should add mcr command")
}

func TestListMCRResourceTagsFunc(t *testing.T) {
	want := map[string]string{"env": "staging"}
	mockSvc := &MockMCRService{ListMCRResourceTagsResult: want}
	client := &megaport.Client{}
	client.MCRService = mockSvc

	got, err := listMCRResourceTagsFunc(context.Background(), client, "mcr-uid-1")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
