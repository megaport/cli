//go:build js && wasm

package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
	"syscall/js"

	"github.com/megaport/megaport-cli/internal/wasm"
)

// wasmBufMu protects the global WASM output buffers from concurrent access.
var wasmBufMu sync.Mutex

// WasmJSONWriter is a global buffer for capturing JSON output in WASM
var WasmJSONWriter = &bytes.Buffer{}

// WasmCSVWriter is a global buffer for capturing CSV output in WASM
var WasmCSVWriter = &bytes.Buffer{}

// WasmXMLWriter is a global buffer for capturing XML output in WASM
var WasmXMLWriter = &bytes.Buffer{}

// printJSON is the WASM-specific implementation that properly captures JSON output
func printJSON[T OutputFields](data []T, opts printOptions) error {
	wasmBufMu.Lock()
	defer wasmBufMu.Unlock()
	WasmJSONWriter.Reset()

	if data == nil {
		data = []T{}
	}

	toEncode, err := prepareJSONData(data, opts)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(WasmJSONWriter)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(toEncode); err != nil {
		js.Global().Get("console").Call("error", "Failed to encode JSON:", err.Error())
		return err
	}

	// Sanitize before this leaves Go: wasmJSONOutput is read back by
	// GetCapturedOutput/GetCompletionOutput and written to xterm without
	// escaping, and a field value can carry an API-controlled control byte
	// that Go's JSON encoder does not escape (e.g. the C1 range). See
	// wasm.SanitizeTerminalOutput.
	jsonOutput := wasm.SanitizeTerminalOutput(WasmJSONWriter.String())
	fmt.Print(jsonOutput)
	js.Global().Set("wasmJSONOutput", jsonOutput)
	return nil
}

// printCSV is the WASM-specific implementation that properly captures CSV output
func printCSV[T OutputFields](data []T, opts printOptions) error {
	wasmBufMu.Lock()
	defer wasmBufMu.Unlock()
	WasmCSVWriter.Reset()

	w := csv.NewWriter(WasmCSVWriter)
	defer w.Flush()

	var sample T
	if len(data) > 0 {
		sample = data[0]
	}
	sampleVal := reflect.ValueOf(sample)
	if !sampleVal.IsValid() {
		return nil
	}
	t := sampleVal.Type()
	if t.Kind() == reflect.Pointer {
		if sampleVal.IsNil() {
			if t.Elem().Kind() != reflect.Struct {
				return nil
			}
			t = t.Elem()
		} else {
			sampleVal = sampleVal.Elem()
			t = sampleVal.Type()
		}
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	// Build headers, json names, and field indices — json names are needed for --fields matching.
	var headers []string
	var jsonNames []string
	var fieldIndices []int
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if !isOutputCompatibleType(field.Type) {
			continue
		}
		csvTag := field.Tag.Get("csv")
		if csvTag == "-" {
			continue
		}
		jsonTag := field.Tag.Get("json")
		if csvTag == "" {
			if jsonTag == "" || jsonTag == "-" {
				continue
			}
			csvTag = jsonTag
		}
		// Derive the json name for --fields matching (strip options like omitempty).
		jn := jsonTag
		if idx := strings.Index(jn, ","); idx != -1 {
			jn = jn[:idx]
		}
		if jn == "" || jn == "-" {
			jn = strings.ToLower(field.Name)
		}
		headers = append(headers, csvTag)
		jsonNames = append(jsonNames, jn)
		fieldIndices = append(fieldIndices, i)
	}

	// Apply --fields filter if set.
	if csvFields := opts.fields; len(csvFields) > 0 {
		var err error
		headers, _, fieldIndices, err = filterByFields(headers, jsonNames, fieldIndices, csvFields)
		if err != nil {
			return err
		}
	}
	if len(headers) == 0 {
		return nil
	}

	if !opts.noHeader {
		if err := w.Write(headers); err != nil {
			return err
		}
	}

	for _, item := range data {
		v := reflect.ValueOf(item)
		if !v.IsValid() {
			continue
		}
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				continue
			}
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			continue
		}
		row := make([]string, 0, len(fieldIndices))
		for _, idx := range fieldIndices {
			if idx >= v.NumField() {
				row = append(row, "")
				continue
			}
			fieldVal := v.Field(idx)
			if !fieldVal.IsValid() || (fieldVal.Kind() == reflect.Pointer && fieldVal.IsNil()) {
				row = append(row, "")
				continue
			}
			row = append(row, fmt.Sprintf("%v", formatFieldValue(fieldVal)))
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}

	w.Flush()

	// See the sanitize comment in printJSON above; the same applies to CSV.
	csvOutput := wasm.SanitizeTerminalOutput(WasmCSVWriter.String())
	fmt.Print(csvOutput)
	js.Global().Set("wasmCSVOutput", csvOutput)
	return nil
}

