package cli_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestExecutableConcurrentWritersPublishExactlyOnce(t *testing.T) {
	bin := buildBinary(t)
	input := inputFile(t, emptySnapshot)
	dir := t.TempDir()
	out := filepath.Join(dir, "report.json")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	want, err := exec.CommandContext(ctx, bin, "simulate", "--input", input, "--format", "json").Output()
	if err != nil {
		t.Fatal(err)
	}
	commands := make([]*exec.Cmd, 12)
	for i := range commands {
		commands[i] = exec.CommandContext(ctx, bin, "simulate", "--input", input, "--format", "json", "--out", out)
		if err := commands[i].Start(); err != nil {
			t.Fatal(err)
		}
	}
	successes := 0
	for _, cmd := range commands {
		err := cmd.Wait()
		if err == nil {
			successes++
		} else if cmd.ProcessState.ExitCode() != 1 {
			t.Fatalf("unexpected exit: %v", err)
		}
	}
	got, err := os.ReadFile(out)
	if err != nil || successes != 1 || !bytes.Equal(got, want) {
		t.Fatalf("successes=%d bytes_equal=%t error=%v", successes, bytes.Equal(got, want), err)
	}
	temps, err := filepath.Glob(filepath.Join(dir, ".report.json.tmp-*"))
	if err != nil || len(temps) != 0 {
		t.Fatalf("temporary files: %v %v", temps, err)
	}
}
