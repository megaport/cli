//go:build !wasm

package output

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		panic(fmt.Sprintf("Failed to copy from pipe: %v", err))
	}
	return buf.String()
}

func TestPrintSuccess(t *testing.T) {
	tests := []struct {
		name             string
		format           string
		args             []interface{}
		noColor          bool
		expectedContains string
	}{
		{
			name:             "with color",
			format:           "Test %s message",
			args:             []interface{}{"success"},
			noColor:          false,
			expectedContains: "Test success message",
		},
		{
			name:             "without color",
			format:           "Test %s message",
			args:             []interface{}{"success"},
			noColor:          true,
			expectedContains: "✓ Test success message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStderr(t, func() {
				PrintSuccess(tt.format, tt.noColor, tt.args...)
			})
			assert.Contains(t, output, tt.expectedContains)
		})
	}
}

func TestFormatSuccess(t *testing.T) {
	origNoColor := color.NoColor
	defer func() { color.NoColor = origNoColor }()

	color.NoColor = false
	result := FormatSuccess("test", false)
	assert.NotEqual(t, "successfully", result)

	result = FormatSuccess("test", true)
	assert.Equal(t, "successfully", result)
}

func TestPrintResourceSuccess(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		action       string
		uid          string
		noColor      bool
		expected     string
	}{
		{
			name:         "action with -ed suffix",
			resourceType: "Port",
			action:       "created",
			uid:          "port-123",
			noColor:      true,
			expected:     "✓ Port created port-123\n",
		},
		{
			name:         "action without -ed suffix",
			resourceType: "Port",
			action:       "create",
			uid:          "port-123",
			noColor:      true,
			expected:     "✓ Port created port-123\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStderr(t, func() {
				PrintResourceSuccess(tt.resourceType, tt.action, tt.uid, tt.noColor)
			})
			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestPrintResourceCreated(t *testing.T) {
	output := captureStderr(t, func() {
		PrintResourceCreated("Port", "port-123", true)
	})
	assert.Equal(t, "✓ Port created port-123\n", output)
}

func TestPrintResourceUpdated(t *testing.T) {
	output := captureStderr(t, func() {
		PrintResourceUpdated("Port", "port-123", true)
	})
	assert.Equal(t, "✓ Port updated port-123\n", output)
}

func TestPrintResourceDeleted(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		uid          string
		immediate    bool
		noColor      bool
		expected     string
	}{
		{
			name:         "delete immediate",
			resourceType: "Port",
			uid:          "port-123",
			immediate:    true,
			noColor:      true,
			expected:     "✓ Port deleted port-123\nThe resource will be deleted immediately\n",
		},
		{
			name:         "delete at end of billing period",
			resourceType: "Port",
			uid:          "port-123",
			immediate:    false,
			noColor:      true,
			expected:     "✓ Port deleted port-123\nThe resource will be deleted at the end of the current billing period\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStderr(t, func() {
				PrintResourceDeleted(tt.resourceType, tt.uid, tt.immediate, tt.noColor)
			})
			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestSpinner(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing-sensitive spinner test")
	}
	orig := isTerminalCached.Load()
	t.Cleanup(func() { SetIsTerminal(orig) })
	SetIsTerminal(true)

	spinner := NewSpinner(true)
	assert.NotNil(t, spinner)
	assert.Equal(t, 100*time.Millisecond, spinner.frameRate)
	assert.True(t, spinner.noColor)

	var stdout string
	stderr := captureStderr(t, func() {
		stdout = captureOutput(func() {
			spinner.Start("Testing spinner")
			time.Sleep(500 * time.Millisecond)
			spinner.Stop()
		})
	})
	assert.NotEmpty(t, stderr, "spinner frames must be written to stderr")
	assert.Empty(t, stdout, "spinner frames must never be written to stdout")
}

