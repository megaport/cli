//go:build js && wasm

package wasm

import "unicode"

// SanitizeTerminalOutput strips control sequences from WASM data output
// (tables, JSON/CSV/XML, narrative messages, errors) before it reaches the
// host terminal. The host writes this text to xterm via terminal.write()
// without escaping, so a crafted resource name or API response field
// carrying ESC/CSI/OSC bytes could otherwise move the cursor or erase
// unrelated lines and repaint a fake prompt over real content. Some of
// these fields are third-party controlled (partner/marketplace listings
// carry other tenants' names; error paths echo API response bodies).
//
// The CLI's own SGR color sequences (ESC '[' <digits/';'> 'm', emitted by
// fatih/color for table and status styling) are allowlisted so intended
// styling survives; every other C0 control, DEL, and C1 control is dropped
// outright, and a non-SGR CSI/OSC sequence is neutralized by dropping its
// introducing ESC (or C1 equivalent), leaving the remainder as inert
// printable text rather than removing it wholesale. \n and \t pass through
// unchanged since they are structural formatting in a multi-line document.
// A lone \r is dropped:
// unlike \n/\t it is itself a cursor-move (back to column 0) that can
// overwrite the start of a rendered line with no ESC/CSI involved, so it is
// only preserved as part of a CRLF pair.
//
// This does not distinguish a CLI-emitted SGR sequence from attacker text
// that merely happens to match SGR syntax once everything is one string.
// That is an accepted gap: SGR only changes subsequent text color/style, it
// cannot move the cursor or erase the screen, so the residual risk is
// cosmetic, not the cursor-hijack/erase attack this sanitizer closes.
func SanitizeTerminalOutput(s string) string {
	return sanitizeControlSequences(s, true)
}

// SanitizeTerminalText strips all C0/C1 controls, DEL, and ESC from a value
// bound for a channel that carries no color of its own: the live prompt channel
// (see the WASM prompt bridge in internal/utils/prompts_wasm.go) and the async
// result.error field, which the host renders as its own styled error line.
// Nothing needs to be allowlisted, and \n/\r/\t are stripped like any other
// control byte since callers supply their own structural line breaks.
func SanitizeTerminalText(s string) string {
	return sanitizeControlSequences(s, false)
}

// sanitizeControlSequences removes terminal control bytes from s. When
// allowSGR is true, \n and \t are preserved as structural bytes, \r is
// preserved only when it is immediately followed by \n (a CRLF pair), and an
// ESC that introduces a well-formed CSI SGR sequence (ESC '[' <digits/';'>*
// 'm') is copied through verbatim. Any other ESC is dropped by itself: this
// leaves the rest of a CSI or OSC sequence (e.g. "[2K" or "]0;evil") as inert
// printable text, since the terminal only interprets it as a command when
// the introducing ESC (or its C1 equivalent) is present. All other Unicode
// control runes, including DEL and the C1 range (some terminals treat
// 0x80-0x9f as 8-bit CSI/OSC introducers, e.g. 0x9b as CSI), are dropped
// outright, including a lone \r: on its own it moves the cursor to column 0,
// which is a cursor-hijack primitive with no ESC/CSI needed.
func sanitizeControlSequences(s string, allowSGR bool) string {
	runes := []rune(s)
	out := make([]rune, 0, len(runes))
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case r == 0x1b:
			if allowSGR {
				if end, ok := sgrSequenceEnd(runes, i); ok {
					out = append(out, runes[i:end+1]...)
					i = end
					continue
				}
			}
			// Not an allowlisted sequence: drop just the ESC byte so any
			// following text is re-evaluated as plain characters, not
			// re-armed as part of a control sequence.
		case allowSGR && (r == '\n' || r == '\t'):
			out = append(out, r)
		case allowSGR && r == '\r':
			if i+1 < len(runes) && runes[i+1] == '\n' {
				out = append(out, r)
			}
			// Lone \r (not followed by \n): drop it.
		case unicode.IsControl(r):
			// Drop C0/C1 controls and DEL.
		default:
			out = append(out, r)
		}
	}
	return string(out)
}

// sgrSequenceEnd reports whether the ESC at runes[start] begins a CSI SGR
// sequence and, if so, returns the index of the final 'm'. Only SGR is
// allowlisted: it is the only CSI class the CLI's own color output emits.
// Cursor movement, erase, and every other CSI/OSC final byte fall through
// and are stripped by the caller.
func sgrSequenceEnd(runes []rune, start int) (int, bool) {
	if start+1 >= len(runes) || runes[start+1] != '[' {
		return 0, false
	}
	for i := start + 2; i < len(runes); i++ {
		switch r := runes[i]; {
		case r >= '0' && r <= '9', r == ';':
			continue
		case r == 'm':
			return i, true
		default:
			return 0, false
		}
	}
	return 0, false
}
