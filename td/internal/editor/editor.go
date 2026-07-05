// Package editor resolves and parses the user's external editor command for
// use by every td call site ($EDITOR/$VISUAL handling). It supports editor
// values that contain a quoted executable path with spaces plus trailing
// arguments, e.g.:
//
//	"C:\Program Files\Microsoft VS Code\bin\code.cmd" --wait
package editor

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Fallback returns the platform default editor: notepad on Windows (vi/vim do
// not exist there), or unixDefault elsewhere.
func Fallback(unixDefault string) string {
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return unixDefault
}

// Parse splits a raw editor command (typically $VISUAL or $EDITOR) into the
// executable name and its arguments. Double-quoted segments are kept intact,
// so paths containing spaces work when quoted. Backslashes are NOT treated as
// escape characters (they are path separators on Windows).
func Parse(raw string) (name string, args []string, err error) {
	fields := splitQuoted(raw)
	if len(fields) == 0 {
		return "", nil, fmt.Errorf("empty editor command")
	}
	return fields[0], fields[1:], nil
}

// Command builds an *exec.Cmd that opens path with the given raw editor
// command, appending path as the final argument.
func Command(raw, path string) (*exec.Cmd, error) {
	name, args, err := Parse(raw)
	if err != nil {
		return nil, err
	}
	return exec.Command(name, append(args, path)...), nil
}

// splitQuoted splits s on whitespace while treating double-quoted runs as a
// single field (quotes stripped). An unterminated quote extends to the end of
// the string.
func splitQuoted(s string) []string {
	var fields []string
	var cur strings.Builder
	inQuote := false
	hasField := false

	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
			hasField = true
		case !inQuote && (r == ' ' || r == '\t'):
			if hasField {
				fields = append(fields, cur.String())
				cur.Reset()
				hasField = false
			}
		default:
			cur.WriteRune(r)
			hasField = true
		}
	}
	if hasField {
		fields = append(fields, cur.String())
	}
	return fields
}
