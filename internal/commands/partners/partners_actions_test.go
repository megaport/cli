package partners

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/commands/config"
	"github.com/megaport/megaport-cli/internal/testutil"
	"github.com/megaport/megaport-cli/internal/utils"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestFindPartners(t *testing.T) {
	cleanup := testutil.SetupLogin(func(c *megaport.Client) {})
	originalPrompt := utils.GetPrompt()
	origPrintPartnersFunc := printPartnersFunc

	defer func() {
		printPartnersFunc = origPrintPartnersFunc
		utils.SetPrompt(originalPrompt)
		cleanup()
	}()

	tests := []struct {
		name           string
		prompts        []string
		expectedError  string
		setupMock      func(*testing.T, *MockPartnerService)
		expectedCount  int
		expectedFormat string
	}{
		{
			name: "successful search with all filters",
			prompts: []string{
				"Test Product",
				"AWS",
				"Amazon",
				"123",
				"blue",
				"table",
			},
			expectedError: "",
			setupMock: func(t *testing.T, m *MockPartnerService) {
				m.listPartnersResponse = []*megaport.PartnerMegaport{
					{
						ProductName:   "Test Product",
						ConnectType:   "AWS",
						CompanyName:   "Amazon",
						LocationId:    123,
						DiversityZone: "blue",
					},
					{
						ProductName:   "Other Product",
						ConnectType:   "AZURE",
						CompanyName:   "Microsoft",
						LocationId:    456,
						DiversityZone: "red",
					},
				}
				m.listPartnersErr = nil
			},
			expectedCount: 1,
		},
		{
			name: "search with no filters",
			prompts: []string{
				"",
				"",
				"",
				"",
				"",
				"json",
			},
			expectedError: "",
			setupMock: func(t *testing.T, m *MockPartnerService) {
				m.listPartnersResponse = []*megaport.PartnerMegaport{
					{
						ProductName:   "Test Product",
						ConnectType:   "AWS",
						CompanyName:   "Amazon",
						LocationId:    123,
						DiversityZone: "blue",
					},
					{
						ProductName:   "Other Product",
						ConnectType:   "AZURE",
						CompanyName:   "Microsoft",
						LocationId:    456,
						DiversityZone: "red",
					},
				}
				m.listPartnersErr = nil
			},
			expectedCount: 2,
		},
		{
			name: "invalid location ID format",
			prompts: []string{
				"",
				"",
				"",
				"not-a-number",
				"",
				"table",
			},
			expectedError: "invalid location ID",
			setupMock: func(t *testing.T, m *MockPartnerService) {
				m.listPartnersResponse = []*megaport.PartnerMegaport{}
				m.listPartnersErr = nil
			},
			expectedCount: 0,
		},
		{
			name: "API error",
			prompts: []string{
				"",
				"",
			},
			expectedError: "failed to list partners",
			setupMock: func(t *testing.T, m *MockPartnerService) {
				m.listPartnersResponse = nil
				m.listPartnersErr = fmt.Errorf("API connection failure")
			},
			expectedCount: 0,
		},
		{
			name: "uppercase format is normalized",
			prompts: []string{
				"", "", "", "", "",
				"JSON",
			},
			expectedError: "",
			setupMock: func(t *testing.T, m *MockPartnerService) {
				m.listPartnersResponse = []*megaport.PartnerMegaport{
					{ProductName: "Test Product", ConnectType: "AWS", CompanyName: "Amazon", LocationId: 123, DiversityZone: "blue"},
				}
				m.listPartnersErr = nil
			},
			expectedCount:  1,
			expectedFormat: "json",
		},
		{
			name: "format with surrounding whitespace is normalized",
			prompts: []string{
				"", "", "", "", "",
				"  table  ",
			},
			expectedError: "",
			setupMock: func(t *testing.T, m *MockPartnerService) {
				m.listPartnersResponse = []*megaport.PartnerMegaport{
					{ProductName: "Test Product", ConnectType: "AWS", CompanyName: "Amazon", LocationId: 123, DiversityZone: "blue"},
				}
				m.listPartnersErr = nil
			},
			expectedCount:  1,
			expectedFormat: "table",
		},
		{
			name: "csv format is honored",
			prompts: []string{
				"", "", "", "", "",
				"csv",
			},
			expectedError: "",
			setupMock: func(t *testing.T, m *MockPartnerService) {
				m.listPartnersResponse = []*megaport.PartnerMegaport{
					{ProductName: "Test Product", ConnectType: "AWS", CompanyName: "Amazon", LocationId: 123, DiversityZone: "blue"},
				}
				m.listPartnersErr = nil
			},
			expectedCount:  1,
			expectedFormat: "csv",
		},
		{
			name: "unrecognized format falls back to table",
			prompts: []string{
				"", "", "", "", "",
				"yaml",
			},
			expectedError: "",
			setupMock: func(t *testing.T, m *MockPartnerService) {
				m.listPartnersResponse = []*megaport.PartnerMegaport{
					{ProductName: "Test Product", ConnectType: "AWS", CompanyName: "Amazon", LocationId: 123, DiversityZone: "blue"},
				}
				m.listPartnersErr = nil
			},
			expectedCount:  1,
			expectedFormat: "table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockPartnerService{}
			if tt.setupMock != nil {
				tt.setupMock(t, mockService)
			}

			config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
				if tt.expectedError == "failed to log in" {
					return nil, fmt.Errorf("login failure")
				}
				return &megaport.Client{
					PartnerService: mockService,
				}, nil
			})

			promptIndex := 0
			utils.SetPrompt(func(message string, noColor bool) (string, error) {
				if promptIndex >= len(tt.prompts) {
					return "", fmt.Errorf("unexpected additional prompt: %s", message)
				}
				response := tt.prompts[promptIndex]
				promptIndex++
				return response, nil
			})

			var capturedPartners []*megaport.PartnerMegaport
			var capturedFormat string
			printPartnersFunc = func(partners []*megaport.PartnerMegaport, format string, noColor bool) error {
				capturedPartners = partners
				capturedFormat = format
				return nil
			}

			cmd := &cobra.Command{}
			err := FindPartners(cmd, []string{}, false)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.prompts), promptIndex, "not all prompts were used")
				assert.Equal(t, tt.expectedCount, len(capturedPartners), "incorrect number of filtered partners")
				if tt.expectedFormat != "" {
					assert.Equal(t, tt.expectedFormat, capturedFormat, "output format not normalized as expected")
				}
			}
		})
	}
}

