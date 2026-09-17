//go:build !wasm

package output

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
)

func printGoTemplate[T OutputFields](data []T, opts printOptions) error {
	tmplStr := opts.template
	funcMap := template.FuncMap{
		"join":  strings.Join,
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"trim":  strings.TrimSpace,
		"json": func(v any) (string, error) {
			b, err := json.Marshal(v)
			return string(b), err
		},
	}
	tmpl, err := template.New("output").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("invalid template: %w", err)
	}
	return tmpl.Execute(os.Stdout, data)
}

func printJSON[T OutputFields](data []T, opts printOptions) error {
	if data == nil {
		data = []T{}
	}

	toEncode, err := prepareJSONData(data, opts)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(toEncode)
}

func calculateColumnWidths(rows [][]string) []int {
	if len(rows) == 0 {
		return nil
	}
	colCount := len(rows[0])
	colWidths := make([]int, colCount)
	for _, row := range rows {
		for i, val := range row {
			if i < colCount && len(val) > colWidths[i] {
				colWidths[i] = len(val)
			}
		}
	}
	return colWidths
}

func printCSV[T OutputFields](data []T, opts printOptions) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	headers, jsonNames, fieldIndices, err := extractCSVFieldInfo(data)
	if err != nil {
		return err
	}
	if csvFields := opts.fields; len(csvFields) > 0 {
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
		if isNilOrInvalid(item) {
			continue
		}
		values := extractRowData(item, fieldIndices)
		if err := w.Write(values); err != nil {
			return err
		}
	}
	return nil
}

func printXML[T OutputFields](data []T, opts printOptions) error {
	if data == nil {
		data = []T{}
	}

	// Use getStructTypeInfo for field metadata — jsonNames are used as XML element
	// names, headers are used for --fields alias matching.
	headers, jsonNames, fieldIndices, err := getStructTypeInfo(data)
	if err != nil {
		return err
	}
	if len(jsonNames) == 0 {
		fmt.Fprint(os.Stdout, xml.Header+"<items></items>\n")
		return nil
	}

	if xmlFields := opts.fields; len(xmlFields) > 0 {
		_, jsonNames, fieldIndices, err = filterByFields(headers, jsonNames, fieldIndices, xmlFields)
		if err != nil {
			return err
		}
	}

	encoder := xml.NewEncoder(os.Stdout)
	encoder.Indent("", "  ")

	fmt.Fprint(os.Stdout, xml.Header)
	start := xml.StartElement{Name: xml.Name{Local: "items"}}
	if err := encoder.EncodeToken(start); err != nil {
		return err
	}

	for _, item := range data {
		if isNilOrInvalid(item) {
			continue
		}
		values := extractRowData(item, fieldIndices)
		if values == nil {
			continue
		}

		itemStart := xml.StartElement{Name: xml.Name{Local: "item"}}
		if err := encoder.EncodeToken(itemStart); err != nil {
			return err
		}

		for i, name := range jsonNames {
			elemStart := xml.StartElement{Name: xml.Name{Local: name}}
			if err := encoder.EncodeToken(elemStart); err != nil {
				return err
			}
			if err := encoder.EncodeToken(xml.CharData(values[i])); err != nil {
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
	fmt.Fprintln(os.Stdout)
	return nil
}

// createTempFile is a variable so tests can replace it to simulate failures.
var createTempFile = func() (*os.File, error) {
	return os.CreateTemp("", "capture-stdout-*")
}

// setStderr reassigns os.Stderr under stdErrStreamMu (declared in common.go),
// synchronizing the change against concurrent spinner writes. Held only for
// the instant of the reassignment, never across the caller's function.
func setStderr(f *os.File) {
	stdErrStreamMu.Lock()
	os.Stderr = f
	stdErrStreamMu.Unlock()
}

// CaptureOutput runs f and returns everything it writes to stdout and stderr
// combined. Status messages route to stderr and data to stdout, so a test that
// wants all user-facing output captures both. Use CaptureStdout when asserting
// that the data stream alone is clean.
// Uses a temporary file instead of an OS pipe to avoid deadlocking when
// f() produces more output than the pipe buffer can hold.
// Must not be called reentrantly (the global stdoutMu is not reentrant).
func CaptureOutput(f func()) string {
	stdoutMu.Lock()
	defer stdoutMu.Unlock()

	oldOut, oldErr := os.Stdout, os.Stderr
	tmp, err := createTempFile()
	if err != nil {
		f()
		return ""
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	os.Stdout = tmp
	setStderr(tmp)
	defer func() {
		os.Stdout = oldOut
		setStderr(oldErr)
	}()

	f()

	// Read back captured output.
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return ""
	}
	data, _ := io.ReadAll(tmp)
	return string(data)
}

// CaptureStdout runs f and returns only what it writes to stdout. Use this when
// a test needs the clean data stream (for example parsing exported JSON/CSV)
// without the status messages that route to stderr.
func CaptureStdout(f func()) string {
	stdoutMu.Lock()
	defer stdoutMu.Unlock()

	old := os.Stdout
	tmp, err := createTempFile()
	if err != nil {
		f()
		return ""
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	os.Stdout = tmp
	defer func() { os.Stdout = old }()

	f()

	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return ""
	}
	data, _ := io.ReadAll(tmp)
	return string(data)
}

// CaptureOutputErr is the error-returning counterpart to CaptureOutput: it
// captures both stdout and stderr.
func CaptureOutputErr(f func() error) (string, error) {
	stdoutMu.Lock()
	defer stdoutMu.Unlock()

	oldOut, oldErr := os.Stdout, os.Stderr
	tmp, err := createTempFile()
	if err != nil {
		runErr := f()
		return "", runErr
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	os.Stdout = tmp
	setStderr(tmp)
	defer func() {
		os.Stdout = oldOut
		setStderr(oldErr)
	}()

	runErr := f()

	// Read back captured output.
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	data, readErr := io.ReadAll(tmp)
	if readErr != nil {
		return "", readErr
	}
	// Return captured output even when f returned an error so callers can
	// inspect partial output for diagnostics.
	return string(data), runErr
}
