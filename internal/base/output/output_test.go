//go:build !wasm

package output

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// extractJSON strips ANSI escape sequences from captured output and extracts
// the first complete JSON value (array or object) using json.Decoder.
func extractJSON(s string) string {
	clean := ansiRegexp.ReplaceAllString(s, "")
	remaining := clean
	for {
		start := strings.IndexAny(remaining, "[{")
		if start == -1 {
			return clean
		}
		dec := json.NewDecoder(strings.NewReader(remaining[start:]))
		var raw json.RawMessage
		if err := dec.Decode(&raw); err == nil {
			return string(raw)
		}
		remaining = remaining[start+1:]
	}
}

var noColor = false

type CustomTagStruct struct {
	ID   int    `json:"id" csv:"csv_id" header:"Custom ID"`
	Name string `json:"name" csv:"csv_name" header:"Custom Name"`
}

type NoTagStruct struct {
	ID   int
	Name string
}

type IllegalTagStruct struct {
	ID    int    `json:"1id"`
	Value string `json:"a/b c"`
}

type CollidingTagStruct struct {
	First  string `json:"a/b"`
	Second string `json:"a b"`
}

func TestPrintCSV_SimpleStruct(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "Item 1", Active: true},
		{ID: 2, Name: "Item 2", Active: false},
	}

	output := CaptureOutput(func() {
		err := printCSV(data, currentPrintOptions())
		assert.NoError(t, err)
	})

	expected := "id,name,active\n" +
		"1,Item 1,true\n" +
		"2,Item 2,false\n"

	assert.Equal(t, expected, output)
}

func TestPrintCSV_ComplexStruct(t *testing.T) {
	now := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	data := []ComplexStruct{
		{
			ID:      1,
			Name:    "Complex Item",
			Created: now,
			Tags:    []string{"tag1", "tag2"},
			Metadata: map[string]string{
				"key1": "value1",
			},
			Reference: &SimpleStruct{ID: 100, Name: "Referenced", Active: true},
		},
	}

	output := CaptureOutput(func() {
		err := printCSV(data, currentPrintOptions())
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "id,name,created,tags,metadata,reference")
	assert.Contains(t, output, "1,Complex Item,2023-01-01")
}

func TestPrintCSV_EmptySlice(t *testing.T) {
	data := []SimpleStruct{}

	output := CaptureOutput(func() {
		err := printCSV(data, currentPrintOptions())
		assert.NoError(t, err)
	})

	expected := "id,name,active\n"
	assert.Equal(t, expected, output)
}

func TestPrintCSV_NilSlice(t *testing.T) {
	var data []SimpleStruct = nil

	_ = CaptureOutput(func() {
		err := printCSV(data, currentPrintOptions())
		assert.NoError(t, err)
	})

	assert.NotPanics(t, func() {
		_ = printCSV(data, currentPrintOptions())
	})
}

func TestPrintCSV_PointerStruct(t *testing.T) {
	s1 := &SimpleStruct{ID: 1, Name: "First", Active: true}
	s2 := &SimpleStruct{ID: 2, Name: "Second", Active: false}
	var s3 *SimpleStruct // nil pointer

	data := []*SimpleStruct{s1, s2, s3}

	output := CaptureOutput(func() {
		err := printCSV(data, currentPrintOptions())
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "id,name,active")
	assert.Contains(t, output, "1,First,true")
	assert.Contains(t, output, "2,Second,false")
	// nil pointer should be skipped
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, 3, len(lines), "header + 2 data rows (nil pointer skipped)")
}

func TestPrintCSV_NoTagStruct(t *testing.T) {
	data := []NoTagStruct{{ID: 1, Name: "Test"}}

	output := CaptureOutput(func() {
		err := printCSV(data, currentPrintOptions())
		assert.NoError(t, err)
	})

	// NoTagStruct has no csv or json tags, so CSV should produce no output
	assert.Empty(t, output)
}

func TestPrintJSON(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "Item 1", Active: true},
		{ID: 2, Name: "Item 2", Active: false},
	}

	output := CaptureOutput(func() {
		err := printJSON(data, currentPrintOptions())
		assert.NoError(t, err)
	})

	var parsed []map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	assert.NoError(t, err)

	expected := []map[string]interface{}{
		{"id": float64(1), "name": "Item 1", "active": true},
		{"id": float64(2), "name": "Item 2", "active": false},
	}

	assert.Equal(t, expected, parsed)
}

