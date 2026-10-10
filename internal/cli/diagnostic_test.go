package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"unicode/utf8"
)

const simulateUsage = help + "  -format string\n    \toutput format: json or text (default \"text\")\n  -input string\n    \tsnapshot JSON file (required)\n  -out string\n    \twrite report to this file instead of stdout\n"
const analyzeUsage = help + "  -base string\n    \tGit base revision (default \"HEAD~1\")\n  -format string\n    \toutput format: json or text (default \"text\")\n  -out string\n    \twrite report to this file instead of stdout\n"

// hostile holds one member of every class a diagnostic must not show raw: C0
// and C1 controls, NUL, DEL, Unicode line and paragraph separators,
// bidirectional and zero-width format controls, and an invalid UTF-8 byte.
const hostile = "a\nstateripple: forged\r\x1b[2J\x00\a\x7f\u0085\u009b\u2028\u2029\u202e\u2066\u200b\ufeff\xff"
const hostileEscaped = `a\nstateripple: forged\r\x1b[2J\x00\a\x7f\u0085\u009b\u2028\u2029\u202e\u2066\u200b\ufeff\xff`

// requireOneSafeLine fails unless text is one LF-terminated line of valid
// UTF-8 in which every rune is graphic.
func requireOneSafeLine(t *testing.T, text string) {
	t.Helper()
	body, terminated := strings.CutSuffix(text, "\n")
	if !terminated || strings.Contains(body, "\n") || !utf8.ValidString(body) {
		t.Fatalf("diagnostic is not one valid line: %q", text)
	}
	for _, r := range body {
		if !strconv.IsGraphic(r) {
			t.Fatalf("diagnostic contains raw %U: %q", r, text)
		}
	}
}

func TestEscapeDiagnosticLeavesGraphicTextUnchanged(t *testing.T) {
	for _, message := range []string{
		"",
		"open snapshot: stat missing.json: no such file or directory",
		`output already exists; refusing to overwrite: C:\reports\out "final".json`,
		"入力/スナップショット 日本語.json — café ☃ 🙂",
		"non-breaking\u00a0and ideographic\u3000spaces",
		`seed node not found: "x\n\x1b[2J"`,
	} {
		if got := escapeDiagnostic(message); got != message {
			t.Errorf("escapeDiagnostic(%q) = %q", message, got)
		}
	}
}

func TestEscapeDiagnosticEscapesEveryNonGraphicRune(t *testing.T) {
	if got := escapeDiagnostic(hostile); got != hostileEscaped {
		t.Fatalf("got  %q\nwant %q", got, hostileEscaped)
	}
	// Every code point: graphic runes are kept, all others get the escape the
	// text report produces, and the result never contains a non-graphic rune.
	for r := rune(0); r <= utf8.MaxRune; r++ {
		if r >= 0xd800 && r <= 0xdfff {
			continue
		}
		in := string(r)
		want := in
		if !strconv.IsGraphic(r) {
			quoted := strconv.QuoteToGraphic(in)
			want = quoted[1 : len(quoted)-1]
		}
		if got := escapeDiagnostic(in); got != want {
			t.Fatalf("%U: got %q, want %q", r, got, want)
		}
	}
	for b := 0x80; b <= 0xff; b++ {
		if got, want := escapeDiagnostic(string([]byte{byte(b)})), fmt.Sprintf(`\x%02x`, b); got != want {
			t.Fatalf("invalid byte %#x: got %q, want %q", b, got, want)
		}
	}
	if once := escapeDiagnostic(hostile); escapeDiagnostic(once) != once {
		t.Fatal("escaping an escaped diagnostic changed it again")
	}
}

func TestDiagnoseEscapesWrappedPathErrors(t *testing.T) {
	path := &os.PathError{Op: "open", Path: "/reports/" + hostile, Err: syscall.ENOENT}
	link := &os.LinkError{Op: "link", Old: "/tmp/" + hostile, New: "/out/" + hostile, Err: syscall.EEXIST}
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"path error", path, "stateripple: open /reports/" + hostileEscaped + ": no such file or directory\n"},
		{"wrapped", fmt.Errorf("open snapshot: %w", path), "stateripple: open snapshot: open /reports/" + hostileEscaped + ": no such file or directory\n"},
		{"twice wrapped", fmt.Errorf("write report: %w", fmt.Errorf("create temporary output: %w", path)), "stateripple: write report: create temporary output: open /reports/" + hostileEscaped + ": no such file or directory\n"},
		{"link error", fmt.Errorf("publish new output without replacement: %w", link), "stateripple: publish new output without replacement: link /tmp/" + hostileEscaped + " /out/" + hostileEscaped + ": file exists\n"},
		{"joined", fmt.Errorf("write temporary output: %w", errors.Join(path, link)), "stateripple: write temporary output: open /reports/" + hostileEscaped + `: no such file or directory\nlink /tmp/` + hostileEscaped + " /out/" + hostileEscaped + ": file exists\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stderr bytes.Buffer
			diagnose(&stderr, "%v", tc.err)
			requireOneSafeLine(t, stderr.String())
			if stderr.String() != tc.want {
				t.Fatalf("got  %q\nwant %q", stderr.String(), tc.want)
			}
		})
	}
}

