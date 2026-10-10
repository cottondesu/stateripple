package adapters

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/cottondesu/stateripple/internal/graph"
	"github.com/cottondesu/stateripple/internal/snapshot"
)

func goProfileBase() snapshot.Snapshot {
	return snapshot.Snapshot{SchemaVersion: 1, Ruleset: snapshot.Ruleset, MaxGenerations: 16,
		Nodes: []graph.Node{{ID: "pkg:A", Kind: "package"}, {ID: "pkg:B", Kind: "package"}, {ID: "test:A", Kind: "test"}},
		Edges: []graph.Edge{
			{From: "pkg:A", To: "pkg:B", Kind: "prod_import", Attenuation: 1},
			{From: "pkg:A", To: "test:A", Kind: "own_test", Attenuation: 0},
			{From: "pkg:B", To: "test:A", Kind: "test_import", Attenuation: 1},
		},
		Seeds: []snapshot.Seed{{ID: "pkg:A", Risk: 3, Reason: "source_changed:a.go"}}}
}

func TestGoProfileAcceptsGoGraph(t *testing.T) {
	if err := validateGoProfile(goProfileBase()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("../../testdata/example.json")
	if err != nil {
		t.Fatal(err)
	}
	s, err := snapshot.Read(strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateGoProfile(s); err != nil {
		t.Fatal(err)
	}
}

// The core used to reject these cases. The generic core now accepts them, so
// the Go adapter must keep rejecting them for analyze to stay fail-closed.
func TestGoProfileRejectsFormerCoreCases(t *testing.T) {
	cases := []struct {
		name string
		edit func(*snapshot.Snapshot)
	}{
		{"node_kind", func(s *snapshot.Snapshot) { s.Nodes[0].Kind = "service" }},
		{"edge_kind", func(s *snapshot.Snapshot) { s.Edges[0].Kind = "depends_on" }},
		{"test_out", func(s *snapshot.Snapshot) {
			s.Edges[0] = graph.Edge{From: "test:A", To: "pkg:B", Kind: "prod_import", Attenuation: 1}
		}},
		{"prod_to_test", func(s *snapshot.Snapshot) { s.Edges[2].Kind = "prod_import" }},
		{"test_to_prod", func(s *snapshot.Snapshot) { s.Edges[0].Kind = "test_import" }},
		{"own_test_to_prod", func(s *snapshot.Snapshot) {
			s.Edges[0] = graph.Edge{From: "pkg:A", To: "pkg:B", Kind: "own_test", Attenuation: 0}
		}},
		{"prod_import_attenuation_0", func(s *snapshot.Snapshot) { s.Edges[0].Attenuation = 0 }},
		{"prod_import_attenuation_2", func(s *snapshot.Snapshot) { s.Edges[0].Attenuation = 2 }},
		{"own_test_attenuation_1", func(s *snapshot.Snapshot) { s.Edges[1].Attenuation = 1 }},
		{"test_import_attenuation_0", func(s *snapshot.Snapshot) { s.Edges[2].Attenuation = 0 }},
		{"test_import_attenuation_3", func(s *snapshot.Snapshot) { s.Edges[2].Attenuation = 3 }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s := goProfileBase()
			tt.edit(&s)
			if _, err := snapshot.Normalize(s); err != nil {
				t.Fatalf("generic core should accept the case: %v", err)
			}
			if err := validateGoProfile(s); err == nil {
				t.Fatal("Go profile accepted invalid graph")
			}
		})
	}
}

// The graph that analyze collects from a real repository must satisfy the Go
// profile; Analyze returns an error otherwise.
func TestAnalyzeCollectsGoProfileGraph(t *testing.T) {
	root := fixture(t, basicFiles())
	writeFile(t, root, "a/a.go", "package a\nfunc A() { _ = 1 }\n")
	commit(t, root)
	s, m, err := Analyze(context.Background(), root, "HEAD~1")
	if err != nil || m.Incomplete {
		t.Fatalf("%v %+v", err, m)
	}
	if err := validateGoProfile(s); err != nil {
		t.Fatal(err)
	}
	if len(s.Nodes) != 4 || len(s.Edges) != 4 {
		t.Fatalf("graph: %+v", s)
	}
}
