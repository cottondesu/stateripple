package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cottondesu/stateripple/internal/cli"
)

func TestSimulateRejectsDirectoryInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run(context.Background(), []string{"simulate", "--input", t.TempDir(), "--format", "json"}, &stdout, &stderr)

	if code != 1 || stdout.Len() != 0 || stderr.Len() == 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestSimulateRejectsOutputThatAliasesInput(t *testing.T) {
	tests := []struct {
		name  string
		alias func(t *testing.T, input string) string
	}{
		{
			name: "same path",
			alias: func(_ *testing.T, input string) string {
				return input
			},
		},
		{
			name: "hard link",
			alias: func(t *testing.T, input string) string {
				t.Helper()
				out := filepath.Join(t.TempDir(), "report.json")
				if err := os.Link(input, out); err != nil {
					t.Fatal(err)
				}
				return out
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := inputFile(t, emptySnapshot)
			before, err := os.ReadFile(input)
			if err != nil {
				t.Fatal(err)
			}
			out := tt.alias(t, input)

			var stdout, stderr bytes.Buffer
			code := cli.Run(context.Background(), []string{"simulate", "--input", input, "--format", "json", "--out", out}, &stdout, &stderr)

			after, err := os.ReadFile(input)
			if err != nil {
				t.Fatal(err)
			}
			if code != 1 || stdout.Len() != 0 || stderr.Len() == 0 || !bytes.Equal(after, before) {
				t.Fatalf("code=%d stdout=%q stderr=%q input_changed=%t", code, stdout.String(), stderr.String(), !bytes.Equal(after, before))
			}
		})
	}
}

func TestSimulateRejectsInputSymlinkAliasAsOutput(t *testing.T) {
	input := inputFile(t, emptySnapshot)
	inputLink := filepath.Join(t.TempDir(), "snapshot-link.json")
	if err := os.Symlink(input, inputLink); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := cli.Run(context.Background(), []string{"simulate", "--input", inputLink, "--format", "json", "--out", input}, &stdout, &stderr)

	after, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 || stdout.Len() != 0 || stderr.Len() == 0 || !bytes.Equal(after, before) {
		t.Fatalf("code=%d stdout=%q stderr=%q input_changed=%t", code, stdout.String(), stderr.String(), !bytes.Equal(after, before))
	}
}

func TestSimulateRejectsOutputSymlinkWithoutChangingTarget(t *testing.T) {
	input := inputFile(t, emptySnapshot)
	target := filepath.Join(t.TempDir(), "existing.json")
	want := []byte("existing report\n")
	if err := os.WriteFile(target, want, 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "report-link.json")
	if err := os.Symlink(target, out); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := cli.Run(context.Background(), []string{"simulate", "--input", input, "--format", "json", "--out", out}, &stdout, &stderr)

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 || stdout.Len() != 0 || stderr.Len() == 0 || !bytes.Equal(got, want) {
		t.Fatalf("code=%d stdout=%q stderr=%q target=%q", code, stdout.String(), stderr.String(), got)
	}
}

func TestSimulateRejectsExistingOutputAndPreservesIdentity(t *testing.T) {
	input := inputFile(t, emptySnapshot)
	out := filepath.Join(t.TempDir(), "report.json")
	if err := os.WriteFile(out, []byte("old report\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := cli.Run(context.Background(), []string{"simulate", "--input", input, "--format", "json", "--out", out}, &stdout, &stderr)

	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 || stdout.Len() != 0 || stderr.Len() == 0 || string(got) != "old report\n" || info.Mode().Perm() != 0o600 || !os.SameFile(before, info) {
		t.Fatalf("code=%d stdout=%q stderr=%q mode=%#o output=%q", code, stdout.String(), stderr.String(), info.Mode().Perm(), got)
	}
}

func TestSimulateRejectsDirectoryAsOutput(t *testing.T) {
	input := inputFile(t, emptySnapshot)
	out := t.TempDir()

	var stdout, stderr bytes.Buffer
	code := cli.Run(context.Background(), []string{"simulate", "--input", input, "--format", "json", "--out", out}, &stdout, &stderr)

	if code != 1 || stdout.Len() != 0 || stderr.Len() == 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestSimulateRejectsMissingOutputDirectory(t *testing.T) {
	input := inputFile(t, emptySnapshot)
	out := filepath.Join(t.TempDir(), "missing", "report.json")

	var stdout, stderr bytes.Buffer
	code := cli.Run(context.Background(), []string{"simulate", "--input", input, "--format", "json", "--out", out}, &stdout, &stderr)

	if code != 1 || stdout.Len() != 0 || stderr.Len() == 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
