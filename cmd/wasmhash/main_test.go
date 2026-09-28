package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func wantHash(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])[:hashLen]
}

func TestHashFileDeterministicAndContentSensitive(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.wasm")
	b := filepath.Join(dir, "b.wasm")
	if err := os.WriteFile(a, []byte("the same bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("the same bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	ha, err := hashFile(a)
	if err != nil {
		t.Fatalf("hashFile(a): %v", err)
	}
	hb, err := hashFile(b)
	if err != nil {
		t.Fatalf("hashFile(b): %v", err)
	}
	if ha != hb {
		t.Fatalf("identical content produced different hashes: %s vs %s", ha, hb)
	}
	if len(ha) != hashLen {
		t.Fatalf("hash length = %d, want %d", len(ha), hashLen)
	}
	if ha != wantHash([]byte("the same bytes")) {
		t.Fatalf("hash %s does not match sha256 prefix", ha)
	}

	// A one-byte change must change the hash (acceptance: new build -> new hash).
	if err := os.WriteFile(b, []byte("the same byteS"), 0o644); err != nil {
		t.Fatal(err)
	}
	hb2, err := hashFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if hb2 == ha {
		t.Fatal("changed content produced the same hash")
	}
}

func TestInsertHash(t *testing.T) {
	got := insertHash(filepath.FromSlash("web/dist/megaport.wasm"), "abcd1234")
	want := filepath.FromSlash("web/dist/megaport.abcd1234.wasm")
	if got != want {
		t.Fatalf("insertHash = %q, want %q", got, want)
	}
}

func TestRenameWithHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "megaport.wasm")
	content := []byte("wasm bytes for hashing")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	newPath, hash, err := renameWithHash(path)
	if err != nil {
		t.Fatalf("renameWithHash: %v", err)
	}
	if hash != wantHash(content) {
		t.Fatalf("hash %s does not match content", hash)
	}
	if filepath.Base(newPath) != "megaport."+hash+".wasm" {
		t.Fatalf("newPath base = %q, want megaport.%s.wasm", filepath.Base(newPath), hash)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("original %s should be gone after rename (stat err: %v)", path, err)
	}
	got, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("read hashed file: %v", err)
	}
	if string(got) != string(content) {
		t.Fatal("hashed file content differs from original")
	}
}

const indexTemplate = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8">
    <title>demo</title>
    <script type="module" crossorigin src="/assets/main-abc.js"></script>
  </head>
  <body><div id="app"></div></body>
