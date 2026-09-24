package output

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

// Spinner style constants.
const (
	SpinnerStyleDefault = "default"
	SpinnerStyleWASM    = "wasm"
	SpinnerStyleFancy   = "fancy"
)

// ansiColorRe is a pre-compiled regex for stripping ANSI color codes.
var ansiColorRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

// spinnerColors is a pre-allocated slice of color functions for spinner animation.
// Allocated once to avoid per-frame allocation in the spinner goroutine loop.
var spinnerColors = []func(...interface{}) string{
	color.New(color.FgHiCyan, color.Bold).SprintFunc(),
	color.New(color.FgHiBlue, color.Bold).SprintFunc(),
	color.New(color.FgHiMagenta, color.Bold).SprintFunc(),
	color.New(color.FgHiGreen, color.Bold).SprintFunc(),
}

// SetVerbosity sets the global verbosity level ("normal", "quiet", or "verbose").
func SetVerbosity(level string) {
	updateOutputConfig(func(c *OutputConfig) { c.Verbosity = level })
}

// IsQuiet returns true when quiet mode is active.
// In quiet mode, informational messages and spinners are suppressed.
func IsQuiet() bool {
	outputCfgMu.RLock()
	defer outputCfgMu.RUnlock()
	return outputCfg.Verbosity == "quiet"
}

// IsVerbose returns true when verbose mode is active.
func IsVerbose() bool {
	outputCfgMu.RLock()
	defer outputCfgMu.RUnlock()
	return outputCfg.Verbosity == "verbose"
}

// newNoOpSpinner returns a spinner that is already stopped.
// Safe to call Start(), Stop(), and StopWithSuccess() on.
// noColor is forwarded so StopWithSuccess respects the --no-color flag.
func newNoOpSpinner(noColor bool) *Spinner {
	return &Spinner{
		stop:         make(chan bool, 1),
		stopped:      true,
		outputFormat: GetOutputFormat(),
		noColor:      noColor,
	}
}

func SetOutputFormat(format string) {
	updateOutputConfig(func(c *OutputConfig) { c.Format = format })
}

// GetOutputFormat returns the currently active output format (e.g. "table", "json").
func GetOutputFormat() string {
	outputCfgMu.RLock()
	defer outputCfgMu.RUnlock()
	return outputCfg.Format
}

// shouldSuppressSpinner returns true when spinner output should be suppressed
// to avoid corrupting machine-readable output formats (csv, xml, json, yaml, etc.).
func shouldSuppressSpinner() bool {
	return shouldSuppressSpinnerForFormat(GetOutputFormat())
}

// shouldSuppressSpinnerForFormat checks a specific format string. Spinners are
// suppressed for any non-table format to avoid corrupting machine-readable output.
func shouldSuppressSpinnerForFormat(format string) bool {
	return IsQuiet() || (format != "" && format != "table")
}

// PrintSuccess, PrintError, PrintWarning, PrintInfo are defined in:
// - messages_native.go for non-WASM builds
// - messages_wasm.go for WASM builds

func PrintSuccessWithOutput(format string, noColor bool, outputFormat string, args ...interface{}) {
	if IsQuiet() {
		return
	}
	msg := fmt.Sprintf(format, args...)
	if outputFormat == "json" {
		if noColor {
			fmt.Fprintf(os.Stderr, "✓ %s\n", msg)
		} else {
			fmt.Fprint(os.Stderr, color.GreenString("✓ "))
			fmt.Fprintln(os.Stderr, msg)
		}
	} else {
		if noColor {
			fmt.Printf("✓ %s\n", msg)
		} else {
			fmt.Print(color.GreenString("✓ "))
			fmt.Println(msg)
		}
	}
}

func FormatSuccess(msg string, noColor bool) string {
	if noColor {
		return "successfully"
	}
	return color.GreenString("successfully")
}

