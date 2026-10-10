package report_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cottondesu/stateripple/internal/graph"
	"github.com/cottondesu/stateripple/internal/report"
)

func TestJSONGoldenEmptyReport(t *testing.T) {
	r := report.Report{SchemaVersion: 1, Ruleset: "risk-propagation-v1", Status: "complete", GenerationCount: 1, StopReason: "converged"}
	got, err := report.JSON(r)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"schema_version":1,"ruleset":"risk-propagation-v1","status":"complete","input_sha256":"","generation_count":1,"cells":[],"test_targets":[],"waves":[],"unknown_changes":[],"warnings":[],"changes":[],"build_context":null,"stop_reason":"converged"}` + "\n"
	if string(got) != want {
		t.Fatalf("golden JSON mismatch\ngot: %s\nwant: %s", got, want)
	}
}

func TestJSONStableOrderingPreservesCaller(t *testing.T) {
	edge := graph.Edge{From: "pkg:A", To: "test:B", Kind: "test_import", Attenuation: 1}
	r := report.Report{Cells: []report.Cell{{ID: "test:B", Causes: []report.Cause{{Source: "pkg:B", Reason: "own"}, {Source: "pkg:A", Edge: &edge, Reason: "import"}}}, {ID: "pkg:A"}}, TestTargets: []report.TestTarget{{ID: "test:C", Risk: 1}, {ID: "test:B", Risk: 2}}, Warnings: []string{"z", "a", "z"}, BuildContext: &report.BuildContext{BuildTags: []string{"z", "a"}}}
	got, err := report.JSON(r)
	if err != nil {
		t.Fatal(err)
	}
	r2 := r
	r2.Cells = []report.Cell{r.Cells[1], r.Cells[0]}
	r2.TestTargets = []report.TestTarget{r.TestTargets[1], r.TestTargets[0]}
	r2.Warnings = []string{"a", "z"}
	again, err := report.JSON(r2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, again) {
		t.Fatalf("logical reports serialize differently: %s versus %s", got, again)
	}
	if r.Cells[0].ID != "test:B" || r.Cells[0].Causes[0].Source != "pkg:B" || r.BuildContext.BuildTags[0] != "z" {
		t.Fatal("serialization mutated caller")
	}
}

func TestTextIncludesAnalysisEvidence(t *testing.T) {
	r := report.Report{Status: "incomplete", StopReason: "generation_limit", Cells: []report.Cell{{ID: "pkg:A", Risk: 3, Causes: []report.Cause{{Source: "pkg:A", Reason: "source_changed"}}}}, TestTargets: []report.TestTarget{{ID: "test:A", Risk: 2}}, UnknownChanges: []string{"Makefile"}, Warnings: []string{"dependency missing"}, Changes: []report.FileChange{{Path: "a.go", Status: "M", Classification: "source"}}, BuildContext: &report.BuildContext{GoVersion: "go1.23", GOOS: "linux", GOARCH: "amd64", Module: "example.test/project"}}
	got := report.Text(r)
	for _, want := range []string{"incomplete", "Critical: 1", "test:A", "source_changed", "Makefile", "dependency missing", "a.go", "linux/amd64", "example.test/project"} {
		if !strings.Contains(got, want) {
			t.Errorf("text missing %q", want)
		}
	}
}

func TestTextOmitsSoftwareSectionsForGenericSimulation(t *testing.T) {
	r := report.Report{Status: "complete", StopReason: "converged", Cells: []report.Cell{{ID: "port", Kind: "warehouse", Risk: 2}}}
	got := report.Text(r)
	for _, unwanted := range []string{"Changed files", "Recommended test targets", "Selected tests", "bug probability", "Git"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("generic text contains %q:\n%s", unwanted, got)
		}
	}
	if !strings.HasPrefix(got, "StateRipple v0.1.0\n") || !strings.HasSuffix(got, "Risk is a propagated priority, not a probability.\n") {
		t.Fatalf("text:\n%s", got)
	}
}

func TestTextShowsTestTargetsWhenSimulationHasTests(t *testing.T) {
	r := report.Report{Status: "complete", StopReason: "converged", TestTargets: []report.TestTarget{{ID: "test:A", Risk: 3}}}
	got := report.Text(r)
	if !strings.Contains(got, "Recommended test targets:\n  risk 3 test:A\n") || strings.Contains(got, "Changed files") || !strings.HasSuffix(got, "Selected tests have not been executed.\n") {
		t.Fatalf("text:\n%s", got)
	}
}

func TestTextShowsGitSectionsForAnalysis(t *testing.T) {
	r := report.Report{Status: "complete", StopReason: "converged", BuildContext: &report.BuildContext{GoVersion: "go1.23"}}
	got := report.Text(r)
	if !strings.Contains(got, "Changed files:\n  none\n") || !strings.Contains(got, "Recommended test targets:\n  none; this does not prove safety\n") || !strings.HasSuffix(got, "Selected tests have not been executed.\n") {
		t.Fatalf("text:\n%s", got)
	}
}
