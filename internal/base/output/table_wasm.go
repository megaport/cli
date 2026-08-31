//go:build js && wasm

package output

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync/atomic"
	"syscall/js"

	prettytable "github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/megaport/megaport-cli/internal/wasm"
)

// isTerminalCached is always false in WASM (no real terminal).
// Uses atomic.Bool for goroutine safety.
var isTerminalCached atomic.Bool

// IsTerminal returns true if stdout is connected to a terminal. Always false in WASM.
func IsTerminal() bool {
	return isTerminalCached.Load()
}

// SetIsTerminal overrides the cached TTY detection result. Intended for tests.
func SetIsTerminal(val bool) {
	isTerminalCached.Store(val)
}

// SetTerminalWidthForTesting pins the terminal width used by table rendering.
// Pass 0 to clear the host-provided width and use the fixed per-header
// fallback layout. Intended for tests only.
func SetTerminalWidthForTesting(width int) {
	wasm.SetTerminalWidth(width)
}

// WasmTableWriter is a global buffer for capturing table output in WASM
var WasmTableWriter = &bytes.Buffer{}

// calculateDynamicWidth calculates the maximum width for a column
func calculateDynamicWidth(termWidth int, minWidth, maxPercentage int) int {
	maxWidth := termWidth * maxPercentage / 100
	if maxWidth < minWidth {
		maxWidth = minWidth
	}
	return maxWidth
}

// printTable is the WASM-specific implementation that properly captures table
// output. In WASM, render into the global WasmTableWriter so the rendered
// table is available to the JS-accessible wasmTableOutput global that the web
// UI reads. An empty slice still renders a header-only table (native parity);
// any "No X found" warning is surfaced above it by GetCapturedOutput, which
// prepends the direct status buffer to the table output.
//
// A command may call this more than once; each call appends to
// WasmTableWriter rather than overwriting it, so the global reflects every
// table emitted so far. The buffer is cleared between WASM invocations by
// resetWasmStructuredBuffers, so callers must not invoke this per data item
// in a loop; it accumulates without bound within an invocation.
func printTable[T OutputFields](data []T, noColor bool, opts printOptions) (err error) {
	wasmBufMu.Lock()
	defer wasmBufMu.Unlock()

	before := WasmTableWriter.Len()
	defer func() {
		if err != nil {
			WasmTableWriter.Truncate(before)
		}
	}()

	if err := printTableToWriter(WasmTableWriter, data, noColor, opts); err != nil {
		return err
	}

	// Sanitize before this leaves Go: wasmTableOutput is read back by
	// GetCapturedOutput/GetCompletionOutput and written to xterm without
	// escaping, and a colorized cell value (colorizeValue) can carry a
	// resource name or other API field an attacker controls. SGR color codes
	// are preserved; see wasm.SanitizeTerminalOutput. Sanitize the new slice
	// on its own rather than slicing the sanitized whole buffer by a raw-byte
	// offset: sanitizing drops bytes, so a raw offset no longer lines up with
	// the sanitized string.
	raw := WasmTableWriter.String()

	// Write the newly-rendered table to stdout so it can be captured by wasm buffers.
	fmt.Print(wasm.SanitizeTerminalOutput(raw[before:]))

	// Also write to a JavaScript-accessible global variable.
	js.Global().Set("wasmTableOutput", wasm.SanitizeTerminalOutput(raw))

	return nil
}

