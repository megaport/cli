//go:build js && wasm

package output

import (
	"strings"
	"syscall/js"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/megaport/megaport-cli/internal/wasm"
)

// TestWasmTableWriter verifies the WASM table writer buffer
func TestWasmTableWriter(t *testing.T) {
	// Clear the buffer
	WasmTableWriter.Reset()

	// Write some data
	testData := "test table data"
	n, err := WasmTableWriter.Write([]byte(testData))

	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)
	assert.Equal(t, testData, WasmTableWriter.String())
}

// TestWasmTableWriter_Reset verifies buffer reset functionality
func TestWasmTableWriter_Reset(t *testing.T) {
	WasmTableWriter.WriteString("test data")
	assert.NotEqual(t, "", WasmTableWriter.String())

	WasmTableWriter.Reset()
	assert.Equal(t, "", WasmTableWriter.String())
}

// TestPrintTable_WASM verifies table printing in WASM environment
func TestPrintTable_WASM(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "Item 1", Active: true},
		{ID: 2, Name: "Item 2", Active: false},
	}

	// Clear the buffer and globals
	WasmTableWriter.Reset()
	js.Global().Delete("wasmTableOutput")

	// Print table
	err := printTable(data, false, currentPrintOptions())
	assert.NoError(t, err)

	// Verify buffer has content
	output := WasmTableWriter.String()
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "ACTIVE")
	assert.Contains(t, output, "Item 1")
	assert.Contains(t, output, "Item 2")

	// Verify global variable is set
	wasmTableGlobal := js.Global().Get("wasmTableOutput")
	assert.False(t, wasmTableGlobal.IsUndefined())
	assert.Equal(t, output, wasmTableGlobal.String())
}

// TestPrintTable_WASM_NoColor verifies table printing without colors
func TestPrintTable_WASM_NoColor(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "Test", Active: true},
	}

	WasmTableWriter.Reset()

	// Print with noColor=true
	err := printTable(data, true, currentPrintOptions())
	assert.NoError(t, err)

	output := WasmTableWriter.String()
	assert.NotEmpty(t, output)

	// Should not contain ANSI color codes (basic check)
	// Note: StyleLight may still have some formatting, but should be minimal
	assert.NotContains(t, output, "\033[38") // 38;5;x color codes
	assert.NotContains(t, output, "\033[48") // 48;5;x background codes
}

// TestPrintTable_WASM_WithColor verifies table printing with colors
func TestPrintTable_WASM_WithColor(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "Test", Active: true},
	}

	WasmTableWriter.Reset()

	// Print with noColor=false (colors enabled)
	err := printTable(data, false, currentPrintOptions())
	assert.NoError(t, err)

	output := WasmTableWriter.String()
	assert.NotEmpty(t, output)

	// Should contain ANSI color codes
	hasColorCodes := strings.Contains(output, "\033[") || strings.Contains(output, "\u001b[")
	assert.True(t, hasColorCodes, "Colored output should contain ANSI escape codes")
}

// TestPrintTable_WASM_EmptyData verifies handling of empty data
func TestPrintTable_WASM_EmptyData(t *testing.T) {
	var data []SimpleStruct

	WasmTableWriter.Reset()

	err := printTable(data, false, currentPrintOptions())
	assert.NoError(t, err)

	output := WasmTableWriter.String()
	// Empty data should still render headers
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "ACTIVE")
}

// TestPrintTable_WASM_ComplexData verifies complex structure handling
func TestPrintTable_WASM_ComplexData(t *testing.T) {
	data := []ComplexStruct{
		{
			ID:   1,
			Name: "Complex Item",
			Tags: []string{"tag1", "tag2"},
			Metadata: map[string]string{
				"key": "value",
			},
		},
	}

	WasmTableWriter.Reset()

	err := printTable(data, false, currentPrintOptions())
	assert.NoError(t, err)

	output := WasmTableWriter.String()
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "Complex Item")
}

