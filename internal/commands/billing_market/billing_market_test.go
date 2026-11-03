package billing_market

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/testutil"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func captureStderr(t *testing.T, fn func()) (result string) {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() { os.Stderr = old }()
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() { defer close(done); _, _ = io.Copy(&buf, r) }()
	defer func() { _ = w.Close(); <-done; _ = r.Close(); result = buf.String() }()
	fn()
	return
}

var mockBillingMarkets = []*megaport.BillingMarket{
	{
		ID:                  1,
		SupplierName:        "Megaport US",
		CurrencyEnum:        "USD",
		Country:             "US",
		Region:              "North America",
		BillingContactName:  "John Doe",
		BillingContactEmail: "john@example.com",
		BillingContactPhone: "+14155552671",
		Address1:            "123 Main St",
		City:                "New York",
		State:               "NY",
		Postcode:            "10001",
		Language:            "en",
		FirstPartyID:        1558,
		Active:              true,
	},
	{
		ID:                  2,
		SupplierName:        "Megaport AU",
		CurrencyEnum:        "AUD",
		Country:             "AU",
		Region:              "Oceania",
		BillingContactName:  "Jane Smith",
		BillingContactEmail: "jane@example.com",
		BillingContactPhone: "+61400000000",
		Address1:            "456 George St",
		City:                "Sydney",
		State:               "NSW",
		Postcode:            "2000",
		Language:            "en",
		FirstPartyID:        808,
		Active:              true,
	},
}

func TestToBillingMarketOutput(t *testing.T) {
	bm := mockBillingMarkets[0]
	out := toBillingMarketOutput(bm)

	assert.Equal(t, bm.ID, out.ID)
	assert.Equal(t, bm.SupplierName, out.SupplierName)
	assert.Equal(t, bm.CurrencyEnum, out.CurrencyEnum)
	assert.Equal(t, bm.Country, out.Country)
	assert.Equal(t, bm.Region, out.Region)
	assert.Equal(t, bm.BillingContactName, out.BillingContactName)
	assert.Equal(t, bm.BillingContactEmail, out.BillingContactEmail)
	assert.Equal(t, bm.Active, out.Active)
}

func TestBillingMarketOutput_Table(t *testing.T) {
	outputs := make([]billingMarketOutput, 0, len(mockBillingMarkets))
	for _, bm := range mockBillingMarkets {
		outputs = append(outputs, toBillingMarketOutput(bm))
	}

	out := output.CaptureOutput(func() {
		err := output.PrintOutput(outputs, "table", true)
		assert.NoError(t, err)
	})

	assert.Contains(t, out, "ID")
	assert.Contains(t, out, "SUPPLIER NAME")
	assert.Contains(t, out, "CURRENCY")
	assert.Contains(t, out, "COUNTRY")
	assert.Contains(t, out, "CONTACT NAME")
	assert.Contains(t, out, "ACTIVE")
	assert.Contains(t, out, "Megaport US")
	assert.Contains(t, out, "Megaport AU")
	assert.Contains(t, out, "USD")
	assert.Contains(t, out, "AUD")
}

func TestBillingMarketOutput_JSON(t *testing.T) {
	outputs := make([]billingMarketOutput, 0, len(mockBillingMarkets))
	for _, bm := range mockBillingMarkets {
		outputs = append(outputs, toBillingMarketOutput(bm))
	}

	out := output.CaptureOutput(func() {
		err := output.PrintOutput(outputs, "json", false)
		assert.NoError(t, err)
	})

	expected := `[
  {
    "id": 1,
    "supplier_name": "Megaport US",
    "currency": "USD",
    "country": "US",
    "region": "North America",
    "billing_contact_name": "John Doe",
    "billing_contact_email": "john@example.com",
    "billing_contact_phone": "+14155552671",
    "address1": "123 Main St",
    "city": "New York",
    "state": "NY",
    "postcode": "10001",
    "language": "en",
    "tax_rate": 0,
    "first_party_id": 1558,
    "second_party_id": 0,
    "payment_term_in_days": 0,
    "active": true,
    "vat_exempt": false
  },
  {
    "id": 2,
    "supplier_name": "Megaport AU",
    "currency": "AUD",
    "country": "AU",
    "region": "Oceania",
    "billing_contact_name": "Jane Smith",
    "billing_contact_email": "jane@example.com",
    "billing_contact_phone": "+61400000000",
    "address1": "456 George St",
    "city": "Sydney",
    "state": "NSW",
    "postcode": "2000",
    "language": "en",
    "tax_rate": 0,
    "first_party_id": 808,
    "second_party_id": 0,
    "payment_term_in_days": 0,
    "active": true,
    "vat_exempt": false
  }
]`
	assert.JSONEq(t, expected, out)
}

