//go:build js && wasm

package output

import (
	"strings"
	"syscall/js"
	"testing"

	"github.com/megaport/megaport-cli/internal/wasm"
	"github.com/stretchr/testify/assert"
)

// TestWasmJSONWriter verifies JSON writer buffer
func TestWasmJSONWriter(t *testing.T) {
	WasmJSONWriter.Reset()

	testData := `{"test": "data"}`
	n, err := WasmJSONWriter.Write([]byte(testData))

	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)
	assert.Equal(t, testData, WasmJSONWriter.String())
}

// TestWasmCSVWriter verifies CSV writer buffer
func TestWasmCSVWriter(t *testing.T) {
	WasmCSVWriter.Reset()

	testData := "id,name,active\n1,test,true\n"
	n, err := WasmCSVWriter.Write([]byte(testData))

	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)
	assert.Equal(t, testData, WasmCSVWriter.String())
}

// TestPrintJSON_WASM verifies JSON output in WASM
func TestPrintJSON_WASM(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "Item 1", Active: true},
		{ID: 2, Name: "Item 2", Active: false},
	}

	// Clear buffers and globals
	WasmJSONWriter.Reset()
	js.Global().Delete("wasmJSONOutput")

	// Print JSON
	err := printJSON(data, currentPrintOptions())
	assert.NoError(t, err)

	// Verify buffer has content
	output := WasmJSONWriter.String()
	assert.NotEmpty(t, output)
	assert.Contains(t, output, `"id"`)
	assert.Contains(t, output, `"name"`)
	assert.Contains(t, output, `"active"`)
	assert.Contains(t, output, `"Item 1"`)
	assert.Contains(t, output, `"Item 2"`)

	// Verify it's valid JSON
	assert.True(t, output[0] == '[', "Should start with [")
	assert.True(t, output[len(output)-1] == '\n' || output[len(output)-2] == ']', "Should end with ]")

	// Verify global variable is set
	wasmJSONGlobal := js.Global().Get("wasmJSONOutput")
	assert.False(t, wasmJSONGlobal.IsUndefined())
	assert.Equal(t, output, wasmJSONGlobal.String())
}

// TestPrintJSON_WASM_EmptyData verifies empty JSON array
func TestPrintJSON_WASM_EmptyData(t *testing.T) {
	var data []SimpleStruct

	WasmJSONWriter.Reset()

	err := printJSON(data, currentPrintOptions())
	assert.NoError(t, err)

	output := WasmJSONWriter.String()
	assert.Contains(t, output, "[]")
}

// TestPrintJSON_WASM_ComplexData verifies complex JSON structures
func TestPrintJSON_WASM_ComplexData(t *testing.T) {
	data := []ComplexStruct{
		{
			ID:   1,
			Name: "Complex",
			Tags: []string{"tag1", "tag2"},
			Metadata: map[string]string{
				"key1": "value1",
			},
		},
	}

	WasmJSONWriter.Reset()

	err := printJSON(data, currentPrintOptions())
	assert.NoError(t, err)

	output := WasmJSONWriter.String()
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "Complex")
	assert.Contains(t, output, "tag1")
	assert.Contains(t, output, "key1")
}

// TestPrintCSV_WASM verifies CSV output in WASM
func TestPrintCSV_WASM(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "Item 1", Active: true},
		{ID: 2, Name: "Item 2", Active: false},
	}

	// Clear buffers and globals
	WasmCSVWriter.Reset()
	js.Global().Delete("wasmCSVOutput")

	// Print CSV
	err := printCSV(data, currentPrintOptions())
	assert.NoError(t, err)

	// Verify buffer has content
	output := WasmCSVWriter.String()
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "id,name,active")
	assert.Contains(t, output, "1,Item 1,true")
	assert.Contains(t, output, "2,Item 2,false")

	// Verify global variable is set
	wasmCSVGlobal := js.Global().Get("wasmCSVOutput")
	assert.False(t, wasmCSVGlobal.IsUndefined())
}

// TestPrintCSV_WASM_EmptyData verifies CSV headers only
func TestPrintCSV_WASM_EmptyData(t *testing.T) {
	var data []SimpleStruct

	WasmCSVWriter.Reset()

	err := printCSV(data, currentPrintOptions())
	assert.NoError(t, err)

	output := WasmCSVWriter.String()
	assert.Contains(t, output, "id,name,active")
}

