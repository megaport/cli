package mcr

import (
	"testing"

	op "github.com/megaport/megaport-cli/internal/base/output"
	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
)

func TestPrintMCRs_XML(t *testing.T) {
	out := op.CaptureOutput(func() {
		err := printMCRs(testMCRs, "xml", noColor)
		assert.NoError(t, err)
	})
	assert.NotEmpty(t, out)
	assert.Contains(t, out, "<items>")
	assert.Contains(t, out, "<uid>")
	assert.Contains(t, out, "mcr-1")
	assert.Contains(t, out, "MyMCROne")
}

func TestToMCROutput_NilMCR(t *testing.T) {
	_, err := toMCROutput(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil value")
}

func TestToMCROutput_Valid(t *testing.T) {
	mcr := &megaport.MCR{
		UID:                "mcr-123",
		Name:               "Test MCR",
		LocationID:         1,
		ProvisioningStatus: "LIVE",
		PortSpeed:          5000,
	}
	mcr.Resources.VirtualRouter.ASN = 65000

	out, err := toMCROutput(mcr)
	assert.NoError(t, err)
	assert.Equal(t, "mcr-123", out.UID)
	assert.Equal(t, "Test MCR", out.Name)
	assert.Equal(t, 65000, out.ASN)
	assert.Equal(t, 5000, out.Speed)
}

func TestToPrefixFilterListOutput_NilPFL(t *testing.T) {
	_, err := toPrefixFilterListOutput(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil value")
}

func TestToPrefixFilterListOutput_Valid(t *testing.T) {
	pfl := &megaport.MCRPrefixFilterList{
		ID:            1,
		Description:   "Test PFL",
		AddressFamily: "IPv4",
		Entries: []*megaport.MCRPrefixListEntry{
			{Action: "permit", Prefix: "10.0.0.0/8", Ge: megaport.PtrTo(16), Le: megaport.PtrTo(24)},
		},
	}

	out, err := toPrefixFilterListOutput(pfl)
	assert.NoError(t, err)
	assert.Equal(t, 1, out.ID)
	assert.Equal(t, "Test PFL", out.Description)
	assert.Equal(t, "IPv4", out.AddressFamily)
	assert.Len(t, out.Entries, 1)
	assert.Equal(t, "permit", out.Entries[0].Action)
}

func TestToMCROutput_WithAddOns(t *testing.T) {
	mcr := &megaport.MCR{
		UID:                "mcr-123",
		Name:               "Test MCR",
		LocationID:         1,
		ProvisioningStatus: "LIVE",
		PortSpeed:          5000,
		AddOns: []*megaport.MCRAddOnIPsecConfig{
			{AddOnUID: "addon-1", AddOnType: "IP_SEC", TunnelCount: 10},
			nil, // nil entries should be skipped
		},
	}

	out, err := toMCROutput(mcr)
	assert.NoError(t, err)
	assert.Len(t, out.AddOns, 1)
	assert.Equal(t, "addon-1", out.AddOns[0].AddOnUID)
	assert.Equal(t, "IP_SEC", out.AddOns[0].AddOnType)
	assert.Equal(t, 10, out.AddOns[0].TunnelCount)
}

func TestDisplayMCRChanges(t *testing.T) {
	tests := []struct {
		name             string
		original         *megaport.MCR
		updated          *megaport.MCR
		expectedContains []string
	}{
		{
			name:     "nil original",
			original: nil,
			updated:  &megaport.MCR{},
		},
		{
			name:     "nil updated",
			original: &megaport.MCR{},
			updated:  nil,
		},
		{
			name:             "no changes",
			original:         &megaport.MCR{Name: "Same", CostCentre: "CC1", ContractTermMonths: 12},
			updated:          &megaport.MCR{Name: "Same", CostCentre: "CC1", ContractTermMonths: 12},
			expectedContains: []string{"No changes detected"},
		},
		{
			name:             "name changed",
			original:         &megaport.MCR{Name: "Old MCR"},
			updated:          &megaport.MCR{Name: "New MCR"},
			expectedContains: []string{"Name", "Old MCR", "New MCR"},
		},
		{
			name:             "cost centre changed from empty",
			original:         &megaport.MCR{CostCentre: ""},
			updated:          &megaport.MCR{CostCentre: "IT-2024"},
			expectedContains: []string{"Cost Centre", "(none)", "IT-2024"},
		},
		{
			name:             "contract term changed",
			original:         &megaport.MCR{ContractTermMonths: 12},
			updated:          &megaport.MCR{ContractTermMonths: 24},
			expectedContains: []string{"Contract Term", "12 months", "24 months"},
		},
		{
			name:             "marketplace visibility changed",
			original:         &megaport.MCR{MarketplaceVisibility: false},
			updated:          &megaport.MCR{MarketplaceVisibility: true},
			expectedContains: []string{"Marketplace Visibility", "No", "Yes"},
		},
		{
			name: "ASN changed",
			original: func() *megaport.MCR {
				m := &megaport.MCR{}
				m.Resources.VirtualRouter.ASN = 65000
				return m
			}(),
			updated: func() *megaport.MCR {
				m := &megaport.MCR{}
				m.Resources.VirtualRouter.ASN = 65001
				return m
			}(),
			expectedContains: []string{"ASN", "65000", "65001"},
		},
		{
			name:             "multiple changes",
			original:         &megaport.MCR{Name: "Old", ContractTermMonths: 12, MarketplaceVisibility: false},
			updated:          &megaport.MCR{Name: "New", ContractTermMonths: 24, MarketplaceVisibility: true},
			expectedContains: []string{"Name", "Contract Term", "Marketplace Visibility"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := op.CaptureOutput(func() {
				displayMCRChanges(tt.original, tt.updated, true)
			})

			for _, expected := range tt.expectedContains {
				assert.Contains(t, output, expected)
			}
		})
	}
}