func TestBillingMarketOutput_CSV(t *testing.T) {
	outputs := make([]billingMarketOutput, 0, len(mockBillingMarkets))
	for _, bm := range mockBillingMarkets {
		outputs = append(outputs, toBillingMarketOutput(bm))
	}

	out := output.CaptureOutput(func() {
		err := output.PrintOutput(outputs, "csv", false)
		assert.NoError(t, err)
	})

	lines := strings.Split(out, "\n")
	assert.GreaterOrEqual(t, len(lines), 3) // header + 2 data lines
	assert.Contains(t, lines[0], "id,supplier_name,currency,country")
	assert.Contains(t, lines[1], "Megaport US")
	assert.Contains(t, lines[2], "Megaport AU")
}

func TestGetBillingMarketsAction(t *testing.T) {
	mockSvc := &MockBillingMarketService{
		GetBillingMarketsResult: mockBillingMarkets,
	}

	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.BillingMarketService = mockSvc
	})
	defer cleanup()

	cmd := &cobra.Command{Use: "get"}

	out := output.CaptureOutput(func() {
		err := GetBillingMarkets(cmd, []string{}, true, "json")
		assert.NoError(t, err)
	})

	assert.Contains(t, out, "Megaport US")
	assert.Contains(t, out, "Megaport AU")
	assert.Contains(t, out, "USD")
	assert.Contains(t, out, "AUD")
}

func TestGetBillingMarketsAction_Error(t *testing.T) {
	mockSvc := &MockBillingMarketService{
		GetBillingMarketsError: assert.AnError,
	}

	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.BillingMarketService = mockSvc
	})
	defer cleanup()

	cmd := &cobra.Command{Use: "get"}

	err := GetBillingMarkets(cmd, []string{}, true, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get billing markets")
}

func TestGetBillingMarketsAction_Empty(t *testing.T) {
	mockSvc := &MockBillingMarketService{
		GetBillingMarketsResult: []*megaport.BillingMarket{},
	}

	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.BillingMarketService = mockSvc
	})
	defer cleanup()

	cmd := &cobra.Command{Use: "get"}

	err := GetBillingMarkets(cmd, []string{}, true, "json")
	assert.NoError(t, err)
}

func TestGetBillingMarketsAction_NilEntriesSkipped(t *testing.T) {
	mockSvc := &MockBillingMarketService{
		GetBillingMarketsResult: []*megaport.BillingMarket{nil, mockBillingMarkets[0], nil},
	}

	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.BillingMarketService = mockSvc
	})
	defer cleanup()

	cmd := &cobra.Command{Use: "get"}

	assert.NotPanics(t, func() {
		out := output.CaptureOutput(func() {
			err := GetBillingMarkets(cmd, []string{}, true, "json")
			assert.NoError(t, err)
		})
		assert.Contains(t, out, "Megaport US")
	})
}