func TestIsOutputCompatibleType(t *testing.T) {
	tests := []struct {
		name      string
		fieldType reflect.Type
		expected  bool
	}{
		{"String", reflect.TypeOf(""), true},
		{"Int", reflect.TypeOf(0), true},
		{"Bool", reflect.TypeOf(false), true},
		{"Float", reflect.TypeOf(0.0), true},
		{"Slice", reflect.TypeOf([]string{}), true},
		{"Map", reflect.TypeOf(map[string]string{}), true},
		{"Struct", reflect.TypeOf(struct{}{}), true},
		{"Pointer", reflect.TypeOf(&struct{}{}), true},
		{"Time", reflect.TypeOf(time.Time{}), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOutputCompatibleType(tt.fieldType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatFieldValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"String", "test", "test"},
		{"Int", 42, "42"},
		{"Bool", true, "true"},
		{"Float", 3.14, "3.14"},
		{"Time", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), "2023-01-01"},
		{"Slice", []string{"one", "two"}, `["one","two"]`},
		{"Map", map[string]string{"key": "value"}, `{"key":"value"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.value)
			result := formatFieldValue(v)

			if tt.name == "Slice" || tt.name == "Map" {
				var expected, actual interface{}
				err := json.Unmarshal([]byte(tt.expected), &expected)
				assert.NoError(t, err)
				err = json.Unmarshal([]byte(result), &actual)
				assert.NoError(t, err)
				assert.Equal(t, expected, actual)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestPrintOutput_MixedFields(t *testing.T) {
	type MixedStruct struct {
		ID         int       `json:"id" csv:"id" header:"ID"`
		Name       string    `json:"name" csv:"name" header:"Name"`
		Created    time.Time `json:"created" csv:"created" header:"Created"`
		Active     bool      `json:"active" csv:"active" header:"Active"`
		NilPtr     *string   `json:"nil_ptr" csv:"nil_ptr" header:"Nil Pointer"`
		unexported string
	}

	timeVal := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	data := []MixedStruct{{ID: 1, Name: "Test Item", Created: timeVal, Active: true, NilPtr: nil, unexported: "hidden"}}

	formats := []string{"table", "csv", "json", "xml"}
	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			output := CaptureOutput(func() {
				err := PrintOutput(data, format, noColor)
				assert.NoError(t, err)
			})

			assert.Contains(t, output, "Test Item")
			assert.Contains(t, output, "2023-01-01")
			assert.NotContains(t, output, "hidden")
			assert.NotPanics(t, func() {
				_ = PrintOutput(data, format, noColor)
			})
		})
	}
}

func TestPrintOutput_InvalidFormat(t *testing.T) {
	data := []SimpleStruct{{ID: 1, Name: "Test"}}

	err := PrintOutput(data, "invalid", noColor)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
}

func TestCalculateColumnWidths(t *testing.T) {
	emptyWidths := calculateColumnWidths([][]string{})
	assert.Nil(t, emptyWidths)

	rows := [][]string{
		{"Header1", "Header2", "LongHeader3"},
		{"Value1", "LongerValue2", "Val3"},
		{"X", "Y", "VeryLongValue3"},
	}

	widths := calculateColumnWidths(rows)
	assert.Equal(t, 3, len(widths))
	assert.Equal(t, 7, widths[0])
	assert.Equal(t, 12, widths[1])
	assert.Equal(t, 14, widths[2])
}

func TestColorizeValue(t *testing.T) {
	originalNoColor := noColor
	defer func() { noColor = originalNoColor }()

	noColor = true
	assert.Equal(t, "ACTIVE", colorizeValue("ACTIVE", "status", noColor))
	assert.Equal(t, "123", colorizeValue("123", "id", noColor))
	assert.Equal(t, "Important", colorizeValue("Important", "name", noColor))
	assert.Equal(t, "12.50", colorizeValue("12.50", "price", noColor))

	noColor = false
	assert.NotPanics(t, func() {
		_ = colorizeValue("ACTIVE", "status", noColor)
		_ = colorizeValue("123", "id", noColor)
		_ = colorizeValue("Important", "name", noColor)
		_ = colorizeValue("12.50", "price", noColor)
	})
}

func TestColorizeStatus(t *testing.T) {
	green := color.New(color.FgHiWhite, color.BgGreen, color.Bold).Sprintf(" %s ", "LIVE")
	yellow := color.New(color.FgBlack, color.BgYellow, color.Bold).Sprintf(" %s ", "CONFIGURED")
	orange := color.New(color.FgHiWhite, color.BgHiRed, color.Bold).Sprintf(" %s ", "DECOMMISSIONING")

	// noColor=true returns plain text
	assert.Equal(t, "LIVE", colorizeStatus("LIVE", true))
	assert.Equal(t, "CONFIGURED", colorizeStatus("CONFIGURED", true))

	// Green group: LIVE, ACTIVE, UP, AVAILABLE
	assert.Equal(t, green, colorizeStatus("live", false))
	assert.Equal(t, color.New(color.FgHiWhite, color.BgGreen, color.Bold).Sprintf(" %s ", "ACTIVE"), colorizeStatus("ACTIVE", false))

	// Yellow group: CONFIGURED, DEPLOYABLE, PENDING, PROVISIONING
	assert.Equal(t, yellow, colorizeStatus("configured", false))
	assert.Equal(t, color.New(color.FgBlack, color.BgYellow, color.Bold).Sprintf(" %s ", "DEPLOYABLE"), colorizeStatus("DEPLOYABLE", false))
	assert.Equal(t, color.New(color.FgBlack, color.BgYellow, color.Bold).Sprintf(" %s ", "PENDING"), colorizeStatus("PENDING", false))

	// Orange group: DECOMMISSIONING, DECOMMISSIONED, CANCELLED, DELETED, DOWN, INACTIVE
	assert.Equal(t, orange, colorizeStatus("decommissioning", false))
	assert.Equal(t, color.New(color.FgHiWhite, color.BgHiRed, color.Bold).Sprintf(" %s ", "CANCELLED"), colorizeStatus("CANCELLED", false))

	// Default: DESIGN and unknown values → blue badge
	assert.Equal(t, color.New(color.FgHiWhite, color.BgBlue, color.Bold).Sprintf(" %s ", "DESIGN"), colorizeStatus("DESIGN", false))
}

func TestExtractFieldInfo(t *testing.T) {
	type TestStruct struct {
		ID       int    `json:"id" header:"Custom ID"`
		Name     string `json:"name" csv:"custom_name"`
		Internal string `json:"-" csv:"-" header:"-"`
	}

	headers, _, indices := extractFieldInfo(reflect.TypeOf(TestStruct{}))

	assert.Equal(t, 2, len(headers))
	assert.Contains(t, headers, "Custom ID")
	assert.Contains(t, headers, "custom_name")
	assert.Equal(t, []int{0, 1}, indices)
}

func TestPrintPrettyTable_SimpleStruct(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "Item 1", Active: true},
		{ID: 2, Name: "Item 2", Active: false},
	}

	output := CaptureOutput(func() {
		err := PrintOutput(data, "table", noColor)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "│")
	assert.Contains(t, output, "─")
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "ACTIVE")
	assert.Contains(t, output, " 1 ")
	assert.Contains(t, output, " Item 1 ")
	assert.Contains(t, output, " true ")
}

func TestPrintTableToWriter(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "Item 1", Active: true},
		{ID: 2, Name: "Item 2", Active: false},
	}

	var buf strings.Builder
	err := PrintTableToWriter(&buf, data, true)
	assert.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "ID")
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "Item 1")
	assert.Contains(t, out, "Item 2")
}

func TestPrintTableToWriter_EmptySlice(t *testing.T) {
	var buf strings.Builder
	err := PrintTableToWriter(&buf, []SimpleStruct{}, true)
	assert.NoError(t, err)
}

func TestPrintTableToWriter_UnknownFieldErrors(t *testing.T) {
	SetOutputFields([]string{"definitely_not_a_field"})
	defer SetOutputFields(nil)

	var buf strings.Builder
	err := PrintTableToWriter(&buf, []SimpleStruct{{ID: 1, Name: "Item 1"}}, true)
	assert.Error(t, err)
}

func TestPrintPrettyTable_ComplexStruct(t *testing.T) {
	reference := SimpleStruct{ID: 100, Name: "Reference", Active: true}
	now := time.Now()
	data := []ComplexStruct{{ID: 1, Name: "Complex Item", Created: now, Tags: []string{"tag1", "tag2"}, Metadata: map[string]string{"key": "value"}, Reference: &reference}}

	output := CaptureOutput(func() {
		err := PrintOutput(data, "table", noColor)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "CREATED")
	assert.Contains(t, output, "TAGS")
	assert.Contains(t, output, "METADATA")
	assert.Contains(t, output, "REFERENCE")
	assert.Contains(t, output, "Complex Item")
}

func TestPrintPrettyTable_EmptySlice(t *testing.T) {
	var data []SimpleStruct

	output := CaptureOutput(func() {
		err := PrintOutput(data, "table", noColor)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "ACTIVE")
	assert.Contains(t, output, "│")
	assert.Contains(t, output, "─")
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, 4, len(lines), "Empty table should have header, separator and closing line")
}

func TestPrintPrettyTable_NilSlice(t *testing.T) {
	var data []SimpleStruct = nil

	output := CaptureOutput(func() {
		err := PrintOutput(data, "table", noColor)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "ACTIVE")
	assert.Contains(t, output, "│")
	assert.Contains(t, output, "─")
}

func TestPrintPrettyTable_CustomHeaders(t *testing.T) {
	data := []CustomTagStruct{{ID: 1, Name: "Custom Item"}}

	output := CaptureOutput(func() {
		err := PrintOutput(data, "table", noColor)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "CUSTOM ID")
	assert.Contains(t, output, "CUSTOM NAME")
}

func TestTableColorization(t *testing.T) {
	type ColorTestStruct struct {
		ID       string `json:"id" header:"ID"`
		Status   string `json:"status" header:"STATUS"`
		Name     string `json:"name" header:"NAME"`
		Price    string `json:"price" header:"PRICE"`
		Speed    int    `json:"speed" header:"SPEED"`
		Location string `json:"location" header:"LOCATION"`
		Empty    string `json:"empty" header:"EMPTY"`
	}

	data := []ColorTestStruct{{ID: "test-123", Status: "ACTIVE", Name: "Test Item", Price: "99.99", Speed: 1000, Location: "NYC", Empty: ""}}

	outputColor := CaptureOutput(func() {
		err := PrintOutput(data, "table", false)
		assert.NoError(t, err)
	})

	outputNoColor := CaptureOutput(func() {
		err := PrintOutput(data, "table", true)
		assert.NoError(t, err)
	})

	hasColorCodes := strings.Contains(outputColor, "\033[") ||
		strings.Contains(outputColor, "\u001b[")

	noColorCodes := !strings.Contains(outputNoColor, "\033[") &&
		!strings.Contains(outputNoColor, "\u001b[")

	assert.True(t, hasColorCodes, "Color output should contain ANSI color codes")
	assert.True(t, noColorCodes, "No-color output should not contain ANSI color codes")
}

func TestPrintPrettyTable_TableStyle(t *testing.T) {
	data := []SimpleStruct{{ID: 1, Name: "Test Item", Active: true}}

	output := CaptureOutput(func() {
		err := PrintOutput(data, "table", noColor)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "│")
	assert.Contains(t, output, "─")
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "ACTIVE")
	assert.Contains(t, output, " ID ")
	assert.Contains(t, output, " NAME ")
	assert.Contains(t, output, " ACTIVE ")
}

func TestJSONOutput_JQCompatibility(t *testing.T) {
	// Test that JSON output is clean and can be parsed by jq-like parsers
	data := []SimpleStruct{
		{ID: 1, Name: "Item 1", Active: true},
		{ID: 2, Name: "Item 2", Active: false},
	}

	output := CaptureOutput(func() {
		err := PrintOutput(data, "json", noColor)
		assert.NoError(t, err)
	})

	// Verify the output is valid JSON
	var parsed []map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	assert.NoError(t, err, "JSON output should be valid and parseable")

	// Verify the structure matches expectations
	assert.Len(t, parsed, 2, "Should have 2 items")
	assert.Equal(t, float64(1), parsed[0]["id"])
	assert.Equal(t, "Item 1", parsed[0]["name"])
	assert.Equal(t, true, parsed[0]["active"])

	// Verify no extra whitespace or control characters
	trimmed := strings.TrimSpace(output)
	assert.True(t, strings.HasPrefix(trimmed, "["), "JSON should start with [")
	assert.True(t, strings.HasSuffix(trimmed, "]"), "JSON should end with ]")

	// Verify no ANSI color codes in JSON output
	assert.False(t, strings.Contains(output, "\033["), "JSON should not contain ANSI escape codes")
	assert.False(t, strings.Contains(output, "\u001b["), "JSON should not contain ANSI escape codes")
}

func TestJSONOutput_ComplexData_JQCompatibility(t *testing.T) {
	// Test complex data structures for jq compatibility
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	reference := &SimpleStruct{ID: 100, Name: "Referenced Item", Active: true}

	data := []ComplexStruct{
		{
			ID:        1,
			Name:      "Complex Item",
			Created:   now,
			Tags:      []string{"tag1", "tag2", "tag3"},
			Metadata:  map[string]string{"key1": "value1", "key2": "value2"},
			Reference: reference,
		},
	}

	output := CaptureOutput(func() {
		err := PrintOutput(data, "json", noColor)
		assert.NoError(t, err)
	})

	// Verify the output is valid JSON
	var parsed []map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	assert.NoError(t, err, "Complex JSON output should be valid and parseable")

	// Verify complex structures are properly serialized
	assert.Len(t, parsed, 1)
	item := parsed[0]

	assert.Equal(t, float64(1), item["id"])
	assert.Equal(t, "Complex Item", item["name"])
	assert.Contains(t, item["created"], "2023-01-01T12:00:00Z")

	// Verify arrays are properly serialized
	tags, ok := item["tags"].([]interface{})
	assert.True(t, ok, "Tags should be an array")
	assert.Len(t, tags, 3)
	assert.Equal(t, "tag1", tags[0])

	// Verify maps are properly serialized
	metadata, ok := item["metadata"].(map[string]interface{})
	assert.True(t, ok, "Metadata should be a map")
	assert.Equal(t, "value1", metadata["key1"])

	// Verify nested objects are properly serialized
	ref, ok := item["reference"].(map[string]interface{})
	assert.True(t, ok, "Reference should be an object")
	assert.Equal(t, float64(100), ref["id"])
	assert.Equal(t, "Referenced Item", ref["name"])
}

func TestJSONOutput_EmptyData_JQCompatibility(t *testing.T) {
	// Test empty data produces valid JSON
	var data []SimpleStruct

	output := CaptureOutput(func() {
		err := PrintOutput(data, "json", noColor)
		assert.NoError(t, err)
	})

	// Verify empty array is valid JSON
	var parsed []interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	assert.NoError(t, err, "Empty JSON array should be valid")
	assert.Len(t, parsed, 0, "Empty array should have no elements")

	// Verify it's exactly "[]" (or "[]" with whitespace)
	trimmed := strings.TrimSpace(output)
	assert.Equal(t, "[]", trimmed, "Empty data should produce clean empty JSON array")
}

func TestJSONOutput_NilData_JQCompatibility(t *testing.T) {
	// Test nil data produces valid JSON
	var data []SimpleStruct = nil

	output := CaptureOutput(func() {
		err := PrintOutput(data, "json", noColor)
		assert.NoError(t, err)
	})

	// Verify nil produces valid JSON
	var parsed []interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	assert.NoError(t, err, "Nil data JSON should be valid")
	assert.Len(t, parsed, 0, "Nil data should produce empty array")
}

func TestJSONOutput_SpecialCharacters_JQCompatibility(t *testing.T) {
	// Test data with special characters that need proper JSON escaping
	data := []SimpleStruct{
		{ID: 1, Name: "Item with \"quotes\" and \nnewlines\tand\ttabs", Active: true},
		{ID: 2, Name: "Item with unicode: 🚀 and backslash: \\", Active: false},
	}

	output := CaptureOutput(func() {
		err := PrintOutput(data, "json", noColor)
		assert.NoError(t, err)
	})

	// Verify the output is valid JSON despite special characters
	var parsed []map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	assert.NoError(t, err, "JSON with special characters should be valid")

	// Verify special characters are properly escaped and preserved
	assert.Contains(t, parsed[0]["name"], "quotes")
	assert.Contains(t, parsed[0]["name"], "newlines")
	assert.Contains(t, parsed[0]["name"], "tabs")
	assert.Contains(t, parsed[1]["name"], "🚀")
	assert.Contains(t, parsed[1]["name"], "\\")
}

func TestJSONOutput_LargeData_JQCompatibility(t *testing.T) {
	// Test with larger dataset to ensure performance and correctness
	var data []SimpleStruct
	for i := 1; i <= 100; i++ {
		data = append(data, SimpleStruct{
			ID:     i,
			Name:   fmt.Sprintf("Item %d", i),
			Active: i%2 == 0,
		})
	}

	output := CaptureOutput(func() {
		err := PrintOutput(data, "json", noColor)
		assert.NoError(t, err)
	})

	// Verify large JSON is still valid
	var parsed []map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	assert.NoError(t, err, "Large JSON output should be valid")
	assert.Len(t, parsed, 100, "Should have all 100 items")

	// Spot check first and last items
	assert.Equal(t, float64(1), parsed[0]["id"])
	assert.Equal(t, "Item 1", parsed[0]["name"])
	assert.Equal(t, false, parsed[0]["active"])

	assert.Equal(t, float64(100), parsed[99]["id"])
	assert.Equal(t, "Item 100", parsed[99]["name"])
	assert.Equal(t, true, parsed[99]["active"])
}

func TestJSONOutput_NoTrailingNewlines(t *testing.T) {
	// Ensure JSON output doesn't have trailing newlines that could interfere with jq
	data := []SimpleStruct{
		{ID: 1, Name: "Test", Active: true},
	}

	output := CaptureOutput(func() {
		err := PrintOutput(data, "json", noColor)
		assert.NoError(t, err)
	})

	// JSON should end with ] and a single newline, not multiple newlines
	assert.True(t, strings.HasSuffix(output, "]\n"), "JSON should end with ] followed by single newline")
	assert.False(t, strings.HasSuffix(output, "]\n\n"), "JSON should not have multiple trailing newlines")
}

func TestCaptureOutputErr_RestoresStdoutOnError(t *testing.T) {
	originalStdout := os.Stdout

	_, err := CaptureOutputErr(func() error {
		return errors.New("simulated error")
	})

	assert.Error(t, err)
	assert.Equal(t, originalStdout, os.Stdout, "os.Stdout should be restored after f() returns an error")
}

func TestCaptureOutputErr_ReturnsPartialOutputOnError(t *testing.T) {
	out, err := CaptureOutputErr(func() error {
		fmt.Print("partial output")
		return errors.New("simulated error")
	})

	assert.Error(t, err)
	assert.Equal(t, "partial output", out, "partial stdout should be returned even when f() errors")
}

func TestCaptureOutputErr_RestoresStdoutOnSuccess(t *testing.T) {
	originalStdout := os.Stdout

	out, err := CaptureOutputErr(func() error {
		fmt.Print("hello")
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "hello", out)
	assert.Equal(t, originalStdout, os.Stdout, "os.Stdout should be restored after successful execution")
}

func Test_extractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean JSON array",
			input:    `[{"uid":"abc-123","name":"test"}]`,
			expected: `[{"uid":"abc-123","name":"test"}]`,
		},
		{
			name:     "clean JSON object",
			input:    `{"key":"value"}`,
			expected: `{"key":"value"}`,
		},
		{
			name:     "JSON array with ANSI spinner prefix",
			input:    "\x1b[K\x1b[1mSpinner...\x1b[0m\n[{\"uid\":\"abc\"}]",
			expected: `[{"uid":"abc"}]`,
		},
		{
			name:     "JSON with ANSI escape sequences throughout",
			input:    "\x1b[32m✓\x1b[0m Getting resource...\x1b[K[{\"name\":\"test\"}]",
			expected: `[{"name":"test"}]`,
		},
		{
			name:     "JSON with trailing text after array",
			input:    `[{"a":1}] some trailing text`,
			expected: `[{"a":1}]`,
		},
		{
			name:     "no JSON content returns cleaned input",
			input:    "just plain text with no JSON",
			expected: "just plain text with no JSON",
		},
		{
			name:     "ANSI-only input with no JSON",
			input:    "\x1b[1mBold text\x1b[0m",
			expected: "Bold text",
		},
		{
			name:     "invalid JSON after bracket returns cleaned input",
			input:    "[not valid json",
			expected: "[not valid json",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "nested JSON object",
			input:    "prefix {\"outer\":{\"inner\":true}} suffix",
			expected: `{"outer":{"inner":true}}`,
		},
		{
			name:     "non-JSON bracket before valid JSON",
			input:    `Command: deploy [a b] [{"uid":"abc-123"}]`,
			expected: `[{"uid":"abc-123"}]`,
		},
		{
			name:     "non-JSON brace before valid JSON",
			input:    `log {invalid [{"name":"test"}]`,
			expected: `[{"name":"test"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPrintXML_SimpleStruct(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "Test Port", Active: true},
		{ID: 2, Name: "Another Port", Active: false},
	}

	output := CaptureOutput(func() {
		err := printXML(data, currentPrintOptions())
		assert.NoError(t, err)
	})

	assert.Contains(t, output, `<?xml version="1.0" encoding="UTF-8"?>`)
	assert.Contains(t, output, "<items>")
	assert.Contains(t, output, "<item>")
	assert.Contains(t, output, "<id>1</id>")
	assert.Contains(t, output, "<name>Test Port</name>")
	assert.Contains(t, output, "<active>true</active>")
	assert.Contains(t, output, "<id>2</id>")
	assert.Contains(t, output, "<name>Another Port</name>")
	assert.Contains(t, output, "<active>false</active>")
	assert.Contains(t, output, "</items>")
}

func TestPrintXML_EmptySlice(t *testing.T) {
	data := []SimpleStruct{}

	output := CaptureOutput(func() {
		err := printXML(data, currentPrintOptions())
		assert.NoError(t, err)
	})

	assert.Contains(t, output, `<?xml version="1.0" encoding="UTF-8"?>`)
	assert.Contains(t, output, "<items>")
	assert.Contains(t, output, "</items>")
	assert.NotContains(t, output, "<item>")
}

func TestPrintXML_NilSlice(t *testing.T) {
	var data []SimpleStruct = nil

	assert.NotPanics(t, func() {
		output := CaptureOutput(func() {
			err := printXML(data, currentPrintOptions())
			assert.NoError(t, err)
		})
		assert.Contains(t, output, "<items>")
	})
}

// assertValidXML decodes output in full and requires it to end in io.EOF, so
// a malformed document (which would otherwise stop the decoder early with a
// syntax error) fails the assertion instead of silently passing.
func assertValidXML(t *testing.T, output string) {
	t.Helper()
	decoder := xml.NewDecoder(strings.NewReader(output))
	for {
		_, err := decoder.Token()
		if err != nil {
			assert.ErrorIs(t, err, io.EOF, "output is not well-formed XML")
			return
		}
	}
}

func TestPrintXML_ComplexStruct(t *testing.T) {
	now := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	ref := &SimpleStruct{ID: 100, Name: "Referenced", Active: true}

	data := []ComplexStruct{
		{
			ID:        1,
			Name:      "Complex Item",
			Created:   now,
			Tags:      []string{"tag1", "tag2"},
			Metadata:  map[string]string{"key1": "value1"},
			Reference: ref,
		},
	}

	output := CaptureOutput(func() {
		err := printXML(data, currentPrintOptions())
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "<id>1</id>")
	assert.Contains(t, output, "<name>Complex Item</name>")
	assert.Contains(t, output, "<created>2023-01-01</created>")
	assert.Contains(t, output, "<tags>")
	assert.Contains(t, output, "<metadata>")
	assert.Contains(t, output, "<reference>")

	// Verify it's parseable XML
	assertValidXML(t, output)
}

func TestPrintXML_PointerStruct(t *testing.T) {
	s1 := &SimpleStruct{ID: 1, Name: "First", Active: true}
	s2 := &SimpleStruct{ID: 2, Name: "Second", Active: false}
	var s3 *SimpleStruct // nil pointer

	data := []*SimpleStruct{s1, s2, s3}

	output := CaptureOutput(func() {
		err := printXML(data, currentPrintOptions())
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "<id>1</id>")
	assert.Contains(t, output, "<name>First</name>")
	assert.Contains(t, output, "<id>2</id>")
	assert.Contains(t, output, "<name>Second</name>")
	// nil pointer should be skipped
	count := strings.Count(output, "<item>")
	assert.Equal(t, 2, count)
}

func TestPrintXML_CustomTagStruct(t *testing.T) {
	data := []CustomTagStruct{
		{ID: 1, Name: "Custom"},
	}

	output := CaptureOutput(func() {
		err := printXML(data, currentPrintOptions())
		assert.NoError(t, err)
	})

	// json tags should be used for element names
	assert.Contains(t, output, "<id>1</id>")
	assert.Contains(t, output, "<name>Custom</name>")
}

func TestPrintXML_NoTagStruct(t *testing.T) {
	data := []NoTagStruct{
		{ID: 42, Name: "NoTags"},
	}

	output := CaptureOutput(func() {
		err := printXML(data, currentPrintOptions())
		assert.NoError(t, err)
	})

	// Should fall back to lowercased field names
	assert.Contains(t, output, "<id>42</id>")
	assert.Contains(t, output, "<name>NoTags</name>")
}

func TestPrintXML_SpecialCharacters(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: `<script>alert("xss")</script> & 'quotes'`, Active: true},
	}

	output := CaptureOutput(func() {
		err := printXML(data, currentPrintOptions())
		assert.NoError(t, err)
	})

	// Special characters should be escaped
	assert.NotContains(t, output, `<script>`)
	assert.Contains(t, output, "&lt;script&gt;")
	assert.Contains(t, output, "&amp;")

	// Should still be parseable
	assertValidXML(t, output)
}

func TestPrintXML_IllegalElementName(t *testing.T) {
	data := []IllegalTagStruct{
		{ID: 1, Value: "test"},
	}

	output := CaptureOutput(func() {
		err := printXML(data, currentPrintOptions())
		assert.NoError(t, err)
	})

	// json tag "1id" starts with a digit, which is illegal as an XML name-start
	// character; json tag "a/b c" contains characters illegal in an XML name.
	assert.NotContains(t, output, "<1id>")
	assert.NotContains(t, output, "<a/b c>")

	assertValidXML(t, output)
}

func TestSanitizeXMLElementName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"already valid", "name", "name"},
		{"empty", "", "field"},
		{"leading digit", "1id", "_1id"},
		{"leading underscore", "_id", "_id"},
		{"illegal characters", "a/b c", "a_b_c"},
		{"xml prefix", "xmlns", "_xmlns"},
		{"mixed case xml prefix", "XMLThing", "_XMLThing"},
		{"colon is namespace-significant, treated as illegal", "a:b", "a_b"},
		{"other punctuation", "a@b#c(d)e,f;g=h", "a_b_c_d_e_f_g_h"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sanitizeXMLElementName(tt.input))
		})
	}
}

