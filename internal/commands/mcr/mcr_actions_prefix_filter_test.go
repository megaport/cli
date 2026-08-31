package mcr

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStderr captures what fn writes to os.Stderr. The read end is drained
// concurrently so fn cannot block on a full pipe buffer.
func captureStderr(t *testing.T, fn func()) (result string) {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	defer func() { os.Stderr = old }()

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = io.Copy(&buf, r)
	}()

	defer func() {
		_ = w.Close()
		<-done
		_ = r.Close()
		result = buf.String()
	}()

	fn()
	return
}

// A non-numeric prefix-filter-list ID fails strconv.Atoi before any login. The
// wrapper must still surface that error on stderr rather than exit silently
// (regression test for ESD-1499).
func TestUpdateMCRPrefixFilterList_NonNumericID_PrintsToStderr(t *testing.T) {
	wrapped := utils.WrapColorAwareRunE(UpdateMCRPrefixFilterList)

	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().Bool("no-color", false, "")
	root.PersistentFlags().String("fields", "", "")
	root.PersistentFlags().String("query", "", "")
	root.PersistentFlags().String("template", "", "")
	root.PersistentFlags().String("output", "table", "")
	child := &cobra.Command{Use: "update-prefix-filter-list"}
	child.Flags().Bool("interactive", false, "")
	child.Flags().String("json", "", "")
	child.Flags().String("json-file", "", "")
	child.Flags().String("description", "", "")
	child.Flags().String("address-family", "", "")
	child.Flags().String("entries", "", "")
	root.AddCommand(child)

	stderr := captureStderr(t, func() {
		err := wrapped(child, []string{"someuid", "notanumber"})
		require.Error(t, err)
	})

	assert.Contains(t, stderr, "Invalid prefix filter list ID", "failing command must print its error to stderr")
}