func TestGetBillingMarketsAction_AllNilWarns(t *testing.T) {
	mockSvc := &MockBillingMarketService{
		GetBillingMarketsResult: []*megaport.BillingMarket{nil, nil},
	}

	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.BillingMarketService = mockSvc
	})
	defer cleanup()

	cmd := &cobra.Command{Use: "get"}

	out := captureStderr(t, func() {
		err := GetBillingMarkets(cmd, []string{}, true, "table")
		assert.NoError(t, err)
	})

	assert.Contains(t, out, "No billing markets found")
}

func TestSetBillingMarketAction(t *testing.T) {
	mockSvc := &MockBillingMarketService{}

	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.BillingMarketService = mockSvc
	})
	defer cleanup()

	cmd := &cobra.Command{Use: "set"}
	cmd.Flags().String("currency", "", "")
	cmd.Flags().String("language", "", "")
	cmd.Flags().String("billing-contact-name", "", "")
	cmd.Flags().String("billing-contact-phone", "", "")
	cmd.Flags().String("billing-contact-email", "", "")
	cmd.Flags().String("address1", "", "")
	cmd.Flags().String("address2", "", "")
	cmd.Flags().String("city", "", "")
	cmd.Flags().String("state", "", "")
	cmd.Flags().String("postcode", "", "")
	cmd.Flags().String("country", "", "")
	cmd.Flags().String("po-number", "", "")
	cmd.Flags().String("tax-number", "", "")
	cmd.Flags().Int("first-party-id", 0, "")

	testutil.SetFlags(t, cmd, map[string]string{
		"currency":              "USD",
		"language":              "en",
		"billing-contact-name":  "John Doe",
		"billing-contact-phone": "+14155552671",
		"billing-contact-email": "john@example.com",
		"address1":              "123 Main St",
		"city":                  "New York",
		"state":                 "NY",
		"postcode":              "10001",
		"country":               "US",
		"first-party-id":        "1558",
	})

	err := SetBillingMarket(cmd, []string{}, true)
	assert.NoError(t, err)

	assert.NotNil(t, mockSvc.CapturedSetBillingMarketRequest)
	assert.Equal(t, "USD", mockSvc.CapturedSetBillingMarketRequest.CurrencyEnum)
	assert.Equal(t, "en", mockSvc.CapturedSetBillingMarketRequest.Language)
	assert.Equal(t, "John Doe", mockSvc.CapturedSetBillingMarketRequest.BillingContactName)
	assert.Equal(t, "+14155552671", mockSvc.CapturedSetBillingMarketRequest.BillingContactPhone)
	assert.Equal(t, "john@example.com", mockSvc.CapturedSetBillingMarketRequest.BillingContactEmail)
	assert.Equal(t, "123 Main St", mockSvc.CapturedSetBillingMarketRequest.Address1)
	assert.Equal(t, "New York", mockSvc.CapturedSetBillingMarketRequest.City)
	assert.Equal(t, "NY", mockSvc.CapturedSetBillingMarketRequest.State)
	assert.Equal(t, "10001", mockSvc.CapturedSetBillingMarketRequest.Postcode)
	assert.Equal(t, "US", mockSvc.CapturedSetBillingMarketRequest.Country)
	assert.Equal(t, 1558, mockSvc.CapturedSetBillingMarketRequest.FirstPartyID)
	assert.Nil(t, mockSvc.CapturedSetBillingMarketRequest.Address2)
}

