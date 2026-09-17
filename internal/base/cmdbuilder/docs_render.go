//go:build !js || !wasm

package cmdbuilder

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"
)

// DocsDirectory is the fallback location for markdown documentation files
var DocsDirectory = "./docs"

// FindDocFile locates the markdown file for a specific command. The second
// return value reports whether the path is a temp file the caller must remove.
func FindDocFile(cmd *cobra.Command) (string, bool, error) {
	cmdPath := getCommandPath(cmd)
	docName := cmdPath + ".md"

	// First try to read from embedded docs. embed.FS paths are always
	// slash-separated, so use path.Join, not filepath.Join (\ on Windows).
	embeddedPath := path.Join("docs", docName)
	content, err := embeddedDocsFS.ReadFile(embeddedPath)
	if err == nil {
		// Stage the embedded content in a temp file; RenderDocFile reads it back
		// by path and the caller removes it after rendering.
		tempFile, err := os.CreateTemp("", "megaport-docs-*.md")
		if err != nil {
			return "", false, fmt.Errorf("failed to create temporary file: %w", err)
		}
		if _, err := tempFile.Write(content); err != nil {
			_ = tempFile.Close()
			_ = os.Remove(tempFile.Name())
			return "", false, fmt.Errorf("failed to write to temporary file: %w", err)
		}
		if err := tempFile.Close(); err != nil {
			_ = os.Remove(tempFile.Name())
			return "", false, fmt.Errorf("failed to close temporary file: %w", err)
		}
		return tempFile.Name(), true, nil
	}

	// If embedded file not found, try local docs directory as fallback
	docPath := filepath.Join(DocsDirectory, docName)

	// Check if the file exists
	if _, err := os.Stat(docPath); os.IsNotExist(err) {
		return "", false, fmt.Errorf("documentation file not found for %s: %w", cmdPath, err)
	}

	return docPath, false, nil
}

// RenderDocFile reads and renders a markdown file using Glamour
func RenderDocFile(filePath string) (string, error) {
	// Read the markdown file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read documentation file: %w", err)
	}

	// Create a glamour renderer with default style
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create markdown renderer: %w", err)
	}

	// Render the markdown content
	rendered, err := renderer.Render(string(content))
	if err != nil {
		return "", fmt.Errorf("failed to render documentation: %w", err)
	}

	return rendered, nil
}

// ShowDocumentation displays rendered documentation for a command
func ShowDocumentation(cmd *cobra.Command) error {
	docPath, isTemp, err := FindDocFile(cmd)
	if err != nil {
		return err
	}

	// If we created a temporary file, ensure it gets deleted
	if isTemp {
		defer os.Remove(docPath)
	}

	rendered, err := RenderDocFile(docPath)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), rendered)
	return err
}