func TestSanitizeXMLElementNames_DisambiguatesCollisions(t *testing.T) {
	// "a/b" and "a b" both sanitize to "a_b"; the second must be renamed so
	// no two elements in the same XML item share a name.
	got := sanitizeXMLElementNames([]string{"a/b", "a b", "c"})
	assert.Equal(t, []string{"a_b", "a_b_2", "c"}, got)
}

func TestPrintXML_CollidingElementNames(t *testing.T) {
	data := []CollidingTagStruct{
		{First: "one", Second: "two"},
	}

	output := CaptureOutput(func() {
		err := printXML(data, currentPrintOptions())
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "<a_b>one</a_b>")
	assert.Contains(t, output, "<a_b_2>two</a_b_2>")

	assertValidXML(t, output)
}

func TestPrintOutput_XMLFormat(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "Test", Active: true},
	}

	output := CaptureOutput(func() {
		err := PrintOutput(data, "xml", noColor)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, `<?xml version="1.0" encoding="UTF-8"?>`)
	assert.Contains(t, output, "<items>")
	assert.Contains(t, output, "<item>")
	assert.Contains(t, output, "<id>1</id>")
	assert.Contains(t, output, "</items>")

	// Verify parseable by xml.Decoder
	assertValidXML(t, output)
}

func TestPrintXML_Parseable(t *testing.T) {
	data := []SimpleStruct{
		{ID: 1, Name: "First", Active: true},
		{ID: 2, Name: "Second", Active: false},
		{ID: 3, Name: "Third", Active: true},
	}

	output := CaptureOutput(func() {
		err := printXML(data, currentPrintOptions())
		assert.NoError(t, err)
	})

	// Parse back and count items
	decoder := xml.NewDecoder(strings.NewReader(output))
	itemCount := 0
	for {
		tok, err := decoder.Token()
		if err != nil {
			assert.ErrorIs(t, err, io.EOF, "output is not well-formed XML")
			break
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "item" {
			itemCount++
		}
	}
	assert.Equal(t, len(data), itemCount, "XML item count should match input data length")
}

// ---- --fields flag tests ----

type fieldsTestStruct struct {
	UID    string `json:"uid" header:"UID"`
	Name   string `json:"name" header:"Name"`
	Status string `json:"status" header:"Status"`
	Speed  int    `json:"port_speed" header:"Port Speed"`
}

func (fieldsTestStruct) isOutput() {}

func fieldsTestData() []fieldsTestStruct {
	return []fieldsTestStruct{
		{UID: "aaa-111", Name: "Port A", Status: "LIVE", Speed: 1000},
		{UID: "bbb-222", Name: "Port B", Status: "INACTIVE", Speed: 10000},
	}
}

func TestSetOutputFields_Table(t *testing.T) {
	origIsTerminal := isTerminalCached.Load()
	t.Cleanup(func() { ResetState(); SetIsTerminal(origIsTerminal) })
	SetOutputFields([]string{"uid", "name"})
	SetIsTerminal(false)

	out := CaptureOutput(func() {
		err := PrintOutput(fieldsTestData(), "table", true)
		assert.NoError(t, err)
	})

	assert.Contains(t, out, "UID")
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "aaa-111")
	assert.Contains(t, out, "Port A")
	assert.NotContains(t, out, "STATUS")
	assert.NotContains(t, out, "PORT SPEED")
}