func TestPrintResourceSpinners(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing-sensitive resource spinner test")
	}
	orig := isTerminalCached.Load()
	t.Cleanup(func() { SetIsTerminal(orig) })
	SetIsTerminal(true)

	tests := []struct {
		name         string
		function     func(string, string, bool) *Spinner
		resourceType string
		uid          string
		noColor      bool
		expected     string
	}{
		{
			name:         "creating spinner",
			function:     PrintResourceCreating,
			resourceType: "Port",
			uid:          "port-123",
			noColor:      true,
			expected:     "Creating Port port-123...",
		},
		{
			name:         "updating spinner",
			function:     PrintResourceUpdating,
			resourceType: "Port",
			uid:          "port-123",
			noColor:      true,
			expected:     "Updating Port port-123...",
		},
		{
			name:         "deleting spinner",
			function:     PrintResourceDeleting,
			resourceType: "Port",
			uid:          "port-123",
			noColor:      true,
			expected:     "Deleting Port port-123...",
		},
		{
			name:         "getting spinner",
			function:     PrintResourceGetting,
			resourceType: "Port",
			uid:          "port-123",
			noColor:      true,
			expected:     "Getting Port port-123 details...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStderr(t, func() {
				spinner := tt.function(tt.resourceType, tt.uid, tt.noColor)
				time.Sleep(200 * time.Millisecond)
				spinner.Stop()
			})
			assert.Contains(t, output, tt.expected)
		})
	}
}

func TestPrintResourceListing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing-sensitive resource listing test")
	}
	orig := isTerminalCached.Load()
	t.Cleanup(func() { SetIsTerminal(orig) })
	SetIsTerminal(true)

	output := captureStderr(t, func() {
		spinner := PrintResourceListing("Port", true)
		time.Sleep(200 * time.Millisecond)
		spinner.Stop()
	})
	assert.Contains(t, output, "Listing Ports...")
}

func TestPrintError(t *testing.T) {
	output := captureStderr(t, func() {
		PrintError("Test %s message", true, "error")
	})
	assert.Equal(t, "✗ Test error message\n", output)

	output = captureStderr(t, func() {
		PrintError("Test %s message", false, "error")
	})
	assert.Contains(t, output, "Test error message")
}

func TestPrintWarning(t *testing.T) {
	output := captureStderr(t, func() {
		PrintWarning("Test %s message", true, "warning")
	})
	assert.Equal(t, "⚠ Test warning message\n", output)

	output = captureStderr(t, func() {
		PrintWarning("Test %s message", false, "warning")
	})
	assert.Contains(t, output, "Test warning message")
}

func TestPrintInfo(t *testing.T) {
	output := captureStderr(t, func() {
		PrintInfo("Test %s message", true, "info")
	})
	assert.Equal(t, "ℹ Test info message\n", output)

	output = captureStderr(t, func() {
		PrintInfo("Test %s message", false, "info")
	})
	assert.Contains(t, output, "Test info message")
}

func TestFormatConfirmation(t *testing.T) {
	result := FormatConfirmation("Delete this resource?", true)
	assert.Equal(t, "Delete this resource? [y/N]", result)

	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	result = FormatConfirmation("Delete this resource?", false)
	assert.Contains(t, result, "Delete this resource?")
	assert.Contains(t, result, "[y/N]")
	assert.NotEqual(t, "Delete this resource? [y/N]", result)
}

func TestFormatPrompt(t *testing.T) {
	result := FormatPrompt("Enter name:", true)
	assert.Equal(t, "Enter name:", result)

	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	result = FormatPrompt("Enter name:", false)
	assert.NotEqual(t, "Enter name:", result)
	assert.Contains(t, StripANSIColors(result), "Enter name:")
}

func TestFormatExample(t *testing.T) {
	result := FormatExample("megaport-cli port list", true)
	assert.Equal(t, "megaport-cli port list", result)

	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	result = FormatExample("megaport-cli port list", false)
	assert.NotEqual(t, "megaport-cli port list", result)
	assert.Contains(t, StripANSIColors(result), "megaport-cli port list")
}