func TestListPartners(t *testing.T) {
	testPartners := []*megaport.PartnerMegaport{
		{
			ProductName:   "AWS Direct Connect",
			ConnectType:   "AWS",
			CompanyName:   "Amazon",
			LocationId:    123,
			DiversityZone: "blue",
		},
		{
			ProductName:   "Azure ExpressRoute",
			ConnectType:   "AZURE",
			CompanyName:   "Microsoft",
			LocationId:    456,
			DiversityZone: "red",
		},
		{
			ProductName:   "Google Cloud Interconnect",
			ConnectType:   "GOOGLE",
			CompanyName:   "Google",
			LocationId:    789,
			DiversityZone: "blue",
		},
	}

	tests := []struct {
		name          string
		partners      []*megaport.PartnerMegaport
		partnersErr   error
		loginErr      error
		flags         map[string]string
		expectedErr   string
		expectedCount int
		expectWarning bool
	}{
		{
			name:          "filter by product-name",
			partners:      testPartners,
			flags:         map[string]string{"product-name": "AWS Direct Connect"},
			expectedCount: 1,
		},
		{
			name:          "filter by connect-type",
			partners:      testPartners,
			flags:         map[string]string{"connect-type": "AZURE"},
			expectedCount: 1,
		},
		{
			name:          "empty result",
			partners:      testPartners,
			flags:         map[string]string{"product-name": "NonExistent"},
			expectedCount: 0,
			expectWarning: true,
		},
		{
			name:        "API error",
			partners:    nil,
			partnersErr: fmt.Errorf("API connection failure"),
			expectedErr: "failed to list partners",
		},
		{
			name:        "login error",
			loginErr:    fmt.Errorf("login failure"),
			expectedErr: "failed to log in",
		},
		{
			name:          "limit results",
			partners:      testPartners,
			flags:         map[string]string{"limit": "2"},
			expectedCount: 2,
		},
		{
			name:        "negative limit returns error",
			partners:    testPartners,
			flags:       map[string]string{"limit": "-1"},
			expectedErr: "--limit must be a non-negative integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockPartnerService{
				listPartnersResponse: tt.partners,
				listPartnersErr:      tt.partnersErr,
			}

			cleanup := testutil.SetupLogin(func(c *megaport.Client) {
				c.PartnerService = mockService
			})
			origPrintPartnersFunc := printPartnersFunc
			defer func() {
				printPartnersFunc = origPrintPartnersFunc
				cleanup()
			}()

			if tt.loginErr != nil {
				config.SetLoginFunc(func(ctx context.Context) (*megaport.Client, error) {
					return nil, tt.loginErr
				})
			}

			var capturedPartners []*megaport.PartnerMegaport
			printPartnersFunc = func(partners []*megaport.PartnerMegaport, format string, noColor bool) error {
				capturedPartners = partners
				return nil
			}

			cmd := testutil.NewCommand("list", nil)
			cmd.Flags().String("product-name", "", "")
			cmd.Flags().String("connect-type", "", "")
			cmd.Flags().String("company-name", "", "")
			cmd.Flags().Int("location-id", 0, "")
			cmd.Flags().String("diversity-zone", "", "")
			cmd.Flags().Int("limit", 0, "")

			testutil.SetFlags(t, cmd, tt.flags)

			var err error
			var capturedOutput string
			capturedStdout := output.CaptureOutput(func() {
				capturedOutput = captureStderr(t, func() {
					err = ListPartners(cmd, []string{}, true, "table")
				})
			})
			_ = capturedStdout

			if tt.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(capturedPartners))
				if tt.expectWarning {
					assert.Contains(t, capturedOutput, "No partner ports found")
				}
			}
		})
	}
}

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