func TestSetOutputFields_CSV(t *testing.T) {
	t.Cleanup(func() { ResetState() })
	SetOutputFields([]string{"uid", "status"})

	out := CaptureOutput(func() {
		err := PrintOutput(fieldsTestData(), "csv", true)
		assert.NoError(t, err)
	})

	assert.Contains(t, out, "uid")
	assert.Contains(t, out, "status")
	assert.Contains(t, out, "aaa-111")
	assert.Contains(t, out, "LIVE")
	assert.NotContains(t, out, "name")
	assert.NotContains(t, out, "port_speed")
}

func TestSetOutputFields_JSON(t *testing.T) {
	t.Cleanup(func() { ResetState() })
	SetOutputFields([]string{"uid", "name"})

	out := CaptureOutput(func() {
		err := PrintOutput(fieldsTestData(), "json", true)
		assert.NoError(t, err)
	})

	var rows []map[string]interface{}
	assert.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out)), &rows))
	assert.Len(t, rows, 2)
	assert.Equal(t, "aaa-111", rows[0]["uid"])
	assert.Equal(t, "Port A", rows[0]["name"])
	_, hasStatus := rows[0]["status"]
	assert.False(t, hasStatus, "status should not appear in filtered JSON")
}

func TestSetOutputFields_XML(t *testing.T) {
	t.Cleanup(func() { ResetState() })
	SetOutputFields([]string{"uid", "name"})

	out := CaptureOutput(func() {
		err := PrintOutput(fieldsTestData(), "xml", true)
		assert.NoError(t, err)
	})

	assert.Contains(t, out, "<uid>aaa-111</uid>")
	assert.Contains(t, out, "<name>Port A</name>")
	assert.NotContains(t, out, "<status>")
	assert.NotContains(t, out, "<port_speed>")
}