func TestSetBillingMarketAction_WithOptionalFields(t *testing.T) {
	mockSvc := &MockBillingMarketService{}

	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.BillingMarketService = mockSvc
	})
	defer cleanup()

	cmd := &cobra.Command{Use: "set"}
	cmd.Flags().String("currency", "", "")
	cmd.Flags().String("language", "", "")
	cmd.Flags().String("billing-contact-name", "", "")
	cmd.Flags().String("billing-contact-phone", "", "")
	cmd.Flags().String("billing-contact-email", "", "")
	cmd.Flags().String("address1", "", "")
	cmd.Flags().String("address2", "", "")
	cmd.Flags().String("city", "", "")
	cmd.Flags().String("state", "", "")
	cmd.Flags().String("postcode", "", "")
	cmd.Flags().String("country", "", "")
	cmd.Flags().String("po-number", "", "")
	cmd.Flags().String("tax-number", "", "")
	cmd.Flags().Int("first-party-id", 0, "")

	testutil.SetFlags(t, cmd, map[string]string{
		"currency":              "AUD",
		"language":              "en",
		"billing-contact-name":  "Jane Smith",
		"billing-contact-phone": "+61400000000",
		"billing-contact-email": "jane@example.com",
		"address1":              "456 George St",
		"address2":              "Level 5",
		"city":                  "Sydney",
		"state":                 "NSW",
		"postcode":              "2000",
		"country":               "AU",
		"po-number":             "PO-12345",
		"tax-number":            "ABN-123456",
		"first-party-id":        "808",
	})

	err := SetBillingMarket(cmd, []string{}, true)
	assert.NoError(t, err)

	req := mockSvc.CapturedSetBillingMarketRequest
	assert.NotNil(t, req)
	assert.Equal(t, "AUD", req.CurrencyEnum)
	assert.Equal(t, 808, req.FirstPartyID)
	assert.NotNil(t, req.Address2)
	assert.Equal(t, "Level 5", *req.Address2)
	assert.Equal(t, "PO-12345", req.YourPONumber)
	assert.Equal(t, "ABN-123456", req.TaxNumber)
}

func TestBuildSetBillingMarketRequest_InvalidFirstPartyID(t *testing.T) {
	tests := []struct {
		name         string
		firstPartyID string
	}{
		{"zero", "0"},
		{"negative", "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "set"}
			cmd.Flags().String("currency", "", "")
			cmd.Flags().String("language", "", "")
			cmd.Flags().String("billing-contact-name", "", "")
			cmd.Flags().String("billing-contact-phone", "", "")
			cmd.Flags().String("billing-contact-email", "", "")
			cmd.Flags().String("address1", "", "")
			cmd.Flags().String("address2", "", "")
			cmd.Flags().String("city", "", "")
			cmd.Flags().String("state", "", "")
			cmd.Flags().String("postcode", "", "")
			cmd.Flags().String("country", "", "")
			cmd.Flags().String("po-number", "", "")
			cmd.Flags().String("tax-number", "", "")
			cmd.Flags().Int("first-party-id", 0, "")

			testutil.SetFlags(t, cmd, map[string]string{
				"currency":       "USD",
				"country":        "US",
				"first-party-id": tt.firstPartyID,
			})

			req, err := buildSetBillingMarketRequest(cmd)
			assert.Nil(t, req)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "first-party-id must be a positive integer")
		})
	}
}

func TestSetBillingMarketAction_Error(t *testing.T) {
	mockSvc := &MockBillingMarketService{
		SetBillingMarketError: assert.AnError,
	}

	cleanup := testutil.SetupLogin(func(c *megaport.Client) {
		c.BillingMarketService = mockSvc
	})
	defer cleanup()

	cmd := &cobra.Command{Use: "set"}
	cmd.Flags().String("currency", "", "")
	cmd.Flags().String("language", "", "")
	cmd.Flags().String("billing-contact-name", "", "")
	cmd.Flags().String("billing-contact-phone", "", "")
	cmd.Flags().String("billing-contact-email", "", "")
	cmd.Flags().String("address1", "", "")
	cmd.Flags().String("address2", "", "")
	cmd.Flags().String("city", "", "")
	cmd.Flags().String("state", "", "")
	cmd.Flags().String("postcode", "", "")
	cmd.Flags().String("country", "", "")
	cmd.Flags().String("po-number", "", "")
	cmd.Flags().String("tax-number", "", "")
	cmd.Flags().Int("first-party-id", 0, "")

	testutil.SetFlags(t, cmd, map[string]string{
		"currency":       "USD",
		"country":        "US",
		"first-party-id": "1558",
	})

	err := SetBillingMarket(cmd, []string{}, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set billing market")
}
