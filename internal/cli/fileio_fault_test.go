package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type faultTemp struct {
	*os.File
	stage string
	err   error
}

func (f faultTemp) Write(p []byte) (int, error) {
	if f.stage == "write" {
		return 0, f.err
	}
	return f.File.Write(p)
}
func (f faultTemp) Chmod(mode os.FileMode) error {
	if f.stage == "chmod" {
		return f.err
	}
	return f.File.Chmod(mode)
}
func (f faultTemp) Sync() error {
	if f.stage == "sync" {
		return f.err
	}
	return f.File.Sync()
}
func (f faultTemp) Close() error {
	err := f.File.Close()
	if f.stage == "close" {
		return errors.Join(err, f.err)
	}
	return err
}

func TestReportFailureStagesNeverPublishPartialOutput(t *testing.T) {
	for _, stage := range []string{"create", "write", "chmod", "sync", "close", "publish"} {
		t.Run(stage, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "report.json")
			injected := errors.New("injected " + stage)
			create := func(dir, pattern string) (reportTempFile, error) {
				if stage == "create" {
					return nil, injected
				}
				f, err := os.CreateTemp(dir, pattern)
				if err != nil {
					return nil, err
				}
				return faultTemp{File: f, stage: stage, err: injected}, nil
			}
			publish := func(from, to string) error {
				if stage == "publish" {
					return injected
				}
				return os.Link(from, to)
			}
			err := writeReportFileWith(path, []byte("complete report"), nil, create, os.Remove, publish)
			if !errors.Is(err, injected) {
				t.Fatalf("error=%v", err)
			}
			if _, err := os.Lstat(path); !os.IsNotExist(err) {
				t.Fatalf("output exists: %v", err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 0 {
				t.Fatalf("cleanup: %v %v", entries, err)
			}
		})
	}
}

func TestReportPublicationRejectsRacingDestination(t *testing.T) {
	for _, kind := range []string{"regular", "symlink", "dangling", "hardlink", "directory"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "report.json")
			target := filepath.Join(dir, "protected")
			if err := os.WriteFile(target, []byte("protected bytes"), 0o640); err != nil {
				t.Fatal(err)
			}
			create := func(dir, pattern string) (reportTempFile, error) { return os.CreateTemp(dir, pattern) }
			publish := func(from, to string) error {
				var err error
				switch kind {
				case "regular":
					err = os.WriteFile(to, []byte("racer bytes"), 0o600)
				case "symlink":
					err = os.Symlink(target, to)
				case "dangling":
					err = os.Symlink(target+"-missing", to)
				case "hardlink":
					err = os.Link(target, to)
				case "directory":
					err = os.Mkdir(to, 0o700)
				}
				if err != nil {
					t.Fatal(err)
				}
				return os.Link(from, to)
			}
			err := writeReportFileWith(path, []byte("new report"), nil, create, os.Remove, publish)
			if !errors.Is(err, os.ErrExist) {
				t.Fatalf("racer replaced: %v", err)
			}
			got, err := os.ReadFile(target)
			if err != nil || string(got) != "protected bytes" {
				t.Fatalf("target changed: %q %v", got, err)
			}
			if kind == "regular" {
				got, err := os.ReadFile(path)
				if err != nil || string(got) != "racer bytes" {
					t.Fatalf("racer changed: %q %v", got, err)
				}
			}
			temps, err := filepath.Glob(filepath.Join(dir, ".report.json.tmp-*"))
			if err != nil || len(temps) != 0 {
				t.Fatalf("temporary files: %v %v", temps, err)
			}
		})
	}
}

func TestReportCleanupFailureReportsPublishedOutput(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "report.json")
	create := func(dir, pattern string) (reportTempFile, error) { return os.CreateTemp(dir, pattern) }
	remove := func(string) error { return errInjectedCleanup }
	err = writeReportFileWith(path, []byte("complete report"), nil, create, remove, os.Link)
	if !errors.Is(err, errInjectedCleanup) || !strings.Contains(err.Error(), "output published at "+path) {
		t.Fatalf("error=%v", err)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(got, []byte("complete report")) {
		t.Fatalf("output=%q error=%v", got, readErr)
	}
	info, statErr := os.Stat(path)
	if statErr != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode: %v %v", info, statErr)
	}
}

func TestReportRejectsChangedParentDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "output")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	create := func(dir, pattern string) (reportTempFile, error) {
		f, err := os.CreateTemp(dir, pattern)
		if err != nil {
			return nil, err
		}
		if err := os.Rename(dir, dir+"-moved"); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		return f, nil
	}
	err := writeReportFileWith(filepath.Join(dir, "report.json"), []byte("report"), nil, create, os.Remove, os.Link)
	if err == nil || !strings.Contains(err.Error(), "directory changed") {
		t.Fatalf("error=%v", err)
	}
	if _, err := os.Lstat(filepath.Join(dir, "report.json")); !os.IsNotExist(err) {
		t.Fatalf("published: %v", err)
	}
}