func TestSetOutputFields_CaseInsensitive(t *testing.T) {
	origIsTerminal := isTerminalCached.Load()
	t.Cleanup(func() { ResetState(); SetIsTerminal(origIsTerminal) })
	SetOutputFields([]string{"UID", "NAME"}) // uppercase
	SetIsTerminal(false)

	out := CaptureOutput(func() {
		err := PrintOutput(fieldsTestData(), "table", true)
		assert.NoError(t, err)
	})

	assert.Contains(t, out, "aaa-111")
	assert.Contains(t, out, "Port A")
	assert.NotContains(t, out, "LIVE")
}

func TestSetOutputFields_HeaderNameAlias(t *testing.T) {
	origIsTerminal := isTerminalCached.Load()
	t.Cleanup(func() { ResetState(); SetIsTerminal(origIsTerminal) })
	// Match by header name "Port Speed" (has a space)
	SetOutputFields([]string{"Port Speed"})
	SetIsTerminal(false)

	out := CaptureOutput(func() {
		err := PrintOutput(fieldsTestData(), "table", true)
		assert.NoError(t, err)
	})

	assert.Contains(t, out, "PORT SPEED")
	assert.Contains(t, out, "1000")
	assert.NotContains(t, out, "UID")
}

func TestSetOutputFields_UnknownField(t *testing.T) {
	origIsTerminal := isTerminalCached.Load()
	t.Cleanup(func() { ResetState(); SetIsTerminal(origIsTerminal) })
	SetOutputFields([]string{"uid", "nonexistent"})
	SetIsTerminal(false)

	err := PrintOutput(fieldsTestData(), "table", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field")
	assert.Contains(t, err.Error(), "nonexistent")
	assert.Contains(t, err.Error(), "available fields")
}

func TestSetOutputFields_Nil_RestoresAll(t *testing.T) {
	origIsTerminal := isTerminalCached.Load()
	t.Cleanup(func() { ResetState(); SetIsTerminal(origIsTerminal) })
	SetIsTerminal(false)

	SetOutputFields([]string{"uid"})
	out1 := CaptureOutput(func() {
		_ = PrintOutput(fieldsTestData(), "table", true)
	})

	SetOutputFields(nil)
	out2 := CaptureOutput(func() {
		_ = PrintOutput(fieldsTestData(), "table", true)
	})

	assert.NotContains(t, out1, "STATUS")
	assert.Contains(t, out2, "STATUS")
}

// ---- --query flag tests ----

func TestSetOutputQuery_FilterArray(t *testing.T) {
	t.Cleanup(func() { ResetState() })
	SetOutputQuery("[?status=='LIVE']")

	out, err := CaptureOutputErr(func() error {
		return PrintOutput(fieldsTestData(), "json", true)
	})
	assert.NoError(t, err)

	var rows []map[string]interface{}
	assert.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out)), &rows))
	assert.Len(t, rows, 1)
	assert.Equal(t, "aaa-111", rows[0]["uid"])
	assert.Equal(t, "LIVE", rows[0]["status"])
}