// TestPrintCSV_WASM_SpecialCharacters verifies CSV escaping
func TestPrintCSV_WASM_SpecialCharacters(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: `Item with "quotes"`, Active: true},
		{ID: 2, Name: "Item with, comma", Active: false},
	}

	WasmCSVWriter.Reset()

	err := printCSV(data, currentPrintOptions())
	assert.NoError(t, err)

	output := WasmCSVWriter.String()
	assert.NotEmpty(t, output)
	// CSV should properly escape quotes and commas
	assert.Contains(t, output, "Item with")
}

// TestPrintJSON_WASM_AccumulatesMultipleCalls verifies that a command emitting
// more than one JSON document in a single invocation has all of them captured,
// not just the last (ESD-1650).
func TestPrintJSON_WASM_AccumulatesMultipleCalls(t *testing.T) {
	first := []SimpleStruct{{ID: 1, Name: "First", Active: true}}
	second := []SimpleStruct{{ID: 2, Name: "Second", Active: false}}

	WasmJSONWriter.Reset()
	js.Global().Delete("wasmJSONOutput")

	assert.NoError(t, printJSON(first, currentPrintOptions()))
	assert.NoError(t, printJSON(second, currentPrintOptions()))

	output := WasmJSONWriter.String()
	assert.Contains(t, output, "First", "first document must survive a second call")
	assert.Contains(t, output, "Second", "second document must also be present")

	global := js.Global().Get("wasmJSONOutput")
	assert.Equal(t, output, global.String(), "global must reflect the full accumulated buffer")

	WasmJSONWriter.Reset()
	js.Global().Delete("wasmJSONOutput")
}

// TestPrintCSV_WASM_AccumulatesMultipleCalls mirrors the JSON accumulation
// test for CSV output.
func TestPrintCSV_WASM_AccumulatesMultipleCalls(t *testing.T) {
	first := []SimpleStruct{{ID: 1, Name: "First", Active: true}}
	second := []SimpleStruct{{ID: 2, Name: "Second", Active: false}}

	WasmCSVWriter.Reset()
	js.Global().Delete("wasmCSVOutput")

	assert.NoError(t, printCSV(first, currentPrintOptions()))
	assert.NoError(t, printCSV(second, currentPrintOptions()))

	output := WasmCSVWriter.String()
	assert.Contains(t, output, "First", "first document must survive a second call")
	assert.Contains(t, output, "Second", "second document must also be present")

	global := js.Global().Get("wasmCSVOutput")
	assert.Equal(t, output, global.String(), "global must reflect the full accumulated buffer")

	WasmCSVWriter.Reset()
	js.Global().Delete("wasmCSVOutput")
}

// TestPrintXML_WASM verifies basic XML output capture in WASM, a case not
// previously covered by this suite.
func TestPrintXML_WASM(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "Item 1", Active: true},
	}

	WasmXMLWriter.Reset()
	js.Global().Delete("wasmXMLOutput")

	err := printXML(data, currentPrintOptions())
	assert.NoError(t, err)

	output := WasmXMLWriter.String()
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "<items>")
	assert.Contains(t, output, "<item>")
	assert.Contains(t, output, "<name>Item 1</name>")

	global := js.Global().Get("wasmXMLOutput")
	assert.False(t, global.IsUndefined())
	assert.Equal(t, output, global.String())

	WasmXMLWriter.Reset()
	js.Global().Delete("wasmXMLOutput")
}

// TestPrintXML_WASM_AccumulatesMultipleCalls mirrors the JSON/CSV
// accumulation tests for XML output.
func TestPrintXML_WASM_AccumulatesMultipleCalls(t *testing.T) {
	first := []SimpleStruct{{ID: 1, Name: "First", Active: true}}
	second := []SimpleStruct{{ID: 2, Name: "Second", Active: false}}

	WasmXMLWriter.Reset()
	js.Global().Delete("wasmXMLOutput")

	assert.NoError(t, printXML(first, currentPrintOptions()))
	assert.NoError(t, printXML(second, currentPrintOptions()))

	output := WasmXMLWriter.String()
	assert.Contains(t, output, "<name>First</name>", "first document must survive a second call")
	assert.Contains(t, output, "<name>Second</name>", "second document must also be present")

	global := js.Global().Get("wasmXMLOutput")
	assert.Equal(t, output, global.String(), "global must reflect the full accumulated buffer")

	WasmXMLWriter.Reset()
	js.Global().Delete("wasmXMLOutput")
}

