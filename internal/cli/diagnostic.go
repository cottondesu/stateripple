package cli

import (
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

// diagnose writes one diagnostic as exactly one stderr line.
func diagnose(stderr io.Writer, format string, args ...any) {
	fmt.Fprintf(stderr, "stateripple: %s\n", escapeDiagnostic(fmt.Sprintf(format, args...)))
}

// escapeDiagnostic makes a message safe to show on a terminal or in a log.
// Paths, revisions, flag names and tool output reach stderr inside wrapped
// errors such as *os.PathError, so the complete message is escaped once, here,
// instead of at every place that formats a path. Each rune that
// strconv.IsGraphic rejects (C0 and C1 controls, Unicode line and paragraph
// separators, bidirectional and other format controls) and each invalid UTF-8
// byte becomes the Go escape that the text report uses. Backslashes and quotes
// are left alone: ordinary paths are unchanged and text that a caller already
// quoted with %q is not escaped a second time.
func escapeDiagnostic(message string) string {
	var b strings.Builder
	for i := 0; i < len(message); {
		r, size := utf8.DecodeRuneInString(message[i:])
		switch {
		case r == utf8.RuneError && size == 1:
			fmt.Fprintf(&b, `\x%02x`, message[i])
		case strconv.IsGraphic(r):
			b.WriteString(message[i : i+size])
		default:
			quoted := strconv.QuoteRuneToGraphic(r)
			b.WriteString(quoted[1 : len(quoted)-1])
		}
		i += size
	}
	return b.String()
}

// writeHelp writes help, usage or version text and reports a failed or short
// write instead of exiting 0.
func writeHelp(w, stderr io.Writer, text string) int {
	n, err := io.WriteString(w, text)
	if err == nil && n != len(text) {
		err = io.ErrShortWrite
	}
	if err != nil {
		diagnose(stderr, "write output: %v", err)
		return 1
	}
	return 0
}

func flagDefaults(fs *flag.FlagSet) string {
	var b strings.Builder
	fs.SetOutput(&b)
	fs.PrintDefaults()
	fs.SetOutput(io.Discard)
	return b.String()
}