// TestPrintTable_WASM_AccumulatesMultipleCalls verifies that a command
// rendering more than one table in a single invocation has all of them
// captured, not just the last (ESD-1650).
func TestPrintTable_WASM_AccumulatesMultipleCalls(t *testing.T) {
	first := []SimpleStruct{{ID: 1, Name: "First", Active: true}}
	second := []SimpleStruct{{ID: 2, Name: "Second", Active: false}}

	WasmTableWriter.Reset()
	js.Global().Delete("wasmTableOutput")

	assert.NoError(t, printTable(first, false, currentPrintOptions()))
	assert.NoError(t, printTable(second, false, currentPrintOptions()))

	output := WasmTableWriter.String()
	assert.Contains(t, output, "First", "first table must survive a second call")
	assert.Contains(t, output, "Second", "second table must also be present")

	global := js.Global().Get("wasmTableOutput")
	assert.Equal(t, output, global.String(), "global must reflect the full accumulated buffer")

	WasmTableWriter.Reset()
	js.Global().Delete("wasmTableOutput")
}

// TestPrintTable_WASM_AccumulatesWhenSanitizerDropsBytes guards the raw-offset
// alignment fixed in ESD-1650. The first table's cell carries injected control
// bytes the sanitizer strips, so the sanitized whole buffer is shorter than the
// raw byte offset recorded before the second call; slicing the sanitized string
// by that raw offset (the pre-fix behavior) reads past its end and panics. The
// global must still equal the sanitized full buffer and both rows must survive.
func TestPrintTable_WASM_AccumulatesWhenSanitizerDropsBytes(t *testing.T) {
	noisy := "acme" + strings.Repeat("\x1b[2K\x1b[H\x7f", 400)
	first := []SimpleStruct{{ID: 1, Name: noisy, Active: true}}
	second := []SimpleStruct{{ID: 2, Name: "Second", Active: false}}

	SetTerminalWidthForTesting(0)
	defer SetTerminalWidthForTesting(0)
	WasmTableWriter.Reset()
	js.Global().Delete("wasmTableOutput")

	assert.NotPanics(t, func() {
		assert.NoError(t, printTable(first, true, currentPrintOptions()))
		assert.NoError(t, printTable(second, true, currentPrintOptions()))
	})

	global := js.Global().Get("wasmTableOutput").String()
	assert.Equal(t, wasm.SanitizeTerminalOutput(WasmTableWriter.String()), global,
		"global must equal the sanitized full accumulated buffer")
	assert.NotContains(t, global, "\x1b[2K", "injected control bytes must be stripped")
	assert.Contains(t, global, "acme")
	assert.Contains(t, global, "Second")

	WasmTableWriter.Reset()
	js.Global().Delete("wasmTableOutput")
}

// TestCalculateDynamicWidth verifies column width calculation
func TestCalculateDynamicWidth(t *testing.T) {
	tests := []struct {
		name          string
		termWidth     int
		minWidth      int
		maxPercentage int
		expectedMin   int
	}{
		{
			name:          "normal width",
			termWidth:     100,
			minWidth:      10,
			maxPercentage: 50,
			expectedMin:   10,
		},
		{
			name:          "exceeds minimum",
			termWidth:     200,
			minWidth:      10,
			maxPercentage: 50,
			expectedMin:   10,
		},
		{
			name:          "below minimum",
			termWidth:     10,
			minWidth:      20,
			maxPercentage: 50,
			expectedMin:   20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateDynamicWidth(tt.termWidth, tt.minWidth, tt.maxPercentage)
			assert.GreaterOrEqual(t, result, tt.expectedMin)
		})
	}
}

