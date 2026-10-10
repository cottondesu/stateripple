//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package cli_test

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/cottondesu/stateripple/internal/cli"
)

// A file name that every supported file system accepts and that would forge a
// log line, clear a terminal or reorder text when shown raw.
const hostileName = "a\nstateripple: forged\r\x1b[2J\a\x7f\u0085\u009b\u2028\u2029\u202e\u2066\u200b\ufeff"
const hostileNameEscaped = `a\nstateripple: forged\r\x1b[2J\a\x7f\u0085\u009b\u2028\u2029\u202e\u2066\u200b\ufeff`

func TestPathDiagnosticsEscapeHostileFileNames(t *testing.T) {
	const kept = "existing data\n"
	write := func(t *testing.T, path, data string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// arrange creates the case below dir, whose in.json is a valid snapshot, and
	// returns the --input and --out arguments ("" omits --out).
	for _, tc := range []struct {
		name    string
		arrange func(t *testing.T, dir string) (input, out string)
		want    string
	}{
		{"missing input", func(t *testing.T, dir string) (string, string) {
			return filepath.Join(dir, hostileName), ""
		}, "open snapshot: stat "},
		{"directory input", func(t *testing.T, dir string) (string, string) {
			if err := os.Mkdir(filepath.Join(dir, hostileName), 0o700); err != nil {
				t.Fatal(err)
			}
			return filepath.Join(dir, hostileName), ""
		}, "open snapshot: input must be a regular file: "},
		{"dangling symlink input", func(t *testing.T, dir string) (string, string) {
			if err := os.Symlink(filepath.Join(dir, "gone"), filepath.Join(dir, hostileName)); err != nil {
				t.Fatal(err)
			}
			return filepath.Join(dir, hostileName), ""
		}, "open snapshot: stat "},
		{"existing output", func(t *testing.T, dir string) (string, string) {
			write(t, filepath.Join(dir, hostileName), kept)
			return filepath.Join(dir, "in.json"), filepath.Join(dir, hostileName)
		}, "write report: output already exists; refusing to overwrite: "},
		{"symlink output", func(t *testing.T, dir string) (string, string) {
			write(t, filepath.Join(dir, "target"), kept)
			if err := os.Symlink(filepath.Join(dir, "target"), filepath.Join(dir, hostileName)); err != nil {
				t.Fatal(err)
			}
			return filepath.Join(dir, "in.json"), filepath.Join(dir, hostileName)
		}, "write report: output must not be a symbolic link: "},
		{"directory output", func(t *testing.T, dir string) (string, string) {
			if err := os.Mkdir(filepath.Join(dir, hostileName), 0o700); err != nil {
				t.Fatal(err)
			}
			return filepath.Join(dir, "in.json"), filepath.Join(dir, hostileName)
		}, "write report: output must be a regular file: "},
		{"output is input", func(t *testing.T, dir string) (string, string) {
			if err := os.Rename(filepath.Join(dir, "in.json"), filepath.Join(dir, hostileName)); err != nil {
				t.Fatal(err)
			}
			return filepath.Join(dir, hostileName), filepath.Join(dir, hostileName)
		}, "write report: output aliases input: "},
		{"output hard link to input", func(t *testing.T, dir string) (string, string) {
			if err := os.Link(filepath.Join(dir, "in.json"), filepath.Join(dir, hostileName)); err != nil {
				t.Fatal(err)
			}
			return filepath.Join(dir, "in.json"), filepath.Join(dir, hostileName)
		}, "write report: output aliases input: "},
		{"missing output directory", func(t *testing.T, dir string) (string, string) {
			return filepath.Join(dir, "in.json"), filepath.Join(dir, hostileName, "out.json")
		}, "write report: resolve output directory: lstat "},
		{"output directory through dangling symlink", func(t *testing.T, dir string) (string, string) {
			// The resolver reports the link target, so the target carries the name.
			if err := os.Symlink(filepath.Join(dir, hostileName), filepath.Join(dir, "link")); err != nil {
				t.Fatal(err)
			}
			return filepath.Join(dir, "in.json"), filepath.Join(dir, "link", "out.json")
		}, "write report: resolve output directory: lstat "},
		{"unwritable output directory", func(t *testing.T, dir string) (string, string) {
			if os.Geteuid() == 0 {
				t.Skip("directory permissions do not restrict the superuser")
			}
			locked := filepath.Join(dir, hostileName)
			if err := os.Mkdir(locked, 0o500); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.Chmod(locked, 0o700) })
			return filepath.Join(dir, "in.json"), filepath.Join(locked, "out.json")
		}, "write report: create temporary output: open "},
		{"trailing separator", func(t *testing.T, dir string) (string, string) {
			return filepath.Join(dir, "in.json"), filepath.Join(dir, hostileName) + string(os.PathSeparator)
		}, "write report: output must name a file without a trailing path separator: "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given
			dir, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			write(t, filepath.Join(dir, "in.json"), emptySnapshot)
			input, out := tc.arrange(t, dir)
			args := []string{"simulate", "--input", input, "--format", "json"}
			if out != "" {
				args = append(args, "--out", out)
			}
			before := treeState(t, dir)
			// When
			var stdout, stderr bytes.Buffer
			code := cli.Run(context.Background(), args, &stdout, &stderr)
			// Then: exit 1, one escaped line naming the path, nothing created or changed.
			diagnostic := stderr.String()
			if code != 1 || stdout.Len() != 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), diagnostic)
			}
			body, terminated := strings.CutSuffix(diagnostic, "\n")
			if !terminated || strings.Contains(body, "\n") || !utf8.ValidString(body) {
				t.Fatalf("diagnostic is not one valid line: %q", diagnostic)
			}
			for _, r := range body {
				if !strconv.IsGraphic(r) {
					t.Fatalf("diagnostic contains raw %U: %q", r, diagnostic)
				}
			}
			if !strings.HasPrefix(diagnostic, "stateripple: "+tc.want) || !strings.Contains(diagnostic, hostileNameEscaped) {
				t.Fatalf("diagnostic %q, want prefix %q and the escaped name", diagnostic, "stateripple: "+tc.want)
			}
			if after := treeState(t, dir); after != before {
				t.Fatalf("files changed:\nbefore:\n%s\nafter:\n%s", before, after)
			}
		})
	}
}

// treeState lists every entry below dir with its type, permissions and, for
// regular files, content.
func treeState(t *testing.T, dir string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		b.WriteString(strconv.Quote(path) + " " + info.Mode().String())
		if info.Mode().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			b.WriteString(" " + strconv.Quote(string(data)))
		}
		b.WriteByte('\n')
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}