func TestSetOutputQuery_ExtractField(t *testing.T) {
	t.Cleanup(func() { ResetState() })
	SetOutputQuery("[*].name")

	out, err := CaptureOutputErr(func() error {
		return PrintOutput(fieldsTestData(), "json", true)
	})
	assert.NoError(t, err)

	var names []string
	assert.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out)), &names))
	assert.Equal(t, []string{"Port A", "Port B"}, names)
}

func TestSetOutputQuery_WithFields(t *testing.T) {
	t.Cleanup(func() { ResetState() })
	SetOutputFields([]string{"uid", "name"})
	SetOutputQuery("[*].uid")

	out, err := CaptureOutputErr(func() error {
		return PrintOutput(fieldsTestData(), "json", true)
	})
	assert.NoError(t, err)

	var uids []string
	assert.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out)), &uids))
	assert.Equal(t, []string{"aaa-111", "bbb-222"}, uids)
}

func TestSetOutputQuery_InvalidQuery(t *testing.T) {
	t.Cleanup(func() { ResetState() })
	SetOutputQuery("INVALID[[")

	err := PrintOutput(fieldsTestData(), "json", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JMESPath query")
}

func TestSetOutputQuery_EmptyData(t *testing.T) {
	t.Cleanup(func() { ResetState() })
	SetOutputQuery("[*].uid")

	out, err := CaptureOutputErr(func() error {
		return PrintOutput([]fieldsTestStruct{}, "json", true)
	})
	assert.NoError(t, err)
	// JMESPath [*].uid on an empty array returns an empty array, not null
	assert.Contains(t, strings.TrimSpace(out), "[]")
}

func TestSetOutputQuery_Reset(t *testing.T) {
	SetOutputQuery("[*].uid")
	SetOutputQuery("") // reset

	out, err := CaptureOutputErr(func() error {
		return PrintOutput(fieldsTestData(), "json", true)
	})
	assert.NoError(t, err)

	var rows []map[string]interface{}
	assert.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out)), &rows))
	assert.Len(t, rows, 2)
	// Full output restored — all fields present
	_, hasUID := rows[0]["uid"]
	_, hasStatus := rows[0]["status"]
	assert.True(t, hasUID)
	assert.True(t, hasStatus)
}

