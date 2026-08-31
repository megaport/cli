//go:build !js || !wasm

package cmdbuilder

import (
	"bytes"
	"embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

//go:embed docs/megaport-cli.md
var embeddedTestDocsFS embed.FS

// ShowDocumentation must write to the command's configured output writer, not
// straight to os.Stdout, so callers (and tests) can capture/redirect docs output.
func TestShowDocumentationWritesToCobraWriter(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "megaport-cli.md"), []byte("# Title\n\nbody text\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir := DocsDirectory
	DocsDirectory = dir
	t.Cleanup(func() { DocsDirectory = origDir })

	origFS := embeddedDocsFS
	embeddedDocsFS = embed.FS{} // force the on-disk fallback
	t.Cleanup(func() { embeddedDocsFS = origFS })

	root := &cobra.Command{Use: "megaport-cli"}
	var buf bytes.Buffer
	root.SetOut(&buf)

	if err := ShowDocumentation(root); err != nil {
		t.Fatalf("ShowDocumentation: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected rendered docs on the cobra writer, got nothing")
	}
}

// ShowDocumentation must stage embedded docs in a temp file and clean it up
// afterwards, since embed.FS content can't be read back by path directly.
func TestShowDocumentationEmbeddedTempFile(t *testing.T) {
	origFS := embeddedDocsFS
	embeddedDocsFS = embeddedTestDocsFS
	t.Cleanup(func() { embeddedDocsFS = origFS })

	root := &cobra.Command{Use: "megaport-cli"}
	var buf bytes.Buffer
	root.SetOut(&buf)

	docPath, isTemp, err := FindDocFile(root)
	if err != nil {
		t.Fatalf("FindDocFile: %v", err)
	}
	if !isTemp {
		t.Fatal("expected embedded doc to be staged in a temp file")
	}
	defer os.Remove(docPath)

	if _, err := os.Stat(docPath); err != nil {
		t.Fatalf("expected temp doc file to exist: %v", err)
	}

	if err := ShowDocumentation(root); err != nil {
		t.Fatalf("ShowDocumentation: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected rendered docs on the cobra writer, got nothing")
	}
}

func TestFindDocFileNotFound(t *testing.T) {
	origFS := embeddedDocsFS
	embeddedDocsFS = embed.FS{} // force the on-disk fallback
	t.Cleanup(func() { embeddedDocsFS = origFS })

	origDir := DocsDirectory
	DocsDirectory = t.TempDir() // empty, no doc files present
	t.Cleanup(func() { DocsDirectory = origDir })

	root := &cobra.Command{Use: "megaport-cli"}
	_, isTemp, err := FindDocFile(root)
	if err == nil {
		t.Fatal("expected an error for a missing doc file")
	}
	if isTemp {
		t.Fatal("expected isTemp to be false when no doc file is found")
	}
}
