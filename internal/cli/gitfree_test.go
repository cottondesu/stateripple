package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cottondesu/stateripple/internal/report"
)

// Pinned before the generic graph change. A legal snapshot must keep its
// normalized hash and its report bytes.
var pinnedGoldens = []struct {
	input, report, inputSHA256, reportSHA256 string
}{
	{"example.json", "example.report.json", "e0c96ca23a1e63430c810825ffd4161ecb4a26c4835398cad2e0b66b072acde8", "809eb4c476c5a45e7d5f764b443fd269aa5cdef0e9865d421d43aff554edf36a"},
	{"cli-empty.json", "cli-empty.golden.json", "a246bcb6c6b2cf0244b5c9d2ba58e6909a89c5bf6527ffb7fcd8626e642e6818", "c215f338ecb25787635a91d230a58eb3e56fc082541b2f3179bc5518eae1bda2"},
	{"generic-supply-chain.json", "generic-supply-chain.report.json", "c34ed81c120cb7e518ee6c407ef1f94c0a77f7cc53819cc76157ffc8db3093de", "ebbb8c55a296b17e3b7e663f6730b58a7f11070b3f313eaa5993bd27b16a854a"},
}

type execResult struct {
	code           int
	stdout, stderr []byte
}

// runIsolated runs the binary with PATH pointing at an empty directory, so no
// git or go executable can be found, and with no other inherited variables.
func runIsolated(t *testing.T, bin, dir string, args ...string) execResult {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + t.TempDir(), "HOME=" + t.TempDir()}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatal(err)
		}
		code = exit.ExitCode()
	}
	return execResult{code, stdout.Bytes(), stderr.Bytes()}
}

func testdata(t *testing.T, name string) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("../../testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func decodeReport(t *testing.T, b []byte) report.Report {
	t.Helper()
	var r report.Report
	if err := json.Unmarshal(b, &r); err != nil {
		t.Fatalf("%v: %s", err, b)
	}
	return r
}