// printTableToWriter renders data as a table directly to w with no global side
// effects. printTable wraps it for the JS-global capture path; the status
// dashboard calls it (via PrintTableToWriter) to compose several tables into one
// writer without the per-call global being overwritten.
func printTableToWriter[T OutputFields](w io.Writer, data []T, noColor bool, opts printOptions) error {
	headers, jsonNames, fieldIndices, err := getStructTypeInfo(data)
	if err != nil {
		return err
	}
	if wasmFields := opts.fields; len(wasmFields) > 0 {
		headers, _, fieldIndices, err = filterByFields(headers, jsonNames, fieldIndices, wasmFields)
		if err != nil {
			return err
		}
	}
	if len(headers) == 0 {
		return nil
	}

	// Create table writer
	t := prettytable.NewWriter()
	t.SetOutputMirror(w)

	// WASM-specific table configuration with improved column widths
	// This ensures consistent, readable column distribution in the browser
	columnConfigs := make([]prettytable.ColumnConfig, len(headers))
	termWidth := wasm.TerminalWidth()
	for i, header := range headers {
		if termWidth > 0 {
			// Host has told us the real viewport width (xterm.js cols): scale
			// widths from it the same way the native renderer does.
			var widthMax int
			if i == 0 {
				widthMax = calculateDynamicWidth(termWidth, 10, 15)
			} else {
				widthMax = calculateDynamicWidth(termWidth, 15, 25)
			}
			columnConfigs[i] = prettytable.ColumnConfig{
				Number:    i + 1,
				WidthMax:  widthMax,
				AutoMerge: false,
			}
			continue
		}

		headerLower := strings.ToLower(header)
		var widthMax int

		// No known viewport width: fall back to fixed widths per column type
		// optimized for web display.
		switch headerLower {
		case "uid", "id":
			widthMax = 38 // Full UUID width for better readability
		case "name", "title":
			widthMax = 30 // Reasonable width for port names
		case "locationid", "location_id", "location id":
			widthMax = 12 // Numeric ID
		case "speed", "port_speed", "port speed":
			widthMax = 10 // Numeric speed
		case "status", "provisioning_status", "provisioning status", "state":
			widthMax = 15 // Status strings
		case "country":
			widthMax = 16
		case "metro", "city":
			widthMax = 16
		case "site code", "code":
			widthMax = 12
		case "type":
			widthMax = 12
		default:
			widthMax = 20
		}

		columnConfigs[i] = prettytable.ColumnConfig{
			Number:    i + 1,
			WidthMax:  widthMax,
			WidthMin:  8, // Minimum width for readability
			AutoMerge: false,
		}
	}
	t.SetColumnConfigs(columnConfigs)

	// WASM with xterm.js: Enable colors since xterm.js supports ANSI codes
	// xterm.js properly renders ANSI color codes and box-drawing characters
	// Keep the noColor parameter from the command flag, don't force it
	// noColor = true  // REMOVED: xterm.js supports colors!

	if noColor {
		// Clean, readable style without colors for WASM
		t.SetStyle(prettytable.StyleLight)
	} else {
		// Enhanced style optimized for web terminal with dark background
		megaportStyle := prettytable.Style{
			Name: "MegaportWebStyle",
			Box: prettytable.BoxStyle{
				BottomLeft:       "└",
				BottomRight:      "┘",
				BottomSeparator:  "┴",
				Left:             "│",
				LeftSeparator:    "├",
				MiddleHorizontal: "─",
				MiddleSeparator:  "┼",
				MiddleVertical:   "│",
				PaddingLeft:      " ", // Single space for compact display
				PaddingRight:     " ",
				Right:            "│",
				RightSeparator:   "┤",
				TopLeft:          "┌",
				TopRight:         "┐",
				TopSeparator:     "┬",
				UnfinishedRow:    " ≡",
			},
			Color: prettytable.ColorOptions{
				// Bright, readable colors for dark terminal background
				Header:       text.Colors{text.FgHiCyan, text.Bold}, // Bright cyan text, no background
				Row:          text.Colors{text.FgHiWhite},           // Bright white for rows
				RowAlternate: text.Colors{text.FgCyan},              // Cyan for alternating rows
				Footer:       text.Colors{text.FgHiCyan, text.Bold}, // Bright cyan footer
				Border:       text.Colors{text.FgBlue},              // Blue borders for subtle frame
			},
			Format: prettytable.FormatOptions{
				Footer: text.FormatDefault,
				Header: text.FormatUpper, // Uppercase headers for clarity
				Row:    text.FormatDefault,
			},
			Options: prettytable.Options{
				DrawBorder:      true,
				SeparateColumns: true,
				SeparateFooter:  true,
				SeparateHeader:  true,
				SeparateRows:    false, // No row separation for compact display
			},
		}
		t.SetStyle(megaportStyle)
	}

	t.Style().Options.DrawBorder = true
	t.Style().Options.SeparateColumns = true
	t.Style().Options.SeparateHeader = true

	headerRow := prettytable.Row{}
	for _, header := range headers {
		headerRow = append(headerRow, strings.ToUpper(header))
	}
	if !opts.noHeader {
		t.AppendHeader(headerRow)
	}

	for _, item := range data {
		v := reflect.ValueOf(item)
		if !v.IsValid() || (v.Kind() == reflect.Pointer && v.IsNil()) {
			continue
		}
		values := extractRowData(item, fieldIndices)
		row := prettytable.Row{}
		for i, val := range values {
			if !noColor {
				val = colorizeValue(val, strings.ToLower(headers[i]), noColor)
			}
			row = append(row, val)
		}
		t.AppendRow(row)
	}

	t.Render()
	return nil
}
