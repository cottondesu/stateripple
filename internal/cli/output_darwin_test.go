package cli_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/cottondesu/stateripple/internal/cli"
)

func TestExistingOutputRetainsDarwinACLAndXattr(t *testing.T) {
	for _, kind := range []string{"acl", "xattr"} {
		t.Run(kind, func(t *testing.T) {
			input := inputFile(t, emptySnapshot)
			out := filepath.Join(t.TempDir(), "protected.json")
			want := []byte("protected output\n")
			if err := os.WriteFile(out, want, 0o640); err != nil {
				t.Fatal(err)
			}
			before, err := os.Stat(out)
			if err != nil {
				t.Fatal(err)
			}
			var inspect func() ([]byte, error)
			if kind == "acl" {
				if data, err := exec.Command("chmod", "+a", "everyone deny read", out).CombinedOutput(); err != nil {
					t.Skipf("ACL unavailable: %v %s", err, data)
				}
				t.Cleanup(func() {
					if data, err := exec.Command("chmod", "-N", out).CombinedOutput(); err != nil {
						t.Errorf("ACL cleanup: %v %s", err, data)
					}
				})
				inspect = func() ([]byte, error) { return exec.Command("ls", "-lde", out).Output() }
			} else {
				if data, err := exec.Command("xattr", "-w", "com.stateripple.audit", "preserve-me", out).CombinedOutput(); err != nil {
					t.Skipf("xattr unavailable: %v %s", err, data)
				}
				inspect = func() ([]byte, error) { return exec.Command("xattr", "-p", "com.stateripple.audit", out).Output() }
			}
			metadata, err := inspect()
			if err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			code := cli.Run(context.Background(), []string{"simulate", "--input", input, "--format", "json", "--out", out}, &stdout, &stderr)
			afterMetadata, err := inspect()
			if err != nil {
				t.Fatal(err)
			}
			after, err := os.Stat(out)
			if err != nil {
				t.Fatal(err)
			}
			if code != 1 || stdout.Len() != 0 || stderr.Len() == 0 || !os.SameFile(before, after) || !bytes.Equal(metadata, afterMetadata) || after.Mode() != before.Mode() {
				t.Fatalf("existing %s changed; exit=%d stderr=%s", kind, code, &stderr)
			}
			if kind == "acl" {
				if data, err := exec.Command("chmod", "-N", out).CombinedOutput(); err != nil {
					t.Fatalf("remove tested ACL before byte inspection: %v %s", err, data)
				}
			}
			got, err := os.ReadFile(out)
			if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("bytes changed: %q %v", got, err)
			}
		})
	}
}