func TestGitFreeAcceptance(t *testing.T) {
	bin := buildBinary(t)
	generic := func(edges string) string {
		return inputFile(t, `{"schema_version":1,"ruleset":"risk-propagation-v1","nodes":[{"id":"src","kind":"sensor"},{"id":"dst","kind":"アクチュエータ"}],"edges":[`+edges+`],"seeds":[{"id":"src","risk":3,"reason":"alarm"}]}`)
	}
	chain := func(n, limit int) string {
		var b strings.Builder
		fmt.Fprintf(&b, `{"schema_version":1,"ruleset":"risk-propagation-v1","max_generations":%d,"nodes":[`, limit)
		for i := 0; i < n; i++ {
			if i > 0 {
				b.WriteByte(',')
			}
			fmt.Fprintf(&b, `{"id":"n%05d","kind":"stage"}`, i)
		}
		b.WriteString(`],"edges":[`)
		for i := 0; i+1 < n; i++ {
			if i > 0 {
				b.WriteByte(',')
			}
			fmt.Fprintf(&b, `{"from":"n%05d","to":"n%05d","kind":"feeds","attenuation":0}`, i, i+1)
		}
		b.WriteString(`],"seeds":[{"id":"n00000","risk":3,"reason":"start"}]}`)
		return inputFile(t, b.String())
	}
	golden := func(t *testing.T, dir string) {
		t.Helper()
		for _, g := range pinnedGoldens {
			res := runIsolated(t, bin, dir, "simulate", "--input", testdata(t, g.input), "--format", "json")
			want, err := os.ReadFile(testdata(t, g.report))
			if err != nil {
				t.Fatal(err)
			}
			if res.code != 0 || len(res.stderr) != 0 || !bytes.Equal(res.stdout, want) {
				t.Fatalf("%s: code=%d stderr=%s stdout=%s", g.input, res.code, res.stderr, res.stdout)
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(res.stdout)); got != g.reportSHA256 {
				t.Fatalf("%s report sha256 %s", g.input, got)
			}
			if r := decodeReport(t, res.stdout); r.InputSHA256 != g.inputSHA256 {
				t.Fatalf("%s input_sha256 %s", g.input, r.InputSHA256)
			}
		}
	}

	t.Run("GF-01 no .git directory", func(t *testing.T) {
		golden(t, t.TempDir())
	})
	t.Run("GF-02 git init without HEAD", func(t *testing.T) {
		git, err := exec.LookPath("git")
		if err != nil {
			t.Skip("git unavailable for fixture setup")
		}
		dir := t.TempDir()
		if b, err := exec.Command(git, "init", "--quiet", dir).CombinedOutput(); err != nil {
			t.Fatalf("%v %s", err, b)
		}
		if err := exec.Command(git, "-C", dir, "rev-parse", "--verify", "HEAD").Run(); err == nil {
			t.Fatal("fixture unexpectedly has HEAD")
		}
		golden(t, dir)
		cmd := exec.Command(bin, "simulate", "--input", testdata(t, "example.json"), "--format", "json")
		cmd.Dir = dir
		cmd.Env = childEnv()
		if out, err := cmd.Output(); err != nil || !bytes.Equal(out, mustRead(t, testdata(t, "example.report.json"))) {
			t.Fatalf("with git on PATH: %v", err)
		}
	})
	t.Run("GF-03 GF-04 no git or go executable", func(t *testing.T) {
		dir := t.TempDir()
		empty := t.TempDir()
		for _, name := range []string{"git", "go"} {
			if _, err := os.Stat(filepath.Join(empty, name)); err == nil {
				t.Fatal("isolated PATH is not empty")
			}
		}
		golden(t, dir)
	})
	t.Run("GF-05 help and version", func(t *testing.T) {
		for _, args := range [][]string{{"help"}, {"--help"}, {"-h"}, {"version"}, {"simulate", "--help"}, {"analyze", "--help"}} {
			res := runIsolated(t, bin, t.TempDir(), args...)
			if res.code != 0 || len(res.stdout) == 0 || len(res.stderr) != 0 {
				t.Fatalf("%v: code=%d stdout=%s stderr=%s", args, res.code, res.stdout, res.stderr)
			}
		}
		if res := runIsolated(t, bin, t.TempDir(), "version"); string(res.stdout) != "stateripple v0.1.0\n" {
			t.Fatalf("version %q", res.stdout)
		}
		if res := runIsolated(t, bin, t.TempDir(), "help"); !bytes.HasPrefix(res.stdout, []byte("StateRipple v0.1.0")) || !bytes.Contains(res.stdout, []byte("stateripple simulate")) {
			t.Fatalf("help %s", res.stdout)
		}
	})
	t.Run("GF-06 GF-07 GF-08 custom kinds and attenuation 0", func(t *testing.T) {
		res := runIsolated(t, bin, t.TempDir(), "simulate", "--input", generic(`{"from":"src","to":"dst","kind":"actuates","attenuation":0}`), "--format", "json")
		r := decodeReport(t, res.stdout)
		if res.code != 0 || r.Status != "complete" || r.Cells[0].ID != "dst" || r.Cells[0].Kind != "アクチュエータ" || r.Cells[0].Risk != 3 || r.Cells[0].Causes[0].Reason != "actuates" || len(r.TestTargets) != 0 {
			t.Fatalf("code=%d report=%+v stderr=%s", res.code, r, res.stderr)
		}
	})
	t.Run("GF-09 attenuation 3", func(t *testing.T) {
		res := runIsolated(t, bin, t.TempDir(), "simulate", "--input", generic(`{"from":"src","to":"dst","kind":"actuates","attenuation":3}`), "--format", "json")
		r := decodeReport(t, res.stdout)
		if res.code != 0 || r.Cells[0].Risk != 0 || r.GenerationCount != 1 || r.Cells[0].FirstActiveGeneration != nil {
			t.Fatalf("code=%d report=%+v", res.code, r)
		}
	})
	t.Run("GF-10 attenuation 4 and 5", func(t *testing.T) {
		for _, a := range []string{"4", "5", "-1", "2.0", `"1"`, "99999999999999999999"} {
			res := runIsolated(t, bin, t.TempDir(), "simulate", "--input", generic(`{"from":"src","to":"dst","kind":"actuates","attenuation":`+a+`}`), "--format", "json")
			if res.code != 1 || len(res.stdout) != 0 || !bytes.Contains(res.stderr, []byte("attenuation")) {
				t.Fatalf("attenuation %s: code=%d stdout=%s stderr=%s", a, res.code, res.stdout, res.stderr)
			}
		}
	})
	t.Run("GF-11 long zero-attenuation chain", func(t *testing.T) {
		for _, tt := range []struct {
			n, code     int
			status      string
			generations int
		}{{1024, 0, "complete", 1024}, {1025, 2, "incomplete", 1024}, {1500, 2, "incomplete", 1024}} {
			res := runIsolated(t, bin, t.TempDir(), "simulate", "--input", chain(tt.n, 1024), "--format", "json")
			r := decodeReport(t, res.stdout)
			wantStop := map[string]string{"complete": "converged", "incomplete": "generation_limit"}[tt.status]
			if res.code != tt.code || r.Status != tt.status || r.StopReason != wantStop || r.GenerationCount != tt.generations {
				t.Fatalf("n=%d code=%d status=%s stop=%s generations=%d", tt.n, res.code, r.Status, r.StopReason, r.GenerationCount)
			}
		}
	})
	t.Run("GF-12 analyze without git", func(t *testing.T) {
		res := runIsolated(t, bin, t.TempDir(), "analyze", "--format", "json")
		if res.code != 1 || len(res.stdout) != 0 || !bytes.Contains(res.stderr, []byte("analyze requires Git 2.45.0 or later")) {
			t.Fatalf("code=%d stdout=%s stderr=%s", res.code, res.stdout, res.stderr)
		}
	})
	t.Run("GF-13 GF-14 repeated runs are byte identical", func(t *testing.T) {
		input := generic(`{"from":"src","to":"dst","kind":"actuates","attenuation":1}`)
		first := runIsolated(t, bin, t.TempDir(), "simulate", "--input", input, "--format", "json")
		for i := 0; i < 25; i++ {
			again := runIsolated(t, bin, t.TempDir(), "simulate", "--input", input, "--format", "json")
			if again.code != 0 || !bytes.Equal(again.stdout, first.stdout) {
				t.Fatalf("run %d differs", i)
			}
		}
	})
	t.Run("GF-15 non-software graph text", func(t *testing.T) {
		res := runIsolated(t, bin, t.TempDir(), "simulate", "--input", testdata(t, "generic-supply-chain.json"), "--format", "text")
		text := string(res.stdout)
		if res.code != 0 || !strings.Contains(text, "store risk=1 first_active=4") || !strings.Contains(text, "Risk is a propagated priority, not a probability.") {
			t.Fatalf("code=%d text=%s", res.code, text)
		}
		for _, software := range []string{"Changed files", "Recommended test targets", "Selected tests", "Git"} {
			if strings.Contains(text, software) {
				t.Fatalf("generic text mentions %q:\n%s", software, text)
			}
		}
	})
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