func PrintResourceSuccess(resourceType, action, uid string, noColor bool) {
	if IsQuiet() {
		return
	}
	uidFormatted := FormatUID(uid, noColor)
	if strings.HasSuffix(action, "ed") {
		PrintSuccess("%s %s %s", noColor, resourceType, action, uidFormatted)
	} else {
		PrintSuccess("%s %sd %s", noColor, resourceType, action, uidFormatted)
	}
}

func PrintResourceCreated(resourceType, uid string, noColor bool) {
	if IsQuiet() {
		return
	}
	PrintSuccess("%s created %s", noColor, resourceType, FormatUID(uid, noColor))
}

func PrintResourceUpdated(resourceType, uid string, noColor bool) {
	if IsQuiet() {
		return
	}
	PrintSuccess("%s updated %s", noColor, resourceType, FormatUID(uid, noColor))
}

func PrintResourceDeleted(resourceType, uid string, immediate, noColor bool) {
	if IsQuiet() {
		return
	}
	msg := fmt.Sprintf("%s deleted %s", resourceType, FormatUID(uid, noColor))
	if immediate {
		msg += "\nThe resource will be deleted immediately"
	} else {
		msg += "\nThe resource will be deleted at the end of the current billing period"
	}
	PrintSuccess(msg, noColor)
}

// Default spinner characters (Braille dots)
var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Enhanced spinner characters for WASM (more visible in browser)
var spinnerCharsWasm = []string{
	"◐", "◓", "◑", "◒", // Circle spinners
}

// Fancy spinner with colors
var spinnerCharsFancy = []string{
	"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷", // Braille block animation
}

type Spinner struct {
	stop         chan bool
	stopped      bool
	frameRate    time.Duration
	mu           sync.Mutex
	noColor      bool
	outputFormat string
	style        string           // "default", "wasm", "fancy"
	wasmSpinner  SpinnerInterface // WASM-specific spinner implementation
}

// SpinnerInterface allows different spinner implementations (WASM vs native)
type SpinnerInterface interface {
	Start(message string)
	Stop()
	StopWithSuccess(msg string)
}

func NewSpinner(noColor bool) *Spinner {
	return &Spinner{
		stop:         make(chan bool, 1),
		frameRate:    100 * time.Millisecond,
		noColor:      noColor,
		outputFormat: "table", // default to table format for backward compatibility
		style:        SpinnerStyleDefault,
	}
}

func NewSpinnerWithOutput(noColor bool, outputFormat string) *Spinner {
	// In WASM builds, this will call the WASM version
	// In non-WASM builds, this will use the regular spinner
	return NewSpinnerWasm(noColor, outputFormat)
}

// NewSpinnerWasm is defined in spinner_wasm.go (WASM) or spinner_native.go (non-WASM)

func (s *Spinner) Start(prefix string) {
	s.runLoop(prefix, nil)
}

// StartWithElapsed starts the spinner, appending "(Xs elapsed)" to the prefix each animation tick.
func (s *Spinner) StartWithElapsed(prefix string) {
	start := time.Now()
	s.runLoop(prefix, &start)
}

// renderFrame returns the styled spinner character for frame index i.
func (s *Spinner) renderFrame(i int) string {
	var chars []string
	switch s.style {
	case SpinnerStyleWASM:
		chars = spinnerCharsWasm
	case SpinnerStyleFancy:
		chars = spinnerCharsFancy
	default:
		chars = spinnerChars
	}

	frame := chars[i%len(chars)]

	if s.noColor {
		return frame
	}
	if s.style == SpinnerStyleFancy || s.style == SpinnerStyleWASM {
		colorFunc := spinnerColors[(i/len(chars))%len(spinnerColors)]
		return colorFunc(frame)
	}
	return color.CyanString(frame)
}

// writeSpinnerLine writes s to os.Stderr, matching the rest of the package's
// status-message convention, synchronized against concurrent os.Stderr
// reassignment via stdErrStreamMu.
func writeSpinnerLine(s string) {
	stdErrStreamMu.RLock()
	defer stdErrStreamMu.RUnlock()
	fmt.Fprint(os.Stderr, s)
}