// TestResetWasmStructuredBuffers_ClearsAllBuffers verifies the between-
// invocation reset hook (wired through output.ResetState) clears every
// structured-output buffer, so state never bleeds from one WASM command
// invocation into the next.
func TestResetWasmStructuredBuffers_ClearsAllBuffers(t *testing.T) {
	WasmJSONWriter.WriteString("stale json")
	WasmCSVWriter.WriteString("stale csv")
	WasmXMLWriter.WriteString("stale xml")
	WasmTableWriter.WriteString("stale table")

	ResetState()

	assert.Empty(t, WasmJSONWriter.String())
	assert.Empty(t, WasmCSVWriter.String())
	assert.Empty(t, WasmXMLWriter.String())
	assert.Empty(t, WasmTableWriter.String())
}

// TestPrintOutput_WASM_Formats verifies all output formats
func TestPrintOutput_WASM_Formats(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "Test", Active: true},
	}

	formats := []struct {
		name   string
		format string
	}{
		{"JSON", "json"},
		{"CSV", "csv"},
		{"Table", "table"},
	}

	for _, tt := range formats {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all buffers
			WasmJSONWriter.Reset()
			WasmCSVWriter.Reset()
			WasmTableWriter.Reset()
			js.Global().Delete("wasmJSONOutput")
			js.Global().Delete("wasmCSVOutput")
			js.Global().Delete("wasmTableOutput")

			err := PrintOutput(data, tt.format, false)
			assert.NoError(t, err)

			// At least one buffer should have content
			hasOutput := WasmJSONWriter.String() != "" ||
				WasmCSVWriter.String() != "" ||
				WasmTableWriter.String() != ""
			assert.True(t, hasOutput, "At least one buffer should have output")
		})
	}
}

// TestWasmGlobalVariables verifies all WASM global variables
func TestWasmGlobalVariables(t *testing.T) {
	// Test JSON global
	testJSON := `{"test": "json"}`
	js.Global().Set("wasmJSONOutput", testJSON)
	result := js.Global().Get("wasmJSONOutput")
	assert.False(t, result.IsUndefined())
	assert.Equal(t, testJSON, result.String())

	// Test CSV global
	testCSV := "id,name\n1,test"
	js.Global().Set("wasmCSVOutput", testCSV)
	result = js.Global().Get("wasmCSVOutput")
	assert.False(t, result.IsUndefined())
	assert.Equal(t, testCSV, result.String())

	// Test Table global
	testTable := "┌─────┐\n│ Test │\n└─────┘"
	js.Global().Set("wasmTableOutput", testTable)
	result = js.Global().Get("wasmTableOutput")
	assert.False(t, result.IsUndefined())
	assert.Equal(t, testTable, result.String())

	// Clean up
	js.Global().Delete("wasmJSONOutput")
	js.Global().Delete("wasmCSVOutput")
	js.Global().Delete("wasmTableOutput")
}

// TestWasmBufferReset verifies buffer reset functionality
func TestWasmBufferReset(t *testing.T) {
	// Write to all buffers
	WasmJSONWriter.WriteString("json data")
	WasmCSVWriter.WriteString("csv data")
	WasmTableWriter.WriteString("table data")

	// Verify they have content
	assert.NotEmpty(t, WasmJSONWriter.String())
	assert.NotEmpty(t, WasmCSVWriter.String())
	assert.NotEmpty(t, WasmTableWriter.String())

	// Reset all
	WasmJSONWriter.Reset()
	WasmCSVWriter.Reset()
	WasmTableWriter.Reset()

	// Verify they're empty
	assert.Empty(t, WasmJSONWriter.String())
	assert.Empty(t, WasmCSVWriter.String())
	assert.Empty(t, WasmTableWriter.String())
}

// TestPrintOutput_WASM_InvalidFormat verifies error handling
func TestPrintOutput_WASM_InvalidFormat(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "Test", Active: true},
	}

	err := PrintOutput(data, "invalid", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
}