// TestPrintTable_WASM_ColumnWidths verifies fixed column width behavior
func TestPrintTable_WASM_ColumnWidths(t *testing.T) {
	data := []struct {
		ID       int    `json:"id" header:"ID"`
		Name     string `json:"name" header:"Name"`
		Country  string `json:"country" header:"Country"`
		Metro    string `json:"metro" header:"Metro"`
		SiteCode string `json:"site_code" header:"Site Code"`
		Status   string `json:"status" header:"Status"`
	}{
		{
			ID:       1,
			Name:     "Very Long Location Name That Might Wrap",
			Country:  "United States",
			Metro:    "New York",
			SiteCode: "NYC01",
			Status:   "Active",
		},
	}

	SetTerminalWidthForTesting(0) // fallback: no host-provided width
	WasmTableWriter.Reset()

	err := printTable(data, false, currentPrintOptions())
	assert.NoError(t, err)

	output := WasmTableWriter.String()
	assert.NotEmpty(t, output)

	// Verify all columns are present
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "COUNTRY")
	assert.Contains(t, output, "METRO")
	assert.Contains(t, output, "SITE CODE")
	assert.Contains(t, output, "STATUS")
}

// wideTableRow is used by the terminal-width tests below; its UID and Name
// columns are long enough to be affected by both the fallback and dynamic
// width paths.
type wideTableRow struct {
	UID  string `json:"uid" header:"UID"`
	Name string `json:"name" header:"Name"`
}

func renderWideTable(t *testing.T) string {
	t.Helper()
	data := []wideTableRow{
		{
			UID:  "12345678-1234-1234-1234-123456789012",
			Name: "A very long resource name that would normally wrap",
		},
	}
	WasmTableWriter.Reset()
	err := printTable(data, false, currentPrintOptions())
	assert.NoError(t, err)
	return WasmTableWriter.String()
}

// TestPrintTable_WASM_TerminalWidth_Absent verifies that with no host width
// set, rendering falls back to the fixed per-header widths (unchanged
// behavior).
func TestPrintTable_WASM_TerminalWidth_Absent(t *testing.T) {
	SetTerminalWidthForTesting(0)
	defer SetTerminalWidthForTesting(0)

	output := renderWideTable(t)
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "UID")
	assert.Contains(t, output, "NAME")
}

// TestPrintTable_WASM_TerminalWidth_Present verifies that a known host width
// scales column widths using the same percentage logic as native, producing
// a visibly narrower table than the fixed fallback for a narrow viewport.
func TestPrintTable_WASM_TerminalWidth_Present(t *testing.T) {
	SetTerminalWidthForTesting(0)
	fallbackOutput := renderWideTable(t)
	fallbackLines := strings.Split(strings.TrimRight(fallbackOutput, "\n"), "\n")
	assert.NotEmpty(t, fallbackLines)

	SetTerminalWidthForTesting(40)
	defer SetTerminalWidthForTesting(0)
	narrowOutput := renderWideTable(t)
	narrowLines := strings.Split(strings.TrimRight(narrowOutput, "\n"), "\n")
	assert.NotEmpty(t, narrowLines)

	// The top border's length reflects the total rendered table width.
	assert.Less(t, len(narrowLines[0]), len(fallbackLines[0]),
		"a narrow host width should render a narrower table than the fixed fallback")
}

// TestPrintTable_WASM_TerminalWidth_Absurd verifies rendering doesn't error
// or panic for extreme (tiny/huge) host-provided widths.
func TestPrintTable_WASM_TerminalWidth_Absurd(t *testing.T) {
	defer SetTerminalWidthForTesting(0)

	for _, width := range []int{1, 100000} {
		SetTerminalWidthForTesting(width)
		assert.NotPanics(t, func() {
			output := renderWideTable(t)
			assert.NotEmpty(t, output)
			assert.Contains(t, output, "UID")
			assert.Contains(t, output, "NAME")
		})
	}
}

// TestPrintTable_WASM_BoxDrawing verifies box drawing characters
func TestPrintTable_WASM_BoxDrawing(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "Test", Active: true},
	}

	WasmTableWriter.Reset()

	err := printTable(data, false, currentPrintOptions())
	assert.NoError(t, err)

	output := WasmTableWriter.String()

	// Check for box drawing characters used in MegaportEnhancedStyle
	assert.Contains(t, output, "│") // Vertical border
	assert.Contains(t, output, "─") // Horizontal border

	// Check for corners (at least one of these should be present)
	hasCorners := strings.Contains(output, "┌") ||
		strings.Contains(output, "┐") ||
		strings.Contains(output, "└") ||
		strings.Contains(output, "┘")
	assert.True(t, hasCorners, "Table should have corner characters")
}