// writeSpinnerLinef is the formatted counterpart to writeSpinnerLine, used in
// the animation loop to avoid a fmt.Sprintf allocation on every frame.
func writeSpinnerLinef(format string, args ...interface{}) {
	stdErrStreamMu.RLock()
	defer stdErrStreamMu.RUnlock()
	fmt.Fprintf(os.Stderr, format, args...)
}

// nonInteractive reports whether the spinner's sink is non-interactive: a
// machine-readable output format, or output not attached to a TTY. In those
// sinks the carriage-return/clear-line escapes don't collapse anything, so the
// spinner emits a single plain status line rather than animating frame-by-frame.
func (s *Spinner) nonInteractive() bool {
	return s.machineReadableFormat() || !IsTerminal()
}

// machineReadableFormat reports whether the spinner's output format is any
// non-table format (json/csv/xml/go-template) rather than table/empty.
func (s *Spinner) machineReadableFormat() bool {
	return s.outputFormat != "" && s.outputFormat != "table"
}

// runLoop is the shared spinner goroutine logic. If startTime is non-nil,
// an elapsed duration is appended to the message on each tick.
func (s *Spinner) runLoop(prefix string, startTime *time.Time) {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}

	if s.wasmSpinner != nil {
		s.mu.Unlock()
		s.wasmSpinner.Start(prefix)
		return
	}
	s.mu.Unlock()

	if s.nonInteractive() {
		writeSpinnerLinef("%s\n", prefix)
		return
	}

	go func() {
		for i := 0; ; i++ {
			select {
			case <-s.stop:
				return
			default:
				s.mu.Lock()
				if s.stopped {
					s.mu.Unlock()
					return
				}

				styledFrame := s.renderFrame(i)

				msg := prefix
				if startTime != nil {
					elapsed := time.Since(*startTime).Truncate(time.Second)
					msg = fmt.Sprintf("%s (%s elapsed)", prefix, elapsed)
				}

				writeSpinnerLinef("\r\033[K%s %s", styledFrame, msg)
				s.mu.Unlock()
				time.Sleep(s.frameRate)
			}
		}
	}()
}

func (s *Spinner) Stop() {
	s.mu.Lock()

	if s.stopped {
		s.mu.Unlock()
		return
	}

	// If WASM spinner is available, delegate to it
	if s.wasmSpinner != nil {
		s.stopped = true
		s.mu.Unlock()
		s.wasmSpinner.Stop()
		return
	}

	s.stopped = true
	s.mu.Unlock()
	s.stop <- true
	// Only the animated TTY path leaves a frame to clear; in a non-interactive
	// sink a bare clear sequence would just be junk in logs.
	if !s.nonInteractive() {
		writeSpinnerLine("\r\033[K")
	}
}

func (s *Spinner) StopWithSuccess(msg string) {
	s.Stop()
	if IsQuiet() {
		return
	}
	// If WASM spinner is available and the output format is table/empty,
	// delegate to it so the message reaches the captured output buffer
	// instead of os.Stderr/os.Stdout. Safe to call Stop() again here:
	// WasmSpinner.Stop() is idempotent (sync.Once). For machine-readable
	// formats, fall through to the stderr branch below instead: WASM has
	// no TTY, so nonInteractive() is always true there, but the captured
	// output buffer has no separate stdout/stderr split, so writing the
	// success line into it risks it becoming the entire captured response
	// for commands that don't otherwise populate a structured output buffer.
	if s.wasmSpinner != nil && !s.machineReadableFormat() {
		s.wasmSpinner.StopWithSuccess(msg)
		return
	}
	// Write success message to stderr for non-interactive sinks to avoid
	// corrupting machine-readable output streams.
	if s.nonInteractive() {
		if s.noColor {
			writeSpinnerLinef("✓ %s\n", msg)
		} else {
			writeSpinnerLine(color.GreenString("✓ ") + msg + "\n")
		}
	} else {
		if s.noColor {
			writeSpinnerLinef("✓ %s\n", msg)
		} else {
			writeSpinnerLine(color.GreenString("✓ ") + msg + "\n")
		}
	}
}

