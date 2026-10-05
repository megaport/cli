//go:build e2e

package e2e

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_Contract drives the compiled binary via argv and asserts its CLI
// contract: exit codes plus stdout/stderr substrings. Every case is hermetic
// (no network, no credentials) and runs in its own process. Substrings are
// asserted rather than full output snapshots, which would be too brittle.
func TestE2E_Contract(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		args           []string
		wantExit       int
		stdoutContains []string
		stderrContains []string
		stderrAbsent   []string
	}{
		{
			name:           "version subcommand exits zero with a version line",
			args:           []string{"version"},
			wantExit:       exitcodes.Success,
			stdoutContains: []string{"Megaport CLI Version:"},
		},
		{
			name:           "root help lists usage and known subcommands",
			args:           []string{"--help"},
			wantExit:       exitcodes.Success,
			stdoutContains: []string{"Usage:", "ports", "vxc", "mcr", "mve", "completion", "version"},
		},
		{
			name:           "subcommand help exits zero with a full help page",
			args:           []string{"ports", "--help"},
			wantExit:       exitcodes.Success,
			stdoutContains: []string{"Usage:", "ports"},
		},
		{
			name:           "group with no subcommand exits zero with its help",
			args:           []string{"ports"},
			wantExit:       exitcodes.Success,
			stdoutContains: []string{"Usage:", "ports"},
		},
		{
			name:           "completion bash emits the bash marker",
			args:           []string{"completion", "bash"},
			wantExit:       exitcodes.Success,
			stdoutContains: []string{"bash completion for megaport-cli"},
		},
		{
			name:           "completion zsh emits the zsh marker",
			args:           []string{"completion", "zsh"},
			wantExit:       exitcodes.Success,
			stdoutContains: []string{"#compdef megaport-cli"},
		},
		{
			name:           "completion fish emits the fish marker",
			args:           []string{"completion", "fish"},
			wantExit:       exitcodes.Success,
			stdoutContains: []string{"fish completion for megaport-cli"},
		},
		{
			name:           "unknown command is a usage error",
			args:           []string{"boguscommand"},
			wantExit:       exitcodes.Usage,
			stderrContains: []string{"unknown command"},
		},
		{
			name:           "unknown subcommand under a group is a usage error",
			args:           []string{"ports", "bogus"},
			wantExit:       exitcodes.Usage,
			stderrContains: []string{`unknown command "bogus" for "megaport-cli ports"`},
		},
		{
			name:           "unknown subcommand under a nested group is a usage error",
			args:           []string{"mcr", "looking-glass", "bogus"},
			wantExit:       exitcodes.Usage,
			stderrContains: []string{`unknown command "bogus" for "megaport-cli mcr looking-glass"`},
		},
		{
			name:           "unknown flag is a usage error",
			args:           []string{"ports", "list", "--nope"},
			wantExit:       exitcodes.Usage,
			stderrContains: []string{"unknown flag"},
		},
		{
			name:           "wrong arg count is a usage error",
			args:           []string{"ports", "get"},
			wantExit:       exitcodes.Usage,
			stderrContains: []string{"arg(s)"},
		},
		{
			// The purchase command enforces required flags only when not interactive
			// and not --json, so it must fail at flag validation before any login.
			name:     "purchase without flags fails at validation before login",
			args:     []string{"ports", "buy"},
			wantExit: exitcodes.Usage,
			// Match the full composite message to avoid false positives from two
			// independent substrings accidentally both appearing in unrelated output.
			stderrContains: []string{"not set when not using interactive or JSON input"},
			// A PreRunE validation failure routes through finishWithError, which
			// sets SilenceUsage, so cobra must not append its usage block.
			stderrAbsent: []string{"Logging in", "Usage:"},
		},
		{
			name:           "invalid output format is a usage error",
			args:           []string{"version", "--output", "bogus"},
			wantExit:       exitcodes.Usage,
			stderrContains: []string{"invalid output format"},
			stderrAbsent:   []string{"Usage:"},
		},
		{
			name:           "negative max-retries is a usage error without usage block",
			args:           []string{"version", "--max-retries=-1"},
			wantExit:       exitcodes.Usage,
			stderrContains: []string{"--max-retries must be >= 0"},
			stderrAbsent:   []string{"Usage:"},
		},
		{
			name:           "non-positive timeout is a usage error without usage block",
			args:           []string{"version", "--timeout", "0"},
			wantExit:       exitcodes.Usage,
			stderrContains: []string{"--timeout must be greater than 0"},
			stderrAbsent:   []string{"Usage:"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := Run(t, tc.args...)

			// Use require so a wrong exit code stops the subtest immediately
			// rather than cascading into confusing stdout/stderr failures.
			require.Equalf(t, tc.wantExit, res.Exit,
				"exit code\nstdout: %s\nstderr: %s", res.Stdout, res.Stderr)
			for _, want := range tc.stdoutContains {
				assert.Containsf(t, res.Stdout, want, "stdout missing %q\nstdout: %s", want, res.Stdout)
			}
			for _, want := range tc.stderrContains {
				assert.Containsf(t, res.Stderr, want, "stderr missing %q\nstderr: %s", want, res.Stderr)
			}
			for _, absent := range tc.stderrAbsent {
				assert.NotContainsf(t, res.Stderr, absent, "stderr unexpectedly contains %q\nstderr: %s", absent, res.Stderr)
			}
		})
	}
}

