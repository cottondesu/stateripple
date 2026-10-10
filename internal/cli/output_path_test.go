package cli_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestExecutableRejectsDirectoryOutputSpellings(t *testing.T) {
	bin := buildBinary(t)
	input, err := filepath.Abs("../../testdata/example.json")
	if err != nil {
		t.Fatal(err)
	}
	sep := string(filepath.Separator)
	for _, format := range []string{"json", "text"} {
		for _, path := range []string{"existing-dir", "existing-dir" + sep, "existing-dir" + sep + sep + sep, "dir-link", "dir-link" + sep, "dangling", ".", "." + sep, "..", ".." + sep, "missing" + sep} {
			t.Run(format+"/"+path, func(t *testing.T) {
				root := t.TempDir()
				if err := os.Mkdir(filepath.Join(root, "existing-dir"), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "existing-dir", "sentinel"), []byte("preserve"), 0o600); err != nil {
					t.Fatal(err)
				}
				for name, target := range map[string]string{"dir-link": "existing-dir", "dangling": "missing-target"} {
					if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
						t.Fatal(err)
					}
				}
				before := outputEntries(t, root)
				cmd := exec.Command(bin, "simulate", "--input", input, "--format", format, "--out", path)
				cmd.Dir = root
				data, err := cmd.CombinedOutput()
				if err == nil || cmd.ProcessState.ExitCode() != 1 {
					t.Fatalf("exit=%v output=%s", err, data)
				}
				if after := outputEntries(t, root); !reflect.DeepEqual(before, after) {
					t.Fatalf("directory changed: before=%v after=%v", before, after)
				}
				if data, err := os.ReadFile(filepath.Join(root, "existing-dir", "sentinel")); err != nil || string(data) != "preserve" {
					t.Fatalf("existing bytes changed: %q %v", data, err)
				}
			})
		}
	}
}

func outputEntries(t *testing.T, root string) []string {
	t.Helper()
	var entries []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		entries = append(entries, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func TestExecutableResolvesOutputParentBeforeDotDot(t *testing.T) {
	bin := buildBinary(t)
	input, err := filepath.Abs("../../testdata/example.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"json", "text"} {
		for _, missing := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/missing=%v", format, missing), func(t *testing.T) {
				root := t.TempDir()
				if err := os.MkdirAll(filepath.Join(root, "deep", "x", "y"), 0o700); err != nil {
					t.Fatal(err)
				}
				if !missing {
					if err := os.Symlink(filepath.Join("deep", "x", "y"), filepath.Join(root, "parent")); err != nil {
						t.Fatal(err)
					}
				}
				cmd := exec.Command(bin, "simulate", "--input", input, "--format", format, "--out", "parent/../report.out")
				cmd.Dir = root
				out, err := cmd.CombinedOutput()
				if (err != nil) != missing {
					t.Fatalf("err=%v output=%s", err, out)
				}
				if _, err := os.Stat(filepath.Join(root, "report.out")); !os.IsNotExist(err) {
					t.Fatalf("lexical destination created: %v", err)
				}
				if !missing {
					if _, err := os.Stat(filepath.Join(root, "deep", "x", "report.out")); err != nil {
						t.Fatal(err)
					}
				}
			})
		}
	}
}