func PrintResourceCreating(resourceType, uid string, noColor bool) *Spinner {
	if shouldSuppressSpinner() {
		return newNoOpSpinner(noColor)
	}
	uidFormatted := FormatUID(uid, noColor)
	msg := fmt.Sprintf("Creating %s %s...", resourceType, uidFormatted)
	spinner := NewSpinnerWithOutput(noColor, GetOutputFormat())
	spinner.Start(msg)
	return spinner
}

func PrintResourceProvisioning(resourceType, uid string, noColor bool) *Spinner {
	if shouldSuppressSpinner() {
		return newNoOpSpinner(noColor)
	}
	uidFormatted := FormatUID(uid, noColor)
	msg := fmt.Sprintf("Provisioning %s %s...", resourceType, uidFormatted)
	spinner := NewSpinnerWithOutput(noColor, GetOutputFormat())
	spinner.StartWithElapsed(msg)
	return spinner
}

func PrintResourceUpdating(resourceType, uid string, noColor bool) *Spinner {
	if shouldSuppressSpinner() {
		return newNoOpSpinner(noColor)
	}
	uidFormatted := FormatUID(uid, noColor)
	msg := fmt.Sprintf("Updating %s %s...", resourceType, uidFormatted)
	spinner := NewSpinnerWithOutput(noColor, GetOutputFormat())
	spinner.Start(msg)
	return spinner
}

func PrintResourceDeleting(resourceType, uid string, noColor bool) *Spinner {
	if shouldSuppressSpinner() {
		return newNoOpSpinner(noColor)
	}
	uidFormatted := FormatUID(uid, noColor)
	msg := fmt.Sprintf("Deleting %s %s...", resourceType, uidFormatted)
	spinner := NewSpinnerWithOutput(noColor, GetOutputFormat())
	spinner.Start(msg)
	return spinner
}

func PrintResourceListing(resourceType string, noColor bool) *Spinner {
	if shouldSuppressSpinner() {
		return newNoOpSpinner(noColor)
	}
	msg := fmt.Sprintf("Listing %ss...", resourceType)
	spinner := NewSpinnerWithOutput(noColor, GetOutputFormat())
	spinner.Start(msg)
	return spinner
}

func PrintResourceGetting(resourceType, uid string, noColor bool) *Spinner {
	if shouldSuppressSpinner() {
		return newNoOpSpinner(noColor)
	}
	uidFormatted := FormatUID(uid, noColor)
	msg := fmt.Sprintf("Getting %s %s details...", resourceType, uidFormatted)
	spinner := NewSpinnerWithOutput(noColor, GetOutputFormat())
	spinner.Start(msg)
	return spinner
}

func PrintResourceGettingWithOutput(resourceType, uid string, noColor bool, outputFormat string) *Spinner {
	if outputFormat == "" {
		outputFormat = GetOutputFormat()
	}
	if shouldSuppressSpinnerForFormat(outputFormat) {
		s := newNoOpSpinner(noColor)
		s.outputFormat = outputFormat
		return s
	}
	uidFormatted := FormatUID(uid, noColor)
	msg := fmt.Sprintf("Getting %s %s details...", resourceType, uidFormatted)
	spinner := NewSpinnerWithOutput(noColor, outputFormat)
	spinner.Start(msg)
	return spinner
}

func PrintListingResourceTags(resourceType, uid string, noColor bool) *Spinner {
	if shouldSuppressSpinner() {
		return newNoOpSpinner(noColor)
	}
	uidFormatted := FormatUID(uid, noColor)
	msg := fmt.Sprintf("Listing resource tags for %s %s...", resourceType, uidFormatted)
	spinner := NewSpinnerWithOutput(noColor, GetOutputFormat())
	spinner.Start(msg)
	return spinner
}

func PrintResourceValidating(resourceType string, noColor bool) *Spinner {
	if shouldSuppressSpinner() {
		return newNoOpSpinner(noColor)
	}
	msg := fmt.Sprintf("Validating %s order...", resourceType)
	spinner := NewSpinnerWithOutput(noColor, GetOutputFormat())
	spinner.Start(msg)
	return spinner
}