func TestFormatCommandName(t *testing.T) {
	result := FormatCommandName("port list", true)
	assert.Equal(t, "port list", result)

	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	result = FormatCommandName("port list", false)
	assert.NotEqual(t, "port list", result)
	assert.Contains(t, StripANSIColors(result), "port list")
}

func TestFormatRequiredFlag(t *testing.T) {
	result := FormatRequiredFlag("--name", "Name of the resource", true)
	assert.Equal(t, "--name (REQUIRED): Name of the resource", result)

	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	result = FormatRequiredFlag("--name", "Name of the resource", false)
	assert.NotEqual(t, "--name (REQUIRED): Name of the resource", result)
	assert.Contains(t, StripANSIColors(result), "--name (REQUIRED): Name of the resource")
}

func TestFormatOptionalFlag(t *testing.T) {
	result := FormatOptionalFlag("--location-id", "ID of the location", true)
	assert.Equal(t, "--location-id: ID of the location", result)

	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	result = FormatOptionalFlag("--location-id", "ID of the location", false)
	assert.NotEqual(t, "--location-id: ID of the location", result)
	assert.Contains(t, StripANSIColors(result), "--location-id: ID of the location")
}

func TestFormatJSONExample(t *testing.T) {
	json := `{"name": "test"}`
	result := FormatJSONExample(json, true)
	assert.Equal(t, json, result)

	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	result = FormatJSONExample(json, false)
	assert.NotEqual(t, json, result)
	assert.Contains(t, StripANSIColors(result), json)
}

func TestFormatUID(t *testing.T) {
	result := FormatUID("port-123", true)
	assert.Equal(t, "port-123", result)

	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	result = FormatUID("port-123", false)
	assert.NotEqual(t, "port-123", result)
	assert.Contains(t, StripANSIColors(result), "port-123")
}

func TestStripANSIColors(t *testing.T) {
	coloredString := "\x1b[31mRed\x1b[0m \x1b[32mGreen\x1b[0m"
	result := StripANSIColors(coloredString)
	assert.Equal(t, "Red Green", result)
}

func TestFormatOldValue(t *testing.T) {
	result := FormatOldValue("old-value", true)
	assert.Equal(t, "old-value", result)

	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	result = FormatOldValue("old-value", false)
	assert.NotEqual(t, "old-value", result)
	assert.Contains(t, StripANSIColors(result), "old-value")
}

func TestFormatNewValue(t *testing.T) {
	result := FormatNewValue("new-value", true)
	assert.Equal(t, "new-value", result)

	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	result = FormatNewValue("new-value", false)
	assert.NotEqual(t, "new-value", result)
	assert.Contains(t, StripANSIColors(result), "new-value")
}

// TestOutputFormatConcurrency verifies that concurrent reads and writes to the
// output config via GetOutputFormat, SetVerbosity, and the Print* helpers do
// not cause data races. Unlike tests that only vary a single field, this one
// has odd-numbered goroutines write Format while even-numbered goroutines write
// Verbosity — exercising concurrent writes to different fields simultaneously.
// Run with: go test -race ./internal/base/output/...
func TestOutputFormatConcurrency(t *testing.T) {
	const goroutines = 10
	const iterations = 50

	orig := GetOutputConfig()
	t.Cleanup(func() { ApplyOutputConfig(orig) })

	// Seed both fields that goroutines will cycle through so assertions
	// cannot observe a value from a previous test.
	SetOutputFormat("table")
	SetVerbosity("normal")

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if id%2 == 0 {
					// Even goroutines write Format.
					if j%2 == 0 {
						SetOutputFormat("json")
					} else {
						SetOutputFormat("table")
					}
				} else {
					// Odd goroutines write Verbosity — different field, same mutex.
					if j%2 == 0 {
						SetVerbosity("quiet")
					} else {
						SetVerbosity("normal")
					}
				}

				// All goroutines read back through the struct — any lost write
				// or torn read will be caught by the race detector.
				cfg := GetOutputConfig()
				assert.Contains(t, []string{"json", "table"}, cfg.Format)
				assert.Contains(t, []string{"quiet", "normal"}, cfg.Verbosity)

				// Exercise Print* helpers that read the config.
				PrintSuccess("concurrent %d", true, id)
				PrintError("concurrent %d", true, id)
				PrintWarning("concurrent %d", true, id)
				PrintInfo("concurrent %d", true, id)

				// Exercise spinner creation which also reads the config.
				s := PrintResourceListing("Port", true)
				s.Stop()
			}
		}(i)
	}

	wg.Wait()
}