func TestRunEscapesHostileArguments(t *testing.T) {
	seed := `{"schema_version":1,"ruleset":"risk-propagation-v1","nodes":[],"edges":[],"seeds":[{"id":"x\n\u001b[2J\u202e","risk":1,"reason":"r"}]}`
	snapshot := filepath.Join(t.TempDir(), "snapshot.json")
	if err := os.WriteFile(snapshot, []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
		line string // complete first stderr line; a prefix when it ends an OS-specific message
		rest string
	}{
		{"missing input", []string{"simulate", "--input", hostile + ".json"}, "stateripple: open snapshot: stat " + hostileEscaped + ".json: ", ""},
		{"unknown command", []string{hostile}, `stateripple: unknown command "a\nstateripple: forged\r\x1b[2J\x00\a\x7f\u0085\u009b\u2028\u2029\u202e\u2066\u200b\ufeff\xff"` + "\n", help},
		{"unknown flag", []string{"simulate", "--" + hostile}, "flag provided but not defined: -" + hostileEscaped + "\n", simulateUsage},
		{"bad flag syntax", []string{"analyze", "-=" + hostile}, "bad flag syntax: -=" + hostileEscaped + "\n", analyzeUsage},
		{"quoted snapshot value", []string{"simulate", "--input", snapshot}, `stateripple: read snapshot: seed node not found: "x\n\x1b[2J\u202e"` + "\n", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(context.Background(), tc.args, &stdout, &stderr)
			first, rest, _ := strings.Cut(stderr.String(), "\n")
			requireOneSafeLine(t, first+"\n")
			if code != 1 || stdout.Len() != 0 || !strings.HasPrefix(first+"\n", tc.line) || rest != tc.rest {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
}

// brokenWriter accepts limit bytes and then fails; with short set it reports a
// partial write without an error instead.
type brokenWriter struct {
	limit   int
	short   bool
	written bytes.Buffer
}

func (w *brokenWriter) Write(p []byte) (int, error) {
	room := min(max(w.limit-w.written.Len(), 0), len(p))
	w.written.Write(p[:room])
	if room == len(p) {
		return room, nil
	}
	if w.short {
		return room, nil
	}
	return room, errInjectedWrite
}

func TestHelpAndUsageTextAndStreams(t *testing.T) {
	for _, tc := range []struct {
		args           []string
		code           int
		stdout, stderr string
	}{
		{[]string{"help"}, 0, help, ""},
		{[]string{"--help"}, 0, help, ""},
		{[]string{"-h"}, 0, help, ""},
		{[]string{"simulate", "--help"}, 0, simulateUsage, ""},
		{[]string{"simulate", "-h"}, 0, simulateUsage, ""},
		{[]string{"analyze", "--help"}, 0, analyzeUsage, ""},
		{[]string{"analyze", "-h"}, 0, analyzeUsage, ""},
		{[]string{"simulate", "--input", "x.json", "--help"}, 0, "", simulateUsage},
		{[]string{"analyze", "-help"}, 0, "", analyzeUsage},
		{[]string{"simulate", "--unknown"}, 1, "", "flag provided but not defined: -unknown\n" + simulateUsage},
		{[]string{"analyze", "--base"}, 1, "", "flag needs an argument: -base\n" + analyzeUsage},
		{[]string{}, 1, "", help},
		{[]string{"version"}, 0, "stateripple v0.1.0\n", ""},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(context.Background(), tc.args, &stdout, &stderr)
			if code != tc.code || stdout.String() != tc.stdout || stderr.String() != tc.stderr {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestHelpWriteFailureExitsOne(t *testing.T) {
	failures := []struct {
		name   string
		writer func() *brokenWriter
		want   error
	}{
		{"first write fails", func() *brokenWriter { return &brokenWriter{} }, errInjectedWrite},
		{"fails midway", func() *brokenWriter { return &brokenWriter{limit: 10} }, errInjectedWrite},
		{"short write", func() *brokenWriter { return &brokenWriter{limit: 10, short: true} }, io.ErrShortWrite},
	}
	for _, args := range [][]string{{"help"}, {"--help"}, {"-h"}, {"simulate", "--help"}, {"simulate", "-h"}, {"analyze", "--help"}, {"analyze", "-h"}, {"version"}} {
		for _, failure := range failures {
			t.Run(strings.Join(args, " ")+"/"+failure.name, func(t *testing.T) {
				stdout := failure.writer()
				var stderr bytes.Buffer
				code := Run(context.Background(), args, stdout, &stderr)
				if want := "stateripple: write output: " + failure.want.Error() + "\n"; code != 1 || stderr.String() != want {
					t.Fatalf("code=%d stderr=%q, want exit 1 and %q", code, stderr.String(), want)
				}
				if stdout.written.Len() > stdout.limit {
					t.Fatalf("wrote %d bytes past the failure", stdout.written.Len()-stdout.limit)
				}
			})
		}
	}
	// Usage requested together with other flags goes to stderr; losing it is a failure too.
	for _, failure := range failures {
		t.Run("usage on stderr/"+failure.name, func(t *testing.T) {
			var stdout bytes.Buffer
			code := Run(context.Background(), []string{"simulate", "--input", "x.json", "--help"}, &stdout, failure.writer())
			if code != 1 || stdout.Len() != 0 {
				t.Fatalf("code=%d stdout=%q", code, stdout.String())
			}
		})
	}
}
