package output

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/jmespath/go-jmespath"
)

// stdoutMu protects os.Stdout during CaptureOutput/CaptureOutputErr calls.
// Both native and WASM builds use this mutex.
var stdoutMu sync.Mutex

// stdErrStreamMu guards os.Stderr reassignment (CaptureOutput/CaptureOutputErr,
// via setStderr in output.go) against concurrent stderr writes (the spinner's
// animation frames and status lines, via writeSpinnerLine/writeSpinnerLinef in
// messages.go). It is held only for the instant of a single read-and-write or
// a single reassignment, never across an entire captured callback, so a
// callback that itself starts and stops a spinner cannot deadlock against it.
var stdErrStreamMu sync.RWMutex

// OutputConfig holds all user-facing output configuration as a single struct.
// Use ApplyOutputConfig to write and GetOutputConfig to read atomically.
type OutputConfig struct {
	Fields    []string // nil = show all
	Query     string   // JMESPath; "" = disabled
	NoHeader  bool
	Template  string // Go template; "" = disabled
	NoPager   bool
	Format    string // "table"|"json"|"csv"|"xml"|"go-template"
	Verbosity string // "normal"|"quiet"|"verbose"
}

// defaultOutputConfig returns the baseline configuration used at startup and by ResetState.
func defaultOutputConfig() OutputConfig {
	return OutputConfig{Format: "table", Verbosity: "normal"}
}

var (
	outputCfg   = defaultOutputConfig()
	outputCfgMu sync.RWMutex
)

// ApplyOutputConfig atomically replaces the entire output configuration.
// Fields is deep-copied before storing so that mutations to the caller's slice
// cannot affect the stored configuration after this call returns.
func ApplyOutputConfig(cfg OutputConfig) {
	if cfg.Fields != nil {
		cp := make([]string, len(cfg.Fields))
		copy(cp, cfg.Fields)
		cfg.Fields = cp
	}
	outputCfgMu.Lock()
	defer outputCfgMu.Unlock()
	outputCfg = cfg
}

// GetOutputConfig returns a snapshot of the current output configuration.
// Fields is deep-copied so that mutations to the returned slice cannot affect
// the stored configuration.
func GetOutputConfig() OutputConfig {
	outputCfgMu.RLock()
	defer outputCfgMu.RUnlock()
	cp := outputCfg
	if outputCfg.Fields != nil {
		cp.Fields = make([]string, len(outputCfg.Fields))
		copy(cp.Fields, outputCfg.Fields)
	}
	return cp
}

// updateOutputConfig holds the write lock and calls fn to mutate the stored
// config in place. Use this in single-field Set* shims to avoid the
// read-modify-write window that snapshot+replace creates.
func updateOutputConfig(fn func(*OutputConfig)) {
	outputCfgMu.Lock()
	defer outputCfgMu.Unlock()
	fn(&outputCfg)
}

// SetOutputFields sets the field filter applied by all PrintOutput calls.
// Pass nil to restore full output (all fields). The slice is deep-copied so
// subsequent mutations to fields do not affect the stored config.
func SetOutputFields(fields []string) {
	var cp []string
	if fields != nil {
		cp = make([]string, len(fields))
		copy(cp, fields)
	}
	updateOutputConfig(func(c *OutputConfig) { c.Fields = cp })
}

// SetOutputQuery sets the JMESPath query applied by printJSON. Pass "" to disable.
func SetOutputQuery(query string) {
	updateOutputConfig(func(c *OutputConfig) { c.Query = query })
}

// SetNoHeader sets whether table and CSV output should suppress column headers.
func SetNoHeader(v bool) {
	updateOutputConfig(func(c *OutputConfig) { c.NoHeader = v })
}

// SetTemplateString sets the Go template string applied by printGoTemplate.
// Pass "" to disable.
func SetTemplateString(s string) {
	updateOutputConfig(func(c *OutputConfig) { c.Template = s })
}

// GetTemplateString returns the current Go template string.
func GetTemplateString() string {
	outputCfgMu.RLock()
	defer outputCfgMu.RUnlock()
	return outputCfg.Template
}