func TestPrintResourceProvisioning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing-sensitive provisioning test")
	}
	orig := isTerminalCached.Load()
	t.Cleanup(func() { SetIsTerminal(orig) })
	SetIsTerminal(true)

	t.Run("shows provisioning message with elapsed time", func(t *testing.T) {
		output := captureStderr(t, func() {
			spinner := PrintResourceProvisioning("Port", "port-123", true)
			time.Sleep(200 * time.Millisecond)
			spinner.Stop()
		})
		assert.Contains(t, output, "Provisioning Port port-123...")
		assert.Contains(t, output, "elapsed")
	})

	t.Run("quiet mode returns no-op spinner", func(t *testing.T) {
		SetVerbosity("quiet")
		t.Cleanup(func() { ResetState() })
		spinner := PrintResourceProvisioning("Port", "port-123", true)
		assert.NotNil(t, spinner)
		assert.True(t, spinner.stopped)
	})
}

func TestStartWithElapsed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing-sensitive elapsed timer test")
	}
	orig := isTerminalCached.Load()
	t.Cleanup(func() { SetIsTerminal(orig) })
	SetIsTerminal(true)

	t.Run("appends elapsed time to message", func(t *testing.T) {
		spinner := NewSpinner(true)
		output := captureStderr(t, func() {
			spinner.StartWithElapsed("Provisioning Port...")
			time.Sleep(1100 * time.Millisecond)
			spinner.Stop()
		})
		assert.Contains(t, output, "Provisioning Port...")
		assert.Contains(t, output, "elapsed")
	})

	t.Run("stop does not panic", func(t *testing.T) {
		spinner := NewSpinner(true)
		spinner.StartWithElapsed("Provisioning...")
		time.Sleep(50 * time.Millisecond)
		assert.NotPanics(t, spinner.Stop)
	})

	t.Run("already stopped spinner is a no-op", func(t *testing.T) {
		spinner := NewSpinner(true)
		spinner.stopped = true
		assert.NotPanics(t, func() { spinner.StartWithElapsed("test") })
	})

	t.Run("wasm style uses wasm chars", func(t *testing.T) {
		spinner := NewSpinner(true)
		spinner.style = "wasm"
		output := captureStderr(t, func() {
			spinner.StartWithElapsed("Provisioning...")
			time.Sleep(200 * time.Millisecond)
			spinner.Stop()
		})
		assert.Contains(t, output, "elapsed")
	})

	t.Run("non-table format writes a single plain line to stderr", func(t *testing.T) {
		spinner := NewSpinnerWithOutput(true, "json")
		// Capture stderr by redirecting os.Stderr
		r, w, _ := os.Pipe()
		oldStderr := os.Stderr
		os.Stderr = w
		spinner.StartWithElapsed("Provisioning...")
		time.Sleep(200 * time.Millisecond)
		spinner.Stop()
		w.Close()
		os.Stderr = oldStderr
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		out := buf.String()
		assert.Contains(t, out, "Provisioning...")
		assert.NotContains(t, out, "elapsed", "non-interactive sink must not animate the elapsed counter")
		assert.NotContains(t, out, "\r", "non-interactive sink must not emit carriage returns")
		assert.NotContains(t, out, "\x1b[K", "non-interactive sink must not emit clear-line escapes")
	})
}

func TestSpinnerStopWithSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing-sensitive spinner stop test")
	}
	orig := isTerminalCached.Load()
	t.Cleanup(func() { SetIsTerminal(orig) })
	SetIsTerminal(true)

	output := captureStderr(t, func() {
		spinner := NewSpinner(true)
		spinner.Start("Testing")
		time.Sleep(200 * time.Millisecond)
		spinner.StopWithSuccess("Operation completed")
	})
	assert.Contains(t, output, "✓ Operation completed")
}

// fakeSpinnerInterface is a native-buildable SpinnerInterface stand-in used to
// verify Spinner.StopWithSuccess delegates to it, without depending on the
// real (WASM-only) WasmSpinner implementation.
type fakeSpinnerInterface struct {
	stopCalls            int
	stopWithSuccessCalls []string
}

func (f *fakeSpinnerInterface) Start(string) {}

func (f *fakeSpinnerInterface) Stop() {
	f.stopCalls++
}

func (f *fakeSpinnerInterface) StopWithSuccess(msg string) {
	f.stopWithSuccessCalls = append(f.stopWithSuccessCalls, msg)
}

func TestSpinnerStopWithSuccessDelegatesToWasmSpinner(t *testing.T) {
	t.Cleanup(func() { ResetState() })
	SetVerbosity("normal")
	fake := &fakeSpinnerInterface{}
	spinner := &Spinner{stop: make(chan bool, 1), wasmSpinner: fake}

	spinner.StopWithSuccess("Operation completed")

	assert.Equal(t, 1, fake.stopCalls)
	assert.Equal(t, []string{"Operation completed"}, fake.stopWithSuccessCalls)
}

func TestSpinnerStopWithSuccessQuietSkipsWasmDelegate(t *testing.T) {
	t.Cleanup(func() { ResetState() })
	SetVerbosity("quiet")
	fake := &fakeSpinnerInterface{}
	spinner := &Spinner{stop: make(chan bool, 1), wasmSpinner: fake}

	spinner.StopWithSuccess("Operation completed")

	assert.Equal(t, 1, fake.stopCalls)
	assert.Empty(t, fake.stopWithSuccessCalls)
}

// TestSpinnerStopWithSuccessMachineFormatSkipsWasmDelegate verifies that
// machine-readable output formats (json/csv/xml) skip WASM delegation and
// fall through to the stderr branch, so a success line can never become the
// entire captured response for a command that otherwise reports no output.
func TestSpinnerStopWithSuccessMachineFormatSkipsWasmDelegate(t *testing.T) {
	t.Cleanup(func() { ResetState() })
	SetVerbosity("normal")
	fake := &fakeSpinnerInterface{}
	spinner := &Spinner{stop: make(chan bool, 1), wasmSpinner: fake, outputFormat: "json"}

	output := captureStderr(t, func() {
		spinner.StopWithSuccess("Operation completed")
	})

	assert.Equal(t, 1, fake.stopCalls)
	assert.Empty(t, fake.stopWithSuccessCalls, "machine-readable formats must not delegate to the WASM spinner")
	assert.Contains(t, output, "Operation completed")
}

func TestShouldSuppressSpinner(t *testing.T) {
	t.Cleanup(func() { ResetState() })

	tests := []struct {
		name     string
		format   string
		quiet    bool
		expected bool
	}{
		{"table format", "table", false, false},
		{"empty format treated like table for spinner suppression", "", false, false},
		{"json format suppressed", "json", false, true},
		{"csv format suppressed", "csv", false, true},
		{"xml format suppressed", "xml", false, true},
		{"quiet mode suppressed", "table", true, true},
		{"quiet mode with csv", "csv", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetOutputFormat(tt.format)
			if tt.quiet {
				SetVerbosity("quiet")
			} else {
				SetVerbosity("normal")
			}
			assert.Equal(t, tt.expected, shouldSuppressSpinner())
		})
	}
}

