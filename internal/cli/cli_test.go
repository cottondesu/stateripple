package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cottondesu/stateripple/internal/cli"
	"github.com/cottondesu/stateripple/internal/report"
)

const emptySnapshot = `{"schema_version":1,"ruleset":"risk-propagation-v1","nodes":[],"edges":[],"seeds":[]}`

func inputFile(t *testing.T, data string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "snapshot.json")
	if err := os.WriteFile(p, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run(context.Background(), []string{"version"}, &stdout, &stderr)
	if code != 0 || stdout.String() != "stateripple v0.1.0\n" || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestSimulateGoldenJSON(t *testing.T) {
	want, err := os.ReadFile("../../testdata/cli-empty.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := cli.Run(context.Background(), []string{"simulate", "--input", "../../testdata/cli-empty.json", "--format", "json"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 || !bytes.Equal(stdout.Bytes(), want) {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
}

func TestHelpUsesStdout(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"simulate", "--help"}, {"analyze", "--help"}} {
		var stdout, stderr bytes.Buffer
		code := cli.Run(context.Background(), args, &stdout, &stderr)
		if code != 0 || stdout.Len() == 0 || stderr.Len() != 0 {
			t.Fatalf("args=%v code=%d stdout=%s stderr=%s", args, code, stdout.String(), stderr.String())
		}
	}
}

