package risk_test

import (
	"reflect"
	"testing"

	"github.com/cottondesu/stateripple/internal/graph"
	"github.com/cottondesu/stateripple/internal/report"
	"github.com/cottondesu/stateripple/internal/rules/risk"
	"github.com/cottondesu/stateripple/internal/snapshot"
)

func fixture(nodes []string, pairs [][2]string, seeds []snapshot.Seed) snapshot.Snapshot {
	s := snapshot.Snapshot{SchemaVersion: 1, Ruleset: snapshot.Ruleset, MaxGenerations: 16, Seeds: seeds}
	for _, id := range nodes {
		s.Nodes = append(s.Nodes, graph.Node{ID: id, Kind: "package"})
	}
	for _, p := range pairs {
		s.Edges = append(s.Edges, graph.Edge{From: p[0], To: p[1], Kind: "prod_import", Attenuation: 1})
	}
	return s
}

func risks(r report.Report) map[string]int {
	m := map[string]int{}
	for _, c := range r.Cells {
		m[c.ID] = c.Risk
	}
	return m
}

func TestPropagationWhenGraphVaries(t *testing.T) {
	cases := []struct {
		name        string
		nodes       []string
		edges       [][2]string
		seeds       []snapshot.Seed
		want        map[string]int
		generations int
	}{
		{"empty", nil, nil, nil, map[string]int{}, 1},
		{"no_seed", []string{"A", "B"}, [][2]string{{"A", "B"}}, nil, map[string]int{"A": 0, "B": 0}, 1},
		{"single", []string{"A"}, nil, []snapshot.Seed{{ID: "A", Risk: 3, Reason: "changed"}}, map[string]int{"A": 3}, 1},
		{"line", []string{"A", "B", "C", "D"}, [][2]string{{"A", "B"}, {"B", "C"}, {"C", "D"}}, []snapshot.Seed{{ID: "A", Risk: 3, Reason: "changed"}}, map[string]int{"A": 3, "B": 2, "C": 1, "D": 0}, 3},
		{"branch", []string{"A", "B", "C", "Z"}, [][2]string{{"A", "B"}, {"A", "C"}}, []snapshot.Seed{{ID: "A", Risk: 3, Reason: "changed"}}, map[string]int{"A": 3, "B": 2, "C": 2, "Z": 0}, 2},
		{"diamond", []string{"A", "B", "C", "D"}, [][2]string{{"A", "B"}, {"A", "C"}, {"B", "D"}, {"C", "D"}}, []snapshot.Seed{{ID: "A", Risk: 3, Reason: "changed"}}, map[string]int{"A": 3, "B": 2, "C": 2, "D": 1}, 3},
		{"cycle", []string{"A", "B", "C"}, [][2]string{{"A", "B"}, {"B", "C"}, {"C", "A"}}, []snapshot.Seed{{ID: "A", Risk: 3, Reason: "changed"}}, map[string]int{"A": 3, "B": 2, "C": 1}, 3},
		{"max_path", []string{"A", "B", "C"}, [][2]string{{"A", "C"}, {"B", "C"}}, []snapshot.Seed{{ID: "A", Risk: 2, Reason: "weak"}, {ID: "B", Risk: 3, Reason: "strong"}}, map[string]int{"A": 2, "B": 3, "C": 2}, 2},
		{"duplicate_seed", []string{"A"}, nil, []snapshot.Seed{{ID: "A", Risk: 1, Reason: "weak"}, {ID: "A", Risk: 3, Reason: "strong"}, {ID: "A", Risk: 3, Reason: "strong"}}, map[string]int{"A": 3}, 1},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s := fixture(tt.nodes, tt.edges, tt.seeds)
			r, err := risk.Run(s)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(risks(r), tt.want) || r.GenerationCount != tt.generations || r.Status != "complete" {
				t.Fatalf("got %v generations %d status %s", risks(r), r.GenerationCount, r.Status)
			}
		})
	}
}

