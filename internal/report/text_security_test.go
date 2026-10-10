package report_test

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/cottondesu/stateripple/internal/graph"
	"github.com/cottondesu/stateripple/internal/report"
)

func TestTextEscapesEveryUntrustedString(t *testing.T) {
	// Given
	edge := graph.Edge{From: "from\nnode", To: "to\rnode", Kind: "prod\timport", Attenuation: 1}
	input := report.Report{
		Ruleset:         "risk\tset",
		Status:          "complete\nStatus: forged",
		InputSHA256:     "sha\x1bvalue",
		StopReason:      "done\rreason",
		GenerationCount: 1,
		Cells: []report.Cell{{
			ID:   "pkg:\ncell",
			Risk: 3,
			Causes: []report.Cause{{
				Source: "source\rnode",
				Edge:   &edge,
				Reason: "reason\nStatus: forged",
			}},
		}},
		TestTargets: []report.TestTarget{{ID: "test:\x1btarget", Risk: 2}},
		Changes: []report.FileChange{{
			Path:           "日本語.go\r\nnext",
			Status:         "M\nStatus: forged",
			Classification: "source\tclass",
		}},
		BuildContext: &report.BuildContext{
			GoVersion:   "go1.23\nnext",
			GOOS:        "lin\rux",
			GOARCH:      "amd\t64",
			CGOEnabled:  "1\x1b",
			Module:      "例.invalid/mod\nnext",
			BuildTags:   []string{"build\ntag"},
			ToolTags:    []string{"tool\rtag"},
			ReleaseTags: []string{"release\ttag"},
		},
		UnknownChanges: []string{"unknown\nStatus: forged"},
		Warnings:       []string{"warning\x1b[2J\nStatus: forged"},
	}
	wantEscaped := []string{
		input.Ruleset,
		input.Status,
		input.InputSHA256,
		input.StopReason,
		input.Cells[0].ID,
		input.Cells[0].Causes[0].Source,
		input.Cells[0].Causes[0].Edge.Kind,
		input.Cells[0].Causes[0].Reason,
		input.TestTargets[0].ID,
		input.Changes[0].Path,
		input.Changes[0].Status,
		input.Changes[0].Classification,
		input.BuildContext.GoVersion,
		input.BuildContext.GOOS,
		input.BuildContext.GOARCH,
		input.BuildContext.CGOEnabled,
		input.BuildContext.Module,
		input.BuildContext.BuildTags[0],
		input.BuildContext.ToolTags[0],
		input.BuildContext.ReleaseTags[0],
		input.UnknownChanges[0],
		input.Warnings[0],
	}

	// When
	got := report.Text(input)

	// Then
	for _, raw := range wantEscaped {
		quoted := strconv.QuoteToGraphic(raw)
		if want := quoted[1 : len(quoted)-1]; !strings.Contains(got, want) {
			t.Errorf("text missing escaped value %s", want)
		}
	}
	if strings.Count(got, "\nStatus: ") != 1 {
		t.Errorf("untrusted value injected a Status line:\n%s", got)
	}
	if strings.ContainsAny(got, "\r\t\x1b") {
		t.Errorf("text contains a raw control character: %q", got)
	}
	if !strings.Contains(got, "日本語") || !strings.Contains(got, "例.invalid") {
		t.Errorf("printable Unicode was not preserved: %q", got)
	}
}

func TestTextPreservesCanonicalJSONAndCaller(t *testing.T) {
	// Given
	edge := graph.Edge{From: "z", To: "a", Kind: "kind\nvalue", Attenuation: 1}
	input := report.Report{
		Cells: []report.Cell{
			{ID: "z", Causes: []report.Cause{{Source: "z", Edge: &edge, Reason: "reason\nvalue"}}},
			{ID: "a"},
		},
		TestTargets: []report.TestTarget{{ID: "z", Risk: 1}, {ID: "a", Risk: 2}},
		Warnings:    []string{"z\nwarning", "a\rwarning"},
		BuildContext: &report.BuildContext{
			BuildTags: []string{"z\ntag", "a\ttag"},
		},
	}
	beforeCanonical, err := report.JSON(input)
	if err != nil {
		t.Fatal(err)
	}
	beforeCaller, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}

	// When
	report.Text(input)

	// Then
	afterCanonical, err := report.JSON(input)
	if err != nil {
		t.Fatal(err)
	}
	afterCaller, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(beforeCanonical, afterCanonical) {
		t.Fatalf("Text changed canonical JSON:\nbefore: %s\nafter:  %s", beforeCanonical, afterCanonical)
	}
	if !bytes.Equal(beforeCaller, afterCaller) {
		t.Fatalf("Text mutated its caller:\nbefore: %s\nafter:  %s", beforeCaller, afterCaller)
	}
}