func TestInvalidArgumentsSeparateDiagnostics(t *testing.T) {
	for _, args := range [][]string{{}, {"bogus"}, {"version", "extra"}, {"simulate"}, {"simulate", "--unknown"}, {"simulate", "--format", "xml"}, {"simulate", "extra"}, {"analyze", "--input", "file"}, {"simulate", "--input", "missing"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := cli.Run(context.Background(), args, &stdout, &stderr)
			if code != 1 || stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestSimulateRejectsMalformedSnapshot(t *testing.T) {
	for _, data := range []string{"{", `{"schema_version":2,"ruleset":"risk-propagation-v1","nodes":[],"edges":[],"seeds":[]}`, `{"schema_version":1,"ruleset":"unknown","nodes":[],"edges":[],"seeds":[]}`, `{"schema_version":1,"ruleset":"risk-propagation-v1","nodes":[{"id":"","kind":"package"}],"edges":[],"seeds":[]}`, `{"schema_version":1,"ruleset":"risk-propagation-v1","nodes":[],"edges":[{"from":"a","to":"b","kind":"prod_import","attenuation":1}],"seeds":[]}`} {
		t.Run(data, func(t *testing.T) {
			p := inputFile(t, data)
			var stdout, stderr bytes.Buffer
			code := cli.Run(context.Background(), []string{"simulate", "--input", p, "--format", "json"}, &stdout, &stderr)
			if code != 1 || stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
}

// Every input starts from a snapshot that simulates successfully, so exit 1 can
// only come from the one field that the case removes or sets to null.
func TestSimulateRejectsOmittedOrNullField(t *testing.T) {
	const valid = `{"schema_version":1,"ruleset":"risk-propagation-v1","nodes":[{"id":"a","kind":"service"},{"id":"b","kind":"store"}],"edges":[{"from":"a","to":"b","kind":"calls","attenuation":2}],"seeds":[{"id":"a","risk":3,"reason":"outage"}]}`
	run := func(t *testing.T, old, replacement string) (int, string, string) {
		t.Helper()
		if n := strings.Count(valid, old); n != 1 {
			t.Fatalf("fixture contains %q %d times, want 1", old, n)
		}
		var stdout, stderr bytes.Buffer
		code := cli.Run(context.Background(), []string{"simulate", "--input", inputFile(t, strings.Replace(valid, old, replacement, 1)), "--format", "json"}, &stdout, &stderr)
		return code, stdout.String(), stderr.String()
	}
	risks := func(t *testing.T, out string) map[string]int {
		t.Helper()
		var r report.Report
		if err := json.Unmarshal([]byte(out), &r); err != nil {
			t.Fatal(err)
		}
		got := map[string]int{}
		for _, c := range r.Cells {
			got[c.ID] = c.Risk
		}
		return got
	}
	if code, out, diagnostic := run(t, `"attenuation":2`, `"attenuation":2`); code != 0 || diagnostic != "" || risks(t, out)["b"] != 1 {
		t.Fatalf("valid snapshot: code=%d stdout=%q stderr=%q", code, out, diagnostic)
	}
	if code, out, diagnostic := run(t, `"attenuation":2`, `"attenuation":0`); code != 0 || diagnostic != "" || risks(t, out)["b"] != 3 {
		t.Fatalf("explicit attenuation 0: code=%d stdout=%q stderr=%q", code, out, diagnostic)
	}
	if code, out, diagnostic := run(t, `"risk":3`, `"risk":0`); code != 0 || diagnostic != "" || risks(t, out)["a"] != 0 {
		t.Fatalf("explicit risk 0: code=%d stdout=%q stderr=%q", code, out, diagnostic)
	}
	for _, tt := range []struct {
		name, old, replacement, want string
	}{
		{"attenuation_omitted", `,"attenuation":2`, ``, `stateripple: read snapshot: edges: missing required field "attenuation"`},
		{"risk_omitted", `"risk":3,`, ``, `stateripple: read snapshot: seeds: missing required field "risk"`},
		{"node_kind_omitted", `{"id":"b","kind":"store"}`, `{"id":"b"}`, `stateripple: read snapshot: nodes: missing required field "kind"`},
		{"edge_kind_omitted", `"kind":"calls",`, ``, `stateripple: read snapshot: edges: missing required field "kind"`},
		{"attenuation_null", `"attenuation":2`, `"attenuation":null`, `stateripple: read snapshot: null is not accepted in snapshot input`},
		{"risk_null", `"risk":3`, `"risk":null`, `stateripple: read snapshot: null is not accepted in snapshot input`},
		{"node_kind_null", `"kind":"store"`, `"kind":null`, `stateripple: read snapshot: null is not accepted in snapshot input`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			code, out, diagnostic := run(t, tt.old, tt.replacement)
			if code != 1 || out != "" || strings.TrimSpace(diagnostic) != tt.want {
				t.Fatalf("code=%d stdout=%q stderr=%q, want exit 1 and %q", code, out, diagnostic, tt.want)
			}
		})
	}
}

func TestSimulateJSONAndText(t *testing.T) {
	p := inputFile(t, emptySnapshot)
	for _, format := range []string{"json", "text"} {
		t.Run(format, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := cli.Run(context.Background(), []string{"simulate", "--input", p, "--format", format}, &stdout, &stderr)
			if code != 0 || stderr.Len() != 0 {
				t.Fatalf("code=%d stderr=%s", code, stderr.String())
			}
			if format == "json" {
				var r report.Report
				if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
					t.Fatal(err)
				}
				if r.Status != "complete" || len(r.InputSHA256) != 64 || r.GenerationCount != 1 {
					t.Fatalf("unexpected report: %+v", r)
				}
			} else if !strings.Contains(stdout.String(), "Status: complete") {
				t.Fatal(stdout.String())
			}
		})
	}
}

func TestSimulateOutputFile(t *testing.T) {
	p := inputFile(t, emptySnapshot)
	out := filepath.Join(t.TempDir(), "report.json")
	var stdout, stderr bytes.Buffer
	code := cli.Run(context.Background(), []string{"simulate", "--input", p, "--format", "json", "--out", out}, &stdout, &stderr)
	if code != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(b) {
		t.Fatalf("invalid JSON: %s", b)
	}
}

func TestSimulateGenerationLimitExitCode(t *testing.T) {
	p := inputFile(t, `{"schema_version":1,"ruleset":"risk-propagation-v1","max_generations":1,"nodes":[{"id":"pkg:A","kind":"package"},{"id":"pkg:B","kind":"package"}],"edges":[{"from":"pkg:A","to":"pkg:B","kind":"prod_import","attenuation":1}],"seeds":[{"id":"pkg:A","risk":3,"reason":"source_changed"}]}`)
	var stdout, stderr bytes.Buffer
	code := cli.Run(context.Background(), []string{"simulate", "--input", p, "--format", "json"}, &stdout, &stderr)
	var r report.Report
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if code != 2 || r.Status != "incomplete" || stderr.Len() != 0 {
		t.Fatalf("code=%d report=%+v stderr=%s", code, r, stderr.String())
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("output unavailable") }

func TestOutputFailureReturnsExecutionError(t *testing.T) {
	p := inputFile(t, emptySnapshot)
	var stderr bytes.Buffer
	code := cli.Run(context.Background(), []string{"simulate", "--input", p}, failingWriter{}, &stderr)
	if code != 1 || stderr.Len() == 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
}

func TestExecutableSimulatesOutsideRepositoryWithoutGoOrGit(t *testing.T) {
	bin := buildBinary(t)
	p := inputFile(t, emptySnapshot)
	cmd := exec.Command(bin, "simulate", "--input", p, "--format", "json")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "PATH="+t.TempDir())
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("execute: %v\n%s", err, stderr.String())
	}
	if !json.Valid(stdout.Bytes()) || stderr.Len() != 0 {
		t.Fatalf("stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

// childEnv returns the parent environment without the Go policy variables
// that analyze reports as unsupported, plus extra entries. It is applied only
// to child processes, so tests never rewrite the test process environment.
func childEnv(extra ...string) []string {
	env := []string{}
	for _, v := range os.Environ() {
		key, _, _ := strings.Cut(v, "=")
		switch key {
		case "GOFLAGS", "GOEXPERIMENT", "GOWORK":
		default:
			env = append(env, v)
		}
	}
	return append(env, extra...)
}

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "stateripple")
	build := exec.Command("go", "build", "-o", bin, "../../cmd/stateripple")
	build.Env = childEnv("GOFLAGS=", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off")
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, b)
	}
	return bin
}

func TestExecutableAnalyzeCleanCommittedRepository(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = childEnv("GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_AUTHOR_NAME=StateRipple Test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=StateRipple Test", "GIT_COMMITTER_EMAIL=test@example.invalid")
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, b)
		}
	}
	write := func(name, data string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "--quiet")
	write("go.mod", "module example.test/cli\n\ngo 1.23\n")
	write("value.go", "package cli\nconst Value = 1\n")
	write("value_test.go", "package cli\nimport \"testing\"\nfunc TestValue(t *testing.T) { if Value < 1 { t.Fatal(Value) } }\n")
	git("add", ".")
	git("commit", "--quiet", "-m", "baseline")
	write("value.go", "package cli\nconst Value = 2\n")
	git("add", ".")
	git("commit", "--quiet", "-m", "source change")
	for _, scenario := range []struct {
		name   string
		code   int
		status string
	}{{"source", 0, "complete"}, {"unknown", 2, "incomplete"}, {"dirty", 1, ""}} {
		t.Run(scenario.name, func(t *testing.T) {
			switch scenario.name {
			case "unknown":
				write("Makefile", "all:\n\ttrue\n")
				git("add", ".")
				git("commit", "--quiet", "-m", "unknown change")
			case "dirty":
				write("value.go", "package cli\nconst Value = 3\n")
			}
			cmd := exec.Command(bin, "analyze", "--format", "json")
			cmd.Dir = dir
			cmd.Env = childEnv()
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
			if code != scenario.code {
				t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
			}
			if code == 1 {
				if stdout.Len() != 0 || stderr.Len() == 0 {
					t.Fatalf("stdout=%s stderr=%s", stdout.String(), stderr.String())
				}
				return
			}
			var r report.Report
			if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
				t.Fatal(err)
			}
			if r.Status != scenario.status || r.BuildContext == nil || r.BuildContext.Module != "example.test/cli" || stderr.Len() != 0 {
				t.Fatalf("report=%+v stderr=%s", r, stderr.String())
			}
			if scenario.name == "source" && (len(r.TestTargets) != 1 || r.TestTargets[0].Risk != 3) {
				t.Fatalf("targets=%+v", r.TestTargets)
			}
			if scenario.name == "unknown" && (len(r.UnknownChanges) != 1 || r.UnknownChanges[0] != "Makefile") {
				t.Fatalf("unknown=%+v", r.UnknownChanges)
			}
		})
	}
}