// printXML is the WASM-specific implementation that properly captures XML output
func printXML[T OutputFields](data []T, opts printOptions) error {
	wasmBufMu.Lock()
	defer wasmBufMu.Unlock()
	WasmXMLWriter.Reset()

	if data == nil {
		data = []T{}
	}

	writeEmpty := func() {
		WasmXMLWriter.WriteString(xml.Header + "<items></items>\n")
		xmlOutput := WasmXMLWriter.String()
		fmt.Print(xmlOutput)
		js.Global().Set("wasmXMLOutput", xmlOutput)
	}

	var sample T
	if len(data) > 0 {
		sample = data[0]
	}
	sampleVal := reflect.ValueOf(sample)
	if !sampleVal.IsValid() {
		writeEmpty()
		return nil
	}
	t := sampleVal.Type()
	if t.Kind() == reflect.Pointer {
		if sampleVal.IsNil() {
			if t.Elem().Kind() != reflect.Struct {
				writeEmpty()
				return nil
			}
			t = t.Elem()
		} else {
			sampleVal = sampleVal.Elem()
			t = sampleVal.Type()
		}
	}
	if t.Kind() != reflect.Struct {
		writeEmpty()
		return nil
	}

	type xmlField struct {
		name        string // json tag name (used as XML element name)
		displayName string // header tag (used for --fields alias matching)
		index       int
	}
	var fields []xmlField
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if !isOutputCompatibleType(field.Type) {
			continue
		}
		name := field.Tag.Get("json")
		if name == "-" {
			continue
		}
		if name == "" {
			name = field.Tag.Get("csv")
			if name == "-" {
				continue
			}
		}
		if name == "" {
			name = strings.ToLower(field.Name)
		}
		if idx := strings.Index(name, ","); idx != -1 {
			name = name[:idx]
		}
		displayName := field.Tag.Get("header")
		if displayName == "" || displayName == "-" {
			displayName = name
		}
		fields = append(fields, xmlField{name: name, displayName: displayName, index: i})
	}

	// Apply --fields filter if set.
	if xmlFields := opts.fields; len(xmlFields) > 0 {
		xmlHeaders := make([]string, len(fields))
		xmlJSONNames := make([]string, len(fields))
		xmlIndices := make([]int, len(fields))
		for i, f := range fields {
			xmlHeaders[i] = f.displayName
			xmlJSONNames[i] = f.name
			xmlIndices[i] = f.index
		}
		_, _, xmlIndices, err := filterByFields(xmlHeaders, xmlJSONNames, xmlIndices, xmlFields)
		if err != nil {
			return err
		}
		nameByIndex := make(map[int]string, len(fields))
		for _, f := range fields {
			nameByIndex[f.index] = f.name
		}
		filtered := make([]xmlField, len(xmlIndices))
		for i, idx := range xmlIndices {
			filtered[i] = xmlField{name: nameByIndex[idx], index: idx}
		}
		fields = filtered
	}

	fieldNames := make([]string, len(fields))
	for i, f := range fields {
		fieldNames[i] = f.name
	}
	xmlNames := sanitizeXMLElementNames(fieldNames)

	encoder := xml.NewEncoder(WasmXMLWriter)
	encoder.Indent("", "  ")

	WasmXMLWriter.WriteString(xml.Header)
	start := xml.StartElement{Name: xml.Name{Local: "items"}}
	if err := encoder.EncodeToken(start); err != nil {
		return err
	}

	for _, item := range data {
		v := reflect.ValueOf(item)
		if !v.IsValid() {
			continue
		}
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				continue
			}
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			continue
		}

		itemStart := xml.StartElement{Name: xml.Name{Local: "item"}}
		if err := encoder.EncodeToken(itemStart); err != nil {
			return err
		}

		for i, f := range fields {
			fieldVal := v.Field(f.index)
			valueStr := formatFieldValue(fieldVal)

			elemStart := xml.StartElement{Name: xml.Name{Local: xmlNames[i]}}
			if err := encoder.EncodeToken(elemStart); err != nil {
				return err
			}
			if err := encoder.EncodeToken(xml.CharData(valueStr)); err != nil {
				return err
			}
			if err := encoder.EncodeToken(elemStart.End()); err != nil {
				return err
			}
		}

		if err := encoder.EncodeToken(itemStart.End()); err != nil {
			return err
		}
	}

	if err := encoder.EncodeToken(start.End()); err != nil {
		return err
	}
	if err := encoder.Flush(); err != nil {
		return err
	}
	WasmXMLWriter.WriteString("\n")

	// See the sanitize comment in printJSON above; the same applies to XML.
	xmlOutput := wasm.SanitizeTerminalOutput(WasmXMLWriter.String())
	fmt.Print(xmlOutput)
	js.Global().Set("wasmXMLOutput", xmlOutput)
	return nil
}

// printGoTemplate is not supported in the WASM build.
// The error message begins with "invalid output format" so classifyError maps it to exitcodes.Usage.
func printGoTemplate[T OutputFields](_ []T, _ printOptions) error {
	return fmt.Errorf("invalid output format: go-template is not supported in the browser version")
}

// CaptureOutput runs a function and captures its stdout output.
// Must not be called reentrantly (the global stdoutMu is not reentrant).
func CaptureOutput(f func()) string {
	stdoutMu.Lock()
	defer stdoutMu.Unlock()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		f()
		return ""
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	defer r.Close()
	defer w.Close()
	f()
	w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

// CaptureStdout mirrors the native helper. In WASM all output already flows
// through a single buffer, so there is no separate data stream to isolate.
func CaptureStdout(f func()) string {
	return CaptureOutput(f)
}

// CaptureOutputErr runs a function and captures its stdout output, also returning any error.
// Must not be called reentrantly (the global stdoutMu is not reentrant).
func CaptureOutputErr(f func() error) (string, error) {
	stdoutMu.Lock()
	defer stdoutMu.Unlock()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		runErr := f()
		return "", runErr
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	runErr := f()

	// Close the write end before reading so io.Copy sees EOF rather than blocking.
	w.Close()

	var buf strings.Builder
	io.Copy(&buf, r) //nolint:errcheck // reading from an in-process pipe is always safe
	r.Close()

	// Return captured output even when f returned an error so callers can
	// inspect partial output for diagnostics.
	return buf.String(), runErr
}