// TestPrintTable_WASM_HeaderFormatting verifies header formatting
func TestPrintTable_WASM_HeaderFormatting(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "test", Active: true},
	}

	WasmTableWriter.Reset()

	err := printTable(data, false, currentPrintOptions())
	assert.NoError(t, err)

	output := WasmTableWriter.String()

	// Headers should be uppercase (FormatUpper)
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "ACTIVE")

	// Should not contain lowercase headers
	lines := strings.Split(output, "\n")
	headerLine := ""
	for _, line := range lines {
		if strings.Contains(line, "NAME") {
			headerLine = line
			break
		}
	}
	assert.NotEmpty(t, headerLine)
}

// TestPrintTable_WASM_GlobalVariable verifies wasmTableOutput global
func TestPrintTable_WASM_GlobalVariable(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "Test", Active: true},
	}

	// Clear the global
	js.Global().Delete("wasmTableOutput")
	WasmTableWriter.Reset()

	// Print table
	err := printTable(data, false, currentPrintOptions())
	assert.NoError(t, err)

	// Verify global is set
	wasmTableGlobal := js.Global().Get("wasmTableOutput")
	assert.False(t, wasmTableGlobal.IsUndefined(), "wasmTableOutput should be set")
	assert.False(t, wasmTableGlobal.IsNull(), "wasmTableOutput should not be null")

	globalContent := wasmTableGlobal.String()
	bufferContent := WasmTableWriter.String()

	assert.Equal(t, bufferContent, globalContent, "Global and buffer content should match")
}

// TestPrintTable_WASM_SanitizesInjectedControlSequences verifies that a
// resource name carrying an injected CSI cursor-move sequence (the kind a
// partner/marketplace listing name could carry) reaches wasmTableOutput with
// the injected bytes stripped, while the CLI's own SGR color styling for the
// cell (via colorizeValue) survives.
func TestPrintTable_WASM_SanitizesInjectedControlSequences(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "acme-corp\x1b[2K\x1b[Hspoofed", Active: true},
	}

	js.Global().Delete("wasmTableOutput")
	WasmTableWriter.Reset()

	err := printTable(data, false, currentPrintOptions())
	assert.NoError(t, err)

	output := js.Global().Get("wasmTableOutput").String()
	assert.NotContains(t, output, "\x1b[2K", "injected erase-line sequence must not reach the host")
	assert.NotContains(t, output, "\x1b[H", "injected cursor-home sequence must not reach the host")
	assert.Contains(t, output, "acme-corp")
	assert.Contains(t, output, "spoofed")
	assert.Contains(t, output, "\x1b[", "the CLI's own SGR color styling for the cell must be preserved")
}

// TestPrintTable_WASM_ConsoleLogging verifies console logging (basic check)
func TestPrintTable_WASM_ConsoleLogging(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "Test", Active: true},
	}

	WasmTableWriter.Reset()

	// This should not panic or error even with console.log calls
	assert.NotPanics(t, func() {
		err := printTable(data, false, currentPrintOptions())
		assert.NoError(t, err)
	})
}

// TestPrintTable_WASM_MegaportStyle verifies Megaport-specific styling
func TestPrintTable_WASM_MegaportStyle(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "Test", Active: true},
		{ID: 2, Name: "Test2", Active: false},
	}

	WasmTableWriter.Reset()

	err := printTable(data, false, currentPrintOptions())
	assert.NoError(t, err)

	output := WasmTableWriter.String()

	// Should have padding (spaces around content)
	assert.Contains(t, output, "  ") // Multiple spaces indicate padding

	// Should have separators between columns
	assert.Contains(t, output, "│")

	// Should have header separator
	lines := strings.Split(output, "\n")
	hasSeparatorLine := false
	for _, line := range lines {
		if strings.Contains(line, "─") && strings.Contains(line, "┼") {
			hasSeparatorLine = true
			break
		}
	}
	assert.True(t, hasSeparatorLine, "Should have header separator line")
}
