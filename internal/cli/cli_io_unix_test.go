//go:build aix || darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd || solaris

package cli_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestExecutableRejectsFIFOInputWithoutBlocking(t *testing.T) {
	bin := buildBinary(t)
	tests := []struct {
		name string
		path func(t *testing.T, fifo string) string
	}{
		{
			name: "fifo",
			path: func(_ *testing.T, fifo string) string {
				return fifo
			},
		},
		{
			name: "symlink to fifo",
			path: func(t *testing.T, fifo string) string {
				t.Helper()
				link := filepath.Join(t.TempDir(), "snapshot-link.json")
				if err := os.Symlink(fifo, link); err != nil {
					t.Fatal(err)
				}
				return link
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fifo := filepath.Join(t.TempDir(), "snapshot.fifo")
			if err := syscall.Mkfifo(fifo, 0o600); err != nil {
				t.Fatal(err)
			}
			input := tt.path(t, fifo)
			out := filepath.Join(t.TempDir(), "existing-report.json")
			want := []byte("existing report\n")
			if err := os.WriteFile(out, want, 0o600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, bin, "simulate", "--input", input, "--format", "json", "--out", out)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr

			err := cmd.Run()

			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				t.Fatalf("simulate blocked on %s", tt.name)
			}
			var exit *exec.ExitError
			got, readErr := os.ReadFile(out)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !errors.As(err, &exit) || exit.ExitCode() != 1 || stdout.Len() != 0 || stderr.Len() == 0 || !bytes.Equal(got, want) {
				t.Fatalf("err=%v stdout=%q stderr=%q output=%q", err, stdout.String(), stderr.String(), got)
			}
		})
	}
}

func TestExecutableRejectsOutputSymlinkWithoutBlocking(t *testing.T) {
	bin := buildBinary(t)
	input := inputFile(t, emptySnapshot)
	target := filepath.Join(t.TempDir(), "existing-report.json")
	want := []byte("existing report\n")
	if err := os.WriteFile(target, want, 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "report-link.json")
	if err := os.Symlink(target, out); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "simulate", "--input", input, "--format", "json", "--out", out)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	err := cmd.Run()

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatal("simulate blocked on output symlink")
	}
	var exit *exec.ExitError
	got, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !errors.As(err, &exit) || exit.ExitCode() != 1 || stdout.Len() != 0 || stderr.Len() == 0 || !bytes.Equal(got, want) {
		t.Fatalf("err=%v stdout=%q stderr=%q target=%q", err, stdout.String(), stderr.String(), got)
	}
}

func TestExecutableCreatesNewOutputWithPrivatePermissionsUnderRestrictiveUmask(t *testing.T) {
	bin := buildBinary(t)
	input := inputFile(t, emptySnapshot)
	out := filepath.Join(t.TempDir(), "report.json")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", `umask 077; exec "$@"`, "sh", bin, "simulate", "--input", input, "--format", "json", "--out", out)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	err := cmd.Run()

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatal("simulate timed out")
	}
	if err != nil || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("err=%v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	info, statErr := os.Stat(out)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%#o", info.Mode().Perm())
	}
}