</html>
`

func TestWasmScriptNeutralizesScriptBreakout(t *testing.T) {
	// A "</script>" in the URL must not close the tag early in the HTML parser.
	got := wasmScript("/megaport.</script><script>alert(1)</script>.wasm")
	if n := strings.Count(got, "</script>"); n != 1 {
		t.Fatalf("payload </script> not neutralized, found %d closing tags: %q", n, got)
	}
	if strings.Contains(got, "<script>alert") {
		t.Fatalf("payload <script> survived: %q", got)
	}
}

func TestInjectWasmURLInsertsBeforeHead(t *testing.T) {
	dir := t.TempDir()
	idx := filepath.Join(dir, "index.html")
	if err := os.WriteFile(idx, []byte(indexTemplate), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := injectWasmURL(idx, "/megaport.abcd1234.wasm"); err != nil {
		t.Fatalf("injectWasmURL: %v", err)
	}
	out, err := os.ReadFile(idx)
	if err != nil {
		t.Fatal(err)
	}
	html := string(out)

	want := `<script>window.__MEGAPORT_WASM_URL__="/megaport.abcd1234.wasm"</script>`
	if !strings.Contains(html, want) {
		t.Fatalf("injected script not found.\n%s", html)
	}
	// Must sit inside <head> (before the closing tag) so it runs before the
	// deferred module script that loads the wasm.
	if strings.Index(html, want) > strings.Index(html, "</head>") {
		t.Fatal("injected script is not before </head>")
	}
}

func TestInjectWasmURLIdempotent(t *testing.T) {
	dir := t.TempDir()
	idx := filepath.Join(dir, "index.html")
	if err := os.WriteFile(idx, []byte(indexTemplate), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := injectWasmURL(idx, "/megaport.aaaaaaaa.wasm"); err != nil {
		t.Fatal(err)
	}
	if err := injectWasmURL(idx, "/megaport.bbbbbbbb.wasm"); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(idx)
	if err != nil {
		t.Fatal(err)
	}
	html := string(out)

	if n := strings.Count(html, "__MEGAPORT_WASM_URL__"); n != 1 {
		t.Fatalf("expected exactly one injected script after re-inject, got %d", n)
	}
	if strings.Contains(html, "aaaaaaaa") {
		t.Fatal("stale wasm URL left behind after re-inject")
	}
	if !strings.Contains(html, "/megaport.bbbbbbbb.wasm") {
		t.Fatal("latest wasm URL missing after re-inject")
	}
}

func TestInjectWasmURLNoHead(t *testing.T) {
	dir := t.TempDir()
	idx := filepath.Join(dir, "index.html")
	if err := os.WriteFile(idx, []byte("<html><body>no head</body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := injectWasmURL(idx, "/megaport.x.wasm"); err == nil {
		t.Fatal("expected error when there is no </head> to inject into")
	}
}

func TestInjectWasmURLMissingFile(t *testing.T) {
	dir := t.TempDir()
	if err := injectWasmURL(filepath.Join(dir, "does-not-exist.html"), "/megaport.x.wasm"); err == nil {
		t.Fatal("expected error when index.html cannot be read")
	}
}

func TestRunHappyPath(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "megaport.wasm")
	content := []byte("hello wasm world")
	if err := os.WriteFile(wasmPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	idx := filepath.Join(dir, "index.html")
	if err := os.WriteFile(idx, []byte(indexTemplate), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := run([]string{"wasmhash", wasmPath, idx}, &stdout, &stderr); code != 0 {
		t.Fatalf("run code = %d, want 0 (stderr: %s)", code, stderr.String())
	}

	hash := wantHash(content)
	wantPath := filepath.Join(dir, "megaport."+hash+".wasm")
	if got := strings.TrimSpace(stdout.String()); got != wantPath {
		t.Fatalf("stdout = %q, want hashed path %q", got, wantPath)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("hashed wasm not created: %v", err)
	}
	html, err := os.ReadFile(idx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "/megaport."+hash+".wasm") {
		t.Fatalf("index.html not pointed at hashed wasm:\n%s", html)
	}
}

func TestRunTooFewArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"wasmhash", "only-one-arg"}, &stdout, &stderr); code != 2 {
		t.Fatalf("run code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("stderr missing usage line: %q", stderr.String())
	}
}

func TestRunHashError(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := run([]string{"wasmhash", filepath.Join(dir, "nope.wasm"), filepath.Join(dir, "index.html")}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run code = %d, want 1", code)
	}
	if stderr.Len() == 0 {
		t.Fatal("expected an error on stderr when the wasm file is missing")
	}
}

func TestRunInjectError(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "megaport.wasm")
	if err := os.WriteFile(wasmPath, []byte("bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	idx := filepath.Join(dir, "index.html")
	if err := os.WriteFile(idx, []byte("<html><body>no head</body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"wasmhash", wasmPath, idx}, &stdout, &stderr); code != 1 {
		t.Fatalf("run code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "no </head>") {
		t.Fatalf("stderr missing inject error: %q", stderr.String())
	}
}

func TestRunRollsBackRenameOnInjectFailure(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "megaport.wasm")
	if err := os.WriteFile(wasmPath, []byte("bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	idx := filepath.Join(dir, "index.html")
	// No </head>: injectWasmURL fails after the rename has already happened.
	if err := os.WriteFile(idx, []byte("<html><body>no head</body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := run([]string{"wasmhash", wasmPath, idx}, &stdout, &stderr); code != 1 {
		t.Fatalf("run code = %d, want 1", code)
	}

	// The original wasm must be restored so a re-run can succeed, with no
	// orphaned hashed file left behind.
	if _, err := os.Stat(wasmPath); err != nil {
		t.Fatalf("original wasm not restored after inject failure: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "megaport.*.wasm"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("orphaned hashed wasm left behind: %v", matches)
	}
}

func TestHashFileLengthResistsCollisions(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.wasm")
	if err := os.WriteFile(p, []byte("some bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := hashFile(p)
	if err != nil {
		t.Fatal(err)
	}
	// At least 64 bits of hash: an immutable-cached URL must never be reusable
	// by different content, or stale wasm would be served for a year.
	if len(h) < 16 {
		t.Fatalf("hash prefix = %d chars (%d bits), want >= 16 chars (64 bits)", len(h), len(h)*4)
	}
}

func TestInjectWasmURLIdempotentCRLF(t *testing.T) {
	dir := t.TempDir()
	idx := filepath.Join(dir, "index.html")
	if err := os.WriteFile(idx, []byte(indexTemplate), 0o644); err != nil {
		t.Fatal(err)
	}

	// Re-inject across rebuilds, normalizing the file to CRLF in between (as git
	// autocrlf or a Windows editor would). The old tag must be fully replaced
	// each time, with no blank-line creep around it.
	for _, url := range []string{"/megaport.aaaa.wasm", "/megaport.bbbb.wasm", "/megaport.cccc.wasm"} {
		if err := injectWasmURL(idx, url); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(idx)
		if err != nil {
			t.Fatal(err)
		}
		lf := strings.ReplaceAll(string(b), "\r\n", "\n")
		crlf := strings.ReplaceAll(lf, "\n", "\r\n")
		if err := os.WriteFile(idx, []byte(crlf), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	out, err := os.ReadFile(idx)
	if err != nil {
		t.Fatal(err)
	}
	html := string(out)
	if n := strings.Count(html, "__MEGAPORT_WASM_URL__"); n != 1 {
		t.Fatalf("expected exactly one injected script, got %d:\n%q", n, html)
	}
	if !strings.Contains(html, "/megaport.cccc.wasm") {
		t.Fatalf("latest url missing:\n%q", html)
	}
	if strings.Contains(html, "</script>\r\n\r\n") || strings.Contains(html, "</script>\n\r\n") {
		t.Fatalf("blank line accumulated after injected tag:\n%q", html)
	}
}