func TestApplyJMESPath_MarshalError(t *testing.T) {
	// Channels cannot be marshalled to JSON — exercises the marshal error path.
	_, err := applyJMESPath("[*]", make(chan int))
	assert.Error(t, err)
}

// ---- --no-header flag ----

func TestNoHeaderTableSuppressesHeader(t *testing.T) {
	SetNoHeader(true)
	t.Cleanup(func() { ResetState() })

	data := []SimpleStruct{{ID: 1, Name: "alpha", Active: true}}
	out := CaptureOutput(func() {
		err := PrintOutput(data, "table", true)
		assert.NoError(t, err)
	})

	assert.NotContains(t, out, "ID", "header row should be suppressed")
	assert.Contains(t, out, "alpha", "data row should still appear")
}

func TestNoHeaderTableWithHeaderEnabled(t *testing.T) {
	SetNoHeader(false)
	t.Cleanup(func() { ResetState() })

	data := []SimpleStruct{{ID: 1, Name: "beta", Active: false}}
	out := CaptureOutput(func() {
		err := PrintOutput(data, "table", true)
		assert.NoError(t, err)
	})

	assert.Contains(t, out, "ID", "header row should appear when no-header is false")
}

func TestNoHeaderCSVSuppressesHeader(t *testing.T) {
	SetNoHeader(true)
	t.Cleanup(func() { ResetState() })

	data := []SimpleStruct{{ID: 42, Name: "gamma", Active: true}}
	out := CaptureOutput(func() {
		err := PrintOutput(data, "csv", true)
		assert.NoError(t, err)
	})

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	assert.Len(t, lines, 1, "only the data row should appear, not a header row")
	assert.Contains(t, lines[0], "42")
}