func TestShouldSuppressSpinnerForFormat(t *testing.T) {
	// Ensure normal verbosity so IsQuiet() doesn't interfere.
	t.Cleanup(func() { ResetState() })
	SetVerbosity("normal")

	assert.False(t, shouldSuppressSpinnerForFormat("table"))
	assert.False(t, shouldSuppressSpinnerForFormat(""))
	assert.True(t, shouldSuppressSpinnerForFormat("json"))
	assert.True(t, shouldSuppressSpinnerForFormat("csv"))
	assert.True(t, shouldSuppressSpinnerForFormat("xml"))
}

// saveOutputFormat captures the current output format and registers a t.Cleanup
// to restore it, avoiding hard-coded assumptions about initial state.
func saveOutputFormat(t *testing.T) {
	t.Helper()
	orig := GetOutputFormat()
	t.Cleanup(func() { SetOutputFormat(orig) })
}

func TestSpinnerNoOpForCSV(t *testing.T) {
	saveOutputFormat(t)
	SetOutputFormat("csv")

	spinner := PrintResourceListing("test", true)
	// A no-op spinner is already stopped at creation
	assert.True(t, spinner.stopped, "spinner should be no-op (stopped) for csv format")
}

func TestSpinnerNoOpForXML(t *testing.T) {
	saveOutputFormat(t)
	SetOutputFormat("xml")

	spinner := PrintResourceGetting("test", "uid-123", true)
	assert.True(t, spinner.stopped, "spinner should be no-op (stopped) for xml format")
}

func TestSpinnerNoOpForJSON(t *testing.T) {
	saveOutputFormat(t)
	SetOutputFormat("json")

	spinner := PrintResourceListing("test", true)
	assert.True(t, spinner.stopped, "spinner should be no-op (stopped) for json format")
}

func TestSpinnerActiveForTable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping spinner test in short mode")
	}
	saveOutputFormat(t)
	SetOutputFormat("table")
	orig := isTerminalCached.Load()
	t.Cleanup(func() { SetIsTerminal(orig) })
	SetIsTerminal(true)

	spinner := PrintResourceListing("test", true)
	assert.False(t, spinner.stopped, "spinner should be active for table format")
	spinner.Stop()
}

func TestAllSpinnerFunctionsSuppressedForCSV(t *testing.T) {
	saveOutputFormat(t)
	SetOutputFormat("csv")

	spinners := []*Spinner{
		PrintResourceCreating("test", "uid", true),
		PrintResourceProvisioning("test", "uid", true),
		PrintResourceUpdating("test", "uid", true),
		PrintResourceDeleting("test", "uid", true),
		PrintResourceListing("test", true),
		PrintResourceGetting("test", "uid", true),
		PrintResourceGettingWithOutput("test", "uid", true, "csv"),
		PrintListingResourceTags("test", "uid", true),
		PrintResourceValidating("test", true),
		PrintCustomSpinner("testing", "uid", true),
	}

	for i, s := range spinners {
		assert.True(t, s.stopped, "spinner %d should be no-op for csv format", i)
	}
}

// TestSpinnersNotSuppressedForTable covers the non-suppressed code paths in
// PrintListingResourceTags, PrintResourceValidating, and PrintCustomSpinner —
// specifically the GetOutputFormat() call and spinner creation that are only
// reached when the output format is "table" and a terminal is active.
func TestSpinnersNotSuppressedForTable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping spinner test in short mode")
	}
	t.Cleanup(func() { ResetState() })
	SetOutputFormat("table")
	orig := isTerminalCached.Load()
	t.Cleanup(func() { SetIsTerminal(orig) })
	SetIsTerminal(true)

	s1 := PrintListingResourceTags("Port", "uid-1", true)
	assert.False(t, s1.stopped, "PrintListingResourceTags should not be suppressed for table format")
	s1.Stop()

	s2 := PrintResourceValidating("VXC", true)
	assert.False(t, s2.stopped, "PrintResourceValidating should not be suppressed for table format")
	s2.Stop()

	s3 := PrintCustomSpinner("Syncing", "uid-2", true)
	assert.False(t, s3.stopped, "PrintCustomSpinner should not be suppressed for table format")
	s3.Stop()
}

