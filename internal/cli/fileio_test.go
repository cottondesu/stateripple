package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

var errInjectedWrite = errors.New("injected write failure")
var errInjectedCleanup = errors.New("injected cleanup failure")

const emptySnapshotJSON = `{"schema_version":1,"ruleset":"risk-propagation-v1","nodes":[],"edges":[],"seeds":[]}`

type failingTempFile struct {
	*os.File
}

func (f failingTempFile) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, errInjectedWrite
	}
	n, _ := f.File.Write(p[:1])
	return n, errInjectedWrite
}

func TestWriteReportFileCleansTemporaryOutputWhenWriteFails(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "report.json")
	want := []byte("existing report\n")
	protected := filepath.Join(dir, "existing.json")
	if err := os.WriteFile(protected, want, 0o600); err != nil {
		t.Fatal(err)
	}
	createFailingTemp := func(dir, pattern string) (reportTempFile, error) {
		f, err := os.CreateTemp(dir, pattern)
		if err != nil {
			return nil, err
		}
		return failingTempFile{File: f}, nil
	}

	err := writeReportFileWith(out, []byte("replacement"), nil, createFailingTemp, os.Remove, os.Link)

	got, readErr := os.ReadFile(protected)
	if readErr != nil {
		t.Fatal(readErr)
	}
	temps, globErr := filepath.Glob(filepath.Join(dir, ".report.json.tmp-*"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	_, statErr := os.Lstat(out)
	if !errors.Is(err, errInjectedWrite) || string(got) != string(want) || len(temps) != 0 || !os.IsNotExist(statErr) {
		t.Fatalf("err=%v output=%q temporary_files=%v", err, got, temps)
	}
}

func TestWriteReportFileReportsWriteAndCleanupFailuresWithoutChangingOutput(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "report.json")
	want := []byte("existing report\n")
	protected := filepath.Join(dir, "existing.json")
	if err := os.WriteFile(protected, want, 0o600); err != nil {
		t.Fatal(err)
	}
	createFailingTemp := func(dir, pattern string) (reportTempFile, error) {
		f, err := os.CreateTemp(dir, pattern)
		if err != nil {
			return nil, err
		}
		return failingTempFile{File: f}, nil
	}
	removeFailingTemp := func(string) error {
		return errInjectedCleanup
	}

	err := writeReportFileWith(out, []byte("replacement"), nil, createFailingTemp, removeFailingTemp, os.Link)

	got, readErr := os.ReadFile(protected)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !errors.Is(err, errInjectedWrite) || !errors.Is(err, errInjectedCleanup) || string(got) != string(want) {
		t.Fatalf("err=%v output=%q", err, got)
	}
}

func TestWriteReportFileRejectsCapturedInputIdentityAfterPathMove(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "snapshot.json")
	if err := os.WriteFile(input, []byte(emptySnapshotJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(input)
	if err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(dir, "moved-snapshot.json")
	if err := os.Rename(input, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("replacement path"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "report.json")
	if err := os.Link(moved, out); err != nil {
		t.Fatal(err)
	}
	ref := &inputFileReference{path: input, info: info}

	err = writeReportFile(out, []byte("report"), ref)

	got, readErr := os.ReadFile(moved)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if err == nil || string(got) != emptySnapshotJSON {
		t.Fatalf("err=%v captured_input=%q", err, got)
	}
}