func TestNoHeaderCSVWithHeaderEnabled(t *testing.T) {
	SetNoHeader(false)
	t.Cleanup(func() { ResetState() })

	data := []SimpleStruct{{ID: 7, Name: "delta", Active: false}}
	out := CaptureOutput(func() {
		err := PrintOutput(data, "csv", true)
		assert.NoError(t, err)
	})

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	assert.GreaterOrEqual(t, len(lines), 2, "header + data row should both appear")
	assert.Contains(t, lines[0], "id", "first line should be the header")
}

func TestNoHeaderDoesNotAffectJSON(t *testing.T) {
	SetNoHeader(true)
	t.Cleanup(func() { ResetState() })

	data := []SimpleStruct{{ID: 3, Name: "epsilon", Active: true}}
	out := CaptureOutput(func() {
		err := PrintOutput(data, "json", true)
		assert.NoError(t, err)
	})

	assert.Contains(t, out, `"name"`, "JSON should include field names regardless of --no-header")
	assert.Contains(t, out, "epsilon")
}

func TestPrintGoTemplate_FieldExtraction(t *testing.T) {
	SetTemplateString(`{{range .}}{{.Name}}{{"\n"}}{{end}}`)
	t.Cleanup(func() { ResetState() })

	data := []SimpleStruct{{ID: 1, Name: "alpha", Active: true}, {ID: 2, Name: "beta", Active: false}}
	out := CaptureOutput(func() {
		err := PrintOutput(data, "go-template", true)
		assert.NoError(t, err)
	})

	assert.Contains(t, out, "alpha")
	assert.Contains(t, out, "beta")
}

func TestPrintGoTemplate_SingleItem(t *testing.T) {
	SetTemplateString(`{{(index . 0).Name}}`)
	t.Cleanup(func() { ResetState() })

	data := []SimpleStruct{{ID: 1, Name: "gamma", Active: true}}
	out := CaptureOutput(func() {
		err := PrintOutput(data, "go-template", true)
		assert.NoError(t, err)
	})

	assert.Equal(t, "gamma", strings.TrimSpace(out))
}

func TestPrintGoTemplate_FuncMap(t *testing.T) {
	SetTemplateString(`{{range .}}{{upper .Name}}{{"\n"}}{{end}}`)
	t.Cleanup(func() { ResetState() })

	data := []SimpleStruct{{ID: 1, Name: "delta"}}
	out := CaptureOutput(func() {
		err := PrintOutput(data, "go-template", true)
		assert.NoError(t, err)
	})

	assert.Contains(t, out, "DELTA")
}

func TestPrintGoTemplate_InvalidTemplate(t *testing.T) {
	SetTemplateString(`{{invalid`)
	t.Cleanup(func() { ResetState() })

	data := []SimpleStruct{{ID: 1, Name: "test"}}
	var err error
	CaptureOutput(func() {
		err = PrintOutput(data, "go-template", true)
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid template")
}

func TestPrintGoTemplate_Count(t *testing.T) {
	SetTemplateString(`{{len .}}`)
	t.Cleanup(func() { ResetState() })

	data := []SimpleStruct{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}, {ID: 3, Name: "c"}}
	out := CaptureOutput(func() {
		err := PrintOutput(data, "go-template", true)
		assert.NoError(t, err)
	})

	assert.Equal(t, "3", strings.TrimSpace(out))
}

func TestPrintGoTemplate_JSONFuncMap(t *testing.T) {
	SetTemplateString(`{{range .}}{{json .}}{{"\n"}}{{end}}`)
	t.Cleanup(func() { ResetState() })

	data := []SimpleStruct{{ID: 1, Name: "epsilon", Active: true}}
	out := CaptureOutput(func() {
		err := PrintOutput(data, "go-template", true)
		assert.NoError(t, err)
	})

	assert.Contains(t, out, `"epsilon"`)
}

func TestResetState_ClearsTemplateString(t *testing.T) {
	SetTemplateString("{{.}}")
	ResetState()
	assert.Equal(t, "", GetTemplateString())
}

func TestGetOutputFormat(t *testing.T) {
	orig := GetOutputFormat()
	t.Cleanup(func() { SetOutputFormat(orig) })
	SetOutputFormat("json")
	assert.Equal(t, "json", GetOutputFormat())
}

func TestApplyOutputConfigConcurrent(t *testing.T) {
	orig := GetOutputConfig()
	t.Cleanup(func() { ApplyOutputConfig(orig) })

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cfg := GetOutputConfig()
			cfg.Format = "json"
			ApplyOutputConfig(cfg)
			_ = GetOutputConfig()
		}()
	}
	wg.Wait()
}