// TestE2E_JSONErrorEnvelope verifies that under --output json an erroring command
// emits a structured JSON envelope on stderr whose code and type reflect a usage
// error. The purchase command parses --json before any login, so an unparseable
// payload triggers the envelope hermetically.
func TestE2E_JSONErrorEnvelope(t *testing.T) {
	t.Parallel()

	res := Run(t, "ports", "buy", "--json", "this-is-not-json", "--output", "json")

	require.Equalf(t, exitcodes.Usage, res.Exit, "invalid JSON payload should be a usage error (exit 2)\nstderr: %s", res.Stderr)
	assert.NotContains(t, res.Stderr, "Logging in", "should fail before any login attempt")

	// The JSON encoder uses SetIndent so the envelope opens with a bare {
	// on its own line. Anchoring on \n{ rather than a bare { guards against
	// a future progress line whose text happens to contain a { character.
	idx := strings.Index(res.Stderr, "\n{")
	require.NotEqualf(t, -1, idx, "no JSON envelope found on stderr: %s", res.Stderr)
	start := idx + 1 // skip past the newline to the opening {

	var envelope struct {
		Error struct {
			Code    int    `json:"code"`
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	dec := json.NewDecoder(strings.NewReader(res.Stderr[start:]))
	require.NoErrorf(t, dec.Decode(&envelope), "envelope should be valid JSON: %s", res.Stderr[start:])

	assert.Equal(t, exitcodes.Usage, envelope.Error.Code, "envelope code should be 2 (usage error)")
	assert.Equal(t, "usage_error", envelope.Error.Type, "envelope type should be usage_error")
	assert.NotEmpty(t, envelope.Error.Message, "envelope should carry an error message")
}

// TestE2E_PreRunJSONErrorEnvelope verifies that PreRunE / PersistentPreRunE
// validation failures (conditional-requirement checks, --max-retries, --timeout)
// emit the same structured JSON envelope under --output json as the RunE error
// paths do. These checks run before any login, so each case is hermetic. It also
// asserts stdout stays empty and the cobra usage block is not appended.
func TestE2E_PreRunJSONErrorEnvelope(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		args []string
	}{
		{
			name: "conditional-requirement failure",
			args: []string{"ports", "buy", "--output", "json"},
		},
		{
			name: "negative max-retries",
			args: []string{"version", "--max-retries=-1", "--output", "json"},
		},
		{
			name: "non-positive timeout",
			args: []string{"version", "--timeout", "0", "--output", "json"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := Run(t, tc.args...)

			require.Equalf(t, exitcodes.Usage, res.Exit,
				"PreRunE validation failure should be a usage error (exit 2)\nstderr: %s", res.Stderr)
			assert.Emptyf(t, res.Stdout, "stdout should stay empty on a validation failure\nstdout: %s", res.Stdout)
			assert.NotContains(t, res.Stderr, "Usage:", "SilenceUsage should suppress the cobra usage block")

			// Anchor on the first '{' so the test is robust whether or not a
			// preceding line (e.g. a config-default warning) is written first.
			idx := strings.IndexByte(res.Stderr, '{')
			require.NotEqualf(t, -1, idx, "no JSON envelope found on stderr: %s", res.Stderr)

			var envelope struct {
				Error struct {
					Code    int    `json:"code"`
					Type    string `json:"type"`
					Message string `json:"message"`
				} `json:"error"`
			}
			dec := json.NewDecoder(strings.NewReader(res.Stderr[idx:]))
			require.NoErrorf(t, dec.Decode(&envelope), "envelope should be valid JSON: %s", res.Stderr[idx:])

			assert.Equal(t, exitcodes.Usage, envelope.Error.Code, "envelope code should be 2 (usage error)")
			assert.Equal(t, "usage_error", envelope.Error.Type, "envelope type should be usage_error")
			assert.NotEmpty(t, envelope.Error.Message, "envelope should carry an error message")
		})
	}
}