// TestWasmOutputConcurrency verifies thread-safe buffer operations
func TestWasmOutputConcurrency(t *testing.T) {
	done := make(chan bool)
	iterations := 50

	// Multiple goroutines writing to different buffers
	go func() {
		for i := 0; i < iterations; i++ {
			WasmJSONWriter.WriteString("j")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < iterations; i++ {
			WasmCSVWriter.WriteString("c")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < iterations; i++ {
			WasmTableWriter.WriteString("t")
		}
		done <- true
	}()

	// Wait for all
	<-done
	<-done
	<-done

	// All writes should have succeeded
	assert.Equal(t, iterations, len(WasmJSONWriter.String()))
	assert.Equal(t, iterations, len(WasmCSVWriter.String()))
	assert.Equal(t, iterations, len(WasmTableWriter.String()))
}

// TestPrintNewline_WASM verifies PrintNewline writes a newline to WasmOutputBuffer.
func TestPrintNewline_WASM(t *testing.T) {
	wasm.WasmOutputBuffer.Reset()
	SetVerbosity("normal")

	PrintNewline()

	assert.Equal(t, "\n", wasm.WasmOutputBuffer.String(), "PrintNewline should write exactly one newline to WasmOutputBuffer")
	wasm.WasmOutputBuffer.Reset()
}

// TestPrintNewline_WASM_QuietSuppresses verifies PrintNewline is a no-op in quiet mode.
func TestPrintNewline_WASM_QuietSuppresses(t *testing.T) {
	wasm.WasmOutputBuffer.Reset()
	SetVerbosity("quiet")
	defer SetVerbosity("normal")

	PrintNewline()

	assert.Empty(t, wasm.WasmOutputBuffer.String(), "PrintNewline should produce no output in quiet mode")
	wasm.WasmOutputBuffer.Reset()
}

// TestGetCapturedOutput_EmptyTableWithWarning verifies that a "No X found"
// warning emitted before an empty-slice table is still visible in the captured
// output rather than being shadowed by the header-only table.
func TestGetCapturedOutput_EmptyTableWithWarning(t *testing.T) {
	wasm.ResetOutputBuffers()
	SetVerbosity("normal")

	PrintWarning("No locations found matching '%s'", true, "xyz")

	var data []SimpleStruct
	err := PrintOutput(data, "table", true)
	assert.NoError(t, err)

	out := wasm.GetCapturedOutput()
	assert.Contains(t, out, "No locations found matching 'xyz'", "warning must survive the priority switch")

	wasm.ResetOutputBuffers()
}

// TestGetCapturedOutput_StatusWithPopulatedTable verifies that status emitted
// alongside a populated table appears first, then the table.
func TestGetCapturedOutput_StatusWithPopulatedTable(t *testing.T) {
	wasm.ResetOutputBuffers()
	SetVerbosity("normal")

	PrintInfo("Found %d locations", true, 2)

	data := []SimpleStruct{
		{ID: 1, Name: "Item 1", Active: true},
		{ID: 2, Name: "Item 2", Active: false},
	}
	err := PrintOutput(data, "table", true)
	assert.NoError(t, err)

	out := wasm.GetCapturedOutput()
	assert.Contains(t, out, "Found 2 locations", "info status must be present")
	assert.Contains(t, out, "Item 1", "table rows must be present")

	statusIdx := strings.Index(out, "Found 2 locations")
	tableIdx := strings.Index(out, "Item 1")
	assert.Less(t, statusIdx, tableIdx, "status should appear before the table")

	wasm.ResetOutputBuffers()
}

// TestGetCapturedOutput_StructuredModesStayClean verifies JSON/CSV/XML modes
// return a pure data stream even when a warning was written to the direct
// buffer (only table mode prepends status).
func TestGetCapturedOutput_StructuredModesStayClean(t *testing.T) {
	formats := []struct {
		format   string
		contains string
	}{
		{"json", `"name"`},
		{"csv", "id,name,active"},
		{"xml", "<name>"},
	}

	for _, f := range formats {
		t.Run(f.format, func(t *testing.T) {
			wasm.ResetOutputBuffers()
			SetVerbosity("normal")

			PrintWarning("some warning", true)

			data := []SimpleStruct{
				{ID: 1, Name: "Item 1", Active: true},
			}
			err := PrintOutput(data, f.format, true)
			assert.NoError(t, err)

			out := wasm.GetCapturedOutput()
			assert.Contains(t, out, f.contains, "structured payload must be returned")
			assert.NotContains(t, out, "some warning", "status text must not leak into the data stream")

			wasm.ResetOutputBuffers()
		})
	}
}

// TestConsoleLogging verifies console.log doesn't cause panics
func TestConsoleLogging(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "Test", Active: true},
	}

	// All print functions should handle console.log gracefully
	assert.NotPanics(t, func() {
		WasmJSONWriter.Reset()
		_ = printJSON(data, currentPrintOptions())
	})

	assert.NotPanics(t, func() {
		WasmCSVWriter.Reset()
		_ = printCSV(data, currentPrintOptions())
	})

	assert.NotPanics(t, func() {
		WasmTableWriter.Reset()
		_ = printTable(data, false, currentPrintOptions())
	})
}