func PrintLoggingIn(noColor bool) *Spinner {
	if IsQuiet() {
		return newNoOpSpinner(noColor)
	}
	msg := "Logging in to Megaport..."
	spinner := NewSpinnerWithOutput(noColor, GetOutputFormat())
	spinner.Start(msg)
	return spinner
}

func PrintLoggingInWithOutput(noColor bool, outputFormat string) *Spinner {
	if IsQuiet() {
		return newNoOpSpinner(noColor)
	}
	if outputFormat == "" {
		outputFormat = GetOutputFormat()
	}
	msg := "Logging in to Megaport..."
	spinner := NewSpinnerWithOutput(noColor, outputFormat)
	spinner.Start(msg)
	return spinner
}

func PrintCustomSpinner(action, resourceId string, noColor bool) *Spinner {
	if shouldSuppressSpinner() {
		return newNoOpSpinner(noColor)
	}
	uidFormatted := FormatUID(resourceId, noColor)
	msg := fmt.Sprintf("%s %s...", action, uidFormatted)
	spinner := NewSpinnerWithOutput(noColor, GetOutputFormat())
	spinner.Start(msg)
	return spinner
}

// PrintVerbose prints a debug message only when verbose mode is active.
func PrintVerbose(format string, noColor bool, args ...interface{}) {
	if !IsVerbose() {
		return
	}
	msg := fmt.Sprintf(format, args...)
	if GetOutputFormat() == "json" {
		if noColor {
			fmt.Fprintf(os.Stderr, "[DEBUG] %s\n", msg)
		} else {
			fmt.Fprint(os.Stderr, color.HiBlackString("[DEBUG] "))
			fmt.Fprintln(os.Stderr, msg)
		}
	} else {
		if noColor {
			fmt.Printf("[DEBUG] %s\n", msg)
		} else {
			fmt.Print(color.HiBlackString("[DEBUG] "))
			fmt.Println(msg)
		}
	}
}

// PrintError, PrintWarning, PrintInfo are defined in:
// - messages_native.go for non-WASM builds
// - messages_wasm.go for WASM builds

func FormatConfirmation(msg string, noColor bool) string {
	if noColor {
		return fmt.Sprintf("%s [y/N]", msg)
	}
	return fmt.Sprintf("%s %s", msg, color.YellowString("[y/N]"))
}

func FormatPrompt(msg string, noColor bool) string {
	if noColor {
		return msg
	}
	return color.BlueString(msg)
}

func FormatExample(example string, noColor bool) string {
	if noColor {
		return example
	}
	return color.CyanString(example)
}

func FormatCommandName(name string, noColor bool) string {
	if noColor {
		return name
	}
	return color.MagentaString(name)
}

func FormatRequiredFlag(flag string, description string, noColor bool) string {
	if noColor {
		return fmt.Sprintf("%s (REQUIRED): %s", flag, description)
	}
	return fmt.Sprintf("%s: %s", color.YellowString("%s (REQUIRED)", flag), description)
}

func FormatOptionalFlag(flag string, description string, noColor bool) string {
	if noColor {
		return fmt.Sprintf("%s: %s", flag, description)
	}
	return fmt.Sprintf("%s: %s", color.BlueString(flag), description)
}

func FormatJSONExample(json string, noColor bool) string {
	if noColor {
		return json
	}
	return color.GreenString(json)
}

func FormatUID(uid string, noColor bool) string {
	if noColor {
		return uid
	}
	return color.CyanString(uid)
}

func StripANSIColors(s string) string {
	return ansiColorRe.ReplaceAllString(s, "")
}

func FormatOldValue(value string, noColor bool) string {
	if noColor {
		return value
	}
	return color.New(color.FgYellow).Sprint(value)
}

func FormatNewValue(value string, noColor bool) string {
	if noColor {
		return value
	}
	return color.New(color.FgGreen).Sprint(value)
}