func TestSynchronousWavesWhenLineChanges(t *testing.T) {
	s := fixture([]string{"A", "B", "C"}, [][2]string{{"A", "B"}, {"B", "C"}}, []snapshot.Seed{{ID: "A", Risk: 3, Reason: "changed"}})
	r, err := risk.Run(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Waves) != 3 || r.Waves[1].Changes[0].ID != "B" || len(r.Waves[1].Changes) != 1 || r.Waves[2].Changes[0].ID != "C" {
		t.Fatalf("asynchronous trace: %+v", r.Waves)
	}
	for i, c := range r.Cells {
		if c.FirstActiveGeneration == nil || *c.FirstActiveGeneration != i {
			t.Fatalf("first activation: %+v", c)
		}
	}
}

func TestIncompleteWhenUpdateLimitReached(t *testing.T) {
	s := fixture([]string{"A", "B", "C"}, [][2]string{{"A", "B"}, {"B", "C"}}, []snapshot.Seed{{ID: "A", Risk: 3, Reason: "changed"}})
	s.MaxGenerations = 2
	r, err := risk.Run(s)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != "incomplete" || r.StopReason != "generation_limit" || r.GenerationCount != 2 {
		t.Fatalf("%+v", r)
	}
}

func TestTestSeedWhenOnlyTestChanges(t *testing.T) {
	s := fixture([]string{"pkg:A"}, nil, []snapshot.Seed{{ID: "test:A", Risk: 2, Reason: "test_changed"}})
	s.Nodes = append(s.Nodes, graph.Node{ID: "test:A", Kind: "test"})
	s.Edges = append(s.Edges, graph.Edge{From: "pkg:A", To: "test:A", Kind: "own_test"})
	r, err := risk.Run(s)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(risks(r), map[string]int{"pkg:A": 0, "test:A": 2}) {
		t.Fatal(risks(r))
	}
}

func TestOwnTestWhenProductionChanges(t *testing.T) {
	s := fixture([]string{"pkg:A"}, nil, []snapshot.Seed{{ID: "pkg:A", Risk: 3, Reason: "source_changed"}})
	s.Nodes = append(s.Nodes, graph.Node{ID: "test:A", Kind: "test"})
	s.Edges = append(s.Edges, graph.Edge{From: "pkg:A", To: "test:A", Kind: "own_test"})
	r, err := risk.Run(s)
	if err != nil {
		t.Fatal(err)
	}
	if risks(r)["test:A"] != 3 || len(r.TestTargets) != 1 {
		t.Fatal(r)
	}
}

func TestCausesWhenMaximumHasTies(t *testing.T) {
	s := fixture([]string{"A", "B", "C", "D"}, [][2]string{{"A", "D"}, {"B", "D"}, {"C", "D"}}, []snapshot.Seed{{ID: "A", Risk: 3, Reason: "a"}, {ID: "B", Risk: 3, Reason: "b"}, {ID: "C", Risk: 2, Reason: "c"}})
	r, err := risk.Run(s)
	if err != nil {
		t.Fatal(err)
	}
	c := r.Cells[3]
	if len(c.Causes) != 2 || c.Causes[0].Source != "A" || c.Causes[1].Source != "B" {
		t.Fatalf("wrong maximum causes: %+v", c.Causes)
	}
}

func TestCausesWhenEqualSupportArrivesLater(t *testing.T) {
	s := fixture([]string{"A", "B", "C", "D"}, [][2]string{{"A", "B"}, {"A", "C"}, {"B", "D"}, {"C", "D"}}, []snapshot.Seed{{ID: "A", Risk: 3, Reason: "a"}, {ID: "B", Risk: 2, Reason: "b"}})
	r, err := risk.Run(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Cells[3].Causes) != 2 || len(r.Waves[1].Changes[1].Causes) != 1 {
		t.Fatalf("history/final support: %+v", r)
	}
}