// errorBody and errorEnvelope are the JSON error envelope types shared by
// PrintErrorJSON across native and WASM builds.
type errorBody struct {
	Code    int    `json:"code"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

// errorEmitted latches once a user-facing error message has been printed for
// the current command invocation. The RunE wrappers reset it before running an
// action and consult it afterward so they print their own error only when the
// action did not already surface one, avoiding a double-printed error.
var (
	errorEmitted   bool
	errorEmittedMu sync.Mutex
)

// markErrorEmitted records that an error message was shown to the user.
func markErrorEmitted() {
	errorEmittedMu.Lock()
	errorEmitted = true
	errorEmittedMu.Unlock()
}

// ErrorEmitted reports whether an error message has been printed since the last
// ResetErrorEmitted call.
func ErrorEmitted() bool {
	errorEmittedMu.Lock()
	defer errorEmittedMu.Unlock()
	return errorEmitted
}

// ResetErrorEmitted clears the error-emitted latch. Called at the start of each
// RunE wrapper invocation.
func ResetErrorEmitted() {
	errorEmittedMu.Lock()
	errorEmitted = false
	errorEmittedMu.Unlock()
}

// ResetState clears all output configuration back to defaults.
// Intended for the WASM entry point to prevent state bleed between invocations.
func ResetState() {
	ApplyOutputConfig(defaultOutputConfig())
	ResetErrorEmitted()
}

// printOptions is the per-call snapshot of the field/query/header/template
// settings that the data-output path consults. PrintOutput reads the config
// once into a printOptions and threads it down through the formatters, so the
// individual formatters never reach into global state (and never take a lock)
// mid-render.
type printOptions struct {
	fields   []string
	query    string
	noHeader bool
	template string
}

// currentPrintOptions snapshots the output config into a printOptions with a
// single locked read. Called once per PrintOutput invocation on the calling
// goroutine.
func currentPrintOptions() printOptions {
	cfg := GetOutputConfig()
	return printOptions{
		fields:   cfg.Fields,
		query:    cfg.Query,
		noHeader: cfg.NoHeader,
		template: cfg.Template,
	}
}

// applyJMESPath applies a JMESPath query to v and returns the result.
// v must be a JSON-compatible value (e.g. []T or []map[string]interface{}).
// The marshal→unmarshal round-trip is intentional: go-jmespath operates on an
// interface{} tree, so typed Go structs must be converted first. This doubles
// memory momentarily but is necessary for correct JMESPath evaluation.
func applyJMESPath(query string, v interface{}) (interface{}, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var parsed interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	result, err := jmespath.Search(query, parsed)
	if err != nil {
		return nil, fmt.Errorf("invalid JMESPath query %q: %w", query, err)
	}
	return result, nil
}

// prepareJSONData applies --fields filtering and --query (JMESPath) to data,
// returning the value ready for JSON encoding. This shared logic is used by
// both native and WASM printJSON implementations.
func prepareJSONData[T OutputFields](data []T, opts printOptions) (interface{}, error) {
	fields := opts.fields
	query := opts.query

	var toEncode interface{}
	if len(fields) > 0 {
		headers, jsonNames, indices, err := getStructTypeInfo(data)
		if err != nil {
			return nil, err
		}
		_, jsonNames, indices, err = filterByFields(headers, jsonNames, indices, fields)
		if err != nil {
			return nil, err
		}
		rows := make([]interface{}, 0, len(data))
		for _, item := range data {
			v := reflect.ValueOf(item)
			if v.Kind() == reflect.Pointer {
				if v.IsNil() {
					rows = append(rows, nil)
					continue
				}
				v = v.Elem()
			}
			if !v.IsValid() || v.Kind() != reflect.Struct {
				rows = append(rows, nil)
				continue
			}
			m := make(map[string]interface{}, len(indices))
			for i, idx := range indices {
				if idx >= v.NumField() {
					continue
				}
				m[jsonNames[i]] = v.Field(idx).Interface()
			}
			rows = append(rows, m)
		}
		toEncode = rows
	} else {
		toEncode = data
	}

	if query != "" {
		var err error
		toEncode, err = applyJMESPath(query, toEncode)
		if err != nil {
			return nil, err
		}
	}

	return toEncode, nil
}

// Output is a marker interface embedded by output structs.
type Output interface {
	isOutput()
}

// OutputFields is a constraint for types that can be output
type OutputFields interface {
	any
}

// ResourceTag represents a key-value tag pair
type ResourceTag struct {
	Key   string `json:"key" header:"Key"`
	Value string `json:"value" header:"Value"`
}

// PrintOutput prints data in the specified format
func PrintOutput[T OutputFields](data []T, format string, noColor bool) error {
	validFormats := map[string]bool{
		"table":       true,
		"json":        true,
		"csv":         true,
		"xml":         true,
		"go-template": true,
	}
	if !validFormats[format] {
		return fmt.Errorf("invalid output format: %s", format)
	}
	opts := currentPrintOptions()
	switch format {
	case "json":
		return printJSON(data, opts)
	case "csv":
		return printCSV(data, opts)
	case "xml":
		return printXML(data, opts)
	case "go-template":
		return printGoTemplate(data, opts)
	default:
		return RunWithPager(func() error {
			return printTable(data, noColor, opts)
		})
	}
}

// PrintTableToWriter renders data as a table directly to w with no global side
// effects: it does not page, write to os.Stdout, or set the WASM table global.
// Use it to compose several tables into one writer (e.g. the status dashboard)
// so the full output is captured in order, including under the WASM transport
// where PrintOutput("table") would otherwise only retain the last section.
// Touching no shared globals, it needs no output-package locking; the caller
// owns w.
func PrintTableToWriter[T OutputFields](w io.Writer, data []T, noColor bool) error {
	return printTableToWriter(w, data, noColor, currentPrintOptions())
}

// getStructTypeInfo extracts header names, json names, and field indices from a struct type.
// jsonNames are the json tag values (used for --fields matching); headers are the display names.
func getStructTypeInfo[T OutputFields](data []T) (headers, jsonNames []string, fieldIndices []int, err error) {
	var sample T
	if len(data) > 0 {
		sample = data[0]
	}
	sampleVal := reflect.ValueOf(sample)
	if !sampleVal.IsValid() {
		return nil, nil, nil, nil
	}
	itemType := sampleVal.Type()
	if itemType.Kind() == reflect.Pointer {
		if sampleVal.IsNil() {
			if itemType.Elem().Kind() != reflect.Struct {
				return nil, nil, nil, nil
			}
			itemType = itemType.Elem()
		} else {
			sampleVal = sampleVal.Elem()
			itemType = sampleVal.Type()
		}
	}
	if itemType.Kind() != reflect.Struct {
		return nil, nil, nil, nil
	}
	headers, jsonNames, fieldIndices = extractFieldInfo(itemType)
	return headers, jsonNames, fieldIndices, nil
}

// extractFieldInfo extracts field information from a struct type.
// Returns headers (display names), jsonNames (json tag names for --fields matching), and field indices.
func extractFieldInfo(itemType reflect.Type) (headers, jsonNames []string, fieldIndices []int) {
	for i := 0; i < itemType.NumField(); i++ {
		field := itemType.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if !isOutputCompatibleType(field.Type) {
			continue
		}
		headerTag := field.Tag.Get("header")
		if headerTag == "-" {
			continue
		}
		if headerTag == "" {
			headerTag = field.Tag.Get("csv")
			if headerTag == "-" {
				continue
			}
		}
		if headerTag == "" {
			headerTag = field.Tag.Get("output")
			if headerTag == "-" {
				continue
			}
		}
		if headerTag == "" {
			headerTag = strings.ToUpper(field.Name)
		}

		// Derive the json name for --fields matching.
		jsonName := field.Tag.Get("json")
		if idx := strings.Index(jsonName, ","); idx != -1 {
			jsonName = jsonName[:idx]
		}
		if jsonName == "" || jsonName == "-" {
			jsonName = strings.ToLower(field.Name)
		}

		headers = append(headers, headerTag)
		jsonNames = append(jsonNames, jsonName)
		fieldIndices = append(fieldIndices, i)
	}
	return headers, jsonNames, fieldIndices
}

// filterByFields filters (headers, jsonNames, indices) to only the user-selected fields.
// Matching is case-insensitive: json names are tried first, then header names.
// Duplicate selections are silently deduplicated. Returns an error listing available
// json names if any selected field is unknown.
func filterByFields(headers, jsonNames []string, indices []int, selected []string) ([]string, []string, []int, error) {
	if len(selected) == 0 {
		return headers, jsonNames, indices, nil
	}
	// Two separate maps avoids the collision that occurs when a header name
	// happens to match a json name of a different field.
	byJSON := make(map[string]int, len(jsonNames))
	for i, jn := range jsonNames {
		byJSON[strings.ToLower(jn)] = i
	}
	byHeader := make(map[string]int, len(headers))
	for i, h := range headers {
		byHeader[strings.ToLower(h)] = i
	}

	// Pre-build the available-fields string once so it is not rebuilt on every error.
	available := make([]string, 0, len(jsonNames))
	for j, jn := range jsonNames {
		if j < len(headers) && !strings.EqualFold(headers[j], jn) {
			available = append(available, fmt.Sprintf("%s (or %q)", jn, headers[j]))
		} else {
			available = append(available, jn)
		}
	}
	availableStr := strings.Join(available, ", ")

	seen := make(map[int]bool, len(selected))
	var outHeaders, outJSONNames []string
	var outIndices []int
	for _, sel := range selected {
		key := strings.ToLower(strings.TrimSpace(sel))
		// Prefer json name match; fall back to header display name.
		i, ok := byJSON[key]
		if !ok {
			i, ok = byHeader[key]
		}
		if !ok {
			return nil, nil, nil, fmt.Errorf("unknown field %q, available fields: %s", sel, availableStr)
		}
		if seen[i] {
			continue // deduplicate repeated field selections
		}
		seen[i] = true
		if len(headers) > 0 {
			outHeaders = append(outHeaders, headers[i])
		}
		outJSONNames = append(outJSONNames, jsonNames[i])
		outIndices = append(outIndices, indices[i])
	}
	return outHeaders, outJSONNames, outIndices, nil
}

// isOutputCompatibleType checks if a type can be output
func isOutputCompatibleType(t reflect.Type) bool {
	// Handle pointer types by checking the element type
	if t.Kind() == reflect.Pointer {
		return isOutputCompatibleType(t.Elem())
	}

	// Special case: time.Time is always compatible
	if t == reflect.TypeOf(time.Time{}) {
		return true
	}

	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		// Slices and arrays are compatible (we can serialize them)
		return true
	case reflect.Map:
		// Maps are compatible (we can serialize them)
		return true
	case reflect.Struct:
		// Structs are compatible (we can serialize them)
		return true
	case reflect.Chan, reflect.Func:
		// Channels and functions cannot be serialized
		return false
	default:
		// All primitive types are compatible
		return true
	}
}

// extractRowData extracts field values from a struct for table/CSV output
func extractRowData(item interface{}, fieldIndices []int) []string {
	itemVal := reflect.ValueOf(item)
	if itemVal.Kind() == reflect.Pointer {
		if itemVal.IsNil() {
			return make([]string, len(fieldIndices))
		}
		itemVal = itemVal.Elem()
	}
	if itemVal.Kind() != reflect.Struct {
		return nil
	}
	values := make([]string, len(fieldIndices))
	for i, idx := range fieldIndices {
		fieldVal := itemVal.Field(idx)
		values[i] = formatFieldValue(fieldVal)
	}
	return values
}

// formatFieldValue formats a field value for display
func formatFieldValue(fieldVal reflect.Value) string {
	if !fieldVal.IsValid() {
		return ""
	}
	if fieldVal.Kind() == reflect.Pointer {
		if fieldVal.IsNil() {
			return ""
		}
		fieldVal = fieldVal.Elem()
	}

	// Special handling for time.Time - format as ISO date
	if fieldVal.Type() == reflect.TypeOf(time.Time{}) {
		t, ok := fieldVal.Interface().(time.Time)
		if !ok || t.IsZero() {
			return ""
		}
		return t.Format("2006-01-02")
	}

	switch fieldVal.Kind() {
	case reflect.Slice, reflect.Array:
		if fieldVal.Len() == 0 {
			return ""
		}
		// For slices of primitives, try JSON serialization first
		if fieldVal.CanInterface() {
			val := fieldVal.Interface()
			if jsonBytes, err := json.Marshal(val); err == nil {
				return string(jsonBytes)
			}
		}
		// Fallback to comma-separated values
		var parts []string
		for i := 0; i < fieldVal.Len(); i++ {
			elem := fieldVal.Index(i)
			parts = append(parts, formatFieldValue(elem))
		}
		return strings.Join(parts, ", ")
	case reflect.Map:
		if fieldVal.Len() == 0 {
			return ""
		}
		// Serialize maps as JSON
		if fieldVal.CanInterface() {
			val := fieldVal.Interface()
			if jsonBytes, err := json.Marshal(val); err == nil {
				return string(jsonBytes)
			}
		}
		return fmt.Sprintf("%v", fieldVal.Interface())
	case reflect.Struct:
		// Serialize structs as JSON
		if fieldVal.CanInterface() {
			val := fieldVal.Interface()
			if jsonBytes, err := json.Marshal(val); err == nil {
				return string(jsonBytes)
			}
		}
		return fmt.Sprintf("%v", fieldVal.Interface())
	case reflect.Bool:
		if fieldVal.Bool() {
			return "true"
		}
		return "false"
	case reflect.String:
		return fieldVal.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", fieldVal.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", fieldVal.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%g", fieldVal.Float())
	default:
		if !fieldVal.CanInterface() {
			return ""
		}
		val := fieldVal.Interface()
		if val == nil {
			return ""
		}
		return fmt.Sprintf("%v", val)
	}
}
