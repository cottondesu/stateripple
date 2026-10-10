package fileio

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRegularReturnsOpenedFileIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input.json")
	want := []byte("input")
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatal(err)
	}

	f, info, err := OpenRegular(path)
	if err != nil {
		t.Fatal(err)
	}
	got, readErr := io.ReadAll(f)
	closeErr := f.Close()
	pathInfo, statErr := os.Stat(path)

	if readErr != nil || closeErr != nil || statErr != nil || string(got) != string(want) || !os.SameFile(info, pathInfo) {
		t.Fatalf("read_err=%v close_err=%v stat_err=%v content=%q same_file=%t", readErr, closeErr, statErr, got, statErr == nil && os.SameFile(info, pathInfo))
	}
}

func TestOpenRegularRejectsDirectory(t *testing.T) {
	f, info, err := OpenRegular(t.TempDir())

	if err == nil || f != nil || info != nil {
		t.Fatalf("file=%v info=%v err=%v", f, info, err)
	}
}