// TestPrintResourceGettingWithOutput_EmptyFormatFallback covers the
// outputFormat = GetOutputFormat() fallback when an empty string is passed.
func TestPrintResourceGettingWithOutput_EmptyFormatFallback(t *testing.T) {
	t.Cleanup(func() { ResetState() })
	SetOutputFormat("csv") // suppresses spinner so the test is fast

	s := PrintResourceGettingWithOutput("Port", "uid-1", true, "")
	assert.True(t, s.stopped, "spinner should be suppressed when format falls back to csv")
	assert.Equal(t, "csv", s.outputFormat)
}

// TestPrintLoggingInWithOutput_EmptyFormatFallback covers the
// outputFormat = GetOutputFormat() fallback when an empty string is passed.
func TestPrintLoggingInWithOutput_EmptyFormatFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping spinner test in short mode")
	}
	t.Cleanup(func() { ResetState() })
	SetOutputFormat("table")
	orig := isTerminalCached.Load()
	t.Cleanup(func() { SetIsTerminal(orig) })
	SetIsTerminal(true)

	s := PrintLoggingInWithOutput(true, "")
	assert.False(t, s.stopped, "login spinner should not be suppressed when format falls back to table")
	s.Stop()
}

func TestLoginSpinnersNotSuppressedForCSV(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping spinner test in short mode")
	}
	saveOutputFormat(t)
	SetOutputFormat("csv")
	orig := isTerminalCached.Load()
	t.Cleanup(func() { SetIsTerminal(orig) })
	SetIsTerminal(true)

	// Login spinners should still show regardless of output format
	spinner := PrintLoggingIn(true)
	assert.False(t, spinner.stopped, "login spinner should not be suppressed for csv format")
	spinner.Stop()

	spinner2 := PrintLoggingInWithOutput(true, "csv")
	assert.False(t, spinner2.stopped, "login spinner with output should not be suppressed")
	spinner2.Stop()
}

func TestStopWithSuccessDoesNotWriteToStdoutForCSV(t *testing.T) {
	saveOutputFormat(t)
	SetOutputFormat("csv")
	orig := isTerminalCached.Load()
	t.Cleanup(func() { SetIsTerminal(orig) })
	SetIsTerminal(true)

	spinner := PrintResourceListing("test", true)

	// Suppress stderr so StopWithSuccess doesn't leak into test output.
	oldStderr := os.Stderr
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	require.NoError(t, err)
	defer devNull.Close()
	os.Stderr = devNull
	defer func() { os.Stderr = oldStderr }()

	stdoutOutput := captureOutput(func() {
		spinner.StopWithSuccess("done")
	})
	assert.Empty(t, stdoutOutput, "StopWithSuccess should not write to stdout for csv format")
}

func TestStopWithSuccessDoesNotWriteToStdoutForXML(t *testing.T) {
	saveOutputFormat(t)
	SetOutputFormat("xml")
	orig := isTerminalCached.Load()
	t.Cleanup(func() { SetIsTerminal(orig) })
	SetIsTerminal(true)

	spinner := PrintResourceGetting("test", "uid", true)

	oldStderr := os.Stderr
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	require.NoError(t, err)
	defer devNull.Close()
	os.Stderr = devNull
	defer func() { os.Stderr = oldStderr }()

	stdoutOutput := captureOutput(func() {
		spinner.StopWithSuccess("done")
	})
	assert.Empty(t, stdoutOutput, "StopWithSuccess should not write to stdout for xml format")
}

func TestNoOpSpinnerCarriesOutputFormat(t *testing.T) {
	saveOutputFormat(t)
	SetOutputFormat("csv")

	spinner := PrintResourceListing("test", true)
	assert.Equal(t, "csv", spinner.outputFormat, "no-op spinner should carry the output format")
}
