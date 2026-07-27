//go:build js && wasm

package megaport

import (
	"net/http"
	"strings"
	"syscall/js"
	"testing"

	"github.com/megaport/megaport-cli/internal/base/output"
	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/megaport/megaport-cli/internal/wasm"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteWithArgs_SessionExpired_StreamsMarkerToHandler drives the full WASM
// entry path (ExecuteWithArgs) with a live-output handler registered, the way a
// host using registerOutputHandler runs. An action that prints its own error
// before returning latches ErrorEmitted and streams the raw (marker-less) error;
// the returned CLIError carries MEGAPORT_SESSION_EXPIRED but ExecuteWithArgs skips
// its fallback buffer write once anything has streamed. So the only way the host
// sees the marker is if finishWithError re-emits it as a fresh streamed chunk.
// This asserts that chunk reaches the handler, not just the returned Go error.
func TestExecuteWithArgs_SessionExpired_StreamsMarkerToHandler(t *testing.T) {
	t.Setenv("MEGAPORT_ACCESS_TOKEN", "some-token")

	wasm.ResetOutputBuffers()

	var streamed []string
	fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			streamed = append(streamed, args[0].String())
		}
		return nil
	})
	defer fn.Release()
	wasm.RegisterOutputCallback(fn.Value)
	defer wasm.UnregisterOutputCallback()

	cmd := &cobra.Command{
		Use: "sessionexpiredtest",
		RunE: utils.WrapRunE(func(cmd *cobra.Command, args []string) error {
			apiErr := &megaport.ErrorResponse{
				Response: &http.Response{StatusCode: 401, Header: http.Header{}, Request: &http.Request{}},
				Message:  "unauthorized",
			}
			// Mirror actions like users_actions.go's GetUser, which stream their
			// own error before returning it.
			output.PrintError("Failed to get user: %v", false, apiErr)
			return apiErr
		}),
	}
	rootCmd.AddCommand(cmd)
	defer rootCmd.RemoveCommand(cmd)

	_ = ExecuteWithArgs([]string{"megaport-cli", "sessionexpiredtest"})

	require.True(t, wasm.DidStreamOutput(), "the error was streamed to the handler")
	assert.Contains(t, strings.Join(streamed, ""), utils.SessionExpiredMarker,
		"a streaming host must receive the re-auth marker even when the action printed its own error first")
}
